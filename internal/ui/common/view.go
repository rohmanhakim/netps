package common

import (
	"fmt"
	"strings"

	"charm.land/lipgloss/v2"
)

func ActionBar(windowWidth int, actions []string) string {
	style := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#FFF"))

	out := style.
		Width(windowWidth).
		Render(strings.Join(actions, " · "))
		// Render("[↑↓] scroll · [s] send signal · [c] copy · [esc] back · [q] quit")
	return out
}

func StatusBar(windowWidth int, modeName string, modeColor string, scrollPercent float64) string {
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
