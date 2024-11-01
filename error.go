package todev

import (
	"errors"
	"fmt"
)

// Application error codes.
const (
	ECONFLICT       = "conflict"
	EINTERNAL       = "internal"
	EINVALID        = "invalid"
	ENOTFOUND       = "not_found"
	ENOTIMPLEMENTED = "not_implemented"
	EUNAUTHORIZED   = "unauthorized"
)

// Error represents an application-specific error.
type Error struct {
	// Machine-readable error code.
	Code string

	// Human-readable error code.
	Message string
}

// Error implements error interface.
func (e *Error) Error() string {
	return fmt.Sprintf("todev error: code=%s message=%s", e.Code, e.Message)
}

// ErrorCode unwraps an application error and returns its code.
// Non-application errors alwasys return EINTERVAL.
func ErrorCode(err error) string {
	var e *Error
	if err == nil {
		return ""
	} else if errors.As(err, &e) {
		return e.Code
	}
	return EINTERNAL
}

// ErrorMessage unwraps an application error and returns its message.
// Non-application errors alwasys return "invetrnal error".
func ErrorMessage(err error) string {
	var e *Error
	if err == nil {
		return ""
	} else if errors.As(err, &e) {
		return e.Message
	}
	return "Internal error."
}

// ErrorF is a helper function to return an Error with a given code and formatted message.
func Errorf(code, format string, args ...interface{}) *Error {
	return &Error{
		Code:    code,
		Message: fmt.Sprintf(format, args...),
	}
}
