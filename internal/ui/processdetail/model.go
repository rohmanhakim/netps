package processdetail

/* PROCESS DETAIL SCREEN
 The screen that shows process's information
 Currently has 4 groups of data:
 - Static ID: PID, name, execution path, command line
 - Resource: all info related to resources such as CPU and memory
 - User: Ownership-related info
 - Sockets: sockets info, shows address, port, protocol, currently
 			scoped to only show Listen, Established, and Closed

	Tech Debts:
		High:
			- Split Lifecycle vs Presentation State
			 	- Move “error dismissed” into UI flags
			   	- Keep lifecycle strictly about data
			- Introduce a Hydration Coordinator
			 	- Replace manual allHydrationFinished / oneHydrationFinished
			  	- Single struct representing hydration graph
			- Guard UI Mutations After Cancellation
			 	- Prevent non-side-effect messages from mutating canceled screens
			- Exploit Error Severity in UX
			 	- Disable retry for permanent errors
			  	- Add inline hints (“retry unlikely to succeed”)
		Medium:
			- Stabilize Viewport Height
				- Reserve fixed space for panels
				- Reduce scroll jumps
			- Reduce computeScreenState() Fan-Out
				- Cache locally within Update cycle (not globally)
				- Or introduce a per-tick derived state
			- Extract Key Handling into a Map
				- Easier rebinding
				- Cleaner logic
		Low:
			- Refactor processDetailSection
				- Separate domain formatting from layout
				- Improves testability only
			- Theme Completeness
				- Eliminate remaining hard-coded colors/glyphs
			- Unit Tests for State Transitions
				- Especially retry and cancellation paths
*/

import (
	"context"
	"fmt"
	"log"
	"netps/internal/process"
	"netps/internal/procfs"
	"netps/internal/socket"
	"netps/internal/sysconf"
	"netps/internal/ui/common"
	"netps/internal/ui/common/command"
	"netps/internal/ui/common/hydration"
	"netps/internal/ui/common/sendsignal"
	"netps/internal/ui/message"
	"strings"

	"charm.land/bubbles/v2/viewport"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
)

const (
	FieldStaticId = "StaticId"
	FieldUser     = "User"
	FieldResource = "Resource"
	FieldSockets  = "Sockets"
)

type Model struct {
	PID         int
	ProcessName string

	staticIdHydration StaticIdData
	resourceHydration ResourceData
	userHydration     UserData
	socketsHydration  SocketsData

	viewportModel viewport.Model

	sendSignalModalModel sendsignal.Model

	processService *process.Service
	socketService  *socket.Service

	errorsToRetry        tea.Cmd
	hydrationCoordinator *hydration.Coordinator

	showErrorsPanel bool

	ctx                       context.Context
	cancel                    context.CancelFunc
	operationMode             common.Mode
	appTheme                  common.Theme
	commandManager            *command.Manager
	windowWidth, windowHeight int
}

type styleFunc func(string) string

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
	socketService := socket.NewService(procfsClient)

	coordinator := hydration.NewCoordinator()
	err = coordinator.Register(FieldStaticId, hydration.RarityRare, true, true)
	if err != nil {
		cancel()
		return Model{}, err
	}

	err = coordinator.Register(FieldUser, hydration.RarityCommon, false, false)
	if err != nil {
		cancel()
		return Model{}, err
	}

	err = coordinator.Register(FieldResource, hydration.RarityCommon, true, false)
	if err != nil {
		cancel()
		return Model{}, err
	}

	err = coordinator.Register(FieldSockets, hydration.RarityCommon, true, false)
	if err != nil {
		cancel()
		return Model{}, err
	}

	return Model{
		processService:       processService,
		socketService:        socketService,
		ctx:                  ctx,
		cancel:               cancel,
		appTheme:             theme,
		operationMode:        common.ModeIdle,
		commandManager:       commandManager,
		sendSignalModalModel: sendSignal,
		hydrationCoordinator: coordinator,
		showErrorsPanel:      true,
	}, nil
}

func (m Model) Init(pid int, name string, width, height int) tea.Cmd {
	return tea.Sequence(
		Initialize(pid, name, width, height),
		tea.Batch(
			HydrateStaticIds(m.ctx, pid, m.processService),
			HydrateResource(m.ctx, pid, m.processService),
			HydrateUser(m.ctx, pid, m.processService),
			HydrateSockets(m.ctx, pid, m.socketService),
		),
	)
}

func (m Model) Update(msg tea.Msg) (Model, tea.Cmd) {
	if _, isSideEffect := msg.(common.SideEffectMsg); isSideEffect {
		if common.ShouldCancelSideEffects(m.ctx) {
			return m, nil
		}
	}

	var (
		cmd                         tea.Cmd
		cmds                        []tea.Cmd
		viewportContentColorChanged bool
	)

	dataChanged := false

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.windowWidth = msg.Width
		m.windowHeight = msg.Height
	case initMsg:
		m.resetAllData()
		m.PID = msg.pid
		m.ProcessName = msg.name
		m.operationMode = common.ModeIdle
		m.windowWidth = msg.width
		m.windowHeight = msg.height
		m.showErrorsPanel = true
		m.sendSignalModalModel.Initialize()
		m.hydrationCoordinator.HydrateAllFields()
	case common.RetryMsg:
		if m.operationMode == common.ModeSendSignal {
			m.operationMode = common.ModeIdle
		}
		m.resetContext()
		commands := m.collectRetryCommands()

		// have to adjust viewport and current command's context here because this retry returns immediately skipping main viewport updating line
		m.adjustViewportSize()
		err := m.setCurrentCommandContext()
		if err != nil {
			log.Fatalf("Process Detail Screen error at retry: %v", err)
		}
		return m, tea.Batch(commands...)
	case staticIdHydratedMsg:
		field, err := m.hydrationCoordinator.GetField(FieldStaticId)
		if err != nil {
			log.Fatalf("Process Detail Screen error at staticIdHydratedMsg: %v", err)
		}
		if msg.Err == nil && field.WouldChange(hydration.StateSuccess, msg.Err) {
			field.SetSuccess()
			m.staticIdHydration.ExecPath = msg.ExecPath
			m.staticIdHydration.Command = msg.Command
			m.staticIdHydration.PPID = msg.PPID
			m.staticIdHydration.ParentName = msg.ParentName
			dataChanged = true
		} else if msg.Err != nil && field.WouldChange(hydration.StateError, msg.Err) {
			field.SetError(msg.Err)
			dataChanged = true
		}
	case resourceHydratedMsg:
		field, err := m.hydrationCoordinator.GetField(FieldResource)
		if err != nil {
			log.Fatalf("Process Detail Screen error at resourceHydratedMsg: %v", err)
		}
		if msg.Err == nil && field.WouldChange(hydration.StateSuccess, msg.Err) {
			field.SetSuccess()
			m.resourceHydration.RSSByte = msg.RSSByte
			m.resourceHydration.StartTime = msg.StartTime
			m.resourceHydration.ElapsedTime = msg.ElapsedTime
			m.resourceHydration.VSZByte = msg.VSZByte
			m.resourceHydration.UTime = msg.UTime
			m.resourceHydration.STime = msg.STime
			dataChanged = true
		} else if msg.Err != nil && field.WouldChange(hydration.StateError, msg.Err) {
			field.SetError(msg.Err)
			dataChanged = true
		}
	case userHydratedMsg:
		field, err := m.hydrationCoordinator.GetField(FieldUser)
		if err != nil {
			log.Fatalf("Process Detail Screen error at userHydratedMsg: %v", err)
		}
		if msg.Err == nil && field.WouldChange(hydration.StateSuccess, msg.Err) {
			field.SetSuccess()
			m.userHydration.UserUID = msg.UserUID
			m.userHydration.UserName = msg.UserName
			m.userHydration.UserPrivileged = msg.UserPrivileged
			dataChanged = true
		} else if msg.Err != nil && field.WouldChange(hydration.StateError, msg.Err) {
			field.SetError(msg.Err)
			dataChanged = true
		}
	case socketsHydratedMsg:
		field, err := m.hydrationCoordinator.GetField(FieldSockets)
		if err != nil {
			log.Fatalf("Process Detail Screen error at socketsHydratedMsg: %v", err)
		}
		if msg.Err == nil && field.WouldChange(hydration.StateSuccess, msg.Err) {
			field.SetSuccess()
			m.socketsHydration.Sockets = msg.Sockets
			dataChanged = true
		} else if msg.Err != nil && field.WouldChange(hydration.StateError, msg.Err) {
			field.SetError(msg.Err)
			dataChanged = true
		}
	case common.SendSignalMsg:
		m.operationMode = common.ModeSendSignal
		m.sendSignalModalModel.SetProcessInfo(msg.ProcessPID, msg.ProcessName)
		err := m.setCurrentCommandContext()
		if err != nil {
			log.Fatalf("[lifecycle.SendSignalMsg] Process Detail Model Error: %v", err)
		}
		viewportContentColorChanged = true // opening send signal modal changed the viewport's content color to dim which required to rerender the viewport
	case common.CloseSendSignalModalMsg:
		m.operationMode = common.ModeIdle
		viewportContentColorChanged = true // closing send signal modal changed the viewport's content color to normal which required to rerender the viewport
	case common.DismissnotificationMsg:
		// Dismissing errors hides the panel but does not change data completeness.
		// dismissal is not errors resolution
		// status bar remains “Data Partial” intentionally so that the user be aware
		// TO-DO: Implement retry on-demand when the data is partial, even after dismiss
		m.showErrorsPanel = false

		// m.hydrationCoordinator.ResetAllField()
	case tea.KeyMsg:
		c := m.commandManager.GetCommand(command.ToKeyPress(msg.String()))
		switch c {
		case command.CommandBack:
			return m.handleBack()
		case command.CommandClose:
			return m.handleCloseSendSignal()
		case command.CommandSendSignal:
			return m.handleOpenSendSignal()
		case command.CommandQuit:
			return m.handleQuit()
		case command.CommandRetry:
			return m.handleErrorRetry()
		case command.CommandDismiss:
			return m.handleNotificationDismiss()
		}
	}

	savedY := m.viewportModel.YOffset()
	if m.baseUIRenderableState() {
		m.adjustViewportSize()
	}
	if dataChanged || viewportContentColorChanged {
		m.viewportModel.SetContent(m.renderContent())
	}
	m.viewportModel.SetYOffset(savedY)
	err := m.setCurrentCommandContext()
	if err != nil {
		log.Fatalf("Process Detail Screen error at update: %v", err)
	}

	if m.operationMode == common.ModeSendSignal {
		m.sendSignalModalModel, cmd = m.sendSignalModalModel.Update(msg)
	} else {
		m.viewportModel, cmd = m.viewportModel.Update(msg)
	}
	cmds = append(cmds, cmd)
	return m, tea.Batch(cmds...)
}

func (m Model) baseUIRenderableState() bool {
	return hydration.DeriveScreenPhase(m.hydrationCoordinator) != hydration.PhaseInit
}

func (m Model) View() tea.View {
	screenPhase := hydration.DeriveScreenPhase(m.hydrationCoordinator)
	var v tea.View
	v.AltScreen = true // Bubble Tea v2 forces you to set the AltScren option on every Model's View() function
	switch screenPhase {
	case hydration.PhaseInit:
		v.SetContent("\n  Initializing...")
	default:
		layers := []*lipgloss.Layer{}
		helpItems := m.commandManager.GenerateContextHelp()

		if m.operationMode == common.ModeSendSignal {
			signalList := m.sendSignalModalModel
			modalLayer := lipgloss.NewLayer(signalList.View().Content).
				X((m.windowWidth / 2) - (signalList.ModalWidth() / 2)).
				Y((m.windowHeight / 2) - (signalList.ModalHeight() / 2)).
				Z(1)
			layers = append(layers, modalLayer)
		}
		completedHydration, totalHydration := m.hydrationCoordinator.GetHydrationProgress()
		ui := renderBaseLayer(
			m.appTheme,
			m.viewportModel.View(),
			m.windowWidth,
			m.modeName(),
			m.modeColor(),
			scrollingInfo(m.getScrollingPercent(), m.getVisibleContentPercent()),
			helpItems,
			m.hydrationCoordinator.GetErrorsAsString(),
			m.showErrorsPanel,
			screenPhase,
			m.hydrationCoordinator.ComputeHydrationSummary(),
			completedHydration,
			totalHydration,
			0,
		)
		layers = append(layers, ui)

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
	showErrorsPanel bool,
	screenPhase hydration.ScreenPhase,
	hydrationSummary hydration.HydrationSummary,
	completedHydration int,
	totalHydration int,
	zIndex int,
) *lipgloss.Layer {
	components := []string{}
	components = append(components, content)

	actionBar := common.ActionBar(width, helpItems)

	var screenStateInfoLabel string
	var statusBar string

	switch screenPhase {
	case hydration.PhaseHydrationsInProgress:
		statusBar = common.NotificationBar(theme, common.ColorModeNeutral, width, fmt.Sprintf("Fetching incomplete data... [completed: %d/%d]", completedHydration, totalHydration))
	case hydration.PhaseHydrationFinished:
		if hydrationSummary.AllSuccess {
			statusBar = common.StatusBar(theme, width, modeName, colorMode, lipgloss.JoinHorizontal(lipgloss.Top, statusBarInfo, screenStateInfoLabel), "Data OK", common.ColorModeSuccess)
		} else if hydrationSummary.AnyError && showErrorsPanel {
			statusBar = common.StatusBar(theme, width, modeName, colorMode, lipgloss.JoinHorizontal(lipgloss.Top, statusBarInfo, screenStateInfoLabel), "Data Partial", common.ColorModeWarning)
			errorPanel := common.ErrorPanel(theme, width, errors)
			components = append(components, errorPanel)
		} else if hydrationSummary.AnyError && !showErrorsPanel {
			statusBar = common.StatusBar(theme, width, modeName, colorMode, lipgloss.JoinHorizontal(lipgloss.Top, statusBarInfo, screenStateInfoLabel), "Data Partial", common.ColorModeWarning)
		}
	}

	components = append(components, statusBar)
	components = append(components, actionBar)
	ui := lipgloss.NewLayer(
		strings.Join(components, "\n"),
	).Z(zIndex)
	return ui
}

// Viewport height is calculated based on other elements (errors panel, status bar, action bar)
// Changes may cause scroll jumps when panels appear/disappear.
// To minimize jumps, all changing elements placed on the bottom part of the screen
func (m *Model) adjustViewportSize() {
	savedY := m.viewportModel.YOffset() // Always preserve scroll position

	statusBar := common.StatusBar(m.appTheme, m.windowWidth, m.modeName(), m.modeColor(), scrollingInfo(m.getScrollingPercent(), m.getVisibleContentPercent()), "", common.ColorModeNeutral)
	var actionBar string
	switch m.operationMode {
	case common.ModeIdle:
		actionBar = common.ActionBar(m.windowWidth, m.commandManager.GenerateContextHelp())
	case common.ModeSendSignal:
		actionBar = common.ActionBar(m.windowWidth, m.commandManager.GenerateContextHelp())
	default:
		actionBar = ""
	}

	statusBarHeight := lipgloss.Height(statusBar)
	actionBarHeight := lipgloss.Height(actionBar)

	// Viewport Height Calculation
	//
	// Viewport height is affected by:
	// - hydration notification
	// - error panel
	// - action bar
	// - status bar
	// Viewport height should be filling the remaining height (flexible) upon other elements' changes
	// Because these appear/disappear asynchronously, scroll position may jump.
	screenState := hydration.DeriveScreenPhase(m.hydrationCoordinator)
	switch screenState {
	case hydration.PhaseInit:
		m.viewportModel = viewport.New(viewport.WithWidth(m.windowWidth), viewport.WithHeight(m.windowHeight-actionBarHeight-statusBarHeight))
	case hydration.PhaseHydrationFinished:
		m.viewportModel.SetWidth(m.windowWidth)
		if m.hydrationCoordinator.ErrorsExist() && m.showErrorsPanel {
			errorPanel := common.ErrorPanel(m.appTheme, m.windowWidth, m.hydrationCoordinator.GetErrorsAsString())
			errorsPanelHeight := lipgloss.Height(errorPanel)
			m.viewportModel.SetHeight(m.windowHeight - errorsPanelHeight - statusBarHeight - actionBarHeight)
		} else {
			m.viewportModel.SetHeight(m.windowHeight - statusBarHeight - actionBarHeight)
		}
	default:
		m.viewportModel.SetWidth(m.windowWidth)
		m.viewportModel.SetHeight(m.windowHeight - statusBarHeight - actionBarHeight)
	}

	m.viewportModel.SetYOffset(savedY) // Restore scroll position
}

func (m *Model) getVisibleContentPercent() float64 {
	totalLines := m.viewportModel.TotalLineCount()
	if totalLines == 0 {
		return 0
	}
	return min(100, float64(m.viewportModel.Height())/float64(totalLines)*100)
}

func (m *Model) getScrollingPercent() float64 {
	return m.viewportModel.ScrollPercent() * 100
}

func (m *Model) modeName() string {
	if m.operationMode == common.ModeSendSignal {
		return "Send Signal"
	}
	return "Process Detail"
}

func (m *Model) modeColor() common.ColorMode {
	if m.operationMode == common.ModeSendSignal {
		return common.ColorModeSpecial
	} else {
		return common.ColorModeNeutral
	}
}

func (m *Model) resetAllData() {
	m.hydrationCoordinator.ResetAllFields()
	m.ProcessName = ""
	m.PID = -1
	m.staticIdHydration = StaticIdData{}
	m.resourceHydration = ResourceData{}
	m.userHydration = UserData{}
	m.socketsHydration = SocketsData{}
	m.viewportModel.SetContent("")
}

func (m *Model) renderContent() string {
	ui := processDetailSection(
		m.operationMode == common.ModeIdle,
		m.appTheme,
		m.windowWidth,
		m.ProcessName,
		m.PID,
		m.staticIdHydration.ExecPath,
		m.staticIdHydration.ParentName,
		m.staticIdHydration.PPID,
		m.staticIdHydration.Command,
		m.socketsHydration.Sockets,
		m.userHydration.UserUID,
		m.userHydration.UserName,
		m.userHydration.UserPrivileged,
		int(m.resourceHydration.RSSByte),
		int(m.resourceHydration.VSZByte),
		m.resourceHydration.StartTime,
		m.resourceHydration.ElapsedTime,
		m.resourceHydration.UTime,
		m.resourceHydration.STime,
	)
	trimmed := strings.TrimSpace(ui)
	return trimmed
}

func (m Model) handleBack() (Model, tea.Cmd) {
	screenState := hydration.DeriveScreenPhase(m.hydrationCoordinator)
	if screenState == hydration.PhaseHydrationsInProgress ||
		screenState == hydration.PhaseInit {
		m.cancel()

	}
	return m, func() tea.Msg {
		m.resetAllData()
		return message.GoBack{}
	}
}

func (m Model) handleCloseSendSignal() (Model, tea.Cmd) {
	return m, func() tea.Msg {
		return common.CloseSendSignalModalMsg{}
	}
}

func (m Model) handleQuit() (Model, tea.Cmd) {
	screenState := hydration.DeriveScreenPhase(m.hydrationCoordinator)
	if m.operationMode == common.ModeSendSignal {
		return m, func() tea.Msg {
			return common.CloseSendSignalModalMsg{}
		}
	} else {
		if screenState == hydration.PhaseHydrationsInProgress ||
			screenState == hydration.PhaseInit {
			m.cancel()
		}
		return m, tea.Quit
	}
}

func (m Model) handleOpenSendSignal() (Model, tea.Cmd) {

	// User can send signal as long as the PID is retrived
	// which is already have passed by process list screen (not from hydrating)
	if m.operationMode == common.ModeIdle && hydration.DeriveScreenPhase(m.hydrationCoordinator) != hydration.PhaseInit {
		return m, func() tea.Msg {
			return common.SendSignalMsg{
				ProcessPID:  m.PID,
				ProcessName: m.ProcessName,
			}
		}
	} else {
		return m, func() tea.Msg {
			return nil
		}
	}
}

func (m Model) handleNotificationDismiss() (Model, tea.Cmd) {
	if m.operationMode == common.ModeSendSignal {
		return m, func() tea.Msg {
			return nil
		}
	}
	switch hydration.DeriveScreenPhase(m.hydrationCoordinator) {
	case hydration.PhaseHydrationFinished:
		if m.hydrationCoordinator.ErrorsExist() {
			return m, func() tea.Msg {
				return common.DismissnotificationMsg{}
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

func (m Model) handleErrorRetry() (Model, tea.Cmd) {
	if m.operationMode == common.ModeSendSignal {
		return m, func() tea.Msg {
			return nil // when mode is send signal, user should not have access to retry error
		}
	}
	switch hydration.DeriveScreenPhase(m.hydrationCoordinator) {
	case hydration.PhaseHydrationFinished:
		if m.hydrationCoordinator.ErrorsExist() {
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

func registerContextualCommands(commandManager *command.Manager) error {
	err := commandManager.RegisterContextCommand(command.ContextProcessDetailScreen, command.KeyUp, command.CommandScroll)
	if err != nil {
		return err
	}
	err = commandManager.RegisterContextCommand(command.ContextProcessDetailScreen, command.KeyDown, command.CommandScroll)
	if err != nil {
		return err
	}
	err = commandManager.RegisterContextCommand(command.ContextProcessDetailScreen, command.KeyS, command.CommandSendSignal)
	if err != nil {
		return err
	}
	err = commandManager.RegisterContextCommand(command.ContextProcessDetailScreen, command.KeyEsc, command.CommandBack)
	if err != nil {
		return err
	}

	err = commandManager.RegisterContextCommand(command.ContextOperableHydrationError, command.KeyR, command.CommandRetry)
	if err != nil {
		return err
	}
	err = commandManager.RegisterContextCommand(command.ContextOperableHydrationError, command.KeyDel, command.CommandDismiss)
	if err != nil {
		return err
	}
	err = commandManager.RegisterContextCommand(command.ContextOperableHydrationError, command.KeyUp, command.CommandScroll)
	if err != nil {
		return err
	}
	err = commandManager.RegisterContextCommand(command.ContextOperableHydrationError, command.KeyDown, command.CommandScroll)
	if err != nil {
		return err
	}
	err = commandManager.RegisterContextCommand(command.ContextOperableHydrationError, command.KeyS, command.CommandSendSignal)
	if err != nil {
		return err
	}
	err = commandManager.RegisterContextCommand(command.ContextOperableHydrationError, command.KeyEsc, command.CommandBack)
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

func (m *Model) setCurrentCommandContext() error {
	var err error
	if m.operationMode == common.ModeSendSignal {
		err = m.commandManager.SetContext(command.ContextSendSignal)
	} else {
		hydrationSummary := m.hydrationCoordinator.ComputeHydrationSummary()
		switch hydration.DeriveScreenPhase(m.hydrationCoordinator) {
		case hydration.PhaseHydrationFinished:
			if hydrationSummary.MandatorySatisfied && m.showErrorsPanel {
				err = m.commandManager.SetContext(command.ContextOperableHydrationError)
			} else if hydrationSummary.MandatorySatisfied && !m.showErrorsPanel {
				err = m.commandManager.SetContext(command.ContextProcessDetailScreen)
			} else {
				err = m.commandManager.SetContext(command.ContextInoperableHydrationError)
			}
		default:
			err = m.commandManager.SetContext(command.ContextHydrating)
		}
	}
	return err
}

func (m *Model) resetContext() {
	ctx, cancel := context.WithCancel(context.Background())
	m.ctx = ctx
	m.cancel = cancel
}

func (m *Model) collectRetryCommands() []tea.Cmd {
	commands := []tea.Cmd{}
	retryMap := map[string]tea.Cmd{
		FieldStaticId: HydrateStaticIds(m.ctx, m.PID, m.processService),
		FieldResource: HydrateResource(m.ctx, m.PID, m.processService),
		FieldUser:     HydrateUser(m.ctx, m.PID, m.processService),
		FieldSockets:  HydrateSockets(m.ctx, m.PID, m.socketService),
	}

	for _, fieldName := range m.hydrationCoordinator.GetFieldsForRetry() {
		m.hydrationCoordinator.HydrateField(fieldName)
		if cmd, ok := retryMap[fieldName]; ok {
			commands = append(commands, cmd)
		}
	}

	return commands
}
