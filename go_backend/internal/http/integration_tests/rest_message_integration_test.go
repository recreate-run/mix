package integration_tests

import (
	"context"
	"net/http"
	"testing"

	"mix/internal/message"
)

// Test 5: Message Sending - POST /api/sessions/{id}/messages
func TestRESTMessageSending(t *testing.T) {
	result := setupIntegrationTestServer(t)
	defer result.Server.Close()

	t.Log("Testing POST /api/sessions/{id}/messages - Send message")

	// Create a session first
	sessionRequest := map[string]interface{}{
		"title": "Message Test Session",
	}

	createResp := makeJSONRequest(t, result.Server, "POST", "/api/sessions", sessionRequest)
	createdSessionData := validateObjectResponse(t, createResp, http.StatusCreated)

	sessionID := createdSessionData["id"].(string)

	// Send a message to the session
	messageRequest := map[string]interface{}{
		"content": "Hello, this is a test message for integration testing",
	}

	msgResp := makeJSONRequest(t, result.Server, "POST", "/api/sessions/"+sessionID+"/messages", messageRequest)
	messageData := validateObjectResponse(t, msgResp, http.StatusOK)

	messageID, ok := messageData["id"].(string)
	if !ok || messageID == "" {
		t.Fatalf("Expected message ID in response, got %v", messageID)
	}

	// Check user input content (for user messages)
	userInput, ok := messageData["userInput"].(string)
	if !ok || userInput != "Hello, this is a test message for integration testing" {
		t.Fatalf("Expected message userInput to match, got %v", userInput)
	}

	role, ok := messageData["role"].(string)
	if !ok || role != "user" {
		t.Fatalf("Expected message role 'user', got %v", role)
	}

	t.Logf("✅ Message sending test passed - Message ID: %s", messageID)
}

// Test 6: Message Listing - GET /api/sessions/{id}/messages
func TestRESTMessageListing(t *testing.T) {
	result := setupIntegrationTestServer(t)
	defer result.Server.Close()

	t.Log("Testing GET /api/sessions/{id}/messages - List session messages")

	// Create a session and add some messages
	sessionRequest := map[string]interface{}{
		"title": "Message List Test Session",
	}

	createResp := makeJSONRequest(t, result.Server, "POST", "/api/sessions", sessionRequest)
	createdSessionData := validateObjectResponse(t, createResp, http.StatusCreated)

	sessionID := createdSessionData["id"].(string)

	// Add a test message directly to database (simpler than going through agent)
	ctx := context.Background()
	testMsg, err := result.App.Messages.Create(ctx, sessionID, message.CreateMessageParams{
		Role: message.User,
		Parts: []message.ContentPart{
			message.TextContent{Text: "Test message for listing"},
		},
		Model: "claude-4-sonnet",
	})
	if err != nil {
		t.Fatalf("Failed to create test message: %v", err)
	}

	// List messages for the session
	listResp := makeJSONRequest(t, result.Server, "GET", "/api/sessions/"+sessionID+"/messages", nil)
	messagesList := validateArrayResponse(t, listResp, http.StatusOK)

	if len(messagesList) == 0 {
		t.Fatalf("Expected at least one message in list, got 0")
	}

	// Validate the message we created is in the list
	found := false
	for _, msgItem := range messagesList {
		msgObj, ok := msgItem.(map[string]interface{})
		if !ok {
			continue
		}
		if msgObj["id"].(string) == testMsg.ID {
			found = true
			// Check userInput exists and matches (since test message is from user)
			if userInput, ok := msgObj["userInput"].(string); !ok {
				t.Fatalf("Expected message userInput to be string, got %T", msgObj["userInput"])
			} else if userInput != "Test message for listing" {
				t.Fatalf("Expected message userInput 'Test message for listing', got %q", userInput)
			}
			break
		}
	}

	if !found {
		t.Fatalf("Test message %s not found in messages list", testMsg.ID)
	}

	t.Logf("✅ Message listing test passed - Found %d messages", len(messagesList))
}

// Test 11: Message History - GET /api/messages/history
func TestRESTMessageHistory(t *testing.T) {
	result := setupIntegrationTestServer(t)
	defer result.Server.Close()

	t.Log("Testing GET /api/messages/history - Get global message history")

	// Create multiple sessions with messages
	session1Request := map[string]interface{}{
		"title": "History Test Session 1",
	}

	session1Resp := makeJSONRequest(t, result.Server, "POST", "/api/sessions", session1Request)
	session1Data := validateObjectResponse(t, session1Resp, http.StatusCreated)
	session1ID := session1Data["id"].(string)

	session2Request := map[string]interface{}{
		"title": "History Test Session 2",
	}

	session2Resp := makeJSONRequest(t, result.Server, "POST", "/api/sessions", session2Request)
	session2Data := validateObjectResponse(t, session2Resp, http.StatusCreated)
	session2ID := session2Data["id"].(string)

	// Add test messages to both sessions
	ctx := context.Background()
	testMsg1, err := result.App.Messages.Create(ctx, session1ID, message.CreateMessageParams{
		Role: message.User,
		Parts: []message.ContentPart{
			message.TextContent{Text: "Message in session 1"},
		},
		Model: "claude-4-sonnet",
	})
	if err != nil {
		t.Fatalf("Failed to create test message in session 1: %v", err)
	}

	testMsg2, err := result.App.Messages.Create(ctx, session2ID, message.CreateMessageParams{
		Role: message.User,
		Parts: []message.ContentPart{
			message.TextContent{Text: "Message in session 2"},
		},
		Model: "claude-4-sonnet",
	})
	if err != nil {
		t.Fatalf("Failed to create test message in session 2: %v", err)
	}

	// Test message history with default pagination
	historyResp := makeJSONRequest(t, result.Server, "GET", "/api/messages/history", nil)
	messagesList := validateArrayResponse(t, historyResp, http.StatusOK)

	if len(messagesList) == 0 {
		t.Fatalf("Expected at least some messages in history, got 0")
	}

	// Validate message structure
	found1, found2 := false, false
	for _, msgItem := range messagesList {
		msgObj, ok := msgItem.(map[string]interface{})
		if !ok {
			continue
		}

		msgID, ok := msgObj["id"].(string)
		if !ok {
			t.Fatalf("Expected message ID to be string, got %T", msgObj["id"])
		}

		if msgID == testMsg1.ID {
			found1 = true
			if userInput, ok := msgObj["userInput"].(string); !ok || userInput != "Message in session 1" {
				t.Fatalf("Expected message userInput 'Message in session 1', got %v", msgObj["userInput"])
			}
		}
		if msgID == testMsg2.ID {
			found2 = true
			if userInput, ok := msgObj["userInput"].(string); !ok || userInput != "Message in session 2" {
				t.Fatalf("Expected message userInput 'Message in session 2', got %v", msgObj["userInput"])
			}
		}
	}

	if !found1 {
		t.Fatalf("Test message 1 %s not found in global history", testMsg1.ID)
	}
	if !found2 {
		t.Fatalf("Test message 2 %s not found in global history", testMsg2.ID)
	}

	// Test pagination with limit
	paginatedResp := makeJSONRequest(t, result.Server, "GET", "/api/messages/history?limit=1&offset=0", nil)
	paginatedList := validateArrayResponse(t, paginatedResp, http.StatusOK)

	if len(paginatedList) > 1 {
		t.Fatalf("Expected at most 1 message with limit=1, got %d", len(paginatedList))
	}

	t.Logf("✅ Message history test passed - Found %d messages total, pagination working", len(messagesList))
}