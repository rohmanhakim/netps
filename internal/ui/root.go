package ui

import (
	"log"
	"netps/internal/ui/common"
	"netps/internal/ui/common/command"
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
	theme          common.Theme
	commandManager command.Manager
	screen         Screen
	width, height  int
	processList    processlist.Model
	processDetail  processdetail.Model
}

func New() (Root, error) {
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

	manager := command.NewManager()
	err := manager.RegisterGlobalCommand(command.KeyQ, command.CommandQuit)
	if err != nil {
		return Root{}, err
	}
	err = manager.RegisterGlobalCommand(command.KeyCtrlC, command.CommandQuit)
	if err != nil {
		return Root{}, err
	}
	err = manager.RegisterGlobalCommand(command.KeyEsc, command.CommandBack)
	if err != nil {
		return Root{}, err
	}

	processlist, err := processlist.New(theme, &manager)
	if err != nil {
		log.Fatalf("Root error at New creating processlist: %v", err)
	}

	processdetail, err := processdetail.New(theme, &manager)
	if err != nil {
		log.Fatalf("Root error at New creating processdetail: %v", err)
	}

	return Root{
		theme:          theme,
		commandManager: manager,
		screen:         ScreenProcessList,
		processList:    processlist,
		processDetail:  processdetail,
	}, nil
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
