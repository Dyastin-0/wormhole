// Package metrics implement a metrics for an io.ReadWriter.
package metrics

import (
	"io"
	"sync/atomic"
	"time"
)

// MetricsReadWriter implements io.ReadWriter.
type MetricsReadWriter struct {
	rw      io.ReadWriter
	metrics *Metrics
}

// NewMetricsReadWriter returns a new MetricsReadWriter.
func NewMetricsReadWriter(rw io.ReadWriter, m *Metrics) *MetricsReadWriter {
	return &MetricsReadWriter{
		rw:      rw,
		metrics: m,
	}
}

// Read reads p using the underlying io.ReadWriter and adds n in the ingress metrics.
func (mrw *MetricsReadWriter) Read(p []byte) (n int, err error) {
	n, err = mrw.rw.Read(p)
	if n > 0 {
		mrw.metrics.AddIngressBytes(uint64(n))
	}
	return n, err
}

// Write writes p using the underlying io.ReadWriter and adds n in the egress metrics.
func (mrw *MetricsReadWriter) Write(p []byte) (n int, err error) {
	n, err = mrw.rw.Write(p)
	if n > 0 {
		mrw.metrics.AddEgressBytes(uint64(n))
	}
	return n, err
}

// Metrics represents data ingress/egress metrics for a network connection.
type Metrics struct {
	// IngressBytes represents the total bytes received from external connections (ingress).
	IngressBytes uint64
	// EgressBytes represents the total outgoing total bytes (egressactive ).
	EgressBytes uint64
	// ConnectionCount specifies the total number of connections.
	ConnectionCount uint64
	// StartTime represents the time stamp when the metrics started.
	StartTime time.Time
	// ActiveConnections represents current active connections.
	ActiveConnections int32
}

// NewMetrics creates a new Metrics instance.
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

// GetUptime returns how long metrics have been collected.
func (m *Metrics) GetUptime() time.Duration {
	return time.Since(m.StartTime)
}

// GetIngressRate return bytes per second since start.
func (m *Metrics) GetIngressRate() float64 {
	uptime := m.GetUptime()
	if uptime == 0 {
		return 0
	}
	return float64(m.GetIngressBytes()) / uptime.Seconds()
}

// GetEgressRate return bytes per second since start.
func (m *Metrics) GetEgressRate() float64 {
	uptime := m.GetUptime()
	if uptime == 0 {
		return 0
	}

	return float64(m.GetEgressBytes()) / uptime.Seconds()
}

// NewProxyReadWriter return a new MetricsReadWriter using the underlying metrics.
func (m *Metrics) NewProxyReadWriter(rw io.ReadWriter) *MetricsReadWriter {
	return &MetricsReadWriter{
		rw:      rw,
		metrics: m,
	}
}
