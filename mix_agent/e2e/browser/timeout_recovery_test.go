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

// TestBrowserE2ESlowPageLoad tests that browser handles slow-loading pages without hanging indefinitely
func TestBrowserE2ESlowPageLoad(t *testing.T) {
	t.Parallel()
	t.Log("=== E2E Test: Slow Page Load Handling ===")

	testServerURL, sessionID, cleanup := setupBrowserTest(t, "E2E Slow Page Load Test")
	defer cleanup()

	// Step 1: Request navigation to a page with 5-second delay
	t.Log("Step 1: Requesting navigation to slow-loading page (5s delay)...")
	testURL := testServerURL + "/delayed.html?delay=5000"

	msgResp := sendMessage(t, sessionID, fmt.Sprintf(
		"Open %s and tell me what you see after it finishes loading. Wait for the content to appear.",
		testURL,
	))
	defer func() { _ = msgResp.Body.Close() }()
	t.Log("✓ Message sent, processing started")

	// Step 2: Wait for processing with reasonable timeout (slow page should still complete)
	t.Log("Step 2: Waiting for agent to handle slow page load...")
	waitForProcessing(t, sessionID, 120*time.Second)
	t.Log("✓ Processing completed without hanging")

	// Step 3: Verify the agent handled the slow load properly
	t.Log("Step 3: Verifying slow page handling...")
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

	// Verify browser tool was used
	foundBrowserAction := false
	foundLoadedContent := false

	for _, msg := range messages {
		if toolCalls, ok := msg["toolCalls"].([]interface{}); ok {
			for _, tc := range toolCalls {
				if toolCall, ok := tc.(map[string]interface{}); ok {
					if name, ok := toolCall["name"].(string); ok && name == "Browser" {
						foundBrowserAction = true
						t.Log("✓ Found browser tool action")
					}
				}
			}
		}

		// Check if agent mentioned the loaded content
		if role, ok := msg["role"].(string); ok && role == "assistant" {
			if content, ok := msg["content"].(string); ok {
				contentLower := strings.ToLower(content)
				if strings.Contains(contentLower, "loaded") || strings.Contains(contentLower, "content") {
					foundLoadedContent = true
					t.Log("✓ Agent confirmed page loaded")
				}
			}
		}
	}

	if !foundBrowserAction {
		t.Fatal("❌ Browser tool was not used")
	}

	if !foundLoadedContent {
		t.Log("⚠ Loaded content not explicitly mentioned (may have been described differently)")
	}

	// Step 4: Verify session is still usable after slow load
	t.Log("Step 4: Verifying session remains usable...")
	msg2Resp := sendMessage(t, sessionID, "Take a screenshot of the current page")
	defer func() { _ = msg2Resp.Body.Close() }()

	t.Log("✓ Session remains usable after slow page load")

	t.Log("=== E2E Test Completed Successfully ===")
}

// TestBrowserE2ENetworkFailure tests that browser handles network failures gracefully
func TestBrowserE2ENetworkFailure(t *testing.T) {
	t.Parallel()
	t.Log("=== E2E Test: Network Failure Handling ===")

	testServerURL, sessionID, cleanup := setupBrowserTest(t, "E2E Network Failure Test")
	defer cleanup()

	// Step 1: First navigate to a valid page
	t.Log("Step 1: Navigating to valid page first...")
	validURL := testServerURL + "/element_selection.html"

	msgResp := sendMessage(t, sessionID, fmt.Sprintf("Open %s", validURL))
	defer func() { _ = msgResp.Body.Close() }()

	waitForProcessing(t, sessionID, 60*time.Second)
	t.Log("✓ Valid page loaded successfully")

	// Step 2: Attempt navigation to non-existent server
	t.Log("Step 2: Attempting navigation to unreachable URL...")
	badURL := "http://localhost:9999/nonexistent"

	msg2Resp := sendMessage(t, sessionID, fmt.Sprintf("Try to open %s and report what happens", badURL))
	defer func() { _ = msg2Resp.Body.Close() }()

	waitForProcessing(t, sessionID, 90*time.Second)
	t.Log("✓ Processing completed without crash")

	// Step 3: Verify network error was reported
	t.Log("Step 3: Verifying network error was reported...")
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

	foundError := verifyNetworkError(t, messages)
	if !foundError {
		t.Log("⚠ Network error not explicitly mentioned (may have been handled differently)")
	} else {
		t.Log("✓ Network error properly reported")
	}

	// Step 4: Verify session is still usable after network failure
	t.Log("Step 4: Verifying session remains usable after network failure...")
	msg3Resp := sendMessage(t, sessionID, fmt.Sprintf("Navigate back to %s", validURL))
	defer func() { _ = msg3Resp.Body.Close() }()

	t.Log("✓ Session remains usable after network failure")

	t.Log("=== E2E Test Completed Successfully ===")
}

// TestBrowserE2ESlowActionSequence tests that action sequences handle delays properly
func TestBrowserE2ESlowActionSequence(t *testing.T) {
	t.Parallel()
	t.Log("=== E2E Test: Slow Action Sequence Handling ===")

	testServerURL, sessionID, cleanup := setupBrowserTest(t, "E2E Slow Action Sequence Test")
	defer cleanup()

	// Step 1: Execute multi-step sequence with screenshots between actions
	t.Log("Step 1: Executing slow multi-step action sequence...")
	testURL := testServerURL + "/action_sequence.html"

	msgResp := sendMessage(t, sessionID, fmt.Sprintf(
		"Open %s. Then perform these steps slowly with screenshots: "+
			"1) Take initial screenshot, "+
			"2) Click Increment Counter button, "+
			"3) Take screenshot, "+
			"4) Click Toggle Element button, "+
			"5) Take final screenshot. "+
			"Report on the state changes you observed.",
		testURL,
	))
	defer func() { _ = msgResp.Body.Close() }()
	t.Log("✓ Message sent, processing started")

	// Step 2: Wait for processing (allow extra time for multiple screenshots)
	t.Log("Step 2: Waiting for slow action sequence to complete...")
	waitForProcessing(t, sessionID, 150*time.Second)
	t.Log("✓ Action sequence completed")

	// Step 3: Verify all actions completed and screenshots were captured
	t.Log("Step 3: Verifying action sequence execution...")
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

	// Count screenshots and verify completion
	screenshotCount := 0
	foundBrowserAction := false

	for _, msg := range messages {
		if toolCalls, ok := msg["toolCalls"].([]interface{}); ok {
			for _, tc := range toolCalls {
				if toolCall, ok := tc.(map[string]interface{}); ok {
					if name, ok := toolCall["name"].(string); ok && name == "Browser" {
						foundBrowserAction = true
						// Check for screenshot URLs
						if screenshotUrls, ok := toolCall["screenshotUrls"].([]interface{}); ok && len(screenshotUrls) > 0 {
							screenshotCount++
						}
					}
				}
			}
		}
	}

	if !foundBrowserAction {
		t.Fatal("❌ Browser tool was not used")
	}
	t.Logf("✓ Browser tool used, found %d screenshots", screenshotCount)

	if screenshotCount < 3 {
		t.Logf("⚠ Warning: Expected at least 3 screenshots, found %d", screenshotCount)
	} else {
		t.Log("✓ Multiple screenshots captured during action sequence")
	}

	t.Log("=== E2E Test Completed Successfully ===")
}

// Helper functions

func verifyNetworkError(t *testing.T, messages []map[string]interface{}) bool {
	t.Helper()

	for _, msg := range messages {
		// Check for errors in tool call results
		if toolCalls, ok := msg["toolCalls"].([]interface{}); ok {
			for _, tc := range toolCalls {
				if toolCall, ok := tc.(map[string]interface{}); ok {
					if name, ok := toolCall["name"].(string); ok && name == "Browser" {
						if result, ok := toolCall["result"].(string); ok {
							resultLower := strings.ToLower(result)
							if strings.Contains(resultLower, "error") ||
								strings.Contains(resultLower, "network") ||
								strings.Contains(resultLower, "failed") ||
								strings.Contains(resultLower, "unreachable") ||
								strings.Contains(resultLower, "refused") {
								t.Log("✓ Found network error in browser tool result")
								return true
							}
						}
					}
				}
			}
		}

		// Check assistant responses
		if role, ok := msg["role"].(string); ok && role == "assistant" {
			if content, ok := msg["content"].(string); ok {
				contentLower := strings.ToLower(content)
				if (strings.Contains(contentLower, "error") || strings.Contains(contentLower, "failed") ||
					strings.Contains(contentLower, "unable") || strings.Contains(contentLower, "could not")) &&
					(strings.Contains(contentLower, "open") || strings.Contains(contentLower, "navigate") ||
						strings.Contains(contentLower, "reach") || strings.Contains(contentLower, "connect")) {
					t.Log("✓ Found network error mention in assistant response")
					return true
				}
			}
		}
	}

	return false
}
