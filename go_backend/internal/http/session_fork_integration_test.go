package http

import (
	"context"
	"encoding/json"
	"os"
	"testing"

	"mix/internal/api"
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

	// Use the standard database connection method so everything is consistent
	ctx := context.Background()
	conn, err := db.Connect(ctx)
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
	session, err := testApp.Sessions.Create(ctx, "Test Fork Session", "/tmp/test-workdir")
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
	sourceSession, err := app.Sessions.Get(ctx, sourceSessionID)
	if err != nil {
		t.Fatalf("Failed to get source session: %v", err)
	}

	// Validate working directory inheritance
	if forkedSession.WorkingDirectory != sourceSession.WorkingDirectory {
		t.Errorf("Expected forked session to inherit working directory %s, got %s",
			sourceSession.WorkingDirectory, forkedSession.WorkingDirectory)
	}

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
	ctx := context.Background()

	// Create test messages (3 pairs = 6 total messages)
	messages := createTestMessages(t, app, sourceSessionID, 3)
	t.Logf("Created %d test messages in source session", len(messages))

	// Create query handler
	handler := api.NewQueryHandler(app)

	// Test forking at message index 4 (should copy first 4 messages)
	forkParams := map[string]interface{}{
		"sourceSessionId": sourceSessionID,
		"messageIndex":    int64(4),
		"title":          "Forked Test Session",
	}

	paramsJSON, err := json.Marshal(forkParams)
	if err != nil {
		t.Fatalf("Failed to marshal fork params: %v", err)
	}

	request := &api.QueryRequest{
		Method: "sessions.fork",
		Params: paramsJSON,
		ID:     1,
	}

	// Execute fork operation
	response := handler.Handle(ctx, request)

	// Validate response
	if response.Error != nil {
		t.Fatalf("Fork operation failed: %s", response.Error.Message)
	}

	// Extract forked session data
	sessionData, ok := response.Result.(api.SessionData)
	if !ok {
		t.Fatalf("Expected SessionData in response, got %T", response.Result)
	}

	t.Logf("Fork successful: created session %s with title '%s'", sessionData.ID, sessionData.Title)

	// Validate fork result
	validateForkResult(t, app, sourceSessionID, sessionData.ID, 4)

	// Validate response data
	if sessionData.Title != "Forked Test Session" {
		t.Errorf("Expected title 'Forked Test Session', got '%s'", sessionData.Title)
	}

	if sessionData.WorkingDirectory == "" {
		t.Error("Expected forked session to have working directory")
	}
}

func TestSessionForkWithDefaultTitle(t *testing.T) {
	app, sourceSessionID := setupTestServerForFork(t)
	ctx := context.Background()

	// Create test messages
	createTestMessages(t, app, sourceSessionID, 2)

	// Create query handler
	handler := api.NewQueryHandler(app)

	// Test forking without custom title
	forkParams := map[string]interface{}{
		"sourceSessionId": sourceSessionID,
		"messageIndex":    int64(2),
	}

	paramsJSON, err := json.Marshal(forkParams)
	if err != nil {
		t.Fatalf("Failed to marshal fork params: %v", err)
	}

	request := &api.QueryRequest{
		Method: "sessions.fork",
		Params: paramsJSON,
		ID:     1,
	}

	// Execute fork operation
	response := handler.Handle(ctx, request)

	// Validate response
	if response.Error != nil {
		t.Fatalf("Fork operation failed: %s", response.Error.Message)
	}

	// Extract forked session data
	sessionData, ok := response.Result.(api.SessionData)
	if !ok {
		t.Fatalf("Expected SessionData in response, got %T", response.Result)
	}

	// Should use default title
	if sessionData.Title != "Forked Session" {
		t.Errorf("Expected default title 'Forked Session', got '%s'", sessionData.Title)
	}

	validateForkResult(t, app, sourceSessionID, sessionData.ID, 2)
}

func TestSessionForkErrorHandling(t *testing.T) {
	app, _ := setupTestServerForFork(t)
	ctx := context.Background()
	handler := api.NewQueryHandler(app)

	testCases := []struct {
		name        string
		params      map[string]interface{}
		expectError bool
		errorMsg    string
	}{
		{
			name: "missing source session ID",
			params: map[string]interface{}{
				"messageIndex": int64(2),
			},
			expectError: true,
			errorMsg:    "Missing required parameter: sourceSessionId",
		},
		{
			name: "invalid source session ID",
			params: map[string]interface{}{
				"sourceSessionId": "invalid-session-id",
				"messageIndex":    int64(2),
			},
			expectError: true,
			errorMsg:    "Failed to fork session: sql: no rows in result set",
		},
		{
			name: "missing message index",
			params: map[string]interface{}{
				"sourceSessionId": "some-session-id",
			},
			expectError: true,
			errorMsg:    "Missing required parameter: messageIndex must be > 0",
		},
		{
			name: "zero message index",
			params: map[string]interface{}{
				"sourceSessionId": "some-session-id",
				"messageIndex":    int64(0),
			},
			expectError: true,
			errorMsg:    "Missing required parameter: messageIndex must be > 0",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			paramsJSON, err := json.Marshal(tc.params)
			if err != nil {
				t.Fatalf("Failed to marshal params: %v", err)
			}

			request := &api.QueryRequest{
				Method: "sessions.fork",
				Params: paramsJSON,
				ID:     1,
			}

			response := handler.Handle(ctx, request)

			if tc.expectError {
				if response.Error == nil {
					t.Errorf("Expected error, but got success")
				} else if response.Error.Message != tc.errorMsg {
					t.Errorf("Expected error message '%s', got '%s'", tc.errorMsg, response.Error.Message)
				}
			} else {
				if response.Error != nil {
					t.Errorf("Unexpected error: %s", response.Error.Message)
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

	handler := api.NewQueryHandler(app)

	// Test forking at exact message boundary
	forkParams := map[string]interface{}{
		"sourceSessionId": sourceSessionID,
		"messageIndex":    int64(5), // Should copy all 5 messages
		"title":          "Boundary Fork Test",
	}

	paramsJSON, err := json.Marshal(forkParams)
	if err != nil {
		t.Fatalf("Failed to marshal fork params: %v", err)
	}

	request := &api.QueryRequest{
		Method: "sessions.fork",
		Params: paramsJSON,
		ID:     1,
	}

	response := handler.Handle(ctx, request)

	if response.Error != nil {
		t.Fatalf("Fork operation failed: %s", response.Error.Message)
	}

	sessionData, ok := response.Result.(api.SessionData)
	if !ok {
		t.Fatalf("Expected SessionData in response, got %T", response.Result)
	}

	// Should copy exactly 5 messages
	validateForkResult(t, app, sourceSessionID, sessionData.ID, 5)
}