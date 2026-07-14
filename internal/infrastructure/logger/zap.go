package logger

import (
	"context"
	"os"

	"github.com/IDTS-LAB/go-codebase/internal/core/domain"
	"github.com/IDTS-LAB/go-codebase/internal/shared/config"
	"go.opentelemetry.io/otel/trace"
	"go.uber.org/fx"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

var Module = fx.Module("zap",
	fx.Provide(NewZapLogger),
	fx.Provide(NewZapStdLogger),
)

type ZapLogger struct {
	logger *zap.Logger
}

func NewZapStdLogger(log domain.Logger) *zap.Logger {
	if z, ok := log.(*ZapLogger); ok {
		return z.logger
	}
	return zap.NewNop()
}

func NewZapLogger(cfg *config.Config) (domain.Logger, error) {
	level := zap.InfoLevel
	if err := level.UnmarshalText([]byte(cfg.Log.Level)); err != nil {
		level = zap.InfoLevel
	}

	var zapCfg zap.Config
	if cfg.App.Env == "development" {
		zapCfg = zap.NewDevelopmentConfig()
	} else {
		zapCfg = zap.NewProductionConfig()
	}

	if cfg.Log.Format == "json" {
		zapCfg.Encoding = "json"
	}

	zapCfg.Level = zap.NewAtomicLevelAt(level)
	zapCfg.EncoderConfig.EncodeTime = zapcore.ISO8601TimeEncoder
	zapCfg.EncoderConfig.EncodeLevel = zapcore.CapitalLevelEncoder
	zapCfg.OutputPaths = []string{"stdout"}
	zapCfg.ErrorOutputPaths = []string{"stderr"}

	if cfg.Log.Format == "console" {
		zapCfg.EncoderConfig.EncodeLevel = zapcore.CapitalColorLevelEncoder
	}

	zapLogger, err := zapCfg.Build(zap.AddCallerSkip(0))
	if err != nil {
		return nil, err
	}

	_ = os.Setenv("TZ", "UTC")

	return &ZapLogger{logger: zapLogger}, nil
}

func (l *ZapLogger) Debug(ctx context.Context, msg string, fields ...domain.Field) {
	l.logger.Debug(msg, appendTraceFields(ctx, toZapFields(fields))...)
}

func (l *ZapLogger) Info(ctx context.Context, msg string, fields ...domain.Field) {
	l.logger.Info(msg, appendTraceFields(ctx, toZapFields(fields))...)
}

func (l *ZapLogger) Warn(ctx context.Context, msg string, fields ...domain.Field) {
	l.logger.Warn(msg, appendTraceFields(ctx, toZapFields(fields))...)
}

func (l *ZapLogger) Error(ctx context.Context, msg string, fields ...domain.Field) {
	l.logger.Error(msg, appendTraceFields(ctx, toZapFields(fields))...)
}

func (l *ZapLogger) Fatal(ctx context.Context, msg string, fields ...domain.Field) {
	l.logger.Fatal(msg, appendTraceFields(ctx, toZapFields(fields))...)
}

func (l *ZapLogger) With(fields ...domain.Field) domain.Logger {
	return &ZapLogger{logger: l.logger.With(toZapFields(fields)...)}
}

func appendTraceFields(ctx context.Context, fields []zap.Field) []zap.Field {
	span := trace.SpanFromContext(ctx)
	if !span.IsRecording() {
		return fields
	}
	spanContext := span.SpanContext()
	if spanContext.HasTraceID() {
		fields = append(fields, zap.String("trace_id", spanContext.TraceID().String()))
	}
	if spanContext.HasSpanID() {
		fields = append(fields, zap.String("span_id", spanContext.SpanID().String()))
	}
	return fields
}

func toZapFields(fields []domain.Field) []zap.Field {
	zapFields := make([]zap.Field, len(fields))
	for i, f := range fields {
		switch v := f.Value.(type) {
		case string:
			zapFields[i] = zap.String(f.Key, v)
		case int:
			zapFields[i] = zap.Int(f.Key, v)
		case error:
			zapFields[i] = zap.Error(v)
		case zap.Field:
			zapFields[i] = v
		default:
			zapFields[i] = zap.Any(f.Key, v)
		}
	}
	return zapFields
}
