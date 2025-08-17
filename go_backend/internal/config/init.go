package config

import (
	"fmt"
	"os"
	"path/filepath"
)

const (
	// InitFlagFilename is the name of the file that indicates whether the project has been initialized
	InitFlagFilename = "init"
)

// ProjectInitFlag represents the initialization status for a project directory
type ProjectInitFlag struct {
	Initialized bool `json:"initialized"`
}

// ShouldShowInitDialog checks if the initialization dialog should be shown for the current directory
func ShouldShowInitDialog() (bool, error) {
	if cfg == nil {
		return false, fmt.Errorf("config not loaded")
	}

	// Create the flag file path
	flagFilePath := filepath.Join(cfg.Data.Directory, InitFlagFilename)

	// Check if the flag file exists
	_, err := os.Stat(flagFilePath)
	if err == nil {
		// File exists, don't show the dialog
		return false, nil
	}

	// If the error is not "file not found", return the error
	if !os.IsNotExist(err) {
		return false, fmt.Errorf("failed to check init flag file: %w", err)
	}

	// File doesn't exist, show the dialog
	return true, nil
}

// MarkProjectInitialized marks the current project as initialized
func MarkProjectInitialized() error {
	if cfg == nil {
		return fmt.Errorf("config not loaded")
	}
	// Create the flag file path
	flagFilePath := filepath.Join(cfg.Data.Directory, InitFlagFilename)

	// Create an empty file to mark the project as initialized
	file, err := os.Create(flagFilePath)
	if err != nil {
		return fmt.Errorf("failed to create init flag file: %w", err)
	}
	defer file.Close()

	return nil
}

// EnsurePromptsDirectory ensures the prompts directory exists with tools subdirectory
func EnsurePromptsDirectory() error {
	if cfg == nil {
		return fmt.Errorf("config not loaded")
	}

	promptsDir := cfg.PromptsDir

	// Create prompts directory if it doesn't exist
	if err := os.MkdirAll(promptsDir, 0755); err != nil {
		return fmt.Errorf("failed to create prompts directory %s: %w", promptsDir, err)
	}
	
	// Create tools subdirectory
	toolsDir := filepath.Join(promptsDir, "tools")
	if err := os.MkdirAll(toolsDir, 0755); err != nil {
		return fmt.Errorf("failed to create tools directory %s: %w", toolsDir, err)
	}

	return nil
}
