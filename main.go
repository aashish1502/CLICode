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

	p := tea.NewProgram(screens.NewRouter(), tea.WithAltScreen())
	if _, err := p.Run(); err != nil {
		fmt.Printf("Error running CLICode: %v\n", err)
		os.Exit(1)
	}
}
