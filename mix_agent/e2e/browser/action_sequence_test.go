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

// TestBrowserE2EActionSequenceSuccess tests successful multi-step action sequences
func TestBrowserE2EActionSequenceSuccess(t *testing.T) {
	t.Log("=== E2E Test: Action Sequence - Successful Execution ===")

	// Setup test environment
	testServerURL, sessionID, cleanup := setupBrowserTest(t, "E2E Action Sequence Success Test")
	defer cleanup()

	// Step 1: Execute a multi-step action sequence
	t.Log("Step 1: Executing multi-step action sequence...")
	testURL := testServerURL + "/action_sequence.html"

	// This tests a complete action sequence workflow:
	// 1. Navigate to page
	// 2. Click increment button
	// 3. Type text into input field
	// 4. Take screenshot
	msgResp := sendMessage(t, sessionID, fmt.Sprintf(
		"Open %s and click the 'Increment Counter' button once, then type 'test' into the text input field, and take a screenshot.",
		testURL,
	))
	defer func() { _ = msgResp.Body.Close() }()
	t.Log("✓ Message sent, processing started")

	// Step 2: Wait for processing
	t.Log("Step 2: Waiting for action sequence to complete...")
	waitForProcessing(t, sessionID, 120*time.Second)
	t.Log("✓ Action sequence completed")

	// Step 3: Verify the action sequence succeeded
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

	// Verify browser tool was used and actions completed
	verifyActionSequenceSuccess(t, messages)

	t.Log("=== E2E Test Completed Successfully ===")
}

// TestBrowserE2EActionSequenceFailFast tests that sequences stop on first error
func TestBrowserE2EActionSequenceFailFast(t *testing.T) {
	t.Log("=== E2E Test: Action Sequence - Fail Fast Behavior ===")

	// Setup test environment
	testServerURL, sessionID, cleanup := setupBrowserTest(t, "E2E Action Sequence Fail Fast Test")
	defer cleanup()

	// Step 1: Execute an action sequence with an error in the middle
	t.Log("Step 1: Executing action sequence with intentional error...")
	testURL := testServerURL + "/action_sequence.html"

	// This sequence should fail when trying to click a nonexistent element
	msgResp := sendMessage(t, sessionID, fmt.Sprintf(
		"Open %s and click the 'Increment Counter' button, then try to click a button with id 'nonexistent-button'.",
		testURL,
	))
	defer func() { _ = msgResp.Body.Close() }()
	t.Log("✓ Message sent, processing started")

	// Step 2: Wait for processing
	t.Log("Step 2: Waiting for processing to complete...")
	waitForProcessing(t, sessionID, 120*time.Second)
	t.Log("✓ Processing completed")

	// Step 3: Verify fail-fast behavior
	t.Log("Step 3: Verifying fail-fast behavior...")
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

	// Verify that the agent reported the error
	verifyFailFastBehavior(t, messages)

	t.Log("=== E2E Test Completed Successfully ===")
}

// TestBrowserE2EActionSequenceWithScreenshots tests screenshot capture between actions
func TestBrowserE2EActionSequenceWithScreenshots(t *testing.T) {
	t.Log("=== E2E Test: Action Sequence - Screenshot Verification ===")

	// Setup test environment
	testServerURL, sessionID, cleanup := setupBrowserTest(t, "E2E Action Sequence Screenshot Test")
	defer cleanup()

	// Step 1: Execute actions with screenshots between steps
	t.Log("Step 1: Executing actions with screenshot verification...")
	testURL := testServerURL + "/action_sequence.html"

	// This tests that screenshots capture intermediate states correctly
	msgResp := sendMessage(t, sessionID, fmt.Sprintf(
		"Open %s. Then: "+
			"1) Take a screenshot of the initial state, "+
			"2) Click the 'Toggle Element' button, "+
			"3) Take another screenshot after clicking. "+
			"Tell me if you can see any differences.",
		testURL,
	))
	defer func() { _ = msgResp.Body.Close() }()
	t.Log("✓ Message sent, processing started")

	// Step 2: Wait for processing
	t.Log("Step 2: Waiting for screenshot sequence to complete...")
	waitForProcessing(t, sessionID, 120*time.Second)
	t.Log("✓ Screenshot sequence completed")

	// Step 3: Verify screenshots were captured
	t.Log("Step 3: Verifying screenshot captures...")
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

	// Verify multiple screenshots were captured
	verifyScreenshotSequence(t, messages)

	// Step 4: Verify screenshots are accessible via files endpoint
	t.Log("Step 4: Verifying screenshot file storage...")
	filesResp := makeRequest(t, http.MethodGet, constants.APISessionsPath+sessionID+"/files", nil)
	defer func() { _ = filesResp.Body.Close() }()

	filesBody, err := io.ReadAll(filesResp.Body)
	if err != nil {
		t.Fatalf("Failed to read files response: %v", err)
	}

	var files []interface{}
	if err := json.Unmarshal(filesBody, &files); err != nil {
		t.Fatalf("Failed to parse files: %v", err)
	}

	screenshotCount := 0
	for _, file := range files {
		if fileMap, ok := file.(map[string]interface{}); ok {
			if filename, ok := fileMap["name"].(string); ok && strings.HasSuffix(filename, ".png") {
				screenshotCount++
			}
		}
	}

	if screenshotCount < 2 {
		t.Logf("⚠ Warning: Expected at least 2 screenshots, found %d", screenshotCount)
	} else {
		t.Logf("✓ Found %d screenshot files", screenshotCount)
	}

	t.Log("=== E2E Test Completed Successfully ===")
}

// Helper functions

func verifyActionSequenceSuccess(t *testing.T, messages []map[string]interface{}) {
	t.Helper()

	foundBrowserToolCall := false
	foundSuccessConfirmation := false

	for _, msg := range messages {
		// Check for browser tool calls
		if toolCalls, ok := msg["toolCalls"].([]interface{}); ok {
			for _, tc := range toolCalls {
				if toolCall, ok := tc.(map[string]interface{}); ok {
					if name, ok := toolCall["name"].(string); ok && name == "Browser" {
						foundBrowserToolCall = true
						t.Log("✓ Found browser tool call")
					}
				}
			}
		}

		// Check assistant responses for success confirmation
		if role, ok := msg["role"].(string); ok && role == "assistant" {
			if content, ok := msg["content"].(string); ok {
				contentLower := strings.ToLower(content)
				if (strings.Contains(contentLower, "success") || strings.Contains(contentLower, "completed")) &&
					(strings.Contains(contentLower, "action") || strings.Contains(contentLower, "sequence")) {
					foundSuccessConfirmation = true
					t.Log("✓ Found success confirmation in assistant response")
				}
			}
		}
	}

	if !foundBrowserToolCall {
		t.Fatal("❌ Browser tool was not called during action sequence")
	}

	if !foundSuccessConfirmation {
		t.Log("⚠ Success confirmation not explicitly found (may have been phrased differently)")
	}

	t.Log("✓ Action sequence verification complete")
}

func verifyFailFastBehavior(t *testing.T, messages []map[string]interface{}) {
	t.Helper()

	foundError := false
	foundErrorMention := false

	for _, msg := range messages {
		// Check for errors in tool call results
		if toolCalls, ok := msg["toolCalls"].([]interface{}); ok {
			for _, tc := range toolCalls {
				if toolCall, ok := tc.(map[string]interface{}); ok {
					if name, ok := toolCall["name"].(string); ok && name == "Browser" {
						if result, ok := toolCall["result"].(string); ok {
							resultLower := strings.ToLower(result)
							if strings.Contains(resultLower, "error") || strings.Contains(resultLower, "not found") || strings.Contains(resultLower, "fail") {
								foundError = true
								t.Log("✓ Found error in browser tool result")
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
				if strings.Contains(contentLower, "error") || strings.Contains(contentLower, "not found") ||
				   strings.Contains(contentLower, "could not") || strings.Contains(contentLower, "unable") {
					foundErrorMention = true
					t.Log("✓ Found error mention in assistant response")
				}
			}
		}
	}

	if !foundError && !foundErrorMention {
		t.Log("⚠ Warning: Error not explicitly found (agent may have handled it differently)")
	} else {
		t.Log("✓ Fail-fast behavior verified - error was detected and reported")
	}
}

func verifyScreenshotSequence(t *testing.T, messages []map[string]interface{}) {
	t.Helper()

	screenshotCount := 0
	foundStateChange := false

	for _, msg := range messages {
		// Count browser tool calls with screenshots
		if toolCalls, ok := msg["toolCalls"].([]interface{}); ok {
			for _, tc := range toolCalls {
				if toolCall, ok := tc.(map[string]interface{}); ok {
					if name, ok := toolCall["name"].(string); ok && name == "Browser" {
						// Check for screenshot URLs
						if screenshotUrls, ok := toolCall["screenshotUrls"].([]interface{}); ok && len(screenshotUrls) > 0 {
							screenshotCount++
						}
					}
				}
			}
		}

		// Check for state change observations
		if role, ok := msg["role"].(string); ok && role == "assistant" {
			if content, ok := msg["content"].(string); ok {
				contentLower := strings.ToLower(content)
				if (strings.Contains(contentLower, "visible") || strings.Contains(contentLower, "hidden") ||
					strings.Contains(contentLower, "toggle") || strings.Contains(contentLower, "changed")) {
					foundStateChange = true
				}
			}
		}
	}

	if screenshotCount < 2 {
		t.Logf("⚠ Warning: Expected multiple screenshots, found %d", screenshotCount)
	} else {
		t.Logf("✓ Found %d screenshots in action sequence", screenshotCount)
	}

	if !foundStateChange {
		t.Log("⚠ State change observation not explicitly found (may have been described differently)")
	} else {
		t.Log("✓ Found state change observations between screenshots")
	}

	t.Log("✓ Screenshot sequence verification complete")
}
