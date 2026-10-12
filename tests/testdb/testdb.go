// Package testdb starts a throwaway Postgres for tests that need real SQL
// semantics.
//
// The things worth asserting about favorite_dao — that the composite primary key
// makes a repeat favorite idempotent, that the cap check is atomic under
// concurrency, that the list join reports a discontinued item as unavailable —
// are all database behaviour. A mock would only assert the SQL strings back to
// themselves, so these tests run against a real server.
//
// Start returns ErrNoDocker, rather than failing, when docker is unavailable,
// so the suite still runs on a machine without it. The runs that matter (CI, and
// the live-stack verification) are where it must actually execute — a SKIP line
// means it did not.
package testdb

import (
	"database/sql"
	"errors"
	"fmt"
	"math/rand"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	_ "github.com/jackc/pgx/stdlib"
)

const (
	image    = "postgres:14-alpine"
	password = "postgres"
	dbName   = "sandbox_svc"
)

// ErrNoDocker means docker is not on PATH, so no database could be started.
var ErrNoDocker = errors.New("docker not available")

// Start launches Postgres, applies the service's real migration files, and
// returns a connection plus a cleanup func.
func Start() (*sql.DB, func(), error) {
	if _, err := exec.LookPath("docker"); err != nil {
		return nil, nil, ErrNoDocker
	}

	name := fmt.Sprintf("sandbox-svc-test-%d", rand.Intn(1_000_000))

	if out, err := exec.Command("docker", "run", "-d", "--name", name,
		"-e", "POSTGRES_PASSWORD="+password,
		"-e", "POSTGRES_DB="+dbName,
		"-p", "127.0.0.1::5432",
		image,
	).CombinedOutput(); err != nil {
		return nil, nil, fmt.Errorf("starting container %s: %w: %s", name, err, out)
	}

	removeContainer := func() {
		if out, err := exec.Command("docker", "rm", "-f", name).CombinedOutput(); err != nil {
			fmt.Fprintf(os.Stderr, "removing test container %s: %v: %s\n", name, err, out)
		}
	}
	fail := func(err error) (*sql.DB, func(), error) {
		removeContainer()
		return nil, nil, err
	}

	port, err := hostPort(name)
	if err != nil {
		return fail(err)
	}

	dsn := fmt.Sprintf(
		"host=127.0.0.1 port=%s user=postgres password=%s dbname=%s sslmode=disable",
		port, password, dbName,
	)

	db, err := sql.Open("pgx", dsn)
	if err != nil {
		return fail(fmt.Errorf("opening test database: %w", err))
	}

	// Wait on a real connection, not on pg_isready inside the container: the
	// entrypoint runs a temporary server on the unix socket while initialising,
	// which answers pg_isready but does not serve TCP. The published port is
	// reachable before the real server is up, so a ping here can come back EOF
	// on a container that is, in fact, still starting.
	if err := waitPing(db); err != nil {
		return fail(err)
	}

	if err := applyMigrations(db); err != nil {
		return fail(err)
	}

	return db, func() {
		if err := db.Close(); err != nil {
			fmt.Fprintf(os.Stderr, "closing test database: %v\n", err)
		}
		removeContainer()
	}, nil
}

// Truncate resets every table between tests, so a case that marks an item
// discontinued or a user at the cap cannot leak into the next one. RESTART
// IDENTITY lets each test use the same small ids without colliding.
//
// The tests share one container (and so one database) because starting one per
// test costs seconds each; this is what makes that safe.
func Truncate(t *testing.T, db *sql.DB) {
	t.Helper()

	if _, err := db.Exec(`TRUNCATE favorite_items, catalog_items, users RESTART IDENTITY CASCADE`); err != nil {
		t.Fatalf("truncating test tables: %v", err)
	}
}

func waitPing(db *sql.DB) error {
	deadline := time.Now().Add(90 * time.Second)

	var lastErr error
	for time.Now().Before(deadline) {
		if lastErr = db.Ping(); lastErr == nil {
			return nil
		}
		time.Sleep(500 * time.Millisecond)
	}

	return fmt.Errorf("test database did not accept connections within 90s: %w", lastErr)
}

func hostPort(name string) (string, error) {
	out, err := exec.Command("docker", "port", name, "5432").CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("reading mapped port for %s: %w: %s", name, err, out)
	}

	// "127.0.0.1:49213" (possibly one line per binding).
	line := strings.TrimSpace(strings.Split(strings.TrimSpace(string(out)), "\n")[0])
	idx := strings.LastIndex(line, ":")
	if idx < 0 {
		return "", fmt.Errorf("unexpected docker port output for %s: %q", name, out)
	}

	return line[idx+1:], nil
}

// applyMigrations runs the same SQL the service runs at startup, by stripping
// each migration down to its Up section. Running the real files keeps the test
// honest about the schema the DAO is written against.
func applyMigrations(db *sql.DB) error {
	paths, err := filepath.Glob(filepath.Join("..", "..", "sql_migrations", "main", "*.sql"))
	if err != nil {
		return fmt.Errorf("finding migration files: %w", err)
	}
	if len(paths) == 0 {
		return errors.New("no migration files found")
	}

	for _, path := range paths {
		raw, err := os.ReadFile(path)
		if err != nil {
			return fmt.Errorf("reading %s: %w", path, err)
		}

		up := string(raw)
		if idx := strings.Index(up, "-- +migrate Down"); idx >= 0 {
			up = up[:idx]
		}

		if _, err := db.Exec(up); err != nil {
			return fmt.Errorf("applying %s: %w", path, err)
		}
	}

	return nil
}
