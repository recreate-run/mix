package http

import (
	"context"
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"mix/internal/app"
	"mix/internal/commands"
	"mix/internal/llm/provider"
	"mix/internal/llm/tools"
	"mix/internal/logging"
	"mix/internal/message"
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
	Content string `json:"content"`
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

	if req.Content == "" {
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

	// If not authenticated, show a clear error message
	if !authenticated {
		helpfulMsg := "⚠️ Authentication required. Please use /login command to authenticate with Claude using an API key.\n\n" +
			"To login:\n" +
			"1. Visit https://console.anthropic.com/settings/keys\n" +
			"2. Create an API key\n" +
			"3. Use the /login command to authenticate"

		result := MessageData{
			ID:                "system-auth-prompt",
			Role:              "assistant",
			UserInput:         req.Content,
			AssistantResponse: helpfulMsg,
		}

		sendJSONResponse(w, http.StatusOK, result)
		return
	}

	// Check if this is a slash command and handle it immediately
	if commands.IsSlashCommand(req.Content) {
		parsed, parseErr := commands.ParseCommand(req.Content)
		if parseErr != nil {
			sendValidationError(w, "content", "Invalid slash command: "+parseErr.Error())
			return
		}

		logging.Info("Executing command", "name", parsed.Name, "args", parsed.Arguments)

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
				logging.Info("Available commands", "commands", commandNames)

				sendErrorResponse(w, ErrorTypeNotFound, fmt.Sprintf("Command '%s' not found. Available commands: %v", parsed.Name, commandNames))
				return
			}

			sendInternalError(w, "executing command", execErr)
			return
		}

		logging.Info("Command executed successfully", "name", parsed.Name, "result_length", len(commandResult))

		// Return the command result immediately as a message
		result := MessageData{
			ID:                "cmd-" + parsed.Name,
			Role:              "assistant",
			UserInput:         req.Content,
			AssistantResponse: commandResult,
		}

		sendJSONResponse(w, http.StatusOK, result)
		return
	}

	// Send message to agent
	done, err := h.app.CoderAgent.Run(ctx, sessionID, req.Content)
	if err != nil {
		sendInternalError(w, "sending message to agent", err)
		return
	}

	// Wait for response
	result := <-done

	// Check for processing errors
	if result.Error != nil {
		// Convert error to user-friendly message
		errorMessage := result.Error.Error()

		// Special handling for auth errors
		if strings.Contains(errorMessage, "401") || strings.Contains(errorMessage, "authentication") {
			authResult := MessageData{
				ID:                "system-auth-prompt",
				Role:              "assistant",
				UserInput:         req.Content,
				AssistantResponse: "⚠️ Authentication required. Please use the /login command to authenticate with Claude API key.",
			}

			sendJSONResponse(w, http.StatusOK, authResult)
			return
		}

		sendInternalError(w, "agent processing", result.Error)
		return
	}

	// Extract text content from the response message
	response := ""
	if result.Message.Content().String() != "" {
		response = result.Message.Content().String()
	}

	messageData := MessageData{
		ID:                result.Message.ID,
		Role:              "user",
		UserInput:         req.Content,
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

	logging.Info("Loading messages for session", "sessionID", sessionID)

	ctx := r.Context()
	messages, err := h.app.Messages.List(ctx, sessionID)
	if err != nil {
		logging.Error("Failed to list session messages", "sessionID", sessionID, "error", err)
		sendInternalError(w, "listing session messages", err)
		return
	}

	logging.Info("Retrieved messages from database", "sessionID", sessionID, "count", len(messages))

	result := h.convertMessagesToData(messages)
	
	logging.Info("Converted messages to API format", "sessionID", sessionID, "resultCount", len(result))
	
	// Log first few message IDs and roles for debugging
	for i, msg := range result {
		if i >= 3 { // Only log first 3 messages to avoid spam
			break
		}
		logging.Info("Message details", "sessionID", sessionID, "index", i, "messageID", msg.ID, "role", msg.Role, "hasUserInput", msg.UserInput != "", "hasAssistantResponse", msg.AssistantResponse != "", "toolCallsCount", len(msg.ToolCalls))
	}

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
	h.app.CoderAgent.Cancel(sessionID)

	result := map[string]string{
		"status":    "cancelled",
		"sessionId": sessionID,
	}

	sendJSONResponse(w, http.StatusOK, result)
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