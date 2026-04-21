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

type tcPane int

const (
	tcCardsPane tcPane = iota
	submissionsPane
)

// editField tracks which textarea is active during TC edit mode.
type editField int

const (
	editInput    editField = iota
	editExpected           // nolint:deadcode,varcheck
)

var (
	cardBorder = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("63")).
			Padding(0, 1)

	cardBorderSelected = lipgloss.NewStyle().
				Border(lipgloss.RoundedBorder()).
				BorderForeground(lipgloss.Color("#eb650c")).
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

// TestCaseScreen shows test cases as scrollable cards on the left (with a
// cursor and inline edit mode) and submission history on the right.
type TestCaseScreen struct {
	problem     *models.Problem
	activePane  tcPane
	tcViewport  viewport.Model
	subViewport viewport.Model

	// cursor / edit state
	selectedTC     int
	editMode       bool
	activeField    editField
	inputEditor    textarea.Model
	expectedEditor textarea.Model
	tcEdits        map[int][2]string // local overrides: index → [input, expected]

	width  int
	height int
	ready  bool
}

func NewTestCaseScreen(problem *models.Problem, width, height int) TestCaseScreen {
	s := TestCaseScreen{
		problem:    problem,
		activePane: tcCardsPane,
		tcEdits:    make(map[int][2]string),
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

// tcInput returns the (possibly edited) input for the given TC index.
func (s TestCaseScreen) tcInput(i int) string {
	if edit, ok := s.tcEdits[i]; ok {
		return edit[0]
	}
	if s.problem == nil || i >= len(s.problem.TestCases) {
		return ""
	}
	return s.problem.TestCases[i].Input
}

// tcExpected returns the (possibly edited) expected output for the given TC index.
func (s TestCaseScreen) tcExpected(i int) string {
	if edit, ok := s.tcEdits[i]; ok {
		return edit[1]
	}
	if s.problem == nil || i >= len(s.problem.TestCases) {
		return ""
	}
	return s.problem.TestCases[i].ExpectedOutput
}

func (s TestCaseScreen) formatCards() string {
	if s.problem == nil || len(s.problem.TestCases) == 0 {
		return "No test cases."
	}

	contentWidth := s.cardsWidth() - 4
	if contentWidth < 10 {
		contentWidth = 10
	}

	var cards []string
	for i := range s.problem.TestCases {
		style := cardBorder.Width(contentWidth)
		if i == s.selectedTC {
			style = cardBorderSelected.Width(contentWidth)
		}

		cursor := "  "
		if i == s.selectedTC {
			cursor = "▶ "
		}

		title := design.Title.Render(fmt.Sprintf("%sTest Case %d", cursor, i+1))

		input := lipgloss.JoinVertical(lipgloss.Left,
			cardLabel.Render("Input"),
			s.tcInput(i),
		)
		expected := lipgloss.JoinVertical(lipgloss.Left,
			cardLabel.Render("Expected"),
			s.tcExpected(i),
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

// enterEditMode initialises the two textareas for the currently selected TC.
func (s TestCaseScreen) enterEditMode() TestCaseScreen {
	editorWidth := s.cardsWidth() - 6
	editorHeight := (s.paneHeight() / 2) - 4
	if editorHeight < 3 {
		editorHeight = 3
	}

	s.inputEditor = textarea.New()
	s.inputEditor.SetWidth(editorWidth)
	s.inputEditor.SetHeight(editorHeight)
	s.inputEditor.SetValue(s.tcInput(s.selectedTC))
	s.inputEditor.Focus()

	s.expectedEditor = textarea.New()
	s.expectedEditor.SetWidth(editorWidth)
	s.expectedEditor.SetHeight(editorHeight)
	s.expectedEditor.SetValue(s.tcExpected(s.selectedTC))
	s.expectedEditor.Blur()

	s.activeField = editInput
	s.editMode = true
	return s
}

// saveAndExitEdit persists textarea values back into tcEdits and returns to card view.
func (s TestCaseScreen) saveAndExitEdit() TestCaseScreen {
	s.tcEdits[s.selectedTC] = [2]string{
		s.inputEditor.Value(),
		s.expectedEditor.Value(),
	}
	s.editMode = false
	s.inputEditor.Blur()
	s.expectedEditor.Blur()
	s.tcViewport.SetContent(s.formatCards())
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
			s.tcViewport.Width = s.cardsWidth()
			s.tcViewport.Height = s.paneHeight()
			s.subViewport.Width = s.subWidth()
			s.subViewport.Height = s.paneHeight()
		}
		return s, nil

	case tea.KeyMsg:
		// ── Edit mode ────────────────────────────────────────────────────────
		if s.editMode {
			switch msg.String() {
			case "esc":
				s = s.saveAndExitEdit()
				return s, nil

			case "tab":
				// switch between input and expected fields
				if s.activeField == editInput {
					s.inputEditor.Blur()
					s.expectedEditor.Focus()
					s.activeField = editExpected
				} else {
					s.expectedEditor.Blur()
					s.inputEditor.Focus()
					s.activeField = editInput
				}
				return s, nil
			}

			// forward keystrokes to the active textarea
			var cmd tea.Cmd
			if s.activeField == editInput {
				s.inputEditor, cmd = s.inputEditor.Update(msg)
			} else {
				s.expectedEditor, cmd = s.expectedEditor.Update(msg)
			}
			return s, cmd
		}

		// ── Normal mode ──────────────────────────────────────────────────────
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

		case "i", "enter":
			if s.activePane == tcCardsPane && s.problem != nil && len(s.problem.TestCases) > 0 {
				s = s.enterEditMode()
				return s, nil
			}

		case "j", "down":
			if s.activePane == tcCardsPane {
				if s.problem != nil && s.selectedTC < len(s.problem.TestCases)-1 {
					s.selectedTC++
					s.tcViewport.SetContent(s.formatCards())
				}
			} else {
				s.subViewport.ScrollDown(1)
			}

		case "k", "up":
			if s.activePane == tcCardsPane {
				if s.selectedTC > 0 {
					s.selectedTC--
					s.tcViewport.SetContent(s.formatCards())
				}
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

	var leftContent string
	if s.editMode {
		inputLabel := cardLabel.Render("Input")
		expectedLabel := cardLabel.Render("Expected")
		if s.activeField == editInput {
			inputLabel = design.ActiveBorder.Render(cardLabel.Render("Input ✎"))
		} else {
			expectedLabel = design.ActiveBorder.Render(cardLabel.Render("Expected ✎"))
		}
		leftContent = lipgloss.JoinVertical(lipgloss.Left,
			design.Title.Render(fmt.Sprintf("Editing Test Case %d", s.selectedTC+1)),
			"",
			inputLabel,
			s.inputEditor.View(),
			"",
			expectedLabel,
			s.expectedEditor.View(),
			"",
			design.Help.Render("Tab: switch field  Esc: save & back"),
		)
	} else {
		leftContent = s.tcViewport.View()
	}

	tcView := tcStyle.PaddingLeft(1).Render(leftContent)
	subView := subStyle.PaddingLeft(1).Render(s.subViewport.View())

	panes := lipgloss.JoinHorizontal(lipgloss.Top, tcView, subView)
	help := design.Help.Render("j/k: navigate  i/Enter: edit  h/l: switch pane  q/esc: back")

	return lipgloss.JoinVertical(lipgloss.Left, title, panes, help)
}
