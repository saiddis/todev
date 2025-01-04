package todev

import (
	"context"
	"time"
)

// User represents a user on the system.
type User struct {
	// Timestamps for user creation and last update.
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`

	// List of associated OAuth authentication objects.
	Auths []*Auth `json:"auths"`

	// Users prefered name and email.
	Name  string `json:"name"`
	Email string `json:"email"`

	// Randomly generated API key for use with the CLI.
	APIKey string `json:"-"`

	ID int `json:"id"`
}

// Validate returns an error if the user has invalid fields.
func (u User) Validate() error {
	if u.Name == "" {
		return Errorf(EINVALID, "User name required.")
	}

	return nil
}

// AvatarURL retruns a URL to the avatar image for the user.
func (u User) AvatarURL() string {
	for _, auth := range u.Auths {
		if s := auth.AvatarURL(); s != "" {
			return s
		}
	}
	return ""
}

// UserService represents a service for managing users.
type UserService interface {
	// Retrieves a user by ID along with their associated object.
	FindUserByID(ctx context.Context, id int) (*User, error)

	// Retrivieves a list of users by filter.
	FindUsers(ctx context.Context, filter UserFilter) ([]*User, int, error)

	// Creates a new user.
	CreateUser(ctx context.Context, user *User) error

	// Updates a user object.
	UpdateUser(ctx context.Context, id int, upd UserUpdate) (*User, error)

	// Permanently a user and all owned repos.
	DeleteUser(ctx context.Context, id int) error
}

// UserFilter represents a filter to FindUsers().
type UserFilter struct {
	// Filtering fields
	ID     *int    `json:"id"`
	Email  *string `json:"email"`
	APIKey *string `json:"apiKey"`

	// Restrict to subset of results.
	Offset int `json:"offset"`
	Limit  int `json:"limit"`
}

// UserUpdate represents a set of fields to be updated via UpdateUser().
type UserUpdate struct {
	Name  *string `json:"name"`
	Email *string `json:"email"`
}
