// Package observer provides OpenTelemetry metrics for the Wormhole server.
package observer

import (
	"context"
	"time"

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"
)

// OTelObserver implements Observer with OpenTelemetry metrics.
type OTelObserver struct {
	// Tunnel lifecycle metrics.
	tunnelsActive  metric.Int64UpDownCounter
	tunnelsCreated metric.Int64Counter
	tunnelsClosed  metric.Int64Counter
	tunnelDuration metric.Float64Histogram

	// Connection metrics per tunnel.
	connectionsTotal   metric.Int64Counter
	connectionsActive  metric.Int64UpDownCounter
	connectionDuration metric.Float64Histogram

	// Traffic metrics per tunnel.
	bytesIngress metric.Int64Counter
	bytesEgress  metric.Int64Counter

	// HTTP-specific metrics per tunnel.
	httpRequestsTotal   metric.Int64Counter
	httpRequestDuration metric.Float64Histogram

	// Performance metrics per tunnel.
	rtt metric.Int64Gauge
}

// NewOTelObserver creates a new OpenTelemetry-based observer.
func NewOTelObserver(mp metric.MeterProvider) (*OTelObserver, error) {
	meter := mp.Meter("wormhole")

	tunnelsActive, err := meter.Int64UpDownCounter(
		"wormhole.tunnels.active",
		metric.WithDescription("Number of currently active tunnels"),
	)
	if err != nil {
		return nil, err
	}

	tunnelsCreated, err := meter.Int64Counter(
		"wormhole.tunnels.created",
		metric.WithDescription("Total number of tunnels created"),
	)
	if err != nil {
		return nil, err
	}

	tunnelsClosed, err := meter.Int64Counter(
		"wormhole.tunnels.closed",
		metric.WithDescription("Total number of tunnels closed"),
	)
	if err != nil {
		return nil, err
	}

	tunnelDuration, err := meter.Float64Histogram(
		"wormhole.tunnel.duration",
		metric.WithDescription("Duration of tunnel sessions"),
		metric.WithUnit("s"),
		metric.WithExplicitBucketBoundaries(1, 2, 4, 8, 16, 32, 64, 128, 256, 512, 1024, 2048, 4096), // 1s to ~1h
	)
	if err != nil {
		return nil, err
	}

	connectionsTotal, err := meter.Int64Counter(
		"wormhole.connections.total",
		metric.WithDescription("Total number of connections handled per tunnel"),
	)
	if err != nil {
		return nil, err
	}

	connectionsActive, err := meter.Int64UpDownCounter(
		"wormhole.connections.active",
		metric.WithDescription("Number of currently active connections per tunnel"),
	)
	if err != nil {
		return nil, err
	}

	connectionDuration, err := meter.Float64Histogram(
		"wormhole.connection.duration",
		metric.WithDescription("Duration of individual connections per tunnel"),
		metric.WithUnit("s"),
		metric.WithExplicitBucketBoundaries(0.001, 0.002, 0.004, 0.008, 0.016, 0.032, 0.064, 0.128, 0.256, 0.512, 1.024, 2.048, 4.096, 8.192, 16.384), // 1ms to ~16s
	)
	if err != nil {
		return nil, err
	}

	bytesIngress, err := meter.Int64Counter(
		"wormhole.bytes.ingress",
		metric.WithDescription("Total bytes received from external clients per tunnel"),
		metric.WithUnit("By"),
	)
	if err != nil {
		return nil, err
	}

	bytesEgress, err := meter.Int64Counter(
		"wormhole.bytes.egress",
		metric.WithDescription("Total bytes sent to external clients per tunnel"),
		metric.WithUnit("By"),
	)
	if err != nil {
		return nil, err
	}

	httpRequestsTotal, err := meter.Int64Counter(
		"wormhole.http.requests.total",
		metric.WithDescription("Total HTTP requests processed per tunnel"),
	)
	if err != nil {
		return nil, err
	}

	httpRequestDuration, err := meter.Float64Histogram(
		"wormhole.http.request.duration",
		metric.WithDescription("HTTP request duration per tunnel"),
		metric.WithUnit("s"),
	)
	if err != nil {
		return nil, err
	}

	rtt, err := meter.Int64Gauge(
		"wormhole.rtt",
		metric.WithDescription("Round-trip time to client in microseconds per tunnel"),
		metric.WithUnit("us"),
	)
	if err != nil {
		return nil, err
	}

	return &OTelObserver{
		tunnelsActive:       tunnelsActive,
		tunnelsCreated:      tunnelsCreated,
		tunnelsClosed:       tunnelsClosed,
		tunnelDuration:      tunnelDuration,
		connectionsTotal:    connectionsTotal,
		connectionsActive:   connectionsActive,
		connectionDuration:  connectionDuration,
		bytesIngress:        bytesIngress,
		bytesEgress:         bytesEgress,
		httpRequestsTotal:   httpRequestsTotal,
		httpRequestDuration: httpRequestDuration,
		rtt:                 rtt,
	}, nil
}

func (o *OTelObserver) RecordTunnelCreated(ctx context.Context, protocol string) {
	o.tunnelsActive.Add(ctx, 1)
	o.tunnelsCreated.Add(ctx, 1, metric.WithAttributes(
		attribute.String("protocol", protocol),
	))
}

func (o *OTelObserver) RecordTunnelClosed(ctx context.Context, protocol, reason string, duration time.Duration) {
	o.tunnelsActive.Add(ctx, -1)
	o.tunnelsClosed.Add(ctx, 1, metric.WithAttributes(
		attribute.String("reason", reason),
	))
	o.tunnelDuration.Record(ctx, duration.Seconds(), metric.WithAttributes(
		attribute.String("protocol", protocol),
	))
}

func (o *OTelObserver) RecordConnectionStart(ctx context.Context, domain, protocol string) {
	o.connectionsTotal.Add(ctx, 1, metric.WithAttributes(
		attribute.String("domain", domain),
		attribute.String("protocol", protocol),
	))
	o.connectionsActive.Add(ctx, 1, metric.WithAttributes(
		attribute.String("domain", domain),
	))
}

func (o *OTelObserver) RecordConnectionEnd(ctx context.Context, domain, protocol string, duration time.Duration) {
	o.connectionsActive.Add(ctx, -1, metric.WithAttributes(
		attribute.String("domain", domain),
	))
	o.connectionDuration.Record(ctx, duration.Seconds(), metric.WithAttributes(
		attribute.String("domain", domain),
		attribute.String("protocol", protocol),
	))
}

func (o *OTelObserver) RecordTraffic(ctx context.Context, domain string, ingress, egress uint64) {
	attrs := metric.WithAttributes(attribute.String("domain", domain))

	if ingress > 0 {
		o.bytesIngress.Add(ctx, int64(ingress), attrs)
	}
	if egress > 0 {
		o.bytesEgress.Add(ctx, int64(egress), attrs)
	}
}

func (o *OTelObserver) RecordHTTPRequest(ctx context.Context, domain, method, statusCode string, duration time.Duration) {
	attrs := metric.WithAttributes(
		attribute.String("domain", domain),
		attribute.String("method", method),
		attribute.String("status_code", statusCode),
	)
	o.httpRequestsTotal.Add(ctx, 1, attrs)

	o.httpRequestDuration.Record(ctx, duration.Seconds(), metric.WithAttributes(
		attribute.String("domain", domain),
		attribute.String("method", method),
	))
}

func (o *OTelObserver) UpdateRTT(ctx context.Context, domain string, rttMicroseconds uint32) {
	o.rtt.Record(ctx, int64(rttMicroseconds), metric.WithAttributes(
		attribute.String("domain", domain),
	))
}

