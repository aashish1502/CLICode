package screens

import (
	"fmt"
	"strings"

	"github.com/aashish1502/clicode/internal/design"
	"github.com/aashish1502/clicode/internal/loader"
	"github.com/aashish1502/clicode/internal/models"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

type ProblemListScreen struct {
	problems      []models.ProblemListItem
	cursor        int
	lastProblemID int
	width         int
	height        int
	err           error
}

func NewProblemListScreen(lastProblemID, width, height int) ProblemListScreen {
	s := ProblemListScreen{
		lastProblemID: lastProblemID,
		width:         width,
		height:        height,
	}

	problems, err := loader.MakeProblemList()
	if err != nil {
		s.err = err
		return s
	}
	s.problems = problems

	// Position cursor at last worked-on problem.
	if lastProblemID > 0 {
		for i, p := range problems {
			if p.Id == lastProblemID {
				s.cursor = i
				break
			}
		}
	}

	return s
}

func (s ProblemListScreen) Init() tea.Cmd { return nil }

// Refresh updates the last-worked-on marker without resetting the cursor or
// scroll, so the list can be reused from the stack rather than rebuilt.
func (s ProblemListScreen) Refresh(lastProblemID int) tea.Model {
	s.lastProblemID = lastProblemID
	return s
}

func (s ProblemListScreen) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		s.width = msg.Width
		s.height = msg.Height
		return s, nil

	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c", "q", "esc":
			return s, func() tea.Msg { return NavigateBackMsg{} }

		case "j", "down":
			if s.cursor < len(s.problems)-1 {
				s.cursor++
			}

		case "k", "up":
			if s.cursor > 0 {
				s.cursor--
			}

		case "enter":
			if len(s.problems) > 0 {
				id := s.problems[s.cursor].Id
				return s, func() tea.Msg {
					return NavigateToProblemMsg{ProblemID: id}
				}
			}
		}
	}
	return s, nil
}

func (s ProblemListScreen) View() string {
	if s.width == 0 {
		return "Loading..."
	}

	if s.err != nil {
		return design.Error.Render(fmt.Sprintf("Error loading problem list: %v\n\nPress q to quit.", s.err))
	}

	title := design.Title.Render("CLICode — Problem List")

	header := lipgloss.NewStyle().
		Foreground(lipgloss.Color("241")).
		Render(fmt.Sprintf("  %-4s %-46s %-8s %s", "#", "Title", "Diff", ""))

	divider := strings.Repeat("─", max(s.width-4, 10))

	var rows []string
	rows = append(rows, header, divider)

	for i, p := range s.problems {
		cursor := "  "
		rowStyle := lipgloss.NewStyle()
		if i == s.cursor {
			cursor = "> "
			rowStyle = design.ListCursor
		}

		diff := diffStyle(p.Difficulty)

		status := ""
		if p.Solved {
			status += design.Solved.Render("✓")
		}
		if p.Id == s.lastProblemID && s.lastProblemID > 0 {
			status += design.LastWorkedOn.Render(" ●")
		}

		titleStr := p.Title
		if len(titleStr) > 45 {
			titleStr = titleStr[:42] + "..."
		}

		row := fmt.Sprintf("%s%-4d %-46s %-8s %s",
			cursor, p.Id, titleStr, diff, status)

		rows = append(rows, rowStyle.Render(row))
	}

	rows = append(rows, divider)

	help := design.Help.Render("j/k: navigate  Enter: solve  q: quit  ● = last worked on")

	return lipgloss.JoinVertical(lipgloss.Left,
		title,
		"",
		strings.Join(rows, "\n"),
		"",
		help,
	)
}

func diffStyle(difficulty string) string {
	switch difficulty {
	case "Easy":
		return design.DifficultyEasy.Render(difficulty)
	case "Medium":
		return design.DifficultyMedium.Render(difficulty)
	case "Hard":
		return design.DifficultyHard.Render(difficulty)
	default:
		return difficulty
	}
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}
