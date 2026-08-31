package screens

import (
	"context"
	"log"
	"strconv"

	"github.com/aashish1502/clicode/internal/catalog"
	"github.com/aashish1502/clicode/internal/config"
	"github.com/aashish1502/clicode/internal/languages"
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

// refresher is implemented by screens whose displayed data can go stale while
// they sit lower in the stack or in the cache — the menu's last-worked-on
// marker, for instance. The router calls Refresh when the screen becomes
// visible again, so the screen keeps its cursor while its data catches up.
type refresher interface {
	Refresh(lastProblemID int) tea.Model
}

// Router is the top-level tea.Model. It owns the screen stack and translates
// navigation messages into pushes and pops.
//
// Navigation is a stack rather than a set of hardcoded transitions: every screen
// knows how to say "go back" without knowing what is behind it, and the app's
// lifecycle is exactly the stack's — when the last screen pops, the app exits.
type Router struct {
	stack         stack
	lastProblemID int
	cfg           config.Config
	width         int
	height        int

	// data is the only way the router reaches problems or saved work. It is an
	// interface so the store, and later the API, stay invisible to screens.
	data catalog.Catalog

	// supported is the set of languages the editor may open a buffer in,
	// resolved once at startup. It is static today; when the API supplies it
	// this becomes a refreshable snapshot rather than a constant.
	supported []languages.Language
}

func NewRouter() Router {
	cfg, _ := config.Load()
	data := catalog.Open()

	// "Continue where you left off" is now derived from the progress table --
	// the most recently opened problem -- rather than a separately stored id
	// that could disagree with it.
	last, err := data.LastOpened(context.Background())
	if err != nil {
		log.Printf("could not read the last opened problem: %v", err)
	}

	supported, err := languages.NewStatic(cfg.Languages).Supported(context.Background())
	if err != nil {
		// A static catalog cannot fail; an API-backed one can, and when it
		// does the editor still needs something to open.
		log.Printf("language catalog unavailable, falling back to defaults: %v", err)
		supported, _ = languages.NewStatic(nil).Supported(context.Background())
	}

	r := Router{
		lastProblemID: last,
		cfg:           cfg,
		supported:     supported,
		data:          data,
	}
	r.stack.reset(Key{ID: titleScreenID}, NewTitleScreen(0, 0))
	return r
}

// Close releases the catalog. Bubble Tea has no teardown hook, so main calls
// this once the program loop has returned.
func (r Router) Close() error {
	if r.data == nil {
		return nil
	}
	return r.data.Close()
}

func (r Router) Init() tea.Cmd {
	if m, ok := r.stack.current(); ok {
		return m.Init()
	}
	return nil
}

func (r Router) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch m := msg.(type) {

	case tea.KeyMsg:
		// Hard quit. Deliberately global and handled before anything else, so
		// it still works if a screen stops responding to everything else.
		if m.String() == "ctrl+q" {
			return r, tea.Quit
		}

	// ── Window resize: every screen on the stack, not just the visible one ────

	case tea.WindowSizeMsg:
		r.width, r.height = m.Width, m.Height
		return r, r.stack.broadcast(m)

	// ── Navigation ───────────────────────────────────────────────────────────

	case NavigateToMenuMsg:
		// Root: the menu is the bottom of the stack, so reaching it from any
		// depth collapses back to a single frame rather than stacking copies.
		return r.navigate(Key{ID: menuScreenID}, Root, func() tea.Model {
			return NewMenuScreen(r.lastProblemID, r.width, r.height)
		})

	case NavigateToProblemListMsg:
		return r.navigate(Key{ID: listScreenID}, Single, func() tea.Model {
			items, err := r.data.List(context.Background())
			return NewProblemListScreen(items, err, r.lastProblemID, r.width, r.height)
		})

	case NavigateToProblemMsg:
		r.lastProblemID = m.ProblemID
		id := m.ProblemID

		return r.navigate(Key{ID: problemScreenID, Arg: strconv.Itoa(id)}, Single, func() tea.Model {
			ctx := context.Background()

			problem, err := r.data.Problem(ctx, id)

			// Reopen in the language this problem was last worked in, and with
			// whatever was saved in each language. Both are best-effort: a
			// failure here costs the convenience, not the problem.
			language := r.cfg.DefaultLanguage
			if last, lerr := r.data.LastLanguage(ctx, id); lerr != nil {
				log.Printf("could not read the last language for problem %d: %v", id, lerr)
			} else if last != "" {
				language = last
			}

			saved, serr := r.data.Solutions(ctx, id)
			if serr != nil {
				log.Printf("could not read saved solutions for problem %d: %v", id, serr)
				saved = nil
			}

			if merr := r.data.MarkOpened(ctx, id, ""); merr != nil {
				log.Printf("could not record problem %d as opened: %v", id, merr)
			}

			return NewProblemScreen(ProblemArgs{
				Problem:   problem,
				Err:       err,
				Width:     r.width,
				Height:    r.height,
				Language:  language,
				Supported: r.supported,
				Saved:     saved,
				Writable:  r.data.Writable(),
			})
		})

	case NavigateToTestCaseMsg:
		p := m.Problem
		return r.navigate(Key{ID: tcScreenID, Arg: strconv.Itoa(p.ID)}, Single, func() tea.Model {
			return NewTestCaseScreen(p, r.width, r.height)
		})

	case NavigateToSettingsMsg:
		return r.navigate(Key{ID: settingsScreenID}, Single, func() tea.Model {
			return NewSettingsScreen(r.width, r.height)
		})

	case SaveSolutionMsg:
		// Screens stay pure: ":w" asks, the router writes, and the outcome
		// comes back as a message the screen can render.
		save := m
		return r, func() tea.Msg {
			ctx := context.Background()
			err := r.data.SaveSolution(ctx, save.ProblemID, save.Language, save.Code)
			if err != nil {
				log.Printf("saving problem %d (%s): %v", save.ProblemID, save.Language, err)
			} else if merr := r.data.MarkOpened(ctx, save.ProblemID, save.Language); merr != nil {
				log.Printf("could not record the language for problem %d: %v", save.ProblemID, merr)
			}
			return SolutionSavedMsg{Language: save.Language, Err: err}
		}

	case NavigateBackMsg:
		cmd, ok := r.stack.pop()
		if !ok {
			// Nothing left behind this screen — the app's lifecycle is the
			// stack's, so an empty stack means we're done.
			return r, tea.Quit
		}
		r.reveal()
		return r, cmd
	}

	// ── Delegate to the visible screen ───────────────────────────────────────

	if m, ok := r.stack.current(); ok {
		updated, cmd := m.Update(msg)
		r.stack.replaceCurrent(updated)
		return r, cmd
	}
	return r, nil
}

func (r Router) View() string {
	if m, ok := r.stack.current(); ok {
		return m.View()
	}
	return "Loading..."
}

// navigate pushes a screen and brings it up to date before it is drawn.
func (r Router) navigate(k Key, mode Mode, build func() tea.Model) (tea.Model, tea.Cmd) {
	cmd := r.stack.push(k, mode, build)
	r.reveal()
	return r, cmd
}

// reveal prepares the now-visible screen: refreshes data that may have gone
// stale while it was hidden, and hands it the current terminal size, which it
// will have missed if it was restored from the cache after a resize.
func (r *Router) reveal() {
	m, ok := r.stack.current()
	if !ok {
		return
	}
	if ref, isRefresher := m.(refresher); isRefresher {
		m = ref.Refresh(r.lastProblemID)
	}
	if r.width > 0 && r.height > 0 {
		m, _ = m.Update(tea.WindowSizeMsg{Width: r.width, Height: r.height})
	}
	r.stack.replaceCurrent(m)
}
