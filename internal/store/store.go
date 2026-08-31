// Package store is CLICode's local database: a single SQLite file holding both
// the problem catalog and everything the user has done to it.
//
// The two are kept apart on purpose. Content tables (problems, examples,
// test_cases, code_stubs) are owned by whoever supplies the catalog -- the
// embedded seed today, an API later -- and a refresh replaces them wholesale.
// Local tables (solutions, progress, submissions) are owned by the user and no
// refresh path touches them. See migrations/0001_init.sql.
package store

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"path/filepath"

	"github.com/aashish1502/clicode/internal/models"

	_ "modernc.org/sqlite" // registers the "sqlite" driver
)

// Store is the read/write surface the rest of the app sees. It exists so
// screens can be handed a fake in tests without a database on disk.
type Store interface {
	// Catalog reads.
	List(ctx context.Context) ([]models.ProblemListItem, error)
	Problem(ctx context.Context, id int) (*models.Problem, error)

	// Catalog writes -- used by seeding now, by the API client later.
	PutProblem(ctx context.Context, p *models.Problem) error
	PutListItem(ctx context.Context, item models.ProblemListItem) error
	Count(ctx context.Context) (int, error)

	// Local state.
	Solution(ctx context.Context, problemID int, language string) (string, bool, error)
	Solutions(ctx context.Context, problemID int) (map[string]string, error)
	SaveSolution(ctx context.Context, problemID int, language, code string) error
	LastLanguage(ctx context.Context, problemID int) (string, error)
	MarkOpened(ctx context.Context, problemID int, language string) error
	LastOpened(ctx context.Context) (int, error)

	Close() error
}

// DB is the SQLite-backed Store.
type DB struct {
	db *sql.DB
}

// compile-time check that DB satisfies the interface it is written against.
var _ Store = (*DB)(nil)

// DefaultPath is ~/.clicode/clicode.db -- the same directory config.json and
// the title art already live in.
func DefaultPath() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, ".clicode", "clicode.db"), nil
}

// Open connects to the database at path, creating it if it does not exist, and
// brings the schema up to date. Pass ":memory:" for a throwaway database.
func Open(path string) (*DB, error) {
	if path != ":memory:" {
		if err := os.MkdirAll(filepath.Dir(path), 0700); err != nil {
			return nil, fmt.Errorf("creating data directory: %w", err)
		}
	}

	// foreign_keys is off by default in SQLite and is per-connection, so it has
	// to be requested here or the ON DELETE CASCADEs in the schema are inert.
	// busy_timeout makes a locked database wait rather than fail immediately.
	dsn := fmt.Sprintf(
		"file:%s?_pragma=foreign_keys(1)&_pragma=journal_mode(wal)&_pragma=busy_timeout(5000)",
		path)

	db, err := sql.Open("sqlite", dsn)
	if err != nil {
		return nil, fmt.Errorf("opening database: %w", err)
	}

	// One connection. This is a single-user TUI with no concurrent work to
	// gain from a pool, and it makes "database is locked" structurally
	// impossible. Also required for ":memory:", where each new connection
	// would otherwise get its own empty database.
	db.SetMaxOpenConns(1)

	if err := db.Ping(); err != nil {
		db.Close()
		return nil, fmt.Errorf("connecting to database: %w", err)
	}

	if err := migrate(db); err != nil {
		db.Close()
		return nil, err
	}

	if path != ":memory:" {
		restrict(path)
	}

	return &DB{db: db}, nil
}

// restrict tightens the database files to 0600, matching config.json and the
// rest of ~/.clicode. SQLite creates them itself and does not ask, so the mode
// has to be corrected afterwards. The -wal and -shm sidecars are created by
// journal_mode(wal) and hold the same data, so they get the same treatment.
//
// Failures are ignored: the 0700 directory is what actually keeps other users
// out, and a mode we could not tighten is not a reason to refuse to start.
func restrict(path string) {
	for _, p := range []string{path, path + "-wal", path + "-shm"} {
		if _, err := os.Stat(p); err == nil {
			_ = os.Chmod(p, 0600)
		}
	}
}

func (d *DB) Close() error {
	return d.db.Close()
}

// tx runs fn inside a transaction, rolling back on any error.
func (d *DB) tx(ctx context.Context, fn func(*sql.Tx) error) error {
	t, err := d.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	if err := fn(t); err != nil {
		t.Rollback() //nolint:errcheck // the original error is what matters
		return err
	}
	return t.Commit()
}
