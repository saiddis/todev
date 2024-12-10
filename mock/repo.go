package mock

import (
	"context"

	"github.com/saiddis/todev"
)

var _ todev.RepoService = (*RepoService)(nil)

type RepoService struct {
	FindRepoByIDFn func(ctx context.Context, id int) (*todev.Repo, error)
	FindReposFn    func(ctx context.Context, filter todev.RepoFilter) ([]*todev.Repo, int, error)
	CreateRepoFn   func(ctx context.Context, repo *todev.Repo) error
	UpdateRepoFn   func(ctx context.Context, id int, upd todev.RepoUpdate) (*todev.Repo, error)
	DeleteRepoFn   func(ctx context.Context, id int) error
}

func (s *RepoService) FindRepoByID(ctx context.Context, id int) (*todev.Repo, error) {
	return s.FindRepoByIDFn(ctx, id)
}

func (s *RepoService) FindRepos(ctx context.Context, filter todev.RepoFilter) ([]*todev.Repo, int, error) {
	return s.FindRepos(ctx, filter)
}

func (s *RepoService) CreateRepo(ctx context.Context, repo *todev.Repo) error {
	return s.CreateRepo(ctx, repo)
}

func (s *RepoService) UpdateRepo(ctx context.Context, id int, upd todev.RepoUpdate) (*todev.Repo, error) {
	return s.UpdateRepo(ctx, id, upd)
}

func (s *RepoService) DeleteRepo(ctx context.Context, id int) error {
	return s.DeleteRepo(ctx, id)
}
