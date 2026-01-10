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
	"netps/internal/process"
	"netps/internal/procfs"
	"netps/internal/socket"
	"netps/internal/sysconf"
	"netps/internal/ui/common"
	"netps/internal/ui/common/sendsignal"
	"netps/internal/ui/message"

	"strings"

	"charm.land/bubbles/v2/viewport"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
)

type ScreenState int
type Mode int
type KeyPress string

/*
 * Screen States
 *
 * During data hydration,the screen is streaming, not strictly phased.
 * User can still interact with the process (e.g sending signal)
 * The reason is so that a critical operation may be in an urgency
 * Errors should not prevent users to do critical operations
 * Screen states describe data completeness, not UI lock-in.
 */
const (
	StateInit ScreenState = iota
	StateHydrationsInProgress
	StateOneHydrationFinished
	StateHydrationsFinishedErrorsExist
	StateHydrationsFinishedErrorDismissed
	StateHydrationsFinishedAllOK
	StateRetryHydrations
)

const (
	ModeIdle Mode = iota
	ModeSendSignal
)

const (
	KeyEnter KeyPress = "enter"
	KeyQ     KeyPress = "q"
	KeyR     KeyPress = "r"
	KeyEsc   KeyPress = "esc"
	KeyDel   KeyPress = "delete"
	KeyS     KeyPress = "s"
	KeyCtrlC KeyPress = "ctrl+c"
)

type Model struct {
	PID         int
	ProcessName string

	staticIdHydration StaticIdHydrationData
	resourceHydration ResourceHydrationData
	userHydration     UserHydrationData
	socketsHydration  SocketsHydrationData

	windowWidth   int
	windowHeight  int
	viewportModel viewport.Model

	operationMode Mode

	sendSignalModalModel sendsignal.Model

	notificationDismissKey KeyPress
	errorRetryKey          KeyPress

	appTheme       common.Theme
	ctx            context.Context
	cancel         context.CancelFunc
	processService *process.Service
	socketService  *socket.Service

	errorsToRetry tea.Cmd
}

type styleFunc func(string) string

func New(theme common.Theme) Model {
	sendSignal := sendsignal.New()
	ctx, cancel := context.WithCancel(context.Background())
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
		sendSignalModalModel:   sendSignal,
		appTheme:               theme,
		staticIdHydration:      StaticIdHydrationData{},
		resourceHydration:      ResourceHydrationData{},
		userHydration:          UserHydrationData{},
		socketsHydration:       SocketsHydrationData{},
		ctx:                    ctx,
		cancel:                 cancel,
		notificationDismissKey: KeyDel,
		errorRetryKey:          KeyR,
		processService:         processService,
		socketService:          socketService,
	}
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
	if _, isSideEffect := msg.(sideEffectMsg); isSideEffect {
		if m.shouldCancelSideEffects() {
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
		m.operationMode = ModeIdle
		m.windowWidth = msg.width
		m.windowHeight = msg.height
		m.sendSignalModalModel.Initialize()
		m.setAllHydrationState(StateHydrating)
	case retryMsg:
		if m.operationMode == ModeSendSignal {
			m.operationMode = ModeIdle
		}
		m.resetContext()
		commands := m.collectRetryCommands()
		m.adjustViewportSize() // have to adjust viewport here because this retry returns immediately skipping main viewport updating line
		return m, tea.Batch(commands...)
	case staticIdHydratedMsg:
		if msg.Err == nil && m.staticIdStatusWouldChange(StateSuccess, msg.Err) {
			m.staticIdHydration.state = StateSuccess
			m.staticIdHydration.err = nil
			m.staticIdHydration.ExecPath = msg.ExecPath
			m.staticIdHydration.Command = msg.Command
			m.staticIdHydration.PPID = msg.PPID
			m.staticIdHydration.ParentName = msg.ParentName
			dataChanged = true
		} else if m.staticIdStatusWouldChange(StateError, msg.Err) {
			m.staticIdHydration.state = StateError
			m.staticIdHydration.err = msg.Err
			dataChanged = true
		}
	case resourceHydratedMsg:
		if msg.Err == nil && m.resourceStatusWouldChange(StateSuccess, msg.Err) {
			m.resourceHydration.state = StateSuccess
			m.resourceHydration.err = nil
			m.resourceHydration.RSSByte = msg.RSSByte
			m.resourceHydration.StartTime = msg.StartTime
			m.resourceHydration.ElapsedTime = msg.ElapsedTime
			m.resourceHydration.VSZByte = msg.VSZByte
			m.resourceHydration.UTime = msg.UTime
			m.resourceHydration.STime = msg.STime
			dataChanged = true
		} else if m.resourceStatusWouldChange(StateError, msg.Err) {
			m.resourceHydration.state = StateError
			m.resourceHydration.err = msg.Err
			dataChanged = true
		}
	case userHydratedMsg:
		if msg.Err == nil && m.userStatusWouldChange(StateSuccess, msg.Err) {
			m.userHydration.state = StateSuccess
			m.userHydration.err = nil
			m.userHydration.UserUID = msg.UserUID
			m.userHydration.UserName = msg.UserName
			m.userHydration.UserPrivileged = msg.UserPrivileged
			dataChanged = true
		} else if m.userStatusWouldChange(StateError, msg.Err) {
			m.userHydration.state = StateError
			m.userHydration.err = msg.Err
			dataChanged = true
		}
	case socketsHydratedMsg:
		if msg.Err == nil && m.socketsStatusWouldChange(StateSuccess, msg.Err) {
			m.socketsHydration.state = StateSuccess
			m.socketsHydration.err = nil
			m.socketsHydration.Sockets = msg.Sockets
			dataChanged = true
		} else if m.socketsStatusWouldChange(StateError, msg.Err) {
			m.socketsHydration.state = StateError
			m.socketsHydration.err = msg.Err
			dataChanged = true
		}
	case sendSignalMsg:
		m.operationMode = ModeSendSignal
		viewportContentColorChanged = true // opening send signal modal changed the viewport's content color to dim which required to rerender the viewport
	case closeSendSignalModalMsg:
		m.operationMode = ModeIdle
		viewportContentColorChanged = true // closing send signal modal changed the viewport's content color to normal which required to rerender the viewport
	case dismissnotificationMsg:
		// Dismissing errors hides the panel but does not change data completeness.
		// dismissal is not errors resolution
		// status bar remains “Data Partial” intentionally so that the user be aware
		// TO-DO: Implement retry on-demand when the data is partial, even after dismiss
		m.resetAllErrors()
	case tea.KeyMsg:
		switch msg.String() {
		case string(KeyEsc):
			return m.handleEsc()
		case string(KeyS):
			return m.handleS()
		case string(KeyQ), string(KeyCtrlC):
			return m.handleQ()
		case string(KeyEnter):
			return m.handleErrorRetryKey()
		case string(m.errorRetryKey):
			return m.handleErrorRetryKey()
		case string(m.notificationDismissKey):
			return m.handleNotificationDismissKey()
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

	if m.operationMode == ModeSendSignal {
		m.sendSignalModalModel, cmd = m.sendSignalModalModel.Update(msg)
		return m, cmd
	} else {
		m.viewportModel, cmd = m.viewportModel.Update(msg)
		cmds = append(cmds, cmd)
		return m, tea.Batch(cmds...)
	}
}

func (m Model) baseUIRenderableState() bool {
	return m.computeScreenState() != StateInit
}

func (m Model) View() tea.View {
	screenState := m.computeScreenState()
	var v tea.View
	v.AltScreen = true // Bubble Tea v2 forces you to set the AltScren option on every Model's View() function
	switch screenState {
	case StateInit:
		v.SetContent("\n  Initializing...")
	default:
		layers := []*lipgloss.Layer{}
		actionItems := renderIdleHelp()

		if m.operationMode == ModeSendSignal {
			signalList := m.sendSignalModalModel
			signalListWidth := lipgloss.Width(signalList.Modal)
			signalListHeight := lipgloss.Height(signalList.Modal)
			modalLayer := lipgloss.NewLayer(signalList.View().Content).
				X((m.windowWidth / 2) - (signalListWidth / 2)).
				Y((m.windowHeight / 2) - (signalListHeight / 2)).
				Z(1)
			actionItems = signalList.SendSignalHelpItems
			layers = append(layers, modalLayer)
		}

		ui := renderBaseLayer(
			m.appTheme,
			m.viewportModel.View(),
			m.windowWidth,
			m.modeName(),
			m.modeColor(),
			scrollingInfo(m.getScrollingPercent(), m.getVisibleContentPercent()),
			actionItems,
			m.renderErrorsHelp(),
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
	errorsHelpItems []string,
	errors []string,
	screenState ScreenState,
	zIndex int,
) *lipgloss.Layer {
	components := []string{}
	components = append(components, content)

	actionBar := common.ActionBar(width, helpItems)

	var screenStateInfoLabel string
	var statusBar string

	switch screenState {
	case StateHydrationsInProgress, StateOneHydrationFinished:
		statusBar = common.NotificationBar(theme, common.ColorModeNeutral, width, "Getting Data...")
		actionBar = common.ActionBar(width, helpItems)
	case StateHydrationsFinishedErrorsExist:
		// User still be shown that the data is partial in status bar, even after dismissal of error notification so that he/she awares
		statusBar = common.StatusBar(theme, width, modeName, colorMode, lipgloss.JoinHorizontal(lipgloss.Top, statusBarInfo, screenStateInfoLabel), "Data Partial", common.ColorModeWarning)
		if len(errors) > 0 {
			errorPanel := common.ErrorPanel(theme, width, errors)
			components = append(components, errorPanel)
			actionBar = common.ActionBar(width, errorsHelpItems)
		}
	case StateHydrationsFinishedErrorDismissed:
		statusBar = common.StatusBar(theme, width, modeName, colorMode, lipgloss.JoinHorizontal(lipgloss.Top, statusBarInfo, screenStateInfoLabel), "Data Partial", common.ColorModeWarning)
	case StateHydrationsFinishedAllOK:
		statusBar = common.StatusBar(theme, width, modeName, colorMode, lipgloss.JoinHorizontal(lipgloss.Top, statusBarInfo, screenStateInfoLabel), "Data OK", common.ColorModeSuccess)
		actionBar = common.ActionBar(width, helpItems)
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
	savedY := m.viewportModel.YOffset() // Always preserve

	errorPanel := common.ErrorPanel(m.appTheme, m.windowWidth, m.getErrorsAsString())
	statusBar := common.StatusBar(m.appTheme, m.windowWidth, m.modeName(), m.modeColor(), scrollingInfo(m.getScrollingPercent(), m.getVisibleContentPercent()), "", common.ColorModeNeutral)
	var actionBar string
	switch m.operationMode {
	case ModeIdle:
		actionBar = common.ActionBar(m.windowWidth, renderIdleHelp())
	case ModeSendSignal:
		actionBar = common.ActionBar(m.windowWidth, m.sendSignalModalModel.SendSignalHelpItems)
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
	case StateInit:
		m.viewportModel = viewport.New(viewport.WithWidth(m.windowWidth), viewport.WithHeight(m.windowHeight-actionBarHeight-statusBarHeight))
	case StateHydrationsFinishedErrorsExist, StateHydrationsFinishedErrorDismissed:
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

	m.viewportModel.SetYOffset(savedY) // Restore
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
	if m.operationMode == ModeSendSignal {
		return "Send Signal"
	}
	return "Process Detail"
}

func (m *Model) modeColor() common.ColorMode {
	if m.operationMode == ModeSendSignal {
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
	m.staticIdHydration.state = StateNotAsked
	m.resourceHydration.state = StateNotAsked
	m.userHydration.state = StateNotAsked
	m.socketsHydration.state = StateNotAsked
}

func (m *Model) renderContent() string {
	ui := processDetailSection(
		m.operationMode == ModeIdle,
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

func renderIdleHelp() []string {
	scrollString := "[↑↓] scroll"
	sendSignalCommanString := fmt.Sprintf("[%s] send signal", KeyS)
	backString := fmt.Sprintf("[%s] back", KeyEsc)
	quitString := fmt.Sprintf("[%s] quit", KeyQ)
	idleHelpItems := []string{
		scrollString,
		sendSignalCommanString,
		backString,
		quitString,
	}
	return idleHelpItems
}

func (m *Model) renderErrorsHelp() []string {
	retryString := fmt.Sprintf("[%s] retry", m.errorRetryKey)
	dismissString := fmt.Sprintf("[%s] dismiss", m.notificationDismissKey)
	scrollString := "[↑↓] scroll"
	sendSignalCommanString := fmt.Sprintf("[%s] send signal", KeyS)
	backString := fmt.Sprintf("[%s] back", KeyEsc)
	quitString := fmt.Sprintf("[%s] quit", KeyQ)
	errorsHelpItems := []string{
		retryString,
		dismissString,
		scrollString,
		sendSignalCommanString,
		backString,
		quitString,
	}
	return errorsHelpItems
}

func (m Model) handleEsc() (Model, tea.Cmd) {
	screenState := m.computeScreenState()
	if m.operationMode == ModeSendSignal {
		return m, func() tea.Msg {
			return closeSendSignalModalMsg{}
		}
	} else {
		if screenState == StateHydrationsInProgress || screenState == StateInit || screenState == StateOneHydrationFinished {
			m.cancel()

		}
		return m, func() tea.Msg {
			m.resetAllData()
			return message.GoBack{}
		}
	}
}

func (m Model) handleQ() (Model, tea.Cmd) {
	screenState := m.computeScreenState()
	if m.operationMode == ModeSendSignal {
		return m, func() tea.Msg {
			return closeSendSignalModalMsg{}
		}
	} else {
		if screenState == StateHydrationsInProgress || screenState == StateInit || screenState == StateOneHydrationFinished {
			m.cancel()
		}
		return m, tea.Quit
	}
}

func (m Model) handleS() (Model, tea.Cmd) {

	// User can send signal as long as the PID is retrived
	// which is already have passed by process list screen (not from hydrating)
	if m.operationMode == ModeIdle && m.computeScreenState() != StateInit {
		return m, func() tea.Msg {
			return sendSignalMsg{}
		}
	} else {
		return m, func() tea.Msg {
			return nil
		}
	}
}

func (m Model) handleNotificationDismissKey() (Model, tea.Cmd) {
	switch m.computeScreenState() {
	case StateHydrationsFinishedErrorsExist:
		if m.operationMode == ModeSendSignal {
			return m, func() tea.Msg {
				return nil // do nothing for now; next will implement proper signal sending logic
			}
		} else {
			return m, func() tea.Msg {
				return dismissnotificationMsg{}
			}
		}
	default:
		return m, func() tea.Msg {
			return nil
		}
	}
}

func (m Model) handleErrorRetryKey() (Model, tea.Cmd) {
	switch m.computeScreenState() {
	case StateHydrationsFinishedErrorsExist:
		if m.operationMode == ModeSendSignal {
			return m, func() tea.Msg {
				return nil // when mode is send signal, user should not have access to retry error
			}
		} else if m.hydrationErrorsExist() {
			return m, func() tea.Msg {
				return retryMsg{}
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
	if m.staticIdHydration.state == StateError && m.staticIdHydration.err != nil {
		errorStrings = append(errorStrings, "[rare][retryable] "+m.staticIdHydration.err.Error())
	}
	if m.resourceHydration.state == StateError && m.resourceHydration.err != nil {
		errorStrings = append(errorStrings, "[common][retryable] "+m.resourceHydration.err.Error())
	}
	if m.userHydration.state == StateError && m.userHydration.err != nil {
		errorStrings = append(errorStrings, "[common][permanent] "+m.userHydration.err.Error())
	}
	if m.socketsHydration.state == StateError && m.socketsHydration.err != nil {
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
	staticIdHydrationFinished := m.staticIdHydration.state == StateSuccess || m.staticIdHydration.state == StateError
	resourceHydrationFinished := m.resourceHydration.state == StateSuccess || m.resourceHydration.state == StateError
	userHydrationFinished := m.userHydration.state == StateSuccess || m.userHydration.state == StateError
	socketsHydrationFinished := m.socketsHydration.state == StateSuccess || m.socketsHydration.state == StateError

	allHydrationFinished := staticIdHydrationFinished ||
		resourceHydrationFinished ||
		userHydrationFinished ||
		socketsHydrationFinished
	return allHydrationFinished
}

func (m *Model) allHydrationFinished() bool {
	staticIdHydrationFinished := m.staticIdHydration.state == StateSuccess || m.staticIdHydration.state == StateError
	resourceHydrationFinished := m.resourceHydration.state == StateSuccess || m.resourceHydration.state == StateError
	userHydrationFinished := m.userHydration.state == StateSuccess || m.userHydration.state == StateError
	socketsHydrationFinished := m.socketsHydration.state == StateSuccess || m.socketsHydration.state == StateError

	allHydrationFinished := staticIdHydrationFinished &&
		resourceHydrationFinished &&
		userHydrationFinished &&
		socketsHydrationFinished
	return allHydrationFinished
}

func (m *Model) allHydrationOK() bool {
	allHydrationOK := m.staticIdHydration.state == StateSuccess &&
		m.resourceHydration.state == StateSuccess &&
		m.userHydration.state == StateSuccess &&
		m.socketsHydration.state == StateSuccess
	return allHydrationOK
}

// Calculate the current screen's state
// Central place to derive screen's state
// Pure state rducer so that no multiple state mutations inside this screen
// Multiple subsystems query computeScreenState() in other place is intentional for now
// TO-DO: State caching
func (m *Model) computeScreenState() ScreenState {
	if m.allHydrationFinished() {
		if m.allHydrationOK() {
			return StateHydrationsFinishedAllOK
		} else if m.hydrationErrorsExist() {
			return StateHydrationsFinishedErrorsExist
		} else {
			return StateHydrationsFinishedErrorDismissed
		}
	} else {
		if m.oneHydrationFinished() {
			return StateOneHydrationFinished
		} else if m.allHydrating() {
			return StateHydrationsInProgress
		} else {
			return StateInit
		}
	}
}

func (m *Model) staticIdStatusWouldChange(newState HydrationState, err error) bool {
	oldState := m.staticIdHydration.state
	oldError := m.staticIdHydration.err
	return oldState != newState || oldError != err
}

func (m *Model) resourceStatusWouldChange(newState HydrationState, err error) bool {
	oldState := m.resourceHydration.state
	oldError := m.resourceHydration.err
	return oldState != newState || oldError != err
}

func (m *Model) userStatusWouldChange(newState HydrationState, err error) bool {
	oldState := m.userHydration.state
	oldError := m.userHydration.err
	return oldState != newState || oldError != err
}

func (m *Model) socketsStatusWouldChange(newState HydrationState, err error) bool {
	oldState := m.socketsHydration.state
	oldError := m.socketsHydration.err
	return oldState != newState || oldError != err
}

func (m *Model) setAllHydrationState(state HydrationState) {
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
		m.staticIdHydration.state = StateHydrating
		commands = append(commands, HydrateStaticIds(m.ctx, m.PID, m.processService))
	}

	if m.shouldRetry(m.resourceHydration.err) {
		m.resourceHydration.err = nil
		m.resourceHydration.state = StateHydrating
		commands = append(commands, HydrateResource(m.ctx, m.PID, m.processService))
	}

	if m.shouldRetry(m.userHydration.err) {
		m.userHydration.err = nil
		m.userHydration.state = StateHydrating
		commands = append(commands, HydrateUser(m.ctx, m.PID, m.processService))
	}

	if m.shouldRetry(m.socketsHydration.err) {
		m.socketsHydration.err = nil
		m.socketsHydration.state = StateHydrating
		commands = append(commands, HydrateSockets(m.ctx, m.PID, m.socketService))
	}

	return commands
}

// determines if retry is needed
func (m *Model) shouldRetry(err error) bool {
	return err != nil
}

func (m *Model) allHydrating() bool {
	return m.staticIdHydration.state == StateHydrating ||
		m.resourceHydration.state == StateHydrating ||
		m.userHydration.state == StateHydrating ||
		m.socketsHydration.state == StateHydrating
}

// helper to check if we should cancel *side effects*
func (m *Model) shouldCancelSideEffects() bool {
	return m.ctx.Err() != nil
}
