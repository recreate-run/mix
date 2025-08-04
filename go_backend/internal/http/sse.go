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

	"mix/internal/api"
	"mix/internal/commands"
	"mix/internal/fileutil"
	"mix/internal/llm/agent"
)

// Connection represents a single SSE connection
type Connection struct {
	SessionID string
	Messages  chan string
	Done      chan struct{}
}

// ConnectionRegistry manages active SSE connections
type ConnectionRegistry struct {
	mu          sync.RWMutex
	connections map[string][]*Connection
}

// Global connection registry
var registry = &ConnectionRegistry{
	connections: make(map[string][]*Connection),
}

// Register adds a connection to the registry
func (r *ConnectionRegistry) Register(sessionID string, conn *Connection) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.connections[sessionID] = append(r.connections[sessionID], conn)
}

// Unregister removes a connection from the registry
func (r *ConnectionRegistry) Unregister(sessionID string, conn *Connection) {
	r.mu.Lock()
	defer r.mu.Unlock()

	connections := r.connections[sessionID]
	for i, c := range connections {
		if c == conn {
			// Remove connection from slice
			r.connections[sessionID] = append(connections[:i], connections[i+1:]...)
			break
		}
	}

	// Clean up empty session entries
	if len(r.connections[sessionID]) == 0 {
		delete(r.connections, sessionID)
	}
}

// Broadcast sends a message to all connections for a sessionID
func (r *ConnectionRegistry) Broadcast(sessionID, message string) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	connections := r.connections[sessionID]
	for _, conn := range connections {
		select {
		case conn.Messages <- message:
		case <-conn.Done:
			// Connection is closed, skip
		default:
			// Channel full, drop message to prevent blocking
		}
	}
}

// HandleSSEStream handles persistent Server-Sent Events streaming for agent responses
func HandleSSEStream(ctx context.Context, handler *api.QueryHandler, w http.ResponseWriter, r *http.Request) {
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
		WriteSSE(w, "error", ErrorEvent{Error: "Missing sessionId parameter"})
		return
	}

	if err := handler.GetApp().SetCurrentSession(sessionID); err != nil {
		WriteSSE(w, "error", ErrorEvent{Error: "Failed to set session: " + err.Error()})
		return
	}

	flusher, ok := w.(http.Flusher)
	if !ok {
		WriteSSE(w, "error", ErrorEvent{Error: "Streaming not supported"})
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
		close(conn.Done)
		close(conn.Messages)
		registry.Unregister(sessionID, conn)
	}()

	// Send connection confirmation
	WriteSSE(w, "connected", ConnectedEvent{SessionID: sessionID})
	flusher.Flush()

	// Heartbeat to prevent browser timeout
	heartbeat := time.NewTicker(45 * time.Second)
	defer heartbeat.Stop()

	// Main event loop - simple and clean
	for {
		select {
		case <-r.Context().Done():
			// Client disconnected
			handler.GetApp().CoderAgent.Cancel(sessionID)
			return

		case <-heartbeat.C:
			WriteSSE(w, "heartbeat", HeartbeatEvent{Type: "ping"})
			flusher.Flush()

		case message, ok := <-conn.Messages:
			if !ok {
				return
			}

			if err := processMessage(ctx, handler, w, flusher, sessionID, message); err != nil {
				return
			}
		}
	}
}

// MessageContent represents the JSON structure sent from frontend
type MessageContent struct {
	Text     string   `json:"text"`
	Media    []string `json:"media,omitempty"`
	Apps     []string `json:"apps,omitempty"`
	PlanMode bool     `json:"plan_mode,omitempty"`
}

// extractText parses JSON content to extract the actual text value
func extractText(content string) string {
	var msgContent MessageContent
	if err := json.Unmarshal([]byte(content), &msgContent); err == nil && msgContent.Text != "" {
		return msgContent.Text
	}
	return content
}

// parseMessageContent parses the complete JSON message structure
func parseMessageContent(content string) (MessageContent, error) {
	var msgContent MessageContent
	if err := json.Unmarshal([]byte(content), &msgContent); err != nil {
		return msgContent, fmt.Errorf("failed to parse message content as JSON: %w", err)
	}
	return msgContent, nil
}

// quotePaths ensures all file paths in the content are properly quoted for shell operations
func quotePaths(text string, mediaPaths []string) string {
	result := text

	// Quote media paths that might be referenced in the text
	for _, path := range mediaPaths {
		quotedPath := fileutil.QuotePath(path)
		// Replace unquoted paths with quoted versions
		result = strings.ReplaceAll(result, path, quotedPath)
	}

	return result
}

// handleShellCommand executes shell commands for ! prefixed messages
func handleShellCommand(ctx context.Context, w http.ResponseWriter, flusher http.Flusher, text string) error {
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

	WriteSSE(w, "complete", CompleteEvent{Type: "complete", Content: result, Done: true})
	flusher.Flush()
	return nil
}

// handleRegularMessage processes regular messages through the agent
func handleRegularMessage(ctx context.Context, handler *api.QueryHandler, w http.ResponseWriter, flusher http.Flusher, sessionID, content string) error {
	msgContent, err := parseMessageContent(content)
	if err != nil {
		WriteSSE(w, "error", ErrorEvent{Error: fmt.Sprintf("Failed to parse message: %s", err.Error())})
		flusher.Flush()
		return nil
	}
	
	events, err := handler.GetApp().CoderAgent.RunWithPlanMode(ctx, sessionID, content, msgContent.PlanMode)
	if err != nil {
		WriteSSE(w, "error", ErrorEvent{Error: fmt.Sprintf("Failed to start agent: %s", err.Error())})
		flusher.Flush()
		return nil
	}

	for {
		select {
		case <-ctx.Done():
			handler.GetApp().CoderAgent.Cancel(sessionID)
			return ctx.Err()

		case event, ok := <-events:
			if !ok {
				var content, messageID, reasoning string
				var reasoningDuration int64
				if messages, err := handler.GetApp().Messages.List(context.Background(), sessionID); err == nil && len(messages) > 0 {
					lastMessage := messages[len(messages)-1]
					if lastMessage.Role == "assistant" {
						content = lastMessage.Content().String()
						messageID = lastMessage.ID
						reasoningContent := lastMessage.ReasoningContent()
						reasoning = reasoningContent.String()
						reasoningDuration = reasoningContent.Duration
					}
				}
				WriteSSE(w, "complete", CompleteEvent{Type: "complete", Content: content, MessageID: messageID, Done: true, Reasoning: reasoning, ReasoningDuration: reasoningDuration})
				flusher.Flush()
				return nil
			}

			if err := WriteAgentEventAsSSE(w, event); err != nil {
				return err
			}
			flusher.Flush()

			if event.Error != nil || event.Done {
				return nil
			}
		}
	}
}

// processMessage processes a single message and streams the response
func processMessage(ctx context.Context, handler *api.QueryHandler, w http.ResponseWriter, flusher http.Flusher, sessionID, content string) error {
	msgContent, err := parseMessageContent(content)
	if err != nil {
		return err
	}

	text := msgContent.Text

	switch {
	case strings.HasPrefix(text, "/"):
		// Quote paths in slash commands if they contain file references
		quotedText := quotePaths(text, msgContent.Media)
		return handleSlashCommandStreaming(ctx, handler, w, flusher, sessionID, quotedText)
	case strings.HasPrefix(text, "!"):
		// Quote paths in shell commands
		quotedText := quotePaths(text, msgContent.Media)
		return handleShellCommand(ctx, w, flusher, quotedText)
	default:
		return handleRegularMessage(ctx, handler, w, flusher, sessionID, content)
	}
}

// handleSlashCommandStreaming processes slash commands for persistent connections
func handleSlashCommandStreaming(ctx context.Context, handler *api.QueryHandler, w http.ResponseWriter, flusher http.Flusher, sessionID, content string) error {
	parsedCmd, err := commands.ParseCommand(content)
	if err != nil {
		WriteSSE(w, "error", ErrorEvent{Error: fmt.Sprintf("Invalid slash command: %s", err.Error())})
		flusher.Flush()
		return nil
	}

	reg := commands.NewRegistry()
	if err := reg.LoadCommands(handler.GetApp()); err != nil {
		WriteSSE(w, "error", ErrorEvent{Error: fmt.Sprintf("Failed to load commands: %s", err.Error())})
		flusher.Flush()
		return nil
	}

	result, err := reg.ExecuteCommand(ctx, parsedCmd.Name, parsedCmd.Arguments)
	if err != nil {
		WriteSSE(w, "error", ErrorEvent{Error: fmt.Sprintf("Command execution failed: %s", err.Error())})
		flusher.Flush()
		return nil
	}

	WriteSSE(w, "complete", CompleteEvent{Type: "complete", Content: result, Done: true})
	flusher.Flush()
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
func WriteAgentEventAsSSE(w http.ResponseWriter, event agent.AgentEvent) error {
	switch event.Type {
	case agent.AgentEventTypeResponse:
		// Stream tool calls - detect new tool calls by checking completion status
		toolCalls := event.Message.ToolCalls()
		for _, toolCall := range toolCalls {
			// Determine tool status
			status := "pending"
			if toolCall.Input != "" {
				if len(toolCall.Input) > 0 {
					status = "running"
				}
				// Check if tool call is complete (has been finished)
				if event.Message.FinishReason() != "" && !event.Done {
					status = "completed"
				}
			}

			if err := WriteSSE(w, "tool", ToolEvent{Type: "tool", Name: toolCall.Name, Input: toolCall.Input, ID: toolCall.ID, Status: status}); err != nil {
				return err
			}
		}

		// Send completion event only for final events, include final content
		if event.Done {
			// Check if this is a permission denied error
			if event.Message.FinishReason() == "permission_denied" {
				if err := WriteSSE(w, "error", ErrorEvent{Error: "Permission denied"}); err != nil {
					return err
				}
			} else {
				content := event.Message.Content().String()
				reasoningContent := event.Message.ReasoningContent()
				reasoning := reasoningContent.String()
				reasoningDuration := reasoningContent.Duration
				if err := WriteSSE(w, "complete", CompleteEvent{Type: "complete", Content: content, MessageID: event.Message.ID, Done: true, Reasoning: reasoning, ReasoningDuration: reasoningDuration}); err != nil {
					return err
				}
			}
		}

	case agent.AgentEventTypeError:
		if err := WriteSSE(w, "error", ErrorEvent{Error: event.Error.Error()}); err != nil {
			return err
		}

	case agent.AgentEventTypeSummarize:
		if err := WriteSSE(w, "summarize", SummarizeEvent{Type: "summarize", Progress: event.Progress, Done: event.Done}); err != nil {
			return err
		}
	}

	return nil
}
