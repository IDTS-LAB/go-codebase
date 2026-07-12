package telemetry

import (
	"context"
	"fmt"

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
)

// RecordError records an application error on the current span.
func RecordError(ctx context.Context, err error) {
	if err == nil {
		return
	}

	span := trace.SpanFromContext(ctx)
	if !span.IsRecording() {
		return
	}

	span.RecordError(err)
	span.SetStatus(codes.Error, err.Error())

	span.AddEvent("exception",
		trace.WithAttributes(
			attribute.String("exception.type", fmt.Sprintf("%T", err)),
			attribute.String("exception.message", err.Error()),
		),
	)
}

// RecordPanic records a recovered panic together with its stack trace.
func RecordPanic(ctx context.Context, recovered any, stack string) {
	span := trace.SpanFromContext(ctx)
	if !span.IsRecording() {
		return
	}

	err := fmt.Errorf("%v", recovered)

	span.RecordError(err)

	span.SetStatus(codes.Error, "panic")

	span.AddEvent("exception",
		trace.WithAttributes(
			attribute.String("exception.type", "panic"),
			attribute.String("exception.message", err.Error()),
			attribute.String("exception.stacktrace", stack),
		),
	)
}
