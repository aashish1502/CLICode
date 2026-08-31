package screens

import (
	"testing"

	tea "github.com/charmbracelet/bubbletea"
)

// newTestRouter builds a router with HOME pointed at a temp dir, so the
// database it opens and seeds never touches the real ~/.clicode.
func newTestRouter(t *testing.T) Router {
	t.Helper()
	t.Setenv("HOME", t.TempDir())

	r := NewRouter()
	t.Cleanup(func() { r.Close() })
	m, _ := r.Update(tea.WindowSizeMsg{Width: 80, Height: 24})
	return m.(Router)
}

func send(t *testing.T, r Router, msg tea.Msg) (Router, tea.Cmd) {
	t.Helper()
	m, cmd := r.Update(msg)
	got, ok := m.(Router)
	if !ok {
		t.Fatalf("Update returned %T, want Router", m)
	}
	return got, cmd
}

// visible returns the screen the user is currently looking at.
func visible(t *testing.T, r Router) tea.Model {
	t.Helper()
	m, ok := r.stack.current()
	if !ok {
		t.Fatal("stack is empty")
	}
	return m
}

// quits reports whether a command is tea.Quit.
func quits(cmd tea.Cmd) bool {
	if cmd == nil {
		return false
	}
	_, isQuit := cmd().(tea.QuitMsg)
	return isQuit
}

func TestStartsOnTheTitleScreen(t *testing.T) {
	r := newTestRouter(t)

	if got := r.stack.depth(); got != 1 {
		t.Errorf("depth = %d at startup, want 1", got)
	}
	if _, ok := visible(t, r).(TitleScreen); !ok {
		t.Error("startup screen is not the title screen")
	}
}

// The splash must not linger behind the menu — going back from the menu should
// quit, not return to the title card.
func TestTitleDoesNotRemainBehindTheMenu(t *testing.T) {
	r := newTestRouter(t)
	r, _ = send(t, r, NavigateToMenuMsg{})

	if got := r.stack.depth(); got != 1 {
		t.Errorf("depth = %d after reaching the menu, want 1", got)
	}
	if _, ok := visible(t, r).(MenuScreen); !ok {
		t.Error("current screen is not the menu")
	}
}

// The lifecycle the stack defines: back from the last screen exits.
func TestBackFromTheRootQuits(t *testing.T) {
	r := newTestRouter(t)
	r, _ = send(t, r, NavigateToMenuMsg{})

	_, cmd := send(t, r, NavigateBackMsg{})

	if !quits(cmd) {
		t.Error("back from the root screen did not quit")
	}
}

func TestBackBelowTheRootDoesNotQuit(t *testing.T) {
	r := newTestRouter(t)
	r, _ = send(t, r, NavigateToMenuMsg{})
	r, _ = send(t, r, NavigateToProblemListMsg{})

	r, cmd := send(t, r, NavigateBackMsg{})

	if quits(cmd) {
		t.Error("back from the list quit instead of returning to the menu")
	}
	if _, ok := visible(t, r).(MenuScreen); !ok {
		t.Error("back did not reveal the menu")
	}
}

// The escape hatch: ctrl+q must quit regardless of where the user is.
func TestCtrlQQuitsFromAnyScreen(t *testing.T) {
	r := newTestRouter(t)
	r, _ = send(t, r, NavigateToMenuMsg{})
	r, _ = send(t, r, NavigateToProblemListMsg{})
	r, _ = send(t, r, NavigateToSettingsMsg{})

	_, cmd := send(t, r, tea.KeyMsg{Type: tea.KeyCtrlQ})

	if !quits(cmd) {
		t.Error("ctrl+q did not quit")
	}
}

// The spam guard, at the router level rather than the stack's.
func TestRepeatedMenuNavigationDoesNotGrowTheStack(t *testing.T) {
	r := newTestRouter(t)
	r, _ = send(t, r, NavigateToMenuMsg{})
	r, _ = send(t, r, NavigateToProblemListMsg{})

	for i := 0; i < 50; i++ {
		r, _ = send(t, r, NavigateToMenuMsg{})
	}

	if got := r.stack.depth(); got != 1 {
		t.Errorf("depth = %d after 50 menu navigations, want 1", got)
	}
}

func TestRepeatedListNavigationDoesNotGrowTheStack(t *testing.T) {
	r := newTestRouter(t)
	r, _ = send(t, r, NavigateToMenuMsg{})

	for i := 0; i < 50; i++ {
		r, _ = send(t, r, NavigateToProblemListMsg{})
	}

	if got := r.stack.depth(); got != 2 {
		t.Errorf("depth = %d after 50 list navigations, want 2", got)
	}
}

// Opening a problem and going back must land on the list, not skip to the menu.
func TestProblemStacksOnTheList(t *testing.T) {
	r := newTestRouter(t)
	r, _ = send(t, r, NavigateToMenuMsg{})
	r, _ = send(t, r, NavigateToProblemListMsg{})
	r, _ = send(t, r, NavigateToProblemMsg{ProblemID: 1})

	if got := r.stack.depth(); got != 3 {
		t.Fatalf("depth = %d, want 3", got)
	}

	r, _ = send(t, r, NavigateBackMsg{})
	if _, ok := visible(t, r).(ProblemListScreen); !ok {
		t.Error("back from a problem did not reveal the list")
	}
}

// A screen reused from the stack still has to show current data.
func TestMenuIsRefreshedWhenRevealed(t *testing.T) {
	r := newTestRouter(t)
	r, _ = send(t, r, NavigateToMenuMsg{})
	r, _ = send(t, r, NavigateToProblemListMsg{})
	r, _ = send(t, r, NavigateToProblemMsg{ProblemID: 2})
	r, _ = send(t, r, NavigateToMenuMsg{})

	menu, ok := visible(t, r).(MenuScreen)
	if !ok {
		t.Fatal("current screen is not the menu")
	}
	if menu.lastProblemID != 2 {
		t.Errorf("menu still shows lastProblemID = %d, want 2", menu.lastProblemID)
	}
}

// A screen restored after a resize must not render at its old size.
func TestRevealedScreenGetsTheCurrentSize(t *testing.T) {
	r := newTestRouter(t)
	r, _ = send(t, r, NavigateToMenuMsg{})
	r, _ = send(t, r, NavigateToProblemListMsg{})

	r, _ = send(t, r, tea.WindowSizeMsg{Width: 120, Height: 40})
	r, _ = send(t, r, NavigateBackMsg{})

	menu, ok := visible(t, r).(MenuScreen)
	if !ok {
		t.Fatal("current screen is not the menu")
	}
	if menu.width != 120 {
		t.Errorf("revealed menu width = %d, want 120", menu.width)
	}
}

// ── persistence ──────────────────────────────────────────────────────────────

// The whole point of step 2: what you type survives leaving the screen.
func TestSavedSolutionIsRestoredWhenTheProblemIsReopened(t *testing.T) {
	r := newTestRouter(t)

	r, _ = send(t, r, NavigateToProblemMsg{ProblemID: 1})
	r, cmd := send(t, r, SaveSolutionMsg{ProblemID: 1, Language: "kotlin", Code: "fun solve() {}"})
	if cmd == nil {
		t.Fatal("SaveSolutionMsg produced no command; nothing was written")
	}
	if saved, ok := cmd().(SolutionSavedMsg); !ok || saved.Err != nil {
		t.Fatalf("save failed: %+v", saved)
	}

	// Walk away far enough that the screen is dropped from the cache, so the
	// buffer can only come back from the database.
	r, _ = send(t, r, NavigateToMenuMsg{})
	for i := 100; i < 110; i++ {
		r, _ = send(t, r, NavigateToProblemMsg{ProblemID: i})
	}
	r, _ = send(t, r, NavigateToMenuMsg{})
	r, _ = send(t, r, NavigateToProblemMsg{ProblemID: 1})

	s, ok := visible(t, r).(ProblemScreen)
	if !ok {
		t.Fatalf("visible screen is %T, want ProblemScreen", visible(t, r))
	}
	if got := s.langEdits["kotlin"]; got != "fun solve() {}" {
		t.Errorf("restored kotlin buffer = %q, want the saved code", got)
	}
}

// Saving records the language, and reopening returns to it -- even though
// kotlin is not the configured default and ships no starter code.
func TestProblemReopensInTheLanguageItWasLastSavedIn(t *testing.T) {
	r := newTestRouter(t)

	r, _ = send(t, r, NavigateToProblemMsg{ProblemID: 1})
	_, cmd := send(t, r, SaveSolutionMsg{ProblemID: 1, Language: "kotlin", Code: "fun solve() {}"})
	cmd()

	r2 := newRouterSharingHome(t)
	r2, _ = send(t, r2, NavigateToProblemMsg{ProblemID: 1})

	s := visible(t, r2).(ProblemScreen)
	if s.language != "kotlin" {
		t.Errorf("reopened in %q, want kotlin (the last language saved)", s.language)
	}
}

// A fresh Router over the same HOME -- the closest thing to restarting the app.
func newRouterSharingHome(t *testing.T) Router {
	t.Helper()
	r := NewRouter()
	t.Cleanup(func() { r.Close() })
	m, _ := r.Update(tea.WindowSizeMsg{Width: 80, Height: 24})
	return m.(Router)
}

// "Continue where you left off" now comes from the progress table.
func TestLastOpenedSurvivesARestart(t *testing.T) {
	r := newTestRouter(t)
	r, _ = send(t, r, NavigateToProblemMsg{ProblemID: 3})

	restarted := newRouterSharingHome(t)
	if restarted.lastProblemID != 3 {
		t.Errorf("lastProblemID after restart = %d, want 3", restarted.lastProblemID)
	}
}

func TestProblemListComesFromTheCatalog(t *testing.T) {
	r := newTestRouter(t)
	r, _ = send(t, r, NavigateToProblemListMsg{})

	s, ok := visible(t, r).(ProblemListScreen)
	if !ok {
		t.Fatalf("visible screen is %T, want ProblemListScreen", visible(t, r))
	}
	if s.err != nil {
		t.Fatalf("list screen carries an error: %v", s.err)
	}
	if len(s.problems) == 0 {
		t.Error("the catalog returned no problems; seeding did not run")
	}
}
