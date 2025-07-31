package main

import (
	"log"
	"purge/internal/tui"

	tea "github.com/charmbracelet/bubbletea"
)

func main() {

	p := tea.NewProgram(tui.LoginModel(), tea.WithMouseCellMotion())
	if _, err := p.Run(); err != nil {
		log.Fatal("Error running tui:", err)
	}
}
