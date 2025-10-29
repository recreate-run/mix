package integration_tests

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
	defer func() { _ = resp.Body.Close() }()
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
	defer func() { _ = createResp.Body.Close() }()
	createdSessionData := validateObjectResponse(t, createResp, http.StatusCreated)

	createdSessionID := createdSessionData["id"].(string)

	// Now list sessions
	listResp := makeJSONRequest(t, result.Server, "GET", "/api/sessions", nil)
	defer func() { _ = listResp.Body.Close() }()
	sessionsArray := validateArrayResponse(t, listResp)

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
	defer func() { _ = createResp.Body.Close() }()
	createdSessionData := validateObjectResponse(t, createResp, http.StatusCreated)

	sessionID := createdSessionData["id"].(string)

	// Retrieve the specific session
	getResp := makeJSONRequest(t, result.Server, "GET", "/api/sessions/"+sessionID, nil)
	defer func() { _ = getResp.Body.Close() }()
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
	defer func() { _ = createResp.Body.Close() }()
	createdSessionData := validateObjectResponse(t, createResp, http.StatusCreated)

	sessionID := createdSessionData["id"].(string)

	// Delete the session
	deleteResp := makeJSONRequest(t, result.Server, "DELETE", "/api/sessions/"+sessionID, nil)
	defer func() { _ = deleteResp.Body.Close() }()
	if deleteResp.StatusCode != http.StatusNoContent {
		t.Fatalf("Expected status code %d for deletion, got %d", http.StatusNoContent, deleteResp.StatusCode)
	}

	// Verify the session is gone - should return 404
	getResp := makeJSONRequest(t, result.Server, "GET", "/api/sessions/"+sessionID, nil)
	defer func() { _ = getResp.Body.Close() }()
	if getResp.StatusCode != http.StatusNotFound {
		t.Fatalf("Expected status code %d when retrieving deleted session, got %d", http.StatusNotFound, getResp.StatusCode)
	}

	// Test deleting non-existent session - should return 404 (not found)
	nonExistentID := "non-existent-session-id"
	deleteNonExistentResp := makeJSONRequest(t, result.Server, "DELETE", "/api/sessions/"+nonExistentID, nil)
	defer func() { _ = deleteNonExistentResp.Body.Close() }()
	if deleteNonExistentResp.StatusCode != http.StatusNotFound {
		t.Fatalf("Expected status code %d when deleting non-existent session, got %d", http.StatusNotFound, deleteNonExistentResp.StatusCode)
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
	defer func() { _ = createResp.Body.Close() }()
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
	defer func() { _ = forkResp.Body.Close() }()
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
	defer func() { _ = sourceGetResp.Body.Close() }()
	validateObjectResponse(t, sourceGetResp, http.StatusOK)

	forkedGetResp := makeJSONRequest(t, result.Server, "GET", "/api/sessions/"+forkedSessionID, nil)
	defer func() { _ = forkedGetResp.Body.Close() }()
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
	defer func() { _ = createResp.Body.Close() }()
	createdSessionData := validateObjectResponse(t, createResp, http.StatusCreated)

	sessionID := createdSessionData["id"].(string)

	// Cancel agent processing for the session
	cancelResp := makeJSONRequest(t, result.Server, "POST", "/api/sessions/"+sessionID+"/cancel", nil)
	defer func() { _ = cancelResp.Body.Close() }()
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
