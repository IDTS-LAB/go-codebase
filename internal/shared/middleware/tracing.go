package middleware

import (
	"fmt"
	"net/http"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/trace"
)

const tracerName = "github.com/IDTS-LAB/go-codebase/internal/shared/middleware"

// Tracing starts a new OpenTelemetry span for each HTTP request, propagates
// incoming trace context, and records the response status on the span.
func Tracing() func(http.Handler) http.Handler {
	tracer := otel.Tracer(tracerName)
	propagator := otel.GetTextMapPropagator()

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ctx := propagator.Extract(r.Context(), propagation.HeaderCarrier(r.Header))

			ctx, span := tracer.Start(ctx, fmt.Sprintf("%s %s", r.Method, r.URL.Path),
				trace.WithAttributes(
					attribute.String("http.method", r.Method),
					attribute.String("http.path", r.URL.Path),
					attribute.String("http.route", r.URL.Path),
					attribute.String("http.target", r.URL.String()),
					attribute.String("http.scheme", r.URL.Scheme),
					attribute.String("http.flavor", r.Proto),
					attribute.String("net.host.name", r.Host),
				),
			)
			defer span.End()

			wrapped := &responseWriter{ResponseWriter: w, statusCode: http.StatusOK}
			next.ServeHTTP(wrapped, r.WithContext(ctx))

			status := wrapped.statusCode
			span.SetAttributes(attribute.Int("http.status_code", status))
			if status >= 500 {
				span.SetStatus(codes.Error, fmt.Sprintf("server error: %d", status))
			} else if status >= 400 {
				span.SetStatus(codes.Error, fmt.Sprintf("client error: %d", status))
			}
		})
	}
}
