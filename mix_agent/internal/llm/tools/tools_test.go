package tools

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"mix/internal/session"
)

// Test GetContextValues function
func TestGetContextValues(t *testing.T) {
	tests := []struct {
		name              string
		setupContext      func() context.Context
		expectedSessionID string
		expectedMessageID string
	}{
		{
			name:              "empty context",
			setupContext:      context.Background,
			expectedSessionID: "",
			expectedMessageID: "",
		},
		{
			name: "context with session ID only",
			setupContext: func() context.Context {
				ctx := context.Background()
				return context.WithValue(ctx, SessionIDContextKey, "session-123")
			},
			expectedSessionID: "session-123",
			expectedMessageID: "",
		},
		{
			name: "context with both session ID and message ID",
			setupContext: func() context.Context {
				ctx := context.Background()
				ctx = context.WithValue(ctx, SessionIDContextKey, "session-456")
				ctx = context.WithValue(ctx, MessageIDContextKey, "message-789")
				return ctx
			},
			expectedSessionID: "session-456",
			expectedMessageID: "message-789",
		},
		{
			name: "context with message ID but no session ID",
			setupContext: func() context.Context {
				ctx := context.Background()
				return context.WithValue(ctx, MessageIDContextKey, "message-only")
			},
			expectedSessionID: "",
			expectedMessageID: "",
		},
		// Note: Non-string session ID would cause a panic in the actual implementation
		// This is expected behavior as the function assumes string values
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := tt.setupContext()
			sessionID, messageID := GetContextValues(ctx)

			assert.Equal(t, tt.expectedSessionID, sessionID)
			assert.Equal(t, tt.expectedMessageID, messageID)
		})
	}
}

// Test GetSessionStorageDirectory function
func TestGetSessionStorageDirectory(t *testing.T) {
	tests := []struct {
		name         string
		setupContext func() context.Context
		expectError  bool
		errorSubstr  string
		expectedDir  string
	}{
		{
			name: "valid storage directory",
			setupContext: func() context.Context {
				ctx := context.Background()
				return context.WithValue(ctx, SessionStorageContextKey, "/storage/session-123")
			},
			expectError: false,
			expectedDir: "/storage/session-123",
		},
		{
			name:         "missing storage directory in context",
			setupContext: context.Background,
			expectError:  true,
			errorSubstr:  "session storage directory not found in context",
		},
		{
			name: "non-string storage directory",
			setupContext: func() context.Context {
				ctx := context.Background()
				return context.WithValue(ctx, SessionStorageContextKey, 12345)
			},
			expectError: true,
			errorSubstr: "session storage directory context value is not a string",
		},
		{
			name: "empty storage directory",
			setupContext: func() context.Context {
				ctx := context.Background()
				return context.WithValue(ctx, SessionStorageContextKey, "")
			},
			expectError: true,
			errorSubstr: "session storage directory context value is empty",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := tt.setupContext()
			dir, err := GetSessionStorageDirectory(ctx)

			if tt.expectError {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.errorSubstr)
				assert.Empty(t, dir)
			} else {
				require.NoError(t, err)
				assert.Equal(t, tt.expectedDir, dir)
			}
		})
	}
}

// Test SetSessionStorageContext function
func TestSetSessionStorageContext(t *testing.T) {
	tests := []struct {
		name        string
		sessionID   string
		config      session.Config
		expectedDir string
	}{
		{
			name:      "default config with session ID",
			sessionID: "session-123",
			config:    session.DefaultConfig(),
			// expectedDir will be {config.BasePath}/session-123 - calculated dynamically
		},
		{
			name:        "custom config with session ID",
			sessionID:   "test-session-456",
			config:      session.Config{BasePath: "/custom/storage"},
			expectedDir: "/custom/storage/test-session-456",
		},
		{
			name:        "empty session ID",
			sessionID:   "",
			config:      session.Config{BasePath: "/test/storage"},
			expectedDir: "/test/storage", // filepath.Join doesn't add trailing slash for empty sessionID
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()
			newCtx := SetSessionStorageContext(ctx, tt.sessionID, tt.config)

			// Extract the storage directory from the new context
			dir, err := GetSessionStorageDirectory(newCtx)
			require.NoError(t, err)

			if tt.expectedDir != "" {
				assert.Equal(t, tt.expectedDir, dir)
			} else {
				// For default config, just verify it contains the session ID
				assert.Contains(t, dir, tt.sessionID)
				assert.Contains(t, dir, "storage") // Should contain default storage path
			}

			// Verify the original context is unchanged
			_, err = GetSessionStorageDirectory(ctx)
			require.Error(t, err) // Original context should not have storage directory
		})
	}
}

// Test type aliases
func TestTypeAliases(t *testing.T) {
	// Test that type aliases are properly defined
	var toolInfo ToolInfo
	var toolResponse ToolResponse
	var toolCall ToolCall

	// Should be able to assign values of the aliased types
	toolInfo = ToolInfo{
		Name:        "test_tool",
		Description: "A test tool",
		Parameters:  map[string]any{"param": "value"},
		Required:    []string{"param"},
	}

	toolResponse = ToolResponse{
		Type:    "text",
		Content: "test content",
		IsError: false,
	}

	toolCall = ToolCall{
		ID:    "call-123",
		Name:  "test_tool",
		Input: `{"param":"value"}`,
	}

	// Verify the values are set correctly
	assert.Equal(t, "test_tool", toolInfo.Name)
	assert.Equal(t, "test content", toolResponse.Content)
	assert.Equal(t, "call-123", toolCall.ID)
}

// Test helper function re-exports
func TestHelperFunctionReExports(t *testing.T) {
	// Test NewTextResponse
	response := NewTextResponse("test content")
	assert.Equal(t, "text", string(response.Type))
	assert.Equal(t, "test content", response.Content)
	assert.False(t, response.IsError)

	// Test NewTextErrorResponse
	errorResponse := NewTextErrorResponse("error message")
	assert.Equal(t, "text", string(errorResponse.Type))
	assert.Equal(t, "error message", errorResponse.Content)
	assert.True(t, errorResponse.IsError)

	// Test WithResponseMetadata
	metadata := map[string]string{"key": "value"}
	responseWithMeta := WithResponseMetadata(response, metadata)
	assert.Equal(t, "test content", responseWithMeta.Content)
	assert.NotEmpty(t, responseWithMeta.Metadata)
	assert.Contains(t, responseWithMeta.Metadata, "key")
	assert.Contains(t, responseWithMeta.Metadata, "value")
}

// Test context key constants
func TestContextKeyConstants(t *testing.T) {
	// Test that constants are defined
	assert.NotNil(t, SessionIDContextKey)
	assert.NotNil(t, MessageIDContextKey)
	assert.NotNil(t, SessionStorageContextKey)

	// Test that constants can be used as context keys
	ctx := context.Background()
	ctx = context.WithValue(ctx, SessionIDContextKey, "session-test")
	ctx = context.WithValue(ctx, MessageIDContextKey, "message-test")
	ctx = context.WithValue(ctx, SessionStorageContextKey, "/storage/test")

	sessionID := ctx.Value(SessionIDContextKey)
	messageID := ctx.Value(MessageIDContextKey)
	storageDir := ctx.Value(SessionStorageContextKey)

	assert.Equal(t, "session-test", sessionID)
	assert.Equal(t, "message-test", messageID)
	assert.Equal(t, "/storage/test", storageDir)
}

// Test integration between context functions
func TestContextFunctionIntegration(t *testing.T) {
	sessionID := "integration-test-session"
	messageID := "integration-test-message"
	config := session.Config{BasePath: "/test/integration/storage"}

	// Start with empty context
	ctx := context.Background()

	// Add session ID and message ID manually
	ctx = context.WithValue(ctx, SessionIDContextKey, sessionID)
	ctx = context.WithValue(ctx, MessageIDContextKey, messageID)

	// Test GetContextValues
	retrievedSessionID, retrievedMessageID := GetContextValues(ctx)
	assert.Equal(t, sessionID, retrievedSessionID)
	assert.Equal(t, messageID, retrievedMessageID)

	// Add session storage using SetSessionStorageContext
	ctx = SetSessionStorageContext(ctx, sessionID, config)

	// Test GetSessionStorageDirectory
	storageDir, err := GetSessionStorageDirectory(ctx)
	require.NoError(t, err)
	assert.Equal(t, "/test/integration/storage/integration-test-session", storageDir)

	// Verify original context values are preserved
	finalSessionID, finalMessageID := GetContextValues(ctx)
	assert.Equal(t, sessionID, finalSessionID)
	assert.Equal(t, messageID, finalMessageID)
}

// Test edge cases and error conditions
func TestEdgeCases(t *testing.T) {
	t.Run("context value type assertion edge cases", func(t *testing.T) {
		ctx := context.Background()

		// Test with nil session ID
		ctx = context.WithValue(ctx, SessionIDContextKey, nil)
		sessionID, messageID := GetContextValues(ctx)
		assert.Empty(t, sessionID)
		assert.Empty(t, messageID)

		// Test panic behavior with non-string session ID
		ctx = context.WithValue(ctx, SessionIDContextKey, 12345)
		assert.Panics(t, func() {
			GetContextValues(ctx)
		}, "Should panic with non-string session ID")

		// Test panic behavior with non-string message ID (when session ID is present)
		ctx = context.Background()
		ctx = context.WithValue(ctx, SessionIDContextKey, "valid-session")
		ctx = context.WithValue(ctx, MessageIDContextKey, 67890)
		assert.Panics(t, func() {
			GetContextValues(ctx)
		}, "Should panic with non-string message ID")
	})

	t.Run("session storage with special characters", func(t *testing.T) {
		sessionID := "session-with-unicode-文件"
		config := session.Config{BasePath: "/storage/with spaces/unicode"}

		ctx := SetSessionStorageContext(context.Background(), sessionID, config)
		storageDir, err := GetSessionStorageDirectory(ctx)

		require.NoError(t, err)
		assert.Contains(t, storageDir, sessionID)
		assert.Contains(t, storageDir, "with spaces")
	})
}
