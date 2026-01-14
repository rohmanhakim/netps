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
	"log"
	"netps/internal/process"
	"netps/internal/procfs"
	"netps/internal/socket"
	"netps/internal/sysconf"
	"netps/internal/ui/common"
	"netps/internal/ui/common/command"
	"netps/internal/ui/common/lifecycle"
	"netps/internal/ui/common/sendsignal"
	"netps/internal/ui/message"

	"strings"

	"charm.land/bubbles/v2/viewport"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
)

type Model struct {
	PID         int
	ProcessName string

	staticIdHydration StaticIdHydrationData
	resourceHydration ResourceHydrationData
	userHydration     UserHydrationData
	socketsHydration  SocketsHydrationData

	viewportModel viewport.Model

	sendSignalModalModel sendsignal.Model

	processService *process.Service
	socketService  *socket.Service

	errorsToRetry tea.Cmd

	ctx                       context.Context
	cancel                    context.CancelFunc
	operationMode             lifecycle.Mode
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

	return Model{
		staticIdHydration:    StaticIdHydrationData{},
		resourceHydration:    ResourceHydrationData{},
		userHydration:        UserHydrationData{},
		socketsHydration:     SocketsHydrationData{},
		processService:       processService,
		socketService:        socketService,
		ctx:                  ctx,
		cancel:               cancel,
		appTheme:             theme,
		operationMode:        lifecycle.ModeIdle,
		commandManager:       commandManager,
		sendSignalModalModel: sendSignal,
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
	if _, isSideEffect := msg.(lifecycle.SideEffectMsg); isSideEffect {
		if lifecycle.ShouldCancelSideEffects(m.ctx) {
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
		m.operationMode = lifecycle.ModeIdle
		m.windowWidth = msg.width
		m.windowHeight = msg.height
		m.sendSignalModalModel.Initialize()
		m.setAllHydrationState(lifecycle.StateHydrating)
	case lifecycle.RetryMsg:
		if m.operationMode == lifecycle.ModeSendSignal {
			m.operationMode = lifecycle.ModeIdle
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
		if msg.Err == nil && m.staticIdStatusWouldChange(lifecycle.StateSuccess, msg.Err) {
			m.staticIdHydration.state = lifecycle.StateSuccess
			m.staticIdHydration.err = nil
			m.staticIdHydration.ExecPath = msg.ExecPath
			m.staticIdHydration.Command = msg.Command
			m.staticIdHydration.PPID = msg.PPID
			m.staticIdHydration.ParentName = msg.ParentName
			dataChanged = true
		} else if m.staticIdStatusWouldChange(lifecycle.StateError, msg.Err) {
			m.staticIdHydration.state = lifecycle.StateError
			m.staticIdHydration.err = msg.Err
			dataChanged = true
		}
	case resourceHydratedMsg:
		if msg.Err == nil && m.resourceStatusWouldChange(lifecycle.StateSuccess, msg.Err) {
			m.resourceHydration.state = lifecycle.StateSuccess
			m.resourceHydration.err = nil
			m.resourceHydration.RSSByte = msg.RSSByte
			m.resourceHydration.StartTime = msg.StartTime
			m.resourceHydration.ElapsedTime = msg.ElapsedTime
			m.resourceHydration.VSZByte = msg.VSZByte
			m.resourceHydration.UTime = msg.UTime
			m.resourceHydration.STime = msg.STime
			dataChanged = true
		} else if m.resourceStatusWouldChange(lifecycle.StateError, msg.Err) {
			m.resourceHydration.state = lifecycle.StateError
			m.resourceHydration.err = msg.Err
			dataChanged = true
		}
	case userHydratedMsg:
		if msg.Err == nil && m.userStatusWouldChange(lifecycle.StateSuccess, msg.Err) {
			m.userHydration.state = lifecycle.StateSuccess
			m.userHydration.err = nil
			m.userHydration.UserUID = msg.UserUID
			m.userHydration.UserName = msg.UserName
			m.userHydration.UserPrivileged = msg.UserPrivileged
			dataChanged = true
		} else if m.userStatusWouldChange(lifecycle.StateError, msg.Err) {
			m.userHydration.state = lifecycle.StateError
			m.userHydration.err = msg.Err
			dataChanged = true
		}
	case socketsHydratedMsg:
		if msg.Err == nil && m.socketsStatusWouldChange(lifecycle.StateSuccess, msg.Err) {
			m.socketsHydration.state = lifecycle.StateSuccess
			m.socketsHydration.err = nil
			m.socketsHydration.Sockets = msg.Sockets
			dataChanged = true
		} else if m.socketsStatusWouldChange(lifecycle.StateError, msg.Err) {
			m.socketsHydration.state = lifecycle.StateError
			m.socketsHydration.err = msg.Err
			dataChanged = true
		}
	case lifecycle.SendSignalMsg:
		m.operationMode = lifecycle.ModeSendSignal
		m.sendSignalModalModel.SetProcessInfo(msg.ProcessPID, msg.ProcessName)
		err := m.setCurrentCommandContext()
		if err != nil {
			log.Fatalf("[lifecycle.SendSignalMsg] Process Detail Model Error: %v", err)
		}
		viewportContentColorChanged = true // opening send signal modal changed the viewport's content color to dim which required to rerender the viewport
	case lifecycle.CloseSendSignalModalMsg:
		m.operationMode = lifecycle.ModeIdle
		viewportContentColorChanged = true // closing send signal modal changed the viewport's content color to normal which required to rerender the viewport
	case lifecycle.DismissnotificationMsg:
		// Dismissing errors hides the panel but does not change data completeness.
		// dismissal is not errors resolution
		// status bar remains “Data Partial” intentionally so that the user be aware
		// TO-DO: Implement retry on-demand when the data is partial, even after dismiss
		m.resetAllErrors()
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

	if m.operationMode == lifecycle.ModeSendSignal {
		m.sendSignalModalModel, cmd = m.sendSignalModalModel.Update(msg)
	} else {
		m.viewportModel, cmd = m.viewportModel.Update(msg)
	}
	cmds = append(cmds, cmd)
	return m, tea.Batch(cmds...)
}

func (m Model) baseUIRenderableState() bool {
	return m.computeScreenState() != lifecycle.StateInit
}

func (m Model) View() tea.View {
	screenState := m.computeScreenState()
	var v tea.View
	v.AltScreen = true // Bubble Tea v2 forces you to set the AltScren option on every Model's View() function
	switch screenState {
	case lifecycle.StateInit:
		v.SetContent("\n  Initializing...")
	default:
		layers := []*lipgloss.Layer{}
		helpItems := m.commandManager.GenerateContextHelp()

		if m.operationMode == lifecycle.ModeSendSignal {
			signalList := m.sendSignalModalModel
			modalLayer := lipgloss.NewLayer(signalList.View().Content).
				X((m.windowWidth / 2) - (signalList.ModalWidth() / 2)).
				Y((m.windowHeight / 2) - (signalList.ModalHeight() / 2)).
				Z(1)
			layers = append(layers, modalLayer)
		}

		ui := renderBaseLayer(
			m.appTheme,
			m.viewportModel.View(),
			m.windowWidth,
			m.modeName(),
			m.modeColor(),
			scrollingInfo(m.getScrollingPercent(), m.getVisibleContentPercent()),
			helpItems,
			m.getErrorsAsString(),
			screenState,
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
	screenState lifecycle.ScreenState,
	zIndex int,
) *lipgloss.Layer {
	components := []string{}
	components = append(components, content)

	actionBar := common.ActionBar(width, helpItems)

	var screenStateInfoLabel string
	var statusBar string

	switch screenState {
	case lifecycle.StateHydrationsInProgress, lifecycle.StateOneHydrationFinished:
		statusBar = common.NotificationBar(theme, common.ColorModeNeutral, width, "Getting Data...")
	case lifecycle.StateHydrationsFinishedErrorsExist:
		// User still be shown that the data is partial in status bar, even after dismissal of error notification so that he/she awares
		statusBar = common.StatusBar(theme, width, modeName, colorMode, lipgloss.JoinHorizontal(lipgloss.Top, statusBarInfo, screenStateInfoLabel), "Data Partial", common.ColorModeWarning)
		if len(errors) > 0 {
			errorPanel := common.ErrorPanel(theme, width, errors)
			components = append(components, errorPanel)
		}
	case lifecycle.StateHydrationsFinishedErrorDismissed:
		statusBar = common.StatusBar(theme, width, modeName, colorMode, lipgloss.JoinHorizontal(lipgloss.Top, statusBarInfo, screenStateInfoLabel), "Data Partial", common.ColorModeWarning)
	case lifecycle.StateHydrationsFinishedAllOK:
		statusBar = common.StatusBar(theme, width, modeName, colorMode, lipgloss.JoinHorizontal(lipgloss.Top, statusBarInfo, screenStateInfoLabel), "Data OK", common.ColorModeSuccess)
	default:
		screenStateInfoLabel = ""
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

	errorPanel := common.ErrorPanel(m.appTheme, m.windowWidth, m.getErrorsAsString())
	statusBar := common.StatusBar(m.appTheme, m.windowWidth, m.modeName(), m.modeColor(), scrollingInfo(m.getScrollingPercent(), m.getVisibleContentPercent()), "", common.ColorModeNeutral)
	var actionBar string
	switch m.operationMode {
	case lifecycle.ModeIdle:
		actionBar = common.ActionBar(m.windowWidth, m.commandManager.GenerateContextHelp())
	case lifecycle.ModeSendSignal:
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
	screenState := m.computeScreenState()
	switch screenState {
	case lifecycle.StateInit:
		m.viewportModel = viewport.New(viewport.WithWidth(m.windowWidth), viewport.WithHeight(m.windowHeight-actionBarHeight-statusBarHeight))
	case lifecycle.StateHydrationsFinishedErrorsExist, lifecycle.StateHydrationsFinishedErrorDismissed:
		m.viewportModel.SetWidth(m.windowWidth)
		errorsPanelHeight := lipgloss.Height(errorPanel)
		if len(m.getErrorsAsString()) > 0 {
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
	if m.operationMode == lifecycle.ModeSendSignal {
		return "Send Signal"
	}
	return "Process Detail"
}

func (m *Model) modeColor() common.ColorMode {
	if m.operationMode == lifecycle.ModeSendSignal {
		return common.ColorModeSpecial
	} else {
		return common.ColorModeNeutral
	}
}

func (m *Model) resetAllData() {
	m.resetAllHydrationStatus()
	m.resetAllErrors()
	m.ProcessName = ""
	m.PID = -1
	m.staticIdHydration = StaticIdHydrationData{}
	m.resourceHydration = ResourceHydrationData{}
	m.userHydration = UserHydrationData{}
	m.socketsHydration = SocketsHydrationData{}
	m.viewportModel.SetContent("")
}

func (m *Model) resetAllErrors() {
	m.resourceHydration.err = nil
	m.userHydration.err = nil
	m.staticIdHydration.err = nil
	m.socketsHydration.err = nil
}

func (m *Model) resetAllHydrationStatus() {
	m.staticIdHydration.state = lifecycle.StateNotAsked
	m.resourceHydration.state = lifecycle.StateNotAsked
	m.userHydration.state = lifecycle.StateNotAsked
	m.socketsHydration.state = lifecycle.StateNotAsked
}

func (m *Model) renderContent() string {
	ui := processDetailSection(
		m.operationMode == lifecycle.ModeIdle,
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
	screenState := m.computeScreenState()
	if screenState == lifecycle.StateHydrationsInProgress ||
		screenState == lifecycle.StateInit ||
		screenState == lifecycle.StateOneHydrationFinished {
		m.cancel()

	}
	return m, func() tea.Msg {
		m.resetAllData()
		return message.GoBack{}
	}
}

func (m Model) handleCloseSendSignal() (Model, tea.Cmd) {
	return m, func() tea.Msg {
		return lifecycle.CloseSendSignalModalMsg{}
	}
}

func (m Model) handleQuit() (Model, tea.Cmd) {
	screenState := m.computeScreenState()
	if m.operationMode == lifecycle.ModeSendSignal {
		return m, func() tea.Msg {
			return lifecycle.CloseSendSignalModalMsg{}
		}
	} else {
		if screenState == lifecycle.StateHydrationsInProgress ||
			screenState == lifecycle.StateInit ||
			screenState == lifecycle.StateOneHydrationFinished {
			m.cancel()
		}
		return m, tea.Quit
	}
}

func (m Model) handleOpenSendSignal() (Model, tea.Cmd) {

	// User can send signal as long as the PID is retrived
	// which is already have passed by process list screen (not from hydrating)
	if m.operationMode == lifecycle.ModeIdle && m.computeScreenState() != lifecycle.StateInit {
		return m, func() tea.Msg {
			return lifecycle.SendSignalMsg{
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
	switch m.computeScreenState() {
	case lifecycle.StateHydrationsFinishedErrorsExist:
		if m.operationMode == lifecycle.ModeSendSignal {
			return m, func() tea.Msg {
				return nil // do nothing for now; next will implement proper signal sending logic
			}
		} else {
			return m, func() tea.Msg {
				return lifecycle.DismissnotificationMsg{}
			}
		}
	default:
		return m, func() tea.Msg {
			return nil
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

func (m *Model) getErrorsAsString() []string {
	errorStrings := []string{}
	if m.staticIdHydration.state == lifecycle.StateError && m.staticIdHydration.err != nil {
		errorStrings = append(errorStrings, "[rare][retryable] "+m.staticIdHydration.err.Error())
	}
	if m.resourceHydration.state == lifecycle.StateError && m.resourceHydration.err != nil {
		errorStrings = append(errorStrings, "[common][retryable] "+m.resourceHydration.err.Error())
	}
	if m.userHydration.state == lifecycle.StateError && m.userHydration.err != nil {
		errorStrings = append(errorStrings, "[common][permanent] "+m.userHydration.err.Error())
	}
	if m.socketsHydration.state == lifecycle.StateError && m.socketsHydration.err != nil {
		errorStrings = append(errorStrings, "[common][retryable] "+m.socketsHydration.err.Error())
	}
	return errorStrings
}

func (m *Model) hydrationErrorsExist() bool {
	return m.staticIdHydration.err != nil ||
		m.resourceHydration.err != nil ||
		m.userHydration.err != nil ||
		m.socketsHydration.err != nil
}

func (m *Model) oneHydrationFinished() bool {
	staticIdHydrationFinished := m.staticIdHydration.state == lifecycle.StateSuccess || m.staticIdHydration.state == lifecycle.StateError
	resourceHydrationFinished := m.resourceHydration.state == lifecycle.StateSuccess || m.resourceHydration.state == lifecycle.StateError
	userHydrationFinished := m.userHydration.state == lifecycle.StateSuccess || m.userHydration.state == lifecycle.StateError
	socketsHydrationFinished := m.socketsHydration.state == lifecycle.StateSuccess || m.socketsHydration.state == lifecycle.StateError

	allHydrationFinished := staticIdHydrationFinished ||
		resourceHydrationFinished ||
		userHydrationFinished ||
		socketsHydrationFinished
	return allHydrationFinished
}

func (m *Model) allHydrationFinished() bool {
	staticIdHydrationFinished := m.staticIdHydration.state == lifecycle.StateSuccess || m.staticIdHydration.state == lifecycle.StateError
	resourceHydrationFinished := m.resourceHydration.state == lifecycle.StateSuccess || m.resourceHydration.state == lifecycle.StateError
	userHydrationFinished := m.userHydration.state == lifecycle.StateSuccess || m.userHydration.state == lifecycle.StateError
	socketsHydrationFinished := m.socketsHydration.state == lifecycle.StateSuccess || m.socketsHydration.state == lifecycle.StateError

	allHydrationFinished := staticIdHydrationFinished &&
		resourceHydrationFinished &&
		userHydrationFinished &&
		socketsHydrationFinished
	return allHydrationFinished
}

func (m *Model) allHydrationOK() bool {
	allHydrationOK := m.staticIdHydration.state == lifecycle.StateSuccess &&
		m.resourceHydration.state == lifecycle.StateSuccess &&
		m.userHydration.state == lifecycle.StateSuccess &&
		m.socketsHydration.state == lifecycle.StateSuccess
	return allHydrationOK
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
		} else if m.hydrationErrorsExist() {
			return lifecycle.StateHydrationsFinishedErrorsExist
		} else {
			return lifecycle.StateHydrationsFinishedErrorDismissed
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

func (m *Model) staticIdStatusWouldChange(newState lifecycle.HydrationState, err error) bool {
	oldState := m.staticIdHydration.state
	oldError := m.staticIdHydration.err
	return oldState != newState || oldError != err
}

func (m *Model) resourceStatusWouldChange(newState lifecycle.HydrationState, err error) bool {
	oldState := m.resourceHydration.state
	oldError := m.resourceHydration.err
	return oldState != newState || oldError != err
}

func (m *Model) userStatusWouldChange(newState lifecycle.HydrationState, err error) bool {
	oldState := m.userHydration.state
	oldError := m.userHydration.err
	return oldState != newState || oldError != err
}

func (m *Model) socketsStatusWouldChange(newState lifecycle.HydrationState, err error) bool {
	oldState := m.socketsHydration.state
	oldError := m.socketsHydration.err
	return oldState != newState || oldError != err
}

func (m *Model) setAllHydrationState(state lifecycle.HydrationState) {
	m.staticIdHydration.state = state
	m.resourceHydration.state = state
	m.userHydration.state = state
	m.socketsHydration.state = state
}

func (m *Model) resetContext() {
	ctx, cancel := context.WithCancel(context.Background())
	m.ctx = ctx
	m.cancel = cancel
}

func (m *Model) collectRetryCommands() []tea.Cmd {
	commands := []tea.Cmd{}

	if m.shouldRetry(m.staticIdHydration.err) {
		m.staticIdHydration.err = nil
		m.staticIdHydration.state = lifecycle.StateHydrating
		commands = append(commands, HydrateStaticIds(m.ctx, m.PID, m.processService))
	}

	if m.shouldRetry(m.resourceHydration.err) {
		m.resourceHydration.err = nil
		m.resourceHydration.state = lifecycle.StateHydrating
		commands = append(commands, HydrateResource(m.ctx, m.PID, m.processService))
	}

	if m.shouldRetry(m.userHydration.err) {
		m.userHydration.err = nil
		m.userHydration.state = lifecycle.StateHydrating
		commands = append(commands, HydrateUser(m.ctx, m.PID, m.processService))
	}

	if m.shouldRetry(m.socketsHydration.err) {
		m.socketsHydration.err = nil
		m.socketsHydration.state = lifecycle.StateHydrating
		commands = append(commands, HydrateSockets(m.ctx, m.PID, m.socketService))
	}

	return commands
}

// determines if retry is needed
func (m *Model) shouldRetry(err error) bool {
	return err != nil
}

func (m *Model) allHydrating() bool {
	return m.staticIdHydration.state == lifecycle.StateHydrating ||
		m.resourceHydration.state == lifecycle.StateHydrating ||
		m.userHydration.state == lifecycle.StateHydrating ||
		m.socketsHydration.state == lifecycle.StateHydrating
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
	if m.operationMode == lifecycle.ModeSendSignal {
		err = m.commandManager.SetContext(command.ContextSendSignal)
	} else {
		switch m.computeScreenState() {
		case lifecycle.StateHydrationsFinishedAllOK, lifecycle.StateHydrationsFinishedErrorDismissed:
			err = m.commandManager.SetContext(command.ContextProcessDetailScreen)
		case lifecycle.StateHydrationsFinishedErrorsExist:
			err = m.commandManager.SetContext(command.ContextOperableHydrationError)
		default:
			err = m.commandManager.SetContext(command.ContextHydrating)
		}
	}
	return err
}
