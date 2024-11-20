package todev

import (
	"context"
	"time"
)

// Contributor represensts a contributor to s repo.
type Contributor struct {
	ID int `json:"id"`

	// Parent repo. This repo updates when a contributor updates
	RepoID int   `json:"repoID"`
	Repo   *Repo `json:"repo"`

	// Only this user can update the membership.
	UserID int   `json:"userID"`
	User   *User `json:"user"`

	// Tasks given to a contributor.
	Tasks []*Task `json:"tasks"`

	// Timestamps for contributor creation and last update.
	CratedAt  time.Time `json:"createdAt"`
	UpdatedAt time.Time `json:"updatedAt"`
}

// CanEditContributor returns true if the current user can edit contributor.
func CanEditContributor(ctx context.Context, contrib Contributor) bool {
	return contrib.UserID == UserIDFromContext(ctx)
}

func CanDeleteContributor(ctx context.Context, contrib *Contributor) bool {

	userID := UserIDFromContext(ctx)
	if contrib.Repo != nil {
		if contrib.Repo.UserID == contrib.UserID {
			return false // repo owner cannot delete contributor
		} else if contrib.Repo.UserID == userID {
			return true // repo owner can delete other contributors
		}
	}
	return contrib.UserID == userID // non-repo owner can delete own membership
}

// Validate returns an error if any of contributor fields are invalid.
func (m Contributor) Validate() error {
	if m.RepoID == 0 {
		return Errorf(EINVALID, "Repo required for contributor.")
	} else if m.UserID == 0 {
		return Errorf(EINVALID, "User required for contributor.")
	}
	return nil
}

type ContributorService interface {
	// Rettrives a contributor by ID along with asscciated repo and user.
	FindContributorByID(ctx context.Context, id int) (*Contributor, error)

	// Retrieves a list of matching contributors based on filter.
	FindContributors(ctx context.Context, filter ContributorFilter) ([]*Contributor, int, error)

	// Creates a new contributor on a repo for the current user.
	CreateContributor(ctx context.Context, contrib Contributor) error

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
	DialID *int `json:"dialID"`
	UserID *int `json:"userID"`

	// Restricts to a subset of results.
	Offset int `json:"offset"`
	Limit  int `json:"limit"`

	// Sorting option for results.
	SortBy string `json:"sortBy"`
}

// ContributorUpdate represents a set of fields to update on a contributor.
type ContributorUpdate struct {
	Tasks *int `json:"tasks"`
}
