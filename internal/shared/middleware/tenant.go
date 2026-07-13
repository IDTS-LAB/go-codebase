package middleware

import (
	"context"
	"net/http"
	"strings"

	"github.com/IDTS-LAB/go-codebase/internal/shared/config"
)

func TenantResolver(cfg *config.TenantConfig) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			tenantID := ""
			if cfg.Enabled {
				tenantID = resolveTenant(r, cfg)
			}
			ctx := context.WithValue(r.Context(), TenantIDKey, tenantID)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

func resolveTenant(r *http.Request, cfg *config.TenantConfig) string {
	if tid := GetTenantIDFromClaims(r.Context()); tid != "" {
		return tid
	}
	if h := r.Header.Get(cfg.TenantHeader); h != "" {
		return h
	}
	if sub := domainFromHost(r.Host, cfg.Domain); sub != "" && sub != "www" {
		return sub
	}
	return ""
}

func domainFromHost(host, domainSuffix string) string {
	host = strings.Split(host, ":")[0]
	if !strings.HasSuffix(host, "."+domainSuffix) {
		return ""
	}
	return strings.TrimSuffix(host, "."+domainSuffix)
}

func GetTenantIDFromClaims(ctx context.Context) string {
	if v, ok := ctx.Value(TenantClaimKey).(string); ok {
		return v
	}
	return ""
}
