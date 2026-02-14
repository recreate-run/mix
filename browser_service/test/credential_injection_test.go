package test

import (
	"encoding/json"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/sarathmenon/browser-service/pkg/protocol"
	"github.com/sarathmenon/browser-service/test/testserver"
)

// TestLoadTaskCredentialsFromConvex tests loading credentials from Convex for a task
func TestLoadTaskCredentialsFromConvex(t *testing.T) {
	skipIfIntegrationTestsDisabled(t)

	// Check for Convex credentials
	convexURL := os.Getenv("CONVEX_URL")
	convexSecretKey := os.Getenv("CONVEX_SECRET_KEY")

	if convexURL == "" || convexSecretKey == "" {
		t.Skip("CONVEX_URL and CONVEX_SECRET_KEY not set - skipping credential injection test")
	}

	// This test requires a real task in Convex
	// Use the first task from PostHog dataset
	testCaseName := "PostHog_Cleaned_020226"
	taskID := "2118230" // First task in PostHog dataset

	ctx, client, cleanup := setupE2ETest(t, 60)
	defer cleanup()

	// Load credentials from Convex
	result, err := client.LoadTaskCredentials(ctx, testCaseName, taskID)
	if err != nil {
		t.Fatalf("Failed to load task credentials: %v", err)
	}

	if !result.Loaded {
		t.Errorf("Expected credentials to be loaded, got Loaded=%v", result.Loaded)
	}

	if result.TaskID != taskID {
		t.Errorf("Expected TaskID=%s, got %s", taskID, result.TaskID)
	}

	t.Logf("✓ Loaded credentials for task %s: %d cookies", taskID, result.CookiesCount)

	// Verify cookies were set
	cookies, err := client.GetCookies(ctx)
	if err != nil {
		t.Fatalf("Failed to get cookies: %v", err)
	}

	t.Logf("✓ Browser has %d cookies after credential load", len(cookies.Cookies))

	// If credentials were loaded, we should have at least some cookies
	if result.CookiesCount > 0 && len(cookies.Cookies) == 0 {
		t.Errorf("Expected at least %d cookies, got 0", result.CookiesCount)
	}
}

// TestLoadTaskCredentialsWithStorageState tests loading credentials with localStorage
func TestLoadTaskCredentialsWithStorageState(t *testing.T) {
	skipIfIntegrationTestsDisabled(t)

	server := testserver.StartTestServer(t)
	defer server.Close()

	ctx, client, cleanup := setupE2ETest(t, 30)
	defer cleanup()

	// First, create a storage state manually to test the legacy loginCookie format
	// Navigate to test page and set some cookies/localStorage
	_, err := client.Navigate(ctx, server.URL+"/storage-test")
	if err != nil {
		t.Fatalf("Failed to navigate: %v", err)
	}

	// Set some test cookies
	testCookies := []protocol.Cookie{
		{
			Name:   "test_session",
			Value:  "abc123",
			Domain: "127.0.0.1",
			Path:   "/",
		},
	}

	_, err = client.SetCookies(ctx, testCookies)
	if err != nil {
		t.Fatalf("Failed to set cookies: %v", err)
	}

	// Set localStorage
	testLocalStorage := map[string]string{
		"user_id":    "test-user-123",
		"theme":      "dark",
		"last_visit": time.Now().Format(time.RFC3339),
	}

	_, err = client.SetLocalStorage(ctx, testLocalStorage)
	if err != nil {
		t.Fatalf("Failed to set localStorage: %v", err)
	}

	// Save storage state
	state, err := client.SaveStorageState(ctx)
	if err != nil {
		t.Fatalf("Failed to save storage state: %v", err)
	}

	t.Logf("✓ Saved storage state with %d cookies and %d origins",
		len(state.State.Cookies), len(state.State.Origins))

	// Clear everything
	_, err = client.ClearCookies(ctx)
	if err != nil {
		t.Fatalf("Failed to clear cookies: %v", err)
	}

	// Verify cleared
	cookies, err := client.GetCookies(ctx)
	if err != nil {
		t.Fatalf("Failed to get cookies: %v", err)
	}

	if len(cookies.Cookies) > 0 {
		t.Logf("Warning: Expected 0 cookies after clear, got %d", len(cookies.Cookies))
	}

	// Now simulate what Convex would return - load the storage state
	// via LoadStorageState (since we don't have a real Convex task with this data)
	_, err = client.LoadStorageState(ctx, state.State)
	if err != nil {
		t.Fatalf("Failed to load storage state: %v", err)
	}

	// Verify cookies restored
	cookies, err = client.GetCookies(ctx)
	if err != nil {
		t.Fatalf("Failed to get cookies: %v", err)
	}

	if len(cookies.Cookies) == 0 {
		t.Errorf("Expected cookies to be restored, got 0")
	}

	foundSession := false
	for _, c := range cookies.Cookies {
		if c.Name == "test_session" && c.Value == "abc123" {
			foundSession = true
			break
		}
	}

	if !foundSession {
		t.Errorf("Expected test_session cookie to be restored")
	}

	// Verify localStorage restored
	_, err = client.Navigate(ctx, server.URL+"/storage-test")
	if err != nil {
		t.Fatalf("Failed to navigate: %v", err)
	}

	localStorage, err := client.GetLocalStorage(ctx)
	if err != nil {
		t.Fatalf("Failed to get localStorage: %v", err)
	}

	if localStorage.Items["user_id"] != "test-user-123" {
		t.Errorf("Expected localStorage user_id=test-user-123, got %s", localStorage.Items["user_id"])
	}

	t.Logf("✓ Storage state successfully loaded and verified")
}

// TestLoadTaskCredentialsInvalidProvider tests error handling
func TestLoadTaskCredentialsInvalidParams(t *testing.T) {
	skipIfIntegrationTestsDisabled(t)

	ctx, client, cleanup := setupE2ETest(t, 30)
	defer cleanup()

	// Test with empty test case name
	_, err := client.LoadTaskCredentials(ctx, "", "task-123")
	if err == nil {
		t.Errorf("Expected error for empty test case name, got nil")
	}

	if err != nil && !strings.Contains(err.Error(), "TestCaseName is required") {
		t.Logf("Error message: %v", err)
	}

	// Test with empty task ID
	_, err = client.LoadTaskCredentials(ctx, "TestCase", "")
	if err == nil {
		t.Errorf("Expected error for empty task ID, got nil")
	}

	if err != nil && !strings.Contains(err.Error(), "TaskID is required") {
		t.Logf("Error message: %v", err)
	}
}

// TestLoadTaskCredentialsMissingEnvVars tests behavior when Convex env vars not set
func TestLoadTaskCredentialsMissingEnvVars(t *testing.T) {
	skipIfIntegrationTestsDisabled(t)

	// Temporarily clear env vars
	originalURL := os.Getenv("CONVEX_URL")
	originalKey := os.Getenv("CONVEX_SECRET_KEY")
	_ = os.Setenv("CONVEX_URL", "")
	_ = os.Setenv("CONVEX_SECRET_KEY", "")
	defer func() {
		_ = os.Setenv("CONVEX_URL", originalURL)
		_ = os.Setenv("CONVEX_SECRET_KEY", originalKey)
	}()

	ctx, client, cleanup := setupE2ETest(t, 30)
	defer cleanup()

	// Should fail when credentials not set
	_, err := client.LoadTaskCredentials(ctx, "TestCase", "task-123")
	if err == nil {
		t.Errorf("Expected error when CONVEX_URL and CONVEX_SECRET_KEY not set, got nil")
	}

	if err != nil && !strings.Contains(err.Error(), "environment variables not set") {
		t.Logf("Error message: %v", err)
	}
}

// TestLoadTaskCredentialsJSONFormat tests that we can parse real Convex task format
func TestLoadTaskCredentialsJSONFormat(t *testing.T) {
	// This is a unit-style test to verify JSON parsing
	taskJSON := `{
		"task_id": "test-task-123",
		"confirmed_task": "Test task description",
		"website": "https://example.com",
		"loginCookie": "{\"cookies\":[{\"name\":\"session\",\"value\":\"xyz\",\"domain\":\".example.com\",\"path\":\"/\"}],\"origins\":[]}"
	}`

	var task struct {
		ID          string `json:"task_id"`
		Text        string `json:"confirmed_task"`
		Website     string `json:"website"`
		LoginCookie string `json:"loginCookie"`
	}

	err := json.Unmarshal([]byte(taskJSON), &task)
	if err != nil {
		t.Fatalf("Failed to parse task JSON: %v", err)
	}

	if task.ID != "test-task-123" {
		t.Errorf("Expected task_id=test-task-123, got %s", task.ID)
	}

	if task.LoginCookie == "" {
		t.Errorf("Expected loginCookie to be non-empty")
	}

	// Parse the storage state from loginCookie
	var state protocol.StorageState
	err = json.Unmarshal([]byte(task.LoginCookie), &state)
	if err != nil {
		t.Fatalf("Failed to parse loginCookie as StorageState: %v", err)
	}

	if len(state.Cookies) != 1 {
		t.Errorf("Expected 1 cookie, got %d", len(state.Cookies))
	}

	if state.Cookies[0].Name != "session" {
		t.Errorf("Expected cookie name=session, got %s", state.Cookies[0].Name)
	}

	t.Logf("✓ Successfully parsed Convex task JSON format")
}
