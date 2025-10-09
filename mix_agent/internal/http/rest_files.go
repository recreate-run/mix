package http

import (
	"fmt"
	"net/http"
	"regexp"
	"strings"
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
	URL          string  `json:"url"`          // Static URL to access the file
}

// FileHandler handles REST endpoints for session file operations
type FileHandler struct {
	app *app.App
}

// NewFileHandler creates a new file handler
func NewFileHandler(app *app.App) *FileHandler {
	return &FileHandler{
		app: app,
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

	// Handle dots - replace all except the one before the file extension
	withoutSpaces := string(result)
	lastDotPos := strings.LastIndex(withoutSpaces, ".")

	if lastDotPos > 0 { // > 0 to preserve hidden files starting with .
		basename := withoutSpaces[:lastDotPos]
		extension := withoutSpaces[lastDotPos:] // includes the dot

		// Replace dots in basename with underscores
		basename = strings.ReplaceAll(basename, ".", "_")

		withoutSpaces = basename + extension
	}

	// Collapse multiple consecutive underscores into a single underscore
	sanitized := regexp.MustCompile(`_+`).ReplaceAllString(withoutSpaces, "_")
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
	defer func() {
		_ = file.Close()
	}()

	// Get filename from upload and sanitize it
	originalFilename := header.Filename
	filename := sanitizeFilename(originalFilename)

	// Upload to storage provider (Supabase or local)
	storageKey := fmt.Sprintf("uploads/%s", filename)
	uploadedFileInfo, err := h.app.StorageProvider.Upload(r.Context(), storageKey, file, header.Header.Get("Content-Type"))
	if err != nil {
		sendInternalError(w, "uploading file", err)
		return
	}

	// Construct absolute URL from request
	scheme := "http"
	if r.TLS != nil || r.Header.Get("X-Forwarded-Proto") == "https" {
		scheme = "https"
	}
	baseURL := fmt.Sprintf("%s://%s", scheme, r.Host)

	result := FileInfo{
		Name:     filename,
		Size:     uploadedFileInfo.Size,
		Modified: 0, // Storage provider doesn't track modification time
		IsDir:    false,
		URL:      fmt.Sprintf("%s/api/sessions/%s/files/%s", baseURL, sessionID, filename),
	}

	// Include original filename if it was different from stored name
	if originalFilename != filename {
		result.OriginalName = &originalFilename
	}

	// Track file upload
	if h.app.Analytics != nil {
		// Determine file type from extension
		fileType := ""
		if idx := strings.LastIndex(filename, "."); idx != -1 {
			fileType = filename[idx+1:]
		}

		// Check if it's a media file (image or video)
		isMedia := false
		mediaExtensions := map[string]bool{
			"jpg": true, "jpeg": true, "png": true, "gif": true, "webp": true,
			"mp4": true, "webm": true, "mov": true, "avi": true,
		}
		if mediaExtensions[strings.ToLower(fileType)] {
			isMedia = true
		}

		fileNameSanitized := originalFilename != filename
		_ = h.app.Analytics.TrackFileUploaded(ctx, sessionID, uploadedFileInfo.Size, fileType, fileNameSanitized, isMedia)
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

	// List files from storage provider
	storageFiles, err := h.app.StorageProvider.List(ctx, "uploads/")
	if err != nil {
		sendInternalError(w, "listing files", err)
		return
	}

	// Construct absolute URL from request
	scheme := "http"
	if r.TLS != nil || r.Header.Get("X-Forwarded-Proto") == "https" {
		scheme = "https"
	}
	baseURL := fmt.Sprintf("%s://%s", scheme, r.Host)

	// Build file list
	files := make([]FileInfo, 0, len(storageFiles))
	for _, storageFile := range storageFiles {
		// Skip thumbnail files - they should not be visible in file listings
		if strings.HasPrefix(storageFile.Key, "thumbnails/") || strings.Contains(storageFile.Key, "/.thumbnails/") {
			continue
		}

		// Extract filename from key (remove "uploads/" prefix)
		name := strings.TrimPrefix(storageFile.Key, "uploads/")
		files = append(files, FileInfo{
			Name:     name,
			Size:     storageFile.Size,
			Modified: 0, // Storage provider doesn't track modification time
			IsDir:    false,
			URL:      fmt.Sprintf("%s/api/sessions/%s/files/%s", baseURL, sessionID, name),
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

	// Delete file from storage provider
	storageKey := fmt.Sprintf("uploads/%s", filename)
	err = h.app.StorageProvider.Delete(ctx, storageKey)
	fileExisted := err == nil

	if err != nil {
		sendInternalError(w, "deleting file", err)
		return
	}

	// Track file deletion
	if h.app.Analytics != nil {
		_ = h.app.Analytics.TrackFileDeleted(ctx, sessionID, filename, fileExisted)
	}

	// Return 204 No Content for successful deletion
	w.WriteHeader(http.StatusNoContent)
}