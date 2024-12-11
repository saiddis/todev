package http_test

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"
	"github.com/saiddis/todev"
	todevhttp "github.com/saiddis/todev/http"
)

// Ensure the HTTP server can return the repo listening in a variety of formats.
func TestRepoIndex(t *testing.T) {
	s := MustOpenServer(t)
	defer MustCloseServer(t, s)

	user0 := &todev.User{ID: 1, Name: "user1", APIKey: "apiKey"}
	ctx0 := todev.NewContextWithUser(context.Background(), user0)

	repo := &todev.Repo{
		ID:         1,
		UserID:     1,
		Name:       "repo1",
		InviteCode: "inviteCode",
		CreatedAt:  time.Date(2000, time.January, 1, 0, 0, 0, 0, time.UTC),
		UpdatedAt:  time.Date(2000, time.January, 1, 0, 0, 0, 0, time.UTC),
	}

	s.RepoService.FindReposFn = func(ctx context.Context, filter todev.RepoFilter) ([]*todev.Repo, int, error) {
		return []*todev.Repo{repo}, 1, nil
	}

	t.Run("JSON", func(t *testing.T) {
		s.UserService.FindUsersFn = func(ctx context.Context, filter todev.UserFilter) ([]*todev.User, int, error) {
			if filter.APIKey == nil || *filter.APIKey != "apiKey" {
				t.Fatalf("unexpected api key: %#v", filter.APIKey)
			}
			return []*todev.User{user0}, 1, nil
		}

		repoService := todevhttp.NewRepoService(todevhttp.NewClient(s.URL()))
		if repos, n, err := repoService.FindRepos(ctx0, todev.RepoFilter{}); err != nil {
			t.Fatal(fmt.Errorf("error finding repos: %v", err))
		} else if got, want := len(repos), 1; got != want {
			t.Fatalf("len=%d, want %d", got, want)
		} else if diff := cmp.Diff(repos[0], repo); diff != "" {
			t.Fatal(diff)
		} else if got, want := n, 1; got != want {
			t.Fatalf("n=%d, want %d", got, want)
		}

	})

}
