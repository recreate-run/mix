package video

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// ExportService handles video export operations
type ExportService struct {
	projectRoot string
}

// NewExportService creates a new video export service
func NewExportService() (*ExportService, error) {
	projectRoot, err := findProjectRoot()
	if err != nil {
		return nil, fmt.Errorf("failed to initialize video export service: %w", err)
	}

	return &ExportService{
		projectRoot: projectRoot,
	}, nil
}

// ExportRequest contains the parameters for video export
type ExportRequest struct {
	ConfigJSON   string `json:"config"`
	OutputPath   string `json:"outputPath"`
	SessionID    string `json:"sessionId,omitempty"`
}

// ExportResponse contains the result of video export
type ExportResponse struct {
	OutputPath string `json:"outputPath"`
	Message    string `json:"message"`
}

// Export exports a video using the given configuration and output path
func (s *ExportService) Export(req ExportRequest) (*ExportResponse, error) {
	// Validate JSON config
	var config map[string]interface{}
	if err := json.Unmarshal([]byte(req.ConfigJSON), &config); err != nil {
		return nil, fmt.Errorf("invalid JSON config: %w", err)
	}

	// Execute the video export
	if err := s.executeVideoExport(req.ConfigJSON, req.OutputPath); err != nil {
		return nil, err
	}

	return &ExportResponse{
		OutputPath: req.OutputPath,
		Message:    "Video exported successfully",
	}, nil
}

// executeVideoExport performs the actual video export operation
func (s *ExportService) executeVideoExport(configJSON, outputPath string) error {
	remotionDir := filepath.Join(s.projectRoot, "packages", "remotion_starter_template")
	
	// Verify Remotion directory exists
	if _, err := os.Stat(remotionDir); os.IsNotExist(err) {
		return fmt.Errorf("Remotion project not found at: %s", remotionDir)
	}


	// Extract output filename (without extension) for export script
	outputDir := filepath.Dir(outputPath)
	outputName := strings.TrimSuffix(filepath.Base(outputPath), filepath.Ext(outputPath))

	// Ensure output directory exists
	if err := os.MkdirAll(outputDir, 0755); err != nil {
		return fmt.Errorf("failed to create output directory: %w", err)
	}

	// Run export script with config via stdin (no temp files)
	cmd := exec.Command("./scripts/export_video.sh", "--output", outputName)
	cmd.Dir = remotionDir
	cmd.Stdin = strings.NewReader(configJSON)

	// Capture output for logging instead of printing directly
	var stdout, stderr strings.Builder
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("video export failed: %w (stdout: %s, stderr: %s)", err, stdout.String(), stderr.String())
	}

	// Move generated video to session output path
	generatedVideo := filepath.Join(remotionDir, "output", outputName+".mp4")
	if _, err := os.Stat(generatedVideo); os.IsNotExist(err) {
		return fmt.Errorf("expected video output not found: %s", generatedVideo)
	}

	// Use copy instead of move to handle cross-filesystem scenarios
	if err := copyFile(generatedVideo, outputPath); err != nil {
		return fmt.Errorf("failed to copy video to session output: %w", err)
	}

	// Clean up generated video in Remotion directory
	os.Remove(generatedVideo)

	return nil
}

// findProjectRoot locates the project root directory
func findProjectRoot() (string, error) {
	// Start from current working directory
	dir, err := os.Getwd()
	if err != nil {
		return "", err
	}

	// Walk up the directory tree looking for packages/remotion_starter_template
	for {
		candidatePath := filepath.Join(dir, "packages", "remotion_starter_template")
		if _, err := os.Stat(candidatePath); err == nil {
			return dir, nil
		}

		parent := filepath.Dir(dir)
		if parent == dir {
			// Reached filesystem root
			break
		}
		dir = parent
	}

	return "", fmt.Errorf("could not find project root (packages/remotion_starter_template not found)")
}

// copyFile copies a file from src to dst, preserving permissions
func copyFile(src, dst string) error {
	sourceFile, err := os.Open(src)
	if err != nil {
		return err
	}
	defer sourceFile.Close()

	destFile, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer destFile.Close()

	_, err = io.Copy(destFile, sourceFile)
	if err != nil {
		return err
	}

	// Copy file permissions
	sourceInfo, err := os.Stat(src)
	if err != nil {
		return err
	}

	return os.Chmod(dst, sourceInfo.Mode())
}