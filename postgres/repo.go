package postgres

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"io"
	"log"
	"slices"
	"strings"

	"github.com/saiddis/todev"
)

type RepoService struct {
	conn *Conn
}

func NewRepoService(conn *Conn) *RepoService {
	return &RepoService{
		conn: conn,
	}
}

// CreateRepo creates a new repo and assigns the current user as the owner of the
// repo. The owner will automatically be added to as a contributors of the repo.
func (s *RepoService) CreateRepo(ctx context.Context, repo *todev.Repo) error {
	tx, err := s.conn.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("error beginning transaction: %w", err)
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

	if err = createRepo(ctx, tx, repo); err != nil {
		return err
	} else if err = attachRepoAssociations(ctx, tx, repo); err != nil {
		return err
	}

	// Listening for incoming events.
	go func() {
		listenForEvents(repo)
	}()

	if err = createSelfContributor(ctx, tx, repo); err != nil {
		return fmt.Errorf("error creating self contributor: %w", err)
	}

	return nil
}

// FindRepoByID retunrs a single repo along with associted contributors.
// Only the repo owner and contributors can see a repo. Returns ENOTFOUND if
// repo does not exist or user does not have premission to view it.
func (s *RepoService) FindRepoByID(ctx context.Context, id int) (*todev.Repo, error) {
	tx, err := s.conn.BeginTx(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("error beginning transaction: %w", err)
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

	repo, err := findRepoByID(ctx, tx, id)
	if err != nil {
		return nil, err
	} else if err = attachRepoAssociations(ctx, tx, repo); err != nil {
		return nil, err
	}

	return repo, nil
}

// FindRepos returns a list of repos based on a filter. Only retruns
// repos that the user owns or is a member of.
func (s *RepoService) FindRepos(ctx context.Context, filter todev.RepoFilter) ([]*todev.Repo, int, error) {
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

	repos, n, err := findRepos(ctx, tx, filter)
	if err != nil {
		return repos, n, err
	}

	for _, repo := range repos {
		if err = attachRepoAssociations(ctx, tx, repo); err != nil {
			return repos, n, err
		}
	}
	return repos, n, nil
}

// UpdateRepo updates an existing repo by id ID. Only the repo owner can update the repo.
// Returns a new update state even if there was an error during the update.
//
// Retursn ENOTFOUND if repo does not exist. Returns EUNAUTHORIZED if user is not the repo owner.
func (s *RepoService) UpdateRepo(ctx context.Context, id int, upd todev.RepoUpdate) (*todev.Repo, error) {
	tx, err := s.conn.BeginTx(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("error beginning transaction: %w", err)
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

	repo, err := updateRepo(ctx, tx, id, upd)
	if err != nil {
		return repo, err
	} else if err = attachRepoAssociations(ctx, tx, repo); err != nil {
		return repo, err
	}

	return repo, nil
}

// DeleteRepo pemanently removes a repo by ID. Only the repo owner may
// delete a repo. Returns ENOTFOUND if the repo does not exist.
// Returns EUNAUTHORIZED if user is not the owner.
func (s *RepoService) DeleteRepo(ctx context.Context, id int) error {
	tx, err := s.conn.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("error beginning transaction: %w", err)
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

	if err = deleteRepo(ctx, tx, id); err != nil {
		return err
	}

	return nil
}

func createRepo(ctx context.Context, tx *Tx, repo *todev.Repo) (err error) {
	// Assign repo to the current user.
	userID := todev.UserIDFromContext(ctx)
	if userID == 0 {
		return todev.Errorf(todev.EUNAUTHORIZED, "You must be logged in to create a repo.")
	}
	repo.UserID = userID

	inviteCode := make([]byte, 16)
	if _, err = io.ReadFull(rand.Reader, inviteCode); err != nil {
		return fmt.Errorf("error generating invite code: %w", err)
	}
	repo.InviteCode = hex.EncodeToString(inviteCode)

	repo.CreatedAt = tx.now
	repo.UpdatedAt = repo.CreatedAt

	if err = repo.Validate(); err != nil {
		return err
	} else if _, err = findUserByID(ctx, tx, repo.UserID); err != nil {
		return fmt.Errorf("error retriving owner of the repo: %w", err)
	}

	err = tx.QueryRowContext(ctx, `
		INSERT INTO repos (
			user_id,
			name,
			invite_code,
			created_at,
			updated_at
		)
		VALUES ($1, $2, $3, $4, $5)
		RETURNING id;`,
		repo.UserID,
		repo.Name,
		repo.InviteCode,
		(*NullTime)(&repo.CreatedAt),
		(*NullTime)(&repo.UpdatedAt),
	).Scan(&repo.ID)
	if err != nil {
		return fmt.Errorf("error creating repo: %w", err)
	}
	return nil
}

func findRepos(ctx context.Context, tx *Tx, filter todev.RepoFilter) ([]*todev.Repo, int, error) {
	where, args := []string{"1 = 1"}, []interface{}{}
	var argIndex int
	if v := filter.ID; v != nil {
		argIndex++
		where, args = append(where, fmt.Sprintf("id = $%d", argIndex)), append(args, *v)
	}
	if v := filter.InviteCode; v != nil {
		argIndex++
		where, args = append(where, fmt.Sprintf("invite_code = $%d", argIndex)), append(args, *v)
	} else {
		argIndex++
		userID := todev.UserIDFromContext(ctx)
		where = append(where, fmt.Sprintf(`(
			id IN (SELECT repo_id FROM contributors c WHERE c.user_id = $%d)
			)`, argIndex))
		args = append(args, userID)
	}

	stmt, err := tx.PrepareContext(ctx, `
		SELECT
			id,
			user_id,
			name,
			invite_code,
			created_at,
			updated_at,
			COUNT(*) OVER()
		FROM repos
		WHERE `+strings.Join(where, " AND ")+`
		GROUP BY id
		ORDER BY id ASC
		`+FormatLimitOffset(filter.Limit, filter.Offset)+`;`,
	)
	if err != nil {
		return nil, 0, fmt.Errorf("error preparing query: %w", err)
	}

	rows, err := stmt.QueryContext(ctx, args...)
	if err != nil {
		return nil, 0, fmt.Errorf("error retrieving repos: %w", err)
	}
	defer func() {
		err := rows.Close()
		if err != nil {
			log.Printf("error closing rows: %v", err)
		}
	}()

	var n int
	repos := make([]*todev.Repo, 0)
	for rows.Next() {
		var repo todev.Repo
		if err = rows.Scan(
			&repo.ID,
			&repo.UserID,
			&repo.Name,
			&repo.InviteCode,
			(*NullTime)(&repo.CreatedAt),
			(*NullTime)(&repo.UpdatedAt),
			&n,
		); err != nil {
			return nil, 0, fmt.Errorf("error scanning: %w", err)
		}

		repos = append(repos, &repo)
	}

	if rows.Err() != nil {
		return nil, 0, fmt.Errorf("error iterating over rows: %w", err)
	}

	return repos, n, nil
}

func findRepoByID(ctx context.Context, tx *Tx, id int) (*todev.Repo, error) {
	repos, _, err := findRepos(ctx, tx, todev.RepoFilter{ID: &id})
	if err != nil {
		return nil, fmt.Errorf("error retrieving repo by ID: %w", err)
	} else if len(repos) == 0 {
		return nil, todev.Errorf(todev.ENOTFOUND, "Repo not found.")
	}
	return repos[0], nil
}

func updateRepo(ctx context.Context, tx *Tx, id int, upd todev.RepoUpdate) (*todev.Repo, error) {
	repo, err := findRepoByID(ctx, tx, id)
	if err != nil {
		return repo, fmt.Errorf("error finding repo to update: %w", err)
	} else if !todev.CanEditRepo(ctx, *repo) {
		return nil, todev.Errorf(todev.EUNAUTHORIZED, "You are not allowed to update this repo.")
	}

	if v := upd.Name; v != nil {
		repo.Name = *v
	}

	repo.UpdatedAt = tx.now

	if err = repo.Validate(); err != nil {
		return repo, fmt.Errorf("error validating repo: %w", err)
	}

	_, err = tx.ExecContext(ctx, `
		UPDATE repos
		SET name = $1, updated_at = $2
		WHERE id = $3;`,
		repo.Name,
		(*NullTime)(&repo.UpdatedAt),
		id,
	)
	if err != nil {
		return repo, fmt.Errorf("error updating repo: %w", err)
	}
	return repo, nil
}

func deleteRepo(ctx context.Context, tx *Tx, id int) (err error) {
	repo, err := findRepoByID(ctx, tx, id)
	if err != nil {
		return fmt.Errorf("error retrieving user by id: %w", err)
	} else if !todev.CanEditRepo(ctx, *repo) {
		return todev.Errorf(todev.EUNAUTHORIZED, "Only the owner can delete a repo.")
	} else if sub, ok := tx.conn.EventService.GetSubscribtion(repo.ID); ok {
		repo.Subscription = sub
	} else {
		return todev.Errorf(todev.ENOTFOUND, "Repo is not subscribed for task event service.")
	}

	if _, err = tx.ExecContext(ctx, "DELETE FROM repos WHERE id = $1;", id); err != nil {
		return fmt.Errorf("error deleting repo: %w", err)
	}

	repo.Subscription.Done()

	return nil
}

// checkRepoExists returns nil if a repo does not exist. Otherwise returns ENOTFOUND.
func checkRepoExists(ctx context.Context, tx *Tx, id int) error {
	var n int
	if err := tx.QueryRowContext(ctx, "SELECT COUNT(1) FROM repos WHERE id = $1", id).Scan(&n); err != nil {
		return fmt.Errorf("error retrieving repo by id: %w", err)
	} else if n == 0 {
		return todev.Errorf(todev.ENOTFOUND, "Repo not found.")
	}
	return nil
}

// listenForEvents is a helper funtion that listens for incoming events
// for the given repo.
func listenForEvents(repo *todev.Repo) {
	for {
		select {
		case event := <-repo.Subscription.C():
			switch payload := event.Payload.(type) {
			case todev.TaskAdded:
				repo.Tasks = append(repo.Tasks, payload.Task)
			case todev.TaskCompletionToggled:
				for _, task := range repo.Tasks {
					if task.ID == payload.ID {
						if task.IsCompleted {
							task.IsCompleted = false
						} else {
							task.IsCompleted = true
						}
						break
					}
				}
			case todev.TaskDescriptionChanged:
				for _, task := range repo.Tasks {
					if task.ID == payload.ID {
						task.Description = payload.Value
						break
					}
				}
			case todev.TaskContributorIDChanged:
				for _, task := range repo.Tasks {
					if task.ID == payload.ID {
						task.ContributorID = &payload.Value
						break
					}
				}
			case todev.TaskDeleted:
				for i, task := range repo.Tasks {
					if task.ID == payload.ID {
						repo.Tasks = slices.Concat(repo.Tasks[:i], repo.Tasks[i+1:])
						break
					}
				}
			case todev.ContributorAdded:
				repo.Contributors = append(repo.Contributors, payload.Contributor)
			case todev.ContributorSetAdmin:
				for _, contributor := range repo.Contributors {
					if contributor.ID == payload.ID {
						contributor.IsAdmin = true
						break
					}
				}
			case todev.ContributorResetAdmin:
				for _, contributor := range repo.Contributors {
					if contributor.ID == payload.ID {
						contributor.IsAdmin = false
						break
					}
				}
			case todev.ContributorDeleted:
				for i, contributor := range repo.Contributors {
					if contributor.ID == payload.ID {
						repo.Contributors = slices.Concat(repo.Contributors[:i], repo.Contributors[i+1:])
						break
					}
				}
			}

		case <-repo.Subscription.Done():
			repo.Subscription.Close()
			return
		}
	}
}

// attachRepoAssociations is a helper function to look up and attach the owner of the repo
// along with associated subscribtion.
func attachRepoAssociations(ctx context.Context, tx *Tx, repo *todev.Repo) (err error) {
	user, err := findUserByID(ctx, tx, repo.UserID)
	if err != nil {
		return fmt.Errorf("error attaching repo user: %w", err)
	}
	repo.UserID = user.ID

	if repo.Contributors, _, err = findContributors(ctx, tx, todev.ContributorFilter{RepoID: &repo.ID}); err != nil {
		return fmt.Errorf("error attaching repo contributors: %w", err)
	} else if repo.Tasks, _, err = findTasks(ctx, tx, todev.TaskFilter{RepoID: &repo.ID}); err != nil {
		return fmt.Errorf("error attaching repo tasks: %w", err)
	}

	if sub, ok := tx.conn.EventService.GetSubscribtion(repo.ID); ok {
		repo.Subscription = sub
		return nil
	}

	ctx = context.WithValue(ctx, "repoID", repo.ID)
	sub, err := tx.conn.EventService.Subscribe(ctx)
	if err != nil {
		return fmt.Errorf("error subscribing for event service: %w", err)
	}
	repo.Subscription = sub
	return nil
}
