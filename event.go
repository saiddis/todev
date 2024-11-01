package todev

import "context"

// Event type constants.
const (
	EventTypeRepoTaskCompleted           = "repo:task_completed"
	EventTypeRepoMembershipTaskCompleted = "repo_membership:task_completed"
	EventTypeRepoTaskAdded               = "repo:task_added"
	EventTypeRepoMembershipTaskAdded     = "repo_membership:tasks_added"
)

// Event represents an event that occurs in the system.
type Event struct {
	// Specifies the type of event that is occuring.
	Type string `json:"type"`

	// The actual data from the event.
	Payload interface{} `json:"payload"`
}

// RepoTaskCompleted represents a payloads for an event.
type RepoTaskCompleted struct {
	ID     int `json:"id"`
	TaskID int `json:"taskID"`
}

// RepoMembershipTaskCompleted represents a payloads for an event.
type RepoMembershipTaskCompleted struct {
	ID     int `json:"id"`
	TaskID int `json:"taskID"`
}

// RepoTaskAdded represents a payloads for an event.
type RepoTaskAdded struct {
	ID   int   `json:"id"`
	Task *Task `json:"task"`
}

// RepoMembershipTaskAdded represents a payloads for an event.
type RepoMembershipTaskAdded struct {
	ID   int   `json:"id"`
	Task *Task `json:"task"`
}

type EventService interface {
	// Publiches an event to a user's event listeners.
	PublishEvent(userID int, event Event)

	// Creates a subscription for the current user's events.
	Subscribe(ctx context.Context) (Subscription, error)
}

type Subscription interface {
	// Event stream for all user's event.
	C() <-chan Event

	// Closes the event stream channel and disconnects from the service.
	Close() error
}
