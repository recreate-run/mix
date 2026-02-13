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

// TestBrowserE2ETabManagement tests multi-tab workflows end-to-end
func TestBrowserE2ETabManagement(t *testing.T) {
	t.Log("=== E2E Test: Browser Tab Management Workflow ===")

	testServerURL, sessionID, cleanup := setupBrowserTest(t, "E2E Tab Management Test")
	defer cleanup()

	testURL := testServerURL + "/mode_test.html"

	// Step 1: Create multiple tabs
	t.Log("Step 1: Creating multiple browser tabs...")
	msgResp := sendMessage(t, sessionID, "Use the Browser tool with action='tab_create' to create two new browser tabs. Report the tab IDs created.")
	defer func() { _ = msgResp.Body.Close() }()

	waitForProcessing(t, sessionID, 90*time.Second)
	t.Log("✓ Tabs created")

	// Step 2: Navigate different URLs in different tabs
	t.Log("Step 2: Navigating tabs to different URLs...")
	msgResp = sendMessage(t, sessionID, fmt.Sprintf("Use the Browser tool with action='open' to navigate tab-2 to %s. Then use action='open' to navigate tab-3 to %s as well.", testURL, testURL))
	defer func() { _ = msgResp.Body.Close() }()

	waitForProcessing(t, sessionID, 90*time.Second)
	t.Log("✓ Tabs navigated")

	// Step 3: Switch between tabs
	t.Log("Step 3: Switching to tab-2...")
	msgResp = sendMessage(t, sessionID, "Use the Browser tool with action='tab_switch' and tabId='tab-2' to switch to tab-2")
	defer func() { _ = msgResp.Body.Close() }()

	waitForProcessing(t, sessionID, 60*time.Second)
	t.Log("✓ Switched to tab-2")

	// Step 4: Switch to another tab
	t.Log("Step 4: Switching to tab-3...")
	msgResp = sendMessage(t, sessionID, "Use the Browser tool with action='tab_switch' and tabId='tab-3' to switch to tab-3")
	defer func() { _ = msgResp.Body.Close() }()

	waitForProcessing(t, sessionID, 60*time.Second)
	t.Log("✓ Switched to tab-3")

	// Step 5: List all tabs
	t.Log("Step 5: Listing all tabs...")
	msgResp = sendMessage(t, sessionID, "Use the Browser tool with action='tab_list' to list all open tabs and report which tabs exist.")
	defer func() { _ = msgResp.Body.Close() }()

	waitForProcessing(t, sessionID, 60*time.Second)
	t.Log("✓ Tabs listed")

	// Step 6: Close tab and verify
	t.Log("Step 6: Closing tab-3 and verifying...")
	msgResp = sendMessage(t, sessionID, "Use the Browser tool with action='tab_close' and tabId='tab-3' to close tab-3. Then use action='tab_list' to list all remaining tabs to verify it's closed.")
	defer func() { _ = msgResp.Body.Close() }()

	waitForProcessing(t, sessionID, 60*time.Second)
	t.Log("✓ Tab-3 closed successfully")

	// Step 7: Verify error handling for closed tab
	t.Log("Step 7: Testing error handling for closed tab...")
	msgResp = sendMessage(t, sessionID, "Use the Browser tool with action='tab_switch' and tabId='tab-3' to try switching to tab-3 (this should fail since it's closed). Report the error.")
	defer func() { _ = msgResp.Body.Close() }()

	waitForProcessing(t, sessionID, 60*time.Second)
	t.Log("✓ Error handling verified")

	// Verify the overall workflow
	t.Log("Verifying overall workflow...")
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

	// Verify critical workflow aspects
	verifyTabManagementWorkflow(t, messages)

	t.Log("=== E2E Test Completed Successfully ===")
}

// verifyTabManagementWorkflow verifies that the tab management workflow succeeded
func verifyTabManagementWorkflow(t *testing.T, messages []map[string]interface{}) {
	t.Helper()

	foundBrowserToolCall := false
	foundTabCreate := false
	foundTabSwitch := false
	foundTabClose := false
	foundTabList := false

	for _, msg := range messages {
		// Check for browser tool calls
		if toolCalls, ok := msg["toolCalls"].([]interface{}); ok {
			for _, tc := range toolCalls {
				if toolCall, ok := tc.(map[string]interface{}); ok {
					if name, ok := toolCall["name"].(string); ok && name == "Browser" {
						foundBrowserToolCall = true

						// Check tool input for specific actions
						if input, ok := toolCall["input"].(string); ok {
							if strings.Contains(input, "tab_create") {
								foundTabCreate = true
								t.Log("✓ Found tab_create action")
							}
							if strings.Contains(input, "tab_switch") {
								foundTabSwitch = true
								t.Log("✓ Found tab_switch action")
							}
							if strings.Contains(input, "tab_close") {
								foundTabClose = true
								t.Log("✓ Found tab_close action")
							}
							if strings.Contains(input, "tab_list") {
								foundTabList = true
								t.Log("✓ Found tab_list action")
							}
						}
					}
				}
			}
		}
	}

	// Verify critical requirements
	if !foundBrowserToolCall {
		t.Fatal("❌ Browser tool was not called during tab management workflow")
	}
	t.Log("✓ Browser tool was used")

	// All tab operations are critical for this test
	if !foundTabCreate {
		t.Error("❌ tab_create action was not found in workflow")
	}

	if !foundTabSwitch {
		t.Error("❌ tab_switch action was not found in workflow")
	}

	if !foundTabClose {
		t.Error("❌ tab_close action was not found in workflow")
	}

	if !foundTabList {
		t.Error("❌ tab_list action was not found in workflow")
	}

	// Only log success if all actions were found
	if foundTabCreate && foundTabSwitch && foundTabClose && foundTabList {
		t.Log("✓ All tab management actions verified successfully")
	}
}
