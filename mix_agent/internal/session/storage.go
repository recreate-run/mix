package session

import (
	"fmt"
	"os"
	"path/filepath"

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

// DefaultConfig returns the default storage configuration with absolute paths
func DefaultConfig() Config {
	// Convert relative path to absolute to prevent working directory issues
	absPath, err := filepath.Abs(StorageRootDir)
	if err != nil {
		panic(fmt.Sprintf("failed to resolve absolute storage path: %v", err))
	}
	return Config{
		BasePath: absPath,
	}
}

// Initialize creates the storage root directory and uploads directory if they don't exist
func Initialize(config Config) error {
	if err := os.MkdirAll(config.BasePath, 0o755); err != nil {
		return fmt.Errorf("failed to create storage directory: %w", err)
	}

	// Create uploads directory
	uploadsPath := GetUploadsStoragePath(config)
	if err := os.MkdirAll(uploadsPath, 0o755); err != nil {
		return fmt.Errorf("failed to create uploads directory: %w", err)
	}

	return nil
}

// GetSessionStoragePath returns the storage path for a session
// Returns /storage/{session-id}/
func GetSessionStoragePath(sessionID string, config Config) string {
	return filepath.Join(config.BasePath, sessionID)
}

// GetUploadsStoragePath returns the storage path for uploads
// Returns /storage/uploads/
func GetUploadsStoragePath(config Config) string {
	return filepath.Join(config.BasePath, "uploads")
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
	return nil
}

// GetSessionRoot returns an os.Root for a session directory
// This provides OS-level protection against path traversal attacks
func GetSessionRoot(sessionID string, config Config) (*os.Root, error) {
	if !ValidateSessionID(sessionID) {
		return nil, fmt.Errorf("invalid session ID format: %s", sessionID)
	}

	sessionDir := GetSessionStoragePath(sessionID, config)

	// Ensure session directory exists - create it if it doesn't
	if _, err := os.Stat(sessionDir); os.IsNotExist(err) {
		if err := os.MkdirAll(sessionDir, 0o755); err != nil {
			return nil, fmt.Errorf("failed to create session directory: %w", err)
		}
	}

	// Open root for session directory - provides path traversal protection
	root, err := os.OpenRoot(sessionDir)
	if err != nil {
		return nil, fmt.Errorf("failed to open session root: %w", err)
	}

	return root, nil
}

// GetUploadsRoot returns an os.Root for the uploads directory
// This provides OS-level protection against path traversal attacks
func GetUploadsRoot(config Config) (*os.Root, error) {
	uploadsDir := GetUploadsStoragePath(config)

	// Ensure uploads directory exists - create it if it doesn't
	if _, err := os.Stat(uploadsDir); os.IsNotExist(err) {
		if err := os.MkdirAll(uploadsDir, 0o755); err != nil {
			return nil, fmt.Errorf("failed to create uploads directory: %w", err)
		}
	}

	// Open root for uploads directory - provides path traversal protection
	root, err := os.OpenRoot(uploadsDir)
	if err != nil {
		return nil, fmt.Errorf("failed to open uploads root: %w", err)
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
	return nil
}
