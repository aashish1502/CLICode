package store

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/aashish1502/clicode/internal/models"
)

// ErrNotFound is returned when a problem id is not in the catalog.
var ErrNotFound = errors.New("problem not found")

// PutProblem writes a problem and all of its content, replacing whatever was
// there before.
//
// The child rows are deleted and reinserted rather than diffed: an example
// removed upstream has to disappear locally, and reinserting a handful of rows
// is cheaper than working out which ones changed. Note what is NOT in this
// function -- solutions, progress, submissions. A content refresh cannot reach
// them, which is the guarantee that makes re-fetching safe.
func (d *DB) PutProblem(ctx context.Context, p *models.Problem) error {
	if p == nil {
		return errors.New("store: nil problem")
	}
	return d.tx(ctx, func(tx *sql.Tx) error {
		// url is preserved: it arrives from the list payload, not this one, and
		// the two are written independently.
		_, err := tx.ExecContext(ctx, `
			INSERT INTO problems (id, title, platform, difficulty, description, fetched_at)
			VALUES (?, ?, ?, ?, ?, ?)
			ON CONFLICT(id) DO UPDATE SET
				title       = excluded.title,
				platform    = excluded.platform,
				difficulty  = excluded.difficulty,
				description = excluded.description,
				fetched_at  = excluded.fetched_at`,
			p.ID, p.Title, p.Platform, p.Difficulty, p.Description, time.Now().Unix())
		if err != nil {
			return fmt.Errorf("writing problem %d: %w", p.ID, err)
		}

		for _, table := range []string{
			"problem_tags", "examples", "problem_constraints", "test_cases", "code_stubs",
		} {
			if _, err := tx.ExecContext(ctx,
				"DELETE FROM "+table+" WHERE problem_id = ?", p.ID); err != nil {
				return fmt.Errorf("clearing %s for problem %d: %w", table, p.ID, err)
			}
		}

		for i, tag := range p.Tags {
			if _, err := tx.ExecContext(ctx,
				"INSERT INTO problem_tags (problem_id, ord, tag) VALUES (?, ?, ?)",
				p.ID, i, tag); err != nil {
				return fmt.Errorf("writing tag %d for problem %d: %w", i, p.ID, err)
			}
		}
		for i, e := range p.Examples {
			if _, err := tx.ExecContext(ctx, `
				INSERT INTO examples (problem_id, ord, input, output, explanation)
				VALUES (?, ?, ?, ?, ?)`,
				p.ID, i, e.Input, e.Output, e.Explanation); err != nil {
				return fmt.Errorf("writing example %d for problem %d: %w", i, p.ID, err)
			}
		}
		for i, c := range p.Constraints {
			if _, err := tx.ExecContext(ctx,
				"INSERT INTO problem_constraints (problem_id, ord, body) VALUES (?, ?, ?)",
				p.ID, i, c); err != nil {
				return fmt.Errorf("writing constraint %d for problem %d: %w", i, p.ID, err)
			}
		}
		for i, tc := range p.TestCases {
			if _, err := tx.ExecContext(ctx, `
				INSERT INTO test_cases (problem_id, ord, input, expected_output)
				VALUES (?, ?, ?, ?)`,
				p.ID, i, tc.Input, tc.ExpectedOutput); err != nil {
				return fmt.Errorf("writing test case %d for problem %d: %w", i, p.ID, err)
			}
		}
		for lang, stub := range p.CodeStubs {
			if _, err := tx.ExecContext(ctx,
				"INSERT INTO code_stubs (problem_id, language, stub) VALUES (?, ?, ?)",
				p.ID, lang, stub); err != nil {
				return fmt.Errorf("writing %s stub for problem %d: %w", lang, p.ID, err)
			}
		}

		for _, s := range p.Submissions {
			if _, err := tx.ExecContext(ctx, `
				INSERT INTO submissions (id, problem_id, language, status, runtime, memory, timestamp)
				VALUES (?, ?, ?, ?, ?, ?, ?)
				ON CONFLICT(id) DO NOTHING`,
				s.ID, p.ID, s.Language, s.Status, s.Runtime, s.Memory, s.Timestamp); err != nil {
				return fmt.Errorf("writing submission %s: %w", s.ID, err)
			}
		}
		return nil
	})
}

// PutListItem writes the fields that only the list payload carries.
//
// The progress half of a list item (solved, attempts, notes...) is written
// only when no progress row exists yet, so seed values show up on a fresh
// install but never overwrite work the user has actually done.
func (d *DB) PutListItem(ctx context.Context, item models.ProblemListItem) error {
	return d.tx(ctx, func(tx *sql.Tx) error {
		_, err := tx.ExecContext(ctx, `
			INSERT INTO problems (id, title, platform, difficulty, url)
			VALUES (?, ?, ?, ?, ?)
			ON CONFLICT(id) DO UPDATE SET
				title      = excluded.title,
				platform   = excluded.platform,
				difficulty = excluded.difficulty,
				url        = excluded.url`,
			item.Id, item.Title, item.Platform, item.Difficulty, item.Url)
		if err != nil {
			return fmt.Errorf("writing list item %d: %w", item.Id, err)
		}

		if _, err := tx.ExecContext(ctx,
			"DELETE FROM problem_tags WHERE problem_id = ?", item.Id); err != nil {
			return err
		}
		for i, tag := range item.Tags {
			if _, err := tx.ExecContext(ctx,
				"INSERT INTO problem_tags (problem_id, ord, tag) VALUES (?, ?, ?)",
				item.Id, i, tag); err != nil {
				return err
			}
		}

		// DO NOTHING, not DO UPDATE -- see the doc comment.
		_, err = tx.ExecContext(ctx, `
			INSERT INTO progress
				(problem_id, solved, review, attempts, time_taken_minutes, date_solved, notes)
			VALUES (?, ?, ?, ?, ?, ?, ?)
			ON CONFLICT(problem_id) DO NOTHING`,
			item.Id, boolToInt(item.Solved), boolToInt(item.Review), item.Attempts,
			item.TimeTakenMinutes, item.DateSolved, item.Notes)
		if err != nil {
			return fmt.Errorf("writing seed progress for %d: %w", item.Id, err)
		}
		return nil
	})
}

// List returns every problem with its progress, ordered by id.
//
// This reads six columns and a tag join -- not whole problem documents. It is
// what replaces data/problems/problems_list.json, the denormalized index file
// that only existed because JSON cannot be read a field at a time.
func (d *DB) List(ctx context.Context) ([]models.ProblemListItem, error) {
	rows, err := d.db.QueryContext(ctx, `
		SELECT p.id, p.title, p.difficulty, p.platform, p.url,
		       COALESCE(g.solved, 0), COALESCE(g.attempts, 0),
		       COALESCE(g.time_taken_minutes, 0), COALESCE(g.date_solved, ''),
		       COALESCE(g.review, 0), COALESCE(g.notes, '')
		FROM problems p
		LEFT JOIN progress g ON g.problem_id = p.id
		ORDER BY p.id`)
	if err != nil {
		return nil, fmt.Errorf("listing problems: %w", err)
	}
	defer rows.Close()

	var items []models.ProblemListItem
	for rows.Next() {
		var it models.ProblemListItem
		var solved, review int
		if err := rows.Scan(&it.Id, &it.Title, &it.Difficulty, &it.Platform, &it.Url,
			&solved, &it.Attempts, &it.TimeTakenMinutes, &it.DateSolved,
			&review, &it.Notes); err != nil {
			return nil, fmt.Errorf("scanning problem list: %w", err)
		}
		it.Solved = solved != 0
		it.Review = review != 0
		items = append(items, it)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	for i := range items {
		tags, err := d.tags(ctx, items[i].Id)
		if err != nil {
			return nil, err
		}
		items[i].Tags = tags
	}
	return items, nil
}

// Problem reads one problem and everything hanging off it.
func (d *DB) Problem(ctx context.Context, id int) (*models.Problem, error) {
	p := &models.Problem{CodeStubs: map[string]string{}}

	err := d.db.QueryRowContext(ctx,
		"SELECT id, title, platform, difficulty, description FROM problems WHERE id = ?", id).
		Scan(&p.ID, &p.Title, &p.Platform, &p.Difficulty, &p.Description)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, fmt.Errorf("%w: %d", ErrNotFound, id)
	}
	if err != nil {
		return nil, fmt.Errorf("reading problem %d: %w", id, err)
	}

	if p.Tags, err = d.tags(ctx, id); err != nil {
		return nil, err
	}

	if err := d.each(ctx,
		"SELECT input, output, explanation FROM examples WHERE problem_id = ? ORDER BY ord", id,
		func(rows *sql.Rows) error {
			var e models.Example
			if err := rows.Scan(&e.Input, &e.Output, &e.Explanation); err != nil {
				return err
			}
			p.Examples = append(p.Examples, e)
			return nil
		}); err != nil {
		return nil, err
	}

	if err := d.each(ctx,
		"SELECT body FROM problem_constraints WHERE problem_id = ? ORDER BY ord", id,
		func(rows *sql.Rows) error {
			var c string
			if err := rows.Scan(&c); err != nil {
				return err
			}
			p.Constraints = append(p.Constraints, c)
			return nil
		}); err != nil {
		return nil, err
	}

	if err := d.each(ctx,
		"SELECT input, expected_output FROM test_cases WHERE problem_id = ? ORDER BY ord", id,
		func(rows *sql.Rows) error {
			var tc models.TestCase
			if err := rows.Scan(&tc.Input, &tc.ExpectedOutput); err != nil {
				return err
			}
			p.TestCases = append(p.TestCases, tc)
			return nil
		}); err != nil {
		return nil, err
	}

	if err := d.each(ctx,
		"SELECT language, stub FROM code_stubs WHERE problem_id = ? ORDER BY language", id,
		func(rows *sql.Rows) error {
			var lang, stub string
			if err := rows.Scan(&lang, &stub); err != nil {
				return err
			}
			p.CodeStubs[lang] = stub
			return nil
		}); err != nil {
		return nil, err
	}

	if err := d.each(ctx, `
		SELECT id, status, language, timestamp, runtime, memory
		FROM submissions WHERE problem_id = ? ORDER BY rowid`, id,
		func(rows *sql.Rows) error {
			var s models.Submission
			if err := rows.Scan(&s.ID, &s.Status, &s.Language, &s.Timestamp,
				&s.Runtime, &s.Memory); err != nil {
				return err
			}
			p.Submissions = append(p.Submissions, s)
			return nil
		}); err != nil {
		return nil, err
	}

	return p, nil
}

// Count reports how many problems are in the catalog. Seeding uses it to
// decide whether there is anything to do.
func (d *DB) Count(ctx context.Context) (int, error) {
	var n int
	err := d.db.QueryRowContext(ctx, "SELECT COUNT(*) FROM problems").Scan(&n)
	return n, err
}

func (d *DB) tags(ctx context.Context, id int) ([]string, error) {
	var tags []string
	err := d.each(ctx,
		"SELECT tag FROM problem_tags WHERE problem_id = ? ORDER BY ord", id,
		func(rows *sql.Rows) error {
			var t string
			if err := rows.Scan(&t); err != nil {
				return err
			}
			tags = append(tags, t)
			return nil
		})
	return tags, err
}

// each runs a single-argument query and calls scan for every row.
func (d *DB) each(ctx context.Context, query string, arg any, scan func(*sql.Rows) error) error {
	rows, err := d.db.QueryContext(ctx, query, arg)
	if err != nil {
		return err
	}
	defer rows.Close()
	for rows.Next() {
		if err := scan(rows); err != nil {
			return err
		}
	}
	return rows.Err()
}

func boolToInt(b bool) int {
	if b {
		return 1
	}
	return 0
}
