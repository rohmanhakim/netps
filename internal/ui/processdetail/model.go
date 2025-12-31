package processdetail

import (
	"fmt"
	"netps/internal/socket"
	"netps/internal/ui/message"
	"netps/internal/util"
	"os"
	"strconv"
	"time"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/charmbracelet/x/term"
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
}

type styleFunc func(string) string

func New() Model {
	return Model{}
}

func (m Model) Init() tea.Cmd { return nil }

func (m Model) Update(msg tea.Msg) (Model, tea.Cmd) {
	var cmd tea.Cmd
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
	case detailHydratedMsg:
		m.ExecPath = msg.ExecPath
		m.Command = msg.Command
		m.PPID = msg.PPID
		m.ParentName = msg.ParentName
	case resourceHydratedMsg:
		m.RSSByte = msg.RSSByte
		m.StartTime = msg.StartTime
		m.ElapsedTime = msg.ElapsedTime
		m.VSZByte = msg.VSZByte
		m.UTime = msg.UTime
		m.STime = msg.STime
	case userHydratedMsg:
		m.UserUID = msg.UserUID
		m.UserName = msg.UserName
		m.UserPrivileged = msg.UserPrivileged
	case socketHydrateMsg:
		m.Sockets = msg.Sockets
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
	return m, cmd
}

func (m Model) View() tea.View {

	baseColor := lipgloss.Color("#EEEEEE")
	base := lipgloss.NewStyle().Foreground(baseColor)

	positive := lipgloss.Color("#aad84c")
	neutral := lipgloss.Color("#56b8f2")

	whiteText := lipgloss.NewStyle().Foreground(baseColor).Render
	grayText := lipgloss.NewStyle().Foreground(lipgloss.Color("#AAAAAA")).Render

	positiveBullet := lipgloss.NewStyle().SetString("•").
		Foreground(positive).
		PaddingRight(1).
		String()

	neutralBullet := lipgloss.NewStyle().SetString("•").
		Foreground(neutral).
		PaddingRight(1).
		String()

	idleBullet := lipgloss.NewStyle().SetString("•").
		Foreground(lipgloss.Color("#333")).
		PaddingRight(1).
		String()

	listenSocketItem := func(s string) string {
		return positiveBullet + whiteText(s)
	}
	establishedSocketItem := func(s string) string {
		return neutralBullet + whiteText(s)
	}
	closedSocketItem := func(s string) string {
		return idleBullet + grayText(s)
	}

	staticIdLabels := []string{
		"Name",
		"PID",
		"Exec Path",
		"Parent",
	}
	staticIdValues := []string{
		m.Name,
		strconv.Itoa(m.PID),
		m.ExecPath,
		fmt.Sprintf("%s (%d)", m.ParentName, m.PPID),
	}
	staticIdSection := labeledList("", staticIdLabels, staticIdValues)

	commandSection := list("Command", []string{grayText(m.Command)})

	socketItems := []string{}
	for _, s := range m.Sockets {
		socketItems = append(socketItems, formatSocketText(s, listenSocketItem, establishedSocketItem, closedSocketItem))
	}
	socketSection := list("Section", socketItems)

	ownerShipLabels := []string{"User", "Privilege"}
	ownerShipValues := []string{
		fmt.Sprintf("%s (%d)", m.UserName, m.UserUID),
		m.UserPrivileged}
	ownerSection := labeledList("Owner", ownerShipLabels, ownerShipValues)

	resourceLabels := []string{
		"Resident Memory",
		"Virtual Memory",
		"Start Time",
		"Elapsed Time",
		"User Time",
		"System Time"}
	resourceValues := []string{
		fmt.Sprintf("%d Bytes", m.RSSByte),
		fmt.Sprintf("%d Bytes", m.VSZByte),
		util.DurationToHHMMSS(m.StartTime),
		util.DurationToHHMMSS(m.ElapsedTime),
		util.DurationToHHMMSS(m.UTime),
		util.DurationToHHMMSS(m.STime)}
	resourceSection := labeledList("Resources", resourceLabels, resourceValues)

	firstSection := verticalGroup(staticIdSection, commandSection)
	secondSection := horizontalGroup(
		verticalGroup(socketSection, ownerSection),
		resourceSection,
	)

	w, h, err := term.GetSize(uintptr(os.Stdout.Fd()))
	if err == nil {
		m.width = w
		m.height = h
	}

	actions := actionBar(m.width)

	contentHeight := m.height - lipgloss.Height(actions) - base.GetVerticalFrameSize()
	ui := lipgloss.NewStyle().
		Height(contentHeight).
		Width(m.width - base.GetHorizontalFrameSize()).
		Render(
			lipgloss.JoinVertical(lipgloss.Left, firstSection, secondSection),
		)

	v := tea.NewView(base.Render(fmt.Sprintf("%s\n%s", ui, actions)))
	v.AltScreen = true

	return v
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
