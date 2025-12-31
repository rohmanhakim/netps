package processdetail

import (
	"context"
	"fmt"
	"netps/internal/process"
	"netps/internal/procfs"
	"netps/internal/socket"
	"netps/internal/sysconf"
	"netps/internal/ui/message"
	"netps/internal/util"
	"os"
	"strconv"
	"time"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"charm.land/lipgloss/v2/table"
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
	UserPrivileged bool

	Sockets []socket.Socket

	width, height int
}

type detailHydratedMsg struct {
	ExecPath   string
	Command    string
	PPID       int
	ParentName string
	Err        error
}

type resourceHydratedMsg struct {
	RSSByte     int64
	StartTime   time.Duration
	ElapsedTime time.Duration
	VSZByte     uint64
	UTime       time.Duration
	STime       time.Duration
}

type userHydratedMsg struct {
	UserUID        int
	UserName       string
	UserPrivileged bool
}

type socketHydrateMsg struct {
	Sockets []socket.Socket
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
	subtle := lipgloss.Color("#383838")

	positive := lipgloss.Color("#aad84c")
	neutral := lipgloss.Color("#56b8f2")

	whiteText := lipgloss.NewStyle().Foreground(baseColor).Render
	grayText := lipgloss.NewStyle().Foreground(lipgloss.Color("#AAAAAA")).Render

	sectionContainer := lipgloss.NewStyle().
		MarginLeft(1).
		MarginRight(1).
		MarginBottom(1)
	sectionHSpacer := lipgloss.NewStyle().
		Width(1)

	sectionHeader := base.
		BorderStyle(lipgloss.NormalBorder()).
		BorderBottom(true).
		BorderForeground(subtle).
		Render

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

	processTable := table.New().Border(lipgloss.HiddenBorder()).Wrap(true)
	processTable.Row(grayText("Name"), whiteText(m.Name))
	processTable.Row(grayText("PID"), whiteText(strconv.Itoa(m.PID)))
	processTable.Row(grayText("Exec Path"), whiteText(m.ExecPath))

	parentInfo := fmt.Sprintf("%s (%d)", m.ParentName, m.PPID)
	processTable.Row(grayText("Parent"), whiteText(parentInfo))

	commandSection := sectionContainer.Render(
		lipgloss.JoinVertical(
			lipgloss.Left,
			sectionHeader("Command"),
			grayText(m.Command),
		),
	)

	socketItems := []string{}
	socketItems = append(socketItems, sectionHeader("Sockets"))
	for _, s := range m.Sockets {
		socketItems = append(socketItems, formatSocketText(s, listenSocketItem, establishedSocketItem, closedSocketItem))
	}
	socketSection := lipgloss.JoinVertical(lipgloss.Left,
		socketItems...,
	)

	userText := fmt.Sprintf("%s (%d)", m.UserName, m.UserUID)
	privilegedText := "unprivileged"
	if m.UserPrivileged {
		privilegedText = "privileged"
	}
	ownerSection := lipgloss.JoinVertical(lipgloss.Left,
		sectionHeader("Owner"),
		lipgloss.JoinHorizontal(
			lipgloss.Top,
			lipgloss.JoinVertical(lipgloss.Left,
				grayText("User"),
				grayText("Privilege"),
			),
			sectionHSpacer.Render(),
			lipgloss.JoinVertical(lipgloss.Left,
				whiteText(userText),
				whiteText(privilegedText),
			),
		),
	)

	rssText := fmt.Sprintf("%d Bytes", m.RSSByte)
	vszText := fmt.Sprintf("%d Bytes", m.VSZByte)
	resourceSection := lipgloss.JoinVertical(lipgloss.Left,
		sectionHeader("Resources"),
		lipgloss.JoinHorizontal(
			lipgloss.Top,
			lipgloss.JoinVertical(lipgloss.Left,
				grayText("Resident Memory"),
				grayText("Virtual Memory"),
				grayText("Start Time"),
				grayText("Elapsed Time"),
				grayText("User Time"),
				grayText("System Time"),
			),
			sectionHSpacer.Render(),
			lipgloss.JoinVertical(lipgloss.Left,
				whiteText(rssText),
				whiteText(vszText),
				whiteText(util.DurationToHHMMSS(m.StartTime)),
				whiteText(util.DurationToHHMMSS(m.ElapsedTime)),
				whiteText(util.DurationToHHMMSS(m.UTime)),
				whiteText(util.DurationToHHMMSS(m.STime)),
			),
		),
	)

	secondSection := lipgloss.JoinHorizontal(
		lipgloss.Top,
		lipgloss.JoinVertical(lipgloss.Left,
			sectionContainer.Render(socketSection),
			sectionContainer.Render(ownerSection),
		),
		sectionContainer.Render(resourceSection),
	)

	footerStyle := lipgloss.NewStyle().
		Height(1). // Fixed height for the footer
		Align(lipgloss.Center).
		BorderTop(true).
		BorderStyle(lipgloss.NormalBorder()).
		BorderForeground(subtle).
		Foreground(lipgloss.Color("#FFFFFF")).
		Render

	statusBar := footerStyle("[s] send signal · [c] copy · [esc] back")

	w, h, err := term.GetSize(uintptr(os.Stdout.Fd()))

	if err == nil {
		m.width = w
		m.height = h
	}

	contentHeight := m.height - lipgloss.Height(footerStyle("placeholder")) - base.GetVerticalFrameSize()
	ui := lipgloss.NewStyle().
		Height(contentHeight).
		Width(m.width - base.GetHorizontalFrameSize()).
		Render(
			lipgloss.JoinVertical(lipgloss.Left, processTable.Render(), commandSection, secondSection),
		)

	v := tea.NewView(base.Render(fmt.Sprintf("%s\n%s", ui, statusBar)))
	v.AltScreen = true

	return v
}

func formatSocketText(sock socket.Socket, lStyle styleFunc, eStyle styleFunc, cStyle styleFunc) string {
	// tcp 127.0.0.1:46431 (LISTEN)"
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

func HydrateStaticIds(pid int) tea.Cmd {
	return func() tea.Msg {
		procfsClient := procfs.NewClient()
		sysconfClient := sysconf.NewClient()

		procesService := process.NewService(
			procfsClient,
			procfsClient,
			sysconfClient,
			sysconfClient,
			procfsClient,
			procfsClient,
			procfsClient,
		)
		processDetail, err := procesService.GetProcessDetail(context.Background(), pid)
		if err != nil {
			panic(err)
		}
		return detailHydratedMsg{
			ExecPath:   processDetail.ExecPath,
			Command:    processDetail.Command,
			PPID:       processDetail.PPID,
			ParentName: processDetail.ParentName,

			Err: err,
		}
	}
}

func HydrateResource(pid int) tea.Cmd {
	return func() tea.Msg {
		procfsClient := procfs.NewClient()
		sysconfClient := sysconf.NewClient()

		processService := process.NewService(
			procfsClient,
			procfsClient,
			sysconfClient,
			sysconfClient,
			procfsClient,
			procfsClient,
			procfsClient,
		)
		processResource, err := processService.GetProcessResource(context.Background(), pid)
		if err != nil {
			panic(err)
		}
		return resourceHydratedMsg{
			RSSByte:     processResource.ResidentSetSizeByte,
			StartTime:   processResource.StartTimeSec,
			ElapsedTime: processResource.ElapsedTimeSec,
			VSZByte:     processResource.VirtualMemorySize,
			UTime:       processResource.UserCPUTimeSecond,
			STime:       processResource.SystemCPUTimeSecond,
		}
	}
}

func HydrateUser(pid int) tea.Cmd {
	return func() tea.Msg {
		procfsClient := procfs.NewClient()
		sysconfClient := sysconf.NewClient()

		processService := process.NewService(
			procfsClient,
			procfsClient,
			sysconfClient,
			sysconfClient,
			procfsClient,
			procfsClient,
			procfsClient,
		)

		processUser, err := processService.GetUser(context.Background(), pid)
		if err != nil {
			panic(err)
		}
		return userHydratedMsg{
			UserUID:        processUser.RealUID,
			UserName:       processUser.Name,
			UserPrivileged: processUser.Privileged,
		}
	}
}

func HydrateSockets(pid int) tea.Cmd {
	return func() tea.Msg {
		procfsClient := procfs.NewClient()
		socketService := socket.NewService(procfsClient)
		socketStates := []string{"LISTEN", "ESTABLISHED", "CLOSE"}
		sockets, err := socketService.GetSocketsByStates(context.Background(), pid, socketStates)
		if err != nil {
			panic(err)
		}
		return socketHydrateMsg{
			Sockets: sockets,
		}
	}
}
