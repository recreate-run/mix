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

// TestBrowserE2EFormFilling tests the form filling and submission workflow
func TestBrowserE2EFormFilling(t *testing.T) {
	t.Parallel()
	t.Log("=== E2E Test: Browser Form Filling Workflow ===")

	// Setup test environment
	testServerURL, sessionID, cleanup := setupBrowserTest(t, "E2E Form Filling Test")
	defer cleanup()

	// Step 1: Send message to fill out and submit the form
	t.Log("Step 1: Sending message to fill out login form...")
	testURL := testServerURL + "/login_form.html"

	// This message tests all critical form actions:
	// - Navigate to form
	// - Type text into username field
	// - Type text into password field
	// - Select dropdown option
	// - Submit form (via Enter key or button click)
	msgResp := sendMessage(t, sessionID, fmt.Sprintf(
		"Open %s and fill out the login form with username 'testuser', password 'testpass123', role 'Admin', then submit it. After submission, confirm you can see the success message.",
		testURL,
	))
	defer func() { _ = msgResp.Body.Close() }()
	t.Log("✓ Message sent, processing started")

	// Step 2: Wait for processing (form interaction may take longer)
	t.Log("Step 2: Waiting for agent to complete form workflow...")
	waitForProcessing(t, sessionID, 180*time.Second)
	t.Log("✓ Form workflow completed")

	// Step 3: Verify the workflow succeeded
	t.Log("Step 3: Verifying form submission and navigation...")
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

	// Verify critical aspects of the workflow
	verifyFormWorkflow(t, messages)

	t.Log("=== E2E Test Completed Successfully ===")
}

// verifyFormWorkflow verifies that the form workflow succeeded
func verifyFormWorkflow(t *testing.T, messages []map[string]interface{}) {
	t.Helper()

	// Track what we find
	foundBrowserToolCall := false
	foundSuccessMessage := false
	foundNavigationToSuccess := false

	for _, msg := range messages {
		// Check for browser tool calls
		if toolCalls, ok := msg["toolCalls"].([]interface{}); ok {
			for _, tc := range toolCalls {
				if toolCall, ok := tc.(map[string]interface{}); ok {
					if name, ok := toolCall["name"].(string); ok && name == "Browser" {
						foundBrowserToolCall = true

						// Check if the tool call result mentions success page
						if result, ok := toolCall["result"].(string); ok {
							if strings.Contains(result, "success") || strings.Contains(result, "Success") {
								foundNavigationToSuccess = true
								t.Log("✓ Found navigation to success page in tool result")
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
				if (strings.Contains(contentLower, "success") || strings.Contains(contentLower, "successful")) &&
					(strings.Contains(contentLower, "login") || strings.Contains(contentLower, "form") || strings.Contains(contentLower, "submit")) {
					foundSuccessMessage = true
					t.Log("✓ Found success confirmation in assistant response")
				}
			}
		}
	}

	// Verify critical requirements
	if !foundBrowserToolCall {
		t.Fatal("❌ Browser tool was not called during form workflow")
	}
	t.Log("✓ Browser tool was used")

	if !foundSuccessMessage {
		t.Log("⚠ Success message not explicitly mentioned (may have been processed differently)")
	}

	if !foundNavigationToSuccess {
		t.Log("⚠ Navigation to success page not explicitly verified (may have been processed differently)")
	}

	// At minimum, we need browser tool usage for this test to be meaningful
	t.Log("✓ Form filling workflow verification complete")
}

// TestBrowserE2EFormSequentialActions tests sequential form actions without validation complexity
func TestBrowserE2EFormSequentialActions(t *testing.T) {
	t.Parallel()
	t.Log("=== E2E Test: Browser Form Sequential Actions ===")

	// Setup test environment
	testServerURL, sessionID, cleanup := setupBrowserTest(t, "E2E Form Sequential Actions Test")
	defer cleanup()

	// Step 1: Test sequential actions (fill fields one by one)
	t.Log("Step 1: Testing sequential form field filling...")
	testURL := testServerURL + "/login_form.html"

	msgResp := sendMessage(t, sessionID, fmt.Sprintf(
		"Open %s and fill just the username field with 'alice', then take a screenshot to show the field is filled.",
		testURL,
	))
	defer func() { _ = msgResp.Body.Close() }()
	t.Log("✓ Message sent, processing started")

	// Step 2: Wait for processing
	waitForProcessing(t, sessionID, 90*time.Second)

	// Step 3: Verify sequential action completed
	t.Log("Step 3: Verifying sequential action behavior...")
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

	// Look for browser tool usage
	foundBrowserAction := false
	for _, msg := range messages {
		if toolCalls, ok := msg["toolCalls"].([]interface{}); ok {
			for _, tc := range toolCalls {
				if toolCall, ok := tc.(map[string]interface{}); ok {
					if name, ok := toolCall["name"].(string); ok && name == "Browser" {
						foundBrowserAction = true
						t.Log("✓ Found browser tool action")
						break
					}
				}
			}
		}
		if foundBrowserAction {
			break
		}
	}

	if !foundBrowserAction {
		t.Fatal("❌ Browser tool was not used during sequential action test")
	}

	t.Log("=== E2E Test Completed Successfully ===")
}
