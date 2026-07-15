package tests

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/IDTS-LAB/go-codebase/internal/authentication/application/command"
	"github.com/IDTS-LAB/go-codebase/internal/authentication/application/query"
	"github.com/IDTS-LAB/go-codebase/internal/authentication/domain/event"
	authRepo "github.com/IDTS-LAB/go-codebase/internal/authentication/infrastructure/persistence"
	httpHandler "github.com/IDTS-LAB/go-codebase/internal/authentication/interfaces/http"
	"github.com/IDTS-LAB/go-codebase/internal/infrastructure/auth"
	"github.com/IDTS-LAB/go-codebase/internal/shared/config"
	"github.com/IDTS-LAB/go-codebase/internal/shared/cqrs"
	"github.com/IDTS-LAB/go-codebase/internal/shared/events"
	"github.com/IDTS-LAB/go-codebase/internal/shared/utils"
	"github.com/IDTS-LAB/go-codebase/internal/shared/validator"
	"github.com/google/uuid"
)

type testHandler struct {
	handler  *httpHandler.Handler
	eventBus *events.InMemoryEventBus
}

func setupTestHandler(t *testing.T, db *sql.DB) *testHandler {
	t.Helper()

	cfg := &config.Config{}
	cfg.Auth.JWTSecret = "test-secret-for-integration-tests"
	cfg.Auth.JWTExpiration = 3600

	tokenService := auth.NewJWTTokenService(cfg)
	userRepo := authRepo.NewUserRepository(db)
	refreshRepo := authRepo.NewRefreshTokenRepository(db)
	bus := events.NewInMemoryEventBus()
	generateTokensHandler := command.NewGenerateTokensHandler(refreshRepo, tokenService)

	cmdBus := cqrs.NewInMemoryCommandBus()
	qryBus := cqrs.NewInMemoryQueryBus()

	cmdBus.Register(command.RegisterUserCommand{}, command.NewRegisterUserHandler(userRepo, bus))
	cmdBus.Register(command.GenerateTokensCommand{}, generateTokensHandler)
	cmdBus.Register(command.VerifyEmailCommand{}, command.NewVerifyEmailHandler(userRepo, bus))
	cmdBus.Register(command.ForgotPasswordCommand{}, command.NewForgotPasswordHandler(userRepo, bus))
	cmdBus.Register(command.ResetPasswordCommand{}, command.NewResetPasswordHandler(userRepo, refreshRepo))
	cmdBus.Register(command.RefreshTokenCommand{}, command.NewRefreshTokenHandler(refreshRepo, userRepo, generateTokensHandler))
	cmdBus.Register(command.LogoutCommand{}, command.NewLogoutHandler(refreshRepo))
	cmdBus.Register(command.LogoutAllCommand{}, command.NewLogoutAllHandler(refreshRepo))

	qryBus.Register(query.LoginQuery{}, query.NewLoginHandler(userRepo))

	v := validator.New()
	h := httpHandler.NewHandler(cmdBus, qryBus, v)

	return &testHandler{handler: h, eventBus: bus}
}

func postJSON(_ *testing.T, hf func(http.ResponseWriter, *http.Request), body []byte) *httptest.ResponseRecorder {
	rr := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	hf(rr, req)
	return rr
}

func getQuery(_ *testing.T, hf func(http.ResponseWriter, *http.Request), query string) *httptest.ResponseRecorder {
	rr := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/?"+query, nil)
	hf(rr, req)
	return rr
}

func registerAndVerify(t *testing.T, th *testHandler, email, password, name string) {
	t.Helper()
	tokenCh := make(chan string, 1)
	th.eventBus.Subscribe(event.UserRegisteredEvent, func(ctx context.Context, evt events.Event) error {
		payload := evt.Payload.(event.UserRegistered)
		tokenCh <- payload.VerificationToken
		return nil
	})
	body := []byte(fmt.Sprintf(`{"email":"%s","password":"%s","name":"%s"}`, email, password, name))
	rr := postJSON(t, th.handler.Register, body)
	if rr.Code != http.StatusCreated {
		t.Fatalf("register expected 201, got %d: %s", rr.Code, rr.Body.String())
	}
	verifyToken := <-tokenCh
	rr = getQuery(t, th.handler.VerifyEmail, "token="+verifyToken)
	if rr.Code != http.StatusOK {
		t.Fatalf("verify expected 200, got %d: %s", rr.Code, rr.Body.String())
	}
}

func decodeResponse(t *testing.T, rr *httptest.ResponseRecorder) utils.APIResponse {
	t.Helper()
	var resp utils.APIResponse
	if err := json.NewDecoder(rr.Body).Decode(&resp); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	return resp
}

func TestRegister_Success(t *testing.T) {
	th := setupTestHandler(t, db)

	email := "handler-register-success-" + uuid.New().String()[:8] + "@example.com"
	rr := postJSON(t, th.handler.Register, []byte(fmt.Sprintf(`{"email":"%s","password":"password123","name":"Test User"}`, email)))
	if rr.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d: %s", rr.Code, rr.Body.String())
	}
	resp := decodeResponse(t, rr)
	if !resp.Success {
		t.Error("expected success=true")
	}
}

func TestRegister_DuplicateEmail(t *testing.T) {
	email := "handler-register-dup-" + uuid.New().String()[:8] + "@example.com"
	th := setupTestHandler(t, db)

	body := []byte(fmt.Sprintf(`{"email":"%s","password":"password123","name":"First"}`, email))
	rr := postJSON(t, th.handler.Register, body)
	if rr.Code != http.StatusCreated {
		t.Fatalf("first register expected 201, got %d", rr.Code)
	}

	rr = postJSON(t, th.handler.Register, body)
	if rr.Code != http.StatusConflict {
		t.Fatalf("expected 409, got %d: %s", rr.Code, rr.Body.String())
	}
	resp := decodeResponse(t, rr)
	if resp.Success {
		t.Error("expected success=false")
	}
	if resp.Error == nil || resp.Error.Code != "CONFLICT" {
		t.Errorf("expected CONFLICT error code, got %+v", resp.Error)
	}
}

func TestRegister_InvalidBody(t *testing.T) {
	th := setupTestHandler(t, db)

	rr := postJSON(t, th.handler.Register, []byte(`not-json`))
	if rr.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d: %s", rr.Code, rr.Body.String())
	}
}

func TestRegister_ValidationError(t *testing.T) {
	th := setupTestHandler(t, db)

	rr := postJSON(t, th.handler.Register, []byte(`{"email":"","password":"password123","name":"Test"}`))
	if rr.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d: %s", rr.Code, rr.Body.String())
	}
	resp := decodeResponse(t, rr)
	if resp.Success {
		t.Error("expected success=false")
	}
}

func TestLogin_Success(t *testing.T) {
	email := "handler-login-success-" + uuid.New().String()[:8] + "@example.com"
	password := "password123"
	th := setupTestHandler(t, db)

	registerAndVerify(t, th, email, password, "Login Test")

	rr := postJSON(t, th.handler.Login, []byte(fmt.Sprintf(`{"email":"%s","password":"%s"}`, email, password)))
	if rr.Code != http.StatusOK {
		t.Fatalf("login expected 200, got %d: %s", rr.Code, rr.Body.String())
	}
	resp := decodeResponse(t, rr)
	if !resp.Success {
		t.Fatal("expected success=true")
	}
	data, ok := resp.Data.(map[string]interface{})
	if !ok {
		t.Fatal("data is not a map")
	}
	if data["access_token"] == "" {
		t.Error("expected non-empty access_token")
	}
	if data["refresh_token"] == "" {
		t.Error("expected non-empty refresh_token")
	}
	if data["token_type"] != "Bearer" {
		t.Errorf("expected Bearer token_type, got %v", data["token_type"])
	}
}

func TestLogin_InvalidCredentials(t *testing.T) {
	email := "handler-login-invalid-" + uuid.New().String()[:8] + "@example.com"
	password := "password123"
	th := setupTestHandler(t, db)

	registerAndVerify(t, th, email, password, "Login Invalid")

	rr := postJSON(t, th.handler.Login, []byte(fmt.Sprintf(`{"email":"%s","password":"wrongpass"}`, email)))
	if rr.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d: %s", rr.Code, rr.Body.String())
	}
	resp := decodeResponse(t, rr)
	if resp.Success {
		t.Error("expected success=false")
	}
}

func TestLogin_AccountDisabled(t *testing.T) {
	email := "handler-login-disabled-" + uuid.New().String()[:8] + "@example.com"
	password := "password123"
	th := setupTestHandler(t, db)

	registerAndVerify(t, th, email, password, "Disabled")

	userRepo := authRepo.NewUserRepository(db)
	user, err := userRepo.GetByEmail(context.Background(), email)
	if err != nil {
		t.Fatalf("get user: %v", err)
	}
	user.IsActive = false
	if err := userRepo.Update(context.Background(), user); err != nil {
		t.Fatalf("update user: %v", err)
	}

	rr := postJSON(t, th.handler.Login, []byte(fmt.Sprintf(`{"email":"%s","password":"%s"}`, email, password)))
	if rr.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d: %s", rr.Code, rr.Body.String())
	}
	resp := decodeResponse(t, rr)
	if resp.Success {
		t.Error("expected success=false")
	}
	if resp.Error == nil || resp.Error.Code != "UNAUTHORIZED" {
		t.Errorf("expected UNAUTHORIZED error, got %+v", resp.Error)
	}
}

func TestVerifyEmail(t *testing.T) {
	email := "handler-verify-email-" + uuid.New().String()[:8] + "@example.com"
	th := setupTestHandler(t, db)

	tokenCh := make(chan string, 1)
	th.eventBus.Subscribe(event.UserRegisteredEvent, func(ctx context.Context, evt events.Event) error {
		payload := evt.Payload.(event.UserRegistered)
		tokenCh <- payload.VerificationToken
		return nil
	})

	rr := postJSON(t, th.handler.Register, []byte(fmt.Sprintf(`{"email":"%s","password":"password123","name":"Verify"}`, email)))
	if rr.Code != http.StatusCreated {
		t.Fatalf("register expected 201, got %d", rr.Code)
	}

	verifyToken := <-tokenCh

	rr = getQuery(t, th.handler.VerifyEmail, "token="+verifyToken)
	if rr.Code != http.StatusOK {
		t.Fatalf("verify expected 200, got %d: %s", rr.Code, rr.Body.String())
	}
	resp := decodeResponse(t, rr)
	if !resp.Success {
		t.Error("expected success=true")
	}
}

func TestForgotPassword(t *testing.T) {
	email := "handler-forgot-password-" + uuid.New().String()[:8] + "@example.com"
	th := setupTestHandler(t, db)

	tokenCh := make(chan string, 1)
	th.eventBus.Subscribe(event.UserRegisteredEvent, func(ctx context.Context, evt events.Event) error {
		payload := evt.Payload.(event.UserRegistered)
		tokenCh <- payload.VerificationToken
		return nil
	})

	rr := postJSON(t, th.handler.Register, []byte(fmt.Sprintf(`{"email":"%s","password":"password123","name":"Forgot"}`, email)))
	if rr.Code != http.StatusCreated {
		t.Fatalf("register expected 201, got %d", rr.Code)
	}
	<-tokenCh

	rr = postJSON(t, th.handler.ForgotPassword, []byte(fmt.Sprintf(`{"email":"%s"}`, email)))
	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rr.Code, rr.Body.String())
	}
	resp := decodeResponse(t, rr)
	if !resp.Success {
		t.Error("expected success=true")
	}
}

func TestResetPassword(t *testing.T) {
	email := "handler-reset-password-" + uuid.New().String()[:8] + "@example.com"
	newPassword := "newpassword123"
	th := setupTestHandler(t, db)

	verifyCh := make(chan string, 1)
	th.eventBus.Subscribe(event.UserRegisteredEvent, func(ctx context.Context, evt events.Event) error {
		payload := evt.Payload.(event.UserRegistered)
		verifyCh <- payload.VerificationToken
		return nil
	})

	rr := postJSON(t, th.handler.Register, []byte(fmt.Sprintf(`{"email":"%s","password":"password123","name":"Reset"}`, email)))
	if rr.Code != http.StatusCreated {
		t.Fatalf("register expected 201, got %d", rr.Code)
	}
	<-verifyCh

	resetCh := make(chan string, 1)
	th.eventBus.Subscribe(event.PasswordResetRequestedEvent, func(ctx context.Context, evt events.Event) error {
		payload := evt.Payload.(event.PasswordResetRequested)
		resetCh <- payload.ResetToken
		return nil
	})

	rr = postJSON(t, th.handler.ForgotPassword, []byte(fmt.Sprintf(`{"email":"%s"}`, email)))
	if rr.Code != http.StatusOK {
		t.Fatalf("forgot password expected 200, got %d: %s", rr.Code, rr.Body.String())
	}

	resetToken := <-resetCh

	rr = postJSON(t, th.handler.ResetPassword, []byte(fmt.Sprintf(`{"token":"%s","new_password":"%s"}`, resetToken, newPassword)))
	if rr.Code != http.StatusOK {
		t.Fatalf("reset password expected 200, got %d: %s", rr.Code, rr.Body.String())
	}
	resp := decodeResponse(t, rr)
	if !resp.Success {
		t.Error("expected success=true")
	}
}
