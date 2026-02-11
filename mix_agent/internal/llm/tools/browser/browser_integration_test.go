package browser

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/gorilla/websocket"
	browserprotocol "github.com/sarathmenon/browser-service/pkg/protocol"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"mix/internal/llm/interfaces"
	"mix/internal/session"
)


// skipIfIntegrationTestsDisabled skips the test if integration tests are disabled
func skipIfIntegrationTestsDisabled(t *testing.T) {
	t.Helper()
	if os.Getenv("SKIP_INTEGRATION_TESTS") != "" {
		t.Skip("Skipping integration test")
	}
}

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool {
		return true
	},
}

// mockBrowserServer creates a mock browser service server for testing
type mockBrowserServer struct {
	server               *httptest.Server
	wsURL                string
	connections          map[*websocket.Conn]bool
	requestCount         int
	tabCounter           int
	tabs                 map[string]map[string]any
	activeTabID          string
	lastClickedBackendID int64
	mu                   sync.Mutex
}

// startMockBrowserServer starts a mock browser service server
func startMockBrowserServer(t *testing.T) *mockBrowserServer {
	t.Helper()

	mbs := &mockBrowserServer{
		connections: make(map[*websocket.Conn]bool),
		tabs:        make(map[string]map[string]any),
		tabCounter:  0,
	}

	// Create initial default tab
	mbs.tabs["tab-1"] = map[string]any{
		"id":       "tab-1",
		"url":      "about:blank",
		"title":    "New Tab",
		"isActive": true,
	}
	mbs.activeTabID = "tab-1"

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		conn, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			t.Logf("Failed to upgrade connection: %v", err)
			return
		}

		mbs.mu.Lock()
		mbs.connections[conn] = true
		mbs.mu.Unlock()
		defer func() {
			mbs.mu.Lock()
			delete(mbs.connections, conn)
			mbs.mu.Unlock()
			_ = conn.Close()
		}()

		// Handle WebSocket messages
		for {
			_, message, err := conn.ReadMessage()
			if err != nil {
				return
			}

			mbs.mu.Lock()
			mbs.requestCount++
			mbs.mu.Unlock()

			// Parse request
			var req browserprotocol.Request
			if err := json.Unmarshal(message, &req); err != nil {
				continue
			}

			// Generate response based on method
			var response browserprotocol.Response
			response.ID = req.ID

			switch req.Method {
			case "Page.navigate":
				response.Result = map[string]any{
					"frameId": "frame-123",
				}

			case "Page.screenshot":
				// Create 50 elements: 20 in viewport, 30 out of viewport
				rawNodes := make([]map[string]any, 50)
				roles := []string{"button", "link", "textbox", "checkbox"}
				backendIDBase := 1000

				// Elements 0-19: In viewport (y: 0-800)
				for i := range 20 {
					role := roles[i%len(roles)]
					rawNodes[i] = map[string]any{
						"role":      role,
						"name":      fmt.Sprintf("%s %d", role, i),
						"backendId": int64(backendIDBase + (i * 5)),
						"bounds": map[string]any{
							"x":      float64(50 + (i%10)*90),
							"y":      float64(50 + (i/10)*400),
							"width":  80.0,
							"height": 30.0,
							},
						}
				}

				// Elements 20-49: Out of viewport (y: 1200+)
				for i := 20; i < 50; i++ {
					role := roles[i%len(roles)]
					rawNodes[i] = map[string]any{
						"role":      role,
						"name":      fmt.Sprintf("%s %d", role, i),
						"backendId": int64(backendIDBase + (i * 5)),
						"bounds": map[string]any{
							"x":      float64(50 + (i%10)*90),
							"y":      float64(1200 + ((i-20)/10)*100),
							"width":  80.0,
							"height": 30.0,
							},
						}
				}

				response.Result = map[string]any{
					"data":        "iVBORw0KGgoAAAANSUhEUgAAAAEAAAABCAYAAAAfFcSJAAAADUlEQVR42mNk+M9QDwADhgGAWjR9awAAAABJRU5ErkJggg==",
					"format":      "png",
					"rawNodes":    rawNodes,
					"rawViewport": map[string]any{
						"x":      0.0,
						"y":      0.0,
						"width":  1920.0,
						"height": 1080.0,
						},
					}
			case "Page.click":
				response.Result = map[string]any{
					"success": true,
				}

			case "Page.clickByBackendID", "Page.rightClickByBackendID", "Page.doubleClickByBackendID", "Page.tripleClickByBackendID":
				// Track which BackendID was clicked
				var clickParams struct {
					BackendID int64 `json:"backendId"`
				}
				if paramData, err := json.Marshal(req.Params); err == nil {
					_ = json.Unmarshal(paramData, &clickParams)
				}

				mbs.mu.Lock()
				mbs.lastClickedBackendID = clickParams.BackendID
				mbs.mu.Unlock()

				response.Result = map[string]any{
					"success": true,
				}

			case "Page.type":
				response.Result = map[string]any{
					"success": true,
				}

			case "Page.scroll":
				response.Result = map[string]any{
					"success": true,
				}

			case "Page.getElements":
				response.Result = map[string]any{
					"elements": []map[string]any{},
				}

			case "Browser.close":
				response.Result = map[string]any{
					"success": true,
				}

			case "Page.uploadFile":
				response.Result = map[string]any{
					"filesUploaded": 1,
					"fileNames":     []string{"test.txt"},
				}

			case "Page.getText":
				response.Result = map[string]any{
					"text":      "This is extracted text from the page with meaningful content.",
					"length":    60,
					"source":    "article",
					"truncated": false,
				}

			case "Page.find":
				response.Result = map[string]any{
					"elements": []map[string]any{
						{
							"index": 1,
							"role":  "button",
							"name":  "Search Button",
							"bounds": map[string]any{
								"x":      50.0,
								"y":      100.0,
								"width":  100.0,
								"height": 40.0,
							},
						},
						{
							"index": 2,
							"role":  "textbox",
							"name":  "search input",
							"bounds": map[string]any{
								"x":      50.0,
								"y":      150.0,
								"width":  200.0,
								"height": 30.0,
							},
						},
					},
					"total":     2,
					"truncated": false,
				}

			case "Page.readPage":
				// Generate mock elements with attributes
				rawNodes := []map[string]any{
					{
						"role":      "link",
						"name":      "Example Link",
						"backendId": int64(1005),
						"frameId":   "frame-123",
						"bounds":    map[string]any{"x": 50.0, "y": 100.0, "width": 120.0, "height": 30.0},
						"attributes": map[string]string{
							"href": "https://example.com",
							"id":   "example-link",
						},
					},
					{
						"role":      "textbox",
						"name":      "Search",
						"backendId": int64(1010),
						"frameId":   "frame-123",
						"bounds":    map[string]any{"x": 50.0, "y": 150.0, "width": 200.0, "height": 30.0},
						"attributes": map[string]string{
							"id":          "search",
							"type":        "search",
							"placeholder": "Enter search term",
						},
					},
					{
						"role":      "button",
						"name":      "Submit",
						"backendId": int64(1015),
						"frameId":   "frame-123",
						"bounds":    map[string]any{"x": 260.0, "y": 150.0, "width": 80.0, "height": 30.0},
						// No attributes
					},
				}

				response.Result = map[string]any{
					"elements": rawNodes,
					"viewport": map[string]any{"x": 0.0, "y": 0.0, "width": 1280.0, "height": 720.0},
				}

			case "Tab.create":
				mbs.mu.Lock()
				mbs.tabCounter++
				newTabID := fmt.Sprintf("tab-%d", mbs.tabCounter+1)
				mbs.tabs[newTabID] = map[string]any{
					"id":       newTabID,
					"url":      "about:blank",
					"title":    "New Tab",
					"isActive": false,
				}
				mbs.mu.Unlock()

				response.Result = map[string]any{
					"tab": map[string]any{
						"id":       newTabID,
						"url":      "about:blank",
						"title":    "New Tab",
						"isActive": false,
					},
				}

			case "Tab.list":
				mbs.mu.Lock()
				tabs := make([]map[string]any, 0, len(mbs.tabs))
				for _, tab := range mbs.tabs {
					tabs = append(tabs, tab)
				}
				mbs.mu.Unlock()

				response.Result = map[string]any{
					"tabs":        tabs,
					"activeTabId": mbs.activeTabID,
				}

			case "Tab.switch":
				var params struct {
					TabID string `json:"tabId"`
				}
				if paramData, err := json.Marshal(req.Params); err == nil {
					_ = json.Unmarshal(paramData, &params)
				}

				mbs.mu.Lock()
				if _, exists := mbs.tabs[params.TabID]; exists {
					// Update active status
					for id := range mbs.tabs {
						tab := mbs.tabs[id]
						tab["isActive"] = (id == params.TabID)
						mbs.tabs[id] = tab
					}
					mbs.activeTabID = params.TabID
					mbs.mu.Unlock()

					response.Result = map[string]any{
						"success": true,
					}
				} else {
					mbs.mu.Unlock()
					response.Error = &browserprotocol.Error{
						Code:    -1,
						Message: "tab not found",
					}
				}

			case "Tab.close":
				var params struct {
					TabID string `json:"tabId"`
				}
				if paramData, err := json.Marshal(req.Params); err == nil {
					_ = json.Unmarshal(paramData, &params)
				}

				mbs.mu.Lock()
				if len(mbs.tabs) == 1 {
					mbs.mu.Unlock()
					response.Error = &browserprotocol.Error{
						Code:    -1,
						Message: "cannot close last tab",
					}
				} else if _, exists := mbs.tabs[params.TabID]; exists {
					delete(mbs.tabs, params.TabID)

					// If we closed the active tab, switch to first remaining tab
					if mbs.activeTabID == params.TabID {
						for id := range mbs.tabs {
							mbs.activeTabID = id
							tab := mbs.tabs[id]
							tab["isActive"] = true
							mbs.tabs[id] = tab
							break
						}
					}
					mbs.mu.Unlock()

					response.Result = map[string]any{
						"success": true,
					}
				} else {
					mbs.mu.Unlock()
					response.Error = &browserprotocol.Error{
						Code:    -1,
						Message: "tab not found",
					}
				}

			case "Page.pressKey":
				response.Result = map[string]any{
					"success": true,
				}

			case "Page.scrollToElement", "Page.scrollIntoView":
				response.Result = map[string]any{
					"success": true,
				}

			case "Page.executeActions":
				var params struct {
					Actions []map[string]any `json:"actions"`
				}
				if paramData, err := json.Marshal(req.Params); err == nil {
					_ = json.Unmarshal(paramData, &params)
				}

				response.Result = map[string]any{
					"success":        true,
					"actionsExecuted": len(params.Actions),
				}

			default:
				response.Error = &browserprotocol.Error{
					Code:    -1,
					Message: fmt.Sprintf("Unknown method: %s", req.Method),
				}
			}

			// Send response
			respData, _ := json.Marshal(response)
			if err := conn.WriteMessage(websocket.TextMessage, respData); err != nil {
				return
			}
		}
	})

	mbs.server = httptest.NewServer(handler)

	// Convert HTTP URL to WebSocket URL
	mbs.wsURL = "ws" + strings.TrimPrefix(mbs.server.URL, "http")

	return mbs
}

func (mbs *mockBrowserServer) Close() {
	mbs.mu.Lock()
	conns := make([]*websocket.Conn, 0, len(mbs.connections))
	for conn := range mbs.connections {
		conns = append(conns, conn)
	}
	mbs.mu.Unlock()

	for _, conn := range conns {
		_ = conn.Close()
	}
	mbs.server.Close()
}

// close is an alias for Close to match test expectations
func (mbs *mockBrowserServer) close() {
	mbs.Close()
}

// getRequestCount returns the number of requests received
func (mbs *mockBrowserServer) getRequestCount() int {
	mbs.mu.Lock()
	defer mbs.mu.Unlock()
	return mbs.requestCount
}

// GetLastClickedBackendID returns the last clicked backend ID
func (mbs *mockBrowserServer) GetLastClickedBackendID() int64 {
	mbs.mu.Lock()
	defer mbs.mu.Unlock()
	return mbs.lastClickedBackendID
}

// Test full browser workflow: open -> screenshot -> click -> type -> scroll -> close
func TestBrowserToolIntegrationFullWorkflow(t *testing.T) {
	skipIfIntegrationTestsDisabled(t)

	// Start mock browser service
	mockServer := startMockBrowserServer(t)
	defer mockServer.Close()

	// Create tool
	mockPermissionService := &MockPermissionService{}
	sessionConfig := session.DefaultConfig()
	tool := NewBrowserTool(mockPermissionService, &MockSessionService{}, mockServer.wsURL, sessionConfig, "", mockClientFactory, nil, nil)

	// Create test context and temporary directory
	tempDir := t.TempDir()
	ctx := createBrowserTestContext("test-session", "test-message", tempDir)

	// Mock permissions to always grant
	mockPermissionService.On("Request", mock.Anything).Return(true)

	// 1. Open URL
	t.Run("open", func(t *testing.T) {
		call := interfaces.ToolCall{
			ID:    "call-open",
			Name:  BrowserToolName,
			Input: `{"action": "open", "url": "https://example.com"}`,
		}

		response, err := tool.Run(ctx, call)
		require.NoError(t, err)
		assert.False(t, response.IsError)
		assert.Contains(t, response.Content, "Successfully navigated")
		assert.Contains(t, response.Content, "https://example.com")
	})

	// 2. Take screenshot
	t.Run("screenshot", func(t *testing.T) {
		call := interfaces.ToolCall{
			ID:    "call-screenshot",
			Name:  BrowserToolName,
			Input: `{"action": "screenshot"}`,
		}

		response, err := tool.Run(ctx, call)
		require.NoError(t, err)
		assert.False(t, response.IsError)
		assert.Contains(t, response.Content, "Screenshot captured")
		assert.Contains(t, response.Content, "Display URL:")

		// Verify screenshot file was created
		files, err := os.ReadDir(tempDir)
		require.NoError(t, err)
		assert.NotEmpty(t, files)

		// Find screenshot file
		foundScreenshot := false
		for _, file := range files {
			if strings.HasPrefix(file.Name(), "screenshot_") && strings.HasSuffix(file.Name(), ".png") {
				foundScreenshot = true
				break
			}
		}
		assert.True(t, foundScreenshot, "Screenshot file should be created")
	})

	// 3. Click element
	t.Run("click", func(t *testing.T) {
		call := interfaces.ToolCall{
			ID:    "call-click",
			Name:  BrowserToolName,
			Input: `{"action": "click", "index": 1}`,
		}

		response, err := tool.Run(ctx, call)
		require.NoError(t, err)
		assert.False(t, response.IsError)
		assert.Contains(t, response.Content, "Successfully clicked element 1")
	})

	// 4. Type text
	t.Run("type", func(t *testing.T) {
		call := interfaces.ToolCall{
			ID:    "call-type",
			Name:  BrowserToolName,
			Input: `{"action": "type", "index": 2, "text": "test input"}`,
		}

		response, err := tool.Run(ctx, call)
		require.NoError(t, err)
		assert.False(t, response.IsError)
		assert.Contains(t, response.Content, "Successfully typed text into element 2")
	})

	// 5. Scroll
	t.Run("scroll", func(t *testing.T) {
		call := interfaces.ToolCall{
			ID:    "call-scroll",
			Name:  BrowserToolName,
			Input: `{"action": "scroll", "direction": "down", "amount": 500}`,
		}

		response, err := tool.Run(ctx, call)
		require.NoError(t, err)
		assert.False(t, response.IsError)
		assert.Contains(t, response.Content, "Successfully scrolled down by 500 pixels")
	})

	// 6. Close browser
	t.Run("close", func(t *testing.T) {
		call := interfaces.ToolCall{
			ID:    "call-close",
			Name:  BrowserToolName,
			Input: `{"action": "close"}`,
		}

		response, err := tool.Run(ctx, call)
		require.NoError(t, err)
		assert.False(t, response.IsError)
		assert.Contains(t, response.Content, "Browser closed successfully")
	})
}

// Test screenshot with and without overlay
func TestBrowserToolIntegrationScreenshotOverlay(t *testing.T) {
	skipIfIntegrationTestsDisabled(t)

	mockServer := startMockBrowserServer(t)
	defer mockServer.Close()

	mockPermissionService := &MockPermissionService{}
	sessionConfig := session.DefaultConfig()
	tool := NewBrowserTool(mockPermissionService, &MockSessionService{}, mockServer.wsURL, sessionConfig, "", mockClientFactory, nil, nil)

	tempDir := t.TempDir()
	ctx := createBrowserTestContext("test-session", "test-message", tempDir)

	mockPermissionService.On("Request", mock.Anything).Return(true)

	// Open page first
	openCall := interfaces.ToolCall{
		ID:    "call-open",
		Name:  BrowserToolName,
		Input: `{"action": "open", "url": "https://example.com"}`,
	}
	_, err := tool.Run(ctx, openCall)
	require.NoError(t, err)

	// Screenshot (raw mode)
	t.Run("with overlay", func(t *testing.T) {
		call := interfaces.ToolCall{
			ID:    "call-screenshot-1",
			Name:  BrowserToolName,
			Input: `{"action": "screenshot"}`,
		}

		response, err := tool.Run(ctx, call)
		require.NoError(t, err)
		assert.False(t, response.IsError)
		assert.Contains(t, response.Content, "Screenshot captured successfully")
	})

	// Screenshot without overlay (same behavior as raw mode doesn't support overlay)
	t.Run("without overlay", func(t *testing.T) {
		call := interfaces.ToolCall{
			ID:    "call-screenshot-2",
			Name:  BrowserToolName,
			Input: `{"action": "screenshot"}`,
		}

		response, err := tool.Run(ctx, call)
		require.NoError(t, err)
		assert.False(t, response.IsError)
		assert.Contains(t, response.Content, "Screenshot captured successfully")
	})
}

// Test error handling when browser service is down
func TestBrowserToolIntegrationServiceDown(t *testing.T) {
	skipIfIntegrationTestsDisabled(t)

	mockPermissionService := &MockPermissionService{}
	sessionConfig := session.DefaultConfig()

	// Use invalid endpoint
	tool := NewBrowserTool(mockPermissionService, &MockSessionService{}, "ws://localhost:99999", sessionConfig, "", mockClientFactory, nil, nil)

	tempDir := t.TempDir()
	ctx := createBrowserTestContext("test-session", "test-message", tempDir)

	mockPermissionService.On("Request", mock.Anything).Return(true)

	call := interfaces.ToolCall{
		ID:    "call-open",
		Name:  BrowserToolName,
		Input: `{"action": "open", "url": "https://example.com"}`,
	}

	response, err := tool.Run(ctx, call)
	require.NoError(t, err)
	assert.True(t, response.IsError)
	assert.Contains(t, response.Content, "Failed to get browser client")
}

// Test session isolation - multiple sessions should have separate connections
func TestBrowserToolIntegrationSessionIsolation(t *testing.T) {
	skipIfIntegrationTestsDisabled(t)

	mockServer := startMockBrowserServer(t)
	defer mockServer.Close()

	mockPermissionService := &MockPermissionService{}
	sessionConfig := session.DefaultConfig()
	tool := NewBrowserTool(mockPermissionService, &MockSessionService{}, mockServer.wsURL, sessionConfig, "", mockClientFactory, nil, nil)

	mockPermissionService.On("Request", mock.Anything).Return(true)

	// Session 1
	tempDir1 := t.TempDir()
	ctx1 := createBrowserTestContext("session-1", "message-1", tempDir1)

	call1 := interfaces.ToolCall{
		ID:    "call-1",
		Name:  BrowserToolName,
		Input: `{"action": "open", "url": "https://example1.com"}`,
	}

	response1, err := tool.Run(ctx1, call1)
	require.NoError(t, err)
	assert.False(t, response1.IsError)

	// Session 2
	tempDir2 := t.TempDir()
	ctx2 := createBrowserTestContext("session-2", "message-2", tempDir2)

	call2 := interfaces.ToolCall{
		ID:    "call-2",
		Name:  BrowserToolName,
		Input: `{"action": "open", "url": "https://example2.com"}`,
	}

	response2, err := tool.Run(ctx2, call2)
	require.NoError(t, err)
	assert.False(t, response2.IsError)

	// Take screenshots in both sessions
	screenshotCall1 := interfaces.ToolCall{
		ID:    "call-screenshot-1",
		Name:  BrowserToolName,
		Input: `{"action": "screenshot"}`,
	}
	_, err = tool.Run(ctx1, screenshotCall1)
	require.NoError(t, err)

	screenshotCall2 := interfaces.ToolCall{
		ID:    "call-screenshot-2",
		Name:  BrowserToolName,
		Input: `{"action": "screenshot"}`,
	}
	_, err = tool.Run(ctx2, screenshotCall2)
	require.NoError(t, err)

	// Verify screenshots are in separate directories
	files1, err := os.ReadDir(tempDir1)
	require.NoError(t, err)
	assert.NotEmpty(t, files1)

	files2, err := os.ReadDir(tempDir2)
	require.NoError(t, err)
	assert.NotEmpty(t, files2)
}

// Test connection reuse
func TestBrowserToolIntegrationConnectionReuse(t *testing.T) {
	skipIfIntegrationTestsDisabled(t)

	mockServer := startMockBrowserServer(t)
	defer mockServer.Close()

	mockPermissionService := &MockPermissionService{}
	sessionConfig := session.DefaultConfig()
	tool := NewBrowserTool(mockPermissionService, &MockSessionService{}, mockServer.wsURL, sessionConfig, "", mockClientFactory, nil, nil)

	tempDir := t.TempDir()
	ctx := createBrowserTestContext("test-session", "test-message", tempDir)

	mockPermissionService.On("Request", mock.Anything).Return(true)

	// Open page
	openCall := interfaces.ToolCall{
		ID:    "call-open",
		Name:  BrowserToolName,
		Input: `{"action": "open", "url": "https://example.com"}`,
	}
	_, err := tool.Run(ctx, openCall)
	require.NoError(t, err)

	initialRequestCount := mockServer.requestCount

	// Take multiple screenshots - should reuse same connection
	for i := range 3 {
		screenshotCall := interfaces.ToolCall{
			ID:    fmt.Sprintf("call-screenshot-%d", i),
			Name:  BrowserToolName,
			Input: `{"action": "screenshot"}`,
		}
		response, err := tool.Run(ctx, screenshotCall)
		require.NoError(t, err)
		assert.False(t, response.IsError)
	}

	// Verify requests were made (connection was reused)
	assert.Greater(t, mockServer.requestCount, initialRequestCount)
}

// Test scroll with default amount
func TestBrowserToolIntegrationScrollDefaultAmount(t *testing.T) {
	skipIfIntegrationTestsDisabled(t)

	mockServer := startMockBrowserServer(t)
	defer mockServer.Close()

	mockPermissionService := &MockPermissionService{}
	sessionConfig := session.DefaultConfig()
	tool := NewBrowserTool(mockPermissionService, &MockSessionService{}, mockServer.wsURL, sessionConfig, "", mockClientFactory, nil, nil)

	tempDir := t.TempDir()
	ctx := createBrowserTestContext("test-session", "test-message", tempDir)

	mockPermissionService.On("Request", mock.Anything).Return(true)

	// Open page first
	openCall := interfaces.ToolCall{
		ID:    "call-open",
		Name:  BrowserToolName,
		Input: `{"action": "open", "url": "https://example.com"}`,
	}
	_, err := tool.Run(ctx, openCall)
	require.NoError(t, err)

	// Scroll without specifying amount (should default to 100)
	scrollCall := interfaces.ToolCall{
		ID:    "call-scroll",
		Name:  BrowserToolName,
		Input: `{"action": "scroll", "direction": "down"}`,
	}

	response, err := tool.Run(ctx, scrollCall)
	require.NoError(t, err)
	assert.False(t, response.IsError)
	assert.Contains(t, response.Content, "Successfully scrolled down by 100 pixels")
}

// Test all scroll directions
func TestBrowserToolIntegrationScrollDirections(t *testing.T) {
	skipIfIntegrationTestsDisabled(t)

	mockServer := startMockBrowserServer(t)
	defer mockServer.Close()

	mockPermissionService := &MockPermissionService{}
	sessionConfig := session.DefaultConfig()
	tool := NewBrowserTool(mockPermissionService, &MockSessionService{}, mockServer.wsURL, sessionConfig, "", mockClientFactory, nil, nil)

	tempDir := t.TempDir()
	ctx := createBrowserTestContext("test-session", "test-message", tempDir)

	mockPermissionService.On("Request", mock.Anything).Return(true)

	// Open page first
	openCall := interfaces.ToolCall{
		ID:    "call-open",
		Name:  BrowserToolName,
		Input: `{"action": "open", "url": "https://example.com"}`,
	}
	_, err := tool.Run(ctx, openCall)
	require.NoError(t, err)

	directions := []string{"up", "down", "left", "right"}

	for _, direction := range directions {
		t.Run(direction, func(t *testing.T) {
			scrollCall := interfaces.ToolCall{
				ID:    fmt.Sprintf("call-scroll-%s", direction),
				Name:  BrowserToolName,
				Input: fmt.Sprintf(`{"action": "scroll", "direction": %q, "amount": 200}`, direction),
			}

			response, err := tool.Run(ctx, scrollCall)
			require.NoError(t, err)
			assert.False(t, response.IsError)
			assert.Contains(t, response.Content, fmt.Sprintf("Successfully scrolled %s by 200 pixels", direction))
		})
	}
}

// Test timeout handling
func TestBrowserToolIntegrationTimeout(t *testing.T) {
	skipIfIntegrationTestsDisabled(t)

	mockServer := startMockBrowserServer(t)
	defer mockServer.Close()

	mockPermissionService := &MockPermissionService{}
	sessionConfig := session.DefaultConfig()
	tool := NewBrowserTool(mockPermissionService, &MockSessionService{}, mockServer.wsURL, sessionConfig, "", mockClientFactory, nil, nil)

	tempDir := t.TempDir()

	// Create context with very short timeout
	baseCtx := createBrowserTestContext("test-session", "test-message", tempDir)
	ctx, cancel := context.WithTimeout(baseCtx, 1*time.Nanosecond)
	defer cancel()

	// Wait for timeout to occur
	time.Sleep(10 * time.Millisecond)

	mockPermissionService.On("Request", mock.Anything).Return(true)

	call := interfaces.ToolCall{
		ID:    "call-open",
		Name:  BrowserToolName,
		Input: `{"action": "open", "url": "https://example.com"}`,
	}

	response, err := tool.Run(ctx, call)
	// Should either timeout or fail with error
	if err == nil {
		assert.True(t, response.IsError)
	}
}

// Test screenshot file URL formatting
func TestBrowserToolIntegrationScreenshotURL(t *testing.T) {
	skipIfIntegrationTestsDisabled(t)

	mockServer := startMockBrowserServer(t)
	defer mockServer.Close()

	mockPermissionService := &MockPermissionService{}
	sessionConfig := session.DefaultConfig()

	// Set custom base URL
	t.Setenv("FRONTEND_URL", "https://custom.example.com")

	tool := NewBrowserTool(mockPermissionService, &MockSessionService{}, mockServer.wsURL, sessionConfig, "", mockClientFactory, nil, nil)

	tempDir := t.TempDir()
	ctx := createBrowserTestContext("test-session-123", "test-message", tempDir)

	mockPermissionService.On("Request", mock.Anything).Return(true)

	// Open and screenshot
	openCall := interfaces.ToolCall{
		ID:    "call-open",
		Name:  BrowserToolName,
		Input: `{"action": "open", "url": "https://example.com"}`,
	}
	_, err := tool.Run(ctx, openCall)
	require.NoError(t, err)

	screenshotCall := interfaces.ToolCall{
		ID:    "call-screenshot",
		Name:  BrowserToolName,
		Input: `{"action": "screenshot"}`,
	}

	response, err := tool.Run(ctx, screenshotCall)
	require.NoError(t, err)
	assert.False(t, response.IsError)

	// Verify URL format
	assert.Contains(t, response.Content, "https://custom.example.com/api/sessions/test-session-123/files/screenshot_")
}

// Test file upload feature
func TestBrowserToolIntegrationFileUpload(t *testing.T) {
	skipIfIntegrationTestsDisabled(t)

	mockServer := startMockBrowserServer(t)
	defer mockServer.Close()

	mockPermissionService := &MockPermissionService{}
	sessionConfig := session.DefaultConfig()
	tool := NewBrowserTool(mockPermissionService, &MockSessionService{}, mockServer.wsURL, sessionConfig, "", mockClientFactory, nil, nil)

	tempDir := t.TempDir()
	ctx := createBrowserTestContext("test-session", "test-message", tempDir)

	mockPermissionService.On("Request", mock.Anything).Return(true)

	// Create a test file in the session directory
	testFilePath := filepath.Join(tempDir, "test.txt")
	err := os.WriteFile(testFilePath, []byte("test content"), 0o644)
	require.NoError(t, err)

	// Open page first
	openCall := interfaces.ToolCall{
		ID:    "call-open",
		Name:  BrowserToolName,
		Input: `{"action": "open", "url": "https://example.com"}`,
	}
	_, err = tool.Run(ctx, openCall)
	require.NoError(t, err)

	// Test upload with absolute path
	t.Run("absolute path", func(t *testing.T) {
		uploadCall := interfaces.ToolCall{
			ID:    "call-upload-abs",
			Name:  BrowserToolName,
			Input: fmt.Sprintf(`{"action": "upload", "index": 1, "filePath": %q}`, testFilePath),
		}

		response, err := tool.Run(ctx, uploadCall)
		require.NoError(t, err)
		assert.False(t, response.IsError)
		assert.Contains(t, response.Content, "Successfully uploaded")
		assert.Contains(t, response.Content, "test.txt")
	})

	// Test upload with relative path
	t.Run("relative path", func(t *testing.T) {
		uploadCall := interfaces.ToolCall{
			ID:    "call-upload-rel",
			Name:  BrowserToolName,
			Input: `{"action": "upload", "index": 1, "filePath": "test.txt"}`,
		}

		response, err := tool.Run(ctx, uploadCall)
		require.NoError(t, err)
		assert.False(t, response.IsError)
		assert.Contains(t, response.Content, "Successfully uploaded")
	})

	// Test upload with non-existent file
	t.Run("non-existent file", func(t *testing.T) {
		uploadCall := interfaces.ToolCall{
			ID:    "call-upload-404",
			Name:  BrowserToolName,
			Input: `{"action": "upload", "index": 1, "filePath": "nonexistent.txt"}`,
		}

		response, err := tool.Run(ctx, uploadCall)
		require.NoError(t, err)
		assert.True(t, response.IsError)
		assert.Contains(t, response.Content, "File not found")
	})

	// Test upload with missing filePath parameter
	t.Run("missing filePath", func(t *testing.T) {
		uploadCall := interfaces.ToolCall{
			ID:    "call-upload-missing",
			Name:  BrowserToolName,
			Input: `{"action": "upload", "index": 1}`,
		}

		response, err := tool.Run(ctx, uploadCall)
		require.NoError(t, err)
		assert.True(t, response.IsError)
		assert.Contains(t, response.Content, "missing filePath parameter")
	})
}

// Test text extraction feature with different strategies
func TestBrowserToolIntegrationTextExtraction(t *testing.T) {
	skipIfIntegrationTestsDisabled(t)

	mockServer := startMockBrowserServer(t)
	defer mockServer.Close()

	mockPermissionService := &MockPermissionService{}
	sessionConfig := session.DefaultConfig()
	tool := NewBrowserTool(mockPermissionService, &MockSessionService{}, mockServer.wsURL, sessionConfig, "", mockClientFactory, nil, nil)

	tempDir := t.TempDir()
	ctx := createBrowserTestContext("test-session", "test-message", tempDir)

	mockPermissionService.On("Request", mock.Anything).Return(true)

	// Open page first
	openCall := interfaces.ToolCall{
		ID:    "call-open",
		Name:  BrowserToolName,
		Input: `{"action": "open", "url": "https://example.com"}`,
	}
	_, err := tool.Run(ctx, openCall)
	require.NoError(t, err)

	// Test different extraction strategies
	strategies := []string{"auto", "article", "main", "body"}

	for _, strategy := range strategies {
		t.Run(strategy, func(t *testing.T) {
			getTextCall := interfaces.ToolCall{
				ID:    fmt.Sprintf("call-gettext-%s", strategy),
				Name:  BrowserToolName,
				Input: fmt.Sprintf(`{"action": "get_text", "strategy": %q}`, strategy),
			}

			response, err := tool.Run(ctx, getTextCall)
			require.NoError(t, err)
			assert.False(t, response.IsError)
			assert.Contains(t, response.Content, "Extracted")
			assert.Contains(t, response.Content, "characters from page")
			assert.Contains(t, response.Content, "=== Page Text ===")
			assert.Contains(t, response.Content, "This is extracted text")
		})
	}

	// Test default strategy (should default to auto)
	t.Run("default strategy", func(t *testing.T) {
		getTextCall := interfaces.ToolCall{
			ID:    "call-gettext-default",
			Name:  BrowserToolName,
			Input: `{"action": "get_text"}`,
		}

		response, err := tool.Run(ctx, getTextCall)
		require.NoError(t, err)
		assert.False(t, response.IsError)
		assert.Contains(t, response.Content, "Extracted")
	})
}

// Test DOM search feature
func TestBrowserToolIntegrationDOMSearch(t *testing.T) {
	skipIfIntegrationTestsDisabled(t)

	mockServer := startMockBrowserServer(t)
	defer mockServer.Close()

	mockPermissionService := &MockPermissionService{}
	sessionConfig := session.DefaultConfig()
	tool := NewBrowserTool(mockPermissionService, &MockSessionService{}, mockServer.wsURL, sessionConfig, "", mockClientFactory, nil, nil)

	tempDir := t.TempDir()
	ctx := createBrowserTestContext("test-session", "test-message", tempDir)

	mockPermissionService.On("Request", mock.Anything).Return(true)

	// Open page first
	openCall := interfaces.ToolCall{
		ID:    "call-open",
		Name:  BrowserToolName,
		Input: `{"action": "open", "url": "https://example.com"}`,
	}
	_, err := tool.Run(ctx, openCall)
	require.NoError(t, err)

	// Test successful search
	t.Run("successful search", func(t *testing.T) {
		findCall := interfaces.ToolCall{
			ID:    "call-find-success",
			Name:  BrowserToolName,
			Input: `{"action": "find", "query": "search"}`,
		}

		response, err := tool.Run(ctx, findCall)
		require.NoError(t, err)
		assert.False(t, response.IsError)
		assert.Contains(t, response.Content, "Found 2 element(s)")
		assert.Contains(t, response.Content, "matching query: search")
		assert.Contains(t, response.Content, "[1] button: Search Button")
		assert.Contains(t, response.Content, "[2] textbox: search input")
	})

	// Test search with missing query parameter
	t.Run("missing query", func(t *testing.T) {
		findCall := interfaces.ToolCall{
			ID:    "call-find-missing",
			Name:  BrowserToolName,
			Input: `{"action": "find"}`,
		}

		response, err := tool.Run(ctx, findCall)
		require.NoError(t, err)
		assert.True(t, response.IsError)
		assert.Contains(t, response.Content, "missing query parameter")
	})
}

// Test that find returns appropriate message when no elements found
func TestBrowserToolIntegrationDOMSearchNoResults(t *testing.T) {
	skipIfIntegrationTestsDisabled(t)

	// Create a custom mock server that returns empty results
	upgrader := websocket.Upgrader{
		CheckOrigin: func(r *http.Request) bool {
			return true
		},
	}

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		conn, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			return
		}
		defer func() { _ = conn.Close() }()

		for {
			_, message, err := conn.ReadMessage()
			if err != nil {
				return
			}

			var req browserprotocol.Request
			if err := json.Unmarshal(message, &req); err != nil {
				continue
			}

			var response browserprotocol.Response
			response.ID = req.ID

			// Return empty results for find
			switch req.Method {
			case "Page.find":
				response.Result = map[string]any{
					"elements":  []map[string]any{},
					"total":     0,
					"truncated": false,
				}
			case "Page.navigate":
				response.Result = map[string]any{
					"frameId": "frame-123",
				}
			}

			respData, _ := json.Marshal(response)
			if err := conn.WriteMessage(websocket.TextMessage, respData); err != nil {
				return
			}
		}
	})

	server := httptest.NewServer(handler)
	defer server.Close()

	wsURL := "ws" + strings.TrimPrefix(server.URL, "http")

	mockPermissionService := &MockPermissionService{}
	sessionConfig := session.DefaultConfig()
	tool := NewBrowserTool(mockPermissionService, &MockSessionService{}, wsURL, sessionConfig, "", mockClientFactory, nil, nil)

	tempDir := t.TempDir()
	ctx := createBrowserTestContext("test-session", "test-message", tempDir)

	mockPermissionService.On("Request", mock.Anything).Return(true)

	// Open page first
	openCall := interfaces.ToolCall{
		ID:    "call-open",
		Name:  BrowserToolName,
		Input: `{"action": "open", "url": "https://example.com"}`,
	}
	_, err := tool.Run(ctx, openCall)
	require.NoError(t, err)

	// Search for non-existent elements
	findCall := interfaces.ToolCall{
		ID:    "call-find-empty",
		Name:  BrowserToolName,
		Input: `{"action": "find", "query": "nonexistent"}`,
	}

	response, err := tool.Run(ctx, findCall)
	require.NoError(t, err)
	assert.False(t, response.IsError)
	assert.Contains(t, response.Content, "No elements found matching query: nonexistent")
}

// TestElementCacheAccuracy verifies that the element cache correctly maps visual indices to BackendIDs
func TestElementCacheAccuracy(t *testing.T) {
	t.Helper()
	skipIfIntegrationTestsDisabled(t)

	mockServer := startMockBrowserServer(t)
	defer mockServer.Close()

	mockPermissionService := &MockPermissionService{}
	sessionConfig := session.DefaultConfig()
	tool := NewBrowserTool(mockPermissionService, &MockSessionService{}, mockServer.wsURL, sessionConfig, "", mockClientFactory, nil, nil)

	tempDir := t.TempDir()
	ctx := createBrowserTestContext("test-session", "test-message", tempDir)

	mockPermissionService.On("Request", mock.Anything).Return(true)

	// Open page first
	openCall := interfaces.ToolCall{
		ID:    "call-open",
		Name:  BrowserToolName,
		Input: `{"action": "open", "url": "https://example.com"}`,
	}
	_, err := tool.Run(ctx, openCall)
	require.NoError(t, err)

	// Take screenshot to populate cache
	screenshotCall := interfaces.ToolCall{
		ID:    "call-screenshot",
		Name:  BrowserToolName,
		Input: `{"action": "screenshot"}`,
	}
	_, err = tool.Run(ctx, screenshotCall)
	require.NoError(t, err)

	// Access the tool's internal cache (using type assertion)
	browserTool, ok := tool.(*browserTool)
	require.True(t, ok, "Failed to cast tool to *browserTool")

	// Get the active tab ID by listing tabs
	client, err := browserTool.getClient(ctx, "test-session")
	require.NoError(t, err)
	tabListResult, err := client.ListTabs(ctx)
	require.NoError(t, err)
	require.NotEmpty(t, tabListResult.ActiveTabID, "Active tab ID should not be empty")

	// Extract element cache for current tab
	cacheKey := "test-session_" + tabListResult.ActiveTabID
	browserTool.cacheMu.RLock()
	cache, exists := browserTool.elementCache[cacheKey]
	browserTool.cacheMu.RUnlock()

	require.True(t, exists, "Cache should exist after screenshot")
	require.NotEmpty(t, cache, "Cache should not be empty")

	// Assert: Cache contains only visible elements (20, not 50)
	assert.Len(t, cache, 20, "Cache should contain exactly 20 visible elements")

	// Assert: Visual index 0 maps to BackendID of first visible element (1000)
	backendID0, found := cache[0]
	require.True(t, found, "Visual index 0 should be in cache")
	assert.Equal(t, int64(1000), backendID0, "Visual index 0 should map to BackendID 1000")

	// Assert: Visual index 19 maps to BackendID of 20th visible element (1095)
	backendID19, found := cache[19]
	require.True(t, found, "Visual index 19 should be in cache")
	assert.Equal(t, int64(1095), backendID19, "Visual index 19 should map to BackendID 1095")

	// Assert: Out-of-viewport elements (BackendIDs 1100+) are NOT in cache
	for _, backendID := range cache {
		assert.Less(t, backendID, int64(1100), "Out-of-viewport elements should not be in cache")
	}

	// Verify sequential visual indices 0-19 are all present
	for i := range 20 {
		_, found := cache[i]
		require.True(t, found, "Visual index should be in cache")
	}
}

// TestClickWithViewportFiltering explicitly tests clicking filtered elements
func TestClickWithViewportFiltering(t *testing.T) {
	t.Helper()
	skipIfIntegrationTestsDisabled(t)

	mockServer := startMockBrowserServer(t)
	defer mockServer.Close()

	mockPermissionService := &MockPermissionService{}
	sessionConfig := session.DefaultConfig()
	tool := NewBrowserTool(mockPermissionService, &MockSessionService{}, mockServer.wsURL, sessionConfig, "", mockClientFactory, nil, nil)

	tempDir := t.TempDir()
	ctx := createBrowserTestContext("test-session", "test-message", tempDir)

	mockPermissionService.On("Request", mock.Anything).Return(true)

	// Open page
	openCall := interfaces.ToolCall{
		ID:    "call-open",
		Name:  BrowserToolName,
		Input: `{"action": "open", "url": "https://example.com"}`,
	}
	_, err := tool.Run(ctx, openCall)
	require.NoError(t, err)

	// Take screenshot (filters to 20 visible elements out of 50 total)
	screenshotCall := interfaces.ToolCall{
		ID:    "call-screenshot",
		Name:  BrowserToolName,
		Input: `{"action": "screenshot"}`,
	}
	_, err = tool.Run(ctx, screenshotCall)
	require.NoError(t, err)

	// Click visual index 15 (the 16th visible element)
	clickCall := interfaces.ToolCall{
		ID:    "call-click",
		Name:  BrowserToolName,
		Input: `{"action": "click", "index": 15}`,
	}
	response, err := tool.Run(ctx, clickCall)
	require.NoError(t, err)
	assert.False(t, response.IsError)

	// Get clicked BackendID from mock
	clickedBackendID := mockServer.GetLastClickedBackendID()

	// Expected: BackendID of 16th visible element
	// The 16th visible element is at array position 15 (0-indexed)
	// BackendID = 1000 + (15 * 5) = 1075
	expectedBackendID := int64(1075)

	// Assert correct BackendID was sent
	assert.Equal(t, expectedBackendID, clickedBackendID,
		"Visual index 15 should click the element that appears 16th in viewport (BackendID 1075)")

	// Verify that the click was for an in-viewport element
	assert.Less(t, clickedBackendID, int64(1100), "Clicked element should be in viewport")
}

func TestReadPageAttributeFormatting(t *testing.T) {
	t.Helper()
	skipIfIntegrationTestsDisabled(t)

	mockServer := startMockBrowserServer(t)
	defer mockServer.Close()

	tool := NewBrowserTool(&MockPermissionService{}, &MockSessionService{}, mockServer.wsURL, session.DefaultConfig(), "", mockClientFactory, nil, nil)
	ctx := createBrowserTestContext("test-session", "test-message", t.TempDir())

	// Navigate first
	navCall := interfaces.ToolCall{
		ID:    "call-nav",
		Name:  BrowserToolName,
		Input: `{"action": "open", "url": "https://example.com"}`,
	}
	_, err := tool.Run(ctx, navCall)
	require.NoError(t, err)

	// Test read_page
	call := interfaces.ToolCall{
		ID:    "call-readpage",
		Name:  BrowserToolName,
		Input: `{"action": "read_page", "interactiveOnly": true}`,
	}

	response, err := tool.Run(ctx, call)
	require.NoError(t, err)
	assert.False(t, response.IsError)

	// Verify new format
	assert.Contains(t, response.Content, `- link "Example Link" [ref=f0_ref_1005]`)
	assert.Contains(t, response.Content, `(x=50,y=100)`)
	assert.Contains(t, response.Content, `href="https://example.com"`)
	assert.Contains(t, response.Content, `id="example-link"`)
	assert.Contains(t, response.Content, `- textbox "Search" [ref=f0_ref_1010]`)
	assert.Contains(t, response.Content, `type="search"`)
	assert.Contains(t, response.Content, `placeholder="Enter search term"`)
	assert.Contains(t, response.Content, `- button "Submit" [ref=f0_ref_1015]`)

	// Verify old format is gone
	assert.NotContains(t, response.Content, "[0]")
	assert.NotContains(t, response.Content, "Position:")
	assert.NotContains(t, response.Content, "Size:")
}
