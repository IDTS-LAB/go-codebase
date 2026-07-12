package domain

type TokenService interface {
	GenerateToken(claims *TokenClaims) (string, error)
	ValidateToken(tokenString string) (*TokenClaims, error)
}

type TokenClaims struct {
	UserID   string
	Email    string
	Role     string
	JTI      string
	TenantID string
}
