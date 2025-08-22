package session

import (
	"crypto/md5"
	"fmt"
	"image"
	"image/jpeg"
	_ "image/png"
	_ "image/gif"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"sync"

	"github.com/nfnt/resize"
	_ "golang.org/x/image/webp"
	_ "golang.org/x/image/bmp"
	_ "golang.org/x/image/tiff"
)

// File size limits for different media types
const (
	MaxVideoSize = 500 * 1024 * 1024  // 500MB for video files
	MaxImageSize = 50 * 1024 * 1024   // 50MB for image files  
	MaxAudioSize = 100 * 1024 * 1024  // 100MB for audio files
)

// FileTypeCategory represents different media categories
type FileTypeCategory string

const (
	CategoryImage FileTypeCategory = "image"
	CategoryVideo FileTypeCategory = "video"
	CategoryAudio FileTypeCategory = "audio"
)

// FileTypeInfo contains file type information
type FileTypeInfo struct {
	Extensions []string          `json:"extensions"`
	MimeTypes  map[string]int64  `json:"mime_types"`
	SizeLimit  int64             `json:"size_limit"`
}

// SupportedFileTypes contains all supported file type configurations
type SupportedFileTypes struct {
	Image FileTypeInfo `json:"image"`
	Video FileTypeInfo `json:"video"`
	Audio FileTypeInfo `json:"audio"`
}

// Global registry of supported file types - single source of truth
var supportedFileTypes = SupportedFileTypes{
	Image: FileTypeInfo{
		Extensions: []string{".jpg", ".jpeg", ".png", ".gif", ".webp", ".bmp", ".tiff"},
		MimeTypes: map[string]int64{
			"image/jpeg": MaxImageSize,
			"image/jpg":  MaxImageSize,
			"image/png":  MaxImageSize,
			"image/gif":  MaxImageSize,
			"image/webp": MaxImageSize,
			"image/bmp":  MaxImageSize,
			"image/tiff": MaxImageSize,
		},
		SizeLimit: MaxImageSize,
	},
	Video: FileTypeInfo{
		Extensions: []string{".mp4", ".webm", ".mov", ".avi", ".mkv", ".wmv", ".flv", ".m4v"},
		MimeTypes: map[string]int64{
			"video/mp4":       MaxVideoSize,
			"video/quicktime": MaxVideoSize,
			"video/webm":      MaxVideoSize,
			"video/avi":       MaxVideoSize,
			"video/x-msvideo": MaxVideoSize,
			"video/x-matroska": MaxVideoSize,
		},
		SizeLimit: MaxVideoSize,
	},
	Audio: FileTypeInfo{
		Extensions: []string{".mp3", ".wav", ".ogg", ".m4a", ".aac", ".flac", ".wma"},
		MimeTypes: map[string]int64{
			"audio/mpeg":  MaxAudioSize,
			"audio/wav":   MaxAudioSize,
			"audio/mp4":   MaxAudioSize,
			"audio/webm":  MaxAudioSize,
			"audio/ogg":   MaxAudioSize,
			"audio/aac":   MaxAudioSize,
			"audio/x-flac": MaxAudioSize,
		},
		SizeLimit: MaxAudioSize,
	},
}

// getAllowedMimeTypes returns a flattened map of all allowed MIME types
func getAllowedMimeTypes() map[string]int64 {
	result := make(map[string]int64)
	for _, mimeTypes := range []map[string]int64{
		supportedFileTypes.Image.MimeTypes,
		supportedFileTypes.Video.MimeTypes,
		supportedFileTypes.Audio.MimeTypes,
	} {
		for mime, size := range mimeTypes {
			result[mime] = size
		}
	}
	return result
}

// AssetServer serves files from a current working directory
type AssetServer struct {
	mu             sync.RWMutex
	currentWorkDir string
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
	boxSizeRegex    = regexp.MustCompile(`^(\d+)$`)         // "100"
	widthSizeRegex  = regexp.MustCompile(`^w(\d+)$`)        // "w100"
	heightSizeRegex = regexp.MustCompile(`^h(\d+)$`)        // "h100"
)

const (
	MaxThumbnailSize = 1024 // Max width or height for thumbnails
	MinThumbnailSize = 16   // Min width or height for thumbnails
)

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

// detectContentType reads file header to determine content type
func (as *AssetServer) detectContentType(filePath string) (string, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return "", err
	}
	defer file.Close()
	
	buffer := make([]byte, 512)
	_, err = file.Read(buffer)
	if err != nil {
		return "", err
	}
	
	return http.DetectContentType(buffer), nil
}

// isVideoFile checks if file is a video based on extension (more reliable than content type)
func (as *AssetServer) isVideoFile(filePath string) bool {
	ext := strings.ToLower(filepath.Ext(filePath))
	for _, videoExt := range supportedFileTypes.Video.Extensions {
		if ext == videoExt {
			return true
		}
	}
	return false
}

// isImageFile checks if file is an image based on extension
func (as *AssetServer) isImageFile(filePath string) bool {
	ext := strings.ToLower(filepath.Ext(filePath))
	for _, imgExt := range supportedFileTypes.Image.Extensions {
		if ext == imgExt {
			return true
		}
	}
	return false
}

// isAudioFile checks if file is an audio based on extension
func (as *AssetServer) isAudioFile(filePath string) bool {
	ext := strings.ToLower(filepath.Ext(filePath))
	for _, audioExt := range supportedFileTypes.Audio.Extensions {
		if ext == audioExt {
			return true
		}
	}
	return false
}

// validateMediaFileWithContentType checks if file is a supported media type and within size limits
func (as *AssetServer) validateMediaFileWithContentType(filePath string, fileInfo os.FileInfo, contentType string) error {
	allowedMimeTypes := getAllowedMimeTypes()
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

	// Detect content type once
	contentType, err := as.detectContentType(fullPath)
	if err != nil {
		http.Error(w, "File access error", http.StatusInternalServerError)
		return
	}

	// Validate media file type and size
	if err := as.validateMediaFileWithContentType(fullPath, fileInfo, contentType); err != nil {
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

	// Check if thumbnail is requested
	if thumbParam := r.URL.Query().Get("thumb"); thumbParam != "" {
		// Generate thumbnails for video and image files
		if !as.isVideoFile(fullPath) && !as.isImageFile(fullPath) {
			http.Error(w, "Thumbnails only supported for video and image files", http.StatusBadRequest)
			return
		}
		
		// Parse optional time parameter for video segments
		timeParam := r.URL.Query().Get("time")
		
		if err := as.serveThumbnail(w, r, fullPath, thumbParam, timeParam); err != nil {
			http.Error(w, fmt.Sprintf("Thumbnail generation failed: %v", err), http.StatusInternalServerError)
			return
		}
		return
	}

	// Serve the file using Go's optimized file server
	http.ServeFile(w, r, fullPath)
}

// parseThumbnailSpec parses and validates thumbnail specification
func (as *AssetServer) parseThumbnailSpec(thumbParam string) (*ThumbnailSpec, error) {
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

// generateThumbnailPath creates a consistent cache path for thumbnails
func (as *AssetServer) generateThumbnailPath(workingDir, originalPath string, spec *ThumbnailSpec, timeOffset float64) string {
	thumbnailDir := filepath.Join(workingDir, ".thumbnails")
	
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
func (as *AssetServer) serveThumbnail(w http.ResponseWriter, r *http.Request, mediaPath, thumbParam, timeParam string) error {
	// Parse thumbnail specification
	spec, err := as.parseThumbnailSpec(thumbParam)
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
	
	as.mu.RLock()
	workingDir := as.currentWorkDir
	as.mu.RUnlock()
	
	// Generate thumbnail path with time offset
	thumbnailPath := as.generateThumbnailPath(workingDir, mediaPath, spec, timeOffset)
	
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
	if as.isVideoFile(mediaPath) {
		if err := as.generateVideoThumbnail(mediaPath, thumbnailPath, spec, timeOffset); err != nil {
			return err
		}
	} else if as.isImageFile(mediaPath) {
		if err := as.generateImageThumbnail(mediaPath, thumbnailPath, spec); err != nil {
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
func (as *AssetServer) generateVideoThumbnail(videoPath, thumbnailPath string, spec *ThumbnailSpec, timeOffset float64) error {
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
	// FFmpeg supports decimal seconds format: 30.5, 125.75, etc.
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
		return fmt.Errorf("ffmpeg failed: %v, output: %s", err, string(output))
	}
	
	// Verify thumbnail was created
	if _, err := os.Stat(thumbnailPath); err != nil {
		return fmt.Errorf("thumbnail file not created: %v", err)
	}
	
	return nil
}

// generateImageThumbnail uses Go's native image processing to resize an image
func (as *AssetServer) generateImageThumbnail(imagePath, thumbnailPath string, spec *ThumbnailSpec) error {
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

// GetCurrentWorkingDirectory returns the current working directory
func (as *AssetServer) GetCurrentWorkingDirectory() string {
	as.mu.RLock()
	defer as.mu.RUnlock()
	return as.currentWorkDir
}

// GetSupportedFileTypes returns the supported file types configuration
func (as *AssetServer) GetSupportedFileTypes() SupportedFileTypes {
	return supportedFileTypes
}