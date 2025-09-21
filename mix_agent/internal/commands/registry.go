package commands

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"mix/internal/app"
)

// Registry manages all available commands
type Registry struct {
	commands map[string]Command
}

// NewRegistry creates a new command registry
func NewRegistry() *Registry {
	return &Registry{
		commands: make(map[string]Command),
	}
}

// LoadCommands loads all commands (builtin and file-based)
func (r *Registry) LoadCommands(app *app.App) error {
	// Load builtin commands
	builtins := GetBuiltinCommands(r, app)
	for name, cmd := range builtins {
		r.commands[name] = cmd
	}

	// Load project commands from .mix/commands/
	projectDir := ".mix/commands"
	if err := r.loadCommandsFromDir(projectDir, "project"); err != nil {
		return fmt.Errorf("failed to load project commands: %w", err)
	}

	// Load user commands from ~/.mix/commands/
	homeDir, err := os.UserHomeDir()
	if err == nil {
		userDir := filepath.Join(homeDir, ".mix", "commands")
		if err := r.loadCommandsFromDir(userDir, "user"); err != nil {
			return fmt.Errorf("failed to load user commands: %w", err)
		}
	}

	return nil
}

func (r *Registry) loadCommandsFromDir(dir, scope string) error {
	commands, err := LoadCommandsFromDirectory(dir)
	if err != nil {
		return err
	}

	// Add scope prefix to command names to avoid conflicts
	for name, cmd := range commands {
		prefixedName := fmt.Sprintf("%s:%s", scope, name)
		r.commands[prefixedName] = cmd

		// Also register without prefix for convenience (last one wins)
		r.commands[name] = cmd
	}

	return nil
}

// GetCommand retrieves a command by name
func (r *Registry) GetCommand(name string) (Command, bool) {
	cmd, exists := r.commands[name]
	return cmd, exists
}

// GetAllCommands returns all registered commands
func (r *Registry) GetAllCommands() map[string]Command {
	result := make(map[string]Command)
	for name, cmd := range r.commands {
		result[name] = cmd
	}
	return result
}

// ExecuteCommand executes a command by name with arguments
func (r *Registry) ExecuteCommand(ctx context.Context, name, args string) (string, error) {
	cmd, exists := r.GetCommand(name)
	if !exists {
		return "", fmt.Errorf("%w: %s", ErrCommandNotFound, name)
	}

	result, err := cmd.Execute(ctx, args)
	if err != nil {
		return "", fmt.Errorf("%w: %v", ErrCommandFailed, err)
	}

	return result, nil
}
