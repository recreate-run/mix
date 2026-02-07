//go:build e2e
// +build e2e

package messaging_test

import (
	"net/http"
	"testing"

	"mix/e2e"
	"mix/internal/http/integration_tests"
)

// TestRESTMessageSending tests the complete end-to-end flow of sending a message and receiving an LLM response
// This E2E test requires a real LLM API key and validates the entire message processing pipeline
func TestRESTMessageSending(t *testing.T) {
	e2e.Setup(t)

	result := integration_tests.SetupIntegrationTestServer(t)
	defer result.Server.Close()

	t.Log("Testing POST /api/sessions/{id}/messages - Send message")

	// Create a session first
	sessionRequest := map[string]interface{}{
		"title": "Message Test Session",
	}

	createResp := integration_tests.MakeJSONRequest(t, result.Server, "POST", "/api/sessions", sessionRequest)
	defer func() { _ = createResp.Body.Close() }()
	createdSessionData := integration_tests.ValidateObjectResponse(t, createResp, http.StatusCreated)

	sessionID := createdSessionData["id"].(string)

	// Send a message to the session
	messageRequest := map[string]interface{}{
		"text": "Hello, this is a test message for integration testing",
	}

	// Make the request
	msgResp := integration_tests.MakeJSONRequest(t, result.Server, "POST", "/api/sessions/"+sessionID+"/messages", messageRequest)
	defer func() { _ = msgResp.Body.Close() }()

	// In an unauthenticated test environment, we expect a 401 Unauthorized
	if msgResp.StatusCode == http.StatusUnauthorized {
		t.Skip("Skipping test - authentication error (no API credentials). This is expected in test environments.")
	}

	// With valid credentials, we should get a 202 Accepted
	if msgResp.StatusCode != http.StatusAccepted {
		t.Fatalf("Expected status code %d, got %d", http.StatusAccepted, msgResp.StatusCode)
	}

	// Parse and verify response structure
	responseData := integration_tests.ValidateObjectResponse(t, msgResp, http.StatusAccepted)

	t.Logf("Message response data: %+v", responseData)

	// The response should have status and sessionId
	status, ok := responseData["status"].(string)
	if !ok || status != "processing" {
		t.Fatalf("Expected status 'processing' in response, got %v", responseData["status"])
	}

	responseSessionID, ok := responseData["sessionId"].(string)
	if !ok || responseSessionID != sessionID {
		t.Fatalf("Expected sessionId '%s' in response, got %v", sessionID, responseData["sessionId"])
	}

	t.Logf("✅ Message sending test passed - Message accepted for processing in session: %s", sessionID)
}
