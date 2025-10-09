package http

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"mix/internal/app"
	"mix/internal/config"
	"mix/internal/db"
	"mix/internal/message"

	_ "github.com/ncruces/go-sqlite3/driver"
	_ "github.com/ncruces/go-sqlite3/embed"
)

// setupTestServerForFork sets up test environment specifically for fork testing
func setupTestServerForFork(t *testing.T) (*app.App, string) {
	// Set up test configuration properly
	testConfigDir := "/tmp/test-mix-fork-" + t.Name()
	testDataDir := "/tmp/test-mix-data-fork-" + t.Name()

	_ = os.Setenv("_CONFIG_DIR", testConfigDir)
	_ = os.Setenv("_DATA_DIR", testDataDir)

	// Create test directories
	if err := os.MkdirAll(testConfigDir, 0755); err != nil {
		t.Fatalf("Failed to create test config dir: %v", err)
	}
	if err := os.MkdirAll(testDataDir, 0755); err != nil {
		t.Fatalf("Failed to create test data dir: %v", err)
	}

	// Initialize config for testing (database-only, no config file needed)
	if _, err := config.Load(testConfigDir, false, false); err != nil {
		t.Fatalf("Failed to load config: %v", err)
	}

	// Use the standard database connection method so everything is consistent
	ctx := context.Background()
	conn, err := db.Connect(ctx, ".mix")
	if err != nil {
		t.Fatalf("Failed to connect to test database: %v", err)
	}

	// Create test app
	testApp, err := app.New(ctx, conn)
	if err != nil {
		t.Fatalf("Failed to create test app: %v", err)
	}

	// Initialize MCP tools like the real app does
	initMCPTools(ctx, testApp)

	// Create test session
	session, err := testApp.Sessions.Create(ctx, "Test Fork Session", "", "default")
	if err != nil {
		t.Fatalf("Failed to create test session: %v", err)
	}

	return testApp, session.ID
}

// createTestMessages creates sample messages for fork testing
func createTestMessages(t *testing.T, app *app.App, sessionID string, messageCount int) []message.Message {
	ctx := context.Background()
	var messages []message.Message

	for i := 0; i < messageCount; i++ {
		// Create user message
		userMsg, err := app.Messages.Create(ctx, sessionID, message.CreateMessageParams{
			Role: message.User,
			Parts: []message.ContentPart{
				message.TextContent{Text: "User message " + string(rune('A'+i))},
			},
			Model: "claude-4-sonnet",
		})
		if err != nil {
			t.Fatalf("Failed to create user message %d: %v", i, err)
		}
		messages = append(messages, userMsg)

		// Create assistant response
		assistantMsg, err := app.Messages.Create(ctx, sessionID, message.CreateMessageParams{
			Role: message.Assistant,
			Parts: []message.ContentPart{
				message.TextContent{Text: "Assistant response " + string(rune('A'+i))},
			},
			Model: "claude-4-sonnet",
		})
		if err != nil {
			t.Fatalf("Failed to create assistant message %d: %v", i, err)
		}
		messages = append(messages, assistantMsg)
	}

	return messages
}

// validateForkResult validates the fork operation results
func validateForkResult(t *testing.T, app *app.App, sourceSessionID, forkedSessionID string, expectedMessageCount int) {
	ctx := context.Background()

	// Validate forked session exists and has correct parent
	forkedSession, err := app.Sessions.Get(ctx, forkedSessionID)
	if err != nil {
		t.Fatalf("Failed to get forked session: %v", err)
	}

	if forkedSession.ParentSessionID != sourceSessionID {
		t.Errorf("Expected parent session ID %s, got %s", sourceSessionID, forkedSession.ParentSessionID)
	}

	// Validate source session still exists
	_, err = app.Sessions.Get(ctx, sourceSessionID)
	if err != nil {
		t.Fatalf("Failed to get source session: %v", err)
	}

	// Note: Working directory validation removed - sessions now use centralized storage

	// Validate message copying
	forkedMessages, err := app.Messages.List(ctx, forkedSessionID)
	if err != nil {
		t.Fatalf("Failed to list forked session messages: %v", err)
	}

	if len(forkedMessages) != expectedMessageCount {
		t.Errorf("Expected %d messages in forked session, got %d", expectedMessageCount, len(forkedMessages))
	}

	// Validate messages have different IDs but same content
	sourceMessages, err := app.Messages.List(ctx, sourceSessionID)
	if err != nil {
		t.Fatalf("Failed to list source session messages: %v", err)
	}

	for i := 0; i < expectedMessageCount && i < len(sourceMessages) && i < len(forkedMessages); i++ {
		sourceMsg := sourceMessages[i]
		forkedMsg := forkedMessages[i]

		// IDs should be different
		if sourceMsg.ID == forkedMsg.ID {
			t.Errorf("Message %d: forked message should have different ID than source", i)
		}

		// Session IDs should be different
		if forkedMsg.SessionID != forkedSessionID {
			t.Errorf("Message %d: forked message should belong to forked session", i)
		}

		// Content should be the same
		if sourceMsg.Content().String() != forkedMsg.Content().String() {
			t.Errorf("Message %d: content mismatch between source and forked message", i)
		}

		// Role should be the same
		if sourceMsg.Role != forkedMsg.Role {
			t.Errorf("Message %d: role mismatch between source (%s) and forked (%s) message",
				i, sourceMsg.Role, forkedMsg.Role)
		}
	}
}

func TestSessionFork(t *testing.T) {
	app, sourceSessionID := setupTestServerForFork(t)

	// Create test messages (3 pairs = 6 total messages)
	messages := createTestMessages(t, app, sourceSessionID, 3)
	t.Logf("Created %d test messages in source session", len(messages))

	// Create REST handler and test server
	sessionHandler := NewSessionHandler(app)
	mux := http.NewServeMux()
	mux.HandleFunc("POST /api/sessions/{id}/fork", sessionHandler.HandleForkSession)
	server := httptest.NewServer(mux)
	defer server.Close()

	// Test forking at message index 4 (should copy first 4 messages)
	forkParams := ForkSessionRequest{
		MessageIndex:    int64(4),
		Title:          "Forked Test Session",
	}

	paramsJSON, err := json.Marshal(forkParams)
	if err != nil {
		t.Fatalf("Failed to marshal fork params: %v", err)
	}

	// Make REST API call to fork session
	url := server.URL + "/api/sessions/" + sourceSessionID + "/fork"
	resp, err := http.Post(url, "application/json", bytes.NewBuffer(paramsJSON))
	if err != nil {
		t.Fatalf("Failed to make fork request: %v", err)
	}
	defer func() { _ = resp.Body.Close() }()

	// Validate response and extract session data (flattened response)
	sessionDataMap := validateObjectResponse(t, resp, http.StatusCreated)

	forkedSessionID, ok := sessionDataMap["id"].(string)
	if !ok {
		t.Fatalf("Expected session ID in response data")
	}

	title, ok := sessionDataMap["title"].(string)
	if !ok {
		t.Fatalf("Expected title in response data")
	}

	t.Logf("Fork successful: created session %s with title '%s'", forkedSessionID, title)

	// Validate fork result
	validateForkResult(t, app, sourceSessionID, forkedSessionID, 4)

	// Validate response data
	if title != "Forked Test Session" {
		t.Errorf("Expected title 'Forked Test Session', got '%s'", title)
	}

	// Note: Working directory field removed - sessions use session-based storage now
}

func TestSessionForkWithDefaultTitle(t *testing.T) {
	app, sourceSessionID := setupTestServerForFork(t)

	// Create test messages
	createTestMessages(t, app, sourceSessionID, 2)

	// Create REST handler and test server
	sessionHandler := NewSessionHandler(app)
	mux := http.NewServeMux()
	mux.HandleFunc("POST /api/sessions/{id}/fork", sessionHandler.HandleForkSession)
	server := httptest.NewServer(mux)
	defer server.Close()

	// Test forking without custom title
	forkParams := ForkSessionRequest{
		MessageIndex:    int64(2),
		// No title - should use default
	}

	paramsJSON, err := json.Marshal(forkParams)
	if err != nil {
		t.Fatalf("Failed to marshal fork params: %v", err)
	}

	// Make REST API call to fork session
	url := server.URL + "/api/sessions/" + sourceSessionID + "/fork"
	resp, err := http.Post(url, "application/json", bytes.NewBuffer(paramsJSON))
	if err != nil {
		t.Fatalf("Failed to make fork request: %v", err)
	}
	defer func() { _ = resp.Body.Close() }()

	// Validate response and extract session data (flattened response)
	sessionDataMap := validateObjectResponse(t, resp, http.StatusCreated)

	title, ok := sessionDataMap["title"].(string)
	if !ok {
		t.Fatalf("Expected title in response data")
	}

	// Should use default title
	if title != "Forked Session" {
		t.Errorf("Expected default title 'Forked Session', got '%s'", title)
	}

	forkedSessionID, ok := sessionDataMap["id"].(string)
	if !ok {
		t.Fatalf("Expected session ID in response data")
	}

	validateForkResult(t, app, sourceSessionID, forkedSessionID, 2)
}

func TestSessionForkErrorHandling(t *testing.T) {
	app, _ := setupTestServerForFork(t)

	// Create REST handler and test server
	sessionHandler := NewSessionHandler(app)
	mux := http.NewServeMux()
	mux.HandleFunc("POST /api/sessions/{id}/fork", sessionHandler.HandleForkSession)
	server := httptest.NewServer(mux)
	defer server.Close()

	testCases := []struct {
		name        string
		request     ForkSessionRequest
		expectError bool
		statusCode  int
		errorType   string
	}{
		{
			name: "non-existent source session ID",
			request: ForkSessionRequest{
				MessageIndex: int64(2),
				// Using dummy session ID that doesn't exist
			},
			expectError: true,
			statusCode:  500,
			errorType:   "internal_error",
		},
		{
			name: "invalid source session ID",
			request: ForkSessionRequest{
				MessageIndex:    int64(2),
			},
			expectError: true,
			statusCode:  500,
			errorType:   "internal_error",
		},
		{
			name: "negative message index",
			request: ForkSessionRequest{
				MessageIndex:    int64(-1),
			},
			expectError: true,
			statusCode:  400,
			errorType:   "validation_error",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			paramsJSON, err := json.Marshal(tc.request)
			if err != nil {
				t.Fatalf("Failed to marshal params: %v", err)
			}

			// Make REST API call
			url := server.URL + "/api/sessions/dummy/fork"
			resp, err := http.Post(url, "application/json", bytes.NewBuffer(paramsJSON))
			if err != nil {
				t.Fatalf("Failed to make fork request: %v", err)
			}
			defer func() { _ = resp.Body.Close() }()

			// Validate response status code
			if resp.StatusCode != tc.statusCode {
				t.Errorf("Expected status code %d, got %d", tc.statusCode, resp.StatusCode)
			}

			if tc.expectError {
				// Validate error response (enveloped)
				errorResponse := validateErrorResponse(t, resp, tc.statusCode)

				if errorResponse.Error.Type != tc.errorType {
					t.Errorf("Expected error type '%s', got '%s'", tc.errorType, errorResponse.Error.Type)
				}
			} else {
				// Should be successful
				if resp.StatusCode >= 400 {
					t.Errorf("Expected successful response, got status %d", resp.StatusCode)
				}
			}
		})
	}
}

func TestSessionForkMessageBoundary(t *testing.T) {
	app, sourceSessionID := setupTestServerForFork(t)
	ctx := context.Background()

	// Create exactly 5 messages
	createTestMessages(t, app, sourceSessionID, 2) // Creates 4 messages
	// Add one more user message to make it 5 total
	_, err := app.Messages.Create(ctx, sourceSessionID, message.CreateMessageParams{
		Role: message.User,
		Parts: []message.ContentPart{
			message.TextContent{Text: "Final user message"},
		},
		Model: "claude-4-sonnet",
	})
	if err != nil {
		t.Fatalf("Failed to create final message: %v", err)
	}

	// Create REST handler and test server
	sessionHandler := NewSessionHandler(app)
	mux := http.NewServeMux()
	mux.HandleFunc("POST /api/sessions/{id}/fork", sessionHandler.HandleForkSession)
	server := httptest.NewServer(mux)
	defer server.Close()

	// Test forking at exact message boundary
	forkParams := ForkSessionRequest{
		MessageIndex:    int64(5), // Should copy all 5 messages
		Title:          "Boundary Fork Test",
	}

	paramsJSON, err := json.Marshal(forkParams)
	if err != nil {
		t.Fatalf("Failed to marshal fork params: %v", err)
	}

	// Make REST API call
	url := server.URL + "/api/sessions/" + sourceSessionID + "/fork"
	resp, err := http.Post(url, "application/json", bytes.NewBuffer(paramsJSON))
	if err != nil {
		t.Fatalf("Failed to make fork request: %v", err)
	}
	defer func() { _ = resp.Body.Close() }()

	// Validate response and extract session data (flattened response)
	sessionDataMap := validateObjectResponse(t, resp, http.StatusCreated)

	forkedSessionID, ok := sessionDataMap["id"].(string)
	if !ok {
		t.Fatalf("Expected session ID in response data")
	}

	// Should copy exactly 5 messages
	validateForkResult(t, app, sourceSessionID, forkedSessionID, 5)
}

func TestSessionForkWithZeroMessages(t *testing.T) {
	app, sourceSessionID := setupTestServerForFork(t)

	// Create test messages
	createTestMessages(t, app, sourceSessionID, 2) // Creates 4 messages

	// Create REST handler and test server
	sessionHandler := NewSessionHandler(app)
	mux := http.NewServeMux()
	mux.HandleFunc("POST /api/sessions/{id}/fork", sessionHandler.HandleForkSession)
	server := httptest.NewServer(mux)
	defer server.Close()

	// Test forking at message index 0 (should copy 0 messages, empty session)
	forkParams := ForkSessionRequest{
		MessageIndex:    int64(0),
		Title:          "Empty Fork Test",
	}

	paramsJSON, err := json.Marshal(forkParams)
	if err != nil {
		t.Fatalf("Failed to marshal fork params: %v", err)
	}

	// Make REST API call
	url := server.URL + "/api/sessions/" + sourceSessionID + "/fork"
	resp, err := http.Post(url, "application/json", bytes.NewBuffer(paramsJSON))
	if err != nil {
		t.Fatalf("Failed to make fork request: %v", err)
	}
	defer func() { _ = resp.Body.Close() }()

	// Validate response and extract session data (flattened response)
	sessionDataMap := validateObjectResponse(t, resp, http.StatusCreated)

	forkedSessionID, ok := sessionDataMap["id"].(string)
	if !ok {
		t.Fatalf("Expected session ID in response data")
	}

	// Should copy exactly 0 messages (empty session)
	validateForkResult(t, app, sourceSessionID, forkedSessionID, 0)
}