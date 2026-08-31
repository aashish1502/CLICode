package store

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"log"

	"github.com/aashish1502/clicode/internal/models"
)

// Seed populates an empty catalog from a filesystem of seed JSON -- normally
// data.Problems(), the copy embedded in the binary.
//
// It is a no-op once the catalog has anything in it, so it runs on first
// launch and then gets out of the way. Re-seeding on every start would fight
// with catalog refreshes later, and PutProblem would rewrite content rows the
// server had just supplied.
//
// The JSON files stay the readable, diffable source of truth in the repo; this
// is only how they get into the database.
func (d *DB) Seed(ctx context.Context, seed fs.FS) error {
	n, err := d.Count(ctx)
	if err != nil {
		return fmt.Errorf("checking whether the catalog is empty: %w", err)
	}
	if n > 0 {
		return nil
	}
	return d.reseed(ctx, seed)
}

// reseed writes the seed set unconditionally. Seed guards it; tests call it
// directly.
func (d *DB) reseed(ctx context.Context, seed fs.FS) error {
	raw, err := fs.ReadFile(seed, "problems_list.json")
	if err != nil {
		return fmt.Errorf("reading seed problem list: %w", err)
	}
	var items []models.ProblemListItem
	if err := json.Unmarshal(raw, &items); err != nil {
		return fmt.Errorf("parsing seed problem list: %w", err)
	}

	// The list is written first so a problem exists before its detail rows
	// reference it, and so url survives -- only the list payload carries it.
	for _, item := range items {
		if err := d.PutListItem(ctx, item); err != nil {
			return err
		}
	}

	for _, item := range items {
		p, err := readSeedProblem(seed, item.Id)
		if err != nil {
			// One malformed or missing seed file should not stop the app from
			// starting. The problem still lists (the list row was written
			// above); opening it is what will fail, and the log says why.
			log.Printf("seed: skipping problem %d: %v", item.Id, err)
			continue
		}
		if err := d.PutProblem(ctx, p); err != nil {
			return err
		}
	}
	return nil
}

// readSeedProblem parses one <id>.json from the seed set.
//
// Unlike loader.LoadProblem this does not reject an incomplete problem. Seed
// data that is missing a field should still be stored and shown as best it
// can be -- rejecting it here would make a problem vanish from a catalog its
// own index file says exists.
func readSeedProblem(seed fs.FS, id int) (*models.Problem, error) {
	name := fmt.Sprintf("%d.json", id)
	raw, err := fs.ReadFile(seed, name)
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return nil, fmt.Errorf("no seed file %s", name)
		}
		return nil, err
	}
	var p models.Problem
	if err := json.Unmarshal(raw, &p); err != nil {
		return nil, fmt.Errorf("parsing %s: %w", name, err)
	}
	// A seed file with no id of its own still belongs to the id that indexed it.
	if p.ID == 0 {
		p.ID = id
	}
	return &p, nil
}
