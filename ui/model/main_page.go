package model

import (
	"fmt"
	"netps/internal/parser"
	"strconv"
	"strings"

	"github.com/charmbracelet/bubbles/table"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

type MainPage struct {
	table table.Model
}

func (m MainPage) Init() tea.Cmd { return nil }

func (m MainPage) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "esc":
			if m.table.Focused() {
				m.table.Blur()
			} else {
				m.table.Focus()
			}
		case "q", "ctrl+c":
			return m, tea.Quit
		case "enter":
			return m, tea.Batch(
				tea.Printf("process %s selected!", m.table.SelectedRow()[0]),
			)
		}
	}
	m.table, cmd = m.table.Update(msg)
	return m, cmd
}

func (m MainPage) View() string {
	var baseStyle = lipgloss.NewStyle().
		BorderStyle(lipgloss.NormalBorder()).
		BorderForeground(lipgloss.Color("240"))

	statusBar := "[↑↓] select · [Enter] inspect/filter · [q] quit"
	return baseStyle.Render(m.table.View()) + "\n" + statusBar + "\n"
}

func mapProcessItem() []table.Row {
	processes, err := parser.ScanListeningPortsProcfs()

	if err != nil {
		panic(err)
	}

	var rows []table.Row
	for _, p := range processes {
		var sb strings.Builder

		_, err := fmt.Fprintf(&sb, "%s:%d/%s", p.SocketInfo.Addr, p.SocketInfo.Port, p.SocketInfo.Proto)
		if err != nil {
			panic(err)
		}
		r := table.Row{
			strconv.Itoa(p.PID),
			p.Name,
			sb.String(),
		}
		rows = append(rows, r)
	}
	return rows
}

func Initialize() tea.Model {
	columns := []table.Column{
		{Title: "PID", Width: 10},
		{Title: "Name", Width: 30},
		{Title: "Ports", Width: 30},
	}

	rows := mapProcessItem()

	t := table.New(
		table.WithColumns(columns),
		table.WithRows(rows),
		table.WithFocused(true),
		table.WithHeight(7),
	)

	s := table.DefaultStyles()
	s.Header = s.Header.
		BorderStyle(lipgloss.NormalBorder()).
		BorderForeground(lipgloss.Color("240")).
		BorderBottom(true).
		Bold(false)
	s.Selected = s.Selected.
		Foreground(lipgloss.Color("229")).
		Background(lipgloss.Color("57")).
		Bold(false)
	t.SetStyles(s)
	m := MainPage{t}
	return m
}
