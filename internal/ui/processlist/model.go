package processlist

/* PROCESS LIST SCREEN
 The screen that shows running processes as a list/table
 Invariants:
	1. This model hydrates exactly once.
	2. Table initialization happens on first WindowSizeMsg.
	3. Focus is forced after hydration to ensure width recalculation is rendered.
	4. This screen does not preserve selection across resizes.
*/

import (
	"context"
	"fmt"
	"netps/internal/process"
	"netps/internal/procfs"
	"netps/internal/sysconf"
	"netps/internal/ui/common"
	"netps/internal/ui/common/command"
	"netps/internal/ui/common/hydration"

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
const FieldProcessSummaries = "processSummaries"

type Model struct {
	processSummaries []process.ProcessSummary
	tableModel       table.Model

	sendSignalModalModel sendsignal.Model

	processService       *process.Service
	hydrationCoordinator *hydration.Coordinator

	ctx                       context.Context
	cancel                    context.CancelFunc
	operationMode             common.Mode
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

	coordinator := hydration.NewCoordinator()
	err = coordinator.Register(FieldProcessSummaries, hydration.RarityCommon, true, true)
	if err != nil {
		cancel()
		return Model{}, err
	}

	return Model{
		ctx:                  ctx,
		cancel:               cancel,
		appTheme:             theme,
		operationMode:        common.ModeIdle,
		commandManager:       commandManager,
		sendSignalModalModel: sendSignal,
		processService:       processService,
		hydrationCoordinator: coordinator,
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
	if _, isSideEffect := msg.(common.SideEffectMsg); isSideEffect {
		if common.ShouldCancelSideEffects(m.ctx) {
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
		m.hydrationCoordinator.HydrateAllFields()
		m.sendSignalModalModel.Initialize()
	case processSummariesHydratedMsg:
		field, err := m.hydrationCoordinator.GetField(FieldProcessSummaries)
		if err != nil {
			log.Fatalf("Process List Screen error at processSummariesHydratedMsg: %v", err)
		}
		if msg.err == nil && field.WouldChange(hydration.StateSuccess, msg.err) {
			field.SetSuccess()
			m.updateTableRows(msg.processSummaries)
			m.updateTableSize()
			m.tableModel.SetCursor(0)
			m.tableModel.Focus() // Safe to auto-focus: if not, the table won't update the screen with the new width from updateTableSize unless you resize the terminal
		} else if msg.err != nil && field.WouldChange(hydration.StateError, msg.err) {
			field.SetError(msg.err)
			m.updateTableRows(msg.processSummaries)
			m.updateTableSize()
			m.tableModel.Focus() // Safe to auto-focus: if not, the table won't update the screen with the new width from updateTableSize unless you resize the terminal
		}
	case common.SendSignalMsg:
		m.operationMode = common.ModeSendSignal
		m.sendSignalModalModel.SetProcessInfo(msg.ProcessPID, msg.ProcessName)
		m.updateTableStyle()
		err := m.setCurrentCommandContext()
		if err != nil {
			log.Fatalf("[lifecycle.SendSignalMsg] Process Detail Model Error: %v", err)
		}
	case common.CloseSendSignalModalMsg:
		m.operationMode = common.ModeIdle
		m.updateTableStyle()
	case common.RetryMsg:
		if m.operationMode == common.ModeSendSignal {
			m.operationMode = common.ModeIdle
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

	if m.operationMode == common.ModeSendSignal {
		m.sendSignalModalModel, cmd = m.sendSignalModalModel.Update(msg)
		return m, cmd
	} else {
		m.tableModel, cmd = m.tableModel.Update(msg)
	}
	return m, cmd
}

func (m Model) View() tea.View {
	screenState := hydration.DeriveScreenPhase(m.hydrationCoordinator)
	var v tea.View
	v.AltScreen = true
	switch screenState {
	case hydration.PhaseInit:
		v.SetContent("\n  Initializing...")
	default:
		layers := []*lipgloss.Layer{}
		completedHydration, totalHydration := m.hydrationCoordinator.GetHydrationProgress()
		baseLayer := renderBaseLayer(
			m.appTheme,
			m.tableModel.View(),
			m.windowWidth,
			m.modeName(),
			m.modeColor(),
			fmt.Sprintf("showing %d from %d processes", m.getShowingProcessCount(), len(m.processSummaries)),
			m.commandManager.GenerateContextHelp(),
			m.hydrationCoordinator.GetErrorsAsString(),
			screenState,
			m.hydrationCoordinator.MandatorySucceeded(),
			completedHydration,
			totalHydration,
			0,
		)
		layers = append(layers, baseLayer)

		if m.operationMode == common.ModeSendSignal {
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
	screenState hydration.ScreenPhase,
	mandatorySatisfied bool,
	completedHydration int,
	totalHydration int,
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
	case hydration.PhaseHydrationsInProgress:
		statusBar = common.NotificationBar(theme, common.ColorModeNeutral, width, fmt.Sprintf("Fetching incomplete data... [completed: %d/%d]", completedHydration, totalHydration))
	case hydration.PhaseHydrationFinished:
		if mandatorySatisfied {
			statusBar = common.StatusBar(theme, width, modeName, colorMode, statusBarInfo, "", common.ColorModeSuccess)
		} else {
			statusBar = common.StatusBar(theme, width, modeName, colorMode, statusBarInfo, "", common.ColorModeWarning)
			if len(errors) > 0 {
				errorPanel := common.ErrorPanel(theme, width, errors)
				baseLayerComponents = append(baseLayerComponents, errorPanel)
			}
		}
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
	if m.operationMode == common.ModeSendSignal {
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

	statusBar := common.StatusBar(m.appTheme, m.windowWidth, m.modeName(), m.modeColor(), m.processCount(), "", common.ColorModeNeutral)
	actionBar := common.ActionBar(m.windowWidth, m.commandManager.GenerateContextHelp())
	statusBarHeight := lipgloss.Height(statusBar)
	actionBarHeight := lipgloss.Height(actionBar)

	screenState := hydration.DeriveScreenPhase(m.hydrationCoordinator)
	switch screenState {
	case hydration.PhaseHydrationFinished:
		if m.hydrationCoordinator.ErrorsExist() {
			errorPanel := common.ErrorPanel(m.appTheme, m.windowWidth, m.hydrationCoordinator.GetErrorsAsString())
			errorsPanelHeight := lipgloss.Height(errorPanel)
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
	if m.operationMode == common.ModeSendSignal {
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
	if m.operationMode == common.ModeSendSignal {
		err = m.commandManager.SetContext(command.ContextSendSignal)
	} else {
		switch hydration.DeriveScreenPhase(m.hydrationCoordinator) {
		case hydration.PhaseHydrationFinished:
			if m.hydrationCoordinator.MandatorySucceeded() {
				err = m.commandManager.SetContext(command.ContextProcessListScreen)
			} else {
				err = m.commandManager.SetContext(command.ContextInoperableHydrationError)
			}
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
	if m.operationMode == common.ModeSendSignal {
		return m, func() tea.Msg {
			return common.CloseSendSignalModalMsg{}
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
	if m.operationMode == common.ModeSendSignal {
		return common.ColorModeSpecial
	} else {
		return common.ColorModeNeutral
	}
}

func (m Model) handleSendSignalOpen(selectedProcessPID int, selectedProcessName string) (Model, tea.Cmd) {
	if m.operationMode == common.ModeIdle {
		return m, func() tea.Msg {
			return common.SendSignalMsg{
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
	if m.operationMode == common.ModeSendSignal {
		return m, func() tea.Msg {
			return common.CloseSendSignalModalMsg{}
		}
	} else {
		return m, func() tea.Msg {
			return nil
		}
	}
}

func (m Model) handleErrorRetry() (Model, tea.Cmd) {
	switch hydration.DeriveScreenPhase(m.hydrationCoordinator) {
	case hydration.PhaseHydrationFinished:
		if m.operationMode == common.ModeSendSignal {
			return m, func() tea.Msg {
				return nil // when mode is send signal, user should not have access to retry error
			}
		} else if m.hydrationCoordinator.ErrorsExist() {
			return m, func() tea.Msg {
				return common.RetryMsg{}
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
	retryMap := map[string]tea.Cmd{
		FieldProcessSummaries: HydrateRunningProcesses(m.ctx, m.processService),
	}

	for _, fieldName := range m.hydrationCoordinator.GetFieldsForRetry() {
		m.hydrationCoordinator.HydrateField(fieldName)
		if cmd, ok := retryMap[fieldName]; ok {
			commands = append(commands, cmd)
		}
	}

	return commands
}
