package server

import (
	"context"
	"fmt"
	"net"
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
}

// From opens a stream (remoteStream) from the session then forwards the stream to it.
func (t *Tunnel) From(ctx context.Context, stream net.Conn) error {
	defer stream.Close()

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

	_, err = remoteStream.Write(serializedHeader)
	if err != nil {
		return fmt.Errorf("failed to write access header: %w", err)
	}

	proxyCtx, proxyCancel := context.WithCancel(ctx)
	defer proxyCancel()

	go func() {
		select {
		case <-t.session.CloseChan():
			stream.Close()
			remoteStream.Close()
		case <-proxyCtx.Done():
			return
		}
	}()

	if t.metrics != nil {
		proxyStream := t.metrics.NewProxyReadWriter(stream)
		return proxy.StreamWithContext(proxyCtx, proxyStream, remoteStream)
	}
	return proxy.StreamWithContext(proxyCtx, stream, remoteStream)
}
