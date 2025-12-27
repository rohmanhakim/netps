package model

import (
	"fmt"
	"netps/internal/parser"
	"strconv"
	"strings"

	"charm.land/bubbles/v2/table"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
)

const HorizontalPadding = 1
const VerticalPadding = 2
const CellPadding = 2
const FirstColumnWidth = 10

type MainPage struct {
	table table.Model
}

func (m MainPage) Init() tea.Cmd { return nil }

func (m MainPage) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		updateTableSize(&m, msg.Width, msg.Height)
		m.table.Focus()
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
			return m, cmd
		}
	}

	m.table, cmd = m.table.Update(msg)
	return m, cmd
}

func (m MainPage) View() tea.View {
	var baseStyle = lipgloss.NewStyle().
		BorderStyle(lipgloss.NormalBorder()).
		BorderForeground(lipgloss.Color("240"))

	statusBar := "[↑↓] select · [Enter] inspect · [q] quit · [s] sort · [f] filter"

	v := tea.NewView(baseStyle.Render(m.table.View()) + "\n" + statusBar + "\n")
	v.AltScreen = true

	return v
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

func updateTableSize(m *MainPage, newWidth int, newHeight int) {
	columnsCount := len(m.table.Columns())

	m.table.SetWidth(newWidth - (HorizontalPadding * 2))
	m.table.SetHeight(newHeight - (VerticalPadding * 2))

	resizedColumnWidth := m.table.Width() - (columnsCount * CellPadding) - FirstColumnWidth
	nameColumnWidth := resizedColumnWidth / 2
	portColumnWidth := resizedColumnWidth - nameColumnWidth

	m.table.Columns()[1].Width = nameColumnWidth
	m.table.Columns()[2].Width = portColumnWidth
}

func Initialize() tea.Model {
	columns := []table.Column{
		{Title: "PID", Width: 10},
		{Title: "Name", Width: 33},
		{Title: "Socket Info", Width: 33},
	}

	rows := mapProcessItem()

	t := table.New(
		table.WithColumns(columns),
		table.WithRows(rows),
		table.WithFocused(true),
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
