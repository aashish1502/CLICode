package screens

import (
	"github.com/aashish1502/clicode/internal/design"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

type SettingsScreen struct {
	width  int
	height int
}

func NewSettingsScreen(width, height int) SettingsScreen {
	return SettingsScreen{width: width, height: height}
}

func (s SettingsScreen) Init() tea.Cmd { return nil }

func (s SettingsScreen) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		s.width = msg.Width
		s.height = msg.Height
		return s, nil

	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c", "q", "esc":
			return s, func() tea.Msg { return NavigateToMenuMsg{} }
		}
	}
	return s, nil
}

func (s SettingsScreen) View() string {
	if s.width == 0 {
		return "Loading..."
	}

	box := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("63")).
		Padding(2, 4).
		Render(lipgloss.JoinVertical(lipgloss.Center,
			design.Title.Render("Settings"),
			"",
			lipgloss.NewStyle().Foreground(lipgloss.Color("250")).Render("Coming soon."),
			"",
			design.Help.Render("q / esc: back to menu"),
		))

	return lipgloss.Place(s.width, s.height, lipgloss.Center, lipgloss.Center, box)
}
