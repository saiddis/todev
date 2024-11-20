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
		WithSchema(t, func(t testing.TB, db *postgres.Conn) {
			s := postgres.NewUserService(db)

			row := db.DB.QueryRow("SHOW search_path;")
			var currPath string
			if err := row.Scan(&currPath); err != nil {
				t.Fatalf("Failed to fetch search_path: %v", err)
			}

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

		})
	})

	t.Run("ErrNameRequired", func(t *testing.T) {
		WithSchema(t, func(t testing.TB, db *postgres.Conn) {
			s := postgres.NewUserService(db)
			if err := s.CreateUser(context.Background(), &todev.User{}); err == nil {
				t.Fatal("error expected")
			}
		})
	})
}

func TestUserService_UpdateUser(t *testing.T) {
	t.Run("OK", func(t *testing.T) {
		WithSchema(t, func(t testing.TB, db *postgres.Conn) {
			s := postgres.NewUserService(db)
			user0, ctx0 := MustCreateUser(t, context.Background(), db, &todev.User{
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
		})
	})
}

// MustCreateUser creates a user in the database. Fatal on error.
func MustCreateUser(tb testing.TB, ctx context.Context, db *postgres.Conn, user *todev.User) (*todev.User, context.Context) {
	tb.Helper()

	if err := postgres.NewUserService(db).CreateUser(ctx, user); err != nil {
		tb.Fatalf("MustCreateUser: %v", err)
	}
	return user, todev.NewContextWithUser(ctx, user)
}
