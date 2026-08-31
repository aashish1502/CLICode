package screens

import "github.com/aashish1502/clicode/internal/models"

// NavigateToProblemMsg tells the router to load and display a specific problem.
type NavigateToProblemMsg struct{ ProblemID int }

// NavigateToProblemListMsg tells the router to show the problem list.
type NavigateToProblemListMsg struct{}

// NavigateToTestCaseMsg tells the router to open the TC screen for the given problem.
type NavigateToTestCaseMsg struct{ Problem *models.Problem }

// NavigateBackMsg returns to the previous logical screen (TC → problem, problem → list).
type NavigateBackMsg struct{}

// NavigateToMenuMsg shows the main menu.
type NavigateToMenuMsg struct{}

// NavigateToSettingsMsg shows the settings screen.
type NavigateToSettingsMsg struct{}

// SaveSolutionMsg asks the router to persist an editor buffer. Screens do no
// I/O of their own -- ":w" emits this and the router does the writing, the same
// way navigation works.
type SaveSolutionMsg struct {
	ProblemID int
	Language  string
	Code      string
}

// SolutionSavedMsg reports the outcome of a SaveSolutionMsg back to the screen
// that asked, so it can show "written" or the reason it could not be.
type SolutionSavedMsg struct {
	Language string
	Err      error
}
