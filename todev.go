package todev

import (
	"context"
	"log"
)

// Build version and commint SHA
var (
	Version string
	Commit  string
)

// ReportError notifies an external service of errors. No-op by default.
var ReportError = func(ctx context.Context, err error, args ...interface{}) {}

// ReportPanic notifies an external service of panics. No-op by default.
var ReportPanic = func(err interface{}) {
	log.Printf("panic: %v", err)
}
