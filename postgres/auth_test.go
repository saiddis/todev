package postgres_test

import (
	"context"
	"reflect"
	"testing"
	"time"

	_ "github.com/lib/pq"
	"github.com/saiddis/todev"
	"github.com/saiddis/todev/postgres"
)

func TestAuthService_CreateAuth(t *testing.T) {
	t.Run("OK", func(t *testing.T) {
		WithSchema(t, createAuth_OK)
	})

	t.Run("ErrSourceIDRequired", func(t *testing.T) {
		WithSchema(t, createAuth_ErrSourceIDRequired)
	})

	t.Run("ErrSourceRequired", func(t *testing.T) {
		WithSchema(t, createAuth_ErrSourceRequired)
	})

	t.Run("ErrAccessTokenRequired", func(t *testing.T) {
		WithSchema(t, createAuth_ErrAccessTokenRequired)
	})

	t.Run("ErrUserRequired", func(t *testing.T) {
		WithSchema(t, createAuth_ErrUserRequired)
	})

}

func TestAuthService_DeleteAuth(t *testing.T) {
	t.Run("OK", func(t *testing.T) {
		WithSchema(t, deleteAuth_OK)
	})

	t.Run("ErrNotFound", func(t *testing.T) {
		WithSchema(t, deleteAuth_ErrNotFound)
	})
	t.Run("ErrUnauthorized", func(t *testing.T) {
		WithSchema(t, deleteAuth_ErrUnauthorized)
	})
}

func TestAuthService_FindAuthByID(t *testing.T) {
	t.Run("ErrNotFound", func(t *testing.T) {
		WithSchema(t, findAuthByID_ErrNotFound)
	})
}

func TestAuthService_FindAuths(t *testing.T) {
	t.Run("OK", func(t *testing.T) {
		WithSchema(t, findAuths_OK)
	})
}

func createAuth_OK(t testing.TB, conn *postgres.Conn) {
	s := postgres.NewAuthService(conn)

	expiry := time.Date(2000, time.January, 1, 0, 0, 0, 0, time.UTC)
	auth := &todev.Auth{
		Source:       todev.AuthSourceGitHub,
		SourceID:     "SOURCEID",
		AccessToken:  "ACCESS",
		RefreshToken: "REFRESH",
		Expiry:       expiry,
		User: &todev.User{
			Name:  "jill",
			Email: "jill@gmail.com",
		},
	}

	// Create new auth object & ensure ID and timestamps are returned.
	if err := s.CreateAuth(context.Background(), auth); err != nil {
		t.Fatal(err)
	} else if got, want := auth.ID, 1; got != want {
		t.Fatalf("ID=%v, want %v", got, want)
	} else if auth.CreatedAt.IsZero() {
		t.Fatal("expected created at")
	} else if auth.UpdatedAt.IsZero() {
		t.Fatal("expected updated at")
	}

	// Fetch auth from dataabase and compare.
	if other, err := s.FindAuthByID(context.Background(), 1); err != nil {
		t.Fatalf("error retrivieng auths by ID: %v", err)
	} else if !reflect.DeepEqual(auth, other) {
		t.Fatalf("mismatch: \n%#v != \n%#v", auth, other)
	}

	// Fetching user should return auths.
	if user, err := postgres.NewUserService(conn).FindUserByID(context.Background(), 1); err != nil {
		t.Fatalf("error retrieving user by ID: %v", err)
	} else if len(user.Auths) != 1 {
		t.Fatal("expected auth")
	} else if auth := user.Auths[0]; auth.ID != 1 {
		t.Fatalf("unexpected auth: %#v", auth)
	}
}

func createAuth_ErrSourceRequired(t testing.TB, conn *postgres.Conn) {
	if err := postgres.NewAuthService(conn).CreateAuth(context.Background(), &todev.Auth{
		User: &todev.User{
			Name: "NAME",
		},
	}); err == nil {
		t.Fatal("error expected")
	} else if todev.ErrorCode(err) != todev.EINVALID || todev.ErrorMessage(err) != "Source required." {
		t.Fatalf("unexpected error: %v", err)
	}
}
func createAuth_ErrSourceIDRequired(t testing.TB, conn *postgres.Conn) {
	if err := postgres.NewAuthService(conn).CreateAuth(context.Background(), &todev.Auth{
		Source: todev.AuthSourceGitHub,
		User:   &todev.User{Name: "NAME"},
	}); err == nil {
		t.Fatal("expected error")
	} else if todev.ErrorCode(err) != todev.EINVALID || todev.ErrorMessage(err) != "Source ID required." {
		t.Fatalf("unexpected error: %#v", err)
	}
}

func createAuth_ErrAccessTokenRequired(t testing.TB, conn *postgres.Conn) {
	if err := postgres.NewAuthService(conn).CreateAuth(context.Background(), &todev.Auth{
		Source:   todev.AuthSourceGitHub,
		SourceID: "X",
		User:     &todev.User{Name: "NAME"},
	}); err == nil {
		t.Fatal("expected error")
	} else if todev.ErrorCode(err) != todev.EINVALID || todev.ErrorMessage(err) != "Access token required." {
		t.Fatalf("unexpected error: %#v", err)
	}
}

func createAuth_ErrUserRequired(t testing.TB, conn *postgres.Conn) {
	if err := postgres.NewAuthService(conn).CreateAuth(context.Background(), &todev.Auth{}); err == nil {
		t.Fatal("expected error")
	} else if todev.ErrorCode(err) != todev.EINVALID || todev.ErrorMessage(err) != "User required." {
		t.Fatalf("unexpected error: %#v", err)
	}
}

func deleteAuth_OK(t testing.TB, conn *postgres.Conn) {
	s := postgres.NewAuthService(conn)
	auth0, ctx0 := MustCreateAuth(t, context.Background(), s, &todev.Auth{
		Source:      todev.AuthSourceGitHub,
		SourceID:    "X",
		AccessToken: "X",
		User:        &todev.User{Name: "NAME"},
	})

	if err := s.DeleteAuth(ctx0, auth0.ID); err != nil {
		t.Fatalf("error deleting user: %v", err)
	} else if _, err := s.FindAuthByID(ctx0, auth0.ID); todev.ErrorCode(err) != todev.ENOTFOUND {
		t.Fatalf("unexpected error: %v", err)
	}
}

func deleteAuth_ErrNotFound(t testing.TB, conn *postgres.Conn) {
	if err := postgres.NewAuthService(conn).DeleteAuth(context.Background(), 1); todev.ErrorCode(err) != todev.ENOTFOUND {
		t.Fatalf("unexpected error: %#v", err)
	}
}

func deleteAuth_ErrUnauthorized(t testing.TB, conn *postgres.Conn) {
	s := postgres.NewAuthService(conn)
	auth0, _ := MustCreateAuth(t, context.Background(), s, &todev.Auth{
		Source:      todev.AuthSourceGitHub,
		SourceID:    "X",
		AccessToken: "X",
		User:        &todev.User{Name: "NAME"},
	})
	_, ctx1 := MustCreateAuth(t, context.Background(), s, &todev.Auth{
		Source:      todev.AuthSourceGitHub,
		SourceID:    "Y",
		AccessToken: "Y",
		User:        &todev.User{Name: "NAME"},
	})

	if err := s.DeleteAuth(ctx1, auth0.ID); err == nil {
		t.Fatal("expected error")
	} else if todev.ErrorCode(err) != todev.EUNAUTHORIZED || todev.ErrorMessage(err) != "You are not allowed to delete this auth." {
		t.Fatalf("unexpected error: %v", err)
	}
}

func findAuthByID_ErrNotFound(t testing.TB, conn *postgres.Conn) {
	if _, err := postgres.NewAuthService(conn).FindAuthByID(context.Background(), 1); todev.ErrorCode(err) != todev.ENOTFOUND {
		t.Fatalf("unexpected error: %#v", err)
	}
}

func findAuths_OK(t testing.TB, conn *postgres.Conn) {
	s := postgres.NewAuthService(conn)
	ctx := context.Background()

	MustCreateAuth(t, context.Background(), s, &todev.Auth{
		Source:      "SRCA",
		SourceID:    "X1",
		AccessToken: "ACCESSX1",
		User:        &todev.User{Name: "X", Email: "x@y.com"},
	})
	MustCreateAuth(t, context.Background(), s, &todev.Auth{
		Source:      "SRCB",
		SourceID:    "X2",
		AccessToken: "ACCESSX2",
		User:        &todev.User{Name: "X", Email: "x@y.com"},
	})
	MustCreateAuth(t, context.Background(), s, &todev.Auth{
		Source:      todev.AuthSourceGitHub,
		SourceID:    "Y",
		AccessToken: "ACCESSY",
		User:        &todev.User{Name: "Y"},
	})

	userID := 1
	if a, n, err := s.FindAuths(ctx, todev.AuthFilter{UserID: &userID}); err != nil {
		t.Fatal(err)
	} else if got, want := len(a), 2; got != want {
		t.Fatalf("len=%v, want %v", got, want)
	} else if got, want := a[0].SourceID, "X1"; got != want {
		t.Fatalf("SourceID=%v, want %v", got, want)
	} else if got, want := a[1].SourceID, "X2"; got != want {
		t.Logf("auth: %#v", a[1])
		t.Fatalf("SourceID=%v, want %v", got, want)
	} else if got, want := n, 2; got != want {
		t.Fatalf("n=%v, want %v", got, want)
	}

}

func MustCreateAuth(tb testing.TB, ctx context.Context, service *postgres.AuthService, auth *todev.Auth) (*todev.Auth, context.Context) {
	tb.Helper()
	if err := service.CreateAuth(ctx, auth); err != nil {
		tb.Fatalf("error creating auth: %v", err)
	}
	return auth, todev.NewContextWithUser(ctx, auth.User)
}
