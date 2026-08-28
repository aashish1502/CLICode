package loader

import (
	"errors"
	"testing"
)

// Every problem in the index must be loadable and valid. This is the guard that
// catches a seed file being added without an index entry, or vice versa.
func TestEverySeedProblemLoads(t *testing.T) {
	list, err := MakeProblemList()
	if err != nil {
		t.Fatalf("MakeProblemList: %v", err)
	}
	if len(list) == 0 {
		t.Fatal("problem list is empty")
	}

	for _, item := range list {
		p, err := LoadProblem(item.Id)
		if err != nil {
			t.Errorf("problem %d (%s): %v", item.Id, item.Title, err)
			continue
		}
		if p.ID != item.Id {
			t.Errorf("problem %d: file reports id %d", item.Id, p.ID)
		}
		if len(p.AvailableLanguages()) == 0 {
			t.Errorf("problem %d (%s): no code stubs", item.Id, item.Title)
		}
	}
}

func TestLoadProblemMissing(t *testing.T) {
	_, err := LoadProblem(999999)

	var notFound *ProblemNotFoundError
	if !errors.As(err, &notFound) {
		t.Fatalf("expected *ProblemNotFoundError, got %T (%v)", err, err)
	}
}

func TestLoadProblemRejectsInvalidID(t *testing.T) {
	for _, id := range []int{0, -1} {
		if _, err := LoadProblem(id); err == nil {
			t.Errorf("id %d: expected an error", id)
		}
	}
}

// The seed set is compiled into the binary, so loading must not depend on the
// working directory. Before it was embedded, this failed outside the repo root.
func TestLoadsFromAnyWorkingDirectory(t *testing.T) {
	t.Chdir(t.TempDir())

	list, err := MakeProblemList()
	if err != nil {
		t.Fatalf("MakeProblemList outside the repo root: %v", err)
	}
	if len(list) == 0 {
		t.Fatal("problem list is empty outside the repo root")
	}
	if _, err := LoadProblem(list[0].Id); err != nil {
		t.Fatalf("LoadProblem outside the repo root: %v", err)
	}
}
