package command

import "errors"

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
