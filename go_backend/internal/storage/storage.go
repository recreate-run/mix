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
	// CommonStorageDir is the directory name for common files
	CommonStorageDir = "common"
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

// Initialize creates the storage root directory and common directory if they don't exist
func Initialize(config Config) error {
	if err := os.MkdirAll(config.BasePath, 0o755); err != nil {
		return fmt.Errorf("failed to create storage directory: %w", err)
	}

	// Create common storage directory
	commonDir := GetCommonStoragePath(config)
	if err := os.MkdirAll(commonDir, 0o755); err != nil {
		return fmt.Errorf("failed to create common storage directory: %w", err)
	}

	return nil
}

// GetSessionStoragePath returns the storage path for a session
// Returns /storage/{session-id}/
func GetSessionStoragePath(sessionID string, config Config) string {
	return filepath.Join(config.BasePath, sessionID)
}

// GetCommonStoragePath returns the storage path for common files
// Returns /storage/common/
func GetCommonStoragePath(config Config) string {
	return filepath.Join(config.BasePath, CommonStorageDir)
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

// GetCommonRoot returns an os.Root for the common storage directory
// This provides OS-level protection against path traversal attacks
func GetCommonRoot(config Config) (*os.Root, error) {
	commonDir := GetCommonStoragePath(config)

	// Ensure common directory exists - create it if it doesn't
	if _, err := os.Stat(commonDir); os.IsNotExist(err) {
		if err := os.MkdirAll(commonDir, 0o755); err != nil {
			return nil, fmt.Errorf("failed to create common directory: %w", err)
		}
	}

	// Open root for common directory - provides path traversal protection
	root, err := os.OpenRoot(commonDir)
	if err != nil {
		return nil, fmt.Errorf("failed to open common root: %w", err)
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

// CommonFileInfo represents a file in common storage with minimal information
type CommonFileInfo struct {
	Filename string `json:"filename"` // Just the filename (e.g., "logo.png")
	Path     string `json:"path"`     // Relative path from common directory (e.g., "images/logo.png")
}

// ListCommonFiles returns a flat list of all files in the common storage directory
func ListCommonFiles(config Config) ([]CommonFileInfo, error) {
	commonDir := GetCommonStoragePath(config)

	// Check if common directory exists
	if _, err := os.Stat(commonDir); os.IsNotExist(err) {
		return []CommonFileInfo{}, nil // Return empty list if directory doesn't exist
	}

	var files []CommonFileInfo

	// Use filepath.WalkDir for efficient recursive traversal
	err := filepath.WalkDir(commonDir, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}

		// Skip directories, only include files
		if d.IsDir() {
			return nil
		}

		// Get relative path from common directory
		relPath, err := filepath.Rel(commonDir, path)
		if err != nil {
			return err
		}

		// Get just the filename
		filename := filepath.Base(path)

		files = append(files, CommonFileInfo{
			Filename: filename,
			Path:     relPath,
		})

		return nil
	})

	if err != nil {
		return nil, fmt.Errorf("failed to walk common directory: %w", err)
	}

	return files, nil
}
