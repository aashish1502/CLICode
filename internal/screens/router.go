package screens

import (
	"github.com/aashish1502/clicode/internal/loader"
	tea "github.com/charmbracelet/bubbletea"
)

type screenID int

const (
	titleScreenID screenID = iota
	listScreenID
	problemScreenID
	tcScreenID
)

// Router is the top-level tea.Model. It owns all screens and routes messages
// between them. Each screen is stored so state (code editor content, scroll
// positions) survives navigation away and back.
type Router struct {
	current       screenID
	screens       map[screenID]tea.Model
	lastProblemID int
	width         int
	height        int
}

func NewRouter() Router {
	r := Router{
		current: titleScreenID,
		screens: make(map[screenID]tea.Model),
	}
	r.screens[titleScreenID] = NewTitleScreen(0, 0)
	r.screens[listScreenID] = NewProblemListScreen(0, 0, 0)
	return r
}

func (r Router) Init() tea.Cmd {
	return r.screens[r.current].Init()
}

func (r Router) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch m := msg.(type) {

	// ── Navigation ────────────────────────────────────────────────────────────

	case NavigateToProblemMsg:
		r.lastProblemID = m.ProblemID
		problem, err := loader.LoadProblem(m.ProblemID)
		ps := NewProblemScreen(problem, err, r.width, r.height)
		r.screens[problemScreenID] = ps
		r.current = problemScreenID
		return r, ps.Init()

	case NavigateToTestCaseMsg:
		tc := NewTestCaseScreen(m.Problem, r.width, r.height)
		r.screens[tcScreenID] = tc
		r.current = tcScreenID
		return r, tc.Init()

	case NavigateBackMsg:
		// TC → problem; problem → list (handled via "m" key in problem.go)
		if r.screens[problemScreenID] != nil {
			r.current = problemScreenID
		} else {
			r.current = listScreenID
		}
		return r, nil

	case NavigateToProblemListMsg:
		// Rebuild the list so the last-worked-on indicator is fresh.
		list := NewProblemListScreen(r.lastProblemID, r.width, r.height)
		r.screens[listScreenID] = list
		r.current = listScreenID
		return r, list.Init()

	// ── Window resize: propagate to all initialised screens ──────────────────

	case tea.WindowSizeMsg:
		r.width = m.Width
		r.height = m.Height
		var cmds []tea.Cmd
		for id, s := range r.screens {
			updated, cmd := s.Update(m)
			r.screens[id] = updated
			cmds = append(cmds, cmd)
		}
		return r, tea.Batch(cmds...)
	}

	// ── Delegate to the active screen ─────────────────────────────────────────

	if s, ok := r.screens[r.current]; ok {
		updated, cmd := s.Update(msg)
		r.screens[r.current] = updated
		return r, cmd
	}

	return r, nil
}

func (r Router) View() string {
	if s, ok := r.screens[r.current]; ok {
		return s.View()
	}
	return "Loading..."
}
