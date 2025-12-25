package model

import (
	"fmt"
	"netps/internal/parser"
	"netps/pkg/model"

	tea "github.com/charmbracelet/bubbletea"
)

type ProcessItem struct {
	Choices  []model.Process
	Cursor   int
	Selected map[int]struct{}
}

func InitialModel() ProcessItem {
	res, err := parser.ScanListeningPortsProcfs()

	if err != nil {
		panic(err)
	}

	return ProcessItem{
		Choices:  res,
		Selected: make(map[int]struct{}),
	}
}

func (pi ProcessItem) Init() tea.Cmd {
	// Just return `nil`, which means "no I/O right now, please."
	return nil
}

func (pi ProcessItem) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {

	// Is it a key press?
	case tea.KeyMsg:

		// Cool, what was the actual key pressed?
		switch msg.String() {

		// These keys should exit the program.
		case "ctrl+c", "q":
			return pi, tea.Quit

		// The "up" and "k" keys move the cursor up
		case "up", "k":
			if pi.Cursor > 0 {
				pi.Cursor--
			}

		// The "down" and "j" keys move the cursor down
		case "down", "j":
			if pi.Cursor < len(pi.Choices)-1 {
				pi.Cursor++
			}

		// The "enter" key and the spacebar (a literal space) toggle
		// the selected state for the item that the cursor is pointing at.
		case "enter", " ":
			_, ok := pi.Selected[pi.Cursor]
			if ok {
				delete(pi.Selected, pi.Cursor)
			} else {
				pi.Selected[pi.Cursor] = struct{}{}
			}
		}
	}

	// Return the updated model to the Bubble Tea runtime for processing.
	// Note that we're not returning a command.
	return pi, nil
}

func (pi ProcessItem) View() string {
	// The header
	s := "Listening Socket and Their PIDs\n\n"

	// Iterate over our choices
	for i, choice := range pi.Choices {

		// Is the cursor pointing at this choice?
		cursor := " " // no cursor
		if pi.Cursor == i {
			cursor = ">" // cursor!
		}

		// Is this choice selected?
		checked := " " // not selected
		if _, ok := pi.Selected[i]; ok {
			checked = "x" // selected!
		}

		// Render the row
		s += fmt.Sprintf("%s [%s] %v\n", cursor, checked, choice)
	}

	// The footer
	s += "\nPress q to quit.\n"

	// Send the UI for rendering
	return s
}
