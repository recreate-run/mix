package integration_tests

import (
	"context"
	"net/http"
	"testing"
)

// NOTE: TestSubagentEventRouting moved to e2e/subagents/subagent_e2e_test.go
// It requires real LLM API and is a true E2E test

// TestSubagentEventRoutingVerifyHierarchy validates session hierarchy and parent-child relationships
// This is a simpler test that doesn't depend on LLM behavior
func TestSubagentEventRoutingVerifyHierarchy(t *testing.T) {
	result := setupIntegrationTestServer(t)
	defer result.Server.Close()

	t.Log("Testing session hierarchy for subagent event routing")

	// Create a main session
	sessionRequest := map[string]interface{}{
		"title": "Main Hierarchy Test",
	}
	createResp := makeJSONRequest(t, result.Server, "POST", "/api/sessions", sessionRequest)
	defer func() { _ = createResp.Body.Close() }()
	sessionData := validateObjectResponse(t, createResp, http.StatusCreated)
	mainSessionID := sessionData["id"].(string)

	// Create a subagent session programmatically
	ctx := context.Background()
	subSession, err := result.App.Sessions.Create(
		ctx,
		"Test Subagent",
		"",
		"default",
		"subagent",          // session type
		"general-purpose",   // subagent type
		mainSessionID,       // parent session ID
		"test-tool-call-id", // parent tool call ID (test value)
	)
	if err != nil {
		t.Fatalf("Failed to create subagent session: %v", err)
	}

	t.Logf("Created subagent session: %s with parent: %s", subSession.ID, subSession.ParentSessionID)

	// Verify hierarchy
	if subSession.ParentSessionID != mainSessionID {
		t.Errorf("Expected subagent parent to be %s, got %s", mainSessionID, subSession.ParentSessionID)
	}

	if string(subSession.SessionType) != "subagent" {
		t.Errorf("Expected session type 'subagent', got %s", subSession.SessionType)
	}

	if string(subSession.SubagentType) != "general-purpose" {
		t.Errorf("Expected subagent type 'general-purpose', got %s", subSession.SubagentType)
	}

	// Verify we can retrieve both sessions
	mainSession, err := result.App.Sessions.Get(ctx, mainSessionID)
	if err != nil {
		t.Fatalf("Failed to retrieve main session: %v", err)
	}

	if mainSession.ParentSessionID != "" {
		t.Errorf("Expected main session to have no parent, got parent: %s", mainSession.ParentSessionID)
	}

	retrievedSubSession, err := result.App.Sessions.Get(ctx, subSession.ID)
	if err != nil {
		t.Fatalf("Failed to retrieve subagent session: %v", err)
	}

	if retrievedSubSession.ParentSessionID != mainSessionID {
		t.Errorf("Retrieved subagent has wrong parent. Expected %s, got %s",
			mainSessionID, retrievedSubSession.ParentSessionID)
	}

	t.Log("✅ Session hierarchy verification passed")
	t.Logf("   - Main session: %s (type: %s, parent: none)", mainSession.ID, mainSession.SessionType)
	t.Logf("   - Subagent session: %s (type: %s, subagent_type: %s, parent: %s)",
		subSession.ID, subSession.SessionType, subSession.SubagentType, subSession.ParentSessionID)
}
