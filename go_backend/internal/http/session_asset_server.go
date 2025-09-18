package http

import (
	"crypto/md5"
	"encoding/json"
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
	"mix/internal/storage"

	"github.com/nfnt/resize"
	_ "golang.org/x/image/bmp"
	_ "golang.org/x/image/tiff"
	_ "golang.org/x/image/webp"
)

// SessionAssetHandler handles session-aware asset serving
type SessionAssetHandler struct {
	app           *app.App
	storageConfig storage.Config
	gsapGlobalRoot *os.Root // Cached root for GSAP global directory
}

// NewSessionAssetHandler creates a new session asset handler
func NewSessionAssetHandler(app *app.App, storageConfig storage.Config) *SessionAssetHandler {
	handler := &SessionAssetHandler{
		app:           app,
		storageConfig: storageConfig,
	}

	// Initialize and validate GSAP global directory at startup
	if err := handler.initializeGSAPGlobalRoot(); err != nil {
		// Log error but don't fail server startup - GSAP functionality will be disabled
		logging.Error("Failed to initialize GSAP global directory", "error", err)
	}

	return handler
}

// initializeGSAPGlobalRoot validates and caches the GSAP global directory root
func (h *SessionAssetHandler) initializeGSAPGlobalRoot() error {
	// Get GSAP global directory from environment
	globalAnimationsDir := os.Getenv("GSAP_GLOBAL_DIR")
	if globalAnimationsDir == "" {
		return fmt.Errorf("GSAP_GLOBAL_DIR environment variable is required but not set")
	}

	// Validate directory exists and is accessible
	if _, err := os.Stat(globalAnimationsDir); err != nil {
		if os.IsNotExist(err) {
			return fmt.Errorf("GSAP_GLOBAL_DIR points to non-existent directory: %s", globalAnimationsDir)
		}
		return fmt.Errorf("GSAP_GLOBAL_DIR directory is not accessible: %s. Error: %v", globalAnimationsDir, err)
	}

	// Create and cache the root for secure operations
	root, err := os.OpenRoot(globalAnimationsDir)
	if err != nil {
		return fmt.Errorf("failed to open GSAP global directory as root: %s. Error: %v", globalAnimationsDir, err)
	}

	h.gsapGlobalRoot = root
	logging.Info("GSAP global directory initialized", "path", globalAnimationsDir)
	return nil
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

	// Check file existence using Root - prevents path traversal
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

	// Get the actual file path for thumbnail generation and serving
	sessionDir := storage.GetSessionStoragePath(sessionID, h.storageConfig)
	filePath := filepath.Join(sessionDir, filename)

	// Check if thumbnail is requested
	if thumbParam := r.URL.Query().Get("thumb"); thumbParam != "" {

		// Generate thumbnails for video and image files
		if !isVideoFile(filePath) && !isImageFile(filePath) {
			logging.Error("Thumbnail request rejected: file type not supported", "filePath", filePath)
			http.Error(w, "Thumbnails only supported for video and image files", http.StatusBadRequest)
			return
		}

		// Parse optional time parameter for video segments
		timeParam := r.URL.Query().Get("time")

		if err := h.serveThumbnail(w, r, sessionID, filePath, thumbParam, timeParam); err != nil {
			logging.Error("Thumbnail generation failed", "filePath", filePath, "error", err)
			http.Error(w, fmt.Sprintf("Thumbnail generation failed: %v", err), http.StatusInternalServerError)
			return
		}
		return
	}

	// Disable caching for development - ensures media changes are immediately visible
	w.Header().Set("Cache-Control", "no-cache, no-store, must-revalidate")
	w.Header().Set("Pragma", "no-cache")
	w.Header().Set("Expires", "0")

	// Serve the file using Root for security
	file, err := root.Open(filename)
	if err != nil {
		sendInternalError(w, "opening file", err)
		return
	}
	defer file.Close()

	// Serve content with proper headers
	http.ServeContent(w, r, filename, fileInfo.ModTime(), file)
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

// generateThumbnailPath creates a consistent cache path for thumbnails in session directory
func (h *SessionAssetHandler) generateThumbnailPath(sessionID, originalPath string, spec *ThumbnailSpec, timeOffset float64) string {
	sessionStorageDir := storage.GetSessionStoragePath(sessionID, h.storageConfig)
	thumbnailDir := filepath.Join(sessionStorageDir, ".thumbnails")

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

	return filepath.Join(thumbnailDir, filename)
}

// serveThumbnail handles thumbnail generation and serving for both videos and images
func (h *SessionAssetHandler) serveThumbnail(w http.ResponseWriter, r *http.Request, sessionID, mediaPath, thumbParam, timeParam string) error {
	// Parse thumbnail specification
	spec, err := h.parseThumbnailSpec(thumbParam)
	if err != nil {
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

	// Generate thumbnail path with time offset
	thumbnailPath := h.generateThumbnailPath(sessionID, mediaPath, spec, timeOffset)

	// Check if thumbnail already exists
	if _, err := os.Stat(thumbnailPath); err == nil {
		// Serve existing thumbnail
		w.Header().Set("Content-Type", "image/jpeg")
		http.ServeFile(w, r, thumbnailPath)
		return nil
	}

	// Create thumbnails directory if it doesn't exist
	thumbnailDir := filepath.Dir(thumbnailPath)
	if err := os.MkdirAll(thumbnailDir, 0755); err != nil {
		return fmt.Errorf("failed to create thumbnail directory: %v", err)
	}

	// Generate thumbnail using FFmpeg based on file type
	if isVideoFile(mediaPath) {
		if err := h.generateVideoThumbnail(mediaPath, thumbnailPath, spec, timeOffset); err != nil {
			return err
		}
	} else if isImageFile(mediaPath) {
		if err := h.generateImageThumbnail(mediaPath, thumbnailPath, spec); err != nil {
			return err
		}
	} else {
		return fmt.Errorf("unsupported file type for thumbnail generation")
	}

	// Serve the generated thumbnail
	w.Header().Set("Content-Type", "image/jpeg")
	http.ServeFile(w, r, thumbnailPath)
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
	defer sourceFile.Close()

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
	defer outputFile.Close()

	// Encode as JPEG with high quality (quality 90 out of 100)
	jpegOptions := &jpeg.Options{Quality: 90}
	if err := jpeg.Encode(outputFile, resizedImage, jpegOptions); err != nil {
		return fmt.Errorf("failed to encode JPEG: %v", err)
	}

	return nil
}

// GSAP Animation Support - Keep separate from session-based file storage
// GSAP animations remain in their global and session-specific directories

// HandleGSAPAnimationsList handles GET /api/gsap_animations
func (h *SessionAssetHandler) HandleGSAPAnimationsList(w http.ResponseWriter, r *http.Request) {
	setCORSHeaders(w)
	if handleCORSPreflight(w, r) {
		return
	}

	if r.Method != "GET" {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	root, err := h.getGSAPGlobalRoot()
	if err != nil {
		http.Error(w, fmt.Sprintf("GSAP functionality unavailable: %v", err), http.StatusServiceUnavailable)
		return
	}

	// For now, only return global animations
	// Session-specific GSAP animations can be added later if needed
	globalAnimations, err := h.scanAnimationDirectoryWithRoot(root, "global")
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to scan global animations: %v", err), http.StatusInternalServerError)
		return
	}

	// Convert to array for JSON response
	allAnimations := make([]map[string]interface{}, 0, len(globalAnimations))
	for _, animation := range globalAnimations {
		allAnimations = append(allAnimations, animation)
	}

	// Set JSON content type and send response
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(allAnimations)
}

// getGSAPGlobalRoot returns the cached GSAP global directory root
func (h *SessionAssetHandler) getGSAPGlobalRoot() (*os.Root, error) {
	if h.gsapGlobalRoot == nil {
		return nil, fmt.Errorf("GSAP global directory not initialized - check server startup logs")
	}
	return h.gsapGlobalRoot, nil
}

// scanAnimationDirectoryWithRoot scans animations using a pre-opened Root for efficiency
func (h *SessionAssetHandler) scanAnimationDirectoryWithRoot(root *os.Root, source string) (map[string]map[string]interface{}, error) {
	animations := make(map[string]map[string]interface{})

	// Read animations directory using cached Root - we need to get the actual path for ReadDir
	// Since os.Root doesn't expose ReadDir directly, we'll need to handle this differently
	// For now, let's iterate through known animation directories by attempting to stat them
	// This is a limitation of the current os.Root API

	// Get list of potential animation names by reading the underlying directory
	// We'll need to read the actual directory path for this
	globalAnimationsDir := os.Getenv("GSAP_GLOBAL_DIR")
	if globalAnimationsDir == "" {
		return nil, fmt.Errorf("GSAP_GLOBAL_DIR environment variable not set")
	}

	entries, err := os.ReadDir(globalAnimationsDir)
	if err != nil {
		return nil, fmt.Errorf("failed to read animations directory: %w", err)
	}

	for _, entry := range entries {
		// Skip non-directories and shared directory
		if !entry.IsDir() || entry.Name() == "shared" {
			continue
		}

		// Verify directory exists using Root for security
		animationInfo, err := root.Stat(entry.Name())
		if err != nil {
			continue // Skip if we can't stat the directory
		}
		if !animationInfo.IsDir() {
			continue // Skip if it's not a directory
		}

		// Open animation subdirectory using Root
		animationRoot, err := root.OpenRoot(entry.Name())
		if err != nil {
			return nil, fmt.Errorf("failed to open animation directory '%s': %w", entry.Name(), err)
		}

		// Read schema.json using Root for secure access
		schemaFile, err := animationRoot.Open("schema.json")
		if err != nil {
			animationRoot.Close()
			return nil, fmt.Errorf("animation '%s' missing required schema.json file: %w", entry.Name(), err)
		}

		schemaBytes, err := io.ReadAll(schemaFile)
		schemaFile.Close()
		animationRoot.Close()
		if err != nil {
			return nil, fmt.Errorf("failed to read schema.json for animation '%s': %w", entry.Name(), err)
		}

		// Parse schema JSON
		var schema map[string]interface{}
		if err := json.Unmarshal(schemaBytes, &schema); err != nil {
			return nil, fmt.Errorf("invalid JSON in schema.json for animation '%s': %w", entry.Name(), err)
		}

		// Validate required schema fields
		if schema["name"] == nil || schema["description"] == nil {
			return nil, fmt.Errorf("animation '%s' schema missing required fields (name and/or description)", entry.Name())
		}

		// Create animation summary with source information
		animationSummary := map[string]interface{}{
			"name":        schema["name"],
			"description": schema["description"],
			"source":      source, // "global"
		}

		animations[entry.Name()] = animationSummary
	}

	return animations, nil
}

// HandleGetAnimationParameters handles GET /api/gsap_animations/{animation_name}/parameters
func (h *SessionAssetHandler) HandleGetAnimationParameters(w http.ResponseWriter, r *http.Request) {
	setCORSHeaders(w)
	if handleCORSPreflight(w, r) {
		return
	}

	if r.Method != "GET" {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	animationName := r.PathValue("animation_name")
	if animationName == "" {
		sendValidationError(w, "animation_name", "animation name is required")
		return
	}

	root, err := h.getGSAPGlobalRoot()
	if err != nil {
		http.Error(w, fmt.Sprintf("GSAP functionality unavailable: %v", err), http.StatusServiceUnavailable)
		return
	}

	// Check if animation directory exists using cached Root
	animationInfo, err := root.Stat(animationName)
	if err != nil {
		if os.IsNotExist(err) {
			sendNotFoundError(w, "Animation", animationName)
			return
		}
		sendValidationError(w, "animation_name", fmt.Sprintf("invalid animation name or path traversal attempt: %s", err.Error()))
		return
	}

	// Verify it's a directory
	if !animationInfo.IsDir() {
		sendValidationError(w, "animation_name", "animation name must refer to a directory")
		return
	}

	// Open animation subdirectory using cached Root
	animationRoot, err := root.OpenRoot(animationName)
	if err != nil {
		sendInternalError(w, "opening animation directory", err)
		return
	}
	defer animationRoot.Close()

	// Read schema.json using Root for secure access
	schemaFile, err := animationRoot.Open("schema.json")
	if err != nil {
		if os.IsNotExist(err) {
			http.Error(w, fmt.Sprintf("Animation '%s' is missing required schema.json file", animationName), http.StatusNotFound)
			return
		}
		sendInternalError(w, "opening schema file", err)
		return
	}
	defer schemaFile.Close()

	schemaBytes, err := io.ReadAll(schemaFile)
	if err != nil {
		sendInternalError(w, "reading schema file", err)
		return
	}

	// Parse schema JSON to validate it's valid JSON
	var schema map[string]interface{}
	if err := json.Unmarshal(schemaBytes, &schema); err != nil {
		http.Error(w, fmt.Sprintf("Invalid JSON in schema.json for animation '%s': %v", animationName, err), http.StatusInternalServerError)
		return
	}

	// Validate required schema fields
	if schema["name"] == nil || schema["description"] == nil || schema["parameters"] == nil {
		http.Error(w, fmt.Sprintf("Animation '%s' schema missing required fields (name, description, and/or parameters)", animationName), http.StatusInternalServerError)
		return
	}

	// Set JSON content type and send the full schema
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	w.Write(schemaBytes)
}

// HandleGetAnimationPreview handles GET /api/gsap_animations/{animation_name}/preview
func (h *SessionAssetHandler) HandleGetAnimationPreview(w http.ResponseWriter, r *http.Request) {
	setCORSHeaders(w)
	if handleCORSPreflight(w, r) {
		return
	}

	if r.Method != "GET" {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	animationName := r.PathValue("animation_name")
	if animationName == "" {
		sendValidationError(w, "animation_name", "animation name is required")
		return
	}

	root, err := h.getGSAPGlobalRoot()
	if err != nil {
		http.Error(w, fmt.Sprintf("GSAP functionality unavailable: %v", err), http.StatusServiceUnavailable)
		return
	}

	// Check if animation directory exists using cached Root
	animationInfo, err := root.Stat(animationName)
	if err != nil {
		if os.IsNotExist(err) {
			sendNotFoundError(w, "Animation", animationName)
			return
		}
		sendValidationError(w, "animation_name", fmt.Sprintf("invalid animation name or path traversal attempt: %s", err.Error()))
		return
	}

	// Verify it's a directory
	if !animationInfo.IsDir() {
		sendValidationError(w, "animation_name", "animation name must refer to a directory")
		return
	}

	// Open animation subdirectory using cached Root
	animationRoot, err := root.OpenRoot(animationName)
	if err != nil {
		sendInternalError(w, "opening animation directory", err)
		return
	}
	defer animationRoot.Close()

	// Read index.html using Root for secure access
	indexFile, err := animationRoot.Open("index.html")
	if err != nil {
		if os.IsNotExist(err) {
			http.Error(w, fmt.Sprintf("Animation '%s' is missing required index.html file", animationName), http.StatusNotFound)
			return
		}
		sendInternalError(w, "opening index.html file", err)
		return
	}
	defer indexFile.Close()

	indexBytes, err := io.ReadAll(indexFile)
	if err != nil {
		sendInternalError(w, "reading index.html file", err)
		return
	}

	// Update relative paths to use our API endpoints
	htmlContent := string(indexBytes)
	htmlContent = strings.ReplaceAll(htmlContent, `href="../shared/`, `href="/api/gsap/shared/`)
	htmlContent = strings.ReplaceAll(htmlContent, `src="../shared/`, `src="/api/gsap/shared/`)

	// Set HTML content type and send the file
	w.Header().Set("Content-Type", "text/html")
	w.WriteHeader(http.StatusOK)
	w.Write([]byte(htmlContent))
}

// HandleGetSharedAsset handles GET /api/gsap_animations/shared/{filepath}
func (h *SessionAssetHandler) HandleGetSharedAsset(w http.ResponseWriter, r *http.Request) {
	setCORSHeaders(w)
	if handleCORSPreflight(w, r) {
		return
	}

	if r.Method != "GET" {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	filepath := r.PathValue("filepath")
	if filepath == "" {
		sendValidationError(w, "filepath", "filepath is required")
		return
	}

	root, err := h.getGSAPGlobalRoot()
	if err != nil {
		http.Error(w, fmt.Sprintf("GSAP functionality unavailable: %v", err), http.StatusServiceUnavailable)
		return
	}

	// Open shared subdirectory using cached Root
	sharedRoot, err := root.OpenRoot("shared")
	if err != nil {
		sendInternalError(w, "opening shared directory", err)
		return
	}
	defer sharedRoot.Close()

	// Read the requested file using Root for secure access
	file, err := sharedRoot.Open(filepath)
	if err != nil {
		if os.IsNotExist(err) {
			sendNotFoundError(w, "Shared asset", filepath)
			return
		}
		sendInternalError(w, "opening shared asset file", err)
		return
	}
	defer file.Close()

	// Get file info for content type detection
	fileInfo, err := sharedRoot.Stat(filepath)
	if err != nil {
		sendInternalError(w, "getting shared asset info", err)
		return
	}

	// Set appropriate content type based on file extension
	contentType := "application/octet-stream" // default
	switch {
	case strings.HasSuffix(filepath, ".js"):
		contentType = "application/javascript"
	case strings.HasSuffix(filepath, ".css"):
		contentType = "text/css"
	case strings.HasSuffix(filepath, ".json"):
		contentType = "application/json"
	case strings.HasSuffix(filepath, ".html"):
		contentType = "text/html"
	}

	// Serve content with proper headers
	w.Header().Set("Content-Type", contentType)
	http.ServeContent(w, r, filepath, fileInfo.ModTime(), file)
}
