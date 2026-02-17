package server

import (
	"context"
	"fmt"
	"net"
	"time"

	"github.com/Dyastin-0/wormhole/core/auth"
	"github.com/Dyastin-0/wormhole/core/proto"
	"github.com/Dyastin-0/wormhole/metrics"
	"github.com/Dyastin-0/wormhole/stream"
	"github.com/hashicorp/yamux"
)

// Tunnel represents a wormhole tunnel.
type Tunnel struct {
	session             *yamux.Session
	proto               uint8
	ttl                 time.Duration
	metrics             *metrics.Metrics
	auth                auth.Authenticator
	allowHTTP           bool
	allowTLSPassthrough bool
	// eventch receives a correlated HTTPEvent for each completed request/response
	// pair. Replaces the old httpLogch+requestch split. Nil if HTTP logging is off.
	eventch       chan stream.HTTPEvent
	domain        string
	createdAt     time.Time
	port          int
	tcpListener   net.Listener
	controlStream net.Conn
}

// HTTPLog is the server-side envelope sent to the client over the log stream.
// It carries the full proto.HTTPLog (Timestamp + Duration) plus the HTTP
// metadata and the raw headers/body copies for TUI inspection.
type HTTPLog struct {
	*proto.HTTPLog
	Method      string
	Path        string
	Status      int
	RespSize    int64
	ReqHeaders  map[string][]string
	ReqBody     []byte
	RespHeaders map[string][]string
	RespBody    []byte
}

// Proxy opens a stream from the session then forwards the stream to it.
func (t *Tunnel) Proxy(ctx context.Context, ystream net.Conn) error {
	defer ystream.Close()

	remoteStream, err := t.session.Open()
	if err != nil {
		return fmt.Errorf("failed to open yamux session: %w", err)
	}
	defer remoteStream.Close()

	header := proto.NewHeader(proto.TypeAccess, 0)
	serializedHeader, err := proto.SerializeHeader(header)
	if err != nil {
		return fmt.Errorf("failed to serialize header: %w", err)
	}
	if _, err = remoteStream.Write(serializedHeader); err != nil {
		return fmt.Errorf("failed to write access header: %w", err)
	}

	if t.metrics != nil {
		return stream.StreamWithContext(ctx, t.metrics.NewProxyConn(ystream), remoteStream)
	}
	return stream.StreamWithContext(ctx, ystream, remoteStream)
}

// ProxyWithInspect opens a stream from the session, proxies HTTP traffic,
// and emits HTTPEvents into t.eventch for each request/response pair.
// The server measures all timing; the client does not participate.
func (t *Tunnel) ProxyWithInspect(ctx context.Context, ystream net.Conn) error {
	defer ystream.Close()

	remoteStream, err := t.session.Open()
	if err != nil {
		return fmt.Errorf("failed to open yamux session: %w", err)
	}
	defer remoteStream.Close()

	header := proto.NewHeader(proto.TypeAccess, 0)
	serializedHeader, err := proto.SerializeHeader(header)
	if err != nil {
		return fmt.Errorf("failed to serialize header: %w", err)
	}
	if _, err = remoteStream.Write(serializedHeader); err != nil {
		return fmt.Errorf("failed to write access header: %w", err)
	}

	src := ystream
	if t.metrics != nil {
		src = t.metrics.NewProxyConn(ystream)
	}

	return stream.StreamHTTPWithContext(ctx, src, remoteStream, t.eventch)
}

// logLoop drains t.eventch, builds HTTPLog entries with server-side timing,
// and forwards them to the provided send function (which writes to the client
// log stream). Run this as a goroutine when the tunnel is created.
func (t *Tunnel) logLoop(ctx context.Context, send func(*HTTPLog) error) {
	for {
		select {
		case <-ctx.Done():
			return
		case ev, ok := <-t.eventch:
			if !ok {
				return
			}

			duration := uint32(time.Since(ev.Start).Microseconds())

			log := &HTTPLog{
				HTTPLog: proto.NewHTTPLog(
					ev.Start.Unix(),
					duration,
				),
				Method:      ev.Method,
				Path:        ev.Path,
				Status:      ev.Status,
				RespSize:    ev.RespSize,
				ReqHeaders:  map[string][]string(ev.ReqHeaders),
				ReqBody:     ev.ReqBody,
				RespHeaders: map[string][]string(ev.RespHeaders),
				RespBody:    ev.RespBody,
			}

			if err := send(log); err != nil {
				return
			}
		}
	}
}
