package store

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"
)

// Solution returns the code the user last saved for a problem in a language.
// The bool reports whether anything was saved -- distinguishing "never opened"
// from "saved an empty buffer", which the caller needs in order to decide
// between restoring a solution and falling back to the code stub.
func (d *DB) Solution(ctx context.Context, problemID int, language string) (string, bool, error) {
	var code string
	err := d.db.QueryRowContext(ctx,
		"SELECT code FROM solutions WHERE problem_id = ? AND language = ?",
		problemID, language).Scan(&code)
	if errors.Is(err, sql.ErrNoRows) {
		return "", false, nil
	}
	if err != nil {
		return "", false, fmt.Errorf("reading solution for problem %d (%s): %w",
			problemID, language, err)
	}
	return code, true, nil
}

// SaveSolution persists an editor buffer. This is what makes :w real.
//
// The whole write is one statement, so it either lands or it does not. There
// is no window where the file on disk holds half a solution -- which is the
// failure mode a rewrite-the-JSON-file approach could not avoid.
func (d *DB) SaveSolution(ctx context.Context, problemID int, language, code string) error {
	if language == "" {
		return errors.New("store: cannot save a solution with no language")
	}
	_, err := d.db.ExecContext(ctx, `
		INSERT INTO solutions (problem_id, language, code, updated_at)
		VALUES (?, ?, ?, ?)
		ON CONFLICT(problem_id, language) DO UPDATE SET
			code       = excluded.code,
			updated_at = excluded.updated_at`,
		problemID, language, code, time.Now().Unix())
	if err != nil {
		return fmt.Errorf("saving solution for problem %d (%s): %w", problemID, language, err)
	}
	return nil
}

// MarkOpened records that the user opened a problem, and in which language.
// LastOpened reads this back; together they replace session.json.
//
// Pass an empty language to leave the recorded one alone -- opening a problem
// before a language has been chosen should not blank the previous choice.
func (d *DB) MarkOpened(ctx context.Context, problemID int, language string) error {
	_, err := d.db.ExecContext(ctx, `
		INSERT INTO progress (problem_id, last_language, last_opened_at)
		VALUES (?, ?, ?)
		ON CONFLICT(problem_id) DO UPDATE SET
			last_opened_at = excluded.last_opened_at,
			last_language  = CASE WHEN excluded.last_language = ''
			                      THEN progress.last_language
			                      ELSE excluded.last_language END`,
		problemID, language, time.Now().Unix())
	if err != nil {
		return fmt.Errorf("marking problem %d opened: %w", problemID, err)
	}
	return nil
}

// LastOpened returns the most recently opened problem id, or 0 if the user has
// never opened one.
//
// This is derived, not stored. session.json kept lastProblemID as a separate
// fact that could drift out of step with the rest of the progress data; here
// there is only one place the answer can come from.
func (d *DB) LastOpened(ctx context.Context) (int, error) {
	var id int
	err := d.db.QueryRowContext(ctx, `
		SELECT problem_id FROM progress
		WHERE last_opened_at > 0
		ORDER BY last_opened_at DESC
		LIMIT 1`).Scan(&id)
	if errors.Is(err, sql.ErrNoRows) {
		return 0, nil
	}
	if err != nil {
		return 0, fmt.Errorf("reading last opened problem: %w", err)
	}
	return id, nil
}

// Solutions returns every saved buffer for a problem, keyed by language.
//
// The problem screen keeps one buffer per language and switches between them,
// so it wants them all at once rather than a query per language switch.
func (d *DB) Solutions(ctx context.Context, problemID int) (map[string]string, error) {
	out := map[string]string{}
	err := d.each(ctx,
		"SELECT language, code FROM solutions WHERE problem_id = ?", problemID,
		func(rows *sql.Rows) error {
			var lang, code string
			if err := rows.Scan(&lang, &code); err != nil {
				return err
			}
			out[lang] = code
			return nil
		})
	if err != nil {
		return nil, fmt.Errorf("reading solutions for problem %d: %w", problemID, err)
	}
	return out, nil
}

// LastLanguage returns the language the editor was last in for a problem, or
// "" if it has never been opened. The caller falls back to its own default.
func (d *DB) LastLanguage(ctx context.Context, problemID int) (string, error) {
	var lang string
	err := d.db.QueryRowContext(ctx,
		"SELECT last_language FROM progress WHERE problem_id = ?", problemID).Scan(&lang)
	if errors.Is(err, sql.ErrNoRows) {
		return "", nil
	}
	if err != nil {
		return "", fmt.Errorf("reading last language for problem %d: %w", problemID, err)
	}
	return lang, nil
}
