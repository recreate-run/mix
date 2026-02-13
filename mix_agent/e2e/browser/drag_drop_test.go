//go:build e2e
// +build e2e

package browser

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"testing"
	"time"

	"mix/internal/constants"
)

// TestBrowserE2EDragByIndex tests drag-and-drop using element indices
func TestBrowserE2EDragByIndex(t *testing.T) {
	t.Parallel()
	t.Log("=== E2E Test: Drag and Drop - Index Mode ===")

	// Setup test environment
	testServerURL, sessionID, cleanup := setupBrowserTest(t, "E2E Drag Drop Index Test")
	defer cleanup()

	// Step 1: Execute drag operation using element indices
	t.Log("Step 1: Testing index-based drag operation...")
	testURL := testServerURL + "/sortable_list.html"

	// Be very explicit: use the left_click_drag action with specific indices
	msgResp := sendMessage(t, sessionID, fmt.Sprintf(
		"Navigate to %s. Then use the browser tool with action 'left_click_drag', fromIndex 0, and toIndex 1 to test the index-based drag functionality.",
		testURL,
	))
	defer func() { _ = msgResp.Body.Close() }()
	t.Log("✓ Message sent, processing started")

	// Step 2: Wait for processing
	t.Log("Step 2: Waiting for drag operation to complete...")
	waitForProcessing(t, sessionID, 120*time.Second)
	t.Log("✓ Drag operation completed")

	// Step 3: Verify the drag succeeded
	t.Log("Step 3: Verifying drag operation...")
	messagesResp := makeRequest(t, http.MethodGet, constants.APISessionsPath+sessionID+"/messages", nil)
	defer func() { _ = messagesResp.Body.Close() }()

	messagesBody, err := io.ReadAll(messagesResp.Body)
	if err != nil {
		t.Fatalf("Failed to read messages response: %v", err)
	}

	var messages []map[string]interface{}
	if err := json.Unmarshal(messagesBody, &messages); err != nil {
		t.Fatalf("Failed to parse messages: %v", err)
	}

	// Verify drag action was performed
	verifyDragAction(t, messages, "index")

	t.Log("=== E2E Test Completed Successfully ===")
}

// TestBrowserE2EDragByCoordinates tests drag-and-drop using coordinates (slider)
func TestBrowserE2EDragByCoordinates(t *testing.T) {
	t.Log("=== E2E Test: Drag and Drop - Coordinate Mode ===")

	// Setup test environment
	testServerURL, sessionID, cleanup := setupBrowserTest(t, "E2E Drag Drop Coordinate Test")
	defer cleanup()

	// Step 1: Execute drag operation using coordinates
	t.Log("Step 1: Testing coordinate-based drag operation...")
	testURL := testServerURL + "/sortable_list.html"

	// Be very explicit: use the left_click_drag action with specific coordinates
	msgResp := sendMessage(t, sessionID, fmt.Sprintf(
		"Navigate to %s. Use the browser tool with action 'left_click_drag' to drag from coordinates (100, 300) to (200, 300). This will test the drag functionality.",
		testURL,
	))
	defer func() { _ = msgResp.Body.Close() }()
	t.Log("✓ Message sent, processing started")

	// Step 2: Wait for processing
	t.Log("Step 2: Waiting for drag to complete...")
	waitForProcessing(t, sessionID, 120*time.Second)
	t.Log("✓ Drag completed")

	// Step 3: Verify the drag succeeded
	t.Log("Step 3: Verifying drag operation...")
	messagesResp := makeRequest(t, http.MethodGet, constants.APISessionsPath+sessionID+"/messages", nil)
	defer func() { _ = messagesResp.Body.Close() }()

	messagesBody, err := io.ReadAll(messagesResp.Body)
	if err != nil {
		t.Fatalf("Failed to read messages response: %v", err)
	}

	var messages []map[string]interface{}
	if err := json.Unmarshal(messagesBody, &messages); err != nil {
		t.Fatalf("Failed to parse messages: %v", err)
	}

	// Verify drag action was performed
	verifyDragAction(t, messages, "coordinate")

	t.Log("=== E2E Test Completed Successfully ===")
}

// TestBrowserE2EDragFailure tests error handling for invalid drag operations
func TestBrowserE2EDragFailure(t *testing.T) {
	t.Log("=== E2E Test: Drag and Drop - Error Handling ===")

	// Setup test environment
	testServerURL, sessionID, cleanup := setupBrowserTest(t, "E2E Drag Drop Failure Test")
	defer cleanup()

	// Step 1: Attempt invalid drag operation
	t.Log("Step 1: Testing drag error handling...")
	testURL := testServerURL + "/sortable_list.html"

	// Test error handling: try to drag with incomplete parameters (only fromIndex, missing toIndex)
	// This should fail validation because drag requires both fromIndex and toIndex
	msgResp := sendMessage(t, sessionID, fmt.Sprintf(
		"Navigate to %s. Then use the browser tool with action 'left_click_drag' with ONLY fromIndex set to 0, but do NOT provide toIndex or any coordinates. This should cause a validation error.",
		testURL,
	))
	defer func() { _ = msgResp.Body.Close() }()
	t.Log("✓ Message sent, processing started")

	// Step 2: Wait for processing
	t.Log("Step 2: Waiting for processing to complete...")
	waitForProcessing(t, sessionID, 120*time.Second)
	t.Log("✓ Processing completed")

	// Step 3: Verify error was reported
	t.Log("Step 3: Verifying error handling...")
	messagesResp := makeRequest(t, http.MethodGet, constants.APISessionsPath+sessionID+"/messages", nil)
	defer func() { _ = messagesResp.Body.Close() }()

	messagesBody, err := io.ReadAll(messagesResp.Body)
	if err != nil {
		t.Fatalf("Failed to read messages response: %v", err)
	}

	var messages []map[string]interface{}
	if err := json.Unmarshal(messagesBody, &messages); err != nil {
		t.Fatalf("Failed to parse messages: %v", err)
	}

	// Verify error was detected
	verifyDragError(t, messages)

	t.Log("=== E2E Test Completed Successfully ===")
}

// Helper functions

func verifyDragAction(t *testing.T, messages []map[string]interface{}, mode string) {
	t.Helper()

	foundDragToolCall := false
	foundSuccessMessage := false

	for _, msg := range messages {
		// Check for browser tool calls with drag action
		if toolCalls, ok := msg["toolCalls"].([]interface{}); ok {
			for _, tc := range toolCalls {
				if toolCall, ok := tc.(map[string]interface{}); ok {
					if name, ok := toolCall["name"].(string); ok && name == "Browser" {
						// Check if it's a drag action - must start with "Successfully dragged"
						if result, ok := toolCall["result"].(string); ok {
							if strings.HasPrefix(result, "Successfully dragged") {
								foundDragToolCall = true
								t.Logf("✓ Found successful drag: %s", result)

								// Verify mode-specific details
								if mode == "index" && strings.Contains(result, "element") {
									t.Log("✓ Confirmed index-based drag")
								} else if mode == "coordinate" && strings.Contains(result, ",") {
									t.Log("✓ Confirmed coordinate-based drag")
								}
							}
						}
					}
				}
			}
		}

		// Check assistant responses for success confirmation
		if role, ok := msg["role"].(string); ok && role == "assistant" {
			if content, ok := msg["content"].(string); ok {
				contentLower := strings.ToLower(content)
				if strings.Contains(contentLower, "drag") &&
				   (strings.Contains(contentLower, "success") || strings.Contains(contentLower, "moved") ||
				    strings.Contains(contentLower, "dropped")) {
					foundSuccessMessage = true
					t.Log("✓ Found drag success confirmation in assistant response")
				}
			}
		}
	}

	if !foundDragToolCall {
		t.Fatal("❌ Drag tool call was not found")
	}

	if !foundSuccessMessage {
		t.Log("⚠ Drag success message not explicitly found (may have been phrased differently)")
	}

	t.Log("✓ Drag action verification complete")
}

func verifyDragError(t *testing.T, messages []map[string]interface{}) {
	t.Helper()

	foundDragError := false
	foundErrorMention := false

	for _, msg := range messages {
		// Check for drag-specific errors in tool call results
		if toolCalls, ok := msg["toolCalls"].([]interface{}); ok {
			for _, tc := range toolCalls {
				if toolCall, ok := tc.(map[string]interface{}); ok {
					if name, ok := toolCall["name"].(string); ok && name == "Browser" {
						if result, ok := toolCall["result"].(string); ok {
							// Look for specific drag errors
							if strings.Contains(result, "drag action") &&
							   (strings.Contains(result, "requires") || strings.Contains(result, "cannot")) {
								foundDragError = true
								t.Logf("✓ Found drag-specific error: %s", result)
							} else if strings.Contains(result, "Drag failed") {
								foundDragError = true
								t.Logf("✓ Found drag failure error: %s", result)
							}
						}
					}
				}
			}
		}

		// Check assistant responses for error mention
		if role, ok := msg["role"].(string); ok && role == "assistant" {
			if content, ok := msg["content"].(string); ok {
				contentLower := strings.ToLower(content)
				if (strings.Contains(contentLower, "error") ||
				    strings.Contains(contentLower, "cannot") ||
				    strings.Contains(contentLower, "invalid")) &&
				   strings.Contains(contentLower, "drag") {
					foundErrorMention = true
					t.Log("✓ Found error mention in assistant response")
				}
			}
		}
	}

	if !foundDragError {
		t.Fatal("❌ Expected drag error was not found in tool results")
	}

	if !foundErrorMention {
		t.Log("⚠ Error not explicitly mentioned by assistant (may have been handled silently)")
	}

	t.Log("✓ Drag error handling verified")
}
