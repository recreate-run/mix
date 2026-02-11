package browser

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"mix/internal/llm/interfaces"
	"mix/internal/session"
)


// TestNewActionsE2E tests the three new browser actions in an end-to-end scenario
func TestNewActionsE2E(t *testing.T) {
	skipIfIntegrationTestsDisabled(t)

	// Start mock browser server
	mockServer := startMockBrowserServer(t)
	defer mockServer.close()

	mockPermissionService := &MockPermissionService{}
	sessionConfig := session.DefaultConfig()
	tool := NewBrowserTool(mockPermissionService, &MockSessionService{}, mockServer.wsURL, sessionConfig, "", mockClientFactory, nil, nil)

	sessionID := "e2e-test-session"
	ctx := createBrowserTestContext(sessionID, "message-1", t.TempDir())

	t.Run("key_action", func(t *testing.T) {
		// First navigate to a page
		navigateCall := interfaces.ToolCall{
			ID:    "nav-1",
			Name:  BrowserToolName,
			Input: `{"action": "open", "url": "https://www.google.com"}`,
		}
		response, err := tool.Run(ctx, navigateCall)
		require.NoError(t, err)
		assert.False(t, response.IsError, "Navigate should succeed")

		// Take screenshot to populate element cache
		screenshotCall := interfaces.ToolCall{
			ID:    "screenshot-1",
			Name:  BrowserToolName,
			Input: `{"action": "screenshot"}`,
		}
		response, err = tool.Run(ctx, screenshotCall)
		require.NoError(t, err)
		assert.False(t, response.IsError, "Screenshot should succeed")

		// Click on an element (e.g., search input at index 0)
		clickCall := interfaces.ToolCall{
			ID:    "click-1",
			Name:  BrowserToolName,
			Input: `{"action": "click", "index": 0}`,
		}
		response, err = tool.Run(ctx, clickCall)
		require.NoError(t, err)
		assert.False(t, response.IsError, "Click should succeed")

		// Test key action: Press Enter
		keyCall := interfaces.ToolCall{
			ID:    "key-1",
			Name:  BrowserToolName,
			Input: `{"action": "key", "key": "Enter"}`,
		}
		response, err = tool.Run(ctx, keyCall)
		require.NoError(t, err)
		assert.False(t, response.IsError, "Key action should succeed")
		assert.Contains(t, response.Content, "pressed key")

		// Verify the mock server received the key command
		assert.Positive(t, mockServer.getRequestCount())
	})

	t.Run("scroll_to_action", func(t *testing.T) {
		// Take screenshot to populate element cache with in-viewport and out-of-viewport elements
		screenshotCall := interfaces.ToolCall{
			ID:    "screenshot-2",
			Name:  BrowserToolName,
			Input: `{"action": "screenshot"}`,
		}
		response, err := tool.Run(ctx, screenshotCall)
		require.NoError(t, err)
		assert.False(t, response.IsError, "Screenshot should succeed")

		// Test scroll_to action: Scroll to element at index 5 (in viewport in mock)
		scrollToCall := interfaces.ToolCall{
			ID:    "scroll-to-1",
			Name:  BrowserToolName,
			Input: `{"action": "scroll_to", "index": 5}`,
		}
		response, err = tool.Run(ctx, scrollToCall)
		require.NoError(t, err)
		assert.False(t, response.IsError, "Scroll-to action should succeed")
		assert.Contains(t, response.Content, "scrolled")

		// Verify the mock server received the scroll command
		assert.Positive(t, mockServer.getRequestCount())
	})

	t.Run("action_batching", func(t *testing.T) {
		// Test action sequence: Click, type (with index), and press Enter
		actionSequenceCall := interfaces.ToolCall{
			ID:    "sequence-1",
			Name:  BrowserToolName,
			Input: `{
				"action": "action",
				"actions": [
					{"type": "left_click", "index": 0},
					{"type": "type", "index": 0, "text": "test search"},
					{"type": "key", "key": "Enter"}
				]
			}`,
		}
		response, err := tool.Run(ctx, actionSequenceCall)
		require.NoError(t, err)
		assert.False(t, response.IsError, "Action sequence should succeed")
		assert.Contains(t, response.Content, "3/3 successful")

		// Verify the mock server received all commands
		assert.Positive(t, mockServer.getRequestCount())
	})

	t.Run("key_combinations", func(t *testing.T) {
		// Test key combination: cmd+a (select all)
		keyCombinationCall := interfaces.ToolCall{
			ID:    "key-combo-1",
			Name:  BrowserToolName,
			Input: `{"action": "key", "key": "cmd+a"}`,
		}
		response, err := tool.Run(ctx, keyCombinationCall)
		require.NoError(t, err)
		assert.False(t, response.IsError, "Key combination should succeed")
		assert.Contains(t, response.Content, "pressed key")

		// Test multiple keys in sequence (double backspace) //nolint:dupword // Intentional duplicate for testing multiple key presses
		multipleKeysCall := interfaces.ToolCall{
			ID:    "key-multi-1",
			Name:  BrowserToolName,
			Input: `{"action": "key", "key": "Backspace Backspace"}`,
		}
		response, err = tool.Run(ctx, multipleKeysCall)
		require.NoError(t, err)
		assert.False(t, response.IsError, "Multiple keys should succeed")
		assert.Contains(t, response.Content, "pressed key")
	})
}

// TestActionBatchingValidation tests validation of action batching
func TestActionBatchingValidation(t *testing.T) {
	skipIfIntegrationTestsDisabled(t)

	mockServer := startMockBrowserServer(t)
	defer mockServer.close()

	mockPermissionService := &MockPermissionService{}
	sessionConfig := session.DefaultConfig()
	tool := NewBrowserTool(mockPermissionService, &MockSessionService{}, mockServer.wsURL, sessionConfig, "", mockClientFactory, nil, nil)

	ctx := createBrowserTestContext("session-123", "message-456", t.TempDir())

	// Navigate to populate session
	navigateCall := interfaces.ToolCall{
		ID:    "nav-1",
		Name:  BrowserToolName,
		Input: `{"action": "open", "url": "https://example.com"}`,
	}
	_, _ = tool.Run(ctx, navigateCall)

	t.Run("empty_actions_array", func(t *testing.T) {
		call := interfaces.ToolCall{
			ID:    "call-1",
			Name:  BrowserToolName,
			Input: `{"action": "action", "actions": []}`,
		}

		response, err := tool.Run(ctx, call)
		require.NoError(t, err)
		assert.True(t, response.IsError)
		assert.Contains(t, response.Content, "missing actions")
	})

	t.Run("missing_type_field", func(t *testing.T) {
		call := interfaces.ToolCall{
			ID:    "call-2",
			Name:  BrowserToolName,
			Input: `{"action": "action", "actions": [{"index": 0}]}`,
		}

		response, err := tool.Run(ctx, call)
		require.NoError(t, err)
		// The sequence returns a result with 0 successful actions
		assert.Contains(t, response.Content, "0/1 successful")
	})

	t.Run("invalid_action_type", func(t *testing.T) {
		call := interfaces.ToolCall{
			ID:    "call-3",
			Name:  BrowserToolName,
			Input: `{"action": "action", "actions": [{"type": "invalid_action"}]}`,
		}

		response, err := tool.Run(ctx, call)
		require.NoError(t, err)
		// The sequence returns a result showing the failure
		assert.Contains(t, response.Content, "unknown sub-action type")
	})
}

// TestKeyActionIntegration tests key action with mock browser
func TestKeyActionIntegration(t *testing.T) {
	skipIfIntegrationTestsDisabled(t)

	mockServer := startMockBrowserServer(t)
	defer mockServer.close()

	mockPermissionService := &MockPermissionService{}
	sessionConfig := session.DefaultConfig()
	tool := NewBrowserTool(mockPermissionService, &MockSessionService{}, mockServer.wsURL, sessionConfig, "", mockClientFactory, nil, nil)

	sessionID := "key-test-session"
	ctx := createBrowserTestContext(sessionID, "message-1", t.TempDir())

	// Navigate first
	navigateCall := interfaces.ToolCall{
		ID:    "nav-1",
		Name:  BrowserToolName,
		Input: `{"action": "open", "url": "https://example.com"}`,
	}
	response, err := tool.Run(ctx, navigateCall)
	require.NoError(t, err)
	assert.False(t, response.IsError)

	tests := []struct {
		name     string
		keyInput string
		wantErr  bool
	}{
		{
			name:     "single_key",
			keyInput: "Enter",
			wantErr:  false,
		},
		{
			name:     "key_combination",
			keyInput: "cmd+c",
			wantErr:  false,
		},
		{
			name:     "multiple_keys_sequence",
			keyInput: "Tab Tab Enter", //nolint:dupword // Intentional duplicate for testing multiple Tab presses
			wantErr:  false,
		},
		{
			name:     "special_keys",
			keyInput: "Escape",
			wantErr:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			input := `{"action": "key", "key": "` + tt.keyInput + `"}`
			call := interfaces.ToolCall{
				ID:    "key-test",
				Name:  BrowserToolName,
				Input: input,
			}

			response, err := tool.Run(ctx, call)
			require.NoError(t, err)

			if tt.wantErr {
				assert.True(t, response.IsError)
			} else {
				assert.False(t, response.IsError)
				assert.Contains(t, response.Content, "pressed key")
			}
		})
	}
}

// TestScrollToActionIntegration tests scroll_to action with mock browser
func TestScrollToActionIntegration(t *testing.T) {
	skipIfIntegrationTestsDisabled(t)

	mockServer := startMockBrowserServer(t)
	defer mockServer.close()

	mockPermissionService := &MockPermissionService{}
	sessionConfig := session.DefaultConfig()
	tool := NewBrowserTool(mockPermissionService, &MockSessionService{}, mockServer.wsURL, sessionConfig, "", mockClientFactory, nil, nil)

	sessionID := "scroll-test-session"
	ctx := createBrowserTestContext(sessionID, "message-1", t.TempDir())

	// Navigate first
	navigateCall := interfaces.ToolCall{
		ID:    "nav-1",
		Name:  BrowserToolName,
		Input: `{"action": "open", "url": "https://example.com"}`,
	}
	response, err := tool.Run(ctx, navigateCall)
	require.NoError(t, err)
	assert.False(t, response.IsError)

	// Take screenshot to populate element cache
	screenshotCall := interfaces.ToolCall{
		ID:    "screenshot-1",
		Name:  BrowserToolName,
		Input: `{"action": "screenshot"}`,
	}
	response, err = tool.Run(ctx, screenshotCall)
	require.NoError(t, err)
	assert.False(t, response.IsError)

	t.Run("scroll_to_visible_element", func(t *testing.T) {
		// Scroll to element at index 5 (in viewport in mock)
		call := interfaces.ToolCall{
			ID:    "scroll-to-1",
			Name:  BrowserToolName,
			Input: `{"action": "scroll_to", "index": 5}`,
		}

		response, err := tool.Run(ctx, call)
		require.NoError(t, err)
		assert.False(t, response.IsError)
		assert.Contains(t, response.Content, "scrolled")
	})

	t.Run("scroll_to_another_element", func(t *testing.T) {
		// Scroll to element at index 10
		call := interfaces.ToolCall{
			ID:    "scroll-to-2",
			Name:  BrowserToolName,
			Input: `{"action": "scroll_to", "index": 10}`,
		}

		response, err := tool.Run(ctx, call)
		require.NoError(t, err)
		assert.False(t, response.IsError)
		assert.Contains(t, response.Content, "scrolled")
	})

	t.Run("scroll_to_invalid_index", func(t *testing.T) {
		// Try to scroll to non-existent element
		call := interfaces.ToolCall{
			ID:    "scroll-to-3",
			Name:  BrowserToolName,
			Input: `{"action": "scroll_to", "index": 9999}`,
		}

		response, err := tool.Run(ctx, call)
		require.NoError(t, err)
		assert.True(t, response.IsError)
		assert.Contains(t, response.Content, "not found")
	})
}
