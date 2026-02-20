package proto

import (
	"bytes"
	"encoding/binary"
	"fmt"
)

// MetricsSize is the fixed size of a metrics' fields in bytes.
const MetricsSize = 40

// Metrics represents the tunnel's incoming and outgoing bytes metrics.
type Metrics struct {
	// Ingress represents the incoming bytes.
	Ingress uint64
	// Egress represents the outgoing bytes.
	Egress uint64
	// Uptime represents the time elapsed since tunnel started in milliseconds.
	Uptime uint64
	// ConnectionCount specifies the total number of connections.
	ConnectionCount uint64
	// ActiveConnections represents current active connections.
	ActiveConnections uint32
	// RTT represents the round-trip time in microseconds.
	RTT uint32
}

// NewMetrics creates a new Metrics with the specified values.
func NewMetrics(ingress, egress, uptime, connectionCount uint64, activeConnections, rtt uint32) *Metrics {
	return &Metrics{
		Ingress:           ingress,
		Egress:            egress,
		Uptime:            uptime,
		ConnectionCount:   connectionCount,
		ActiveConnections: activeConnections,
		RTT:               rtt,
	}
}

// SerializeMetrics serializes Metrics into byte slice using a pooled buffer.
//
// Wire format:
//
//	[8] Ingress 						(uint64, big-endian)
//	[8] Egress 							(uint64, big-endian)
//	[8] Uptime 							(uint64, big-endian)
//	[8] Connection count		(uint64, big-endian)
//	[4] Active connections	(uint32, big-endian)
//	[4] RTT 								(uint32, big-endian)
func SerializeMetrics(metrics *Metrics) ([]byte, error) {
	bufPtr := metricsBufferPool.Get().(*[]byte)
	defer metricsBufferPool.Put(bufPtr)

	*bufPtr = (*bufPtr)[:0]
	buf := bytes.NewBuffer(*bufPtr)

	if err := binary.Write(buf, binary.BigEndian, metrics.Ingress); err != nil {
		return nil, fmt.Errorf("failed to write ingress: %w", err)
	}
	if err := binary.Write(buf, binary.BigEndian, metrics.Egress); err != nil {
		return nil, fmt.Errorf("failed to write egress: %w", err)
	}
	if err := binary.Write(buf, binary.BigEndian, metrics.Uptime); err != nil {
		return nil, fmt.Errorf("failed to write uptime: %w", err)
	}
	if err := binary.Write(buf, binary.BigEndian, metrics.ConnectionCount); err != nil {
		return nil, fmt.Errorf("failed to write connection count: %w", err)
	}
	if err := binary.Write(buf, binary.BigEndian, metrics.ActiveConnections); err != nil {
		return nil, fmt.Errorf("failed to write active connections: %w", err)
	}
	if err := binary.Write(buf, binary.BigEndian, metrics.RTT); err != nil {
		return nil, fmt.Errorf("failed to write rtt: %w", err)
	}

	result := make([]byte, buf.Len())
	copy(result, buf.Bytes())
	return result, nil
}

// DeserializeMetrics deserializes a byte slice into Metrics.
func DeserializeMetrics(data []byte) (*Metrics, error) {
	if len(data) < int(MetricsSize) {
		return nil, ErrInvalidMetricsSize
	}

	reader := bytes.NewReader(data[:MetricsSize])
	var metrics Metrics

	if err := binary.Read(reader, binary.BigEndian, &metrics); err != nil {
		return nil, fmt.Errorf("failed to read metrics: %w", err)
	}

	return &metrics, nil
}
