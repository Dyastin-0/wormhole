// Package observer provides observability for the Wormhole server.
package observer

import (
	"context"
	"time"
)

// Observer defines the interface for observability operations.
type Observer interface {
	// Tunnel lifecycle.
	RecordTunnelCreated(ctx context.Context, protocol string)
	RecordTunnelClosed(ctx context.Context, protocol, reason string, duration time.Duration)

	// Connection metrics per tunnel.
	RecordConnectionStart(ctx context.Context, domain, protocol string)
	RecordConnectionEnd(ctx context.Context, domain, protocol string, duration time.Duration)

	// Traffic metrics per tunnel.
	RecordTraffic(ctx context.Context, domain string, ingress, egress uint64)

	// HTTP metrics per tunnel.
	RecordHTTPRequest(ctx context.Context, domain, method, statusCode string, duration time.Duration)

	// Performance per tunnel.
	UpdateRTT(ctx context.Context, domain string, rttMicroseconds uint32)
}

