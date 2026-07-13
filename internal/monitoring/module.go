package monitoring

import (
	"github.com/IDTS-LAB/go-codebase/internal/monitoring/domain"
	"github.com/IDTS-LAB/go-codebase/internal/monitoring/infrastructure/prometheus"
	metricsHttp "github.com/IDTS-LAB/go-codebase/internal/monitoring/interfaces/http"
	"go.uber.org/fx"
)

var Module = fx.Module("monitoring",
	fx.Provide(
		prometheus.NewRecorder,
		fx.Annotate(
			func(r *prometheus.Recorder) domain.MetricsRecorder { return r },
			fx.As(new(domain.MetricsRecorder)),
		),
		metricsHttp.NewMetricsHandler,
	),
)
