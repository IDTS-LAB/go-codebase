package domain

import "context"

type MetricsRecorder interface {
	IncrementCounter(ctx context.Context, name string, labels ...string)
	ObserveHistogram(ctx context.Context, name string, value float64, labels ...string)
	SetGauge(ctx context.Context, name string, value float64, labels ...string)
}
