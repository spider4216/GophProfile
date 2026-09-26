package tracer

import (
	"context"
	"fmt"
	"time"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/trace"
)

type Tracer struct {
	provider *sdktrace.TracerProvider
	tracer   trace.Tracer
}

func NewTracer() *Tracer {
	return &Tracer{}
}

func (t *Tracer) Init(ctx context.Context, serviceName string) (func(), error) {
	// Создаём gRPC Exporter (порт 4317)
	exporter, err := otlptracegrpc.New(ctx)

	if err != nil {
		return nil, fmt.Errorf("failed to create OTLP trace exporter: %w", err)
	}

	// Метаинформация (Resource)
	res, err := resource.New(ctx,
		resource.WithFromEnv(),
		resource.WithTelemetrySDK(),
	)
	if err != nil {
		return nil, fmt.Errorf("failded to create resource: %w", err)
	}

	// Инициализируем TracerProvider
	tracerProvider := sdktrace.NewTracerProvider(
		sdktrace.WithResource(res),
		// Используем BatchSpanProcessor для эффективности (отправляет пачками)
		sdktrace.WithBatcher(exporter),
		// Sampler: AlwaysSample пишет 100% запросов (хорошо для дебага)
		sdktrace.WithSampler(sdktrace.AlwaysSample()),
	)

	t.provider = tracerProvider
	t.tracer = tracerProvider.Tracer(serviceName)

	// Регистрируем глобальный провайдер
	otel.SetTracerProvider(tracerProvider)

	// Настраиваем propagation контекста.
	// Это позволяет передавать TraceID в заголовках запросов к другим сервисам.
	otel.SetTextMapPropagator(propagation.NewCompositeTextMapPropagator(
		propagation.TraceContext{},
		propagation.Baggage{},
	))

	// Возвращаем функцию для корректного завершения (flush данных перед выходом)
	return func() {
		ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
		defer cancel()

		if err := tracerProvider.Shutdown(ctx); err != nil {
			otel.Handle(err)
		}
	}, nil
}

func (t *Tracer) Start(ctx context.Context, name string) (context.Context, trace.Span) {
	return t.tracer.Start(ctx, name)
}
