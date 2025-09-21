package commands

import (
	"strings"
)

// ParsedCommand represents a parsed slash command
type ParsedCommand struct {
	Name      string
	Arguments string
	RawInput  string
}

// ParseCommand parses a slash command input
func ParseCommand(input string) (*ParsedCommand, error) {
	// Remove leading slash
	if !strings.HasPrefix(input, "/") {
		return nil, ErrNotSlashCommand
	}

	content := strings.TrimPrefix(input, "/")
	// Allow empty content for showing all commands
	if content == "" {
		return &ParsedCommand{
			Name:      "",
			Arguments: "",
			RawInput:  input,
		}, nil
	}

	// Split command and arguments
	parts := strings.SplitN(content, " ", 2)
	name := parts[0]

	var arguments string
	if len(parts) > 1 {
		arguments = strings.TrimSpace(parts[1])
	}

	return &ParsedCommand{
		Name:      name,
		Arguments: arguments,
		RawInput:  input,
	}, nil
}

// IsSlashCommand checks if input is a slash command
func IsSlashCommand(input string) bool {
	return strings.HasPrefix(strings.TrimSpace(input), "/")
}
