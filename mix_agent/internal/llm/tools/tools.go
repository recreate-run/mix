package tools

import (
	"context"
	"encoding/json"
	"fmt"

	"mix/internal/session"
)

type ToolInfo struct {
	Name        string
	Description string
	Parameters  map[string]any
	Required    []string
}

type toolResponseType string

type (
	sessionIDContextKey       string
	messageIDContextKey       string
	sessionStorageContextKey  string
)

const (
	ToolResponseTypeText  toolResponseType = "text"
	ToolResponseTypeImage toolResponseType = "image"

	SessionIDContextKey       sessionIDContextKey       = "session_id"
	MessageIDContextKey       messageIDContextKey       = "message_id"
	SessionStorageContextKey  sessionStorageContextKey  = "session_storage"
)

type ToolResponse struct {
	Type     toolResponseType `json:"type"`
	Content  string           `json:"content"`
	Metadata string           `json:"metadata,omitempty"`
	IsError  bool             `json:"is_error"`
}

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

type ToolCall struct {
	ID    string `json:"id"`
	Name  string `json:"name"`
	Input string `json:"input"`
}

type BaseTool interface {
	Info() ToolInfo
	Run(ctx context.Context, params ToolCall) (ToolResponse, error)
}

func GetContextValues(ctx context.Context) (string, string) {
	sessionID := ctx.Value(SessionIDContextKey)
	messageID := ctx.Value(MessageIDContextKey)
	if sessionID == nil {
		return "", ""
	}
	if messageID == nil {
		return sessionID.(string), ""
	}
	return sessionID.(string), messageID.(string)
}

// GetSessionStorageDirectory safely extracts the session storage directory from context
func GetSessionStorageDirectory(ctx context.Context) (string, error) {
	value := ctx.Value(SessionStorageContextKey)
	if value == nil {
		return "", fmt.Errorf("session storage directory not found in context")
	}
	storageDir, ok := value.(string)
	if !ok {
		return "", fmt.Errorf("session storage directory context value is not a string")
	}
	if storageDir == "" {
		return "", fmt.Errorf("session storage directory context value is empty")
	}
	return storageDir, nil
}

// SetSessionStorageContext adds session storage directory to context for tools
func SetSessionStorageContext(ctx context.Context, sessionID string, storageConfig session.Config) context.Context {
	sessionStorageDir := session.GetSessionStoragePath(sessionID, storageConfig)
	return context.WithValue(ctx, SessionStorageContextKey, sessionStorageDir)
}
