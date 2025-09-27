package http

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os/exec"
	"strings"
	"sync"
	"time"

	"mix/internal/app"
	"mix/internal/commands"
	"mix/internal/llm/agent"
	"mix/internal/llm/provider"
	"mix/internal/llm/tools"
	"mix/internal/pubsub"
)

// Connection represents a single SSE connection
type Connection struct {
	SessionID string
	Messages  chan string
	Done      chan struct{}
	closeOnce sync.Once
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

// Broadcast sends a message to all connections for a sessionID
func (r *ConnectionRegistry) Broadcast(sessionID, message string) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	connections := r.connections[sessionID]
	
	for conn := range connections {
		select {
		case conn.Messages <- message:
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
		if err := sseWriter.WriteEvent("error", ErrorEvent{Error: fmt.Sprintf("Invalid session ID: %s", sessionID)}); err != nil {
			// Error already handled by SSEWriter
		}
		return
	}


	// Create connection
	conn := &Connection{
		SessionID: sessionID,
		Messages:  make(chan string, 100),
		Done:      make(chan struct{}),
	}

	// Register connection and ensure cleanup
	registry.Register(sessionID, conn)
	defer func() {
		registry.Unregister(sessionID, conn)
		conn.closeOnce.Do(func() {
			close(conn.Done)
			close(conn.Messages)
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

	// Heartbeat to prevent browser timeout
	heartbeat := time.NewTicker(45 * time.Second)
	defer heartbeat.Stop()

	// Main event loop - simple and clean
	for {
		select {
		case <-r.Context().Done():
			// Client disconnected
			app.CoderAgent.Cancel(sessionID)
			return

		case <-ctx.Done():
			// Handler context cancelled (server shutdown, timeout, etc.)
			app.CoderAgent.Cancel(sessionID)
			return

		case <-heartbeat.C:
			if err := sseWriter.WriteEvent("heartbeat", HeartbeatEvent{Type: "ping"}); err != nil {
				// Heartbeat error handled by SSEWriter
				return
			}

		case message, ok := <-conn.Messages:
			if !ok {
				return
			}

			if err := processMessage(ctx, app, sseWriter, sessionID, message); err != nil {
				return
			}
		}
	}
}

// MessageContent represents the JSON structure sent from frontend
type MessageContent struct {
	Text     string `json:"text"`
	PlanMode bool   `json:"plan_mode,omitempty"`
}


// parseMessageContent parses the complete JSON message structure
func parseMessageContent(content string) (MessageContent, error) {
	var msgContent MessageContent
	if err := json.Unmarshal([]byte(content), &msgContent); err != nil {
		return msgContent, fmt.Errorf("failed to parse message content as JSON: %w", err)
	}
	return msgContent, nil
}


// handleShellCommand executes shell commands for ! prefixed messages
func handleShellCommand(ctx context.Context, sseWriter *SSEWriter, text string) error {
	command := strings.TrimSpace(strings.TrimPrefix(text, "!"))
	if command == "" {
		command = "echo 'No command specified'"
	}

	cmd := exec.CommandContext(ctx, "sh", "-c", command)
	output, err := cmd.CombinedOutput()

	result := string(output)
	if err != nil {
		result = fmt.Sprintf("Error: %v\n%s", err, result)
	}

	if err := sseWriter.WriteEvent("complete", CompleteEvent{Type: "complete", Content: result, Done: true}); err != nil {
		return fmt.Errorf("failed to write complete event: %w", err)
	}
	return nil
}

// handleRegularMessage processes regular messages through the agent
func handleRegularMessage(ctx context.Context, app *app.App, sseWriter *SSEWriter, sessionID, text string, planMode bool) error {
	
	// Check authentication status before processing the message using the centralized function
	authenticated, _, authErr := provider.IsAuthenticated(ctx, "")
	if authErr != nil {
		if err := sseWriter.WriteEvent("error", ErrorEvent{Error: fmt.Sprintf("Error checking authentication: %s", authErr.Error())}); err != nil {
			// Auth error handled by SSEWriter
		}
		return nil
	}
	
	
	// If not authenticated, show a provider-specific error message
	if !authenticated {
		helpfulMsg := getAuthenticationErrorMessage(ctx)
		
		if err := sseWriter.WriteEvent("error", ErrorEvent{
			Error: helpfulMsg,
			Type: "authentication_error",
		}); err != nil {
			// Auth error handled by SSEWriter
		}
		return nil
	}
	
	// If authenticated, proceed with normal message processing
	events, err := app.CoderAgent.RunWithPlanMode(ctx, sessionID, text, planMode)
	if err != nil {
		if err := sseWriter.WriteEvent("error", ErrorEvent{Error: fmt.Sprintf("Failed to start agent: %s", err.Error())}); err != nil {
			// Agent error handled by SSEWriter
		}
		return nil
	}
	

	for {
		select {
		case <-ctx.Done():
			app.CoderAgent.Cancel(sessionID)
			return ctx.Err()

		case event, ok := <-events:
			if !ok {
				var content, messageID, reasoning string
				var reasoningDuration int64
				if messages, err := app.Messages.List(context.Background(), sessionID); err == nil && len(messages) > 0 {
					lastMessage := messages[len(messages)-1]
					if lastMessage.Role == "assistant" {
						content = lastMessage.Content().String()
						messageID = lastMessage.ID
						reasoningContent := lastMessage.ReasoningContent()
						reasoning = reasoningContent.String()
						reasoningDuration = reasoningContent.Duration
					}
				}
				if err := sseWriter.WriteEvent("complete", CompleteEvent{Type: "complete", Content: content, MessageID: messageID, Done: true, Reasoning: reasoning, ReasoningDuration: reasoningDuration}); err != nil {
					// Complete event error handled by SSEWriter
				}
				return nil
			}

			if err := WriteAgentEventAsSSE(sseWriter, event); err != nil {
				return err
			}

			if event.Error != nil || event.Done {
				return nil
			}
		}
	}
}

// processMessage processes a single message and streams the response
func processMessage(ctx context.Context, app *app.App, sseWriter *SSEWriter, sessionID, content string) error {
	
	msgContent, err := parseMessageContent(content)
	if err != nil {
		return err
	}

	text := msgContent.Text

	switch {
	case strings.HasPrefix(text, "/"):
		return handleSlashCommandStreaming(ctx, app, sseWriter, sessionID, text)
	case strings.HasPrefix(text, "!"):
		return handleShellCommand(ctx, sseWriter, text)
	default:
		return handleRegularMessage(ctx, app, sseWriter, sessionID, text, msgContent.PlanMode)
	}
}

// handleSlashCommandStreaming processes slash commands for persistent connections
func handleSlashCommandStreaming(ctx context.Context, app *app.App, sseWriter *SSEWriter, sessionID, content string) error {

	parsedCmd, err := commands.ParseCommand(content)
	if err != nil {
		if err := sseWriter.WriteEvent("error", ErrorEvent{Error: fmt.Sprintf("Invalid slash command: %s", err.Error())}); err != nil {
			// Command error handled by SSEWriter
		}
		return nil
	}


	reg := commands.NewRegistry()
	if err := reg.LoadCommands(app); err != nil {
		if err := sseWriter.WriteEvent("error", ErrorEvent{Error: fmt.Sprintf("Failed to load commands: %s", err.Error())}); err != nil {
			// Load commands error handled by SSEWriter
		}
		return nil
	}

	// Add session context for commands that need session information
	cmdCtx := context.WithValue(ctx, tools.SessionIDContextKey, sessionID)
	
	result, err := reg.ExecuteCommand(cmdCtx, parsedCmd.Name, parsedCmd.Arguments)
	if err != nil {
		if err := sseWriter.WriteEvent("error", ErrorEvent{Error: fmt.Sprintf("Command execution failed: %s", err.Error())}); err != nil {
			// Command execution error handled by SSEWriter
		}
		return nil
	}

	if err := sseWriter.WriteEvent("complete", CompleteEvent{Type: "complete", Content: result, Done: true}); err != nil {
		// Command complete error handled by SSEWriter
	}
	return nil
}

// HandleMessageQueue handles POST requests to add messages to session queues
func HandleMessageQueue(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Methods", "POST, OPTIONS")
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type")

	if r.Method == "OPTIONS" {
		w.WriteHeader(http.StatusOK)
		return
	}

	if r.Method != "POST" {
		http.Error(w, "Only POST method allowed", http.StatusMethodNotAllowed)
		return
	}

	pathParts := strings.Split(strings.Trim(r.URL.Path, "/"), "/")
	if len(pathParts) < 2 || pathParts[0] != "stream" {
		http.Error(w, "Invalid URL path", http.StatusBadRequest)
		return
	}
	sessionID := pathParts[1]

	body, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, "Failed to read request body", http.StatusBadRequest)
		return
	}

	var reqData struct {
		Content string `json:"content"`
	}
	if err := json.Unmarshal(body, &reqData); err != nil {
		http.Error(w, "Invalid JSON in request body", http.StatusBadRequest)
		return
	}

	if reqData.Content == "" {
		http.Error(w, "Missing content parameter", http.StatusBadRequest)
		return
	}

	// Broadcast message to all active connections for this session
	registry.Broadcast(sessionID, reqData.Content)

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	response := map[string]interface{}{
		"status":    "broadcasted",
		"sessionId": sessionID,
	}
	json.NewEncoder(w).Encode(response)
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
				Error: "This request would exceed your account's rate limit. The application will automatically retry.",
				Type: "rate_limit_error",
				RetryAfter: retryAfter,
				Attempt: attempt,
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

	case agent.AgentEventTypeSummarize:
		if err := sseWriter.WriteEvent("summarize", SummarizeEvent{Type: "summarize", Progress: event.Progress, Done: event.Done}); err != nil {
			return err
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
