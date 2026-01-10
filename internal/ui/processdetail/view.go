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

func normalList(
	active bool,
	theme common.Theme,
	inactiveColor color.Color,
	name string,
	values []string,
) string {
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
		MarginTop(theme.SpacingSmall).
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

func labeledList(
	active bool,
	theme common.Theme,
	inactiveColor color.Color,
	name string,
	labels []string,
	values []string,
) string {
	baseForegroundColor := lipgloss.Color(theme.ColorForegroundBase)
	subtleForegroundColor := lipgloss.Color(theme.ColorForegroundSubtle)
	darkCharcoalColor := lipgloss.Color(theme.ColorInactive)

	var baseStyle, subtleStyle lipgloss.Style
	if active {
		baseStyle = lipgloss.NewStyle().Foreground(baseForegroundColor)
		subtleStyle = lipgloss.NewStyle().Foreground(subtleForegroundColor)
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
		MarginTop(theme.SpacingSmall).
		Render

	styledLabels := []string{}
	for _, l := range labels {
		styledLabels = append(styledLabels, grayText(l))
	}
	styledValues := []string{}
	for _, val := range values {
		styledValues = append(styledValues, whiteText(val))
	}

	var v string
	if strings.TrimSpace(name) == "" {
		v = baseStyle.MarginTop(theme.SpacingSmall).Render(lipgloss.JoinHorizontal(
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

func horizontalGroup(theme common.Theme, sections ...string) string {
	container := lipgloss.NewStyle().MarginRight(theme.SpacingMedium)
	margined := []string{}
	for _, s := range sections {
		margined = append(margined, container.Render(s))
	}
	return container.Render(lipgloss.JoinHorizontal(lipgloss.Top, margined...))
}

func processDetailSection(
	active bool,
	theme common.Theme,
	width int,
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

	baseForegroundColor := lipgloss.Color(theme.ColorForegroundBase) // COlorWhite
	subtleForegroundColor := lipgloss.Color(theme.ColorForegroundSubtle)
	positiveColor := lipgloss.Color(theme.ColorSuccess)
	neutralColor := lipgloss.Color(theme.ColorNeutral)
	inactiveColor := lipgloss.Color(theme.ColorInactive)

	var baseForegroundStyle, subtleForegroundStyle lipgloss.Style

	if active {
		baseForegroundStyle = lipgloss.NewStyle().Foreground(baseForegroundColor)
		subtleForegroundStyle = lipgloss.NewStyle().Foreground(subtleForegroundColor)
	} else {
		baseForegroundStyle = lipgloss.NewStyle().Foreground(inactiveColor)
		subtleForegroundStyle = lipgloss.NewStyle().Foreground(inactiveColor)
	}

	var positiveBulletColor, neutralBulletColor color.Color
	if active {
		positiveBulletColor = positiveColor
		neutralBulletColor = neutralColor
	} else {
		positiveBulletColor = inactiveColor
		neutralBulletColor = inactiveColor
	}

	var baseForegroundText, subtleForegroundText func(strs ...string) string

	baseForegroundText = baseForegroundStyle.Render
	subtleForegroundText = subtleForegroundStyle.Render

	positiveBullet := lipgloss.NewStyle().SetString("•").
		Foreground(positiveBulletColor).
		PaddingRight(theme.SpacingSmall).
		String()

	neutralBullet := lipgloss.NewStyle().SetString("•").
		Foreground(neutralBulletColor).
		PaddingRight(theme.SpacingSmall).
		String()

	idleBullet := lipgloss.NewStyle().SetString("•").
		Foreground(inactiveColor).
		PaddingRight(theme.SpacingSmall).
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
	staticIdSection := labeledList(active, theme, lipgloss.Color(theme.ColorInactive), "", staticIdLabels, staticIdValues)

	commandSection := normalList(active, theme, lipgloss.Color(theme.ColorInactive), "Command", []string{subtleForegroundText(command)})

	socketItems := []string{}
	for _, s := range sockets {
		socketItems = append(socketItems, formatSocketText(s, listenSocketItem, establishedSocketItem, closedSocketItem))
	}
	aggregated := socket.Aggregate(sockets)
	socket := fmt.Sprintf("Sockets · %dL %dE %dC (%d)", aggregated.ListenCount, aggregated.EstablishedCount, aggregated.CloseCount, len(socketItems))
	socketSection := normalList(active, theme, lipgloss.Color(theme.ColorInactive), socket, socketItems)

	ownerShipLabels := []string{"User", "Privilege"}
	ownerShipValues := []string{
		fmt.Sprintf("%s (%d)", userName, userUid),
		userPrivileged}
	ownerSection := labeledList(active, theme, lipgloss.Color(theme.ColorInactive), "Ownership", ownerShipLabels, ownerShipValues)

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
	resourceSection := labeledList(active, theme, lipgloss.Color(theme.ColorInactive), "Resources", resourceLabels, resourceValues)

	firstSection := verticalGroup(staticIdSection, commandSection)
	secondSection := horizontalGroup(
		theme,
		verticalGroup(resourceSection, ownerSection),
		verticalGroup(socketSection),
	)

	ui := lipgloss.NewStyle().
		Width(width - baseForegroundStyle.GetHorizontalFrameSize()).
		Render(
			lipgloss.JoinVertical(lipgloss.Left, firstSection, secondSection),
		)

	return ui
}

func formatSocketText(sock socket.Socket, lStyle styleFunc, eStyle styleFunc, cStyle styleFunc) string {
	text := ""
	switch sock.State {
	case socket.StateListen:
		text = lStyle(fmt.Sprintf("%s %s:%d (%s)", sock.Proto, sock.Addr, sock.Port, "LISTEN"))
	case socket.StateEstablished:
		text = eStyle(fmt.Sprintf("%s %s:%d (%s)", sock.Proto, sock.Addr, sock.Port, "ESTABLISHED"))
	case socket.StateClose:
		text = cStyle(fmt.Sprintf("%s %s:%d (%s)", sock.Proto, sock.Addr, sock.Port, "CLOSE"))
	}
	return text
}

func scrollingInfo(scrollingPercent float64, visibleContentPercent float64) string {
	return fmt.Sprintf("scrolling %3.f%% · showing %3.f%%", scrollingPercent, visibleContentPercent)
}
