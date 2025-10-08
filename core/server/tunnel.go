package server

import (
	"context"
	"fmt"
	"io"
	"net"

	"github.com/Dyastin-0/wormhole/core/metrics"
	"github.com/Dyastin-0/wormhole/core/proto"
	"github.com/Dyastin-0/wormhole/core/proxy"
	"github.com/Dyastin-0/wormhole/dnsmanager"
	"github.com/hashicorp/yamux"
)

// Tunnel represents a wormhole tunnel.
type Tunnel struct {
	// session specifies the long-live yamux session.
	session *yamux.Session
	// dnsRecord specifies the tunnel's DNS record.
	dnsRecord *dnsmanager.DNSRecord
	// proto specifies the tunnel protocol.
	proto uint8
	// metrics represents the ingress/egress metrics for the underlying tunnel.
	metrics *metrics.Metrics
}

// From opens a stream (remoteStream) from the session then forwards the stream to it.
func (t *Tunnel) From(ctx context.Context, stream net.Conn) error {
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

	buf := make([]byte, proto.HeaderSize)
	_, err = io.ReadFull(remoteStream, buf)
	if err != nil {
		return fmt.Errorf("failed to read ack header: %w", err)
	}

	ackHeader, err := proto.DeserializeHeader(buf)
	if err != nil {
		return fmt.Errorf("failed to deserialize ack header: %w", err)
	}

	if ackHeader.Type != proto.TypeAck {
		return fmt.Errorf("unexpected header type: %v", ackHeader.Type)
	}

	proxyCtx, proxyCancel := context.WithCancel(context.Background())
	defer proxyCancel()

	done := make(chan struct{})
	defer close(done)

	go func() {
		select {
		case <-ctx.Done():
			proxyCancel()
		case <-t.session.CloseChan():
			proxyCancel()
		case <-done:
			return
		}
	}()

	proxyStream := t.metrics.NewProxyReadWriter(stream)

	return proxy.StreamWithContext(proxyCtx, proxyStream, remoteStream)
}
