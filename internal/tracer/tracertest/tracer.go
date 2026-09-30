package tracertest

import (
	"context"

	"go.opentelemetry.io/otel/trace"
)

type Tracer struct{}

func NewTracer() *Tracer {
	return &Tracer{}
}

func (t *Tracer) Start(ctx context.Context, name string) (context.Context, trace.Span) {
	return ctx, trace.SpanFromContext(ctx)
}
