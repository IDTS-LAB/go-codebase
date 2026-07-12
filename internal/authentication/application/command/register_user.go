package command

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"time"

	"github.com/IDTS-LAB/go-codebase/internal/authentication/domain/entity"
	"github.com/IDTS-LAB/go-codebase/internal/authentication/domain/event"
	"github.com/IDTS-LAB/go-codebase/internal/authentication/domain/repository"
	"github.com/IDTS-LAB/go-codebase/internal/shared/events"
	"golang.org/x/crypto/bcrypt"
)

type RegisterUserCommand struct {
	Email    string
	Password string
	Name     string
}

type RegisterUserHandler struct {
	userRepo repository.UserRepository
	bus      events.EventBus
}

func NewRegisterUserHandler(userRepo repository.UserRepository, bus events.EventBus) *RegisterUserHandler {
	return &RegisterUserHandler{userRepo: userRepo, bus: bus}
}

func (h *RegisterUserHandler) Handle(ctx context.Context, cmd any) (any, error) {
	c := cmd.(RegisterUserCommand)
	existing, _ := h.userRepo.GetByEmail(ctx, c.Email)
	if existing != nil {
		return nil, ErrEmailAlreadyExists
	}

	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(c.Password), bcrypt.DefaultCost)
	if err != nil {
		return nil, fmt.Errorf("hash password: %w", err)
	}

	user := entity.NewUser(c.Email, string(hashedPassword), c.Name)
	if err := h.userRepo.Create(ctx, user); err != nil {
		return nil, err
	}

	token, err := generateToken()
	if err != nil {
		return nil, fmt.Errorf("generate verification token: %w", err)
	}
	expires := time.Now().Add(24 * time.Hour)
	hashed := hashToken(token)
	user.EmailVerifyToken = &hashed
	user.EmailVerifyExpires = &expires
	if err := h.userRepo.Update(ctx, user); err != nil {
		return nil, err
	}

	_ = h.bus.Publish(ctx, events.Event{
		Type: event.UserRegisteredEvent,
		Payload: event.UserRegistered{
			Email:             user.Email,
			Name:              user.Name,
			VerificationToken: token,
		},
	})

	return user, nil
}

func generateToken() (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return hex.EncodeToString(b), nil
}

func hashToken(token string) string {
	h := sha256.Sum256([]byte(token))
	return hex.EncodeToString(h[:])
}
