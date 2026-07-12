package auth

import (
	"fmt"
	"time"

	"github.com/IDTS-LAB/go-codebase/internal/core/domain"
	"github.com/IDTS-LAB/go-codebase/internal/shared/config"
	"github.com/golang-jwt/jwt/v5"
	"go.uber.org/fx"
)

var Module = fx.Module("jwt", fx.Provide(NewJWTTokenService))

type JWTTokenService struct {
	secret     string
	expiration time.Duration
}

func NewJWTTokenService(cfg *config.Config) domain.TokenService {
	return &JWTTokenService{
		secret:     cfg.Auth.JWTSecret,
		expiration: time.Duration(cfg.Auth.JWTExpiration) * time.Second,
	}
}

type jwtClaims struct {
	UserID   string `json:"user_id"`
	Email    string `json:"email"`
	Role     string `json:"role"`
	TenantID string `json:"tenant_id"`
	jwt.RegisteredClaims
}

func (s *JWTTokenService) GenerateToken(tc *domain.TokenClaims) (string, error) {
	claims := jwtClaims{
		UserID:   tc.UserID,
		Email:    tc.Email,
		Role:     tc.Role,
		TenantID: tc.TenantID,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(s.expiration)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			Subject:   tc.UserID,
			ID:        tc.JTI,
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenString, err := token.SignedString([]byte(s.secret))
	if err != nil {
		return "", fmt.Errorf("sign token: %w", err)
	}

	return tokenString, nil
}

func (s *JWTTokenService) ValidateToken(tokenString string) (*domain.TokenClaims, error) {
	token, err := jwt.ParseWithClaims(tokenString, &jwtClaims{}, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return []byte(s.secret), nil
	})
	if err != nil {
		return nil, fmt.Errorf("parse token: %w", err)
	}

	claims, ok := token.Claims.(*jwtClaims)
	if !ok || !token.Valid {
		return nil, fmt.Errorf("invalid token")
	}

	return &domain.TokenClaims{
		UserID:   claims.UserID,
		Email:    claims.Email,
		Role:     claims.Role,
		JTI:      claims.ID,
		TenantID: claims.TenantID,
	}, nil
}
