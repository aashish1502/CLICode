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
	tcListPane tcPane = iota
	tcContentPane
)

// TestCaseScreen is a 2-viewport screen.
// Left viewport: navigable list of test cases.
// Right viewport: content of the selected test case.
// Each TC's right-pane scroll offset is saved independently so switching TCs
// never carries scroll state from one case to another.
type TestCaseScreen struct {
	problem         *models.Problem
	cursor          int
	activePane      tcPane
	tcList          viewport.Model
	tcContent       viewport.Model
	tcScrollOffsets map[int]int
	width           int
	height          int
	ready           bool
}

func NewTestCaseScreen(problem *models.Problem, width, height int) TestCaseScreen {
	s := TestCaseScreen{
		problem:         problem,
		cursor:          0,
		activePane:      tcListPane,
		tcScrollOffsets: make(map[int]int),
		width:           width,
		height:          height,
	}

	if width > 0 && height > 0 && problem != nil {
		s = s.initViewports()
	}

	return s
}

func (s TestCaseScreen) listWidth() int  { return (s.width / 3) - 4 }
func (s TestCaseScreen) rightWidth() int { return (s.width * 2 / 3) - 4 }
func (s TestCaseScreen) paneHeight() int { return s.height - 4 }

func (s TestCaseScreen) initViewports() TestCaseScreen {
	s.tcList = viewport.New(s.listWidth(), s.paneHeight())
	s.tcContent = viewport.New(s.rightWidth(), s.paneHeight())

	s.tcList.SetContent(s.formatList())
	if s.problem != nil && len(s.problem.TestCases) > 0 {
		s.tcContent.SetContent(s.problem.FormatTestCase(s.cursor))
	}

	s.ready = true
	return s
}

// formatList builds the left-pane content string with a cursor indicator.
func (s TestCaseScreen) formatList() string {
	if s.problem == nil || len(s.problem.TestCases) == 0 {
		return "No test cases."
	}

	var sb strings.Builder
	for i := range s.problem.TestCases {
		if i == s.cursor {
			sb.WriteString(design.ListCursor.Render(fmt.Sprintf("> TC %d", i+1)))
		} else {
			sb.WriteString(fmt.Sprintf("  TC %d", i+1))
		}
		sb.WriteString("\n")
	}
	return sb.String()
}

// moveCursor saves the current right-pane scroll offset, moves the cursor, then
// loads the new TC's content and restores its previously saved scroll position.
func (s TestCaseScreen) moveCursor(newCursor int) TestCaseScreen {
	if s.problem == nil {
		return s
	}
	total := len(s.problem.TestCases)
	if total == 0 || newCursor < 0 || newCursor >= total {
		return s
	}

	// Save current TC's scroll offset.
	s.tcScrollOffsets[s.cursor] = s.tcContent.YOffset

	s.cursor = newCursor

	// Refresh left-pane list to show new cursor position.
	s.tcList.SetContent(s.formatList())

	// Load new TC content into right pane, then restore its saved offset.
	s.tcContent.SetContent(s.problem.FormatTestCase(s.cursor))
	s.tcContent.GotoTop()
	if saved, ok := s.tcScrollOffsets[s.cursor]; ok && saved > 0 {
		s.tcContent.ScrollDown(saved)
	}

	return s
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
			s.tcList.Height = s.paneHeight()
			s.tcList.Width = s.listWidth()
			s.tcContent.Height = s.paneHeight()
			s.tcContent.Width = s.rightWidth()
		}
		return s, nil

	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c", "q", "esc":
			return s, func() tea.Msg { return NavigateBackMsg{} }

		case "ctrl+w":
			if s.activePane == tcListPane {
				s.activePane = tcContentPane
			} else {
				s.activePane = tcListPane
			}

		case "h", "left":
			s.activePane = tcListPane

		case "l", "right":
			s.activePane = tcContentPane

		case "j", "down":
			if s.activePane == tcListPane {
				s = s.moveCursor(s.cursor + 1)
			} else {
				s.tcContent.ScrollDown(1)
			}

		case "k", "up":
			if s.activePane == tcListPane {
				s = s.moveCursor(s.cursor - 1)
			} else {
				s.tcContent.ScrollUp(1)
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

	listStyle := design.Border
	contentStyle := design.Border
	if s.activePane == tcListPane {
		listStyle = design.ActiveBorder
	} else {
		contentStyle = design.ActiveBorder
	}

	title := design.Title.Render("CLICode — Test Cases")

	listView := listStyle.PaddingLeft(1).Render(s.tcList.View())
	contentView := contentStyle.PaddingLeft(1).Render(s.tcContent.View())

	panes := lipgloss.JoinHorizontal(lipgloss.Top, listView, contentView)
	help := design.Help.Render("j/k: navigate TCs  h/l: switch pane  scroll: j/k in right pane  q/esc: back")

	return lipgloss.JoinVertical(lipgloss.Left, title, panes, help)
}
