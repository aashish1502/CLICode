// Package editor provides a Vim-modal text editor component for Bubble Tea.
package editor

import (
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// Mode represents the current Vim editing mode.
type Mode int

const (
	Normal Mode = iota
	Insert
)

var (
	cursorNormal = lipgloss.NewStyle().Reverse(true)
	cursorInsert = lipgloss.NewStyle().
			Foreground(lipgloss.Color("0")).
			Background(lipgloss.Color("39"))
	modeNormal = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("214"))
	modeInsert = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("39"))
	tildeStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("240"))
)

// VimEditor is a self-contained Vim-modal editor that implements the Bubble Tea
// component pattern (value receiver, returns modified copy).
type VimEditor struct {
	lines   []string
	row     int
	col     int
	scroll  int
	height  int
	width   int
	Mode    Mode
	yank    string // line yank buffer
	pending string // buffered first key of a two-key sequence (dd, yy, gg)
}

// New returns an empty VimEditor sized to width × height.
func New(width, height int) VimEditor {
	return VimEditor{
		lines:  []string{""},
		width:  width,
		height: height,
		Mode:   Normal,
	}
}

// SetSize resizes the editor viewport.
func (e VimEditor) SetSize(width, height int) VimEditor {
	e.width = width
	e.height = height
	return e.clampScroll()
}

// SetValue replaces the buffer contents and resets the cursor.
func (e VimEditor) SetValue(content string) VimEditor {
	e.lines = strings.Split(content, "\n")
	if len(e.lines) == 0 {
		e.lines = []string{""}
	}
	e.row = 0
	e.col = 0
	e.scroll = 0
	e.pending = ""
	return e
}

// Value returns the current buffer as a single string.
func (e VimEditor) Value() string {
	return strings.Join(e.lines, "\n")
}

// ── Internal helpers ──────────────────────────────────────────────────────────

func (e VimEditor) currentLine() string {
	if e.row >= 0 && e.row < len(e.lines) {
		return e.lines[e.row]
	}
	return ""
}

// maxCol returns the largest valid column for the current mode.
func (e VimEditor) maxCol() int {
	runes := []rune(e.currentLine())
	if e.Mode == Normal {
		if len(runes) == 0 {
			return 0
		}
		return len(runes) - 1
	}
	return len(runes)
}

func (e VimEditor) clampCol() VimEditor {
	max := e.maxCol()
	if e.col > max {
		e.col = max
	}
	if e.col < 0 {
		e.col = 0
	}
	return e
}

func (e VimEditor) clampRow() VimEditor {
	if e.row >= len(e.lines) {
		e.row = len(e.lines) - 1
	}
	if e.row < 0 {
		e.row = 0
	}
	return e
}

func (e VimEditor) clampScroll() VimEditor {
	if e.height <= 0 {
		return e
	}
	if e.row < e.scroll {
		e.scroll = e.row
	}
	if e.row >= e.scroll+e.height {
		e.scroll = e.row - e.height + 1
	}
	if e.scroll < 0 {
		e.scroll = 0
	}
	return e
}

func leadingSpaces(line string) string {
	var b strings.Builder
	for _, ch := range line {
		if ch == ' ' || ch == '\t' {
			b.WriteRune(ch)
		} else {
			break
		}
	}
	return b.String()
}

// ── Insert-mode operations ────────────────────────────────────────────────────

func (e VimEditor) insertRune(r rune) VimEditor {
	line := []rune(e.currentLine())
	col := e.col
	if col > len(line) {
		col = len(line)
	}
	newLine := append(append(append([]rune{}, line[:col]...), r), line[col:]...)
	e.lines[e.row] = string(newLine)
	e.col++
	return e
}

func (e VimEditor) insertNewline() VimEditor {
	line := e.currentLine()
	runes := []rune(line)
	col := e.col
	if col > len(runes) {
		col = len(runes)
	}
	before := string(runes[:col])
	after := string(runes[col:])
	indent := leadingSpaces(before)

	newLines := make([]string, 0, len(e.lines)+1)
	newLines = append(newLines, e.lines[:e.row+1]...)
	newLines[e.row] = before
	newLines = append(newLines, indent+after)
	newLines = append(newLines, e.lines[e.row+1:]...)
	e.lines = newLines
	e.row++
	e.col = len([]rune(indent))
	return e.clampScroll()
}

func (e VimEditor) backspace() VimEditor {
	if e.col > 0 {
		line := []rune(e.currentLine())
		e.lines[e.row] = string(append(line[:e.col-1], line[e.col:]...))
		e.col--
	} else if e.row > 0 {
		prev := e.lines[e.row-1]
		cur := e.currentLine()
		e.col = len([]rune(prev))
		e.lines[e.row-1] = prev + cur
		e.lines = append(e.lines[:e.row], e.lines[e.row+1:]...)
		e.row--
	}
	return e.clampScroll()
}

// ── Normal-mode operations ────────────────────────────────────────────────────

func (e VimEditor) deleteLine() VimEditor {
	e.yank = e.currentLine() + "\n"
	if len(e.lines) == 1 {
		e.lines[0] = ""
		e.col = 0
		return e
	}
	e.lines = append(e.lines[:e.row], e.lines[e.row+1:]...)
	e = e.clampRow().clampCol().clampScroll()
	return e
}

func (e VimEditor) yankLine() VimEditor {
	e.yank = e.currentLine() + "\n"
	return e
}

func (e VimEditor) paste(above bool) VimEditor {
	if e.yank == "" {
		return e
	}
	content := strings.TrimSuffix(e.yank, "\n")
	at := e.row + 1
	if above {
		at = e.row
	}
	newLines := make([]string, 0, len(e.lines)+1)
	newLines = append(newLines, e.lines[:at]...)
	newLines = append(newLines, content)
	newLines = append(newLines, e.lines[at:]...)
	e.lines = newLines
	e.row = at
	e.col = 0
	return e.clampScroll()
}

func (e VimEditor) wordForward() VimEditor {
	line := []rune(e.currentLine())
	col := e.col
	for col < len(line) && line[col] != ' ' {
		col++
	}
	for col < len(line) && line[col] == ' ' {
		col++
	}
	if col >= len(line) && e.row < len(e.lines)-1 {
		e.row++
		e.col = 0
		return e.clampScroll()
	}
	e.col = col
	return e.clampCol()
}

func (e VimEditor) wordBack() VimEditor {
	line := []rune(e.currentLine())
	col := e.col
	if col == 0 {
		if e.row > 0 {
			e.row--
			e.col = len([]rune(e.lines[e.row]))
			return e.clampCol().clampScroll()
		}
		return e
	}
	col--
	for col > 0 && line[col] == ' ' {
		col--
	}
	for col > 0 && line[col-1] != ' ' {
		col--
	}
	e.col = col
	return e.clampCol()
}

func (e VimEditor) deleteChar() VimEditor {
	line := []rune(e.currentLine())
	if e.col < len(line) {
		e.lines[e.row] = string(append(line[:e.col], line[e.col+1:]...))
		e = e.clampCol()
	}
	return e
}

// ── Update ────────────────────────────────────────────────────────────────────

func (e VimEditor) Update(msg tea.Msg) (VimEditor, tea.Cmd) {
	key, ok := msg.(tea.KeyMsg)
	if !ok {
		return e, nil
	}
	if e.Mode == Insert {
		return e.updateInsert(key)
	}
	return e.updateNormal(key)
}

func (e VimEditor) updateInsert(key tea.KeyMsg) (VimEditor, tea.Cmd) {
	switch key.Type {
	// ctrl+c leaves insert mode, as it does in vim. It must be handled here so
	// that typing is never interrupted by the app-level back binding.
	case tea.KeyEsc, tea.KeyCtrlC:
		e.Mode = Normal
		e = e.clampCol() // normal mode max col is one less
	case tea.KeyEnter:
		e = e.insertNewline()
	case tea.KeyBackspace, tea.KeyDelete:
		e = e.backspace()
	case tea.KeyTab:
		for i := 0; i < 4; i++ {
			e = e.insertRune(' ')
		}
	case tea.KeySpace:
		e = e.insertRune(' ')
	case tea.KeyLeft:
		if e.col > 0 {
			e.col--
		}
	case tea.KeyRight:
		line := []rune(e.currentLine())
		if e.col < len(line) {
			e.col++
		}
	case tea.KeyUp:
		if e.row > 0 {
			e.row--
			e = e.clampCol().clampScroll()
		}
	case tea.KeyDown:
		if e.row < len(e.lines)-1 {
			e.row++
			e = e.clampCol().clampScroll()
		}
	case tea.KeyHome:
		e.col = 0
	case tea.KeyEnd:
		e.col = len([]rune(e.currentLine()))
	case tea.KeyRunes:
		for _, r := range key.Runes {
			e = e.insertRune(r)
		}
	}
	return e, nil
}

func (e VimEditor) updateNormal(key tea.KeyMsg) (VimEditor, tea.Cmd) {
	k := key.String()

	// ── Resolve two-key sequences ─────────────────────────────────────────────
	if e.pending != "" {
		seq := e.pending + k
		e.pending = ""
		switch seq {
		case "dd":
			return e.deleteLine(), nil
		case "yy":
			return e.yankLine(), nil
		case "gg":
			e.row = 0
			e.col = 0
			e.scroll = 0
			return e, nil
		}
		// Unrecognised sequence — fall through and process k on its own.
	}

	switch k {
	// ── Enter insert mode ─────────────────────────────────────────────────────
	case "i":
		e.Mode = Insert
	case "a":
		e.Mode = Insert
		line := []rune(e.currentLine())
		if e.col < len(line) {
			e.col++
		}
	case "A":
		e.Mode = Insert
		e.col = len([]rune(e.currentLine()))
	case "o":
		e.Mode = Insert
		indent := leadingSpaces(e.currentLine())
		at := e.row + 1
		newLines := make([]string, 0, len(e.lines)+1)
		newLines = append(newLines, e.lines[:at]...)
		newLines = append(newLines, indent)
		newLines = append(newLines, e.lines[at:]...)
		e.lines = newLines
		e.row = at
		e.col = len([]rune(indent))
		e = e.clampScroll()
	case "O":
		e.Mode = Insert
		indent := leadingSpaces(e.currentLine())
		newLines := make([]string, 0, len(e.lines)+1)
		newLines = append(newLines, e.lines[:e.row]...)
		newLines = append(newLines, indent)
		newLines = append(newLines, e.lines[e.row:]...)
		e.lines = newLines
		e.col = len([]rune(indent))
		e = e.clampScroll()

	// ── Motion ────────────────────────────────────────────────────────────────
	case "h", "left":
		if e.col > 0 {
			e.col--
		}
	case "l", "right":
		line := []rune(e.currentLine())
		if e.col < len(line)-1 {
			e.col++
		}
	case "j", "down":
		if e.row < len(e.lines)-1 {
			e.row++
			e = e.clampCol().clampScroll()
		}
	case "k", "up":
		if e.row > 0 {
			e.row--
			e = e.clampCol().clampScroll()
		}
	case "0":
		e.col = 0
	case "$":
		line := []rune(e.currentLine())
		if len(line) > 0 {
			e.col = len(line) - 1
		}
	case "G":
		e.row = len(e.lines) - 1
		e = e.clampCol().clampScroll()
	case "w":
		e = e.wordForward()
	case "b":
		e = e.wordBack()
	case "ctrl+d":
		half := e.height / 2
		e.row += half
		if e.row >= len(e.lines) {
			e.row = len(e.lines) - 1
		}
		e = e.clampCol().clampScroll()
	case "ctrl+u":
		half := e.height / 2
		e.row -= half
		if e.row < 0 {
			e.row = 0
		}
		e = e.clampCol().clampScroll()

	// ── Edit ──────────────────────────────────────────────────────────────────
	case "x":
		e = e.deleteChar()
	case "p":
		e = e.paste(false)
	case "P":
		e = e.paste(true)

	// ── Two-key sequence starters ─────────────────────────────────────────────
	case "d", "y", "g":
		e.pending = k
	}

	return e, nil
}

// ── View ──────────────────────────────────────────────────────────────────────

// View renders the editor content into a string that fits exactly within the
// configured width × height. Lines longer than width are truncated so the
// border around the editor never changes size.
func (e VimEditor) View() string {
	if e.height <= 0 || e.width <= 0 {
		return ""
	}

	end := e.scroll + e.height
	if end > len(e.lines) {
		end = len(e.lines)
	}
	visible := e.lines[e.scroll:end]

	var sb strings.Builder
	for i, line := range visible {
		if i > 0 {
			sb.WriteByte('\n')
		}
		absRow := e.scroll + i
		runes := []rune(line)

		// Truncate runes to editor width so long lines never push the border out.
		maxVisible := e.width
		if len(runes) > maxVisible {
			runes = runes[:maxVisible]
		}

		if absRow != e.row {
			sb.WriteString(string(runes))
			continue
		}

		// Render cursor on the active row.
		col := e.col
		if col > len(runes) {
			col = len(runes)
		}

		before := string(runes[:col])
		var curCh string
		var after string
		if col < len(runes) {
			curCh = string(runes[col])
			after = string(runes[col+1:])
		} else {
			curCh = " " // cursor past end of line
		}

		if e.Mode == Insert {
			sb.WriteString(before + cursorInsert.Render(curCh) + after)
		} else {
			sb.WriteString(before + cursorNormal.Render(curCh) + after)
		}
	}

	// Fill remaining lines with vim-style tildes.
	rendered := end - e.scroll
	for i := rendered; i < e.height; i++ {
		sb.WriteByte('\n')
		sb.WriteString(tildeStyle.Render("~"))
	}

	return sb.String()
}

// ModeString returns a status-bar string for the current mode.
func (e VimEditor) ModeString() string {
	if e.Mode == Insert {
		return modeInsert.Render("-- INSERT --")
	}
	return modeNormal.Render("-- NORMAL --")
}
