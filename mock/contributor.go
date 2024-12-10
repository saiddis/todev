package mock

import (
	"context"

	"github.com/saiddis/todev"
)

var _ todev.ContributorService = (*ContributorService)(nil)

type ContributorService struct {
	FindContributorByIDFn func(ctx context.Context, id int) (*todev.Contributor, error)
	FindContributorsFn    func(ctx context.Context, filter todev.ContributorFilter) ([]*todev.Contributor, int, error)
	CreateContributorFn   func(ctx context.Context, repo *todev.Contributor) error
	UpdateContributorFn   func(ctx context.Context, id int, upd todev.ContributorUpdate) (*todev.Contributor, error)
	DeleteContributorFn   func(ctx context.Context, id int) error
}

func (s *ContributorService) FindContributorByID(ctx context.Context, id int) (*todev.Contributor, error) {
	return s.FindContributorByIDFn(ctx, id)
}

func (s *ContributorService) FindContributors(ctx context.Context, filter todev.ContributorFilter) ([]*todev.Contributor, int, error) {
	return s.FindContributors(ctx, filter)
}

func (s *ContributorService) CreateContributor(ctx context.Context, contributor *todev.Contributor) error {
	return s.CreateContributor(ctx, contributor)
}

func (s *ContributorService) UpdateContributor(ctx context.Context, id int, upd todev.ContributorUpdate) (*todev.Contributor, error) {
	return s.UpdateContributor(ctx, id, upd)
}

func (s *ContributorService) DeleteContributor(ctx context.Context, id int) error {
	return s.DeleteContributor(ctx, id)
}
