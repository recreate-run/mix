package http

import (
	"encoding/json"
	"fmt"
	"math"
	"net/http"
	"sync/atomic"

	"mix/internal/constants"
)

const (
	eventTypeError    = "error"
	eventTypeComplete = "complete"
)

// SSE Event Types - Keep structs for type safety but remove interface overhead

type ErrorEvent struct {
	Error            string `json:"error"`
	Type             string `json:"type,omitempty"`
	RetryAfter       int    `json:"retryAfter,omitempty"`
	Attempt          int    `json:"attempt,omitempty"`
	MaxAttempts      int    `json:"maxAttempts,omitempty"`
	ParentToolCallID string `json:"parentToolCallId,omitempty"`
}

type ConnectedEvent struct {
	SessionID string `json:"sessionId"`
}

type HeartbeatEvent struct {
	Type string `json:"type"`
}

type CompleteEvent struct {
	Type              string `json:"type"`
	Content           string `json:"content,omitempty"`
	MessageID         string `json:"messageId,omitempty"`
	Done              bool   `json:"done"`
	Reasoning         string `json:"reasoning,omitempty"`
	ReasoningDuration int64  `json:"reasoningDuration,omitempty"`
	ParentToolCallID  string `json:"parentToolCallId,omitempty"`
}

type ToolUseStartEvent struct {
	Type               string `json:"type"`
	Name               string `json:"name"`
	ID                 string `json:"id"`
	ParentToolCallID   string `json:"parentToolCallId,omitempty"`
	AssistantMessageID string `json:"assistantMessageId,omitempty"`
}

type ToolUseParameterStreamingCompleteEvent struct {
	Type               string `json:"type"`
	Name               string `json:"name"`
	Input              string `json:"input"`
	ID                 string `json:"id"`
	ParentToolCallID   string `json:"parentToolCallId,omitempty"`
	AssistantMessageID string `json:"assistantMessageId,omitempty"`
}

type ToolUseParameterDeltaEvent struct {
	Type               string `json:"type"`
	ToolCallID         string `json:"toolCallId"`
	Input              string `json:"input"` // Partial JSON parameter delta
	ParentToolCallID   string `json:"parentToolCallId,omitempty"`
	AssistantMessageID string `json:"assistantMessageId,omitempty"`
}

type PermissionEvent struct {
	Type             string      `json:"type"`
	ID               string      `json:"id"`
	SessionID        string      `json:"sessionId"`
	ToolName         string      `json:"toolName"`
	Description      string      `json:"description"`
	Action           string      `json:"action"`
	Path             string      `json:"path"`
	Params           interface{} `json:"params"`
	ParentToolCallID string      `json:"parentToolCallId,omitempty"`
}

type ThinkingEvent struct {
	Type               string `json:"type"`
	Content            string `json:"content"`
	ParentToolCallID   string `json:"parentToolCallId,omitempty"`
	AssistantMessageID string `json:"assistantMessageId,omitempty"`
}

type ToolExecutionStartEvent struct {
	Type             string `json:"type"`
	ToolName         string `json:"toolName"`
	Progress         string `json:"progress"`
	ToolCallID       string `json:"toolCallId"`
	ParentToolCallID string `json:"parentToolCallId,omitempty"`
}

type ToolExecutionCompleteEvent struct {
	Type             string `json:"type"`
	ToolName         string `json:"toolName"`
	Progress         string `json:"progress"`
	Success          bool   `json:"success"`
	ToolCallID       string `json:"toolCallId"`
	ParentToolCallID string `json:"parentToolCallId,omitempty"`
}

type ContentEvent struct {
	Type               string `json:"type"`
	Content            string `json:"content"`
	ParentToolCallID   string `json:"parentToolCallId,omitempty"`
	AssistantMessageID string `json:"assistantMessageId,omitempty"`
}

type SessionCreatedEvent struct {
	Type      string `json:"type"`
	SessionID string `json:"sessionId"`
	Title     string `json:"title"`
	CreatedAt int64  `json:"createdAt"`
}

type SessionDeletedEvent struct {
	Type      string `json:"type"`
	SessionID string `json:"sessionId"`
}

type UserMessageCreatedEvent struct {
	Type             string `json:"type"`
	MessageID        string `json:"messageId"`
	Content          string `json:"content"`
	ParentToolCallID string `json:"parentToolCallId,omitempty"`
}

// SSEWriter handles session-scoped SSE writing with automatic compliance
type SSEWriter struct {
	w         http.ResponseWriter
	sessionID string
	eventID   int64
	flusher   http.Flusher
}

// NewSSEWriter creates a new session-aware SSE writer
func NewSSEWriter(w http.ResponseWriter, sessionID string, flusher http.Flusher) *SSEWriter {
	return &SSEWriter{
		w:         w,
		sessionID: sessionID,
		eventID:   0,
		flusher:   flusher,
	}
}

// WriteEvent writes an SSE-compliant event with automatic ID generation and retry logic
func (s *SSEWriter) WriteEvent(eventType string, data interface{}) error {
	// Generate sequential event ID for this session
	eventID := atomic.AddInt64(&s.eventID, 1)

	// Marshal the data payload
	jsonData, err := json.Marshal(data)
	if err != nil {
		return fmt.Errorf("failed to marshal SSE event data: %w", err)
	}

	// Calculate retry interval based on event type
	retryMs := s.getRetryInterval(eventType, data)

	// Write SSE-compliant event with all standard fields
	_, err = fmt.Fprintf(s.w, "id: %d\nevent: %s\ndata: %s\nretry: %d\n\n",
		eventID, eventType, string(jsonData), retryMs)
	if err != nil {
		return fmt.Errorf("failed to write SSE event: %w", err)
	}

	// Flush immediately for real-time streaming
	if s.flusher != nil {
		s.flusher.Flush()
	}

	return nil
}

// getRetryInterval calculates appropriate retry interval based on event type and data
func (s *SSEWriter) getRetryInterval(eventType string, data interface{}) int {
	switch eventType {
	case eventTypeError:
		// Check if this is an ErrorEvent with exponential backoff
		if errorEvent, ok := data.(ErrorEvent); ok && errorEvent.RetryAfter > 0 {
			return errorEvent.RetryAfter
		}
		// Default error retry with exponential backoff calculation
		return s.calculateExponentialBackoff(1) // Start with attempt 1
	case "heartbeat":
		return 45000 // 45 seconds for heartbeat
	case eventTypeComplete:
		return 30000 // 30 seconds after completion
	default:
		return 30000 // Default 30 second retry for all other events
	}
}

// calculateExponentialBackoff implements exponential backoff with jitter
func (s *SSEWriter) calculateExponentialBackoff(attempt int) int {
	if attempt <= 0 {
		attempt = 1
	}

	// Exponential backoff: RetryInitialInterval * (RetryBackoffExponent ^ attempt) with max RetryMaxInterval
	backoffMs := int(float64(constants.RetryInitialInterval) * math.Pow(constants.RetryBackoffExponent, float64(attempt)))

	if backoffMs > constants.RetryMaxInterval {
		return constants.RetryMaxInterval
	}

	return backoffMs
}
