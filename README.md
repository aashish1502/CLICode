# CLICode

A Vim-based TUI (Terminal User Interface) for solving competitive programming problems from LeetCode and Codeforces directly in your terminal.

## Overview

CLICode brings the competitive programming experience to your terminal with a clean, distraction-free interface inspired by Vim keybindings. No more context-switching between browser and editor - code, test, and submit all in one place.

## Features (Planned)

### Phase 1 - Core UI (In Progress)
- [x] Split-pane interface (problem description | code editor)
- [x] Vim-style navigation between panes
- [x] Problem loading from JSON (will transition to API)
- [x] Multi-language support (Python, C++, Go, etc.)
- [ ] Scrolling in problem pane
- [ ] Command mode (`:tc`, `:submit`, `:menu`)
- [ ] Multiple screen views (Problem, Test Cases, Submissions, Menu)

### Phase 2 - API Integration
- [ ] LeetCode API integration
- [ ] Codeforces API integration
- [ ] Problem fetching and caching
- [ ] Authentication and session management

### Phase 3 - Advanced Features
- [ ] Code execution and testing
- [ ] Submission to platforms
- [ ] Solution tracking and history
- [ ] Custom test case support

## Installation

### Prerequisites
- Go 1.21 or higher
- Terminal with true color support

### Build from Source
```bash
git clone https://github.com/yourusername/clicode.git
cd clicode
go mod tidy
go build -o clicode
./clicode
```

### Quick Run (Development)
```bash
go run .
```

## Usage

### Current Keybindings
- `h` / `l` - Switch between problem and code panes
- `Ctrl+w` - Toggle between panes
- `q` - Quit application

### Planned Keybindings
- `:tc` - View test cases
- `:submit` - Submit solution
- `:menu` - Return to problem menu
- `gt` / `gT` - Next/previous problem
- `:lang <language>` - Switch programming language

## Project Structure
```
clicode/
├── main.go                 # Application entry point
├── internal/
│   ├── models/
│   │   └── problem.go      # Problem data structures
│   └── loader/
│       └── loader.go       # Problem loading logic
└── data/
    └── problems/           # Problem JSON files (temporary)
        └── 110.json
```

## Data Format

Problems are currently loaded from JSON files. Example structure:
```json
{
  "id": 110,
  "title": "Balanced Binary Tree",
  "platform": "leetcode",
  "difficulty": "Easy",
  "tags": ["Tree", "Binary Tree", "DFS"],
  "description": "Problem description...",
  "example": [...],
  "constraints": "...",
  "testCases": [...],
  "codeStubs": {
    "python": "class Solution:\n    ...",
    "cpp": "class Solution {...}",
    "go": "func solve() {...}"
  }
}
```

## Development Status

🚧 **Currently in Phase 1 - Core UI Development**

This project is in active development. The basic UI is functional, but many features are still being implemented.

### What Works
- Problem display with descriptions, examples, constraints
- Code stub loading for multiple languages
- Pane navigation with Vim-style keybindings
- Error handling and validation

### What's Next
- Scrollable problem descriptions
- Command mode implementation
- Screen switching (test cases, submissions)
- Actual code editing capabilities

## Contributing

This is a personal learning project to explore Go and TUI development. However, suggestions and feedback are welcome!

## Tech Stack

- **Language**: Go
- **TUI Framework**: [Bubble Tea](https://github.com/charmbracelet/bubbletea)
- **Styling**: [Lipgloss](https://github.com/charmbracelet/lipgloss)

## License

MIT (or whatever you choose)

## Acknowledgments

- Inspired by terminal-based workflow tools like `lazygit` and `k9s`
- Built to scratch the itch of doing LeetCode without leaving the terminal
