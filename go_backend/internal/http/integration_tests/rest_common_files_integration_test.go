package integration_tests

import (
	"encoding/json"
	"io"
	"net/http"
	"testing"

	"mix/internal/storage"
)

func TestCommonFilesAPI(t *testing.T) {
	// Start test server
	result := setupIntegrationTestServer(t)
	defer result.Server.Close()

	// Test GET /api/common endpoint
	resp, err := http.Get(result.Server.URL + "/api/common")
	if err != nil {
		t.Fatalf("Failed to make request: %v", err)
	}
	defer resp.Body.Close()

	// Read response body for debugging
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("Failed to read response body: %v", err)
	}

	// Check status code
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("Expected status 200, got %d. Response: %s", resp.StatusCode, string(body))
	}

	// Check response format
	var files []storage.CommonFileInfo
	if err := json.Unmarshal(body, &files); err != nil {
		t.Fatalf("Failed to decode response: %v. Response: %s", err, string(body))
	}

	// Should return at least the existing test files
	if len(files) == 0 {
		t.Error("Expected at least one file in common storage")
	}

	// Verify structure of first file
	if len(files) > 0 {
		file := files[0]
		if file.Filename == "" {
			t.Error("Filename should not be empty")
		}
		if file.Path == "" {
			t.Error("Path should not be empty")
		}
	}
}