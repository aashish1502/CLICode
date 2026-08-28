package screens

import (
	"strings"

	"github.com/aashish1502/clicode/internal/design"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

type menuOption int

const (
	menuContinue menuOption = iota
	menuProblemList
	menuSettings
	menuQuit
)

var (
	menuItemActive = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#eb650c")).
			Bold(true)

	menuItemNormal = lipgloss.NewStyle().
			Foreground(lipgloss.Color("250"))

	menuItemDimmed = lipgloss.NewStyle().
			Foreground(lipgloss.Color("240"))

	menuBox = lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("63")).
		Padding(1, 4)
)

type MenuScreen struct {
	cursor        int
	lastProblemID int
	width         int
	height        int
}

func NewMenuScreen(lastProblemID, width, height int) MenuScreen {
	return MenuScreen{
		cursor:        0,
		lastProblemID: lastProblemID,
		width:         width,
		height:        height,
	}
}

func (s MenuScreen) Init() tea.Cmd { return nil }

// Refresh updates the last-worked-on marker without disturbing the cursor, so
// the menu can be reused from the stack instead of rebuilt on every visit.
func (s MenuScreen) Refresh(lastProblemID int) tea.Model {
	s.lastProblemID = lastProblemID
	return s
}

func (s MenuScreen) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		s.width = msg.Width
		s.height = msg.Height
		return s, nil

	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c", "esc":
			return s, func() tea.Msg { return NavigateBackMsg{} }

		case "j", "down":
			if s.cursor < int(menuQuit) {
				s.cursor++
			}

		case "k", "up":
			if s.cursor > 0 {
				s.cursor--
			}

		case "enter", " ":
			return s, s.activate()
		}
	}
	return s, nil
}

func (s MenuScreen) activate() tea.Cmd {
	switch menuOption(s.cursor) {
	case menuContinue:
		if s.lastProblemID > 0 {
			id := s.lastProblemID
			return func() tea.Msg { return NavigateToProblemMsg{ProblemID: id} }
		}
		return func() tea.Msg { return NavigateToProblemListMsg{} }

	case menuProblemList:
		return func() tea.Msg { return NavigateToProblemListMsg{} }

	case menuSettings:
		return func() tea.Msg { return NavigateToSettingsMsg{} }

	case menuQuit:
		return tea.Quit
	}
	return nil
}

func (s MenuScreen) View() string {
	if s.width == 0 {
		return "Loading..."
	}

	type entry struct {
		label   string
		sub     string // optional dim subtitle
		enabled bool
	}

	continueLabel := "Continue where you left off"
	continueSub := ""
	if s.lastProblemID == 0 {
		continueSub = "no recent problem"
	}

	entries := []entry{
		{continueLabel, continueSub, true},
		{"Problem List", "", true},
		{"Settings", "", true},
		{"Quit", "", true},
	}

	var rows []string
	for i, e := range entries {
		cursor := "  "
		if i == s.cursor {
			cursor = "▶ "
		}

		var line string
		switch {
		case i == s.cursor:
			line = menuItemActive.Render(cursor + e.label)
		default:
			line = menuItemNormal.Render(cursor + e.label)
		}

		if e.sub != "" {
			sub := menuItemDimmed.Render("    " + e.sub)
			line = lipgloss.JoinVertical(lipgloss.Left, line, sub)
		}

		rows = append(rows, line)
		if i < len(entries)-1 {
			rows = append(rows, "")
		}
	}

	box := menuBox.Render(strings.Join(rows, "\n"))

	title := design.Title.Render("CLICode")
	help := design.Help.Render("j/k: navigate  Enter: select  ctrl+q: quit")

	block := lipgloss.JoinVertical(lipgloss.Center,
		title,
		"",
		box,
		"",
		help,
	)

	return lipgloss.Place(s.width, s.height, lipgloss.Center, lipgloss.Center, block)
}
