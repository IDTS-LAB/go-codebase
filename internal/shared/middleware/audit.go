package middleware

import (
	"bytes"
	"io"
	"net/http"
	"time"

	"github.com/IDTS-LAB/go-codebase/internal/shared/auditlog"
	"github.com/google/uuid"
)

func AuditLog(repo *auditlog.Repository) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			start := time.Now()

			var reqBody []byte
			if r.Body != nil && (r.Method == http.MethodPost || r.Method == http.MethodPut || r.Method == http.MethodPatch) {
				reqBody, _ = io.ReadAll(r.Body)
				r.Body = io.NopCloser(bytes.NewBuffer(reqBody))
			}

			wrapped := &auditResponseWriter{ResponseWriter: w, statusCode: http.StatusOK}

			next.ServeHTTP(wrapped, r)

			duration := time.Since(start)

			userID := GetUserID(r.Context())
			userEmail := GetUserEmail(r.Context())
			requestID := GetRequestID(r.Context())

			entry := &auditlog.AuditLog{
				ID:          uuid.New().String(),
				RequestID:   requestID,
				Method:      r.Method,
				Path:        r.URL.Path,
				StatusCode:  wrapped.statusCode,
				DurationMs:  duration.Milliseconds(),
				IP:          r.RemoteAddr,
				UserAgent:   r.UserAgent(),
				ResponseSize: wrapped.bytesWritten,
				CreatedAt:   time.Now(),
			}

			if userID != "" {
				entry.UserID = &userID
			}
			if userEmail != "" {
				entry.UserEmail = &userEmail
			}
			if len(reqBody) > 0 {
				body := string(reqBody)
				entry.RequestBody = &body
			}

			_ = repo.InsertAuditLog(r.Context(), entry)
		})
	}
}

type auditResponseWriter struct {
	http.ResponseWriter
	statusCode   int
	bytesWritten int
}

func (rw *auditResponseWriter) WriteHeader(code int) {
	rw.statusCode = code
	rw.ResponseWriter.WriteHeader(code)
}

func (rw *auditResponseWriter) Write(b []byte) (int, error) {
	n, err := rw.ResponseWriter.Write(b)
	rw.bytesWritten += n
	return n, err
}
