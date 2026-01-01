package processdetail

import (
	"fmt"
	"io"
	"strings"

	"charm.land/bubbles/v2/list"
	tea "charm.land/bubbletea/v2"
)

type commandListItemDelegate struct {
	styles *commandListStyles
}

func (d commandListItemDelegate) Height() int  { return 1 }
func (d commandListItemDelegate) Spacing() int { return 0 }
func (d commandListItemDelegate) Update(m tea.Msg, l *list.Model) tea.Cmd {
	return nil
}
func (d commandListItemDelegate) Render(w io.Writer, m list.Model, index int, listItem list.Item) {
	i, ok := listItem.(commandListItem)
	if !ok {
		return
	}

	str := fmt.Sprintf("%s", i)

	fn := d.styles.item.Render
	if index == m.Index() {
		fn = func(s ...string) string {
			return d.styles.selectedItem.Render("> " + strings.Join(s, " "))
		}
	}

	fmt.Fprint(w, fn(str))
}
