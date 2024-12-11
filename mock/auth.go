package mock

import (
	"context"

	"github.com/saiddis/todev"
)

var _ todev.AuthService = (*AuthService)(nil)

type AuthService struct {
	FindAuthByIDFn func(ctx context.Context, id int) (*todev.Auth, error)
	FindAuthsFn    func(ctx context.Context, filter todev.AuthFilter) ([]*todev.Auth, int, error)
	CreateAuthFn   func(ctx context.Context, auth *todev.Auth) error
	DeleteAuthFn   func(ctx context.Context, id int) error
}

func (s *AuthService) FindAuthByID(ctx context.Context, id int) (*todev.Auth, error) {
	return s.FindAuthByIDFn(ctx, id)
}

func (s *AuthService) FindAuths(ctx context.Context, filter todev.AuthFilter) ([]*todev.Auth, int, error) {
	return s.FindAuthsFn(ctx, filter)
}

func (s *AuthService) CreateAuth(ctx context.Context, auth *todev.Auth) error {
	return s.CreateAuthFn(ctx, auth)
}

func (s *AuthService) DeleteAuth(ctx context.Context, id int) error {
	return s.DeleteAuthFn(ctx, id)
}
