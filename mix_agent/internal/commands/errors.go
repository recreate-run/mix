package commands

import "errors"

var (
	ErrNotSlashCommand = errors.New("input is not a slash command")
	ErrEmptyCommand    = errors.New("command cannot be empty")
	ErrCommandNotFound = errors.New("command not found")
	ErrCommandFailed   = errors.New("command execution failed")
)
