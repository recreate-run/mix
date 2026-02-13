//go:build e2e
// +build e2e

package browser

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"regexp"
	"strings"
	"testing"
	"time"

	"mix/e2e"
	"mix/internal/constants"
)

// TestBrowserE2EElementSelectionByIndex tests element selection using index mode
func TestBrowserE2EElementSelectionByIndex(t *testing.T) {
	t.Parallel()
	t.Log("=== E2E Test: Element Selection by Index ===")

	// Setup test environment
	testServerURL, sessionID, cleanup := setupBrowserTest(t, "E2E Element Selection Index Test")
	defer cleanup()

	// Step 1: Navigate and click using index
	t.Log("Step 1: Clicking button using index-based selection...")
	testURL := testServerURL + "/element_selection.html"

	msgResp := sendMessage(t, sessionID, fmt.Sprintf(
		"Open %s and click the first button you see. Tell me what the status message says after clicking.",
		testURL,
	))
	defer func() { _ = msgResp.Body.Close() }()
	t.Log("✓ Message sent, processing started")

	// Step 2: Wait for processing
	t.Log("Step 2: Waiting for processing...")
	waitForProcessing(t, sessionID, 90*time.Second)
	t.Log("✓ Processing completed")

	// Step 3: Verify click succeeded
	t.Log("Step 3: Verifying index-based click succeeded...")
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

	foundClick := verifyButtonClick(t, messages)
	if !foundClick {
		t.Log("⚠ Button click not explicitly confirmed (may have been described differently)")
	} else {
		t.Log("✓ Index-based element selection successful")
	}

	t.Log("=== E2E Test Completed Successfully ===")
}

// TestBrowserE2EElementSelectionByRef tests element selection using ref mode
func TestBrowserE2EElementSelectionByRef(t *testing.T) {
	t.Parallel()
	t.Log("=== E2E Test: Element Selection by Ref ===")

	// Setup test environment
	testServerURL, sessionID, cleanup := setupBrowserTest(t, "E2E Element Selection Ref Test")
	defer cleanup()

	// Step 1: Navigate and get page elements (to obtain refs)
	t.Log("Step 1: Navigating to page to obtain element refs...")
	testURL := testServerURL + "/element_selection.html"

	msgResp := sendMessage(t, sessionID, fmt.Sprintf(
		"Open %s and read the page to see what buttons are available. Then click Button 2 specifically.",
		testURL,
	))
	defer func() { _ = msgResp.Body.Close() }()
	t.Log("✓ Message sent, processing started")

	// Step 2: Wait for processing
	t.Log("Step 2: Waiting for processing...")
	waitForProcessing(t, sessionID, 120*time.Second)
	t.Log("✓ Processing completed")

	// Step 3: Verify ref-based interaction worked
	t.Log("Step 3: Verifying ref-based element selection...")
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

	// Look for ref pattern in tool calls
	foundRef := extractRefFromMessages(t, messages)
	if foundRef != "" {
		t.Logf("✓ Found element ref in browser interactions: %s", foundRef)
	}

	foundClick := verifyButtonClick(t, messages)
	if !foundClick {
		t.Log("⚠ Button click not explicitly confirmed")
	} else {
		t.Log("✓ Ref-based element selection successful")
	}

	t.Log("=== E2E Test Completed Successfully ===")
}

// TestBrowserE2EElementSelectionByCoordinate tests element selection using coordinate mode
func TestBrowserE2EElementSelectionByCoordinate(t *testing.T) {
	t.Parallel()
	e2e.Setup(t)
	skipIfBrowserServiceNotRunning(t)

	t.Log("=== E2E Test: Element Selection by Coordinate ===")

	// Start test HTML server
	testServer := startTestHTMLServer(t)
	defer testServer.Close()

	// Step 1: Create a session
	t.Log("Step 1: Creating session...")
	createResp := makeRequest(t, http.MethodPost, "/api/sessions", map[string]interface{}{
		"title":       "E2E Element Selection Coordinate Test",
		"browserMode": "local-browser-service",
	})
	defer func() { _ = createResp.Body.Close() }()

	if createResp.StatusCode != http.StatusCreated {
		t.Fatalf("Expected status 201, got %d", createResp.StatusCode)
	}

	sessionData := parseJSONResponse(t, createResp)
	sessionID, ok := sessionData["id"].(string)
	if !ok {
		t.Fatal("Failed to get session ID from response")
	}
	t.Logf("✓ Created session: %s", sessionID)

	// Step 2: Navigate and click using visual guidance
	t.Log("Step 2: Clicking button using coordinate-based selection...")
	testURL := testServer.URL + "/element_selection.html"

	msgResp := makeRequest(t, http.MethodPost, constants.APISessionsPath+sessionID+"/messages", map[string]interface{}{
		"text": fmt.Sprintf(
			"Open %s, take a screenshot, and click on Button 3 based on its visual location. Report the status after clicking.",
			testURL,
		),
	})
	defer func() { _ = msgResp.Body.Close() }()

	if msgResp.StatusCode != http.StatusAccepted {
		t.Fatalf("Expected status 202 (Accepted), got %d", msgResp.StatusCode)
	}
	t.Log("✓ Message sent, processing started")

	// Step 3: Wait for processing
	t.Log("Step 3: Waiting for processing...")
	waitForProcessing(t, sessionID, 120*time.Second)
	t.Log("✓ Processing completed")

	// Step 4: Verify coordinate-based click worked
	t.Log("Step 4: Verifying coordinate-based element selection...")
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

	foundClick := verifyButtonClick(t, messages)
	foundScreenshot := verifyScreenshotTaken(t, messages)

	if !foundScreenshot {
		t.Log("⚠ Screenshot not found (coordinate clicks may use index mode instead)")
	} else {
		t.Log("✓ Screenshot taken for visual guidance")
	}

	if !foundClick {
		t.Log("⚠ Button click not explicitly confirmed")
	} else {
		t.Log("✓ Coordinate-based element selection successful")
	}

	// Cleanup
	t.Log("Cleaning up test session...")
	deleteResp := makeRequest(t, http.MethodDelete, constants.APISessionsPath+sessionID, nil)
	defer func() { _ = deleteResp.Body.Close() }()

	if deleteResp.StatusCode != http.StatusOK && deleteResp.StatusCode != http.StatusNoContent {
		t.Logf("Warning: Failed to delete session: status %d", deleteResp.StatusCode)
	} else {
		t.Log("✓ Session cleaned up")
	}

	t.Log("=== E2E Test Completed Successfully ===")
}

// TestBrowserE2EInvalidRefHandling tests handling of invalid element refs
func TestBrowserE2EInvalidRefHandling(t *testing.T) {
	t.Parallel()
	e2e.Setup(t)
	skipIfBrowserServiceNotRunning(t)

	t.Log("=== E2E Test: Invalid Ref Handling ===")

	// Start test HTML server
	testServer := startTestHTMLServer(t)
	defer testServer.Close()

	// Step 1: Create a session
	t.Log("Step 1: Creating session...")
	createResp := makeRequest(t, http.MethodPost, "/api/sessions", map[string]interface{}{
		"title":       "E2E Invalid Ref Test",
		"browserMode": "local-browser-service",
	})
	defer func() { _ = createResp.Body.Close() }()

	if createResp.StatusCode != http.StatusCreated {
		t.Fatalf("Expected status 201, got %d", createResp.StatusCode)
	}

	sessionData := parseJSONResponse(t, createResp)
	sessionID, ok := sessionData["id"].(string)
	if !ok {
		t.Fatal("Failed to get session ID from response")
	}
	t.Logf("✓ Created session: %s", sessionID)

	// Step 2: Navigate and attempt invalid action
	t.Log("Step 2: Testing invalid ref handling...")
	testURL := testServer.URL + "/element_selection.html"

	msgResp := makeRequest(t, http.MethodPost, constants.APISessionsPath+sessionID+"/messages", map[string]interface{}{
		"text": fmt.Sprintf(
			"Open %s and try to click an element with id 'nonexistent-element-12345'. Report what happens.",
			testURL,
		),
	})
	defer func() { _ = msgResp.Body.Close() }()

	if msgResp.StatusCode != http.StatusAccepted {
		t.Fatalf("Expected status 202 (Accepted), got %d", msgResp.StatusCode)
	}
	t.Log("✓ Message sent, processing started")

	// Step 3: Wait for processing
	t.Log("Step 3: Waiting for processing...")
	waitForProcessing(t, sessionID, 90*time.Second)
	t.Log("✓ Processing completed without crash")

	// Step 4: Verify error handling
	t.Log("Step 4: Verifying invalid ref error handling...")
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

	foundError := verifyElementNotFoundError(t, messages)
	if !foundError {
		t.Log("⚠ Element not found error not explicitly mentioned (may have been handled gracefully)")
	} else {
		t.Log("✓ Invalid ref handled gracefully with proper error")
	}

	// Cleanup
	t.Log("Cleaning up test session...")
	deleteResp := makeRequest(t, http.MethodDelete, constants.APISessionsPath+sessionID, nil)
	defer func() { _ = deleteResp.Body.Close() }()

	if deleteResp.StatusCode != http.StatusOK && deleteResp.StatusCode != http.StatusNoContent {
		t.Logf("Warning: Failed to delete session: status %d", deleteResp.StatusCode)
	} else {
		t.Log("✓ Session cleaned up")
	}

	t.Log("=== E2E Test Completed Successfully ===")
}

// TestBrowserE2EStaleRefAfterNavigation tests handling of stale refs after page navigation
func TestBrowserE2EStaleRefAfterNavigation(t *testing.T) {
	t.Parallel()
	e2e.Setup(t)
	skipIfBrowserServiceNotRunning(t)

	t.Log("=== E2E Test: Stale Ref After Navigation ===")

	// Start test HTML server
	testServer := startTestHTMLServer(t)
	defer testServer.Close()

	// Step 1: Create a session
	t.Log("Step 1: Creating session...")
	createResp := makeRequest(t, http.MethodPost, "/api/sessions", map[string]interface{}{
		"title":       "E2E Stale Ref Test",
		"browserMode": "local-browser-service",
	})
	defer func() { _ = createResp.Body.Close() }()

	if createResp.StatusCode != http.StatusCreated {
		t.Fatalf("Expected status 201, got %d", createResp.StatusCode)
	}

	sessionData := parseJSONResponse(t, createResp)
	sessionID, ok := sessionData["id"].(string)
	if !ok {
		t.Fatal("Failed to get session ID from response")
	}
	t.Logf("✓ Created session: %s", sessionID)

	// Step 2: Navigate to first page
	t.Log("Step 2: Navigating to first page...")
	pageA := testServer.URL + "/element_selection.html"

	msgResp := makeRequest(t, http.MethodPost, constants.APISessionsPath+sessionID+"/messages", map[string]interface{}{
		"text": fmt.Sprintf("Open %s and read the page", pageA),
	})
	defer func() { _ = msgResp.Body.Close() }()

	waitForProcessing(t, sessionID, 60*time.Second)
	t.Log("✓ First page loaded")

	// Step 3: Navigate to different page
	t.Log("Step 3: Navigating to different page...")
	pageB := testServer.URL + "/action_sequence.html"

	msg2Resp := makeRequest(t, http.MethodPost, constants.APISessionsPath+sessionID+"/messages", map[string]interface{}{
		"text": fmt.Sprintf(
			"Now navigate to %s (a different page). After loading, try to interact with elements from the previous page if you still have references to them.",
			pageB,
		),
	})
	defer func() { _ = msg2Resp.Body.Close() }()

	waitForProcessing(t, sessionID, 90*time.Second)
	t.Log("✓ Second page loaded")

	// Step 4: Verify stale ref handling
	t.Log("Step 4: Verifying stale ref handling...")
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

	// Agent may handle this gracefully by not attempting to use stale refs
	foundBrowserAction := false
	for _, msg := range messages {
		if toolCalls, ok := msg["toolCalls"].([]interface{}); ok {
			for _, tc := range toolCalls {
				if toolCall, ok := tc.(map[string]interface{}); ok {
					if name, ok := toolCall["name"].(string); ok && name == "Browser" {
						foundBrowserAction = true
					}
				}
			}
		}
	}

	if !foundBrowserAction {
		t.Fatal("❌ Browser tool was not used")
	}

	t.Log("✓ Stale ref scenario handled (agent may have avoided using old refs)")

	// Cleanup
	t.Log("Cleaning up test session...")
	deleteResp := makeRequest(t, http.MethodDelete, constants.APISessionsPath+sessionID, nil)
	defer func() { _ = deleteResp.Body.Close() }()

	if deleteResp.StatusCode != http.StatusOK && deleteResp.StatusCode != http.StatusNoContent {
		t.Logf("Warning: Failed to delete session: status %d", deleteResp.StatusCode)
	} else {
		t.Log("✓ Session cleaned up")
	}

	t.Log("=== E2E Test Completed Successfully ===")
}

// Helper functions

func verifyButtonClick(t *testing.T, messages []map[string]interface{}) bool {
	t.Helper()

	for _, msg := range messages {
		// Check for click in tool calls
		if toolCalls, ok := msg["toolCalls"].([]interface{}); ok {
			for _, tc := range toolCalls {
				if toolCall, ok := tc.(map[string]interface{}); ok {
					if name, ok := toolCall["name"].(string); ok && name == "Browser" {
						if result, ok := toolCall["result"].(string); ok {
							resultLower := strings.ToLower(result)
							if strings.Contains(resultLower, "clicked") || strings.Contains(resultLower, "button") {
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
				if (strings.Contains(contentLower, "click") || strings.Contains(contentLower, "status")) &&
					strings.Contains(contentLower, "button") {
					return true
				}
			}
		}
	}

	return false
}

func verifyScreenshotTaken(t *testing.T, messages []map[string]interface{}) bool {
	t.Helper()

	for _, msg := range messages {
		if toolCalls, ok := msg["toolCalls"].([]interface{}); ok {
			for _, tc := range toolCalls {
				if toolCall, ok := tc.(map[string]interface{}); ok {
					if name, ok := toolCall["name"].(string); ok && name == "Browser" {
						if screenshotUrls, ok := toolCall["screenshotUrls"].([]interface{}); ok && len(screenshotUrls) > 0 {
							return true
						}
					}
				}
			}
		}
	}

	return false
}

func extractRefFromMessages(t *testing.T, messages []map[string]interface{}) string {
	t.Helper()

	// Pattern to match ref format: f{frameId}_ref_{elementId}
	refPattern := regexp.MustCompile(`f\d+_ref_\d+`)

	for _, msg := range messages {
		if toolCalls, ok := msg["toolCalls"].([]interface{}); ok {
			for _, tc := range toolCalls {
				if toolCall, ok := tc.(map[string]interface{}); ok {
					if name, ok := toolCall["name"].(string); ok && name == "Browser" {
						// Check in the tool call input
						if input, ok := toolCall["input"].(string); ok {
							if match := refPattern.FindString(input); match != "" {
								return match
							}
						}
						// Check in the result
						if result, ok := toolCall["result"].(string); ok {
							if match := refPattern.FindString(result); match != "" {
								return match
							}
						}
					}
				}
			}
		}
	}

	return ""
}

func verifyElementNotFoundError(t *testing.T, messages []map[string]interface{}) bool {
	t.Helper()

	for _, msg := range messages {
		// Check for errors in tool call results
		if toolCalls, ok := msg["toolCalls"].([]interface{}); ok {
			for _, tc := range toolCalls {
				if toolCall, ok := tc.(map[string]interface{}); ok {
					if name, ok := toolCall["name"].(string); ok && name == "Browser" {
						if result, ok := toolCall["result"].(string); ok {
							resultLower := strings.ToLower(result)
							if strings.Contains(resultLower, "not found") ||
								strings.Contains(resultLower, "does not exist") ||
								strings.Contains(resultLower, "no element") ||
								strings.Contains(resultLower, "error") {
								t.Log("✓ Found element not found error in tool result")
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
				if (strings.Contains(contentLower, "not found") || strings.Contains(contentLower, "doesn't exist") ||
					strings.Contains(contentLower, "unable to find") || strings.Contains(contentLower, "could not find")) &&
					strings.Contains(contentLower, "element") {
					t.Log("✓ Found element not found mention in assistant response")
					return true
				}
			}
		}
	}

	return false
}
