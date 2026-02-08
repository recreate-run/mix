package browser

import (
	"fmt"
	"os"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"mix/internal/llm/interfaces"
	"mix/internal/session"
)


// extractTabID extracts tab ID from browser tool response
// Response format: "Created new tab: tab-X (URL: ...)"
func extractTabID(content string) string {
	parts := strings.Split(content, "Created new tab: ")
	if len(parts) > 1 {
		idPart := strings.TrimSpace(parts[1])
		return strings.Fields(idPart)[0]
	}
	return ""
}

// TestBrowserTool_TabCreate tests creating a new tab
func TestBrowserTool_TabCreate(t *testing.T) {
	skipIfIntegrationTestsDisabled(t)

	mockServer := startMockBrowserServer(t)
	defer mockServer.Close()

	mockPermissionService := &MockPermissionService{}
	sessionConfig := session.DefaultConfig()
	tool := NewBrowserTool(mockPermissionService, mockServer.wsURL, sessionConfig, "", mockClientFactory, nil, nil)

	tempDir := t.TempDir()
	ctx := createBrowserTestContext("test-session", "test-message", tempDir)

	mockPermissionService.On("Request", mock.Anything).Return(true)

	// Create a new tab
	call := interfaces.ToolCall{
		ID:    "call-tab-create",
		Name:  BrowserToolName,
		Input: `{"action": "tab_create"}`,
	}

	response, err := tool.Run(ctx, call)
	require.NoError(t, err)
	assert.False(t, response.IsError, "Response should not be error: %s", response.Content)
	assert.Contains(t, response.Content, "Created new tab:")
	assert.Contains(t, response.Content, "tab-", "Response should contain tab ID")
	t.Logf("Tab create response: %s", response.Content)
}

// TestBrowserTool_TabList tests listing all tabs
func TestBrowserTool_TabList(t *testing.T) {
	skipIfIntegrationTestsDisabled(t)

	mockServer := startMockBrowserServer(t)
	defer mockServer.Close()

	mockPermissionService := &MockPermissionService{}
	sessionConfig := session.DefaultConfig()
	tool := NewBrowserTool(mockPermissionService, mockServer.wsURL, sessionConfig, "", mockClientFactory, nil, nil)

	tempDir := t.TempDir()
	ctx := createBrowserTestContext("test-session", "test-message", tempDir)

	mockPermissionService.On("Request", mock.Anything).Return(true)

	// Create 2 tabs first
	for i := 0; i < 2; i++ {
		createCall := interfaces.ToolCall{
			ID:    fmt.Sprintf("call-create-%d", i),
			Name:  BrowserToolName,
			Input: `{"action": "tab_create"}`,
		}
		_, err := tool.Run(ctx, createCall)
		require.NoError(t, err)
	}

	// List tabs
	listCall := interfaces.ToolCall{
		ID:    "call-tab-list",
		Name:  BrowserToolName,
		Input: `{"action": "tab_list"}`,
	}

	response, err := tool.Run(ctx, listCall)
	require.NoError(t, err)
	assert.False(t, response.IsError)
	assert.Contains(t, response.Content, "Total tabs:")
	assert.Contains(t, response.Content, "Active tab:")

	// Should have at least 3 tabs (1 initial + 2 created)
	tabCount := strings.Count(response.Content, "tab-")
	assert.GreaterOrEqual(t, tabCount, 3, "Should have at least 3 tabs listed")

	t.Logf("Tab list response: %s", response.Content)
}

// TestBrowserTool_TabSwitch tests switching between tabs
func TestBrowserTool_TabSwitch(t *testing.T) {
	skipIfIntegrationTestsDisabled(t)

	mockServer := startMockBrowserServer(t)
	defer mockServer.Close()

	mockPermissionService := &MockPermissionService{}
	sessionConfig := session.DefaultConfig()
	tool := NewBrowserTool(mockPermissionService, mockServer.wsURL, sessionConfig, "", mockClientFactory, nil, nil)

	tempDir := t.TempDir()
	ctx := createBrowserTestContext("test-session", "test-message", tempDir)

	mockPermissionService.On("Request", mock.Anything).Return(true)

	// Create a tab
	createCall := interfaces.ToolCall{
		ID:    "call-create",
		Name:  BrowserToolName,
		Input: `{"action": "tab_create"}`,
	}
	createResp, err := tool.Run(ctx, createCall)
	require.NoError(t, err)

	// Extract tab ID from create response
	tabID := extractTabID(createResp.Content)
	if tabID == "" {
		t.Fatal("Failed to extract tab ID from create response")
	}

	// Switch to the new tab
	switchCall := interfaces.ToolCall{
		ID:    "call-switch",
		Name:  BrowserToolName,
		Input: fmt.Sprintf(`{"action": "tab_switch", "tabId": %q}`, tabID),
	}

	response, err := tool.Run(ctx, switchCall)
	require.NoError(t, err)
	assert.False(t, response.IsError)
	assert.Contains(t, response.Content, "Switched to tab")
	assert.Contains(t, response.Content, tabID)

	t.Logf("Tab switch response: %s", response.Content)
}

// TestBrowserTool_TabClose tests closing a tab
func TestBrowserTool_TabClose(t *testing.T) {
	skipIfIntegrationTestsDisabled(t)

	mockServer := startMockBrowserServer(t)
	defer mockServer.Close()

	mockPermissionService := &MockPermissionService{}
	sessionConfig := session.DefaultConfig()
	tool := NewBrowserTool(mockPermissionService, mockServer.wsURL, sessionConfig, "", mockClientFactory, nil, nil)

	tempDir := t.TempDir()
	ctx := createBrowserTestContext("test-session", "test-message", tempDir)

	mockPermissionService.On("Request", mock.Anything).Return(true)

	// Create 3 tabs to ensure we can close one
	tabIDs := make([]string, 3)
	for i := 0; i < 3; i++ {
		createCall := interfaces.ToolCall{
			ID:    fmt.Sprintf("call-create-%d", i),
			Name:  BrowserToolName,
			Input: `{"action": "tab_create"}`,
		}
		createResp, err := tool.Run(ctx, createCall)
		require.NoError(t, err)

		// Extract tab ID
		tabIDs[i] = extractTabID(createResp.Content)
	}

	// Close the second tab
	closeCall := interfaces.ToolCall{
		ID:    "call-close",
		Name:  BrowserToolName,
		Input: fmt.Sprintf(`{"action": "tab_close", "tabId": %q}`, tabIDs[1]),
	}

	response, err := tool.Run(ctx, closeCall)
	require.NoError(t, err)
	assert.False(t, response.IsError)
	assert.Contains(t, response.Content, "Closed tab")
	assert.Contains(t, response.Content, tabIDs[1])

	t.Logf("Tab close response: %s", response.Content)
}

// TestBrowserTool_TabCloseLastTabError tests error when closing the last tab
func TestBrowserTool_TabCloseLastTabError(t *testing.T) {
	skipIfIntegrationTestsDisabled(t)

	mockServer := startMockBrowserServer(t)
	defer mockServer.Close()

	mockPermissionService := &MockPermissionService{}
	sessionConfig := session.DefaultConfig()
	tool := NewBrowserTool(mockPermissionService, mockServer.wsURL, sessionConfig, "", mockClientFactory, nil, nil)

	tempDir := t.TempDir()
	ctx := createBrowserTestContext("test-session", "test-message", tempDir)

	mockPermissionService.On("Request", mock.Anything).Return(true)

	// List tabs to get the default tab ID
	listCall := interfaces.ToolCall{
		ID:    "call-list",
		Name:  BrowserToolName,
		Input: `{"action": "tab_list"}`,
	}
	listResp, err := tool.Run(ctx, listCall)
	require.NoError(t, err)

	// Extract first tab ID from list
	// Format: "tab-1 [ACTIVE]\n  URL: ..."
	lines := strings.Split(listResp.Content, "\n")
	var firstTabID string
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "tab-") {
			// Extract tab ID (before space or [ACTIVE])
			parts := strings.Fields(line)
			if len(parts) > 0 {
				firstTabID = parts[0]
				break
			}
		}
	}

	if firstTabID == "" {
		t.Fatal("Failed to extract tab ID from list response")
	}

	// Try to close the only tab
	closeCall := interfaces.ToolCall{
		ID:    "call-close-last",
		Name:  BrowserToolName,
		Input: fmt.Sprintf(`{"action": "tab_close", "tabId": %q}`, firstTabID),
	}

	response, err := tool.Run(ctx, closeCall)
	require.NoError(t, err)
	assert.True(t, response.IsError, "Should return error when closing last tab")
	assert.Contains(t, response.Content, "cannot close", "Error message should indicate cannot close")

	t.Logf("Expected error response: %s", response.Content)
}

// TestBrowserTool_NavigateSpecificTab tests navigating different tabs to different URLs
func TestBrowserTool_NavigateSpecificTab(t *testing.T) {
	skipIfIntegrationTestsDisabled(t)

	mockServer := startMockBrowserServer(t)
	defer mockServer.Close()

	mockPermissionService := &MockPermissionService{}
	sessionConfig := session.DefaultConfig()
	tool := NewBrowserTool(mockPermissionService, mockServer.wsURL, sessionConfig, "", mockClientFactory, nil, nil)

	tempDir := t.TempDir()
	ctx := createBrowserTestContext("test-session", "test-message", tempDir)

	mockPermissionService.On("Request", mock.Anything).Return(true)

	// Create 2 tabs
	tab1CreateCall := interfaces.ToolCall{
		ID:    "call-create-1",
		Name:  BrowserToolName,
		Input: `{"action": "tab_create"}`,
	}
	tab1Resp, err := tool.Run(ctx, tab1CreateCall)
	require.NoError(t, err)

	// Extract tab1 ID
	tab1ID := extractTabID(tab1Resp.Content)
	require.NotEmpty(t, tab1ID, "Failed to extract tab1 ID")

	tab2CreateCall := interfaces.ToolCall{
		ID:    "call-create-2",
		Name:  BrowserToolName,
		Input: `{"action": "tab_create"}`,
	}
	tab2Resp, err := tool.Run(ctx, tab2CreateCall)
	require.NoError(t, err)

	// Extract tab2 ID
	tab2ID := extractTabID(tab2Resp.Content)
	require.NotEmpty(t, tab2ID, "Failed to extract tab2 ID")

	// Navigate tab1 to example.com
	nav1Call := interfaces.ToolCall{
		ID:    "call-nav-1",
		Name:  BrowserToolName,
		Input: fmt.Sprintf(`{"action": "open", "url": "https://example.com", "tabId": %q}`, tab1ID),
	}
	nav1Resp, err := tool.Run(ctx, nav1Call)
	require.NoError(t, err)
	assert.False(t, nav1Resp.IsError)
	assert.Contains(t, nav1Resp.Content, "example.com")

	// Navigate tab2 to google.com
	nav2Call := interfaces.ToolCall{
		ID:    "call-nav-2",
		Name:  BrowserToolName,
		Input: fmt.Sprintf(`{"action": "open", "url": "https://google.com", "tabId": %q}`, tab2ID),
	}
	nav2Resp, err := tool.Run(ctx, nav2Call)
	require.NoError(t, err)
	assert.False(t, nav2Resp.IsError)
	assert.Contains(t, nav2Resp.Content, "google.com")

	t.Logf("Successfully navigated tab1 to example.com and tab2 to google.com")
}

// TestBrowserTool_ScreenshotSpecificTab tests taking screenshots of different tabs
func TestBrowserTool_ScreenshotSpecificTab(t *testing.T) {
	skipIfIntegrationTestsDisabled(t)

	mockServer := startMockBrowserServer(t)
	defer mockServer.Close()

	mockPermissionService := &MockPermissionService{}
	sessionConfig := session.DefaultConfig()
	tool := NewBrowserTool(mockPermissionService, mockServer.wsURL, sessionConfig, "", mockClientFactory, nil, nil)

	tempDir := t.TempDir()
	ctx := createBrowserTestContext("test-session", "test-message", tempDir)

	mockPermissionService.On("Request", mock.Anything).Return(true)

	// Create a tab
	createCall := interfaces.ToolCall{
		ID:    "call-create",
		Name:  BrowserToolName,
		Input: `{"action": "tab_create"}`,
	}
	createResp, err := tool.Run(ctx, createCall)
	require.NoError(t, err)

	// Extract tab ID
	tabID := extractTabID(createResp.Content)
	require.NotEmpty(t, tabID, "Failed to extract tab ID")

	// Navigate the tab
	navCall := interfaces.ToolCall{
		ID:    "call-nav",
		Name:  BrowserToolName,
		Input: fmt.Sprintf(`{"action": "open", "url": "https://example.com", "tabId": %q}`, tabID),
	}
	_, err = tool.Run(ctx, navCall)
	require.NoError(t, err)

	// Take screenshot of specific tab
	screenshotCall := interfaces.ToolCall{
		ID:    "call-screenshot",
		Name:  BrowserToolName,
		Input: fmt.Sprintf(`{"action": "screenshot", "tabId": %q}`, tabID),
	}

	response, err := tool.Run(ctx, screenshotCall)
	require.NoError(t, err)
	assert.False(t, response.IsError)
	assert.Contains(t, response.Content, "Screenshot captured")
	assert.Contains(t, response.Content, "Display URL:")

	// Verify screenshot file was created
	files, err := os.ReadDir(tempDir)
	require.NoError(t, err)
	assert.NotEmpty(t, files, "Screenshot file should be created")

	foundScreenshot := false
	for _, file := range files {
		if strings.HasPrefix(file.Name(), "screenshot_") && strings.HasSuffix(file.Name(), ".png") {
			foundScreenshot = true
			break
		}
	}
	assert.True(t, foundScreenshot, "Screenshot file should exist")

	t.Logf("Screenshot response: %s", response.Content)
}

// TestBrowserTool_ClickInSpecificTab tests clicking elements in a specific tab
func TestBrowserTool_ClickInSpecificTab(t *testing.T) {
	skipIfIntegrationTestsDisabled(t)

	mockServer := startMockBrowserServer(t)
	defer mockServer.Close()

	mockPermissionService := &MockPermissionService{}
	sessionConfig := session.DefaultConfig()
	tool := NewBrowserTool(mockPermissionService, mockServer.wsURL, sessionConfig, "", mockClientFactory, nil, nil)

	tempDir := t.TempDir()
	ctx := createBrowserTestContext("test-session", "test-message", tempDir)

	mockPermissionService.On("Request", mock.Anything).Return(true)

	// Create a tab
	createCall := interfaces.ToolCall{
		ID:    "call-create",
		Name:  BrowserToolName,
		Input: `{"action": "tab_create"}`,
	}
	createResp, err := tool.Run(ctx, createCall)
	require.NoError(t, err)

	// Extract tab ID
	tabID := extractTabID(createResp.Content)
	require.NotEmpty(t, tabID, "Failed to extract tab ID")

	// Navigate the tab
	navCall := interfaces.ToolCall{
		ID:    "call-nav",
		Name:  BrowserToolName,
		Input: fmt.Sprintf(`{"action": "open", "url": "https://example.com", "tabId": %q}`, tabID),
	}
	_, err = tool.Run(ctx, navCall)
	require.NoError(t, err)

	// Take screenshot to populate element cache
	screenshotCall := interfaces.ToolCall{
		ID:    "call-screenshot",
		Name:  BrowserToolName,
		Input: fmt.Sprintf(`{"action": "screenshot", "tabId": %q}`, tabID),
	}
	_, err = tool.Run(ctx, screenshotCall)
	require.NoError(t, err)

	// Click element in specific tab
	clickCall := interfaces.ToolCall{
		ID:    "call-click",
		Name:  BrowserToolName,
		Input: fmt.Sprintf(`{"action": "click", "index": 1, "tabId": %q}`, tabID),
	}

	response, err := tool.Run(ctx, clickCall)
	require.NoError(t, err)
	assert.False(t, response.IsError)
	assert.Contains(t, response.Content, "Successfully clicked element 1")

	t.Logf("Click response: %s", response.Content)
}

// TestBrowserTool_TabSwitchInvalidID tests error handling for invalid tab ID
func TestBrowserTool_TabSwitchInvalidID(t *testing.T) {
	skipIfIntegrationTestsDisabled(t)

	mockServer := startMockBrowserServer(t)
	defer mockServer.Close()

	mockPermissionService := &MockPermissionService{}
	sessionConfig := session.DefaultConfig()
	tool := NewBrowserTool(mockPermissionService, mockServer.wsURL, sessionConfig, "", mockClientFactory, nil, nil)

	tempDir := t.TempDir()
	ctx := createBrowserTestContext("test-session", "test-message", tempDir)

	mockPermissionService.On("Request", mock.Anything).Return(true)

	// Try to switch to invalid tab
	switchCall := interfaces.ToolCall{
		ID:    "call-switch-invalid",
		Name:  BrowserToolName,
		Input: `{"action": "tab_switch", "tabId": "invalid-tab-id"}`,
	}

	response, err := tool.Run(ctx, switchCall)
	require.NoError(t, err)
	assert.True(t, response.IsError, "Should return error for invalid tab ID")
	assert.Contains(t, response.Content, "Failed", "Error message should indicate failure")

	t.Logf("Expected error response: %s", response.Content)
}

// TestBrowserTool_TabMissingIDForSwitch tests error when tabId is missing for tab_switch
func TestBrowserTool_TabMissingIDForSwitch(t *testing.T) {
	skipIfIntegrationTestsDisabled(t)

	mockServer := startMockBrowserServer(t)
	defer mockServer.Close()

	mockPermissionService := &MockPermissionService{}
	sessionConfig := session.DefaultConfig()
	tool := NewBrowserTool(mockPermissionService, mockServer.wsURL, sessionConfig, "", mockClientFactory, nil, nil)

	tempDir := t.TempDir()
	ctx := createBrowserTestContext("test-session", "test-message", tempDir)

	mockPermissionService.On("Request", mock.Anything).Return(true)

	// Try tab_switch without tabId
	switchCall := interfaces.ToolCall{
		ID:    "call-switch-no-id",
		Name:  BrowserToolName,
		Input: `{"action": "tab_switch"}`,
	}

	response, err := tool.Run(ctx, switchCall)
	require.NoError(t, err)
	assert.True(t, response.IsError)
	assert.Contains(t, response.Content, "missing tabId", "Error should indicate missing tabId")

	t.Logf("Expected error response: %s", response.Content)
}

// TestBrowserTool_TabMissingIDForClose tests error when tabId is missing for tab_close
func TestBrowserTool_TabMissingIDForClose(t *testing.T) {
	skipIfIntegrationTestsDisabled(t)

	mockServer := startMockBrowserServer(t)
	defer mockServer.Close()

	mockPermissionService := &MockPermissionService{}
	sessionConfig := session.DefaultConfig()
	tool := NewBrowserTool(mockPermissionService, mockServer.wsURL, sessionConfig, "", mockClientFactory, nil, nil)

	tempDir := t.TempDir()
	ctx := createBrowserTestContext("test-session", "test-message", tempDir)

	mockPermissionService.On("Request", mock.Anything).Return(true)

	// Try tab_close without tabId
	closeCall := interfaces.ToolCall{
		ID:    "call-close-no-id",
		Name:  BrowserToolName,
		Input: `{"action": "tab_close"}`,
	}

	response, err := tool.Run(ctx, closeCall)
	require.NoError(t, err)
	assert.True(t, response.IsError)
	assert.Contains(t, response.Content, "missing tabId", "Error should indicate missing tabId")

	t.Logf("Expected error response: %s", response.Content)
}

// TestBrowserTool_TypeInSpecificTab tests typing in a specific tab
func TestBrowserTool_TypeInSpecificTab(t *testing.T) {
	skipIfIntegrationTestsDisabled(t)

	mockServer := startMockBrowserServer(t)
	defer mockServer.Close()

	mockPermissionService := &MockPermissionService{}
	sessionConfig := session.DefaultConfig()
	tool := NewBrowserTool(mockPermissionService, mockServer.wsURL, sessionConfig, "", mockClientFactory, nil, nil)

	tempDir := t.TempDir()
	ctx := createBrowserTestContext("test-session", "test-message", tempDir)

	mockPermissionService.On("Request", mock.Anything).Return(true)

	// Create a tab
	createCall := interfaces.ToolCall{
		ID:    "call-create",
		Name:  BrowserToolName,
		Input: `{"action": "tab_create"}`,
	}
	createResp, err := tool.Run(ctx, createCall)
	require.NoError(t, err)

	// Extract tab ID
	tabID := extractTabID(createResp.Content)
	require.NotEmpty(t, tabID, "Failed to extract tab ID")

	// Navigate the tab
	navCall := interfaces.ToolCall{
		ID:    "call-nav",
		Name:  BrowserToolName,
		Input: fmt.Sprintf(`{"action": "open", "url": "https://example.com", "tabId": %q}`, tabID),
	}
	_, err = tool.Run(ctx, navCall)
	require.NoError(t, err)

	// Take screenshot to populate element cache
	screenshotCall := interfaces.ToolCall{
		ID:    "call-screenshot",
		Name:  BrowserToolName,
		Input: fmt.Sprintf(`{"action": "screenshot", "tabId": %q}`, tabID),
	}
	_, err = tool.Run(ctx, screenshotCall)
	require.NoError(t, err)

	// Type in specific tab
	typeCall := interfaces.ToolCall{
		ID:    "call-type",
		Name:  BrowserToolName,
		Input: fmt.Sprintf(`{"action": "type", "index": 2, "text": "test input", "tabId": %q}`, tabID),
	}

	response, err := tool.Run(ctx, typeCall)
	require.NoError(t, err)
	assert.False(t, response.IsError)
	assert.Contains(t, response.Content, "Successfully typed text into element 2")

	t.Logf("Type response: %s", response.Content)
}

// TestBrowserTool_MultipleTabsWorkflow tests a complete workflow with multiple tabs
func TestBrowserTool_MultipleTabsWorkflow(t *testing.T) {
	skipIfIntegrationTestsDisabled(t)

	mockServer := startMockBrowserServer(t)
	defer mockServer.Close()

	mockPermissionService := &MockPermissionService{}
	sessionConfig := session.DefaultConfig()
	tool := NewBrowserTool(mockPermissionService, mockServer.wsURL, sessionConfig, "", mockClientFactory, nil, nil)

	tempDir := t.TempDir()
	ctx := createBrowserTestContext("test-session", "test-message", tempDir)

	mockPermissionService.On("Request", mock.Anything).Return(true)

	// Step 1: Create two tabs
	tab1Call := interfaces.ToolCall{
		ID:    "call-create-1",
		Name:  BrowserToolName,
		Input: `{"action": "tab_create"}`,
	}
	tab1Resp, err := tool.Run(ctx, tab1Call)
	require.NoError(t, err)
	assert.Contains(t, tab1Resp.Content, "Created new tab")

	tab2Call := interfaces.ToolCall{
		ID:    "call-create-2",
		Name:  BrowserToolName,
		Input: `{"action": "tab_create"}`,
	}
	tab2Resp, err := tool.Run(ctx, tab2Call)
	require.NoError(t, err)
	assert.Contains(t, tab2Resp.Content, "Created new tab")

	// Step 2: List tabs to verify
	listCall := interfaces.ToolCall{
		ID:    "call-list",
		Name:  BrowserToolName,
		Input: `{"action": "tab_list"}`,
	}
	listResp, err := tool.Run(ctx, listCall)
	require.NoError(t, err)
	assert.Contains(t, listResp.Content, "Total tabs:")
	tabCount := strings.Count(listResp.Content, "tab-")
	assert.GreaterOrEqual(t, tabCount, 3, "Should have at least 3 tabs")

	// Step 3: Extract tab IDs from responses
	tab1ID := extractTabID(tab1Resp.Content)
	tab2ID := extractTabID(tab2Resp.Content)
	require.NotEmpty(t, tab1ID)
	require.NotEmpty(t, tab2ID)

	// Step 4: Navigate each tab to different URLs
	nav1Call := interfaces.ToolCall{
		ID:    "call-nav-1",
		Name:  BrowserToolName,
		Input: fmt.Sprintf(`{"action": "open", "url": "https://example.com", "tabId": %q}`, tab1ID),
	}
	nav1Resp, err := tool.Run(ctx, nav1Call)
	require.NoError(t, err)
	assert.Contains(t, nav1Resp.Content, "Successfully navigated")

	nav2Call := interfaces.ToolCall{
		ID:    "call-nav-2",
		Name:  BrowserToolName,
		Input: fmt.Sprintf(`{"action": "open", "url": "https://google.com", "tabId": %q}`, tab2ID),
	}
	nav2Resp, err := tool.Run(ctx, nav2Call)
	require.NoError(t, err)
	assert.Contains(t, nav2Resp.Content, "Successfully navigated")

	// Step 5: Take screenshots of both tabs
	screen1Call := interfaces.ToolCall{
		ID:    "call-screen-1",
		Name:  BrowserToolName,
		Input: fmt.Sprintf(`{"action": "screenshot", "tabId": %q}`, tab1ID),
	}
	screen1Resp, err := tool.Run(ctx, screen1Call)
	require.NoError(t, err)
	assert.Contains(t, screen1Resp.Content, "Screenshot captured")

	screen2Call := interfaces.ToolCall{
		ID:    "call-screen-2",
		Name:  BrowserToolName,
		Input: fmt.Sprintf(`{"action": "screenshot", "tabId": %q}`, tab2ID),
	}
	screen2Resp, err := tool.Run(ctx, screen2Call)
	require.NoError(t, err)
	assert.Contains(t, screen2Resp.Content, "Screenshot captured")

	// Step 6: Switch between tabs
	switchCall := interfaces.ToolCall{
		ID:    "call-switch",
		Name:  BrowserToolName,
		Input: fmt.Sprintf(`{"action": "tab_switch", "tabId": %q}`, tab1ID),
	}
	switchResp, err := tool.Run(ctx, switchCall)
	require.NoError(t, err)
	assert.Contains(t, switchResp.Content, "Switched to tab")

	// Step 7: Close one tab
	closeCall := interfaces.ToolCall{
		ID:    "call-close",
		Name:  BrowserToolName,
		Input: fmt.Sprintf(`{"action": "tab_close", "tabId": %q}`, tab2ID),
	}
	closeResp, err := tool.Run(ctx, closeCall)
	require.NoError(t, err)
	assert.Contains(t, closeResp.Content, "Closed tab")

	// Step 8: Verify tab was closed
	finalListCall := interfaces.ToolCall{
		ID:    "call-final-list",
		Name:  BrowserToolName,
		Input: `{"action": "tab_list"}`,
	}
	finalListResp, err := tool.Run(ctx, finalListCall)
	require.NoError(t, err)
	assert.NotContains(t, finalListResp.Content, tab2ID, "Closed tab should not be in list")

	t.Log("Successfully completed multi-tab workflow")
}
