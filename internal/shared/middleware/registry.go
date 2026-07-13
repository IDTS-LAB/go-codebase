package middleware

import (
	"context"
	"net/http"
	"time"

	coredomain "github.com/IDTS-LAB/go-codebase/internal/core/domain"
	monitoringdomain "github.com/IDTS-LAB/go-codebase/internal/monitoring/domain"
	"github.com/IDTS-LAB/go-codebase/internal/shared/auditlog"
	"github.com/IDTS-LAB/go-codebase/internal/shared/config"
	"github.com/redis/go-redis/v9"
)

type Registry struct {
	Auth          func(http.Handler) http.Handler
	Logger        func(http.Handler) http.Handler
	ErrorHandler  func(http.Handler) http.Handler
	ErrorRecorder func(http.Handler) http.Handler
	AuditLog      func(http.Handler) http.Handler
	RateLimit     func(http.Handler) http.Handler
	Idempotency   func(http.Handler) http.Handler
	MaxBodySize   func(http.Handler) http.Handler
	Tracing       func(http.Handler) http.Handler
	Metrics       func(http.Handler) http.Handler
	Authorizer    Authorizer
}

func NewRegistry(
	tokenSvc coredomain.TokenService,
	rdb *redis.Client,
	cfg *config.Config,
	log coredomain.Logger,
	errorRepo *auditlog.Repository,
	authorizer Authorizer,
	metricsRecorder monitoringdomain.MetricsRecorder,
) Registry {
	auth := Authentication(tokenSvc)
	if cfg.Auth.TokenDenylist {
		auth = AuthenticationWithDenylist(tokenSvc, func(ctx context.Context, jti string) bool {
			val, err := rdb.Get(ctx, "token:blacklist:"+jti).Result()
			return err == nil && val == "1"
		})
	}
	return Registry{
		Auth:          auth,
		Logger:        Logger(log),
		ErrorHandler:  ErrorHandler(log, errorRepo),
		ErrorRecorder: ErrorRecorder(log, errorRepo),
		AuditLog:      AuditLog(errorRepo),
		RateLimit:     RateLimit(rdb, cfg.RateLimit.Requests, time.Duration(cfg.RateLimit.Window)*time.Second),
		Idempotency:   Idempotency(rdb, time.Duration(cfg.Idempotency.TTL)*time.Second),
		MaxBodySize:   MaxBodySize(int64(cfg.Server.MaxRequestBodySize)),
		Tracing:       Tracing(),
		Metrics:       Metrics(metricsRecorder),
		Authorizer:    authorizer,
	}
}
