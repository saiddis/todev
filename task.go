package todev

import (
	"context"
	"unicode/utf8"
)

// Task constants.
const (
	MaxTaskStatementLen = 150
)

// Task represents a task that is added by the owner of the repo.
type Task struct {
	ID int `json:"id"`

	// Statement for the task.
	Statement string `json:"statement"`

	// To indicate whether the task is done or not.
	IsCompleted bool `json:"isCompleted"`

	// Message attached to the task for deteails (optional).
	Message string `json:"message"`

	// ID of a user to whome the task given (optional).
	RepoMembershipID int `json:"membershipID"`
}

// Validate retruns an error if a task has invalid fields.
func (t Task) Validate() error {
	if len(t.Statement) == 0 {
		return Errorf(EINVALID, "Task statement required.")
	} else if utf8.RuneCountInString(t.Statement) > MaxTaskStatementLen {
		return Errorf(EINVALID, "Task statement too long.")
	}
	return nil
}

// CanEditTask returns true if the current user can edit the repo.
func CanEditTask(ctx context.Context, repo *Repo) bool {
	return repo.UserID == UserIDFromContext(ctx)
}

// TaskService represents a service for managing a task.
type TaskService interface {
	// Retrieves a single task by ID along with associated membership ID (if set).
	FindTaskByID(ctx context.Context, id int) (*Task, error)

	// Creates a new task.
	CreateTask(ctx context.Context, task Task) error

	// Updates an existing task by ID. Only the repo owner can update a task.
	UpdateTask(ctx context.Context, id int, upd TaskUpdate) (*Task, error)

	// Permanently deletes a taks by ID. Only the repo owner can delete a task.
	DeleteTaks(ctx context.Context, id int) error
}

// TaskUpdate represents a set of fields to update on a task.
type TaskUpdate struct {
	Statement *string `json:"statement"`
	Message   *string `json:"message"`
}
