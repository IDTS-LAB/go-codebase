package events

import (
	"context"

	"github.com/IDTS-LAB/go-codebase/internal/core/domain"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
)

// LoggingEventBus wraps an EventBus and logs any errors returned by Publish.
// It ensures event publish failures are never silently lost.
type LoggingEventBus struct {
	inner EventBus
	log   domain.Logger
}

// NewLoggingEventBus creates a new LoggingEventBus wrapping the provided EventBus.
func NewLoggingEventBus(inner EventBus, log domain.Logger) EventBus {
	return &LoggingEventBus{inner: inner, log: log}
}

// Publish delegates to the wrapped EventBus and logs any error.
func (b *LoggingEventBus) Publish(ctx context.Context, event Event) error {
	err := b.inner.Publish(ctx, event)
	if err != nil {
		if span := trace.SpanFromContext(ctx); span.IsRecording() {
			span.SetStatus(codes.Error, "event publish failed")
			span.RecordError(err)
		}
		b.log.Error(ctx, "event publish failed",
			domain.String("event_type", event.Type),
			domain.Error(err),
		)
	}
	return err
}

// Subscribe delegates to the wrapped EventBus.
func (b *LoggingEventBus) Subscribe(eventType string, handler Handler) {
	b.inner.Subscribe(eventType, handler)
}
