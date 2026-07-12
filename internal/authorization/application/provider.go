package application

import (
	"context"

	"github.com/IDTS-LAB/go-codebase/internal/authorization/domain/repository"
	"github.com/IDTS-LAB/go-codebase/internal/authorization/public"
	"github.com/google/uuid"
)

type authorizationProvider struct {
	userRoleRepo repository.UserRoleRepository
}

func NewAuthorizationProvider(userRoleRepo repository.UserRoleRepository) public.AuthorizationProvider {
	return &authorizationProvider{userRoleRepo: userRoleRepo}
}

func (p *authorizationProvider) GetUserRoles(ctx context.Context, userID uuid.UUID) ([]string, error) {
	roles, err := p.userRoleRepo.GetRolesByUserID(ctx, userID)
	if err != nil {
		return nil, err
	}
	names := make([]string, len(roles))
	for i, r := range roles {
		names[i] = r.Name
	}
	return names, nil
}
