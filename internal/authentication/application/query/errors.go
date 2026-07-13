package query

import "errors"

var (
	ErrInvalidCredentials = errors.New("invalid email or password")
	ErrAccountDisabled    = errors.New("account is disabled")
	ErrAccountLocked      = errors.New("account is temporarily locked")
	ErrEmailNotVerified   = errors.New("email not verified")
)
