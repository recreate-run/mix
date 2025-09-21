package integration_tests

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"

	"mix/internal/app"
	"mix/internal/config"
	"mix/internal/db"
	httphandlers "mix/internal/http"
	_ "mix/internal/llm/models"

	_ "github.com/ncruces/go-sqlite3/driver"
	_ "github.com/ncruces/go-sqlite3/embed"
)

// TestServerResult contains the initialized test server components
type TestServerResult struct {
	Server    *httptest.Server
	App       *app.App
	SessionID string
	ConfigDir string
	DataDir   string
}

// initMCPTools mock implementation for testing
func initMCPTools(ctx context.Context, app *app.App) {
	// Mock implementation - in real app this initializes MCP tools
	// For tests, we just need to ensure the app doesn't crash
}

// setupIntegrationTestServer sets up a complete test environment for integration testing
// This function consolidates the common setup logic shared between REST and SSE tests
func setupIntegrationTestServer(t *testing.T) *TestServerResult {
	// Auto-generate test name from test function name
	testName := t.Name()

	// Set up isolated test environment
	testConfigDir := "/tmp/test-mix-" + testName
	testDataDir := "/tmp/test-mix-data-" + testName

	os.Setenv("_CONFIG_DIR", testConfigDir)
	os.Setenv("_DATA_DIR", testDataDir)

	// Create test directories
	if err := os.MkdirAll(testConfigDir, 0755); err != nil {
		t.Fatalf("Failed to create test config dir: %v", err)
	}
	if err := os.MkdirAll(testDataDir, 0755); err != nil {
		t.Fatalf("Failed to create test data dir: %v", err)
	}
	if err := os.MkdirAll(testDataDir+"/gsap_animations", 0755); err != nil {
		t.Fatalf("Failed to create GSAP animations dir: %v", err)
	}

	// Create minimal test configuration
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

	// Initialize configuration
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

	// Create test session
	session, err := testApp.Sessions.Create(ctx, "Test Integration Session")
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
	fileHandler := httphandlers.NewFileHandler(testApp, testApp.StorageConfig)
	sessionAssetHandler := httphandlers.NewSessionAssetHandler(testApp, testApp.StorageConfig)

	// Set up HTTP multiplexer with all REST endpoints
	mux := http.NewServeMux()

	// Session management endpoints
	mux.HandleFunc("GET /api/sessions", sessionHandler.HandleListSessions)
	mux.HandleFunc("POST /api/sessions", sessionHandler.HandleCreateSession)
	mux.HandleFunc("GET /api/sessions/{id}", sessionHandler.HandleGetSession)
	mux.HandleFunc("DELETE /api/sessions/{id}", sessionHandler.HandleDeleteSession)
	mux.HandleFunc("POST /api/sessions/{id}/fork", sessionHandler.HandleForkSession)

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
	mux.HandleFunc("POST /api/auth/login", systemHandler.HandleAuthLogin)
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

	// Stream sub-path endpoint for SSE with paths
	mux.HandleFunc("/stream/", func(w http.ResponseWriter, r *http.Request) {
		t.Logf("Stream sub-path request received: %s %s", r.Method, r.URL.String())
		if strings.HasSuffix(r.URL.Path, "/message") {
			httphandlers.HandleMessageQueue(w, r)
		} else {
			// Handle other stream sub-paths by streaming events
			httphandlers.HandleSSEStream(ctx, testApp, w, r)
		}
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
		w.Write([]byte("400 Bad Request"))
	})

	// Create test server
	server := httptest.NewServer(mux)

	return &TestServerResult{
		Server:    server,
		App:       testApp,
		SessionID: session.ID,
		ConfigDir: testConfigDir,
		DataDir:   testDataDir,
	}
}

// makeJSONRequest makes an HTTP request with JSON payload and returns the response
func makeJSONRequest(t *testing.T, server *httptest.Server, method, path string, payload interface{}) *http.Response {
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

	req, err := http.NewRequest(method, server.URL+path, body)
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
	defer resp.Body.Close()

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
func validateArrayResponse(t *testing.T, resp *http.Response, expectedStatus int) []interface{} {
	defer resp.Body.Close()

	if resp.StatusCode != expectedStatus {
		t.Fatalf("Expected status code %d, got %d", expectedStatus, resp.StatusCode)
	}

	var responseData []interface{}
	if err := json.NewDecoder(resp.Body).Decode(&responseData); err != nil {
		t.Fatalf("Failed to decode array response: %v", err)
	}

	return responseData
}

// mockOAuthFlow creates a mock OAuth flow for testing purposes
func mockOAuthFlow(t *testing.T, server *httptest.Server, provider string) (string, string) {
	// Generate a state token and auth URL
	stateToken := "mock-oauth-state-" + provider
	authURL := "https://example.com/oauth/authorize?client_id=mock&state=" + stateToken

	// Return the mock state token and auth URL
	return stateToken, authURL
}

// validateErrorResponse validates that response has proper structure and status for error responses (enveloped)
func validateErrorResponse(t *testing.T, resp *http.Response, expectedStatus int) httphandlers.ErrorResponse {
	defer resp.Body.Close()

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

// createJSONMessage creates a proper JSON message structure for testing
func createJSONMessage(text string) string {
	msgContent := map[string]interface{}{
		"text": text,
	}
	jsonData, _ := json.Marshal(msgContent)
	return string(jsonData)
}

// makeMultipartFileRequest creates and sends a multipart file upload request
func makeMultipartFileRequest(t *testing.T, server *httptest.Server, path, filename, content string) *http.Response {
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
	req, err := http.NewRequest("POST", server.URL+path, &body)
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
	req, err := http.NewRequest("POST", server.URL+path, &body)
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
