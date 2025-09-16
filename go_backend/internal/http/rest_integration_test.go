package http

import (
	"context"
	"net/http"
	"testing"

	"mix/internal/message"
)


// Test 2: Session Creation - POST /api/sessions
func TestRESTSessionCreation(t *testing.T) {
	result := setupIntegrationTestServer(t)
	defer result.Server.Close()

	t.Log("Testing POST /api/sessions - Create session")

	// Create session request
	sessionRequest := map[string]interface{}{
		"title": "Integration Test Session",
	}

	resp := makeJSONRequest(t, result.Server, "POST", "/api/sessions", sessionRequest)
	sessionData := validateObjectResponse(t, resp, http.StatusCreated)

	sessionID, ok := sessionData["id"].(string)
	if !ok || sessionID == "" {
		t.Fatalf("Expected session ID in response, got %v", sessionID)
	}

	title, ok := sessionData["title"].(string)
	if !ok || title != "Integration Test Session" {
		t.Fatalf("Expected title 'Integration Test Session', got %v", title)
	}

	// Note: Working directory removed from session data - using centralized storage

	t.Logf("✅ Session creation test passed - Session ID: %s", sessionID)
}

// Test 3: Session Listing - GET /api/sessions
func TestRESTSessionListing(t *testing.T) {
	result := setupIntegrationTestServer(t)
	defer result.Server.Close()

	t.Log("Testing GET /api/sessions - List sessions")

	// First create a session to list
	sessionRequest := map[string]interface{}{
		"title": "List Test Session",
	}

	createResp := makeJSONRequest(t, result.Server, "POST", "/api/sessions", sessionRequest)
	createdSessionData := validateObjectResponse(t, createResp, http.StatusCreated)

	createdSessionID := createdSessionData["id"].(string)

	// Now list sessions
	listResp := makeJSONRequest(t, result.Server, "GET", "/api/sessions", nil)
	sessionsArray := validateArrayResponse(t, listResp, http.StatusOK)

	// Find our created session in the list
	found := false
	for _, sessionItem := range sessionsArray {
		sessionObj, ok := sessionItem.(map[string]interface{})
		if !ok {
			continue
		}
		if sessionObj["id"].(string) == createdSessionID {
			found = true
			if sessionObj["title"].(string) != "List Test Session" {
				t.Fatalf("Expected session title 'List Test Session', got %v", sessionObj["title"])
			}
			break
		}
	}

	if !found {
		t.Fatalf("Created session %s not found in sessions list", createdSessionID)
	}

	t.Logf("✅ Session listing test passed - Found %d sessions", len(sessionsArray))
}

// Test 4: Session Retrieval - GET /api/sessions/{id}
func TestRESTSessionRetrieval(t *testing.T) {
	result := setupIntegrationTestServer(t)
	defer result.Server.Close()

	t.Log("Testing GET /api/sessions/{id} - Get specific session")

	// Create a session first
	sessionRequest := map[string]interface{}{
		"title": "Retrieval Test Session",
	}

	createResp := makeJSONRequest(t, result.Server, "POST", "/api/sessions", sessionRequest)
	createdSessionData := validateObjectResponse(t, createResp, http.StatusCreated)

	sessionID := createdSessionData["id"].(string)

	// Retrieve the specific session
	getResp := makeJSONRequest(t, result.Server, "GET", "/api/sessions/"+sessionID, nil)
	retrievedSession := validateObjectResponse(t, getResp, http.StatusOK)

	retrievedID, ok := retrievedSession["id"].(string)
	if !ok || retrievedID != sessionID {
		t.Fatalf("Expected session ID %s, got %v", sessionID, retrievedID)
	}

	retrievedTitle, ok := retrievedSession["title"].(string)
	if !ok || retrievedTitle != "Retrieval Test Session" {
		t.Fatalf("Expected title 'Retrieval Test Session', got %v", retrievedTitle)
	}

	t.Logf("✅ Session retrieval test passed - Retrieved session: %s", retrievedID)
}

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

// Test 7: Commands Listing - GET /api/commands
func TestRESTCommandsListing(t *testing.T) {
	result := setupIntegrationTestServer(t)
	defer result.Server.Close()

	t.Log("Testing GET /api/commands - List available commands")

	// List available commands
	listResp := makeJSONRequest(t, result.Server, "GET", "/api/commands", nil)
	commandsList := validateArrayResponse(t, listResp, http.StatusOK)

	if len(commandsList) == 0 {
		t.Fatalf("Expected at least one command in list, got 0")
	}

	// Validate command structure
	for _, cmdItem := range commandsList {
		cmdObj, ok := cmdItem.(map[string]interface{})
		if !ok {
			t.Fatalf("Expected command to be object, got %T", cmdItem)
		}

		name, ok := cmdObj["name"].(string)
		if !ok || name == "" {
			t.Fatalf("Expected command name to be non-empty string, got %v", name)
		}

		_, ok = cmdObj["description"].(string)
		if !ok {
			t.Fatalf("Expected command description to be string, got %T", cmdObj["description"])
		}

		cmdType, ok := cmdObj["type"].(string)
		if !ok || (cmdType != "builtin" && cmdType != "file") {
			t.Fatalf("Expected command type to be 'builtin' or 'file', got %v", cmdType)
		}
	}

	t.Logf("✅ Commands listing test passed - Found %d commands", len(commandsList))
}

// Test 8: Session Deletion - DELETE /api/sessions/{id}
func TestRESTSessionDeletion(t *testing.T) {
	result := setupIntegrationTestServer(t)
	defer result.Server.Close()

	t.Log("Testing DELETE /api/sessions/{id} - Delete session")

	// Create a session to delete
	sessionRequest := map[string]interface{}{
		"title": "Deletion Test Session",
	}

	createResp := makeJSONRequest(t, result.Server, "POST", "/api/sessions", sessionRequest)
	createdSessionData := validateObjectResponse(t, createResp, http.StatusCreated)

	sessionID := createdSessionData["id"].(string)

	// Delete the session
	deleteResp := makeJSONRequest(t, result.Server, "DELETE", "/api/sessions/"+sessionID, nil)
	if deleteResp.StatusCode != http.StatusNoContent {
		t.Fatalf("Expected status code %d for deletion, got %d", http.StatusNoContent, deleteResp.StatusCode)
	}

	// Verify the session is gone - should return 404
	getResp := makeJSONRequest(t, result.Server, "GET", "/api/sessions/"+sessionID, nil)
	if getResp.StatusCode != http.StatusNotFound {
		t.Fatalf("Expected status code %d when retrieving deleted session, got %d", http.StatusNotFound, getResp.StatusCode)
	}

	// Test deleting non-existent session - should return 500 (internal error from business logic)
	nonExistentID := "non-existent-session-id"
	deleteNonExistentResp := makeJSONRequest(t, result.Server, "DELETE", "/api/sessions/"+nonExistentID, nil)
	if deleteNonExistentResp.StatusCode != http.StatusInternalServerError {
		t.Fatalf("Expected status code %d when deleting non-existent session, got %d", http.StatusInternalServerError, deleteNonExistentResp.StatusCode)
	}

	t.Logf("✅ Session deletion test passed - Deleted session: %s", sessionID)
}

// Test 9: Session Forking - POST /api/sessions/{id}/fork
func TestRESTSessionForking(t *testing.T) {
	result := setupIntegrationTestServer(t)
	defer result.Server.Close()

	t.Log("Testing POST /api/sessions/{id}/fork - Fork session")

	// Create a source session
	sessionRequest := map[string]interface{}{
		"title": "Source Session for Forking",
	}

	createResp := makeJSONRequest(t, result.Server, "POST", "/api/sessions", sessionRequest)
	createdSessionData := validateObjectResponse(t, createResp, http.StatusCreated)

	sourceSessionID := createdSessionData["id"].(string)

	// Add a test message to the source session directly to database
	ctx := context.Background()
	_, err := result.App.Messages.Create(ctx, sourceSessionID, message.CreateMessageParams{
		Role: message.User,
		Parts: []message.ContentPart{
			message.TextContent{Text: "Test message in source session"},
		},
		Model: "claude-4-sonnet",
	})
	if err != nil {
		t.Fatalf("Failed to create test message in source session: %v", err)
	}

	// Fork the session
	forkRequest := map[string]interface{}{
		"messageIndex": int64(1),
		"title":        "Forked Session",
	}

	forkResp := makeJSONRequest(t, result.Server, "POST", "/api/sessions/"+sourceSessionID+"/fork", forkRequest)
	forkedSessionData := validateObjectResponse(t, forkResp, http.StatusCreated)

	forkedSessionID, ok := forkedSessionData["id"].(string)
	if !ok || forkedSessionID == "" {
		t.Fatalf("Expected forked session ID in response, got %v", forkedSessionID)
	}

	if forkedSessionID == sourceSessionID {
		t.Fatalf("Expected forked session to have different ID from source session")
	}

	forkedTitle, ok := forkedSessionData["title"].(string)
	if !ok || forkedTitle != "Forked Session" {
		t.Fatalf("Expected forked session title 'Forked Session', got %v", forkedTitle)
	}

	// Verify both sessions exist independently
	sourceGetResp := makeJSONRequest(t, result.Server, "GET", "/api/sessions/"+sourceSessionID, nil)
	validateObjectResponse(t, sourceGetResp, http.StatusOK)

	forkedGetResp := makeJSONRequest(t, result.Server, "GET", "/api/sessions/"+forkedSessionID, nil)
	validateObjectResponse(t, forkedGetResp, http.StatusOK)

	t.Logf("✅ Session forking test passed - Source: %s, Forked: %s", sourceSessionID, forkedSessionID)
}

// Test 10: Agent Cancellation - POST /api/sessions/{id}/cancel
func TestRESTAgentCancellation(t *testing.T) {
	result := setupIntegrationTestServer(t)
	defer result.Server.Close()

	t.Log("Testing POST /api/sessions/{id}/cancel - Cancel agent")

	// Create a session
	sessionRequest := map[string]interface{}{
		"title": "Cancel Test Session",
	}

	createResp := makeJSONRequest(t, result.Server, "POST", "/api/sessions", sessionRequest)
	createdSessionData := validateObjectResponse(t, createResp, http.StatusCreated)

	sessionID := createdSessionData["id"].(string)

	// Cancel agent processing for the session
	cancelResp := makeJSONRequest(t, result.Server, "POST", "/api/sessions/"+sessionID+"/cancel", nil)
	cancellationData := validateObjectResponse(t, cancelResp, http.StatusOK)

	status, ok := cancellationData["status"].(string)
	if !ok || status != "cancelled" {
		t.Fatalf("Expected cancellation status 'cancelled', got %v", status)
	}

	returnedSessionID, ok := cancellationData["sessionId"].(string)
	if !ok || returnedSessionID != sessionID {
		t.Fatalf("Expected session ID %s in cancellation response, got %v", sessionID, returnedSessionID)
	}

	t.Logf("✅ Agent cancellation test passed - Session: %s", sessionID)
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

// Test 12: MCP Servers Listing - GET /api/mcp
func TestRESTMCPServersListing(t *testing.T) {
	result := setupIntegrationTestServer(t)
	defer result.Server.Close()

	t.Log("Testing GET /api/mcp - List MCP servers")

	// List MCP servers
	listResp := makeJSONRequest(t, result.Server, "GET", "/api/mcp", nil)
	mcpServersList := validateArrayResponse(t, listResp, http.StatusOK)

	// MCP servers list can be empty, that's valid
	t.Logf("✅ MCP servers listing test passed - Found %d MCP servers", len(mcpServersList))

	// If there are MCP servers, validate their structure
	for i, serverItem := range mcpServersList {
		serverObj, ok := serverItem.(map[string]interface{})
		if !ok {
			t.Fatalf("Expected MCP server %d to be object, got %T", i, serverItem)
		}

		// Validate required fields exist
		if _, ok := serverObj["name"]; !ok {
			t.Fatalf("Expected MCP server %d to have 'name' field", i)
		}

		if _, ok := serverObj["status"]; !ok {
			t.Fatalf("Expected MCP server %d to have 'status' field", i)
		}
	}
}

// Test 13: Health Check - GET /health
func TestRESTHealthCheck(t *testing.T) {
	result := setupIntegrationTestServer(t)
	defer result.Server.Close()

	t.Log("Testing GET /health - Health check")

	// Check health endpoint
	healthResp := makeJSONRequest(t, result.Server, "GET", "/health", nil)
	healthData := validateObjectResponse(t, healthResp, http.StatusOK)

	// Validate basic health fields
	status, ok := healthData["status"].(string)
	if !ok || (status != "ok" && status != "healthy") {
		t.Fatalf("Expected health status to be 'ok' or 'healthy', got %v", status)
	}

	// Optional timestamp field validation
	if timestamp, exists := healthData["timestamp"]; exists {
		if _, ok := timestamp.(string); !ok {
			t.Fatalf("Expected health timestamp to be string, got %T", timestamp)
		}
	}

	t.Logf("✅ Health check test passed - Status: %s", status)
}

// Test 14: Stream Endpoint - GET /stream
func TestRESTStreamEndpoint(t *testing.T) {
	result := setupIntegrationTestServer(t)
	defer result.Server.Close()

	t.Log("Testing GET /stream - Stream endpoint")

	// Make request to stream endpoint
	req, err := http.NewRequest("GET", result.Server.URL+"/stream", nil)
	if err != nil {
		t.Fatalf("Failed to create stream request: %v", err)
	}

	// Accept Server-Sent Events
	req.Header.Set("Accept", "text/event-stream")
	req.Header.Set("Cache-Control", "no-cache")

	client := &http.Client{
		Timeout: 5000000000, // 5 seconds timeout for stream connection
	}
	
	resp, err := client.Do(req)
	if err != nil {
		t.Fatalf("Failed to make stream request: %v", err)
	}
	defer resp.Body.Close()

	// Validate SSE response headers
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("Expected status code %d for stream endpoint, got %d", http.StatusOK, resp.StatusCode)
	}

	contentType := resp.Header.Get("Content-Type")
	if contentType != "text/event-stream" {
		t.Fatalf("Expected Content-Type 'text/event-stream', got '%s'", contentType)
	}

	cacheControl := resp.Header.Get("Cache-Control")
	if cacheControl != "no-cache" {
		t.Fatalf("Expected Cache-Control 'no-cache', got '%s'", cacheControl)
	}

	// Read a small portion of the stream to verify it's working
	buffer := make([]byte, 100)
	n, err := resp.Body.Read(buffer)
	if err != nil && n == 0 {
		t.Fatalf("Failed to read from stream: %v", err)
	}

	// Basic validation that we got some SSE-like content
	streamContent := string(buffer[:n])
	if len(streamContent) == 0 {
		t.Fatalf("Expected some stream content, got empty response")
	}

	t.Logf("✅ Stream endpoint test passed - Content-Type: %s, Read %d bytes", contentType, n)
}

// Test 15: Stream Sub-path Endpoint - GET /stream/{path...}
func TestRESTStreamSubPathEndpoint(t *testing.T) {
	result := setupIntegrationTestServer(t)
	defer result.Server.Close()

	t.Log("Testing GET /stream/{path...} - Stream sub-path endpoint")

	// Test stream sub-path with a sample path
	testPath := "events/session-updates"
	req, err := http.NewRequest("GET", result.Server.URL+"/stream/"+testPath, nil)
	if err != nil {
		t.Fatalf("Failed to create stream sub-path request: %v", err)
	}

	// Accept Server-Sent Events
	req.Header.Set("Accept", "text/event-stream")
	req.Header.Set("Cache-Control", "no-cache")

	client := &http.Client{
		Timeout: 5000000000, // 5 seconds timeout for stream connection
	}
	
	resp, err := client.Do(req)
	if err != nil {
		t.Fatalf("Failed to make stream sub-path request: %v", err)
	}
	defer resp.Body.Close()

	// Validate SSE response
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("Expected status code %d for stream sub-path endpoint, got %d", http.StatusOK, resp.StatusCode)
	}

	contentType := resp.Header.Get("Content-Type")
	if contentType != "text/event-stream" {
		t.Fatalf("Expected Content-Type 'text/event-stream', got '%s'", contentType)
	}

	// Read a small portion to verify stream is working
	buffer := make([]byte, 100)
	n, err := resp.Body.Read(buffer)
	if err != nil && n == 0 {
		t.Fatalf("Failed to read from stream sub-path: %v", err)
	}

	streamContent := string(buffer[:n])
	if len(streamContent) == 0 {
		t.Fatalf("Expected some stream content from sub-path, got empty response")
	}

	t.Logf("✅ Stream sub-path endpoint test passed - Path: %s, Read %d bytes", testPath, n)
}

// Test 16: Permission Grant - POST /api/permissions/{id}/grant
func TestRESTPermissionGrant(t *testing.T) {
	result := setupIntegrationTestServer(t)
	defer result.Server.Close()

	t.Log("Testing POST /api/permissions/{id}/grant - Grant permission")

	// Use a test permission ID
	testPermissionID := "test-permission-grant-id"

	// Grant the permission
	grantResp := makeJSONRequest(t, result.Server, "POST", "/api/permissions/"+testPermissionID+"/grant", nil)
	grantData := validateObjectResponse(t, grantResp, http.StatusOK)

	status, ok := grantData["status"].(string)
	if !ok || status != "granted" {
		t.Fatalf("Expected status 'granted', got %v", status)
	}

	id, ok := grantData["id"].(string)
	if !ok || id != testPermissionID {
		t.Fatalf("Expected permission ID %s, got %v", testPermissionID, id)
	}

	message, ok := grantData["message"].(string)
	if !ok || message == "" {
		t.Fatalf("Expected non-empty message, got %v", message)
	}

	t.Logf("✅ Permission grant test passed - Permission ID: %s, Status: %s", id, status)
}

// Test 17: Permission Deny - POST /api/permissions/{id}/deny
func TestRESTPermissionDeny(t *testing.T) {
	result := setupIntegrationTestServer(t)
	defer result.Server.Close()

	t.Log("Testing POST /api/permissions/{id}/deny - Deny permission")

	// Use a test permission ID
	testPermissionID := "test-permission-deny-id"

	// Deny the permission
	denyResp := makeJSONRequest(t, result.Server, "POST", "/api/permissions/"+testPermissionID+"/deny", nil)
	denyData := validateObjectResponse(t, denyResp, http.StatusOK)

	status, ok := denyData["status"].(string)
	if !ok || status != "denied" {
		t.Fatalf("Expected status 'denied', got %v", status)
	}

	id, ok := denyData["id"].(string)
	if !ok || id != testPermissionID {
		t.Fatalf("Expected permission ID %s, got %v", testPermissionID, id)
	}

	message, ok := denyData["message"].(string)
	if !ok || message == "" {
		t.Fatalf("Expected non-empty message, got %v", message)
	}

	t.Logf("✅ Permission deny test passed - Permission ID: %s, Status: %s", id, status)
}

// Test 18: Permission Invalid ID - Error validation
func TestRESTPermissionInvalidID(t *testing.T) {
	result := setupIntegrationTestServer(t)
	defer result.Server.Close()

	t.Log("Testing permission endpoints with invalid ID - Error validation")

	// Test grant with empty permission ID - should return 400
	emptyGrantResp := makeJSONRequest(t, result.Server, "POST", "/api/permissions//grant", nil)
	if emptyGrantResp.StatusCode != http.StatusBadRequest {
		t.Fatalf("Expected status code %d for empty permission ID in grant, got %d", http.StatusBadRequest, emptyGrantResp.StatusCode)
	}

	// Test deny with empty permission ID - should return 400
	emptyDenyResp := makeJSONRequest(t, result.Server, "POST", "/api/permissions//deny", nil)
	if emptyDenyResp.StatusCode != http.StatusBadRequest {
		t.Fatalf("Expected status code %d for empty permission ID in deny, got %d", http.StatusBadRequest, emptyDenyResp.StatusCode)
	}

	t.Logf("✅ Permission invalid ID test passed - Both endpoints properly validate required ID")
}

// Test 19: Command Details - GET /api/commands/{name}
func TestRESTCommandDetails(t *testing.T) {
	result := setupIntegrationTestServer(t)
	defer result.Server.Close()

	t.Log("Testing GET /api/commands/{name} - Get specific command details")

	// First, get the list of available commands to find a valid command name
	listResp := makeJSONRequest(t, result.Server, "GET", "/api/commands", nil)
	commandsList := validateArrayResponse(t, listResp, http.StatusOK)

	if len(commandsList) == 0 {
		t.Skip("No commands available to test command details endpoint")
	}

	// Get the first available command name
	firstCommand := commandsList[0].(map[string]interface{})
	commandName := firstCommand["name"].(string)

	// Test getting valid command details
	detailsResp := makeJSONRequest(t, result.Server, "GET", "/api/commands/"+commandName, nil)
	commandDetails := validateObjectResponse(t, detailsResp, http.StatusOK)

	// Validate required fields in detailed response
	name, ok := commandDetails["name"].(string)
	if !ok || name != commandName {
		t.Fatalf("Expected command name '%s', got %v", commandName, name)
	}

	description, ok := commandDetails["description"].(string)
	if !ok {
		t.Fatalf("Expected command description to be string, got %T", commandDetails["description"])
	}

	cmdType, ok := commandDetails["type"].(string)
	if !ok || (cmdType != "builtin" && cmdType != "file") {
		t.Fatalf("Expected command type to be 'builtin' or 'file', got %v", cmdType)
	}

	// Test getting non-existent command (should return 404)
	nonExistentName := "non-existent-command-name"
	notFoundResp := makeJSONRequest(t, result.Server, "GET", "/api/commands/"+nonExistentName, nil)
	if notFoundResp.StatusCode != http.StatusNotFound {
		t.Fatalf("Expected status code %d for non-existent command, got %d", http.StatusNotFound, notFoundResp.StatusCode)
	}

	t.Logf("✅ Command details test passed - Command: %s, Type: %s, Description: %.50s...", name, cmdType, description)
}

// Test 20: File Upload and List - POST /api/sessions/{id}/files/upload + GET /api/sessions/{id}/files
func TestRESTFileUploadAndList(t *testing.T) {
	result := setupIntegrationTestServer(t)
	defer result.Server.Close()

	t.Log("Testing file upload and listing functionality")

	// Create a session first
	sessionRequest := map[string]interface{}{
		"title": "File Upload Test Session",
	}

	createResp := makeJSONRequest(t, result.Server, "POST", "/api/sessions", sessionRequest)
	createdSessionData := validateObjectResponse(t, createResp, http.StatusCreated)
	sessionID := createdSessionData["id"].(string)

	// Upload a test file
	testContent := "This is test file content for upload testing"
	testFilename := "test-upload.txt"
	
	uploadResp := makeMultipartFileRequest(t, result.Server, 
		"/api/sessions/"+sessionID+"/files/upload", 
		testFilename, 
		testContent)
	
	uploadData := validateObjectResponse(t, uploadResp, http.StatusCreated)
	
	// Validate upload response
	uploadedFilename, ok := uploadData["name"].(string)
	if !ok || uploadedFilename != testFilename {
		t.Fatalf("Expected uploaded filename '%s', got %v", testFilename, uploadedFilename)
	}
	
	uploadedSize, ok := uploadData["size"].(float64)
	if !ok || int(uploadedSize) != len(testContent) {
		t.Fatalf("Expected uploaded file size %d, got %v", len(testContent), uploadedSize)
	}

	// List files in the session
	listResp := makeJSONRequest(t, result.Server, "GET", "/api/sessions/"+sessionID+"/files", nil)
	filesList := validateArrayResponse(t, listResp, http.StatusOK)

	// Verify the uploaded file appears in the list
	found := false
	for _, fileItem := range filesList {
		fileObj, ok := fileItem.(map[string]interface{})
		if !ok {
			continue
		}
		if fileObj["name"].(string) == testFilename {
			found = true
			// Verify file properties in list
			if int(fileObj["size"].(float64)) != len(testContent) {
				t.Fatalf("Expected file size %d in list, got %v", len(testContent), fileObj["size"])
			}
			break
		}
	}

	if !found {
		t.Fatalf("Uploaded file '%s' not found in session file list", testFilename)
	}

	t.Logf("✅ File upload and list test passed - Uploaded and listed file: %s", testFilename)
}

// Test 21: File Serving - Upload and download file, verify content
func TestRESTFileServing(t *testing.T) {
	result := setupIntegrationTestServer(t)
	defer result.Server.Close()

	t.Log("Testing file serving functionality")

	// Create a session first
	sessionRequest := map[string]interface{}{
		"title": "File Serving Test Session",
	}

	createResp := makeJSONRequest(t, result.Server, "POST", "/api/sessions", sessionRequest)
	createdSessionData := validateObjectResponse(t, createResp, http.StatusCreated)
	sessionID := createdSessionData["id"].(string)

	// Upload a test file with specific content
	testContent := "This is test content for file serving validation.\nIt has multiple lines.\nAnd special characters: !@#$%^&*()"
	testFilename := "test-serve.txt"
	
	uploadResp := makeMultipartFileRequest(t, result.Server, 
		"/api/sessions/"+sessionID+"/files/upload", 
		testFilename, 
		testContent)
	
	validateObjectResponse(t, uploadResp, http.StatusCreated)

	// Download the file
	downloadResp := makeJSONRequest(t, result.Server, "GET", 
		"/api/sessions/"+sessionID+"/files/"+testFilename, nil)
	
	if downloadResp.StatusCode != http.StatusOK {
		t.Fatalf("Expected status code %d for file download, got %d", http.StatusOK, downloadResp.StatusCode)
	}

	// Read the downloaded content
	downloadedContent := make([]byte, len(testContent)+100) // Extra buffer to detect size issues
	n, err := downloadResp.Body.Read(downloadedContent)
	downloadResp.Body.Close()
	
	if err != nil && err.Error() != "EOF" {
		t.Fatalf("Failed to read downloaded file content: %v", err)
	}

	// Verify content matches exactly
	actualContent := string(downloadedContent[:n])
	if actualContent != testContent {
		t.Fatalf("Downloaded content doesn't match uploaded content.\nExpected: %q\nActual: %q", testContent, actualContent)
	}

	// Verify content type
	contentType := downloadResp.Header.Get("Content-Type")
	if contentType == "" {
		t.Logf("Warning: No Content-Type header set for downloaded file")
	}

	// Test downloading non-existent file (should return 404)
	nonExistentResp := makeJSONRequest(t, result.Server, "GET", 
		"/api/sessions/"+sessionID+"/files/non-existent.txt", nil)
	
	if nonExistentResp.StatusCode != http.StatusNotFound {
		t.Fatalf("Expected status code %d for non-existent file, got %d", http.StatusNotFound, nonExistentResp.StatusCode)
	}

	t.Logf("✅ File serving test passed - Downloaded file content matches: %s", testFilename)
}

// Test 22: File Path Security - Test path traversal prevention
func TestRESTFilePathSecurity(t *testing.T) {
	result := setupIntegrationTestServer(t)
	defer result.Server.Close()

	t.Log("Testing file path security and traversal prevention")

	// Create a session first
	sessionRequest := map[string]interface{}{
		"title": "Security Test Session",
	}

	createResp := makeJSONRequest(t, result.Server, "POST", "/api/sessions", sessionRequest)
	createdSessionData := validateObjectResponse(t, createResp, http.StatusCreated)
	sessionID := createdSessionData["id"].(string)

	// Note: Multipart form uploads automatically sanitize filenames via Go's multipart library
	// This is an additional security layer - dangerous path components are stripped
	// We verify this behavior but know it's not our os.Root protection
	
	// Test that dangerous filenames get sanitized by multipart library (expected behavior)
	dangerousFilename := "../etc/passwd"
	testContent := "This should not be written outside session directory"
	
	t.Logf("Testing multipart filename sanitization for: %s", dangerousFilename)
	
	uploadResp := makeMultipartFileRequest(t, result.Server, 
		"/api/sessions/"+sessionID+"/files/upload", 
		dangerousFilename, 
		testContent)
	
	// Multipart library sanitizes this to just "passwd", so upload should succeed
	if uploadResp.StatusCode != http.StatusCreated {
		t.Fatalf("Expected status code %d after multipart sanitization, got %d", 
			http.StatusCreated, uploadResp.StatusCode)
	}
	uploadResp.Body.Close()
	
	// Verify the file was created with sanitized name "passwd"
	listResp := makeJSONRequest(t, result.Server, "GET", "/api/sessions/"+sessionID+"/files", nil)
	filesList := validateArrayResponse(t, listResp, http.StatusOK)
	
	found := false
	for _, fileItem := range filesList {
		fileObj := fileItem.(map[string]interface{})
		if fileObj["name"].(string) == "passwd" { // Sanitized filename
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("Expected to find sanitized filename 'passwd' in file list")
	}

	// Test dangerous file access paths (should be rejected)
	dangerousAccessPaths := []string{
		"../other-session-file.txt",
		"../../config.json",
		"../../../../../etc/passwd",
		"./../sensitive.txt",
	}

	for _, dangerousPath := range dangerousAccessPaths {
		t.Logf("Testing file access rejection for dangerous path: %s", dangerousPath)
		
		accessResp := makeJSONRequest(t, result.Server, "GET", 
			"/api/sessions/"+sessionID+"/files/"+dangerousPath, nil)
		
		// Should be rejected - either 400 (validation error) or 404 (router rejects path)
		// Both are acceptable security responses
		if accessResp.StatusCode != http.StatusBadRequest && accessResp.StatusCode != http.StatusNotFound {
			t.Fatalf("Expected status code %d or %d for dangerous path '%s', got %d", 
				http.StatusBadRequest, http.StatusNotFound, dangerousPath, accessResp.StatusCode)
		}
		accessResp.Body.Close()
	}

	// Test that normal filenames still work (positive test)
	normalFilename := "normal-file.txt"
	normalContent := "This is a normal file"
	
	normalUploadResp := makeMultipartFileRequest(t, result.Server, 
		"/api/sessions/"+sessionID+"/files/upload", 
		normalFilename, 
		normalContent)
	
	if normalUploadResp.StatusCode != http.StatusCreated {
		t.Fatalf("Expected status code %d for normal filename, got %d", 
			http.StatusCreated, normalUploadResp.StatusCode)
	}
	normalUploadResp.Body.Close()

	// Verify normal file can be accessed
	normalAccessResp := makeJSONRequest(t, result.Server, "GET", 
		"/api/sessions/"+sessionID+"/files/"+normalFilename, nil)
	
	if normalAccessResp.StatusCode != http.StatusOK {
		t.Fatalf("Expected status code %d for normal file access, got %d", 
			http.StatusOK, normalAccessResp.StatusCode)
	}
	normalAccessResp.Body.Close()

	t.Logf("✅ File path security test passed - All dangerous paths properly rejected")
}

// Test 23: File Session Isolation - Verify files are isolated between sessions
func TestRESTFileSessionIsolation(t *testing.T) {
	result := setupIntegrationTestServer(t)
	defer result.Server.Close()

	t.Log("Testing file session isolation")

	// Create two separate sessions
	session1Request := map[string]interface{}{
		"title": "Isolation Test Session 1",
	}
	session2Request := map[string]interface{}{
		"title": "Isolation Test Session 2",
	}

	// Create session 1
	createResp1 := makeJSONRequest(t, result.Server, "POST", "/api/sessions", session1Request)
	session1Data := validateObjectResponse(t, createResp1, http.StatusCreated)
	session1ID := session1Data["id"].(string)

	// Create session 2
	createResp2 := makeJSONRequest(t, result.Server, "POST", "/api/sessions", session2Request)
	session2Data := validateObjectResponse(t, createResp2, http.StatusCreated)
	session2ID := session2Data["id"].(string)

	// Upload a file to session 1
	session1FileName := "session1-private.txt"
	session1Content := "This file belongs to session 1 and should not be accessible from session 2"
	
	uploadResp1 := makeMultipartFileRequest(t, result.Server, 
		"/api/sessions/"+session1ID+"/files/upload", 
		session1FileName, 
		session1Content)
	
	validateObjectResponse(t, uploadResp1, http.StatusCreated)

	// Upload a file to session 2 (different file, same name to test isolation)
	session2FileName := "session1-private.txt" // Same filename as session 1
	session2Content := "This file belongs to session 2 with the same filename"
	
	uploadResp2 := makeMultipartFileRequest(t, result.Server, 
		"/api/sessions/"+session2ID+"/files/upload", 
		session2FileName, 
		session2Content)
	
	validateObjectResponse(t, uploadResp2, http.StatusCreated)

	// Verify session 1 can access its own file
	access1Resp := makeJSONRequest(t, result.Server, "GET", 
		"/api/sessions/"+session1ID+"/files/"+session1FileName, nil)
	
	if access1Resp.StatusCode != http.StatusOK {
		t.Fatalf("Session 1 should be able to access its own file, got status %d", access1Resp.StatusCode)
	}
	
	// Read content to verify it's session 1's file
	content1 := make([]byte, len(session1Content)+10)
	n1, _ := access1Resp.Body.Read(content1)
	access1Resp.Body.Close()
	actualContent1 := string(content1[:n1])
	
	if actualContent1 != session1Content {
		t.Fatalf("Session 1 file content mismatch. Expected: %q, Got: %q", session1Content, actualContent1)
	}

	// Verify session 2 can access its own file
	access2Resp := makeJSONRequest(t, result.Server, "GET", 
		"/api/sessions/"+session2ID+"/files/"+session2FileName, nil)
	
	if access2Resp.StatusCode != http.StatusOK {
		t.Fatalf("Session 2 should be able to access its own file, got status %d", access2Resp.StatusCode)
	}
	
	// Read content to verify it's session 2's file
	content2 := make([]byte, len(session2Content)+10)
	n2, _ := access2Resp.Body.Read(content2)
	access2Resp.Body.Close()
	actualContent2 := string(content2[:n2])
	
	if actualContent2 != session2Content {
		t.Fatalf("Session 2 file content mismatch. Expected: %q, Got: %q", session2Content, actualContent2)
	}

	// Test file listing isolation - session 1 should only see its own files
	list1Resp := makeJSONRequest(t, result.Server, "GET", "/api/sessions/"+session1ID+"/files", nil)
	files1List := validateArrayResponse(t, list1Resp, http.StatusOK)
	
	// Session 1 should see exactly 1 file
	if len(files1List) != 1 {
		t.Fatalf("Session 1 should see exactly 1 file, got %d", len(files1List))
	}

	// Test file listing isolation - session 2 should only see its own files
	list2Resp := makeJSONRequest(t, result.Server, "GET", "/api/sessions/"+session2ID+"/files", nil)
	files2List := validateArrayResponse(t, list2Resp, http.StatusOK)
	
	// Session 2 should see exactly 1 file
	if len(files2List) != 1 {
		t.Fatalf("Session 2 should see exactly 1 file, got %d", len(files2List))
	}

	// Test cross-session access attempt (using wrong session IDs)
	// This tests the session validation, not just file existence
	wrongSessionResp := makeJSONRequest(t, result.Server, "GET", 
		"/api/sessions/wrong-session-id/files/"+session1FileName, nil)
	
	// Should fail with validation error (invalid session ID format returns 400 Bad Request)
	if wrongSessionResp.StatusCode != http.StatusBadRequest {
		t.Fatalf("Expected status code %d for invalid session ID format, got %d", 
			http.StatusBadRequest, wrongSessionResp.StatusCode)
	}
	wrongSessionResp.Body.Close()

	t.Logf("✅ File session isolation test passed - Sessions properly isolated")
}

// TestRESTAPIIntegration runs all REST API integration tests sequentially
func TestRESTAPIIntegration(t *testing.T) {
	t.Log("🚀 Starting comprehensive REST API integration tests")

	t.Run("SessionCreation", TestRESTSessionCreation)
	t.Run("SessionListing", TestRESTSessionListing)
	t.Run("SessionRetrieval", TestRESTSessionRetrieval)
	t.Run("SessionDeletion", TestRESTSessionDeletion)
	t.Run("SessionForking", TestRESTSessionForking)
	t.Run("MessageSending", TestRESTMessageSending)
	t.Run("MessageListing", TestRESTMessageListing)
	t.Run("MessageHistory", TestRESTMessageHistory)
	t.Run("AgentCancellation", TestRESTAgentCancellation)
	t.Run("CommandsListing", TestRESTCommandsListing)
	t.Run("CommandDetails", TestRESTCommandDetails)
	t.Run("MCPServersListing", TestRESTMCPServersListing)
	t.Run("HealthCheck", TestRESTHealthCheck)
	t.Run("StreamEndpoint", TestRESTStreamEndpoint)
	t.Run("StreamSubPathEndpoint", TestRESTStreamSubPathEndpoint)
	t.Run("PermissionGrant", TestRESTPermissionGrant)
	t.Run("PermissionDeny", TestRESTPermissionDeny)
	t.Run("PermissionInvalidID", TestRESTPermissionInvalidID)
	
	// File management tests
	t.Run("FileUploadAndList", TestRESTFileUploadAndList)
	t.Run("FileServing", TestRESTFileServing)
	t.Run("FilePathSecurity", TestRESTFilePathSecurity) // Now using os.Root for robust security
	t.Run("FileSessionIsolation", TestRESTFileSessionIsolation)

	t.Log("🎉 All REST API integration tests completed successfully!")
}
