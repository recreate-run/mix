package tools

import (
	"context"
	"encoding/json"
	"fmt"
)

type ToolInfo struct {
	Name        string
	Description string
	Parameters  map[string]any
	Required    []string
}

type toolResponseType string

type (
	sessionIDContextKey        string
	messageIDContextKey        string
	workingDirectoryContextKey string
)

const (
	ToolResponseTypeText  toolResponseType = "text"
	ToolResponseTypeImage toolResponseType = "image"

	SessionIDContextKey        sessionIDContextKey        = "session_id"
	MessageIDContextKey        messageIDContextKey        = "message_id"
	WorkingDirectoryContextKey workingDirectoryContextKey = "working_directory"
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

// GetWorkingDirectory safely extracts the working directory from context
func GetWorkingDirectory(ctx context.Context) (string, error) {
	value := ctx.Value(WorkingDirectoryContextKey)
	if value == nil {
		return "", fmt.Errorf("working directory not found in context")
	}
	workingDir, ok := value.(string)
	if !ok {
		return "", fmt.Errorf("working directory context value is not a string")
	}
	if workingDir == "" {
		return "", fmt.Errorf("working directory context value is empty")
	}
	return workingDir, nil
}
