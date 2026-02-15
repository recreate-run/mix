package browser

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"mix/internal/llm/interfaces"
)

// handleUpload uploads a file to a file input element
func (b *browserTool) handleUpload(ctx context.Context, params BrowserParams, sessionID, sessionStorageDir string) interfaces.ToolResponse {
	// Validate parameters
	if params.FilePath == "" {
		return interfaces.NewTextErrorResponse("missing filePath parameter for upload action")
	}

	// Resolve file path
	var absolutePath string
	if filepath.IsAbs(params.FilePath) {
		// Absolute path - use as is
		absolutePath = params.FilePath
	} else {
		// Relative path - try session storage first, then uploads directory
		sessionPath := filepath.Join(sessionStorageDir, params.FilePath)
		if _, err := os.Stat(sessionPath); err == nil {
			absolutePath = sessionPath
		} else {
			// Try uploads directory
			uploadsPath := filepath.Join(b.sessionConfig.BasePath, "uploads", params.FilePath)
			if _, err := os.Stat(uploadsPath); err == nil {
				absolutePath = uploadsPath
			} else {
				return interfaces.NewTextErrorResponse(fmt.Sprintf("File not found: %s (tried session storage and uploads directory)", params.FilePath))
			}
		}
	}

	// Security check - ensure path is within allowed directories
	allowedDirs := []string{sessionStorageDir, filepath.Join(b.sessionConfig.BasePath, "uploads")}
	isAllowed := false
	for _, dir := range allowedDirs {
		if strings.HasPrefix(absolutePath, dir) {
			isAllowed = true
			break
		}
	}
	if !isAllowed {
		return interfaces.NewTextErrorResponse(fmt.Sprintf("File path outside allowed directories: %s", params.FilePath))
	}

	// Verify file exists and is not a directory
	info, err := os.Stat(absolutePath)
	if err != nil {
		return interfaces.NewTextErrorResponse(fmt.Sprintf("File not found: %s", absolutePath))
	}
	if info.IsDir() {
		return interfaces.NewTextErrorResponse(fmt.Sprintf("Path is a directory, not a file: %s", absolutePath))
	}

	// Get browser connection
	client, err := b.getClient(ctx, sessionID)
	if err != nil {
		return interfaces.NewTextErrorResponse(fmt.Sprintf("Failed to get browser client: %v", err))
	}

	// browser-service mode: refresh element cache to avoid stale element errors
	if adapter, ok := client.(*ServiceClientAdapter); ok {
		_, readErr := adapter.ReadPage(ctx, true, params.TabID)
		if readErr != nil {
			return interfaces.NewTextErrorResponse(fmt.Sprintf("Failed to read page elements: %v", readErr))
		}
	}

	// Upload file (tabID is always required and validated)
	result, err := client.UploadFile(ctx, params.Index, []string{absolutePath}, params.TabID)
	if err != nil {
		return interfaces.NewTextErrorResponse(fmt.Sprintf("Upload failed: %v", err))
	}

	return interfaces.NewTextResponse(fmt.Sprintf("Successfully uploaded %d file(s): %s", result.FilesUploaded, strings.Join(result.FileNames, ", ")))
}
