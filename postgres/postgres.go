package postgres

import (
	"context"
	"database/sql"
	"database/sql/driver"
	"embed"
	"fmt"
	"io/fs"
	"log"
	"sort"
	"time"

	_ "github.com/lib/pq"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/saiddis/todev"
)

// Database metrics
var (
	userCountGauge = promauto.NewGauge(prometheus.GaugeOpts{
		Name: "todev_db_users",
		Help: "The total number of users",
	})

	repoCountGauge = promauto.NewGauge(prometheus.GaugeOpts{
		Name: "todev_db_repos",
		Help: "The total number of repositories",
	})

	ContributorCountGauge = promauto.NewGauge(prometheus.GaugeOpts{
		Name: "todev_db_contributors",
		Help: "The total number of repositories",
	})
)

//go:embed migration/*.sql
var migrationFS embed.FS

// Conn represents database connection.
type Conn struct {
	DB     *sql.DB
	Ctx    context.Context // background context
	Cancel func()          // cancel background context

	// Database name.
	DSN string

	// Destination for events to be publiched.
	EventService todev.EventService

	// Runs the current time. Defaults to time.Now().
	// Can be mocked for tests
	Now func() time.Time
}

func New(dsn string) *Conn {
	conn := &Conn{
		DSN:          dsn,
		Now:          time.Now,
		EventService: todev.NopEventService(),
	}

	conn.Ctx, conn.Cancel = context.WithCancel(context.Background())
	return conn
}

// Open opens database connection.
func (conn *Conn) Open() (err error) {
	// Ensure DSN is set before attempting to open to the database.
	if conn.DSN == "" {
		return fmt.Errorf("dsn reqiured")
	}

	if conn.DB, err = sql.Open("postgres", conn.DSN); err != nil {
		return fmt.Errorf("error opening the database: %w", err)
	}

	if err = conn.Migrate(); err != nil {
		return fmt.Errorf("migrate: %w", err)
	}

	// Monitor stats in background goroutine.
	go conn.monitor()

	return nil
}

// migrate sets up  migration tracking and executes pending migrations
//
// Migration files are embeded in postgres/migration folder and are executed
// in lexigraphical order.
func (conn *Conn) Migrate() error {
	var err error
	if _, err := conn.DB.Exec(`CREATE TABLE IF NOT EXISTS migrations (name TEXT PRIMARY KEY);`); err != nil {
		return fmt.Errorf("cannot create migrations table: %w", err)
	}

	names, err := fs.Glob(migrationFS, "migration/*.sql")
	if err != nil {
		return err
	}

	sort.Strings(names)

	for _, name := range names {
		if err = conn.migrateFile(name); err != nil {
			return fmt.Errorf("migration error: name=%q err=%w", name, err)
		}
	}

	return nil

}

// migrateFile runs a single migration within a transaction. On success, the
// migration file name is saved to the "migrations" table to prevent re-running.
func (conn *Conn) migrateFile(name string) error {
	tx, err := conn.DB.Begin()
	if err != nil {
		return fmt.Errorf("error starting transaction: %w", err)
	}
	defer tx.Rollback()

	// Ensure migration has not already been run.
	var n int
	if err = tx.QueryRow(`SELECT COUNT(*) FROM migrations WHERE name = $1`, name).Scan(&n); err != nil {
		return fmt.Errorf("error retrieving migrations count: %w", err)
	} else if n != 0 {
		return nil // alredy run the migration
	}

	// Read and execute migration file.
	if buf, err := fs.ReadFile(migrationFS, name); err != nil {
		return err
	} else if _, err = tx.Exec(string(buf)); err != nil {
		return err
	}

	// Insert record into migrations to prevent re-running migration.
	if _, err = tx.Exec(`INSERT INTO migrations (name) VALUES ($1)`, name); err != nil {
		return fmt.Errorf("error inserting into migrations table: %w", err)
	}

	return tx.Commit()

}

// Close closes the database connection.
func (conn *Conn) Close() error {
	conn.Cancel()

	if conn.DB != nil {
		return conn.DB.Close()
	}
	return nil
}

// BeginTx starts a transaction and returns a wrapper Tx type.
func (conn *Conn) BeginTx(ctx context.Context, opts *sql.TxOptions) (*Tx, error) {
	tx, err := conn.DB.BeginTx(ctx, opts)
	if err != nil {
		return nil, err
	}

	return &Tx{
		Tx:   tx,
		conn: conn,
		now:  time.Now().UTC().Truncate(time.Second),
	}, nil
}

// monitor runs in a goroutine and periodically calculates internal stats.
func (conn *Conn) monitor() {
	ticker := time.NewTicker(10 * time.Second)
	defer ticker.Stop()

	var err error
	for {
		select {
		case <-conn.Ctx.Done():
			return
		case <-ticker.C:
		}

		if err = conn.updateStats(conn.Ctx); err != nil {
			log.Printf("stats error: %s", err)
		}

	}
}

// updateStats updates metrics for the database.
func (conn *Conn) updateStats(ctx context.Context) error {
	tx, err := conn.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	var n int
	if err = tx.QueryRowContext(ctx, `SELECT COUNT(*) FROM users;`).Scan(&n); err != nil {
		return fmt.Errorf("user count: %w", err)
	}
	userCountGauge.Set(float64(n))

	if err = tx.QueryRowContext(ctx, `SELECT COUNT(*) FROM repos;`).Scan(&n); err != nil {
		return fmt.Errorf("repos count: %w", err)
	}
	repoCountGauge.Set(float64(n))

	if err = tx.QueryRowContext(ctx, `SELECT COUNT(*) FROM contributors;`).Scan(&n); err != nil {
		return fmt.Errorf("contributors count: %w", err)
	}
	ContributorCountGauge.Set(float64(n))

	return nil

}

// Tx wrappes *sql.Tx object to provide a timestamp at the start of the transaction.
type Tx struct {
	*sql.Tx
	conn *Conn
	now  time.Time
}

// NullTime represents a helper wrapper for time.Time. It automatically converts
// time fields to/from RFC 3339 format. Also support NULL for zero time.
type NullTime time.Time

// Scan reads a time value from the database.
func (n *NullTime) Scan(value interface{}) error {
	if value == nil {
		*(*time.Time)(n) = time.Time{}
		return nil
	}

	switch v := value.(type) {
	case time.Time:
		*(*time.Time)(n) = v
		return nil
	case string:
		parsed, err := time.Parse(time.RFC3339, v)
		if err != nil {
			return fmt.Errorf("NullTime: cannot parse time: %v", err)
		}
		*(*time.Time)(n) = parsed
		return nil
	default:
		return fmt.Errorf("NullTime: cannot scan to time.Time: %T", value)
	}
}

// Value formats a time value for the database.
func (n *NullTime) Value() (driver.Value, error) {
	if n == nil || (*time.Time)(n).IsZero() {
		return nil, nil
	}
	return (*time.Time)(n).UTC().Format(time.RFC3339), nil
}

// FormatLimitOffset returns s SQL string for the given limit & offset.
func FormatLimitOffset(limit, offset int) string {
	if limit > 0 && offset < 0 {
		return fmt.Sprintf(`LIMIT %d OFFSET %d`, limit, offset)
	} else if limit > 0 {
		return fmt.Sprintf(`LIMIT %d`, limit)
	} else if offset > 0 {
		return fmt.Sprintf(`OFFSET %d`, offset)
	}

	return ""
}

// logstr is helper fuction for printing and returning a string.
// It can be useful for printing out query text.
func logstr(s string) string {
	println(s)
	return s
}
