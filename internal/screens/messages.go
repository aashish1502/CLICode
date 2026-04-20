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
