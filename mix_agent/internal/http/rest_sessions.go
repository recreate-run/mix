package http

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"time"

	"mix/internal/app"
	"mix/internal/constants"
	"mix/internal/llm/interfaces"
	session2 "mix/internal/session"
)

// Prompt size limits
const (
	MaxReplacePromptSize = 100 * 1024 // 100KB
	MaxAppendPromptSize  = 50 * 1024  // 50KB
)

// Sentinel errors
var (
	ErrMessageNotFound = errors.New("message not found")
)

// SessionData represents session information for REST API
type SessionData struct {
	ID                    string                      `json:"id"`
	ParentSessionID       string                      `json:"parentSessionId,omitempty"`
	ParentToolCallID      string                      `json:"parentToolCallId,omitempty"`
	Title                 string                      `json:"title"`
	SessionType           string                      `json:"sessionType"`
	SubagentType          string                      `json:"subagentType,omitempty"`
	BrowserMode           string                      `json:"browserMode"`
	CdpUrl                string                      `json:"cdpUrl,omitempty"`
	UserMessageCount      int64                       `json:"userMessageCount"`
	AssistantMessageCount int64                       `json:"assistantMessageCount"`
	ToolCallCount         int64                       `json:"toolCallCount"`
	PromptTokens          int64                       `json:"promptTokens"`
	CompletionTokens      int64                       `json:"completionTokens"`
	Cost                  float64                     `json:"cost"`
	CreatedAt             time.Time                   `json:"createdAt"`
	FirstUserMessage      string                      `json:"firstUserMessage,omitempty"`
	Callbacks             []interfaces.CallbackConfig `json:"callbacks,omitempty"` // Session-level callbacks
}

// SessionHandler handles REST endpoints for session operations
type SessionHandler struct {
	app *app.App
}

// sessionToData converts a Session to SessionData for API responses.
// Returns an error if callback parsing fails.
func sessionToData(session session2.Session) (SessionData, error) {
	callbacks, err := session.GetCallbacks()
	if err != nil {
		return SessionData{}, fmt.Errorf("failed to parse session callbacks: %w", err)
	}

	return SessionData{
		ID:                    session.ID,
		ParentSessionID:       session.ParentSessionID,
		ParentToolCallID:      session.ParentToolCallID,
		Title:                 session.Title,
		SessionType:           session.SessionType.String(),
		SubagentType:          session.SubagentType.String(),
		BrowserMode:           session.BrowserMode,
		CdpUrl:                session.CdpUrl,
		UserMessageCount:      session.UserMessageCount,
		AssistantMessageCount: session.AssistantMessageCount,
		ToolCallCount:         session.ToolCallCount,
		PromptTokens:          session.PromptTokens,
		CompletionTokens:      session.CompletionTokens,
		Cost:                  session.Cost,
		CreatedAt:             time.Unix(session.CreatedAt, 0),
		Callbacks:             callbacks,
	}, nil
}

// NewSessionHandler creates a new session handler
func NewSessionHandler(a *app.App) *SessionHandler {
	return &SessionHandler{app: a}
}

// HandleListSessions handles GET /api/sessions
func (h *SessionHandler) HandleListSessions(w http.ResponseWriter, r *http.Request) {
	setCORSHeaders(w)
	if handleCORSPreflight(w, r) {
		return
	}

	if r.Method != http.MethodGet {
		http.Error(w, constants.MethodNotAllowed, http.StatusMethodNotAllowed)
		return
	}

	ctx := r.Context()

	// Check if subagent sessions should be included (for loading subagent timeline)
	includeSubagents := r.URL.Query().Get("includeSubagents") == "true"

	sessions, err := h.app.Sessions.ListWithContent(ctx)
	if err != nil {
		sendInternalError(w, "listing sessions", err)
		return
	}

	// Initialize as empty slice instead of nil to ensure JSON encodes as [] not null
	result := make([]SessionData, 0)
	for i := range sessions {
		// Only include main sessions by default - hide subagent sessions
		// unless explicitly requested via query parameter
		if sessions[i].SessionType == "subagent" && !includeSubagents {
			continue
		}

		result = append(result, SessionData{
			ID:                    sessions[i].ID,
			ParentSessionID:       sessions[i].ParentSessionID.String,
			ParentToolCallID:      sessions[i].ParentToolCallID.String,
			Title:                 sessions[i].Title,
			SessionType:           sessions[i].SessionType,         // String field from db.ListSessionsWithContentRow
			SubagentType:          sessions[i].SubagentType.String, // String field from db.ListSessionsWithContentRow
			BrowserMode:           sessions[i].BrowserMode,
			CdpUrl:                sessions[i].CdpUrl.String,
			UserMessageCount:      sessions[i].UserMessageCount,
			AssistantMessageCount: sessions[i].AssistantMessageCount,
			ToolCallCount:         sessions[i].ToolCallCount,
			PromptTokens:          sessions[i].PromptTokens,
			CompletionTokens:      sessions[i].CompletionTokens,
			Cost:                  sessions[i].Cost,
			CreatedAt:             time.Unix(sessions[i].CreatedAt, 0),
			// FirstUserMessage is intentionally left empty (default empty string)
		})
	}

	sendJSONResponse(w, http.StatusOK, result)
}

// HandleGetSession handles GET /api/sessions/{id}
func (h *SessionHandler) HandleGetSession(w http.ResponseWriter, r *http.Request) {
	setCORSHeaders(w)
	if handleCORSPreflight(w, r) {
		return
	}

	if r.Method != http.MethodGet {
		http.Error(w, constants.MethodNotAllowed, http.StatusMethodNotAllowed)
		return
	}

	sessionID := r.PathValue("id")
	if sessionID == "" {
		sendValidationError(w, "id", "session ID is required")
		return
	}

	ctx := r.Context()
	session, err := h.app.Sessions.Get(ctx, sessionID)
	if err != nil {
		sendNotFoundError(w, "Session", sessionID)
		return
	}

	result, err := sessionToData(session)
	if err != nil {
		sendInternalError(w, "converting session data", err)
		return
	}

	sendJSONResponse(w, http.StatusOK, result)
}

// CreateSessionRequest represents the request body for creating a session
type CreateSessionRequest struct {
	Title              string                      `json:"title"`
	CustomSystemPrompt string                      `json:"customSystemPrompt,omitempty"`
	PromptMode         string                      `json:"promptMode,omitempty"`
	SessionType        string                      `json:"sessionType,omitempty"`  // Only "main" or empty allowed
	SubagentType       string                      `json:"subagentType,omitempty"` // Must be empty for API-created sessions
	BrowserMode        string                      `json:"browserMode"`            // Required: "electron-embedded-browser", "local-browser-service", or "remote-cdp-websocket"
	CdpUrl             string                      `json:"cdpUrl,omitempty"`       // Required if browserMode is "remote-cdp-websocket"
	Callbacks          []interfaces.CallbackConfig `json:"callbacks,omitempty"`    // Session-level callbacks
}

// HandleCreateSession handles POST /api/sessions
func (h *SessionHandler) HandleCreateSession(w http.ResponseWriter, r *http.Request) {
	setCORSHeaders(w)
	if handleCORSPreflight(w, r) {
		return
	}

	if r.Method != http.MethodPost {
		http.Error(w, constants.MethodNotAllowed, http.StatusMethodNotAllowed)
		return
	}

	var req CreateSessionRequest
	if err := parseJSONBody(r, &req); err != nil {
		sendValidationError(w, "body", err.Error())
		return
	}

	if req.Title == "" {
		sendValidationError(w, "title", "title is required")
		return
	}

	// Truncate title to enforce maximum length
	title := session2.TruncateTitle(req.Title)

	// Set default prompt mode if not specified
	promptMode := req.PromptMode
	if promptMode == "" {
		promptMode = "default"
	}

	// Validate prompt mode
	if promptMode != "default" && promptMode != "append" && promptMode != "replace" {
		sendValidationError(w, "promptMode", "promptMode must be 'default', 'append', or 'replace'")
		return
	}

	// Validate custom prompt size based on mode
	if req.CustomSystemPrompt != "" {
		promptSize := len(req.CustomSystemPrompt)

		switch promptMode {
		case "replace":
			if promptSize > MaxReplacePromptSize {
				sendValidationError(w, "customSystemPrompt", fmt.Sprintf("Custom prompt size (%dKB) exceeds replace mode limit of %dKB", promptSize/1024, MaxReplacePromptSize/1024))
				return
			}
		case "append":
			if promptSize > MaxAppendPromptSize {
				sendValidationError(w, "customSystemPrompt", fmt.Sprintf("Custom prompt size (%dKB) exceeds append mode limit of %dKB", promptSize/1024, MaxAppendPromptSize/1024))
				return
			}
			// default mode ignores custom prompt, so no size check needed
		}
	}

	// Validate session type - API can only create main sessions
	// Subagent sessions are created programmatically through dedicated flows
	if req.SessionType != "" && req.SessionType != "main" {
		sendValidationError(w, "sessionType", "API can only create main sessions. Subagent sessions are created automatically.")
		return
	}

	// Subagent type must not be set for API-created sessions
	if req.SubagentType != "" {
		sendValidationError(w, "subagentType", "subagentType cannot be set for API-created sessions. Subagent sessions are created programmatically by the task delegation system.")
		return
	}

	ctx := r.Context()
	// Browser mode and cdpUrl validation is handled by session service
	session, err := h.app.Sessions.Create(ctx, title, req.CustomSystemPrompt, promptMode, session2.SessionTypeMain, "", "", "", req.BrowserMode, req.CdpUrl)
	if err != nil {
		sendInternalError(w, "creating session", err)
		return
	}

	// Set callbacks if provided
	if len(req.Callbacks) > 0 {
		if err := session.SetCallbacks(req.Callbacks); err != nil {
			sendValidationError(w, "callbacks", err.Error())
			return
		}

		// Save session with callbacks
		session, err = h.app.Sessions.Save(ctx, session)
		if err != nil {
			sendInternalError(w, "saving session callbacks", err)
			return
		}
	}

	// Track session creation
	if h.app.Analytics != nil {
		hasCustomPrompt := req.CustomSystemPrompt != ""
		customPromptLength := len(req.CustomSystemPrompt)
		_ = h.app.Analytics.TrackSessionCreated(ctx, session.ID, req.Title, hasCustomPrompt, promptMode, customPromptLength)
	}

	result, err := sessionToData(session)
	if err != nil {
		sendInternalError(w, "converting session data", err)
		return
	}

	sendJSONResponse(w, http.StatusCreated, result)
}

// HandleDeleteSession handles DELETE /api/sessions/{id}
func (h *SessionHandler) HandleDeleteSession(w http.ResponseWriter, r *http.Request) {
	setCORSHeaders(w)
	if handleCORSPreflight(w, r) {
		return
	}

	if r.Method != http.MethodDelete {
		http.Error(w, constants.MethodNotAllowed, http.StatusMethodNotAllowed)
		return
	}

	sessionID := r.PathValue("id")
	if sessionID == "" {
		sendValidationError(w, "id", "session ID is required")
		return
	}

	ctx := r.Context()

	// Get session data before deletion for analytics
	session, err := h.app.Sessions.Get(ctx, sessionID)
	if err != nil {
		sendNotFoundError(w, "Session", sessionID)
		return
	}

	// Calculate session age in seconds
	sessionAgeSeconds := time.Now().Unix() - session.CreatedAt

	// Get message count for analytics
	messages, err := h.app.Messages.List(ctx, sessionID)
	messageCount := 0
	if err == nil {
		messageCount = len(messages)
	}

	err = h.app.Sessions.Delete(ctx, sessionID)
	if err != nil {
		// Check if the error is because the session doesn't exist
		if errors.Is(err, sql.ErrNoRows) {
			sendNotFoundError(w, "Session", sessionID)
			return
		}
		sendInternalError(w, "deleting session", err)
		return
	}

	// Track session deletion
	if h.app.Analytics != nil {
		_ = h.app.Analytics.TrackSessionDeleted(ctx, sessionID, sessionAgeSeconds, messageCount, session.Cost)
	}

	// Return 204 No Content for successful deletion
	w.WriteHeader(http.StatusNoContent)
}

// UpdateCallbacksRequest represents the request body for updating session callbacks
type UpdateCallbacksRequest struct {
	Callbacks []interfaces.CallbackConfig `json:"callbacks"`
}

// HandleUpdateSessionCallbacks handles PATCH /api/sessions/{id}/callbacks
func (h *SessionHandler) HandleUpdateSessionCallbacks(w http.ResponseWriter, r *http.Request) {
	setCORSHeaders(w)
	if handleCORSPreflight(w, r) {
		return
	}

	if r.Method != http.MethodPatch {
		http.Error(w, constants.MethodNotAllowed, http.StatusMethodNotAllowed)
		return
	}

	sessionID := r.PathValue("id")
	if sessionID == "" {
		sendValidationError(w, "id", "session ID is required")
		return
	}

	var req UpdateCallbacksRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		sendValidationError(w, "body", fmt.Sprintf("Invalid JSON: %v", err))
		return
	}

	ctx := r.Context()

	// Get existing session
	session, err := h.app.Sessions.Get(ctx, sessionID)
	if err != nil {
		sendNotFoundError(w, "Session", sessionID)
		return
	}

	// Update callbacks (validation happens inside SetCallbacks)
	if err := session.SetCallbacks(req.Callbacks); err != nil {
		sendValidationError(w, "callbacks", err.Error())
		return
	}

	// Save updated session
	updatedSession, err := h.app.Sessions.Save(ctx, session)
	if err != nil {
		sendInternalError(w, "updating session callbacks", err)
		return
	}

	result, err := sessionToData(updatedSession)
	if err != nil {
		sendInternalError(w, "converting session data", err)
		return
	}

	sendJSONResponse(w, http.StatusOK, result)
}

// RewindSessionRequest represents the request body for rewinding a session
type RewindSessionRequest struct {
	MessageID    string `json:"messageId"`    // Keep messages up to and including this message ID, delete rest
	CleanupMedia bool   `json:"cleanupMedia"` // Whether to clean up associated media files
}

// rewindPoint holds the result of finding a rewind point in the message list
type rewindPoint struct {
	timestamp       int64 // timestamp of the first message to delete (or rewind message if no messages after)
	messageIndex    int
	messagesDeleted int
}

// findRewindPoint finds the message with the given ID and calculates rewind metadata
func (h *SessionHandler) findRewindPoint(ctx context.Context, sessionID, messageID string) (*rewindPoint, error) {
	allMessages, err := h.app.Messages.List(ctx, sessionID)
	if err != nil {
		return nil, fmt.Errorf("fetching messages: %w", err)
	}

	for i := range allMessages {
		if allMessages[i].ID == messageID {
			// Use the timestamp of the NEXT message (first to delete) if it exists
			// Otherwise use the rewind message timestamp (no cleanup needed)
			cleanupTimestamp := allMessages[i].CreatedAt
			if i+1 < len(allMessages) {
				cleanupTimestamp = allMessages[i+1].CreatedAt
			}

			return &rewindPoint{
				timestamp:       cleanupTimestamp,
				messageIndex:    i,
				messagesDeleted: len(allMessages) - i - 1,
			}, nil
		}
	}

	return nil, fmt.Errorf("%w: %s", ErrMessageNotFound, messageID)
}

// performRewindCleanup deletes messages and optionally cleans up media files
func (h *SessionHandler) performRewindCleanup(ctx context.Context, sessionID string, point *rewindPoint, cleanupMedia bool) error {
	// Delete messages after the rewind point
	if err := h.app.Messages.DeleteAfterIndex(ctx, sessionID, int64(point.messageIndex)); err != nil {
		return fmt.Errorf("rewinding session: %w", err)
	}

	// Clean up media files created after rewind timestamp
	if cleanupMedia {
		sessionStorageDir := session2.GetSessionStoragePath(sessionID, h.app.StorageConfig)
		err := session2.CleanupMediaByTimestamp(sessionStorageDir, point.timestamp)
		if err != nil {
			// Log error but don't fail the request - media cleanup is non-critical
			fmt.Printf("Warning: Failed to cleanup media for session %s: %v\n", sessionID, err)
		}
	}

	return nil
}

// buildSessionResponse creates a SessionData response from a session
func buildSessionResponse(sess session2.Session) SessionData {
	return SessionData{
		ID:                    sess.ID,
		ParentSessionID:       sess.ParentSessionID,
		ParentToolCallID:      sess.ParentToolCallID,
		Title:                 sess.Title,
		SessionType:           sess.SessionType.String(),
		SubagentType:          sess.SubagentType.String(),
		UserMessageCount:      sess.UserMessageCount,
		AssistantMessageCount: sess.AssistantMessageCount,
		ToolCallCount:         sess.ToolCallCount,
		PromptTokens:          sess.PromptTokens,
		CompletionTokens:      sess.CompletionTokens,
		Cost:                  sess.Cost,
		CreatedAt:             time.Unix(sess.CreatedAt, 0),
	}
}

// HandleRewindSession handles POST /api/sessions/{id}/rewind
func (h *SessionHandler) HandleRewindSession(w http.ResponseWriter, r *http.Request) {
	setCORSHeaders(w)
	if handleCORSPreflight(w, r) {
		return
	}

	if r.Method != http.MethodPost {
		http.Error(w, constants.MethodNotAllowed, http.StatusMethodNotAllowed)
		return
	}

	sessionID := r.PathValue("id")
	if sessionID == "" {
		sendValidationError(w, "id", "session ID is required")
		return
	}

	var req RewindSessionRequest
	if err := parseJSONBody(r, &req); err != nil {
		sendValidationError(w, "body", err.Error())
		return
	}

	if req.MessageID == "" {
		sendValidationError(w, "messageId", "message ID is required")
		return
	}

	ctx := r.Context()

	// Verify session exists
	if _, err := h.app.Sessions.Get(ctx, sessionID); err != nil {
		sendNotFoundError(w, "Session", sessionID)
		return
	}

	// Find the rewind point
	point, err := h.findRewindPoint(ctx, sessionID, req.MessageID)
	if err != nil {
		if errors.Is(err, ErrMessageNotFound) {
			sendNotFoundError(w, "Message", req.MessageID)
			return
		}
		sendInternalError(w, err.Error(), err)
		return
	}

	// Perform cleanup operations
	if err := h.performRewindCleanup(ctx, sessionID, point, req.CleanupMedia); err != nil {
		sendInternalError(w, err.Error(), err)
		return
	}

	// Track session rewind
	if h.app.Analytics != nil {
		_ = h.app.Analytics.TrackSessionRewound(ctx, sessionID, req.MessageID, point.messagesDeleted, req.CleanupMedia)
	}

	// Get updated session data
	updatedSession, err := h.app.Sessions.Get(ctx, sessionID)
	if err != nil {
		sendInternalError(w, "fetching session", err)
		return
	}

	sendJSONResponse(w, http.StatusOK, buildSessionResponse(updatedSession))
}
