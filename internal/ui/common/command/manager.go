package command

import (
	"fmt"
	"sort"
	"strings"
)

type Command string
type CommandSource int
type Context string

const (
	CommandBack           Command = "Back"
	CommandDismiss        Command = "Dismiss"
	CommandExecute        Command = "Execute"
	CommandFilter         Command = "Filter"
	CommandInspect        Command = "Inspect"
	CommandMove           Command = "Move"
	CommandMultipleSelect Command = "Mult. Select"
	CommandOrder          Command = "Order"
	CommandQuit           Command = "Quit"
	CommandRetry          Command = "Retry"
	CommandScroll         Command = "Scroll"
	CommandSelect         Command = "Select"
	CommandSendSignal     Command = "Send Signal"
	CommandUnknown        Command = "Unknown"
)

const (
	ContextUnknown             Context = "UnknownContext"
	ContextProcessListScreen   Context = "ProcessListScreen"
	ContextProcessDetailScreen Context = "ProcessDetailScreen"
	ContextHydrating           Context = "Hydrating"
	ContextHydrationError      Context = "HydrationError"
	ContextSendSignal          Context = "SendSignal"
)

const (
	SourceGlobal CommandSource = iota
	SourceContext
)

type CommandData struct {
	KeyPresses  []KeyPress
	Description string
}

type KeyDisplay struct {
	Raw     KeyPress
	Symbol  string
	Verbose string
}

type AvailableCommand struct {
	Command     Command
	Source      CommandSource
	KeyPress    KeyPress
	Description string
}

var keyDisplayMap = map[KeyPress]KeyDisplay{
	KeyCtrlC: {
		Raw:     KeyCtrlC,
		Symbol:  "^c",
		Verbose: "Ctrl+C",
	},
	KeyUp: {
		Raw:     KeyUp,
		Symbol:  "↑",
		Verbose: "Up Arrow",
	},
	KeyDown: {
		Raw:     KeyDown,
		Symbol:  "↓",
		Verbose: "Down Arrow",
	},
	KeyEnter: {
		Raw:     KeyEnter,
		Symbol:  "⏎",
		Verbose: "Enter",
	},
}

func (cd *CommandData) Symbol() string {
	symbols := []string{}
	for _, k := range cd.KeyPresses {
		if display, ok := keyDisplayMap[k]; ok {
			symbols = append(symbols, display.Symbol)
		} else {
			symbols = append(symbols, string(k))
		}
	}
	return strings.Join(symbols, "/")
}

func (cd *CommandData) VerboseSymbol() string {
	symbols := []string{}
	for _, k := range cd.KeyPresses {
		if display, ok := keyDisplayMap[k]; ok {
			symbols = append(symbols, display.Verbose)
		} else {
			symbols = append(symbols, string(k))
		}
	}
	return strings.Join(symbols, " or ")
}

type Manager struct {
	globalCommands  map[KeyPress]Command // Sacred globals
	contextCommands map[Context]map[KeyPress]Command
	data            map[Command]CommandData
	currentContext  Context

	cachedHelp    []string
	cachedContext Context
}

func NewManager() Manager {
	return Manager{
		globalCommands:  map[KeyPress]Command{},
		contextCommands: map[Context]map[KeyPress]Command{},
		currentContext:  ContextUnknown,
		data: map[Command]CommandData{
			CommandMove: {
				KeyPresses:  []KeyPress{KeyUp, KeyDown},
				Description: "Move cursor up/down",
			},
			CommandScroll: {
				KeyPresses:  []KeyPress{KeyUp, KeyDown},
				Description: "Scroll up/down",
			},
			CommandMultipleSelect: {
				KeyPresses:  []KeyPress{KeyM},
				Description: "Select multiple items",
			},
			CommandSelect: {
				KeyPresses:  []KeyPress{KeyEnter},
				Description: "Select item",
			},
			CommandInspect: {
				KeyPresses:  []KeyPress{KeyEnter},
				Description: "Inspect item",
			},
			CommandExecute: {
				KeyPresses:  []KeyPress{KeyEnter},
				Description: "Execute selected",
			},
			CommandSendSignal: {
				KeyPresses:  []KeyPress{KeyS},
				Description: "Send signal to item",
			},
			CommandFilter: {
				KeyPresses:  []KeyPress{KeyF},
				Description: "Filter items",
			},
			CommandOrder: {
				KeyPresses:  []KeyPress{KeyO},
				Description: "Order items",
			},
			CommandBack: {
				KeyPresses:  []KeyPress{KeyEsc},
				Description: "Go back",
			},
			CommandRetry: {
				KeyPresses:  []KeyPress{KeyR},
				Description: "Retry",
			},
			CommandDismiss: {
				KeyPresses:  []KeyPress{KeyDel},
				Description: "Dismiss",
			},
			CommandQuit: {
				KeyPresses:  []KeyPress{KeyQ, KeyCtrlC},
				Description: "Quit application",
			},
		},
	}
}

func (m *Manager) RegisterGlobalCommand(key KeyPress, command Command) error {
	if _, exists := m.data[command]; !exists {
		return fmt.Errorf("command %s not found in data", command)
	}
	m.globalCommands[key] = command
	return nil
}

func (m *Manager) RegisterContextCommand(ctx Context, key KeyPress, command Command) error {
	if _, exists := m.data[command]; !exists {
		return fmt.Errorf("command %s not found in data", command)
	}

	if _, isGlobal := m.globalCommands[key]; isGlobal {
		return fmt.Errorf("cannot override global key %s", key)
	}

	if m.contextCommands[ctx] == nil {
		m.contextCommands[ctx] = make(map[KeyPress]Command)
	}
	m.contextCommands[ctx][key] = command
	return nil
}

func (m *Manager) UnregisterGlobalCommand(key KeyPress) {
	delete(m.globalCommands, key)
}

func (m *Manager) UnregisterContextCommand(ctx Context, key KeyPress) {
	if contextMap, ok := m.contextCommands[ctx]; ok {
		delete(contextMap, key)
	}
}

func (m *Manager) GetCommand(key KeyPress) Command {
	if cmd, ok := m.globalCommands[key]; ok {
		return cmd
	}

	if contextMap, ok := m.contextCommands[m.currentContext]; ok {
		if cmd, ok := contextMap[key]; ok {
			return cmd
		}
	}

	return CommandUnknown
}

func (m *Manager) GetCommandData(cmd Command) (CommandData, bool) {
	data, ok := m.data[cmd]
	return data, ok
}

func (m *Manager) SetContext(ctx Context) error {
	if ctx == ContextUnknown {
		return fmt.Errorf("cannot set unknown context")
	}

	// Invalidate cache when context changes
	if m.currentContext != ctx {
		m.cachedHelp = nil
	}

	m.currentContext = ctx
	return nil
}

func (m *Manager) GetAvailableCommands() []AvailableCommand {
	result := []AvailableCommand{}
	seen := make(map[Command]bool)

	// Add global commands
	for key, cmd := range m.globalCommands {
		if !seen[cmd] {
			data, _ := m.data[cmd]
			result = append(result, AvailableCommand{
				Command:     cmd,
				Source:      SourceGlobal,
				KeyPress:    key,
				Description: data.Description,
			})
			seen[cmd] = true
		}
	}

	// Add context commands
	if contextMap, ok := m.contextCommands[m.currentContext]; ok {
		for key, cmd := range contextMap {
			if !seen[cmd] {
				data, _ := m.data[cmd]
				result = append(result, AvailableCommand{
					Command:     cmd,
					Source:      SourceContext,
					KeyPress:    key,
					Description: data.Description,
				})
				seen[cmd] = true
			}
		}
	}

	return result
}

func (m *Manager) IsCommandAvailable(cmd Command) bool {
	for _, globalCmd := range m.globalCommands {
		if globalCmd == cmd {
			return true
		}
	}

	if contextMap, ok := m.contextCommands[m.currentContext]; ok {
		for _, contextCmd := range contextMap {
			if contextCmd == cmd {
				return true
			}
		}
	}

	return false
}

func (m *Manager) GetContext() Context {
	return m.currentContext
}

// GenerateContextHelp generates formatted help text showing available commands
// grouped by global vs context-specific
func (m *Manager) GenerateContextHelp() []string {
	// Return cached version if context hasn't changed
	if m.cachedContext == m.currentContext && m.cachedHelp != nil {
		return m.cachedHelp
	}

	help := []string{}
	commands := m.GetAvailableCommands()

	// Group by source
	globalCmds := []AvailableCommand{}
	contextCmds := []AvailableCommand{}

	for _, cmd := range commands {
		if cmd.Source == SourceGlobal {
			globalCmds = append(globalCmds, cmd)
		} else {
			contextCmds = append(contextCmds, cmd)
		}
	}

	// Phase 2: Sort each group alphabetically
	sort.Slice(globalCmds, func(i, j int) bool {
		return globalCmds[i].Command < globalCmds[j].Command // Global: A-Z
	})
	sort.Slice(contextCmds, func(i, j int) bool {
		return contextCmds[i].Command < contextCmds[j].Command // Contextual: A-Z
	})

	// Format global commands
	if len(globalCmds) > 0 {
		for _, cmd := range globalCmds {
			if data, ok := m.data[cmd.Command]; ok {
				help = append(help, fmt.Sprintf("[%s] %s", data.Symbol(), cmd.Command))
			}
		}
	}

	// Format context commands
	if len(contextCmds) > 0 {
		for _, cmd := range contextCmds {
			if data, ok := m.data[cmd.Command]; ok {
				help = append(help, fmt.Sprintf("[%s] %s", data.Symbol(), cmd.Command))
			}
		}
	}

	// Cache result
	m.cachedHelp = help
	m.cachedContext = m.currentContext

	return help
}
