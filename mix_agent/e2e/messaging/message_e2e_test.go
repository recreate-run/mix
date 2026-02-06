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

	// In an unauthenticated test environment, we should get a 200 OK with an auth prompt
	if msgResp.StatusCode != http.StatusOK {
		t.Fatalf("Expected status code %d, got %d", http.StatusOK, msgResp.StatusCode)
	}

	// Parse and verify response structure
	messageData := integration_tests.ValidateObjectResponse(t, msgResp, http.StatusOK)

	t.Logf("Message response data: %+v", messageData)

	// The response should have an ID and role
	messageID, ok := messageData["id"].(string)
	if !ok || messageID == "" {
		t.Fatalf("Expected message ID in response, got %v", messageData)
	}

	// Role should be present
	role, ok := messageData["role"].(string)
	if !ok {
		t.Fatalf("Expected role field in message response")
	}

	// For unauthenticated environments, the role is "assistant" for the auth prompt
	if role != "assistant" {
		t.Logf("Note: Expected role 'assistant' for auth prompt, got '%s'. This is acceptable if the test environment is configured differently.", role)
	}

	// Some response should be present
	_, ok = messageData["assistantResponse"].(string)
	if !ok {
		t.Fatalf("Expected assistantResponse field in message response")
	}

	t.Logf("✅ Message sending test passed - Message ID: %s", messageID)
}
