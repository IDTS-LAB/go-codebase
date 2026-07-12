package service

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"time"

	"github.com/IDTS-LAB/go-codebase/internal/authentication/domain/entity"
	"github.com/IDTS-LAB/go-codebase/internal/authentication/domain/event"
	"github.com/IDTS-LAB/go-codebase/internal/authentication/domain/repository"
	"github.com/IDTS-LAB/go-codebase/internal/core/domain"
	"github.com/IDTS-LAB/go-codebase/internal/shared/events"
	"github.com/IDTS-LAB/go-codebase/internal/shared/middleware"
	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"
)

var (
	ErrInvalidCredentials  = errors.New("invalid email or password")
	ErrEmailAlreadyExists  = errors.New("email already exists")
	ErrUserNotFound        = errors.New("user not found")
	ErrInvalidRefreshToken = errors.New("invalid or expired refresh token")
	ErrAccountDisabled     = errors.New("account is disabled")
	ErrAccountLocked       = errors.New("account is temporarily locked")
	ErrEmailNotVerified    = errors.New("email not verified")
	ErrInvalidVerifyToken  = errors.New("invalid or expired verification token")
	ErrVerifyTokenExpired  = errors.New("verification token expired")
	ErrInvalidResetToken   = errors.New("invalid or expired reset token")
	ErrResetTokenExpired   = errors.New("reset token expired")
)

type AuthenticationService struct {
	userRepo         repository.UserRepository
	refreshRepo      repository.RefreshTokenRepository
	tokenService     domain.TokenService
	bus              events.EventBus
	denylist         func(ctx context.Context, jti string, ttl time.Duration) error
	accessTokenTTL   time.Duration
	refreshTokenTTL  time.Duration
	maxLoginAttempts int
	lockoutDuration  time.Duration
}

func NewAuthenticationService(
	userRepo repository.UserRepository,
	refreshRepo repository.RefreshTokenRepository,
	tokenService domain.TokenService,
	bus events.EventBus,
) *AuthenticationService {
	return &AuthenticationService{
		userRepo:         userRepo,
		refreshRepo:      refreshRepo,
		tokenService:     tokenService,
		bus:              bus,
		accessTokenTTL:   15 * time.Minute,
		refreshTokenTTL:  7 * 24 * time.Hour,
		maxLoginAttempts: 5,
		lockoutDuration:  15 * time.Minute,
	}
}

func (s *AuthenticationService) SetDenylist(fn func(ctx context.Context, jti string, ttl time.Duration) error) {
	s.denylist = fn
}

func (s *AuthenticationService) SetLockoutConfig(maxAttempts int, lockoutDuration time.Duration) {
	s.maxLoginAttempts = maxAttempts
	s.lockoutDuration = lockoutDuration
}

func (s *AuthenticationService) Register(ctx context.Context, email, password, name string) (*entity.User, error) {
	existing, _ := s.userRepo.GetByEmail(ctx, email)
	if existing != nil {
		return nil, ErrEmailAlreadyExists
	}

	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return nil, fmt.Errorf("hash password: %w", err)
	}

	user := entity.NewUser(email, string(hashedPassword), name)
	if err := s.userRepo.Create(ctx, user); err != nil {
		return nil, err
	}

	token, err := generateRefreshToken()
	if err != nil {
		return nil, fmt.Errorf("generate verification token: %w", err)
	}
	expires := time.Now().Add(24 * time.Hour)
	hashed := hashToken(token)
	user.EmailVerifyToken = &hashed
	user.EmailVerifyExpires = &expires
	if err := s.userRepo.Update(ctx, user); err != nil {
		return nil, err
	}

	_ = s.bus.Publish(ctx, events.Event{
		Type: event.UserRegisteredEvent,
		Payload: event.UserRegistered{
			Email:             user.Email,
			Name:              user.Name,
			VerificationToken: token,
		},
	})

	return user, nil
}

func (s *AuthenticationService) Login(ctx context.Context, email, password string) (*entity.User, error) {
	user, err := s.userRepo.GetByEmail(ctx, email)
	if err != nil {
		return nil, ErrInvalidCredentials
	}

	if !user.IsActive {
		return nil, ErrAccountDisabled
	}

	if user.IsLocked() {
		return nil, ErrAccountLocked
	}

	if !user.EmailVerified {
		return nil, ErrEmailNotVerified
	}

	if err := bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(password)); err != nil {
		user.FailedLoginAttempts++
		if user.FailedLoginAttempts >= s.maxLoginAttempts {
			user.Lock(s.lockoutDuration)
		}
		_ = s.userRepo.Update(ctx, user)
		return nil, ErrInvalidCredentials
	}

	user.Unlock()
	_ = s.userRepo.Update(ctx, user)

	return user, nil
}

func (s *AuthenticationService) GenerateTokens(ctx context.Context, user *entity.User) (*TokenPair, error) {
	tc := &domain.TokenClaims{
		UserID:   user.ID.String(),
		Email:    user.Email,
		Role:     "user",
		JTI:      uuid.New().String(),
		TenantID: middleware.GetTenantID(ctx),
	}
	accessToken, err := s.tokenService.GenerateToken(tc)
	if err != nil {
		return nil, fmt.Errorf("generate access token: %w", err)
	}

	refreshTokenStr, err := generateRefreshToken()
	if err != nil {
		return nil, fmt.Errorf("generate refresh token: %w", err)
	}

	refreshToken := entity.NewRefreshToken(user.ID, refreshTokenStr, time.Now().Add(s.refreshTokenTTL))
	if err := s.refreshRepo.Create(ctx, refreshToken); err != nil {
		return nil, err
	}

	return &TokenPair{
		AccessToken:  accessToken,
		RefreshToken: refreshTokenStr,
		ExpiresIn:    int(s.accessTokenTTL.Seconds()),
	}, nil
}

func (s *AuthenticationService) RefreshToken(ctx context.Context, refreshTokenStr string) (*TokenPair, error) {
	refreshToken, err := s.refreshRepo.GetByToken(ctx, refreshTokenStr)
	if err != nil {
		return nil, ErrInvalidRefreshToken
	}

	if refreshToken.IsExpired() || refreshToken.IsRevoked() {
		return nil, ErrInvalidRefreshToken
	}

	user, err := s.userRepo.GetByID(ctx, refreshToken.UserID)
	if err != nil {
		return nil, ErrUserNotFound
	}

	if !user.IsActive {
		return nil, ErrAccountDisabled
	}

	if !user.EmailVerified {
		return nil, ErrEmailNotVerified
	}

	if err := s.refreshRepo.Revoke(ctx, refreshTokenStr); err != nil {
		return nil, err
	}

	return s.GenerateTokens(ctx, user)
}

func (s *AuthenticationService) Logout(ctx context.Context, refreshTokenStr string, accessTokenJTI string, accessTokenTTL time.Duration) error {
	if s.denylist != nil && accessTokenJTI != "" {
		_ = s.denylist(ctx, accessTokenJTI, accessTokenTTL)
	}
	return s.refreshRepo.Revoke(ctx, refreshTokenStr)
}

func (s *AuthenticationService) LogoutAll(ctx context.Context, userID uuid.UUID) error {
	return s.refreshRepo.RevokeAllByUserID(ctx, userID)
}

func (s *AuthenticationService) VerifyEmail(ctx context.Context, token string) error {
	user, err := s.userRepo.GetByVerifyToken(ctx, hashToken(token))
	if err != nil {
		return ErrInvalidVerifyToken
	}
	if user.EmailVerifyExpires != nil && time.Now().After(*user.EmailVerifyExpires) {
		return ErrVerifyTokenExpired
	}

	user.EmailVerified = true
	user.EmailVerifyToken = nil
	user.EmailVerifyExpires = nil
	if err := s.userRepo.Update(ctx, user); err != nil {
		return err
	}

	_ = s.bus.Publish(ctx, events.Event{
		Type: event.EmailVerifiedEvent,
		Payload: event.EmailVerified{
			UserID: user.ID.String(),
			Email:  user.Email,
			Name:   user.Name,
		},
	})
	return nil
}

func (s *AuthenticationService) ForgotPassword(ctx context.Context, email string) error {
	user, err := s.userRepo.GetByEmail(ctx, email)
	if err != nil {
		return nil
	}

	token, err := generateRefreshToken()
	if err != nil {
		return err
	}
	expires := time.Now().Add(1 * time.Hour)
	hashed := hashToken(token)
	user.PasswordResetToken = &hashed
	user.PasswordResetExpires = &expires
	if err := s.userRepo.Update(ctx, user); err != nil {
		return err
	}

	_ = s.bus.Publish(ctx, events.Event{
		Type: event.PasswordResetRequestedEvent,
		Payload: event.PasswordResetRequested{
			Email:      user.Email,
			Name:       user.Name,
			ResetToken: token,
		},
	})
	return nil
}

func (s *AuthenticationService) ResetPassword(ctx context.Context, token, newPassword string) error {
	user, err := s.userRepo.GetByResetToken(ctx, hashToken(token))
	if err != nil {
		return ErrInvalidResetToken
	}
	if user.PasswordResetExpires != nil && time.Now().After(*user.PasswordResetExpires) {
		return ErrResetTokenExpired
	}

	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(newPassword), bcrypt.DefaultCost)
	if err != nil {
		return err
	}

	user.Password = string(hashedPassword)
	user.PasswordResetToken = nil
	user.PasswordResetExpires = nil
	_ = s.refreshRepo.RevokeAllByUserID(ctx, user.ID)
	return s.userRepo.Update(ctx, user)
}

func (s *AuthenticationService) ResendVerification(ctx context.Context, email string) error {
	user, err := s.userRepo.GetByEmail(ctx, email)
	if err != nil {
		return nil
	}
	if user.EmailVerified {
		return nil
	}

	token, err := generateRefreshToken()
	if err != nil {
		return err
	}
	expires := time.Now().Add(24 * time.Hour)
	hashed := hashToken(token)
	user.EmailVerifyToken = &hashed
	user.EmailVerifyExpires = &expires
	if err := s.userRepo.Update(ctx, user); err != nil {
		return err
	}

	_ = s.bus.Publish(ctx, events.Event{
		Type: event.UserRegisteredEvent,
		Payload: event.UserRegistered{
			Email:             user.Email,
			Name:              user.Name,
			VerificationToken: token,
		},
	})
	return nil
}

type TokenPair struct {
	AccessToken  string
	RefreshToken string
	ExpiresIn    int
}

func generateRefreshToken() (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return hex.EncodeToString(b), nil
}

func hashToken(token string) string {
	h := sha256.Sum256([]byte(token))
	return hex.EncodeToString(h[:])
}
