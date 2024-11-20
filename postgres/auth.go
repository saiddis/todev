package postgres

import (
	"context"
	"database/sql"
	"fmt"
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

// findAuths returns a list of auth objects that match a filter. Also returns
// a total count of matches which may differ from results if filter.Limit is set.
func findAuths(ctx context.Context, tx *Tx, filter todev.AuthFilter) (_ []*todev.Auth, n int, err error) {
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
		where, args = append(where, fmt.Sprintf("source = $%d", argIndex)), append(args, *v)
	}
	if v := filter.SourceID; v != nil {
		argIndex++
		where, args = append(where, fmt.Sprintf("source_id = $%d", argIndex)), append(args, *v)
	}

	// Execute the query with WHERE clause and LIMIT/OFFSET injected.
	rows, err := tx.QueryContext(ctx, `
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
		    COUNT(*)
		FROM auths
		WHERE `+strings.Join(where, " AND ")+`
		ORDER BY id ASC
		`+FormatLimitOffset(filter.Limit, filter.Offset)+`
	`,
		args...,
	)
	if err != nil {
		return nil, n, FormatError(err)
	}
	defer rows.Close()

	// Iterate over result set and deserialize rows into Auth objects.
	auths := make([]*todev.Auth, 0)
	for rows.Next() {
		var auth todev.Auth
		var expiry sql.NullString
		if err := rows.Scan(
			&auth.ID,
			&auth.UserID,
			&auth.Source,
			&auth.SourceID,
			&auth.AccessToken,
			&auth.RefreshToken,
			&expiry,
			(*NullTime)(&auth.CreatedAt),
			(*NullTime)(&auth.UpdatedAt),
			&n,
		); err != nil {
			return nil, 0, err
		}

		if expiry.Valid {
			if v, _ := time.Parse(time.RFC3339, expiry.String); !v.IsZero() {
				auth.Expiry = &v
			}
		}

		auths = append(auths, &auth)
	}
	if err := rows.Err(); err != nil {
		return nil, 0, FormatError(err)
	}

	return auths, n, nil
}
