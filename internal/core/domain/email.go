package domain

type Emailer interface {
	SendVerification(to, name, token string) error
	SendPasswordReset(to, name, token string) error
	SendWelcome(to, name string) error
	SendInvite(to, name, inviterName string) error
}
