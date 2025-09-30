package session

import (
	"fmt"
	"os"
	"path/filepath"
)

// CleanupMediaByTimestamp removes ALL media files in the session storage directory
// that were created AFTER the rewind timestamp. This is simple and reliable - we don't
// need to parse message content, just compare file timestamps.
// Errors are logged but don't fail the operation.
func CleanupMediaByTimestamp(sessionStorageDir string, rewindTimestamp int64) error {
	fmt.Printf("=== Starting media cleanup ===\n")
	fmt.Printf("Session storage dir: %s\n", sessionStorageDir)
	fmt.Printf("Rewind timestamp: %d\n", rewindTimestamp)

	// Read all files in the session storage directory
	entries, err := os.ReadDir(sessionStorageDir)
	if err != nil {
		if os.IsNotExist(err) {
			fmt.Printf("Session storage directory doesn't exist, nothing to clean up\n")
			return nil
		}
		return fmt.Errorf("failed to read session storage directory: %w", err)
	}

	var cleanupErrors []string
	deletedCount := 0

	for _, entry := range entries {
		if entry.IsDir() {
			continue // Skip subdirectories
		}

		fullPath := filepath.Join(sessionStorageDir, entry.Name())

		// Get file info
		fileInfo, err := os.Stat(fullPath)
		if err != nil {
			if !os.IsNotExist(err) {
				cleanupErrors = append(cleanupErrors, fmt.Sprintf("failed to stat %s: %v", fullPath, err))
			}
			continue
		}

		// Get file modification time as Unix timestamp
		fileTimestamp := fileInfo.ModTime().Unix()

		// Only delete if file was created AFTER the rewind timestamp
		if fileTimestamp > rewindTimestamp {
			if err := os.Remove(fullPath); err != nil {
				if !os.IsNotExist(err) {
					cleanupErrors = append(cleanupErrors, fmt.Sprintf("failed to delete %s: %v", fullPath, err))
				}
			} else {
				fmt.Printf("Deleted: %s (created at %d, rewind at %d)\n", entry.Name(), fileTimestamp, rewindTimestamp)
				deletedCount++
			}
		} else {
			fmt.Printf("Kept: %s (created at %d, rewind at %d)\n", entry.Name(), fileTimestamp, rewindTimestamp)
		}
	}

	fmt.Printf("=== Media cleanup complete: deleted %d files ===\n", deletedCount)

	if len(cleanupErrors) > 0 {
		fmt.Printf("Media cleanup completed with errors:\n")
		for _, err := range cleanupErrors {
			fmt.Printf("  - %s\n", err)
		}
	}

	return nil // Always return nil - cleanup failures are non-fatal
}