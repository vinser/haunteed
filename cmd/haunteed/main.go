package main

import (
	"os"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/vinser/haunteed/internal/app"
)

var version = "dev"

func main() {
	p := tea.NewProgram(app.New(version), tea.WithAltScreen())
	if _, err := p.Run(); err != nil {
		println("Error:", err)
		os.Exit(1)
	}
}
