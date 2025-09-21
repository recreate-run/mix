package http

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"regexp"
	"unicode"

	"mix/internal/app"
	"mix/internal/session"
)

// FileInfo represents information about a file in session storage
type FileInfo struct {
	Name         string  `json:"name"`         // The stored filename (sanitized)
	OriginalName *string `json:"originalName,omitempty"` // The original filename if different from stored name
	Size         int64   `json:"size"`
	Modified     int64   `json:"modified"` // Unix timestamp
	IsDir        bool    `json:"isDir"`
}

// FileHandler handles REST endpoints for session file operations
type FileHandler struct {
	app           *app.App
	storageConfig session.Config
}

// NewFileHandler creates a new file handler
func NewFileHandler(app *app.App, storageConfig session.Config) *FileHandler {
	return &FileHandler{
		app:           app,
		storageConfig: storageConfig,
	}
}

// sanitizeFilename removes spaces and other problematic characters from filenames
func sanitizeFilename(filename string) string {
	// Use unicode.IsSpace to catch ALL Unicode whitespace characters
	// This includes U+202F (narrow no-break space) used in macOS screenshots
	var result []rune
	for _, r := range filename {
		if unicode.IsSpace(r) {
			result = append(result, '_')
		} else {
			result = append(result, r)
		}
	}
	
	// Collapse multiple consecutive underscores into a single underscore
	sanitized := regexp.MustCompile(`_+`).ReplaceAllString(string(result), "_")
	return sanitized
}

// HandleUploadFile handles POST /api/sessions/{id}/files/upload
func (h *FileHandler) HandleUploadFile(w http.ResponseWriter, r *http.Request) {
	setCORSHeaders(w)
	if handleCORSPreflight(w, r) {
		return
	}

	if r.Method != "POST" {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	sessionID := r.PathValue("id")
	if sessionID == "" {
		sendValidationError(w, "id", "session ID is required")
		return
	}

	// Validate session exists and session ID format
	if !session.ValidateSessionID(sessionID) {
		sendValidationError(w, "id", "invalid session ID format")
		return
	}

	ctx := r.Context()
	_, err := h.app.Sessions.Get(ctx, sessionID)
	if err != nil {
		sendNotFoundError(w, "Session", sessionID)
		return
	}

	// Parse multipart form
	err = r.ParseMultipartForm(32 << 20) // 32MB max
	if err != nil {
		sendValidationError(w, "form", "failed to parse multipart form")
		return
	}

	file, header, err := r.FormFile("file")
	if err != nil {
		sendValidationError(w, "file", "file upload required")
		return
	}
	defer file.Close()

	// Get filename from upload and sanitize it
	originalFilename := header.Filename
	filename := sanitizeFilename(originalFilename)
	
	// Use os.Root for secure file operations - prevents path traversal
	root, err := session.GetUploadsRoot(h.storageConfig)
	if err != nil {
		sendInternalError(w, "getting uploads root", err)
		return
	}
	defer root.Close()

	// Create file using Root - this prevents path traversal attacks
	dst, err := root.Create(filename)
	if err != nil {
		sendValidationError(w, "filename", fmt.Sprintf("invalid filename or path traversal attempt: %s", err.Error()))
		return
	}
	defer dst.Close()

	// Copy uploaded file to destination
	_, err = io.Copy(dst, file)
	if err != nil {
		sendInternalError(w, "saving file", err)
		return
	}

	// Get file info for response using Root
	fileInfo, err := root.Stat(filename)
	if err != nil {
		sendInternalError(w, "getting file info", err)
		return
	}

	result := FileInfo{
		Name:     filename,
		Size:     fileInfo.Size(),
		Modified: fileInfo.ModTime().Unix(),
		IsDir:    fileInfo.IsDir(),
	}
	
	// Include original filename if it was different from stored name
	if originalFilename != filename {
		result.OriginalName = &originalFilename
	}

	sendJSONResponse(w, http.StatusCreated, result)
}

// HandleListFiles handles GET /api/sessions/{id}/files
func (h *FileHandler) HandleListFiles(w http.ResponseWriter, r *http.Request) {
	setCORSHeaders(w)
	if handleCORSPreflight(w, r) {
		return
	}

	if r.Method != "GET" {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	sessionID := r.PathValue("id")
	if sessionID == "" {
		sendValidationError(w, "id", "session ID is required")
		return
	}

	// Validate session exists and session ID format
	if !session.ValidateSessionID(sessionID) {
		sendValidationError(w, "id", "invalid session ID format")
		return
	}

	ctx := r.Context()
	_, err := h.app.Sessions.Get(ctx, sessionID)
	if err != nil {
		sendNotFoundError(w, "Session", sessionID)
		return
	}

	// Get uploads storage directory
	uploadsDir := session.GetUploadsStoragePath(h.storageConfig)

	// Read uploads files
	entries, err := os.ReadDir(uploadsDir)
	if err != nil {
		if os.IsNotExist(err) {
			// Uploads directory doesn't exist yet, return empty list
			sendJSONResponse(w, http.StatusOK, []FileInfo{})
			return
		}
		sendInternalError(w, "reading uploads directory", err)
		return
	}

	// Build file list from uploads files
	files := make([]FileInfo, 0, len(entries))
	for _, entry := range entries {
		info, err := entry.Info()
		if err != nil {
			continue // Skip entries we can't get info for
		}

		files = append(files, FileInfo{
			Name:     info.Name(),
			Size:     info.Size(),
			Modified: info.ModTime().Unix(),
			IsDir:    info.IsDir(),
		})
	}

	sendJSONResponse(w, http.StatusOK, files)
}


// HandleDeleteFile handles DELETE /api/sessions/{id}/files/{filename}
func (h *FileHandler) HandleDeleteFile(w http.ResponseWriter, r *http.Request) {
	setCORSHeaders(w)
	if handleCORSPreflight(w, r) {
		return
	}

	if r.Method != "DELETE" {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	sessionID := r.PathValue("id")
	if sessionID == "" {
		sendValidationError(w, "id", "session ID is required")
		return
	}

	filename := r.PathValue("filename")
	if filename == "" {
		sendValidationError(w, "filename", "filename is required")
		return
	}

	// Validate session exists and session ID format
	if !session.ValidateSessionID(sessionID) {
		sendValidationError(w, "id", "invalid session ID format")
		return
	}

	ctx := r.Context()
	_, err := h.app.Sessions.Get(ctx, sessionID)
	if err != nil {
		sendNotFoundError(w, "Session", sessionID)
		return
	}

	// Use os.Root for secure file operations
	root, err := session.GetUploadsRoot(h.storageConfig)
	if err != nil {
		sendInternalError(w, "getting uploads root", err)
		return
	}
	defer root.Close()

	// Check if file exists using Root - prevents path traversal
	if _, err := root.Stat(filename); err != nil {
		if os.IsNotExist(err) {
			sendNotFoundError(w, "File", filename)
			return
		}
		sendValidationError(w, "filename", fmt.Sprintf("invalid filename or path traversal attempt: %s", err.Error()))
		return
	}

	// Delete the file using Root
	err = root.Remove(filename)
	if err != nil {
		sendInternalError(w, "deleting file", err)
		return
	}

	// Return 204 No Content for successful deletion
	w.WriteHeader(http.StatusNoContent)
}