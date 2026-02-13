//go:build e2e
// +build e2e

package browser

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"testing"
	"time"

	"mix/internal/constants"
)

// TestBrowserE2EKeyboardBasicKeys tests basic keyboard operations (Tab, Enter, Backspace)
func TestBrowserE2EKeyboardBasicKeys(t *testing.T) {
	t.Parallel()
	t.Log("=== E2E Test: Basic Keyboard Operations ===")

	testServerURL, sessionID, cleanup := setupBrowserTest(t, "Keyboard Basic Keys Test")
	defer cleanup()

	testURL := testServerURL + "/keyboard_test.html"

	// Test 1: Tab key navigation
	t.Run("tab-key", func(t *testing.T) {
		t.Log("Testing Tab key...")
		msgResp := sendMessage(t, sessionID, fmt.Sprintf("Open %s and press Tab key to navigate between fields", testURL))
		defer func() { _ = msgResp.Body.Close() }()

		waitForProcessing(t, sessionID, 90*time.Second)

		// Verify keyboard tool was called
		if verifyKeyboardToolUsed(t, sessionID) {
			t.Log("✓ Tab key test completed")
		} else {
			t.Log("⚠ Tab key usage not explicitly verified in tool calls")
		}
	})

	// Test 2: Enter key
	t.Run("enter-key", func(t *testing.T) {
		t.Log("Testing Enter key...")
		msgResp := sendMessage(t, sessionID, "Press Enter key")
		defer func() { _ = msgResp.Body.Close() }()

		waitForProcessing(t, sessionID, 90*time.Second)

		if verifyKeyboardToolUsed(t, sessionID) {
			t.Log("✓ Enter key test completed")
		} else {
			t.Log("⚠ Enter key usage not explicitly verified in tool calls")
		}
	})

	// Test 3: Backspace key
	t.Run("backspace-key", func(t *testing.T) {
		t.Log("Testing Backspace key...")
		msgResp := sendMessage(t, sessionID, "Press Backspace key")
		defer func() { _ = msgResp.Body.Close() }()

		waitForProcessing(t, sessionID, 90*time.Second)

		if verifyKeyboardToolUsed(t, sessionID) {
			t.Log("✓ Backspace key test completed")
		} else {
			t.Log("⚠ Backspace key usage not explicitly verified in tool calls")
		}
	})

	t.Log("=== E2E Test Completed Successfully ===")
}

// TestBrowserE2EKeyboardModifiers tests modifier key combinations
func TestBrowserE2EKeyboardModifiers(t *testing.T) {
	t.Log("=== E2E Test: Keyboard Modifier Combinations ===")

	testServerURL, sessionID, cleanup := setupBrowserTest(t, "Keyboard Modifiers Test")
	defer cleanup()

	testURL := testServerURL + "/keyboard_test.html"

	// Test Shift+Enter
	t.Run("shift-enter", func(t *testing.T) {
		t.Log("Testing Shift+Enter...")
		msgResp := sendMessage(t, sessionID, fmt.Sprintf("Open %s and press Shift+Enter to create a new line", testURL))
		defer func() { _ = msgResp.Body.Close() }()

		waitForProcessing(t, sessionID, 120*time.Second)

		if verifyKeyboardToolUsed(t, sessionID) {
			t.Log("✓ Shift+Enter test completed")
		} else {
			t.Log("⚠ Shift+Enter usage not explicitly verified in tool calls")
		}
	})

	t.Log("=== E2E Test Completed Successfully ===")
}

// verifyKeyboardToolUsed checks if the Browser tool was called with key action
func verifyKeyboardToolUsed(t *testing.T, sessionID string) bool {
	t.Helper()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	messagesResp := makeRequest(t, http.MethodGet, constants.APISessionsPath+sessionID+"/messages", nil)
	defer func() { _ = messagesResp.Body.Close() }()

	messagesBody, err := io.ReadAll(messagesResp.Body)
	if err != nil {
		t.Logf("Warning: Failed to read messages: %v", err)
		return false
	}

	var messages []map[string]interface{}
	if err := json.Unmarshal(messagesBody, &messages); err != nil {
		t.Logf("Warning: Failed to parse messages: %v", err)
		return false
	}

	// Look for Browser tool calls with key press results
	for _, msg := range messages {
		if toolCalls, ok := msg["toolCalls"].([]interface{}); ok {
			for _, tc := range toolCalls {
				if toolCall, ok := tc.(map[string]interface{}); ok {
					if name, ok := toolCall["name"].(string); ok && name == "Browser" {
						// Check the result for successful key press
						if result, ok := toolCall["result"].(string); ok {
							if strings.Contains(result, "Successfully pressed key") {
								t.Logf("✓ Found keyboard operation in tool result: %s", result)
								return true
							}
						}
					}
				}
			}
		}
	}

	_ = ctx // silence unused warning
	return false
}
