package main

import (
	"fmt"
	"log"
	"os"
	"strings"

	"github.com/aashish1502/clicode/internal/loader"
	"github.com/aashish1502/clicode/internal/models"
	"github.com/charmbracelet/bubbles/textarea"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// =============================================================================
// Global Styles
// =============================================================================

var (
	borderStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("63"))

	activePaneStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("#eb650c"))

	titleStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("230"))

	errorStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("196")).
			Bold(true)

	helpStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("241"))

	selectedItemStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("#eb650c")).
				Bold(true)

	menuItemStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("250"))

	cmdStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("214"))

	fieldLabelStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("39")).
			Bold(true)

	activeFieldLabelStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("#eb650c")).
				Bold(true)
)

// =============================================================================
// Types & Constants
// =============================================================================

type pane int
type screen int
type tcPane int
type cardField int

const (
	problemPane pane = iota
	editorPane
)

const (
	menuScreen screen = iota
	problemScreen
	testCaseScreen
)

const (
	tcCasesPane tcPane = iota
	tcSubsPane
)

const (
	tcInputField cardField = iota
	tcExpectedField
)

const TabSpace = 4

var menuItems = []string{
	"Solve a Problem",
	"Problem List",
	"Quit",
}

const noSubmissionsArt = `
     _____
    /     \
   ( >   < )
    \  ∧  /
     |   |
     |___|

 No Submissions Yet!

 Solve it and submit~`

// =============================================================================
// Messages
// =============================================================================

type switchScreenMsg struct{ target screen }
type openTestCasesMsg struct{ testCases []models.TestCase }
type errMsg struct{ err error }

func (e errMsg) Error() string { return e.err.Error() }

func switchScreen(s screen) tea.Cmd {
	return func() tea.Msg { return switchScreenMsg{target: s} }
}

func openTestCases(tc []models.TestCase) tea.Cmd {
	return func() tea.Msg { return openTestCasesMsg{testCases: tc} }
}

// =============================================================================
// Root App Model
// =============================================================================

type appModel struct {
	currentScreen screen
	width         int
	height        int
	menu          menuScreenModel
	problem       problemScreenModel
	testCase      testCaseScreenModel
}

func initialAppModel() appModel {
	return appModel{
		currentScreen: menuScreen,
		menu:          initialMenuModel(),
		problem:       initialProblemModel(),
		testCase:      testCaseScreenModel{activePane: tcCasesPane},
	}
}

func (a appModel) Init() tea.Cmd { return nil }

func (a appModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case switchScreenMsg:
		a.currentScreen = msg.target
		return a, nil

	case openTestCasesMsg:
		a.testCase = initialTestCaseModel(msg.testCases)
		if a.width > 0 {
			updated, _ := a.testCase.Update(tea.WindowSizeMsg{Width: a.width, Height: a.height})
			a.testCase = updated.(testCaseScreenModel)
		}
		a.currentScreen = testCaseScreen
		return a, nil

	case tea.WindowSizeMsg:
		a.width = msg.Width
		a.height = msg.Height
		u1, c1 := a.menu.Update(msg)
		a.menu = u1.(menuScreenModel)
		u2, c2 := a.problem.Update(msg)
		a.problem = u2.(problemScreenModel)
		u3, c3 := a.testCase.Update(msg)
		a.testCase = u3.(testCaseScreenModel)
		return a, tea.Batch(c1, c2, c3)
	}

	switch a.currentScreen {
	case menuScreen:
		updated, cmd := a.menu.Update(msg)
		a.menu = updated.(menuScreenModel)
		return a, cmd
	case problemScreen:
		updated, cmd := a.problem.Update(msg)
		a.problem = updated.(problemScreenModel)
		return a, cmd
	case testCaseScreen:
		updated, cmd := a.testCase.Update(msg)
		a.testCase = updated.(testCaseScreenModel)
		return a, cmd
	}

	return a, nil
}

func (a appModel) View() string {
	switch a.currentScreen {
	case menuScreen:
		return a.menu.View()
	case problemScreen:
		return a.problem.View()
	case testCaseScreen:
		return a.testCase.View()
	}
	return ""
}

// =============================================================================
// Menu Screen
// =============================================================================

type menuScreenModel struct {
	cursor int
	width  int
	height int
}

func initialMenuModel() menuScreenModel { return menuScreenModel{} }

func (m menuScreenModel) Init() tea.Cmd { return nil }

func (m menuScreenModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height

	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c", "q":
			return m, tea.Quit
		case "j", "down":
			if m.cursor < len(menuItems)-1 {
				m.cursor++
			}
		case "k", "up":
			if m.cursor > 0 {
				m.cursor--
			}
		case "enter":
			switch menuItems[m.cursor] {
			case "Solve a Problem":
				return m, switchScreen(problemScreen)
			case "Quit":
				return m, tea.Quit
			}
		}
	}
	return m, nil
}

func (m menuScreenModel) View() string {
	if m.width == 0 {
		return "Loading..."
	}

	var items strings.Builder
	for i, item := range menuItems {
		if i == m.cursor {
			items.WriteString(selectedItemStyle.Render("> " + item))
		} else {
			items.WriteString(menuItemStyle.Render("  " + item))
		}
		items.WriteString("\n")
	}

	box := borderStyle.Padding(1, 6).Render(
		lipgloss.JoinVertical(lipgloss.Left,
			titleStyle.Render("CLICode"),
			"",
			items.String(),
		),
	)

	help := helpStyle.Render("j/k: navigate  |  enter: select  |  q: quit")

	return lipgloss.Place(
		m.width, m.height,
		lipgloss.Center, lipgloss.Center,
		lipgloss.JoinVertical(lipgloss.Center, box, "", help),
	)
}

// =============================================================================
// Problem Screen
// =============================================================================

type problemScreenModel struct {
	activePane         pane
	problem            *models.Problem
	language           string
	codeText           string
	width              int
	height             int
	problemDescription viewport.Model
	codeEditor         textarea.Model
	editingMode        bool
	cmdMode            bool
	cmdBuf             string
	ready              bool
	err                error
}

func initialProblemModel() problemScreenModel {
	m := problemScreenModel{
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

func (m problemScreenModel) Init() tea.Cmd { return nil }

func countLeadingSpace(line string) int {
	indent := 0
	for _, ch := range line {
		if ch == ' ' {
			indent++
		} else {
			break
		}
	}
	return indent
}

func (m problemScreenModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		pw := (m.width / 2) - 4
		ph := m.height - 4

		if !m.ready && m.err == nil {
			m.problemDescription = viewport.New(pw, ph)
			m.codeEditor = textarea.New()
			m.ready = true

			formattedProblem, err := m.problem.FormatProblemFromProblemStruct()
			if err != nil {
				formattedProblem = fmt.Sprintf("Error formatting problem: %v", err)
			}
			m.problemDescription.SetContent(formattedProblem)
			m.codeEditor.SetValue(m.codeText)
		}

		m.problemDescription.Height = ph
		m.problemDescription.Width = pw
		m.codeEditor.SetHeight(ph)
		m.codeEditor.SetWidth(pw)
		return m, nil

	case errMsg:
		m.err = msg.err
		return m, nil

	case tea.KeyMsg:
		// Code editor insert mode
		if m.editingMode {
			var cmd tea.Cmd
			switch msg.String() {
			case "tab":
				m.codeEditor, cmd = m.codeEditor.Update(tea.KeyMsg{
					Type:  tea.KeyRunes,
					Runes: []rune(strings.Repeat(" ", TabSpace)),
				})
			case "esc":
				m.editingMode = false
				m.codeEditor.Blur()
			default:
				m.codeEditor, cmd = m.codeEditor.Update(msg)
			}
			return m, cmd
		}

		// Vim command mode
		if m.cmdMode {
			switch msg.String() {
			case "esc":
				m.cmdMode = false
				m.cmdBuf = ""
			case "enter":
				cmd := m.executeCommand(m.cmdBuf)
				m.cmdMode = false
				m.cmdBuf = ""
				return m, cmd
			case "backspace":
				if len(m.cmdBuf) > 0 {
					m.cmdBuf = m.cmdBuf[:len(m.cmdBuf)-1]
				}
			default:
				if len(msg.String()) == 1 {
					m.cmdBuf += msg.String()
				}
			}
			return m, nil
		}

		// Normal mode
		switch msg.String() {
		case "ctrl+c":
			return m, tea.Quit
		case "q":
			return m, switchScreen(menuScreen)
		case ":":
			m.cmdMode = true
			m.cmdBuf = ""
		case "ctrl+w":
			if m.err == nil {
				if m.activePane == problemPane {
					m.activePane = editorPane
				} else {
					m.activePane = problemPane
				}
			}
		case "h", "left":
			if m.err == nil && m.activePane == editorPane {
				m.activePane = problemPane
			}
		case "l", "right":
			if m.err == nil && m.activePane == problemPane {
				m.activePane = editorPane
			}
		case "j", "down":
			if m.activePane == problemPane {
				m.problemDescription.ScrollDown(1)
			} else {
				m.codeEditor.CursorDown()
			}
		case "k", "up":
			if m.activePane == problemPane {
				m.problemDescription.ScrollUp(1)
			} else {
				m.codeEditor.CursorUp()
			}
		case "i":
			if m.activePane == editorPane {
				m.editingMode = true
				m.codeEditor.Focus()
			}
		}
	}

	return m, nil
}

func (m problemScreenModel) executeCommand(cmd string) tea.Cmd {
	switch strings.TrimSpace(cmd) {
	case "tc":
		if m.problem != nil {
			return openTestCases(m.problem.TestCases)
		}
	case "q":
		return switchScreen(menuScreen)
	}
	return nil
}

func (m problemScreenModel) View() string {
	if m.width == 0 {
		return "Loading..."
	}
	if m.err != nil {
		return m.renderErrorView()
	}

	problemStyle := borderStyle
	editorStyle := borderStyle
	if m.activePane == problemPane {
		problemStyle = activePaneStyle
	} else {
		editorStyle = activePaneStyle
	}

	problemView := problemStyle.PaddingLeft(1).Render(m.problemDescription.View())
	editorView := editorStyle.PaddingLeft(1).Render(m.codeEditor.View())

	title := titleStyle.Render(fmt.Sprintf("CLICode - %s", m.language))

	var bottom string
	if m.cmdMode {
		bottom = cmdStyle.Render(":" + m.cmdBuf + "█")
	} else {
		bottom = helpStyle.Render("h/l: panes  |  j/k: scroll  |  ctrl+w: toggle  |  i: edit  |  :tc test cases  |  q: menu")
	}

	content := lipgloss.JoinHorizontal(lipgloss.Top, problemView, editorView)
	return lipgloss.JoinVertical(lipgloss.Left, title, content, bottom)
}

func (m problemScreenModel) renderErrorView() string {
	errorBox := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("196")).
		Padding(2, 4).
		Width(m.width - 10).
		Align(lipgloss.Center)

	content := lipgloss.JoinVertical(
		lipgloss.Center,
		errorStyle.Render("Error"),
		"",
		m.err.Error(),
		"",
		helpStyle.Render("Press 'q' to go back to menu"),
	)

	return lipgloss.Place(m.width, m.height,
		lipgloss.Center, lipgloss.Center,
		errorBox.Render(content),
	)
}

// =============================================================================
// Test Case Screen
// =============================================================================

type cardEditor struct {
	inputTA    textarea.Model
	expectedTA textarea.Model
}

type submission struct {
	Language string
	Status   string
	Runtime  string
	Memory   string
	At       string
}

type testCaseScreenModel struct {
	activePane   tcPane
	testCases    []models.TestCase
	editors      []cardEditor
	selectedCard int
	editingCard  bool
	activeField  cardField
	submissions  []submission
	selectedSub  int
	listViewport viewport.Model
	width        int
	height       int
	ready        bool
}

func initialTestCaseModel(testCases []models.TestCase) testCaseScreenModel {
	return testCaseScreenModel{
		activePane: tcCasesPane,
		testCases:  testCases,
		editors:    make([]cardEditor, len(testCases)),
	}
}

func (m testCaseScreenModel) Init() tea.Cmd { return nil }

func makeCardEditor(tc models.TestCase, paneWidth int) cardEditor {
	fieldW := paneWidth - 8

	inputTA := textarea.New()
	inputTA.SetWidth(fieldW)
	inputTA.SetHeight(2)
	inputTA.SetValue(tc.Input)

	expectedTA := textarea.New()
	expectedTA.SetWidth(fieldW)
	expectedTA.SetHeight(2)
	expectedTA.SetValue(tc.ExpectedOutput)

	return cardEditor{inputTA: inputTA, expectedTA: expectedTA}
}

func renderTestCaseCard(idx int, tc models.TestCase, selected bool, paneWidth int) string {
	style := borderStyle
	if selected {
		style = activePaneStyle
	}

	header := titleStyle.Render(fmt.Sprintf("Test Case %d", idx+1))
	inputSec := lipgloss.JoinVertical(lipgloss.Left,
		fieldLabelStyle.Render("Input:"),
		"  "+tc.Input,
	)
	expectedSec := lipgloss.JoinVertical(lipgloss.Left,
		fieldLabelStyle.Render("Expected:"),
		"  "+tc.ExpectedOutput,
	)

	body := lipgloss.JoinVertical(lipgloss.Left,
		header, "", inputSec, "", expectedSec,
	)

	return style.Width(paneWidth - 4).Padding(0, 1).Render(body)
}

// refreshListViewport re-renders all cards into the viewport and scrolls to selectedCard.
func refreshListViewport(m testCaseScreenModel) testCaseScreenModel {
	if !m.ready || m.width == 0 {
		return m
	}
	pw := (m.width / 2) - 4

	var parts []string
	for i, tc := range m.testCases {
		parts = append(parts, renderTestCaseCard(i, tc, i == m.selectedCard, pw))
	}
	m.listViewport.SetContent(strings.Join(parts, "\n"))

	// scroll so the selected card is visible
	lineOffset := 0
	for i := 0; i < m.selectedCard; i++ {
		card := renderTestCaseCard(i, m.testCases[i], false, pw)
		lineOffset += lipgloss.Height(card) + 1
	}
	m.listViewport.YOffset = lineOffset

	return m
}

func (m testCaseScreenModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		pw := (m.width / 2) - 4
		ph := m.height - 4

		if !m.ready && len(m.testCases) > 0 {
			m.listViewport = viewport.New(pw, ph)
			for i, tc := range m.testCases {
				m.editors[i] = makeCardEditor(tc, pw)
			}
			m.ready = true
		}

		if m.ready {
			m.listViewport.Width = pw
			m.listViewport.Height = ph
			m = refreshListViewport(m)
		}
		return m, nil

	case tea.KeyMsg:
		if m.editingCard {
			return m.handleEditKey(msg)
		}

		switch msg.String() {
		case "ctrl+c":
			return m, tea.Quit
		case "q":
			return m, switchScreen(problemScreen)
		case "ctrl+w":
			if m.activePane == tcCasesPane {
				m.activePane = tcSubsPane
			} else {
				m.activePane = tcCasesPane
			}
		case "h", "left":
			m.activePane = tcCasesPane
		case "l", "right":
			m.activePane = tcSubsPane
		case "j", "down":
			if m.activePane == tcCasesPane {
				if m.selectedCard < len(m.testCases)-1 {
					m.selectedCard++
					m = refreshListViewport(m)
				}
			} else {
				if m.selectedSub < len(m.submissions)-1 {
					m.selectedSub++
				}
			}
		case "k", "up":
			if m.activePane == tcCasesPane {
				if m.selectedCard > 0 {
					m.selectedCard--
					m = refreshListViewport(m)
				}
			} else {
				if m.selectedSub > 0 {
					m.selectedSub--
				}
			}
		case "i":
			if m.activePane == tcCasesPane && len(m.testCases) > 0 && m.ready {
				m.editingCard = true
				m.activeField = tcInputField
				editor := m.editors[m.selectedCard]
				editor.inputTA.SetValue(m.testCases[m.selectedCard].Input)
				editor.expectedTA.SetValue(m.testCases[m.selectedCard].ExpectedOutput)
				editor.inputTA.Focus()
				editor.expectedTA.Blur()
				m.editors[m.selectedCard] = editor
			}
		}
	}

	return m, nil
}

func (m testCaseScreenModel) handleEditKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	editor := m.editors[m.selectedCard]

	switch msg.String() {
	case "esc":
		editor.inputTA.Blur()
		editor.expectedTA.Blur()
		m.testCases[m.selectedCard].Input = editor.inputTA.Value()
		m.testCases[m.selectedCard].ExpectedOutput = editor.expectedTA.Value()
		m.editors[m.selectedCard] = editor
		m.editingCard = false
		m = refreshListViewport(m)
		return m, nil

	case "tab":
		if m.activeField == tcInputField {
			m.activeField = tcExpectedField
			editor.inputTA.Blur()
			editor.expectedTA.Focus()
		} else {
			m.activeField = tcInputField
			editor.expectedTA.Blur()
			editor.inputTA.Focus()
		}
		m.editors[m.selectedCard] = editor
		return m, nil

	default:
		var cmd tea.Cmd
		if m.activeField == tcInputField {
			editor.inputTA, cmd = editor.inputTA.Update(msg)
		} else {
			editor.expectedTA, cmd = editor.expectedTA.Update(msg)
		}
		m.editors[m.selectedCard] = editor
		return m, cmd
	}
}

func (m testCaseScreenModel) View() string {
	if m.width == 0 {
		return "Loading..."
	}

	pw := (m.width / 2) - 4
	ph := m.height - 4

	// --- Left pane: test case cards ---
	var leftContent string
	switch {
	case m.editingCard && len(m.editors) > 0:
		editor := m.editors[m.selectedCard]

		inputLabel := fieldLabelStyle.Render("Input:")
		expectedLabel := fieldLabelStyle.Render("Expected:")
		if m.activeField == tcInputField {
			inputLabel = activeFieldLabelStyle.Render("Input: [editing]")
		} else {
			expectedLabel = activeFieldLabelStyle.Render("Expected: [editing]")
		}

		leftContent = lipgloss.JoinVertical(lipgloss.Left,
			titleStyle.Render(fmt.Sprintf("Test Case %d", m.selectedCard+1)),
			"",
			inputLabel,
			editor.inputTA.View(),
			"",
			expectedLabel,
			editor.expectedTA.View(),
			"",
			helpStyle.Render("tab: switch field  |  esc: save & close"),
		)

	case len(m.testCases) == 0:
		leftContent = lipgloss.Place(pw-4, ph,
			lipgloss.Center, lipgloss.Center,
			helpStyle.Render("No test cases available"),
		)

	default:
		leftContent = m.listViewport.View()
	}

	leftStyle := borderStyle
	if m.activePane == tcCasesPane {
		leftStyle = activePaneStyle
	}
	leftPane := leftStyle.Width(pw).PaddingLeft(1).Render(leftContent)

	// --- Right pane: submissions ---
	var rightContent string
	if len(m.submissions) == 0 {
		rightContent = lipgloss.Place(pw-4, ph,
			lipgloss.Center, lipgloss.Center,
			helpStyle.Render(noSubmissionsArt),
		)
	} else {
		var sb strings.Builder
		for i, sub := range m.submissions {
			line := fmt.Sprintf("[%s] %s  %s  %s", sub.Language, sub.Status, sub.Runtime, sub.At)
			if i == m.selectedSub {
				sb.WriteString(selectedItemStyle.Render("> " + line))
			} else {
				sb.WriteString(menuItemStyle.Render("  " + line))
			}
			sb.WriteString("\n")
		}
		rightContent = sb.String()
	}

	rightStyle := borderStyle
	if m.activePane == tcSubsPane {
		rightStyle = activePaneStyle
	}
	rightPane := rightStyle.Width(pw).PaddingLeft(1).Render(rightContent)

	// --- Assemble ---
	title := titleStyle.Render("CLICode - Test Cases")
	help := helpStyle.Render("h/l: panes  |  j/k: navigate  |  i: edit card  |  tab: switch field  |  q: back")

	content := lipgloss.JoinHorizontal(lipgloss.Top, leftPane, rightPane)
	return lipgloss.JoinVertical(lipgloss.Left, title, content, help)
}

// =============================================================================
// Entry Point
// =============================================================================

func main() {
	logFile, err := os.OpenFile("clicode.log", os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
	if err == nil {
		defer func(logFile *os.File) {
			_ = logFile.Close()
		}(logFile)
		log.SetOutput(logFile)
	}

	p := tea.NewProgram(initialAppModel(), tea.WithAltScreen())

	if _, err := p.Run(); err != nil {
		fmt.Printf("Error running CLICode: %v\n", err)
		os.Exit(1)
	}
}
