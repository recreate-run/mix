package http

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/joho/godotenv"
	"mix/internal/app"
	"mix/internal/config"
	"mix/internal/db"
	session2 "mix/internal/session"

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

// loadEnvFile loads environment variables from .env file for testing
func loadEnvFile(t *testing.T) {
	t.Helper()

	// Test working directory is /mix_agent/internal/http, so .env is three levels up
	envPath := filepath.Join("..", "..", "..", ".env")

	// Load .env file - ignore error if file doesn't exist
	_ = godotenv.Load(envPath)
}

// initMCPTools mock implementation for testing
func initMCPTools(ctx context.Context, a *app.App) {
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

	// Create test session (title must be ≤20 chars due to DB constraint)
	session, err := testApp.Sessions.Create(ctx, "Test Session", "", "default", session2.SessionTypeMain, "", "", "", "local-browser-service", "")
	if err != nil {
		t.Fatalf("Failed to create test session: %v", err)
	}

	// Create REST handlers
	sessionHandler := NewSessionHandler(testApp)
	messageHandler := NewMessageHandler(testApp)
	systemHandler := NewSystemHandler(testApp)
	preferencesHandler := NewPreferencesHandler(testApp)

	// Create file management handlers
	fileHandler := NewFileHandler(testApp)
	sessionAssetHandler := NewSessionAssetHandler(testApp, testApp.StorageConfig)

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
	mux.HandleFunc("GET /api/messages/history", messageHandler.HandleMessageHistory)
	mux.HandleFunc("POST /api/sessions/{id}/cancel", messageHandler.HandleCancelAgent)

	// System endpoints
	mux.HandleFunc("GET /api/mcp", systemHandler.HandleListMCPServers)
	mux.HandleFunc("GET /api/commands", systemHandler.HandleListCommands)
	mux.HandleFunc("GET /api/commands/{name}", systemHandler.HandleGetCommand)
	mux.HandleFunc("POST /api/auth/apikey", systemHandler.HandleSetAPIKey)
	mux.HandleFunc("POST /api/permissions/{id}/grant", systemHandler.HandleGrantPermission)
	mux.HandleFunc("POST /api/permissions/{id}/deny", systemHandler.HandleDenyPermission)

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
		HandleSSEStream(ctx, testApp, w, r)
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
		_, _ = w.Write([]byte("400 Bad Request"))
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

// validateObjectResponse validates success response as object (flattened)
//
//nolint:unparam // expectedStatus may vary in future tests
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

