package interfaces

import (
	"context"
	"encoding/json"
)

// Context keys for session and message identification
type (
	sessionIDContextKey      string
	sessionContextKey        string
	messageIDContextKey      string
	sessionStorageContextKey string
	planModeContextKey       string
	thinkingBudgetContextKey string
)

const (
	SessionIDContextKey      sessionIDContextKey      = "session_id"
	SessionContextKey        sessionContextKey        = "session"
	MessageIDContextKey      messageIDContextKey      = "message_id"
	SessionStorageContextKey sessionStorageContextKey = "session_storage"
	PlanModeContextKey       planModeContextKey       = "plan_mode"
	ThinkingBudgetContextKey thinkingBudgetContextKey = "thinking_budget"
)

type ToolInfo struct {
	Name        string
	Description string
	Parameters  map[string]any
	Required    []string
}

type toolResponseType string

const (
	ToolResponseTypeText  toolResponseType = "text"
	ToolResponseTypeImage toolResponseType = "image"
)

type ToolResponse struct {
	Type     toolResponseType `json:"type"`
	Content  string           `json:"content"`
	Metadata string           `json:"metadata,omitempty"`
	IsError  bool             `json:"is_error"`
}

type ToolCall struct {
	ID    string `json:"id"`
	Name  string `json:"name"`
	Input string `json:"input"`
}

type BaseTool interface {
	Info() ToolInfo
	Run(ctx context.Context, params ToolCall) (ToolResponse, error)
}

// Helper functions for creating responses
func NewTextResponse(content string) ToolResponse {
	return ToolResponse{
		Type:    ToolResponseTypeText,
		Content: content,
	}
}

func WithResponseMetadata(response ToolResponse, metadata any) ToolResponse {
	if metadata != nil {
		metadataBytes, err := json.Marshal(metadata)
		if err != nil {
			return response
		}
		response.Metadata = string(metadataBytes)
	}
	return response
}

func NewTextErrorResponse(content string) ToolResponse {
	return ToolResponse{
		Type:    ToolResponseTypeText,
		Content: content,
		IsError: true,
	}
}
