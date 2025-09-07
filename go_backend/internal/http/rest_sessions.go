package http

import (
	"net/http"
	"time"

	"mix/internal/app"
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
	WorkingDirectory      string    `json:"workingDirectory,omitempty"`
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

	var result []SessionData
	for _, s := range sessions {
		workingDir := ""
		if s.WorkingDirectory.Valid {
			workingDir = s.WorkingDirectory.String
		}

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
			WorkingDirectory:      workingDir,
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
		WorkingDirectory:      session.WorkingDirectory,
	}

	sendJSONResponse(w, http.StatusOK, result)
}

// CreateSessionRequest represents the request body for creating a session
type CreateSessionRequest struct {
	Title            string `json:"title"`
	WorkingDirectory string `json:"workingDirectory,omitempty"`
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

	ctx := r.Context()
	session, err := h.app.Sessions.Create(ctx, req.Title, req.WorkingDirectory)
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
		WorkingDirectory:      session.WorkingDirectory,
	}

	sendJSONResponse(w, http.StatusCreated, result)
}

// ForkSessionRequest represents the request body for forking a session
type ForkSessionRequest struct {
	SourceSessionID string `json:"sourceSessionId"`
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

	var req ForkSessionRequest
	if err := parseJSONBody(r, &req); err != nil {
		sendValidationError(w, "body", err.Error())
		return
	}

	if req.SourceSessionID == "" {
		sendValidationError(w, "sourceSessionId", "source session ID is required")
		return
	}

	if req.MessageIndex <= 0 {
		sendValidationError(w, "messageIndex", "message index must be > 0")
		return
	}

	// Use a default title if not provided
	title := req.Title
	if title == "" {
		title = "Forked Session"
	}

	ctx := r.Context()
	newSession, err := h.app.Sessions.Fork(ctx, req.SourceSessionID, title)
	if err != nil {
		sendInternalError(w, "forking session", err)
		return
	}

	// Copy messages to the new session
	err = h.app.Messages.CopyMessagesToSession(ctx, req.SourceSessionID, newSession.ID, req.MessageIndex)
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
		WorkingDirectory:      newSession.WorkingDirectory,
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
		sendInternalError(w, "deleting session", err)
		return
	}

	// Return 204 No Content for successful deletion
	w.WriteHeader(http.StatusNoContent)
}