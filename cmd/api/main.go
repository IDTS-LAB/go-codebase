package main

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/IDTS-LAB/go-codebase/internal/authentication"
	authHTTP "github.com/IDTS-LAB/go-codebase/internal/authentication/interfaces/http"
	"github.com/IDTS-LAB/go-codebase/internal/authentication/application/service"
	"github.com/IDTS-LAB/go-codebase/internal/authorization"
	"github.com/IDTS-LAB/go-codebase/internal/authorization/infrastructure/casbin"
	authzHTTP "github.com/IDTS-LAB/go-codebase/internal/authorization/interfaces/http"
	"github.com/IDTS-LAB/go-codebase/internal/core/domain"
	"github.com/IDTS-LAB/go-codebase/internal/infrastructure/auth"
	"github.com/IDTS-LAB/go-codebase/internal/infrastructure/cache"
	"github.com/IDTS-LAB/go-codebase/internal/infrastructure/logger"
	"github.com/IDTS-LAB/go-codebase/internal/infrastructure/messaging"
	"github.com/IDTS-LAB/go-codebase/internal/shared/auditlog"
	"github.com/IDTS-LAB/go-codebase/internal/shared/config"
	"github.com/IDTS-LAB/go-codebase/internal/shared/database"
	"github.com/IDTS-LAB/go-codebase/internal/shared/middleware"
	"github.com/IDTS-LAB/go-codebase/internal/shared/router"
	"github.com/IDTS-LAB/go-codebase/internal/shared/telemetry"
	"github.com/IDTS-LAB/go-codebase/internal/shared/validator"
	"github.com/IDTS-LAB/go-codebase/internal/todo"
	todoHTTP "github.com/IDTS-LAB/go-codebase/internal/todo/interfaces/http"
	"github.com/IDTS-LAB/go-codebase/internal/user"
	userHTTP "github.com/IDTS-LAB/go-codebase/internal/user/interfaces/http"
	"github.com/redis/go-redis/v9"
	"go.uber.org/fx"
)

func main() {
	cfg, err := config.New()
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to load config: %v\n", err)
		os.Exit(1)
	}

	var (
		authHandler  *authHTTP.Handler
		todoHandler  *todoHTTP.Handler
		authzHandler *authzHTTP.Handler
		userHandler  *userHTTP.Handler
		enforcer     *casbin.Enforcer
		log          domain.Logger
		rdb          *redis.Client
		tokenSvc     domain.TokenService
		errorRepo    *auditlog.Repository
	)

	app := fx.New(
		fx.Supply(cfg),

		// Infrastructure
		logger.Module,
		cache.Module,
		auth.Module,
		messaging.Module,
		database.Module,
		telemetry.Module,
		validator.Module,

		// Modules
		authentication.Module,
		authorization.Module,
		todo.Module,
		user.Module,

		// Shared
		fx.Provide(auditlog.NewRepository),

		// Denylist helper
		fx.Invoke(func(authSvc *service.AuthenticationService, rdb *redis.Client, cfg *config.Config) {
			if cfg.Auth.TokenDenylist {
				authSvc.SetDenylist(func(ctx context.Context, jti string, ttl time.Duration) error {
					return rdb.Set(ctx, "token:blacklist:"+jti, "1", ttl).Err()
				})
			}
			authSvc.SetLockoutConfig(cfg.Auth.MaxLoginAttempts, time.Duration(cfg.Auth.LockoutDuration)*time.Second)
		}),

		// Extract
		fx.Populate(&authHandler),
		fx.Populate(&todoHandler),
		fx.Populate(&authzHandler),
		fx.Populate(&userHandler),
		fx.Populate(&enforcer),
		fx.Populate(&log),
		fx.Populate(&rdb),
		fx.Populate(&tokenSvc),
		fx.Populate(&errorRepo),
	)

	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer cancel()

	if err := app.Start(ctx); err != nil {
		fmt.Fprintf(os.Stderr, "failed to start app: %v\n", err)
		os.Exit(1)
	}

	mw := middleware.NewRegistry(tokenSvc, rdb, cfg, log, errorRepo, enforcer)

	root := router.NewRouter(router.Handlers{
		Auth:  authHTTP.NewRouter(authHandler, mw.Auth),
		Todo:  todoHTTP.NewRouter(todoHandler, mw.Auth, enforcer),
		Authz: authzHTTP.NewRouter(authzHandler, mw.Auth, enforcer),
		User:  userHTTP.NewRouter(userHandler, mw.Auth, enforcer),
	}, mw, log, cfg)

	srv := &http.Server{
		Addr:         fmt.Sprintf(":%d", cfg.Server.Port),
		Handler:      root,
		ReadTimeout:  time.Duration(cfg.Server.ReadTimeout) * time.Second,
		WriteTimeout: time.Duration(cfg.Server.WriteTimeout) * time.Second,
	}

	go func() {
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatal(ctx, "server failed", domain.Error(err))
		}
	}()

	log.Info(ctx, "starting server", domain.String("addr", srv.Addr))

	<-ctx.Done()

	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 15)
	defer shutdownCancel()

	if err := srv.Shutdown(shutdownCtx); err != nil {
		fmt.Fprintf(os.Stderr, "failed to stop app: %v\n", err)
		os.Exit(1)
	}

	if err := app.Stop(ctx); err != nil {
		fmt.Fprintf(os.Stderr, "failed to stop app: %v\n", err)
		os.Exit(1)
	}
}
