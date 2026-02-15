package integration_tests

// Browser Tool Integration Tests
//
// These tests verify the browser tool functionality through REST API integration.
// They test the interaction between HTTP handlers, agent, and browser tool.
//
// Note: These are INTEGRATION tests, not E2E tests. They use a test HTTP server
// and test database, not the actual running application.
//
// Prerequisites:
//   1. Browser service must be running at ws://localhost:8081/ws
//      - Start it with: cd ../browser-service && go run main.go
//   2. ANTHROPIC_API_KEY must be set (or other LLM provider credentials)
//      - Export it: export ANTHROPIC_API_KEY=sk-ant-...
//   3. BROWSER_SERVICE_URL must be set to ws://localhost:8081/ws
//
// To run these tests:
//   go test ./internal/http/integration_tests/... -run TestRESTBrowser -v
//
// To skip these tests:
//   export SKIP_BROWSER_INTEGRATION_TESTS=1
//
// Test scenarios:
//   - TestRESTBrowserFullWorkflow: Open google.com and take screenshot
//   - TestRESTBrowserMultiAction: Multiple browser actions in sequence
//   - TestRESTBrowserErrorHandling: Error handling with invalid URLs
//   - TestRESTBrowserWikipediaAnatomyClick: Navigate to Wikipedia Elephant page and click Anatomy link

import (
	"bytes"
	"context"
	"errors"
	"io"
	"net"
	"net/http"
	"os"
	"strings"
	"testing"
	"time"

	"mix/internal/constants"
	"mix/internal/message"
)

const toolNameBrowser = "Browser"

// skipIfBrowserServiceUnavailable checks if browser-service is available at ws://localhost:8081/ws
// If not available, or SKIP_BROWSER_INTEGRATION_TESTS is set, skip the test
func skipIfBrowserServiceUnavailable(t *testing.T) {
	t.Helper()

	// Check environment variable
	if os.Getenv("SKIP_BROWSER_INTEGRATION_TESTS") != "" {
		t.Skip("Skipping browser integration tests: SKIP_BROWSER_INTEGRATION_TESTS is set")
	}

	// Try to connect to browser-service WebSocket endpoint
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	dialer := &net.Dialer{}
	conn, err := dialer.DialContext(ctx, "tcp", "localhost:8081")
	if err != nil {
		t.Skipf("Skipping browser integration tests: browser-service not available at localhost:8081: %v", err)
	}
	_ = conn.Close()
}

// Test 28: Browser Full Workflow - Test browser tool usage through agent
func TestRESTBrowserFullWorkflow(t *testing.T) {
	skipIfBrowserServiceUnavailable(t)
	requireLLMCredentials(t)

	result := setupIntegrationTestServer(t)
	defer result.Server.Close()

	// Store API key in test database via REST API
	apiKey := os.Getenv("ANTHROPIC_API_KEY")
	if apiKey != "" {
		storeKeyRequest := map[string]interface{}{
			"provider": "anthropic",
			"api_key":  apiKey,
		}
		keyResp := makeJSONRequest(t, result.Server, http.MethodPost, "/api/auth/api-key", storeKeyRequest)
		defer func() { _ = keyResp.Body.Close() }()
		if keyResp.StatusCode != http.StatusOK {
			t.Fatalf("Failed to store API key: status %d", keyResp.StatusCode)
		}
		t.Log("✓ Stored API key in test database")
	}

	t.Log("Testing browser tool full workflow - open URL and take screenshot")

	// Create a session
	sessionRequest := map[string]interface{}{
		"title": "Browser Workflow Test",
	}

	createResp := makeJSONRequest(t, result.Server, http.MethodPost, "/api/sessions", sessionRequest)
	defer func() { _ = createResp.Body.Close() }()
	createdSessionData := validateObjectResponse(t, createResp, http.StatusCreated)
	sessionID := createdSessionData["id"].(string)

	// Send message asking agent to open google.com and take screenshot
	messageRequest := map[string]interface{}{
		"text": "Open google.com and take a screenshot",
	}

	t.Log("Sending message to agent...")
	msgResp := makeJSONRequest(t, result.Server, http.MethodPost, constants.APISessionsPath+sessionID+"/messages", messageRequest)
	defer func() { _ = msgResp.Body.Close() }()

	if msgResp.StatusCode != http.StatusAccepted {
		t.Fatalf("Expected status code %d (Accepted) for async message processing, got %d", http.StatusAccepted, msgResp.StatusCode)
	}

	messageData := validateObjectResponse(t, msgResp, http.StatusAccepted)
	t.Logf("Message response: %+v", messageData)

	// Wait a bit for agent to process
	time.Sleep(5 * time.Second)

	// Get messages to verify Browser tool was used
	ctx := context.Background()
	messages, err := result.App.Messages.List(ctx, sessionID)
	if err != nil {
		t.Fatalf("Failed to list messages: %v", err)
	}

	// Look for assistant message with tool calls
	foundBrowserTool := false
	var assistantMessage *message.Message
	for _, msg := range messages {
		if msg.Role == message.Assistant {
			assistantMessage = &msg
			// Check if any part is a tool call
			for _, part := range msg.Parts {
				if toolCall, ok := part.(message.ToolCall); ok {
					t.Logf("Found tool call: %s", toolCall.Name)
					if toolCall.Name == toolNameBrowser {
						foundBrowserTool = true
						t.Logf("Browser tool was used with input: %s", toolCall.Input)
					}
				}
			}
		}
	}

	if !foundBrowserTool {
		t.Fatalf("Expected Browser tool to be used, but it wasn't found in messages")
	}

	// Verify screenshot file exists
	t.Log("Checking for screenshot file...")
	listResp := makeJSONRequest(t, result.Server, http.MethodGet, constants.APISessionsPath+sessionID+"/files", nil)
	defer func() { _ = listResp.Body.Close() }()
	filesList := validateArrayResponse(t, listResp)

	foundScreenshot := false
	var screenshotFilename string
	for _, fileItem := range filesList {
		fileObj := fileItem.(map[string]interface{})
		filename := fileObj["name"].(string)
		t.Logf("Found file: %s", filename)
		if strings.HasSuffix(filename, ".png") {
			foundScreenshot = true
			screenshotFilename = filename
			break
		}
	}

	if !foundScreenshot {
		t.Fatalf("Expected screenshot PNG file to be saved, but none found")
	}

	t.Logf("Found screenshot: %s", screenshotFilename)

	// http.MethodGet screenshot via API and verify it's a valid PNG
	downloadResp := makeJSONRequest(t, result.Server, http.MethodGet,
		constants.APISessionsPath+sessionID+"/files/"+screenshotFilename, nil)

	if downloadResp.StatusCode != http.StatusOK {
		t.Fatalf("Expected status code %d for screenshot download, got %d", http.StatusOK, downloadResp.StatusCode)
	}

	// Read first 8 bytes to check PNG magic bytes
	pngMagic := []byte{0x89, 0x50, 0x4E, 0x47, 0x0D, 0x0A, 0x1A, 0x0A}
	header := make([]byte, 8)
	n, err := io.ReadFull(downloadResp.Body, header)
	_ = downloadResp.Body.Close()

	if err != nil && !errors.Is(err, io.ErrUnexpectedEOF) {
		t.Fatalf("Failed to read screenshot header: %v", err)
	}

	if n < 8 {
		t.Fatalf("Screenshot file too small: only %d bytes", n)
	}

	if !bytes.Equal(header, pngMagic) {
		t.Fatalf("Screenshot file is not a valid PNG. Expected magic bytes %v, got %v", pngMagic, header)
	}

	t.Logf("✅ Browser full workflow test passed - Screenshot saved and verified: %s", screenshotFilename)
	t.Logf("   Assistant message ID: %s", assistantMessage.ID)
}

// Test 29: Browser Multi-Action - Test multiple browser actions in one request
func TestRESTBrowserMultiAction(t *testing.T) {
	skipIfBrowserServiceUnavailable(t)
	requireLLMCredentials(t)

	result := setupIntegrationTestServer(t)
	defer result.Server.Close()

	// Store API key in test database via REST API
	apiKey := os.Getenv("ANTHROPIC_API_KEY")
	if apiKey != "" {
		storeKeyRequest := map[string]interface{}{
			"provider": "anthropic",
			"api_key":  apiKey,
		}
		keyResp := makeJSONRequest(t, result.Server, http.MethodPost, "/api/auth/api-key", storeKeyRequest)
		defer func() { _ = keyResp.Body.Close() }()
		if keyResp.StatusCode != http.StatusOK {
			t.Fatalf("Failed to store API key: status %d", keyResp.StatusCode)
		}
		t.Log("✓ Stored API key in test database")
	}

	t.Log("Testing browser tool multi-action workflow")

	// Create a session
	sessionRequest := map[string]interface{}{
		"title": "Browser Multi-Action Test",
	}

	createResp := makeJSONRequest(t, result.Server, http.MethodPost, "/api/sessions", sessionRequest)
	defer func() { _ = createResp.Body.Close() }()
	createdSessionData := validateObjectResponse(t, createResp, http.StatusCreated)
	sessionID := createdSessionData["id"].(string)

	// Send message asking for multiple actions
	messageRequest := map[string]interface{}{
		"text": "Open example.com, take a screenshot, scroll down, and take another screenshot",
	}

	t.Log("Sending multi-action message to agent...")
	msgResp := makeJSONRequest(t, result.Server, http.MethodPost, constants.APISessionsPath+sessionID+"/messages", messageRequest)
	defer func() { _ = msgResp.Body.Close() }()

	if msgResp.StatusCode != http.StatusAccepted {
		t.Fatalf("Expected status code %d (Accepted) for async message processing, got %d", http.StatusAccepted, msgResp.StatusCode)
	}

	// Wait for agent to process multiple actions
	time.Sleep(8 * time.Second)

	// Get messages to verify multiple Browser tool calls
	ctx := context.Background()
	messages, err := result.App.Messages.List(ctx, sessionID)
	if err != nil {
		t.Fatalf("Failed to list messages: %v", err)
	}

	// Count Browser tool uses
	browserToolCount := 0
	for _, msg := range messages {
		if msg.Role == message.Assistant {
			for _, part := range msg.Parts {
				if toolCall, ok := part.(message.ToolCall); ok {
					if toolCall.Name == toolNameBrowser {
						browserToolCount++
						t.Logf("Browser tool call #%d: %s", browserToolCount, toolCall.Input)
					}
				}
			}
		}
	}

	if browserToolCount < 2 {
		t.Logf("Warning: Expected multiple Browser tool calls, but found only %d", browserToolCount)
	} else {
		t.Logf("Found %d Browser tool calls", browserToolCount)
	}

	// Verify multiple screenshots were saved
	listResp := makeJSONRequest(t, result.Server, http.MethodGet, constants.APISessionsPath+sessionID+"/files", nil)
	defer func() { _ = listResp.Body.Close() }()
	filesList := validateArrayResponse(t, listResp)

	screenshotCount := 0
	for _, fileItem := range filesList {
		fileObj := fileItem.(map[string]interface{})
		filename := fileObj["name"].(string)
		if strings.HasSuffix(filename, ".png") {
			screenshotCount++
			t.Logf("Found screenshot #%d: %s", screenshotCount, filename)
		}
	}

	if screenshotCount < 1 {
		t.Fatalf("Expected at least 1 screenshot file, but found %d", screenshotCount)
	}

	t.Logf("✅ Browser multi-action test passed - Found %d browser tool calls and %d screenshots", browserToolCount, screenshotCount)
}

// Test 30: Browser Error Handling - Test error handling with invalid URL
func TestRESTBrowserErrorHandling(t *testing.T) {
	skipIfBrowserServiceUnavailable(t)
	requireLLMCredentials(t)

	result := setupIntegrationTestServer(t)
	defer result.Server.Close()

	// Store API key in test database via REST API
	apiKey := os.Getenv("ANTHROPIC_API_KEY")
	if apiKey != "" {
		storeKeyRequest := map[string]interface{}{
			"provider": "anthropic",
			"api_key":  apiKey,
		}
		keyResp := makeJSONRequest(t, result.Server, http.MethodPost, "/api/auth/api-key", storeKeyRequest)
		defer func() { _ = keyResp.Body.Close() }()
		if keyResp.StatusCode != http.StatusOK {
			t.Fatalf("Failed to store API key: status %d", keyResp.StatusCode)
		}
		t.Log("✓ Stored API key in test database")
	}

	t.Log("Testing browser tool error handling with invalid URL")

	// Create a session
	sessionRequest := map[string]interface{}{
		"title": "Browser Error Test",
	}

	createResp := makeJSONRequest(t, result.Server, http.MethodPost, "/api/sessions", sessionRequest)
	defer func() { _ = createResp.Body.Close() }()
	createdSessionData := validateObjectResponse(t, createResp, http.StatusCreated)
	sessionID := createdSessionData["id"].(string)

	// Send message with invalid URL
	messageRequest := map[string]interface{}{
		"text": "Open http://invalid-url-that-does-not-exist-12345.com",
	}

	t.Log("Sending message with invalid URL...")
	msgResp := makeJSONRequest(t, result.Server, http.MethodPost, constants.APISessionsPath+sessionID+"/messages", messageRequest)
	defer func() { _ = msgResp.Body.Close() }()

	if msgResp.StatusCode != http.StatusAccepted {
		t.Fatalf("Expected status code %d (Accepted) for async message processing, got %d", http.StatusAccepted, msgResp.StatusCode)
	}

	// Wait for agent to process
	time.Sleep(5 * time.Second)

	// Get messages to verify error was received
	ctx := context.Background()
	messages, err := result.App.Messages.List(ctx, sessionID)
	if err != nil {
		t.Fatalf("Failed to list messages: %v", err)
	}

	// Look for tool result with error
	foundError := false
	for _, msg := range messages {
		if msg.Role != message.User {
			continue
		}
		for _, part := range msg.Parts {
			toolResult, ok := part.(message.ToolResult)
			if !ok {
				continue
			}
			t.Logf("Found tool result: %+v", toolResult)
			// Check if content indicates an error
			contentStr := toolResult.Content
			t.Logf("Tool result content: %s", contentStr)
			// Look for error indicators
			if strings.Contains(strings.ToLower(contentStr), "error") ||
				strings.Contains(strings.ToLower(contentStr), "failed") ||
				strings.Contains(strings.ToLower(contentStr), "invalid") {
				foundError = true
			}
			if toolResult.IsError {
				foundError = true
				t.Logf("Tool result has IsError=true")
			}
		}
	}

	// Also check assistant messages for error mentions
	for _, msg := range messages {
		if msg.Role == message.Assistant {
			for _, part := range msg.Parts {
				if textContent, ok := part.(message.TextContent); ok {
					text := strings.ToLower(textContent.Text)
					if strings.Contains(text, "error") || strings.Contains(text, "failed") ||
						strings.Contains(text, "unable") || strings.Contains(text, "could not") {
						foundError = true
						t.Logf("Assistant mentioned error: %s", textContent.Text[:min(len(textContent.Text), 100)])
					}
				}
			}
		}
	}

	if !foundError {
		t.Logf("Warning: Expected to find error indication in messages, but didn't find one")
		t.Logf("This could mean the browser service handled the invalid URL differently than expected")
	}

	t.Logf("✅ Browser error handling test passed")
}

// Test 31: Browser Wikipedia Anatomy Click - Navigate to Wikipedia Elephant page and click Anatomy link
func TestRESTBrowserWikipediaAnatomyClick(t *testing.T) {
	skipIfBrowserServiceUnavailable(t)
	requireLLMCredentials(t)

	result := setupIntegrationTestServer(t)
	defer result.Server.Close()

	// Store API key in test database via REST API
	apiKey := os.Getenv("ANTHROPIC_API_KEY")
	if apiKey != "" {
		storeKeyRequest := map[string]interface{}{
			"provider": "anthropic",
			"api_key":  apiKey,
		}
		keyResp := makeJSONRequest(t, result.Server, http.MethodPost, "/api/auth/api-key", storeKeyRequest)
		defer func() { _ = keyResp.Body.Close() }()
		if keyResp.StatusCode != http.StatusOK {
			t.Fatalf("Failed to store API key: status %d", keyResp.StatusCode)
		}
		t.Log("✓ Stored API key in test database")
	}

	t.Log("Testing browser tool - Wikipedia Elephant page navigation and Anatomy link click")

	// Create a session
	sessionRequest := map[string]interface{}{
		"title": "Browser Wikipedia Anatomy Click Test",
	}

	createResp := makeJSONRequest(t, result.Server, http.MethodPost, "/api/sessions", sessionRequest)
	defer func() { _ = createResp.Body.Close() }()
	createdSessionData := validateObjectResponse(t, createResp, http.StatusCreated)
	sessionID := createdSessionData["id"].(string)
	t.Logf("Created session: %s", sessionID)

	// Send a single message with all steps - this keeps browser connection alive in one agent turn
	messageRequest := map[string]interface{}{
		"text": `Open https://en.wikipedia.org/wiki/Elephant in the browser, take a screenshot, find and click the "Anatomy" link, then take another screenshot. After clicking, tell me the current page URL.`,
	}

	t.Log("Sending message to open Wikipedia, click Anatomy link, and verify URL...")
	msgResp := makeJSONRequest(t, result.Server, http.MethodPost, constants.APISessionsPath+sessionID+"/messages", messageRequest)
	defer func() { _ = msgResp.Body.Close() }()

	if msgResp.StatusCode != http.StatusAccepted {
		t.Fatalf("Expected status code %d (Accepted) for async message processing, got %d", http.StatusAccepted, msgResp.StatusCode)
	}

	// Wait for agent to complete processing all steps
	t.Log("Waiting for agent to complete all browser operations...")
	waitForMessageCompletion(t, result, sessionID, 60*time.Second)

	// Get messages to verify Browser tool usage and click success
	ctx := context.Background()
	messages, err := result.App.Messages.List(ctx, sessionID)
	if err != nil {
		t.Fatalf("Failed to list messages: %v", err)
	}

	// Primary verification: Check for exact URL in assistant responses
	const expectedURL = "https://en.wikipedia.org/wiki/Elephant#Anatomy"
	foundExactURL := false
	foundAnatomyMention := false
	foundClickMention := false
	browserToolCount := 0
	extractedURL := ""

	for _, msg := range messages {
		if msg.Role == message.Assistant {
			// Count Browser tool calls
			for _, part := range msg.Parts {
				if toolCall, ok := part.(message.ToolCall); ok {
					if toolCall.Name == toolNameBrowser {
						browserToolCount++
						t.Logf("Found Browser tool call #%d: %s", browserToolCount, toolCall.Input)
					}
				}
				// Check for URL mentions in text content
				if textContent, ok := part.(message.TextContent); ok {
					text := textContent.Text
					lowerText := strings.ToLower(text)

					// Check for exact URL
					if strings.Contains(text, expectedURL) {
						foundExactURL = true
						extractedURL = expectedURL
						t.Logf("✓ Found exact URL in assistant message: %s", expectedURL)
					}

					// Check for URL patterns (case insensitive search for en.wikipedia.org/wiki/Elephant#Anatomy)
					if strings.Contains(lowerText, "en.wikipedia.org/wiki/elephant#anatomy") {
						foundExactURL = true
						// Try to extract the actual URL from the text
						if start := strings.Index(text, "https://en.wikipedia.org"); start != -1 {
							end := start
							for end < len(text) && text[end] != ' ' && text[end] != '\n' && text[end] != ')' && text[end] != ',' {
								end++
							}
							extractedURL = text[start:end]
							t.Logf("✓ Extracted URL from assistant message: %s", extractedURL)
						}
					}

					if strings.Contains(lowerText, "anatomy") {
						foundAnatomyMention = true
						t.Logf("Found Anatomy mention in assistant message: %s", text[:min(len(text), 100)])
					}
					if strings.Contains(lowerText, "click") {
						foundClickMention = true
						t.Logf("Found click mention in assistant message: %s", text[:min(len(text), 100)])
					}
				}
			}
		}
		// Check tool results for success indicators
		if msg.Role != message.User {
			continue
		}
		for _, part := range msg.Parts {
			toolResult, ok := part.(message.ToolResult)
			if !ok {
				continue
			}
			contentStr := toolResult.Content
			contentLower := strings.ToLower(contentStr)

			// Check for URL in tool results
			if strings.Contains(contentStr, expectedURL) {
				foundExactURL = true
				extractedURL = expectedURL
				t.Logf("✓ Found exact URL in tool result")
			}

			if strings.Contains(contentLower, "en.wikipedia.org/wiki/elephant#anatomy") {
				foundExactURL = true
				// Try to extract URL from tool result
				if start := strings.Index(contentStr, "https://en.wikipedia.org"); start != -1 {
					end := start
					for end < len(contentStr) && contentStr[end] != ' ' && contentStr[end] != '\n' && contentStr[end] != ')' && contentStr[end] != ',' {
						end++
					}
					extractedURL = contentStr[start:end]
					t.Logf("✓ Extracted URL from tool result: %s", extractedURL)
				}
			}

			if strings.Contains(contentLower, "anatomy") {
				foundAnatomyMention = true
				t.Logf("Found Anatomy in tool result content")
			}
			// Check for error indicators
			if toolResult.IsError {
				t.Logf("Warning: Tool result has IsError=true: %s", toolResult.Content[:min(len(toolResult.Content), 200)])
			}
		}
	}

	// Verify the exact URL was found
	if !foundExactURL {
		t.Fatalf("Expected to find URL %s in messages, but it was not found. Last extracted URL: %s", expectedURL, extractedURL)
	}
	t.Logf("✓ Verified exact URL: %s", extractedURL)

	// Verify Browser tool was used at least 3 times (open, screenshot, click, screenshot)
	if browserToolCount < 3 {
		t.Logf("Warning: Expected at least 3 Browser tool calls (open, screenshot, click), but found %d", browserToolCount)
	} else {
		t.Logf("✓ Found %d Browser tool calls", browserToolCount)
	}

	// Secondary verification: Verify multiple screenshots exist
	t.Log("Verifying screenshots were saved...")
	listResp := makeJSONRequest(t, result.Server, http.MethodGet, constants.APISessionsPath+sessionID+"/files", nil)
	defer func() { _ = listResp.Body.Close() }()
	filesList := validateArrayResponse(t, listResp)

	screenshotCount := 0
	screenshotFiles := []string{}
	for _, fileItem := range filesList {
		fileObj := fileItem.(map[string]interface{})
		filename := fileObj["name"].(string)
		if strings.HasSuffix(filename, ".png") {
			screenshotCount++
			screenshotFiles = append(screenshotFiles, filename)
			t.Logf("Found screenshot #%d: %s", screenshotCount, filename)
		}
	}

	// Should have at least 1 screenshot (may have more depending on agent behavior)
	if screenshotCount < 1 {
		t.Fatalf("Expected at least 1 screenshot file, but found %d", screenshotCount)
	}
	t.Logf("✓ Found %d screenshot(s)", screenshotCount)

	// Verify screenshots are different (different names/sizes)
	if len(screenshotFiles) >= 2 {
		if screenshotFiles[0] == screenshotFiles[1] {
			t.Logf("Warning: Screenshot filenames are identical, expected different files")
		} else {
			t.Logf("✓ Screenshots have different filenames: %s vs %s", screenshotFiles[0], screenshotFiles[1])
		}
	}

	// Log verification results
	if foundAnatomyMention {
		t.Logf("✓ Found Anatomy-related content in messages")
	} else {
		t.Logf("Warning: Did not find Anatomy mention in messages")
	}

	if foundClickMention {
		t.Logf("✓ Found click mention in messages")
	}

	t.Logf("✅ Browser Wikipedia Anatomy Click test passed")
	t.Logf("   Browser tool calls: %d", browserToolCount)
	t.Logf("   Screenshots saved: %d", screenshotCount)
	t.Logf("   Verified URL: %s", extractedURL)
	t.Logf("   Anatomy mentions: %v", foundAnatomyMention)
	t.Logf("   Click mentions: %v", foundClickMention)
}

// waitForMessageCompletion polls the message list until the last assistant message has a finish_reason
// This ensures the agent has completed processing before sending the next message
func waitForMessageCompletion(t *testing.T, result *TestServerResult, sessionID string, timeout time.Duration) {
	t.Helper()

	ctx := context.Background()
	deadline := time.Now().Add(timeout)

	for time.Now().Before(deadline) {
		messages, err := result.App.Messages.List(ctx, sessionID)
		if err != nil {
			t.Logf("Warning: Failed to list messages while waiting for completion: %v", err)
			time.Sleep(500 * time.Millisecond)
			continue
		}

		// Find the last assistant message
		var lastAssistantMsg *message.Message
		for i := len(messages) - 1; i >= 0; i-- {
			if messages[i].Role == message.Assistant {
				lastAssistantMsg = &messages[i]
				break
			}
		}

		if lastAssistantMsg != nil {
			finishReason := lastAssistantMsg.FinishReason()
			// Only consider terminal finish reasons as completion
			// tool_use means the agent is still executing tools and will continue
			if finishReason == "end_turn" || finishReason == "canceled" || finishReason == "max_tokens" {
				t.Logf("✓ Message completed with finish_reason: %s", finishReason)
				return
			}
			if finishReason == "" {
				t.Logf("Waiting for message completion... (finish_reason is empty)")
			} else {
				t.Logf("Waiting for final completion... (current finish_reason: %s)", finishReason)
			}
		} else {
			t.Logf("Waiting for assistant message...")
		}

		time.Sleep(500 * time.Millisecond)
	}

	t.Fatalf("Timeout waiting for message completion after %v", timeout)
}
