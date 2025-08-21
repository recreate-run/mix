package session

import (
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
)

// File size limits for different media types
const (
	MaxVideoSize = 500 * 1024 * 1024  // 500MB for video files
	MaxImageSize = 50 * 1024 * 1024   // 50MB for image files  
	MaxAudioSize = 100 * 1024 * 1024  // 100MB for audio files
)

// Allowed MIME types mapped to their size limits
var allowedMimeTypes = map[string]int64{
	// Image types
	"image/jpeg": MaxImageSize,
	"image/jpg":  MaxImageSize,
	"image/png":  MaxImageSize,
	"image/gif":  MaxImageSize,
	"image/webp": MaxImageSize,
	// Video types
	"video/mp4":       MaxVideoSize,
	"video/quicktime": MaxVideoSize,
	"video/webm":      MaxVideoSize,
	"video/avi":       MaxVideoSize,
	"video/x-msvideo": MaxVideoSize,
	// Audio types
	"audio/mpeg": MaxAudioSize,
	"audio/wav":  MaxAudioSize,
	"audio/mp4":  MaxAudioSize,
	"audio/webm": MaxAudioSize,
}

// AssetServer serves files from a current working directory
type AssetServer struct {
	mu             sync.RWMutex
	currentWorkDir string
}

// NewAssetServer creates a new asset server
func NewAssetServer() *AssetServer {
	return &AssetServer{}
}

// SetWorkingDirectory sets the current working directory to serve assets from
func (as *AssetServer) SetWorkingDirectory(workingDir string) error {
	as.mu.Lock()
	defer as.mu.Unlock()
	
	normalizedDir, err := filepath.Abs(workingDir)
	if err != nil {
		return err
	}
	
	as.currentWorkDir = normalizedDir
	return nil
}

// validateMediaFile checks if file is a supported media type and within size limits
func (as *AssetServer) validateMediaFile(filePath string, fileInfo os.FileInfo) error {
	
	// Detect content type by reading first 512 bytes
	file, err := os.Open(filePath)
	if err != nil {
		return err
	}
	defer file.Close()
	
	buffer := make([]byte, 512)
	_, err = file.Read(buffer)
	if err != nil {
		return err
	}
	
	contentType := http.DetectContentType(buffer)
	maxSize, allowed := allowedMimeTypes[contentType]
	if !allowed {
		return fmt.Errorf("unsupported media type: %s", contentType)
	}
	
	if fileInfo.Size() > maxSize {
		return fmt.Errorf("file too large: %d bytes (max: %d)", fileInfo.Size(), maxSize)
	}
	
	return nil
}

// ServeHTTP handles asset serving requests from the current working directory
func (as *AssetServer) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	as.mu.RLock()
	workingDir := as.currentWorkDir
	as.mu.RUnlock()
	
	if workingDir == "" {
		http.NotFound(w, r)
		return
	}
	
	// URL format: /input/videos/file.mp4
	filePath := strings.TrimPrefix(r.URL.Path, "/")
	
	// Construct full file path
	fullPath := filepath.Join(workingDir, filePath)
	
	// Security check: ensure path is within working directory
	if !strings.HasPrefix(fullPath, workingDir) {
		http.Error(w, "Forbidden", http.StatusForbidden)
		return
	}

	// Check file existence first
	fileInfo, err := os.Stat(fullPath)
	if err != nil {
		if os.IsNotExist(err) {
			http.NotFound(w, r)
			return
		}
		http.Error(w, "File access error", http.StatusInternalServerError)
		return
	}

	// Validate media file type and size
	if err := as.validateMediaFile(fullPath, fileInfo); err != nil {
		if strings.Contains(err.Error(), "unsupported media type") {
			http.Error(w, "Unsupported media type", http.StatusUnsupportedMediaType)
		} else if strings.Contains(err.Error(), "file too large") {
			http.Error(w, "File too large", http.StatusRequestEntityTooLarge)
		} else {
			http.Error(w, "File validation failed", http.StatusInternalServerError)
		}
		return
	}

	// Set CORS headers for frontend access
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Methods", "GET, OPTIONS")
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type")

	if r.Method == "OPTIONS" {
		w.WriteHeader(http.StatusOK)
		return
	}

	// Serve the file using Go's optimized file server
	http.ServeFile(w, r, fullPath)
}

// GetCurrentWorkingDirectory returns the current working directory
func (as *AssetServer) GetCurrentWorkingDirectory() string {
	as.mu.RLock()
	defer as.mu.RUnlock()
	return as.currentWorkDir
}