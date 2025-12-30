package processlist

import (
	"context"
	"fmt"
	"netps/internal/process"
	"netps/internal/procfs"
	"netps/internal/sysconf"
	"netps/internal/ui/message"

	"os"
	"strconv"
	"strings"

	"charm.land/bubbles/v2/table"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/charmbracelet/x/term"
)

const HorizontalPadding = 1
const VerticalPadding = 2
const CellPadding = 2
const FirstColumnWidth = 10

type Model struct {
	processSummaries []process.ProcessSummary
	table            table.Model
}

func New() Model {
	return initializeProcessListScreen()
}

func (m Model) Init() tea.Cmd { return nil }

func (m Model) Update(msg tea.Msg) (Model, tea.Cmd) {
	var cmd tea.Cmd

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		updateTableSize(&m, msg.Width, msg.Height)
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
			pid, err := strconv.Atoi(m.table.SelectedRow()[0])
			if err != nil {
				panic(err)
			}
			return m, func() tea.Msg {
				return message.GoToProcessDetail{
					PID:  pid,
					Name: m.table.SelectedRow()[1],
				}
			}
		}
	}

	m.table, cmd = m.table.Update(msg)
	return m, cmd
}

func (m Model) View() tea.View {
	var baseStyle = lipgloss.NewStyle().
		BorderStyle(lipgloss.NormalBorder()).
		BorderForeground(lipgloss.Color("240"))
	w, h, err := term.GetSize(uintptr(os.Stdout.Fd()))

	footerStyle := lipgloss.NewStyle().
		Align(lipgloss.Center).
		Foreground(lipgloss.Color("#FFFFFF"))

	statusBar := footerStyle.Render("[↑↓] select · [Enter] inspect · [q] quit · [s] sort · [f] filter")
	newHeight := h - lipgloss.Height(statusBar)
	if err == nil {
		updateTableSize(&m, w, newHeight)
		baseStyle = baseStyle.Height(newHeight).Width(w)
	}

	m.table.Focus()

	v := tea.NewView(baseStyle.Render(m.table.View()) + "\n" + statusBar + "\n")
	v.AltScreen = true

	return v
}

func mapProcessItem(processSummaries []process.ProcessSummary) []table.Row {
	var rows []table.Row
	for _, p := range processSummaries {
		var sb strings.Builder

		_, err := fmt.Fprintf(&sb, "%dL %dE %dC", p.LSocketCount, p.ESocketCount, p.CSocketCount)
		if err != nil {
			panic(err)
		}

		r := table.Row{
			strconv.Itoa(p.PID),
			p.Name,
			sb.String(),
			p.LPortsText,
		}
		rows = append(rows, r)
	}
	return rows
}

func updateTableSize(m *Model, newWidth int, newHeight int) {
	columnsCount := len(m.table.Columns())

	newTableWidth := newWidth - (HorizontalPadding * 10)
	m.table.SetWidth(newTableWidth)
	m.table.SetHeight(newHeight - (VerticalPadding * 2))

	avgColWidth := newTableWidth / (columnsCount)
	firstColumnWidth := newTableWidth - ((columnsCount - 1) * avgColWidth)
	m.table.Columns()[0].Width = firstColumnWidth
	for i := 1; i < columnsCount; i++ {
		m.table.Columns()[i].Width = avgColWidth
	}
}

func initializeProcessListScreen() Model {

	procfsClient := procfs.NewClient()
	sysconfClient := sysconf.NewClient()

	service := process.NewService(procfsClient, procfsClient, sysconfClient, sysconfClient, procfsClient, procfsClient)
	processSummaries, err := service.GetRunningSummaries(context.Background())

	if err != nil {
		panic(err)
	}

	columns := []table.Column{
		{Title: "PID"},
		{Title: "NAME"},
		{Title: "SOCKS"},
		{Title: "L.PORTS"},
	}

	rows := mapProcessItem(processSummaries)

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

	return Model{
		processSummaries: processSummaries,
		table:            t,
	}
}
