package postgres

import (
	"context"
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

// CreateTask creates a new task in a repo.
// Returns ECONFLICT if contributor creating a task is not the owner.
func (s *TaskService) CreateTask(ctx context.Context, task *todev.Task) error {
	tx, err := s.conn.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("error beginning transaction: %w", err)
	}
	defer func() {
		if err != nil {
			err = fmt.Errorf("CreateTask: %w", err)
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
	} else if attachTaskAssociations(ctx, tx, task); err != nil {
		return err
	} else if task.OwnerID != todev.UserIDFromContext(ctx) {
		return todev.Errorf(todev.ECONFLICT, "Only repo owner can create tasks.")
	} else if err = publishRepoEvent(ctx, tx, task.RepoID, todev.Event{
		Type: todev.EventTypeTaskAdded,
		Payload: todev.TaskAdded{
			Task: task,
		},
	}); err != nil {
		return err
	}

	return nil
}

// FindTasks retrieves a list of matching tasks based on filter.
// Only returns tasks that belong to the current contributor, or all the tasks
// if the the current contributor is the owner.
func (s *TaskService) FindTasks(ctx context.Context, filter todev.TaskFilter) ([]*todev.Task, int, error) {
	tx, err := s.conn.BeginTx(ctx, nil)
	if err != nil {
		return nil, 0, fmt.Errorf("error beginning transaction: %w", err)
	}
	defer func() {
		if err != nil {
			err = fmt.Errorf("FindTasks: %w", err)
			if err := tx.Rollback(); err != nil {
				log.Printf("failed to rollback: %v", err)
			}
		} else {
			if err := tx.Commit(); err != nil {
				log.Printf("failed to commit: %v", err)
			}
		}
	}()

	tasks, n, err := findTasks(ctx, tx, filter)
	if err != nil {
		return nil, 0, err
	}

	for _, task := range tasks {
		if err = attachTaskAssociations(ctx, tx, task); err != nil {
			return nil, 0, err
		}
	}

	return tasks, n, nil
}

// FindContributorsByID retrieves a contributor by ID along with associated repo and contributor. Returns ENOTFOUND
// if task does not exist or contributor does not have permission to view it.
func (s *TaskService) FindTaskByID(ctx context.Context, id int) (*todev.Task, error) {
	tx, err := s.conn.BeginTx(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("error beginning transaction: %w", err)
	}
	defer func() {
		if err != nil {
			err = fmt.Errorf("FindTasks: %w", err)
			if err := tx.Rollback(); err != nil {
				log.Printf("failed to rollback: %v", err)
			}
		} else {
			if err := tx.Commit(); err != nil {
				log.Printf("failed to commit: %v", err)
			}
		}
	}()

	task, err := findTaskByID(ctx, tx, id)
	if err != nil {
		return nil, err
	} else if err = attachTaskAssociations(ctx, tx, task); err != nil {
		return nil, err
	}
	return task, nil
}

func (s *TaskService) UpdateTask(ctx context.Context, id int, upd todev.TaskUpdate) (*todev.Task, error) {
	tx, err := s.conn.BeginTx(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("error beginning transaction: %w", err)
	}
	defer func() {
		if err != nil {
			err = fmt.Errorf("FindTasks: %w", err)
			if err := tx.Rollback(); err != nil {
				log.Printf("failed to rollback: %v", err)
			}
		} else {
			if err := tx.Commit(); err != nil {
				log.Printf("failed to commit: %v", err)
			}
		}
	}()

	task, err := updateTask(ctx, tx, id, upd)
	if err != nil {
		return nil, err
	}
	return task, nil
}

func (s *TaskService) DeleteTask(ctx context.Context, id int) error {
	tx, err := s.conn.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("error beginning transaction: %w", err)
	}
	defer func() {
		if err != nil {
			err = fmt.Errorf("FindTasks: %w", err)
			if err := tx.Rollback(); err != nil {
				log.Printf("failed to rollback: %v", err)
			}
		} else {
			if err := tx.Commit(); err != nil {
				log.Printf("failed to commit: %v", err)
			}
		}
	}()

	if err = deleteTask(ctx, tx, id); err != nil {
		return err
	}

	return nil
}

func createTask(ctx context.Context, tx *Tx, task *todev.Task) (err error) {
	task.CreatedAt = tx.now
	task.UpdatedAt = task.CreatedAt

	if err = task.Validate(); err != nil {
		return err
	} else if err = checkRepoExists(ctx, tx, task.RepoID); err != nil {
		return err
	}

	args := []interface{}{
		task.Description,
		task.RepoID,
		(*NullTime)(&task.CreatedAt),
		(*NullTime)(&task.UpdatedAt),
	}
	insertQuery := []string{"description", "repo_id", "created_at", "updated_at"}
	valuesQuery := []string{"$1", "$2", "$3", "$4"}

	if task.ContributorID != nil {
		args = append(args, task.ContributorID)
		insertQuery = append(insertQuery, "contributor_id")
		valuesQuery = append(valuesQuery, "$5")
	}
	var id int
	err = tx.QueryRowContext(ctx, `
		INSERT INTO tasks (`+strings.Join(insertQuery, ",")+`)
		VALUES (`+strings.Join(valuesQuery, ",")+`)
		RETURNING id;
		`,
		args...,
	).Scan(&id)
	if err != nil {
		return fmt.Errorf("error scanning: %w", err)
	}

	task.ID = id

	return nil
}

func findTaskByID(ctx context.Context, tx *Tx, id int) (*todev.Task, error) {
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
		t.repo_id IN (SELECT repo_id FROM contributors WHERE user_id = $%d)
		)`, argIndex, argIndex+1),
	)
	argIndex++

	userID := todev.UserIDFromContext(ctx)
	var sortBy string
	switch filter.SortBy {
	case todev.TasksSortByCreatedAtDesc:
		sortBy = "t.created_at DESC"
	default:
		sortBy = `t.is_completed DESC`
	}
	args = append(args, userID, userID)

	stmt, err := tx.PrepareContext(ctx, `
		SELECT
			t.id,
			t.repo_id,
			t.contributor_id,
			t.is_completed,
			t.description,
			t.created_at,
			t.updated_at,
			COUNT(*) OVER()
		FROM tasks t
		JOIN repos r ON t.repo_id = r.id
		WHERE `+strings.Join(where, " AND ")+`
		GROUP BY t.id
		ORDER BY `+sortBy+`
		`+FormatLimitOffset(filter.Limit, filter.Offset),
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

	var n int
	for rows.Next() {
		var task todev.Task
		if err = rows.Scan(
			&task.ID,
			&task.RepoID,
			&task.ContributorID,
			&task.IsCompleted,
			&task.Description,
			&task.CreatedAt,
			&task.UpdatedAt,
			&n,
		); err != nil {
			return nil, 0, fmt.Errorf("error scanning: %w", err)
		}
		tasks = append(tasks, &task)
	}

	if err = rows.Err(); err != nil {
		return nil, 0, fmt.Errorf("error itarating over rows: %w", err)
	}

	return tasks, n, nil
}

func updateTask(ctx context.Context, tx *Tx, id int, upd todev.TaskUpdate) (_ *todev.Task, err error) {
	task, err := findTaskByID(ctx, tx, id)
	if err != nil {
		return nil, err
	} else if err = attachTaskAssociations(ctx, tx, task); err != nil {
		return nil, err
	} else if !todev.CanEditTask(ctx, *task) {
		return nil, todev.Errorf(todev.ECONFLICT, "You are not allowed to update tasks.")
	}

	if v := upd.ContributorID; v != nil {
		defer func() {
			if err == nil {
				err = publishRepoEvent(ctx, tx, task.RepoID, todev.Event{
					Type: todev.EventTypeTaskAdded,
					Payload: todev.TaskContributorIDChanged{
						ID:    task.ID,
						Value: *v,
					},
				})
			}
		}()
		task.ContributorID = v
	}
	if v := upd.Description; v != nil {
		defer func() {
			err = publishRepoEvent(ctx, tx, task.RepoID, todev.Event{
				Type: todev.EventTypeTaskDescriptionChanged,
				Payload: todev.TaskDescriptionChanged{
					ID:    task.ID,
					Value: *v,
				},
			})
		}()
		task.Description = *v
	}
	if upd.ToggleCompletion {
		defer func() {
			err = publishRepoEvent(ctx, tx, task.RepoID, todev.Event{
				Type: todev.EventTypeTaskCompletionToggled,
				Payload: todev.TaskCompletionToggled{
					ID: task.ID,
				},
			})
		}()
		if task.IsCompleted {
			task.IsCompleted = false
		} else {
			task.IsCompleted = true
		}
	}

	if err = task.Validate(); err != nil {
		return nil, err
	}

	task.UpdatedAt = tx.now

	args := []interface{}{
		task.Description,
		task.RepoID,
		task.IsCompleted,
		(*NullTime)(&task.UpdatedAt),
	}
	idArgIndex := "5;"
	updateQuery := []string{"description = $1", "repo_id = $2", "is_completed = $3", "updated_at = $4"}
	if task.ContributorID != nil {
		args = append(args, *task.ContributorID)
		updateQuery = append(updateQuery, "contributor_id = $5")
		idArgIndex = "6;"
	}
	args = append(args, id)

	_, err = tx.ExecContext(ctx, `
		UPDATE tasks
		SET `+strings.Join(updateQuery, ",")+` WHERE id = $`+idArgIndex,
		args...,
	)
	if err != nil {
		return task, fmt.Errorf("error updating task: %w", err)
	}

	return task, err
}

func deleteTask(ctx context.Context, tx *Tx, id int) error {
	task, err := findTaskByID(ctx, tx, id)
	if err != nil {
		return err
	} else if err = attachTaskAssociations(ctx, tx, task); err != nil {
		return err
	} else if !todev.CanEditTask(ctx, *task) {
		return todev.Errorf(todev.ECONFLICT, "You are not allowed to delete tasks.")
	}

	if _, err = tx.ExecContext(ctx, "DELETE FROM tasks WHERE id = $1;", id); err != nil {
		return fmt.Errorf("error deleting task: %w", err)
	} else if err = publishRepoEvent(ctx, tx, task.RepoID, todev.Event{
		Type: todev.EventTypeTaskDeleted,
		Payload: todev.TaskDeleted{
			ID: task.ID,
		},
	}); err != nil {
		return err
	}

	return nil
}

func attachTaskAssociations(ctx context.Context, tx *Tx, task *todev.Task) (err error) {
	repo, err := findRepoByID(ctx, tx, task.RepoID)
	if err != nil {
		return fmt.Errorf("error attaching task repo: %w", err)
	}
	task.OwnerID = repo.UserID
	return nil
}
