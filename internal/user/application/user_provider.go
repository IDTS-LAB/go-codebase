package application

import (
	"context"

	authEntity "github.com/IDTS-LAB/go-codebase/internal/authentication/domain/entity"
	"github.com/IDTS-LAB/go-codebase/internal/authorization/public"
	"github.com/IDTS-LAB/go-codebase/internal/shared/cqrs"
	"github.com/IDTS-LAB/go-codebase/internal/user/application/query"
	userPublic "github.com/IDTS-LAB/go-codebase/internal/user/public"
	"github.com/google/uuid"
)

type userProfileProvider struct {
	queryBus     cqrs.QueryBus
	authProvider public.AuthorizationProvider
}

func NewUserProfileProvider(queryBus cqrs.QueryBus, authProvider public.AuthorizationProvider) userPublic.UserProfileProvider {
	return &userProfileProvider{queryBus: queryBus, authProvider: authProvider}
}

func (p *userProfileProvider) GetProfile(ctx context.Context, userID uuid.UUID) (*userPublic.UserProfile, error) {
	resp, err := p.queryBus.Ask(ctx, query.GetUserQuery{ID: userID})
	if err != nil {
		return nil, err
	}

	user := resp.(*authEntity.User)

	profile := &userPublic.UserProfile{
		ID:        user.ID.String(),
		Email:     user.Email,
		Name:      user.Name,
		IsActive:  user.IsActive,
		CreatedAt: user.CreatedAt.Format("2006-01-02T15:04:05Z"),
		UpdatedAt: user.UpdatedAt.Format("2006-01-02T15:04:05Z"),
	}

	roles, err := p.authProvider.GetUserRoles(ctx, userID)
	if err == nil {
		profile.Roles = roles
	}

	return profile, nil
}
