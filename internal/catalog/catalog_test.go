package catalog

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"
)

func TestOpenSeedsAndServesFromTheDatabase(t *testing.T) {
	ctx := context.Background()
	c := openAt(filepath.Join(t.TempDir(), "clicode.db"))
	defer c.Close()

	if !c.Writable() {
		t.Fatal("a healthy database reported itself read-only")
	}
	items, err := c.List(ctx)
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if len(items) == 0 {
		t.Error("catalog is empty; Open did not seed")
	}
}

// The degraded path: the app must still start and still open problems when the
// database cannot be created at all.
func TestAnUnopenableDatabaseFallsBackToTheSeed(t *testing.T) {
	ctx := context.Background()

	// A path whose parent is a file, not a directory -- MkdirAll cannot
	// succeed here.
	blocked := filepath.Join(t.TempDir(), "notadir")
	if err := writeFile(blocked); err != nil {
		t.Fatalf("setting up: %v", err)
	}

	c := openAt(filepath.Join(blocked, "clicode.db"))
	defer c.Close()

	if c.Writable() {
		t.Error("the fallback catalog claims to be writable")
	}
	items, err := c.List(ctx)
	if err != nil {
		t.Fatalf("List on the fallback: %v", err)
	}
	if len(items) == 0 {
		t.Error("the fallback catalog served no problems")
	}
	if _, err := c.Problem(ctx, items[0].Id); err != nil {
		t.Errorf("the fallback could not open a problem: %v", err)
	}
}

// Saving against the fallback must fail loudly. A silent no-op would look
// exactly like a successful write to the user.
func TestSavingOnTheFallbackReportsAnError(t *testing.T) {
	err := seedOnly{}.SaveSolution(context.Background(), 1, "go", "code")
	if !errors.Is(err, ErrReadOnly) {
		t.Errorf("err = %v, want ErrReadOnly", err)
	}
}

func TestSolutionsSurviveReopening(t *testing.T) {
	ctx := context.Background()
	path := filepath.Join(t.TempDir(), "clicode.db")

	first := openAt(path)
	if err := first.SaveSolution(ctx, 1, "go", "func main() {}"); err != nil {
		t.Fatalf("SaveSolution: %v", err)
	}
	if err := first.MarkOpened(ctx, 1, "go"); err != nil {
		t.Fatalf("MarkOpened: %v", err)
	}
	first.Close()

	second := openAt(path)
	defer second.Close()

	saved, err := second.Solutions(ctx, 1)
	if err != nil {
		t.Fatalf("Solutions: %v", err)
	}
	if saved["go"] != "func main() {}" {
		t.Errorf("solutions after reopen = %v, want the saved buffer", saved)
	}
	if last, _ := second.LastOpened(ctx); last != 1 {
		t.Errorf("LastOpened after reopen = %d, want 1", last)
	}
	if lang, _ := second.LastLanguage(ctx, 1); lang != "go" {
		t.Errorf("LastLanguage after reopen = %q, want go", lang)
	}
}

// Reopening must not wipe the catalog and re-seed over saved work.
func TestReopeningDoesNotReseed(t *testing.T) {
	ctx := context.Background()
	path := filepath.Join(t.TempDir(), "clicode.db")

	first := openAt(path)
	before, err := first.List(ctx)
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	first.Close()

	second := openAt(path)
	defer second.Close()
	after, err := second.List(ctx)
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if len(after) != len(before) {
		t.Errorf("catalog went from %d to %d problems across a restart", len(before), len(after))
	}
}

func writeFile(path string) error {
	return os.WriteFile(path, []byte("not a directory"), 0600)
}
