package telemetry

import (
	"context"

	"github.com/IDTS-LAB/go-codebase/internal/shared/config"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracehttp"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.24.0"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

var Module = fx.Module("telemetry", fx.Provide(NewTracerProvider))

func NewTracerProvider(cfg *config.Config, log *zap.Logger) (*sdktrace.TracerProvider, error) {
	if cfg.Telemetry.ExporterEndpoint == "" {
		log.Warn("telemetry exporter endpoint not configured, tracing disabled")
		tp := sdktrace.NewTracerProvider()
		otel.SetTracerProvider(tp)
		return tp, nil
	}

	ctx := context.Background()

	exporter, err := otlptracehttp.New(ctx,
		otlptracehttp.WithEndpoint(cfg.Telemetry.ExporterEndpoint),
		otlptracehttp.WithInsecure(),
	)
	if err != nil {
		log.Warn("failed to create trace exporter", zap.Error(err))
		tp := sdktrace.NewTracerProvider()
		otel.SetTracerProvider(tp)
		return tp, nil
	}

	res, err := resource.New(ctx,
		resource.WithAttributes(
			semconv.ServiceNameKey.String(cfg.Telemetry.ServiceName),
			attribute.String("environment", "production"),
		),
	)
	if err != nil {
		log.Warn("failed to create resource", zap.Error(err))
	}

	tp := sdktrace.NewTracerProvider(
		sdktrace.WithBatcher(exporter),
		sdktrace.WithResource(res),
		sdktrace.WithSampler(sdktrace.ParentBased(sdktrace.TraceIDRatioBased(cfg.Telemetry.SampleRate))),
	)

	otel.SetTracerProvider(tp)
	log.Info("telemetry initialized",
		zap.String("service", cfg.Telemetry.ServiceName),
		zap.String("endpoint", cfg.Telemetry.ExporterEndpoint),
	)

	return tp, nil
}
