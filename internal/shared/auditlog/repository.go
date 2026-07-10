package auditlog

import (
	"context"
	"database/sql"
	"encoding/json"
	"time"

	"github.com/IDTS-LAB/go-codebase/internal/shared/auditlog/sqlc"
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
	CreatedAt    time.Time `json:"created_at"`
}

type ErrorLog struct {
	ID           string          `json:"id"`
	RequestID    string          `json:"request_id"`
	UserID       *string         `json:"user_id,omitempty"`
	UserEmail    *string         `json:"user_email,omitempty"`
	Level        string          `json:"level"`
	Message      string          `json:"message"`
	Error        string          `json:"error"`
	StackTrace   string          `json:"stack_trace"`
	Method       string          `json:"method"`
	Path         string          `json:"path"`
	StatusCode   int             `json:"status_code"`
	IP           string          `json:"ip"`
	UserAgent    string          `json:"user_agent"`
	RequestBody  *string         `json:"request_body,omitempty"`
	Metadata     json.RawMessage `json:"metadata,omitempty"`
	CreatedAt    time.Time       `json:"created_at"`
}

type Repository struct {
	db *sql.DB
}

func NewRepository(db *sql.DB) *Repository {
	return &Repository{db: db}
}

func (r *Repository) InsertAuditLog(ctx context.Context, log *AuditLog) error {
	q := sqlc.New(r.db)
	id, err := uuid.Parse(log.ID)
	if err != nil {
		return err
	}
	return q.InsertAuditLog(ctx, sqlc.InsertAuditLogParams{
		ID:           id,
		RequestID:    log.RequestID,
		UserID:       ptrStringToNullUUID(log.UserID),
		UserEmail:    ptrStringToNullString(log.UserEmail),
		Method:       log.Method,
		Path:         log.Path,
		StatusCode:   int32(log.StatusCode),
		DurationMs:   log.DurationMs,
		Ip:           log.IP,
		UserAgent:    log.UserAgent,
		RequestBody:  ptrStringToNullString(log.RequestBody),
		ResponseSize: int32(log.ResponseSize),
		CreatedAt:    log.CreatedAt,
	})
}

func (r *Repository) InsertErrorLog(ctx context.Context, log *ErrorLog) error {
	q := sqlc.New(r.db)
	id, err := uuid.Parse(log.ID)
	if err != nil {
		return err
	}
	return q.InsertErrorLog(ctx, sqlc.InsertErrorLogParams{
		ID:          id,
		RequestID:   log.RequestID,
		UserID:      ptrStringToNullUUID(log.UserID),
		UserEmail:   ptrStringToNullString(log.UserEmail),
		Level:       log.Level,
		Message:     log.Message,
		Error:       log.Error,
		StackTrace:  log.StackTrace,
		Method:      log.Method,
		Path:        log.Path,
		StatusCode:  int32(log.StatusCode),
		Ip:          log.IP,
		UserAgent:   log.UserAgent,
		RequestBody: ptrStringToNullString(log.RequestBody),
		Column15:    log.Metadata,
		CreatedAt:   log.CreatedAt,
	})
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
