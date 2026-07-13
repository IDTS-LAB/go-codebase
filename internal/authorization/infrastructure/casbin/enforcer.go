package casbin

import (
	"context"
	_ "embed"
	"fmt"

	"github.com/casbin/casbin/v2"
	"github.com/casbin/casbin/v2/model"
	"github.com/google/uuid"
	"go.uber.org/fx"
)

//go:embed model.conf
var modelConf string

var Module = fx.Module("casbin", fx.Provide(NewEnforcer))

type Enforcer struct {
	enforcer *casbin.CachedEnforcer
	adapter  *Adapter
}

func NewEnforcer(adapter *Adapter) (*Enforcer, error) {
	m, err := model.NewModelFromString(modelConf)
	if err != nil {
		return nil, fmt.Errorf("parse casbin model: %w", err)
	}

	enforcer, err := casbin.NewCachedEnforcer(m, adapter)
	if err != nil {
		return nil, fmt.Errorf("create casbin enforcer: %w", err)
	}

	e := &Enforcer{
		enforcer: enforcer,
		adapter:  adapter,
	}

	if err := e.ReloadPolicies(context.Background()); err != nil {
		return nil, fmt.Errorf("load initial policies: %w", err)
	}

	return e, nil
}

func (e *Enforcer) ReloadPolicies(ctx context.Context) error {
	if err := SyncAllPolicies(ctx, e.adapter.db, e.enforcer); err != nil {
		return fmt.Errorf("reload policies: %w", err)
	}
	return nil
}

func (e *Enforcer) ReloadUserPolicies(ctx context.Context, userID uuid.UUID) error {
	if err := SyncUserPolicies(ctx, e.adapter.db, e.enforcer, userID); err != nil {
		return fmt.Errorf("reload user policies: %w", err)
	}
	return nil
}

func (e *Enforcer) Enforce(userID uuid.UUID, resource, action string) (bool, error) {
	return e.enforcer.Enforce(userID.String(), resource, action)
}
