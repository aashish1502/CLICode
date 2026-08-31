package screens

import (
	"context"
	"fmt"
	"sort"
	"strings"

	"github.com/aashish1502/clicode/internal/design"
	"github.com/aashish1502/clicode/internal/editor"
	"github.com/aashish1502/clicode/internal/languages"
	"github.com/aashish1502/clicode/internal/models"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

type pane int

const (
	problemPane pane = iota
	editorPane
)

var (
	cmdStyle  = lipgloss.NewStyle().Foreground(lipgloss.Color("250"))
	cmdPrompt = lipgloss.NewStyle().Foreground(lipgloss.Color("214")).Bold(true)
)

type ProblemScreen struct {
	activePane pane
	problem    *models.Problem

	// language is the current buffer's language id; supported is the set the
	// judge accepts, which is deliberately NOT limited to the languages this
	// problem ships starter code for.
	language  string
	supported []languages.Language

	langEdits          map[string]string
	width              int
	height             int
	problemDescription viewport.Model
	codeEditor         editor.VimEditor
	ready              bool
	err                error

	// vim-style command input (activated by ":" in normal mode)
	cmdMode bool
	cmdBuf  string

	// status is the transient line shown after a command -- "written", or why
	// it could not be. Cleared on the next keypress.
	status string

	// writable is false when the catalog has no database behind it, so ":w"
	// can say so instead of appearing to work.
	writable bool

	// picker is the modal language list, opened with ctrl+l.
	picker languagePicker
}

// ProblemArgs is everything the router hands a problem screen.
//
// A struct rather than positional parameters: the list grew past the point
// where NewProblemScreen(p, nil, 80, 24, "python", langs, nil, true) told a
// reader anything, and the two trailing values are exactly the kind that get
// silently transposed.
type ProblemArgs struct {
	Problem *models.Problem
	// Err is whatever the catalog returned when loading Problem. Non-nil means
	// the screen renders an error card instead of the split pane.
	Err           error
	Width, Height int

	// Language is the buffer to open in: the problem's last-used language when
	// there is one, otherwise the configured default.
	Language string
	// Supported is the set the judge accepts -- deliberately not limited to the
	// languages this problem ships starter code for.
	Supported []languages.Language

	// Saved holds buffers already persisted for this problem, keyed by
	// language. They take precedence over the code stub, so reopening returns
	// the user to their own work rather than starting over.
	Saved map[string]string

	// Writable is false when the catalog has no database behind it, so ":w"
	// can say so rather than appearing to work.
	Writable bool
}

func NewProblemScreen(a ProblemArgs) ProblemScreen {
	s := ProblemScreen{
		activePane: problemPane,
		langEdits:  make(map[string]string),
		width:      a.Width,
		height:     a.Height,
		err:        a.Err,
		problem:    a.Problem,
		supported:  a.Supported,
		writable:   a.Writable,
	}
	for lang, code := range a.Saved {
		s.langEdits[lang] = code
	}
	s.language = s.initialLanguage(a.Language)

	if a.Width > 0 && a.Height > 0 && a.Err == nil && a.Problem != nil {
		s = s.initViewports()
	}

	return s
}

// initialLanguage picks the buffer to open on: the configured default when the
// judge accepts it, otherwise a language this problem ships starter code for,
// otherwise the first supported one. Preferring a stubbed language is a
// convenience only — it never restricts what can be selected afterwards.
func (s ProblemScreen) initialLanguage(preferred string) string {
	langs := s.languageSet()

	for _, l := range langs {
		if l.ID == preferred {
			return l.ID
		}
	}
	if s.problem != nil {
		stubbed := s.problem.AvailableLanguages()
		for _, l := range langs {
			for _, id := range stubbed {
				if l.ID == id {
					return l.ID
				}
			}
		}
	}
	return langs[0].ID
}

func (s ProblemScreen) paneWidth() int  { return (s.width / 2) - 4 }
func (s ProblemScreen) paneHeight() int { return s.height - 4 }

func (s ProblemScreen) initViewports() ProblemScreen {
	pw, ph := s.paneWidth(), s.paneHeight()

	s.problemDescription = viewport.New(pw, ph)
	s.problemDescription.SetContent(s.problem.Format())

	s.codeEditor = editor.New(pw, ph)
	s.codeEditor = s.codeEditor.SetValue(s.stubForLang(s.language))

	s.ready = true
	return s
}

func (s ProblemScreen) stubForLang(lang string) string {
	if saved, ok := s.langEdits[lang]; ok {
		return saved
	}
	if s.problem == nil {
		return ""
	}
	if stub, ok := s.problem.GetCodeStub(lang); ok {
		return stub
	}
	// No starter code for this language — open a comment in that language
	// rather than a "//" that would be a syntax error in half of them.
	return languages.Get(lang).FallbackStub()
}

// languageSet is what the editor can cycle through: every language the judge
// accepts, plus any this problem ships a stub for that the catalog has not heard
// of. It is never narrowed to the stubbed set — a judge that takes Kotlin takes
// it whether or not this problem has Kotlin starter code.
func (s ProblemScreen) languageSet() []languages.Language {
	set := s.supported
	if len(set) == 0 {
		set, _ = languages.NewStatic(nil).Supported(context.Background())
	}
	if s.problem != nil {
		set = languages.Union(set, s.problem.AvailableLanguages())
	}

	// Anything the user already has a buffer in belongs in the list even if the
	// configured set no longer contains it. Otherwise saving a solution in a
	// language and then narrowing config would strand that work somewhere the
	// picker cannot reach.
	return languages.Union(set, s.savedLanguages())
}

// savedLanguages is the languages this problem has a buffer for, sorted so the
// list stays stable between renders.
func (s ProblemScreen) savedLanguages() []string {
	ids := make([]string, 0, len(s.langEdits))
	for id := range s.langEdits {
		ids = append(ids, id)
	}
	sort.Strings(ids)
	return ids
}

// setLanguage swaps the buffer to another language, stashing the current one's
// edits first so switching away and back does not lose work.
func (s ProblemScreen) setLanguage(id string) ProblemScreen {
	if id == "" || id == s.language {
		return s
	}
	s.langEdits[s.language] = s.codeEditor.Value()
	s.language = id
	s.codeEditor = s.codeEditor.SetValue(s.stubForLang(id))
	return s
}

// openLanguagePicker builds the modal over everything selectable, marking the
// languages this problem ships starter code for.
func (s ProblemScreen) openLanguagePicker() ProblemScreen {
	var stubbed []string
	if s.problem != nil {
		stubbed = s.problem.AvailableLanguages()
	}
	s.picker = openPicker(s.languageSet(), stubbed, s.language)
	return s
}

// runCmd processes a ":<cmd>" and returns the resulting model + command.
func (s ProblemScreen) runCmd(cmd string) (ProblemScreen, tea.Cmd) {
	cmd = strings.TrimSpace(cmd)
	switch cmd {
	case "q", "q!":
		return s, func() tea.Msg { return NavigateToMenuMsg{} }
	case "w", "wq", "x":
		saved, save := s.save()
		if cmd == "w" {
			return saved, save
		}
		// :wq and :x -- write, then leave. Both commands are sequenced so the
		// save is issued before the navigation.
		return saved, tea.Batch(save, func() tea.Msg { return NavigateBackMsg{} })
	}
	s.status = "not an editor command: :" + cmd
	return s, nil
}

// save emits a write for the current buffer. The router does the writing --
// this screen never touches the database.
func (s ProblemScreen) save() (ProblemScreen, tea.Cmd) {
	if s.problem == nil {
		s.status = "nothing to write"
		return s, nil
	}
	if !s.writable {
		s.status = "cannot write: no database (see clicode.log)"
		return s, nil
	}

	// Keep the in-memory copy in step with what was sent, so switching
	// language and back shows the same text even before the write lands.
	code := s.codeEditor.Value()
	s.langEdits[s.language] = code
	s.status = "writing..."

	id, lang := s.problem.ID, s.language
	return s, func() tea.Msg {
		return SaveSolutionMsg{ProblemID: id, Language: lang, Code: code}
	}
}

func (s ProblemScreen) Init() tea.Cmd { return nil }

func (s ProblemScreen) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		s.width = msg.Width
		s.height = msg.Height

		if !s.ready && s.err == nil && s.problem != nil {
			s = s.initViewports()
			return s, nil
		}
		if s.ready {
			pw, ph := s.paneWidth(), s.paneHeight()
			s.problemDescription.Height = ph
			s.problemDescription.Width = pw
			s.codeEditor = s.codeEditor.SetSize(pw, ph)
		}
		return s, nil

	case SolutionSavedMsg:
		if msg.Err != nil {
			s.status = "write failed: " + msg.Err.Error()
		} else {
			s.status = fmt.Sprintf("written (%s)", msg.Language)
		}
		return s, nil

	case tea.KeyMsg:
		// A keypress means the user has moved on; the previous result has been
		// read or is no longer relevant.
		s.status = ""

		// ── Language picker — modal, swallows every key while open ────────────
		if s.picker.open {
			var chosen string
			s.picker, chosen = s.picker.Update(msg)
			return s.setLanguage(chosen), nil
		}

		// ── Command mode (e.g. ":q") ──────────────────────────────────────────
		if s.cmdMode {
			switch msg.Type {
			case tea.KeyEsc:
				s.cmdMode = false
				s.cmdBuf = ""
			case tea.KeyBackspace:
				if len(s.cmdBuf) > 0 {
					s.cmdBuf = s.cmdBuf[:len(s.cmdBuf)-1]
				} else {
					s.cmdMode = false
				}
			case tea.KeyEnter:
				s.cmdMode = false
				updated, cmd := s.runCmd(s.cmdBuf)
				s.cmdBuf = ""
				return updated, cmd
			case tea.KeyRunes:
				s.cmdBuf += string(msg.Runes)
			}
			return s, nil
		}

		// ── Editor pane — INSERT mode: all keys go to the vim editor ──────────
		if s.activePane == editorPane && s.codeEditor.Mode == editor.Insert {
			var cmd tea.Cmd
			s.codeEditor, cmd = s.codeEditor.Update(msg)
			return s, cmd
		}

		// ── Editor pane — NORMAL mode ─────────────────────────────────────────
		if s.activePane == editorPane {
			switch msg.String() {
			case "ctrl+c":
				return s, func() tea.Msg { return NavigateBackMsg{} }
			case ":":
				s.cmdMode = true
				s.cmdBuf = ""
				return s, nil
			case "m":
				return s, func() tea.Msg { return NavigateToMenuMsg{} }
			case "t":
				if s.problem != nil {
					p := s.problem
					return s, func() tea.Msg { return NavigateToTestCaseMsg{Problem: p} }
				}
			case "h", "left", "ctrl+w":
				s.activePane = problemPane
				return s, nil
			case "ctrl+l":
				return s.openLanguagePicker(), nil
			default:
				// All other vim normal-mode keys (i, j, k, l, dd, yy, w, b…)
				var cmd tea.Cmd
				s.codeEditor, cmd = s.codeEditor.Update(msg)
				return s, cmd
			}
		}

		// ── Problem description pane ──────────────────────────────────────────
		switch msg.String() {
		case "ctrl+c", "q":
			return s, func() tea.Msg { return NavigateBackMsg{} }
		case "m":
			return s, func() tea.Msg { return NavigateToMenuMsg{} }
		case "t":
			if s.problem != nil {
				p := s.problem
				return s, func() tea.Msg { return NavigateToTestCaseMsg{Problem: p} }
			}
		case "l", "right", "ctrl+w":
			if s.err == nil {
				s.activePane = editorPane
			}
		case "j", "down":
			s.problemDescription.ScrollDown(1)
		case "k", "up":
			s.problemDescription.ScrollUp(1)
		case "ctrl+l":
			return s.openLanguagePicker(), nil
		}
	}
	return s, nil
}

func (s ProblemScreen) View() string {
	if s.width == 0 {
		return "Loading..."
	}
	if s.err != nil {
		return s.renderError()
	}
	if !s.ready {
		return "Loading problem..."
	}

	if s.picker.open {
		return s.picker.View(s.width, s.height, s.language)
	}

	pw := s.paneWidth()

	problemStyle := design.Border.Width(pw)
	editorStyle := design.Border.Width(pw)
	if s.activePane == problemPane {
		problemStyle = design.ActiveBorder.Width(pw)
	} else {
		editorStyle = design.ActiveBorder.Width(pw)
	}

	problemView := problemStyle.Render(s.problemDescription.View())
	editorView := editorStyle.Render(s.codeEditor.View())

	// Title shows the problem name and current language.
	lang := languages.Get(s.language).DisplayName
	if _, ok := s.problem.GetCodeStub(s.language); !ok {
		lang += " · no starter code"
	}
	title := design.Title.Render(fmt.Sprintf("CLICode — %s  [%s]", s.problem.Title, lang))

	content := lipgloss.JoinHorizontal(lipgloss.Top, problemView, editorView)

	// Bottom bar: vim mode indicator OR command input.
	var bottomBar string
	if s.cmdMode {
		bottomBar = cmdPrompt.Render(":") + cmdStyle.Render(s.cmdBuf)
	} else if s.status != "" {
		bottomBar = design.Help.Render(s.status)
	} else if s.activePane == editorPane {
		bottomBar = s.codeEditor.ModeString()
	} else {
		bottomBar = design.Help.Render("l/ctrl+w: editor  j/k: scroll  ctrl+l: language  t: tests  m: menu  q: back")
	}

	return lipgloss.JoinVertical(lipgloss.Left, title, content, bottomBar)
}

func (s ProblemScreen) renderError() string {
	box := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("196")).
		Padding(2, 4).
		Width(s.width - 10).
		Align(lipgloss.Center)

	content := lipgloss.JoinVertical(
		lipgloss.Center,
		design.Error.Render("Error loading problem"),
		"",
		s.err.Error(),
		"",
		design.Help.Render("Press 'm' for menu  |  'q' to quit"),
	)

	return lipgloss.Place(s.width, s.height, lipgloss.Center, lipgloss.Center, box.Render(content))
}
