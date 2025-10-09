package http

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"mix/internal/app"
	"mix/internal/config"
	"mix/internal/db"
	"mix/internal/message"

	_ "github.com/ncruces/go-sqlite3/driver"
	_ "github.com/ncruces/go-sqlite3/embed"
)

// setupTestServerForRewind sets up test environment specifically for rewind testing
func setupTestServerForRewind(t *testing.T) (*app.App, string) {
	// Set up test configuration properly
	testConfigDir := "/tmp/test-mix-rewind-" + t.Name()
	testDataDir := "/tmp/test-mix-data-rewind-" + t.Name()

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
	session, err := testApp.Sessions.Create(ctx, "Test Rewind Session", "", "default")
	if err != nil {
		t.Fatalf("Failed to create test session: %v", err)
	}

	return testApp, session.ID
}

// createTestMessagesForRewind creates sample messages for rewind testing
func createTestMessagesForRewind(t *testing.T, app *app.App, sessionID string, messageCount int) []message.Message {
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

func TestSessionRewindBasic(t *testing.T) {
	app, sessionID := setupTestServerForRewind(t)

	// Create 6 test messages (3 pairs)
	messages := createTestMessagesForRewind(t, app, sessionID, 3)
	t.Logf("Created %d test messages in session", len(messages))

	// Create REST handler and test server
	sessionHandler := NewSessionHandler(app)
	mux := http.NewServeMux()
	mux.HandleFunc("POST /api/sessions/{id}/rewind", sessionHandler.HandleRewindSession)
	server := httptest.NewServer(mux)
	defer server.Close()

	// Rewind to message 3 (keep first 4 messages, delete last 2)
	rewindParams := RewindSessionRequest{
		MessageID:    messages[3].ID, // Keep up to and including message at index 3
		CleanupMedia: false,          // No media in this test
	}

	paramsJSON, err := json.Marshal(rewindParams)
	if err != nil {
		t.Fatalf("Failed to marshal rewind params: %v", err)
	}

	// Make REST API call to rewind session
	url := server.URL + "/api/sessions/" + sessionID + "/rewind"
	resp, err := http.Post(url, "application/json", bytes.NewBuffer(paramsJSON))
	if err != nil {
		t.Fatalf("Failed to make rewind request: %v", err)
	}
	defer func() { _ = resp.Body.Close() }()

	// Validate response
	sessionDataMap := validateObjectResponse(t, resp, http.StatusOK)

	returnedSessionID, ok := sessionDataMap["id"].(string)
	if !ok || returnedSessionID != sessionID {
		t.Errorf("Expected session ID %s, got %v", sessionID, sessionDataMap["id"])
	}

	// Verify messages were deleted
	ctx := context.Background()
	remainingMessages, err := app.Messages.List(ctx, sessionID)
	if err != nil {
		t.Fatalf("Failed to list messages after rewind: %v", err)
	}

	expectedCount := 4
	if len(remainingMessages) != expectedCount {
		t.Errorf("Expected %d messages after rewind, got %d", expectedCount, len(remainingMessages))
	}

	// Verify correct messages remain (indices 0-3)
	for i := 0; i < expectedCount; i++ {
		expectedContent := messages[i].Content().String()
		actualContent := remainingMessages[i].Content().String()
		if expectedContent != actualContent {
			t.Errorf("Message %d content mismatch: expected %q, got %q", i, expectedContent, actualContent)
		}
	}

	t.Logf("Rewind successful: kept %d messages", len(remainingMessages))
}

func TestSessionRewindWithMediaCleanup(t *testing.T) {
	app, sessionID := setupTestServerForRewind(t)
	ctx := context.Background()

	// Create a message with image content
	_, err := app.Messages.Create(ctx, sessionID, message.CreateMessageParams{
		Role: message.User,
		Parts: []message.ContentPart{
			message.TextContent{Text: "User message with image"},
		},
		Model: "claude-4-sonnet",
	})
	if err != nil {
		t.Fatalf("Failed to create user message: %v", err)
	}

	// Create assistant message with image URL
	testImagePath := "test_image.jpg"
	_, err = app.Messages.Create(ctx, sessionID, message.CreateMessageParams{
		Role: message.Assistant,
		Parts: []message.ContentPart{
			message.TextContent{Text: "Here's an image"},
			message.ImageURLContent{URL: testImagePath},
		},
		Model: "claude-4-sonnet",
	})
	if err != nil {
		t.Fatalf("Failed to create assistant message: %v", err)
	}

	// Create the test image file in session storage
	sessionStorageDir := filepath.Join(os.Getenv("_DATA_DIR"), "storage", sessionID)
	if err := os.MkdirAll(sessionStorageDir, 0755); err != nil {
		t.Fatalf("Failed to create session storage directory: %v", err)
	}
	testImageFullPath := filepath.Join(sessionStorageDir, testImagePath)
	err = os.WriteFile(testImageFullPath, []byte("test image data"), 0644)
	if err != nil {
		t.Fatalf("Failed to create test image: %v", err)
	}

	// Verify image exists
	if _, err := os.Stat(testImageFullPath); os.IsNotExist(err) {
		t.Fatalf("Test image was not created")
	}

	// Create REST handler and test server
	sessionHandler := NewSessionHandler(app)
	mux := http.NewServeMux()
	mux.HandleFunc("POST /api/sessions/{id}/rewind", sessionHandler.HandleRewindSession)
	server := httptest.NewServer(mux)
	defer server.Close()

	// Get the first message ID for rewinding
	messages, err := app.Messages.List(ctx, sessionID)
	if err != nil {
		t.Fatalf("Failed to list messages: %v", err)
	}

	// Rewind to first message (keep only first message, delete the one with image)
	rewindParams := RewindSessionRequest{
		MessageID:    messages[0].ID,
		CleanupMedia: true,
	}

	paramsJSON, err := json.Marshal(rewindParams)
	if err != nil {
		t.Fatalf("Failed to marshal rewind params: %v", err)
	}

	// Make REST API call
	url := server.URL + "/api/sessions/" + sessionID + "/rewind"
	resp, err := http.Post(url, "application/json", bytes.NewBuffer(paramsJSON))
	if err != nil {
		t.Fatalf("Failed to make rewind request: %v", err)
	}
	defer func() { _ = resp.Body.Close() }()

	validateObjectResponse(t, resp, http.StatusOK)

	// Verify image was deleted
	if _, err := os.Stat(testImageFullPath); !os.IsNotExist(err) {
		t.Errorf("Test image should have been deleted but still exists")
	}

	// Verify only 1 message remains
	remainingMessages, err := app.Messages.List(ctx, sessionID)
	if err != nil {
		t.Fatalf("Failed to list messages after rewind: %v", err)
	}

	if len(remainingMessages) != 1 {
		t.Errorf("Expected 1 message after rewind, got %d", len(remainingMessages))
	}

	t.Logf("Rewind with media cleanup successful")
}

func TestSessionRewindToEmpty(t *testing.T) {
	app, sessionID := setupTestServerForRewind(t)

	// Create test messages
	createTestMessagesForRewind(t, app, sessionID, 2) // Creates 4 messages

	// Create REST handler and test server
	sessionHandler := NewSessionHandler(app)
	mux := http.NewServeMux()
	mux.HandleFunc("POST /api/sessions/{id}/rewind", sessionHandler.HandleRewindSession)
	server := httptest.NewServer(mux)
	defer server.Close()

	// Get messages for rewinding
	ctx := context.Background()
	messages, err := app.Messages.List(ctx, sessionID)
	if err != nil {
		t.Fatalf("Failed to list messages: %v", err)
	}

	// Rewind to first message (keep just 1 message)
	rewindParams := RewindSessionRequest{
		MessageID:    messages[0].ID,
		CleanupMedia: false,
	}

	paramsJSON, err := json.Marshal(rewindParams)
	if err != nil {
		t.Fatalf("Failed to marshal rewind params: %v", err)
	}

	url := server.URL + "/api/sessions/" + sessionID + "/rewind"
	resp, err := http.Post(url, "application/json", bytes.NewBuffer(paramsJSON))
	if err != nil {
		t.Fatalf("Failed to make rewind request: %v", err)
	}
	defer func() { _ = resp.Body.Close() }()

	validateObjectResponse(t, resp, http.StatusOK)

	// Verify only 1 message remains
	ctx = context.Background()
	remainingMessages, err := app.Messages.List(ctx, sessionID)
	if err != nil {
		t.Fatalf("Failed to list messages after rewind: %v", err)
	}

	if len(remainingMessages) != 1 {
		t.Errorf("Expected 1 message after rewind to index 0, got %d", len(remainingMessages))
	}

	t.Logf("Rewind to near-empty session successful")
}

func TestSessionRewindErrorHandling(t *testing.T) {
	app, sessionID := setupTestServerForRewind(t)

	// Create test messages
	messages := createTestMessagesForRewind(t, app, sessionID, 2)

	// Create REST handler and test server
	sessionHandler := NewSessionHandler(app)
	mux := http.NewServeMux()
	mux.HandleFunc("POST /api/sessions/{id}/rewind", sessionHandler.HandleRewindSession)
	server := httptest.NewServer(mux)
	defer server.Close()

	testCases := []struct {
		name        string
		sessionID   string
		request     RewindSessionRequest
		expectError bool
		statusCode  int
	}{
		{
			name:      "empty message ID",
			sessionID: sessionID,
			request: RewindSessionRequest{
				MessageID:    "",
				CleanupMedia: false,
			},
			expectError: true,
			statusCode:  400,
		},
		{
			name:      "non-existent session",
			sessionID: "non-existent-session-id",
			request: RewindSessionRequest{
				MessageID:    messages[0].ID,
				CleanupMedia: false,
			},
			expectError: true,
			statusCode:  404,
		},
		{
			name:      "non-existent message ID",
			sessionID: sessionID,
			request: RewindSessionRequest{
				MessageID:    "non-existent-message-id",
				CleanupMedia: false,
			},
			expectError: true,
			statusCode:  404,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			paramsJSON, err := json.Marshal(tc.request)
			if err != nil {
				t.Fatalf("Failed to marshal params: %v", err)
			}

			url := server.URL + "/api/sessions/" + tc.sessionID + "/rewind"
			resp, err := http.Post(url, "application/json", bytes.NewBuffer(paramsJSON))
			if err != nil {
				t.Fatalf("Failed to make rewind request: %v", err)
			}
			defer func() { _ = resp.Body.Close() }()

			if resp.StatusCode != tc.statusCode {
				t.Errorf("Expected status code %d, got %d", tc.statusCode, resp.StatusCode)
			}

			if tc.expectError && resp.StatusCode < 400 {
				t.Errorf("Expected error response but got success status %d", resp.StatusCode)
			}
		})
	}
}

func TestSessionRewindBoundary(t *testing.T) {
	app, sessionID := setupTestServerForRewind(t)

	// Create exactly 5 messages
	createTestMessagesForRewind(t, app, sessionID, 2) // Creates 4 messages
	ctx := context.Background()

	// Add one more user message
	finalMsg, err := app.Messages.Create(ctx, sessionID, message.CreateMessageParams{
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
	mux.HandleFunc("POST /api/sessions/{id}/rewind", sessionHandler.HandleRewindSession)
	server := httptest.NewServer(mux)
	defer server.Close()

	// Rewind to exact boundary (last message) - should keep all messages
	rewindParams := RewindSessionRequest{
		MessageID:    finalMsg.ID, // Keep all messages up to and including the last one
		CleanupMedia: false,
	}

	paramsJSON, err := json.Marshal(rewindParams)
	if err != nil {
		t.Fatalf("Failed to marshal rewind params: %v", err)
	}

	url := server.URL + "/api/sessions/" + sessionID + "/rewind"
	resp, err := http.Post(url, "application/json", bytes.NewBuffer(paramsJSON))
	if err != nil {
		t.Fatalf("Failed to make rewind request: %v", err)
	}
	defer func() { _ = resp.Body.Close() }()

	validateObjectResponse(t, resp, http.StatusOK)

	// Verify all 5 messages remain
	remainingMessages, err := app.Messages.List(ctx, sessionID)
	if err != nil {
		t.Fatalf("Failed to list messages after rewind: %v", err)
	}

	if len(remainingMessages) != 5 {
		t.Errorf("Expected 5 messages after rewind to boundary, got %d", len(remainingMessages))
	}

	t.Logf("Rewind at exact boundary successful")
}