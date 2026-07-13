# Email Service Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Add a loosely-coupled email service with HTML templates for user verification, password reset, welcome, and invite emails with swappable providers.

**Architecture:** Domain interface (`domain.Emailer`) with three provider implementations (console, SMTP, SendGrid). Templates embedded via `go:embed`. User entity extended with verification/reset fields. New auth endpoints for verification and password reset flows.

**Tech Stack:** Go, html/template, embed.FS, net/smtp, SendGrid v3 API, godotenv/koanf (existing config)

## Global Constraints

- Follow existing codebase patterns (DDD, Fx modules, clean architecture)
- Email provider selected via `EMAIL_PROVIDER` env var (default: `console`)
- Templates embedded in binary (no external file dependencies)
- All new code must have tests
- Frequent atomic commits

---

## File Structure

| File | Purpose |
|------|---------|
| `internal/core/domain/email.go` | `Emailer` interface |
| `internal/shared/config/config.go` | Add `Email` config struct |
| `internal/infrastructure/email/email.go` | Factory function |
| `internal/infrastructure/email/console.go` | Console provider |
| `internal/infrastructure/email/smtp.go` | SMTP provider |
| `internal/infrastructure/email/sendgrid.go` | SendGrid provider |
| `internal/infrastructure/email/templates/verification.html` | Verification email template |
| `internal/infrastructure/email/templates/password_reset.html` | Password reset template |
| `internal/infrastructure/email/templates/welcome.html` | Welcome email template |
| `internal/infrastructure/email/templates/invite.html` | Invite email template |
| `internal/infrastructure/email/email_test.go` | Provider tests |
| `internal/authentication/domain/entity/user.go` | Add verification/reset fields |
| `internal/authentication/application/service/authentication_service.go` | Modify register/login, add reset flows |
| `internal/authentication/application/dto/authentication_dto.go` | Add new DTOs |
| `internal/authentication/interfaces/http/handlers.go` | Add new endpoints |
| `internal/authentication/interfaces/http/routes.go` | Add new routes |
| `internal/authentication/module.go` | Wire emailer |
| `migrations/005_add_email_verification.sql` | DB migration |
| `cmd/api/main.go` | Wire email module |
| `configs/config.yaml` | Add email config section |

---

### Task 1: Domain Interface + Config

**Files:**
- Create: `internal/core/domain/email.go`
- Modify: `internal/shared/config/config.go`
- Modify: `configs/config.yaml`

**Interfaces:**
- Produces: `domain.Emailer` interface, `config.Email` struct

- [ ] **Step 1: Create the Emailer interface**

```go
// internal/core/domain/email.go
package domain

type Emailer interface {
    SendVerification(to, name, token string) error
    SendPasswordReset(to, name, token string) error
    SendWelcome(to, name string) error
    SendInvite(to, name, inviterName string) error
}
```

- [ ] **Step 2: Add Email config to config.go**

Add to `internal/shared/config/config.go`:

```go
type EmailConfig struct {
    Provider    string     `koanf:"provider"`
    From        string     `koanf:"from"`
    FromName    string     `koanf:"from_name"`
    FrontendURL string     `koanf:"frontend_url"`
    SMTP        SMTPConfig `koanf:"smtp"`
    SendGrid    SendGridConfig `koanf:"sendgrid"`
}

type SMTPConfig struct {
    Host     string `koanf:"host"`
    Port     int    `koanf:"port"`
    Username string `koanf:"username"`
    Password string `koanf:"password"`
    UseTLS   bool   `koanf:"use_tls"`
}

type SendGridConfig struct {
    APIKey string `koanf:"api_key"`
}
```

Add `Email EmailConfig` field to the `Config` struct. Add env override tags: `EMAIL_PROVIDER`, `EMAIL_FROM`, `EMAIL_FROM_NAME`, `FRONTEND_URL`, `SMTP_HOST`, `SMTP_PORT`, `SMTP_USERNAME`, `SMTP_PASSWORD`, `SMTP_USE_TLS`, `SENDGRID_API_KEY`.

- [ ] **Step 3: Add default email config**

In the config defaults section, add:

```go
cfg.Email = EmailConfig{
    Provider:    "console",
    From:        "no-reply@example.com",
    FromName:    "App",
    FrontendURL: "http://localhost:3000",
    SMTP: SMTPConfig{
        Host:   "localhost",
        Port:   587,
        UseTLS: true,
    },
}
```

- [ ] **Step 4: Add email section to configs/config.yaml**

```yaml
email:
  provider: console
  from: "no-reply@example.com"
  from_name: "App"
  frontend_url: "http://localhost:3000"
  smtp:
    host: localhost
    port: 587
    username: ""
    password: ""
    use_tls: true
  sendgrid:
    api_key: ""
```

- [ ] **Step 5: Verify build**

Run: `go build ./...`
Expected: PASS

- [ ] **Step 6: Commit**

```bash
git add internal/core/domain/email.go internal/shared/config/config.go configs/config.yaml
git commit -m "email: add domain interface and config"
```

---

### Task 2: Console Provider

**Files:**
- Create: `internal/infrastructure/email/console.go`
- Create: `internal/infrastructure/email/email.go`

**Interfaces:**
- Produces: `ConsoleMailer` struct implementing `domain.Emailer`, `NewEmailer()` factory

- [ ] **Step 1: Create console provider**

```go
// internal/infrastructure/email/console.go
package email

import (
    "fmt"
    "log"
)

type ConsoleMailer struct {
    from     string
    fromName string
}

func NewConsoleMailer(from, fromName string) *ConsoleMailer {
    return &ConsoleMailer{from: from, fromName: fromName}
}

func (m *ConsoleMailer) SendVerification(to, name, token string) error {
    log.Printf("[EMAIL] To: %s | Subject: Verify your email | Link: %s/verify-email?token=%s", to, name, token)
    return nil
}

func (m *ConsoleMailer) SendPasswordReset(to, name, token string) error {
    log.Printf("[EMAIL] To: %s | Subject: Reset your password | Link: %s/reset-password?token=%s", to, name, token)
    return nil
}

func (m *ConsoleMailer) SendWelcome(to, name string) error {
    log.Printf("[EMAIL] To: %s | Subject: Welcome %s!", to, name)
    return nil
}

func (m *ConsoleMailer) SendInvite(to, name, inviterName string) error {
    log.Printf("[EMAIL] To: %s | Subject: %s invited you to join", to, inviterName)
    return nil
}
```

- [ ] **Step 2: Create factory function**

```go
// internal/infrastructure/email/email.go
package email

import (
    "github.com/IDTS-LAB/go-codebase/internal/core/domain"
    "github.com/IDTS-LAB/go-codebase/internal/shared/config"
    "go.uber.org/fx"
)

var Module = fx.Module("email", fx.Provide(NewEmailer))

func NewEmailer(cfg *config.Config) domain.Emailer {
    switch cfg.Email.Provider {
    case "smtp":
        return NewSMTPMailer(cfg.Email.SMTP.Host, cfg.Email.SMTP.Port,
            cfg.Email.SMTP.Username, cfg.Email.SMTP.Password,
            cfg.Email.SMTP.UseTLS, cfg.Email.From, cfg.Email.FromName)
    case "sendgrid":
        return NewSendGridMailer(cfg.Email.SendGrid.APIKey,
            cfg.Email.From, cfg.Email.FromName)
    default:
        return NewConsoleMailer(cfg.Email.From, cfg.Email.FromName)
    }
}
```

- [ ] **Step 3: Verify build**

Run: `go build ./...`
Expected: PASS

- [ ] **Step 4: Commit**

```bash
git add internal/infrastructure/email/
git commit -m "email: add console provider and factory"
```

---

### Task 3: SMTP Provider

**Files:**
- Create: `internal/infrastructure/email/smtp.go`

**Interfaces:**
- Produces: `SMTPMailer` struct implementing `domain.Emailer`

- [ ] **Step 1: Create SMTP provider**

```go
// internal/infrastructure/email/smtp.go
package email

import (
    "crypto/tls"
    "fmt"
    "net/smtp"
)

type SMTPMailer struct {
    host     string
    port     int
    username string
    password string
    useTLS   bool
    from     string
    fromName string
}

func NewSMTPMailer(host string, port int, username, password string, useTLS bool, from, fromName string) *SMTPMailer {
    return &SMTPMailer{
        host: host, port: port, username: username,
        password: password, useTLS: useTLS, from: from, fromName: fromName,
    }
}

func (m *SMTPMailer) SendVerification(to, name, token string) error {
    subject := "Verify your email address"
    body := fmt.Sprintf("Hello %s,\n\nPlease verify your email by clicking the link below:\n\n%s/verify-email?token=%s\n\nIf you didn't create an account, please ignore this email.", name, token, token)
    return m.send(to, subject, body)
}

func (m *SMTPMailer) SendPasswordReset(to, name, token string) error {
    subject := "Reset your password"
    body := fmt.Sprintf("Hello %s,\n\nYou requested a password reset. Click the link below to reset your password:\n\n%s/reset-password?token=%s\n\nThis link expires in 1 hour. If you didn't request this, please ignore this email.", name, token, token)
    return m.send(to, subject, body)
}

func (m *SMTPMailer) SendWelcome(to, name string) error {
    subject := fmt.Sprintf("Welcome %s!", name)
    body := fmt.Sprintf("Hello %s,\n\nWelcome to our platform! Your account is now active.", name)
    return m.send(to, subject, body)
}

func (m *SMTPMailer) SendInvite(to, name, inviterName string) error {
    subject := fmt.Sprintf("%s invited you to join", inviterName)
    body := fmt.Sprintf("Hello %s,\n\n%s has invited you to join our platform.\n\nClick the link below to get started.", name, inviterName)
    return m.send(to, subject, body)
}

func (m *SMTPMailer) send(to, subject, body string) error {
    addr := fmt.Sprintf("%s:%d", m.host, m.port)
    auth := smtp.PlainAuth("", m.username, m.password, m.host)

    msg := fmt.Sprintf("From: %s <%s>\r\nTo: %s\r\nSubject: %s\r\nMIME-Version: 1.0\r\nContent-Type: text/plain; charset=UTF-8\r\n\r\n%s",
        m.fromName, m.from, to, subject, body)

    if m.useTLS {
        return m.sendWithTLS(addr, auth, to, msg)
    }
    return smtp.SendMail(addr, auth, m.from, []string{to}, []byte(msg))
}

func (m *SMTPMailer) sendWithTLS(addr string, auth smtp.Auth, to, msg string) error {
    conn, err := tls.Dial("tcp", addr, &tls.Config{ServerName: m.host})
    if err != nil {
        return err
    }
    client, err := smtp.NewClient(conn, m.host)
    if err != nil {
        return err
    }
    defer client.Close()
    if err = client.Auth(auth); err != nil {
        return err
    }
    if err = client.Mail(m.from); err != nil {
        return err
    }
    if err = client.Rcpt(to); err != nil {
        return err
    }
    w, err := client.Data()
    if err != nil {
        return err
    }
    if _, err = w.Write([]byte(msg)); err != nil {
        return err
    }
    if err = w.Close(); err != nil {
        return err
    }
    return client.Quit()
}
```

- [ ] **Step 2: Verify build**

Run: `go build ./...`
Expected: PASS

- [ ] **Step 3: Commit**

```bash
git add internal/infrastructure/email/smtp.go
git commit -m "email: add SMTP provider"
```

---

### Task 4: SendGrid Provider

**Files:**
- Create: `internal/infrastructure/email/sendgrid.go`

**Interfaces:**
- Produces: `SendGridMailer` struct implementing `domain.Emailer`

- [ ] **Step 1: Add SendGrid dependency**

Run: `go get github.com/sendgrid/sendgrid-go`
Run: `go mod tidy`

- [ ] **Step 2: Create SendGrid provider**

```go
// internal/infrastructure/email/sendgrid.go
package email

import (
    "fmt"

    "github.com/sendgrid/sendgrid-go"
    "github.com/sendgrid/sendgrid-go/helpers/mail"
)

type SendGridMailer struct {
    apiKey string
    from   string
    fromName string
}

func NewSendGridMailer(apiKey, from, fromName string) *SendGridMailer {
    return &SendGridMailer{apiKey: apiKey, from: from, fromName: fromName}
}

func (m *SendGridMailer) SendVerification(to, name, token string) error {
    subject := "Verify your email address"
    htmlContent := fmt.Sprintf(`<p>Hello %s,</p><p>Please verify your email by clicking the link below:</p><p><a href="%%s/verify-email?token=%s">Verify Email</a></p><p>If you didn't create an account, please ignore this email.</p>`, name, token)
    return m.send(to, subject, htmlContent)
}

func (m *SendGridMailer) SendPasswordReset(to, name, token string) error {
    subject := "Reset your password"
    htmlContent := fmt.Sprintf(`<p>Hello %s,</p><p>You requested a password reset. Click the link below:</p><p><a href="%%s/reset-password?token=%s">Reset Password</a></p><p>This link expires in 1 hour.</p>`, name, token)
    return m.send(to, subject, htmlContent)
}

func (m *SendGridMailer) SendWelcome(to, name string) error {
    subject := fmt.Sprintf("Welcome %s!", name)
    htmlContent := fmt.Sprintf(`<p>Hello %s,</p><p>Welcome to our platform! Your account is now active.</p>`, name)
    return m.send(to, subject, htmlContent)
}

func (m *SendGridMailer) SendInvite(to, name, inviterName string) error {
    subject := fmt.Sprintf("%s invited you to join", inviterName)
    htmlContent := fmt.Sprintf(`<p>Hello %s,</p><p>%s has invited you to join our platform.</p>`, name, inviterName)
    return m.send(to, subject, htmlContent)
}

func (m *SendGridMailer) send(to, subject, htmlContent string) error {
    from := mail.NewEmail(m.fromName, m.from)
    message := mail.NewSingleEmail(from, subject, mail.NewEmail(to, to), "", htmlContent)
    client := sendgrid.NewSendClient(m.apiKey)
    _, err := client.Send(message)
    return err
}
```

- [ ] **Step 3: Verify build**

Run: `go build ./...`
Expected: PASS

- [ ] **Step 4: Commit**

```bash
git add internal/infrastructure/email/sendgrid.go go.mod go.sum
git commit -m "email: add SendGrid provider"
```

---

### Task 5: HTML Templates

**Files:**
- Create: `internal/infrastructure/email/templates/verification.html`
- Create: `internal/infrastructure/email/templates/password_reset.html`
- Create: `internal/infrastructure/email/templates/welcome.html`
- Create: `internal/infrastructure/email/templates/invite.html`

**Interfaces:**
- Produces: Embedded HTML templates

- [ ] **Step 1: Create verification template**

```html
<!-- internal/infrastructure/email/templates/verification.html -->
<!DOCTYPE html>
<html>
<head><meta charset="UTF-8"></head>
<body style="font-family: Arial, sans-serif; max-width: 600px; margin: 0 auto; padding: 20px;">
  <h2 style="color: #333;">Verify Your Email</h2>
  <p>Hello {{.Name}},</p>
  <p>Thanks for signing up! Please verify your email address by clicking the button below:</p>
  <p style="text-align: center; margin: 30px 0;">
    <a href="{{.VerifyURL}}" style="background-color: #4CAF50; color: white; padding: 12px 24px; text-decoration: none; border-radius: 4px;">Verify Email</a>
  </p>
  <p style="color: #666; font-size: 14px;">If you didn't create an account, please ignore this email.</p>
  <hr style="border: none; border-top: 1px solid #eee; margin: 20px 0;">
  <p style="color: #999; font-size: 12px;">This link will expire in 24 hours.</p>
</body>
</html>
```

- [ ] **Step 2: Create password reset template**

```html
<!-- internal/infrastructure/email/templates/password_reset.html -->
<!DOCTYPE html>
<html>
<head><meta charset="UTF-8"></head>
<body style="font-family: Arial, sans-serif; max-width: 600px; margin: 0 auto; padding: 20px;">
  <h2 style="color: #333;">Reset Your Password</h2>
  <p>Hello {{.Name}},</p>
  <p>You requested a password reset. Click the button below to set a new password:</p>
  <p style="text-align: center; margin: 30px 0;">
    <a href="{{.ResetURL}}" style="background-color: #2196F3; color: white; padding: 12px 24px; text-decoration: none; border-radius: 4px;">Reset Password</a>
  </p>
  <p style="color: #666; font-size: 14px;">If you didn't request this, please ignore this email.</p>
  <hr style="border: none; border-top: 1px solid #eee; margin: 20px 0;">
  <p style="color: #999; font-size: 12px;">This link will expire in 1 hour.</p>
</body>
</html>
```

- [ ] **Step 3: Create welcome template**

```html
<!-- internal/infrastructure/email/templates/welcome.html -->
<!DOCTYPE html>
<html>
<head><meta charset="UTF-8"></head>
<body style="font-family: Arial, sans-serif; max-width: 600px; margin: 0 auto; padding: 20px;">
  <h2 style="color: #333;">Welcome!</h2>
  <p>Hello {{.Name}},</p>
  <p>Welcome to our platform! Your account is now active and ready to use.</p>
  <p>If you have any questions, feel free to reach out to our support team.</p>
  <hr style="border: none; border-top: 1px solid #eee; margin: 20px 0;">
  <p style="color: #999; font-size: 12px;">This is an automated message, please do not reply.</p>
</body>
</html>
```

- [ ] **Step 4: Create invite template**

```html
<!-- internal/infrastructure/email/templates/invite.html -->
<!DOCTYPE html>
<html>
<head><meta charset="UTF-8"></head>
<body style="font-family: Arial, sans-serif; max-width: 600px; margin: 0 auto; padding: 20px;">
  <h2 style="color: #333;">You've Been Invited</h2>
  <p>Hello {{.Name}},</p>
  <p><strong>{{.InviterName}}</strong> has invited you to join our platform.</p>
  <p style="text-align: center; margin: 30px 0;">
    <a href="{{.InviteURL}}" style="background-color: #9C27B0; color: white; padding: 12px 24px; text-decoration: none; border-radius: 4px;">Accept Invitation</a>
  </p>
  <hr style="border: none; border-top: 1px solid #eee; margin: 20px 0;">
  <p style="color: #999; font-size: 12px;">This is an automated message, please do not reply.</p>
</body>
</html>
```

- [ ] **Step 5: Commit**

```bash
git add internal/infrastructure/email/templates/
git commit -m "email: add HTML templates for verification, reset, welcome, invite"
```

---

### Task 6: Template Rendering in Providers

**Files:**
- Modify: `internal/infrastructure/email/console.go`
- Modify: `internal/infrastructure/email/smtp.go`
- Modify: `internal/infrastructure/email/sendgrid.go`

**Interfaces:**
- Consumes: HTML templates from Task 5

- [ ] **Step 1: Create template renderer**

Create `internal/infrastructure/email/renderer.go`:

```go
package email

import (
    "embed"
    "html/template"
    "bytes"
)

//go:embed templates/*.html
var templateFS embed.FS

type TemplateData struct {
    Name       string
    VerifyURL  string
    ResetURL   string
    InviteURL  string
    InviterName string
}

func renderTemplate(name string, data TemplateData) (string, error) {
    tmplPath := "templates/" + name + ".html"
    t, err := template.ParseFS(templateFS, tmplPath)
    if err != nil {
        return "", err
    }
    var buf bytes.Buffer
    if err := t.Execute(&buf, data); err != nil {
        return "", err
    }
    return buf.String(), nil
}
```

- [ ] **Step 2: Update ConsoleMailer to use templates**

Replace the console provider's methods to use `renderTemplate`:

```go
func (m *ConsoleMailer) SendVerification(to, name, token string) error {
    verifyURL := m.frontendURL + "/verify-email?token=" + token
    content, err := renderTemplate("verification", TemplateData{Name: name, VerifyURL: verifyURL})
    if err != nil {
        return err
    }
    log.Printf("[EMAIL] To: %s | Subject: Verify your email\n%s", to, content)
    return nil
}

func (m *ConsoleMailer) SendPasswordReset(to, name, token string) error {
    resetURL := m.frontendURL + "/reset-password?token=" + token
    content, err := renderTemplate("password_reset", TemplateData{Name: name, ResetURL: resetURL})
    if err != nil {
        return err
    }
    log.Printf("[EMAIL] To: %s | Subject: Reset your password\n%s", to, content)
    return nil
}

func (m *ConsoleMailer) SendWelcome(to, name string) error {
    content, err := renderTemplate("welcome", TemplateData{Name: name})
    if err != nil {
        return err
    }
    log.Printf("[EMAIL] To: %s | Subject: Welcome %s!\n%s", to, name, content)
    return nil
}

func (m *ConsoleMailer) SendInvite(to, name, inviterName string) error {
    content, err := renderTemplate("invite", TemplateData{Name: name, InviterName: inviterName})
    if err != nil {
        return err
    }
    log.Printf("[EMAIL] To: %s | Subject: %s invited you\n%s", to, inviterName, content)
    return nil
}
```

Update `ConsoleMailer` struct to include `frontendURL` field. Update `NewConsoleMailer` to accept it.

- [ ] **Step 3: Update SMTPMailer to use HTML content type**

Update the `send` method to use `text/html` instead of `text/plain`, and use `renderTemplate` in each method.

- [ ] **Step 4: Update SendGridMailer to use templates**

Update each method to use `renderTemplate` for the HTML content.

- [ ] **Step 5: Update factory to pass frontendURL**

```go
func NewEmailer(cfg *config.Config) domain.Emailer {
    switch cfg.Email.Provider {
    case "smtp":
        return NewSMTPMailer(cfg.Email.SMTP.Host, cfg.Email.SMTP.Port,
            cfg.Email.SMTP.Username, cfg.Email.SMTP.Password,
            cfg.Email.SMTP.UseTLS, cfg.Email.From, cfg.Email.FromName, cfg.Email.FrontendURL)
    case "sendgrid":
        return NewSendGridMailer(cfg.Email.SendGrid.APIKey,
            cfg.Email.From, cfg.Email.FromName, cfg.Email.FrontendURL)
    default:
        return NewConsoleMailer(cfg.Email.From, cfg.Email.FromName, cfg.Email.FrontendURL)
    }
}
```

- [ ] **Step 6: Verify build**

Run: `go build ./...`
Expected: PASS

- [ ] **Step 7: Commit**

```bash
git add internal/infrastructure/email/
git commit -m "email: add template rendering to all providers"
```

---

### Task 7: User Entity Migration

**Files:**
- Create: `migrations/005_add_email_verification.sql`
- Modify: `internal/authentication/domain/entity/user.go`

**Interfaces:**
- Produces: Extended User entity with verification/reset fields

- [ ] **Step 1: Create migration**

```sql
-- migrations/005_add_email_verification.sql
-- +goose Up
ALTER TABLE users ADD COLUMN email_verified BOOLEAN DEFAULT false;
ALTER TABLE users ADD COLUMN email_verify_token VARCHAR(255);
ALTER TABLE users ADD COLUMN email_verify_expires TIMESTAMPTZ;
ALTER TABLE users ADD COLUMN password_reset_token VARCHAR(255);
ALTER TABLE users ADD COLUMN password_reset_expires TIMESTAMPTZ;

CREATE INDEX idx_users_email_verify_token ON users(email_verify_token) WHERE email_verify_token IS NOT NULL;
CREATE INDEX idx_users_password_reset_token ON users(password_reset_token) WHERE password_reset_token IS NOT NULL;

-- +goose Down
DROP INDEX IF EXISTS idx_users_email_verify_token;
DROP INDEX IF EXISTS idx_users_password_reset_token;
ALTER TABLE users DROP COLUMN IF EXISTS email_verified;
ALTER TABLE users DROP COLUMN IF EXISTS email_verify_token;
ALTER TABLE users DROP COLUMN IF EXISTS email_verify_expires;
ALTER TABLE users DROP COLUMN IF EXISTS password_reset_token;
ALTER TABLE users DROP COLUMN IF EXISTS password_reset_expires;
```

- [ ] **Step 2: Update User entity**

Add fields to `internal/authentication/domain/entity/user.go`:

```go
type User struct {
    domain.Entity
    Email               string     `json:"email"`
    Password            string     `json:"-"`
    Name                string     `json:"name"`
    IsActive            bool       `json:"is_active"`
    FailedLoginAttempts int        `json:"failed_login_attempts"`
    LockedUntil         *time.Time `json:"locked_until,omitempty"`
    EmailVerified       bool       `json:"email_verified"`
    EmailVerifyToken    *string    `json:"-"`
    EmailVerifyExpires  *time.Time `json:"-"`
    PasswordResetToken  *string    `json:"-"`
    PasswordResetExpires *time.Time `json:"-"`
}
```

- [ ] **Step 3: Update UserRepository interface**

Add to `internal/authentication/domain/repository/authentication_repository.go`:

```go
type UserRepository interface {
    Create(ctx context.Context, user *entity.User) error
    GetByID(ctx context.Context, id uuid.UUID) (*entity.User, error)
    GetByEmail(ctx context.Context, email string) (*entity.User, error)
    GetByVerifyToken(ctx context.Context, token string) (*entity.User, error)
    GetByResetToken(ctx context.Context, token string) (*entity.User, error)
    Update(ctx context.Context, user *entity.User) error
}
```

- [ ] **Step 4: Update UserRepository implementation**

Add `GetByVerifyToken` and `GetByResetToken` methods to `internal/authentication/infrastructure/persistence/user_repository.go`.

- [ ] **Step 5: Verify build**

Run: `go build ./...`
Expected: PASS

- [ ] **Step 6: Commit**

```bash
git add migrations/005_add_email_verification.sql internal/authentication/domain/entity/user.go internal/authentication/domain/repository/authentication_repository.go internal/authentication/infrastructure/persistence/user_repository.go
git commit -m "auth: add email verification and password reset fields to user entity"
```

---

### Task 8: Auth Service Changes

**Files:**
- Modify: `internal/authentication/application/service/authentication_service.go`
- Modify: `internal/authentication/application/dto/authentication_dto.go`

**Interfaces:**
- Consumes: `domain.Emailer` from Task 1-6
- Produces: Modified `Register`, `Login` methods; new `VerifyEmail`, `ForgotPassword`, `ResetPassword`, `ResendVerification` methods

- [ ] **Step 1: Add new DTOs**

Add to `internal/authentication/application/dto/authentication_dto.go`:

```go
type ForgotPasswordRequest struct {
    Email string `json:"email" validate:"required,email"`
}

type ResetPasswordRequest struct {
    Token       string `json:"token" validate:"required"`
    NewPassword string `json:"new_password" validate:"required,min=8"`
}

type ResendVerificationRequest struct {
    Email string `json:"email" validate:"required,email"`
}
```

- [ ] **Step 2: Update AuthenticationService struct**

Add `emailer domain.Emailer` and `frontendURL string` fields:

```go
type AuthenticationService struct {
    userRepo       repository.UserRepository
    refreshRepo    repository.RefreshTokenRepository
    tokenService   domain.TokenService
    mailer         domain.Emailer
    frontendURL    string
}
```

Update `NewAuthenticationService` to accept `emailer` and `frontendURL`.

- [ ] **Step 3: Add token generation helpers**

```go
func generateToken() (string, error) {
    bytes := make([]byte, 32)
    if _, err := rand.Read(bytes); err != nil {
        return "", err
    }
    return hex.EncodeToString(bytes), nil
}
```

- [ ] **Step 4: Update Register method**

After creating user, generate verification token, store it, send verification email:

```go
func (s *AuthenticationService) Register(ctx context.Context, email, password, name string) (*entity.User, error) {
    // ... existing validation ...

    user := entity.NewUser(email, hashedPassword, name)
    if err := s.userRepo.Create(ctx, user); err != nil {
        return nil, err
    }

    // Generate verification token
    token, err := generateToken()
    if err != nil {
        return nil, err
    }
    expires := time.Now().Add(24 * time.Hour)
    user.EmailVerifyToken = &token
    user.EmailVerifyExpires = &expires
    if err := s.userRepo.Update(ctx, user); err != nil {
        return nil, err
    }

    // Send verification email
    s.mailer.SendVerification(user.Email, user.Name, token)

    return user, nil
}
```

- [ ] **Step 5: Update Login method**

Add email verification check:

```go
func (s *AuthenticationService) Login(ctx context.Context, email, password string) (*entity.User, error) {
    user, err := s.userRepo.GetByEmail(ctx, email)
    if err != nil {
        return nil, ErrInvalidCredentials
    }

    if !user.EmailVerified {
        return nil, ErrEmailNotVerified
    }

    // ... existing login logic ...
}
```

Add `ErrEmailNotVerified` error.

- [ ] **Step 6: Add VerifyEmail method**

```go
func (s *AuthenticationService) VerifyEmail(ctx context.Context, token string) error {
    user, err := s.userRepo.GetByVerifyToken(ctx, token)
    if err != nil {
        return fmt.Errorf("invalid or expired verification token")
    }
    if user.EmailVerifyExpires != nil && time.Now().After(*user.EmailVerifyExpires) {
        return fmt.Errorf("verification token expired")
    }

    user.EmailVerified = true
    user.EmailVerifyToken = nil
    user.EmailVerifyExpires = nil
    if err := s.userRepo.Update(ctx, user); err != nil {
        return err
    }

    s.mailer.SendWelcome(user.Email, user.Name)
    return nil
}
```

- [ ] **Step 7: Add ForgotPassword method**

```go
func (s *AuthenticationService) ForgotPassword(ctx context.Context, email string) error {
    user, err := s.userRepo.GetByEmail(ctx, email)
    if err != nil {
        return nil // Silently return to prevent enumeration
    }

    token, err := generateToken()
    if err != nil {
        return err
    }
    expires := time.Now().Add(1 * time.Hour)
    user.PasswordResetToken = &token
    user.PasswordResetExpires = &expires
    if err := s.userRepo.Update(ctx, user); err != nil {
        return err
    }

    s.mailer.SendPasswordReset(user.Email, user.Name, token)
    return nil
}
```

- [ ] **Step 8: Add ResetPassword method**

```go
func (s *AuthenticationService) ResetPassword(ctx context.Context, token, newPassword string) error {
    user, err := s.userRepo.GetByResetToken(ctx, token)
    if err != nil {
        return fmt.Errorf("invalid or expired reset token")
    }
    if user.PasswordResetExpires != nil && time.Now().After(*user.PasswordResetExpires) {
        return fmt.Errorf("reset token expired")
    }

    hashedPassword, err := bcrypt.GenerateFromPassword([]byte(newPassword), bcrypt.DefaultCost)
    if err != nil {
        return err
    }

    user.Password = string(hashedPassword)
    user.PasswordResetToken = nil
    user.PasswordResetExpires = nil
    return s.userRepo.Update(ctx, user)
}
```

- [ ] **Step 9: Add ResendVerification method**

```go
func (s *AuthenticationService) ResendVerification(ctx context.Context, email string) error {
    user, err := s.userRepo.GetByEmail(ctx, email)
    if err != nil {
        return nil // Silently return
    }
    if user.EmailVerified {
        return nil // Already verified
    }

    token, err := generateToken()
    if err != nil {
        return err
    }
    expires := time.Now().Add(24 * time.Hour)
    user.EmailVerifyToken = &token
    user.EmailVerifyExpires = &expires
    if err := s.userRepo.Update(ctx, user); err != nil {
        return err
    }

    return s.mailer.SendVerification(user.Email, user.Name, token)
}
```

- [ ] **Step 10: Verify build**

Run: `go build ./...`
Expected: PASS

- [ ] **Step 11: Commit**

```bash
git add internal/authentication/application/
git commit -m "auth: add email verification and password reset flows"
```

---

### Task 9: HTTP Handlers + Routes

**Files:**
- Modify: `internal/authentication/interfaces/http/handlers.go`
- Modify: `internal/authentication/interfaces/http/routes.go`

**Interfaces:**
- Consumes: Modified `AuthenticationService` from Task 8

- [ ] **Step 1: Add new handler methods**

Add to `internal/authentication/interfaces/http/handlers.go`:

```go
// VerifyEmail godoc
// @Summary Verify email address
// @Description Verify user email with token from email
// @Tags authentication
// @Param token query string true "Verification token"
// @Success 200 {object} utils.SuccessResponse
// @Failure 400 {object} utils.ErrorResponse
// @Router /auth/verify-email [get]
func (h *Handler) VerifyEmail(w http.ResponseWriter, r *http.Request) {
    token := r.URL.Query().Get("token")
    if token == "" {
        utils.RespondBadRequest(w, "token is required")
        return
    }
    if err := h.svc.VerifyEmail(r.Context(), token); err != nil {
        utils.RespondBadRequest(w, err.Error())
        return
    }
    utils.RespondSuccess(w, map[string]string{"message": "email verified successfully"})
}

// ForgotPassword godoc
// @Summary Request password reset
// @Description Send password reset email
// @Tags authentication
// @Accept json
// @Produce json
// @Param request body dto.ForgotPasswordRequest true "Email address"
// @Success 200 {object} utils.SuccessResponse
// @Router /auth/forgot-password [post]
func (h *Handler) ForgotPassword(w http.ResponseWriter, r *http.Request) {
    var req dto.ForgotPasswordRequest
    if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
        utils.RespondBadRequest(w, "invalid request body")
        return
    }
    if err := h.svc.ForgotPassword(r.Context(), req.Email); err != nil {
        utils.RespondInternalError(w, "failed to process request")
        return
    }
    utils.RespondSuccess(w, map[string]string{"message": "if the email exists, a reset link has been sent"})
}

// ResetPassword godoc
// @Summary Reset password
// @Description Reset password with token from email
// @Tags authentication
// @Accept json
// @Produce json
// @Param request body dto.ResetPasswordRequest true "Token and new password"
// @Success 200 {object} utils.SuccessResponse
// @Failure 400 {object} utils.ErrorResponse
// @Router /auth/reset-password [post]
func (h *Handler) ResetPassword(w http.ResponseWriter, r *http.Request) {
    var req dto.ResetPasswordRequest
    if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
        utils.RespondBadRequest(w, "invalid request body")
        return
    }
    if err := h.validator.Validate(req); err != nil {
        utils.RespondBadRequest(w, err.Error())
        return
    }
    if err := h.svc.ResetPassword(r.Context(), req.Token, req.NewPassword); err != nil {
        utils.RespondBadRequest(w, err.Error())
        return
    }
    utils.RespondSuccess(w, map[string]string{"message": "password reset successfully"})
}

// ResendVerification godoc
// @Summary Resend verification email
// @Description Resend email verification link
// @Tags authentication
// @Accept json
// @Produce json
// @Param request body dto.ResendVerificationRequest true "Email address"
// @Success 200 {object} utils.SuccessResponse
// @Router /auth/resend-verification [post]
func (h *Handler) ResendVerification(w http.ResponseWriter, r *http.Request) {
    var req dto.ResendVerificationRequest
    if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
        utils.RespondBadRequest(w, "invalid request body")
        return
    }
    if err := h.svc.ResendVerification(r.Context(), req.Email); err != nil {
        utils.RespondInternalError(w, "failed to resend verification")
        return
    }
    utils.RespondSuccess(w, map[string]string{"message": "if the email exists, a verification link has been sent"})
}
```

- [ ] **Step 2: Add routes**

Add to `internal/authentication/interfaces/http/routes.go`:

```go
r.Get("/verify-email", handler.VerifyEmail)
r.Post("/forgot-password", handler.ForgotPassword)
r.Post("/reset-password", handler.ResetPassword)
r.Post("/resend-verification", handler.ResendVerification)
```

These are public routes (no auth middleware).

- [ ] **Step 3: Verify build**

Run: `go build ./...`
Expected: PASS

- [ ] **Step 4: Commit**

```bash
git add internal/authentication/interfaces/http/
git commit -m "auth: add verification and password reset HTTP endpoints"
```

---

### Task 10: Wire Email Module

**Files:**
- Modify: `internal/authentication/module.go`
- Modify: `cmd/api/main.go`

**Interfaces:**
- Consumes: `email.Module` from Task 2

- [ ] **Step 1: Update authentication module**

Update `internal/authentication/module.go` to accept emailer:

```go
var Module = fx.Module("authentication",
    fx.Provide(
        persistence.NewUserRepository,
        persistence.NewRefreshTokenRepository,
        service.NewAuthenticationService,
        httpHandler.NewHandler,
    ),
)
```

The `NewAuthenticationService` will now receive `domain.Emailer` via Fx injection.

- [ ] **Step 2: Update main.go**

Add email module to Fx:

```go
import "github.com/IDTS-LAB/go-codebase/internal/infrastructure/email"

// In fx.New():
email.Module,
```

- [ ] **Step 3: Verify build**

Run: `go build ./cmd/api`
Expected: PASS

- [ ] **Step 4: Verify app starts**

Run: `make run` (Ctrl+C after seeing "starting server")
Expected: Server starts without errors

- [ ] **Step 5: Commit**

```bash
git add internal/authentication/module.go cmd/api/main.go
git commit -m "auth: wire email module into Fx container"
```

---

### Task 11: Tests

**Files:**
- Create: `internal/infrastructure/email/email_test.go`
- Create: `internal/authentication/application/service/authentication_service_test.go`

**Interfaces:**
- Consumes: All previous tasks

- [ ] **Step 1: Test console provider**

```go
// internal/infrastructure/email/email_test.go
package email

import (
    "testing"
)

func TestConsoleMailer(t *testing.T) {
    mailer := NewConsoleMailer("test@example.com", "Test App", "http://localhost:3000")

    if err := mailer.SendVerification("user@test.com", "Test User", "abc123"); err != nil {
        t.Errorf("SendVerification failed: %v", err)
    }
    if err := mailer.SendPasswordReset("user@test.com", "Test User", "xyz789"); err != nil {
        t.Errorf("SendPasswordReset failed: %v", err)
    }
    if err := mailer.SendWelcome("user@test.com", "Test User"); err != nil {
        t.Errorf("SendWelcome failed: %v", err)
    }
    if err := mailer.SendInvite("user@test.com", "Test User", "Admin"); err != nil {
        t.Errorf("SendInvite failed: %v", err)
    }
}

func TestNewEmailer(t *testing.T) {
    // Test default (console) provider
    cfg := &config.Config{}
    cfg.Email.Provider = "console"
    cfg.Email.From = "test@example.com"
    cfg.Email.FromName = "Test"
    cfg.Email.FrontendURL = "http://localhost:3000"

    mailer := NewEmailer(cfg)
    if mailer == nil {
        t.Error("NewEmailer returned nil")
    }
}
```

- [ ] **Step 2: Run tests**

Run: `go test ./internal/infrastructure/email/ -v`
Expected: PASS

- [ ] **Step 3: Commit**

```bash
git add internal/infrastructure/email/email_test.go
git commit -m "email: add provider tests"
```

---

### Task 12: Swagger + Final Verification

**Files:**
- Run: `make swagger`

- [ ] **Step 1: Regenerate swagger docs**

Run: `make swagger`
Expected: New endpoints appear in docs

- [ ] **Step 2: Verify all endpoints in swagger**

Run: `python3 -c "import json; d=json.load(open('docs/swagger.json')); [print(p) for p in sorted(d['paths'].keys()) if 'auth' in p]"`
Expected: Shows verify-email, forgot-password, reset-password, resend-verification

- [ ] **Step 3: Full build and vet**

Run: `go build ./... && go vet ./...`
Expected: PASS

- [ ] **Step 4: Commit**

```bash
git add docs/
git commit -m "docs: regenerate swagger with email endpoints"
```
