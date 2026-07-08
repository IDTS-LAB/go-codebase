package main

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/IDTS-LAB/go-codebase/docs"
	"github.com/IDTS-LAB/go-codebase/internal/authentication"
	"github.com/IDTS-LAB/go-codebase/internal/authentication/application/service"
	"github.com/IDTS-LAB/go-codebase/internal/authorization"
	"github.com/IDTS-LAB/go-codebase/internal/authorization/infrastructure/casbin"
	"github.com/IDTS-LAB/go-codebase/internal/core/domain"
	"github.com/IDTS-LAB/go-codebase/internal/infrastructure/auth"
	"github.com/IDTS-LAB/go-codebase/internal/infrastructure/cache"
	"github.com/IDTS-LAB/go-codebase/internal/infrastructure/logger"
	"github.com/IDTS-LAB/go-codebase/internal/infrastructure/messaging"
	"github.com/IDTS-LAB/go-codebase/internal/shared/auditlog"
	"github.com/IDTS-LAB/go-codebase/internal/shared/config"
	"github.com/IDTS-LAB/go-codebase/internal/shared/database"
	"github.com/IDTS-LAB/go-codebase/internal/shared/middleware"
	"github.com/IDTS-LAB/go-codebase/internal/shared/telemetry"
	"github.com/IDTS-LAB/go-codebase/internal/todo"
	"github.com/go-chi/chi/v5"
	"github.com/redis/go-redis/v9"
	httpSwagger "github.com/swaggo/http-swagger"
	"go.uber.org/fx"
)

func main() {
	cfg, err := config.New()
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to load config: %v\n", err)
		os.Exit(1)
	}

	app := fx.New(
		fx.Supply(cfg),

		// Infrastructure
		logger.Module,
		cache.Module,
		auth.Module,
		messaging.Module,
		database.Module,
		telemetry.Module,

		// Modules
		authentication.Module,
		authorization.Module,
		todo.Module,

		// Audit log repository
		fx.Provide(auditlog.NewRepository),

		// Middleware providers
		fx.Provide(func(tokenSvc domain.TokenService, rdb *redis.Client, cfg *config.Config) func(http.Handler) http.Handler {
			if cfg.Auth.TokenDenylist {
				return middleware.AuthenticationWithDenylist(tokenSvc, func(ctx context.Context, jti string) bool {
					val, err := rdb.Get(ctx, "token:blacklist:"+jti).Result()
					return err == nil && val == "1"
				})
			}
			return middleware.Authentication(tokenSvc)
		}),
		fx.Provide(func(log domain.Logger) func(http.Handler) http.Handler {
			return middleware.Logger(log)
		}),
		fx.Provide(func(log domain.Logger, errorRepo *auditlog.Repository) func(http.Handler) http.Handler {
			return middleware.ErrorHandler(log, errorRepo)
		}),
		fx.Provide(func(errorRepo *auditlog.Repository) func(http.Handler) http.Handler {
			return middleware.ErrorRecorder(nil, errorRepo)
		}),
		fx.Provide(func(errorRepo *auditlog.Repository) func(http.Handler) http.Handler {
			return middleware.AuditLog(errorRepo)
		}),
		fx.Provide(func(rdb *redis.Client, cfg *config.Config) func(http.Handler) http.Handler {
			return middleware.RateLimit(rdb, cfg.RateLimit.Requests, time.Duration(cfg.RateLimit.Window)*time.Second)
		}),
		fx.Provide(func(rdb *redis.Client, cfg *config.Config) func(http.Handler) http.Handler {
			return middleware.Idempotency(rdb, time.Duration(cfg.Idempotency.TTL)*time.Second)
		}),
		fx.Provide(func(cfg *config.Config) func(http.Handler) http.Handler {
			return middleware.MaxBodySize(int64(cfg.Server.MaxRequestBodySize))
		}),

		// Authorization middleware
		fx.Provide(func(enf *casbin.Enforcer) middleware.Authorizer {
			return enf
		}),

		// Denylist helper for auth service
		fx.Invoke(func(authSvc *service.AuthenticationService, rdb *redis.Client, cfg *config.Config) {
			if cfg.Auth.TokenDenylist {
				authSvc.SetDenylist(func(ctx context.Context, jti string, ttl time.Duration) error {
					return rdb.Set(ctx, "token:blacklist:"+jti, "1", ttl).Err()
				})
			}
			authSvc.SetLockoutConfig(cfg.Auth.MaxLoginAttempts, time.Duration(cfg.Auth.LockoutDuration)*time.Second)
		}),

		// Root router
		fx.Provide(newRootRouter),

		fx.Invoke(startServer),
	)

	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer cancel()

	if err := app.Start(ctx); err != nil {
		fmt.Fprintf(os.Stderr, "failed to start app: %v\n", err)
		os.Exit(1)
	}

	<-ctx.Done()

	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 15)
	defer shutdownCancel()

	if err := app.Stop(shutdownCtx); err != nil {
		fmt.Fprintf(os.Stderr, "failed to stop app: %v\n", err)
		os.Exit(1)
	}
}

func newRootRouter(
	todoRouter *chi.Mux,
	authRouter *chi.Mux,
	authzRouter *chi.Mux,
	authMiddleware func(http.Handler) http.Handler,
	log domain.Logger,
	errorHandler func(http.Handler) http.Handler,
	errorRecorder func(http.Handler) http.Handler,
	rateLimit func(http.Handler) http.Handler,
	maxBodySize func(http.Handler) http.Handler,
	idempotency func(http.Handler) http.Handler,
	auditLog func(http.Handler) http.Handler,
	cfg *config.Config,
) *chi.Mux {
	r := chi.NewRouter()

	// Global middleware
	r.Use(middleware.RequestID)
	r.Use(errorHandler)
	r.Use(middleware.SecurityHeaders)
	r.Use(middleware.CORS(&cfg.CORS))
	r.Use(errorRecorder)
	r.Use(auditLog)
	r.Use(rateLimit)
	r.Use(middleware.Logger(log))

	r.Get("/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"status":"ok"}`))
	})
	r.Get("/ready", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"status":"ready"}`))
	})
	r.Get("/live", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"status":"alive"}`))
	})

	// Swagger UI (not in production)
	if cfg.App.Env != "production" {
		docs.SwaggerInfo.BasePath = "/api/v1"
		r.Get("/swagger/*", httpSwagger.Handler(
			httpSwagger.URL("/swagger/doc.json"),
		))
	}

	r.Route("/api/v1", func(r chi.Router) {
		// Public auth routes
		r.Group(func(r chi.Router) {
			r.Use(maxBodySize)
			r.Use(idempotency)
			r.Mount("/auth", authRouter)
		})

		// Protected routes
		r.Group(func(r chi.Router) {
			r.Use(authMiddleware)
			r.Use(maxBodySize)
			r.Mount("/todos", todoRouter)
			r.Mount("/auth/sessions", authzRouter)
		})
	})

	return r
}

func startServer(lc fx.Lifecycle, cfg *config.Config, log domain.Logger, mux *chi.Mux) {
	var srv *http.Server
	lc.Append(fx.Hook{
		OnStart: func(ctx context.Context) error {
			addr := fmt.Sprintf(":%d", cfg.Server.Port)
			log.Info(ctx, "starting server", domain.String("addr", addr))
			srv = &http.Server{
				Addr:         addr,
				Handler:      mux,
				ReadTimeout:  time.Duration(cfg.Server.ReadTimeout) * time.Second,
				WriteTimeout: time.Duration(cfg.Server.WriteTimeout) * time.Second,
			}
			go func() {
				if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
					log.Fatal(ctx, "server failed", domain.Error(err))
				}
			}()
			return nil
		},
		OnStop: func(ctx context.Context) error {
			log.Info(ctx, "stopping server")
			return srv.Shutdown(ctx)
		},
	})
}
