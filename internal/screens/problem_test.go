package screens

import (
	"errors"
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/aashish1502/clicode/internal/languages"
	"github.com/aashish1502/clicode/internal/models"
)

func supported(ids ...string) []languages.Language {
	out := make([]languages.Language, 0, len(ids))
	for _, id := range ids {
		out = append(out, languages.Get(id))
	}
	return out
}

// problemWithStubs builds a renderable problem shipping starter code for the
// given languages only.
func problemWithStubs(langs ...string) *models.Problem {
	stubs := make(map[string]string, len(langs))
	for _, l := range langs {
		stubs[l] = "stub for " + l
	}
	return &models.Problem{
		ID:          1,
		Title:       "Two Sum",
		Description: "Find two numbers that add up to target.",
		Examples:    []models.Example{{Input: "a", Output: "b"}},
		Constraints: []string{"n < 10"},
		TestCases:   []models.TestCase{{Input: "a", ExpectedOutput: "b"}},
		CodeStubs:   stubs,
	}
}

func languageIDs(s ProblemScreen) []string {
	return languages.IDs(s.languageSet())
}

// The regression this file exists for: a judge that accepts Kotlin accepts it
// whether or not this particular problem ships Kotlin starter code. Deriving the
// selectable set from the stubs made a Kotlin-capable judge unreachable.
func TestLanguageSetIsNotLimitedToStubs(t *testing.T) {
	p := problemWithStubs("python")
	s := NewProblemScreen(ProblemArgs{Problem: p, Width: 80, Height: 24, Language: "python", Supported: supported("python", "go", "kotlin"), Writable: true})

	got := languageIDs(s)
	for _, want := range []string{"python", "go", "kotlin"} {
		if !contains(got, want) {
			t.Errorf("%q not selectable; got %v", want, got)
		}
	}
}

// A stub for a language the catalog has never heard of must not be dropped.
func TestLanguageSetIncludesUnknownStubbedLanguage(t *testing.T) {
	p := problemWithStubs("python", "fortran")
	s := NewProblemScreen(ProblemArgs{Problem: p, Width: 80, Height: 24, Language: "python", Supported: supported("python", "go"), Writable: true})

	if got := languageIDs(s); !contains(got, "fortran") {
		t.Errorf("stubbed language dropped; got %v", got)
	}
}

// key builds a KeyMsg the way bubbletea would deliver it.
func key(s string) tea.KeyMsg {
	switch s {
	case "enter":
		return tea.KeyMsg{Type: tea.KeyEnter}
	case "esc":
		return tea.KeyMsg{Type: tea.KeyEsc}
	case "ctrl+l":
		return tea.KeyMsg{Type: tea.KeyCtrlL}
	default:
		return tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune(s)}
	}
}

// press feeds one key to the screen and returns the updated screen.
func press(t *testing.T, s ProblemScreen, k string) ProblemScreen {
	t.Helper()
	m, _ := s.Update(key(k))
	got, ok := m.(ProblemScreen)
	if !ok {
		t.Fatalf("Update returned %T, want ProblemScreen", m)
	}
	return got
}

func TestCtrlLOpensThePicker(t *testing.T) {
	s := NewProblemScreen(ProblemArgs{Problem: problemWithStubs("python"), Width: 80, Height: 24, Language: "python", Supported: supported("python", "go"), Writable: true})

	if s.picker.open {
		t.Fatal("picker starts open")
	}
	if s = press(t, s, "ctrl+l"); !s.picker.open {
		t.Error("ctrl+l did not open the picker")
	}
}

// The picker lists everything selectable, not just the stubbed languages, and
// knows which is which so it can colour them.
func TestPickerListsEverySelectableLanguage(t *testing.T) {
	s := NewProblemScreen(ProblemArgs{Problem: problemWithStubs("python"), Width: 80, Height: 24, Language: "python", Supported: supported("python", "go", "kotlin"), Writable: true})
	s = press(t, s, "ctrl+l")

	if got := languages.IDs(s.picker.langs); len(got) != 3 {
		t.Fatalf("picker lists %v, want all three", got)
	}
	if !s.picker.stubbed["python"] {
		t.Error("python has starter code but is not marked stubbed")
	}
	if s.picker.stubbed["kotlin"] {
		t.Error("kotlin has no starter code but is marked stubbed")
	}
}

func TestPickerOpensOnTheCurrentLanguage(t *testing.T) {
	s := NewProblemScreen(ProblemArgs{Problem: problemWithStubs("python"), Width: 80, Height: 24, Language: "go", Supported: supported("python", "go", "kotlin"), Writable: true})
	s = press(t, s, "ctrl+l")

	if got := s.picker.langs[s.picker.cursor].ID; got != "go" {
		t.Errorf("cursor starts on %q, want the current language go", got)
	}
}

// Selecting a language with no starter code must work — that is the whole point
// of not deriving the list from the stubs.
func TestPickerSelectsAnUnstubbedLanguage(t *testing.T) {
	s := NewProblemScreen(ProblemArgs{Problem: problemWithStubs("python"), Width: 80, Height: 24, Language: "python", Supported: supported("python", "go", "kotlin"), Writable: true})

	s = press(t, s, "ctrl+l")
	s = press(t, s, "j")
	s = press(t, s, "j")
	s = press(t, s, "enter")

	if s.picker.open {
		t.Error("picker still open after selection")
	}
	if s.language != "kotlin" {
		t.Errorf("language = %q, want kotlin", s.language)
	}
}

func TestPickerCancelKeepsCurrentLanguage(t *testing.T) {
	s := NewProblemScreen(ProblemArgs{Problem: problemWithStubs("python"), Width: 80, Height: 24, Language: "python", Supported: supported("python", "go"), Writable: true})

	s = press(t, s, "ctrl+l")
	s = press(t, s, "j")
	s = press(t, s, "esc")

	if s.picker.open {
		t.Error("picker still open after cancel")
	}
	if s.language != "python" {
		t.Errorf("cancel changed the language to %q", s.language)
	}
}

// Switching away and back must not lose what was typed.
func TestSetLanguagePreservesEdits(t *testing.T) {
	s := NewProblemScreen(ProblemArgs{Problem: problemWithStubs("python", "go"), Width: 80, Height: 24, Language: "python", Supported: supported("python", "go"), Writable: true})

	s.codeEditor = s.codeEditor.SetValue("my python work")
	s = s.setLanguage("go")
	s = s.setLanguage("python")

	if got := s.codeEditor.Value(); !strings.Contains(got, "my python work") {
		t.Errorf("edits lost on round trip: %q", got)
	}
}

func TestPickerViewShowsEveryLanguageAndLegend(t *testing.T) {
	s := NewProblemScreen(ProblemArgs{Problem: problemWithStubs("python"), Width: 80, Height: 24, Language: "python", Supported: supported("python", "go", "kotlin"), Writable: true})
	s = press(t, s, "ctrl+l")

	view := s.View()
	for _, want := range []string{"Python", "Go", "Kotlin", "starter code"} {
		if !strings.Contains(view, want) {
			t.Errorf("picker view missing %q", want)
		}
	}
}

// Selecting a language with no starter code is allowed, and must open a comment
// valid in that language rather than a bare "//".
func TestStubForLangFallsBackPerLanguage(t *testing.T) {
	p := problemWithStubs("go")
	s := NewProblemScreen(ProblemArgs{Problem: p, Width: 80, Height: 24, Language: "go", Supported: supported("go", "python"), Writable: true})

	if got := s.stubForLang("go"); got != "stub for go" {
		t.Errorf("go: got %q, want the real stub", got)
	}
	if got := s.stubForLang("python"); !strings.HasPrefix(got, "# ") {
		t.Errorf("python fallback = %q, want a # comment", got)
	}
}

func TestInitialLanguage(t *testing.T) {
	tests := []struct {
		name      string
		stubs     []string
		supported []string
		preferred string
		want      string
	}{
		{"configured default wins", []string{"python"}, []string{"python", "go"}, "go", "go"},
		{"unsupported default prefers a stubbed language", []string{"go"}, []string{"python", "go"}, "haskell", "go"},
		{"no stubs at all falls to first supported", nil, []string{"python", "go"}, "haskell", "python"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := NewProblemScreen(ProblemArgs{Problem: problemWithStubs(tt.stubs...), Width: 80, Height: 24, Language: tt.preferred, Supported: supported(tt.supported...), Writable: true})
			if s.language != tt.want {
				t.Errorf("language = %q, want %q", s.language, tt.want)
			}
		})
	}
}

// The router passes whatever config holds; an empty set must not leave the
// editor with nothing to open.
func TestEmptySupportedSetFallsBackToDefaults(t *testing.T) {
	s := NewProblemScreen(ProblemArgs{Problem: problemWithStubs("python"), Width: 80, Height: 24, Language: "python", Writable: true})

	if len(languageIDs(s)) == 0 {
		t.Fatal("no languages available with an empty supported set")
	}
}

func contains(haystack []string, needle string) bool {
	for _, s := range haystack {
		if s == needle {
			return true
		}
	}
	return false
}

// ── saved work ───────────────────────────────────────────────────────────────

// A saved buffer wins over the problem's starter code: reopening a problem
// returns the user to their own work.
func TestSavedBufferIsOpenedInsteadOfTheStub(t *testing.T) {
	s := NewProblemScreen(ProblemArgs{
		Problem:   problemWithStubs("python"),
		Width:     80,
		Height:    24,
		Language:  "python",
		Supported: supported("python", "go"),
		Saved:     map[string]string{"python": "my own code"},
		Writable:  true,
	})

	if got := s.codeEditor.Value(); got != "my own code" {
		t.Errorf("editor opened with %q, want the saved buffer", got)
	}
}

// The gap found while wiring the router: a solution saved in a language the
// configured set does not contain must still be reachable, or the work is
// stranded where the picker cannot get to it.
func TestALanguageWithSavedWorkIsAlwaysSelectable(t *testing.T) {
	s := NewProblemScreen(ProblemArgs{
		Problem:   problemWithStubs("python"),
		Width:     80,
		Height:    24,
		Language:  "python",
		Supported: supported("python", "go"), // no kotlin
		Saved:     map[string]string{"kotlin": "fun solve() {}"},
		Writable:  true,
	})

	if !contains(languageIDs(s), "kotlin") {
		t.Errorf("languages = %v, want kotlin included because work is saved in it",
			languageIDs(s))
	}
}

func TestWriteCommandAsksTheRouterToSave(t *testing.T) {
	s := NewProblemScreen(ProblemArgs{
		Problem:   problemWithStubs("python"),
		Width:     80,
		Height:    24,
		Language:  "python",
		Supported: supported("python"),
		Writable:  true,
	})

	s, cmd := s.runCmd("w")
	if cmd == nil {
		t.Fatal(":w produced no command")
	}
	msg, ok := cmd().(SaveSolutionMsg)
	if !ok {
		t.Fatalf(":w emitted %T, want SaveSolutionMsg", cmd())
	}
	if msg.ProblemID != 1 || msg.Language != "python" {
		t.Errorf("save = problem %d in %q, want problem 1 in python", msg.ProblemID, msg.Language)
	}
	if msg.Code == "" {
		t.Error("save carried an empty buffer")
	}
}

// A read-only catalog must say so rather than letting a write look like it
// worked.
func TestWriteIsRefusedWithoutADatabase(t *testing.T) {
	s := NewProblemScreen(ProblemArgs{
		Problem:   problemWithStubs("python"),
		Width:     80,
		Height:    24,
		Language:  "python",
		Supported: supported("python"),
		Writable:  false,
	})

	s, cmd := s.runCmd("w")
	if cmd != nil {
		t.Error(":w issued a write against a read-only catalog")
	}
	if s.status == "" {
		t.Error(":w failed silently; the user was told nothing")
	}
}

func TestSaveResultIsShownToTheUser(t *testing.T) {
	s := NewProblemScreen(ProblemArgs{
		Problem:   problemWithStubs("python"),
		Width:     80,
		Height:    24,
		Language:  "python",
		Supported: supported("python"),
		Writable:  true,
	})

	updated, _ := s.Update(SolutionSavedMsg{Language: "python"})
	if got := updated.(ProblemScreen).status; got == "" {
		t.Error("a successful save reported nothing")
	}

	updated, _ = s.Update(SolutionSavedMsg{Language: "python", Err: errWriteFailed})
	if got := updated.(ProblemScreen).status; got == "" {
		t.Error("a failed save reported nothing")
	}
}

var errWriteFailed = errors.New("disk on fire")

// The status line is a response to the last command, not a permanent fixture.
func TestStatusClearsOnTheNextKeypress(t *testing.T) {
	s := NewProblemScreen(ProblemArgs{
		Problem:   problemWithStubs("python"),
		Width:     80,
		Height:    24,
		Language:  "python",
		Supported: supported("python"),
		Writable:  true,
	})
	s.status = "written (python)"

	updated, _ := s.Update(key("j"))
	if got := updated.(ProblemScreen).status; got != "" {
		t.Errorf("status = %q after a keypress, want it cleared", got)
	}
}
