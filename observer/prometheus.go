package observer

import (
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

// PrometheusObserver implements Observer with Prometheus metrics.
type PrometheusObserver struct {
	// Tunnel lifecycle metrics.
	tunnelsActive  prometheus.Gauge
	tunnelsCreated *prometheus.CounterVec
	tunnelsClosed  *prometheus.CounterVec
	tunnelDuration *prometheus.HistogramVec

	// Connection metrics per tunnel.
	connectionsTotal   *prometheus.CounterVec
	connectionsActive  *prometheus.GaugeVec
	connectionDuration *prometheus.HistogramVec

	// Traffic metrics per tunnel.
	bytesIngress *prometheus.CounterVec
	bytesEgress  *prometheus.CounterVec

	// HTTP-specific metrics per tunnel.
	httpRequestsTotal   *prometheus.CounterVec
	httpRequestDuration *prometheus.HistogramVec

	// Performance metrics per tunnel.
	rtt *prometheus.GaugeVec
}

// NewPrometheusObserver creates a new Prometheus-based observer.
func NewPrometheusObserver(registry prometheus.Registerer) *PrometheusObserver {
	if registry == nil {
		registry = prometheus.DefaultRegisterer
	}

	factory := promauto.With(registry)

	return &PrometheusObserver{
		tunnelsActive: factory.NewGauge(prometheus.GaugeOpts{
			Name: "wormhole_tunnels_active",
			Help: "Number of currently active tunnels",
		}),
		tunnelsCreated: factory.NewCounterVec(prometheus.CounterOpts{
			Name: "wormhole_tunnels_created_total",
			Help: "Total number of tunnels created",
		}, []string{"protocol"}),
		tunnelsClosed: factory.NewCounterVec(prometheus.CounterOpts{
			Name: "wormhole_tunnels_closed_total",
			Help: "Total number of tunnels closed",
		}, []string{"reason"}),
		tunnelDuration: factory.NewHistogramVec(prometheus.HistogramOpts{
			Name:    "wormhole_tunnel_duration_seconds",
			Help:    "Duration of tunnel sessions",
			Buckets: prometheus.ExponentialBuckets(1, 2, 12), // 1s to ~1h
		}, []string{"protocol"}),

		connectionsTotal: factory.NewCounterVec(prometheus.CounterOpts{
			Name: "wormhole_connections_total",
			Help: "Total number of connections handled per tunnel",
		}, []string{"domain", "protocol"}),
		connectionsActive: factory.NewGaugeVec(prometheus.GaugeOpts{
			Name: "wormhole_connections_active",
			Help: "Number of currently active connections per tunnel",
		}, []string{"domain"}),
		connectionDuration: factory.NewHistogramVec(prometheus.HistogramOpts{
			Name:    "wormhole_connection_duration_seconds",
			Help:    "Duration of individual connections per tunnel",
			Buckets: prometheus.ExponentialBuckets(0.001, 2, 15), // 1ms to ~16s
		}, []string{"domain", "protocol"}),

		bytesIngress: factory.NewCounterVec(prometheus.CounterOpts{
			Name: "wormhole_bytes_ingress_total",
			Help: "Total bytes received from external clients per tunnel",
		}, []string{"domain"}),
		bytesEgress: factory.NewCounterVec(prometheus.CounterOpts{
			Name: "wormhole_bytes_egress_total",
			Help: "Total bytes sent to external clients per tunnel",
		}, []string{"domain"}),

		httpRequestsTotal: factory.NewCounterVec(prometheus.CounterOpts{
			Name: "wormhole_http_requests_total",
			Help: "Total HTTP requests processed per tunnel",
		}, []string{"domain", "method", "status_code"}),
		httpRequestDuration: factory.NewHistogramVec(prometheus.HistogramOpts{
			Name:    "wormhole_http_request_duration_seconds",
			Help:    "HTTP request duration per tunnel",
			Buckets: prometheus.DefBuckets,
		}, []string{"domain", "method"}),

		rtt: factory.NewGaugeVec(prometheus.GaugeOpts{
			Name: "wormhole_rtt_microseconds",
			Help: "Round-trip time to client in microseconds per tunnel",
		}, []string{"domain"}),
	}
}

func (p *PrometheusObserver) RecordTunnelCreated(protocol string) {
	p.tunnelsActive.Inc()
	p.tunnelsCreated.WithLabelValues(protocol).Inc()
}

func (p *PrometheusObserver) RecordTunnelClosed(protocol, reason string, duration time.Duration) {
	p.tunnelsActive.Dec()
	p.tunnelsClosed.WithLabelValues(reason).Inc()
	p.tunnelDuration.WithLabelValues(protocol).Observe(duration.Seconds())
}

func (p *PrometheusObserver) RecordConnectionStart(domain, protocol string) {
	p.connectionsTotal.WithLabelValues(domain, protocol).Inc()
	p.connectionsActive.WithLabelValues(domain).Inc()
}

func (p *PrometheusObserver) RecordConnectionEnd(domain, protocol string, duration time.Duration) {
	p.connectionsActive.WithLabelValues(domain).Dec()
	p.connectionDuration.WithLabelValues(domain, protocol).Observe(duration.Seconds())
}

func (p *PrometheusObserver) RecordTraffic(domain string, ingress, egress uint64) {
	if ingress > 0 {
		p.bytesIngress.WithLabelValues(domain).Add(float64(ingress))
	}
	if egress > 0 {
		p.bytesEgress.WithLabelValues(domain).Add(float64(egress))
	}
}

func (p *PrometheusObserver) RecordHTTPRequest(domain, method, statusCode string, duration time.Duration) {
	p.httpRequestsTotal.WithLabelValues(domain, method, statusCode).Inc()
	p.httpRequestDuration.WithLabelValues(domain, method).Observe(duration.Seconds())
}

func (p *PrometheusObserver) UpdateRTT(domain string, rttMicroseconds uint32) {
	p.rtt.WithLabelValues(domain).Set(float64(rttMicroseconds))
}
