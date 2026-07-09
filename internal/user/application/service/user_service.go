package service

import (
	"context"
	"fmt"

	"github.com/IDTS-LAB/go-codebase/internal/authentication/domain/entity"
	"github.com/IDTS-LAB/go-codebase/internal/user/domain/repository"
	"github.com/google/uuid"
)

type UserService struct {
	repo repository.UserRepository
}

func NewUserService(repo repository.UserRepository) *UserService {
	return &UserService{repo: repo}
}

func (s *UserService) List(ctx context.Context, offset, limit int) ([]*entity.User, int, error) {
	return s.repo.List(ctx, offset, limit)
}

func (s *UserService) GetByID(ctx context.Context, id uuid.UUID) (*entity.User, error) {
	user, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("user not found: %w", err)
	}
	return user, nil
}

func (s *UserService) Update(ctx context.Context, id uuid.UUID, name string, email string, isActive bool) (*entity.User, error) {
	user, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("user not found: %w", err)
	}
	if name != "" {
		user.Name = name
	}
	if email != "" {
		user.Email = email
	}
	user.IsActive = isActive
	user.Touch()

	if err := s.repo.Update(ctx, user); err != nil {
		return nil, fmt.Errorf("update user: %w", err)
	}
	return user, nil
}

func (s *UserService) Delete(ctx context.Context, id uuid.UUID) error {
	user, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return fmt.Errorf("user not found: %w", err)
	}
	user.SoftDelete()
	return s.repo.Update(ctx, user)
}
