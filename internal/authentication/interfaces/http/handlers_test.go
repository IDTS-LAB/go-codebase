package http

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"

	"github.com/IDTS-LAB/go-codebase/internal/authentication/application/service"
	"github.com/IDTS-LAB/go-codebase/internal/authentication/domain/entity"
	"github.com/IDTS-LAB/go-codebase/internal/core/domain"
	"github.com/IDTS-LAB/go-codebase/internal/shared/validator"
	"github.com/google/uuid"
)

type mockUserRepo struct {
	mu      sync.Mutex
	byID    map[uuid.UUID]*entity.User
	byEmail map[string]*entity.User
}

func newMockUserRepo() *mockUserRepo {
	return &mockUserRepo{
		byID:    make(map[uuid.UUID]*entity.User),
		byEmail: make(map[string]*entity.User),
	}
}

func (m *mockUserRepo) Create(_ context.Context, user *entity.User) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if _, ok := m.byEmail[user.Email]; ok {
		return errors.New("email already exists")
	}
	m.byID[user.ID] = user
	m.byEmail[user.Email] = user
	return nil
}

func (m *mockUserRepo) GetByID(_ context.Context, id uuid.UUID) (*entity.User, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	u, ok := m.byID[id]
	if !ok {
		return nil, errors.New("user not found")
	}
	return u, nil
}

func (m *mockUserRepo) GetByEmail(_ context.Context, email string) (*entity.User, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	u, ok := m.byEmail[email]
	if !ok {
		return nil, errors.New("user not found")
	}
	return u, nil
}

func (m *mockUserRepo) GetByVerifyToken(_ context.Context, token string) (*entity.User, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	for _, u := range m.byID {
		if u.EmailVerifyToken != nil && *u.EmailVerifyToken == token {
			return u, nil
		}
	}
	return nil, errors.New("user not found")
}

func (m *mockUserRepo) GetByResetToken(_ context.Context, token string) (*entity.User, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	for _, u := range m.byID {
		if u.PasswordResetToken != nil && *u.PasswordResetToken == token {
			return u, nil
		}
	}
	return nil, errors.New("user not found")
}

func (m *mockUserRepo) Update(_ context.Context, user *entity.User) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if _, ok := m.byID[user.ID]; !ok {
		return errors.New("user not found")
	}
	m.byID[user.ID] = user
	m.byEmail[user.Email] = user
	return nil
}

type mockRefreshRepo struct {
	tokens map[string]*entity.RefreshToken
}

func newMockRefreshRepo() *mockRefreshRepo {
	return &mockRefreshRepo{tokens: make(map[string]*entity.RefreshToken)}
}

func (m *mockRefreshRepo) Create(_ context.Context, t *entity.RefreshToken) error {
	m.tokens[t.Token] = t
	return nil
}

func (m *mockRefreshRepo) GetByToken(_ context.Context, token string) (*entity.RefreshToken, error) {
	t, ok := m.tokens[token]
	if !ok {
		return nil, errors.New("token not found")
	}
	return t, nil
}

func (m *mockRefreshRepo) GetByUserID(_ context.Context, _ uuid.UUID) ([]*entity.RefreshToken, error) {
	return nil, nil
}

func (m *mockRefreshRepo) Revoke(_ context.Context, token string) error {
	if t, ok := m.tokens[token]; ok {
		t.Revoke()
	}
	return nil
}

func (m *mockRefreshRepo) RevokeAllByUserID(_ context.Context, _ uuid.UUID) error {
	return nil
}

func (m *mockRefreshRepo) DeleteExpired(_ context.Context) error {
	return nil
}

type mockMailer struct {
	verifications []emailRecord
	resets        []emailRecord
	welcomes      []emailRecord
}

type emailRecord struct {
	to   string
	name string
	args []string
}

func newMockMailer() *mockMailer {
	return &mockMailer{}
}

func (m *mockMailer) SendVerification(to, name, token string) error {
	m.verifications = append(m.verifications, emailRecord{to, name, []string{token}})
	return nil
}

func (m *mockMailer) SendPasswordReset(to, name, token string) error {
	m.resets = append(m.resets, emailRecord{to, name, []string{token}})
	return nil
}

func (m *mockMailer) SendWelcome(to, name string) error {
	m.welcomes = append(m.welcomes, emailRecord{to, name, nil})
	return nil
}

func (m *mockMailer) SendInvite(to, name, inviterName string) error {
	return nil
}

type mockTokenService struct{}

func (mockTokenService) GenerateToken(_, _, _ string) (string, error) {
	return "mock-access-token", nil
}

func (mockTokenService) ValidateToken(_ string) (*domain.TokenClaims, error) {
	return &domain.TokenClaims{}, nil
}

func newTestHandler(repo *mockUserRepo, mailer *mockMailer) *Handler {
	if repo == nil {
		repo = newMockUserRepo()
	}
	if mailer == nil {
		mailer = newMockMailer()
	}
	svc := service.NewAuthenticationService(repo, newMockRefreshRepo(), mockTokenService{}, mailer)
	return NewHandler(svc, validator.New())
}

func TestVerifyEmail_MissingToken(t *testing.T) {
	h := newTestHandler(nil, nil)

	r := httptest.NewRequest(http.MethodGet, "/auth/verify-email", nil)
	w := httptest.NewRecorder()
	h.VerifyEmail(w, r)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected status 400, got %d", w.Code)
	}
}

func TestVerifyEmail_Success(t *testing.T) {
	repo := newMockUserRepo()
	mailer := newMockMailer()
	h := newTestHandler(repo, mailer)

	user, err := h.svc.Register(context.Background(), "verify@example.com", "password123", "User")
	if err != nil {
		t.Fatalf("Register failed: %v", err)
	}
	_ = user
	token := mailer.verifications[0].args[0]

	r := httptest.NewRequest(http.MethodGet, "/auth/verify-email?token="+token, nil)
	w := httptest.NewRecorder()
	h.VerifyEmail(w, r)

	if w.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", w.Code)
	}

	updated, _ := repo.GetByEmail(context.Background(), "verify@example.com")
	if !updated.EmailVerified {
		t.Error("user should be verified after request")
	}
}

func TestVerifyEmail_InvalidToken(t *testing.T) {
	h := newTestHandler(nil, nil)

	r := httptest.NewRequest(http.MethodGet, "/auth/verify-email?token=does-not-exist", nil)
	w := httptest.NewRecorder()
	h.VerifyEmail(w, r)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected status 400, got %d", w.Code)
	}
}

func TestForgotPassword_Success(t *testing.T) {
	repo := newMockUserRepo()
	mailer := newMockMailer()
	h := newTestHandler(repo, mailer)

	_, _ = h.svc.Register(context.Background(), "forgot@example.com", "password123", "User")

	body := map[string]string{"email": "forgot@example.com"}
	b, _ := json.Marshal(body)
	r := httptest.NewRequest(http.MethodPost, "/auth/forgot-password", bytes.NewReader(b))
	r.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	h.ForgotPassword(w, r)

	if w.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", w.Code)
	}
	if len(mailer.resets) != 1 {
		t.Errorf("expected 1 reset email, got %d", len(mailer.resets))
	}
}

func TestForgotPassword_InvalidBody(t *testing.T) {
	h := newTestHandler(nil, nil)

	r := httptest.NewRequest(http.MethodPost, "/auth/forgot-password", bytes.NewReader([]byte("{not json")))
	r.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	h.ForgotPassword(w, r)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected status 400, got %d", w.Code)
	}
}

func TestResetPassword_Success(t *testing.T) {
	repo := newMockUserRepo()
	mailer := newMockMailer()
	h := newTestHandler(repo, mailer)

	_, _ = h.svc.Register(context.Background(), "reset@example.com", "password123", "User")
	_ = h.svc.ForgotPassword(context.Background(), "reset@example.com")

	token := mailer.resets[0].args[0]

	body := map[string]string{"token": token, "new_password": "newpassword123"}
	b, _ := json.Marshal(body)
	r := httptest.NewRequest(http.MethodPost, "/auth/reset-password", bytes.NewReader(b))
	r.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	h.ResetPassword(w, r)

	if w.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", w.Code)
	}

	updated, _ := repo.GetByEmail(context.Background(), "reset@example.com")
	if updated.PasswordResetToken != nil {
		t.Error("reset token should be cleared")
	}
}

func TestResetPassword_InvalidBody(t *testing.T) {
	h := newTestHandler(nil, nil)

	r := httptest.NewRequest(http.MethodPost, "/auth/reset-password", bytes.NewReader([]byte("{not json")))
	r.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	h.ResetPassword(w, r)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected status 400, got %d", w.Code)
	}
}

func TestResendVerification_Success(t *testing.T) {
	repo := newMockUserRepo()
	mailer := newMockMailer()
	h := newTestHandler(repo, mailer)

	_, _ = h.svc.Register(context.Background(), "resend@example.com", "password123", "User")
	initialCount := len(mailer.verifications)

	body := map[string]string{"email": "resend@example.com"}
	b, _ := json.Marshal(body)
	r := httptest.NewRequest(http.MethodPost, "/auth/resend-verification", bytes.NewReader(b))
	r.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	h.ResendVerification(w, r)

	if w.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", w.Code)
	}
	if len(mailer.verifications) != initialCount+1 {
		t.Errorf("expected verification count to increase by 1, got %d", len(mailer.verifications))
	}
}
