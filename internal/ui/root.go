package ui

import (
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
	screen        Screen
	processList   processlist.Model
	processDetail processdetail.Model
}

func New() Root {
	return Root{
		screen:        ScreenProcessList,
		processList:   processlist.New(),
		processDetail: processdetail.New(),
	}
}

func (m Root) Init() tea.Cmd { return nil }

func (m Root) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {

	case message.GoToProcessDetail:
		m.processDetail = processdetail.Model{
			PID:  msg.PID,
			Name: msg.Name,
		}
		m.screen = ScreenProcessDetail
		return m, tea.Batch(
			processdetail.HydrateStaticIds(msg.PID),
			processdetail.HydrateResource(msg.PID),
		)

	case message.GoBack:
		m.screen = ScreenProcessList
		return m, nil
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
