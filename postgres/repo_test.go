package postgres_test

import (
	"context"
	"reflect"
	"strings"
	"testing"

	"github.com/saiddis/todev"
	"github.com/saiddis/todev/postgres"
)

func TestRepoService_CreateRepo(t *testing.T) {
	t.Run("OK", func(t *testing.T) {
		WithSchema(t, createRepo_OK)
	})

	t.Run("Errors", func(t *testing.T) {
		WithSchema(t, func(t testing.TB, conn *postgres.Conn) {
			createRepo_Errors(t.(*testing.T), conn)
		})
	})
}

func TestRepoService_UpdateRepo(t *testing.T) {
	t.Run("OK", func(t *testing.T) {
		WithSchema(t, updateRepo_OK)
	})
}

func TestRepoService_FindRepos(t *testing.T) {
	t.Run("owned", func(t *testing.T) {
		WithSchema(t, findRepos_owned)
	})
	t.Run("Member_of", func(t *testing.T) {
		WithSchema(t, findRepos_MemberOf)
	})
	t.Run("InviteCode", func(t *testing.T) {
		WithSchema(t, findRepos_InviteCode)
	})
}

func TestRepoService_DeleteRepo(t *testing.T) {
	t.Run("OK", func(t *testing.T) {
		WithSchema(t, deleteRepo_OK)
	})
}

func createRepo_OK(t testing.TB, conn *postgres.Conn) {
	_, ctx0 := MustCreateUser(t, context.Background(), conn, &todev.User{Name: "said", Email: "said@gmail.com"})

	s := postgres.NewRepoService(conn)
	repo := &todev.Repo{
		Name: "NewRepo",
	}

	if err := s.CreateRepo(ctx0, repo); err != nil {
		t.Fatal(err)
	} else if got, want := repo.ID, 1; got != want {
		t.Fatalf("ID=%d, want %d", got, want)
	} else if got, want := repo.UserID, 1; got != want {
		t.Fatalf("UserID=%d, want %d", got, want)
	} else if repo.InviteCode == "" {
		t.Fatal("expected invite code genearation")
	} else if repo.CreatedAt.IsZero() {
		t.Fatal("expected created at")
	} else if repo.UpdatedAt.IsZero() {
		t.Fatal("expected updated at")
	} else if repo.User == nil {
		t.Fatal("expected user")
	}

	if other, err := s.FindRepoByID(ctx0, repo.ID); err != nil {
		t.Fatal(err)
	} else if !reflect.DeepEqual(repo, other) {
		t.Fatalf("mismatch: %#v !=\n %#v", repo, other)
	}
}

func createRepo_Errors(t *testing.T, conn *postgres.Conn) {
	_, ctx0 := MustCreateUser(t, context.Background(), conn, &todev.User{Name: "said", Email: "said@gmail.com"})
	type testData struct {
		ctx      context.Context
		input    *todev.Repo
		expected error
	}
	tests := map[string]testData{
		"ErrNameRequired": testData{
			ctx:   ctx0,
			input: &todev.Repo{},
			expected: &todev.Error{
				Code:    todev.EINVALID,
				Message: "Repo name required.",
			},
		},
		"ErrNameTooLong": testData{
			ctx: ctx0,
			input: &todev.Repo{
				Name: strings.Repeat("X", todev.MaxRepoNameLen+1),
			},
			expected: &todev.Error{
				Code:    todev.EINVALID,
				Message: "Repo name too long.",
			},
		},
		"ErrUserRequired": testData{
			ctx: context.Background(),
			input: &todev.Repo{
				Name: strings.Repeat("X", todev.MaxRepoNameLen+1),
			},
			expected: &todev.Error{
				Code:    todev.EUNAUTHORIZED,
				Message: "You must be logged in to create a repo.",
			},
		},
	}

	s := postgres.NewRepoService(conn)
	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			if err := s.CreateRepo(tt.ctx, tt.input); err.Error() != tt.expected.Error() {
				t.Fatalf("unexpected error: %#v, want %#v", err, tt.expected)
			}
		})
	}
}

func updateRepo_OK(t testing.TB, conn *postgres.Conn) {
	s := postgres.NewRepoService(conn)

	_, ctx0 := MustCreateUser(t, context.Background(), conn, &todev.User{Name: "said", Email: "said@gmail.com"})
	repo := MustCreateRepo(t, ctx0, conn, &todev.Repo{Name: "NAME"})

	newName := "myrepo"
	uu, err := s.UpdateRepo(ctx0, repo.ID, todev.RepoUpdate{Name: &newName})
	if err != nil {
		t.Fatal(err)
	} else if got, want := uu.Name, "myrepo"; got != want {
		t.Fatalf("Name=%s, want %s", got, want)
	}

	if other, err := s.FindRepoByID(ctx0, 1); err != nil {
		t.Fatal(err)
	} else if !reflect.DeepEqual(uu, other) {
		t.Fatalf("mismatch: %#v !=\n %#v", uu, other)
	}
}

func findRepos_owned(t testing.TB, conn *postgres.Conn) {
	ctx := context.Background()
	s := postgres.NewRepoService(conn)
	_, ctx0 := MustCreateUser(t, ctx, conn, &todev.User{Name: "bob", Email: "bob@gmail.com"})
	_, ctx1 := MustCreateUser(t, ctx, conn, &todev.User{Name: "judy", Email: "judy@gmail.com"})

	MustCreateRepo(t, ctx0, conn, &todev.Repo{Name: "repo1"})
	MustCreateRepo(t, ctx0, conn, &todev.Repo{Name: "repo2"})
	MustCreateRepo(t, ctx1, conn, &todev.Repo{Name: "repo3"})

	if repos, n, err := s.FindRepos(ctx0, todev.RepoFilter{}); err != nil {
		t.Fatal(err)
	} else if got, want := len(repos), 2; got != want {
		t.Fatalf("len=%d, want %d", got, want)
	} else if got, want := repos[0].Name, "repo1"; got != want {
		t.Fatalf("repos[0].Name=%s, want %s", got, want)
	} else if got, want := repos[1].Name, "repo2"; got != want {
		t.Fatalf("repos[1].Name=%s, want %s", got, want)
	} else if got, want := n, 2; got != want {
		t.Fatalf("n=%d, want %d", got, want)
	}
}

func findRepos_MemberOf(t testing.TB, conn *postgres.Conn) {
	rs := postgres.NewRepoService(conn)

	ctx := context.Background()
	_, ctx0 := MustCreateUser(t, ctx, conn, &todev.User{Name: "bob", Email: "bob@gmail.com"})
	user1, ctx1 := MustCreateUser(t, ctx, conn, &todev.User{Name: "judy", Email: "judy@gmail.com"})

	repo1 := MustCreateRepo(t, ctx0, conn, &todev.Repo{Name: "repo1"})
	MustCreateRepo(t, ctx0, conn, &todev.Repo{Name: "repo2"})
	MustCreateContributor(t, ctx1, conn, &todev.Contributor{RepoID: repo1.ID, UserID: user1.ID})

	if repos, n, err := rs.FindRepos(ctx1, todev.RepoFilter{}); err != nil {
		t.Fatal(err)
	} else if got, want := len(repos), 1; got != want {
		t.Fatalf("len=%d, want %d", got, want)
	} else if got, want := repos[0].Name, "repo1"; got != want {
		t.Fatalf("repos[0].Name=%s, want %s", got, want)
	} else if got, want := n, 1; got != want {
		t.Fatalf("n=%d, want %d", got, want)
	}
}

func findRepos_InviteCode(t testing.TB, conn *postgres.Conn) {
	s := postgres.NewRepoService(conn)
	ctx := context.Background()
	_, ctx0 := MustCreateUser(t, ctx, conn, &todev.User{Name: "bob", Email: "bob@gmail.com"})

	repo1 := MustCreateRepo(t, ctx0, conn, &todev.Repo{Name: "repo1"})
	MustCreateRepo(t, ctx0, conn, &todev.Repo{Name: "repo2"})

	if repos, n, err := s.FindRepos(ctx0, todev.RepoFilter{InviteCode: &repo1.InviteCode}); err != nil {
		t.Fatal(err)
	} else if got, want := len(repos), 1; got != want {
		t.Fatalf("len=%d, want %d", got, want)
	} else if got, want := repos[0].Name, "repo1"; got != want {
		t.Fatalf("repos[0].Name=%s, want %s", got, want)
	} else if got, want := n, 1; got != want {
		t.Fatalf("n=%d, want %d", got, want)
	}
}

func deleteRepo_OK(t testing.TB, conn *postgres.Conn) {
	s := postgres.NewRepoService(conn)
	_, ctx0 := MustCreateUser(t, context.Background(), conn, &todev.User{Name: "bob", Email: "bob@gmail.com"})
	repo := MustCreateRepo(t, ctx0, conn, &todev.Repo{Name: "NAME"})

	if err := s.DeleteRepo(ctx0, repo.ID); err != nil {
		t.Fatal(err)
	} else if _, err := s.FindRepoByID(ctx0, repo.ID); todev.ErrorCode(err) != todev.ENOTFOUND {
		t.Fatalf("unexpected error: %v", err)
	}
}

func MustFindRepoByID(tb testing.TB, ctx context.Context, s *postgres.RepoService, id int) *todev.Repo {
	tb.Helper()
	repo, err := s.FindRepoByID(ctx, id)
	if err != nil {
		tb.Fatalf("MustFindRepoByID: %v", err)
	}
	return repo
}

func MustCreateRepo(tb testing.TB, ctx context.Context, conn *postgres.Conn, repo *todev.Repo) *todev.Repo {
	tb.Helper()
	if err := postgres.NewRepoService(conn).CreateRepo(ctx, repo); err != nil {
		tb.Fatalf("MustCreateRepo: %v", err)
	}
	return repo
}
