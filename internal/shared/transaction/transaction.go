package transaction

import (
	"context"
	"database/sql"

	"go.uber.org/fx"
)

var Module = fx.Module("transaction", fx.Provide(NewManager))

type Manager struct {
	db *sql.DB
}

func NewManager(db *sql.DB) *Manager {
	return &Manager{db: db}
}

type ctxKey struct{}

func (m *Manager) WithTx(ctx context.Context, fn func(ctx context.Context) error) error {
	tx, err := m.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}

	ctx = context.WithValue(ctx, ctxKey{}, tx)

	if err := fn(ctx); err != nil {
		_ = tx.Rollback()
		return err
	}

	return tx.Commit()
}

func (m *Manager) DB() *sql.DB {
	return m.db
}

func GetTx(ctx context.Context) (*sql.Tx, bool) {
	tx, ok := ctx.Value(ctxKey{}).(*sql.Tx)
	return tx, ok
}
