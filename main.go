package main

import (
	"fmt"
	"log"
	"os"

	"github.com/aashish1502/clicode/internal/loader"
	"github.com/aashish1502/clicode/internal/models"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

var (
	borderStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("63"))

	titleStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("230"))

	activePane = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("#eb650c"))

	errorStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("196")).
			Bold(true)
)

type pane int

// iota in go is a
const (
	problemPane pane = iota
	editorPane
)

type model struct {
	activePane         pane
	problem            *models.Problem
	language           string
	codeText           string
	width              int
	height             int
	problemDescription viewport.Model
	codeEditor         viewport.Model
	ready              bool
	err                error
}

type errMsg struct{ err error }

func (e errMsg) Error() string { return e.err.Error() }

func initialModel() model {
	m := model{
		activePane: problemPane,
		language:   "python",
	}

	problem, err := loader.LoadProblem(110)

	if err != nil {
		m.err = err
		return m
	}

	m.problem = problem

	codeText := m.problem.GetCodeStub(m.language)

	if codeText == "" {
		codeText = "// Write your solution here\n"
	}

	m.codeText = codeText
	return m
}

func (m model) Init() tea.Cmd {
	return nil
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height

		paneWidth := (m.width / 2) - 4
		paneHeight := m.height - 4

		if !m.ready && m.err == nil {
			m.problemDescription = viewport.New(0, 0)
			m.codeEditor = viewport.New(0, 0)
			m.ready = true

			formattedProblem, err := m.problem.FormatProblemFromProblemStruct()
			if err != nil {
				// If formatting fails, show error in the problem pane
				formattedProblem = fmt.Sprintf("Error formatting problem: %v", err)
			}
			m.problemDescription.SetContent(formattedProblem)
			m.codeEditor.SetContent(m.codeText)
		}

		m.problemDescription.Height = paneHeight
		m.problemDescription.Width = paneWidth

		m.codeEditor.Height = paneHeight
		m.codeEditor.Width = paneWidth

		return m, nil

	case errMsg:
		m.err = msg.err
		return m, nil

	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c", "q":
			return m, tea.Quit

		case "ctrl+w":
			if m.err != nil {
				return m, nil
			}
			if m.activePane == problemPane {
				m.activePane = editorPane
			} else {
				m.activePane = problemPane
			}
			return m, nil

		case "h", "left":
			if m.err == nil && m.activePane == editorPane {
				m.activePane = problemPane
			}
			return m, nil

		case "l", "right":
			if m.err == nil && m.activePane == problemPane {
				m.activePane = editorPane
			}
			return m, nil

		case "j", "down":
			if m.activePane == problemPane {
				m.problemDescription.LineDown(1)
			} else {
				m.codeEditor.LineDown(1)
			}
			return m, nil

		case "k", "up":
			if m.activePane == problemPane {
				m.problemDescription.LineUp(1)
			} else {
				m.codeEditor.LineUp(1)
			}
			return m, nil

		}
	}

	return m, nil
}

func (m model) View() string {
	if m.width == 0 {
		return "Loading... "
	}

	if m.err != nil {
		return m.renderErrorView()
	}

	problemStyle := borderStyle
	editorStyle := borderStyle

	if m.activePane == problemPane {
		problemStyle = activePane
	} else {
		editorStyle = activePane
	}

	// Handle error from FormatProblemFromProblemStruct

	problemView := problemStyle.
		PaddingLeft(1).
		Render(m.problemDescription.View())

	editorView := editorStyle.
		PaddingLeft(1).
		Render(m.codeEditor.View())

	title := titleStyle.Render(fmt.Sprintf("CLICode - %s", m.language))

	help := lipgloss.NewStyle().
		Foreground(lipgloss.Color("241")).
		Render("h/l: switch panes | ctrl+w: toggle | q: quit | :tc for test cases")

	content := lipgloss.JoinHorizontal(
		lipgloss.Top,
		problemView,
		editorView,
	)

	return lipgloss.JoinVertical(
		lipgloss.Left,
		title,
		content,
		help,
	)
}

func (m model) renderErrorView() string {
	errorBox := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("196")).
		Padding(2, 4).
		Width(m.width - 10).
		Align(lipgloss.Center)

	content := lipgloss.JoinVertical(
		lipgloss.Center,
		errorStyle.Render("❌ Error"),
		"",
		m.err.Error(),
		"",
		lipgloss.NewStyle().
			Foreground(lipgloss.Color("241")).
			Render("Press 'q' to quit"),
	)

	return lipgloss.Place(
		m.width,
		m.height,
		lipgloss.Center,
		lipgloss.Center,
		errorBox.Render(content),
	)
}

func main() {
	logFile, err := os.OpenFile("clicode.log", os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
	if err == nil {
		defer func(logFile *os.File) {
			err := logFile.Close()
			if err != nil {

			}
		}(logFile)
		log.SetOutput(logFile)
	}

	p := tea.NewProgram(
		initialModel(),
		tea.WithAltScreen(),
	)

	if _, err := p.Run(); err != nil {
		fmt.Printf("Error running CLICode: %v\n", err)
		os.Exit(1)
	}
}
