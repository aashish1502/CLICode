package models

import (
	"fmt"
	"log"
	"runtime"
	"sort"
	"strings"
)

type Example struct {
	Input       string `json:"input"`
	Output      string `json:"output"`
	Explanation string `json:"explanation"`
}
type TestCase struct {
	Input          string `json:"input"`
	ExpectedOutput string `json:"expectedOutput"`
}

type Submission struct {
	ID        string `json:"id"`
	Status    string `json:"status"`
	Language  string `json:"language"`
	Timestamp string `json:"timestamp"`
	Runtime   string `json:"runtime"`
	Memory    string `json:"memory"`
}

type Problem struct {
	ID          int               `json:"id"`
	Title       string            `json:"title"`
	Platform    string            `json:"platform"`
	Tags        []string          `json:"tags"`
	Difficulty  string            `json:"difficulty"`
	Description string            `json:"description"`
	Examples    []Example         `json:"examples"`
	Constraints []string          `json:"constraints"`
	TestCases   []TestCase        `json:"testCases"`
	CodeStubs   map[string]string `json:"codeStubs"`
	Submissions []Submission      `json:"submissions"`
}

// ValidationError reports which required fields a problem payload is missing.
// The message is safe to show in the UI; the caller that triggered the failure
// is written to the log instead, so it can be traced without a debugger.
type ValidationError struct {
	ID      int
	Missing []string
}

func (e *ValidationError) Error() string {
	return fmt.Sprintf("problem %d is incomplete: missing %s", e.ID, strings.Join(e.Missing, ", "))
}

// ValidateProblem checks that a problem carries everything the screens need to
// render it. A failure means the payload was truncated or the fetch failed, not
// that the user did anything wrong.
//
// The immediate caller is recorded in clicode.log so a validation failure can be
// traced back to its origin from the log alone.
func (p *Problem) ValidateProblem() error {
	var missing []string

	if p.ID == 0 {
		missing = append(missing, "id")
	}
	if p.Title == "" {
		missing = append(missing, "title")
	}
	if p.Description == "" {
		missing = append(missing, "description")
	}
	if len(p.Examples) == 0 {
		missing = append(missing, "examples")
	}
	if len(p.Constraints) == 0 {
		missing = append(missing, "constraints")
	}
	if len(p.TestCases) == 0 {
		missing = append(missing, "testCases")
	}

	if len(missing) == 0 {
		return nil
	}

	log.Printf("validation failed for problem %d: missing %s (called from %s)",
		p.ID, strings.Join(missing, ", "), callerName(2))

	return &ValidationError{ID: p.ID, Missing: missing}
}

// callerName resolves the function skip levels up the stack, for log context.
func callerName(skip int) string {
	pc, _, line, ok := runtime.Caller(skip)
	if !ok {
		return "unknown caller (stack unreadable)"
	}
	return fmt.Sprintf("%s:%d", runtime.FuncForPC(pc).Name(), line)
}

// Format renders a problem as the plain text shown in the description pane.
// It never fails: a problem that is missing pieces renders the pieces it has.
// Call ValidateProblem separately when completeness actually matters.
func (p *Problem) Format() string {
	var sb strings.Builder

	sb.WriteString(fmt.Sprintf("%v %s [%s] \n", p.ID, p.Title, p.Difficulty))
	sb.WriteString(fmt.Sprintf("Platform: %s\n\n", p.Platform))
	sb.WriteString(p.Description + "\n\n")

	for _, t := range p.Examples {
		sb.WriteString(fmt.Sprintf("  - %v\n", t.Input))
		sb.WriteString(fmt.Sprintf("  - %v\n", t.Output))
		if len(t.Explanation) != 0 {
			sb.WriteString(fmt.Sprintf(" - %v\n", t.Explanation))
		}
	}

	if len(p.Constraints) > 0 {
		sb.WriteString("Constraints: \n\n")
		sb.WriteString(fmt.Sprintf("%v\n", strings.Join(p.Constraints, "\n")))
	}

	if len(p.Tags) > 0 {
		sb.WriteString("Tags: ")
		sb.WriteString(fmt.Sprintf("%v", strings.Join(p.Tags, ", ")))
	}

	return sb.String()
}

func (p *Problem) FormatTestCase(index int) string {
	if index < 0 || index >= len(p.TestCases) {
		return "No test case available."
	}
	tc := p.TestCases[index]
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("Test Case %d of %d\n\n", index+1, len(p.TestCases)))
	sb.WriteString("Input:\n")
	sb.WriteString(tc.Input + "\n\n")
	sb.WriteString("Expected Output:\n")
	sb.WriteString(tc.ExpectedOutput + "\n")
	return sb.String()
}

// AvailableLanguages lists the languages this problem ships a code stub for,
// sorted so the UI cycles through them in a stable order. Not every problem
// offers every language, so the UI should only present what comes back here.
func (p *Problem) AvailableLanguages() []string {
	langs := make([]string, 0, len(p.CodeStubs))
	for lang := range p.CodeStubs {
		langs = append(langs, lang)
	}
	sort.Strings(langs)
	return langs
}

// GetCodeStub returns the starter code for a language. The bool reports whether
// this problem actually has one — use AvailableLanguages to find out which do.
func (p *Problem) GetCodeStub(language string) (string, bool) {
	stub, ok := p.CodeStubs[language]
	return stub, ok
}
