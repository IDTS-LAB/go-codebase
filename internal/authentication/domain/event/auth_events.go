package event

type UserRegistered struct {
	Email            string
	Name             string
	VerificationToken string
}

type EmailVerified struct {
	UserID string
	Email  string
	Name   string
}

type PasswordResetRequested struct {
	Email      string
	Name       string
	ResetToken string
}

const (
	UserRegisteredEvent         = "auth.user.registered"
	EmailVerifiedEvent          = "auth.user.email_verified"
	PasswordResetRequestedEvent = "auth.user.password_reset_requested"
)
