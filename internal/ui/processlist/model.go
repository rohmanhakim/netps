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
	"netps/internal/procfs"
	"netps/internal/sysconf"
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

	processService *process.Service

	processSumarriesHydration processSummariesHydrationData

	ctx                       context.Context
	cancel                    context.CancelFunc
	operationMode             lifecycle.Mode
	appTheme                  common.Theme
	commandManager            *command.Manager
	windowWidth, windowHeight int
}

func New(theme common.Theme, commandManager *command.Manager) (Model, error) {
	sendSignal := sendsignal.New(theme)
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

	procfsClient := procfs.NewClient()
	sysconfClient := sysconf.NewClient()

	cfg := process.Config{
		Process:   procfsClient,
		Detail:    procfsClient,
		Clocktick: sysconfClient,
		PageSize:  sysconfClient,
		UpTime:    procfsClient,
		Resource:  procfsClient,
		User:      procfsClient,
	}
	processService := process.NewProcessService(cfg)

	return Model{
		ctx:                  ctx,
		cancel:               cancel,
		appTheme:             theme,
		operationMode:        lifecycle.ModeIdle,
		commandManager:       commandManager,
		sendSignalModalModel: sendSignal,
		processService:       processService,
	}, nil
}

func (m Model) Init(width, height int) tea.Cmd {
	return tea.Sequence(
		InitWindow(width, height),
		tea.Batch(
			HydrateRunningProcesses(m.ctx, m.processService),
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
		m.updateTableSize()
	case initMsg:
		m.updateWindowSize(msg.Width, msg.Height)
		m.updateTableSize() // need to update so that it recalculates table size after back from detail screen
		m.sendSignalModalModel.Initialize()
	case processSummariesHydratedMsg:
		if msg.err == nil && m.processSummariesWouldChange(lifecycle.StateSuccess, msg.err) {
			m.processSumarriesHydration.state = lifecycle.StateSuccess
			m.processSumarriesHydration.err = nil
			m.updateTableRows(msg.processSummaries)
			m.updateTableSize()
			m.tableModel.SetCursor(0)
			m.tableModel.Focus() // Safe to auto-focus: if not, the table won't update the screen with the new width from updateTableSize unless you resize the terminal
		} else if m.processSummariesWouldChange(lifecycle.StateError, msg.err) {
			m.processSumarriesHydration.state = lifecycle.StateError
			m.processSumarriesHydration.err = msg.err
			m.updateTableRows(msg.processSummaries)
			m.updateTableSize()
			m.tableModel.Focus() // Safe to auto-focus: if not, the table won't update the screen with the new width from updateTableSize unless you resize the terminal
		}
	case lifecycle.SendSignalMsg:
		m.operationMode = lifecycle.ModeSendSignal
		m.sendSignalModalModel.SetProcessInfo(msg.ProcessPID, msg.ProcessName)
		m.updateTableStyle()
		err := m.setCurrentCommandContext()
		if err != nil {
			log.Fatalf("[lifecycle.SendSignalMsg] Process Detail Model Error: %v", err)
		}
	case lifecycle.CloseSendSignalModalMsg:
		m.operationMode = lifecycle.ModeIdle
		m.updateTableStyle()
	case lifecycle.RetryMsg:
		if m.operationMode == lifecycle.ModeSendSignal {
			m.operationMode = lifecycle.ModeIdle
		}
		m.resetContext()
		commands := m.collectRetryCommands()

		// have to adjust viewport and current command's context here because this retry returns immediately skipping main viewport updating line
		m.updateTableSize()

		err := m.setCurrentCommandContext()
		if err != nil {
			log.Fatalf("Process Detail Screen error at retry: %v", err)
		}
		return m, tea.Batch(commands...)
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
			pid, err := m.getSelectedProcessPID()
			if err != nil {
				log.Fatalf("command.CommandSendSignal error: %s", err.Error())
			}
			name := m.getSelectedProcessName()
			return m.handleSendSignalOpen(pid, name)
		case command.CommandRetry:
			return m.handleErrorRetry()
		}
	}

	err := m.setCurrentCommandContext()
	if err != nil {
		log.Fatalf("[setCurrentCommandContext] Process Detail Model Error: %v", err)
	}

	if m.operationMode == lifecycle.ModeSendSignal {
		m.sendSignalModalModel, cmd = m.sendSignalModalModel.Update(msg)
		return m, cmd
	} else {
		m.tableModel, cmd = m.tableModel.Update(msg)
	}
	return m, cmd
}

func (m Model) View() tea.View {
	screenState := m.computeScreenState()
	var v tea.View
	v.AltScreen = true
	switch screenState {
	case lifecycle.StateInit:
		v.SetContent("\n  Initializing...")
	default:
		layers := []*lipgloss.Layer{}
		baseLayer := renderBaseLayer(
			m.appTheme,
			m.tableModel.View(),
			m.windowWidth,
			m.modeName(),
			m.modeColor(),
			fmt.Sprintf("showing %d from %d processes", m.getShowingProcessCount(), len(m.processSummaries)),
			m.commandManager.GenerateContextHelp(),
			m.getErrorsAsString(),
			screenState,
			0,
		)
		layers = append(layers, baseLayer)

		if m.operationMode == lifecycle.ModeSendSignal {
			signalList := m.sendSignalModalModel
			modalLayer := lipgloss.NewLayer(signalList.View().Content).
				X((m.windowWidth / 2) - (signalList.ModalWidth() / 2)).
				Y((m.windowHeight / 2) - (signalList.ModalHeight() / 2)).
				Z(1)
			layers = append(layers, modalLayer)
		}

		canvas := lipgloss.NewCanvas(layers...)
		v.SetContent(canvas)
	}
	return v
}

func renderBaseLayer(
	theme common.Theme,
	content string,
	width int,
	modeName string,
	colorMode common.ColorMode,
	statusBarInfo string,
	helpItems []string,
	errors []string,
	screenState lifecycle.ScreenState,
	zIndex int,
) *lipgloss.Layer {
	baseLayerComponents := []string{}
	var baseStyle = lipgloss.NewStyle().
		BorderStyle(lipgloss.NormalBorder()).
		BorderForeground(lipgloss.Color(theme.ColorForegroundDecoration))
	baseLayerComponents = append(baseLayerComponents, baseStyle.Render(content))

	actionBar := common.ActionBar(width, helpItems)

	var statusBar string
	switch screenState {
	case lifecycle.StateHydrationsInProgress, lifecycle.StateOneHydrationFinished:
		statusBar = common.NotificationBar(theme, common.ColorModeNeutral, width, "Getting Data...")
	case lifecycle.StateHydrationsFinishedErrorsExist:
		statusBar = common.StatusBar(theme, width, modeName, colorMode, statusBarInfo, "", common.ColorModeWarning)
		if len(errors) > 0 {
			errorPanel := common.ErrorPanel(theme, width, errors)
			baseLayerComponents = append(baseLayerComponents, errorPanel)
		}
	case lifecycle.StateHydrationsFinishedAllOK:
		statusBar = common.StatusBar(theme, width, modeName, colorMode, statusBarInfo, "", common.ColorModeSuccess)
	}

	baseLayerComponents = append(baseLayerComponents, statusBar)
	baseLayerComponents = append(baseLayerComponents, actionBar)
	baseLayer := lipgloss.NewLayer(
		strings.Join(baseLayerComponents, "\n"),
	).Z(zIndex)
	return baseLayer
}

func InitWindow(w, h int) tea.Cmd {
	return func() tea.Msg {
		return initMsg{
			Width:  w,
			Height: h,
		}
	}
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

func (m *Model) processCount() string {
	return fmt.Sprintf("showing %d from %d processes", m.getShowingProcessCount(), len(m.processSummaries))
}

func (m *Model) updateTableSize() {
	newTableWidth := m.windowWidth - (HorizontalPadding * (len(m.tableModel.Columns()) - HorizontalPadding))
	m.tableModel.SetWidth(newTableWidth)

	errorPanel := common.ErrorPanel(m.appTheme, m.windowWidth, m.getErrorsAsString())
	statusBar := common.StatusBar(m.appTheme, m.windowWidth, m.modeName(), m.modeColor(), m.processCount(), "", common.ColorModeNeutral)
	actionBar := common.ActionBar(m.windowWidth, m.commandManager.GenerateContextHelp())
	statusBarHeight := lipgloss.Height(statusBar)
	actionBarHeight := lipgloss.Height(actionBar)

	screenState := m.computeScreenState()
	switch screenState {
	case lifecycle.StateHydrationsFinishedErrorsExist:
		errorsPanelHeight := lipgloss.Height(errorPanel)
		if len(m.getErrorsAsString()) > 0 {
			m.tableModel.SetHeight(m.windowHeight - VerticalPadding - errorsPanelHeight - statusBarHeight - actionBarHeight)
		} else {
			m.tableModel.SetHeight(m.windowHeight - VerticalPadding - statusBarHeight - actionBarHeight)
		}
	default:
		m.tableModel.SetHeight(m.windowHeight - VerticalPadding - statusBarHeight - actionBarHeight)
	}

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
		switch m.computeScreenState() {
		case lifecycle.StateHydrationsFinishedAllOK:
			err = m.commandManager.SetContext(command.ContextProcessListScreen)
		case lifecycle.StateHydrationsFinishedErrorsExist:
			err = m.commandManager.SetContext(command.ContextInoperableHydrationError)
		default:
			err = m.commandManager.SetContext(command.ContextHydrating)
		}
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

	err = commandManager.RegisterContextCommand(command.ContextInoperableHydrationError, command.KeyR, command.CommandRetry)
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

func (m *Model) getSelectedProcessPID() (int, error) {
	row := m.tableModel.SelectedRow()
	if len(row) == 0 {
		return -1, nil
	}
	pid, err := strconv.Atoi(row[0])
	if err != nil {
		return -1, err
	}
	return pid, nil
}

func (m *Model) getSelectedProcessName() string {
	row := m.tableModel.SelectedRow()
	if len(row) == 0 {
		return ""
	}
	return row[1]
}

func (m Model) handleInspect() (Model, tea.Cmd) {
	pid, err := m.getSelectedProcessPID()
	if err != nil {
		log.Fatalf("handleInspect() get error: %s", err.Error())
	}
	name := m.getSelectedProcessName()
	return m, func() tea.Msg {
		return message.GoToProcessDetail{
			PID:  pid,
			Name: name,
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

func (m Model) handleSendSignalOpen(selectedProcessPID int, selectedProcessName string) (Model, tea.Cmd) {
	if m.operationMode == lifecycle.ModeIdle {
		return m, func() tea.Msg {
			return lifecycle.SendSignalMsg{
				ProcessPID:  selectedProcessPID,
				ProcessName: selectedProcessName,
			}
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

func (m *Model) getErrorsAsString() []string {
	errorStrings := []string{}
	if m.processSumarriesHydration.state == lifecycle.StateError && m.processSumarriesHydration.err != nil {
		errorStrings = append(errorStrings, "[retryable] "+m.processSumarriesHydration.err.Error())
	}
	return errorStrings
}

func (m *Model) processSummariesWouldChange(newState lifecycle.HydrationState, err error) bool {
	oldState := m.processSumarriesHydration.state
	oldError := m.processSumarriesHydration.err
	return oldState != newState || oldError != err
}

func (m *Model) hydrationErrorsExist() bool {
	return m.processSumarriesHydration.err != nil
}

func (m *Model) allHydrationFinished() bool {
	processSummariesHydrationFinished := m.processSumarriesHydration.state == lifecycle.StateSuccess || m.processSumarriesHydration.state == lifecycle.StateError

	return processSummariesHydrationFinished
}

func (m *Model) allHydrationOK() bool {
	allHydrationOK := m.processSumarriesHydration.state == lifecycle.StateSuccess
	return allHydrationOK
}

func (m *Model) oneHydrationFinished() bool {
	processSummariesHydrationFinished := m.processSumarriesHydration.state == lifecycle.StateSuccess || m.processSumarriesHydration.state == lifecycle.StateError

	return processSummariesHydrationFinished
}

func (m *Model) allHydrating() bool {
	return m.processSumarriesHydration.state == lifecycle.StateHydrating
}

// Calculate the current screen's state
// Central place to derive screen's state
// Pure state rducer so that no multiple state mutations inside this screen
// Multiple subsystems query computeScreenState() in other place is intentional for now
// TO-DO: State caching
func (m *Model) computeScreenState() lifecycle.ScreenState {
	if m.allHydrationFinished() {
		if m.allHydrationOK() {
			return lifecycle.StateHydrationsFinishedAllOK
		} else {
			return lifecycle.StateHydrationsFinishedErrorsExist
		}
	} else {
		if m.oneHydrationFinished() {
			return lifecycle.StateOneHydrationFinished
		} else if m.allHydrating() {
			return lifecycle.StateHydrationsInProgress
		} else {
			return lifecycle.StateInit
		}
	}
}

func (m Model) handleErrorRetry() (Model, tea.Cmd) {
	switch m.computeScreenState() {
	case lifecycle.StateHydrationsFinishedErrorsExist:
		if m.operationMode == lifecycle.ModeSendSignal {
			return m, func() tea.Msg {
				return nil // when mode is send signal, user should not have access to retry error
			}
		} else if m.hydrationErrorsExist() {
			return m, func() tea.Msg {
				return lifecycle.RetryMsg{}
			}
		} else {
			return m, func() tea.Msg {
				return nil
			}
		}

	default:
		return m, func() tea.Msg {
			return nil
		}
	}
}

func (m *Model) resetContext() {
	ctx, cancel := context.WithCancel(context.Background())
	m.ctx = ctx
	m.cancel = cancel
}

func (m *Model) collectRetryCommands() []tea.Cmd {
	commands := []tea.Cmd{}

	if m.shouldRetry(m.processSumarriesHydration.err) {
		m.processSumarriesHydration.err = nil
		m.processSumarriesHydration.state = lifecycle.StateHydrating
		commands = append(commands, HydrateRunningProcesses(m.ctx, m.processService))
	}

	return commands
}

func (m *Model) shouldRetry(err error) bool {
	return err != nil
}
