package todev

import "context"

// Event type constants.
const (
	EventTypeTaskAdded               = "task:added"
	EventTypeTaskCompletionToggled   = "task:completion_toggled"
	EventTypeTaskDescriptionChanged  = "task:description_changed"
	EventTypeTaskAttachContributor   = "task:attach_contributor"
	EventTypeTaskUnattachContributor = "task:unattach_contributor"
	EventTypeTaskDeleted             = "task:deleted"
	EventTypeContributorAdded        = "contributor:added"
	EventTypeContributorSetAdmin     = "contributor:set_admin"
	EventTypeContributorResetAdmin   = "contributor:reset_admin"
	EventTypeContributorDeleted      = "contributor:deleted"
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
type ContributorAdded struct {
	Contributor *Contributor `json:"contributor"`
}

// RepoTaskCompletionToggled represents a payload for an event and
// is due to update IsCompleted field of a task object to the
// opposite of the current one.
type ContributorSetAdmin struct {
	ID int `json:"id"`
}

type ContributorResetAdmin struct {
	ID int `json:"id"`
}

// RepoTaskDeleted represents a payload for an event and
// is due to delete a task object from a repo by ID.
type ContributorDeleted struct {
	ID int `json:"id"`
}

// RepoTaskAdded represents a payload for an event and
// is due to create a new task object.
type TaskAdded struct {
	Task *Task `json:"task"`
}

// RepoTaskCompletionToggled represents a payload for an event and
// is due to update IsCompleted field of a task object to the
// opposite of the current one.
type TaskCompletionToggled struct {
	ID int `json:"id"`
}

// TaskContributorAttached represents a payload for an event and
// is due to attach a contributor ID into the task for the given task ID.
type TaskContributorAttached struct {
	ContributorID int `json:"contributorID"`
	TaskID        int `json:"taskID"`
}

// TaskContributorAttached represents a payload for an event and
// is due to unattach a contributor ID into the task for the given task ID.
type TaskContributorUnattached struct {
	ContributorID int `json:"contributorID"`
	TaskID        int `json:"taskID"`
}

// RepoTaskDescriptionChanged represents a payload for an event and
// is due to update Description field of a task object.
type TaskDescriptionChanged struct {
	ID    int    `json:"id"`
	Value string `json:"value"`
}

// RepoTaskDescriptionChanged represents a payload for an event and
// is due to update ContributorID field of a task object.
type TaskContributorIDChanged struct {
	ID    int `json:"id"`
	Value int `json:"value"`
}

// RepoTaskDeleted represents a payload for an event and
// is due to delete a task object from a repo by ID.
type TaskDeleted struct {
	ID int `json:"id"`
}

type EventService interface {
	// Publiches an event to a user's event listeners.
	PublishEvent(id int, event Event)

	// Creates a subscription for the current user's events.
	Subscribe(ctx context.Context) (Subscription, error)
}

type Subscription interface {
	// Event stream for all user's event.
	C() <-chan Event

	// For cleaning up after calling Done().
	Close()
}

// NopEventService returns an event service that does nothing.
func NopEventService() EventService { return &nopEventService{} }

type nopEventService struct{}

func (*nopEventService) PublishEvent(id int, event Event) {}

func (*nopEventService) Subscribe(ctx context.Context) (Subscription, error) {
	panic("not implemented")
}
