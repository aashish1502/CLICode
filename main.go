package main

import (
	"fmt"
	"log"
	"os"

	"github.com/aashish1502/clicode/internal/screens"
	tea "github.com/charmbracelet/bubbletea"
)

func main() {
	logFile, err := os.OpenFile("clicode.log", os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
	if err == nil {
		defer logFile.Close()
		log.SetOutput(logFile)
	}

	router := screens.NewRouter()

	p := tea.NewProgram(router, tea.WithAltScreen())
	_, runErr := p.Run()

	// Closed explicitly rather than with defer: os.Exit below would skip a
	// deferred call, and the database should be released either way.
	if err := router.Close(); err != nil {
		log.Printf("closing the catalog: %v", err)
	}

	if runErr != nil {
		fmt.Printf("Error running CLICode: %v\n", runErr)
		os.Exit(1)
	}
}
