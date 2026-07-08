package domain

type TokenService interface {
	GenerateToken(userID, email, role string) (string, error)
	ValidateToken(tokenString string) (*TokenClaims, error)
}

type TokenClaims struct {
	UserID string
	Email  string
	Role   string
}
