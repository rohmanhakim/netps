package processdetail

import (
	"fmt"
	"netps/internal/socket"
	"netps/internal/util"
	"strconv"
	"strings"
	"time"

	"charm.land/lipgloss/v2"
)

func normalList(name string, values []string) string {
	baseColor := lipgloss.Color("#EEEEEE")
	base := lipgloss.NewStyle().Foreground(baseColor)
	subtle := lipgloss.Color("#383838")

	listHeader := base.
		BorderStyle(lipgloss.NormalBorder()).
		BorderBottom(true).
		BorderForeground(subtle).
		MarginTop(1).
		Render

	v := lipgloss.JoinVertical(lipgloss.Left,
		listHeader(name),
		lipgloss.JoinVertical(
			lipgloss.Left,
			lipgloss.JoinVertical(lipgloss.Left,
				values...,
			),
		),
	)

	return v
}

func labeledList(name string, labels []string, values []string) string {
	baseColor := lipgloss.Color("#EEEEEE")
	base := lipgloss.NewStyle().Foreground(baseColor)
	subtle := lipgloss.Color("#383838")

	grayText := lipgloss.NewStyle().Foreground(lipgloss.Color("#AAA")).Render
	whiteText := lipgloss.NewStyle().Foreground(lipgloss.Color("#FFF")).Render

	hSpacer := lipgloss.NewStyle().
		Width(1)

	listHeader := base.
		BorderStyle(lipgloss.NormalBorder()).
		BorderBottom(true).
		BorderForeground(subtle).
		MarginTop(1).
		Render

	styledLabels := []string{}
	for _, l := range labels {
		styledLabels = append(styledLabels, grayText(l))
	}
	styledValues := []string{}
	for _, t := range values {
		styledValues = append(styledValues, whiteText(t))
	}

	var v string
	if strings.TrimSpace(name) == "" {
		v = base.MarginTop(1).Render(lipgloss.JoinHorizontal(
			lipgloss.Top,
			lipgloss.JoinVertical(lipgloss.Left,
				styledLabels...,
			),
			hSpacer.Render(),
			lipgloss.JoinVertical(lipgloss.Left,
				styledValues...,
			),
		))
	} else {
		v = lipgloss.JoinVertical(lipgloss.Left,
			listHeader(name),
			lipgloss.JoinHorizontal(
				lipgloss.Top,
				lipgloss.JoinVertical(lipgloss.Left,
					styledLabels...,
				),
				hSpacer.Render(),
				lipgloss.JoinVertical(lipgloss.Left,
					styledValues...,
				),
			),
		)
	}

	return v
}

func verticalGroup(sections ...string) string {
	return lipgloss.JoinVertical(lipgloss.Left, sections...)
}

func horizontalGroup(sections ...string) string {
	container := lipgloss.NewStyle().MarginRight(2)
	margined := []string{}
	for _, s := range sections {
		margined = append(margined, container.Render(s))
	}
	return container.Render(lipgloss.JoinHorizontal(lipgloss.Top, margined...))
}

func actionBar(windowWidth int, actions []string) string {
	style := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#FFF"))

	out := style.
		Width(windowWidth).
		Render(strings.Join(actions, " · "))
		// Render("[↑↓] scroll · [s] send signal · [c] copy · [esc] back · [q] quit")
	return out
}

func statusBar(windowWidth int, modeName string, modeColor string, scrollPercent float64) string {
	scrollingInfoStyle := lipgloss.NewStyle().
		Background(lipgloss.Color("237")).
		Foreground(lipgloss.Color("#C1C6B2"))

	modeStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("255")).
		Background(lipgloss.Color(modeColor)).
		Align(lipgloss.Left)

	mode := modeStyle.Render(modeName)
	info := scrollingInfoStyle.
		Render(fmt.Sprintf("scrolling %3.f%%", scrollPercent*100))
	lineStyle := lipgloss.NewStyle().
		Background(lipgloss.Color("237"))
	line := lineStyle.
		Render(strings.Repeat(" ", max(0, windowWidth-lipgloss.Width(mode)-lipgloss.Width(info))))
	return lipgloss.JoinHorizontal(lipgloss.Center, mode, line, info)
}

func horizontalSpacer(height int) string {
	hSpacer := lipgloss.NewStyle().
		Height(height)
	return hSpacer.Render()
}

func processDetailSection(
	width int,
	height int,
	name string,
	pid int,
	execPath string,
	parentName string,
	ppid int,
	command string,
	sockets []socket.Socket,
	userUid int,
	userName string,
	userPrivileged string,
	rssByte int,
	vszByte int,
	startTime time.Duration,
	elapsedTime time.Duration,
	uTime time.Duration,
	sTime time.Duration,
) string {

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
		name,
		strconv.Itoa(pid),
		execPath,
		fmt.Sprintf("%s (%d)", parentName, ppid),
	}
	staticIdSection := labeledList("", staticIdLabels, staticIdValues)

	commandSection := normalList("Command", []string{grayText(command)})

	socketItems := []string{}
	for _, s := range sockets {
		socketItems = append(socketItems, formatSocketText(s, listenSocketItem, establishedSocketItem, closedSocketItem))
	}
	aggregated := socket.Aggregate(sockets)
	socketSection := normalList(fmt.Sprintf("Sockets · %dL %dE %dC (%d)", aggregated.ListenCount, aggregated.EstablishedCount, aggregated.CloseCount, len(socketItems)), socketItems)

	ownerShipLabels := []string{"User", "Privilege"}
	ownerShipValues := []string{
		fmt.Sprintf("%s (%d)", userName, userUid),
		userPrivileged}
	ownerSection := labeledList("Ownership", ownerShipLabels, ownerShipValues)

	resourceLabels := []string{
		"Resident Memory",
		"Virtual Memory",
		"Start Time",
		"Elapsed Time",
		"User Time",
		"System Time"}
	resourceValues := []string{
		fmt.Sprintf("%d Bytes", rssByte),
		fmt.Sprintf("%d Bytes", vszByte),
		util.DurationToHHMMSS(startTime),
		util.DurationToHHMMSS(elapsedTime),
		util.DurationToHHMMSS(uTime),
		util.DurationToHHMMSS(sTime)}
	resourceSection := labeledList("Resources", resourceLabels, resourceValues)

	firstSection := verticalGroup(staticIdSection, commandSection)
	secondSection := horizontalGroup(
		verticalGroup(resourceSection, ownerSection),
		verticalGroup(socketSection),
	)

	contentHeight := height - lipgloss.Height(actionBar(width, []string{""})) - base.GetVerticalFrameSize()
	ui := lipgloss.NewStyle().
		Height(contentHeight).
		Width(width - base.GetHorizontalFrameSize()).
		Render(
			lipgloss.JoinVertical(lipgloss.Left, firstSection, secondSection, horizontalSpacer(1)),
		)

	return ui
}

func commandModal(text string) string {

	modalStyle := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("57")).
		PaddingLeft(1).
		PaddingRight(1).
		Align(lipgloss.Center, lipgloss.Center)

	return modalStyle.Render(text)
}
