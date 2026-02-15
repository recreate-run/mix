package browser

import (
	"encoding/base64"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Test MaxScreenshotSize constant
func TestMaxScreenshotSize(t *testing.T) {
	assert.Equal(t, 10*1024*1024, MaxScreenshotSize)
}

// Test saveScreenshot with valid data
func TestSaveScreenshotValid(t *testing.T) {
	// Create temporary directory
	tempDir := t.TempDir()

	// Create small PNG image data (1x1 pixel transparent PNG)
	pngData := []byte{
		0x89, 0x50, 0x4E, 0x47, 0x0D, 0x0A, 0x1A, 0x0A, // PNG signature
		0x00, 0x00, 0x00, 0x0D, 0x49, 0x48, 0x44, 0x52, // IHDR chunk
		0x00, 0x00, 0x00, 0x01, 0x00, 0x00, 0x00, 0x01,
		0x08, 0x06, 0x00, 0x00, 0x00, 0x1F, 0x15, 0xC4,
		0x89, 0x00, 0x00, 0x00, 0x0A, 0x49, 0x44, 0x41, // IDAT chunk
		0x54, 0x78, 0x9C, 0x63, 0x00, 0x01, 0x00, 0x00,
		0x05, 0x00, 0x01, 0x0D, 0x0A, 0x2D, 0xB4, 0x00, // IEND chunk
		0x00, 0x00, 0x00, 0x49, 0x45, 0x4E, 0x44, 0xAE,
		0x42, 0x60, 0x82,
	}
	base64Data := base64.StdEncoding.EncodeToString(pngData)

	// Save screenshot
	filename, err := saveScreenshot(base64Data, tempDir)

	require.NoError(t, err)
	assert.NotEmpty(t, filename)
	assert.True(t, strings.HasPrefix(filename, "screenshot_"))
	assert.True(t, strings.HasSuffix(filename, ".png"))

	// Verify file exists
	fullPath := filepath.Join(tempDir, filename)
	_, err = os.Stat(fullPath)
	require.NoError(t, err)

	// Verify file contents
	savedData, err := os.ReadFile(fullPath)
	require.NoError(t, err)
	assert.Equal(t, pngData, savedData)
}

// Test saveScreenshot with invalid base64
func TestSaveScreenshotInvalidBase64(t *testing.T) {
	tempDir := t.TempDir()

	// Invalid base64 string
	invalidBase64 := "this is not valid base64!!!"

	_, err := saveScreenshot(invalidBase64, tempDir)

	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to decode screenshot")
}

// Test saveScreenshot with oversized data
func TestSaveScreenshotOversized(t *testing.T) {
	tempDir := t.TempDir()

	// Create data larger than MaxScreenshotSize
	largeData := make([]byte, MaxScreenshotSize+1)
	base64Data := base64.StdEncoding.EncodeToString(largeData)

	_, err := saveScreenshot(base64Data, tempDir)

	require.Error(t, err)
	assert.Contains(t, err.Error(), "exceeds maximum allowed size")
}

// Test saveScreenshot with invalid directory
func TestSaveScreenshotInvalidDirectory(t *testing.T) {
	// Use non-existent directory
	nonExistentDir := "/non/existent/directory/path"

	pngData := []byte{0x89, 0x50, 0x4E, 0x47}
	base64Data := base64.StdEncoding.EncodeToString(pngData)

	_, err := saveScreenshot(base64Data, nonExistentDir)

	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to save screenshot")
}

// Test saveScreenshot filename format
func TestSaveScreenshotFilenameFormat(t *testing.T) {
	tempDir := t.TempDir()

	pngData := []byte{0x89, 0x50, 0x4E, 0x47}
	base64Data := base64.StdEncoding.EncodeToString(pngData)

	// Save a screenshot and verify the filename format
	filename, err := saveScreenshot(base64Data, tempDir)
	require.NoError(t, err)

	// Verify filename follows the format screenshot_YYYYMMDD_HHMMSS.png
	assert.Regexp(t, `^screenshot_\d{8}_\d{6}\.png$`, filename)

	// Verify file actually exists
	fullPath := filepath.Join(tempDir, filename)
	_, err = os.Stat(fullPath)
	assert.NoError(t, err)
}

// Test formatScreenshotResponse basic functionality
func TestFormatScreenshotResponse(t *testing.T) {
	response := formatScreenshotResponse(
		"screenshot_20240101_120000.png",
		"session-123",
		"http://localhost:3020",
	)

	assert.Contains(t, response, "Screenshot captured successfully")
	assert.Contains(t, response, "Display URL: http://localhost:3020/api/sessions/session-123/files/screenshot_20240101_120000.png")
}

// Test formatScreenshotResponse URL construction
func TestFormatScreenshotResponseURLConstruction(t *testing.T) {
	tests := []struct {
		name        string
		filename    string
		sessionID   string
		baseURL     string
		expectedURL string
	}{
		{
			name:        "standard URL",
			filename:    "screenshot.png",
			sessionID:   "session-123",
			baseURL:     "http://localhost:3020",
			expectedURL: "http://localhost:3020/api/sessions/session-123/files/screenshot.png",
		},
		{
			name:        "HTTPS URL",
			filename:    "test.png",
			sessionID:   "abc",
			baseURL:     "https://example.com",
			expectedURL: "https://example.com/api/sessions/abc/files/test.png",
		},
		{
			name:        "URL with port",
			filename:    "img.png",
			sessionID:   "xyz",
			baseURL:     "http://localhost:8080",
			expectedURL: "http://localhost:8080/api/sessions/xyz/files/img.png",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			response := formatScreenshotResponse(
				tt.filename,
				tt.sessionID,
				tt.baseURL,
			)

			assert.Contains(t, response, tt.expectedURL)
		})
	}
}

// Test saveScreenshot file permissions
func TestSaveScreenshotFilePermissions(t *testing.T) {
	tempDir := t.TempDir()

	pngData := []byte{0x89, 0x50, 0x4E, 0x47}
	base64Data := base64.StdEncoding.EncodeToString(pngData)

	filename, err := saveScreenshot(base64Data, tempDir)
	require.NoError(t, err)

	fullPath := filepath.Join(tempDir, filename)
	info, err := os.Stat(fullPath)
	require.NoError(t, err)

	// Verify file permissions are 0644
	mode := info.Mode()
	assert.Equal(t, os.FileMode(0o644), mode.Perm())
}

// Test saveScreenshot with empty data
func TestSaveScreenshotEmptyData(t *testing.T) {
	tempDir := t.TempDir()

	// Empty base64 string (decodes to empty byte slice)
	emptyBase64 := base64.StdEncoding.EncodeToString([]byte{})

	filename, err := saveScreenshot(emptyBase64, tempDir)

	// Should succeed with empty file
	require.NoError(t, err)
	assert.NotEmpty(t, filename)

	// Verify file exists and is empty
	fullPath := filepath.Join(tempDir, filename)
	info, err := os.Stat(fullPath)
	require.NoError(t, err)
	assert.Equal(t, int64(0), info.Size())
}
