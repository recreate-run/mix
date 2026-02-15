package e2e

import (
	"context"
	"net/http"
	"os"
	"testing"
	"time"
)

const (
	DefaultServerURL = "http://localhost:8088"
	DefaultTimeout   = 60 * time.Second
)

// Setup performs common E2E test setup and skips tests if requirements aren't met
func Setup(t *testing.T) {
	t.Helper()

	// Check if E2E tests are disabled
	if os.Getenv("SKIP_E2E_TESTS") != "" {
		t.Skip("Skipping E2E tests: SKIP_E2E_TESTS is set")
	}

	// Check server is running
	serverURL := GetServerURL()
	client := &http.Client{Timeout: 2 * time.Second}
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, serverURL+"/health", http.NoBody)
	if err != nil {
		t.Fatalf("Failed to create health check request: %v", err)
	}
	resp, err := client.Do(req)
	if err != nil {
		t.Skipf("Skipping E2E test: server not running at %s: %v", serverURL, err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		t.Skipf("Skipping E2E test: server returned non-OK status: %d", resp.StatusCode)
	}

	t.Logf("✓ E2E setup complete - server running at %s", serverURL)
}

// GetServerURL returns the E2E server URL from environment or default
func GetServerURL() string {
	if url := os.Getenv("E2E_SERVER_URL"); url != "" {
		return url
	}
	return DefaultServerURL
}
