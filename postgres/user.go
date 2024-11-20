package postgres

import (
	"context"
	"crypto/rand"
	"database/sql"
	"encoding/hex"
	"fmt"
	"io"
	"log"
	"strings"

	_ "github.com/lib/pq"
	"github.com/saiddis/todev"
)

// UserService represents a service for managing users.
type UserService struct {
	conn *Conn
}

func NewUserService(conn *Conn) *UserService {
	return &UserService{conn: conn}
}

// FindUserByID retrieves user by ID along with associated auth objects.
// Returns ENOTFOUND if user does not exists.
func (s *UserService) FindUserByID(ctx context.Context, id int) (*todev.User, error) {
	var err error
	tx, err := s.conn.BeginTx(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("error begining trasaction: %v", err)
	}
	defer func() {
		if err != nil {
			log.Println("Rollback!")
			tx.Rollback()
		} else {
			tx.Commit()
		}
	}()

	user, err := findUserByID(ctx, tx, id)
	if err != nil {
		return nil, err
	} else if err := attachUserAuths(ctx, tx, user); err != nil {
		return user, err
	}

	return user, nil
}

// FindUsers retrieves a list of users by filter. Also returns total count of
// matching users which may differ from returned results if filter.Limit is specified.
func (s *UserService) FindUsers(ctx context.Context, filter todev.UserFilter) ([]*todev.User, int, error) {
	var err error
	tx, err := s.conn.BeginTx(ctx, nil)
	if err != nil {
		return nil, 0, fmt.Errorf("error begining trasaction: %v", err)
	}
	defer func() {
		if err != nil {
			log.Println("Rollback!")
			tx.Rollback()
		} else {
			tx.Commit()
		}
	}()

	return findUsers(ctx, tx, filter)
}

// CreateUser creates a new user. This is only used for testing since users are
// typically created during the OAuth creation process in AuthService.CreateUser().
func (s *UserService) CreateUser(ctx context.Context, user *todev.User) error {
	var err error
	tx, err := s.conn.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("error begining trasaction: %v", err)
	}
	defer func() {
		if err != nil {
			log.Println("Rollback!")
			tx.Rollback()
		} else {
			tx.Commit()
		}
	}()

	if err = createUser(tx, user); err != nil {
		return fmt.Errorf("error creating user: %v", err)
	} else if err := attachUserAuths(ctx, tx, user); err != nil {
		return fmt.Errorf("error attaching auth objects to user: %v", err)
	}
	if err != nil {
		return fmt.Errorf("error commiting transaction: %v", err)
	}
	return nil

}

// UpdateUser updates a user object. Returns EUNAUTHORIZED if current user is
// not the user that is being updated. Returns ENOTFOUND if user does not exist.
func (s *UserService) UpdateUser(ctx context.Context, id int, upd todev.UserUpdate) (*todev.User, error) {
	var err error
	tx, err := s.conn.BeginTx(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("error beginning trasaction: %w", err)
	}
	defer tx.Rollback()
	//defer func() {
	//	if err != nil {
	//		log.Println("Rollback!")
	//		tx.Rollback()
	//	} else {
	//		tx.Commit()
	//	}
	//}()

	user, err := updateUser(ctx, tx, id, upd)
	if err != nil {
		return nil, fmt.Errorf("error updating user: %w", err)
		// } else if err := attachUserAuths(ctx, tx, user); err != nil {
		// 	return user, fmt.Errorf("error attaching user auths: %w", err)
	} else if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("error commiting transaction: %w", err)
	}

	//err = tx.Commit()
	//if err != nil {
	//	return nil, fmt.Errorf("error commiting transaction: %w", err)
	//}
	return user, nil
}

// DeleteUser permanently deletes a user and all owned repos.
// Returns EUNAUTHORIZED if current user is not the user being deleted.
// Returns ENOTFOUND if user does not exist.
func (s *UserService) DeleteUser(ctx context.Context, id int) error {
	var err error
	tx, err := s.conn.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("error begining trasaction: %w", err)
	}
	defer func() {
		if err != nil {
			log.Println("Rollback!")
			tx.Rollback()
		} else {
			tx.Commit()
		}
	}()

	if err := deleteUser(ctx, tx, id); err != nil {
		return fmt.Errorf("error deleting user: %w", err)
	}

	return nil
}

// findUsers returns a list of users matching a filter. Also returns a count of
// total matching users which may differ if filter.Limit is set.
func findUsers(ctx context.Context, tx *Tx, filter todev.UserFilter) (_ []*todev.User, n int, err error) {
	// Build WHERE clause.
	var argIndex int
	where, args := []string{"1 = 1"}, []interface{}{}
	if v := filter.ID; v != nil {
		argIndex++
		where, args = append(where, fmt.Sprintf("id = $%d", argIndex)), append(args, *v)
	}
	if v := filter.Email; v != nil {
		argIndex++
		where, args = append(where, fmt.Sprintf("email = $%d", argIndex)), append(args, *v)
	}
	if v := filter.APIKey; v != nil {
		argIndex++
		where, args = append(where, fmt.Sprintf("api_key = $%d", argIndex)), append(args, *v)
	}

	// Execute query to fetch user rows.
	rows, err := tx.QueryContext(ctx, `
		SELECT
			id,
			name,
			email,
			api_key,
			created_at,
			updated_at,
			COUNT(*)
		FROM users
		WHERE `+strings.Join(where, " AND ")+`
		GROUP BY id
		ORDER BY id ASC
		`+FormatLimitOffset(filter.Limit, filter.Offset), args...)
	if err != nil {
		return nil, n, fmt.Errorf("error retrieving users: %v", err)
	}
	defer rows.Close()

	users := make([]*todev.User, 0)
	var email sql.NullString
	var user todev.User

	// Deserialize rows into User objects.
	for rows.Next() {
		if err := rows.Scan(
			&user.ID,
			&user.Name,
			&email,
			&user.APIKey,
			(*NullTime)(&user.CreatedAt),
			(*NullTime)(&user.UpdatedAt),
			&n,
		); err != nil {
			return nil, 0, fmt.Errorf("error deserializing rows: %v", err)
		}

		if email.Valid {
			user.Email = email.String
		}

		users = append(users, &user)
	}

	if err := rows.Err(); err != nil {
		return nil, 0, fmt.Errorf("error iterating over rows: %v", err)
	}

	return users, n, nil
}

// findUserByID is a helper funtion to fetch a user by ID.
// Returns ENOTFOUND if user does not exist.
func findUserByID(ctx context.Context, tx *Tx, id int) (*todev.User, error) {
	users, _, err := findUsers(ctx, tx, todev.UserFilter{ID: &id})
	if err != nil {
		return nil, fmt.Errorf("error retrieving users: %v", err)
	} else if len(users) == 0 {
		return nil, todev.Errorf(todev.ENOTFOUND, "User not found.")
	}

	return users[0], nil

}

// findUserByEmail is a helper function to fetch a user by email.
// Returns ENOTFOUND if user does not exist.
func findUserByEmail(ctx context.Context, tx *Tx, email string) (*todev.User, error) {
	users, _, err := findUsers(ctx, tx, todev.UserFilter{Email: &email})
	if err != nil {
		return nil, fmt.Errorf("error finding users: %w", err)
	} else if len(users) == 0 {
		return nil, todev.Errorf(todev.ENOTFOUND, "User not found.")
	}
	return users[0], nil
}

// createUser creates a new user. Sets the new database ID to user.ID and sets
// the timestamps to the current time.
func createUser(tx *Tx, user *todev.User) error {
	// Set timestamps to the current time.
	user.CreatedAt = tx.now
	user.UpdatedAt = user.CreatedAt

	// Perform basic field validation.
	if err := user.Validate(); err != nil {
		return fmt.Errorf("error validating user: %v", err)
	}

	// Email is nullable and has a UNIQUE constraint so ensure we store blank
	// fields as NULLs.
	var email *string
	if user.Email != "" {
		email = &user.Email
	}

	// Generate random API key.
	apiKey := make([]byte, 32)
	if _, err := io.ReadFull(rand.Reader, apiKey); err != nil {
		return fmt.Errorf("error creating api key: %v", err)
	}
	user.APIKey = hex.EncodeToString(apiKey)

	// Execute insertion query.
	var id int
	err := tx.QueryRow(`
		INSERT INTO users (
			name,
			email,
			api_key,
			created_at,
			updated_at
		)
		VALUES ($1, $2, $3, $4, $5)
		RETURNING id;`,
		user.Name,
		email,
		user.APIKey,
		(*NullTime)(&user.CreatedAt),
		(*NullTime)(&user.UpdatedAt),
	).Scan(&id)
	if err != nil {
		return fmt.Errorf("error scanning: %v", err)
	}

	user.ID = int(id)

	return nil
}

// updateUser updates fields on a user object. Returns EUNAUTHORIZED if current
// user is not the user being updated.
func updateUser(ctx context.Context, tx *Tx, id int, upd todev.UserUpdate) (*todev.User, error) {
	user, err := findUserByID(ctx, tx, id)
	if err != nil {
		return user, fmt.Errorf("error retrieving user by ID: %w", err)
	} else if user.ID != todev.UserIDFromContext(ctx) {
		return nil, todev.Errorf(todev.EUNAUTHORIZED, "You are not allowed to update this user.")
	}

	if v := upd.Name; v != nil {
		user.Name = *v
	}
	if v := upd.Email; v != nil {
		user.Email = *v
	}

	user.UpdatedAt = tx.now

	if err := user.Validate(); err != nil {
		return user, fmt.Errorf("error validating user: %w", err)
	}

	// Email is nullable and has a UNIQUE constraint so ensure we store blank
	// fields as NULLs.
	var email *string
	if user.Email != "" {
		email = &user.Email
	}

	if _, err := tx.ExecContext(ctx, `
		UPDATE users
		SET name = %1, email = $2, updated_at = $3
		WHERE id = $4`,
		user.Name,
		email,
		(*NullTime)(&user.UpdatedAt),
		id,
	); err != nil {
		return user, FormatError(err)
	}

	return user, nil
}

// deleteUser permanently removes user by ID. Returns EUNAUTHORIZED if current
// user is not the user being deleted.
func deleteUser(ctx context.Context, tx *Tx, id int) error {
	if user, err := findUserByID(ctx, tx, id); err != nil {
		return fmt.Errorf("error retrieving user by id: %w", err)
	} else if user.ID != todev.UserIDFromContext(ctx) {
		return todev.Errorf(todev.EUNAUTHORIZED, "You are not allowed to delete this user.")
	}

	if _, err := tx.ExecContext(ctx, "DELETE FROM users WHERE id = $1", id); err != nil {
		return fmt.Errorf("error deleting user: %w", err)
	}
	return nil
}

// attachUserAuths attaches OAuth objects associated with the user.
func attachUserAuths(ctx context.Context, tx *Tx, user *todev.User) (err error) {
	if user.Auths, _, err = findAuths(ctx, tx, todev.AuthFilter{UserID: &user.ID}); err != nil {
		return fmt.Errorf("attach user auths: %w", err)
	}
	return nil
}
