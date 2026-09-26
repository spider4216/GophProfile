package meter

import (
	"context"
	"fmt"
	"os"
	"time"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/exporters/otlp/otlpmetric/otlpmetrichttp"
	ometric "go.opentelemetry.io/otel/metric"
	"go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/resource"
	semconv "go.opentelemetry.io/otel/semconv/v1.37.0"
)

type Meter struct {
	cli ometric.Meter
}

func NewMeter() *Meter {
	return &Meter{}
}

func (m *Meter) Init(ctx context.Context, serviceName string, metricName string) (func(), error) {
	// Создаём OTel Exporter
	exporter, err := otlpmetrichttp.New(
		ctx,
		otlpmetrichttp.WithInsecure(),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create OTLP exporter: %w", err)
	}

	// Добавляем метаинформацию о сервисе
	res, err := resource.New(ctx,
		resource.WithFromEnv(),
		resource.WithProcess(),
		resource.WithTelemetrySDK(),
		resource.WithHost(),
		resource.WithOS(),
		resource.WithAttributes(
			semconv.ServiceNameKey.String(serviceName),
			attribute.String("environment", os.Getenv("GO_ENV")),
		),
	)
	if err != nil {
		return nil, fmt.Errorf("cannot create metric resource: %w", err)
	}

	// Инициализируем MeterProvider
	meterProvider := metric.NewMeterProvider(
		metric.WithResource(res),
		metric.WithReader(
			metric.NewPeriodicReader(
				exporter,
				metric.WithInterval(2*time.Second),
			),
		),
	)
	otel.SetMeterProvider(meterProvider)

	m.cli = meterProvider.Meter(metricName)

	return func() {
		ctx, cancel := context.WithTimeout(ctx, time.Second)
		defer cancel()

		if err := meterProvider.Shutdown(ctx); err != nil {
			otel.Handle(err)
		}
	}, nil
}

// name - пример tasks_processed_total
// desc - описание задачи
// t - тип таска, например "upload"
func (m *Meter) Count(ctx context.Context, name string, desc string, t string) error {
	tasksCounter, err := m.cli.Int64Counter(
		name,
		ometric.WithDescription(desc),
	)
	if err != nil {
		return fmt.Errorf("cannot create counter: %w", err)
	}

	tasksCounter.Add(ctx, 1, ometric.WithAttributes(
		attribute.String("task_type", t),
	))

	return nil
}
