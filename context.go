package todev

import "context"

// contextKey represents an initial key for adding context fields.
type contextKey int

// List of keys.
// These are used to store request-scoped information.
const (
	// Stores the current logged in user in the context.
	userContextKey = contextKey(iota + 1)

	// Stores the "flash" in the context.
	flashContextKey
)

// NewContextWithUser returns a new context with the given user.
func NewContextWithUser(ctx context.Context, user *User) context.Context {
	return context.WithValue(ctx, userContextKey, user)
}

// UserFromContext returns the current logged in user.
func UserFromContext(ctx context.Context) *User {
	user, _ := ctx.Value(userContextKey).(*User)
	return user
}

// UserIDFromContext is helper function that returns the ID of the current logged in user.
func UserIDFromContext(ctx context.Context) int {
	if user := UserFromContext(ctx); user != nil {
		return user.ID
	}
	return 0
}

// NewContextWithFlash returns a new context with the given flash value.
func NewContextWithFlash(ctx context.Context, v string) context.Context {
	return context.WithValue(ctx, flashContextKey, v)
}

// FlashFromContext is helper function that returns the flash value for the current request.
func FlashFromContext(ctx context.Context) string {
	flash, _ := ctx.Value(flashContextKey).(string)
	return flash
}
