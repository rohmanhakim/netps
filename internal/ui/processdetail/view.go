package processdetail

import (
	"strings"

	"charm.land/lipgloss/v2"
)

func list(name string, values []string) string {
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

func renderActionBar(windowWidth int) string {

	statusBarStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#C1C6B2")).
		Background(lipgloss.Color("#353533"))
	statusText := lipgloss.NewStyle().Inherit(statusBarStyle)

	statusVal := statusText.
		Width(windowWidth).
		Render("[s] send signal · [c] copy · [esc] back")
	return statusVal
}
