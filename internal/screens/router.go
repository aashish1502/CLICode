package screens

import (
	"github.com/aashish1502/clicode/internal/config"
	"github.com/aashish1502/clicode/internal/loader"
	"github.com/aashish1502/clicode/internal/session"
	tea "github.com/charmbracelet/bubbletea"
)

type screenID int

const (
	titleScreenID screenID = iota
	menuScreenID
	listScreenID
	problemScreenID
	tcScreenID
	settingsScreenID
)

// Router is the top-level tea.Model. It owns all screens and routes messages
// between them. Each screen is stored so state (code editor content, scroll
// positions) survives navigation away and back.
type Router struct {
	current       screenID
	screens       map[screenID]tea.Model
	lastProblemID int
	cfg           config.Config
	width         int
	height        int
}

func NewRouter() Router {
	sess, _ := session.Load()
	cfg, _ := config.Load()

	r := Router{
		current:       titleScreenID,
		screens:       make(map[screenID]tea.Model),
		lastProblemID: sess.LastProblemID,
		cfg:           cfg,
	}
	r.screens[titleScreenID] = NewTitleScreen(0, 0)
	r.screens[menuScreenID] = NewMenuScreen(r.lastProblemID, 0, 0)
	r.screens[listScreenID] = NewProblemListScreen(r.lastProblemID, 0, 0)
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
		_ = session.Save(session.Session{LastProblemID: r.lastProblemID})
		problem, err := loader.LoadProblem(m.ProblemID)
		ps := NewProblemScreen(problem, err, r.width, r.height, r.cfg.DefaultLanguage)
		r.screens[problemScreenID] = ps
		r.current = problemScreenID
		return r, ps.Init()

	case NavigateToTestCaseMsg:
		tc := NewTestCaseScreen(m.Problem, r.width, r.height)
		r.screens[tcScreenID] = tc
		r.current = tcScreenID
		return r, tc.Init()

	case NavigateBackMsg:
		// TC → problem screen (state preserved).
		if r.screens[problemScreenID] != nil {
			r.current = problemScreenID
		} else {
			r.current = menuScreenID
		}
		return r, nil

	case NavigateToMenuMsg:
		// Rebuild the menu so the last-worked-on indicator is fresh.
		menu := NewMenuScreen(r.lastProblemID, r.width, r.height)
		r.screens[menuScreenID] = menu
		r.current = menuScreenID
		return r, menu.Init()

	case NavigateToProblemListMsg:
		// Rebuild the list so the last-worked-on indicator is fresh.
		list := NewProblemListScreen(r.lastProblemID, r.width, r.height)
		r.screens[listScreenID] = list
		r.current = listScreenID
		return r, list.Init()

	case NavigateToSettingsMsg:
		r.screens[settingsScreenID] = NewSettingsScreen(r.width, r.height)
		r.current = settingsScreenID
		return r, nil

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
