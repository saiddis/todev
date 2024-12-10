package mock

import (
	"context"

	"github.com/saiddis/todev"
)

var _ todev.UserService = (*UserService)(nil)

type UserService struct {
	FindUserByIDFn func(ctx context.Context, id int) (*todev.User, error)
	FindUsersFn    func(ctx context.Context, filter todev.UserFilter) ([]*todev.User, int, error)
	CreateUserFn   func(ctx context.Context, user *todev.User) error
	UpdateUserFn   func(ctx context.Context, id int, upd todev.UserUpdate) (*todev.User, error)
	DeleteUserFn   func(ctx context.Context, id int) error
}

func (s *UserService) FindUserByID(ctx context.Context, id int) (*todev.User, error) {
	return s.FindUserByID(ctx, id)
}

func (s *UserService) FindUsers(ctx context.Context, filter todev.UserFilter) ([]*todev.User, int, error) {
	return s.FindUsersFn(ctx, filter)
}

func (s *UserService) CreateUser(ctx context.Context, user *todev.User) error {
	return s.CreateUserFn(ctx, user)
}

func (s *UserService) UpdateUser(ctx context.Context, id int, upd todev.UserUpdate) (*todev.User, error) {
	return s.UpdateUserFn(ctx, id, upd)
}

func (s *UserService) DeleteUser(ctx context.Context, id int) error {
	return s.DeleteUserFn(ctx, id)
}
