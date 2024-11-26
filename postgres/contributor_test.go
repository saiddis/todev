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

func TestContributorService_FindContributors(t *testing.T) {
	t.Run("RestrictToRepoContributor", func(t *testing.T) {
		WithSchema(t, findContributors_RestrictToRepoMember)
	})

	t.Run("FilterByID", func(t *testing.T) {
		WithSchema(t, findContributors_FilterByRepoID)
	})

	t.Run("FilterByUserID", func(t *testing.T) {
		WithSchema(t, findContributors_FilterByUserID)
	})
}

// TODO: Test UpdateContributor function

// func TestContributorService_UpdateContributor(t *testing.T) {
// 	t.Run("OK", func(t *testing.T) {
// 	})
// }
// func updateContributor(t testing.TB, conn *postgres.Conn) {
// 	ctx := context.Background()
// 	s := postgres.NewContrubutorService(conn)
//
// 	_, ctx0 := MustCreateUser(t, ctx, conn, &todev.User{Name: "bob"})
// 	_, ctx1 := MustCreateUser(t, ctx, conn, &todev.User{Name: "judy"})
//
// 	repo := MustCreateRepo(t, ctx0, conn, &todev.Repo{Name: "repo"})
// 	contributor := MustCreateContributor(t, ctx1, conn, &todev.Contributor{RepoID: repo.ID})
//
// }

func TestContributorService_DeleteContributor(t *testing.T) {
	t.Run("OK", func(t *testing.T) {
		WithSchema(t, func(t testing.TB, conn *postgres.Conn) {
			deleteContributor_OK(t.(*testing.T), conn)
		})
	})
	t.Run("Errors", func(t *testing.T) {
		WithSchema(t, func(t testing.TB, conn *postgres.Conn) {
			deleteContributor_Errors(t.(*testing.T), conn)
		})
	})
}

type testData struct {
	input    *todev.Contributor
	expected error
	ctx      context.Context
}

func deleteContributor_OK(t *testing.T, conn *postgres.Conn) {
	ctx := context.Background()
	s := postgres.NewContrubutorService(conn)

	_, ctx0 := MustCreateUser(t, ctx, conn, &todev.User{Name: "bob"})
	_, ctx1 := MustCreateUser(t, ctx, conn, &todev.User{Name: "judy"})
	_, ctx2 := MustCreateUser(t, ctx, conn, &todev.User{Name: "george"})
	repo := MustCreateRepo(t, ctx0, conn, &todev.Repo{Name: "repo"})

	tests := map[string]testData{
		"ByOwner": testData{
			input: MustCreateContributor(t, ctx1, conn, &todev.Contributor{RepoID: repo.ID}),
			expected: &todev.Error{
				Code:    todev.ENOTFOUND,
				Message: "Contributor not found.",
			},
			ctx: ctx0,
		},
		"ByContributor": testData{
			input: MustCreateContributor(t, ctx2, conn, &todev.Contributor{RepoID: repo.ID}),
			expected: &todev.Error{
				Code:    todev.ENOTFOUND,
				Message: "Contributor not found.",
			},
			ctx: ctx2,
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			if err := s.DeleteContritbutor(tt.ctx, tt.input.ID); err != nil {
				t.Fatal(err)
			}

			if _, err := s.FindContributorByID(tt.ctx, tt.input.ID); err == nil {
				t.Fatal("expected error")
			} else if tt.expected.Error() != err.Error() {
				t.Fatalf("unexpected error: %v", err)
			}
		})
	}
}

func deleteContributor_Errors(t *testing.T, conn *postgres.Conn) {
	ctx := context.Background()
	s := postgres.NewContrubutorService(conn)

	_, ctx0 := MustCreateUser(t, ctx, conn, &todev.User{Name: "bob"})
	_, ctx1 := MustCreateUser(t, ctx, conn, &todev.User{Name: "judy"})
	_, ctx2 := MustCreateUser(t, ctx, conn, &todev.User{Name: "george"})
	repo := MustCreateRepo(t, ctx0, conn, &todev.Repo{Name: "repo"})
	MustCreateContributor(t, ctx2, conn, &todev.Contributor{RepoID: repo.ID})

	tests := map[string]testData{
		"ErrCannotDeleteOwnerContributor": testData{
			input: &todev.Contributor{ID: 1},
			expected: &todev.Error{
				Code:    todev.ECONFLICT,
				Message: "Repo owner cannot be deleted.",
			},
			ctx: ctx0,
		},
		"ErrUnAuthorized": testData{
			input: MustCreateContributor(t, ctx1, conn, &todev.Contributor{RepoID: repo.ID}),
			expected: &todev.Error{
				Code:    todev.EUNAUTHORIZED,
				Message: "You do not have permission to delete the contributor.",
			},
			ctx: ctx2,
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			if err := s.DeleteContritbutor(tt.ctx, tt.input.ID); err == nil {
				t.Fatal("expected error")
			} else if tt.expected.Error() != err.Error() {
				t.Fatalf("unexpected error: %v", err)
			}
		})
	}
}

func findContributors_RestrictToRepoMember(t testing.TB, conn *postgres.Conn) {
	ctx := context.Background()
	s := postgres.NewContrubutorService(conn)

	_, ctx0 := MustCreateUser(t, ctx, conn, &todev.User{Name: "bob"})
	_, ctx1 := MustCreateUser(t, ctx, conn, &todev.User{Name: "judy"})
	_, ctx2 := MustCreateUser(t, ctx, conn, &todev.User{Name: "george"})

	repo0 := MustCreateRepo(t, ctx0, conn, &todev.Repo{Name: "repo0"})
	contributor0 := MustFindContributorByID(t, ctx0, conn, 1)
	contributor1 := MustCreateContributor(t, ctx1, conn, &todev.Contributor{RepoID: repo0.ID})
	contributor2 := MustCreateContributor(t, ctx2, conn, &todev.Contributor{RepoID: repo0.ID})

	repo1 := MustCreateRepo(t, ctx1, conn, &todev.Repo{Name: "repo1"})
	MustCreateContributor(t, ctx0, conn, &todev.Contributor{RepoID: repo1.ID})

	contributors, n, err := s.FindContributors(ctx2, todev.ContributorFilter{})
	if err != nil {
		t.Fatal(err)
	} else if got, want := len(contributors), 3; got != want {
		t.Fatalf("len=%d, want %d", got, want)
	} else if got, want := n, 3; got != want {
		t.Fatalf("n=%d, want %d", got, want)
	}

	// Contributor that requested a list of contributors must appear first.
	if got, want := contributors[0], contributor2; got.ID != want.ID {
		t.Fatalf("ID=%v, want %v", got.ID, want.ID)
	}

	// Remaining contributors should appear sorted by user name.
	if got, want := contributors[1], contributor0; got.ID != want.ID {
		t.Fatalf("ID=%d, want %d", got.ID, want.ID)
	}
	if got, want := contributors[2], contributor1; got.ID != want.ID {
		t.Fatalf("ID=%d, want %d", got.ID, want.ID)
	}
}

func findContributors_FilterByRepoID(t testing.TB, conn *postgres.Conn) {
	s := postgres.NewContrubutorService(conn)

	ctx := context.Background()
	_, ctx0 := MustCreateUser(t, ctx, conn, &todev.User{Name: "bob"})

	repo0 := MustCreateRepo(t, ctx0, conn, &todev.Repo{Name: "repo0"})
	MustCreateRepo(t, ctx0, conn, &todev.Repo{Name: "repo1"})

	// These repos will automatically create owner-contributor(1, 2)
	contributors, n, err := s.FindContributors(ctx0, todev.ContributorFilter{RepoID: &repo0.ID})
	if err != nil {
		t.Fatal(err)
	} else if got, want := len(contributors), 1; got != want {
		t.Fatalf("len=%d, want %d", got, want)
	} else if got, want := n, 1; got != want {
		t.Fatalf("n=%d, want %d", got, want)
	}
}

func findContributors_FilterByUserID(t testing.TB, conn *postgres.Conn) {
	s := postgres.NewContrubutorService(conn)

	ctx := context.Background()
	_, ctx0 := MustCreateUser(t, ctx, conn, &todev.User{Name: "bob"})
	user1, ctx1 := MustCreateUser(t, ctx, conn, &todev.User{Name: "judy"})

	repo0 := MustCreateRepo(t, ctx0, conn, &todev.Repo{Name: "repo0"})
	contributor0 := MustCreateContributor(t, ctx1, conn, &todev.Contributor{RepoID: repo0.ID})

	// These repos will automatically create owner-contributor(1, 2)
	contributors, n, err := s.FindContributors(ctx0, todev.ContributorFilter{UserID: &user1.ID})
	if err != nil {
		t.Fatal(err)
	} else if got, want := len(contributors), 1; got != want {
		t.Fatalf("len=%d, want %d", got, want)
	} else if got, want := n, 1; got != want {
		t.Fatalf("n=%d, want %d", got, want)
	} else if got, want := contributors[0].ID, contributor0.ID; got != want {
		t.Fatalf("ID=%d, want %d", got, want)
	}
}

func createContributor_OK(t testing.TB, conn *postgres.Conn) {
	ctx := context.Background()
	_, ctx0 := MustCreateUser(t, ctx, conn, &todev.User{Name: "bob", Email: "bob@gmail.com"})
	_, ctx1 := MustCreateUser(t, ctx, conn, &todev.User{Name: "judy", Email: "judy@gmail.com"})
	repo := MustCreateRepo(t, ctx0, conn, &todev.Repo{Name: "NAME"})

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

	cs := postgres.NewContrubutorService(conn)
	MustCreateRepo(t, ctx0, conn, &todev.Repo{Name: "NAME"})
	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			if err := cs.CreateContributor(tt.ctx, tt.input); err.Error() != tt.expected.Error() {
				t.Fatalf("unexpected error: %#v, want %#v", err, tt.expected)
			}
		})
	}
}

func MustCreateContributor(tb testing.TB, ctx context.Context, conn *postgres.Conn, contributor *todev.Contributor) *todev.Contributor {
	tb.Helper()

	err := postgres.NewContrubutorService(conn).CreateContributor(ctx, contributor)
	if err != nil {
		tb.Fatal(err)
	}
	return contributor
}

func MustFindContributorByID(tb testing.TB, ctx context.Context, conn *postgres.Conn, id int) *todev.Contributor {
	tb.Helper()

	contributor, err := postgres.NewContrubutorService(conn).FindContributorByID(ctx, id)
	if err != nil {
		tb.Fatal(err)
	}
	return contributor
}
