package middleware

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"runtime/debug"
	"time"

	"github.com/IDTS-LAB/go-codebase/internal/core/domain"
	"github.com/IDTS-LAB/go-codebase/internal/shared/auditlog"
	"github.com/IDTS-LAB/go-codebase/internal/shared/config"
	"github.com/IDTS-LAB/go-codebase/internal/shared/utils"
	"github.com/google/uuid"
	"github.com/rs/cors"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
)

type contextKey string

const (
	RequestIDKey  contextKey = "request_id"
	UserIDKey     contextKey = "user_id"
	UserEmailKey  contextKey = "user_email"
	UserRoleKey   contextKey = "user_role"
	TenantIDKey   contextKey = "tenant_id"
	TenantClaimKey contextKey = "tenant_claim"
)

func ErrorHandler(log domain.Logger, errorRepo *auditlog.Repository) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			defer func() {
				if err := recover(); err != nil {
					stack := string(debug.Stack())
					ctx := r.Context()

					if span := trace.SpanFromContext(ctx); span.IsRecording() {
						span.SetStatus(codes.Error, "panic recovered")
						span.RecordError(fmt.Errorf("%v", err))
					}

					log.Error(ctx, "panic recovered",
						domain.String("error", fmt.Sprintf("%v", err)),
						domain.String("stack", stack),
					)

					persistError(r, errorRepo, log, http.StatusInternalServerError,
						"panic recovered", fmt.Sprintf("%v", err), stack)

					utils.RespondInternalError(w, "internal server error")
				}
			}()
			next.ServeHTTP(w, r)
		})
	}
}

func ErrorRecorder(log domain.Logger, errorRepo *auditlog.Repository) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			wrapped := &errorCaptureWriter{ResponseWriter: w, statusCode: http.StatusOK}
			next.ServeHTTP(wrapped, r)

			if wrapped.statusCode >= 500 {
				ctx := r.Context()
				if span := trace.SpanFromContext(ctx); span.IsRecording() {
					span.SetStatus(codes.Error, http.StatusText(wrapped.statusCode))
					if wrapped.errMsg != "" {
						span.RecordError(fmt.Errorf("%s", wrapped.errMsg))
					}
				}

				msg := http.StatusText(wrapped.statusCode)
				errMsg := wrapped.errMsg
				stack := wrapped.stack
				if info, ok := utils.GetErrorInfo(ctx); ok && info.Err != nil {
					msg = info.Err.Error()
					errMsg = info.Err.Error()
					stack = info.Stack
				}

				persistError(r, errorRepo, log, wrapped.statusCode, msg, errMsg, stack)
			}
		})
	}
}

func persistError(r *http.Request, repo *auditlog.Repository, log domain.Logger, status int, message, errMsg, stack string) {
	userID := GetUserID(r.Context())
	userEmail := GetUserEmail(r.Context())
	requestID := GetRequestID(r.Context())

	entry := &auditlog.ErrorLog{
		ID:         uuid.New().String(),
		RequestID:  requestID,
		Level:      "error",
		Message:    message,
		Error:      errMsg,
		StackTrace: stack,
		Method:     r.Method,
		Path:       r.URL.Path,
		StatusCode: status,
		IP:         r.RemoteAddr,
		UserAgent:  r.UserAgent(),
		TenantID:   GetTenantID(r.Context()),
		CreatedAt:  time.Now(),
	}

	if userID != "" {
		entry.UserID = &userID
	}
	if userEmail != "" {
		entry.UserEmail = &userEmail
	}

	if r.Body != nil && (r.Method == http.MethodPost || r.Method == http.MethodPut || r.Method == http.MethodPatch) {
		body, err := io.ReadAll(r.Body)
		r.Body = io.NopCloser(bytes.NewBuffer(body))
		if err == nil && len(body) > 0 {
			b := string(body)
			entry.RequestBody = &b
		}
	}

	meta := map[string]interface{}{
		"query": r.URL.RawQuery,
	}
	metaBytes, _ := json.Marshal(meta)
	entry.Metadata = metaBytes

	if err := repo.InsertErrorLog(r.Context(), entry); err != nil {
		log.Error(r.Context(), "failed to persist error log",
			domain.String("error", err.Error()),
		)
	}
}

type errorCaptureWriter struct {
	http.ResponseWriter
	statusCode int
	errMsg     string
	stack      string
}

func (w *errorCaptureWriter) WriteHeader(code int) {
	w.statusCode = code
	w.ResponseWriter.WriteHeader(code)
}

func (w *errorCaptureWriter) CaptureError(msg, stack string) {
	w.errMsg = msg
	w.stack = stack
}

func RequestID(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		id := r.Header.Get("X-Request-ID")
		if id == "" {
			id = uuid.New().String()
		}
		ctx := context.WithValue(r.Context(), RequestIDKey, id)
		w.Header().Set("X-Request-ID", id)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

func Logger(log domain.Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			start := time.Now()
			wrapped := &responseWriter{ResponseWriter: w, statusCode: http.StatusOK}
			next.ServeHTTP(wrapped, r)

			fields := []domain.Field{
				domain.String("method", r.Method),
				domain.String("path", r.URL.Path),
				domain.Int("status", wrapped.statusCode),
				domain.String("duration", time.Since(start).String()),
				domain.String("request_id", GetRequestID(r.Context())),
			}

			if userID := GetUserID(r.Context()); userID != "" {
				fields = append(fields, domain.String("user_id", userID))
			}
			if email := GetUserEmail(r.Context()); email != "" {
				fields = append(fields, domain.String("user_email", email))
			}

			log.Info(r.Context(), "request", fields...)
		})
	}
}

func Authentication(tokenSvc domain.TokenService) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			tokenStr := r.Header.Get("Authorization")
			if tokenStr == "" {
				utils.RespondUnauthorized(w, "missing token")
				return
			}

			if len(tokenStr) > 7 && tokenStr[:7] == "Bearer " {
				tokenStr = tokenStr[7:]
			}

			claims, err := tokenSvc.ValidateToken(tokenStr)
			if err != nil {
				utils.RespondUnauthorized(w, "invalid token")
				return
			}

			ctx := context.WithValue(r.Context(), UserIDKey, claims.UserID)
			ctx = context.WithValue(ctx, UserEmailKey, claims.Email)
			ctx = context.WithValue(ctx, UserRoleKey, claims.Role)
			ctx = context.WithValue(ctx, TenantClaimKey, claims.TenantID)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

func AuthenticationWithDenylist(tokenSvc domain.TokenService, denylistChecker func(ctx context.Context, jti string) bool) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			tokenStr := r.Header.Get("Authorization")
			if tokenStr == "" {
				utils.RespondUnauthorized(w, "missing token")
				return
			}

			if len(tokenStr) > 7 && tokenStr[:7] == "Bearer " {
				tokenStr = tokenStr[7:]
			}

			claims, err := tokenSvc.ValidateToken(tokenStr)
			if err != nil {
				utils.RespondUnauthorized(w, "invalid token")
				return
			}

			if claims.JTI != "" && denylistChecker(r.Context(), claims.JTI) {
				utils.RespondUnauthorized(w, "token has been revoked")
				return
			}

			ctx := context.WithValue(r.Context(), UserIDKey, claims.UserID)
			ctx = context.WithValue(ctx, UserEmailKey, claims.Email)
			ctx = context.WithValue(ctx, UserRoleKey, claims.Role)
			ctx = context.WithValue(ctx, TenantClaimKey, claims.TenantID)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

func CORS(cfg *config.CORSConfig) func(http.Handler) http.Handler {
	c := cors.New(cors.Options{
		AllowedOrigins:   cfg.AllowedOrigins,
		AllowedMethods:   cfg.AllowedMethods,
		AllowedHeaders:   cfg.AllowedHeaders,
		AllowCredentials: cfg.AllowCredentials,
		MaxAge:           cfg.MaxAge,
	})
	return c.Handler
}

func Timeout(timeout time.Duration) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ctx, cancel := context.WithTimeout(r.Context(), timeout)
			defer cancel()
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

func GetRequestID(ctx context.Context) string {
	if v, ok := ctx.Value(RequestIDKey).(string); ok {
		return v
	}
	return ""
}

func GetUserID(ctx context.Context) string {
	if v, ok := ctx.Value(UserIDKey).(string); ok {
		return v
	}
	return ""
}

func GetUserEmail(ctx context.Context) string {
	if v, ok := ctx.Value(UserEmailKey).(string); ok {
		return v
	}
	return ""
}

func GetUserRole(ctx context.Context) string {
	if v, ok := ctx.Value(UserRoleKey).(string); ok {
		return v
	}
	return ""
}

func GetTenantID(ctx context.Context) string {
	if v, ok := ctx.Value(TenantIDKey).(string); ok {
		return v
	}
	return ""
}

type Authorizer interface {
	Enforce(userID uuid.UUID, resource, action string) (bool, error)
}

func Authorization(authorizer Authorizer, resource, action string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			userID := GetUserID(r.Context())
			if userID == "" {
				utils.RespondUnauthorized(w, "user not authenticated")
				return
			}

			uid, err := uuid.Parse(userID)
			if err != nil {
				utils.RespondUnauthorized(w, "invalid user ID")
				return
			}

			allowed, err := authorizer.Enforce(uid, resource, action)
			if err != nil {
				utils.RespondInternalError(w, "authorization check failed")
				return
			}

			if !allowed {
				utils.RespondForbidden(w, "FORBIDDEN", "insufficient permissions")
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

type responseWriter struct {
	http.ResponseWriter
	statusCode int
}

func (rw *responseWriter) WriteHeader(code int) {
	rw.statusCode = code
	rw.ResponseWriter.WriteHeader(code)
}
