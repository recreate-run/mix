package http

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"

	"mix/internal/api"
	"mix/internal/app"
	"mix/internal/config"
	"mix/internal/db"

	_ "github.com/ncruces/go-sqlite3/driver"
	_ "github.com/ncruces/go-sqlite3/embed"
)

func TestVideoExportHTTPEndpoint(t *testing.T) {
	if os.Getenv("SKIP_INTEGRATION_TESTS") != "" {
		t.Skip("Skipping integration test")
	}

	// Set up test database
	conn, cleanup := setupTestDB(t)
	defer cleanup()

	ctx := context.Background()
	testApp, err := app.New(ctx, conn)
	if err != nil {
		t.Fatalf("Failed to create app: %v", err)
	}
	defer testApp.Shutdown()

	// Create query handler
	handler := api.NewQueryHandler(testApp)

	t.Run("Export Video with Valid Config", func(t *testing.T) {
		requestBody := map[string]interface{}{
			"config": map[string]interface{}{
				"composition": map[string]interface{}{
					"durationInFrames": 120,
					"fps":              30,
					"format":           "vertical",
				},
				"elements": []map[string]interface{}{
					{
						"type":             "text",
						"content":          "Test Video Export",
						"from":             0,
						"durationInFrames": 90,
						"layout":           "top-center",
						"style": map[string]interface{}{
							"fontSize": 72,
							"color":    "#ffffff",
						},
						"animation": map[string]interface{}{
							"type":     "fadeIn",
							"duration": 30,
						},
					},
				},
			},
			"outputPath": "/tmp/test_video.mp4",
		}

		body, _ := json.Marshal(requestBody)
		req, _ := http.NewRequest("POST", "/api/video/export", bytes.NewBuffer(body))
		req.Header.Set("Content-Type", "application/json")

		rr := httptest.NewRecorder()
		HandleVideoExport(ctx, handler, rr, req)

		// Check HTTP status for success (no need for redundant success field)
		if rr.Code != http.StatusOK {
			t.Errorf("Expected HTTP 200, got %d. Body: %s", rr.Code, rr.Body.String())
		}

		var response map[string]interface{}
		if err := json.NewDecoder(rr.Body).Decode(&response); err != nil {
			t.Fatalf("Failed to decode response: %v", err)
		}

		if outputPath, exists := response["outputPath"]; !exists || outputPath != "/tmp/test_video.mp4" {
			t.Errorf("Expected outputPath: /tmp/test_video.mp4, got %v", outputPath)
		}
	})

	t.Run("Export Video with Missing Config", func(t *testing.T) {
		requestBody := map[string]interface{}{
			"outputPath": "/tmp/test_video.mp4",
		}

		body, _ := json.Marshal(requestBody)
		req, _ := http.NewRequest("POST", "/api/video/export", bytes.NewBuffer(body))
		req.Header.Set("Content-Type", "application/json")

		rr := httptest.NewRecorder()
		HandleVideoExport(ctx, handler, rr, req)

		if rr.Code != http.StatusBadRequest {
			t.Errorf("Expected HTTP 400, got %d", rr.Code)
		}

		var response map[string]interface{}
		if err := json.NewDecoder(rr.Body).Decode(&response); err != nil {
			t.Fatalf("Failed to decode response: %v", err)
		}

		if message, exists := response["message"]; !exists || !strings.Contains(message.(string), "Missing required parameter: config") {
			t.Errorf("Expected missing config error, got: %v", message)
		}
	})

	t.Run("Export Video with Missing Output Path", func(t *testing.T) {
		requestBody := map[string]interface{}{
			"config": map[string]interface{}{
				"composition": map[string]interface{}{
					"durationInFrames": 60,
					"fps":              30,
					"format":           "vertical",
				},
			},
		}

		body, _ := json.Marshal(requestBody)
		req, _ := http.NewRequest("POST", "/api/video/export", bytes.NewBuffer(body))
		req.Header.Set("Content-Type", "application/json")

		rr := httptest.NewRecorder()
		HandleVideoExport(ctx, handler, rr, req)

		if rr.Code != http.StatusBadRequest {
			t.Errorf("Expected HTTP 400, got %d", rr.Code)
		}

		var response map[string]interface{}
		if err := json.NewDecoder(rr.Body).Decode(&response); err != nil {
			t.Fatalf("Failed to decode response: %v", err)
		}

		if message, exists := response["message"]; !exists || !strings.Contains(message.(string), "Missing required parameter: outputPath") {
			t.Errorf("Expected missing outputPath error, got: %v", message)
		}
	})

	t.Run("Export Video with Invalid JSON", func(t *testing.T) {
		invalidJSON := `{"config": {invalid json}}`
		req, _ := http.NewRequest("POST", "/api/video/export", strings.NewReader(invalidJSON))
		req.Header.Set("Content-Type", "application/json")

		rr := httptest.NewRecorder()
		HandleVideoExport(ctx, handler, rr, req)

		if rr.Code != http.StatusBadRequest {
			t.Errorf("Expected HTTP 400, got %d", rr.Code)
		}

		var response map[string]interface{}
		if err := json.NewDecoder(rr.Body).Decode(&response); err != nil {
			t.Fatalf("Failed to decode response: %v", err)
		}

		if message, exists := response["message"]; !exists || !strings.Contains(message.(string), "Failed to parse JSON body") {
			t.Errorf("Expected JSON parse error, got: %v", message)
		}
	})

	t.Run("Export Video with Wrong HTTP Method", func(t *testing.T) {
		req, _ := http.NewRequest("GET", "/api/video/export", nil)
		
		rr := httptest.NewRecorder()
		HandleVideoExport(ctx, handler, rr, req)

		if rr.Code != http.StatusMethodNotAllowed {
			t.Errorf("Expected HTTP 405, got %d", rr.Code)
		}

		var response map[string]interface{}
		if err := json.NewDecoder(rr.Body).Decode(&response); err != nil {
			t.Fatalf("Failed to decode response: %v", err)
		}

		if message, exists := response["message"]; !exists || !strings.Contains(message.(string), "Only POST method is allowed") {
			t.Errorf("Expected method not allowed error, got: %v", message)
		}
	})
}

// setupTestDB creates a test database connection and returns cleanup function
func setupTestDB(t *testing.T) (*sql.DB, func()) {
	// Set up test configuration directories
	testConfigDir := "/tmp/test-mix-video-" + t.Name()
	testDataDir := "/tmp/test-mix-data-video-" + t.Name()

	os.Setenv("_CONFIG_DIR", testConfigDir)
	os.Setenv("_DATA_DIR", testDataDir)

	// Create test directories
	os.MkdirAll(testConfigDir, 0755)
	os.MkdirAll(testDataDir, 0755)

	// Create minimal config file for testing
	configContent := `{
  "$schema": "./mix-schema.json",
  "agents": {
    "main": {
      "model": "claude-4-sonnet",
      "maxTokens": 4096
    },
    "sub": {
      "model": "claude-4-sonnet", 
      "maxTokens": 2048
    }
  },
  "mcpServers": {}
}`
	configPath := testConfigDir + "/.mix.json"
	if err := os.WriteFile(configPath, []byte(configContent), 0644); err != nil {
		t.Fatalf("Failed to create test config: %v", err)
	}

	// Initialize config for testing
	if _, err := config.Load(testConfigDir, false, false); err != nil {
		t.Fatalf("Failed to load config: %v", err)
	}

	// Use the standard database connection method
	ctx := context.Background()
	conn, err := db.Connect(ctx)
	if err != nil {
		t.Fatalf("Failed to connect to test database: %v", err)
	}

	cleanup := func() {
		conn.Close()
		os.RemoveAll(testConfigDir)
		os.RemoveAll(testDataDir)
	}

	return conn, cleanup
}