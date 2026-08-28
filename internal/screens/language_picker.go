package screens

import (
	"strings"

	"github.com/aashish1502/clicode/internal/design"
	"github.com/aashish1502/clicode/internal/languages"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

var (
	pickerBox = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("#eb650c")).
			Padding(1, 3)

	pickerTitle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("230"))

	pickerCursor = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#eb650c")).
			Bold(true)
)

// languagePicker is a modal list of every language the buffer can be opened in.
//
// It is a sub-component of ProblemScreen rather than a routed screen: opening it
// must not disturb the editor buffer or the description scroll position, and
// keeping it inside the same model gets that for free.
type languagePicker struct {
	open    bool
	langs   []languages.Language
	stubbed map[string]bool // which of langs ship starter code
	cursor  int
}

// openPicker builds the modal over the given languages, starting the cursor on
// whichever one is currently loaded.
func openPicker(langs []languages.Language, stubbed []string, current string) languagePicker {
	set := make(map[string]bool, len(stubbed))
	for _, id := range stubbed {
		set[id] = true
	}

	p := languagePicker{open: true, langs: langs, stubbed: set}
	for i, l := range langs {
		if l.ID == current {
			p.cursor = i
			break
		}
	}
	return p
}

// Update handles one key while the modal is open. The returned id is non-empty
// only when the user actually chose a language; a cancel closes with "".
func (p languagePicker) Update(msg tea.KeyMsg) (languagePicker, string) {
	switch msg.String() {
	case "j", "down", "ctrl+n":
		if p.cursor < len(p.langs)-1 {
			p.cursor++
		}

	case "k", "up", "ctrl+p":
		if p.cursor > 0 {
			p.cursor--
		}

	case "g", "home":
		p.cursor = 0

	case "G", "end":
		p.cursor = len(p.langs) - 1

	case "enter", " ":
		p.open = false
		if p.cursor < len(p.langs) {
			return p, p.langs[p.cursor].ID
		}

	case "esc", "q", "ctrl+l", "ctrl+c":
		p.open = false
	}

	return p, ""
}

// View renders the modal centred in the terminal. It replaces the split-pane
// view while open rather than floating over it — compositing over existing ANSI
// output is a layout concern, and belongs with the rest of the layout work.
func (p languagePicker) View(width, height int, current string) string {
	rows := make([]string, 0, len(p.langs))

	for i, l := range p.langs {
		cursor := "  "
		if i == p.cursor {
			cursor = pickerCursor.Render("▸ ")
		}

		name := l.DisplayName
		if l.ID == current {
			name += " (current)"
		}

		// Green means this problem ships starter code for the language.
		// Everything else is still selectable — it just opens on a comment.
		style := design.LanguageBare
		if p.stubbed[l.ID] {
			style = design.LanguageStubbed
		}

		rows = append(rows, cursor+style.Render(name))
	}

	body := lipgloss.JoinVertical(lipgloss.Left,
		pickerTitle.Render("Select language"),
		"",
		strings.Join(rows, "\n"),
		"",
		design.Help.Render("green = starter code available"),
		design.Help.Render("j/k: move   enter: select   esc: cancel"),
	)

	return lipgloss.Place(width, height, lipgloss.Center, lipgloss.Center, pickerBox.Render(body))
}
