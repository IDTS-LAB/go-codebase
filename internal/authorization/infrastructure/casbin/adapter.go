package casbin

import (
	"database/sql"
	"fmt"
	"strings"

	"github.com/casbin/casbin/v2/model"
	"github.com/casbin/casbin/v2/persist"
)

type Adapter struct {
	db *sql.DB
}

func NewAdapter(db *sql.DB) *Adapter {
	return &Adapter{db: db}
}

func (a *Adapter) LoadPolicy(model model.Model) error {
	rows, err := a.db.Query(`SELECT ptype, v0, v1, v2, v3, v4, v5 FROM casbin_rule`)
	if err != nil {
		return fmt.Errorf("load policy: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		var ptype string
		var v0, v1, v2, v3, v4, v5 string
		if err := rows.Scan(&ptype, &v0, &v1, &v2, &v3, &v4, &v5); err != nil {
			return fmt.Errorf("scan policy: %w", err)
		}
		line := ptype + ", " + v0 + ", " + v1 + ", " + v2 + ", " + v3 + ", " + v4 + ", " + v5
		line = strings.TrimRight(line, ", ")
		if err := persist.LoadPolicyLine(line, model); err != nil {
			return fmt.Errorf("load policy line: %w", err)
		}
	}
	return rows.Err()
}

func (a *Adapter) SavePolicy(model model.Model) error {
	tx, err := a.db.Begin()
	if err != nil {
		return fmt.Errorf("save policy begin tx: %w", err)
	}
	defer func() { _ = tx.Rollback() }()

	if _, err = tx.Exec(`DELETE FROM casbin_rule`); err != nil {
		return fmt.Errorf("save policy delete: %w", err)
	}

	stmt, err := tx.Prepare(`INSERT INTO casbin_rule (ptype, v0, v1, v2, v3, v4, v5) VALUES ($1, $2, $3, $4, $5, $6, $7)`)
	if err != nil {
		return fmt.Errorf("save policy prepare: %w", err)
	}
	defer stmt.Close()

	for ptype, assertion := range model["p"] {
		for _, rule := range assertion.Policy {
			values := []interface{}{ptype}
			ruleParts := rule
			for len(ruleParts) < 6 {
				ruleParts = append(ruleParts, "")
			}
			for _, p := range ruleParts {
				values = append(values, p)
			}
			if _, err = stmt.Exec(values...); err != nil {
				return fmt.Errorf("save policy insert: %w", err)
			}
		}
	}

	for ptype, assertion := range model["g"] {
		for _, rule := range assertion.Policy {
			values := []interface{}{ptype}
			ruleParts := rule
			for len(ruleParts) < 6 {
				ruleParts = append(ruleParts, "")
			}
			for _, p := range ruleParts {
				values = append(values, p)
			}
			if _, err = stmt.Exec(values...); err != nil {
				return fmt.Errorf("save policy insert g: %w", err)
			}
		}
	}

	return tx.Commit()
}

func (a *Adapter) AddPolicy(sec string, ptype string, rule []string) error {
	values := []interface{}{ptype}
	for _, v := range rule {
		values = append(values, v)
	}
	for len(values) < 8 {
		values = append(values, "")
	}

	_, err := a.db.Exec(
		`INSERT INTO casbin_rule (ptype, v0, v1, v2, v3, v4, v5) VALUES ($1, $2, $3, $4, $5, $6, $7)`,
		values...,
	)
	if err != nil {
		return fmt.Errorf("add policy: %w", err)
	}
	return nil
}

func (a *Adapter) RemovePolicy(sec string, ptype string, rule []string) error {
	query := `DELETE FROM casbin_rule WHERE ptype = $1`
	args := []interface{}{ptype}
	for i, v := range rule {
		query += fmt.Sprintf(" AND v%d = $%d", i, i+2)
		args = append(args, v)
	}

	_, err := a.db.Exec(query, args...)
	if err != nil {
		return fmt.Errorf("remove policy: %w", err)
	}
	return nil
}

func (a *Adapter) RemoveFilteredPolicy(sec string, ptype string, fieldIndex int, fieldValues ...string) error {
	query := `DELETE FROM casbin_rule WHERE ptype = $1`
	args := []interface{}{ptype}

	for i, v := range fieldValues {
		if v == "" {
			continue
		}
		col := fieldIndex + i
		query += fmt.Sprintf(" AND v%d = $%d", col, len(args)+1)
		args = append(args, v)
	}

	_, err := a.db.Exec(query, args...)
	if err != nil {
		return fmt.Errorf("remove filtered policy: %w", err)
	}
	return nil
}

var _ persist.Adapter = (*Adapter)(nil)
