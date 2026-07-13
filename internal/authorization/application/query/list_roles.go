package query

import (
	"context"

	"github.com/IDTS-LAB/go-codebase/internal/authorization/domain/entity"
	"github.com/IDTS-LAB/go-codebase/internal/authorization/domain/repository"
)

type ListRolesQuery struct {
	Cursor *string
	Limit  int
}

type ListRolesResult struct {
	Roles      []*entity.Role
	NextCursor *string
	PrevCursor *string
	HasNext    bool
	HasPrev    bool
}

type ListRolesHandler struct {
	roleRepo repository.RoleRepository
}

func NewListRolesHandler(roleRepo repository.RoleRepository) *ListRolesHandler {
	return &ListRolesHandler{roleRepo: roleRepo}
}

func (h *ListRolesHandler) Handle(ctx context.Context, query any) (any, error) {
	q := query.(ListRolesQuery)
	roles, nextCursor, prevCursor, hasNext, hasPrev, err := h.roleRepo.GetAll(ctx, q.Cursor, q.Limit)
	if err != nil {
		return nil, err
	}
	return ListRolesResult{
		Roles:      roles,
		NextCursor: nextCursor,
		PrevCursor: prevCursor,
		HasNext:    hasNext,
		HasPrev:    hasPrev,
	}, nil
}
