package http

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"mix/internal/logging"
)


// URLVideoExportRequest represents the request payload for URL video export
type URLVideoExportRequest struct {
	URL         string   `json:"url"`
	OutputPath  string   `json:"outputPath"`             // Required absolute path where video will be saved
	FPS         *int     `json:"fps,omitempty"`          // default: 30
	AspectRatio *string  `json:"aspectRatio,omitempty"`  // default: "9/16" (format like "16/9", "4/3")
	Height      *int     `json:"height,omitempty"`       // default: 640
	Duration    *float64 `json:"duration,omitempty"`     // default: 3.0
}

// URLVideoExportResponse represents the response for URL video export
type URLVideoExportResponse struct {
	Success    bool   `json:"success"`
	OutputPath string `json:"outputPath,omitempty"`    // Path where video was saved
	Message    string `json:"message,omitempty"`
	Error      string `json:"error,omitempty"`
}

// AspectRatio represents parsed aspect ratio values
type AspectRatio struct {
	Width   float64
	Height  float64
	Decimal float64
}

// parseAspectRatio parses aspect ratio string like "16/9" into numeric values
func parseAspectRatio(aspectRatioStr string) (AspectRatio, error) {
	if aspectRatioStr == "" {
		return AspectRatio{Width: 9, Height: 16, Decimal: 9.0 / 16.0}, nil
	}

	parts := strings.Split(aspectRatioStr, "/")
	if len(parts) != 2 {
		return AspectRatio{}, fmt.Errorf("invalid aspect ratio format: %s (expected format like '16/9')", aspectRatioStr)
	}

	width, err := strconv.ParseFloat(parts[0], 64)
	if err != nil || width <= 0 {
		return AspectRatio{}, fmt.Errorf("invalid aspect ratio width: %s", parts[0])
	}

	height, err := strconv.ParseFloat(parts[1], 64)
	if err != nil || height <= 0 {
		return AspectRatio{}, fmt.Errorf("invalid aspect ratio height: %s", parts[1])
	}

	return AspectRatio{
		Width:   width,
		Height:  height,
		Decimal: width / height,
	}, nil
}



// HandleURLVideoExport handles POST /api/video/export-url
// Exports a URL as a video using Playwright-based frame capture
func HandleURLVideoExport(w http.ResponseWriter, r *http.Request) {
	// Set CORS headers
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Methods", "POST, OPTIONS")
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type")

	// Handle preflight OPTIONS request
	if r.Method == "OPTIONS" {
		w.WriteHeader(http.StatusOK)
		return
	}

	if r.Method != "POST" {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Parse request body
	var req URLVideoExportRequest
	decoder := json.NewDecoder(r.Body)
	if err := decoder.Decode(&req); err != nil {
		http.Error(w, "Invalid JSON payload", http.StatusBadRequest)
		return
	}

	// Validate required fields
	if req.URL == "" {
		http.Error(w, "URL is required", http.StatusBadRequest)
		return
	}

	if req.OutputPath == "" {
		http.Error(w, "outputPath is required", http.StatusBadRequest)
		return
	}

	// Validate output path is absolute
	if !filepath.IsAbs(req.OutputPath) {
		http.Error(w, "outputPath must be an absolute path", http.StatusBadRequest)
		return
	}

	// Security check: ensure output path doesn't contain dangerous patterns
	cleanPath := filepath.Clean(req.OutputPath)
	if cleanPath != req.OutputPath {
		http.Error(w, "Invalid characters in outputPath", http.StatusBadRequest)
		return
	}

	// Ensure the file extension is .mp4
	if !strings.HasSuffix(strings.ToLower(req.OutputPath), ".mp4") {
		http.Error(w, "outputPath must end with .mp4", http.StatusBadRequest)
		return
	}

	// Set defaults for optional parameters
	fps := 30
	if req.FPS != nil {
		fps = *req.FPS
	}
	
	aspectRatioStr := "9/16"
	if req.AspectRatio != nil {
		aspectRatioStr = *req.AspectRatio
	}
	
	height := 640
	if req.Height != nil {
		height = *req.Height
	}
	
	duration := 3.0
	if req.Duration != nil {
		duration = *req.Duration
	}

	// Parse aspect ratio and calculate width
	aspectRatio, err := parseAspectRatio(aspectRatioStr)
	if err != nil {
		http.Error(w, fmt.Sprintf("Invalid aspect ratio: %v", err), http.StatusBadRequest)
		return
	}
	
	// Calculate width from height and aspect ratio
	width := int(float64(height) * aspectRatio.Decimal)

	// Validate parameters
	if fps <= 0 || fps > 120 {
		http.Error(w, "FPS must be between 1 and 120", http.StatusBadRequest)
		return
	}
	if height <= 0 || height > 4096 {
		http.Error(w, "Height must be between 1 and 4096", http.StatusBadRequest)
		return
	}
	if duration <= 0 || duration > 60 {
		http.Error(w, "Duration must be between 0.1 and 60 seconds", http.StatusBadRequest)
		return
	}
	// Width is calculated from height and aspect ratio, so validate the calculated result
	if width <= 0 || width > 8192 {
		http.Error(w, fmt.Sprintf("Calculated width (%d) is invalid for height %d and aspect ratio %s", width, height, aspectRatioStr), http.StatusBadRequest)
		return
	}

	// Get current working directory for script path
	workingDir, err := os.Getwd()
	if err != nil {
		http.Error(w, "Failed to get working directory", http.StatusInternalServerError)
		return
	}

	// Create parent directory for output file if it doesn't exist
	outputDir := filepath.Dir(req.OutputPath)
	if err := os.MkdirAll(outputDir, 0755); err != nil {
		logging.Debug("Failed to create output directory: %v", err)
		http.Error(w, "Failed to create output directory", http.StatusInternalServerError)
		return
	}

	// Create unique temporary directory for frames only
	timestamp := time.Now().Format("20060102-150405")
	framesDir := filepath.Join(workingDir, "go_backend", "temp", "video_frames", timestamp)
	if err := os.MkdirAll(framesDir, 0755); err != nil {
		logging.Debug("Failed to create frames directory: %v", err)
		http.Error(w, "Failed to create frames directory", http.StatusInternalServerError)
		return
	}

	// Cleanup function for temporary frames directory
	defer func() {
		if err := os.RemoveAll(framesDir); err != nil {
			logging.Debug("Failed to cleanup frames directory %s: %v", framesDir, err)
		}
	}()

	// Path to Node.js capture script
	scriptPath := filepath.Join(workingDir, "go_backend", "scripts", "capture-url.mjs")
	
	// Check if Node.js script exists
	if _, err := os.Stat(scriptPath); os.IsNotExist(err) {
		logging.Debug("Capture script not found at: %s", scriptPath)
		http.Error(w, "Video capture script not found", http.StatusInternalServerError)
		return
	}

	// Prepare command arguments - use user's outputPath directly
	args := []string{
		scriptPath,
		req.URL,
		"--fps=" + strconv.Itoa(fps),
		"--width=" + strconv.Itoa(width),
		"--height=" + strconv.Itoa(height),
		"--duration=" + fmt.Sprintf("%.1f", duration),
		"--output=" + req.OutputPath,
		"--tempDir=" + framesDir,
	}

	// Execute Node.js capture script
	cmd := exec.Command("node", args...)
	cmd.Dir = workingDir

	logging.Debug("Executing video export: %s %v", cmd.Path, args)
	
	// Set reasonable timeout (5 minutes max)
	timeout := time.Duration(300) * time.Second
	
	// Create context with timeout
	ctx, cancel := context.WithTimeout(r.Context(), timeout)
	defer cancel()
	cmd = exec.CommandContext(ctx, "node", args...)
	cmd.Dir = workingDir

	// Capture output for debugging
	output, err := cmd.CombinedOutput()
	if err != nil {
		logging.Debug("Video export failed: %v\nOutput: %s", err, string(output))
		
		// Check for specific error types
		if ctx.Err() == context.DeadlineExceeded {
			http.Error(w, "Video export timed out", http.StatusRequestTimeout)
			return
		}
		
		response := URLVideoExportResponse{
			Success: false,
			Error:   fmt.Sprintf("Video export failed: %v", err),
		}
		
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(response)
		return
	}

	logging.Debug("Video export completed successfully\nOutput: %s", string(output))

	// Check if output file was created
	if _, err := os.Stat(req.OutputPath); os.IsNotExist(err) {
		logging.Debug("Output file not found: %s", req.OutputPath)
		response := URLVideoExportResponse{
			Success: false,
			Error:   "Video file was not generated",
		}
		
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(response)
		return
	}

	// Return success response with output path
	response := URLVideoExportResponse{
		Success:    true,
		OutputPath: req.OutputPath,
		Message:    "Video exported successfully",
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(response)
}