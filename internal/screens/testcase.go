package screens

import (
	"fmt"
	"strings"

	"github.com/aashish1502/clicode/internal/design"
	"github.com/aashish1502/clicode/internal/models"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

type tcPane int

const (
	tcCardsPane tcPane = iota
	submissionsPane
)

var (
	cardBorder = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("63")).
			Padding(0, 1)

	cardLabel = lipgloss.NewStyle().
			Foreground(lipgloss.Color("39")).
			Bold(true)

	submissionHeader = lipgloss.NewStyle().
				Bold(true).
				Foreground(lipgloss.Color("230"))

	statusAccepted = lipgloss.NewStyle().
			Foreground(lipgloss.Color("82")).
			Bold(true)

	statusFailed = lipgloss.NewStyle().
			Foreground(lipgloss.Color("196")).
			Bold(true)

	statusOther = lipgloss.NewStyle().
			Foreground(lipgloss.Color("214")).
			Bold(true)

	subMeta = lipgloss.NewStyle().
		Foreground(lipgloss.Color("241"))
)

// TestCaseScreen shows all test cases as scrollable cards on the left and
// the submission history on the right.
type TestCaseScreen struct {
	problem     *models.Problem
	activePane  tcPane
	tcViewport  viewport.Model
	subViewport viewport.Model
	width       int
	height      int
	ready       bool
}

func NewTestCaseScreen(problem *models.Problem, width, height int) TestCaseScreen {
	s := TestCaseScreen{
		problem:    problem,
		activePane: tcCardsPane,
		width:      width,
		height:     height,
	}

	if width > 0 && height > 0 && problem != nil {
		s = s.initViewports()
	}

	return s
}

func (s TestCaseScreen) cardsWidth() int { return (s.width * 3 / 5) - 4 }
func (s TestCaseScreen) subWidth() int   { return (s.width * 2 / 5) - 4 }
func (s TestCaseScreen) paneHeight() int { return s.height - 4 }

func (s TestCaseScreen) initViewports() TestCaseScreen {
	s.tcViewport = viewport.New(s.cardsWidth(), s.paneHeight())
	s.subViewport = viewport.New(s.subWidth(), s.paneHeight())
	s.tcViewport.SetContent(s.formatCards())
	s.subViewport.SetContent(s.formatSubmissions())
	s.ready = true
	return s
}

func (s TestCaseScreen) formatCards() string {
	if s.problem == nil || len(s.problem.TestCases) == 0 {
		return "No test cases."
	}

	// Card content width: viewport width minus border (2) and padding (2).
	contentWidth := s.cardsWidth() - 4
	if contentWidth < 10 {
		contentWidth = 10
	}

	style := cardBorder.Width(contentWidth)

	var cards []string
	for i, tc := range s.problem.TestCases {
		title := design.Title.Render(fmt.Sprintf("Test Case %d", i+1))

		input := lipgloss.JoinVertical(lipgloss.Left,
			cardLabel.Render("Input"),
			tc.Input,
		)
		expected := lipgloss.JoinVertical(lipgloss.Left,
			cardLabel.Render("Expected"),
			tc.ExpectedOutput,
		)

		body := lipgloss.JoinVertical(lipgloss.Left,
			title, "", input, "", expected,
		)

		cards = append(cards, style.Render(body))
	}

	return strings.Join(cards, "\n\n")
}

func (s TestCaseScreen) formatSubmissions() string {
	contentWidth := s.subWidth() - 4
	if contentWidth < 8 {
		contentWidth = 8
	}
	divider := strings.Repeat("─", contentWidth)

	var lines []string
	lines = append(lines, submissionHeader.Render("Submissions"))
	lines = append(lines, divider)

	if s.problem == nil || len(s.problem.Submissions) == 0 {
		lines = append(lines, subMeta.Render("No submissions yet."))
		return strings.Join(lines, "\n")
	}

	for _, sub := range s.problem.Submissions {
		var statusStyle lipgloss.Style
		var icon string
		switch sub.Status {
		case "Accepted":
			statusStyle = statusAccepted
			icon = "✓"
		case "Wrong Answer":
			statusStyle = statusFailed
			icon = "✗"
		default:
			statusStyle = statusOther
			icon = "⏱"
		}

		lines = append(lines,
			statusStyle.Render(fmt.Sprintf("%s  %s", icon, sub.Status)),
			subMeta.Render(fmt.Sprintf("   %s  ·  %s  ·  %s", sub.Language, sub.Runtime, sub.Memory)),
			subMeta.Render(fmt.Sprintf("   %s", sub.Timestamp)),
			"",
		)
	}

	return strings.Join(lines, "\n")
}

func (s TestCaseScreen) Init() tea.Cmd { return nil }

func (s TestCaseScreen) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		s.width = msg.Width
		s.height = msg.Height
		if !s.ready && s.problem != nil {
			s = s.initViewports()
		} else if s.ready {
			s.tcViewport.Width = s.cardsWidth()
			s.tcViewport.Height = s.paneHeight()
			s.subViewport.Width = s.subWidth()
			s.subViewport.Height = s.paneHeight()
		}
		return s, nil

	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c", "q", "esc":
			return s, func() tea.Msg { return NavigateBackMsg{} }

		case "ctrl+w":
			if s.activePane == tcCardsPane {
				s.activePane = submissionsPane
			} else {
				s.activePane = tcCardsPane
			}

		case "h", "left":
			s.activePane = tcCardsPane

		case "l", "right":
			s.activePane = submissionsPane

		case "j", "down":
			if s.activePane == tcCardsPane {
				s.tcViewport.ScrollDown(1)
			} else {
				s.subViewport.ScrollDown(1)
			}

		case "k", "up":
			if s.activePane == tcCardsPane {
				s.tcViewport.ScrollUp(1)
			} else {
				s.subViewport.ScrollUp(1)
			}
		}
	}
	return s, nil
}

func (s TestCaseScreen) View() string {
	if s.width == 0 {
		return "Loading..."
	}

	if !s.ready {
		return "Loading test cases..."
	}

	tcStyle := design.Border
	subStyle := design.Border
	if s.activePane == tcCardsPane {
		tcStyle = design.ActiveBorder
	} else {
		subStyle = design.ActiveBorder
	}

	title := design.Title.Render("CLICode — Test Cases & Submissions")

	tcView := tcStyle.PaddingLeft(1).Render(s.tcViewport.View())
	subView := subStyle.PaddingLeft(1).Render(s.subViewport.View())

	panes := lipgloss.JoinHorizontal(lipgloss.Top, tcView, subView)
	help := design.Help.Render("j/k: scroll  h/l: switch pane  ctrl+w: toggle  q/esc: back to problem")

	return lipgloss.JoinVertical(lipgloss.Left, title, panes, help)
}
