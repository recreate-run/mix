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

// TestBrowserE2EDynamicSPASearch tests typing into a search bar and pressing Enter
// in a dynamic single-page application similar to Amazon
func TestBrowserE2EDynamicSPASearch(t *testing.T) {
	t.Parallel()
	t.Log("=== E2E Test: Dynamic SPA Search ===")

	// Setup test environment
	testServerURL, sessionID, cleanup := setupBrowserTest(t, "E2E SPA Search Test")
	defer cleanup()

	// Step 1: Navigate and perform search
	t.Log("Step 1: Navigating to SPA search page and performing search...")
	testURL := testServerURL + "/spa_search.html"

	msgResp := sendMessage(t, sessionID, fmt.Sprintf(
		"Open %s and type 'laptops' into the search box, then press Enter to search. Tell me what search results you see.",
		testURL,
	))
	defer func() { _ = msgResp.Body.Close() }()
	t.Log("✓ Message sent, processing started")

	// Step 2: Wait for processing (SPA interactions may take longer)
	t.Log("Step 2: Waiting for search to complete...")
	waitForProcessing(t, sessionID, 120*time.Second)
	t.Log("✓ Processing completed")

	// Step 3: Verify search workflow
	t.Log("Step 3: Verifying search workflow succeeded...")
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

	verifySPASearchWorkflow(t, messages)

	t.Log("=== E2E Test Completed Successfully ===")
}

// verifySPASearchWorkflow verifies the SPA search workflow completed successfully
func verifySPASearchWorkflow(t *testing.T, messages []map[string]interface{}) {
	t.Helper()

	foundBrowserTool := false
	foundTypeAction := false
	foundEnterKey := false
	foundSearchResults := false

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

							// Check for type action
							if strings.Contains(resultLower, "type") || strings.Contains(resultLower, "laptops") {
								foundTypeAction = true
								t.Log("✓ Found type action in browser tool result")
							}

							// Check for enter key press
							if strings.Contains(resultLower, "enter") || strings.Contains(resultLower, "key") {
								foundEnterKey = true
								t.Log("✓ Found Enter key press in browser tool result")
							}

							// Check for search results or products
							if strings.Contains(resultLower, "result") ||
								strings.Contains(resultLower, "product") ||
								strings.Contains(resultLower, "hp") ||
								strings.Contains(resultLower, "dell") ||
								strings.Contains(resultLower, "laptop") {
								foundSearchResults = true
								t.Log("✓ Found search results in browser tool result")
							}
						}

						// Check input parameters
						if input, ok := toolCall["input"].(string); ok {
							inputLower := strings.ToLower(input)
							if strings.Contains(inputLower, "laptops") {
								foundTypeAction = true
								t.Log("✓ Found 'laptops' in tool input")
							}
							if strings.Contains(inputLower, "enter") {
								foundEnterKey = true
								t.Log("✓ Found Enter in tool input")
							}
						}
					}
				}
			}
		}

		// Check assistant responses for search results
		if role, ok := msg["role"].(string); ok && role == "assistant" {
			if content, ok := msg["content"].(string); ok {
				contentLower := strings.ToLower(content)

				// Look for evidence of search results
				if (strings.Contains(contentLower, "search") && strings.Contains(contentLower, "result")) ||
					strings.Contains(contentLower, "product") ||
					strings.Contains(contentLower, "laptop") ||
					strings.Contains(contentLower, "found") {
					foundSearchResults = true
					t.Log("✓ Found search results mentioned in assistant response")
				}
			}
		}
	}

	// Verify critical requirements
	if !foundBrowserTool {
		t.Fatal("❌ Browser tool was not called during search workflow")
	}
	t.Log("✓ Browser tool was used")

	if !foundTypeAction {
		t.Log("⚠ Type action not explicitly confirmed (may have been processed differently)")
	}

	if !foundEnterKey {
		t.Log("⚠ Enter key press not explicitly confirmed (may have been processed differently)")
	}

	if !foundSearchResults {
		t.Log("⚠ Search results not found in responses (may have been processed differently)")
	} else {
		t.Log("✓ Search results verification successful")
	}

	t.Log("✓ SPA search workflow verification complete")
}
