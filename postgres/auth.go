package postgres

import (
	"context"
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/saiddis/todev"
)

type AuthService struct {
	conn *Conn
}

func NewAuthService(conn *Conn) *AuthService {
	return &AuthService{conn: conn}
}

// FindAuthByID retrieves an authentication object by ID along with associated user.
// Returns ENOTFOUND if the user is not exist.
func (s *AuthService) FindAuthByID(ctx context.Context, id int) (*todev.Auth, error) {
	tx, err := s.conn.BeginTx(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("error beginning transaction: %w", err)
	}

	defer func() {
		if err != nil {
			err = fmt.Errorf("FindAuthByID: %w", err)
			// Shadowing err variable, so as to only log rollback errors.
			if err := tx.Rollback(); err != nil {
				log.Printf("failed to rollback: %v", err)
			}
		} else {
			if err := tx.Commit(); err != nil {
				log.Printf("failed to commit: %v", err)
			}
		}
	}()

	auth, err := findAuthByID(ctx, tx, id)
	if err != nil {
		return nil, err
	} else if err = attachAuthAssociations(ctx, tx, auth); err != nil {
		return nil, err
	}

	return auth, nil

}

// FindAuths retrieves authentication objects based on filter.
func (s *AuthService) FindAuths(ctx context.Context, filter todev.AuthFilter) ([]*todev.Auth, int, error) {
	tx, err := s.conn.BeginTx(ctx, nil)
	if err != nil {
		return nil, 0, fmt.Errorf("error beginning transaction: %w", err)
	}

	defer func() {
		if err != nil {
			err = fmt.Errorf("FindAuths: %w", err)
			if err := tx.Rollback(); err != nil {
				log.Printf("failed to rollback: %v", err)
			}
		} else {
			if err := tx.Commit(); err != nil {
				log.Printf("failed to commit: %v", err)
			}
		}
	}()

	auths, n, err := findAuths(ctx, tx, filter)
	if err != nil {
		return nil, 0, fmt.Errorf("error beginning transaction: %w", err)
	}

	for _, auth := range auths {
		if err = attachAuthAssociations(ctx, tx, auth); err != nil {
			return nil, 0, err
		}
	}

	return auths, n, nil
}

// CreateAuth creates a new authentication object if a user is attached to auth,
// then the auth object is linked to an existing user. Otherwise a new user object created.
func (s *AuthService) CreateAuth(ctx context.Context, auth *todev.Auth) error {
	tx, err := s.conn.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("error beginning transaction: %w", err)
	}

	defer func() {
		if err != nil {
			err = fmt.Errorf("CreateAuth: %w", err)
			if err := tx.Rollback(); err != nil {
				log.Printf("failed to rollback: %v", err)
			}
		} else {
			if err := tx.Commit(); err != nil {
				log.Printf("failed to commit: %v", err)
			}
		}
	}()

	// Check if the auth already exists for the given source.
	other, err := findAuthBySourceID(ctx, tx, auth.Source, auth.SourceID)
	if err == nil {
		// If an auth already exist for the source user, update with the new token.
		if other, err = updateAuth(ctx, tx, other.ID, auth.AccessToken, auth.RefreshToken, auth.Expiry); err != nil {
			return err
		} else if err = attachAuthAssociations(ctx, tx, other); err != nil {
			return err
		}

		*auth = *other
		return nil
	} else if todev.ErrorCode(err) != todev.ENOTFOUND {
		return fmt.Errorf("cannot find auth by source user: %w", err)
	}

	// Check if auth has new auth object passed in. It is considered "new" if
	// the caller doesn't know the database ID for the user.
	if auth.UserID == 0 && auth.User != nil {
		// Look up the user by email. If no user can be found then create a new
		// user with the auth.User object passed in.
		if user, err := findUserByEmail(ctx, tx, auth.User.Email); err == nil {
			auth.User = user
		} else if todev.ErrorCode(err) == todev.ENOTFOUND {
			if err = createUser(ctx, tx, auth.User); err != nil {
				return fmt.Errorf("cannot create user: %w", err)
			}
		} else {
			return fmt.Errorf("cannot find user by email: %w", err)
		}

		auth.UserID = auth.User.ID
	}

	// Create new auth object and attach associated auth object.
	if err = createAuth(ctx, tx, auth); err != nil {
		return fmt.Errorf("CreateAuth: %w", err)
	} else if err = attachAuthAssociations(ctx, tx, auth); err != nil {
		return fmt.Errorf("CreateAuth: %w", err)
	}

	return nil
}

// DeleteAuth permanently removes an authentication object from the system by ID.
// The parent user object is not removed.
func (s *AuthService) DeleteAuth(ctx context.Context, id int) error {
	tx, err := s.conn.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("error beginning transaction: %w", err)
	}

	defer func() {
		if err != nil {
			err = fmt.Errorf("DeleteAuth: %w", err)
			if err := tx.Rollback(); err != nil {
				log.Printf("failed to rollback: %v", err)
			}
		} else {
			if err := tx.Commit(); err != nil {
				log.Printf("failed to commit: %v", err)
			}
		}
	}()

	if err = deleteAuth(ctx, tx, id); err != nil {
		return err
	}

	return nil
}

// findAuthBySourceID is a helper function to return an auth object by source ID.
// Returns ENOTFOUND if auth doesn't exist.
func findAuthBySourceID(ctx context.Context, tx *Tx, source, sourceID string) (*todev.Auth, error) {
	auths, _, err := findAuths(ctx, tx, todev.AuthFilter{Source: &source, SourceID: &sourceID})
	if err != nil {
		return nil, fmt.Errorf("error retrieving auths: %w", err)
	} else if len(auths) == 0 {
		return nil, todev.Errorf(todev.ENOTFOUND, "Auth not found.")
	}

	return auths[0], nil
}

// createAuth creates a new auth object in the database. On success, the
// ID is set to the new database ID and timestamp fields are set to the current time.
func createAuth(ctx context.Context, tx *Tx, auth *todev.Auth) (err error) {
	auth.CreatedAt = tx.now
	auth.UpdatedAt = auth.CreatedAt

	if err = auth.Validate(); err != nil {
		return err
	}

	var id int
	err = tx.QueryRowContext(ctx, `
		INSERT INTO auths (
			user_id,
			source,
			source_id,
			access_token,
			refresh_token,
			expiry,
			created_at,
			updated_at
		)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
		RETURNING id;`,
		auth.UserID,
		auth.Source,
		auth.SourceID,
		auth.AccessToken,
		auth.RefreshToken,
		(*NullTime)(&auth.Expiry),
		(*NullTime)(&auth.CreatedAt),
		(*NullTime)(&auth.UpdatedAt),
	).Scan(&id)
	if err != nil {
		return FormatError(err)
	}

	auth.ID = id

	return nil
}

// findAuthByID is a helper function to return an auth object by ID.
// Returns ENOTFOUND if auth doesn't exist.
func findAuthByID(ctx context.Context, tx *Tx, id int) (*todev.Auth, error) {
	auths, _, err := findAuths(ctx, tx, todev.AuthFilter{ID: &id})
	if err != nil {
		return nil, fmt.Errorf("error retrieving auths: %w", err)
	} else if len(auths) == 0 {
		return nil, todev.Errorf(todev.ENOTFOUND, "Auth not found.")
	}

	return auths[0], nil
}

// findAuths returns a list of auth objects that match a filter. Also returns
// a total count of matches which may differ from results if filter.Limit is set.
func findAuths(ctx context.Context, tx *Tx, filter todev.AuthFilter) ([]*todev.Auth, int, error) {
	// Build WHERE clause. Each part of the clause is AND-ed together to further
	// restrict the results. Placeholders are added to "args" and are used
	// to avoid SQL injection.
	//
	// Each filter field is optional.
	where, args := []string{"1 = 1"}, []interface{}{}
	var argIndex int
	if v := filter.ID; v != nil {
		argIndex++
		where, args = append(where, fmt.Sprintf("id = $%d", argIndex)), append(args, *v)
	}
	if v := filter.UserID; v != nil {
		argIndex++
		where, args = append(where, fmt.Sprintf("user_id = $%d", argIndex)), append(args, *v)
	}
	if v := filter.Source; v != nil {
		argIndex++
		where, args = append(where, fmt.Sprintf("source = $%v", argIndex)), append(args, *v)
	}
	if v := filter.SourceID; v != nil {
		argIndex++
		where, args = append(where, fmt.Sprintf("source_id = $%v", argIndex)), append(args, *v)
	}

	stmt, err := tx.PrepareContext(ctx, `
		SELECT 
			id,
			user_id,
			source,
			source_id,
			access_token,
			refresh_token,
			expiry,
			created_at,
			updated_at,
			COUNT(*) OVER()
		FROM auths
		WHERE `+strings.Join(where, " AND ")+`
		GROUP BY id
		ORDER BY id ASC
		`+FormatLimitOffset(filter.Limit, filter.Offset)+`;`,
	)
	if err != nil {
		return nil, 0, fmt.Errorf("error preparing query: %w", err)
	}
	// Execute the query with WHERE clause and LIMIT/OFFSET injected.
	rows, err := stmt.QueryContext(ctx, args...)
	if err != nil {
		return nil, 0, FormatError(err)
	}
	defer func() {
		err := rows.Close()
		if err != nil {
			log.Printf("error closing rows: %v", err)
		}
	}()

	// Iterate over result set and deserialize rows into Auth objects.
	var n int
	auths := make([]*todev.Auth, 0)
	for rows.Next() {
		var auth todev.Auth
		if err = rows.Scan(
			&auth.ID,
			&auth.UserID,
			&auth.Source,
			&auth.SourceID,
			&auth.AccessToken,
			&auth.RefreshToken,
			(*NullTime)(&auth.Expiry),
			(*NullTime)(&auth.CreatedAt),
			(*NullTime)(&auth.UpdatedAt),
			&n,
		); err != nil {
			return nil, 0, fmt.Errorf("error scanning: %w", err)
		}

		auths = append(auths, &auth)
	}
	if err = rows.Err(); err != nil {
		return nil, 0, FormatError(err)
	}

	return auths, n, nil
}

// updateAuth updates tokens and expiry on existing auth object.
// Returns new state of the auth object.
func updateAuth(ctx context.Context, tx *Tx, id int, accessToken, refreshToken string, expiry time.Time) (*todev.Auth, error) {
	auth, err := findAuthByID(ctx, tx, id)
	if err != nil {
		return nil, fmt.Errorf("error retrieving auth by ID: %w", err)
	}

	auth.AccessToken = accessToken
	auth.RefreshToken = refreshToken
	auth.Expiry = expiry
	auth.UpdatedAt = tx.now

	if err = auth.Validate(); err != nil {
		return auth, err
	}

	_, err = tx.ExecContext(ctx, `
		UPDATE auths
		SET
			access_token = $1,
			refresh_token = $2,
			expiry = $3,
			updated_at = $4
		WHERE id = $5;`,
		auth.AccessToken,
		auth.RefreshToken,
		(*NullTime)(&auth.Expiry),
		id,
	)
	if err != nil {
		return auth, FormatError(err)
	}

	return auth, nil
}

// deleteAuth permanently removes auth object by ID.
func deleteAuth(ctx context.Context, tx *Tx, id int) (err error) {
	if auth, err := findAuthByID(ctx, tx, id); err != nil {
		return fmt.Errorf("error retrieving error by ID: %w", err)
	} else if auth.ID != todev.UserIDFromContext(ctx) {
		return todev.Errorf(todev.EUNAUTHORIZED, "You are not allowed to delete this auth.")
	}

	_, err = tx.ExecContext(ctx, "DELETE FROM auths WHERE id = $1;", id)
	if err != nil {
		return FormatError(err)
	}
	return nil
}

// attachAuthAssociations is helper function to fetch and attach the associated user to the auth object
func attachAuthAssociations(ctx context.Context, tx *Tx, auth *todev.Auth) (err error) {
	if auth.User, err = findUserByID(ctx, tx, auth.UserID); err != nil {
		return fmt.Errorf("attach auth user: %w", err)
	}
	return nil
}
