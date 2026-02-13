package models

import (
	"fmt"
	"runtime"
	"strings"
)

type Example struct {
	Input       string `json:"input"`
	Output      string `json:"output"`
	Explanition string `json:"explanition"`
}
type TestCase struct {
	Input          string `json:"input"`
	ExpectedOutput string `json:"expectedOutput"`
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
}

func (p *Problem) ValidateProblem() error {

	pc, _, _, ok := runtime.Caller(1)
	callerName := "IDK man something worse has happened i cannot read the stack"

	if ok {
		callerName = runtime.FuncForPC(pc).Name()
	}

	if p.ID == 0 || p.Description == "" || p.Title == "" || len(p.TestCases) == 0 || len(p.Examples) == 0 || p.Examples == nil || len(p.Constraints) == 0 {
		return fmt.Errorf("problem struct either failed to read the API or the problem API fetched incomplete payload %v", callerName)
	}

	return nil

}

func (p *Problem) FormatProblemFromProblemStruct() (string, error) {

	var sb strings.Builder

	var err error = p.ValidateProblem()

	sb.WriteString(fmt.Sprintf("%v %s [%s] \n", p.ID, p.Title, p.Difficulty))
	sb.WriteString(fmt.Sprintf("Platform: %s\n\n", p.Platform))
	sb.WriteString(p.Description + "\n\n")

	for _, t := range p.Examples {
		sb.WriteString(fmt.Sprintf("  - %v\n", t.Input))
		sb.WriteString(fmt.Sprintf("  - %v\n", t.Output))
		if len(t.Explanition) != 0 {
			sb.WriteString(fmt.Sprintf(" - %v\n", t.Explanition))
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

	return sb.String(), err

}

func (p *Problem) GetCodeStub(language string) string {

	if stub, ok := p.CodeStubs[language]; ok {
		return stub
	}

	// Return available languages for error message
	available := make([]string, 0, len(p.CodeStubs))
	for lang := range p.CodeStubs {
		available = append(available, lang)
	}

	return ""

}
