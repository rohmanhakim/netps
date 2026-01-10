package ui

import (
	"netps/internal/ui/common"
	"netps/internal/ui/message"
	"netps/internal/ui/processdetail"
	"netps/internal/ui/processlist"

	tea "charm.land/bubbletea/v2"
)

type Screen int

const (
	ScreenProcessList Screen = iota
	ScreenProcessDetail
)

type Root struct {
	theme         common.Theme
	screen        Screen
	width, height int
	processList   processlist.Model
	processDetail processdetail.Model
}

func New() Root {
	theme := common.Theme{
		ColorForegroundBase:      common.ColorWhite,
		ColorForegroundSubtle:    common.ColorDarkGray,
		ColorBackgroundSecondary: common.ColorDarkOlive,
		ColorForegroundSecondary: common.ColorArgent,
		ColorAccent:              common.ColorElectricIndigo,
		ColorHighlight:           common.ColorElectricIndigo,
		ColorHighlightSubtle:     common.ColorBoulder,
		ColorInactive:            common.ColorDarkCharcoal,
		ColorSuccess:             common.ColorOfficeGreen,
		ColorNeutral:             common.ColorBlueJeans,
		ColorDanger:              common.ColorBrightRed,
		ColorWarning:             common.ColorGoldenRod,

		SpacingSmall:  1,
		SpacingMedium: 2,
	}
	return Root{
		theme:         theme,
		screen:        ScreenProcessList,
		processList:   processlist.New(theme),
		processDetail: processdetail.New(theme),
	}
}

func (m Root) Init() tea.Cmd {
	return func() tea.Msg {
		return message.GoBack{}
	}
}

func (m Root) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
	case message.GoToProcessDetail:
		m.screen = ScreenProcessDetail
		return m, m.processDetail.Init(msg.PID, msg.Name, m.width, m.height)
	case message.GoBack:
		m.screen = ScreenProcessList
		return m, m.processList.Init(m.width, m.height)
	}

	// delegate update
	switch m.screen {
	case ScreenProcessList:
		var cmd tea.Cmd
		pl, cmd := m.processList.Update(msg)
		m.processList = pl
		return m, cmd

	case ScreenProcessDetail:
		var cmd tea.Cmd
		pd, cmd := m.processDetail.Update(msg)
		m.processDetail = pd
		return m, cmd
	}

	return m, nil
}

func (m Root) View() tea.View {
	switch m.screen {
	case ScreenProcessList:
		return m.processList.View()

	case ScreenProcessDetail:
		return m.processDetail.View()
	}
	return tea.View{}
}
