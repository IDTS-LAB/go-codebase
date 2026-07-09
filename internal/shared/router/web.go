package router

import (
	"net/http"

	"github.com/IDTS-LAB/go-codebase/docs"
	"github.com/IDTS-LAB/go-codebase/internal/shared/config"
	"github.com/go-chi/chi/v5"
	httpSwagger "github.com/swaggo/http-swagger"
)

func registerWeb(r chi.Router, cfg *config.Config) {
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

	if cfg.App.Env != "production" {
		docs.SwaggerInfo.BasePath = APIPrefix
		r.Get("/swagger/*", httpSwagger.Handler(
			httpSwagger.URL("/swagger/doc.json"),
		))
	}
}
