package http

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"sync"
	"time"

	"mix/internal/app"
	"mix/internal/llm/agent"
	"mix/internal/logging"
	"mix/internal/pubsub"
	"mix/internal/session"
)

// Connection represents a single SSE connection
type Connection struct {
	SessionID       string
	BroadcastEvents chan BroadcastEvent
	Done            chan struct{}
	closeOnce       sync.Once
}

// BroadcastEvent represents an SSE event to be broadcast to all connections
type BroadcastEvent struct {
	EventType string
	Data      interface{}
}

// ConnectionRegistry manages active SSE connections
type ConnectionRegistry struct {
	mu          sync.RWMutex
	connections map[string]map[*Connection]struct{}
}

// Global connection registry
var registry = &ConnectionRegistry{
	connections: make(map[string]map[*Connection]struct{}),
}

// Global session event broadcaster - ensures single subscription
var sessionEventBroadcaster sync.Once

// Register adds a connection to the registry
func (r *ConnectionRegistry) Register(sessionID string, conn *Connection) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.connections[sessionID] == nil {
		r.connections[sessionID] = make(map[*Connection]struct{})
	}
	r.connections[sessionID][conn] = struct{}{}
}

// Unregister removes a connection from the registry
func (r *ConnectionRegistry) Unregister(sessionID string, conn *Connection) {
	r.mu.Lock()
	defer r.mu.Unlock()

	if connections, exists := r.connections[sessionID]; exists {
		delete(connections, conn)
		// Clean up empty session entries
		if len(connections) == 0 {
			delete(r.connections, sessionID)
		}
	}
}

// CountForSession returns the number of active connections for a specific session
func (r *ConnectionRegistry) CountForSession(sessionID string) int {
	r.mu.RLock()
	defer r.mu.RUnlock()
	if connections, exists := r.connections[sessionID]; exists {
		return len(connections)
	}
	return 0
}

// BroadcastToAll sends an SSE event to all active connections regardless of session
func (r *ConnectionRegistry) BroadcastToAll(eventType string, data interface{}) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	event := BroadcastEvent{
		EventType: eventType,
		Data:      data,
	}

	for _, connections := range r.connections {
		for conn := range connections {
			select {
			case conn.BroadcastEvents <- event:
			case <-conn.Done:
			default:
			}
		}
	}
}

// BroadcastEvent broadcasts a structured event to all connections for a specific session
func (r *ConnectionRegistry) BroadcastEvent(sessionID string, eventType string, data interface{}) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	connections := r.connections[sessionID]

	event := BroadcastEvent{
		EventType: eventType,
		Data:      data,
	}

	for conn := range connections {
		select {
		case conn.BroadcastEvents <- event:
		case <-conn.Done:
		default:
		}
	}
}

// HandleSSEStream handles persistent Server-Sent Events streaming for agent responses
func HandleSSEStream(ctx context.Context, app *app.App, w http.ResponseWriter, r *http.Request) {
	// Set SSE headers
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Headers", "Cache-Control")

	if r.Method == "OPTIONS" {
		w.WriteHeader(http.StatusOK)
		return
	}

	sessionID := r.URL.Query().Get("sessionId")

	if sessionID == "" {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	// Get flusher for SSE streaming
	flusher, ok := w.(http.Flusher)
	if !ok {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	// Initialize SSEWriter for this session
	sseWriter := NewSSEWriter(w, sessionID, flusher)

	// TODO: Last-Event-ID header support is documented in OpenAPI spec but not implemented.
	// This feature was intentionally omitted from the initial implementation as it adds
	// significant complexity for minimal benefit in this chat interface context.
	//
	// For chat interfaces with persistent connections, clients typically don't need to resume
	// mid-message after disconnection. The connection stays open across multiple messages,
	// and users can simply reconnect to continue the conversation from the current state.
	// This design choice prioritizes simplicity and maintainability over rarely-used
	// reconnection scenarios.
	//
	// Implementation would require:
	// - Event ID generation and tracking per connection
	// - Event replay buffer management with memory/storage considerations
	// - Client-side Last-Event-ID header parsing and validation
	// - Complex state synchronization between server and client
	//
	// Current workaround: Clients can reconnect and continue from current conversation state.

	// Validate session exists
	_, err := app.Sessions.Get(ctx, sessionID)
	if err != nil {
		// Error already handled by SSEWriter
		_ = sseWriter.WriteEvent("error", ErrorEvent{Error: fmt.Sprintf("Invalid session ID: %s", sessionID)})
		return
	}

	// Create connection
	conn := &Connection{
		SessionID:       sessionID,
		BroadcastEvents: make(chan BroadcastEvent, 50),
		Done:            make(chan struct{}),
	}

	// Register connection and ensure cleanup
	registry.Register(sessionID, conn)

	defer func() {
		registry.Unregister(sessionID, conn)
		conn.closeOnce.Do(func() {
			close(conn.Done)
			close(conn.BroadcastEvents)
		})
	}()

	// Send connection confirmation with SSE-compliant event
	if err := sseWriter.WriteEvent("connected", ConnectedEvent{SessionID: sessionID}); err != nil {
		// Connection error handled by SSEWriter
		return
	}

	// Subscribe to permission events for this session
	permissionEvents := app.Permissions.Subscribe(ctx)

	// Handle permission events in a separate goroutine with high priority
	go func() {
		for {
			select {
			case <-ctx.Done():
				return
			case <-r.Context().Done():
				return
			case permissionEvent, ok := <-permissionEvents:
				if !ok {
					return
				}

				// Process permission events for current session

				// Only send permission events for the current session
				if permissionEvent.Type == pubsub.CreatedEvent && permissionEvent.Payload.SessionID == sessionID {
					// Send permission event to frontend

					permEvent := PermissionEvent{
						Type:        "permission",
						ID:          permissionEvent.Payload.ID,
						SessionID:   permissionEvent.Payload.SessionID,
						ToolName:    permissionEvent.Payload.ToolName,
						Description: permissionEvent.Payload.Description,
						Action:      permissionEvent.Payload.Action,
						Path:        permissionEvent.Payload.Path,
						Params:      permissionEvent.Payload.Params,
					}

					if err := sseWriter.WriteEvent("permission", permEvent); err != nil {
						return
					}
				}
				// Permission events filtered by session
			}
		}
	}()

	// Initialize global session event broadcaster (runs exactly once)
	sessionEventBroadcaster.Do(func() {
		go func() {
			// Use background context - this subscription outlives individual connections
			sessionEvents := app.Sessions.Subscribe(context.Background())
			for sessionEvent := range sessionEvents {
				switch sessionEvent.Type {
				case pubsub.CreatedEvent:
					// Only broadcast session_created for main and forked sessions
					// Subagent sessions are internal implementation details
					if sessionEvent.Payload.SessionType == session.SessionTypeSubagent {
						continue
					}

					evt := SessionCreatedEvent{
						Type:      "session_created",
						SessionID: sessionEvent.Payload.ID,
						Title:     sessionEvent.Payload.Title,
						CreatedAt: sessionEvent.Payload.CreatedAt,
					}
					registry.BroadcastToAll("session_created", evt)
				case pubsub.DeletedEvent:
					evt := SessionDeletedEvent{
						Type:      "session_deleted",
						SessionID: sessionEvent.Payload.ID,
					}
					registry.BroadcastToAll("session_deleted", evt)
				}
			}
		}()
	})

	// Heartbeat to prevent browser timeout
	heartbeat := time.NewTicker(45 * time.Second)
	defer heartbeat.Stop()

	// Main event loop - simple and clean
	for {
		select {
		case <-r.Context().Done():
			// Client disconnected - SSE is just an event stream view, disconnecting doesn't stop agent processing
			logging.Debug("SSE client disconnected",
				"sessionID", sessionID,
				"remoteAddr", r.RemoteAddr,
				"activeConnections", registry.CountForSession(sessionID))
			return

		case <-ctx.Done():
			// Handler context cancelled (server shutdown, timeout, etc.)
			app.CoderAgent.CancelWithReason(sessionID, "server_shutdown")
			return

		case <-heartbeat.C:
			if err := sseWriter.WriteEvent("heartbeat", HeartbeatEvent{Type: "ping"}); err != nil {
				// Heartbeat error handled by SSEWriter
				return
			}

		case event, ok := <-conn.BroadcastEvents:
			if !ok {
				return
			}

			// Write broadcast event to SSE stream
			if err := sseWriter.WriteEvent(event.EventType, event.Data); err != nil {
				return
			}
		}
	}
}

// WriteAgentEventAsSSE converts an AgentEvent to SSE format using unified event types
func WriteAgentEventAsSSE(sseWriter *SSEWriter, event agent.AgentEvent) error {
	switch event.Type {
	case agent.AgentEventTypeThinking:
		// Send thinking delta event
		if err := sseWriter.WriteEvent("thinking", ThinkingEvent{Type: "thinking", Content: event.Thinking}); err != nil {
			return err
		}

	case agent.AgentEventTypeContentDelta:
		// Stream content deltas for text between tool calls
		if event.Content != "" {
			if err := sseWriter.WriteEvent("content", ContentEvent{Type: "content", Content: event.Content}); err != nil {
				return err
			}
		}

	case agent.AgentEventTypeToolParameterDelta:
		// Stream tool parameter deltas for real-time parameter visibility
		if err := sseWriter.WriteEvent("tool_parameter_delta", ToolParameterDeltaEvent{
			Type:       "tool_parameter_delta",
			ToolCallID: event.ToolCallID,
			Input:      event.Content, // Delta is stored in Content field
		}); err != nil {
			return err
		}

	case agent.AgentEventTypeResponse:

		// Stream tool calls - detect new tool calls by checking completion status
		toolCalls := event.Message.ToolCalls()
		for _, toolCall := range toolCalls {
			// Determine tool status - tools start with complete parameters
			status := "running"
			if toolCall.Finished {
				status = "completed"
			}

			if err := sseWriter.WriteEvent("tool", ToolEvent{Type: "tool", Name: toolCall.Name, Input: toolCall.Input, ID: toolCall.ID, Status: status}); err != nil {
				return err
			}
		}

		// Send completion event only for final events, include final content
		if event.Done {
			// Check if this is a permission denied error
			if event.Message.FinishReason() == "permission_denied" {
				if err := sseWriter.WriteEvent("error", ErrorEvent{Error: "Permission denied"}); err != nil {
					return err
				}
			} else {
				content := event.Message.Content().String()
				reasoningContent := event.Message.ReasoningContent()
				reasoning := reasoningContent.String()
				reasoningDuration := reasoningContent.Duration
				if err := sseWriter.WriteEvent("complete", CompleteEvent{Type: "complete", Content: content, MessageID: event.Message.ID, Done: true, Reasoning: reasoning, ReasoningDuration: reasoningDuration}); err != nil {
					return err
				}
			}
		}

	case agent.AgentEventTypeError:
		errMsg := event.Error.Error()

		// Special handling for rate limit errors
		if strings.Contains(errMsg, "rate_limit_error") {
			// Extract retry information if available
			retryAfter := 60 // Default retry after 60 seconds
			attempt := 1
			maxAttempts := 8

			// Try to extract retry info from error message
			// Check if this contains retry attempt information
			if strings.Contains(errMsg, "Retrying due to rate limit") {
				// Try to parse attempt numbers like "attempt 1 of 8"
				var currentAttempt, totalAttempts int
				_, err := fmt.Sscanf(errMsg, "Retrying due to rate limit... attempt %d of %d", &currentAttempt, &totalAttempts)
				if err == nil && currentAttempt > 0 && totalAttempts > 0 {
					attempt = currentAttempt
					maxAttempts = totalAttempts
				}
			}

			errorEvent := ErrorEvent{
				Error:       "This request would exceed your account's rate limit. The application will automatically retry.",
				Type:        "rate_limit_error",
				RetryAfter:  retryAfter,
				Attempt:     attempt,
				MaxAttempts: maxAttempts,
			}

			if err := sseWriter.WriteEvent("rate_limit_error", errorEvent); err != nil {
				return err
			}

			// Special handling for authentication errors
		} else if strings.Contains(errMsg, "authentication_error") ||
			strings.Contains(errMsg, "x-api-key header is required") ||
			strings.Contains(errMsg, "401 Unauthorized") {
			// Create a more helpful error message
			helpfulMsg := "Authentication failed: Not logged in or token expired. Please use /login to authenticate with Claude Code."
			if err := sseWriter.WriteEvent("error", ErrorEvent{Error: helpfulMsg}); err != nil {
				return err
			}
		} else {
			// Normal error handling
			if err := sseWriter.WriteEvent("error", ErrorEvent{Error: errMsg}); err != nil {
				return err
			}
		}

	case agent.AgentEventTypeToolExecutionStart:
		// Extract tool name from progress message
		toolName := "tool" // Default fallback
		if strings.Contains(event.Progress, "Executing ") && strings.Contains(event.Progress, " tool") {
			// Extract tool name from "Executing {toolName} tool"
			start := strings.Index(event.Progress, "Executing ") + len("Executing ")
			end := strings.Index(event.Progress[start:], " tool")
			if end > 0 {
				toolName = event.Progress[start : start+end]
			}
		}

		if err := sseWriter.WriteEvent("tool_execution_start", ToolExecutionStartEvent{
			Type:       "tool_execution_start",
			ToolName:   toolName,
			Progress:   event.Progress,
			ToolCallID: event.ToolCallID,
		}); err != nil {
			return err
		}

	case agent.AgentEventTypeToolExecutionComplete:
		// Extract tool name and success status from progress message
		toolName := "tool" // Default fallback
		success := true    // Default to success

		if strings.Contains(event.Progress, "Completed ") && strings.Contains(event.Progress, " tool") {
			// Extract tool name from "Completed {toolName} tool in {duration}"
			start := strings.Index(event.Progress, "Completed ") + len("Completed ")
			end := strings.Index(event.Progress[start:], " tool")
			if end > 0 {
				toolName = event.Progress[start : start+end]
			}
		} else if strings.Contains(event.Progress, "Failed ") && strings.Contains(event.Progress, " tool") {
			// Extract tool name from "Failed {toolName} tool after {duration}: {error}"
			start := strings.Index(event.Progress, "Failed ") + len("Failed ")
			end := strings.Index(event.Progress[start:], " tool")
			if end > 0 {
				toolName = event.Progress[start : start+end]
			}
			success = false
		}

		if err := sseWriter.WriteEvent("tool_execution_complete", ToolExecutionCompleteEvent{
			Type:       "tool_execution_complete",
			ToolName:   toolName,
			Progress:   event.Progress,
			Success:    success,
			ToolCallID: event.ToolCallID,
		}); err != nil {
			return err
		}
	}

	return nil
}
