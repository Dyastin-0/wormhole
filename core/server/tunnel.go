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
	// allowTLSPassthrough
	allowTLSPassthrough bool
	// httpLogch is used to send HTTP logs to the client.
	httpLogch chan *HTTPLog
	// domain specifies the tunnel's subdomain.
	domain string
	// createdAt specifies the tunnel's creation time.
	createdAt time.Time
	// port is the allocated port for this tunnel (443 if HTTP/TLS).
	port int
	// tcpListener is the tunnel's listener.
	tcpListener net.Listener
	// controlStream is a yamux.Stream used to handle tunnel request and controls.
	controlStream net.Conn
}

type HTTPLog struct {
	*proto.HTTPLog
	Method string
	Path   string
	Status int
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

	_, err = remoteStream.Write(serializedHeader)
	if err != nil {
		return fmt.Errorf("failed to write access header: %w", err)
	}

	if t.metrics != nil {
		proxyStream := t.metrics.NewProxyConn(ystream)
		return stream.StreamWithContext(ctx, proxyStream, remoteStream)
	}
	return stream.StreamWithContext(ctx, ystream, remoteStream)
}

// ProxyWithInspect opens a stream from the session, forwards the stream to it,
// then inspects and logs the response.
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

	_, err = remoteStream.Write(serializedHeader)
	if err != nil {
		return fmt.Errorf("failed to write access header: %w", err)
	}

	if t.metrics != nil {
		proxyStream := t.metrics.NewProxyConn(ystream)
		return stream.StreamHTTPWithContext(ctx, proxyStream, remoteStream, func(start time.Time, method, path string, status int) {
			t.logHTTPRequest(start, method, path, status)
		})
	}
	return stream.StreamHTTPWithContext(ctx, ystream, remoteStream, func(start time.Time, method, path string, status int) {
		t.logHTTPRequest(start, method, path, status)
	})
}

// logHTTPRequest logs an HTTP request to the tunnel's HTTP log channel.
func (t *Tunnel) logHTTPRequest(start time.Time, method, path string, status int) {
	duration := uint32(time.Since(start).Microseconds())

	log := proto.NewHTTPLog(
		time.Now().Unix(),
		duration,
	)

	select {
	case t.httpLogch <- &HTTPLog{
		HTTPLog: log,
		Method:  method,
		Path:    path,
		Status:  status,
	}:

	default:
	}
}
