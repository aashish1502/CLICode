package screens

import (
	"fmt"
	"strings"

	"github.com/aashish1502/clicode/internal/design"
	"github.com/aashish1502/clicode/internal/editor"
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

var knownLanguages = []string{"python", "cpp", "go"}

var (
	cmdStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("250"))
	cmdPrompt = lipgloss.NewStyle().Foreground(lipgloss.Color("214")).Bold(true)
)

type ProblemScreen struct {
	activePane         pane
	problem            *models.Problem
	language           string
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
}

func NewProblemScreen(problem *models.Problem, err error, width, height int, language string) ProblemScreen {
	if language == "" {
		language = "python"
	}
	s := ProblemScreen{
		activePane: problemPane,
		language:   language,
		langEdits:  make(map[string]string),
		width:      width,
		height:     height,
		err:        err,
		problem:    problem,
	}

	if width > 0 && height > 0 && err == nil && problem != nil {
		s = s.initViewports()
	}

	return s
}

func (s ProblemScreen) paneWidth() int  { return (s.width / 2) - 4 }
func (s ProblemScreen) paneHeight() int { return s.height - 4 }

func (s ProblemScreen) initViewports() ProblemScreen {
	pw, ph := s.paneWidth(), s.paneHeight()

	s.problemDescription = viewport.New(pw, ph)
	formatted, fmtErr := s.problem.FormatProblemFromProblemStruct()
	if fmtErr != nil {
		formatted = fmt.Sprintf("Error formatting problem: %v", fmtErr)
	}
	s.problemDescription.SetContent(formatted)

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
	stub := s.problem.GetCodeStub(lang)
	if stub == "" {
		stub = "// Write your solution here\n"
	}
	return stub
}

func (s ProblemScreen) switchLanguage(delta int) ProblemScreen {
	s.langEdits[s.language] = s.codeEditor.Value()
	idx := 0
	for i, l := range knownLanguages {
		if l == s.language {
			idx = i
			break
		}
	}
	idx = ((idx + delta) % len(knownLanguages) + len(knownLanguages)) % len(knownLanguages)
	s.language = knownLanguages[idx]
	s.codeEditor = s.codeEditor.SetValue(s.stubForLang(s.language))
	return s
}

// runCmd processes a ":<cmd>" and returns the resulting model + command.
func (s ProblemScreen) runCmd(cmd string) (ProblemScreen, tea.Cmd) {
	cmd = strings.TrimSpace(cmd)
	switch cmd {
	case "q", "q!":
		return s, func() tea.Msg { return NavigateToMenuMsg{} }
	case "w":
		// future: save to disk
	}
	return s, nil
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

	case tea.KeyMsg:
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
				return s, tea.Quit
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
				s = s.switchLanguage(1)
				return s, nil
			case "ctrl+h":
				s = s.switchLanguage(-1)
				return s, nil
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
			return s, tea.Quit
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
			s = s.switchLanguage(1)
		case "ctrl+h":
			s = s.switchLanguage(-1)
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
	title := design.Title.Render(fmt.Sprintf("CLICode — %s  [%s]", s.problem.Title, s.language))

	content := lipgloss.JoinHorizontal(lipgloss.Top, problemView, editorView)

	// Bottom bar: vim mode indicator OR command input.
	var bottomBar string
	if s.cmdMode {
		bottomBar = cmdPrompt.Render(":") + cmdStyle.Render(s.cmdBuf)
	} else if s.activePane == editorPane {
		bottomBar = s.codeEditor.ModeString()
	} else {
		bottomBar = design.Help.Render("l/ctrl+w: editor  j/k: scroll  ctrl+l/h: lang  t: tests  m: menu  q: quit")
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
