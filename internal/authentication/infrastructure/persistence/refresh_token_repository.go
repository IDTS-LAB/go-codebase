package persistence

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/IDTS-LAB/go-codebase/internal/authentication/domain/entity"
	"github.com/IDTS-LAB/go-codebase/internal/authentication/domain/repository"
	"github.com/IDTS-LAB/go-codebase/internal/authentication/infrastructure/persistence/sqlc"
	"github.com/IDTS-LAB/go-codebase/internal/core/domain"
	"github.com/google/uuid"
)

type refreshTokenRepository struct {
	db *sql.DB
}

func NewRefreshTokenRepository(db *sql.DB) repository.RefreshTokenRepository {
	return &refreshTokenRepository{db: db}
}

func (r *refreshTokenRepository) Create(ctx context.Context, token *entity.RefreshToken) error {
	q := sqlc.New(r.db)
	err := q.CreateRefreshToken(ctx, sqlc.CreateRefreshTokenParams{
		ID:        token.ID,
		UserID:    token.UserID,
		Token:     token.Token,
		ExpiresAt: token.ExpiresAt,
		CreatedAt: token.CreatedAt,
		UpdatedAt: token.UpdatedAt,
	})
	if err != nil {
		return fmt.Errorf("insert refresh token: %w", err)
	}
	return nil
}

func (r *refreshTokenRepository) GetByToken(ctx context.Context, token string) (*entity.RefreshToken, error) {
	q := sqlc.New(r.db)
	row, err := q.GetRefreshTokenByToken(ctx, token)
	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("refresh token not found")
	}
	if err != nil {
		return nil, fmt.Errorf("get refresh token: %w", err)
	}
	return mapSqlcRefreshTokenToEntity(row), nil
}

func (r *refreshTokenRepository) GetByUserID(ctx context.Context, userID uuid.UUID) ([]*entity.RefreshToken, error) {
	q := sqlc.New(r.db)
	rows, err := q.GetRefreshTokensByUserID(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("get refresh tokens: %w", err)
	}
	tokens := make([]*entity.RefreshToken, len(rows))
	for i, row := range rows {
		tokens[i] = mapSqlcRefreshTokenToEntity(row)
	}
	return tokens, nil
}

func (r *refreshTokenRepository) Revoke(ctx context.Context, token string) error {
	q := sqlc.New(r.db)
	err := q.RevokeRefreshToken(ctx, token)
	if err != nil {
		return fmt.Errorf("revoke refresh token: %w", err)
	}
	return nil
}

func (r *refreshTokenRepository) RevokeAllByUserID(ctx context.Context, userID uuid.UUID) error {
	q := sqlc.New(r.db)
	err := q.RevokeAllRefreshTokensByUserID(ctx, userID)
	if err != nil {
		return fmt.Errorf("revoke all refresh tokens: %w", err)
	}
	return nil
}

func (r *refreshTokenRepository) DeleteExpired(ctx context.Context) error {
	q := sqlc.New(r.db)
	err := q.DeleteExpiredRefreshTokens(ctx)
	if err != nil {
		return fmt.Errorf("delete expired tokens: %w", err)
	}
	return nil
}

func mapSqlcRefreshTokenToEntity(row sqlc.RefreshToken) *entity.RefreshToken {
	return &entity.RefreshToken{
		Entity: domain.Entity{
			ID:        row.ID,
			CreatedAt: row.CreatedAt,
			UpdatedAt: row.UpdatedAt,
			DeletedAt: nullTimeToPtr(row.DeletedAt),
		},
		UserID:    row.UserID,
		Token:     row.Token,
		ExpiresAt: row.ExpiresAt,
		RevokedAt: nullTimeToPtr(row.RevokedAt),
	}
}
