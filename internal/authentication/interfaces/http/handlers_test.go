package http

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/IDTS-LAB/go-codebase/internal/authentication/application/command"
	"github.com/IDTS-LAB/go-codebase/internal/shared/cqrs"
	"github.com/IDTS-LAB/go-codebase/internal/shared/validator"
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
