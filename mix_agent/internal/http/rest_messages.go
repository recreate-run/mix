package http

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"mix/internal/app"
	"mix/internal/commands"
	"mix/internal/llm/agent"
	"mix/internal/llm/provider"
	"mix/internal/llm/tools"
	"mix/internal/logging"
	"mix/internal/message"
	"mix/internal/session"
)

// ToolCallData represents tool call information for REST API
type ToolCallData struct {
	ID       string `json:"id"`
	Name     string `json:"name"`
	Input    string `json:"input"`
	Type     string `json:"type"`
	Finished bool   `json:"finished"`
	Result   string `json:"result,omitempty"`
	IsError  bool   `json:"isError,omitempty"`
}

// MessageData represents message information for REST API
type MessageData struct {
	ID                string         `json:"id"`
	SessionID         string         `json:"sessionId"`
	Role              string         `json:"role"`
	UserInput         string         `json:"userInput"`
	AssistantResponse string         `json:"assistantResponse,omitempty"`
	ToolCalls         []ToolCallData `json:"toolCalls,omitempty"`
	Reasoning         string         `json:"reasoning,omitempty"`
	ReasoningDuration int64          `json:"reasoningDuration,omitempty"`
}

// ExportToolCall represents comprehensive tool call information for transcript export
type ExportToolCall struct {
	ID         string      `json:"id"`
	Name       string      `json:"name"`
	Input      string      `json:"input"`
	InputJSON  interface{} `json:"inputJson,omitempty"` // Parsed JSON for structured tools
	Type       string      `json:"type"`
	Finished   bool        `json:"finished"`
	Result     string      `json:"result,omitempty"`
	Metadata   string      `json:"metadata,omitempty"`
	IsError    bool        `json:"isError,omitempty"`
}

// ExportMessage represents comprehensive message information for transcript export
type ExportMessage struct {
	ID                    string              `json:"id"`
	Role                  string              `json:"role"`
	Content               string              `json:"content"`
	ToolCalls             []ExportToolCall    `json:"toolCalls,omitempty"`
	Reasoning             string              `json:"reasoning,omitempty"`
	ReasoningDuration     int64               `json:"reasoningDuration,omitempty"`
	ThinkingBlocks        []string            `json:"thinkingBlocks,omitempty"`
	RedactedThinkingBlocks []string           `json:"redactedThinkingBlocks,omitempty"`
	Model                 string              `json:"model,omitempty"`
	FinishReason          string              `json:"finishReason,omitempty"`
	CreatedAt             time.Time           `json:"createdAt"`
	UpdatedAt             time.Time           `json:"updatedAt"`
}

// ExportSession represents comprehensive session information for transcript export
type ExportSession struct {
	ID                    string          `json:"id"`
	Title                 string          `json:"title"`
	UserMessageCount      int64           `json:"userMessageCount"`
	AssistantMessageCount int64           `json:"assistantMessageCount"`
	ToolCallCount         int64           `json:"toolCallCount"`
	PromptTokens          int64           `json:"promptTokens"`
	CompletionTokens      int64           `json:"completionTokens"`
	Cost                  float64         `json:"cost"`
	CreatedAt             time.Time       `json:"createdAt"`
	UpdatedAt             time.Time       `json:"updatedAt"`
	Messages              []ExportMessage `json:"messages"`
}

// MessageHandler handles REST endpoints for message operations
type MessageHandler struct {
	app             *app.App
	commandRegistry *commands.Registry
}

// NewMessageHandler creates a new message handler
func NewMessageHandler(app *app.App) *MessageHandler {
	// Create command registry
	registry := commands.NewRegistry()
	if err := registry.LoadCommands(app); err != nil {
		logging.Error("Failed to load commands", "error", err)
		// Continue with empty registry - API will return proper errors
	}
	
	return &MessageHandler{
		app:             app,
		commandRegistry: registry,
	}
}

// SendMessageRequest represents the request body for sending a message
type SendMessageRequest struct {
	Text     string `json:"text"`
	PlanMode bool   `json:"plan_mode,omitempty"`
}

// Helper function to get command names for logging
func getCommandNames(commands map[string]commands.Command) []string {
	names := make([]string, 0, len(commands))
	for name := range commands {
		names = append(names, name)
	}
	return names
}

// HandleSendMessage handles POST /api/sessions/{id}/messages
func (h *MessageHandler) HandleSendMessage(w http.ResponseWriter, r *http.Request) {
	setCORSHeaders(w)
	if handleCORSPreflight(w, r) {
		return
	}

	if r.Method != "POST" {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	sessionID := r.PathValue("id")
	if sessionID == "" {
		sendValidationError(w, "id", "session ID is required")
		return
	}

	var req SendMessageRequest
	if err := parseJSONBody(r, &req); err != nil {
		sendValidationError(w, "body", err.Error())
		return
	}

	if req.Text == "" {
		sendValidationError(w, "content", "message content is required")
		return
	}

	ctx := r.Context()

	// Check authentication status before processing the message
	authenticated, _, authErr := provider.IsAuthenticated(ctx, "")
	if authErr != nil {
		sendInternalError(w, "checking authentication", authErr)
		return
	}

	// If not authenticated, show a provider-specific error message
	if !authenticated {
		helpfulMsg := getAuthenticationErrorMessage(ctx)

		// Broadcast error event to SSE so frontend stops processing
		registry.BroadcastEvent(sessionID, "error", ErrorEvent{Error: helpfulMsg})

		result := MessageData{
			ID:                "system-auth-prompt",
			Role:              "assistant",
			UserInput:         req.Text,
			AssistantResponse: helpfulMsg,
		}

		sendJSONResponse(w, http.StatusOK, result)
		return
	}

	// Check if this is a slash command and handle it immediately
	if commands.IsSlashCommand(req.Text) {
		parsed, parseErr := commands.ParseCommand(req.Text)
		if parseErr != nil {
			sendValidationError(w, "content", "Invalid slash command: "+parseErr.Error())
			return
		}


		// Add session context for commands that need session information
		cmdCtx := context.WithValue(ctx, tools.SessionIDContextKey, sessionID)
		
		commandResult, execErr := h.commandRegistry.ExecuteCommand(cmdCtx, parsed.Name, parsed.Arguments)
		if execErr != nil {
			logging.Error("Command execution failed", "name", parsed.Name, "error", execErr)

			// Check if it's a "command not found" error
			if strings.Contains(execErr.Error(), "command not found") {
				// List available commands for debugging
				allCommands := h.commandRegistry.GetAllCommands()
				commandNames := getCommandNames(allCommands)

				sendErrorResponse(w, ErrorTypeNotFound, fmt.Sprintf("Command '%s' not found. Available commands: %v", parsed.Name, commandNames))
				return
			}

			sendInternalError(w, "executing command", execErr)
			return
		}


		// Return the command result immediately as a message
		result := MessageData{
			ID:                "cmd-" + parsed.Name,
			Role:              "assistant",
			UserInput:         req.Text,
			AssistantResponse: commandResult,
		}

		sendJSONResponse(w, http.StatusOK, result)
		return
	}

	// Send message to agent
	// Use context.Background() instead of r.Context() to prevent HTTP request timeouts/cancellations
	// from killing long-running agent tasks. The agent can still be cancelled via the SSE disconnect
	// or the explicit cancel endpoint.
	agentCtx := context.Background()

	events, err := h.app.CoderAgent.RunWithPlanMode(agentCtx, sessionID, req.Text, req.PlanMode)
	if err != nil {
		sendInternalError(w, "sending message to agent", err)
		return
	}

	// Forward all events to active SSE connections while processing
	var lastEvent agent.AgentEvent
	for event := range events {
		// Broadcast event to all SSE connections for this session
		BroadcastAgentEventToSSE(sessionID, event)

		// Store last event for REST response
		lastEvent = event
	}

	// Check for processing errors
	if lastEvent.Error != nil {
		// Convert error to user-friendly message
		errorMessage := lastEvent.Error.Error()

		// Special handling for auth errors
		if strings.Contains(errorMessage, "401") || strings.Contains(errorMessage, "authentication") {
			authResult := MessageData{
				ID:                "system-auth-prompt",
				Role:              "assistant",
				UserInput:         req.Text,
				AssistantResponse: "⚠️ Authentication required. Please use the /login command to authenticate with Claude API key.",
			}

			sendJSONResponse(w, http.StatusOK, authResult)
			return
		}

		sendInternalError(w, "agent processing", lastEvent.Error)
		return
	}

	// Extract text content from the response message
	response := ""
	if lastEvent.Message.Content().String() != "" {
		response = lastEvent.Message.Content().String()
	}

	messageData := MessageData{
		ID:                lastEvent.Message.ID,
		Role:              "user",
		UserInput:         req.Text,
		AssistantResponse: response,
	}

	sendJSONResponse(w, http.StatusOK, messageData)
}

// HandleListSessionMessages handles GET /api/sessions/{id}/messages
func (h *MessageHandler) HandleListSessionMessages(w http.ResponseWriter, r *http.Request) {
	setCORSHeaders(w)
	if handleCORSPreflight(w, r) {
		return
	}

	if r.Method != "GET" {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	sessionID := r.PathValue("id")
	if sessionID == "" {
		sendValidationError(w, "id", "session ID is required")
		return
	}


	ctx := r.Context()
	messages, err := h.app.Messages.List(ctx, sessionID)
	if err != nil {
		logging.Error("Failed to list session messages", "sessionID", sessionID, "error", err)
		sendInternalError(w, "listing session messages", err)
		return
	}


	result := h.convertMessagesToData(messages)
	
	

	sendJSONResponse(w, http.StatusOK, result)
}

// HandleMessageHistory handles GET /api/messages/history
func (h *MessageHandler) HandleMessageHistory(w http.ResponseWriter, r *http.Request) {
	setCORSHeaders(w)
	if handleCORSPreflight(w, r) {
		return
	}

	if r.Method != "GET" {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Parse query parameters for pagination
	limitStr := r.URL.Query().Get("limit")
	offsetStr := r.URL.Query().Get("offset")

	var limit, offset int64 = 50, 0 // defaults

	if limitStr != "" {
		if parsedLimit, err := strconv.ParseInt(limitStr, 10, 64); err == nil && parsedLimit > 0 {
			limit = parsedLimit
		}
	}

	if offsetStr != "" {
		if parsedOffset, err := strconv.ParseInt(offsetStr, 10, 64); err == nil && parsedOffset >= 0 {
			offset = parsedOffset
		}
	}

	ctx := r.Context()
	messages, err := h.app.Messages.ListUserMessageHistory(ctx, limit, offset)
	if err != nil {
		sendInternalError(w, "getting message history", err)
		return
	}

	result := h.convertMessagesToData(messages)
	sendJSONResponse(w, http.StatusOK, result)
}

// HandleCancelAgent handles POST /api/sessions/{id}/cancel
func (h *MessageHandler) HandleCancelAgent(w http.ResponseWriter, r *http.Request) {
	setCORSHeaders(w)
	if handleCORSPreflight(w, r) {
		return
	}

	if r.Method != "POST" {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	sessionID := r.PathValue("id")
	if sessionID == "" {
		sendValidationError(w, "id", "session ID is required")
		return
	}

	// Cancel the agent processing for this session
	logging.Info("User cancelled session via REST API", "sessionID", sessionID)
	h.app.CoderAgent.CancelWithReason(sessionID, "user_api_cancellation")

	result := map[string]string{
		"status":    "cancelled",
		"sessionId": sessionID,
	}

	sendJSONResponse(w, http.StatusOK, result)
}

// HandleExportSession handles GET /api/sessions/{id}/export
func (h *MessageHandler) HandleExportSession(w http.ResponseWriter, r *http.Request) {
	setCORSHeaders(w)
	if handleCORSPreflight(w, r) {
		return
	}

	if r.Method != "GET" {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	sessionID := r.PathValue("id")
	if sessionID == "" {
		sendValidationError(w, "id", "session ID is required")
		return
	}

	ctx := r.Context()

	// Get session metadata
	session, err := h.app.Sessions.Get(ctx, sessionID)
	if err != nil {
		sendNotFoundError(w, "Session", sessionID)
		return
	}

	// Get all messages from the session
	messages, err := h.app.Messages.List(ctx, sessionID)
	if err != nil {
		logging.Error("Failed to list session messages for export", "sessionID", sessionID, "error", err)
		sendInternalError(w, "listing session messages", err)
		return
	}

	// Convert to comprehensive export format
	exportData := h.convertToExportSession(session, messages)

	// Track session export
	if h.app.Analytics != nil {
		totalTokens := session.PromptTokens + session.CompletionTokens
		_ = h.app.Analytics.TrackSessionExported(ctx, sessionID, len(messages), session.Cost, totalTokens)
	}

	// Set content disposition header for file download
	filename := fmt.Sprintf("session_%s_transcript.json", sessionID)
	w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=%s", filename))

	sendJSONResponse(w, http.StatusOK, exportData)
}

// convertMessagesToData converts message objects to MessageData for REST response
func (h *MessageHandler) convertMessagesToData(messages []message.Message) []MessageData {
	result := []MessageData{}
	for _, msg := range messages {
		// Extract tool calls and match with tool results
		toolCalls := msg.ToolCalls()
		toolResults := msg.ToolResults()
		
		// Extract reasoning content from both ReasoningContent and ThinkingBlock parts
		var reasoning string
		var reasoningDuration int64
		for _, part := range msg.Parts {
			if reasoningContent, ok := part.(message.ReasoningContent); ok {
				if reasoning == "" { // Use first reasoning content found
					reasoning = reasoningContent.Thinking
					reasoningDuration = reasoningContent.Duration
				}
			} else if thinkingBlock, ok := part.(message.ThinkingBlock); ok {
				if reasoning == "" { // Use thinking block if no reasoning content found
					reasoning = thinkingBlock.Thinking
					reasoningDuration = 0 // ThinkingBlock doesn't have duration
				}
			}
		}
		
		// Create a map of tool results by tool call ID for quick lookup
		resultsByID := make(map[string]message.ToolResult)
		for _, tr := range toolResults {
			resultsByID[tr.ToolCallID] = tr
		}
		
		toolCallsData := make([]ToolCallData, len(toolCalls))
		for i, tc := range toolCalls {
			toolCallData := ToolCallData{
				ID:       tc.ID,
				Name:     tc.Name,
				Input:    tc.Input,
				Type:     tc.Type,
				Finished: tc.Finished,
			}
			
			// Add result if available
			if toolResult, exists := resultsByID[tc.ID]; exists {
				toolCallData.Result = toolResult.Content
				toolCallData.IsError = toolResult.IsError
			}
			
			toolCallsData[i] = toolCallData
		}

		// Get message content
		content := msg.Content().String()
		
		messageData := MessageData{
			ID:        msg.ID,
			SessionID: msg.SessionID,
			Role:      string(msg.Role),
			UserInput: content, // All messages put their content in userInput, frontend uses role to determine how to display
		}
		
		// For assistant messages, also set assistantResponse field
		if msg.Role != message.User {
			messageData.AssistantResponse = content
		}
		
		// Only set tool calls if there are any
		if len(toolCallsData) > 0 {
			messageData.ToolCalls = toolCallsData
		}
		
		// Add reasoning content if present
		if reasoning != "" {
			messageData.Reasoning = reasoning
			messageData.ReasoningDuration = reasoningDuration
		}
		
		result = append(result, messageData)
	}
	return result
}

// convertToExportSession converts session and messages to comprehensive export format
func (h *MessageHandler) convertToExportSession(session session.Session, messages []message.Message) ExportSession {
	exportMessages := make([]ExportMessage, 0, len(messages))

	for _, msg := range messages {
		exportMsg := ExportMessage{
			ID:        msg.ID,
			Role:      string(msg.Role),
			Content:   msg.Content().Text,
			Model:     string(msg.Model),
			CreatedAt: time.Unix(msg.CreatedAt, 0),
			UpdatedAt: time.Unix(msg.UpdatedAt, 0),
		}

		// Extract tool calls with comprehensive information
		toolCalls := msg.ToolCalls()
		toolResults := msg.ToolResults()

		if len(toolCalls) > 0 {
			exportToolCalls := make([]ExportToolCall, 0, len(toolCalls))

			// Create a map of tool results by tool call ID for quick lookup
			resultsByID := make(map[string]message.ToolResult)
			for _, tr := range toolResults {
				resultsByID[tr.ToolCallID] = tr
			}

			for _, tc := range toolCalls {
				exportTC := ExportToolCall{
					ID:       tc.ID,
					Name:     tc.Name,
					Input:    tc.Input,
					Type:     tc.Type,
					Finished: tc.Finished,
				}

				// Try to parse input as JSON for structured display
				var inputJSON interface{}
				if err := json.Unmarshal([]byte(tc.Input), &inputJSON); err == nil {
					exportTC.InputJSON = inputJSON
				}

				// Add result if available
				if toolResult, exists := resultsByID[tc.ID]; exists {
					exportTC.Result = toolResult.Content
					exportTC.Metadata = toolResult.Metadata
					exportTC.IsError = toolResult.IsError
				}

				exportToolCalls = append(exportToolCalls, exportTC)
			}

			exportMsg.ToolCalls = exportToolCalls
		}

		// Extract reasoning content
		reasoningContent := msg.ReasoningContent()
		if reasoningContent.Thinking != "" {
			exportMsg.Reasoning = reasoningContent.Thinking
			exportMsg.ReasoningDuration = reasoningContent.Duration
		}

		// Extract thinking blocks
		thinkingBlocks := msg.ThinkingBlocks()
		if len(thinkingBlocks) > 0 {
			exportMsg.ThinkingBlocks = make([]string, 0, len(thinkingBlocks))
			for _, block := range thinkingBlocks {
				exportMsg.ThinkingBlocks = append(exportMsg.ThinkingBlocks, block.Thinking)
			}
		}

		// Extract redacted thinking blocks
		redactedBlocks := msg.RedactedThinkingBlocks()
		if len(redactedBlocks) > 0 {
			exportMsg.RedactedThinkingBlocks = make([]string, 0, len(redactedBlocks))
			for _, block := range redactedBlocks {
				exportMsg.RedactedThinkingBlocks = append(exportMsg.RedactedThinkingBlocks, block.Data)
			}
		}

		// Extract finish reason
		if finishPart := msg.FinishPart(); finishPart != nil {
			exportMsg.FinishReason = string(finishPart.Reason)
		}

		exportMessages = append(exportMessages, exportMsg)
	}

	return ExportSession{
		ID:                    session.ID,
		Title:                 session.Title,
		UserMessageCount:      session.UserMessageCount,
		AssistantMessageCount: session.AssistantMessageCount,
		ToolCallCount:         session.ToolCallCount,
		PromptTokens:          session.PromptTokens,
		CompletionTokens:      session.CompletionTokens,
		Cost:                  session.Cost,
		CreatedAt:             time.Unix(session.CreatedAt, 0),
		UpdatedAt:             time.Unix(session.UpdatedAt, 0),
		Messages:              exportMessages,
	}
}