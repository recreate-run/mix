package http

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"unicode"

	"mix/internal/app"
	"mix/internal/logging"
	"mix/internal/storage"
)

// FileInfo represents information about a file in session storage
type FileInfo struct {
	Name         string  `json:"name"`         // The stored filename (sanitized)
	OriginalName *string `json:"originalName,omitempty"` // The original filename if different from stored name
	Size         int64   `json:"size"`
	Modified     int64   `json:"modified"` // Unix timestamp
	IsDir        bool    `json:"isDir"`
	IsCommon     bool    `json:"isCommon"` // Whether file is from common storage
}

// FileHandler handles REST endpoints for session file operations
type FileHandler struct {
	app           *app.App
	storageConfig storage.Config
}

// NewFileHandler creates a new file handler
func NewFileHandler(app *app.App, storageConfig storage.Config) *FileHandler {
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
	if !storage.ValidateSessionID(sessionID) {
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
	root, err := storage.GetSessionRoot(sessionID, h.storageConfig)
	if err != nil {
		sendInternalError(w, "getting session root", err)
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
	if !storage.ValidateSessionID(sessionID) {
		sendValidationError(w, "id", "invalid session ID format")
		return
	}

	ctx := r.Context()
	_, err := h.app.Sessions.Get(ctx, sessionID)
	if err != nil {
		sendNotFoundError(w, "Session", sessionID)
		return
	}

	// Get session storage directory
	sessionDir := storage.GetSessionStoragePath(sessionID, h.storageConfig)

	// Create a map to track files (session files override common files)
	filesMap := make(map[string]FileInfo)

	// First, add common files to the map
	commonFiles, err := storage.ListCommonFiles(h.storageConfig)
	if err != nil {
		// Log but don't fail - session files should still be listed
		logging.Error("Failed to list common files", "error", err)
	} else {
		commonDir := storage.GetCommonStoragePath(h.storageConfig)
		for _, cf := range commonFiles {
			// Get file info for size and modified time
			fullPath := filepath.Join(commonDir, cf.Path)
			if info, err := os.Stat(fullPath); err == nil && !info.IsDir() {
				filesMap[cf.Path] = FileInfo{
					Name:     cf.Path,  // Use the path as the name for common files
					Size:     info.Size(),
					Modified: info.ModTime().Unix(),
					IsDir:    false,
					IsCommon: true,
				}
			}
		}
	}

	// Then, add session files (these override any common files with same name)
	entries, err := os.ReadDir(sessionDir)
	if err != nil {
		if os.IsNotExist(err) {
			// Session directory doesn't exist yet, return only common files
			files := make([]FileInfo, 0, len(filesMap))
			for _, file := range filesMap {
				files = append(files, file)
			}
			sendJSONResponse(w, http.StatusOK, files)
			return
		}
		sendInternalError(w, "reading session directory", err)
		return
	}

	// Add session files to the map (overriding common files if names match)
	for _, entry := range entries {
		info, err := entry.Info()
		if err != nil {
			continue // Skip entries we can't get info for
		}

		filesMap[info.Name()] = FileInfo{
			Name:     info.Name(),
			Size:     info.Size(),
			Modified: info.ModTime().Unix(),
			IsDir:    info.IsDir(),
			IsCommon: false,  // Session files are not common
		}
	}

	// Convert map to slice for response
	files := make([]FileInfo, 0, len(filesMap))
	for _, file := range filesMap {
		files = append(files, file)
	}

	sendJSONResponse(w, http.StatusOK, files)
}

// HandleServeFile handles GET /api/sessions/{id}/files/{filename}
func (h *FileHandler) HandleServeFile(w http.ResponseWriter, r *http.Request) {
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

	filename := r.PathValue("filename")
	if filename == "" {
		sendValidationError(w, "filename", "filename is required")
		return
	}

	// Validate session exists and session ID format
	if !storage.ValidateSessionID(sessionID) {
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
	root, err := storage.GetSessionRoot(sessionID, h.storageConfig)
	if err != nil {
		sendInternalError(w, "getting session root", err)
		return
	}
	defer root.Close()

	// Check if file exists using Root - prevents path traversal
	fileInfo, err := root.Stat(filename)
	if err != nil {
		if os.IsNotExist(err) {
			sendNotFoundError(w, "File", filename)
			return
		}
		sendValidationError(w, "filename", fmt.Sprintf("invalid filename or path traversal attempt: %s", err.Error()))
		return
	}

	// Don't serve directories
	if fileInfo.IsDir() {
		http.Error(w, "Cannot serve directory", http.StatusBadRequest)
		return
	}

	// Open file using Root for secure serving
	file, err := root.Open(filename)
	if err != nil {
		sendInternalError(w, "opening file", err)
		return
	}
	defer file.Close()

	// Set appropriate content type and serve
	http.ServeContent(w, r, filename, fileInfo.ModTime(), file)
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
	if !storage.ValidateSessionID(sessionID) {
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
	root, err := storage.GetSessionRoot(sessionID, h.storageConfig)
	if err != nil {
		sendInternalError(w, "getting session root", err)
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