// Package catalog is the only data API the screens see.
//
// It decides where a problem comes from -- the local SQLite store today, a
// server later -- so no screen ever learns that a network exists. That is what
// lets the client ship and run before the API does.
//
// It also guarantees the app starts. If the database cannot be opened at all,
// Open falls back to a read-only catalog served straight from the embedded seed
// JSON: problems still list and still open, saving is the only thing lost. A
// broken database is never a reason to refuse to launch.
package catalog

import (
	"context"
	"log"

	"github.com/aashish1502/clicode/data"
	"github.com/aashish1502/clicode/internal/loader"
	"github.com/aashish1502/clicode/internal/models"
	"github.com/aashish1502/clicode/internal/store"
)

// Catalog is the read/write surface the router calls.
type Catalog interface {
	List(ctx context.Context) ([]models.ProblemListItem, error)
	Problem(ctx context.Context, id int) (*models.Problem, error)

	// Solutions returns every saved buffer for a problem, keyed by language.
	Solutions(ctx context.Context, problemID int) (map[string]string, error)
	SaveSolution(ctx context.Context, problemID int, language, code string) error

	// LastLanguage and LastOpened are what replaced session.json.
	LastLanguage(ctx context.Context, problemID int) (string, error)
	LastOpened(ctx context.Context) (int, error)
	MarkOpened(ctx context.Context, problemID int, language string) error

	// Writable reports whether saving actually persists. False on the seed
	// fallback, so the UI can say so instead of silently dropping writes.
	Writable() bool

	Close() error
}

// Open returns the catalog the app should use.
//
// It never returns an error: a database that will not open degrades to the
// read-only seed rather than stopping the app. The reason is logged.
func Open() Catalog {
	path, err := store.DefaultPath()
	if err != nil {
		log.Printf("catalog: no home directory (%v); running from the embedded seed, saves will not persist", err)
		return seedOnly{}
	}
	return openAt(path)
}

func openAt(path string) Catalog {
	db, err := store.Open(path)
	if err != nil {
		log.Printf("catalog: cannot open %s (%v); running from the embedded seed, saves will not persist", path, err)
		return seedOnly{}
	}
	if err := db.Seed(context.Background(), data.Problems()); err != nil {
		// Seeding failed but the database is open. Keep it: an empty catalog
		// is recoverable on the next launch, and any solutions already saved
		// are still reachable.
		log.Printf("catalog: seeding failed: %v", err)
	}
	return &local{db: db}
}

// local is the normal case: everything served from SQLite.
type local struct {
	db *store.DB
}

func (l *local) List(ctx context.Context) ([]models.ProblemListItem, error) {
	return l.db.List(ctx)
}

func (l *local) Problem(ctx context.Context, id int) (*models.Problem, error) {
	return l.db.Problem(ctx, id)
}

func (l *local) Solutions(ctx context.Context, problemID int) (map[string]string, error) {
	return l.db.Solutions(ctx, problemID)
}

func (l *local) SaveSolution(ctx context.Context, problemID int, language, code string) error {
	return l.db.SaveSolution(ctx, problemID, language, code)
}

func (l *local) LastLanguage(ctx context.Context, problemID int) (string, error) {
	return l.db.LastLanguage(ctx, problemID)
}

func (l *local) LastOpened(ctx context.Context) (int, error) {
	return l.db.LastOpened(ctx)
}

func (l *local) MarkOpened(ctx context.Context, problemID int, language string) error {
	return l.db.MarkOpened(ctx, problemID, language)
}

func (l *local) Writable() bool { return true }

func (l *local) Close() error { return l.db.Close() }

// seedOnly serves the embedded JSON directly, with no persistence.
//
// This is the degraded path -- read-only disk, no home directory, a corrupt
// database file. The app is fully usable for reading and solving; only saving
// is lost, and Writable() reports that so the UI can say so.
type seedOnly struct{}

func (seedOnly) List(context.Context) ([]models.ProblemListItem, error) {
	return loader.MakeProblemList()
}

func (seedOnly) Problem(_ context.Context, id int) (*models.Problem, error) {
	return loader.LoadProblem(id)
}

func (seedOnly) Solutions(context.Context, int) (map[string]string, error) {
	return map[string]string{}, nil
}

func (seedOnly) SaveSolution(context.Context, int, string, string) error {
	return ErrReadOnly
}

func (seedOnly) LastLanguage(context.Context, int) (string, error) { return "", nil }

func (seedOnly) LastOpened(context.Context) (int, error) { return 0, nil }

func (seedOnly) MarkOpened(context.Context, int, string) error { return nil }

func (seedOnly) Writable() bool { return false }

func (seedOnly) Close() error { return nil }
