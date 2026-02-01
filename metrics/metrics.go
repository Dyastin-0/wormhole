// Package metrics implement a metrics for an io.ReadWriteCloser.
package metrics

import (
	"io"
	"sync/atomic"
	"time"
)

// MetricsReadWriteCLoser implements io.ReadWriteCloser.
type MetricsReadWriteCloser struct {
	rwc     io.ReadWriteCloser
	metrics *Metrics
}

// NewMetricsReadWriteCloser returns a new MetricsReadWriteCloser.
func NewMetricsReadWriteCloser(rwc io.ReadWriteCloser, m *Metrics) *MetricsReadWriteCloser {
	return &MetricsReadWriteCloser{
		rwc:     rwc,
		metrics: m,
	}
}

// Read reads p using the underlying io.ReadWriteCLoser and adds n in the ingress metrics.
func (mrwc *MetricsReadWriteCloser) Read(p []byte) (n int, err error) {
	n, err = mrwc.rwc.Read(p)
	if n > 0 {
		mrwc.metrics.AddIngressBytes(uint64(n))
	}
	return n, err
}

// Write writes p using the underlying io.ReadWriteCLoser and adds n in the egress metrics.
func (mrwc *MetricsReadWriteCloser) Write(p []byte) (n int, err error) {
	n, err = mrwc.rwc.Write(p)
	if n > 0 {
		mrwc.metrics.AddEgressBytes(uint64(n))
	}
	return n, err
}

// Close closes the underlying io.ReadWriteCloser.
func (mrwc *MetricsReadWriteCloser) Close() error {
	return mrwc.rwc.Close()
}

// Metrics represents data ingress/egress metrics for a network connection.
type Metrics struct {
	// IngressBytes represents the total bytes received from external connections (ingress).
	IngressBytes uint64
	// EgressBytes represents the total outgoing total bytes (egress).
	EgressBytes uint64
	// ConnectionCount specifies the total number of connections.
	ConnectionCount uint64
	// StartTime represents the time stamp when the metrics started.
	StartTime time.Time
	// ActiveConnections represents current active connections.
	ActiveConnections int32
	// RTT represent the single roundtrip latency.
	RTT uint32
	// Track last reported values for delta calculation.
	lastIngressBytes uint64
	lastEgressBytes  uint64
}

// New creates a new Metrics instance.
func New() *Metrics {
	return &Metrics{
		StartTime: time.Now(),
	}
}

// AddEgressBytes atomically add bytes to the egress counter.
func (m *Metrics) AddEgressBytes(bytes uint64) {
	atomic.AddUint64(&m.EgressBytes, bytes)
}

// AddIngressBytes atomically add bytes to the ingress counter.
func (m *Metrics) AddIngressBytes(bytes uint64) {
	atomic.AddUint64(&m.IngressBytes, bytes)
}

// IncrementConnections atomically increments the connection counter.
func (m *Metrics) IncrementConnections() {
	atomic.AddUint64(&m.ConnectionCount, 1)
	atomic.AddInt32(&m.ActiveConnections, 1)
}

// DecrementActiveConnections atomically decrements active connection counter.
func (m *Metrics) DecrementActiveConnections() {
	atomic.AddInt32(&m.ActiveConnections, -1)
}

// GetIngressBytes returns the current ingress byte count.
func (m *Metrics) GetIngressBytes() uint64 {
	return atomic.LoadUint64(&m.IngressBytes)
}

// GetEgressBytes returns the current egress byte count.
func (m *Metrics) GetEgressBytes() uint64 {
	return atomic.LoadUint64(&m.EgressBytes)
}

// GetConnectionCount returns the total connection count.
func (m *Metrics) GetConnectionCount() uint64 {
	return atomic.LoadUint64(&m.ConnectionCount)
}

// GetActiveConnections returns current active connections.
func (m *Metrics) GetActiveConnections() int32 {
	return atomic.LoadInt32(&m.ActiveConnections)
}

// GetIngressBytesDelta returns the delta since last check and updates the last value.
func (m *Metrics) GetIngressBytesDelta() uint64 {
	current := atomic.LoadUint64(&m.IngressBytes)
	last := atomic.LoadUint64(&m.lastIngressBytes)
	delta := current - last
	atomic.StoreUint64(&m.lastIngressBytes, current)
	return delta
}

// GetEgressBytesDelta returns the delta since last check and updates the last value.
func (m *Metrics) GetEgressBytesDelta() uint64 {
	current := atomic.LoadUint64(&m.EgressBytes)
	last := atomic.LoadUint64(&m.lastEgressBytes)
	delta := current - last
	atomic.StoreUint64(&m.lastEgressBytes, current)
	return delta
}

// SetRTT atomically sets the RTT value.
func (m *Metrics) SetRTT(rtt uint32) {
	atomic.StoreUint32(&m.RTT, rtt)
}

// GetRTT atomically gets the RTT value.
func (m *Metrics) GetRTT() uint32 {
	return atomic.LoadUint32(&m.RTT)
}

// GetUptime returns the duration since metrics started.
func (m *Metrics) GetUptime() time.Duration {
	return time.Since(m.StartTime)
}

// NewProxyReadWriteCloser return a new MetricsReadWriteCLoser using the underlying metrics.
func (m *Metrics) NewProxyReadWriteCloser(rwc io.ReadWriteCloser) *MetricsReadWriteCloser {
	return NewMetricsReadWriteCloser(rwc, m)
}
