package service

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"time"

	"github.com/IDTS-LAB/go-codebase/internal/authentication/domain/entity"
	"github.com/IDTS-LAB/go-codebase/internal/authentication/domain/repository"
	"github.com/IDTS-LAB/go-codebase/internal/core/domain"
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
)

type AuthenticationService struct {
	userRepo         repository.UserRepository
	refreshRepo      repository.RefreshTokenRepository
	tokenService     domain.TokenService
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
) *AuthenticationService {
	return &AuthenticationService{
		userRepo:         userRepo,
		refreshRepo:      refreshRepo,
		tokenService:     tokenService,
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
	accessToken, err := s.tokenService.GenerateToken(user.ID.String(), user.Email, "user")
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
