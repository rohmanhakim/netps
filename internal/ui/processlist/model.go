// Invariants:
// 1. This model hydrates exactly once.
// 2. Table initialization happens on first WindowSizeMsg.
// 3. Focus is forced after hydration to ensure width recalculation is rendered.
// 4. This screen does not preserve selection across resizes.

package processlist

import (
	"context"
	"fmt"
	"netps/internal/process"
	"netps/internal/ui/common"
	"netps/internal/ui/message"

	"strconv"

	"charm.land/bubbles/v2/table"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
)

const HorizontalPadding = 1
const VerticalPadding = 2

type Model struct {
	processSummaries []process.ProcessSummary
	table            table.Model
	idleHelpItems    []string
	ctx              context.Context
	cancel           context.CancelFunc
	width, height    int
	mode             string
	modeColor        string
}

func New() Model {
	idleHelpItems := []string{
		"[↑↓] select",
		"[m] mult.select",
		"[i] inspect",
		"[s] send signal",
		"[f] filter",
		"[o] order",
		"[q] quit",
	}
	ctx, cancel := context.WithCancel(context.Background())
	return Model{
		mode:          "Process List",
		idleHelpItems: idleHelpItems,
		ctx:           ctx,
		cancel:        cancel,
	}
}

func (m Model) Init() tea.Cmd {
	return tea.Batch(
		HydrateRunningProcesses(m.ctx),
	)
}

func (m Model) Update(msg tea.Msg) (Model, tea.Cmd) {
	var cmd tea.Cmd

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		if len(m.table.Columns()) == 0 {
			m.table = m.initProcessTable()
		}
		m.updateWindowSize(msg.Width, msg.Height)
		m.updateTableSize(msg.Width, msg.Height)
	case processSummariesLoadedMsg:
		m.updateTableRows(msg.ProcessSummaries)
		m.updateTableSize(m.width, m.height)
		m.table.Focus() // Safe to auto-focus: if not, the table won't update the screen with the new width from updateTableSize unless you resize the terminal
	case hydrationErrorMsg:
		m.cancel()
		return m, tea.Quit // might later add an error view. No action needed now.
	case tea.KeyMsg:
		switch msg.String() {
		case "esc":
			if m.table.Focused() {
				m.table.Blur()
			} else {
				m.table.Focus()
			}
		case "q", "ctrl+c":
			m.cancel()
			return m, tea.Quit
		case "enter":
			row := m.table.SelectedRow()
			if len(row) == 0 {
				return m, nil
			}
			pid, err := strconv.Atoi(row[0])
			if err != nil {
				return m, func() tea.Msg {
					return hydrationErrorMsg{Error: err}
				}
			}
			return m, func() tea.Msg {
				return message.GoToProcessDetail{
					PID:  pid,
					Name: row[1],
				}
			}
		}
	}

	if m.mode == "Process List" {
		m.modeColor = "243"
	} else {
		m.modeColor = "57"
	}

	m.table, cmd = m.table.Update(msg)
	return m, cmd
}

func (m Model) View() tea.View {
	var baseStyle = lipgloss.NewStyle().
		BorderStyle(lipgloss.NormalBorder()).
		BorderForeground(lipgloss.Color("240"))
	processCount := fmt.Sprintf("showing %d from %d processes", m.getShowingProcessCount(), len(m.processSummaries))
	statusBar := common.StatusBar(m.width, m.mode, m.modeColor, processCount)
	actionBar := common.ActionBar(m.width, m.idleHelpItems)
	v := tea.NewView(baseStyle.Render(m.table.View()) + "\n" + statusBar + "\n" + actionBar + "\n")
	v.AltScreen = true
	return v
}

func mapProcessItem(processSummaries []process.ProcessSummary) []table.Row {
	var rows []table.Row
	for _, p := range processSummaries {
		r := table.Row{
			strconv.Itoa(p.PID),
			p.Name,
			formatSocketText(p.LSocketCount, p.ESocketCount, p.CSocketCount),
			p.LPortsText,
		}
		rows = append(rows, r)
	}
	return rows
}

func formatSocketText(lCount int, eCount int, cCount int) string {
	return fmt.Sprintf("%dL %dE %dC", lCount, eCount, cCount)
}

func (m *Model) updateWindowSize(w int, h int) {
	m.width = w
	m.height = h
}

func (m *Model) updateTableSize(newWidth int, newHeight int) {
	newTableWidth := newWidth - (HorizontalPadding * (len(m.table.Columns()) - 1))
	m.table.SetWidth(newTableWidth)
	processCount := fmt.Sprintf("showing %d from %d processes", m.getShowingProcessCount(), len(m.processSummaries))

	statusBarHeight := lipgloss.Height(common.StatusBar(m.width, m.mode, m.modeColor, processCount))
	actionBarHeight := lipgloss.Height(common.ActionBar(m.width, m.idleHelpItems))
	m.table.SetHeight(newHeight - VerticalPadding - statusBarHeight - actionBarHeight)

	maxFieldLenghts := maxFieldLengths(m.processSummaries)
	columnsTotalWidth := 0
	for _, fieldLength := range maxFieldLenghts {
		columnsTotalWidth += fieldLength
	}
	lastColumnWidth := max(1, newTableWidth-columnsTotalWidth)

	for i := 0; i < len(m.table.Columns())-1; i++ {
		title := m.table.Columns()[i].Title
		m.table.Columns()[i].Width = maxFieldLenghts[title]
	}

	m.table.Columns()[len(m.table.Columns())-1].Width = lastColumnWidth
}

func (m *Model) initProcessTable() table.Model {
	columns := []table.Column{
		{Title: "PID"},
		{Title: "NAME"},
		{Title: "SOCKS"},
		{Title: "L.PORTS"},
	}
	t := table.New(
		table.WithColumns(columns),
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
	return t
}

func (m *Model) updateTableRows(summaries []process.ProcessSummary) {
	m.processSummaries = summaries
	rows := mapProcessItem(summaries)
	m.table.SetRows(rows)
}

func maxFieldLengths(summaries []process.ProcessSummary) map[string]int {
	maxLens := map[string]int{
		"PID":     3, // set initial value to column header's length
		"NAME":    4,
		"SOCKS":   5,
		"L.PORTS": 7,
	}
	if len(summaries) == 0 {
		return maxLens
	}
	for _, p := range summaries {
		maxLens["PID"] = max(maxLens["PID"], len(strconv.Itoa(p.PID)))
		maxLens["NAME"] = max(maxLens["NAME"], len(p.Name))
		maxLens["SOCKS"] = max(maxLens["SOCKS"], len(formatSocketText(p.LSocketCount, p.ESocketCount, p.CSocketCount)))
		maxLens["L.PORTS"] = max(maxLens["L.PORTS"], len(p.LPortsText))
	}
	return maxLens
}

func (m Model) getShowingProcessCount() int {
	processCount := len(m.processSummaries)
	tableHeight := m.table.Height()

	showingProcessCount := min(processCount, tableHeight)
	return showingProcessCount
}
