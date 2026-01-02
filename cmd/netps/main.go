package main

import (
	"fmt"
	"os"

	"netps/internal/ui"

	tea "charm.land/bubbletea/v2"
)

func main() {
	root := ui.New()

	p := tea.NewProgram(root)
	if _, err := p.Run(); err != nil {
		fmt.Printf("Alas, there's been an error: %v", err)
		os.Exit(1)
	}
}
