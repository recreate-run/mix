package http

import (
	"encoding/json"
	"fmt"
	"net/http"
)

// SSE Event Types - Keep structs for type safety but remove interface overhead

type ErrorEvent struct {
	Error string `json:"error"`
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
}

type ToolEvent struct {
	Type   string `json:"type"`
	Name   string `json:"name"`
	Input  string `json:"input"`
	ID     string `json:"id"`
	Status string `json:"status"`
}

type SummarizeEvent struct {
	Type     string `json:"type"`
	Progress string `json:"progress"`
	Done     bool   `json:"done"`
}

// WriteSSE serializes and writes an SSE event to the response writer
func WriteSSE(w http.ResponseWriter, eventType string, data interface{}) error {
	jsonData, err := json.Marshal(data)
	if err != nil {
		return fmt.Errorf("failed to marshal SSE event data: %w", err)
	}
	
	_, err = fmt.Fprintf(w, "event: %s\ndata: %s\n\n", eventType, string(jsonData))
	if err != nil {
		return fmt.Errorf("failed to write SSE event: %w", err)
	}
	
	return nil
}