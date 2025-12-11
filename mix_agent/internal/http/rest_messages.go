package http

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
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

// CallbackResultData represents callback result information for REST API
type CallbackResultData struct {
	ToolCallID     string `json:"tool_call_id"`
	ToolName       string `json:"tool_name"`
	CallbackName   string `json:"callback_name,omitempty"`
	CallbackType   string `json:"callback_type"`
	Stdout         string `json:"stdout,omitempty"`
	Stderr         string `json:"stderr,omitempty"`
	ExitCode       int    `json:"exit_code,omitempty"`
	SubAgentID     string `json:"subagent_id,omitempty"`
	SubAgentResult string `json:"subagent_result,omitempty"`
	Success        bool   `json:"success"`
	Error          string `json:"error,omitempty"`
}

// MessageData represents message information for REST API
type MessageData struct {
	ID                string               `json:"id"`
	SessionID         string               `json:"sessionId"`
	Role              string               `json:"role"`
	UserInput         string               `json:"userInput"`
	AssistantResponse string               `json:"assistantResponse,omitempty"`
	ToolCalls         []ToolCallData       `json:"toolCalls,omitempty"`
	CallbackResults   []CallbackResultData `json:"callbackResults,omitempty"`
	Reasoning         string               `json:"reasoning,omitempty"`
	ReasoningDuration int64                `json:"reasoningDuration,omitempty"`
}

// ExportToolCall represents comprehensive tool call information for transcript export
type ExportToolCall struct {
	ID        string      `json:"id"`
	Name      string      `json:"name"`
	Input     string      `json:"input"`
	InputJSON interface{} `json:"inputJson,omitempty"` // Parsed JSON for structured tools
	Type      string      `json:"type"`
	Finished  bool        `json:"finished"`
	Result    string      `json:"result,omitempty"`
	Metadata  string      `json:"metadata,omitempty"`
	IsError   bool        `json:"isError,omitempty"`
}

// ExportMessage represents comprehensive message information for transcript export
type ExportMessage struct {
	ID                     string           `json:"id"`
	Role                   string           `json:"role"`
	Content                string           `json:"content"`
	ToolCalls              []ExportToolCall `json:"toolCalls,omitempty"`
	Reasoning              string           `json:"reasoning,omitempty"`
	ReasoningDuration      int64            `json:"reasoningDuration,omitempty"`
	ThinkingBlocks         []string         `json:"thinkingBlocks,omitempty"`
	RedactedThinkingBlocks []string         `json:"redactedThinkingBlocks,omitempty"`
	Model                  string           `json:"model,omitempty"`
	FinishReason           string           `json:"finishReason,omitempty"`
	CreatedAt              time.Time        `json:"createdAt"`
	UpdatedAt              time.Time        `json:"updatedAt"`
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
func NewMessageHandler(a *app.App) *MessageHandler {
	// Create command registry
	registry := commands.NewRegistry()
	if err := registry.LoadCommands(a); err != nil {
		logging.Error("Failed to load commands", "error", err)
		// Continue with empty registry - API will return proper errors
	}

	return &MessageHandler{
		app:             a,
		commandRegistry: registry,
	}
}

// SendMessageRequest represents the request body for sending a message
type SendMessageRequest struct {
	Text          string  `json:"text"`
	PlanMode      bool    `json:"plan_mode,omitempty"`
	ThinkingLevel *string `json:"thinking_level,omitempty"`
}

// thinkingLevelToBudget converts thinking level enum to token budget
func thinkingLevelToBudget(level *string) *int {
	if level == nil {
		return nil
	}

	var budget int
	switch *level {
	case "off":
		budget = 0
	case "basic":
		budget = 4000
	case "medium":
		budget = 10000
	case "maximum":
		budget = 31999
	default:
		return nil // Invalid level
	}

	return &budget
}

// Helper function to get command names for logging
func getCommandNames(cmds map[string]commands.Command) []string {
	names := make([]string, 0, len(cmds))
	for name := range cmds {
		names = append(names, name)
	}
	return names
}

// generateRequestID generates a unique request ID for correlation tracking
func generateRequestID() string {
	bytes := make([]byte, 8)
	if _, err := rand.Read(bytes); err != nil {
		// Fallback to timestamp if random generation fails
		return fmt.Sprintf("req-%d", time.Now().UnixNano())
	}
	return "req-" + hex.EncodeToString(bytes)
}

// HandleSendMessage handles POST /api/sessions/{id}/messages
func (h *MessageHandler) HandleSendMessage(w http.ResponseWriter, r *http.Request) {
	requestID := generateRequestID()
	requestStartTime := time.Now()

	defer func() {
		duration := time.Since(requestStartTime)
		logging.Debug("HTTP handler exiting",
			"requestID", requestID,
			"duration", duration.String(),
			"timestamp", time.Now().Format(time.RFC3339Nano))
	}()

	setCORSHeaders(w)
	if handleCORSPreflight(w, r) {
		return
	}

	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	sessionID, req, thinkingBudget, ok := h.validateSendMessageRequest(w, r)
	if !ok {
		return
	}

	ctx := r.Context()
	if !h.checkAuthentication(w, ctx, sessionID) {
		return
	}

	if commands.IsSlashCommand(req.Text) {
		h.handleSlashCommand(w, ctx, sessionID, req.Text)
		return
	}

	h.startAgentProcessing(w, sessionID, requestID, req, thinkingBudget)
}

func (h *MessageHandler) validateSendMessageRequest(w http.ResponseWriter, r *http.Request) (sessionID string, req SendMessageRequest, thinkingBudget *int, ok bool) {
	sessionID = r.PathValue("id")
	if sessionID == "" {
		sendValidationError(w, "id", "session ID is required")
		return "", SendMessageRequest{}, nil, false
	}

	if err := parseJSONBody(r, &req); err != nil {
		sendValidationError(w, "body", err.Error())
		return "", SendMessageRequest{}, nil, false
	}

	if req.Text == "" {
		sendValidationError(w, "content", "message content is required")
		return "", SendMessageRequest{}, nil, false
	}

	if req.ThinkingLevel != nil {
		level := *req.ThinkingLevel
		validLevels := map[string]bool{
			"off":     true,
			"basic":   true,
			"medium":  true,
			"maximum": true,
		}
		if !validLevels[level] {
			sendValidationError(w, "thinking_level", "must be one of: off, basic, medium, maximum")
			return "", SendMessageRequest{}, nil, false
		}
	}

	thinkingBudget = thinkingLevelToBudget(req.ThinkingLevel)
	return sessionID, req, thinkingBudget, true
}

func (h *MessageHandler) checkAuthentication(w http.ResponseWriter, ctx context.Context, sessionID string) bool {
	authenticated, _, authErr := provider.IsAuthenticated(ctx, "")
	if authErr != nil {
		sendInternalError(w, "checking authentication", authErr)
		return false
	}

	if !authenticated {
		helpfulMsg := getAuthenticationErrorMessage(ctx)
		registry.BroadcastEvent(sessionID, "error", ErrorEvent{Error: helpfulMsg})
		sendErrorResponse(w, ErrorTypeUnauthorized, helpfulMsg)
		return false
	}

	return true
}

func (h *MessageHandler) handleSlashCommand(w http.ResponseWriter, ctx context.Context, sessionID, text string) {
	parsed, parseErr := commands.ParseCommand(text)
	if parseErr != nil {
		sendValidationError(w, "content", "Invalid slash command: "+parseErr.Error())
		return
	}

	cmdCtx := context.WithValue(ctx, tools.SessionIDContextKey, sessionID)
	commandResult, execErr := h.commandRegistry.ExecuteCommand(cmdCtx, parsed.Name, parsed.Arguments)
	if execErr != nil {
		logging.Error("Command execution failed", "name", parsed.Name, "error", execErr)

		if strings.Contains(execErr.Error(), "command not found") {
			allCommands := h.commandRegistry.GetAllCommands()
			commandNames := getCommandNames(allCommands)
			sendErrorResponse(w, ErrorTypeNotFound, fmt.Sprintf("Command '%s' not found. Available commands: %v", parsed.Name, commandNames))
			return
		}

		sendInternalError(w, "executing command", execErr)
		return
	}

	result := MessageData{
		ID:                "cmd-" + parsed.Name,
		Role:              "assistant",
		UserInput:         text,
		AssistantResponse: commandResult,
	}

	sendJSONResponse(w, http.StatusOK, result)
}

func (h *MessageHandler) startAgentProcessing(w http.ResponseWriter, sessionID, requestID string, req SendMessageRequest, thinkingBudget *int) {
	agentCtx := context.Background()

	events, err := h.app.CoderAgent.RunWithPlanMode(agentCtx, sessionID, req.Text, req.PlanMode, thinkingBudget)
	if err != nil {
		logging.Error("Failed to start agent processing",
			"sessionID", sessionID,
			"requestID", requestID,
			"error", err)

		// Check if the error is due to session not found
		if errors.Is(err, session.ErrSessionNotFound) {
			sendNotFoundError(w, "Session", sessionID)
			return
		}

		sendInternalError(w, "sending message to agent", err)
		return
	}

	sendJSONResponse(w, http.StatusAccepted, map[string]string{
		"status":    "processing",
		"sessionId": sessionID,
	})

	go h.processAgentEvents(sessionID, requestID, events)
}

func (h *MessageHandler) processAgentEvents(sessionID, requestID string, events <-chan agent.AgentEvent) {
	defer func() {
		if r := recover(); r != nil {
			logging.Error("Panic in background event processing",
				"sessionID", sessionID,
				"requestID", requestID,
				"panic", r)
		}
	}()

	var lastEvent agent.AgentEvent
	for event := range events {
		BroadcastAgentEventToSSE(sessionID, event)
		lastEvent = event
	}

	if lastEvent.Error != nil {
		h.broadcastErrorToSSE(sessionID, lastEvent.Error)
	}
}

func (h *MessageHandler) broadcastErrorToSSE(sessionID string, err error) {
	errorMessage := err.Error()

	if strings.Contains(errorMessage, "401") || strings.Contains(errorMessage, "authentication") {
		registry.BroadcastEvent(sessionID, "error", ErrorEvent{
			Error: "⚠️ Authentication required. Please use the /login command to authenticate with Claude API key.",
		})
	} else {
		registry.BroadcastEvent(sessionID, "error", ErrorEvent{
			Error: errorMessage,
		})
	}
}

// HandleListSessionMessages handles GET /api/sessions/{id}/messages
func (h *MessageHandler) HandleListSessionMessages(w http.ResponseWriter, r *http.Request) {
	setCORSHeaders(w)
	if handleCORSPreflight(w, r) {
		return
	}

	if r.Method != http.MethodGet {
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

	if r.Method != http.MethodGet {
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

	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	sessionID := r.PathValue("id")
	if sessionID == "" {
		sendValidationError(w, "id", "session ID is required")
		return
	}

	// Cancel the agent processing for this session
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

	if r.Method != http.MethodGet {
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
	sess, err := h.app.Sessions.Get(ctx, sessionID)
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
	exportData := h.convertToExportSession(sess, messages)

	// Track session export
	if h.app.Analytics != nil {
		totalTokens := sess.PromptTokens + sess.CompletionTokens
		_ = h.app.Analytics.TrackSessionExported(ctx, sessionID, len(messages), sess.Cost, totalTokens)
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
		callbackResults := msg.CallbackResults()

		// Extract reasoning content from both ReasoningContent and ThinkingBlock parts
		var reasoning string
		var reasoningDuration int64
		for _, part := range msg.Parts {
			switch v := part.(type) {
			case message.ReasoningContent:
				if reasoning == "" { // Use first reasoning content found
					reasoning = v.Thinking
					reasoningDuration = v.Duration
				}
			case message.ThinkingBlock:
				if reasoning == "" { // Use thinking block if no reasoning content found
					reasoning = v.Thinking
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

		// Convert callback results
		if len(callbackResults) > 0 {
			callbackResultsData := make([]CallbackResultData, len(callbackResults))
			for i := range callbackResults {
				callbackResultsData[i] = CallbackResultData{
					ToolCallID:     callbackResults[i].ToolCallID,
					ToolName:       callbackResults[i].ToolName,
					CallbackName:   callbackResults[i].CallbackName,
					CallbackType:   callbackResults[i].CallbackType,
					Stdout:         callbackResults[i].Stdout,
					Stderr:         callbackResults[i].Stderr,
					ExitCode:       callbackResults[i].ExitCode,
					SubAgentID:     callbackResults[i].SubAgentID,
					SubAgentResult: callbackResults[i].SubAgentResult,
					Success:        callbackResults[i].Success,
					Error:          callbackResults[i].Error,
				}
			}
			messageData.CallbackResults = callbackResultsData
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
func (h *MessageHandler) convertToExportSession(sess session.Session, messages []message.Message) ExportSession {
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
		ID:                    sess.ID,
		Title:                 sess.Title,
		UserMessageCount:      sess.UserMessageCount,
		AssistantMessageCount: sess.AssistantMessageCount,
		ToolCallCount:         sess.ToolCallCount,
		PromptTokens:          sess.PromptTokens,
		CompletionTokens:      sess.CompletionTokens,
		Cost:                  sess.Cost,
		CreatedAt:             time.Unix(sess.CreatedAt, 0),
		UpdatedAt:             time.Unix(sess.UpdatedAt, 0),
		Messages:              exportMessages,
	}
}
