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
	loader   *PolicyLoader
}

func NewEnforcer(loader *PolicyLoader) (*Enforcer, error) {
	m, err := model.NewModelFromString(modelConf)
	if err != nil {
		return nil, fmt.Errorf("parse casbin model: %w", err)
	}

	enforcer, err := casbin.NewCachedEnforcer(m)
	if err != nil {
		return nil, fmt.Errorf("create casbin enforcer: %w", err)
	}

	e := &Enforcer{
		enforcer: enforcer,
		loader:   loader,
	}

	if err := e.ReloadPolicies(context.Background()); err != nil {
		return nil, fmt.Errorf("load initial policies: %w", err)
	}

	return e, nil
}

func (e *Enforcer) ReloadPolicies(ctx context.Context) error {
	policies, err := e.loader.LoadAllPolicies(ctx)
	if err != nil {
		return err
	}

	e.enforcer.ClearPolicy()
	for _, p := range policies {
		if _, err := e.enforcer.AddPolicy(p.Subject, p.Object, p.Action); err != nil {
			return fmt.Errorf("add policy: %w", err)
		}
	}
	return nil
}

func (e *Enforcer) ReloadUserPolicies(ctx context.Context, userID uuid.UUID) error {
	policies, err := e.loader.LoadUserPolicies(ctx, userID)
	if err != nil {
		return err
	}

	subject := userID.String()
	e.enforcer.RemoveFilteredPolicy(0, subject)
	for _, p := range policies {
		if _, err := e.enforcer.AddPolicy(p.Subject, p.Object, p.Action); err != nil {
			return fmt.Errorf("add user policy: %w", err)
		}
	}
	return nil
}

func (e *Enforcer) Enforce(userID uuid.UUID, resource, action string) (bool, error) {
	return e.enforcer.Enforce(userID.String(), resource, action)
}
