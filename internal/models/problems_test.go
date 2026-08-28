package models

import (
	"encoding/json"
	"errors"
	"reflect"
	"strings"
	"testing"
)

// complete returns a problem that passes validation, for tests to mutate.
func complete() Problem {
	return Problem{
		ID:          1,
		Title:       "Two Sum",
		Platform:    "leetcode",
		Difficulty:  "Easy",
		Description: "Find two numbers that add up to target.",
		Examples:    []Example{{Input: "[2,7], 9", Output: "[0,1]"}},
		Constraints: []string{"2 <= nums.length <= 10^4"},
		TestCases:   []TestCase{{Input: "[2,7], 9", ExpectedOutput: "[0,1]"}},
		CodeStubs:   map[string]string{"python": "def two_sum(): pass"},
	}
}

// The seed files spell this "explanation"; the struct tag used to spell it
// "explanition", so every explanation silently unmarshalled empty.
func TestExampleUnmarshalsExplanation(t *testing.T) {
	const payload = `{"input":"a","output":"b","explanation":"because"}`

	var got Example
	if err := json.Unmarshal([]byte(payload), &got); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}

	want := Example{Input: "a", Output: "b", Explanation: "because"}
	if got != want {
		t.Errorf("got %+v, want %+v", got, want)
	}
}

func TestFormatIncludesExplanation(t *testing.T) {
	p := complete()
	p.Examples[0].Explanation = "2 + 7 = 9"

	if out := p.Format(); !strings.Contains(out, "2 + 7 = 9") {
		t.Errorf("Format() dropped the explanation:\n%s", out)
	}
}

// Format is deliberately total: an incomplete problem still renders whatever it
// has, so a validation failure can never blank the description pane.
func TestFormatRendersIncompleteProblem(t *testing.T) {
	p := Problem{ID: 7, Title: "Partial", Description: "Only some fields."}

	out := p.Format()
	for _, want := range []string{"Partial", "Only some fields."} {
		if !strings.Contains(out, want) {
			t.Errorf("Format() missing %q:\n%s", want, out)
		}
	}
}

func TestValidateProblem(t *testing.T) {
	tests := []struct {
		name    string
		mutate  func(*Problem)
		missing []string
	}{
		{"complete", func(*Problem) {}, nil},
		{"no id", func(p *Problem) { p.ID = 0 }, []string{"id"}},
		{"no title", func(p *Problem) { p.Title = "" }, []string{"title"}},
		{"no description", func(p *Problem) { p.Description = "" }, []string{"description"}},
		{"no examples", func(p *Problem) { p.Examples = nil }, []string{"examples"}},
		{"no constraints", func(p *Problem) { p.Constraints = nil }, []string{"constraints"}},
		{"no test cases", func(p *Problem) { p.TestCases = nil }, []string{"testCases"}},
		{
			"several missing",
			func(p *Problem) { p.Title, p.Examples, p.TestCases = "", nil, nil },
			[]string{"title", "examples", "testCases"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := complete()
			tt.mutate(&p)

			err := p.ValidateProblem()

			if tt.missing == nil {
				if err != nil {
					t.Fatalf("expected valid, got %v", err)
				}
				return
			}

			var ve *ValidationError
			if !errors.As(err, &ve) {
				t.Fatalf("expected *ValidationError, got %T (%v)", err, err)
			}
			if !reflect.DeepEqual(ve.Missing, tt.missing) {
				t.Errorf("missing = %v, want %v", ve.Missing, tt.missing)
			}
		})
	}
}

// The error reaches the UI, so it should name the problem and what's missing —
// and carry none of the stack detail that belongs in the log.
func TestValidationErrorMessage(t *testing.T) {
	err := &ValidationError{ID: 42, Missing: []string{"title", "examples"}}

	got := err.Error()
	for _, want := range []string{"42", "title", "examples"} {
		if !strings.Contains(got, want) {
			t.Errorf("Error() = %q, missing %q", got, want)
		}
	}
	if strings.Contains(got, "clicode/internal") {
		t.Errorf("Error() leaked stack detail into the UI message: %q", got)
	}
}

func TestGetCodeStub(t *testing.T) {
	p := complete()
	p.CodeStubs = map[string]string{
		"python": "def solve(): pass",
		"go":     "func solve() {}",
	}

	t.Run("language present", func(t *testing.T) {
		stub, ok := p.GetCodeStub("go")
		if !ok {
			t.Fatal("expected a stub for go")
		}
		if stub != "func solve() {}" {
			t.Errorf("got %q", stub)
		}
	})

	t.Run("language absent", func(t *testing.T) {
		stub, ok := p.GetCodeStub("cpp")
		if ok {
			t.Errorf("expected no stub for cpp, got %q", stub)
		}
	})
}

// The UI cycles through this list, so it must contain only languages the
// problem actually ships, in a stable order.
func TestAvailableLanguages(t *testing.T) {
	tests := []struct {
		name  string
		stubs map[string]string
		want  []string
	}{
		{"sorted", map[string]string{"python": "x", "cpp": "y", "go": "z"}, []string{"cpp", "go", "python"}},
		{"partial", map[string]string{"python": "x"}, []string{"python"}},
		{"none", map[string]string{}, []string{}},
		{"nil map", nil, []string{}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := complete()
			p.CodeStubs = tt.stubs

			if got := p.AvailableLanguages(); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("got %v, want %v", got, tt.want)
			}
		})
	}
}

func TestFormatTestCaseOutOfRange(t *testing.T) {
	p := complete()

	for _, idx := range []int{-1, len(p.TestCases)} {
		if got := p.FormatTestCase(idx); got != "No test case available." {
			t.Errorf("index %d: got %q", idx, got)
		}
	}
}
