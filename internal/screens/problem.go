package screens

import (
	"fmt"
	"strings"

	"github.com/aashish1502/clicode/internal/design"
	"github.com/aashish1502/clicode/internal/models"
	"github.com/charmbracelet/bubbles/textarea"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

type pane int

const (
	problemPane pane = iota
	editorPane
)

const tabSpace = 4

type ProblemScreen struct {
	activePane         pane
	problem            *models.Problem
	language           string
	width              int
	height             int
	problemDescription viewport.Model
	codeEditor         textarea.Model
	editingMode        bool
	ready              bool
	err                error
}

func NewProblemScreen(problem *models.Problem, err error, width, height int) ProblemScreen {
	s := ProblemScreen{
		activePane: problemPane,
		language:   "python",
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

func (s ProblemScreen) initViewports() ProblemScreen {
	paneWidth := (s.width / 2) - 4
	paneHeight := s.height - 4

	s.problemDescription = viewport.New(paneWidth, paneHeight)
	formatted, fmtErr := s.problem.FormatProblemFromProblemStruct()
	if fmtErr != nil {
		formatted = fmt.Sprintf("Error formatting problem: %v", fmtErr)
	}
	s.problemDescription.SetContent(formatted)

	s.codeEditor = textarea.New()
	codeText := s.problem.GetCodeStub(s.language)
	if codeText == "" {
		codeText = "// Write your solution here\n"
	}
	s.codeEditor.SetValue(codeText)
	s.codeEditor.SetHeight(paneHeight)
	s.codeEditor.SetWidth(paneWidth)

	s.ready = true
	return s
}

func (s ProblemScreen) Init() tea.Cmd { return nil }

func (s ProblemScreen) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		s.width = msg.Width
		s.height = msg.Height
		paneWidth := (s.width / 2) - 4
		paneHeight := s.height - 4

		if !s.ready && s.err == nil && s.problem != nil {
			s = s.initViewports()
		}

		if s.ready {
			s.problemDescription.Height = paneHeight
			s.problemDescription.Width = paneWidth
			s.codeEditor.SetHeight(paneHeight)
			s.codeEditor.SetWidth(paneWidth)
		}
		return s, nil

	case tea.KeyMsg:
		if s.editingMode {
			var cmd tea.Cmd
			switch msg.String() {
			case "tab":
				s.codeEditor, cmd = s.codeEditor.Update(tea.KeyMsg{
					Type:  tea.KeyRunes,
					Runes: []rune(strings.Repeat(" ", tabSpace)),
				})
			case "esc":
				s.editingMode = false
				s.codeEditor.Blur()
			default:
				s.codeEditor, cmd = s.codeEditor.Update(msg)
			}
			return s, cmd
		}

		switch msg.String() {
		case "ctrl+c", "q":
			return s, tea.Quit

		case "m":
			return s, func() tea.Msg { return NavigateToProblemListMsg{} }

		case "t":
			if s.problem != nil {
				p := s.problem
				return s, func() tea.Msg { return NavigateToTestCaseMsg{Problem: p} }
			}

		case "ctrl+w":
			if s.err == nil {
				if s.activePane == problemPane {
					s.activePane = editorPane
				} else {
					s.activePane = problemPane
				}
			}

		case "h", "left":
			if s.err == nil && s.activePane == editorPane {
				s.activePane = problemPane
			}

		case "l", "right":
			if s.err == nil && s.activePane == problemPane {
				s.activePane = editorPane
			}

		case "j", "down":
			if s.activePane == problemPane {
				s.problemDescription.ScrollDown(1)
			} else {
				s.codeEditor.CursorDown()
			}

		case "k", "up":
			if s.activePane == problemPane {
				s.problemDescription.ScrollUp(1)
			} else {
				s.codeEditor.CursorUp()
			}

		case "i":
			if s.activePane == editorPane {
				s.editingMode = true
				s.codeEditor.Focus()
			}
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

	problemStyle := design.Border
	editorStyle := design.Border
	if s.activePane == problemPane {
		problemStyle = design.ActiveBorder
	} else {
		editorStyle = design.ActiveBorder
	}

	problemView := problemStyle.PaddingLeft(1).Render(s.problemDescription.View())
	editorView := editorStyle.PaddingLeft(1).Render(s.codeEditor.View())

	title := design.Title.Render(fmt.Sprintf("CLICode — %s  [%s]", s.problem.Title, s.language))
	help := design.Help.Render("h/l: switch  j/k: scroll  i: edit  t: test cases  m: menu  q: quit")

	content := lipgloss.JoinHorizontal(lipgloss.Top, problemView, editorView)
	return lipgloss.JoinVertical(lipgloss.Left, title, content, help)
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
		design.Help.Render("Press 'm' for problem list  |  'q' to quit"),
	)

	return lipgloss.Place(s.width, s.height, lipgloss.Center, lipgloss.Center, box.Render(content))
}
