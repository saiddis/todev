package todev

import (
	"context"
	"time"
	"unicode/utf8"
)

// Task constants.
const (
	MaxTaskDescriptionLen = 150
)

// Task represents a task that is added by the owner of the repo.
type Task struct {
	ID int `json:"id"`

	// Description for the task.
	Description string `json:"statement"`

	// To indicate whether the task is done or not.
	IsCompleted bool `json:"isCompleted"`

	// ID of a repo in which the task was given.
	RepoID int   `json:"repoID"`
	Repo   *Repo `json:"repo"`

	// ID of a contributor to whome the task was given (optional).
	ContributorID int          `json:"conributorID"`
	Contributor   *Contributor `json:"contributor"`

	// Timestamps for task creation and last update.
	CreatedAt time.Time `json:"createdAt"`
	UpdatedAt time.Time `json:"updatedAt"`
}

const (
	TasksSortByUpdatedAtDesc     = "updated_at_desc"
	TasksSortByCreatedAtDesc     = "created_at_desc"
	TasksSortByIsCompletedAtDesc = "is_completed_at_desc"
)

// Validte retruns an error if a task has invalid fields.
func (t Task) Validate() error {
	if len(t.Description) == 0 {
		return Errorf(EINVALID, "Task statement required.")
	} else if utf8.RuneCountInString(t.Description) > MaxTaskDescriptionLen {
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
	// Retrieves a single task by ID along with associated conributor ID (if set).
	FindTaskByID(ctx context.Context, id int) (*Task, error)

	FindTasks(ctx context.Context, filter TaskFilter) ([]*Task, int, error)

	// Creates a new task.
	CreateTask(ctx context.Context, task Task) error

	// Updates an existing task by ID. Only the repo owner can update a task.
	UpdateTask(ctx context.Context, id int, upd TaskUpdate) (*Task, error)

	// Permanently deletes a taks by ID. Only the repo owner can delete a task.
	DeleteTaks(ctx context.Context, id int) error
}

// TaskFilter represents a filter used by FindTasks().
type TaskFilter struct {
	ID            *int  `json:"id"`
	ContributorID *int  `json:"contributorID"`
	RepoID        *int  `json:"repoID"`
	IsCompleted   *bool `json:"isCompleted"`

	// Restricts to a subset of results.
	Offset int `json:"offset"`
	Limit  int `json:"limit"`

	// Sorting option for results.
	SortBy string `json:"sortBy"`
}

// TaskUpdate represents a set of fields to update on a task.
type TaskUpdate struct {
	Description   *string `json:"statement"`
	ContributorID *int    `json:"contributorID"`
	IsCompleted   *bool   `json:"isCompleted"`
}
