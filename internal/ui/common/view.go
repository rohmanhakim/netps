package common

import (
	"fmt"
	"strings"

	"charm.land/lipgloss/v2"
)

type ColorMode int
type ColorState int

const (
	ColorModeNeutral ColorMode = iota
	ColorModeSpecial
	ColorModeDanger
	ColorModeWarning
	ColorModeSuccess
)

const ActionBarItemSeparator = " · "

type Notification struct {
	ColorState ColorState
	Info       string
}

func ErrorPanel(theme Theme, width int, errorsStrings []string) string {
	headerString := fmt.Sprintf("%d error(s) found:", len(errorsStrings))
	header := NotificationBar(theme, ColorModeDanger, width, headerString)
	rendereds := []string{header}

	for i, e := range errorsStrings {
		rendereds = append(rendereds, NotificationBar(theme, ColorModeDanger, width, fmt.Sprintf("%d. %s", i+1, e)))
	}
	return strings.Join(rendereds, "\n")
}

func NotificationBar(theme Theme, colorMode ColorMode, width int, info string) string {
	var infoStyle lipgloss.Style
	switch colorMode {
	case ColorModeDanger:
		infoStyle = lipgloss.NewStyle().
			Width(width).
			Foreground(lipgloss.Color(theme.ColorForegroundBase)).
			Background(lipgloss.Color(theme.ColorDanger))
	case ColorModeSuccess:
		infoStyle = lipgloss.NewStyle().
			Width(width).
			Foreground(lipgloss.Color(theme.ColorForegroundBase)).
			Background(lipgloss.Color(theme.ColorSuccess))
	case ColorModeNeutral:
		infoStyle = lipgloss.NewStyle().
			Width(width).
			Foreground(lipgloss.Color(theme.ColorForegroundBase)).
			Background(lipgloss.Color(theme.ColorNeutral))
	case ColorModeWarning:
		infoStyle = lipgloss.NewStyle().
			Width(width).
			Foreground(lipgloss.Color(theme.ColorForegroundBase)).
			Background(lipgloss.Color(theme.ColorWarning))
	}

	infoText := infoStyle.Render(info)

	return lipgloss.JoinHorizontal(lipgloss.Center, infoText)
}

func ActionBar(windowWidth int, actions []string) string {
	style := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#FFF")) // ColorWhite

	out := style.
		Width(windowWidth).
		Render(strings.Join(actions, ActionBarItemSeparator))
	return out
}

func StatusBar(
	theme Theme,
	windowWidth int,
	modeName string,
	colorMode ColorMode,
	additionalInfo string,
	rightInfo string,
	rightInfoColorMode ColorMode,
) string {
	additionalInfoStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color(theme.ColorForegroundSecondary)).
		Background(lipgloss.Color(theme.ColorBackgroundSecondary)).
		PaddingRight(theme.SpacingSmall)

	modeStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color(theme.ColorForegroundBase)).
		PaddingLeft(theme.SpacingSmall).
		PaddingRight(theme.SpacingSmall).
		Align(lipgloss.Left)

	rightInfoStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color(theme.ColorForegroundBase)).
		PaddingLeft(theme.SpacingSmall).
		PaddingRight(theme.SpacingSmall).
		Align(lipgloss.Right)

	switch colorMode {
	case ColorModeSpecial:
		modeStyle = modeStyle.Background(lipgloss.Color(theme.ColorHighlight))
	case ColorModeWarning:
		modeStyle = modeStyle.Background(lipgloss.Color(theme.ColorWarning))
	default:
		modeStyle = modeStyle.Background(lipgloss.Color(theme.ColorHighlightSubtle))
	}

	switch rightInfoColorMode {
	case ColorModeSuccess:
		rightInfoStyle = rightInfoStyle.Background(lipgloss.Color(theme.ColorSuccess))
	case ColorModeWarning:
		rightInfoStyle = rightInfoStyle.Background(lipgloss.Color(theme.ColorWarning))
	default:
		rightInfoStyle = rightInfoStyle.Background(lipgloss.Color(theme.ColorNeutral))
	}

	modeNameText := modeStyle.Render(modeName)
	additionalInfoText := additionalInfoStyle.
		Render(additionalInfo)
	rightInfoText := rightInfoStyle.Render(rightInfo)
	spacerStyle := lipgloss.NewStyle().
		Background(lipgloss.Color(theme.ColorBackgroundSecondary))

	rightInfoWidth := 0
	if strings.TrimSpace(rightInfo) != "" {
		rightInfoWidth = lipgloss.Width(rightInfoText)
	}
	spacer := spacerStyle.
		Render(strings.Repeat(" ", max(0, windowWidth-lipgloss.Width(modeNameText)-lipgloss.Width(additionalInfoText)-rightInfoWidth)))
	return lipgloss.JoinHorizontal(lipgloss.Center, modeNameText, spacer, additionalInfoText, rightInfoText)
}

func CommandModal(text string) string {

	modalStyle := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("57")).
		PaddingLeft(1).
		PaddingRight(1).
		Align(lipgloss.Center, lipgloss.Center)

	return modalStyle.Render(text)
}
