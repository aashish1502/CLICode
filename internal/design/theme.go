package design

import "github.com/charmbracelet/lipgloss"

var (
	Border = lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("63"))

	ActiveBorder = lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("#eb650c"))

	Title = lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("230"))

	Help = lipgloss.NewStyle().
		Foreground(lipgloss.Color("241"))

	Error = lipgloss.NewStyle().
		Foreground(lipgloss.Color("196")).
		Bold(true)

	ListCursor = lipgloss.NewStyle().
		Foreground(lipgloss.Color("#eb650c")).
		Bold(true)

	DifficultyEasy = lipgloss.NewStyle().
		Foreground(lipgloss.Color("82"))

	DifficultyMedium = lipgloss.NewStyle().
		Foreground(lipgloss.Color("214"))

	DifficultyHard = lipgloss.NewStyle().
		Foreground(lipgloss.Color("196"))

	Solved = lipgloss.NewStyle().
		Foreground(lipgloss.Color("82"))

	LastWorkedOn = lipgloss.NewStyle().
		Foreground(lipgloss.Color("39")).
		Bold(true)
)
