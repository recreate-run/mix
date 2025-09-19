package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

// URLVideoExportRequest represents the request payload for URL video export
type URLVideoExportRequest struct {
	URL         string   `json:"url"`                  // URL to capture
	S3URL       string   `json:"s3Url,omitempty"`      // Optional S3 URL to upload the video to
	FPS         *int     `json:"fps,omitempty"`         // default: 30
	AspectRatio *string  `json:"aspectRatio,omitempty"` // default: "9/16" (format like "16/9", "4/3")
	Height      *int     `json:"height,omitempty"`      // default: 640
	Duration    *float64 `json:"duration,omitempty"`    // default: 3.0
}

// URLVideoExportResponse represents the response for URL video export
type URLVideoExportResponse struct {
	Success bool   `json:"success"`
	URL     string `json:"url,omitempty"`      // URL to access the video (local server)
	S3URL   string `json:"s3Url,omitempty"`    // S3 URL where video was uploaded (if requested)
	Message string `json:"message,omitempty"`
	Error   string `json:"error,omitempty"`
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

// handleExport handles POST /export
// Exports a URL as a video using Playwright-based frame capture
func handleExport(w http.ResponseWriter, r *http.Request) {
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
	if width <= 0 || width > 8192 {
		http.Error(w, fmt.Sprintf("Calculated width (%d) is invalid for height %d and aspect ratio %s", width, height, aspectRatioStr), http.StatusBadRequest)
		return
	}

	// Generate unique filename for the export
	filename := GenerateUniqueFilename()
	outputPath := GetStorageFilePath(filename)

	// Get current working directory for script path
	serverDir := getAnimationsDir()

	// Create unique temporary directory for frames only
	timestamp := time.Now().Format("20060102-150405")
	framesDir := filepath.Join(serverDir, "temp", "video_frames", timestamp)
	if err := os.MkdirAll(framesDir, 0755); err != nil {
		log.Printf("Failed to create frames directory: %v", err)
		http.Error(w, "Failed to create frames directory", http.StatusInternalServerError)
		return
	}

	// Cleanup function for temporary frames directory
	defer func() {
		if err := os.RemoveAll(framesDir); err != nil {
			log.Printf("Failed to cleanup frames directory %s: %v", framesDir, err)
		}
	}()

	// Path to Node.js capture script
	scriptPath := filepath.Join(serverDir, "scripts", "capture-url.mjs")

	// Check if Node.js script exists
	if _, err := os.Stat(scriptPath); os.IsNotExist(err) {
		log.Printf("Capture script not found at: %s", scriptPath)
		http.Error(w, "Video capture script not found", http.StatusInternalServerError)
		return
	}

	// Prepare command arguments with our storage output path
	args := []string{
		scriptPath,
		req.URL,
		"--fps=" + strconv.Itoa(fps),
		"--width=" + strconv.Itoa(width),
		"--height=" + strconv.Itoa(height),
		"--duration=" + fmt.Sprintf("%.1f", duration),
		"--output=" + outputPath,
		"--tempDir=" + framesDir,
	}

	// Execute Node.js capture script
	cmd := exec.Command("node", args...)
	cmd.Dir = serverDir

	log.Printf("Executing video export: %s %v", cmd.Path, args)

	// Set reasonable timeout (5 minutes max)
	timeout := time.Duration(300) * time.Second

	// Create context with timeout
	ctx, cancel := context.WithTimeout(r.Context(), timeout)
	defer cancel()
	cmd = exec.CommandContext(ctx, "node", args...)
	cmd.Dir = serverDir

	// Capture output for debugging
	output, err := cmd.CombinedOutput()
	if err != nil {
		log.Printf("Video export failed: %v\nOutput: %s", err, string(output))

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

	log.Printf("Video export completed successfully\nOutput: %s", string(output))

	// Check if output file was created
	if _, err := os.Stat(outputPath); os.IsNotExist(err) {
		log.Printf("Output file not found: %s", outputPath)
		response := URLVideoExportResponse{
			Success: false,
			Error:   "Video file was not generated",
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(response)
		return
	}

	// Generate URL for accessing the file
	fileURL := GetStorageURL(r, filename)

	// Initialize response
	response := URLVideoExportResponse{
		Success: true,
		URL:     fileURL,
		Message: "Video exported successfully",
	}

	// Upload to S3 if requested
	if req.S3URL != "" {
		s3URL, err := UploadToS3(r.Context(), req.S3URL, outputPath)
		if err != nil {
			log.Printf("S3 upload failed: %v", err)
			// Still return success since the local export worked
			response.Message = fmt.Sprintf("Video exported successfully, but S3 upload failed: %v", err)
		} else {
			response.S3URL = s3URL
			response.Message = "Video exported successfully and uploaded to S3"
		}
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(response)
}