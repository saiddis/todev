package todev

import (
	"context"
	"time"
)

// Contributor represensts a contributor to s repo.
type Contributor struct {
	// Tasks that are visible and/or due to the current contributor.
	Tasks []*Task

	// Associated user.
	User *User `json:"user"`

	// Timestamps for contributor creation and last update.
	CreatedAt time.Time `json:"createdAt"`
	UpdatedAt time.Time `json:"updatedAt"`

	// OwnerID is a user ID of contributor's associated repo object.
	OwnerID int `json:"ownerID"`

	// Associated IDs
	RepoID int `json:"repoID"`
	UserID int `json:"userID"`
	ID     int `json:"id"`

	IsAdmin bool `json:"isAdmin"`
}

// CanEditContributor returns true if the current user can edit contributor.
func CanEditContributor(ctx context.Context, contrib Contributor) bool {
	return contrib.UserID == UserIDFromContext(ctx)
}

func CanDeleteContributor(ctx context.Context, contributor Contributor) error {
	userID := UserIDFromContext(ctx)
	// Verify user is the contributor or ownes parent repo.
	if contributor.UserID != userID && contributor.OwnerID != userID {
		return Errorf(EUNAUTHORIZED, "You do not have permission to delete the contributor.")
	} else if contributor.UserID == contributor.OwnerID { // Do not let repo owner delete their own contributor object.
		return Errorf(ECONFLICT, "Repo owner cannot be deleted.")
	}

	return nil
}

// Validate returns an error if any of contributor fields are invalid.
func (m Contributor) Validate() error {
	if m.RepoID == 0 {
		return Errorf(EINVALID, "Repo required for contributing.")
	} else if m.UserID == 0 {
		return Errorf(EINVALID, "User required for contributing.")
	}
	return nil
}

type ContributorService interface {
	// Rettrives a contributor by ID along with asscciated repo and user.
	FindContributorByID(ctx context.Context, id int) (*Contributor, error)

	// Retrieves a list of matching contributors based on filter.
	FindContributors(ctx context.Context, filter ContributorFilter) ([]*Contributor, int, error)

	// Creates a new contributor on a repo for the current user.
	CreateContributor(ctx context.Context, contrib *Contributor) error

	// Updates the value of a contributor.
	UpdateContributor(ctx context.Context, id int, upd ContributorUpdate) (*Contributor, error)

	// Premanetly deletes contributor by ID.
	DeleteContributor(ctx context.Context, id int) error
}

const (
	ContributorSortByUpdatedAtDesc = "updated_at_desc"
)

// ContributorFilter represents a filter used by FindContributors().
type ContributorFilter struct {
	ID     *int `json:"id"`
	RepoID *int `json:"dialID"`
	UserID *int `json:"userID"`

	// Restricts to a subset of results.
	Offset int `json:"offset"`
	Limit  int `json:"limit"`

	// Sorting option for results.
	SortBy string `json:"sortBy"`
}

// ContributorUpdate represents a set of fields to update on a contributor.
type ContributorUpdate struct {
	IsAdmin *bool
}
