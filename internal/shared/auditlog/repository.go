package auditlog

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
)

type AuditLog struct {
	ID           string    `json:"id"`
	RequestID    string    `json:"request_id"`
	UserID       *string   `json:"user_id,omitempty"`
	UserEmail    *string   `json:"user_email,omitempty"`
	Method       string    `json:"method"`
	Path         string    `json:"path"`
	StatusCode   int       `json:"status_code"`
	DurationMs   int64     `json:"duration_ms"`
	IP           string    `json:"ip"`
	UserAgent    string    `json:"user_agent"`
	RequestBody  *string   `json:"request_body,omitempty"`
	ResponseSize int       `json:"response_size"`
	TenantID     string    `json:"tenant_id,omitempty"`
	CreatedAt    time.Time `json:"created_at"`
}

type ErrorLog struct {
	ID          string          `json:"id"`
	RequestID   string          `json:"request_id"`
	UserID      *string         `json:"user_id,omitempty"`
	UserEmail   *string         `json:"user_email,omitempty"`
	Level       string          `json:"level"`
	Message     string          `json:"message"`
	Error       string          `json:"error"`
	StackTrace  string          `json:"stack_trace"`
	Method      string          `json:"method"`
	Path        string          `json:"path"`
	StatusCode  int             `json:"status_code"`
	IP          string          `json:"ip"`
	UserAgent   string          `json:"user_agent"`
	RequestBody *string         `json:"request_body,omitempty"`
	Metadata    json.RawMessage `json:"metadata,omitempty"`
	TenantID    string          `json:"tenant_id,omitempty"`
	CreatedAt   time.Time       `json:"created_at"`
}

type Repository struct {
	db *sql.DB
}

func NewRepository(db *sql.DB) *Repository {
	return &Repository{db: db}
}

func (r *Repository) InsertAuditLog(ctx context.Context, log *AuditLog) error {
	id, err := uuid.Parse(log.ID)
	if err != nil {
		return err
	}
	_, err = r.db.ExecContext(ctx,
		`INSERT INTO audit_logs (id, request_id, user_id, user_email, method, path, status_code, duration_ms, ip, user_agent, request_body, response_size, tenant_id, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14)`,
		id,
		log.RequestID,
		ptrStringToNullUUID(log.UserID),
		ptrStringToNullString(log.UserEmail),
		log.Method,
		log.Path,
		int32(log.StatusCode),
		log.DurationMs,
		log.IP,
		log.UserAgent,
		ptrStringToNullString(log.RequestBody),
		int32(log.ResponseSize),
		log.TenantID,
		log.CreatedAt,
	)
	if err != nil {
		return fmt.Errorf("insert audit log: %w", err)
	}
	return nil
}

func (r *Repository) InsertErrorLog(ctx context.Context, log *ErrorLog) error {
	id, err := uuid.Parse(log.ID)
	if err != nil {
		return err
	}
	_, err = r.db.ExecContext(ctx,
		`INSERT INTO error_logs (id, request_id, user_id, user_email, level, message, error, stack_trace, method, path, status_code, ip, user_agent, request_body, metadata, tenant_id, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15::jsonb, $16, $17)`,
		id,
		log.RequestID,
		ptrStringToNullUUID(log.UserID),
		ptrStringToNullString(log.UserEmail),
		log.Level,
		log.Message,
		log.Error,
		log.StackTrace,
		log.Method,
		log.Path,
		int32(log.StatusCode),
		log.IP,
		log.UserAgent,
		ptrStringToNullString(log.RequestBody),
		log.Metadata,
		log.TenantID,
		log.CreatedAt,
	)
	if err != nil {
		return fmt.Errorf("insert error log: %w", err)
	}
	return nil
}

func ptrStringToNullString(s *string) sql.NullString {
	if s == nil {
		return sql.NullString{Valid: false}
	}
	return sql.NullString{String: *s, Valid: true}
}

func ptrStringToNullUUID(s *string) uuid.NullUUID {
	if s == nil {
		return uuid.NullUUID{Valid: false}
	}
	uid, err := uuid.Parse(*s)
	if err != nil {
		return uuid.NullUUID{Valid: false}
	}
	return uuid.NullUUID{UUID: uid, Valid: true}
}
