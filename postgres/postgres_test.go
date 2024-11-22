package postgres_test

import (
	"crypto/rand"
	"encoding/hex"
	"flag"
	"fmt"
	"log"
	"net/url"
	"sync"
	"testing"

	"github.com/joho/godotenv"
	"github.com/saiddis/todev/postgres"
)

var (
	once    sync.Once
	envData map[string]string
)

// GetEnv returns environment variables by key.
func GetEnv(key string) (value string, ok bool) {
	var err error
	once.Do(func() {
		envData, err = godotenv.Read("../.env")
		if err != nil {
			log.Fatalf("error reading env file: %v", err)
		}
	})

	value, ok = envData[key]
	return
}

var pgaddr = flag.String("database", "", "database address")

// Ensure the test database can open & close.
func TestConn(t *testing.T) {
	WithSchema(t, nil)
}

type testFunc func(t testing.TB, conn *postgres.Conn)

// WithSchema create a new schema runs given test argument on it.
func WithSchema[TB testing.TB](tb TB, test testFunc) {
	if *pgaddr == "" {
		if url, ok := GetEnv("DB_URL"); ok != false {
			*pgaddr = url
		} else {
			tb.Skip("-database flag is nog defined; skipping test")
		}
	}

	// We need to create a unique schema name so that our parallel tests don't clash.
	id := make([]byte, 4)
	rand.Read(id)
	name := "test_" + hex.EncodeToString(id)

	db := postgres.New(*pgaddr)

	if err := db.Open(); err != nil {
		tb.Fatalf("error connecting to the database: %v", err)
	}
	defer func() { _ = db.Close() }()

	defer func() {
		if err := dropSchema(db, name); err != nil {
			tb.Fatal(err)
		}
	}()

	if err := createSchema(db, name); err != nil {
		tb.Fatal(err)
	}

	// rows, err := db.DB.Query(`SELECT * FROM migrations;`)
	// if err != nil {
	// 	tb.Fatalf("error retrieving migrations: %v", err)
	// }
	//
	// tables := make([]string, 0, 8)
	// var table string
	// for rows.Next() {
	// 	if err := rows.Scan(&table); err != nil {
	// 		tb.Fatalf("error scanning rows: %v", err)
	// 	} else {
	// 		tables = append(tables, table)
	// 	}
	//
	// }
	// tb.Logf("tables: %v", tables)
	if test != nil {
		test(tb, db)
	}

}

// connstrWithSchema adds the search_path argument to the connection string.
func connstrWithSchema(connstr, schema string) (string, error) {
	u, err := url.Parse(connstr)
	if err != nil {
		return "", fmt.Errorf("invalid connstr: %q, error: %v", connstr, err)
	}

	// Get query parameters, set search_path, and encode back
	q := u.Query()
	q.Set("search_path", schema)
	u.RawQuery = q.Encode()

	return u.String(), nil
}

// createSchema createa a new schema in the database.
func createSchema(conn *postgres.Conn, schema string) error {
	_, err := conn.DB.Exec("CREATE SCHEMA IF NOT EXISTS " + schema + ";")
	if err != nil {
		return fmt.Errorf("error creating schema: %v", err)
	}
	_, err = conn.DB.Exec(fmt.Sprintf("SET search_path TO %s, public;", schema))
	if err != nil {
		return fmt.Errorf("error setting search_path: %v", err)
	}

	//tables := []string{"users", "auths", "repos", "tasks", "contributors"}
	//for _, table := range tables {
	//	query := fmt.Sprintf("CREATE TABLE %s.%s AS TABLE public.%s WITH NO DATA;", schema, table, table)
	//	_, err := conn.DB.Exec(query)
	//	if err != nil {
	//		return fmt.Errorf("error creating schema table: %v", err)
	//	}
	//}
	if err := conn.Migrate(); err != nil {
		return fmt.Errorf("error migrating: %v", err)
	}

	return nil
}

// dropSchema drops the specified schema and associated data.
func dropSchema(conn *postgres.Conn, schema string) error {
	_, err := conn.DB.Exec("DROP SCHEMA " + schema + " CASCADE;")
	if err != nil {
		return fmt.Errorf("error dropping schema: %v", err)
	}
	return nil
}
