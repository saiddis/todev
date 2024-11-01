package postgres

import (
	"context"
	"database/sql"
	"embed"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

var (
	userCountGauge = promauto.NewGauge(prometheus.GaugeOpts{
		Name: "todev_db_users",
		Help: "The total number of users",
	})

	repoCountGauge = promauto.NewGauge(prometheus.GaugeOpts{
		Name: "todev_db_repos",
		Help: "The total number of repositories",
	})

	repoMembershipCountGauge = promauto.NewGauge(prometheus.GaugeOpts{
		Name: "todev_db_repo_memberships",
		Help: "The total number of repositories",
	})
)

var migrationFS embed.FS

// DB represents database connection.
type DB struct {
	db     *sql.DB
	ctx    context.Context // background context
	cancel func()          // cancel background context

	// Database name.
	DSN string

	// Destination for events to be publiched.

	// Runs the current time. Defaults to time.Now().
	// Can be mocked for tests
	Now func() time.Time
}
