package command

import "strings"

type KeyPress string

const (
	KeyEnter KeyPress = "enter"
	KeyQ     KeyPress = "q"
	KeyR     KeyPress = "r"
	KeyM     KeyPress = "m"
	KeyF     KeyPress = "f"
	KeyO     KeyPress = "o"
	KeyEsc   KeyPress = "esc"
	KeyDel   KeyPress = "delete"
	KeyS     KeyPress = "s"
	KeyCtrlC KeyPress = "ctrl+c"
	KeyUp    KeyPress = "up"
	KeyDown  KeyPress = "down"
)

func ToKeyPress(raw string) KeyPress {
	normalized := strings.ToLower(strings.TrimSpace(raw))
	return KeyPress(normalized)
}
