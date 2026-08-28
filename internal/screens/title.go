package screens

import (
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/aashish1502/clicode/data"
	"github.com/aashish1502/clicode/internal/design"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// userArtPath is where a user drops their own title art to override the one
// built into the binary.
const userArtPath = "art/current.txt"

// titleTickMsg fires when the auto-advance timer expires.
type titleTickMsg struct{}

var (
	artStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#eb650c"))

	hintStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("241")).
			Italic(true)
)

// TitleScreen displays ASCII art, then transitions to the problem list on any
// keypress or after a short timer.
//
// The art stays swappable without recompiling: drop a file at
// ~/.clicode/art/current.txt and it wins over the copy built into the binary.
type TitleScreen struct {
	art    string
	width  int
	height int
}

func NewTitleScreen(width, height int) TitleScreen {
	return TitleScreen{
		art:    loadArt(),
		width:  width,
		height: height,
	}
}

// loadArt resolves the title art, most specific source first: the user's own
// file, then the copy embedded in the binary, then a compiled-in banner. Each
// step falls through on any error, so the app always starts.
func loadArt() string {
	if art, ok := clean(userArt()); ok {
		return art
	}
	if raw, ok := data.Art(); ok {
		if art, ok := clean(raw); ok {
			return art
		}
	}
	return fallbackArt()
}

// userArt reads ~/.clicode/art/current.txt, if it exists.
func userArt() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return ""
	}
	raw, err := os.ReadFile(filepath.Join(home, ".clicode", userArtPath))
	if err != nil {
		return ""
	}
	return string(raw)
}

// clean trims a candidate banner and reports whether anything is left of it.
func clean(raw string) (string, bool) {
	art := strings.TrimRight(raw, "\n")
	if strings.TrimSpace(art) == "" {
		return "", false
	}
	return art, true
}

func fallbackArt() string {
	return `
  ________    ____   ______          __
 / ____/ /   /  _/  / ____/___  ____/ /__
/ /   / /    / /   / /   / __ \/ __  / _ \
\___/_____/___/   \____/\____/\__,_/\___/`
}

func (s TitleScreen) Init() tea.Cmd {
	return tea.Tick(4*time.Second, func(time.Time) tea.Msg {
		return titleTickMsg{}
	})
}

func (s TitleScreen) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg.(type) {
	case tea.WindowSizeMsg:
		m := msg.(tea.WindowSizeMsg)
		s.width = m.Width
		s.height = m.Height
		return s, nil

	// Timer expired or any key pressed — move to the main menu.
	case titleTickMsg, tea.KeyMsg:
		return s, func() tea.Msg { return NavigateToMenuMsg{} }
	}
	return s, nil
}

func (s TitleScreen) View() string {
	if s.width == 0 {
		return ""
	}

	art := artStyle.Render(s.art)
	version := design.Help.Render("v0.1.0-alpha")
	hint := hintStyle.Render("press any key to continue")

	block := lipgloss.JoinVertical(lipgloss.Center,
		art,
		"",
		version,
		"",
		hint,
	)

	return lipgloss.Place(s.width, s.height, lipgloss.Center, lipgloss.Center, block)
}
