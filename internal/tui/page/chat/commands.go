package chat

import (
	"strings"

	tea "charm.land/bubbletea/v2"

	"github.com/guilhermegouw/cdd/internal/tui/util"
)

// Command message types.
type (
	// OpenModelsModalMsg requests opening the models/connections modal.
	OpenModelsModalMsg struct{}

	// CloseModelsModalMsg requests closing the models/connections modal.
	CloseModelsModalMsg struct{}

	// UnknownCommandMsg indicates an unknown slash command was entered.
	UnknownCommandMsg struct {
		Command string
	}
)

// Command represents a slash command.
type Command struct {
	Name        string
	Description string
	Handler     func(args []string) tea.Msg
}

// CommandRegistry holds registered slash commands.
type CommandRegistry struct {
	commands map[string]Command
}

// NewCommandRegistry creates a new command registry with default commands.
func NewCommandRegistry() *CommandRegistry {
	r := &CommandRegistry{
		commands: make(map[string]Command),
	}

	// Register default commands.
	r.Register(Command{
		Name:        "models",
		Description: "Manage API connections and model selection",
		Handler:     func(args []string) tea.Msg { return OpenModelsModalMsg{} },
	})

	return r
}

// Register adds a command to the registry.
func (r *CommandRegistry) Register(cmd Command) {
	r.commands[cmd.Name] = cmd
}

// Parse attempts to parse input as a slash command.
// Returns the command message and true if it's a command, nil and false otherwise.
func (r *CommandRegistry) Parse(input string) (tea.Msg, bool) {
	input = strings.TrimSpace(input)
	if !strings.HasPrefix(input, "/") {
		return nil, false
	}

	// Split command and args.
	parts := strings.Fields(input[1:]) // Remove leading "/"
	if len(parts) == 0 {
		return nil, false
	}

	cmdName := strings.ToLower(parts[0])
	args := parts[1:]

	// Look up command.
	cmd, ok := r.commands[cmdName]
	if !ok {
		return UnknownCommandMsg{Command: cmdName}, true
	}

	return cmd.Handler(args), true
}

// GetCommands returns all registered commands.
func (r *CommandRegistry) GetCommands() []Command {
	cmds := make([]Command, 0, len(r.commands))
	for _, cmd := range r.commands {
		cmds = append(cmds, cmd)
	}
	return cmds
}

// parseCommand is a helper method for the chat Model.
// Returns a tea.Cmd if the input is a command, nil otherwise.
func (m *Model) parseCommand(input string) tea.Cmd {
	if m.commandRegistry == nil {
		m.commandRegistry = NewCommandRegistry()
	}

	msg, isCmd := m.commandRegistry.Parse(input)
	if !isCmd {
		return nil
	}

	return util.CmdHandler(msg)
}
