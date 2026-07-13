package casbin

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/casbin/casbin/v2"
	"github.com/google/uuid"
)

type flattenedPolicy []string

func SyncUserPolicies(ctx context.Context, db *sql.DB, enforcer *casbin.CachedEnforcer, userID uuid.UUID) error {
	policies, err := loadUserFlattenedPolicies(ctx, db, userID)
	if err != nil {
		return fmt.Errorf("load flattened policies: %w", err)
	}

	subject := userID.String()

	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin tx: %w", err)
	}
	defer tx.Rollback()

	if _, err := tx.ExecContext(ctx, `DELETE FROM casbin_rule WHERE ptype = 'p' AND v0 = $1`, subject); err != nil {
		return fmt.Errorf("delete existing policies: %w", err)
	}

	stmt, err := tx.PrepareContext(ctx, `INSERT INTO casbin_rule (ptype, v0, v1, v2) VALUES ('p', $1, $2, $3)`)
	if err != nil {
		return fmt.Errorf("prepare insert: %w", err)
	}
	defer stmt.Close()

	for _, p := range policies {
		if _, err := stmt.ExecContext(ctx, p[0], p[1], p[2]); err != nil {
			return fmt.Errorf("insert policy: %w", err)
		}
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit: %w", err)
	}

	if err := enforcer.LoadPolicy(); err != nil {
		return fmt.Errorf("load policy: %w", err)
	}

	return nil
}

func SyncAllPolicies(ctx context.Context, db *sql.DB, enforcer *casbin.CachedEnforcer) error {
	policies, err := loadAllFlattenedPolicies(ctx, db)
	if err != nil {
		return fmt.Errorf("load all flattened policies: %w", err)
	}

	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin tx: %w", err)
	}
	defer tx.Rollback()

	if _, err := tx.ExecContext(ctx, `DELETE FROM casbin_rule WHERE ptype = 'p'`); err != nil {
		return fmt.Errorf("delete existing policies: %w", err)
	}

	stmt, err := tx.PrepareContext(ctx, `INSERT INTO casbin_rule (ptype, v0, v1, v2) VALUES ('p', $1, $2, $3)`)
	if err != nil {
		return fmt.Errorf("prepare insert: %w", err)
	}
	defer stmt.Close()

	for _, p := range policies {
		if _, err := stmt.ExecContext(ctx, p[0], p[1], p[2]); err != nil {
			return fmt.Errorf("insert policy: %w", err)
		}
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit: %w", err)
	}

	return enforcer.LoadPolicy()
}

func loadUserFlattenedPolicies(ctx context.Context, db *sql.DB, userID uuid.UUID) ([]flattenedPolicy, error) {
	query := `
		SELECT ur.user_id::text, p.resource, p.action
		FROM user_roles ur
		JOIN role_permissions rp ON ur.role_id = rp.role_id
		JOIN permissions p ON rp.permission_id = p.id
		JOIN roles r ON ur.role_id = r.id
		WHERE ur.user_id = $1 AND r.deleted_at IS NULL AND p.deleted_at IS NULL
		GROUP BY ur.user_id, p.resource, p.action`

	rows, err := db.QueryContext(ctx, query, userID)
	if err != nil {
		return nil, fmt.Errorf("query user policies: %w", err)
	}
	defer rows.Close()

	var policies []flattenedPolicy
	for rows.Next() {
		var sub, obj, act string
		if err := rows.Scan(&sub, &obj, &act); err != nil {
			return nil, fmt.Errorf("scan: %w", err)
		}
		policies = append(policies, flattenedPolicy{sub, obj, act})
	}
	return policies, rows.Err()
}

func loadAllFlattenedPolicies(ctx context.Context, db *sql.DB) ([]flattenedPolicy, error) {
	query := `
		SELECT ur.user_id::text, p.resource, p.action
		FROM user_roles ur
		JOIN role_permissions rp ON ur.role_id = rp.role_id
		JOIN permissions p ON rp.permission_id = p.id
		JOIN roles r ON ur.role_id = r.id
		WHERE r.deleted_at IS NULL AND p.deleted_at IS NULL
		GROUP BY ur.user_id, p.resource, p.action`

	rows, err := db.QueryContext(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("query all policies: %w", err)
	}
	defer rows.Close()

	var policies []flattenedPolicy
	for rows.Next() {
		var sub, obj, act string
		if err := rows.Scan(&sub, &obj, &act); err != nil {
			return nil, fmt.Errorf("scan: %w", err)
		}
		policies = append(policies, flattenedPolicy{sub, obj, act})
	}
	return policies, rows.Err()
}
