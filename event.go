package todev

import "context"

// Event type constants.
const (
	EventTypeRepoTaskAdded                = "repo:task_added"
	EventTypeContributorTaskAdded         = "contributor:tasks_added"
	EventTypeRepoTaskCompletionToggled    = "repo:task_completion_toggled"
	EventTypeRepoTaskDescriptionChanged   = "repo:task_description_changed"
	EventTypeRepoTaskContributorIDChanged = "repo:task_contributor_id_changed"
	EventTypeRepoTaskDeleted              = "repo:task_deleted"
	EventTypeContributorTaskDeleted       = "contirbutor:task_deleted"
)

// Event represents an event that occurs in the system.
type Event struct {
	// Specifies the type of event that is occuring.
	Type string `json:"type"`

	// The actual data from the event.
	Payload interface{} `json:"payload"`
}

// RepoTaskAdded represents a payload for an event and
// is due to create a new task object.
type RepoTaskAdded struct {
	Task *Task `json:"task"`
}

// ContributorTaskAdded represents a payload for an event and
// is due to create a new task object.
type ContributorTaskAdded struct {
	Task *Task `json:"task"`
}

// RepoTaskCompletionToggled represents a payload for an event and
// is due to update IsCompleted field of a task object to the
// opposite of the current one.
type RepoTaskCompletionToggled struct {
	ID int `json:"id"`
}

// RepoTaskDescriptionChanged represents a payload for an event and
// is due to update Description field of a task object.
type RepoTaskDescriptionChanged struct {
	ID    int    `json:"id"`
	Value string `json:"value"`
}

// RepoTaskDescriptionChanged represents a payload for an event and
// is due to update ContributorID field of a task object.
type RepoTaskContributorIDChanged struct {
	ID    int `json:"id"`
	Value int `json:"value"`
}

// RepoTaskDeleted represents a payload for an event and
// is due to delete a task object from a repo by ID.
type RepoTaskDeleted struct {
	ID int `json:"id"`
}

type EventService interface {
	// Publiches an event to a user's event listeners.
	PublishEvent(id int, event Event) error

	// Creates a subscription for the current user's events.
	Subscribe(ctx context.Context) (Subscription, error)

	// Returns subscribtions and true if exists, otherwise Subscribtion
	// zero value and false.
	GetSubscribtion(id int) (Subscription, bool)
}

type Subscription interface {
	// Event stream for all user's event.
	C() <-chan Event

	// For unsubscribing from service.
	Done() chan struct{}

	// For cleaning up after calling Done().
	Close()
}
