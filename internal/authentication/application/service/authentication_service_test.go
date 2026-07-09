package service

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/IDTS-LAB/go-codebase/internal/authentication/domain/entity"
	"github.com/IDTS-LAB/go-codebase/internal/core/domain"
	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"
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
	verifications  []emailRecord
	resets         []emailRecord
	welcomes       []emailRecord
	invites        []emailRecord
	verifyErr      error
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
	if m.verifyErr != nil {
		return m.verifyErr
	}
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
	m.invites = append(m.invites, emailRecord{to, name, []string{inviterName}})
	return nil
}

type mockTokenService struct{}

func (mockTokenService) GenerateToken(_, _, _ string) (string, error) {
	return "mock-access-token", nil
}

func (mockTokenService) ValidateToken(_ string) (*domain.TokenClaims, error) {
	return &domain.TokenClaims{}, nil
}

func newTestService(repo *mockUserRepo, mailer *mockMailer) *AuthenticationService {
	if repo == nil {
		repo = newMockUserRepo()
	}
	if mailer == nil {
		mailer = newMockMailer()
	}
	return NewAuthenticationService(repo, newMockRefreshRepo(), mockTokenService{}, mailer)
}

func TestRegister_SendsVerificationEmail(t *testing.T) {
	repo := newMockUserRepo()
	mailer := newMockMailer()
	svc := newTestService(repo, mailer)

	user, err := svc.Register(context.Background(), "test@example.com", "password123", "Test User")
	if err != nil {
		t.Fatalf("Register failed: %v", err)
	}

	if user.EmailVerified {
		t.Error("new user should not be verified")
	}
	if user.EmailVerifyToken == nil {
		t.Error("user should have a verification token")
	}
	if user.EmailVerifyExpires == nil {
		t.Error("user should have a verification expiry")
	}

	if len(mailer.verifications) != 1 {
		t.Fatalf("expected 1 verification email, got %d", len(mailer.verifications))
	}
	if mailer.verifications[0].to != "test@example.com" {
		t.Errorf("expected email to test@example.com, got %s", mailer.verifications[0].to)
	}
}

func TestRegister_DuplicateEmail(t *testing.T) {
	svc := newTestService(nil, nil)
	_, _ = svc.Register(context.Background(), "dup@example.com", "password123", "User")
	_, err := svc.Register(context.Background(), "dup@example.com", "password123", "User2")
	if err != ErrEmailAlreadyExists {
		t.Errorf("expected ErrEmailAlreadyExists, got %v", err)
	}
}

func TestLogin_RejectsUnverifiedUser(t *testing.T) {
	repo := newMockUserRepo()
	mailer := newMockMailer()
	svc := newTestService(repo, mailer)

	_, _ = svc.Register(context.Background(), "unverified@example.com", "password123", "User")
	_, err := svc.Login(context.Background(), "unverified@example.com", "password123")
	if err != ErrEmailNotVerified {
		t.Errorf("expected ErrEmailNotVerified, got %v", err)
	}
}

func TestLogin_AcceptsVerifiedUser(t *testing.T) {
	repo := newMockUserRepo()
	mailer := newMockMailer()
	svc := newTestService(repo, mailer)

	user, _ := svc.Register(context.Background(), "verified@example.com", "password123", "User")
	user.EmailVerified = true
	_ = repo.Update(context.Background(), user)

	_, err := svc.Login(context.Background(), "verified@example.com", "password123")
	if err != nil {
		t.Errorf("expected no error, got %v", err)
	}
}

func TestVerifyEmail_HappyPath(t *testing.T) {
	repo := newMockUserRepo()
	mailer := newMockMailer()
	svc := newTestService(repo, mailer)

	user, _ := svc.Register(context.Background(), "verify@example.com", "password123", "User")
	token := *user.EmailVerifyToken

	err := svc.VerifyEmail(context.Background(), token)
	if err != nil {
		t.Fatalf("VerifyEmail failed: %v", err)
	}

	updated, _ := repo.GetByEmail(context.Background(), "verify@example.com")
	if !updated.EmailVerified {
		t.Error("user should be verified")
	}
	if updated.EmailVerifyToken != nil {
		t.Error("verify token should be cleared")
	}

	if len(mailer.welcomes) != 1 {
		t.Errorf("expected 1 welcome email, got %d", len(mailer.welcomes))
	}
}

func TestVerifyEmail_InvalidToken(t *testing.T) {
	svc := newTestService(nil, nil)
	err := svc.VerifyEmail(context.Background(), "invalid-token")
	if err != ErrInvalidVerifyToken {
		t.Errorf("expected ErrInvalidVerifyToken, got %v", err)
	}
}

func TestVerifyEmail_ExpiredToken(t *testing.T) {
	repo := newMockUserRepo()
	mailer := newMockMailer()
	svc := newTestService(repo, mailer)

	user, _ := svc.Register(context.Background(), "expired@example.com", "password123", "User")
	past := time.Now().Add(-1 * time.Hour)
	user.EmailVerifyExpires = &past
	_ = repo.Update(context.Background(), user)

	err := svc.VerifyEmail(context.Background(), *user.EmailVerifyToken)
	if err != ErrVerifyTokenExpired {
		t.Errorf("expected ErrVerifyTokenExpired, got %v", err)
	}
}

func TestForgotPassword_HappyPath(t *testing.T) {
	repo := newMockUserRepo()
	mailer := newMockMailer()
	svc := newTestService(repo, mailer)

	_, _ = svc.Register(context.Background(), "forgot@example.com", "password123", "User")

	err := svc.ForgotPassword(context.Background(), "forgot@example.com")
	if err != nil {
		t.Fatalf("ForgotPassword failed: %v", err)
	}

	if len(mailer.resets) != 1 {
		t.Fatalf("expected 1 reset email, got %d", len(mailer.resets))
	}

	user, _ := repo.GetByEmail(context.Background(), "forgot@example.com")
	if user.PasswordResetToken == nil {
		t.Error("user should have reset token")
	}
	if user.PasswordResetExpires == nil {
		t.Error("user should have reset expiry")
	}
}

func TestForgotPassword_UnknownEmail(t *testing.T) {
	svc := newTestService(nil, nil)
	err := svc.ForgotPassword(context.Background(), "nonexistent@example.com")
	if err != nil {
		t.Errorf("expected nil for unknown email, got %v", err)
	}
}

func TestResetPassword_HappyPath(t *testing.T) {
	repo := newMockUserRepo()
	mailer := newMockMailer()
	svc := newTestService(repo, mailer)

	_, _ = svc.Register(context.Background(), "reset@example.com", "password123", "User")
	_ = svc.ForgotPassword(context.Background(), "reset@example.com")

	user, _ := repo.GetByEmail(context.Background(), "reset@example.com")
	token := *user.PasswordResetToken

	err := svc.ResetPassword(context.Background(), token, "newpassword456")
	if err != nil {
		t.Fatalf("ResetPassword failed: %v", err)
	}

	updated, _ := repo.GetByEmail(context.Background(), "reset@example.com")
	if updated.PasswordResetToken != nil {
		t.Error("reset token should be cleared")
	}
	if bcrypt.CompareHashAndPassword([]byte(updated.Password), []byte("newpassword456")) != nil {
		t.Error("password should be updated")
	}
}

func TestResetPassword_InvalidToken(t *testing.T) {
	svc := newTestService(nil, nil)
	err := svc.ResetPassword(context.Background(), "invalid-token", "newpassword456")
	if err != ErrInvalidResetToken {
		t.Errorf("expected ErrInvalidResetToken, got %v", err)
	}
}

func TestResetPassword_ExpiredToken(t *testing.T) {
	repo := newMockUserRepo()
	mailer := newMockMailer()
	svc := newTestService(repo, mailer)

	user, _ := svc.Register(context.Background(), "expiredreset@example.com", "password123", "User")
	past := time.Now().Add(-1 * time.Hour)
	user.PasswordResetToken = &[]string{"reset-token-expired"}[0]
	user.PasswordResetExpires = &past
	_ = repo.Update(context.Background(), user)

	err := svc.ResetPassword(context.Background(), "reset-token-expired", "newpassword456")
	if err != ErrResetTokenExpired {
		t.Errorf("expected ErrResetTokenExpired, got %v", err)
	}
}

func TestResendVerification_HappyPath(t *testing.T) {
	repo := newMockUserRepo()
	mailer := newMockMailer()
	svc := newTestService(repo, mailer)

	_, _ = svc.Register(context.Background(), "resend@example.com", "password123", "User")
	initialCount := len(mailer.verifications)

	err := svc.ResendVerification(context.Background(), "resend@example.com")
	if err != nil {
		t.Fatalf("ResendVerification failed: %v", err)
	}

	if len(mailer.verifications) != initialCount+1 {
		t.Errorf("expected verification count to increase by 1")
	}
}

func TestResendVerification_AlreadyVerified(t *testing.T) {
	repo := newMockUserRepo()
	mailer := newMockMailer()
	svc := newTestService(repo, mailer)

	user, _ := svc.Register(context.Background(), "already@example.com", "password123", "User")
	user.EmailVerified = true
	_ = repo.Update(context.Background(), user)

	initialCount := len(mailer.verifications)
	err := svc.ResendVerification(context.Background(), "already@example.com")
	if err != nil {
		t.Errorf("expected nil for already verified, got %v", err)
	}
	if len(mailer.verifications) != initialCount {
		t.Error("should not send verification to already-verified user")
	}
}

func TestResendVerification_UnknownEmail(t *testing.T) {
	svc := newTestService(nil, nil)
	err := svc.ResendVerification(context.Background(), "nonexistent@example.com")
	if err != nil {
		t.Errorf("expected nil for unknown email, got %v", err)
	}
}
