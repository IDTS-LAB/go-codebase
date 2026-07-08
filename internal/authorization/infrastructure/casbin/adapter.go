package casbin

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/google/uuid"
)

type Policy struct {
	Subject string
	Object  string
	Action  string
}

type PolicyLoader struct {
	db *sql.DB
}

func NewPolicyLoader(db *sql.DB) *PolicyLoader {
	return &PolicyLoader{db: db}
}

func (l *PolicyLoader) LoadAllPolicies(ctx context.Context) ([]Policy, error) {
	query := `
		SELECT
			ur.user_id::text,
			p.resource,
			p.action
		FROM user_roles ur
		JOIN role_permissions rp ON ur.role_id = rp.role_id
		JOIN permissions p ON rp.permission_id = p.id
		JOIN roles r ON ur.role_id = r.id
		WHERE r.deleted_at IS NULL AND p.deleted_at IS NULL`

	rows, err := l.db.QueryContext(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("load policies: %w", err)
	}
	defer rows.Close()

	var policies []Policy
	for rows.Next() {
		var pol Policy
		if err := rows.Scan(&pol.Subject, &pol.Object, &pol.Action); err != nil {
			return nil, fmt.Errorf("scan policy: %w", err)
		}
		policies = append(policies, pol)
	}
	return policies, nil
}

func (l *PolicyLoader) LoadUserPolicies(ctx context.Context, userID uuid.UUID) ([]Policy, error) {
	query := `
		SELECT
			ur.user_id::text,
			p.resource,
			p.action
		FROM user_roles ur
		JOIN role_permissions rp ON ur.role_id = rp.role_id
		JOIN permissions p ON rp.permission_id = p.id
		JOIN roles r ON ur.role_id = r.id
		WHERE ur.user_id = $1 AND r.deleted_at IS NULL AND p.deleted_at IS NULL`

	rows, err := l.db.QueryContext(ctx, query, userID)
	if err != nil {
		return nil, fmt.Errorf("load user policies: %w", err)
	}
	defer rows.Close()

	var policies []Policy
	for rows.Next() {
		var pol Policy
		if err := rows.Scan(&pol.Subject, &pol.Object, &pol.Action); err != nil {
			return nil, fmt.Errorf("scan policy: %w", err)
		}
		policies = append(policies, pol)
	}
	return policies, nil
}
