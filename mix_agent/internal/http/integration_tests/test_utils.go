package integration_tests

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/joho/godotenv"
	"mix/internal/app"
	"mix/internal/config"
	"mix/internal/db"
	httphandlers "mix/internal/http"
	_ "mix/internal/llm/models"
	"mix/internal/session"

	_ "github.com/ncruces/go-sqlite3/driver"
	_ "github.com/ncruces/go-sqlite3/embed"
)

// loadEnvFile loads environment variables from .env file for testing
func loadEnvFile(t *testing.T) {
	t.Helper()

	// Test working directory is /mix_agent/internal/http/integration_tests, so .env is four levels up
	envPath := filepath.Join("..", "..", "..", "..", ".env")

	// Load .env file - ignore error if file doesn't exist
	_ = godotenv.Load(envPath)
}

// requireLLMCredentials skips tests that require LLM API access if credentials aren't configured
func requireLLMCredentials(t *testing.T) {
	t.Helper()

	// Check environment variable first (fast path)
	if os.Getenv("ANTHROPIC_API_KEY") != "" {
		return
	}

	// Check database credentials (requires initialized config)
	credService := config.GetAPICredentials()
	if credService != nil {
		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer cancel()

		hasKey, err := credService.HasAPIKey(ctx, "anthropic")
		if err == nil && hasKey {
			t.Log("✓ Using Anthropic API key from database")
			return
		}
	}

	t.Skip("Skipping test: no Anthropic API key found in environment or database")
}

// TestServerResult contains the initialized test server components
type TestServerResult struct {
	Server    *httptest.Server
	App       *app.App
	SessionID string
	ConfigDir string
	DataDir   string
}

// initMCPTools mock implementation for testing
func initMCPTools(ctx context.Context, appInstance *app.App) {
	// Mock implementation - in real app this initializes MCP tools
	// For tests, we just need to ensure the app doesn't crash
}

// setupIntegrationTestServer sets up a complete test environment for integration testing
// This function consolidates the common setup logic shared between REST and SSE tests
func setupIntegrationTestServer(t *testing.T) *TestServerResult {
	t.Helper()

	// Load environment variables from .env file
	loadEnvFile(t)

	// Auto-generate test name from test function name
	testName := t.Name()

	// Set up isolated test environment
	testConfigDir := "/tmp/test-mix-" + testName
	testDataDir := "/tmp/test-mix-data-" + testName

	_ = os.Setenv("_CONFIG_DIR", testConfigDir)
	_ = os.Setenv("_DATA_DIR", testDataDir)

	// Create test directories
	if err := os.MkdirAll(testConfigDir, 0o750); err != nil {
		t.Fatalf("Failed to create test config dir: %v", err)
	}
	if err := os.MkdirAll(testDataDir, 0o750); err != nil {
		t.Fatalf("Failed to create test data dir: %v", err)
	}

	// Initialize configuration (database-only, no config file needed)
	if _, err := config.Load(testConfigDir, false, false); err != nil {
		t.Fatalf("Failed to load test config: %v", err)
	}

	// Connect to database
	ctx := context.Background()
	conn, err := db.Connect(ctx, ".mix")
	if err != nil {
		t.Fatalf("Failed to connect to test database: %v", err)
	}

	// Create test application
	testApp, err := app.New(ctx, conn)
	if err != nil {
		t.Fatalf("Failed to create test app: %v", err)
	}

	// Initialize MCP tools
	initMCPTools(ctx, testApp)

	// Ensure credentials service is fully initialized
	// This is needed because credentials are preloaded in a background goroutine
	time.Sleep(100 * time.Millisecond)

	// Create test session (max 20 chars per DB constraint)
	testSession, err := testApp.Sessions.Create(ctx, "Test Session", "", "default", session.SessionTypeMain, "", "", "")
	if err != nil {
		t.Fatalf("Failed to create test session: %v", err)
	}

	// Create REST handlers
	sessionHandler := httphandlers.NewSessionHandler(testApp)
	messageHandler := httphandlers.NewMessageHandler(testApp)
	systemHandler := httphandlers.NewSystemHandler(testApp)
	preferencesHandler := httphandlers.NewPreferencesHandler(testApp)
	authHandler := httphandlers.NewAuthHandler(testApp)

	// Create file management handlers
	fileHandler := httphandlers.NewFileHandler(testApp)
	sessionAssetHandler := httphandlers.NewSessionAssetHandler(testApp, testApp.StorageConfig)

	// Set up HTTP multiplexer with all REST endpoints
	mux := http.NewServeMux()

	// Session management endpoints
	mux.HandleFunc("GET /api/sessions", sessionHandler.HandleListSessions)
	mux.HandleFunc("POST /api/sessions", sessionHandler.HandleCreateSession)
	mux.HandleFunc("GET /api/sessions/{id}", sessionHandler.HandleGetSession)
	mux.HandleFunc("DELETE /api/sessions/{id}", sessionHandler.HandleDeleteSession)

	// Message endpoints
	mux.HandleFunc("POST /api/sessions/{id}/messages", messageHandler.HandleSendMessage)
	mux.HandleFunc("GET /api/sessions/{id}/messages", messageHandler.HandleListSessionMessages)
	mux.HandleFunc("GET /api/sessions/{id}/export", messageHandler.HandleExportSession)
	mux.HandleFunc("GET /api/messages/history", messageHandler.HandleMessageHistory)
	mux.HandleFunc("POST /api/sessions/{id}/cancel", messageHandler.HandleCancelAgent)

	// System endpoints
	mux.HandleFunc("GET /api/mcp", systemHandler.HandleListMCPServers)
	mux.HandleFunc("GET /api/commands", systemHandler.HandleListCommands)
	mux.HandleFunc("GET /api/commands/{name}", systemHandler.HandleGetCommand)
	mux.HandleFunc("POST /api/auth/apikey", systemHandler.HandleSetAPIKey)
	mux.HandleFunc("POST /api/permissions/{id}/grant", systemHandler.HandleGrantPermission)
	mux.HandleFunc("POST /api/permissions/{id}/deny", systemHandler.HandleDenyPermission)

	// Auth endpoints
	mux.HandleFunc("POST /api/auth/api-key", authHandler.HandleStoreAPIKey)
	mux.HandleFunc("DELETE /api/auth/{provider}", authHandler.HandleDeleteCredentials)
	mux.HandleFunc("GET /api/auth/status", authHandler.HandleAuthStatus)
	mux.HandleFunc("GET /api/auth/validate", authHandler.HandleValidatePreferredProvider)
	mux.HandleFunc("POST /api/auth/oauth/{provider}", authHandler.HandleStartOAuth)
	mux.HandleFunc("POST /api/auth/oauth-callback", authHandler.HandleOAuthCallback)

	// User preferences endpoints
	mux.HandleFunc("GET /api/preferences", preferencesHandler.HandleGetPreferences)
	mux.HandleFunc("POST /api/preferences", preferencesHandler.HandleUpdatePreferences)
	mux.HandleFunc("GET /api/preferences/providers", preferencesHandler.HandleGetAvailableProviders)
	mux.HandleFunc("POST /api/preferences/reset", preferencesHandler.HandleResetPreferences)

	// File management endpoints
	mux.HandleFunc("POST /api/sessions/{id}/files/upload", fileHandler.HandleUploadFile)
	mux.HandleFunc("GET /api/sessions/{id}/files", fileHandler.HandleListFiles)
	mux.HandleFunc("GET /api/sessions/{id}/files/{filename}", sessionAssetHandler.HandleServeFile)
	mux.HandleFunc("DELETE /api/sessions/{id}/files/{filename}", fileHandler.HandleDeleteFile)

	// SSE endpoints (always enabled)
	// SSE endpoint
	mux.HandleFunc("/stream", func(w http.ResponseWriter, r *http.Request) {
		t.Logf("Stream request received: %s %s", r.Method, r.URL.String())
		httphandlers.HandleSSEStream(ctx, testApp, w, r)
	})

	// Health check endpoint (always enabled)
	mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		healthData := map[string]interface{}{
			"status": "ok",
		}
		sendJSONResponse(w, http.StatusOK, healthData)
	})

	// Default handler for debugging
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		t.Logf("Unhandled request received: %s %s", r.Method, r.URL.String())
		w.WriteHeader(http.StatusBadRequest)
		if _, err := w.Write([]byte("400 Bad Request")); err != nil {
			t.Logf("Failed to write error response: %v", err)
		}
	})

	// Create test server
	server := httptest.NewServer(mux)

	return &TestServerResult{
		Server:    server,
		App:       testApp,
		SessionID: testSession.ID,
		ConfigDir: testConfigDir,
		DataDir:   testDataDir,
	}
}

// makeJSONRequest makes an HTTP request with JSON payload and returns the response
func makeJSONRequest(t *testing.T, server *httptest.Server, method, path string, payload interface{}) *http.Response {
	t.Helper()
	var body *bytes.Buffer
	if payload != nil {
		jsonData, err := json.Marshal(payload)
		if err != nil {
			t.Fatalf("Failed to marshal JSON payload: %v", err)
		}
		body = bytes.NewBuffer(jsonData)
	} else {
		body = bytes.NewBuffer(nil)
	}

	ctx := context.Background()
	req, err := http.NewRequestWithContext(ctx, method, server.URL+path, body)
	if err != nil {
		t.Fatalf("Failed to create HTTP request: %v", err)
	}

	if payload != nil {
		req.Header.Set("Content-Type", "application/json")
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("Failed to make HTTP request to %s %s: %v", method, path, err)
	}

	return resp
}

// sendJSONResponse sends a JSON response (local implementation for tests)
func sendJSONResponse(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(data); err != nil {
		// For tests, we can just log to stderr
		panic(fmt.Sprintf("Failed to encode JSON response: %v", err))
	}
}

// validateObjectResponse validates success response as object (flattened)
func validateObjectResponse(t *testing.T, resp *http.Response, expectedStatus int) map[string]interface{} {
	t.Helper()
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != expectedStatus {
		t.Fatalf("Expected status code %d, got %d", expectedStatus, resp.StatusCode)
	}

	var responseData map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&responseData); err != nil {
		t.Fatalf("Failed to decode object response: %v", err)
	}

	return responseData
}

// validateArrayResponse validates success response as array (flattened)
func validateArrayResponse(t *testing.T, resp *http.Response) []interface{} {
	t.Helper()
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("Expected status code %d, got %d", http.StatusOK, resp.StatusCode)
	}

	var responseData []interface{}
	if err := json.NewDecoder(resp.Body).Decode(&responseData); err != nil {
		t.Fatalf("Failed to decode array response: %v", err)
	}

	return responseData
}

// validateErrorResponse validates that response has proper structure and status for error responses (enveloped)
func validateErrorResponse(t *testing.T, resp *http.Response, expectedStatus int) httphandlers.ErrorResponse {
	t.Helper()
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != expectedStatus {
		t.Fatalf("Expected status code %d, got %d", expectedStatus, resp.StatusCode)
	}

	var errorResponse httphandlers.ErrorResponse
	if err := json.NewDecoder(resp.Body).Decode(&errorResponse); err != nil {
		t.Fatalf("Failed to decode error response: %v", err)
	}

	if errorResponse.Error == nil {
		t.Fatalf("Expected error response to have error field")
	}

	return errorResponse
}

// makeMultipartFileRequest creates and sends a multipart file upload request
func makeMultipartFileRequest(t *testing.T, server *httptest.Server, path, filename, content string) *http.Response {
	t.Helper()
	// Create multipart form
	var body bytes.Buffer
	writer := multipart.NewWriter(&body)

	// Create file part
	part, err := writer.CreateFormFile("file", filename)
	if err != nil {
		t.Fatalf("Failed to create form file: %v", err)
	}

	// Write file content
	_, err = part.Write([]byte(content))
	if err != nil {
		t.Fatalf("Failed to write file content: %v", err)
	}

	// Close writer to finalize multipart form
	err = writer.Close()
	if err != nil {
		t.Fatalf("Failed to close multipart writer: %v", err)
	}

	// Create request
	ctx := context.Background()
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, server.URL+path, &body)
	if err != nil {
		t.Fatalf("Failed to create multipart request: %v", err)
	}

	// Set proper content type for multipart
	req.Header.Set("Content-Type", writer.FormDataContentType())

	// Send request
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("Failed to send multipart request: %v", err)
	}

	return resp
}

// makeMultipartFileRequestFromBytes creates and sends a multipart file upload request with byte data
func makeMultipartFileRequestFromBytes(t *testing.T, server *httptest.Server, path, filename string, content []byte) *http.Response {
	t.Helper()
	// Create multipart form
	var body bytes.Buffer
	writer := multipart.NewWriter(&body)

	// Create file part
	part, err := writer.CreateFormFile("file", filename)
	if err != nil {
		t.Fatalf("Failed to create form file: %v", err)
	}

	// Write file content
	_, err = part.Write(content)
	if err != nil {
		t.Fatalf("Failed to write file content: %v", err)
	}

	// Close writer to finalize multipart form
	err = writer.Close()
	if err != nil {
		t.Fatalf("Failed to close multipart writer: %v", err)
	}

	// Create request
	ctx := context.Background()
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, server.URL+path, &body)
	if err != nil {
		t.Fatalf("Failed to create multipart request: %v", err)
	}

	// Set proper content type for multipart
	req.Header.Set("Content-Type", writer.FormDataContentType())

	// Send request
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("Failed to send multipart request: %v", err)
	}

	return resp
}

// SSEEvent represents a parsed Server-Sent Event
type SSEEvent struct {
	Type string                 `json:"type"`
	Data map[string]interface{} `json:"data"`
}

// connectSSE establishes a connection to the SSE stream for a given session
func connectSSE(t *testing.T, serverURL, sessionID string) (*http.Response, context.CancelFunc) {
	t.Helper()
	url := fmt.Sprintf("%s/stream?sessionId=%s", serverURL, sessionID)

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, http.NoBody)
	if err != nil {
		cancel()
		t.Fatalf("Failed to create SSE request: %v", err)
	}
	req.Header.Set("Accept", "text/event-stream")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		cancel()
		t.Fatalf("Failed to connect to SSE stream: %v", err)
	}

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		_ = resp.Body.Close()
		cancel()
		t.Fatalf("Expected status 200, got %d. Response: %s", resp.StatusCode, string(body))
	}

	return resp, cancel
}

// waitForEvents waits for and parses events from an SSE stream connection
func waitForEvents(t *testing.T, resp *http.Response, expectedMinEvents int, timeout time.Duration) []SSEEvent {
	t.Helper()
	var events []SSEEvent
	eventChan := make(chan SSEEvent, 10)

	// Start parsing events in background
	go func() {
		defer close(eventChan)
		scanner := bufio.NewScanner(resp.Body)

		var currentEvent SSEEvent
		for scanner.Scan() {
			line := strings.TrimSpace(scanner.Text())

			if line == "" {
				// Empty line indicates end of event
				if currentEvent.Type != "" {
					eventChan <- currentEvent
					currentEvent = SSEEvent{}
				}
				continue
			}

			if strings.HasPrefix(line, "event: ") {
				currentEvent.Type = strings.TrimPrefix(line, "event: ")
			} else if strings.HasPrefix(line, "data: ") {
				dataStr := strings.TrimPrefix(line, "data: ")
				var data map[string]interface{}
				if err := json.Unmarshal([]byte(dataStr), &data); err != nil {
					t.Logf("Failed to parse event data: %v, data: %s", err, dataStr)
					continue
				}
				currentEvent.Data = data
			}
		}

		// Handle last event if stream ended without empty line
		if currentEvent.Type != "" {
			eventChan <- currentEvent
		}
	}()

	// Collect events until we have enough or timeout
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	for {
		select {
		case event, ok := <-eventChan:
			if !ok {
				// Channel closed
				if len(events) >= expectedMinEvents {
					return events
				}
				t.Fatalf("Event stream closed, got %d events, expected at least %d", len(events), expectedMinEvents)
				return events
			}
			events = append(events, event)

			// Return early if we have enough events
			if len(events) >= expectedMinEvents {
				return events
			}

		case <-ctx.Done():
			t.Logf("Timeout reached, got %d events, expected at least %d", len(events), expectedMinEvents)
			for i, event := range events {
				t.Logf("Event %d: type=%s, data=%v", i, event.Type, event.Data)
			}
			if len(events) >= expectedMinEvents {
				return events
			}
			t.Fatalf("Timeout waiting for %d events after %v", expectedMinEvents, timeout)
			return events
		}
	}
}

// Exported wrappers for E2E tests to use

// RequireLLMCredentials skips tests that require LLM API access if credentials aren't configured
func RequireLLMCredentials(t *testing.T) {
	t.Helper()
	requireLLMCredentials(t)
}

// SetupIntegrationTestServer sets up a complete test environment for integration and E2E testing
func SetupIntegrationTestServer(t *testing.T) *TestServerResult {
	t.Helper()
	return setupIntegrationTestServer(t)
}

// MakeJSONRequest makes an HTTP request with JSON payload and returns the response
func MakeJSONRequest(t *testing.T, server *httptest.Server, method, path string, payload interface{}) *http.Response {
	t.Helper()
	return makeJSONRequest(t, server, method, path, payload)
}

// ValidateObjectResponse validates success response as object (flattened)
func ValidateObjectResponse(t *testing.T, resp *http.Response, expectedStatus int) map[string]interface{} {
	t.Helper()
	return validateObjectResponse(t, resp, expectedStatus)
}

// ValidateArrayResponse validates success response as array (flattened)
func ValidateArrayResponse(t *testing.T, resp *http.Response) []interface{} {
	t.Helper()
	return validateArrayResponse(t, resp)
}

// ConnectSSE establishes a connection to the SSE stream for a given session
func ConnectSSE(t *testing.T, serverURL, sessionID string) (*http.Response, context.CancelFunc) {
	t.Helper()
	return connectSSE(t, serverURL, sessionID)
}

// WaitForEvents waits for and parses events from an SSE stream connection
func WaitForEvents(t *testing.T, resp *http.Response, expectedMinEvents int, timeout time.Duration) []SSEEvent {
	t.Helper()
	return waitForEvents(t, resp, expectedMinEvents, timeout)
}
