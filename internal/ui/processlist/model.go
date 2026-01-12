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
	"netps/internal/ui/common/command"
	"netps/internal/ui/common/lifecycle"
	"netps/internal/ui/common/sendsignal"
	"netps/internal/ui/message"
	"strings"

	"strconv"

	"log"

	"charm.land/bubbles/v2/table"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
)

const HorizontalPadding = 1
const VerticalPadding = 2

type Model struct {
	processSummaries []process.ProcessSummary
	tableModel       table.Model

	sendSignalModalModel sendsignal.Model

	ctx                       context.Context
	cancel                    context.CancelFunc
	operationMode             lifecycle.Mode
	appTheme                  common.Theme
	commandManager            *command.Manager
	windowWidth, windowHeight int
}

func New(theme common.Theme, commandManager *command.Manager) (Model, error) {
	sendSignal := sendsignal.New()
	ctx, cancel := context.WithCancel(context.Background())

	err := commandManager.SetContext(command.ContextProcessListScreen)
	if err != nil {
		cancel()
		return Model{}, err
	}

	err = registerContextualCommands(commandManager)
	if err != nil {
		cancel()
		return Model{}, err
	}

	return Model{
		ctx:                  ctx,
		cancel:               cancel,
		appTheme:             theme,
		operationMode:        lifecycle.ModeIdle,
		commandManager:       commandManager,
		sendSignalModalModel: sendSignal,
	}, nil
}

func (m Model) Init(width, height int) tea.Cmd {
	return tea.Sequence(
		InitWindow(width, height),
		tea.Batch(
			HydrateRunningProcesses(m.ctx),
		),
	)
}

func (m Model) Update(msg tea.Msg) (Model, tea.Cmd) {
	if _, isSideEffect := msg.(lifecycle.SideEffectMsg); isSideEffect {
		if lifecycle.ShouldCancelSideEffects(m.ctx) {
			return m, nil
		}
	}

	var cmd tea.Cmd

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		if len(m.tableModel.Columns()) == 0 {
			m.tableModel = m.initProcessTable() // init the table here because tea.WindowSizeMsg is the first message to be triggered for some reason
			m.updateTableStyle()
		}
		m.updateWindowSize(msg.Width, msg.Height)
		m.updateTableSize(msg.Width, msg.Height)
	case initMsg:
		m.updateWindowSize(msg.Width, msg.Height)
		m.updateTableSize(m.windowWidth, m.windowHeight) // need to update so that it recalculates table size after back from detail screen
		m.sendSignalModalModel.Initialize()
	case processSummariesLoadedMsg:
		m.updateTableRows(msg.ProcessSummaries)
		m.updateTableSize(m.windowWidth, m.windowHeight)
		m.tableModel.Focus() // Safe to auto-focus: if not, the table won't update the screen with the new width from updateTableSize unless you resize the terminal
	case hydrationErrorMsg:
		m.cancel()
		return m, tea.Quit // might later add an error view. No action needed now.
	case lifecycle.SendSignalMsg:
		m.operationMode = lifecycle.ModeSendSignal
		m.updateTableStyle()
	case lifecycle.CloseSendSignalModalMsg:
		m.operationMode = lifecycle.ModeIdle
		m.updateTableStyle()
	case tea.KeyMsg:
		c := m.commandManager.GetCommand(command.ToKeyPress(msg.String()))
		switch c {
		case command.CommandQuit:
			return m.handleQuit()
		case command.CommandClose:
			return m.handleBack()
		case command.CommandInspect:
			return m.handleInspect()
		case command.CommandSendSignal:
			return m.handleSendSignalOpen()
		}
	}

	err := m.setCurrentCommandContext()
	if err != nil {
		log.Fatalf("Process Detail Model Error: %v", err)
	}

	if m.operationMode == lifecycle.ModeSendSignal {
		m.sendSignalModalModel, cmd = m.sendSignalModalModel.Update(msg)
		return m, cmd
	} else {
		m.tableModel, cmd = m.tableModel.Update(msg)
		return m, cmd
	}

}

func (m Model) View() tea.View {
	var v tea.View
	v.AltScreen = true
	layers := []*lipgloss.Layer{}
	baseLayerComponents := []string{}
	var baseStyle = lipgloss.NewStyle().
		BorderStyle(lipgloss.NormalBorder()).
		BorderForeground(lipgloss.Color("240"))
	processCount := fmt.Sprintf("showing %d from %d processes", m.getShowingProcessCount(), len(m.processSummaries))
	statusBar := common.StatusBar(m.appTheme, m.windowWidth, m.modeName(), m.modeColor(), processCount, "", common.ColorModeNeutral)
	actionBar := common.ActionBar(m.windowWidth, m.commandManager.GenerateContextHelp())
	baseLayerComponents = append(baseLayerComponents, baseStyle.Render(m.tableModel.View()))
	baseLayerComponents = append(baseLayerComponents, statusBar)
	baseLayerComponents = append(baseLayerComponents, actionBar)
	baseLayer := lipgloss.NewLayer(
		strings.Join(baseLayerComponents, "\n"),
	).Z(0)
	layers = append(layers, baseLayer)

	if m.operationMode == lifecycle.ModeSendSignal {
		signalList := m.sendSignalModalModel
		signalListWidth := lipgloss.Width(signalList.Modal)
		signalListHeight := lipgloss.Height(signalList.Modal)
		modalLayer := lipgloss.NewLayer(signalList.View().Content).
			X((m.windowWidth / 2) - (signalListWidth / 2)).
			Y((m.windowHeight / 2) - (signalListHeight / 2)).
			Z(1)
		layers = append(layers, modalLayer)
	}

	canvas := lipgloss.NewCanvas(layers...)
	v.SetContent(canvas)

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
	m.windowWidth = w
	m.windowHeight = h
}

func (m *Model) modeName() string {
	if m.operationMode == lifecycle.ModeSendSignal {
		return "Send Signal"
	}
	return "Process List"
}

func (m *Model) updateTableSize(newWidth int, newHeight int) {
	newTableWidth := newWidth - (HorizontalPadding * (len(m.tableModel.Columns()) - 1))
	m.tableModel.SetWidth(newTableWidth)
	processCount := fmt.Sprintf("showing %d from %d processes", m.getShowingProcessCount(), len(m.processSummaries))

	statusBarHeight := lipgloss.Height(common.StatusBar(m.appTheme, m.windowWidth, m.modeName(), m.modeColor(), processCount, "", common.ColorModeNeutral))
	actionBarHeight := lipgloss.Height(common.ActionBar(m.windowWidth, m.commandManager.GenerateContextHelp()))
	m.tableModel.SetHeight(newHeight - VerticalPadding - statusBarHeight - actionBarHeight)

	maxFieldLenghts := maxFieldLengths(m.processSummaries)
	columnsTotalWidth := 0
	for _, fieldLength := range maxFieldLenghts {
		columnsTotalWidth += fieldLength
	}
	lastColumnWidth := max(1, newTableWidth-columnsTotalWidth)

	for i := 0; i < len(m.tableModel.Columns())-1; i++ {
		title := m.tableModel.Columns()[i].Title
		m.tableModel.Columns()[i].Width = maxFieldLenghts[title]
	}

	m.tableModel.Columns()[len(m.tableModel.Columns())-1].Width = lastColumnWidth
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
	return t
}

func (m *Model) updateTableStyle() {
	s := table.DefaultStyles()
	s.Header = s.Header.
		BorderStyle(lipgloss.NormalBorder()).
		BorderBottom(true).
		Bold(false)
	s.Selected = s.Selected.
		Bold(false)
	if m.operationMode == lifecycle.ModeSendSignal {
		s.Header = s.Header.BorderForeground(lipgloss.Color(m.appTheme.ColorInactive))
		s.Selected = s.Selected.
			Foreground(lipgloss.Color(m.appTheme.ColorForegroundSubtle)).
			Background(lipgloss.Color(m.appTheme.ColorInactive))
	} else {
		s.Header = s.Header.BorderForeground(lipgloss.Color(m.appTheme.ColorForegroundDecoration))
		s.Selected = s.Selected.
			Foreground(lipgloss.Color(m.appTheme.ColorForegroundBase)).
			Background(lipgloss.Color(m.appTheme.ColorHighlight))
	}
	m.tableModel.SetStyles(s)
}

func (m *Model) updateTableRows(summaries []process.ProcessSummary) {
	m.processSummaries = summaries
	rows := mapProcessItem(summaries)
	m.tableModel.SetRows(rows)
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
	tableHeight := m.tableModel.Height()

	showingProcessCount := min(processCount, tableHeight)
	return showingProcessCount
}

func (m *Model) setCurrentCommandContext() error {
	var err error
	if m.operationMode == lifecycle.ModeSendSignal {
		err = m.commandManager.SetContext(command.ContextSendSignal)
	} else {
		err = m.commandManager.SetContext(command.ContextProcessListScreen)
	}

	return err
}

func registerContextualCommands(commandManager *command.Manager) error {
	err := commandManager.RegisterContextCommand(command.ContextProcessListScreen, command.KeyUp, command.CommandMove)
	if err != nil {
		return err
	}
	err = commandManager.RegisterContextCommand(command.ContextProcessListScreen, command.KeyDown, command.CommandMove)
	if err != nil {
		return err
	}
	err = commandManager.RegisterContextCommand(command.ContextProcessListScreen, command.KeyEnter, command.CommandInspect)
	if err != nil {
		return err
	}
	err = commandManager.RegisterContextCommand(command.ContextProcessListScreen, command.KeyS, command.CommandSendSignal)
	if err != nil {
		return err
	}

	err = commandManager.RegisterContextCommand(command.ContextSendSignal, command.KeyUp, command.CommandMove)
	if err != nil {
		return err
	}
	err = commandManager.RegisterContextCommand(command.ContextSendSignal, command.KeyDown, command.CommandMove)
	if err != nil {
		return err
	}
	err = commandManager.RegisterContextCommand(command.ContextSendSignal, command.KeyEnter, command.CommandExecute)
	if err != nil {
		return err
	}
	err = commandManager.RegisterContextCommand(command.ContextSendSignal, command.KeyEsc, command.CommandClose)
	if err != nil {
		return err
	}

	return nil
}

func (m Model) handleQuit() (Model, tea.Cmd) {
	if m.operationMode == lifecycle.ModeSendSignal {
		return m, func() tea.Msg {
			return lifecycle.CloseSendSignalModalMsg{}
		}
	} else {
		return m, tea.Quit
	}
}

func (m Model) handleInspect() (Model, tea.Cmd) {
	row := m.tableModel.SelectedRow()
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

func (m *Model) modeColor() common.ColorMode {
	if m.operationMode == lifecycle.ModeSendSignal {
		return common.ColorModeSpecial
	} else {
		return common.ColorModeNeutral
	}
}

func (m Model) handleSendSignalOpen() (Model, tea.Cmd) {
	if m.operationMode == lifecycle.ModeIdle {
		return m, func() tea.Msg {
			return lifecycle.SendSignalMsg{}
		}
	} else {
		return m, func() tea.Msg {
			return nil
		}
	}
}

func (m Model) handleBack() (Model, tea.Cmd) {
	if m.operationMode == lifecycle.ModeSendSignal {
		return m, func() tea.Msg {
			return lifecycle.CloseSendSignalModalMsg{}
		}
	} else {
		return m, func() tea.Msg {
			return nil
		}
	}
}
