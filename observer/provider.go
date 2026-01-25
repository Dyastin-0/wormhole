package observer

import (
	"context"
	"time"

	"go.opentelemetry.io/otel/exporters/otlp/otlpmetric/otlpmetricgrpc"
	"go.opentelemetry.io/otel/exporters/prometheus"
	"go.opentelemetry.io/otel/sdk/metric"
)

// NewMeterProviderWithPromExporter returns a MeterProvider with prometheus exporter.
func NewMeterProviderWithPromExporter(ctx context.Context) (*metric.MeterProvider, error) {
	exporter, err := prometheus.New()
	if err != nil {
		return nil, err
	}

	provider := metric.NewMeterProvider(
		metric.WithReader(exporter),
	)

	return provider, nil
}

// NewMeterProvider returns a MeterProvider without a reader.
func NewMeterProvider(ctx context.Context, endpoint string) (*metric.MeterProvider, error) {
	exporter, err := otlpmetricgrpc.New(ctx,
		otlpmetricgrpc.WithEndpoint(endpoint),
		otlpmetricgrpc.WithInsecure(),
	)
	if err != nil {
		return nil, err
	}

	provider := metric.NewMeterProvider(
		metric.WithReader(metric.NewPeriodicReader(exporter,
			metric.WithInterval(1*time.Second),
		)),
	)

	return provider, nil
}
