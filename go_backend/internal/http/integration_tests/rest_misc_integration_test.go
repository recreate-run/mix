package integration_tests

import (
	"net/http"
	"testing"
)

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
	t.Run("FileDeletion", TestRESTFileDeletion)
	t.Run("FileThumbnailGeneration", TestRESTFileThumbnailGeneration)
	t.Run("LargeFileHandling", TestRESTLargeFileHandling)

	t.Log("🎉 All REST API integration tests completed successfully!")
}