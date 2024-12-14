package todev

import (
	"context"
	"log"
	"sync"

	"github.com/joho/godotenv"
)

// Build version and commint SHA
var (
	Version string
	Commit  string
)

// ReportError notifies an external service of errors. No-op by default.
var ReportError = func(ctx context.Context, err error, args ...interface{}) {}

// ReportPanic notifies an external service of panics. No-op by default.
var ReportPanic = func(err interface{}) {}

var (
	once    sync.Once
	envData map[string]string
)

const DefaultEnvFilePath = "~/todev/.env"

// GetEnv returns a function for returning config values by key from the given
// config path.
func GetFromEnv(path, key string) (string, bool) {
	var err error
	once.Do(func() {
		envData, err = godotenv.Read(path)
		if err != nil {
			log.Fatalf("error reading env file: %v", err)
		}
	})

	val, ok := envData[key]
	return val, ok
}
