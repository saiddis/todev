package postgres

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"strings"

	"github.com/saiddis/todev"
)

type TaskService struct {
	conn *Conn
}

func NewTaskService(conn *Conn) *TaskService {
	return &TaskService{conn: conn}
}

func (s *TaskService) CreateTask(ctx context.Context, task *todev.Task) error {
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

	if err = createTask(ctx, tx, task); err != nil {
		return err
	} else if err = attachTaksAssociations(ctx, tx, task); err != nil {
		return err
	}
	return nil
}

func createTask(ctx context.Context, tx *Tx, task *todev.Task) (err error) {
	task.CreatedAt = tx.now
	task.UpdatedAt = task.CreatedAt

	if err = task.Validate(); err != nil {
		return err
	}

	if err = checkRepoExists(ctx, tx, task.RepoID); err != nil {
		return err
	} else if _, err = findContributorByID(ctx, tx, task.ContributorID); err != nil {
		return nil
	}

	err = tx.QueryRowContext(ctx, `
		INSERT INTO tasks (
			description,
			repo_id,
			contributor_id,
			created_at,
			updated_at
		)
		VALUES ($1, $2, $3, $4, $5)
		RETURNING id;
		`,
		&task.Description,
		&task.RepoID,
		&task.ContributorID,
		(*NullTime)(&task.CreatedAt),
		(*NullTime)(&task.UpdatedAt),
	).Scan(task.ID)
	if err != nil {
		return fmt.Errorf("error scanning: %w", err)
	}

	return nil
}

func findTasksByID(ctx context.Context, tx *Tx, id int) (*todev.Task, error) {
	tasks, _, err := findTasks(ctx, tx, todev.TaskFilter{ID: &id})
	if err != nil {
		return nil, fmt.Errorf("error retrieving task by ID: %w", err)
	} else if len(tasks) == 0 {
		return nil, todev.Errorf(todev.ENOTFOUND, "Task not found.")
	}

	return tasks[0], nil
}

func findTasks(ctx context.Context, tx *Tx, filter todev.TaskFilter) ([]*todev.Task, int, error) {
	where, args := []string{"1 = 1"}, []interface{}{}
	var argIndex int
	if v := filter.ID; v != nil {
		argIndex++
		where, args = append(where, fmt.Sprintf("t.id = $%d", argIndex)), append(args, *v)
	}
	if v := filter.RepoID; v != nil {
		argIndex++
		where, args = append(where, fmt.Sprintf("t.repo_id = $%d", argIndex)), append(args, *v)
	}
	if v := filter.ContributorID; v != nil {
		argIndex++
		where, args = append(where, fmt.Sprintf("t.contributor_id = $%d", argIndex)), append(args, *v)
	}
	if v := filter.IsCompleted; v != nil {
		argIndex++
		where, args = append(where, fmt.Sprintf("t.is_completed = $%d", argIndex)), append(args, *v)
	}

	argIndex++
	where = append(where, fmt.Sprintf(`(
		r.user_id = $%d OR
		c.repo_id IN (SELECT c1.repo_id FROM contributors c1 WHERE c1.user_id = $%d)
		)`, argIndex, argIndex+1),
	)
	argIndex++

	userID := todev.UserIDFromContext(ctx)
	var sortBy string
	switch filter.SortBy {
	case todev.TasksSortByCreatedAtDesc:
		sortBy = "t.created_at DESC"
	default:
		argIndex++
		sortBy = fmt.Sprintf(`CASE c.uesr_id WHEN $%d THEN 0 ELSE 1 END ASC, t.is_completed DESC`, argIndex)
		args = append(args, userID)
	}
	args = append(args, userID, userID)

	stmt, err := tx.PrepareContext(ctx, `
		SELECT
			t.id
			t.repo_id
			t.contributor_id,
			t.description,
			t.is_completed,
			t.created_at,
			t.updated_at,
			COUNT(*) OVER()
		FROM tasks t
		JOIN repos r ON t.repo_id = r.id
		JOIN contributors c ON t.contributor_id = c.id
		JOIN users u ON c.user_id = u.id
		WHERE `+strings.Join(where, " AND ")+`
		GROUP BY t.id, r.user_id, t.is_completed
		ORDER BY `+sortBy+`
		`+FormatLimitOffset(filter.Limit, filter.Offset)+`;`,
	)
	if err != nil {
		return nil, 0, fmt.Errorf("error preparing query: %w", err)
	}

	rows, err := stmt.QueryContext(ctx, args...)
	if err != nil {
		return nil, 0, fmt.Errorf("error retrieving tasks: %w", err)
	}
	defer func() {
		err := rows.Close()
		if err != nil {
			log.Printf("error closing rows: %v", err)
		}
	}()

	tasks := make([]*todev.Task, 0)

	// Contributor ID is nullable since this field is optional.
	var contributorID sql.NullInt32
	var n int

	for rows.Next() {
		var task todev.Task
		if err = rows.Scan(
			&task.ID,
			&task.RepoID,
			&contributorID,
			&task.IsCompleted,
			&task.CreatedAt,
			&task.UpdatedAt,
			&n,
		); err != nil {
			return nil, 0, fmt.Errorf("error scanning: %w", err)
		}
		if contributorID.Valid {
			task.ContributorID = int(contributorID.Int32)
		}

		tasks = append(tasks, &task)
	}

	if err = rows.Err(); err != nil {
		return nil, 0, fmt.Errorf("error itarating over rows: %w", err)
	}

	return tasks, n, nil
}

func updateTask(ctx context.Context, tx *Tx, id int, upd todev.TaskUpdate) (*todev.Task, error) {
	task, err := findTasksByID(ctx, tx, id)
	if err != nil {
		return nil, fmt.Errorf("retrieving task by ID: %w", err)
	} else if todev.UserIDFromContext(ctx) != task.Repo.UserID {
		return nil, todev.Errorf(todev.ECONFLICT, "You are not allowed to create tasks.")
	}

	if v := upd.ContributorID; v != nil {
		task.ContributorID = *v
	}
	if v := upd.IsCompleted; v != nil {
		task.IsCompleted = *v
	}
	if v := upd.Description; v != nil {
		task.Description = *v
	}

	task.UpdatedAt = tx.now

	if err = task.Validate(); err != nil {
		return nil, err
	}

	_, err = tx.ExecContext(ctx, `
		UPDATE tasks
		SET
			contributor_id = $1, 
			is_completed = $2, 
			description = $3 
			updated_at = $4 
		WHERE id = $5;`,
		task.ContributorID,
		task.IsCompleted,
		task.Description,
		task.UpdatedAt,
		id,
	)
	if err != nil {
		return task, fmt.Errorf("error updating task: %w", err)
	}

	if err = publishRepoEvent(ctx, tx, task.RepoID, todev.Event{
		Type: todev.EventTypeRepoTaskAdded,
		Payload: &todev.RepoTaskAdded{
			ID:   id,
			Task: task,
		},
	}); err != nil {
		return nil, fmt.Errorf("error publishing task event: %w", err)
	}

	return task, nil
}

func deleteTask(ctx context.Context, tx *Tx, id int) error {
	userID := todev.UserIDFromContext(ctx)

	task, err := findTasksByID(ctx, tx, id)
	if err != nil {
		return fmt.Errorf("error retrieving task by ID: %w", err)
	} else if err = attachTaksAssociations(ctx, tx, task); err != nil {
		return err
	}

	// Verify user is the repo owner.
	if task.Repo.UserID != userID {
		return todev.Errorf(todev.EUNAUTHORIZED, "You do not have permission to delete the task.")
	}

	return nil
}

func attachTaksAssociations(ctx context.Context, tx *Tx, task *todev.Task) (err error) {
	if task.Repo, err = findRepoByID(ctx, tx, task.RepoID); err != nil {
		return fmt.Errorf("error attaching task repo: %w", err)
	} else if task.Contributor, err = findContributorByID(ctx, tx, task.ContributorID); err != nil {
		return fmt.Errorf("error attaching task contributor: %w", err)
	}
	return nil
}
