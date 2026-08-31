package store

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"
	"testing/fstest"

	"github.com/aashish1502/clicode/data"
	"github.com/aashish1502/clicode/internal/models"
)

// open returns a fresh database in a temp directory, closed when the test ends.
func open(t *testing.T) *DB {
	t.Helper()
	db, err := Open(filepath.Join(t.TempDir(), "test.db"))
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	t.Cleanup(func() { db.Close() })
	return db
}

func sampleProblem() *models.Problem {
	return &models.Problem{
		ID:          1,
		Title:       "Two Sum",
		Platform:    "LeetCode",
		Difficulty:  "Easy",
		Description: "Given an array...",
		Tags:        []string{"Array", "Hash Table"},
		Examples: []models.Example{
			{Input: "nums = [2,7]", Output: "[0,1]", Explanation: "2 + 7 == 9"},
			{Input: "nums = [3,3]", Output: "[0,1]"},
		},
		Constraints: []string{"2 <= n", "-10^9 <= x"},
		TestCases: []models.TestCase{
			{Input: "[2,7]\n9", ExpectedOutput: "[0,1]"},
		},
		CodeStubs: map[string]string{"python": "class Solution:", "go": "func twoSum()"},
	}
}

func TestMigrateIsIdempotent(t *testing.T) {
	path := filepath.Join(t.TempDir(), "test.db")

	first, err := Open(path)
	if err != nil {
		t.Fatalf("first Open: %v", err)
	}
	if err := first.PutProblem(context.Background(), sampleProblem()); err != nil {
		t.Fatalf("PutProblem: %v", err)
	}
	first.Close()

	// Reopening must not re-run migration 1 -- CREATE TABLE would fail, and the
	// data written above has to survive.
	second, err := Open(path)
	if err != nil {
		t.Fatalf("reopening a migrated database: %v", err)
	}
	defer second.Close()

	n, err := second.Count(context.Background())
	if err != nil {
		t.Fatalf("Count: %v", err)
	}
	if n != 1 {
		t.Errorf("problem count after reopen = %d, want 1", n)
	}
}

func TestDatabaseFromANewerBuildIsRejected(t *testing.T) {
	db := open(t)
	// Pretend a future clicode wrote this file.
	if _, err := db.db.Exec("PRAGMA user_version = 99"); err != nil {
		t.Fatalf("bumping user_version: %v", err)
	}

	err := migrate(db.db)
	if err == nil {
		t.Fatal("migrate accepted a database from a newer build; want an error")
	}
	// Refusing beats silently running old code against a newer schema.
	if !contains(err.Error(), "newer clicode") {
		t.Errorf("error = %q, want it to mention a newer clicode", err)
	}
}

func TestProblemRoundTrips(t *testing.T) {
	ctx := context.Background()
	db := open(t)
	want := sampleProblem()

	if err := db.PutProblem(ctx, want); err != nil {
		t.Fatalf("PutProblem: %v", err)
	}
	got, err := db.Problem(ctx, 1)
	if err != nil {
		t.Fatalf("Problem: %v", err)
	}

	if got.Title != want.Title || got.Description != want.Description {
		t.Errorf("scalars did not survive: got %q/%q", got.Title, got.Description)
	}
	if len(got.Examples) != 2 {
		t.Fatalf("examples = %d, want 2", len(got.Examples))
	}
	// Ordered child rows must come back in insertion order, which is what the
	// `ord` column exists for.
	if got.Examples[0].Input != "nums = [2,7]" || got.Examples[1].Input != "nums = [3,3]" {
		t.Errorf("examples came back out of order: %+v", got.Examples)
	}
	if got.Examples[0].Explanation != "2 + 7 == 9" {
		t.Errorf("explanation = %q, want it preserved", got.Examples[0].Explanation)
	}
	if len(got.Constraints) != 2 || got.Constraints[0] != "2 <= n" {
		t.Errorf("constraints = %v", got.Constraints)
	}
	if len(got.Tags) != 2 || got.Tags[0] != "Array" {
		t.Errorf("tags = %v, want them in order", got.Tags)
	}
	if got.CodeStubs["python"] != "class Solution:" {
		t.Errorf("code stubs = %v", got.CodeStubs)
	}
}

func TestMissingProblemIsNotFound(t *testing.T) {
	_, err := open(t).Problem(context.Background(), 404)
	if !errors.Is(err, ErrNotFound) {
		t.Errorf("err = %v, want ErrNotFound", err)
	}
}

// The guarantee the whole content/local split exists for.
func TestContentRefreshLeavesLocalStateAlone(t *testing.T) {
	ctx := context.Background()
	db := open(t)

	if err := db.PutProblem(ctx, sampleProblem()); err != nil {
		t.Fatalf("PutProblem: %v", err)
	}
	if err := db.SaveSolution(ctx, 1, "python", "my hard-won solution"); err != nil {
		t.Fatalf("SaveSolution: %v", err)
	}
	if err := db.MarkOpened(ctx, 1, "python"); err != nil {
		t.Fatalf("MarkOpened: %v", err)
	}

	// The server sends a rewritten problem: different title, different
	// examples, a stub removed.
	updated := sampleProblem()
	updated.Title = "Two Sum (revised)"
	updated.Examples = []models.Example{{Input: "changed"}}
	updated.CodeStubs = map[string]string{"python": "new stub"}
	if err := db.PutProblem(ctx, updated); err != nil {
		t.Fatalf("refreshing content: %v", err)
	}

	got, err := db.Problem(ctx, 1)
	if err != nil {
		t.Fatalf("Problem: %v", err)
	}
	if got.Title != "Two Sum (revised)" {
		t.Errorf("content was not refreshed: title = %q", got.Title)
	}
	if len(got.Examples) != 1 {
		t.Errorf("stale examples survived: %d rows, want 1", len(got.Examples))
	}
	if len(got.CodeStubs) != 1 {
		t.Errorf("stale code stubs survived: %v", got.CodeStubs)
	}

	code, ok, err := db.Solution(ctx, 1, "python")
	if err != nil {
		t.Fatalf("Solution: %v", err)
	}
	if !ok || code != "my hard-won solution" {
		t.Errorf("the refresh ate the user's code: got %q (present=%v)", code, ok)
	}
	if last, _ := db.LastOpened(ctx); last != 1 {
		t.Errorf("the refresh cleared progress: LastOpened = %d, want 1", last)
	}
}

func TestSaveSolutionOverwrites(t *testing.T) {
	ctx := context.Background()
	db := open(t)

	if err := db.SaveSolution(ctx, 7, "go", "first"); err != nil {
		t.Fatalf("SaveSolution: %v", err)
	}
	if err := db.SaveSolution(ctx, 7, "go", "second"); err != nil {
		t.Fatalf("SaveSolution (again): %v", err)
	}
	code, _, err := db.Solution(ctx, 7, "go")
	if err != nil {
		t.Fatalf("Solution: %v", err)
	}
	if code != "second" {
		t.Errorf("code = %q, want %q", code, "second")
	}
}

func TestSolutionsAreKeptPerLanguage(t *testing.T) {
	ctx := context.Background()
	db := open(t)

	if err := db.SaveSolution(ctx, 7, "go", "go code"); err != nil {
		t.Fatalf("SaveSolution(go): %v", err)
	}
	if err := db.SaveSolution(ctx, 7, "python", "python code"); err != nil {
		t.Fatalf("SaveSolution(python): %v", err)
	}

	for lang, want := range map[string]string{"go": "go code", "python": "python code"} {
		got, ok, err := db.Solution(ctx, 7, lang)
		if err != nil || !ok {
			t.Fatalf("Solution(%s): %v (present=%v)", lang, err, ok)
		}
		if got != want {
			t.Errorf("Solution(%s) = %q, want %q", lang, got, want)
		}
	}
}

// A saved empty buffer and a never-saved one look identical if you only return
// a string; the caller needs the difference to know whether to load the stub.
func TestSolutionDistinguishesEmptyFromAbsent(t *testing.T) {
	ctx := context.Background()
	db := open(t)

	if _, ok, err := db.Solution(ctx, 7, "go"); err != nil || ok {
		t.Fatalf("unsaved solution reported present=%v (err %v)", ok, err)
	}
	if err := db.SaveSolution(ctx, 7, "go", ""); err != nil {
		t.Fatalf("SaveSolution: %v", err)
	}
	code, ok, err := db.Solution(ctx, 7, "go")
	if err != nil {
		t.Fatalf("Solution: %v", err)
	}
	if !ok || code != "" {
		t.Errorf("saved empty buffer = %q (present=%v), want \"\" and present", code, ok)
	}
}

func TestSaveSolutionRequiresALanguage(t *testing.T) {
	if err := open(t).SaveSolution(context.Background(), 7, "", "code"); err == nil {
		t.Error("SaveSolution accepted an empty language; want an error")
	}
}

func TestLastOpenedIsTheMostRecent(t *testing.T) {
	ctx := context.Background()
	db := open(t)

	if id, err := db.LastOpened(ctx); err != nil || id != 0 {
		t.Fatalf("LastOpened on a fresh database = %d, %v; want 0, nil", id, err)
	}

	for _, id := range []int{3, 9, 5} {
		if err := db.MarkOpened(ctx, id, "go"); err != nil {
			t.Fatalf("MarkOpened(%d): %v", id, err)
		}
		// Timestamps are whole seconds, so nudge the clock by hand rather than
		// sleeping through a real second three times.
		if _, err := db.db.Exec(
			"UPDATE progress SET last_opened_at = ? WHERE problem_id = ?", id*100, id); err != nil {
			t.Fatalf("adjusting timestamp: %v", err)
		}
	}

	got, err := db.LastOpened(ctx)
	if err != nil {
		t.Fatalf("LastOpened: %v", err)
	}
	if got != 9 {
		t.Errorf("LastOpened = %d, want 9 (the highest timestamp)", got)
	}
}

func TestMarkOpenedWithNoLanguageKeepsThePreviousOne(t *testing.T) {
	ctx := context.Background()
	db := open(t)

	if err := db.MarkOpened(ctx, 1, "kotlin"); err != nil {
		t.Fatalf("MarkOpened: %v", err)
	}
	if err := db.MarkOpened(ctx, 1, ""); err != nil {
		t.Fatalf("MarkOpened (no language): %v", err)
	}

	var lang string
	if err := db.db.QueryRow(
		"SELECT last_language FROM progress WHERE problem_id = 1").Scan(&lang); err != nil {
		t.Fatalf("reading last_language: %v", err)
	}
	if lang != "kotlin" {
		t.Errorf("last_language = %q, want it left as kotlin", lang)
	}
}

func TestSeedLoadsTheEmbeddedProblemSet(t *testing.T) {
	ctx := context.Background()
	db := open(t)

	if err := db.Seed(ctx, data.Problems()); err != nil {
		t.Fatalf("Seed: %v", err)
	}

	items, err := db.List(ctx)
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if len(items) == 0 {
		t.Fatal("Seed produced an empty catalog")
	}
	if items[0].Title == "" || items[0].Difficulty == "" {
		t.Errorf("list item is missing content: %+v", items[0])
	}
	if len(items[0].Tags) == 0 {
		t.Errorf("list item has no tags: %+v", items[0])
	}

	// The list payload is the only place url lives, so this proves
	// PutListItem's fields were not clobbered by PutProblem afterwards.
	if items[0].Url == "" {
		t.Errorf("url was lost between PutListItem and PutProblem: %+v", items[0])
	}

	p, err := db.Problem(ctx, items[0].Id)
	if err != nil {
		t.Fatalf("Problem(%d): %v", items[0].Id, err)
	}
	if p.Description == "" || len(p.TestCases) == 0 {
		t.Errorf("seeded problem is missing detail: %+v", p)
	}
}

func TestSeedDoesNothingWhenTheCatalogIsPopulated(t *testing.T) {
	ctx := context.Background()
	db := open(t)

	if err := db.PutProblem(ctx, sampleProblem()); err != nil {
		t.Fatalf("PutProblem: %v", err)
	}
	if err := db.Seed(ctx, data.Problems()); err != nil {
		t.Fatalf("Seed: %v", err)
	}

	n, err := db.Count(ctx)
	if err != nil {
		t.Fatalf("Count: %v", err)
	}
	if n != 1 {
		t.Errorf("Seed ran against a non-empty catalog: count = %d, want 1", n)
	}
}

// Seed progress is a starting value, not an override.
func TestSeedingDoesNotOverwriteRealProgress(t *testing.T) {
	ctx := context.Background()
	db := open(t)

	item := models.ProblemListItem{Id: 1, Title: "Two Sum", Solved: true, Attempts: 1,
		Notes: "seed note"}
	if err := db.PutListItem(ctx, item); err != nil {
		t.Fatalf("PutListItem: %v", err)
	}
	if _, err := db.db.Exec(
		"UPDATE progress SET notes = 'my note', attempts = 12 WHERE problem_id = 1"); err != nil {
		t.Fatalf("recording user progress: %v", err)
	}

	// Same seed data arrives again -- a re-seed, or a catalog refresh.
	if err := db.PutListItem(ctx, item); err != nil {
		t.Fatalf("PutListItem (again): %v", err)
	}

	items, err := db.List(ctx)
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if items[0].Notes != "my note" || items[0].Attempts != 12 {
		t.Errorf("seed data overwrote real progress: notes=%q attempts=%d",
			items[0].Notes, items[0].Attempts)
	}
}

// A missing detail file must not stop the app from starting.
func TestSeedSurvivesAMissingProblemFile(t *testing.T) {
	ctx := context.Background()
	db := open(t)

	seed := fstest.MapFS{
		"problems_list.json": {Data: []byte(
			`[{"id":1,"title":"Present"},{"id":2,"title":"Detail file missing"}]`)},
		"1.json": {Data: []byte(`{"id":1,"title":"Present","description":"here"}`)},
	}

	if err := db.Seed(ctx, seed); err != nil {
		t.Fatalf("Seed: %v", err)
	}
	items, err := db.List(ctx)
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if len(items) != 2 {
		t.Fatalf("catalog = %d problems, want both listed even though one has no detail", len(items))
	}
	if _, err := db.Problem(ctx, 1); err != nil {
		t.Errorf("the intact problem should still open: %v", err)
	}
}

// STRICT tables make the declared type a real constraint. Without it SQLite
// would happily store 'not a number' in an INTEGER column.
func TestStrictTablesRejectTheWrongType(t *testing.T) {
	db := open(t)
	_, err := db.db.Exec(
		"INSERT INTO solutions (problem_id, language, code, updated_at) VALUES ('abc', 'go', '', 0)")
	if err == nil {
		t.Error("a text value was accepted into an INTEGER column; STRICT is not in effect")
	}
}

func TestForeignKeysCascadeContentButNotSolutions(t *testing.T) {
	ctx := context.Background()
	db := open(t)

	if err := db.PutProblem(ctx, sampleProblem()); err != nil {
		t.Fatalf("PutProblem: %v", err)
	}
	if err := db.SaveSolution(ctx, 1, "python", "keep me"); err != nil {
		t.Fatalf("SaveSolution: %v", err)
	}

	// The problem is withdrawn from the catalog entirely.
	if _, err := db.db.Exec("DELETE FROM problems WHERE id = 1"); err != nil {
		t.Fatalf("deleting problem: %v", err)
	}

	var examples int
	if err := db.db.QueryRow("SELECT COUNT(*) FROM examples").Scan(&examples); err != nil {
		t.Fatalf("counting examples: %v", err)
	}
	if examples != 0 {
		t.Errorf("content rows survived the cascade: %d examples left", examples)
	}

	code, ok, err := db.Solution(ctx, 1, "python")
	if err != nil {
		t.Fatalf("Solution: %v", err)
	}
	if !ok || code != "keep me" {
		t.Errorf("the user's code was cascaded away: %q (present=%v)", code, ok)
	}
}

func contains(s, sub string) bool {
	for i := 0; i+len(sub) <= len(s); i++ {
		if s[i:i+len(sub)] == sub {
			return true
		}
	}
	return false
}

// Everything in ~/.clicode is written 0600 (config.json, session.json); the
// database holds the user's solutions and gets the same treatment.
func TestDatabaseFileIsNotWorldReadable(t *testing.T) {
	path := filepath.Join(t.TempDir(), "perm.db")
	db, err := Open(path)
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	defer db.Close()

	info, err := os.Stat(path)
	if err != nil {
		t.Fatalf("Stat: %v", err)
	}
	if mode := info.Mode().Perm(); mode != 0600 {
		t.Errorf("database file mode = %o, want 600", mode)
	}
}
