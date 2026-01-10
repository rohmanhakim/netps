package sendsignal

import (
	"netps/internal/ui/common"

	"charm.land/bubbles/v2/list"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
)

type commandListItem string

func (i commandListItem) FilterValue() string { return "" }

type Model struct {
	List                list.Model
	SendSignalHelpItems []string
	CommandListItems    []list.Item
	Modal               string
}

func New() Model {
	return Model{
		List: list.New([]list.Item{}, commandListItemDelegate{}, 25, 6),
		SendSignalHelpItems: []string{
			"[↑↓] scroll",
			"[enter] send",
			"[esc] back",
			"[q] quit",
		},
		CommandListItems: []list.Item{
			commandListItem("SIGTERM (15) · graceful termination"),
			commandListItem("SIGKILL (9) · immediate termination"),
			commandListItem("SIGINT (2) · interrupt"),
			commandListItem("SIGHUP (1) · reload / restart hint"),
		},
	}
}

func (m *Model) Initialize() {

	const defaultWidth = 25
	const listHeight = 6

	l := list.New(m.CommandListItems, commandListItemDelegate{}, defaultWidth, listHeight)
	l.Title = "Send Signal to Process"
	l.SetShowStatusBar(false)
	l.SetFilteringEnabled(false)
	l.SetShowPagination(false)
	l.DisableQuitKeybindings()
	l.SetShowHelp(false)

	m.List = l
	m.updateStyles()
}

func (m Model) Init() tea.Cmd { return nil }

func (m Model) Update(msg tea.Msg) (Model, tea.Cmd) {
	var cmd tea.Cmd
	m.List, cmd = m.List.Update(msg)
	m.Modal = common.CommandModal(m.List.View())
	return m, cmd
}

func (m Model) View() tea.View {
	var v tea.View
	v.SetContent(m.Modal)
	return v
}

func (m *Model) updateStyles() {
	var s commandListStyles
	s.title = lipgloss.NewStyle().Foreground(lipgloss.Color("255"))               // ColorWhite
	s.item = lipgloss.NewStyle().PaddingLeft(2).Foreground(lipgloss.Color("255")) // ColorWhite
	s.selectedItem = lipgloss.NewStyle().Foreground(lipgloss.Color("57"))         // ColorAccent

	m.List.Styles.Title = s.title
	m.List.SetDelegate(commandListItemDelegate{styles: &s})
}
