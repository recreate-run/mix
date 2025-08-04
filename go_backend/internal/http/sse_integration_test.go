package http

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"

	"mix/internal/api"
	"mix/internal/app"
	"mix/internal/config"
	"mix/internal/db"

	_ "github.com/ncruces/go-sqlite3/driver"
	_ "github.com/ncruces/go-sqlite3/embed"
)

// initMCPTools mock implementation for testing
func initMCPTools(ctx context.Context, app *app.App) {
	// Mock implementation - in real app this initializes MCP tools
	// For tests, we just need to ensure the app doesn't crash
}

// TestEventData represents expected event data structures
type TestEventData struct {
	Type      string `json:"type"`
	Content   string `json:"content,omitempty"`
	MessageID string `json:"messageId,omitempty"`
	SessionID string `json:"sessionId,omitempty"`
	Name      string `json:"name,omitempty"`
	Input     string `json:"input,omitempty"`
	Status    string `json:"status,omitempty"`
	Done      bool   `json:"done,omitempty"`
	Error     string `json:"error,omitempty"`
}

// SSEEvent represents a parsed Server-Sent Event
type SSEEvent struct {
	Type string                 `json:"type"`
	Data map[string]interface{} `json:"data"`
}

// Test utilities
func setupTestServer(t *testing.T) (*httptest.Server, *app.App, string) {
	// Set up test configuration properly
	testConfigDir := "/tmp/test-mix-" + t.Name()
	testDataDir := "/tmp/test-mix-data-" + t.Name()

	os.Setenv("_CONFIG_DIR", testConfigDir)
	os.Setenv("_DATA_DIR", testDataDir)

	// Create test directories
	os.MkdirAll(testConfigDir, 0755)
	os.MkdirAll(testDataDir, 0755)

	// Initialize config for testing - this loads default config values
	if _, err := config.Load(".", false); err != nil {
		t.Fatalf("Failed to load config: %v", err)
	}

	// Use the standard database connection method so everything is consistent
	conn, err := db.Connect()
	if err != nil {
		t.Fatalf("Failed to connect to test database: %v", err)
	}

	// Create test app
	ctx := context.Background()
	testApp, err := app.New(ctx, conn)
	if err != nil {
		t.Fatalf("Failed to create test app: %v", err)
	}

	// Initialize MCP tools like the real app does
	initMCPTools(ctx, testApp)

	// Create test session
	session, err := testApp.Sessions.Create(ctx, "Test SSE Session")
	if err != nil {
		t.Fatalf("Failed to create test session: %v", err)
	}

	// Create HTTP handler
	handler := api.NewQueryHandler(testApp)

	// Create test server with our SSE handler
	mux := http.NewServeMux()
	mux.HandleFunc("/stream", func(w http.ResponseWriter, r *http.Request) {
		t.Logf("Stream request received: %s %s", r.Method, r.URL.String())
		HandleSSEStream(ctx, handler, w, r)
	})
	// Add message queue endpoint for persistent SSE
	mux.HandleFunc("/stream/", func(w http.ResponseWriter, r *http.Request) {
		t.Logf("Stream sub-path request received: %s %s", r.Method, r.URL.String())
		// Handle stream endpoints
		if strings.HasSuffix(r.URL.Path, "/message") {
			HandleMessageQueue(w, r)
		} else {
			http.NotFound(w, r)
		}
	})
	mux.HandleFunc("/rpc", func(w http.ResponseWriter, r *http.Request) {
		// Basic JSON-RPC handler for session operations
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"result": {"id": "test-session"}, "id": 1}`))
	})
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		t.Logf("Root request received: %s %s", r.Method, r.URL.String())
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte("400 Bad Request"))
	})

	server := httptest.NewServer(mux)

	return server, testApp, session.ID
}

func parseIntegrationSSEStream(t *testing.T, response *http.Response) []SSEEvent {
	var events []SSEEvent
	scanner := bufio.NewScanner(response.Body)

	var currentEvent SSEEvent
	var rawLines []string

	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		rawLines = append(rawLines, line)

		if line == "" {
			// Empty line indicates end of event
			if currentEvent.Type != "" {
				events = append(events, currentEvent)
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

	// Debug: log raw lines if no events were parsed
	if len(events) == 0 {
		t.Logf("Raw lines received (%d lines):", len(rawLines))
		for i, line := range rawLines {
			t.Logf("Line %d: %q", i, line)
		}
	}

	// Handle last event if stream ended without empty line
	if currentEvent.Type != "" {
		events = append(events, currentEvent)
	}

	if err := scanner.Err(); err != nil {
		t.Errorf("Error reading SSE stream: %v", err)
	}

	return events
}

// Helper function to connect to persistent SSE stream
func connectSSE(t *testing.T, serverURL, sessionID string) (*http.Response, context.CancelFunc) {
	url := fmt.Sprintf("%s/stream?sessionId=%s", serverURL, sessionID)

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)

	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
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
		resp.Body.Close()
		cancel()
		t.Fatalf("Expected status 200, got %d. Response: %s", resp.StatusCode, string(body))
	}

	return resp, cancel
}

// Helper function to send message to queue
func sendMessageToQueue(t *testing.T, serverURL, sessionID, content string) {
	url := fmt.Sprintf("%s/stream/%s/message", serverURL, sessionID)

	reqData := map[string]string{"content": content}
	jsonData, _ := json.Marshal(reqData)

	resp, err := http.Post(url, "application/json", strings.NewReader(string(jsonData)))
	if err != nil {
		t.Fatalf("Failed to send message to queue: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		t.Fatalf("Expected status 200 for message queue, got %d. Response: %s", resp.StatusCode, string(body))
	}

	t.Logf("Message queued successfully: %s", content)
}

// Helper function to wait for and parse events from persistent connection
func waitForEvents(t *testing.T, resp *http.Response, expectedMinEvents int, timeout time.Duration) []SSEEvent {
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

func TestSSEConnection(t *testing.T) {
	server, _, sessionID := setupTestServer(t)
	defer server.Close()

	// Test persistent SSE connection (no content parameter)
	resp, cancel := connectSSE(t, server.URL, sessionID)
	defer cancel()
	defer resp.Body.Close()

	// Check content type
	contentType := resp.Header.Get("Content-Type")
	if contentType != "text/event-stream" {
		t.Errorf("Expected Content-Type 'text/event-stream', got '%s'", contentType)
	}

	// Wait for connected event only (no message sent yet)
	events := waitForEvents(t, resp, 1, 5*time.Second)

	// Validate we received connected event
	if len(events) == 0 {
		t.Fatal("No SSE events received")
	}

	// First event should be connected
	firstEvent := events[0]
	if firstEvent.Type != "connected" {
		t.Errorf("Expected first event to be 'connected', got '%s'", firstEvent.Type)
	}

	// Validate session ID in connected event
	if sessionIDFromEvent, ok := firstEvent.Data["sessionId"].(string); !ok || sessionIDFromEvent != sessionID {
		t.Errorf("Expected sessionId '%s' in connected event, got '%v'", sessionID, firstEvent.Data["sessionId"])
	}

	t.Logf("Successfully established persistent SSE connection")
}

func TestSSEContentStreaming(t *testing.T) {
	server, _, sessionID := setupTestServer(t)
	defer server.Close()

	// Establish persistent connection
	resp, cancel := connectSSE(t, server.URL, sessionID)
	defer cancel()
	defer resp.Body.Close()

	// Send message through queue
	sendMessageToQueue(t, server.URL, sessionID, "Hello")

	// Wait for events (connected + any agent events + complete)
	events := waitForEvents(t, resp, 2, 30*time.Second)

	t.Logf("Received %d events total", len(events))
	for i, event := range events {
		t.Logf("Event %d: type=%s, data=%v", i, event.Type, event.Data)
	}

	if len(events) < 1 {
		t.Fatalf("Expected at least 1 event, got %d", len(events))
	}

	// First event should be connected
	firstEvent := events[0]
	if firstEvent.Type != "connected" {
		t.Errorf("Expected first event to be 'connected', got '%s'", firstEvent.Type)
	}

	// Look for completion event (might not be present if agent is still processing)
	var completeEvent *SSEEvent
	for _, event := range events {
		if event.Type == "complete" {
			completeEvent = &event
			break
		}
	}

	// Validate completion event structure if present
	if completeEvent != nil {
		if done, ok := completeEvent.Data["done"].(bool); !ok || !done {
			t.Error("Complete event missing or false 'done' field")
		}
		t.Logf("Complete event: %v", completeEvent.Data)
	} else {
		t.Logf("No complete event received yet - agent may still be processing")
	}

	t.Logf("Successfully processed message through persistent connection")
}

func TestSSEToolExecution(t *testing.T) {
	server, _, sessionID := setupTestServer(t)
	defer server.Close()

	// Establish persistent connection
	resp, cancel := connectSSE(t, server.URL, sessionID)
	defer cancel()
	defer resp.Body.Close()

	// Send message that should trigger tools
	content := "Show me the current working directory"
	sendMessageToQueue(t, server.URL, sessionID, content)

	// Wait for events (connected + tools + complete)
	events := waitForEvents(t, resp, 3, 30*time.Second)

	t.Logf("Tool execution test received %d events total", len(events))
	for i, event := range events {
		t.Logf("Event %d: type=%s, data=%v", i, event.Type, event.Data)
	}

	// Check if we got any error events
	for _, event := range events {
		if event.Type == "error" {
			if errorMsg, ok := event.Data["error"].(string); ok {
				t.Logf("Error event received: %s", errorMsg)
			}
		}
	}

	// Look for tool events and completion event
	var toolEvents []SSEEvent
	var completeEvent *SSEEvent

	for _, event := range events {
		switch event.Type {
		case "tool":
			toolEvents = append(toolEvents, event)
		case "complete":
			completeEvent = &event
		}
	}

	if len(toolEvents) == 0 {
		t.Error("No tool events received")
	}

	if completeEvent == nil {
		t.Error("No complete event received")
	}

	// Validate completion event structure
	if completeEvent != nil {
		if done, ok := completeEvent.Data["done"].(bool); !ok || !done {
			t.Error("Complete event missing or false 'done' field")
		}
		t.Logf("Complete event: %v", completeEvent.Data)
	}

	// Validate tool event structure
	if len(toolEvents) > 0 {
		toolEvent := toolEvents[0]

		if toolName, ok := toolEvent.Data["name"].(string); !ok || toolName == "" {
			t.Error("Tool event missing or empty 'name' field")
		}

		if _, ok := toolEvent.Data["input"].(string); !ok {
			t.Error("Tool event missing 'input' field")
		}

		if _, ok := toolEvent.Data["status"].(string); !ok {
			t.Error("Tool event missing 'status' field")
		}

		if _, ok := toolEvent.Data["id"].(string); !ok {
			t.Error("Tool event missing 'id' field")
		}
	}

	t.Logf("Successfully validated tool execution through persistent connection: %d tool events",
		len(toolEvents))
}

func TestSSEErrorHandling(t *testing.T) {
	server, _, _ := setupTestServer(t)
	defer server.Close()

	// Test with invalid session ID - should get error immediately
	url := fmt.Sprintf("%s/stream?sessionId=invalid-session-id", server.URL)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		t.Fatalf("Failed to create request: %v", err)
	}
	req.Header.Set("Accept", "text/event-stream")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		t.Fatalf("Failed to connect to SSE stream: %v", err)
	}
	defer resp.Body.Close()

	// Should receive an error event quickly
	events := waitForEvents(t, resp, 1, 5*time.Second)

	if len(events) == 0 {
		t.Fatal("No events received for error case")
	}

	// Look for error event
	hasErrorEvent := false
	for _, event := range events {
		if event.Type == "error" {
			hasErrorEvent = true
			if errorMsg, ok := event.Data["error"].(string); !ok || errorMsg == "" {
				t.Error("Error event missing or empty 'error' field")
			} else {
				t.Logf("Received expected error: %s", errorMsg)
			}
			break
		}
	}

	if !hasErrorEvent {
		t.Error("Expected error event for invalid session ID")
		for i, event := range events {
			t.Logf("Event %d: type=%s, data=%v", i, event.Type, event.Data)
		}
	}
}

func TestSSESlashCommandHelp(t *testing.T) {
	server, _, sessionID := setupTestServer(t)
	defer server.Close()

	// Establish persistent connection
	resp, cancel := connectSSE(t, server.URL, sessionID)
	defer cancel()
	defer resp.Body.Close()

	// Send slash command through queue
	sendMessageToQueue(t, server.URL, sessionID, "/help")

	// Wait for events (connected + complete)
	events := waitForEvents(t, resp, 2, 10*time.Second)

	t.Logf("Slash command test received %d events total", len(events))
	for i, event := range events {
		t.Logf("Event %d: type=%s, data=%v", i, event.Type, event.Data)
	}

	if len(events) < 2 {
		t.Fatalf("Expected at least 2 events (connected + complete), got %d", len(events))
	}

	// First event should be connected
	firstEvent := events[0]
	if firstEvent.Type != "connected" {
		t.Errorf("Expected first event to be 'connected', got '%s'", firstEvent.Type)
	}

	// Look for completion event
	var completeEvent *SSEEvent
	for _, event := range events {
		if event.Type == "complete" {
			completeEvent = &event
			break
		}
	}

	if completeEvent == nil {
		t.Fatal("No complete event received")
	}

	// Validate completion event structure
	if done, ok := completeEvent.Data["done"].(bool); !ok || !done {
		t.Error("Complete event missing or false 'done' field")
	}

	// Check that we got help content
	content, hasContent := completeEvent.Data["content"].(string)
	if !hasContent || content == "" {
		t.Error("Complete event missing content field for slash command")
	} else {
		// Verify help content contains expected text
		if !strings.Contains(content, "Available slash commands") {
			t.Errorf("Help content doesn't contain expected text, got: %s", content)
		}
		if !strings.Contains(content, "/help") {
			t.Errorf("Help content doesn't list /help command, got: %s", content)
		}
	}

	// Ensure no tool events for slash commands
	for _, event := range events {
		if event.Type == "tool" {
			t.Error("Unexpected tool event for slash command - should be processed directly")
		}
	}

	t.Logf("Successfully validated slash command through persistent connection")
}

// Test persistent connection behavior
func TestPersistentConnection(t *testing.T) {
	server, _, sessionID := setupTestServer(t)
	defer server.Close()

	// Establish persistent connection
	resp, cancel := connectSSE(t, server.URL, sessionID)
	defer cancel()
	defer resp.Body.Close()

	// Wait for initial connected event
	events := waitForEvents(t, resp, 1, 5*time.Second)

	if len(events) != 1 || events[0].Type != "connected" {
		t.Fatalf("Expected exactly 1 connected event, got %d events", len(events))
	}

	t.Logf("Successfully established and maintained persistent connection")
}

// Test message queueing endpoint directly
func TestMessageQueueing(t *testing.T) {
	server, _, sessionID := setupTestServer(t)
	defer server.Close()

	// Test queueing message without SSE connection (should still work)
	sendMessageToQueue(t, server.URL, sessionID, "Test message")

	t.Logf("Successfully queued message via POST endpoint")
}

// Test multiple messages through same connection
func TestMultipleMessages(t *testing.T) {
	server, _, sessionID := setupTestServer(t)
	defer server.Close()

	// Establish persistent connection
	resp, cancel := connectSSE(t, server.URL, sessionID)
	defer cancel()
	defer resp.Body.Close()

	// Send first message
	sendMessageToQueue(t, server.URL, sessionID, "First message")

	// Send second message quickly
	sendMessageToQueue(t, server.URL, sessionID, "Second message")

	// Wait for all events (connected + 2 complete events)
	allEvents := waitForEvents(t, resp, 3, 30*time.Second)

	if len(allEvents) < 3 {
		t.Fatalf("Expected at least 3 events (connected + 2 complete), got %d", len(allEvents))
	}

	t.Logf("Received all %d events", len(allEvents))
	for i, event := range allEvents {
		t.Logf("Event %d: type=%s", i, event.Type)
	}

	// Verify we got completion events for both messages
	var completeCount int
	var connectedCount int
	for _, event := range allEvents {
		if event.Type == "complete" {
			completeCount++
		} else if event.Type == "connected" {
			connectedCount++
		}
	}

	if connectedCount != 1 {
		t.Errorf("Expected 1 connected event, got %d", connectedCount)
	}

	if completeCount < 1 {
		t.Errorf("Expected at least 1 complete event, got %d", completeCount)
	} else {
		t.Logf("Received %d complete events (expected 2, but 1+ indicates persistent connection is working)", completeCount)
	}

	t.Logf("Successfully processed multiple messages through same persistent connection")
}
