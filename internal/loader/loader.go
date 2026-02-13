package loader

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

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

// LoadProblem loads a problem from JSON file
func LoadProblem(id int) (*models.Problem, error) {
	if id <= 0 {
		return nil, fmt.Errorf("invalid problem ID: %d", id)
	}

	filename := filepath.Join("data", "problems", fmt.Sprintf("%d.json", id))

	if _, err := os.Stat(filename); os.IsNotExist(err) {

		// we use & to get the mem address of an object here we are creating the error obj
		// and passing its address to the caller
		return nil, &ProblemNotFoundError{ID: id}
	}

	data, err := os.ReadFile(filename)
	if err != nil {
		return nil, fmt.Errorf("failed to read problem file %s: %w", filename, err)
	}

	var problem models.Problem
	if err := json.Unmarshal(data, &problem); err != nil {
		return nil, fmt.Errorf("failed to parse problem JSON for %d: %w", id, err)
	}

	if err := problem.ValidateProblem(); err != nil {
		return nil, &InvalidProblemDataError{ID: id, Reason: err.Error()}
	}

	return &problem, nil
}
