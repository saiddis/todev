package todev

import (
	"context"
	"fmt"
	"time"
	"unicode/utf8"
)

// Repo constants.
const (
	MaxRepoNameLen = 32
)

// Repo represents a github project on which the the owner of the repo
// adds tasks for the team (contributors).
type Repo struct {
	// List of associated members and their contributing tasks.
	Contributors []*Contributor `json:"contributors,omitempty"`

	// List of the tasks attached to the repo.
	Tasks []*Task `json:"tasks"`

	// Subscription object for recieving events from an event service.
	Subscription Subscription `json:"subscribtion"`

	// Timestamps for repo creation and last update.
	CreatedAt time.Time `json:"createdAt"`
	UpdatedAt time.Time `json:"updatedAt"`

	// Human-readable name of repo.
	Name string `json:"name"`

	// Code used to share the repo with other users.
	InviteCode string `json:"inviteCode,omitempty"`

	// user ID of the repo owner.
	UserID int `json:"userID"`

	ID int `json:"id"`
}

// ContributorByUserID returns the contributor attached to the repo for the given user ID.
func (r Repo) ContributorByUserID(ctx context.Context, userID int) *Contributor {
	for _, m := range r.Contributors {
		if m.UserID == userID {
			return m
		}
	}
	return nil
}

// TasksByContributorID returns the tasks attached to contributor by the given contributor ID.
func (r Repo) TasksByContributorID(ctx context.Context, contribID int) ([]*Task, error) {
	for _, c := range r.Contributors {
		if c.ID == contribID {
			tasks := make([]*Task, len(r.Tasks))
			for i := 0; i < len(tasks); i++ {
				tasks[i] = r.Tasks[i]
			}
			return tasks, nil
		}
	}
	return nil, Errorf(ENOTFOUND, "No contributor for the given id: %d", contribID)
}

// Validate retruns an error if a repo has invalid fields.
func (r Repo) Validate() error {
	if r.Name == "" {
		return Errorf(EINVALID, "Repo name required.")
	} else if utf8.RuneCountInString(r.Name) > MaxRepoNameLen {
		return Errorf(EINVALID, "Repo name too long.")
	} else if r.UserID == 0 {
		return Errorf(EINVALID, "Repo creator required.")
	}
	return nil
}

// CanEditRepo returns true if the current user can edit the repo.
func CanEditRepo(ctx context.Context, repo Repo) bool {
	return repo.UserID == UserIDFromContext(ctx)
}

// RepoService represents a service for managing repos.
type RepoService interface {
	// Retrieves a single repo by ID along with associated contributors.
	FindRepoByID(ctx context.Context, id int) (*Repo, error)

	// Retrieves a list of repos based on a filter.
	FindRepos(ctx context.Context, filter RepoFilter) ([]*Repo, int, error)

	// Creates a new repo and assigns the current user as the owner.
	CreateRepo(ctx context.Context, repo *Repo) error

	// Updates an existing repo by ID. Only the repo owner can update a repo.
	UpdateRepo(ctx context.Context, id int, upd RepoUpdate) (*Repo, error)

	// Permanently deletes a repo by ID. Only the repo owner can delete a repo.
	DeleteRepo(ctx context.Context, id int) error

	// Sets a task for the given user's contributor in a repo.
	// SetContributorTask(ctx context.Context, repoID int, task Task) error

	// TasksLeftReport returns a report of all the tasks due in this repo.
	// TasksLeftReport(ctx context.Context, start, end time.Time, interval time.Duration) (*RepoTasksReport, error)
}

// RepoFilter represents a filter used by FilterRepo().
type RepoFilter struct {
	// Filtering fields.
	ID         *int    `json:"id"`
	InviteCode *string `json:"inviteCode"`

	// Restrict to subset of range.
	Offset int `json:"offset"`
	Limit  int `json:"limit"`
}

// RepoUpdate represents a set of fields to update on a repo.
type RepoUpdate struct {
	Name *string `json:"name"`
}

// RepoTasksReport represents a report generated by TasksLeftReport().
type RepoTasksReport struct {
	Records []*RepoTasksRecord
}

// RepoTasksRecord represents all uncompleted tasks.
type RepoTasksRecord struct {
	Tasks     []Task    `json:"tasks"`
	Timestamp time.Time `json:"timestamp"`
}

// GoString prints a more easily readable report representation for debugging.
func (r *RepoTasksRecord) GoString() string {
	return fmt.Sprintf("&todev.RepoTasksRecord(\n    Tasks:%v,\n    Timestamp:%q)", r.Tasks, r.Timestamp.Format(time.RFC3339))
}
