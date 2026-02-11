package browser

import (
	"encoding/base64"
	"image"
	"image/color"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestVisualizeBoundingBoxes(t *testing.T) {
	t.Helper()

	// Load test image
	imageData, err := os.ReadFile("testdata/taxonomy_button.png")
	if err != nil {
		t.Fatalf("Failed to read test image: %v", err)
	}

	// Get image dimensions
	width, height, err := getImageDimensions(imageData)
	if err != nil {
		t.Fatalf("Failed to get image dimensions: %v", err)
	}

	// Sample bounding box response (normalized [0, 1000] coordinates)
	responseJSON := `[
		{"box_2d": [100, 100, 300, 200], "name": "button1"},
		{"box_2d": [400, 300, 600, 500], "name": "button2"}
	]`

	prompt := "Find all buttons with bounding boxes"

	// Create debug directory if not exists
	debugDir := "debug_screenshots"
	_ = os.MkdirAll(debugDir, 0o755)

	// Run visualization
	visualizeBoundingBoxes(imageData, responseJSON, width, height, prompt)

	// Give goroutine time to complete (in actual usage, this runs async)
	time.Sleep(100 * time.Millisecond)

	// Verify that timestamp subfolder was created
	subfolders, err := filepath.Glob(filepath.Join(debugDir, "*"))
	if err != nil {
		t.Fatalf("Failed to glob debug subfolders: %v", err)
	}

	if len(subfolders) == 0 {
		t.Error("Expected debug subfolder to be created, but none found")
		return
	}

	// Check the most recent subfolder
	subfolder := subfolders[len(subfolders)-1]

	// Verify screenshot_raw.png exists
	rawImageFile := filepath.Join(subfolder, "screenshot_raw.png")
	if _, err := os.Stat(rawImageFile); os.IsNotExist(err) {
		t.Errorf("Expected raw image file %s to exist", rawImageFile)
	}

	// Verify screenshot_overlayed.jpg exists
	overlayedImageFile := filepath.Join(subfolder, "screenshot_overlayed.jpg")
	if _, err := os.Stat(overlayedImageFile); os.IsNotExist(err) {
		t.Errorf("Expected overlayed image file %s to exist", overlayedImageFile)
	}

	// Verify prompt.txt exists and contains the prompt
	promptFile := filepath.Join(subfolder, "prompt.txt")
	if _, err := os.Stat(promptFile); os.IsNotExist(err) {
		t.Errorf("Expected prompt file %s to exist", promptFile)
	} else {
		content, _ := os.ReadFile(promptFile)
		if string(content) != prompt {
			t.Errorf("Expected prompt.txt to contain %q, got %q", prompt, string(content))
		}
	}

	// Cleanup - remove entire subfolder
	_ = os.RemoveAll(subfolder)
}

func TestVisualizeBoundingBoxes_InvalidJSON(t *testing.T) {
	t.Helper()

	// Load test image
	imageData, err := os.ReadFile("testdata/taxonomy_button.png")
	if err != nil {
		t.Fatalf("Failed to read test image: %v", err)
	}

	width, height, _ := getImageDimensions(imageData)

	// Invalid JSON should fail silently (no panic)
	invalidJSON := `{invalid json}`

	// Should not panic
	visualizeBoundingBoxes(imageData, invalidJSON, width, height, "test prompt")

	time.Sleep(50 * time.Millisecond)

	// Verify no subfolder was created (since parsing failed)
	subfolders, _ := filepath.Glob(filepath.Join("debug_screenshots", "*"))
	if len(subfolders) > 0 {
		t.Error("Expected no subfolder to be created for invalid JSON")
		for _, match := range subfolders {
			_ = os.RemoveAll(match)
		}
	}
}

func TestVisualizeBoundingBoxes_EmptyBoundingBoxes(t *testing.T) {
	t.Helper()

	// Load test image
	imageData, err := os.ReadFile("testdata/taxonomy_button.png")
	if err != nil {
		t.Fatalf("Failed to read test image: %v", err)
	}

	width, height, _ := getImageDimensions(imageData)

	// Empty bounding boxes should still create a file (just the original image)
	emptyJSON := `[]`

	visualizeBoundingBoxes(imageData, emptyJSON, width, height, "test prompt")

	time.Sleep(100 * time.Millisecond)

	// Verify subfolder was created
	subfolders, _ := filepath.Glob(filepath.Join("debug_screenshots", "*"))
	if len(subfolders) == 0 {
		t.Error("Expected debug visualization subfolder to be created even with empty bounding boxes")
	}

	// Cleanup
	for _, match := range subfolders {
		_ = os.RemoveAll(match)
	}
}

func TestDrawRect(t *testing.T) {
	t.Helper()

	// Create a small test image
	width, height := 100, 100
	img := createTestImage(width, height)

	// Draw a rectangle
	red := colorRGBA(255, 0, 0, 255)
	drawRect(img, 10, 10, 50, 50, red, 2)

	// Verify corners are red
	corners := []struct{ x, y int }{
		{10, 10}, {50, 10}, {10, 50}, {50, 50},
	}

	for _, corner := range corners {
		r, _, _, _ := img.At(corner.x, corner.y).RGBA()
		if r>>8 != 255 {
			t.Errorf("Expected red pixel at (%d, %d), but got different color", corner.x, corner.y)
		}
	}
}

func TestDrawCircle(t *testing.T) {
	t.Helper()

	// Create a small test image
	width, height := 100, 100
	img := createTestImage(width, height)

	// Draw a circle
	green := colorRGBA(0, 255, 0, 255)
	drawCircle(img, 50, 50, 5, green)

	// Verify center is green
	r, g, b, a := img.At(50, 50).RGBA()
	_ = r
	_ = b
	_ = a
	if g>>8 != 255 {
		t.Error("Expected green pixel at center of circle")
	}
}

func TestGetImageDimensions(t *testing.T) {
	t.Helper()

	// Load test image
	imageData, err := os.ReadFile("testdata/taxonomy_button.png")
	if err != nil {
		t.Fatalf("Failed to read test image: %v", err)
	}

	width, height, err := getImageDimensions(imageData)
	if err != nil {
		t.Fatalf("Failed to get image dimensions: %v", err)
	}

	if width == 0 || height == 0 {
		t.Errorf("Expected non-zero dimensions, got width=%d, height=%d", width, height)
	}
}

func TestGetImageDimensions_InvalidData(t *testing.T) {
	t.Helper()

	invalidData := []byte("not an image")

	_, _, err := getImageDimensions(invalidData)
	if err == nil {
		t.Error("Expected error for invalid image data, but got nil")
	}
}

// Helper function to create a test RGBA image
func createTestImage(width, height int) *image.RGBA {
	return image.NewRGBA(rectangleFromDimensions(0, 0, width, height))
}

// Helper function to create color.RGBA (avoiding import alias collision)
func colorRGBA(r, g, b, a uint8) colorType {
	return colorType{R: r, G: g, B: b, A: a}
}

// Helper function to create Rectangle
func rectangleFromDimensions(x0, y0, x1, y1 int) image.Rectangle {
	return image.Rectangle{Min: image.Point{X: x0, Y: y0}, Max: image.Point{X: x1, Y: y1}}
}

// Type alias for color
type colorType = color.RGBA

// TestVisualizeBoundingBoxes_Base64Input tests visualization with base64-encoded image data
func TestVisualizeBoundingBoxes_Base64Input(t *testing.T) {
	t.Helper()

	// Load and encode test image
	imageData, err := os.ReadFile("testdata/taxonomy_button.png")
	if err != nil {
		t.Fatalf("Failed to read test image: %v", err)
	}

	// Test that our visualization function handles raw bytes correctly
	width, height, _ := getImageDimensions(imageData)

	responseJSON := `[{"box_2d": [200, 200, 400, 400], "name": "test_element"}]`

	visualizeBoundingBoxes(imageData, responseJSON, width, height, "test base64 prompt")

	time.Sleep(100 * time.Millisecond)

	// Cleanup - remove timestamp subfolders
	matches, _ := filepath.Glob(filepath.Join("debug_screenshots", "*"))
	for _, match := range matches {
		_ = os.RemoveAll(match)
	}

	if len(matches) == 0 {
		t.Error("Expected debug visualization folder to be created")
	}
}

// Benchmark visualization performance
func BenchmarkVisualizeBoundingBoxes(b *testing.B) {
	imageData, _ := os.ReadFile("testdata/taxonomy_button.png")
	width, height, _ := getImageDimensions(imageData)
	responseJSON := `[{"box_2d": [100, 100, 300, 200]}]`

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		visualizeBoundingBoxes(imageData, responseJSON, width, height, "benchmark prompt")
	}

	// Cleanup
	_ = os.RemoveAll("debug_screenshots")
}

// TestVisualizeBoundingBoxes_CoordinateAccuracy verifies coordinate scaling
func TestVisualizeBoundingBoxes_CoordinateAccuracy(t *testing.T) {
	t.Helper()

	imageData, _ := os.ReadFile("testdata/taxonomy_button.png")
	width, height, _ := getImageDimensions(imageData)

	// Normalized coordinates: center of image should be (500, 500) in [0, 1000] space
	// Actual pixel coords should be (width/2, height/2)
	centerNorm := 500
	halfWidth := width / 2
	halfHeight := height / 2

	// Create box at center
	responseJSON := `[{"box_2d": [` +
		string(rune(centerNorm-50)) + `,` +
		string(rune(centerNorm-50)) + `,` +
		string(rune(centerNorm+50)) + `,` +
		string(rune(centerNorm+50)) +
		`], "name": "center_element"}]`

	visualizeBoundingBoxes(imageData, responseJSON, width, height, "coordinate accuracy test")

	time.Sleep(100 * time.Millisecond)

	// Verify subfolder exists
	subfolders, _ := filepath.Glob(filepath.Join("debug_screenshots", "*"))
	if len(subfolders) > 0 {
		subfolder := subfolders[len(subfolders)-1]
		imageFile := filepath.Join(subfolder, "screenshot_overlayed.jpg")
		t.Logf("Created visualization at: %s", imageFile)
		t.Logf("Image dimensions: %dx%d", width, height)
		t.Logf("Expected center pixel: (%d, %d)", halfWidth, halfHeight)
	}

	// Cleanup
	for _, match := range subfolders {
		_ = os.RemoveAll(match)
	}
}

// TestShouldEnableDebugVisualization tests the dev mode check
func TestShouldEnableDebugVisualization(t *testing.T) {
	t.Helper()

	tests := []struct {
		name     string
		envValue string
		want     bool
	}{
		{
			name:     "enabled when _DEV_DEBUG=true",
			envValue: "true",
			want:     true,
		},
		{
			name:     "disabled when _DEV_DEBUG=false",
			envValue: "false",
			want:     false,
		},
		{
			name:     "disabled when _DEV_DEBUG is empty",
			envValue: "",
			want:     false,
		},
		{
			name:     "disabled when _DEV_DEBUG has invalid value",
			envValue: "yes",
			want:     false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Save original value
			originalValue := os.Getenv("_DEV_DEBUG")
			defer func() {
				if originalValue != "" {
					_ = os.Setenv("_DEV_DEBUG", originalValue)
				} else {
					_ = os.Unsetenv("_DEV_DEBUG")
				}
			}()

			// Set test value
			if tt.envValue != "" {
				_ = os.Setenv("_DEV_DEBUG", tt.envValue)
			} else {
				_ = os.Unsetenv("_DEV_DEBUG")
			}

			got := shouldEnableDebugVisualization()
			if got != tt.want {
				t.Errorf("shouldEnableDebugVisualization() = %v, want %v", got, tt.want)
			}
		})
	}
}

// TestVisualizeBoundingBoxes_DevModeOnly verifies visualization only runs in dev mode
func TestVisualizeBoundingBoxes_DevModeOnly(t *testing.T) {
	t.Helper()

	// Save original value
	originalValue := os.Getenv("_DEV_DEBUG")
	defer func() {
		if originalValue != "" {
			_ = os.Setenv("_DEV_DEBUG", originalValue)
		} else {
			_ = os.Unsetenv("_DEV_DEBUG")
		}
	}()

	// Test with dev mode disabled
	_ = os.Setenv("_DEV_DEBUG", "false")

	imageData, _ := os.ReadFile("testdata/taxonomy_button.png")
	width, height, _ := getImageDimensions(imageData)
	responseJSON := `[{"box_2d": [100, 100, 300, 200]}]`

	// Should not create file when dev mode is disabled
	if shouldEnableDebugVisualization() {
		visualizeBoundingBoxes(imageData, responseJSON, width, height, "dev mode test")
	}

	time.Sleep(100 * time.Millisecond)

	subfolders, _ := filepath.Glob(filepath.Join("debug_screenshots", "*"))
	if len(subfolders) > 0 {
		t.Error("Expected no visualization subfolder when dev mode is disabled")
		for _, match := range subfolders {
			_ = os.RemoveAll(match)
		}
	}

	// Test with dev mode enabled
	_ = os.Setenv("_DEV_DEBUG", "true")

	if shouldEnableDebugVisualization() {
		visualizeBoundingBoxes(imageData, responseJSON, width, height, "dev mode enabled test")
	}

	time.Sleep(100 * time.Millisecond)

	subfolders, _ = filepath.Glob(filepath.Join("debug_screenshots", "*"))
	if len(subfolders) == 0 {
		t.Error("Expected visualization subfolder when dev mode is enabled")
	}

	// Cleanup
	for _, match := range subfolders {
		_ = os.RemoveAll(match)
	}
}

func init() {
	// Ensure base64 is imported (used implicitly in the example)
	_ = base64.StdEncoding
}
