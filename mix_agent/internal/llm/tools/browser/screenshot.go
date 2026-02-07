package browser

import (
	"encoding/base64"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	browserprotocol "github.com/sarathmenon/browser-service/pkg/protocol"
)

const (
	// MaxScreenshotSize is the maximum allowed screenshot size (10MB)
	MaxScreenshotSize = 10 * 1024 * 1024
)

// saveScreenshot decodes a base64-encoded screenshot and saves it to session storage
func saveScreenshot(base64Data, sessionStorageDir string) (string, error) {
	// Decode base64 data
	imgData, err := base64.StdEncoding.DecodeString(base64Data)
	if err != nil {
		return "", fmt.Errorf("failed to decode screenshot: %w", err)
	}

	// Validate size
	if len(imgData) > MaxScreenshotSize {
		return "", fmt.Errorf("screenshot size %d bytes exceeds maximum allowed size %d bytes", len(imgData), MaxScreenshotSize)
	}

	// Generate timestamped filename
	timestamp := time.Now().Format("20060102_150405")
	filename := fmt.Sprintf("screenshot_%s.png", timestamp)
	filePath := filepath.Join(sessionStorageDir, filename)

	// Save to file
	if err := os.WriteFile(filePath, imgData, 0o644); err != nil {
		return "", fmt.Errorf("failed to save screenshot: %w", err)
	}

	return filename, nil
}

// formatScreenshotResponse formats the screenshot response for the LLM
func formatScreenshotResponse(filename, sessionID, baseURL string, elements []browserprotocol.Element, withOverlay bool) string {
	var sb strings.Builder

	if withOverlay && len(elements) > 0 {
		sb.WriteString(fmt.Sprintf("Screenshot captured with %d interactive elements. Use element indices for click/type actions.\n\n", len(elements)))
		sb.WriteString("Element details:\n")
		for _, elem := range elements {
			sb.WriteString(fmt.Sprintf("  [%d] %s: %s (x:%.0f y:%.0f)\n",
				elem.Index, elem.Role, elem.Name, elem.Bounds.X, elem.Bounds.Y))
		}
	} else {
		sb.WriteString("Screenshot captured successfully.\n")
	}

	// Construct file URL
	fileURL := fmt.Sprintf("%s/api/sessions/%s/files/%s", baseURL, sessionID, filename)
	sb.WriteString(fmt.Sprintf("\nDisplay URL: %s", fileURL))

	return sb.String()
}
