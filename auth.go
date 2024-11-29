package todev

import (
	"context"
	"fmt"
	"time"
)

// Authentication providers.
const (
	AuthSourceGitHub = "github"
)

// Auth represents a set of OAuth creadentials.
type Auth struct {
	// OAuth fields returned by the authentication provider.
	AccessToken  string    `json:"-"`
	RefreshToken string    `json:"-"`
	Expiry       time.Time `json:"-"`

	// Timestamps of creation and last update.
	UpdatedAt time.Time `json:"updatedAt"`
	CreatedAt time.Time `json:"createdAt"`

	// User can have one or more metheds of authentication.
	// However, only one per source is allowed per user.
	User *User `json:"user"`

	// The authentication source and the source provider's user ID.
	Source string `json:"source"`

	SourceID string `json:"sourceID"`
	UserID   int    `json:"UserID"`
	ID       int    `json:"id"`
}

// Validate returns an error if any of the fields are invalid on the Auth object.
func (a *Auth) Validate() error {
	if a.UserID == 0 {
		return Errorf(EINVALID, "User required.")
	} else if a.Source == "" {
		return Errorf(EINVALID, "Source required.")
	} else if a.SourceID == "" {
		return Errorf(EINVALID, "Source ID required.")
	} else if a.AccessToken == "" {
		return Errorf(EINVALID, "Access token required.")
	}
	return nil
}

func (a *Auth) AvatarURL(size int) string {
	switch a.Source {
	case AuthSourceGitHub:
		return fmt.Sprintf("https://avatars1.githubusercontent.com/u/%s?s=%d", a.SourceID, size)
	default:
		return ""
	}
}

// AuthService represents a service for managing auths.
type AuthService interface {
	// Looks up authentication object by ID along with the associated object.
	// returns ENOTFOUND if ID doesn't exist.
	FindAuthByID(ctx context.Context, id int) (*Auth, error)

	// Retrieves authentication objects based on filter.
	FindAuths(ctx context.Context)
}

// AuthFilter represents a filter accepted by FindAuths().
type AuthFilter struct {
	// Filtering fields
	ID       *int    `json:"id"`
	UserID   *int    `json:"userId"`
	Source   *string `json:"source"`
	SourceID *string `json:"sourceID"`

	// Restrics results to a subset of the total range.
	Offset int `json:"offset"`
	Limit  int `json:"limit"`
}
