package tools

import (
	"context"
	"fmt"

	"mix/internal/llm/interfaces"
	"mix/internal/session"
)

// Type aliases for shared interfaces to maintain backward compatibility
type ToolInfo = interfaces.ToolInfo
type ToolResponse = interfaces.ToolResponse
type ToolCall = interfaces.ToolCall
type BaseTool = interfaces.BaseTool

// Helper functions re-exported for convenience
var NewTextResponse = interfaces.NewTextResponse
var NewTextErrorResponse = interfaces.NewTextErrorResponse
var WithResponseMetadata = interfaces.WithResponseMetadata

// Context key aliases for backward compatibility
const (
	SessionIDContextKey       = interfaces.SessionIDContextKey
	MessageIDContextKey       = interfaces.MessageIDContextKey
	SessionStorageContextKey  = interfaces.SessionStorageContextKey
)

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
