// Package observer provides observability for the Wormhole server.
package observer

import (
	"time"
)

// Observer defines the interface for observability operations.
type Observer interface {
	// Tunnel lifecycle.
	RecordTunnelCreated(protocol string)
	RecordTunnelClosed(protocol, reason string, duration time.Duration)

	// Connection metrics per tunnel.
	RecordConnectionStart(domain, protocol string)
	RecordConnectionEnd(domain, protocol string, duration time.Duration)

	// Traffic metrics per tunnel.
	RecordTraffic(domain string, ingress, egress uint64)

	// HTTP metrics per tunnel.
	RecordHTTPRequest(domain, method, statusCode string, duration time.Duration)

	// Performance per tunnel.
	UpdateRTT(domain string, rttMicroseconds uint32)
}
