package postgres_test

import (
	"context"
	"reflect"
	"testing"

	"github.com/saiddis/todev"
	"github.com/saiddis/todev/postgres"
)

func TestContributorService_CreateContributor(t *testing.T) {
	t.Run("OK", func(t *testing.T) {
		WithSchema(t, createContributor_OK)
	})

	t.Run("Errors", func(t *testing.T) {
		WithSchema(t, func(t testing.TB, conn *postgres.Conn) {
			createContributors_Errors(t.(*testing.T), conn)
		})
	})
}

func createContributor_OK(t testing.TB, conn *postgres.Conn) {
	rs := postgres.NewRepoService(conn)
	ctx := context.Background()
	_, ctx0 := MustCreateUser(t, ctx, conn, &todev.User{Name: "bob", Email: "bob@gmail.com"})
	_, ctx1 := MustCreateUser(t, ctx, conn, &todev.User{Name: "judy", Email: "judy@gmail.com"})
	repo := MustCreateRepo(t, ctx0, rs, &todev.Repo{Name: "NAME"})

	cs := postgres.NewContrubutorService(conn)

	contributor := &todev.Contributor{
		RepoID: repo.ID,
	}

	if err := cs.CreateContributor(ctx1, contributor); err != nil {
		t.Fatal(err)
	} else if got, want := contributor.ID, 2; got != want {
		t.Fatalf("ID=%d, want %d", got, want)
	}

	if other, err := cs.FindContributorByID(ctx1, contributor.ID); err != nil {
		t.Fatal(err)
	} else if !reflect.DeepEqual(contributor, other) {
		t.Fatalf("mismatch: %#v !=\n %#v", contributor, other)
	}
}

func createContributors_Errors(t *testing.T, conn *postgres.Conn) {
	ctx := context.Background()
	repo, ctx0 := MustCreateUser(t, ctx, conn, &todev.User{Name: "said", Email: "said@gmail.com"})
	type testData struct {
		ctx      context.Context
		input    *todev.Contributor
		expected error
	}
	tests := map[string]testData{
		"ErrDialRequired": testData{
			ctx:   ctx0,
			input: &todev.Contributor{},
			expected: &todev.Error{
				Code:    todev.EINVALID,
				Message: "Repo required for contributing.",
			},
		},
		"ErrUserRequired": testData{
			ctx:   ctx,
			input: &todev.Contributor{RepoID: repo.ID},
			expected: &todev.Error{
				Code:    todev.EUNAUTHORIZED,
				Message: "You must be logged in to join a repo.",
			},
		},
	}

	rs := postgres.NewRepoService(conn)
	cs := postgres.NewContrubutorService(conn)
	MustCreateRepo(t, ctx0, rs, &todev.Repo{Name: "NAME"})
	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			if err := cs.CreateContributor(tt.ctx, tt.input); err.Error() != tt.expected.Error() {
				t.Fatalf("unexpected error: %#v, want %#v", err, tt.expected)
			}
		})
	}
}

func MustCreateContributor(tb testing.TB, ctx context.Context, s *postgres.ContributorService, contributor *todev.Contributor) *todev.Contributor {
	tb.Helper()

	err := s.CreateContributor(ctx, contributor)
	if err != nil {
		tb.Fatal(err)
	}
	return contributor
}

func MustFindContributorByID(tb testing.TB, ctx context.Context, s *postgres.ContributorService, id int) *todev.Contributor {
	tb.Helper()

	contributor, err := s.FindContributorByID(ctx, id)
	if err != nil {
		tb.Fatal(err)
	}
	return contributor
}
