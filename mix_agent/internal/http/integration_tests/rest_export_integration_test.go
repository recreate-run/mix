package integration_tests

import (
	"context"
	"net/http"
	"testing"
	"time"

	"mix/internal/message"
)

// Test Export: Session Export - GET /api/sessions/{id}/export
func TestRESTSessionExport(t *testing.T) {
	result := setupIntegrationTestServer(t)
	defer result.Server.Close()

	t.Log("Testing GET /api/sessions/{id}/export - Export session transcript")

	// Create a session with some test data
	sessionRequest := map[string]interface{}{
		"title": "Export Test Session",
	}

	createResp := makeJSONRequest(t, result.Server, "POST", "/api/sessions", sessionRequest)
	createdSessionData := validateObjectResponse(t, createResp, http.StatusCreated)
	sessionID := createdSessionData["id"].(string)

	// Create some test messages directly in the database to ensure we have content to export
	ctx := context.Background()

	// Create a user message
	_, err := result.App.Messages.Create(ctx, sessionID, message.CreateMessageParams{
		Role: message.User,
		Parts: []message.ContentPart{
			message.TextContent{Text: "Hello, this is a test message"},
		},
		Model: "claude-4-sonnet",
	})
	if err != nil {
		t.Fatalf("Failed to create test user message: %v", err)
	}

	// Create an assistant message with tool call
	_, err = result.App.Messages.Create(ctx, sessionID, message.CreateMessageParams{
		Role: message.Assistant,
		Parts: []message.ContentPart{
			message.TextContent{Text: "I'll help you with that"},
			message.ToolCall{
				ID:    "test-tool-call-123",
				Name:  "search",
				Input: `{"query": "test query", "type": "web"}`,
				Type:  "tool_use",
			},
		},
		Model: "claude-4-sonnet",
	})
	if err != nil {
		t.Fatalf("Failed to create test assistant message: %v", err)
	}

	// Create a tool result message
	_, err = result.App.Messages.Create(ctx, sessionID, message.CreateMessageParams{
		Role: message.Tool,
		Parts: []message.ContentPart{
			message.ToolResult{
				ToolCallID: "test-tool-call-123",
				Name:       "search",
				Content:    "Search results: test result data",
				IsError:    false,
			},
		},
	})
	if err != nil {
		t.Fatalf("Failed to create test tool result message: %v", err)
	}

	// Test the export endpoint
	exportResp := makeJSONRequest(t, result.Server, "GET", "/api/sessions/"+sessionID+"/export", nil)
	exportData := validateObjectResponse(t, exportResp, http.StatusOK)

	// Verify the export structure
	if exportData["id"].(string) != sessionID {
		t.Errorf("Expected session ID %s, got %s", sessionID, exportData["id"].(string))
	}

	if exportData["title"].(string) != "Export Test Session" {
		t.Errorf("Expected title 'Export Test Session', got %s", exportData["title"].(string))
	}

	// Verify messages are present
	messages := exportData["messages"].([]interface{})
	if len(messages) != 3 {
		t.Errorf("Expected 3 messages, got %d", len(messages))
	}

	// Verify first message (user message)
	firstMsg := messages[0].(map[string]interface{})
	if firstMsg["role"].(string) != "user" {
		t.Errorf("Expected first message role to be 'user', got %s", firstMsg["role"].(string))
	}
	if firstMsg["content"].(string) != "Hello, this is a test message" {
		t.Errorf("Expected first message content to match test content")
	}

	// Verify second message (assistant with tool call)
	secondMsg := messages[1].(map[string]interface{})
	if secondMsg["role"].(string) != "assistant" {
		t.Errorf("Expected second message role to be 'assistant', got %s", secondMsg["role"].(string))
	}

	// Verify tool calls are exported
	toolCalls := secondMsg["toolCalls"].([]interface{})
	if len(toolCalls) != 1 {
		t.Errorf("Expected 1 tool call, got %d", len(toolCalls))
	}

	firstToolCall := toolCalls[0].(map[string]interface{})
	if firstToolCall["id"].(string) != "test-tool-call-123" {
		t.Errorf("Expected tool call ID 'test-tool-call-123', got %s", firstToolCall["id"].(string))
	}
	if firstToolCall["name"].(string) != "search" {
		t.Errorf("Expected tool call name 'search', got %s", firstToolCall["name"].(string))
	}

	// Verify input JSON parsing
	if firstToolCall["inputJson"] == nil {
		t.Error("Expected inputJson to be parsed and present")
	} else {
		inputJSON := firstToolCall["inputJson"].(map[string]interface{})
		if inputJSON["query"].(string) != "test query" {
			t.Errorf("Expected parsed query 'test query', got %s", inputJSON["query"].(string))
		}
	}

	// Verify third message (tool result)
	thirdMsg := messages[2].(map[string]interface{})
	if thirdMsg["role"].(string) != "tool" {
		t.Errorf("Expected third message role to be 'tool', got %s", thirdMsg["role"].(string))
	}

	// Verify timestamps are properly formatted
	createdAt := exportData["createdAt"].(string)
	_, err = time.Parse(time.RFC3339, createdAt)
	if err != nil {
		t.Errorf("Expected createdAt to be valid RFC3339 timestamp, got parse error: %v", err)
	}

	// Verify Content-Disposition header for download
	contentDisposition := exportResp.Header.Get("Content-Disposition")
	expectedFilename := "session_" + sessionID + "_transcript.json"
	if contentDisposition != "attachment; filename="+expectedFilename {
		t.Errorf("Expected Content-Disposition header with filename %s", expectedFilename)
	}

	t.Logf("✅ Export test passed - Successfully exported %d messages with tool calls and proper structure", len(messages))
}

// Test Export: Non-existent session - GET /api/sessions/{id}/export
func TestRESTSessionExportNotFound(t *testing.T) {
	result := setupIntegrationTestServer(t)
	defer result.Server.Close()

	t.Log("Testing GET /api/sessions/{id}/export - Non-existent session")

	// Try to export a non-existent session
	fakeSessionID := "non-existent-session-id"
	exportResp := makeJSONRequest(t, result.Server, "GET", "/api/sessions/"+fakeSessionID+"/export", nil)

	// Should return 404 Not Found
	validateErrorResponse(t, exportResp, http.StatusNotFound)

	t.Logf("✅ Export not found test passed - Correctly returned 404 for non-existent session")
}