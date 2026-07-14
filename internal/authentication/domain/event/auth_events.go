package event

type UserRegistered struct {
	Email             string `json:"email"`
	Name              string `json:"name"`
	VerificationToken string `json:"verification_token"`
}

type EmailVerified struct {
	UserID string `json:"user_id"`
	Email  string `json:"email"`
	Name   string `json:"name"`
}

type PasswordResetRequested struct {
	Email      string `json:"email"`
	Name       string `json:"name"`
	ResetToken string `json:"reset_token"`
}

const (
	UserRegisteredEvent         = "auth.user.registered"
	EmailVerifiedEvent          = "auth.user.email_verified"
	PasswordResetRequestedEvent = "auth.user.password_reset_requested"
)
