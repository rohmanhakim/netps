package processdetail

import (
	"context"
	"fmt"
	"netps/internal/process"
	"netps/internal/procfs"
	"netps/internal/ui/message"
	"os"
	"strconv"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"charm.land/lipgloss/v2/table"
	"github.com/charmbracelet/x/term"
)

type Model struct {
	PID           int
	Name          string
	ExecPath      string
	Command       string
	PPID          int
	ParentName    string
	width, height int
}

type detailHydratedMsg struct {
	ExecPath   string
	Command    string
	PPID       int
	ParentName string
	Err        error
}

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

	base := lipgloss.NewStyle().Foreground(lipgloss.Color("#EEEEEE"))
	subtle := lipgloss.Color("#383838")
	special := lipgloss.Color("#73F59F")

	whiteText := lipgloss.NewStyle().Foreground(lipgloss.Color("#EEEEEE")).Render
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

	bullet := lipgloss.NewStyle().SetString("•").
		Foreground(special).
		PaddingRight(1).
		String()

	bulletSectionItem := func(s string) string {
		return bullet + grayText(s)
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

	socketSection := lipgloss.JoinVertical(lipgloss.Left,
		sectionHeader("Sockets"),
		bulletSectionItem("tcp 127.0.0.1:46431 (LISTEN)"),
		bulletSectionItem("tcp [::]:1716 (LISTEN)"),
	)

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
				whiteText("arif (1000)"),
				whiteText("unprivileged"),
			),
		),
	)
	resourceSection := lipgloss.JoinVertical(lipgloss.Left,
		sectionHeader("Resources"),
		lipgloss.JoinHorizontal(
			lipgloss.Top,
			lipgloss.JoinVertical(lipgloss.Left,
				grayText("CPU"),
				grayText("Memory"),
				grayText("Threads"),
				grayText("Started"),
			),
			sectionHSpacer.Render(),
			lipgloss.JoinVertical(lipgloss.Left,
				whiteText("0.2%"),
				whiteText("42 MB RSS"),
				whiteText("7"),
				whiteText("12m ago"),
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

func HydrateStaticIds(pid int) tea.Cmd {
	return func() tea.Msg {
		client := procfs.NewClient()
		service := process.NewService(client, client)
		processDetail, err := service.GetProcessDetail(context.Background(), pid)
		if err != nil {
			panic(err)
		}
		return detailHydratedMsg{
			ExecPath:   processDetail.ExecPath,
			Command:    processDetail.Command,
			PPID:       processDetail.PPID,
			ParentName: processDetail.ParentName,
			Err:        err,
		}
	}
}
