package todev

import (
	"context"
	"time"
)

// RepoMembership represensts a contributor to s repo.
type RepoMembership struct {
	ID int `json:"id"`

	// Parent repo. This repo updates when a membership updates
	RepoID int   `json:"repoID"`
	Repo   *Repo `json:"repo"`

	// Owner of the membership. Only this user can update the membership.
	UserID int   `json:"userID"`
	User   *User `json:"user"`

	// Tasks given to a contributor.
	Tasks []*Task `json:"tasks"`

	// Timestamps for membership creation and last update.
	CratedAt  time.Time `json:"createdAt"`
	UpdatedAt time.Time `json:"updatedAt"`
}

// CanEditRepoMembership returns true if the current user can edit membership.
func CanEditRepoMembership(ctx context.Context, membership RepoMembership) bool {
	return membership.UserID == UserIDFromContext(ctx)
}

func CanDeleteRepoMembership(ctx context.Context, membership *RepoMembership) bool {

	userID := UserIDFromContext(ctx)
	if membership.Repo != nil {
		if membership.Repo.UserID == membership.UserID {
			return false // repo owner cannot delete membership
		} else if membership.Repo.UserID == userID {
			return true // repo owner can delete other memberships
		}
	}
	return membership.UserID == userID // non-repo owner can delete own membership
}

// Validate returns an error if any of membership fields are invalid.
func (m RepoMembership) Validate() error {
	if m.RepoID == 0 {
		return Errorf(EINVALID, "Repo required for membership.")
	} else if m.UserID == 0 {
		return Errorf(EINVALID, "User required for membership.")
	}
	return nil
}

type RepoMembershipService interface {
	// Rettrives a membership by ID along with asscciated repo and user.
	FindRepoMembershipByID(ctx context.Context, id int) (*RepoMembership, error)

	// Retrieves a list of matching memberships based on filter.
	FindRepoMemberships(ctx context.Context, filter RepoMembershipFilter) ([]*RepoMembership, int, error)

	// Creates a new membership on a repo for the current user.
	CreateRepoMembership(ctx context.Context, membership RepoMembership) error

	// Updates the value of a membership.
	UpdateRepoMembership(ctx context.Context, id int, upd RepoMembershipUpdate) (*RepoMembership, error)

	// Premanetly deletes membership by ID.
	DeleteRepoMembership(ctx context.Context, id int) error
}

const (
	RepoMembershipSortByUpdatedAtDesc = "updated_at_desc"
)

// RepoMembershipFilter represents a filter used by FindRepoMemberships().
type RepoMembershipFilter struct {
	ID     *int `json:"id"`
	DialID *int `json:"dialID"`
	UserID *int `json:"userID"`

	// Restricts to a subset of results.
	Offset int `json:"offset"`
	Limit  int `json:"limit"`

	// Sorting option for results.
	SortBy string `json:"sortBy"`
}

// RepoMembershipUpdate represents a set of fields to update on a membership.
type RepoMembershipUpdate struct {
	Tasks *int `json:"tasks"`
}
