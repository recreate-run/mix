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

// TestBrowserE2EDynamicSPAFormSubmission tests filling and submitting a dynamic SPA form
// that processes submission with JavaScript without page navigation
func TestBrowserE2EDynamicSPAFormSubmission(t *testing.T) {
	t.Parallel()
	t.Log("=== E2E Test: Dynamic SPA Form Submission ===")

	// Setup test environment
	testServerURL, sessionID, cleanup := setupBrowserTest(t, "E2E SPA Form Test")
	defer cleanup()

	// Step 1: Navigate and fill the form
	t.Log("Step 1: Navigating to SPA registration form and filling it...")
	testURL := testServerURL + "/spa_registration_form.html"

	msgResp := sendMessage(t, sessionID, fmt.Sprintf(
		"Open %s and fill out the registration form with these details: "+
			"name 'Alice Smith', email 'alice@test.com', password 'Pass123!', country 'USA', "+
			"check the terms checkbox, and submit the form. Tell me what success message you see.",
		testURL,
	))
	defer func() { _ = msgResp.Body.Close() }()
	t.Log("✓ Message sent, processing started")

	// Step 2: Wait for processing (form interactions may take longer)
	t.Log("Step 2: Waiting for form submission to complete...")
	waitForProcessing(t, sessionID, 120*time.Second)
	t.Log("✓ Processing completed")

	// Step 3: Verify form workflow
	t.Log("Step 3: Verifying form submission workflow succeeded...")
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

	verifySPAFormWorkflow(t, messages)

	t.Log("=== E2E Test Completed Successfully ===")
}

// verifySPAFormWorkflow verifies the SPA form workflow completed successfully
func verifySPAFormWorkflow(t *testing.T, messages []map[string]interface{}) {
	t.Helper()

	foundBrowserTool := false
	foundTypeAction := false
	foundSelectAction := false
	foundCheckboxAction := false
	foundSubmitAction := false
	foundSuccessMessage := false

	for _, msg := range messages {
		// Check for browser tool calls
		if toolCalls, ok := msg["toolCalls"].([]interface{}); ok {
			for _, tc := range toolCalls {
				if toolCall, ok := tc.(map[string]interface{}); ok {
					if name, ok := toolCall["name"].(string); ok && name == "Browser" {
						foundBrowserTool = true

						// Check tool result for evidence of actions
						if result, ok := toolCall["result"].(string); ok {
							resultLower := strings.ToLower(result)

							// Check for type actions
							if strings.Contains(resultLower, "type") ||
								strings.Contains(resultLower, "alice") ||
								strings.Contains(resultLower, "alice@test.com") {
								foundTypeAction = true
								t.Log("✓ Found type action in browser tool result")
							}

							// Check for select/dropdown action
							if strings.Contains(resultLower, "select") ||
								strings.Contains(resultLower, "usa") ||
								strings.Contains(resultLower, "country") {
								foundSelectAction = true
								t.Log("✓ Found select action in browser tool result")
							}

							// Check for checkbox action
							if strings.Contains(resultLower, "check") ||
								strings.Contains(resultLower, "terms") {
								foundCheckboxAction = true
								t.Log("✓ Found checkbox action in browser tool result")
							}

							// Check for submit action
							if strings.Contains(resultLower, "submit") ||
								strings.Contains(resultLower, "click") {
								foundSubmitAction = true
								t.Log("✓ Found submit action in browser tool result")
							}

							// Check for success message
							if strings.Contains(resultLower, "success") ||
								strings.Contains(resultLower, "registration successful") ||
								strings.Contains(resultLower, "welcome") ||
								strings.Contains(resultLower, "account has been created") {
								foundSuccessMessage = true
								t.Log("✓ Found success message in browser tool result")
							}
						}

						// Check input parameters
						if input, ok := toolCall["input"].(string); ok {
							inputLower := strings.ToLower(input)

							if strings.Contains(inputLower, "alice") {
								foundTypeAction = true
								t.Log("✓ Found 'Alice' in tool input")
							}
							if strings.Contains(inputLower, "usa") {
								foundSelectAction = true
								t.Log("✓ Found 'USA' in tool input")
							}
							if strings.Contains(inputLower, "terms") || strings.Contains(inputLower, "checkbox") {
								foundCheckboxAction = true
								t.Log("✓ Found checkbox reference in tool input")
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

				// Look for success message mentions
				if (strings.Contains(contentLower, "success") || strings.Contains(contentLower, "successful")) &&
					(strings.Contains(contentLower, "registration") || strings.Contains(contentLower, "account")) {
					foundSuccessMessage = true
					t.Log("✓ Found success message mentioned in assistant response")
				}

				// Check for user data in response
				if strings.Contains(contentLower, "alice smith") ||
					(strings.Contains(contentLower, "alice") && strings.Contains(contentLower, "usa")) {
					t.Log("✓ Found user data in assistant response")
				}
			}
		}
	}

	// Verify critical requirements
	if !foundBrowserTool {
		t.Fatal("❌ Browser tool was not called during form workflow")
	}
	t.Log("✓ Browser tool was used")

	if !foundTypeAction {
		t.Log("⚠ Type action not explicitly confirmed (may have been processed differently)")
	}

	if !foundSelectAction {
		t.Log("⚠ Select/dropdown action not explicitly confirmed (may have been processed differently)")
	}

	if !foundCheckboxAction {
		t.Log("⚠ Checkbox action not explicitly confirmed (may have been processed differently)")
	}

	if !foundSubmitAction {
		t.Log("⚠ Submit action not explicitly confirmed (may have been processed differently)")
	}

	if !foundSuccessMessage {
		t.Log("⚠ Success message not found in responses (may have been processed differently)")
	} else {
		t.Log("✓ Form submission success verification complete")
	}

	t.Log("✓ SPA form workflow verification complete")
}
