package service

import (
	"context"
	"errors"
	"testing"

	"github.com/IDTS-LAB/go-codebase/internal/authentication/domain/entity"
	"github.com/IDTS-LAB/go-codebase/internal/core/domain"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

type mockRepo struct {
	mock.Mock
}

func (m *mockRepo) List(ctx context.Context, offset, limit int) ([]*entity.User, int, error) {
	args := m.Called(ctx, offset, limit)
	return args.Get(0).([]*entity.User), args.Int(1), args.Error(2)
}

func (m *mockRepo) GetByID(ctx context.Context, id uuid.UUID) (*entity.User, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*entity.User), args.Error(1)
}

func (m *mockRepo) Update(ctx context.Context, user *entity.User) error {
	args := m.Called(ctx, user)
	return args.Error(0)
}

func (m *mockRepo) Delete(ctx context.Context, id uuid.UUID) error {
	args := m.Called(ctx, id)
	return args.Error(0)
}

func makeUser(id uuid.UUID) *entity.User {
	u := entity.NewUser("test@example.com", "password", "Test User")
	u.ID = id
	return u
}

func TestUserService_List(t *testing.T) {
	repo := new(mockRepo)
	svc := NewUserService(repo)

	id1 := uuid.New()
	id2 := uuid.New()
	users := []*entity.User{makeUser(id1), makeUser(id2)}
	repo.On("List", mock.Anything, 0, 10).Return(users, 2, nil)

	result, total, err := svc.List(context.Background(), 0, 10)

	assert.NoError(t, err)
	assert.Equal(t, 2, total)
	assert.Len(t, result, 2)
	repo.AssertExpectations(t)
}

func TestUserService_GetByID_Found(t *testing.T) {
	repo := new(mockRepo)
	svc := NewUserService(repo)

	id := uuid.New()
	user := makeUser(id)
	repo.On("GetByID", mock.Anything, id).Return(user, nil)

	result, err := svc.GetByID(context.Background(), id)

	assert.NoError(t, err)
	assert.Equal(t, id, result.ID)
	repo.AssertExpectations(t)
}

func TestUserService_GetByID_NotFound(t *testing.T) {
	repo := new(mockRepo)
	svc := NewUserService(repo)

	id := uuid.New()
	repo.On("GetByID", mock.Anything, id).Return(nil, domain.ErrNotFound)

	result, err := svc.GetByID(context.Background(), id)

	assert.Error(t, err)
	assert.Nil(t, result)
	assert.ErrorIs(t, err, domain.ErrNotFound)
	repo.AssertExpectations(t)
}

func TestUserService_Update_Success(t *testing.T) {
	repo := new(mockRepo)
	svc := NewUserService(repo)

	id := uuid.New()
	user := makeUser(id)
	user.Email = "old@example.com"
	user.Name = "Old Name"
	user.IsActive = true

	repo.On("GetByID", mock.Anything, id).Return(user, nil)
	repo.On("Update", mock.Anything, mock.MatchedBy(func(u *entity.User) bool {
		return u.ID == id && u.Name == "New Name" && u.Email == "new@example.com" && u.IsActive == false
	})).Return(nil)

	result, err := svc.Update(context.Background(), id, "New Name", "new@example.com", false)

	assert.NoError(t, err)
	assert.Equal(t, "New Name", result.Name)
	assert.Equal(t, "new@example.com", result.Email)
	assert.False(t, result.IsActive)
	repo.AssertExpectations(t)
}

func TestUserService_Update_NotFound(t *testing.T) {
	repo := new(mockRepo)
	svc := NewUserService(repo)

	id := uuid.New()
	repo.On("GetByID", mock.Anything, id).Return(nil, domain.ErrNotFound)

	result, err := svc.Update(context.Background(), id, "Name", "email@example.com", true)

	assert.Error(t, err)
	assert.Nil(t, result)
	assert.ErrorIs(t, err, domain.ErrNotFound)
	repo.AssertExpectations(t)
}

func TestUserService_Update_Partial(t *testing.T) {
	repo := new(mockRepo)
	svc := NewUserService(repo)

	id := uuid.New()
	user := makeUser(id)
	user.Name = "Original"
	user.Email = "original@example.com"
	user.IsActive = true

	repo.On("GetByID", mock.Anything, id).Return(user, nil)
	repo.On("Update", mock.Anything, mock.MatchedBy(func(u *entity.User) bool {
		return u.ID == id && u.Name == "Original" && u.Email == "original@example.com" && u.IsActive == true
	})).Return(nil)

	result, err := svc.Update(context.Background(), id, "", "", true)

	assert.NoError(t, err)
	assert.Equal(t, "Original", result.Name)
	assert.Equal(t, "original@example.com", result.Email)
	assert.True(t, result.IsActive)
	repo.AssertExpectations(t)
}

func TestUserService_Update_RepoError(t *testing.T) {
	repo := new(mockRepo)
	svc := NewUserService(repo)

	id := uuid.New()
	user := makeUser(id)
	repo.On("GetByID", mock.Anything, id).Return(user, nil)
	repo.On("Update", mock.Anything, mock.Anything).Return(errors.New("db error"))

	result, err := svc.Update(context.Background(), id, "Name", "email@example.com", true)

	assert.Error(t, err)
	assert.Nil(t, result)
	assert.Contains(t, err.Error(), "update user")
	repo.AssertExpectations(t)
}

func TestUserService_Delete_Success(t *testing.T) {
	repo := new(mockRepo)
	svc := NewUserService(repo)

	id := uuid.New()
	user := makeUser(id)
	repo.On("GetByID", mock.Anything, id).Return(user, nil)
	repo.On("Update", mock.Anything, mock.MatchedBy(func(u *entity.User) bool {
		return u.ID == id && u.IsDeleted()
	})).Return(nil)

	err := svc.Delete(context.Background(), id)

	assert.NoError(t, err)
	repo.AssertExpectations(t)
}

func TestUserService_Delete_NotFound(t *testing.T) {
	repo := new(mockRepo)
	svc := NewUserService(repo)

	id := uuid.New()
	repo.On("GetByID", mock.Anything, id).Return(nil, domain.ErrNotFound)

	err := svc.Delete(context.Background(), id)

	assert.Error(t, err)
	assert.ErrorIs(t, err, domain.ErrNotFound)
	repo.AssertExpectations(t)
}
