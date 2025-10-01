package http

import (
	"fmt"
	"net/http"
	"strings"
	"time"

	"mix/internal/app"
	session2 "mix/internal/session"
)

// Prompt size limits
const (
	MaxReplacePromptSize = 100 * 1024 // 100KB
	MaxAppendPromptSize  = 50 * 1024  // 50KB
)

// SessionData represents session information for REST API
type SessionData struct {
	ID                    string    `json:"id"`
	Title                 string    `json:"title"`
	UserMessageCount      int64     `json:"userMessageCount"`
	AssistantMessageCount int64     `json:"assistantMessageCount"`
	ToolCallCount         int64     `json:"toolCallCount"`
	PromptTokens          int64     `json:"promptTokens"`
	CompletionTokens      int64     `json:"completionTokens"`
	Cost                  float64   `json:"cost"`
	CreatedAt             time.Time `json:"createdAt"`
	FirstUserMessage      string    `json:"firstUserMessage,omitempty"`
}

// SessionHandler handles REST endpoints for session operations
type SessionHandler struct {
	app *app.App
}

// NewSessionHandler creates a new session handler
func NewSessionHandler(app *app.App) *SessionHandler {
	return &SessionHandler{app: app}
}

// HandleListSessions handles GET /api/sessions
func (h *SessionHandler) HandleListSessions(w http.ResponseWriter, r *http.Request) {
	setCORSHeaders(w)
	if handleCORSPreflight(w, r) {
		return
	}

	if r.Method != "GET" {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	ctx := r.Context()
	sessions, err := h.app.Sessions.ListWithContent(ctx)
	if err != nil {
		sendInternalError(w, "listing sessions", err)
		return
	}

	// Initialize as empty slice instead of nil to ensure JSON encodes as [] not null
	result := make([]SessionData, 0)
	for _, s := range sessions {
		result = append(result, SessionData{
			ID:                    s.ID,
			Title:                 s.Title,
			UserMessageCount:      s.UserMessageCount,
			AssistantMessageCount: s.AssistantMessageCount,
			ToolCallCount:         s.ToolCallCount,
			PromptTokens:          s.PromptTokens,
			CompletionTokens:      s.CompletionTokens,
			Cost:                  s.Cost,
			CreatedAt:             time.Unix(s.CreatedAt, 0),
			FirstUserMessage:      s.FirstUserMessage,
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
	session, err := h.app.Sessions.Get(ctx, sessionID)
	if err != nil {
		sendNotFoundError(w, "Session", sessionID)
		return
	}

	result := SessionData{
		ID:                    session.ID,
		Title:                 session.Title,
		UserMessageCount:      session.UserMessageCount,
		AssistantMessageCount: session.AssistantMessageCount,
		ToolCallCount:         session.ToolCallCount,
		PromptTokens:          session.PromptTokens,
		CompletionTokens:      session.CompletionTokens,
		Cost:                  session.Cost,
		CreatedAt:             time.Unix(session.CreatedAt, 0),
	}

	sendJSONResponse(w, http.StatusOK, result)
}

// CreateSessionRequest represents the request body for creating a session
type CreateSessionRequest struct {
	Title              string `json:"title"`
	CustomSystemPrompt string `json:"customSystemPrompt,omitempty"`
	PromptMode         string `json:"promptMode,omitempty"`
}

// HandleCreateSession handles POST /api/sessions
func (h *SessionHandler) HandleCreateSession(w http.ResponseWriter, r *http.Request) {
	setCORSHeaders(w)
	if handleCORSPreflight(w, r) {
		return
	}

	if r.Method != "POST" {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
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

	ctx := r.Context()
	session, err := h.app.Sessions.Create(ctx, req.Title, req.CustomSystemPrompt, promptMode)
	if err != nil {
		sendInternalError(w, "creating session", err)
		return
	}

	result := SessionData{
		ID:                    session.ID,
		Title:                 session.Title,
		UserMessageCount:      session.UserMessageCount,
		AssistantMessageCount: session.AssistantMessageCount,
		ToolCallCount:         session.ToolCallCount,
		PromptTokens:          session.PromptTokens,
		CompletionTokens:      session.CompletionTokens,
		Cost:                  session.Cost,
		CreatedAt:             time.Unix(session.CreatedAt, 0),
	}

	sendJSONResponse(w, http.StatusCreated, result)
}

// ForkSessionRequest represents the request body for forking a session
type ForkSessionRequest struct {
	MessageIndex    int64  `json:"messageIndex"`
	Title           string `json:"title,omitempty"`
}

// HandleForkSession handles POST /api/sessions/{id}/fork
func (h *SessionHandler) HandleForkSession(w http.ResponseWriter, r *http.Request) {
	setCORSHeaders(w)
	if handleCORSPreflight(w, r) {
		return
	}

	if r.Method != "POST" {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	sourceSessionID := r.PathValue("id")
	if sourceSessionID == "" {
		sendValidationError(w, "id", "source session ID is required")
		return
	}

	var req ForkSessionRequest
	if err := parseJSONBody(r, &req); err != nil {
		sendValidationError(w, "body", err.Error())
		return
	}

	if req.MessageIndex < 0 {
		sendValidationError(w, "messageIndex", "message index must be >= 0")
		return
	}

	// Use a default title if not provided
	title := req.Title
	if title == "" {
		title = "Forked Session"
	}

	ctx := r.Context()
	newSession, err := h.app.Sessions.Fork(ctx, sourceSessionID, title)
	if err != nil {
		sendInternalError(w, "forking session", err)
		return
	}

	// Copy messages to the new session
	err = h.app.Messages.CopyMessagesToSession(ctx, sourceSessionID, newSession.ID, req.MessageIndex)
	if err != nil {
		sendInternalError(w, "copying messages", err)
		return
	}

	result := SessionData{
		ID:                    newSession.ID,
		Title:                 newSession.Title,
		UserMessageCount:      newSession.UserMessageCount,
		AssistantMessageCount: newSession.AssistantMessageCount,
		ToolCallCount:         newSession.ToolCallCount,
		PromptTokens:          newSession.PromptTokens,
		CompletionTokens:      newSession.CompletionTokens,
		Cost:                  newSession.Cost,
		CreatedAt:             time.Unix(newSession.CreatedAt, 0),
	}

	sendJSONResponse(w, http.StatusCreated, result)
}

// HandleDeleteSession handles DELETE /api/sessions/{id}
func (h *SessionHandler) HandleDeleteSession(w http.ResponseWriter, r *http.Request) {
	setCORSHeaders(w)
	if handleCORSPreflight(w, r) {
		return
	}

	if r.Method != "DELETE" {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	sessionID := r.PathValue("id")
	if sessionID == "" {
		sendValidationError(w, "id", "session ID is required")
		return
	}

	ctx := r.Context()
	err := h.app.Sessions.Delete(ctx, sessionID)
	if err != nil {
		// Check if the error is because the session doesn't exist
		if strings.Contains(err.Error(), "not found") || strings.Contains(err.Error(), "no rows") {
			sendNotFoundError(w, "Session", sessionID)
			return
		}
		sendInternalError(w, "deleting session", err)
		return
	}

	// Return 204 No Content for successful deletion
	w.WriteHeader(http.StatusNoContent)
}

// RewindSessionRequest represents the request body for rewinding a session
type RewindSessionRequest struct {
	MessageID    string `json:"messageId"`    // Keep messages up to and including this message ID, delete rest
	CleanupMedia bool   `json:"cleanupMedia"` // Whether to clean up associated media files
}

// HandleRewindSession handles POST /api/sessions/{id}/rewind
func (h *SessionHandler) HandleRewindSession(w http.ResponseWriter, r *http.Request) {
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
	_, err := h.app.Sessions.Get(ctx, sessionID)
	if err != nil {
		sendNotFoundError(w, "Session", sessionID)
		return
	}

	// Get all messages to find the rewind point
	allMessages, err := h.app.Messages.List(ctx, sessionID)
	if err != nil {
		sendInternalError(w, "fetching messages", err)
		return
	}

	// Find the message with the given ID and get its timestamp
	var rewindTimestamp int64
	var messageIndex int = -1
	for i, msg := range allMessages {
		if msg.ID == req.MessageID {
			rewindTimestamp = msg.CreatedAt
			messageIndex = i
			break
		}
	}

	if messageIndex == -1 {
		sendNotFoundError(w, "Message", req.MessageID)
		return
	}

	// Delete messages after the rewind point
	err = h.app.Messages.DeleteAfterIndex(ctx, sessionID, int64(messageIndex))
	if err != nil {
		sendInternalError(w, "rewinding session", err)
		return
	}

	// Clean up media files created after rewind timestamp
	if req.CleanupMedia {
		sessionStorageDir := session2.GetSessionStoragePath(sessionID, h.app.StorageConfig)
		err := session2.CleanupMediaByTimestamp(sessionStorageDir, rewindTimestamp)
		if err != nil {
			// Log error but don't fail the request - media cleanup is non-critical
			fmt.Printf("Warning: Failed to cleanup media for session %s: %v\n", sessionID, err)
		}
	}

	// Get updated session data
	updatedSession, err := h.app.Sessions.Get(ctx, sessionID)
	if err != nil {
		sendInternalError(w, "fetching session", err)
		return
	}

	result := SessionData{
		ID:                    updatedSession.ID,
		Title:                 updatedSession.Title,
		UserMessageCount:      updatedSession.UserMessageCount,
		AssistantMessageCount: updatedSession.AssistantMessageCount,
		ToolCallCount:         updatedSession.ToolCallCount,
		PromptTokens:          updatedSession.PromptTokens,
		CompletionTokens:      updatedSession.CompletionTokens,
		Cost:                  updatedSession.Cost,
		CreatedAt:             time.Unix(updatedSession.CreatedAt, 0),
	}

	sendJSONResponse(w, http.StatusOK, result)
}