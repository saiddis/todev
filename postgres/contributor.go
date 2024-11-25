package postgres

import (
	"context"
	"fmt"
	"log"
	"strings"

	"github.com/saiddis/todev"
)

type ContributorService struct {
	conn *Conn
}

func NewContrubutorService(conn *Conn) *ContributorService {
	return &ContributorService{conn: conn}
}

// CreateContributor creates a new contributor on a repo for the current user.
// Returns EUNAUTHORIZED if there is no current user logged in.
func (s *ContributorService) CreateContributor(ctx context.Context, contributor *todev.Contributor) error {
	tx, err := s.conn.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("error beginning transaction: %w", err)
	}
	defer func() {
		if err != nil {
			err = fmt.Errorf("CreateContributor: %w", err)
			if err := tx.Rollback(); err != nil {
				log.Printf("failed to rollback: %v", err)
			}
		} else {
			if err := tx.Commit(); err != nil {
				log.Printf("failed to commit: %v", err)
			}
		}
	}()

	userID := todev.UserIDFromContext(ctx)
	if userID == 0 {
		return todev.Errorf(todev.EUNAUTHORIZED, "You must be logged in to join a repo.")
	}
	contributor.UserID = userID

	if err = createContributor(ctx, tx, contributor); err != nil {
		return err
	} else if err = attachContributorAssociations(ctx, tx, contributor); err != nil {
		return err
	}
	return nil
}

// FindContributors retrieves a list of matching contributors based on filter.
// Only returns contributors that belong to repos that the current user is a member of.
func (s *ContributorService) FindContributors(ctx context.Context, filter todev.ContributorFilter) ([]*todev.Contributor, int, error) {
	tx, err := s.conn.BeginTx(ctx, nil)
	if err != nil {
		return nil, 0, fmt.Errorf("error beginning transaction: %w", err)
	}
	defer func() {
		if err != nil {
			err = fmt.Errorf("CreateContributor: %w", err)
			if err := tx.Rollback(); err != nil {
				log.Printf("failed to rollback: %v", err)
			}
		} else {
			if err := tx.Commit(); err != nil {
				log.Printf("failed to commit: %v", err)
			}
		}
	}()

	contributors, n, err := findContributors(ctx, tx, filter)
	if err != nil {
		return nil, 0, err
	}

	for _, contributor := range contributors {
		if err = attachContributorAssociations(ctx, tx, contributor); err != nil {
			return contributors, n, err
		}
	}

	return contributors, n, nil
}

// FindContributorsByID retrieves a contributor by ID along with associated repo and user. Returns ENOTFOUND
// if contributor does not exist or user does not have permission to view it.
func (s *ContributorService) FindContributorByID(ctx context.Context, id int) (*todev.Contributor, error) {
	tx, err := s.conn.BeginTx(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("error beginning transaction: %w", err)
	}
	defer func() {
		if err != nil {
			err = fmt.Errorf("CreateContributor: %w", err)
			if err = tx.Rollback(); err != nil {
				log.Printf("failed to rollback: %v", err)
			}
		} else {
			if err = tx.Commit(); err != nil {
				log.Printf("failed to commit: %v", err)
			}
		}
	}()

	contributor, err := findContributorByID(ctx, tx, id)
	if err != nil {
		return nil, err
	} else if err = attachContributorAssociations(ctx, tx, contributor); err != nil {
		return nil, err
	}

	return contributor, nil
}

// TODO: Devise logic for updating it through TaskService.
func (s *ContributorService) UpdateContributor(ctx context.Context, id int, upd todev.ContributorUpdate) (*todev.Contributor, error) {
	tx, err := s.conn.BeginTx(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("error beginning transaction: %w", err)
	}
	defer func() {
		if err != nil {
			err = fmt.Errorf("CreateContributor: %w", err)
			if err := tx.Rollback(); err != nil {
				log.Printf("failed to rollback: %v", err)
			}
		} else {
			if err := tx.Commit(); err != nil {
				log.Printf("failed to commit: %v", err)
			}
		}
	}()

	contributor, err := updateContributor(ctx, tx, id, upd)
	if err != nil {
		return nil, err
	} else if attachContributorAssociations(ctx, tx, contributor); err != nil {
		return nil, err
	}
	return contributor, nil
}

// DeleteContritbutor permanently removes contributor by ID. Only the repo owner
// and contributor's associated user can delete a contributor.
func (s *ContributorService) DeleteContritbutor(ctx context.Context, id int) error {
	tx, err := s.conn.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("error beginning transaction: %w", err)
	}
	defer func() {
		if err != nil {
			err = fmt.Errorf("CreateContributor: %w", err)
			if err := tx.Rollback(); err != nil {
				log.Printf("failed to rollback: %v", err)
			}
		} else {
			if err := tx.Commit(); err != nil {
				log.Printf("failed to commit: %v", err)
			}
		}
	}()

	if err := deleteContirbutor(ctx, tx, id); err != nil {
		return err
	}
	return nil
}

func createContributor(ctx context.Context, tx *Tx, contributor *todev.Contributor) (err error) {
	contributor.CreatedAt = tx.now
	contributor.UpdatedAt = contributor.CreatedAt

	if err = contributor.Validate(); err != nil {
		return err
	}

	if err = checkRepoExists(ctx, tx, contributor.RepoID); err != nil {
		return err
	} else if _, err = findUserByID(ctx, tx, contributor.UserID); err != nil {
		return err
	}

	err = tx.QueryRowContext(ctx, `
		INSERT INTO contributors (
			repo_id,
			user_id,
			created_at,
			updated_at
		)
		VALUES ($1,$2,$3,$4)
		RETURNING id;`,
		contributor.RepoID,
		contributor.UserID,
		(*NullTime)(&contributor.CreatedAt),
		(*NullTime)(&contributor.UpdatedAt),
	).Scan(&contributor.ID)
	if err != nil {
		return fmt.Errorf("error inserting contributor: %w", err)
	}
	return nil
}

func findContributors(ctx context.Context, tx *Tx, filter todev.ContributorFilter) ([]*todev.Contributor, int, error) {
	where, args := []string{"1 = 1"}, []interface{}{}
	var argIndex int
	if v := filter.ID; v != nil {
		argIndex++
		where, args = append(where, fmt.Sprintf("c.id = $%d", argIndex)), append(args, *v)
	}
	if v := filter.RepoID; v != nil {
		argIndex++
		where, args = append(where, fmt.Sprintf("c.repo_id = $%d", argIndex)), append(args, *v)
	}
	if v := filter.UserID; v != nil {
		argIndex++
		where, args = append(where, fmt.Sprintf("c.user_id = $%d", argIndex)), append(args, *v)
	}

	userID := todev.UserIDFromContext(ctx)
	argIndex++
	where = append(where, fmt.Sprintf(`(
		r.user_id = $%d OR
		c.repo_id IN (SELECT c1.repo_id FROM contributors c1 WHERE c1.user_id = $%d)
		)`, argIndex, argIndex+1),
	)
	argIndex++

	var sortBy string
	switch filter.SortBy {
	case todev.ContributorSortByUpdatedAtDesc:
		sortBy = "c.updated_at DESC"
	default:
		argIndex++
		sortBy = fmt.Sprintf(`CASE c.user_id WHEN $%d THEN 0 ELSE 1 END ASC, u.name ASC`, argIndex)
		args = append(args, userID)
	}
	args = append(args, userID, userID)

	stmt, err := tx.PrepareContext(ctx, `
		SELECT
			c.id,
			c.repo_id,
			c.user_id,
			c.created_at,
			c.updated_at,
			r.user_id AS repo_user_id,
			COUNT(*) OVER()
		FROM contributors c
		JOIN repos r ON c.repo_id = r.id
		JOIN users u ON c.user_id = u.id
		WHERE `+strings.Join(where, " AND ")+`
		GROUP BY c.id, r.user_id, u.name
		ORDER BY `+sortBy+`
		`+FormatLimitOffset(filter.Limit, filter.Offset)+`;`,
	)
	if err != nil {
		return nil, 0, fmt.Errorf("error preparing query: %w", err)
	}

	rows, err := stmt.QueryContext(ctx, args...)
	if err != nil {
		return nil, 0, fmt.Errorf("error retrieving contributors: %w", err)
	}
	defer func() {
		err := rows.Close()
		if err != nil {
			log.Printf("error closing rows: %v", err)
		}
	}()

	contributors := make([]*todev.Contributor, 0)
	var n int
	for rows.Next() {
		var repoUserID int
		var contributor todev.Contributor
		if err = rows.Scan(
			&contributor.ID,
			&contributor.RepoID,
			&contributor.UserID,
			(*NullTime)(&contributor.CreatedAt),
			(*NullTime)(&contributor.UpdatedAt),
			&repoUserID,
			&n,
		); err != nil {
			return nil, 0, fmt.Errorf("error scanning: %w", err)
		}
		contributors = append(contributors, &contributor)
	}

	if err = rows.Err(); err != nil {
		return nil, 0, fmt.Errorf("error iterating over rows: %w", err)
	}

	return contributors, n, nil
}

func findContributorByID(ctx context.Context, tx *Tx, id int) (*todev.Contributor, error) {
	contributors, _, err := findContributors(ctx, tx, todev.ContributorFilter{ID: &id})
	if err != nil {
		return nil, fmt.Errorf("error retrieving contributors by ID: %w", err)
	} else if len(contributors) == 0 {
		return nil, todev.Errorf(todev.ENOTFOUND, "Contributor not found.")
	}
	return contributors[0], nil
}

func updateContributor(ctx context.Context, tx *Tx, id int, upd todev.ContributorUpdate) (*todev.Contributor, error) {
	contributor, err := findContributorByID(ctx, tx, id)
	if err != nil {
		return nil, fmt.Errorf("error updating contributor: %w", err)
	} else if contributor.UserID != todev.UserIDFromContext(ctx) {
		return contributor, todev.Errorf(todev.EUNAUTHORIZED, "You don't have permission to update the contributor.")
	}

	if v := upd.Tasks; v != nil {
		contributor.Tasks = v
	}

	contributor.UpdatedAt = tx.now

	if err = contributor.Validate(); err != nil {
		return nil, err
	}

	_, err = tx.ExecContext(ctx, `
		UPDATE contribotors
		SET updated_at = $1
		WHERE id = $2;`,
		(*NullTime)(&contributor.UpdatedAt),
		id,
	)
	if err != nil {
		return contributor, fmt.Errorf("error updating contributor: %w", err)
	}

	if err = publishRepoEvent(ctx, tx, contributor.RepoID, todev.Event{
		Type: todev.EventTypeContributorTaskAdded,
		Payload: &todev.ContributorTaskAdded{
			ID:   id,
			Task: contributor.Tasks[len(contributor.Tasks)-1],
		},
	}); err != nil {
		return nil, fmt.Errorf("publish repo event: %w", err)
	}

	return contributor, nil
}

func deleteContirbutor(ctx context.Context, tx *Tx, id int) error {
	userID := todev.UserIDFromContext(ctx)

	contributor, err := findContributorByID(ctx, tx, id)
	if err != nil {
		return err
	} else if err = attachContributorAssociations(ctx, tx, contributor); err != nil {
		return err
	}

	// Verify user is the contributor or ownes parent repo.
	if contributor.UserID != userID && contributor.Repo.UserID != userID {
		return todev.Errorf(todev.EUNAUTHORIZED, "You do not have permission to delete the contributor.")
	}

	// Do not let repo owner delete their own contributor object.
	if contributor.UserID == contributor.Repo.UserID {
		return todev.Errorf(todev.ECONFLICT, "Repo owner may not delete itself from contributors.")
	}

	if _, err = tx.ExecContext(ctx, "DELETE FROM contributors WHERE id = $1", id); err != nil {
		return fmt.Errorf("error deleting contributor: %w", err)
	}

	return nil
}

func attachContributorAssociations(ctx context.Context, tx *Tx, contributor *todev.Contributor) (err error) {
	if contributor.Repo, err = findRepoByID(ctx, tx, contributor.RepoID); err != nil {
		return fmt.Errorf("error attaching contributor repo: %w", err)
	} else if contributor.User, err = findUserByID(ctx, tx, contributor.UserID); err != nil {
		return fmt.Errorf("error attaching contributor user: %w", err)
	}
	return nil
}
