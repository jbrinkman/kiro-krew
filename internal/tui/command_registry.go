package tui

import (
	"strings"
)

// Command represents a command with its metadata for autocomplete
type Command struct {
	Name        string   // Primary command name
	Description string   // Help text
	Subcommands []string // Available subcommands
	HasArgs     bool     // Whether command accepts arguments
	ArgPattern  string   // Pattern for arguments (e.g., "<issue>", "[desc]")
}

// CommandRegistry manages all available commands for autocomplete
type CommandRegistry struct {
	commands map[string]*Command
}

// NewCommandRegistry creates a new command registry with all REPL commands
func NewCommandRegistry() *CommandRegistry {
	registry := &CommandRegistry{
		commands: make(map[string]*Command),
	}

	// Register all commands
	registry.register(&Command{
		Name:        "watch",
		Description: "Start or stop the issue watcher",
		Subcommands: []string{"start", "stop"},
	})

	registry.register(&Command{
		Name:        "status",
		Description: "Show system status and running agents",
	})

	registry.register(&Command{
		Name:       "stop",
		Description: "Stop agent for specific issue number",
		HasArgs:    true,
		ArgPattern: "<issue>",
	})

	registry.register(&Command{
		Name:       "plan",
		Description: "Start interactive planning session",
		HasArgs:    true,
		ArgPattern: "[desc]",
	})

	registry.register(&Command{
		Name:       "theme",
		Description: "Show current theme or switch to new theme",
		HasArgs:    true,
		ArgPattern: "[name]",
	})

	registry.register(&Command{
		Name:        "about",
		Description: "Show version information and check for updates",
	})

	registry.register(&Command{
		Name:        "exit",
		Description: "Exit the application",
	})

	registry.register(&Command{
		Name:        "help",
		Description: "Show available commands",
	})

	registry.register(&Command{
		Name:        "logs",
		Description: "View incident logs",
	})

	return registry
}

// register adds a command to the registry
func (r *CommandRegistry) register(cmd *Command) {
	r.commands[cmd.Name] = cmd
}

// GetCommand returns a command by name
func (r *CommandRegistry) GetCommand(name string) (*Command, bool) {
	cmd, exists := r.commands[name]
	return cmd, exists
}

// GetAllCommands returns all registered commands
func (r *CommandRegistry) GetAllCommands() []*Command {
	commands := make([]*Command, 0, len(r.commands))
	for _, cmd := range r.commands {
		commands = append(commands, cmd)
	}
	return commands
}

// FilterCommands returns commands that match the input prefix
func (r *CommandRegistry) FilterCommands(input string) []*Command {
	if input == "" {
		return r.GetAllCommands()
	}

	parts := strings.Fields(input)
	if len(parts) == 0 {
		return r.GetAllCommands()
	}

	prefix := strings.ToLower(parts[0])
	matches := make([]*Command, 0)

	for _, cmd := range r.commands {
		if strings.HasPrefix(strings.ToLower(cmd.Name), prefix) {
			matches = append(matches, cmd)
		}
	}

	return matches
}

// GetSubcommands returns subcommands for a given command and input
func (r *CommandRegistry) GetSubcommands(input string) []string {
	parts := strings.Fields(input)
	if len(parts) < 1 {
		return nil
	}

	cmdName := strings.ToLower(parts[0])
	cmd, exists := r.commands[cmdName]
	if !exists || len(cmd.Subcommands) == 0 {
		return nil
	}

	// If we have exactly one part, return all subcommands
	if len(parts) == 1 {
		return cmd.Subcommands
	}

	// If we have two parts, filter subcommands by prefix
	if len(parts) == 2 {
		subPrefix := strings.ToLower(parts[1])
		matches := make([]string, 0)
		for _, sub := range cmd.Subcommands {
			if strings.HasPrefix(strings.ToLower(sub), subPrefix) {
				matches = append(matches, sub)
			}
		}
		return matches
	}

	return nil
}

// GetBestMatch returns the best completion match for the input
func (r *CommandRegistry) GetBestMatch(input string) string {
	if input == "" {
		return ""
	}

	parts := strings.Fields(input)
	if len(parts) == 0 {
		return ""
	}

	// Check for command match
	commands := r.FilterCommands(input)
	if len(commands) > 0 {
		cmdName := commands[0].Name
		
		// If input is just the command name, return it
		if len(parts) == 1 {
			return cmdName
		}

		// Check for subcommand match
		subcommands := r.GetSubcommands(input)
		if len(subcommands) > 0 {
			return cmdName + " " + subcommands[0]
		}

		return cmdName
	}

	return ""
}

// IsValidCommand checks if the input represents a valid command
func (r *CommandRegistry) IsValidCommand(input string) bool {
	input = strings.TrimSpace(input)
	if input == "" {
		return false
	}

	parts := strings.Fields(input)
	if len(parts) == 0 {
		return false
	}

	cmdName := strings.ToLower(parts[0])
	cmd, exists := r.commands[cmdName]
	if !exists {
		return false
	}

	// Check subcommand if present
	if len(parts) > 1 && len(cmd.Subcommands) > 0 {
		subCmd := strings.ToLower(parts[1])
		for _, validSub := range cmd.Subcommands {
			if strings.ToLower(validSub) == subCmd {
				return true
			}
		}
		return false
	}

	return true
}
