package http

import (
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"mix/internal/app"
	"mix/internal/logging"
)

// ScreenshotHandler handles screenshot retrieval endpoints
type ScreenshotHandler struct {
	app *app.App
}

// NewScreenshotHandler creates a new screenshot handler
func NewScreenshotHandler(a *app.App) *ScreenshotHandler {
	return &ScreenshotHandler{app: a}
}

// HandleGetScreenshot serves screenshot images for a session
// GET /api/sessions/{sessionID}/screenshots/{filename}
func (h *ScreenshotHandler) HandleGetScreenshot(w http.ResponseWriter, r *http.Request) {
	sessionID := r.PathValue("sessionID")
	filename := r.PathValue("filename")

	// Validate sessionID is not empty
	if sessionID == "" {
		sendValidationError(w, "sessionID", "session ID is required")
		return
	}

	// Validate filename is not empty
	if filename == "" {
		sendValidationError(w, "filename", "filename is required")
		return
	}

	// Validate session exists
	ctx := r.Context()
	_, err := h.app.Sessions.Get(ctx, sessionID)
	if err != nil {
		http.NotFound(w, r)
		return
	}

	// Validate filename (prevent path traversal)
	if strings.Contains(filename, "..") || strings.Contains(filename, "/") || strings.Contains(filename, "\\") {
		logging.Error("Invalid screenshot filename", "filename", filename, "sessionID", sessionID)
		http.Error(w, "Invalid filename", http.StatusBadRequest)
		return
	}

	// Construct file path
	screenshotPath := filepath.Join(h.app.StorageConfig.BasePath, sessionID, "screenshots", filename)

	// Check file exists
	if _, err := os.Stat(screenshotPath); os.IsNotExist(err) {
		http.NotFound(w, r)
		return
	}

	// Read file
	imageData, err := os.ReadFile(screenshotPath)
	if err != nil {
		logging.Error("Failed to read screenshot", "error", err, "path", screenshotPath)
		http.Error(w, "Failed to read screenshot", http.StatusInternalServerError)
		return
	}

	// Set headers
	w.Header().Set("Content-Type", "image/png")
	w.Header().Set("Content-Length", fmt.Sprintf("%d", len(imageData)))
	w.Header().Set("Cache-Control", "public, max-age=31536000") // Cache for 1 year (screenshots are immutable)

	// Write image data
	if _, err := w.Write(imageData); err != nil {
		logging.Error("Failed to write screenshot response", "error", err)
	}
}
