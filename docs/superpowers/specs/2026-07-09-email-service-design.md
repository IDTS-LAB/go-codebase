# Email Service Design

## Overview

Add a loosely-coupled email service with HTML templates for user verification, password reset, welcome, and invite emails. The email provider is swappable via config with zero code changes.

## Goals

- Email provider abstraction (switch SMTP → SendGrid → Console via config)
- HTML email templates embedded in binary
- Email verification required before login
- Password reset flow via email
- Welcome and invite emails

## Architecture

### Domain Interface

```go
// internal/core/domain/email.go
type Emailer interface {
    SendVerification(to, name, token string) error
    SendPasswordReset(to, name, token string) error
    SendWelcome(to, name string) error
    SendInvite(to, name, inviterName string) error
}
```

### Provider Implementations

All in `internal/infrastructure/email/`:

| Provider | File | Use case |
|----------|------|----------|
| `console` | `console.go` | Dev — prints to stdout |
| `smtp` | `smtp.go` | Production — standard SMTP |
| `sendgrid` | `sendgrid.go` | Production — SendGrid API v3 |

Factory function `NewEmailer(cfg *config.Config) domain.Emailer` selects provider based on `cfg.Email.Provider`.

### Config

```yaml
email:
  provider: console
  from: "no-reply@myapp.com"
  from_name: "MyApp"
  smtp:
    host: localhost
    port: 587
    username: ""
    password: ""
    use_tls: true
  sendgrid:
    api_key: ""
  frontend_url: "http://localhost:3000"
```

Env overrides: `EMAIL_PROVIDER`, `EMAIL_FROM`, `EMAIL_FROM_NAME`, `SMTP_HOST`, `SMTP_PORT`, `SMTP_USERNAME`, `SMTP_PASSWORD`, `SMTP_USE_TLS`, `SENDGRID_API_KEY`, `FRONTEND_URL`.

### Templates

Embedded via `go:embed` in `internal/infrastructure/email/templates/`. HTML with inline CSS (email-safe).

| Template | Variables |
|----------|-----------|
| `verification.html` | Name, VerifyURL |
| `password_reset.html` | Name, ResetURL |
| `welcome.html` | Name |
| `invite.html` | Name, InviterName, InviteURL |

### User Entity Changes

Migration adds to `users` table:

| Column | Type | Default |
|--------|------|---------|
| `email_verified` | BOOLEAN | false |
| `email_verify_token` | VARCHAR(255) | NULL |
| `email_verify_expires` | TIMESTAMPTZ | NULL |
| `password_reset_token` | VARCHAR(255) | NULL |
| `password_reset_expires` | TIMESTAMPTZ | NULL |

### New Endpoints

| Method | Path | Description |
|--------|------|-------------|
| `GET` | `/auth/verify-email?token=xxx` | Verify email address |
| `POST` | `/auth/forgot-password` | Request password reset |
| `POST` | `/auth/reset-password` | Reset password with token |
| `POST` | `/auth/resend-verification` | Resend verification email |

### Flow Changes

**Register:**
1. Create user with `email_verified=false`
2. Generate verification token (crypto random, 32 bytes hex)
3. Store token + expiry (24h) in user record
4. Send verification email
5. Return success message (no tokens issued)

**Login:**
1. Check `email_verified` — if false, reject with "email not verified, please check your inbox"
2. Proceed with existing auth flow

**Verify Email:**
1. Look up user by `email_verify_token`
2. Check token not expired
3. Set `email_verified=true`, clear token fields
4. Send welcome email
5. Redirect to frontend success page

**Forgot Password:**
1. Look up user by email (silently return if not found — prevent enumeration)
2. Generate reset token (crypto random, 32 bytes hex)
3. Store token + expiry (1h) in user record
4. Send password reset email
5. Return success message

**Reset Password:**
1. Look up user by `password_reset_token`
2. Check token not expired
3. Hash new password, update user, clear reset token fields
4. Revoke all refresh tokens (force re-login)
5. Return success message

**Resend Verification:**
1. Look up user by email
2. If already verified, return success (idempotent)
3. Generate new verification token
4. Send verification email
5. Return success message

## File Structure

```
internal/
  core/domain/
    email.go                    # Emailer interface
  infrastructure/email/
    email.go                    # Factory function
    console.go                  # Console provider
    smtp.go                     # SMTP provider
    sendgrid.go                 # SendGrid provider
    templates/
      verification.html
      password_reset.html
      welcome.html
      invite.html
  shared/config/
    config.go                   # Add email config fields
  authentication/
    domain/entity/user.go       # Add verification/reset fields
    application/service/
      authentication_service.go # Modify register/login flows
    interfaces/http/
      handlers.go               # Add new endpoints
      routes.go                 # Add new routes
migrations/
      005_add_email_verification.sql
```

## Testing

- Unit test each provider (console captures output, SMTP/SendGrid mocked)
- Unit test template rendering
- Integration test register → verify → login flow
- Integration test forgot password → reset flow

## Swapping Providers

Change one config value, no code changes:

```yaml
# Development
email:
  provider: console

# Production with SMTP
email:
  provider: smtp
  smtp:
    host: smtp.gmail.com
    port: 587
    username: you@gmail.com
    password: app-password

# Production with SendGrid
email:
  provider: sendgrid
  sendgrid:
    api_key: SG.xxxx
```
