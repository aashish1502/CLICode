package store

import (
	"database/sql"
	"embed"
	"fmt"
	"io/fs"
	"sort"
)

//go:embed migrations/*.sql
var migrationFiles embed.FS

// migration is one numbered step. The number is positional, not parsed from
// the filename: the Nth file in sorted order is version N.
type migration struct {
	name string
	body string
}

// loadMigrations reads the embedded .sql files in filename order. Names are
// zero-padded (0001_, 0002_) so lexical order is numeric order.
func loadMigrations() ([]migration, error) {
	entries, err := fs.ReadDir(migrationFiles, "migrations")
	if err != nil {
		return nil, fmt.Errorf("reading migrations: %w", err)
	}

	names := make([]string, 0, len(entries))
	for _, e := range entries {
		if !e.IsDir() {
			names = append(names, e.Name())
		}
	}
	sort.Strings(names)

	out := make([]migration, 0, len(names))
	for _, n := range names {
		body, err := fs.ReadFile(migrationFiles, "migrations/"+n)
		if err != nil {
			return nil, fmt.Errorf("reading migration %s: %w", n, err)
		}
		out = append(out, migration{name: n, body: string(body)})
	}
	return out, nil
}

// migrate brings db up to the latest schema version.
//
// The version lives in SQLite's built-in user_version pragma rather than a
// table of our own, so there is no bootstrapping problem: a brand new database
// reports 0 without anything having to exist first.
//
// Each migration runs inside a transaction together with its version bump, so
// a failure part-way leaves the database at the previous version rather than
// half-upgraded.
func migrate(db *sql.DB) error {
	migrations, err := loadMigrations()
	if err != nil {
		return err
	}

	var version int
	if err := db.QueryRow("PRAGMA user_version").Scan(&version); err != nil {
		return fmt.Errorf("reading schema version: %w", err)
	}

	if version > len(migrations) {
		return fmt.Errorf(
			"database is at schema version %d but this build only knows %d: "+
				"it was written by a newer clicode", version, len(migrations))
	}

	for i := version; i < len(migrations); i++ {
		m := migrations[i]
		if err := applyMigration(db, m, i+1); err != nil {
			return fmt.Errorf("applying %s: %w", m.name, err)
		}
	}
	return nil
}

func applyMigration(db *sql.DB, m migration, version int) error {
	tx, err := db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback() //nolint:errcheck // no-op once Commit has succeeded

	if _, err := tx.Exec(m.body); err != nil {
		return err
	}
	// PRAGMA does not accept bound parameters, so the value is interpolated.
	// It is an int we computed, never anything user-supplied.
	if _, err := tx.Exec(fmt.Sprintf("PRAGMA user_version = %d", version)); err != nil {
		return err
	}
	return tx.Commit()
}
