package main

import (
	"fmt"
	"os"

	uimodel "netps/ui/model"

	tea "charm.land/bubbletea/v2"
)

func main() {
	p := tea.NewProgram(uimodel.ProcessListScreen{}.Initialize())
	if _, err := p.Run(); err != nil {
		fmt.Printf("Alas, there's been an error: %v", err)
		os.Exit(1)
	}
}
