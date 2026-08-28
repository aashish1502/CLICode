package loader

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"

	"github.com/aashish1502/clicode/data"
	"github.com/aashish1502/clicode/internal/models"
)

// ProblemNotFoundError indicates a problem file doesn't exist
type ProblemNotFoundError struct {
	ID int
}

func (e *ProblemNotFoundError) Error() string {
	return fmt.Sprintf("problem %d not found", e.ID)
}

// InvalidProblemDataError indicates problem data failed validation
type InvalidProblemDataError struct {
	ID     int
	Reason string
}

func (e *InvalidProblemDataError) Error() string {
	return fmt.Sprintf("invalid problem data for %d: %s", e.ID, e.Reason)
}

// LoadProblem reads one problem from the embedded seed set.
func LoadProblem(id int) (*models.Problem, error) {
	if id <= 0 {
		return nil, fmt.Errorf("invalid problem ID: %d", id)
	}

	filename := fmt.Sprintf("%d.json", id)

	raw, err := fs.ReadFile(data.Problems(), filename)
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			// we use & to get the mem address of an object here we are creating the error obj
			// and passing its address to the caller
			return nil, &ProblemNotFoundError{ID: id}
		}
		return nil, fmt.Errorf("failed to read problem file %s: %w", filename, err)
	}

	var problem models.Problem
	if err := json.Unmarshal(raw, &problem); err != nil {
		return nil, fmt.Errorf("failed to parse problem JSON for %d: %w", id, err)
	}

	if err := problem.ValidateProblem(); err != nil {
		return nil, &InvalidProblemDataError{ID: id, Reason: err.Error()}
	}

	return &problem, nil
}

// MakeProblemList reads the problem index from the embedded seed set.
func MakeProblemList() ([]models.ProblemListItem, error) {
	raw, err := fs.ReadFile(data.Problems(), "problems_list.json")
	if err != nil {
		return nil, fmt.Errorf("failed to read problem list: %w", err)
	}

	var problems []models.ProblemListItem
	if err := json.Unmarshal(raw, &problems); err != nil {
		return nil, fmt.Errorf("failed to parse problem list: %w", err)
	}

	return problems, nil
}
