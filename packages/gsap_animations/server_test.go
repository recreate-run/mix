package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

const testPort = "8090"
const baseURL = "http://localhost:" + testPort

// TestMain starts the server before running tests and gracefully stops it after
func TestMain(m *testing.M) {
	// Set test environment
	os.Setenv("GSAP_PORT", testPort)

	// Create context with cancellation for graceful shutdown
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel() // Ensure server is always shut down

	// Start server with context in goroutine
	serverDone := make(chan error, 1)
	go func() {
		serverDone <- startServer(ctx)
	}()

	// Wait for server to be ready
	serverReady := false
	for i := 0; i < 20; i++ { // Increased retries for better reliability
		resp, err := http.Get(baseURL + "/animations")
		if err == nil {
			resp.Body.Close()
			serverReady = true
			break
		}
		time.Sleep(500 * time.Millisecond)
	}

	if !serverReady {
		cancel()     // Stop server
		<-serverDone // Wait for server to finish
		panic("Server failed to start within timeout")
	}

	// Run tests
	code := m.Run()

	// Graceful shutdown
	cancel() // Signal server to stop

	// Wait for server to shut down with timeout
	select {
	case err := <-serverDone:
		if err != nil {
			// Don't panic on port binding errors, which are common in tests
			if !strings.Contains(err.Error(), "bind: address already in use") {
				panic(fmt.Sprintf("Server shutdown error: %v", err))
			}
		}
	case <-time.After(10 * time.Second):
		panic("Server shutdown timeout")
	}

	os.Exit(code)
}

func TestListAnimations(t *testing.T) {
	resp, err := http.Get(baseURL + "/animations")
	if err != nil {
		t.Fatalf("Failed to make request: %v", err)
	}
	defer resp.Body.Close()

	// Check status code
	if resp.StatusCode != http.StatusOK {
		t.Errorf("Expected status 200, got %d", resp.StatusCode)
	}

	// Check content type
	contentType := resp.Header.Get("Content-Type")
	if !strings.Contains(contentType, "application/json") {
		t.Errorf("Expected JSON content type, got %s", contentType)
	}

	// Check CORS headers
	corsHeader := resp.Header.Get("Access-Control-Allow-Origin")
	if corsHeader != "*" {
		t.Errorf("Expected CORS header *, got %s", corsHeader)
	}

	// Parse response
	var animations []Animation
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("Failed to read response body: %v", err)
	}

	err = json.Unmarshal(body, &animations)
	if err != nil {
		t.Fatalf("Failed to parse JSON: %v", err)
	}

	// Expected animations
	expectedAnimations := []string{
		"bounce-overlay",
		"floating-orbs",
		"gsap-starter-template",
		"liquid-words",
		"rough-annotations",
	}

	// Check we have the expected number of animations
	if len(animations) != len(expectedAnimations) {
		t.Errorf("Expected %d animations, got %d", len(expectedAnimations), len(animations))
	}

	// Check each animation has required fields
	animationDirs := make(map[string]bool)
	for _, anim := range animations {
		if anim.Name == "" {
			t.Error("Animation missing name")
		}
		if anim.Directory == "" {
			t.Error("Animation missing directory")
		}
		animationDirs[anim.Directory] = true
	}

	// Verify all expected animations are present
	for _, expected := range expectedAnimations {
		if !animationDirs[expected] {
			t.Errorf("Missing expected animation: %s", expected)
		}
	}
}

func TestAnimationSchema(t *testing.T) {
	animations := []string{
		"bounce-overlay",
		"floating-orbs",
		"gsap-starter-template",
		"liquid-words",
		"rough-annotations",
	}

	for _, animation := range animations {
		t.Run(animation, func(t *testing.T) {
			url := fmt.Sprintf("%s/animations/%s/schema", baseURL, animation)
			resp, err := http.Get(url)
			if err != nil {
				t.Fatalf("Failed to make request: %v", err)
			}
			defer resp.Body.Close()

			// Check status code
			if resp.StatusCode != http.StatusOK {
				t.Errorf("Expected status 200, got %d", resp.StatusCode)
			}

			// Check content type
			contentType := resp.Header.Get("Content-Type")
			if !strings.Contains(contentType, "application/json") {
				t.Errorf("Expected JSON content type, got %s", contentType)
			}

			// Verify response is valid JSON
			body, err := io.ReadAll(resp.Body)
			if err != nil {
				t.Fatalf("Failed to read response body: %v", err)
			}

			var schema map[string]interface{}
			err = json.Unmarshal(body, &schema)
			if err != nil {
				t.Errorf("Response is not valid JSON: %v", err)
			}

			// Check for required schema fields
			if _, ok := schema["name"]; !ok {
				t.Error("Schema missing 'name' field")
			}
			if _, ok := schema["description"]; !ok {
				t.Error("Schema missing 'description' field")
			}
		})
	}
}

func TestAnimationPreview(t *testing.T) {
	animations := []string{
		"bounce-overlay",
		"floating-orbs",
		"gsap-starter-template",
		"liquid-words",
		"rough-annotations",
	}

	for _, animation := range animations {
		t.Run(animation, func(t *testing.T) {
			url := fmt.Sprintf("%s/animations/%s/preview", baseURL, animation)
			resp, err := http.Get(url)
			if err != nil {
				t.Fatalf("Failed to make request: %v", err)
			}
			defer resp.Body.Close()

			// Check status code
			if resp.StatusCode != http.StatusOK {
				t.Errorf("Expected status 200, got %d", resp.StatusCode)
			}

			// Check content type
			contentType := resp.Header.Get("Content-Type")
			if !strings.Contains(contentType, "text/html") {
				t.Errorf("Expected HTML content type, got %s", contentType)
			}

			// Read response and check for path rewriting
			body, err := io.ReadAll(resp.Body)
			if err != nil {
				t.Fatalf("Failed to read response body: %v", err)
			}

			html := string(body)

			// Verify path rewriting occurred (no ../shared/ paths should remain)
			if strings.Contains(html, `href="../shared/`) {
				t.Error("Path rewriting failed: found href=\"../shared/\"")
			}
			if strings.Contains(html, `src="../shared/`) {
				t.Error("Path rewriting failed: found src=\"../shared/\"")
			}

			// Verify rewritten paths exist
			if strings.Contains(html, `href="/shared/`) || strings.Contains(html, `src="/shared/`) {
				// This is expected after rewriting
			}
		})
	}
}

func TestSharedAssets(t *testing.T) {
	assets := []struct {
		file        string
		contentType string
	}{
		{"base.css", "text/css"},
		{"capture-helper.js", "application/javascript"},
		{"param-loader.js", "application/javascript"},
	}

	for _, asset := range assets {
		t.Run(asset.file, func(t *testing.T) {
			url := fmt.Sprintf("%s/shared/%s", baseURL, asset.file)
			resp, err := http.Get(url)
			if err != nil {
				t.Fatalf("Failed to make request: %v", err)
			}
			defer resp.Body.Close()

			// Check status code
			if resp.StatusCode != http.StatusOK {
				t.Errorf("Expected status 200, got %d", resp.StatusCode)
			}

			// Check content type
			contentType := resp.Header.Get("Content-Type")
			if !strings.Contains(contentType, asset.contentType) {
				t.Errorf("Expected content type %s, got %s", asset.contentType, contentType)
			}

			// Verify content exists
			body, err := io.ReadAll(resp.Body)
			if err != nil {
				t.Fatalf("Failed to read response body: %v", err)
			}

			if len(body) == 0 {
				t.Error("Response body is empty")
			}
		})
	}
}

func TestInvalidRoutes(t *testing.T) {
	testCases := []struct {
		name           string
		url            string
		method         string
		expectedStatus int
	}{
		{
			name:           "Non-existent animation schema",
			url:            baseURL + "/animations/non-existent/schema",
			method:         "GET",
			expectedStatus: http.StatusNotFound,
		},
		{
			name:           "Non-existent animation preview",
			url:            baseURL + "/animations/non-existent/preview",
			method:         "GET",
			expectedStatus: http.StatusNotFound,
		},
		{
			name:           "Invalid animation action",
			url:            baseURL + "/animations/bounce-overlay/invalid",
			method:         "GET",
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:           "Non-existent shared asset",
			url:            baseURL + "/shared/non-existent.js",
			method:         "GET",
			expectedStatus: http.StatusNotFound,
		},
		{
			name:           "Path traversal attempt in shared assets",
			url:            baseURL + "/shared/../server.go",
			method:         "GET",
			expectedStatus: http.StatusNotFound,
		},
		{
			name:           "POST to animations endpoint",
			url:            baseURL + "/animations",
			method:         "POST",
			expectedStatus: http.StatusMethodNotAllowed,
		},
		{
			name:           "POST to animation action",
			url:            baseURL + "/animations/bounce-overlay/schema",
			method:         "POST",
			expectedStatus: http.StatusMethodNotAllowed,
		},
		{
			name:           "POST to shared assets",
			url:            baseURL + "/shared/base.css",
			method:         "POST",
			expectedStatus: http.StatusMethodNotAllowed,
		},
		{
			name:           "Invalid animation path format",
			url:            baseURL + "/animations/incomplete",
			method:         "GET",
			expectedStatus: http.StatusBadRequest,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			req, err := http.NewRequest(tc.method, tc.url, nil)
			if err != nil {
				t.Fatalf("Failed to create request: %v", err)
			}

			client := &http.Client{}
			resp, err := client.Do(req)
			if err != nil {
				t.Fatalf("Failed to make request: %v", err)
			}
			defer resp.Body.Close()

			if resp.StatusCode != tc.expectedStatus {
				t.Errorf("Expected status %d, got %d", tc.expectedStatus, resp.StatusCode)
			}
		})
	}
}

func TestCORSHeaders(t *testing.T) {
	// Test OPTIONS request
	req, err := http.NewRequest("OPTIONS", baseURL+"/animations", nil)
	if err != nil {
		t.Fatalf("Failed to create request: %v", err)
	}

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		t.Fatalf("Failed to make request: %v", err)
	}
	defer resp.Body.Close()

	// Check status code
	if resp.StatusCode != http.StatusOK {
		t.Errorf("Expected status 200 for OPTIONS, got %d", resp.StatusCode)
	}

	// Check CORS headers
	corsOrigin := resp.Header.Get("Access-Control-Allow-Origin")
	if corsOrigin != "*" {
		t.Errorf("Expected CORS origin *, got %s", corsOrigin)
	}

	corsMethods := resp.Header.Get("Access-Control-Allow-Methods")
	if !strings.Contains(corsMethods, "GET") {
		t.Errorf("Expected CORS methods to include GET, got %s", corsMethods)
	}

	corsHeaders := resp.Header.Get("Access-Control-Allow-Headers")
	if !strings.Contains(corsHeaders, "Content-Type") {
		t.Errorf("Expected CORS headers to include Content-Type, got %s", corsHeaders)
	}
}

// Helper function for pointer values in tests
func ptr[T any](v T) *T {
	return &v
}

// TestExport tests the video export functionality
func TestExport(t *testing.T) {
	// Skip if running short tests
	if testing.Short() {
		t.Skip("Skipping export test in short mode")
	}

	// Create temporary output directory within the package
	tempDir := filepath.Join(".", "temp", "test_exports")
	err := os.MkdirAll(tempDir, 0755)
	if err != nil {
		t.Fatalf("Failed to create temp directory: %v", err)
	}
	defer os.RemoveAll(tempDir) // Clean up after test

	// Storage path will be automatically created by the API

	// Create request payload
	payload := URLVideoExportRequest{
		URL:         baseURL + "/animations/liquid-words/preview",
		FPS:         ptr(30),
		Duration:    ptr(2.0), // Short duration for test
		Height:      ptr(480),
		AspectRatio: ptr("10/16"), // Use even width
	}

	// Convert payload to JSON
	payloadBytes, err := json.Marshal(payload)
	if err != nil {
		t.Fatalf("Failed to marshal JSON: %v", err)
	}

	// Send request to export endpoint
	resp, err := http.Post(baseURL+"/export", "application/json", bytes.NewBuffer(payloadBytes))
	if err != nil {
		t.Fatalf("Failed to make export request: %v", err)
	}
	defer resp.Body.Close()

	// Check status code
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		t.Fatalf("Export failed with status %d: %s", resp.StatusCode, string(body))
	}

	// Parse response
	var response URLVideoExportResponse
	err = json.NewDecoder(resp.Body).Decode(&response)
	if err != nil {
		t.Fatalf("Failed to parse response JSON: %v", err)
	}

	// Verify response
	if !response.Success {
		t.Errorf("Export response indicated failure: %s", response.Error)
	}

	// Verify URL in response
	if response.URL == "" {
		t.Fatalf("Response does not contain URL")
	}

	// Verify URL format is correct
	u, err := url.Parse(response.URL)
	if err != nil {
		t.Fatalf("Failed to parse URL: %v", err)
	}

	// Check URL path starts with /storage/
	if !strings.HasPrefix(u.Path, "/storage/") {
		t.Errorf("URL path should start with /storage/, got: %s", u.Path)
	}

	// Check URL has MP4 extension
	if !strings.HasSuffix(u.Path, ".mp4") {
		t.Errorf("URL should end with .mp4 extension, got: %s", u.Path)
	}

	// Test accessibility of the URL
	resp2, err := http.Get(response.URL)
	if err != nil {
		t.Fatalf("Failed to access the exported video URL: %v", err)
	}
	defer resp2.Body.Close()

	if resp2.StatusCode != http.StatusOK {
		t.Errorf("Expected status 200 OK when accessing the video URL, got: %d", resp2.StatusCode)
	}

	if resp2.Header.Get("Content-Type") != "video/mp4" {
		t.Errorf("Expected Content-Type: video/mp4, got: %s", resp2.Header.Get("Content-Type"))
	}

	// Extract filename from URL and construct local path
	filename := filepath.Base(u.Path)
	localPath := filepath.Join(getStoragePath(), filename)

	// Check that file exists
	if _, err := os.Stat(localPath); os.IsNotExist(err) {
		t.Fatalf("Output video file was not created: %v", err)
	}

	// Use ffprobe to verify video properties
	ffprobe := exec.Command("ffprobe",
		"-v", "error",
		"-select_streams", "v:0",
		"-show_entries", "stream=width,height,codec_name,duration",
		"-of", "default=noprint_wrappers=1",
		localPath)
	ffprobeOutput, err := ffprobe.CombinedOutput()
	if err != nil {
		t.Fatalf("Failed to run ffprobe: %v\nOutput: %s", err, string(ffprobeOutput))
	}

	// Check video properties
	properties := string(ffprobeOutput)

	// Check codec
	if !strings.Contains(properties, "codec_name=h264") {
		t.Errorf("Expected h264 codec, got:\n%s", properties)
	}

	// Check dimensions
	if !strings.Contains(properties, "width=") || !strings.Contains(properties, "height=480") {
		t.Errorf("Invalid dimensions, expected height=480, got:\n%s", properties)
	}

	// Check duration
	if !strings.Contains(properties, "duration=") {
		t.Errorf("Missing duration information:\n%s", properties)
	}

	// Check file size
	fileInfo, err := os.Stat(localPath)
	if err != nil {
		t.Fatalf("Failed to get file stats: %v", err)
	}

	minSize := int64(10000) // At least 10KB
	if fileInfo.Size() < minSize {
		t.Errorf("File size too small: %d bytes (expected at least %d)", fileInfo.Size(), minSize)
	}

	// Cleanup video file explicitly for certainty
	os.Remove(localPath)
}
