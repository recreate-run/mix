package tools

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	"mix/internal/logging"
	"mix/internal/notification"
)

type notifyTool struct {
	notifications notification.Service
}

type NotifyParams struct {
	Type         string   `json:"type"`
	Title        string   `json:"title"`
	Message      string   `json:"message"`
	ResponseType string   `json:"responseType"`
	Choices      []string `json:"choices,omitempty"`
	Timeout      int      `json:"timeout,omitempty"`
}

func NewNotifyTool(notifications notification.Service) BaseTool {
	return &notifyTool{
		notifications: notifications,
	}
}

func (t *notifyTool) Info() ToolInfo {
	return ToolInfo{
		Name:        "Notify",
		Description: LoadToolDescription("notify"),
		Parameters: map[string]any{
			"type": map[string]any{
				"type":        "string",
				"description": "Notification severity/type",
				"enum":        []string{"info", "warning", "error", "question"},
			},
			"title": map[string]any{
				"type":        "string",
				"description": "Short notification title (max 100 characters)",
			},
			"message": map[string]any{
				"type":        "string",
				"description": "Detailed notification message explaining what is needed from the user",
			},
			"responseType": map[string]any{
				"type":        "string",
				"description": "How the user should respond to this notification",
				"enum":        []string{"acknowledge", "text", "choice"},
			},
			"choices": map[string]any{
				"type":        "array",
				"description": "List of choices for 'choice' response type. Required if responseType is 'choice'.",
				"items": map[string]any{
					"type": "string",
				},
			},
			"timeout": map[string]any{
				"type":        "integer",
				"description": "Timeout in seconds (1-300). Default: 60 seconds. After timeout, request fails.",
				"minimum":     1,
				"maximum":     300,
			},
		},
		Required: []string{"type", "title", "message", "responseType"},
	}
}

func (t *notifyTool) Run(ctx context.Context, call ToolCall) (ToolResponse, error) {
	logging.Debug("Notify tool received input", "input", call.Input)

	var params NotifyParams
	if err := json.Unmarshal([]byte(call.Input), &params); err != nil {
		return NewTextErrorResponse(fmt.Sprintf("Invalid parameters: %v", err)), nil
	}

	// Validate required fields
	if params.Type == "" {
		return NewTextErrorResponse("Missing required field: type"), nil
	}
	if params.Title == "" {
		return NewTextErrorResponse("Missing required field: title"), nil
	}
	if params.Message == "" {
		return NewTextErrorResponse("Missing required field: message"), nil
	}
	if params.ResponseType == "" {
		return NewTextErrorResponse("Missing required field: responseType"), nil
	}

	// Validate type
	validTypes := map[string]bool{
		"info":     true,
		"warning":  true,
		"error":    true,
		"question": true,
	}
	if !validTypes[params.Type] {
		return NewTextErrorResponse(fmt.Sprintf("Invalid type '%s'. Must be one of: info, warning, error, question", params.Type)), nil
	}

	// Validate responseType
	validResponseTypes := map[string]bool{
		"acknowledge": true,
		"text":        true,
		"choice":      true,
	}
	if !validResponseTypes[params.ResponseType] {
		return NewTextErrorResponse(fmt.Sprintf("Invalid responseType '%s'. Must be one of: acknowledge, text, choice", params.ResponseType)), nil
	}

	// Validate choices for choice type
	if params.ResponseType == "choice" {
		if len(params.Choices) == 0 {
			return NewTextErrorResponse("choices array is required when responseType is 'choice'"), nil
		}
		if len(params.Choices) < 2 {
			return NewTextErrorResponse("choices array must contain at least 2 options"), nil
		}
	}

	// Validate timeout
	if params.Timeout < 0 || params.Timeout > 300 {
		return NewTextErrorResponse("timeout must be between 0 and 300 seconds"), nil
	}

	// Get session ID from context
	sessionID, ok := ctx.Value(SessionIDContextKey).(string)
	if !ok || sessionID == "" {
		return NewTextErrorResponse("Failed to get session ID from context"), nil
	}

	// Create notification request
	req := notification.CreateNotificationRequest{
		SessionID:    sessionID,
		Type:         notification.NotificationType(params.Type),
		Title:        params.Title,
		Message:      params.Message,
		ResponseType: notification.NotificationResponseType(params.ResponseType),
		Choices:      params.Choices,
		Timeout:      params.Timeout,
	}

	// BLOCKING: Wait for user response
	response, err := t.notifications.Request(req)
	if err != nil {
		if errors.Is(err, notification.ErrNotificationTimeout) {
			return NewTextErrorResponse("Notification request timed out - user did not respond in time"), nil
		}
		return NewTextErrorResponse(fmt.Sprintf("Notification request failed: %v", err)), nil
	}

	// Format response message based on response type
	var message string
	switch params.ResponseType {
	case "acknowledge":
		message = "User acknowledged the notification"
	case "text":
		if response.Value == "" {
			message = "User responded (no text provided)"
		} else {
			message = fmt.Sprintf("User response: %s", response.Value)
		}
	case "choice":
		if response.Value == "" {
			message = "User responded (no choice selected)"
		} else {
			message = fmt.Sprintf("User selected: %s", response.Value)
		}
	}

	return ToolResponse{
		Type:    "text",
		Content: message,
	}, nil
}
