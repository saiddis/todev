package postgres_test

import (
	"context"
	"reflect"
	"testing"

	_ "github.com/lib/pq"
	"github.com/saiddis/todev"
	"github.com/saiddis/todev/postgres"
)

func TestUserService_CreateUser(t *testing.T) {
	t.Run("OK", func(t *testing.T) {
		WithSchema(t, createUser_OK)
	})

	t.Run("ErrNameRequired", func(t *testing.T) {
		WithSchema(t, createUser_ErrInvalid)
	})
}

func TestUserService_UpdateUser(t *testing.T) {
	t.Run("OK", func(t *testing.T) {
		WithSchema(t, updateUser_OK)
	})

	t.Run("ErrUnauthorized", func(t *testing.T) {
		WithSchema(t, updateUser_ErrUnauthorized)
	})
}

func TestUserService_DeleteUser(t *testing.T) {
	t.Run("OK", func(t *testing.T) {
		WithSchema(t, deleteUser_OK)
	})

	t.Run("ErrNotAuthorized", func(t *testing.T) {
		WithSchema(t, deleteUser_ErrNotAuthorized)
	})
}

func TestUserService_FindUserByID(t *testing.T) {
	t.Run("ErrNotFound", func(t *testing.T) {
		WithSchema(t, findUserByID_ErrNotFound)
	})
}

func TestUserService_FindUsers(t *testing.T) {
	t.Run("OK", func(t *testing.T) {
		WithSchema(t, findUsers_OK)
	})
}

// MustCreateUser creates a user in the database. Fatal on error.
func MustCreateUser(tb testing.TB, ctx context.Context, conn *postgres.Conn, user *todev.User) (*todev.User, context.Context) {
	tb.Helper()

	if err := postgres.NewUserService(conn).CreateUser(ctx, user); err != nil {
		tb.Fatalf("MustCreateUser: %v", err)
	}
	return user, todev.NewContextWithUser(ctx, user)
}

func createUser_OK(t testing.TB, conn *postgres.Conn) {
	s := postgres.NewUserService(conn)

	u := &todev.User{
		Name:  "said",
		Email: "said@gmail.com",
	}

	if err := s.CreateUser(context.Background(), u); err != nil {
		t.Fatal(err)
	}

	if got, want := u.ID, 1; got != want {
		t.Errorf("ID=%v, want %v", got, want)
	}
	if u.CreatedAt.IsZero() {
		t.Error("expected created at")
	}
	if u.UpdatedAt.IsZero() {
		t.Error("expected updated at")
	}

	u2 := &todev.User{Name: "jane"}
	if err := s.CreateUser(context.Background(), u2); err != nil {
		t.Fatal(err)
	} else if got, want := u2.ID, 2; got != want {
		t.Errorf("ID=%v, want %v", got, want)
	}

	if other, err := s.FindUserByID(context.Background(), 1); err != nil {
		t.Fatal(err)
	} else if !reflect.DeepEqual(u, other) {
		t.Fatalf("mismatch:\n%#v !=\n %#v", u, other)
	}

}

func createUser_ErrInvalid(t testing.TB, conn *postgres.Conn) {
	s := postgres.NewUserService(conn)
	if err := s.CreateUser(context.Background(), &todev.User{}); err == nil {
		t.Fatal("error expected")
	} else if todev.ErrorCode(err) != todev.EINVALID || todev.ErrorMessage(err) != "User name required." {
		t.Fatalf("unexpected error: %#v", err)
	}
}

func updateUser_OK(t testing.TB, conn *postgres.Conn) {
	s := postgres.NewUserService(conn)
	user0, ctx0 := MustCreateUser(t, context.Background(), conn, &todev.User{
		Name:  "susy",
		Email: "susy@gmail.com",
	})

	// Update user.
	newName, newEmail := "jill", "jill@gmail.com"
	uu, err := s.UpdateUser(ctx0, user0.ID, todev.UserUpdate{
		Name:  &newName,
		Email: &newEmail,
	})
	if err != nil {
		t.Fatal(err)
	} else if got, want := uu.Name, "jill"; got != want {
		t.Fatalf("Name=%v, want %v", got, want)
	} else if got, want := uu.Email, "jill@gmail.com"; got != want {
		t.Fatalf("Email=%v, want %v", got, want)
	}

	// Fetch user from database & compare.
	if other, err := s.FindUserByID(context.Background(), 1); err != nil {
		t.Fatal(err)
	} else if !reflect.DeepEqual(uu, other) {
		t.Fatalf("mismatch: %#v != %#v", uu, other)
	}
}

func updateUser_ErrUnauthorized(t testing.TB, conn *postgres.Conn) {
	s := postgres.NewUserService(conn)
	user0, _ := MustCreateUser(t, context.Background(), conn, &todev.User{Name: "NAME0"})
	_, ctx1 := MustCreateUser(t, context.Background(), conn, &todev.User{Name: "NAME1"})

	newName := "NEWNAME"
	if _, err := s.UpdateUser(ctx1, user0.ID, todev.UserUpdate{Name: &newName}); err == nil {
		t.Fatal("error expected")
	} else if todev.ErrorCode(err) != todev.EUNAUTHORIZED || todev.ErrorMessage(err) != "You are not allowed to update this user." {
		t.Fatalf("unexpected error: %#v", err)
	}
}

func deleteUser_OK(t testing.TB, conn *postgres.Conn) {
	s := postgres.NewUserService(conn)
	user0, ctx0 := MustCreateUser(t, context.Background(), conn, &todev.User{Name: "john"})

	if err := s.DeleteUser(ctx0, user0.ID); err != nil {
		t.Fatalf("error deleting user: %v", err)
	}
}

func deleteUser_ErrNotAuthorized(t testing.TB, conn *postgres.Conn) {
	s := postgres.NewUserService(conn)
	user0, ctx0 := MustCreateUser(t, context.Background(), conn, &todev.User{Name: "john"})

	_ = s.DeleteUser(ctx0, user0.ID)

	if user1, err := s.FindUserByID(ctx0, user0.ID); user1 != nil {
		t.Fatalf("found deleted user: %+v", user1)
	} else if todev.ErrorCode(err) != todev.ENOTFOUND || todev.ErrorMessage(err) != "User not found." {
		t.Fatalf("unexpected error: %#v", err)
	}
}

func findUserByID_ErrNotFound(t testing.TB, conn *postgres.Conn) {
	s := postgres.NewUserService(conn)
	if _, err := s.FindUserByID(context.Background(), 1); todev.ErrorCode(err) != todev.ENOTFOUND {
		t.Fatalf("unexpected error: %#v", err)
	}

}

func findUsers_OK(t testing.TB, conn *postgres.Conn) {
	s := postgres.NewUserService(conn)
	ctx := context.Background()
	MustCreateUser(t, ctx, conn, &todev.User{Name: "john", Email: "john@gmail.com"})
	MustCreateUser(t, ctx, conn, &todev.User{Name: "judy", Email: "judy@gmail.com"})
	MustCreateUser(t, ctx, conn, &todev.User{Name: "bob", Email: "bob@gmail.com"})
	MustCreateUser(t, ctx, conn, &todev.User{Name: "george", Email: "george@gmail.com"})

	email := "bob@gmail.com"
	if users, n, err := s.FindUsers(ctx, todev.UserFilter{Email: &email}); err != nil {
		t.Fatalf("error retrieving users: %v", err)
	} else if got, want := len(users), 1; got != want {
		t.Fatalf("len=%d, want %d", got, want)
	} else if got, want := users[0].Name, "bob"; got != want {
		t.Fatalf("name=%s, want %s", got, want)
	} else if got, want := n, 1; got != want {
		t.Fatalf("n=%d, want %d", got, want)
	}
}
