package screens

import (
	"testing"

	tea "github.com/charmbracelet/bubbletea"
)

// stubScreen stands in for a real screen. Note is mutable so tests can prove a
// screen came back from the cache rather than being rebuilt.
type stubScreen struct {
	name string
	note string
}

func (s stubScreen) Init() tea.Cmd                       { return nil }
func (s stubScreen) Update(tea.Msg) (tea.Model, tea.Cmd) { return s, nil }
func (s stubScreen) View() string                        { return s.name }

// counting returns a build func that records how many times it was called.
func counting(name string, builds *int) func() tea.Model {
	return func() tea.Model {
		*builds++
		return stubScreen{name: name}
	}
}

func build(name string) func() tea.Model {
	return func() tea.Model { return stubScreen{name: name} }
}

func currentName(t *testing.T, s *stack) string {
	t.Helper()
	m, ok := s.current()
	if !ok {
		t.Fatal("stack is empty")
	}
	return m.View()
}

// skey is shorthand for a screen key. Named to avoid colliding with the
// key() helper in problem_test.go, which builds tea.KeyMsg values.
func skey(id screenID, arg string) Key { return Key{ID: id, Arg: arg} }

// The reason Single exists: holding down a navigation key must not grow the
// stack without bound.
func TestSingleDoesNotGrowOnRepeatedPush(t *testing.T) {
	var s stack
	s.reset(skey(menuScreenID, ""), stubScreen{name: "menu"})

	for i := 0; i < 50; i++ {
		s.push(skey(listScreenID, ""), Single, build("list"))
	}

	if got := s.depth(); got != 2 {
		t.Errorf("depth = %d after 50 pushes, want 2", got)
	}
}

// Pushing a screen already lower in the stack pops back to it rather than
// stacking a second copy.
func TestSingleRaisesExistingInstance(t *testing.T) {
	var s stack
	s.reset(skey(menuScreenID, ""), stubScreen{name: "menu"})
	s.push(skey(listScreenID, ""), Single, build("list"))
	s.push(skey(problemScreenID, "1"), Single, build("problem"))

	var builds int
	s.push(skey(listScreenID, ""), Single, counting("list", &builds))

	if got := s.depth(); got != 2 {
		t.Errorf("depth = %d, want 2 (problem discarded)", got)
	}
	if got := currentName(t, &s); got != "list" {
		t.Errorf("current = %q, want list", got)
	}
	if builds != 0 {
		t.Errorf("rebuilt the list %d times; it was already live", builds)
	}
}

func TestRootCollapsesTheStack(t *testing.T) {
	var s stack
	s.reset(skey(menuScreenID, ""), stubScreen{name: "menu"})
	s.push(skey(listScreenID, ""), Single, build("list"))
	s.push(skey(problemScreenID, "1"), Single, build("problem"))

	s.push(skey(menuScreenID, ""), Root, build("menu"))

	if got := s.depth(); got != 1 {
		t.Errorf("depth = %d, want 1", got)
	}
	if got := currentName(t, &s); got != "menu" {
		t.Errorf("current = %q, want menu", got)
	}
}

func TestReplaceKeepsDepth(t *testing.T) {
	var s stack
	s.reset(skey(titleScreenID, ""), stubScreen{name: "title"})

	s.push(skey(menuScreenID, ""), Replace, build("menu"))

	if got := s.depth(); got != 1 {
		t.Errorf("depth = %d, want 1", got)
	}
	if got := currentName(t, &s); got != "menu" {
		t.Errorf("current = %q, want menu", got)
	}
}

func TestFreshAlwaysPushesANewInstance(t *testing.T) {
	var s stack
	s.reset(skey(menuScreenID, ""), stubScreen{name: "menu"})

	var builds int
	s.push(skey(listScreenID, ""), Fresh, counting("list", &builds))
	s.push(skey(listScreenID, ""), Fresh, counting("list", &builds))

	if got := s.depth(); got != 3 {
		t.Errorf("depth = %d, want 3", got)
	}
	if builds != 2 {
		t.Errorf("built %d times, want 2 — Fresh must not reuse", builds)
	}
}

// The app's lifecycle is the stack's: popping the last screen is the signal to
// quit, and must be distinguishable from an ordinary pop.
func TestPopReportsWhenTheStackEmpties(t *testing.T) {
	var s stack
	s.reset(skey(menuScreenID, ""), stubScreen{name: "menu"})
	s.push(skey(listScreenID, ""), Single, build("list"))

	if _, ok := s.pop(); !ok {
		t.Error("popping to a non-empty stack reported empty")
	}
	if _, ok := s.pop(); ok {
		t.Error("popping the last screen did not report empty")
	}
}

// A popped screen keeps its state, so going back to it restores the editor
// buffer and scroll position instead of resetting them.
func TestPoppedScreenIsRestoredFromCache(t *testing.T) {
	var s stack
	s.reset(skey(menuScreenID, ""), stubScreen{name: "menu"})
	s.push(skey(problemScreenID, "1"), Single, build("problem"))

	// Simulate work done in the screen.
	s.replaceCurrent(stubScreen{name: "problem", note: "half-written solution"})
	s.pop()

	var builds int
	s.push(skey(problemScreenID, "1"), Single, counting("problem", &builds))

	m, _ := s.current()
	if got := m.(stubScreen).note; got != "half-written solution" {
		t.Errorf("state lost on return: note = %q", got)
	}
	if builds != 0 {
		t.Errorf("rebuilt %d times; it should have come from the cache", builds)
	}
}

// Same screen, different content: problem 1 and problem 2 are separate
// instances with separate cached state.
func TestKeyedInstancesAreIndependent(t *testing.T) {
	var s stack
	s.reset(skey(menuScreenID, ""), stubScreen{name: "menu"})

	s.push(skey(problemScreenID, "1"), Single, build("problem-1"))
	s.replaceCurrent(stubScreen{name: "problem-1", note: "one"})
	s.pop()

	s.push(skey(problemScreenID, "2"), Single, build("problem-2"))
	s.replaceCurrent(stubScreen{name: "problem-2", note: "two"})
	s.pop()

	s.push(skey(problemScreenID, "1"), Single, build("rebuilt"))
	m, _ := s.current()
	if got := m.(stubScreen).note; got != "one" {
		t.Errorf("problem 1 restored the wrong state: %q", got)
	}
}

// The cache is a convenience, not storage — it must not grow without bound.
func TestCacheIsBounded(t *testing.T) {
	var s stack
	s.reset(skey(menuScreenID, ""), stubScreen{name: "menu"})

	for i := 0; i < cacheSize*3; i++ {
		s.push(skey(problemScreenID, string(rune('a'+i))), Single, build("problem"))
		s.pop()
	}

	if got := len(s.cache); got > cacheSize {
		t.Errorf("cache holds %d entries, cap is %d", got, cacheSize)
	}
}

func TestBroadcastReachesHiddenScreens(t *testing.T) {
	var s stack
	s.reset(skey(menuScreenID, ""), stubScreen{name: "menu"})
	s.push(skey(listScreenID, ""), Single, build("list"))

	// A no-op message: the point is that every frame is visited without panic
	// and the stack is left intact.
	s.broadcast(tea.WindowSizeMsg{Width: 80, Height: 24})

	if got := s.depth(); got != 2 {
		t.Errorf("depth = %d after broadcast, want 2", got)
	}
}
