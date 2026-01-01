package processdetail

import (
	"fmt"
	"netps/internal/socket"
	"netps/internal/ui/message"
	"time"

	"charm.land/bubbles/v2/viewport"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
)

type Model struct {
	PID        int
	Name       string
	ExecPath   string
	Command    string
	PPID       int
	ParentName string

	RSSByte     int64
	StartTime   time.Duration
	ElapsedTime time.Duration
	VSZByte     uint64
	UTime       time.Duration
	STime       time.Duration

	UserUID        int
	UserName       string
	UserPrivileged string

	Sockets []socket.Socket

	width, height int
	viewport      viewport.Model
	viewportReady bool
	viewReady     bool
	content       string
}

type styleFunc func(string) string

func New() Model {
	return Model{}
}

func (m Model) Init() tea.Cmd { return nil }

func (m Model) Update(msg tea.Msg) (Model, tea.Cmd) {
	var (
		cmd  tea.Cmd
		cmds []tea.Cmd
	)

	rerenderContent := false

	switch msg := msg.(type) {
	case initMsg:
		m.content = m.renderContent()
		m.updateViewport(msg.width, msg.height)
	case tea.WindowSizeMsg:
		m.content = m.renderContent()
		m.updateViewport(msg.Width, msg.Height)
	case detailHydratedMsg:
		m.ExecPath = msg.ExecPath
		m.Command = msg.Command
		m.PPID = msg.PPID
		m.ParentName = msg.ParentName
		rerenderContent = true
	case resourceHydratedMsg:
		m.RSSByte = msg.RSSByte
		m.StartTime = msg.StartTime
		m.ElapsedTime = msg.ElapsedTime
		m.VSZByte = msg.VSZByte
		m.UTime = msg.UTime
		m.STime = msg.STime
		rerenderContent = true
	case userHydratedMsg:
		m.UserUID = msg.UserUID
		m.UserName = msg.UserName
		m.UserPrivileged = msg.UserPrivileged
		rerenderContent = true
	case socketHydrateMsg:
		m.Sockets = msg.Sockets
		rerenderContent = true
	case tea.KeyMsg:
		switch msg.String() {
		case "esc":
			return m, func() tea.Msg {
				return message.GoBack{}
			}
		case "q", "ctrl+c":
			return m, tea.Quit
		}
	}

	if rerenderContent {
		m.content = m.renderContent()
		if m.viewportReady {
			m.viewport.SetContent(m.content)
		}
	}

	m.viewport.SetContent(m.content)

	m.viewport, cmd = m.viewport.Update(msg)
	cmds = append(cmds, cmd)

	return m, tea.Batch(cmds...)
}

func (m Model) View() tea.View {
	var v tea.View
	v.AltScreen = true // use the full size of the terminal in its "alternate screen buffer"
	if !m.viewportReady {
		v.SetContent("\n  Initializing...")
	} else {
		v.SetContent(
			fmt.Sprintf("%s\n%s\n%s",
				m.viewport.View(),
				actionBar(m.width),
				statusBar(m.viewport.ScrollPercent()),
			),
		)
	}

	return v
}

func (m Model) renderContent() string {
	return processDetailSection(
		m.width,
		m.height,
		m.Name,
		m.PID,
		m.ExecPath,
		m.ParentName,
		m.PPID,
		m.Command,
		m.Sockets,
		m.UserUID,
		m.UserName,
		m.UserPrivileged,
		int(m.RSSByte),
		int(m.VSZByte),
		m.StartTime,
		m.ElapsedTime,
		m.UTime,
		m.STime,
	)
}

func formatSocketText(sock socket.Socket, lStyle styleFunc, eStyle styleFunc, cStyle styleFunc) string {
	text := ""
	switch sock.State {
	case "LISTEN":
		text = lStyle(fmt.Sprintf("%s %s:%d (%s)", sock.Proto, sock.Addr, sock.Port, sock.State))
	case "ESTABLISHED":
		text = eStyle(fmt.Sprintf("%s %s:%d (%s)", sock.Proto, sock.Addr, sock.Port, sock.State))
	case "CLOSE":
		text = cStyle(fmt.Sprintf("%s %s:%d (%s)", sock.Proto, sock.Addr, sock.Port, sock.State))
	}
	return text
}

func Initialize(width, height int) tea.Cmd {
	return func() tea.Msg {
		return initMsg{
			width:  width,
			height: height,
		}
	}
}

func (m *Model) updateViewport(width, height int) {
	m.width = width
	m.height = height
	actionBarHeight := lipgloss.Height(actionBar(m.width))
	statusBarHeight := lipgloss.Height(statusBar(m.viewport.ScrollPercent()))
	if !m.viewportReady {
		m.viewport = viewport.New(viewport.WithWidth(width), viewport.WithHeight(height-actionBarHeight-statusBarHeight))
		m.viewport.SetContent(m.content)
		m.viewportReady = true
	} else {
		m.viewport.SetWidth(width)
		m.viewport.SetHeight(height - actionBarHeight - statusBarHeight)
	}
}
