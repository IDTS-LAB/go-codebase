package domain

import "errors"

var (
	ErrNotFound      = errors.New("entity not found")
	ErrAlreadyExists = errors.New("entity already exists")
	ErrInvalidID     = errors.New("invalid entity ID")
	ErrDeleted       = errors.New("entity is deleted")
	ErrConflict      = errors.New("entity conflict")
	ErrValidation    = errors.New("validation failed")
	ErrForbidden     = errors.New("forbidden")
	ErrUnauthorized  = errors.New("unauthorized")
)

type DomainError struct {
	Err     error
	Code    string
	Message string
}

func (e *DomainError) Error() string {
	if e.Message != "" {
		return e.Message
	}
	return e.Err.Error()
}

func (e *DomainError) Unwrap() error {
	return e.Err
}

func NewDomainError(err error, code, message string) *DomainError {
	return &DomainError{
		Err:     err,
		Code:    code,
		Message: message,
	}
}

func IsNotFound(err error) bool {
	return errors.Is(err, ErrNotFound)
}

func IsConflict(err error) bool {
	return errors.Is(err, ErrConflict)
}
