package model

import (
	"fmt"
	"os"
	"strconv"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"charm.land/lipgloss/v2/table"
	"github.com/charmbracelet/x/term"
)

type ProcessDetail struct {
	Name          string
	PID           int
	ExecPath      string
	Command       string
	Parent        int
	width, height int
}

func (m ProcessDetail) Init() tea.Cmd { return nil }

func (m ProcessDetail) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
	case tea.KeyMsg:
		switch msg.String() {
		case "esc":
			return MainPage{}.Initialize(), cmd
		case "q", "ctrl+c":
			return m, tea.Quit
		}
	}
	return m, cmd
}

func (m ProcessDetail) View() tea.View {

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

	processCommand := `rsync -avz --delete --partial --progress --bwlimit=5000 --compress-level=9 --exclude='.git' --exclude='node_modules' --exclude='*.log' -e "ssh -o StrictHostKeyChecking=no -o Compression=yes -c aes256-gcm@openssh.com" /local/project/ user@remote:/var/www/project/ `
	processTable := table.New().Border(lipgloss.HiddenBorder()).Wrap(true)
	processTable.Row(grayText("Name"), whiteText("dlv"))
	processTable.Row(grayText("PID"), whiteText(strconv.Itoa(123)))
	processTable.Row(grayText("Exec Path"), whiteText("/usr/bin/dlv"))
	processTable.Row(grayText("Parent"), whiteText("go"))

	commandSection := sectionContainer.Render(
		lipgloss.JoinVertical(
			lipgloss.Left,
			sectionHeader("Command"),
			grayText(processCommand),
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
