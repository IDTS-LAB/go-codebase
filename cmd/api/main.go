package main

import (
	"context"
	"database/sql"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/IDTS-LAB/go-codebase/internal/authentication"
	authEventBus "github.com/IDTS-LAB/go-codebase/internal/authentication/infrastructure/eventbus"
	authHTTP "github.com/IDTS-LAB/go-codebase/internal/authentication/interfaces/http"
	"github.com/IDTS-LAB/go-codebase/internal/authorization"
	"github.com/IDTS-LAB/go-codebase/internal/authorization/infrastructure/casbin"
	authzHTTP "github.com/IDTS-LAB/go-codebase/internal/authorization/interfaces/http"
	"github.com/IDTS-LAB/go-codebase/internal/core/domain"
	"github.com/IDTS-LAB/go-codebase/internal/infrastructure/auth"
	"github.com/IDTS-LAB/go-codebase/internal/infrastructure/cache"
	"github.com/IDTS-LAB/go-codebase/internal/infrastructure/email"
	"github.com/IDTS-LAB/go-codebase/internal/infrastructure/logger"
	"github.com/IDTS-LAB/go-codebase/internal/infrastructure/messaging"
	"github.com/IDTS-LAB/go-codebase/internal/monitoring"
	monitoringDomain "github.com/IDTS-LAB/go-codebase/internal/monitoring/domain"
	"github.com/IDTS-LAB/go-codebase/internal/shared/auditlog"
	"github.com/IDTS-LAB/go-codebase/internal/shared/config"
	"github.com/IDTS-LAB/go-codebase/internal/shared/cqrs"
	"github.com/IDTS-LAB/go-codebase/internal/shared/database"
	"github.com/IDTS-LAB/go-codebase/internal/shared/events"
	"github.com/IDTS-LAB/go-codebase/internal/shared/middleware"
	"github.com/IDTS-LAB/go-codebase/internal/shared/router"
	"github.com/IDTS-LAB/go-codebase/internal/shared/telemetry"
	"github.com/IDTS-LAB/go-codebase/internal/shared/tenantfilter"
	"github.com/IDTS-LAB/go-codebase/internal/shared/validator"
	"github.com/IDTS-LAB/go-codebase/internal/tenant"
	tenantHTTP "github.com/IDTS-LAB/go-codebase/internal/tenant/interfaces/http"
	"github.com/IDTS-LAB/go-codebase/internal/todo"
	todoHTTP "github.com/IDTS-LAB/go-codebase/internal/todo/interfaces/http"
	"github.com/IDTS-LAB/go-codebase/internal/user"
	userHTTP "github.com/IDTS-LAB/go-codebase/internal/user/interfaces/http"
	"github.com/redis/go-redis/v9"
	"go.uber.org/fx"
)

func main() {
	if err := run(); err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}
}

func run() error {
	cfg, err := config.New()
	if err != nil {
		return fmt.Errorf("load config: %w", err)
	}

	fmt.Printf("[config] env=%s log_format=%s log_level=%s\n", cfg.App.Env, cfg.Log.Format, cfg.Log.Level)

	var (
		authHandler     *authHTTP.Handler
		todoHandler     *todoHTTP.Handler
		authzHandler    *authzHTTP.Handler
		userHandler     *userHTTP.Handler
		tenantHandler   *tenantHTTP.Handler
		enforcer        *casbin.Enforcer
		log             domain.Logger
		db              *sql.DB
		rdb             *redis.Client
		tokenSvc        domain.TokenService
		errorRepo       *auditlog.Repository
		metricsRecorder monitoringDomain.MetricsRecorder
		metricsHandler  http.Handler
	)

	app := fx.New(
		fx.Supply(cfg),

		// Infrastructure
		cqrs.Module,
		logger.Module,
		cache.Module,
		auth.Module,
		messaging.Module,
		database.Module,
		telemetry.Module,
		validator.Module,
		email.Module,

		// Modules
		events.Module,
		authentication.Module,
		authorization.Module,
		monitoring.Module,
		todo.Module,
		user.Module,
		tenant.Module,

		// Shared
		fx.Provide(auditlog.NewRepository),
		fx.Provide(func(cfg *config.Config) *tenantfilter.Config {
			return &tenantfilter.Config{Enabled: cfg.Tenant.Enabled}
		}),

		// Event handlers
		fx.Invoke(func(bus events.EventBus, eh *authEventBus.EmailHandler) {
			eh.Register(bus)
		}),

		// Extract
		fx.Populate(&authHandler),
		fx.Populate(&todoHandler),
		fx.Populate(&authzHandler),
		fx.Populate(&userHandler),
		fx.Populate(&tenantHandler),
		fx.Populate(&enforcer),
		fx.Populate(&log),
		fx.Populate(&db),
		fx.Populate(&rdb),
		fx.Populate(&tokenSvc),
		fx.Populate(&errorRepo),
		fx.Populate(&metricsRecorder),
		fx.Populate(&metricsHandler),
	)

	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer cancel()

	if err := app.Start(ctx); err != nil {
		return fmt.Errorf("start app: %w", err)
	}

	mw := middleware.NewRegistry(tokenSvc, rdb, cfg, log, errorRepo, enforcer, metricsRecorder)

	root := router.NewRouter(router.Handlers{
		Auth:           authHTTP.NewRouter(authHandler, mw.Auth),
		Todo:           todoHTTP.NewRouter(todoHandler, mw.Auth, enforcer),
		Authz:          authzHTTP.NewRouter(authzHandler, mw.Auth, enforcer),
		User:           userHTTP.NewRouter(userHandler, mw.Auth, enforcer),
		Tenant:         tenantHTTP.NewRouter(tenantHandler, mw.Auth, enforcer),
		MetricsHandler: metricsHandler,
	}, mw, log, cfg, db)

	srv := &http.Server{
		Addr:         fmt.Sprintf(":%d", cfg.Server.Port),
		Handler:      root,
		ReadTimeout:  time.Duration(cfg.Server.ReadTimeout) * time.Second,
		WriteTimeout: time.Duration(cfg.Server.WriteTimeout) * time.Second,
		IdleTimeout:  time.Duration(cfg.Server.IdleTimeout) * time.Second,
	}

	go func() {
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatal(ctx, "server failed", domain.Error(err))
		}
	}()

	log.Info(ctx, "starting server", domain.String("addr", srv.Addr))

	<-ctx.Done()

	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer shutdownCancel()

	if err := srv.Shutdown(shutdownCtx); err != nil {
		return fmt.Errorf("shutdown server: %w", err)
	}

	if err := app.Stop(context.Background()); err != nil {
		return fmt.Errorf("stop app: %w", err)
	}

	return nil
}
