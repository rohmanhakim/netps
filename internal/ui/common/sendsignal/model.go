package sendsignal

import (
	"fmt"
	"netps/internal/ui/common"

	"charm.land/bubbles/v2/list"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
)

type commandListItem string

func (i commandListItem) FilterValue() string { return "" }

type Model struct {
	processPID       int
	processName      string
	theme            common.Theme
	list             list.Model
	commandListItems []list.Item
	modal            string
}

func New(theme common.Theme) Model {
	return Model{
		theme: theme,
		list:  list.New([]list.Item{}, commandListItemDelegate{}, 25, 6),
		commandListItems: []list.Item{
			commandListItem("SIGTERM (15) · graceful termination"),
			commandListItem("SIGKILL (9) · immediate termination"),
			commandListItem("SIGINT (2) · interrupt"),
			commandListItem("SIGHUP (1) · reload / restart hint"),
		},
	}
}

func (m *Model) Initialize() {

	const defaultWidth = 50
	const listHeight = 6

	m.processPID = -1

	l := list.New(m.commandListItems, commandListItemDelegate{}, defaultWidth, listHeight)
	l.Title = fmt.Sprintf("Send Signal to: %s (%d)", m.processName, m.processPID)
	l.SetShowStatusBar(false)
	l.SetFilteringEnabled(false)
	l.SetShowPagination(false)
	l.DisableQuitKeybindings()
	l.SetShowHelp(false)

	m.list = l
	m.updateStyles()
	m.modal = common.CommandModal(m.list.View())
}

func (m Model) Init() tea.Cmd { return nil }

func (m Model) Update(msg tea.Msg) (Model, tea.Cmd) {
	var cmd tea.Cmd

	m.list, cmd = m.list.Update(msg)
	m.list.Title = fmt.Sprintf("Send Signal to %s (%d)", m.processName, m.processPID)
	m.modal = common.CommandModal(m.list.View())
	return m, cmd
}

func (m Model) View() tea.View {
	var v tea.View
	v.SetContent(m.modal)
	return v
}

func (m *Model) updateStyles() {
	var s commandListStyles
	s.title = lipgloss.NewStyle().Foreground(lipgloss.Color(m.theme.ColorForegroundBase))               // ColorWhite
	s.item = lipgloss.NewStyle().PaddingLeft(2).Foreground(lipgloss.Color(m.theme.ColorForegroundBase)) // ColorWhite
	s.selectedItem = lipgloss.NewStyle().Foreground(lipgloss.Color(m.theme.ColorHighlight))             // ColorAccent

	m.list.Styles.Title = s.title
	m.list.SetDelegate(commandListItemDelegate{styles: &s})
}

func (m *Model) ModalWidth() int {
	return lipgloss.Width(m.modal)
}

func (m *Model) ModalHeight() int {
	return lipgloss.Height(m.modal)
}

func (m *Model) SetProcessInfo(pid int, name string) {
	m.processPID = pid
	m.processName = name
}
