package server

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"net"
	"net/http"
	"time"

	"github.com/Dyastin-0/wormhole/core/auth"
	"github.com/Dyastin-0/wormhole/core/metrics"
	"github.com/Dyastin-0/wormhole/core/proto"
	"github.com/Dyastin-0/wormhole/core/proxy"
	"github.com/hashicorp/yamux"
)

// Tunnel represents a wormhole tunnel.
type Tunnel struct {
	// session specifies the long-live yamux session.
	session *yamux.Session
	// proto specifies the tunnel protocol.
	proto uint8
	// ttl specifies the tunnel's time-to-live in hour.
	ttl time.Duration
	// metrics represents the ingress/egress metrics for the underlying tunnel.
	metrics *metrics.Metrics
	// auth represents the authentication method that will be used for HTTP tunnels.
	auth auth.Authenticator
	// allowHTTP specifies if this tunnel allows HTTP requests, ignored if tunnel protocol is HTTP.
	allowHTTP bool
	// httpLogch is used to send HTTP logs to the client.
	httpLogch chan *proto.HTTPLog
}

// Proxy opens a stream from the session then forwards the traffic.
// If method and path are provided, it intercepts the response to log the status code.
func (t *Tunnel) Proxy(ctx context.Context, localStream net.Conn, method, path string) error {
	defer localStream.Close()

	if t.metrics != nil {
		t.metrics.IncrementConnections()
		defer t.metrics.DecrementActiveConnections()
	}

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

	proxyCtx, proxyCancel := context.WithCancel(ctx)
	defer proxyCancel()

	go func() {
		select {
		case <-t.session.CloseChan():
			localStream.Close()
			remoteStream.Close()
		case <-proxyCtx.Done():
			return
		}
	}()

	var local io.ReadWriter = localStream
	if t.metrics != nil {
		local = t.metrics.NewProxyReadWriter(localStream)
	}

	if t.httpLogch != nil && method != "" {
		return t.proxyAndLog(proxyCtx, local, remoteStream, method, path)
	}

	return proxy.StreamWithContext(proxyCtx, local, remoteStream)
}

func (t *Tunnel) proxyAndLog(ctx context.Context, local io.ReadWriter, remote io.ReadWriter, method, path string) error {
	start := time.Now()

	br := bufio.NewReader(remote)
	resp, err := http.ReadResponse(br, nil)

	if err == nil {
		t.sendLog(start, method, path, int32(resp.StatusCode))

		if err := resp.Write(local); err != nil {
			return fmt.Errorf("failed to write response headers: %w", err)
		}
		resp.Body.Close()
	} else {
		t.sendLog(start, method, path, 0)
	}

	wrappedRemote := &ReadWriterWrapper{
		Reader: br,
		Writer: remote,
	}

	return proxy.StreamWithContext(ctx, local, wrappedRemote)
}

func (t *Tunnel) sendLog(start time.Time, method, path string, status int32) {
	duration := uint32(time.Since(start).Microseconds())
	logEntry := proto.NewHTTPLog(
		time.Now().Unix(),
		method,
		path,
		status,
		duration,
	)

	select {
	case t.httpLogch <- logEntry:
	default:
	}
}

type ReadWriterWrapper struct {
	io.Reader
	io.Writer
}
