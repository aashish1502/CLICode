# CLICode

A Vim-keybinding TUI (Terminal User Interface) for solving competitive programming problems from LeetCode and Codeforces — directly in your terminal.

## Overview

CLICode brings the competitive programming experience into your terminal with a clean, distraction-free interface. No more context-switching between browser and editor — browse problems, read descriptions, write code, inspect test cases, and track submissions all in one place.

## Features

### What's Working Now

- **Splash screen** — ASCII art title card with auto-advance (swap `data/art/current.txt` to customise without recompiling)
- **Main menu** — Continue / Problem List / Settings / Quit with Vim-style cursor navigation
- **Problem list** — scrollable list with difficulty colouring, solved indicators, and last-worked-on marker
- **Problem screen** — split-pane layout: problem description (left) + code editor (right)
  - Vim-style pane switching (`h`/`l`, `ctrl+w`)
  - Scrollable problem description
  - Multi-language code stubs (Python, C++, Go)
  - Insert mode editing (`i` to enter, `Esc` to exit, `Tab` for indentation)
- **Test case screen** — card-based layout with per-TC cursor and edit mode
  - `j`/`k` to navigate between test cases
  - `i`/`Enter` to edit a test case's input or expected output
  - `Tab` to switch between Input and Expected fields, `Esc` to save
  - Submission history pane (right side) with colour-coded status icons
- **Session persistence** — last opened problem is remembered across restarts (`~/.clicode/session.json`)
- **Config file** — default language and auth token stored in `~/.clicode/config.json`

### Planned

#### Phase 2 — API Integration
- [ ] LeetCode API integration
- [ ] Codeforces API integration
- [ ] Problem fetching and caching
- [ ] Real authentication flow

#### Phase 3 — Advanced Features
- [ ] Code execution and local test running
- [ ] Submission to platforms
- [ ] Solution tracking and history

## Installation

### Prerequisites

- Go 1.21 or higher
- Terminal with true color support

### Build from Source

```bash
git clone https://github.com/aashish1502/clicode.git
cd clicode
go mod tidy
go build -o clicode
./clicode
```

### Quick Run (Development)

```bash
go run .
```

## Keybindings

### Global
| Key | Action |
|-----|--------|
| `ctrl+c` | Quit |

### Menu
| Key | Action |
|-----|--------|
| `j` / `k` | Navigate options |
| `Enter` | Select |

### Problem Screen
| Key | Action |
|-----|--------|
| `h` / `l` | Switch panes |
| `ctrl+w` | Toggle active pane |
| `j` / `k` | Scroll description / move cursor in editor |
| `i` | Enter insert mode (editor pane) |
| `Esc` | Exit insert mode |
| `Tab` | Insert 4 spaces |
| `t` | Open test cases screen |
| `m` | Return to menu |
| `q` | Quit |

### Test Case Screen
| Key | Action |
|-----|--------|
| `j` / `k` | Navigate test cases (cards pane) or scroll (submissions pane) |
| `i` / `Enter` | Edit selected test case |
| `Tab` | Switch between Input and Expected fields |
| `Esc` | Save edits and return to card view |
| `h` / `l` | Switch between cards and submissions panes |
| `ctrl+w` | Toggle active pane |
| `q` / `Esc` | Back to problem screen |

## Configuration

CLICode stores user data in `~/.clicode/`:

| File | Contents |
|------|----------|
| `session.json` | Last opened problem ID |
| `config.json` | Default language, auth token |

Example `config.json`:
```json
{
  "defaultLanguage": "python",
  "authToken": ""
}
```

To change the splash screen art, replace `data/art/current.txt` with any ASCII art file — no recompile needed.

## Project Structure

```
clicode/
├── main.go
├── data/
│   ├── art/
│   │   └── current.txt          # Splash screen ASCII art
│   └── problems/
│       ├── problems_list.json   # Problem index
│       └── *.json               # Individual problem files
└── internal/
    ├── config/
    │   └── config.go            # Config load/save (~/.clicode/config.json)
    ├── design/
    │   └── theme.go             # Centralised Lipgloss styles
    ├── loader/
    │   └── loader.go            # Problem JSON loading
    ├── models/
    │   ├── problems.go          # Problem / TestCase / Submission structs
    │   └── ProblemListItem.go   # Problem list entry struct
    ├── screens/
    │   ├── router.go            # Top-level screen router
    │   ├── messages.go          # Navigation message types
    │   ├── title.go             # Splash screen
    │   ├── menu.go              # Main menu
    │   ├── problem_list.go      # Problem browser
    │   ├── problem.go           # Problem + code editor
    │   ├── testcase.go          # Test cases + submissions
    │   └── settings.go          # Settings (placeholder)
    └── session/
        └── session.go           # Session load/save (~/.clicode/session.json)
```

## Tech Stack

- **Language**: Go
- **TUI Framework**: [Bubble Tea](https://github.com/charmbracelet/bubbletea)
- **Styling**: [Lipgloss](https://github.com/charmbracelet/lipgloss)
- **Components**: [Bubbles](https://github.com/charmbracelet/bubbles) (viewport, textarea)

## Acknowledgments

- Inspired by terminal-first tools like `lazygit` and `k9s`
- Built to scratch the itch of doing LeetCode without leaving the terminal
