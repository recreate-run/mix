package http

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"testing"
	"time"
)

const (
	sseEventPrefix     = "event: "
	sseDataPrefix      = "data: "
	eventTypeConnected = "connected"
)

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

// Helper function to connect to persistent SSE stream
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


// Helper function to wait for and parse events from persistent connection
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

			if strings.HasPrefix(line, sseEventPrefix) {
				currentEvent.Type = strings.TrimPrefix(line, sseEventPrefix)
			} else if strings.HasPrefix(line, sseDataPrefix) {
				dataStr := strings.TrimPrefix(line, sseDataPrefix)
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
	result := setupIntegrationTestServer(t)
	defer result.Server.Close()

	// Test persistent SSE connection (no content parameter)
	resp, cancel := connectSSE(t, result.Server.URL, result.SessionID)
	defer cancel()
	defer func() { _ = resp.Body.Close() }()

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
	if firstEvent.Type != eventTypeConnected {
		t.Errorf("Expected first event to be 'connected', got '%s'", firstEvent.Type)
	}

	// Validate session ID in connected event
	if sessionIDFromEvent, ok := firstEvent.Data["sessionId"].(string); !ok || sessionIDFromEvent != result.SessionID {
		t.Errorf("Expected sessionId '%s' in connected event, got '%v'", result.SessionID, firstEvent.Data["sessionId"])
	}

	t.Logf("Successfully established persistent SSE connection")
}



func TestSSEErrorHandling(t *testing.T) {
	result := setupIntegrationTestServer(t)
	defer result.Server.Close()

	// Test with invalid session ID - should get error immediately
	url := fmt.Sprintf("%s/stream?sessionId=invalid-session-id", result.Server.URL)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, http.NoBody)
	if err != nil {
		t.Fatalf("Failed to create request: %v", err)
	}
	req.Header.Set("Accept", "text/event-stream")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		t.Fatalf("Failed to connect to SSE stream: %v", err)
	}
	defer func() { _ = resp.Body.Close() }()

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


// Test persistent connection behavior
func TestPersistentConnection(t *testing.T) {
	result := setupIntegrationTestServer(t)
	defer result.Server.Close()

	// Establish persistent connection
	resp, cancel := connectSSE(t, result.Server.URL, result.SessionID)
	defer cancel()
	defer func() { _ = resp.Body.Close() }()

	// Wait for initial connected event
	events := waitForEvents(t, resp, 1, 5*time.Second)

	if len(events) != 1 || events[0].Type != eventTypeConnected {
		t.Fatalf("Expected exactly 1 connected event, got %d events", len(events))
	}

	t.Logf("Successfully established and maintained persistent connection")
}


