package observer

import "time"

// NoopObserver implements Observer but does nothing (for when metrics are disabled).
type NoopObserver struct{}

func (n *NoopObserver) RecordTunnelCreated(protocol string)                                         {}
func (n *NoopObserver) RecordTunnelClosed(protocol, reason string, duration time.Duration)          {}
func (n *NoopObserver) RecordConnectionStart(domain, protocol string)                               {}
func (n *NoopObserver) RecordConnectionEnd(domain, protocol string, duration time.Duration)         {}
func (n *NoopObserver) RecordTraffic(domain string, ingress, egress uint64)                         {}
func (n *NoopObserver) RecordHTTPRequest(domain, method, statusCode string, duration time.Duration) {}
func (n *NoopObserver) UpdateRTT(domain string, rttMicroseconds uint32)                             {}
