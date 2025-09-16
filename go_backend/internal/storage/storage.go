package storage

import (
	"fmt"
	"os"
	"path/filepath"

	"mix/internal/logging"

	"github.com/google/uuid"
)

const (
	// StorageRootDir is the centralized storage directory name
	StorageRootDir = "storage"
)

// Config holds storage configuration
type Config struct {
	// BasePath is the base directory for storage (defaults to ./storage)
	BasePath string
}

// DefaultConfig returns the default storage configuration
func DefaultConfig() Config {
	return Config{
		BasePath: StorageRootDir,
	}
}

// Initialize creates the storage root directory if it doesn't exist
func Initialize(config Config) error {
	if err := os.MkdirAll(config.BasePath, 0o755); err != nil {
		return fmt.Errorf("failed to create storage directory: %w", err)
	}
	
	logging.Info("Storage system initialized", "basePath", config.BasePath)
	return nil
}

// GetSessionStoragePath returns the storage path for a session
// Returns /storage/{session-id}/
func GetSessionStoragePath(sessionID string, config Config) string {
	return filepath.Join(config.BasePath, sessionID)
}

// ValidateSessionID validates that a session ID is a valid UUID format
func ValidateSessionID(sessionID string) bool {
	_, err := uuid.Parse(sessionID)
	return err == nil
}

// CreateSessionDirectory creates a storage directory for a session
func CreateSessionDirectory(sessionID string, config Config) error {
	if !ValidateSessionID(sessionID) {
		return fmt.Errorf("invalid session ID format: %s", sessionID)
	}
	
	sessionDir := GetSessionStoragePath(sessionID, config)
	
	// Create the session directory
	if err := os.MkdirAll(sessionDir, 0o755); err != nil {
		return fmt.Errorf("failed to create session directory: %w", err)
	}
	
	logging.Info("Created session storage directory", "sessionID", sessionID, "path", sessionDir)
	return nil
}

// GetSessionRoot returns an os.Root for a session directory
// This provides OS-level protection against path traversal attacks
func GetSessionRoot(sessionID string, config Config) (*os.Root, error) {
	if !ValidateSessionID(sessionID) {
		return nil, fmt.Errorf("invalid session ID format: %s", sessionID)
	}
	
	sessionDir := GetSessionStoragePath(sessionID, config)
	
	// Ensure session directory exists
	if _, err := os.Stat(sessionDir); os.IsNotExist(err) {
		return nil, fmt.Errorf("session directory does not exist: %s", sessionDir)
	}
	
	// Open root for session directory - provides path traversal protection
	root, err := os.OpenRoot(sessionDir)
	if err != nil {
		return nil, fmt.Errorf("failed to open session root: %w", err)
	}
	
	return root, nil
}

// GetSessionFilePath returns the full path to a file within a session
// DEPRECATED: Use GetSessionRoot and Root operations for better security
func GetSessionFilePath(sessionID, filename string, config Config) (string, error) {
	if !ValidateSessionID(sessionID) {
		return "", fmt.Errorf("invalid session ID format: %s", sessionID)
	}
	
	// Use os.Root for validation - this will fail if filename tries to escape
	root, err := GetSessionRoot(sessionID, config)
	if err != nil {
		return "", err
	}
	defer root.Close()
	
	// Test that we can stat the file path - this validates it's within root
	_, err = root.Stat(filename)
	if err != nil && !os.IsNotExist(err) {
		// If stat fails for reasons other than file not existing, it's likely a traversal attempt
		return "", fmt.Errorf("invalid file path: %s", filename)
	}
	
	sessionDir := GetSessionStoragePath(sessionID, config)
	return filepath.Join(sessionDir, filename), nil
}

// ListSessionFiles returns a list of files in a session's storage directory
func ListSessionFiles(sessionID string, config Config) ([]string, error) {
	if !ValidateSessionID(sessionID) {
		return nil, fmt.Errorf("invalid session ID format: %s", sessionID)
	}
	
	sessionDir := GetSessionStoragePath(sessionID, config)
	
	// Check if session directory exists
	if _, err := os.Stat(sessionDir); os.IsNotExist(err) {
		return []string{}, nil // Return empty list if directory doesn't exist
	}
	
	entries, err := os.ReadDir(sessionDir)
	if err != nil {
		return nil, fmt.Errorf("failed to read session directory: %w", err)
	}
	
	var files []string
	for _, entry := range entries {
		if !entry.IsDir() {
			files = append(files, entry.Name())
		}
	}
	
	return files, nil
}

// DeleteSessionDirectory removes a session's storage directory and all its contents
func DeleteSessionDirectory(sessionID string, config Config) error {
	if !ValidateSessionID(sessionID) {
		return fmt.Errorf("invalid session ID format: %s", sessionID)
	}
	
	sessionDir := GetSessionStoragePath(sessionID, config)
	
	// Check if directory exists
	if _, err := os.Stat(sessionDir); os.IsNotExist(err) {
		return nil // Nothing to delete
	}
	
	if err := os.RemoveAll(sessionDir); err != nil {
		return fmt.Errorf("failed to delete session directory: %w", err)
	}
	
	logging.Info("Deleted session storage directory", "sessionID", sessionID, "path", sessionDir)
	return nil
}