package http

import (
	"crypto/md5"
	"fmt"
	"image"
	_ "image/gif"
	"image/jpeg"
	_ "image/png"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"

	"mix/internal/app"
	"mix/internal/logging"
	"mix/internal/session"

	"github.com/nfnt/resize"
	_ "golang.org/x/image/bmp"
	_ "golang.org/x/image/tiff"
	_ "golang.org/x/image/webp"
)

// SessionAssetHandler handles session-aware asset serving
type SessionAssetHandler struct {
	app           *app.App
	storageConfig session.Config
}

// NewSessionAssetHandler creates a new session asset handler
func NewSessionAssetHandler(app *app.App, storageConfig session.Config) *SessionAssetHandler {
	handler := &SessionAssetHandler{
		app:           app,
		storageConfig: storageConfig,
	}

	return handler
}

// Thumbnail specification types
type ThumbnailSpec struct {
	Type   string // "box", "width", "height"
	Size   int    // the dimension value
	Width  int    // calculated width (0 means auto)
	Height int    // calculated height (0 means auto)
}

// Thumbnail parameter validation
var (
	boxSizeRegex    = regexp.MustCompile(`^(\d+)$`)  // "100"
	widthSizeRegex  = regexp.MustCompile(`^w(\d+)$`) // "w100"
	heightSizeRegex = regexp.MustCompile(`^h(\d+)$`) // "h100"
)

const (
	MaxThumbnailSize = 1024 // Max width or height for thumbnails
	MinThumbnailSize = 16   // Min width or height for thumbnails
)

// File type checking functions
func isVideoFile(filePath string) bool {
	ext := strings.ToLower(filepath.Ext(filePath))
	videoExts := []string{".mp4", ".webm", ".mov", ".avi", ".mkv", ".wmv", ".flv", ".m4v"}
	for _, videoExt := range videoExts {
		if ext == videoExt {
			return true
		}
	}
	return false
}

func isImageFile(filePath string) bool {
	ext := strings.ToLower(filepath.Ext(filePath))
	imageExts := []string{".jpg", ".jpeg", ".png", ".gif", ".webp", ".bmp", ".tiff"}
	for _, imgExt := range imageExts {
		if ext == imgExt {
			return true
		}
	}
	return false
}

// HandleServeFile serves files from session storage directories
// URL format: /api/sessions/{session-id}/files/{filename}
func (h *SessionAssetHandler) HandleServeFile(w http.ResponseWriter, r *http.Request) {
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

	// Try session-specific storage first
	served, err := h.tryServeFromSessionStorage(w, r, sessionID, filename)
	if err != nil {
		sendInternalError(w, "checking session storage", err)
		return
	}
	if served {
		return // File was found and served from session storage
	}

	// Fall back to shared uploads storage using StorageProvider
	uploadKey := "uploads/" + filename

	// Check file existence in storage
	exists, err := h.app.StorageProvider.Exists(ctx, uploadKey)
	if err != nil {
		sendInternalError(w, "checking file existence", err)
		return
	}
	if !exists {
		sendNotFoundError(w, "File", filename)
		return
	}

	// Check if thumbnail is requested
	if thumbParam := r.URL.Query().Get("thumb"); thumbParam != "" {
		// Generate thumbnails for video and image files
		if !isVideoFile(filename) && !isImageFile(filename) {
			logging.Error("Thumbnail request rejected: file type not supported", "filename", filename)
			http.Error(w, "Thumbnails only supported for video and image files", http.StatusBadRequest)
			return
		}

		// Download original file from storage to temp location for thumbnail generation
		reader, err := h.app.StorageProvider.Download(ctx, uploadKey)
		if err != nil {
			logging.Error("Failed to download original file for thumbnail", "uploadKey", uploadKey, "error", err)
			sendInternalError(w, "downloading original file", err)
			return
		}
		defer func() {
			_ = reader.Close()
		}()

		// Create temp file with proper extension for ffmpeg
		ext := filepath.Ext(filename)
		tempFile, err := os.CreateTemp("", "thumb-*"+ext)
		if err != nil {
			logging.Error("Failed to create temp file", "error", err)
			sendInternalError(w, "creating temp file", err)
			return
		}
		tempPath := tempFile.Name()
		defer func() {
			_ = os.Remove(tempPath)
		}()

		// Write downloaded content to temp file
		if _, err := io.Copy(tempFile, reader); err != nil {
			_ = tempFile.Close()
			logging.Error("Failed to write temp file", "error", err)
			sendInternalError(w, "writing temp file", err)
			return
		}
		_ = tempFile.Close()

		// Parse optional time parameter for video segments
		timeParam := r.URL.Query().Get("time")

		// Generate thumbnail using temp file
		if err := h.serveThumbnail(w, r, sessionID, tempPath, thumbParam, timeParam); err != nil {
			logging.Error("Thumbnail generation failed", "tempPath", tempPath, "error", err)
			http.Error(w, fmt.Sprintf("Thumbnail generation failed: %v", err), http.StatusInternalServerError)
			return
		}
		return
	}

	// Disable caching for development - ensures media changes are immediately visible
	w.Header().Set("Cache-Control", "no-cache, no-store, must-revalidate")
	w.Header().Set("Pragma", "no-cache")
	w.Header().Set("Expires", "0")

	// Download and serve the file from storage
	reader, err := h.app.StorageProvider.Download(ctx, uploadKey)
	if err != nil {
		sendInternalError(w, "downloading file", err)
		return
	}
	defer func() {
		_ = reader.Close()
	}()

	// Serve content with proper headers
	w.Header().Set("Content-Disposition", fmt.Sprintf("inline; filename=%q", filename))
	_, _ = io.Copy(w, reader)
}

// parseThumbnailSpec parses and validates thumbnail specification
func (h *SessionAssetHandler) parseThumbnailSpec(thumbParam string) (*ThumbnailSpec, error) {
	// Try box format: "100" (fit within 100x100)
	if matches := boxSizeRegex.FindStringSubmatch(thumbParam); len(matches) == 2 {
		size, err := strconv.Atoi(matches[1])
		if err != nil {
			return nil, fmt.Errorf("invalid size: %v", err)
		}
		if size < MinThumbnailSize || size > MaxThumbnailSize {
			return nil, fmt.Errorf("size must be between %d and %d", MinThumbnailSize, MaxThumbnailSize)
		}
		return &ThumbnailSpec{Type: "box", Size: size, Width: size, Height: size}, nil
	}

	// Try width format: "w100" (width 100, height auto)
	if matches := widthSizeRegex.FindStringSubmatch(thumbParam); len(matches) == 2 {
		size, err := strconv.Atoi(matches[1])
		if err != nil {
			return nil, fmt.Errorf("invalid width: %v", err)
		}
		if size < MinThumbnailSize || size > MaxThumbnailSize {
			return nil, fmt.Errorf("width must be between %d and %d", MinThumbnailSize, MaxThumbnailSize)
		}
		return &ThumbnailSpec{Type: "width", Size: size, Width: size, Height: 0}, nil
	}

	// Try height format: "h100" (height 100, width auto)
	if matches := heightSizeRegex.FindStringSubmatch(thumbParam); len(matches) == 2 {
		size, err := strconv.Atoi(matches[1])
		if err != nil {
			return nil, fmt.Errorf("invalid height: %v", err)
		}
		if size < MinThumbnailSize || size > MaxThumbnailSize {
			return nil, fmt.Errorf("height must be between %d and %d", MinThumbnailSize, MaxThumbnailSize)
		}
		return &ThumbnailSpec{Type: "height", Size: size, Width: 0, Height: size}, nil
	}

	return nil, fmt.Errorf("invalid thumbnail format, use: 100 (box), w100 (width), or h100 (height)")
}

// generateThumbnailKey creates a consistent storage key for thumbnails
func (h *SessionAssetHandler) generateThumbnailKey(sessionID, originalPath string, spec *ThumbnailSpec, timeOffset float64) string {
	// Create hash of original path for consistent naming
	hash := fmt.Sprintf("%x", md5.Sum([]byte(originalPath)))

	// Generate filename based on thumbnail type and time offset
	var filename string
	timeSuffix := ""
	if timeOffset > 0 {
		// Use 1 decimal place precision to avoid cache collisions
		timeSuffix = fmt.Sprintf("_t%.1f", timeOffset)
	}

	switch spec.Type {
	case "box":
		filename = fmt.Sprintf("%s_box%d%s.jpg", hash, spec.Size, timeSuffix)
	case "width":
		filename = fmt.Sprintf("%s_w%d%s.jpg", hash, spec.Size, timeSuffix)
	case "height":
		filename = fmt.Sprintf("%s_h%d%s.jpg", hash, spec.Size, timeSuffix)
	default:
		filename = fmt.Sprintf("%s_unknown%s.jpg", hash, timeSuffix)
	}

	return fmt.Sprintf("thumbnails/%s", filename)
}

// serveThumbnail handles thumbnail generation and serving for both videos and images
func (h *SessionAssetHandler) serveThumbnail(w http.ResponseWriter, r *http.Request, sessionID, mediaPath, thumbParam, timeParam string) error {
	ctx := r.Context()

	// Parse thumbnail specification
	spec, err := h.parseThumbnailSpec(thumbParam)
	if err != nil {
		logging.Error("Failed to parse thumbnail spec", "thumbParam", thumbParam, "error", err)
		return err
	}

	// Parse and validate time offset for video segments (default to 1 second)
	timeOffset := 1.0
	if timeParam != "" {
		if parsedTime, err := strconv.ParseFloat(timeParam, 64); err == nil {
			// Clamp time to reasonable bounds: 0 to 24 hours
			if parsedTime >= 0 && parsedTime <= 86400 {
				timeOffset = parsedTime
			}
			// Invalid time values fall back to default 1 second
		}
	}

	// Generate thumbnail storage key with time offset
	thumbnailKey := h.generateThumbnailKey(sessionID, mediaPath, spec, timeOffset)

	// Check if thumbnail already exists in storage
	exists, err := h.app.StorageProvider.Exists(ctx, thumbnailKey)
	if err != nil {
		logging.Error("Failed to check thumbnail existence", "key", thumbnailKey, "error", err)
		// Continue to generate - non-fatal error
	} else if exists {
		// Download and serve existing thumbnail directly (works for both local and remote storage)
		reader, err := h.app.StorageProvider.Download(ctx, thumbnailKey)
		if err != nil {
			logging.Error("Failed to download existing thumbnail", "key", thumbnailKey, "error", err)
			// Continue to regenerate
		} else {
			defer func() {
				_ = reader.Close()
			}()
			w.Header().Set("Content-Type", "image/jpeg")
			w.Header().Set("Cache-Control", "public, max-age=86400") // Cache for 24 hours
			_, _ = io.Copy(w, reader)
			return nil
		}
	}

	// Generate thumbnail to temporary file
	tmpFile, err := os.CreateTemp("", "thumbnail-*.jpg")
	if err != nil {
		logging.Error("Failed to create temp file for thumbnail", "error", err)
		return fmt.Errorf("failed to create temp file: %v", err)
	}
	tmpPath := tmpFile.Name()
	_ = tmpFile.Close()
	defer func() {
		_ = os.Remove(tmpPath)
	}() // Clean up temp file after upload

	// Generate thumbnail using FFmpeg based on file type
	if isVideoFile(mediaPath) {
		if err := h.generateVideoThumbnail(mediaPath, tmpPath, spec, timeOffset); err != nil {
			logging.Error("Failed to generate video thumbnail", "error", err)
			return err
		}
	} else if isImageFile(mediaPath) {
		if err := h.generateImageThumbnail(mediaPath, tmpPath, spec); err != nil {
			logging.Error("Failed to generate image thumbnail", "error", err)
			return err
		}
	} else {
		logging.Error("Unsupported file type for thumbnail", "mediaPath", mediaPath)
		return fmt.Errorf("unsupported file type for thumbnail generation")
	}

	// Upload thumbnail to storage provider
	thumbnailFile, err := os.Open(tmpPath)
	if err != nil {
		logging.Error("Failed to open generated thumbnail", "tmpPath", tmpPath, "error", err)
		return fmt.Errorf("failed to open generated thumbnail: %v", err)
	}
	defer func() {
		_ = thumbnailFile.Close()
	}()

	_, err = h.app.StorageProvider.Upload(ctx, thumbnailKey, thumbnailFile, "image/jpeg")
	if err != nil {
		logging.Error("Failed to upload thumbnail to storage", "key", thumbnailKey, "error", err)
		return fmt.Errorf("failed to upload thumbnail: %v", err)
	}

	// Serve the thumbnail directly (works for both local and remote storage)
	if _, err := thumbnailFile.Seek(0, 0); err != nil {
		logging.Error("Failed to seek thumbnail file", "error", err)
		return fmt.Errorf("failed to seek thumbnail file: %v", err)
	}
	w.Header().Set("Content-Type", "image/jpeg")
	w.Header().Set("Cache-Control", "public, max-age=86400") // Cache for 24 hours
	_, _ = io.Copy(w, thumbnailFile)
	return nil
}

// generateVideoThumbnail uses FFmpeg to extract a frame as thumbnail with aspect ratio preservation
func (h *SessionAssetHandler) generateVideoThumbnail(videoPath, thumbnailPath string, spec *ThumbnailSpec, timeOffset float64) error {
	// Build FFmpeg scale filter based on thumbnail specification
	var scaleFilter string
	switch spec.Type {
	case "box":
		// Fit within box while maintaining aspect ratio
		scaleFilter = fmt.Sprintf("scale=%d:%d:force_original_aspect_ratio=decrease", spec.Size, spec.Size)
	case "width":
		// Fixed width, auto height (maintains aspect ratio)
		scaleFilter = fmt.Sprintf("scale=%d:-1", spec.Size)
	case "height":
		// Fixed height, auto width (maintains aspect ratio)
		scaleFilter = fmt.Sprintf("scale=-1:%d", spec.Size)
	default:
		return fmt.Errorf("unknown thumbnail type: %s", spec.Type)
	}

	// Format time offset for FFmpeg with fractional seconds
	timeStr := fmt.Sprintf("%.2f", timeOffset)

	// FFmpeg command to extract frame at specified time, scale maintaining aspect ratio, and save as JPEG
	cmd := exec.Command("ffmpeg",
		"-i", videoPath,
		"-ss", timeStr,
		"-frames:v", "1",
		"-vf", scaleFilter, // Use video filter for proper scaling
		"-q:v", "2", // High quality JPEG
		"-y", // Overwrite output file
		thumbnailPath,
	)

	// Execute FFmpeg command
	output, err := cmd.CombinedOutput()
	if err != nil {
		// Log detailed FFmpeg error information
		logging.Error("FFmpeg thumbnail generation failed",
			"videoPath", videoPath,
			"thumbnailPath", thumbnailPath,
			"timeOffset", timeOffset,
			"scaleFilter", scaleFilter,
			"ffmpegCommand", cmd.Args,
			"error", err,
			"ffmpegOutput", string(output))
		return fmt.Errorf("ffmpeg failed: %v, output: %s", err, string(output))
	}

	// Verify thumbnail was created
	if _, err := os.Stat(thumbnailPath); err != nil {
		logging.Error("FFmpeg thumbnail verification failed",
			"expectedThumbnailPath", thumbnailPath,
			"verificationError", err)
		return fmt.Errorf("thumbnail file not created: %v", err)
	}

	return nil
}

// generateImageThumbnail uses Go's native image processing to resize an image
func (h *SessionAssetHandler) generateImageThumbnail(imagePath, thumbnailPath string, spec *ThumbnailSpec) error {
	// Open source image file
	sourceFile, err := os.Open(imagePath)
	if err != nil {
		return fmt.Errorf("failed to open source image: %v", err)
	}
	defer func() {
		_ = sourceFile.Close()
	}()

	// Decode image (supports JPEG, PNG, GIF automatically via imported decoders)
	sourceImage, _, err := image.Decode(sourceFile)
	if err != nil {
		return fmt.Errorf("failed to decode image: %v", err)
	}

	// Get original dimensions
	bounds := sourceImage.Bounds()
	originalWidth := bounds.Dx()
	originalHeight := bounds.Dy()

	// Calculate target dimensions based on thumbnail specification
	var targetWidth, targetHeight uint

	switch spec.Type {
	case "box":
		// Fit within box while maintaining aspect ratio
		if originalWidth > originalHeight {
			targetWidth = uint(spec.Size)
			targetHeight = 0 // Auto-calculate to maintain aspect ratio
		} else {
			targetWidth = 0 // Auto-calculate to maintain aspect ratio
			targetHeight = uint(spec.Size)
		}
	case "width":
		// Fixed width, auto height (maintains aspect ratio)
		targetWidth = uint(spec.Size)
		targetHeight = 0
	case "height":
		// Fixed height, auto width (maintains aspect ratio)
		targetWidth = 0
		targetHeight = uint(spec.Size)
	default:
		return fmt.Errorf("unknown thumbnail type: %s", spec.Type)
	}

	// Resize image using high-quality Lanczos resampling
	resizedImage := resize.Resize(targetWidth, targetHeight, sourceImage, resize.Lanczos3)

	// Create output file
	outputFile, err := os.Create(thumbnailPath)
	if err != nil {
		return fmt.Errorf("failed to create thumbnail file: %v", err)
	}
	defer func() {
		_ = outputFile.Close()
	}()

	// Encode as JPEG with high quality (quality 90 out of 100)
	jpegOptions := &jpeg.Options{Quality: 90}
	if err := jpeg.Encode(outputFile, resizedImage, jpegOptions); err != nil {
		return fmt.Errorf("failed to encode JPEG: %v", err)
	}

	return nil
}

// tryServeFromSessionStorage attempts to serve a file from session-specific storage
// Returns true if file was found and served, false if file doesn't exist in session storage
func (h *SessionAssetHandler) tryServeFromSessionStorage(w http.ResponseWriter, r *http.Request, sessionID, filename string) (bool, error) {
	// Try to get session-specific storage root
	sessionRoot, err := session.GetSessionRoot(sessionID, h.storageConfig)
	if err != nil {
		return false, fmt.Errorf("getting session root: %v", err)
	}
	defer func() { _ = sessionRoot.Close() }()

	// Check if file exists in session storage
	fileInfo, err := sessionRoot.Stat(filename)
	if err != nil {
		if os.IsNotExist(err) {
			return false, nil // File doesn't exist in session storage, try shared storage
		}
		return false, fmt.Errorf("invalid filename or path traversal attempt: %s", err.Error())
	}

	// Don't serve directories
	if fileInfo.IsDir() {
		http.Error(w, "Cannot serve directory", http.StatusBadRequest)
		return true, nil // We handled the request (with error)
	}

	// Get the actual file path for thumbnail generation
	sessionDir := session.GetSessionStoragePath(sessionID, h.storageConfig)
	filePath := filepath.Join(sessionDir, filename)

	// Check if thumbnail is requested
	if thumbParam := r.URL.Query().Get("thumb"); thumbParam != "" {
		// Generate thumbnails for video and image files
		if !isVideoFile(filePath) && !isImageFile(filePath) {
			logging.Error("Thumbnail request rejected: file type not supported", "filePath", filePath)
			http.Error(w, "Thumbnails only supported for video and image files", http.StatusBadRequest)
			return true, nil
		}

		// Parse optional time parameter for video segments
		timeParam := r.URL.Query().Get("time")

		if err := h.serveThumbnail(w, r, sessionID, filePath, thumbParam, timeParam); err != nil {
			logging.Error("Thumbnail generation failed", "filePath", filePath, "error", err)
			http.Error(w, fmt.Sprintf("Thumbnail generation failed: %v", err), http.StatusInternalServerError)
			return true, nil
		}
		return true, nil
	}

	// Disable caching for development - ensures media changes are immediately visible
	w.Header().Set("Cache-Control", "no-cache, no-store, must-revalidate")
	w.Header().Set("Pragma", "no-cache")
	w.Header().Set("Expires", "0")

	// Serve the file using session storage Root for security
	file, err := sessionRoot.Open(filename)
	if err != nil {
		return false, fmt.Errorf("opening session file: %v", err)
	}
	defer func() {
		_ = file.Close()
	}()

	// Serve content with proper headers
	http.ServeContent(w, r, filename, fileInfo.ModTime(), file)
	return true, nil
}
