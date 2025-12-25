package main

import (
	"fmt"
	uimodel "netps/ui/model"
	"os"

	tea "github.com/charmbracelet/bubbletea"
)

func main() {
	p := tea.NewProgram(uimodel.InitialModel())
	if _, err := p.Run(); err != nil {
		fmt.Printf("Alas, there's been an error: %v", err)
		os.Exit(1)
	}
}
