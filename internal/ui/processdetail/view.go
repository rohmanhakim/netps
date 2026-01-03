package processdetail

import (
	"fmt"
	"image/color"
	"netps/internal/socket"
	"netps/internal/ui/common"
	"netps/internal/util"
	"strconv"
	"strings"
	"time"

	"charm.land/lipgloss/v2"
)

func normalList(active bool, inactiveColor color.Color, name string, values []string) string {
	baseForegroundColor := lipgloss.Color("255")
	darkCharcoalColor := lipgloss.Color("236")

	var baseStyle lipgloss.Style
	if active {
		baseStyle = lipgloss.NewStyle().Foreground(baseForegroundColor)
	} else {
		baseStyle = lipgloss.NewStyle().Foreground(inactiveColor)
	}

	listHeader := baseStyle.
		BorderStyle(lipgloss.NormalBorder()).
		BorderBottom(true).
		BorderForeground(darkCharcoalColor).
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

func labeledList(active bool, inactiveColor color.Color, name string, labels []string, values []string) string {
	baseForegroundColor := lipgloss.Color("255")
	darkGrayColor := lipgloss.Color("248")
	darkCharcoalColor := lipgloss.Color("236")

	var baseStyle, subtleStyle lipgloss.Style
	if active {
		baseStyle = lipgloss.NewStyle().Foreground(baseForegroundColor)
		subtleStyle = lipgloss.NewStyle().Foreground(darkGrayColor)
	} else {
		baseStyle = lipgloss.NewStyle().Foreground(inactiveColor)
		subtleStyle = lipgloss.NewStyle().Foreground(inactiveColor)
	}

	grayText := subtleStyle.Render
	whiteText := baseStyle.Render

	hSpacer := lipgloss.NewStyle().
		Width(1)

	listHeader := baseStyle.
		BorderStyle(lipgloss.NormalBorder()).
		BorderBottom(true).
		BorderForeground(darkCharcoalColor).
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
		v = baseStyle.MarginTop(1).Render(lipgloss.JoinHorizontal(
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

func horizontalSpacer(height int) string {
	hSpacer := lipgloss.NewStyle().
		Height(height)
	return hSpacer.Render()
}

func processDetailSection(
	active bool,
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

	whiteColor := lipgloss.Color("255")
	grayColor := lipgloss.Color("248")
	canaryColor := lipgloss.Color("191")
	blueJeansColor := lipgloss.Color("75")
	darkCharcoalColor := lipgloss.Color("236")

	var baseForegroundStyle, subtleForegroundStyle lipgloss.Style

	if active {
		baseForegroundStyle = lipgloss.NewStyle().Foreground(whiteColor)
		subtleForegroundStyle = lipgloss.NewStyle().Foreground(grayColor)
	} else {
		baseForegroundStyle = lipgloss.NewStyle().Foreground(darkCharcoalColor)
		subtleForegroundStyle = lipgloss.NewStyle().Foreground(darkCharcoalColor)
	}

	var positiveColor, neutralColor color.Color
	if active {
		positiveColor = canaryColor
		neutralColor = blueJeansColor
	} else {
		positiveColor = darkCharcoalColor
		neutralColor = darkCharcoalColor
	}

	var baseForegroundText, subtleForegroundText func(strs ...string) string

	baseForegroundText = baseForegroundStyle.Render
	subtleForegroundText = subtleForegroundStyle.Render

	positiveBullet := lipgloss.NewStyle().SetString("•").
		Foreground(positiveColor).
		PaddingRight(1).
		String()

	neutralBullet := lipgloss.NewStyle().SetString("•").
		Foreground(neutralColor).
		PaddingRight(1).
		String()

	idleBullet := lipgloss.NewStyle().SetString("•").
		Foreground(darkCharcoalColor).
		PaddingRight(1).
		String()

	listenSocketItem := func(s string) string {
		return positiveBullet + baseForegroundText(s)
	}
	establishedSocketItem := func(s string) string {
		return neutralBullet + baseForegroundText(s)
	}
	closedSocketItem := func(s string) string {
		return idleBullet + subtleForegroundText(s)
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
	staticIdSection := labeledList(active, darkCharcoalColor, "", staticIdLabels, staticIdValues)

	commandSection := normalList(active, darkCharcoalColor, "Command", []string{subtleForegroundText(command)})

	socketItems := []string{}
	for _, s := range sockets {
		socketItems = append(socketItems, formatSocketText(s, listenSocketItem, establishedSocketItem, closedSocketItem))
	}
	aggregated := socket.Aggregate(sockets)
	socket := fmt.Sprintf("Sockets · %dL %dE %dC (%d)", aggregated.ListenCount, aggregated.EstablishedCount, aggregated.CloseCount, len(socketItems))
	socketSection := normalList(active, darkCharcoalColor, socket, socketItems)

	ownerShipLabels := []string{"User", "Privilege"}
	ownerShipValues := []string{
		fmt.Sprintf("%s (%d)", userName, userUid),
		userPrivileged}
	ownerSection := labeledList(active, darkCharcoalColor, "Ownership", ownerShipLabels, ownerShipValues)

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
	resourceSection := labeledList(active, darkCharcoalColor, "Resources", resourceLabels, resourceValues)

	firstSection := verticalGroup(staticIdSection, commandSection)
	secondSection := horizontalGroup(
		verticalGroup(resourceSection, ownerSection),
		verticalGroup(socketSection),
	)

	contentHeight := height - lipgloss.Height(common.ActionBar(width, []string{""})) - baseForegroundStyle.GetVerticalFrameSize()
	ui := lipgloss.NewStyle().
		Height(contentHeight).
		Width(width - baseForegroundStyle.GetHorizontalFrameSize()).
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
