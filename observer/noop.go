package observer

import (
	"context"
	"time"
)

// NoopObserver implements Observer but does nothing (for when metrics are disabled).
type NoopObserver struct{}

func (n *NoopObserver) RecordTunnelCreated(ctx context.Context, protocol string)                                         {}
func (n *NoopObserver) RecordTunnelClosed(ctx context.Context, protocol, reason string, duration time.Duration)          {}
func (n *NoopObserver) RecordConnectionStart(ctx context.Context, domain, protocol string)                               {}
func (n *NoopObserver) RecordConnectionEnd(ctx context.Context, domain, protocol string, duration time.Duration)         {}
func (n *NoopObserver) RecordTraffic(ctx context.Context, domain string, ingress, egress uint64)                         {}
func (n *NoopObserver) RecordHTTPRequest(ctx context.Context, domain, method, statusCode string, duration time.Duration) {}
func (n *NoopObserver) UpdateRTT(ctx context.Context, domain string, rttMicroseconds uint32)                             {}

