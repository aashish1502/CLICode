package screens

import (
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/aashish1502/clicode/internal/design"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

const artPath = "data/art/current.txt"

// titleTickMsg fires when the auto-advance timer expires.
type titleTickMsg struct{}

var (
	artStyle = lipgloss.NewStyle().
		Foreground(lipgloss.Color("#eb650c"))

	hintStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("241")).
			Italic(true)
)

// TitleScreen displays ASCII art loaded from artPath, then transitions to the
// problem list on any keypress or after a short timer.
// The art itself lives entirely outside the binary — swap data/art/current.txt
// to change what appears without recompiling.
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

// loadArt reads the art file. Falls back to a minimal banner on any error so
// the app always starts even if the file is missing or malformed.
func loadArt() string {
	data, err := os.ReadFile(filepath.Clean(artPath))
	if err != nil {
		return fallbackArt()
	}
	art := strings.TrimRight(string(data), "\n")
	if strings.TrimSpace(art) == "" {
		return fallbackArt()
	}
	return art
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

	// Timer expired or any key pressed — move to the problem list.
	case titleTickMsg, tea.KeyMsg:
		return s, func() tea.Msg { return NavigateToProblemListMsg{} }
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
