package tenantfilter

import (
	"context"
	"fmt"

	"github.com/IDTS-LAB/go-codebase/internal/shared/middleware"
)

type Config struct {
	Enabled bool
}

func Where(ctx context.Context, config *Config, nextPosition int) (string, interface{}) {
	if config == nil || !config.Enabled {
		return "", nil
	}
	tenantID := middleware.GetTenantID(ctx)
	if tenantID == "" {
		return "", nil
	}
	return fmt.Sprintf("tenant_id = $%d", nextPosition), tenantID
}

func WhereAnd(ctx context.Context, config *Config, nextPosition int) (string, interface{}) {
	clause, val := Where(ctx, config, nextPosition)
	if clause == "" {
		return "", nil
	}
	return "AND " + clause, val
}
