package http

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"

	"mix/internal/app"
	"mix/internal/config"
	"mix/internal/db"

	_ "github.com/ncruces/go-sqlite3/driver"
	_ "github.com/ncruces/go-sqlite3/embed"
)


// TestServerResult contains the initialized test server components
type TestServerResult struct {
	Server     *httptest.Server
	App        *app.App
	SessionID  string
	ConfigDir  string
	DataDir    string
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
	os.Setenv("WORKING_DIR", testDataDir)                        // Set WORKING_DIR for asset server
	os.Setenv("GSAP_GLOBAL_DIR", testDataDir+"/gsap_animations") // Set GSAP_GLOBAL_DIR for animations

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
	conn, err := db.Connect(ctx)
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

	// Create test session with default working directory
	session, err := testApp.Sessions.Create(ctx, "Test Integration Session", testDataDir)
	if err != nil {
		t.Fatalf("Failed to create test session: %v", err)
	}

	// Create REST handlers
	sessionHandler := NewSessionHandler(testApp)
	messageHandler := NewMessageHandler(testApp)
	systemHandler := NewSystemHandler(testApp)

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

	// SSE endpoints (always enabled)
	// SSE endpoint
	mux.HandleFunc("/stream", func(w http.ResponseWriter, r *http.Request) {
		t.Logf("Stream request received: %s %s", r.Method, r.URL.String())
		HandleSSEStream(ctx, testApp, w, r)
	})
	
	// Stream sub-path endpoint for SSE with paths
	mux.HandleFunc("/stream/", func(w http.ResponseWriter, r *http.Request) {
		t.Logf("Stream sub-path request received: %s %s", r.Method, r.URL.String())
		if strings.HasSuffix(r.URL.Path, "/message") {
			HandleMessageQueue(w, r)
		} else {
			// Handle other stream sub-paths by streaming events
			HandleSSEStream(ctx, testApp, w, r)
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

// validateRESTResponse validates that response has proper structure and status
func validateRESTResponse(t *testing.T, resp *http.Response, expectedStatus int) RESTResponse {
	defer resp.Body.Close()

	if resp.StatusCode != expectedStatus {
		t.Fatalf("Expected status code %d, got %d", expectedStatus, resp.StatusCode)
	}

	var restResponse RESTResponse
	if err := json.NewDecoder(resp.Body).Decode(&restResponse); err != nil {
		t.Fatalf("Failed to decode REST response: %v", err)
	}

	return restResponse
}

// createJSONMessage creates a proper JSON message structure for testing
func createJSONMessage(text string) string {
	msgContent := map[string]interface{}{
		"text": text,
	}
	jsonData, _ := json.Marshal(msgContent)
	return string(jsonData)
}