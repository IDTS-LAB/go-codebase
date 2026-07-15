package http

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/IDTS-LAB/go-codebase/internal/authentication/application/command"
	"github.com/IDTS-LAB/go-codebase/internal/authentication/application/query"
	"github.com/IDTS-LAB/go-codebase/internal/authentication/domain/entity"
	"github.com/IDTS-LAB/go-codebase/internal/shared/cqrs"
	"github.com/IDTS-LAB/go-codebase/internal/shared/middleware"
	"github.com/IDTS-LAB/go-codebase/internal/shared/validator"
	"github.com/google/uuid"
)

type mockHandler struct {
	result any
	err    error
}

func (h *mockHandler) Handle(ctx context.Context, _ any) (any, error) {
	return h.result, h.err
}

func TestVerifyEmail_MissingToken(t *testing.T) {
	cmdBus := cqrs.NewInMemoryCommandBus()
	qBus := cqrs.NewInMemoryQueryBus()
	h := NewHandler(cmdBus, qBus, validator.New())

	r := httptest.NewRequest(http.MethodGet, "/auth/verify-email", nil)
	w := httptest.NewRecorder()
	h.VerifyEmail(w, r)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected status 400, got %d", w.Code)
	}
	var resp map[string]interface{}
	json.NewDecoder(w.Body).Decode(&resp)
	if resp["success"].(bool) {
		t.Error("expected success false")
	}
	if resp["data"] != nil {
		t.Error("expected data null")
	}
	if resp["error"].(map[string]interface{})["code"] != "VALIDATION_ERROR" {
		t.Errorf("expected VALIDATION_ERROR, got %v", resp["error"].(map[string]interface{})["code"])
	}
}

func TestVerifyEmail_Success(t *testing.T) {
	cmdBus := cqrs.NewInMemoryCommandBus()
	qBus := cqrs.NewInMemoryQueryBus()
	h := NewHandler(cmdBus, qBus, validator.New())

	cmdBus.Register(command.VerifyEmailCommand{}, &mockHandler{
		result: map[string]string{"message": "email verified successfully"},
	})

	r := httptest.NewRequest(http.MethodGet, "/auth/verify-email?token=valid-token", nil)
	w := httptest.NewRecorder()
	h.VerifyEmail(w, r)

	if w.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", w.Code)
	}

	var resp map[string]interface{}
	json.NewDecoder(w.Body).Decode(&resp)
	if !resp["success"].(bool) {
		t.Error("expected success true")
	}
	if resp["data"] == nil {
		t.Error("expected data not null")
	}
	if resp["meta"] != nil {
		t.Error("expected meta null")
	}
}

func TestVerifyEmail_InvalidToken(t *testing.T) {
	cmdBus := cqrs.NewInMemoryCommandBus()
	qBus := cqrs.NewInMemoryQueryBus()
	h := NewHandler(cmdBus, qBus, validator.New())

	cmdBus.Register(command.VerifyEmailCommand{}, &mockHandler{
		err: command.ErrInvalidVerifyToken,
	})

	r := httptest.NewRequest(http.MethodGet, "/auth/verify-email?token=does-not-exist", nil)
	w := httptest.NewRecorder()
	h.VerifyEmail(w, r)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected status 400, got %d", w.Code)
	}
	var resp map[string]interface{}
	json.NewDecoder(w.Body).Decode(&resp)
	if resp["success"].(bool) {
		t.Error("expected success false")
	}
	if resp["data"] != nil {
		t.Error("expected data null")
	}
	if resp["error"].(map[string]interface{})["code"] != "VALIDATION_ERROR" {
		t.Errorf("expected VALIDATION_ERROR, got %v", resp["error"].(map[string]interface{})["code"])
	}
}

func TestForgotPassword_Success(t *testing.T) {
	cmdBus := cqrs.NewInMemoryCommandBus()
	qBus := cqrs.NewInMemoryQueryBus()
	h := NewHandler(cmdBus, qBus, validator.New())

	cmdBus.Register(command.ForgotPasswordCommand{}, &mockHandler{})

	body := map[string]string{"email": "forgot@example.com"}
	b, _ := json.Marshal(body)
	r := httptest.NewRequest(http.MethodPost, "/auth/forgot-password", bytes.NewReader(b))
	r.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	h.ForgotPassword(w, r)

	if w.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", w.Code)
	}
	var resp map[string]interface{}
	json.NewDecoder(w.Body).Decode(&resp)
	if !resp["success"].(bool) {
		t.Error("expected success true")
	}
	if resp["data"] == nil {
		t.Error("expected data not null")
	}
	if resp["meta"] != nil {
		t.Error("expected meta null")
	}
}

func TestForgotPassword_InvalidBody(t *testing.T) {
	cmdBus := cqrs.NewInMemoryCommandBus()
	qBus := cqrs.NewInMemoryQueryBus()
	h := NewHandler(cmdBus, qBus, validator.New())

	r := httptest.NewRequest(http.MethodPost, "/auth/forgot-password", bytes.NewReader([]byte("{not json")))
	r.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	h.ForgotPassword(w, r)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected status 400, got %d", w.Code)
	}
	var resp map[string]interface{}
	json.NewDecoder(w.Body).Decode(&resp)
	if resp["success"].(bool) {
		t.Error("expected success false")
	}
	if resp["data"] != nil {
		t.Error("expected data null")
	}
	if resp["error"].(map[string]interface{})["code"] != "VALIDATION_ERROR" {
		t.Errorf("expected VALIDATION_ERROR, got %v", resp["error"].(map[string]interface{})["code"])
	}
}

func TestResetPassword_Success(t *testing.T) {
	cmdBus := cqrs.NewInMemoryCommandBus()
	qBus := cqrs.NewInMemoryQueryBus()
	h := NewHandler(cmdBus, qBus, validator.New())

	cmdBus.Register(command.ResetPasswordCommand{}, &mockHandler{})

	body := map[string]string{"token": "valid-reset-token", "new_password": "newpassword123"}
	b, _ := json.Marshal(body)
	r := httptest.NewRequest(http.MethodPost, "/auth/reset-password", bytes.NewReader(b))
	r.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	h.ResetPassword(w, r)

	if w.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", w.Code)
	}

	var resp map[string]interface{}
	json.NewDecoder(w.Body).Decode(&resp)
	if !resp["success"].(bool) {
		t.Error("expected success true")
	}
	if resp["data"] == nil {
		t.Error("expected data not null")
	}
	if resp["meta"] != nil {
		t.Error("expected meta null")
	}
}

func TestResetPassword_InvalidBody(t *testing.T) {
	cmdBus := cqrs.NewInMemoryCommandBus()
	qBus := cqrs.NewInMemoryQueryBus()
	h := NewHandler(cmdBus, qBus, validator.New())

	r := httptest.NewRequest(http.MethodPost, "/auth/reset-password", bytes.NewReader([]byte("{not json")))
	r.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	h.ResetPassword(w, r)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected status 400, got %d", w.Code)
	}
	var resp map[string]interface{}
	json.NewDecoder(w.Body).Decode(&resp)
	if resp["success"].(bool) {
		t.Error("expected success false")
	}
	if resp["data"] != nil {
		t.Error("expected data null")
	}
	if resp["error"].(map[string]interface{})["code"] != "VALIDATION_ERROR" {
		t.Errorf("expected VALIDATION_ERROR, got %v", resp["error"].(map[string]interface{})["code"])
	}
}

func TestResendVerification_Success(t *testing.T) {
	cmdBus := cqrs.NewInMemoryCommandBus()
	qBus := cqrs.NewInMemoryQueryBus()
	h := NewHandler(cmdBus, qBus, validator.New())

	cmdBus.Register(command.ResendVerificationCommand{}, &mockHandler{})

	body := map[string]string{"email": "resend@example.com"}
	b, _ := json.Marshal(body)
	r := httptest.NewRequest(http.MethodPost, "/auth/resend-verification", bytes.NewReader(b))
	r.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	h.ResendVerification(w, r)

	if w.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", w.Code)
	}
	var resp map[string]interface{}
	json.NewDecoder(w.Body).Decode(&resp)
	if !resp["success"].(bool) {
		t.Error("expected success true")
	}
	if resp["data"] == nil {
		t.Error("expected data not null")
	}
	if resp["meta"] != nil {
		t.Error("expected meta null")
	}
}

func TestRegister_Success(t *testing.T) {
	cmdBus := cqrs.NewInMemoryCommandBus()
	qBus := cqrs.NewInMemoryQueryBus()
	h := NewHandler(cmdBus, qBus, validator.New())

	cmdBus.Register(command.RegisterUserCommand{}, &mockHandler{
		result: &entity.User{Email: "test@example.com"},
	})

	body := map[string]string{
		"email":    "test@example.com",
		"password": "password123",
		"name":     "Test User",
	}
	b, _ := json.Marshal(body)
	r := httptest.NewRequest(http.MethodPost, "/auth/register", bytes.NewReader(b))
	r.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	h.Register(w, r)

	if w.Code != http.StatusCreated {
		t.Fatalf("expected status 201, got %d", w.Code)
	}
	var resp map[string]interface{}
	json.NewDecoder(w.Body).Decode(&resp)
	if !resp["success"].(bool) {
		t.Error("expected success true")
	}
	if resp["data"] == nil {
		t.Error("expected data not null")
	}
	if resp["meta"] != nil {
		t.Error("expected meta null")
	}
}

func TestRegister_ValidationError(t *testing.T) {
	cmdBus := cqrs.NewInMemoryCommandBus()
	qBus := cqrs.NewInMemoryQueryBus()
	h := NewHandler(cmdBus, qBus, validator.New())

	body := map[string]string{
		"email":    "",
		"password": "password123",
		"name":     "Test User",
	}
	b, _ := json.Marshal(body)
	r := httptest.NewRequest(http.MethodPost, "/auth/register", bytes.NewReader(b))
	r.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	h.Register(w, r)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected status 400, got %d", w.Code)
	}
	var resp map[string]interface{}
	json.NewDecoder(w.Body).Decode(&resp)
	if resp["success"].(bool) {
		t.Error("expected success false")
	}
	if resp["data"] != nil {
		t.Error("expected data null")
	}
}

func TestRegister_InvalidBody(t *testing.T) {
	cmdBus := cqrs.NewInMemoryCommandBus()
	qBus := cqrs.NewInMemoryQueryBus()
	h := NewHandler(cmdBus, qBus, validator.New())

	r := httptest.NewRequest(http.MethodPost, "/auth/register", bytes.NewReader([]byte("{not json")))
	r.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	h.Register(w, r)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected status 400, got %d", w.Code)
	}
	var resp map[string]interface{}
	json.NewDecoder(w.Body).Decode(&resp)
	if resp["success"].(bool) {
		t.Error("expected success false")
	}
	if resp["data"] != nil {
		t.Error("expected data null")
	}
}

func TestRegister_EmailAlreadyExists(t *testing.T) {
	cmdBus := cqrs.NewInMemoryCommandBus()
	qBus := cqrs.NewInMemoryQueryBus()
	h := NewHandler(cmdBus, qBus, validator.New())

	cmdBus.Register(command.RegisterUserCommand{}, &mockHandler{
		err: command.ErrEmailAlreadyExists,
	})

	body := map[string]string{
		"email":    "existing@example.com",
		"password": "password123",
		"name":     "Test User",
	}
	b, _ := json.Marshal(body)
	r := httptest.NewRequest(http.MethodPost, "/auth/register", bytes.NewReader(b))
	r.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	h.Register(w, r)

	if w.Code != http.StatusConflict {
		t.Fatalf("expected status 409, got %d", w.Code)
	}
	var resp map[string]interface{}
	json.NewDecoder(w.Body).Decode(&resp)
	if resp["success"].(bool) {
		t.Error("expected success false")
	}
	if resp["data"] != nil {
		t.Error("expected data null")
	}
}

func TestLogin_Success(t *testing.T) {
	cmdBus := cqrs.NewInMemoryCommandBus()
	qBus := cqrs.NewInMemoryQueryBus()
	h := NewHandler(cmdBus, qBus, validator.New())

	qBus.Register(query.LoginQuery{}, &mockHandler{
		result: &entity.User{},
	})
	cmdBus.Register(command.GenerateTokensCommand{}, &mockHandler{
		result: &command.TokenPair{
			AccessToken:  "access-token",
			RefreshToken: "refresh-token",
			ExpiresIn:    900,
		},
	})

	body := map[string]string{
		"email":    "test@example.com",
		"password": "password123",
	}
	b, _ := json.Marshal(body)
	r := httptest.NewRequest(http.MethodPost, "/auth/login", bytes.NewReader(b))
	r.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	h.Login(w, r)

	if w.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", w.Code)
	}
	var resp map[string]interface{}
	json.NewDecoder(w.Body).Decode(&resp)
	if !resp["success"].(bool) {
		t.Error("expected success true")
	}
	if resp["data"] == nil {
		t.Error("expected data not null")
	}
}

func TestLogin_ValidationError(t *testing.T) {
	cmdBus := cqrs.NewInMemoryCommandBus()
	qBus := cqrs.NewInMemoryQueryBus()
	h := NewHandler(cmdBus, qBus, validator.New())

	body := map[string]string{
		"email":    "",
		"password": "password123",
	}
	b, _ := json.Marshal(body)
	r := httptest.NewRequest(http.MethodPost, "/auth/login", bytes.NewReader(b))
	r.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	h.Login(w, r)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected status 400, got %d", w.Code)
	}
	var resp map[string]interface{}
	json.NewDecoder(w.Body).Decode(&resp)
	if resp["success"].(bool) {
		t.Error("expected success false")
	}
	if resp["data"] != nil {
		t.Error("expected data null")
	}
}

func TestLogin_InvalidBody(t *testing.T) {
	cmdBus := cqrs.NewInMemoryCommandBus()
	qBus := cqrs.NewInMemoryQueryBus()
	h := NewHandler(cmdBus, qBus, validator.New())

	r := httptest.NewRequest(http.MethodPost, "/auth/login", bytes.NewReader([]byte("{not json")))
	r.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	h.Login(w, r)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected status 400, got %d", w.Code)
	}
	var resp map[string]interface{}
	json.NewDecoder(w.Body).Decode(&resp)
	if resp["success"].(bool) {
		t.Error("expected success false")
	}
	if resp["data"] != nil {
		t.Error("expected data null")
	}
}

func TestLogin_InvalidCredentials(t *testing.T) {
	cmdBus := cqrs.NewInMemoryCommandBus()
	qBus := cqrs.NewInMemoryQueryBus()
	h := NewHandler(cmdBus, qBus, validator.New())

	qBus.Register(query.LoginQuery{}, &mockHandler{
		err: query.ErrInvalidCredentials,
	})

	body := map[string]string{
		"email":    "wrong@example.com",
		"password": "wrongpassword",
	}
	b, _ := json.Marshal(body)
	r := httptest.NewRequest(http.MethodPost, "/auth/login", bytes.NewReader(b))
	r.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	h.Login(w, r)

	if w.Code != http.StatusUnauthorized {
		t.Fatalf("expected status 401, got %d", w.Code)
	}
	var resp map[string]interface{}
	json.NewDecoder(w.Body).Decode(&resp)
	if resp["success"].(bool) {
		t.Error("expected success false")
	}
	if resp["data"] != nil {
		t.Error("expected data null")
	}
}

func TestLogin_AccountDisabled(t *testing.T) {
	cmdBus := cqrs.NewInMemoryCommandBus()
	qBus := cqrs.NewInMemoryQueryBus()
	h := NewHandler(cmdBus, qBus, validator.New())

	qBus.Register(query.LoginQuery{}, &mockHandler{
		err: query.ErrAccountDisabled,
	})

	body := map[string]string{
		"email":    "disabled@example.com",
		"password": "password123",
	}
	b, _ := json.Marshal(body)
	r := httptest.NewRequest(http.MethodPost, "/auth/login", bytes.NewReader(b))
	r.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	h.Login(w, r)

	if w.Code != http.StatusUnauthorized {
		t.Fatalf("expected status 401, got %d", w.Code)
	}
	var resp map[string]interface{}
	json.NewDecoder(w.Body).Decode(&resp)
	if resp["success"].(bool) {
		t.Error("expected success false")
	}
	if resp["data"] != nil {
		t.Error("expected data null")
	}
}

func TestLogin_AccountLocked(t *testing.T) {
	cmdBus := cqrs.NewInMemoryCommandBus()
	qBus := cqrs.NewInMemoryQueryBus()
	h := NewHandler(cmdBus, qBus, validator.New())

	qBus.Register(query.LoginQuery{}, &mockHandler{
		err: query.ErrAccountLocked,
	})

	body := map[string]string{
		"email":    "locked@example.com",
		"password": "password123",
	}
	b, _ := json.Marshal(body)
	r := httptest.NewRequest(http.MethodPost, "/auth/login", bytes.NewReader(b))
	r.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	h.Login(w, r)

	if w.Code != http.StatusForbidden {
		t.Fatalf("expected status 403, got %d", w.Code)
	}
	var resp map[string]interface{}
	json.NewDecoder(w.Body).Decode(&resp)
	if resp["success"].(bool) {
		t.Error("expected success false")
	}
	if resp["data"] != nil {
		t.Error("expected data null")
	}
	if resp["error"].(map[string]interface{})["code"] != "ACCOUNT_LOCKED" {
		t.Errorf("expected ACCOUNT_LOCKED, got %v", resp["error"].(map[string]interface{})["code"])
	}
}

func TestLogin_EmailNotVerified(t *testing.T) {
	cmdBus := cqrs.NewInMemoryCommandBus()
	qBus := cqrs.NewInMemoryQueryBus()
	h := NewHandler(cmdBus, qBus, validator.New())

	qBus.Register(query.LoginQuery{}, &mockHandler{
		err: query.ErrEmailNotVerified,
	})

	body := map[string]string{
		"email":    "unverified@example.com",
		"password": "password123",
	}
	b, _ := json.Marshal(body)
	r := httptest.NewRequest(http.MethodPost, "/auth/login", bytes.NewReader(b))
	r.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	h.Login(w, r)

	if w.Code != http.StatusForbidden {
		t.Fatalf("expected status 403, got %d", w.Code)
	}
	var resp map[string]interface{}
	json.NewDecoder(w.Body).Decode(&resp)
	if resp["success"].(bool) {
		t.Error("expected success false")
	}
	if resp["data"] != nil {
		t.Error("expected data null")
	}
	if resp["error"].(map[string]interface{})["code"] != "EMAIL_NOT_VERIFIED" {
		t.Errorf("expected EMAIL_NOT_VERIFIED, got %v", resp["error"].(map[string]interface{})["code"])
	}
}

func TestRefreshToken_Success(t *testing.T) {
	cmdBus := cqrs.NewInMemoryCommandBus()
	qBus := cqrs.NewInMemoryQueryBus()
	h := NewHandler(cmdBus, qBus, validator.New())

	cmdBus.Register(command.RefreshTokenCommand{}, &mockHandler{
		result: &command.TokenPair{
			AccessToken:  "new-access-token",
			RefreshToken: "new-refresh-token",
			ExpiresIn:    900,
		},
	})

	body := map[string]string{"refresh_token": "valid-refresh-token"}
	b, _ := json.Marshal(body)
	r := httptest.NewRequest(http.MethodPost, "/auth/refresh", bytes.NewReader(b))
	r.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	h.RefreshToken(w, r)

	if w.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", w.Code)
	}
	var resp map[string]interface{}
	json.NewDecoder(w.Body).Decode(&resp)
	if !resp["success"].(bool) {
		t.Error("expected success true")
	}
	if resp["data"] == nil {
		t.Error("expected data not null")
	}
}

func TestRefreshToken_ValidationError(t *testing.T) {
	cmdBus := cqrs.NewInMemoryCommandBus()
	qBus := cqrs.NewInMemoryQueryBus()
	h := NewHandler(cmdBus, qBus, validator.New())

	body := map[string]string{"refresh_token": ""}
	b, _ := json.Marshal(body)
	r := httptest.NewRequest(http.MethodPost, "/auth/refresh", bytes.NewReader(b))
	r.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	h.RefreshToken(w, r)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected status 400, got %d", w.Code)
	}
	var resp map[string]interface{}
	json.NewDecoder(w.Body).Decode(&resp)
	if resp["success"].(bool) {
		t.Error("expected success false")
	}
	if resp["data"] != nil {
		t.Error("expected data null")
	}
}

func TestRefreshToken_InvalidBody(t *testing.T) {
	cmdBus := cqrs.NewInMemoryCommandBus()
	qBus := cqrs.NewInMemoryQueryBus()
	h := NewHandler(cmdBus, qBus, validator.New())

	r := httptest.NewRequest(http.MethodPost, "/auth/refresh", bytes.NewReader([]byte("{not json")))
	r.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	h.RefreshToken(w, r)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected status 400, got %d", w.Code)
	}
	var resp map[string]interface{}
	json.NewDecoder(w.Body).Decode(&resp)
	if resp["success"].(bool) {
		t.Error("expected success false")
	}
	if resp["data"] != nil {
		t.Error("expected data null")
	}
}

func TestRefreshToken_InvalidToken(t *testing.T) {
	cmdBus := cqrs.NewInMemoryCommandBus()
	qBus := cqrs.NewInMemoryQueryBus()
	h := NewHandler(cmdBus, qBus, validator.New())

	cmdBus.Register(command.RefreshTokenCommand{}, &mockHandler{
		err: command.ErrInvalidRefreshToken,
	})

	body := map[string]string{"refresh_token": "invalid-token"}
	b, _ := json.Marshal(body)
	r := httptest.NewRequest(http.MethodPost, "/auth/refresh", bytes.NewReader(b))
	r.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	h.RefreshToken(w, r)

	if w.Code != http.StatusUnauthorized {
		t.Fatalf("expected status 401, got %d", w.Code)
	}
	var resp map[string]interface{}
	json.NewDecoder(w.Body).Decode(&resp)
	if resp["success"].(bool) {
		t.Error("expected success false")
	}
	if resp["data"] != nil {
		t.Error("expected data null")
	}
}

func TestLogout_Success(t *testing.T) {
	cmdBus := cqrs.NewInMemoryCommandBus()
	qBus := cqrs.NewInMemoryQueryBus()
	h := NewHandler(cmdBus, qBus, validator.New())

	cmdBus.Register(command.LogoutCommand{}, &mockHandler{})

	body := map[string]string{"refresh_token": "valid-refresh-token"}
	b, _ := json.Marshal(body)
	r := httptest.NewRequest(http.MethodPost, "/auth/logout", bytes.NewReader(b))
	r.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	h.Logout(w, r)

	if w.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", w.Code)
	}
	var resp map[string]interface{}
	json.NewDecoder(w.Body).Decode(&resp)
	if !resp["success"].(bool) {
		t.Error("expected success true")
	}
	if resp["data"] == nil {
		t.Error("expected data not null")
	}
}

func TestLogout_InvalidBody(t *testing.T) {
	cmdBus := cqrs.NewInMemoryCommandBus()
	qBus := cqrs.NewInMemoryQueryBus()
	h := NewHandler(cmdBus, qBus, validator.New())

	r := httptest.NewRequest(http.MethodPost, "/auth/logout", bytes.NewReader([]byte("{not json")))
	r.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	h.Logout(w, r)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected status 400, got %d", w.Code)
	}
	var resp map[string]interface{}
	json.NewDecoder(w.Body).Decode(&resp)
	if resp["success"].(bool) {
		t.Error("expected success false")
	}
	if resp["data"] != nil {
		t.Error("expected data null")
	}
}

func TestLogoutAll_Success(t *testing.T) {
	cmdBus := cqrs.NewInMemoryCommandBus()
	qBus := cqrs.NewInMemoryQueryBus()
	h := NewHandler(cmdBus, qBus, validator.New())

	cmdBus.Register(command.LogoutAllCommand{}, &mockHandler{})

	userID := uuid.New()
	r := httptest.NewRequest(http.MethodPost, "/auth/logout-all", nil)
	r = r.WithContext(context.WithValue(r.Context(), middleware.UserIDKey, userID.String()))
	w := httptest.NewRecorder()
	h.LogoutAll(w, r)

	if w.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", w.Code)
	}
	var resp map[string]interface{}
	json.NewDecoder(w.Body).Decode(&resp)
	if !resp["success"].(bool) {
		t.Error("expected success true")
	}
	if resp["data"] == nil {
		t.Error("expected data not null")
	}
}

func TestLogoutAll_NoUserID(t *testing.T) {
	cmdBus := cqrs.NewInMemoryCommandBus()
	qBus := cqrs.NewInMemoryQueryBus()
	h := NewHandler(cmdBus, qBus, validator.New())

	r := httptest.NewRequest(http.MethodPost, "/auth/logout-all", nil)
	w := httptest.NewRecorder()
	h.LogoutAll(w, r)

	if w.Code != http.StatusUnauthorized {
		t.Fatalf("expected status 401, got %d", w.Code)
	}
	var resp map[string]interface{}
	json.NewDecoder(w.Body).Decode(&resp)
	if resp["success"].(bool) {
		t.Error("expected success false")
	}
	if resp["data"] != nil {
		t.Error("expected data null")
	}
}

func TestLogoutAll_InvalidUserID(t *testing.T) {
	cmdBus := cqrs.NewInMemoryCommandBus()
	qBus := cqrs.NewInMemoryQueryBus()
	h := NewHandler(cmdBus, qBus, validator.New())

	r := httptest.NewRequest(http.MethodPost, "/auth/logout-all", nil)
	r = r.WithContext(context.WithValue(r.Context(), middleware.UserIDKey, "not-a-uuid"))
	w := httptest.NewRecorder()
	h.LogoutAll(w, r)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected status 400, got %d", w.Code)
	}
	var resp map[string]interface{}
	json.NewDecoder(w.Body).Decode(&resp)
	if resp["success"].(bool) {
		t.Error("expected success false")
	}
	if resp["data"] != nil {
		t.Error("expected data null")
	}
}

func TestMe_Success(t *testing.T) {
	cmdBus := cqrs.NewInMemoryCommandBus()
	qBus := cqrs.NewInMemoryQueryBus()
	h := NewHandler(cmdBus, qBus, validator.New())

	userID := uuid.New()
	r := httptest.NewRequest(http.MethodGet, "/auth/me", nil)
	ctx := context.WithValue(r.Context(), middleware.UserIDKey, userID.String())
	ctx = context.WithValue(ctx, middleware.UserEmailKey, "test@example.com")
	r = r.WithContext(ctx)
	w := httptest.NewRecorder()
	h.Me(w, r)

	if w.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", w.Code)
	}
	var resp map[string]interface{}
	json.NewDecoder(w.Body).Decode(&resp)
	if !resp["success"].(bool) {
		t.Error("expected success true")
	}
	if resp["data"] == nil {
		t.Error("expected data not null")
	}
}

func TestMe_NoUserID(t *testing.T) {
	cmdBus := cqrs.NewInMemoryCommandBus()
	qBus := cqrs.NewInMemoryQueryBus()
	h := NewHandler(cmdBus, qBus, validator.New())

	r := httptest.NewRequest(http.MethodGet, "/auth/me", nil)
	w := httptest.NewRecorder()
	h.Me(w, r)

	if w.Code != http.StatusUnauthorized {
		t.Fatalf("expected status 401, got %d", w.Code)
	}
	var resp map[string]interface{}
	json.NewDecoder(w.Body).Decode(&resp)
	if resp["success"].(bool) {
		t.Error("expected success false")
	}
	if resp["data"] != nil {
		t.Error("expected data null")
	}
}

func TestRegister_BusError(t *testing.T) {
	cmdBus := cqrs.NewInMemoryCommandBus()
	qBus := cqrs.NewInMemoryQueryBus()
	h := NewHandler(cmdBus, qBus, validator.New())

	cmdBus.Register(command.RegisterUserCommand{}, &mockHandler{
		err: errors.New("db error"),
	})

	body := map[string]string{
		"email":    "test@example.com",
		"password": "password123",
		"name":     "Test User",
	}
	b, _ := json.Marshal(body)
	r := httptest.NewRequest(http.MethodPost, "/auth/register", bytes.NewReader(b))
	r.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	h.Register(w, r)

	if w.Code != http.StatusInternalServerError {
		t.Fatalf("expected status 500, got %d", w.Code)
	}
	var resp map[string]interface{}
	json.NewDecoder(w.Body).Decode(&resp)
	if resp["success"].(bool) {
		t.Error("expected success false")
	}
	if resp["data"] != nil {
		t.Error("expected data nil")
	}
}

func TestLogin_UnexpectedError(t *testing.T) {
	cmdBus := cqrs.NewInMemoryCommandBus()
	qBus := cqrs.NewInMemoryQueryBus()
	h := NewHandler(cmdBus, qBus, validator.New())

	qBus.Register(query.LoginQuery{}, &mockHandler{
		err: errors.New("unexpected"),
	})

	body := map[string]string{
		"email":    "test@example.com",
		"password": "password123",
	}
	b, _ := json.Marshal(body)
	r := httptest.NewRequest(http.MethodPost, "/auth/login", bytes.NewReader(b))
	r.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	h.Login(w, r)

	if w.Code != http.StatusInternalServerError {
		t.Fatalf("expected status 500, got %d", w.Code)
	}
	var resp map[string]interface{}
	json.NewDecoder(w.Body).Decode(&resp)
	if resp["success"].(bool) {
		t.Error("expected success false")
	}
	if resp["data"] != nil {
		t.Error("expected data nil")
	}
}

func TestLogin_TokenGenerationError(t *testing.T) {
	cmdBus := cqrs.NewInMemoryCommandBus()
	qBus := cqrs.NewInMemoryQueryBus()
	h := NewHandler(cmdBus, qBus, validator.New())

	qBus.Register(query.LoginQuery{}, &mockHandler{
		result: &entity.User{},
	})
	cmdBus.Register(command.GenerateTokensCommand{}, &mockHandler{
		err: errors.New("token generation failed"),
	})

	body := map[string]string{
		"email":    "test@example.com",
		"password": "password123",
	}
	b, _ := json.Marshal(body)
	r := httptest.NewRequest(http.MethodPost, "/auth/login", bytes.NewReader(b))
	r.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	h.Login(w, r)

	if w.Code != http.StatusInternalServerError {
		t.Fatalf("expected status 500, got %d", w.Code)
	}
	var resp map[string]interface{}
	json.NewDecoder(w.Body).Decode(&resp)
	if resp["success"].(bool) {
		t.Error("expected success false")
	}
	if resp["data"] != nil {
		t.Error("expected data nil")
	}
}

func TestRefreshToken_BusError(t *testing.T) {
	cmdBus := cqrs.NewInMemoryCommandBus()
	qBus := cqrs.NewInMemoryQueryBus()
	h := NewHandler(cmdBus, qBus, validator.New())

	cmdBus.Register(command.RefreshTokenCommand{}, &mockHandler{
		err: errors.New("db error"),
	})

	body := map[string]string{"refresh_token": "some-refresh-token"}
	b, _ := json.Marshal(body)
	r := httptest.NewRequest(http.MethodPost, "/auth/refresh", bytes.NewReader(b))
	r.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	h.RefreshToken(w, r)

	if w.Code != http.StatusInternalServerError {
		t.Fatalf("expected status 500, got %d", w.Code)
	}
	var resp map[string]interface{}
	json.NewDecoder(w.Body).Decode(&resp)
	if resp["success"].(bool) {
		t.Error("expected success false")
	}
	if resp["data"] != nil {
		t.Error("expected data nil")
	}
}

func TestLogout_BusError(t *testing.T) {
	cmdBus := cqrs.NewInMemoryCommandBus()
	qBus := cqrs.NewInMemoryQueryBus()
	h := NewHandler(cmdBus, qBus, validator.New())

	cmdBus.Register(command.LogoutCommand{}, &mockHandler{
		err: errors.New("db error"),
	})

	body := map[string]string{"refresh_token": "some-refresh-token"}
	b, _ := json.Marshal(body)
	r := httptest.NewRequest(http.MethodPost, "/auth/logout", bytes.NewReader(b))
	r.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	h.Logout(w, r)

	if w.Code != http.StatusInternalServerError {
		t.Fatalf("expected status 500, got %d", w.Code)
	}
	var resp map[string]interface{}
	json.NewDecoder(w.Body).Decode(&resp)
	if resp["success"].(bool) {
		t.Error("expected success false")
	}
	if resp["data"] != nil {
		t.Error("expected data nil")
	}
}

func TestLogoutAll_BusError(t *testing.T) {
	cmdBus := cqrs.NewInMemoryCommandBus()
	qBus := cqrs.NewInMemoryQueryBus()
	h := NewHandler(cmdBus, qBus, validator.New())

	userID := uuid.New()
	cmdBus.Register(command.LogoutAllCommand{}, &mockHandler{
		err: errors.New("db error"),
	})

	r := httptest.NewRequest(http.MethodPost, "/auth/logout-all", nil)
	r = r.WithContext(context.WithValue(r.Context(), middleware.UserIDKey, userID.String()))
	w := httptest.NewRecorder()
	h.LogoutAll(w, r)

	if w.Code != http.StatusInternalServerError {
		t.Fatalf("expected status 500, got %d", w.Code)
	}
	var resp map[string]interface{}
	json.NewDecoder(w.Body).Decode(&resp)
	if resp["success"].(bool) {
		t.Error("expected success false")
	}
	if resp["data"] != nil {
		t.Error("expected data nil")
	}
}

func TestVerifyEmail_BusError(t *testing.T) {
	cmdBus := cqrs.NewInMemoryCommandBus()
	qBus := cqrs.NewInMemoryQueryBus()
	h := NewHandler(cmdBus, qBus, validator.New())

	cmdBus.Register(command.VerifyEmailCommand{}, &mockHandler{
		err: errors.New("unexpected error"),
	})

	r := httptest.NewRequest(http.MethodGet, "/auth/verify-email?token=some-token", nil)
	w := httptest.NewRecorder()
	h.VerifyEmail(w, r)

	if w.Code != http.StatusInternalServerError {
		t.Fatalf("expected status 500, got %d", w.Code)
	}
	var resp map[string]interface{}
	json.NewDecoder(w.Body).Decode(&resp)
	if resp["success"].(bool) {
		t.Error("expected success false")
	}
	if resp["data"] != nil {
		t.Error("expected data nil")
	}
}

func TestResetPassword_BusError(t *testing.T) {
	cmdBus := cqrs.NewInMemoryCommandBus()
	qBus := cqrs.NewInMemoryQueryBus()
	h := NewHandler(cmdBus, qBus, validator.New())

	cmdBus.Register(command.ResetPasswordCommand{}, &mockHandler{
		err: errors.New("unexpected error"),
	})

	body := map[string]string{"token": "some-token", "new_password": "newpassword123"}
	b, _ := json.Marshal(body)
	r := httptest.NewRequest(http.MethodPost, "/auth/reset-password", bytes.NewReader(b))
	r.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	h.ResetPassword(w, r)

	if w.Code != http.StatusInternalServerError {
		t.Fatalf("expected status 500, got %d", w.Code)
	}
	var resp map[string]interface{}
	json.NewDecoder(w.Body).Decode(&resp)
	if resp["success"].(bool) {
		t.Error("expected success false")
	}
	if resp["data"] != nil {
		t.Error("expected data nil")
	}
}
