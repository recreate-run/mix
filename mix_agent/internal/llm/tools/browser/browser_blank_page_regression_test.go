package browser

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"

	"github.com/gorilla/websocket"
	browserprotocol "github.com/sarathmenon/browser-service/pkg/protocol"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"mix/internal/llm/interfaces"
	"mix/internal/session"
)


const (
	urlAboutBlank  = "about:blank"
	methodNavigate = "Page.navigate"
)

// mockBrowserServerBlankPage extends mockBrowserServer for blank page testing
type mockBrowserServerBlankPage struct {
	server         *httptest.Server
	wsURL          string
	connections    map[*websocket.Conn]bool
	requestCount   int
	tabCounter     int
	tabs           map[string]map[string]any
	activeTabID    string
	mu             sync.Mutex
	blankPageMode  bool             // Returns nil RawNodes when true
	tabURLs        map[string]string // Track URLs per tab
}

// startMockBrowserServerBlankPage starts a mock browser service server with blank page support
func startMockBrowserServerBlankPage(t *testing.T) *mockBrowserServerBlankPage {
	t.Helper()

	mbs := &mockBrowserServerBlankPage{
		connections: make(map[*websocket.Conn]bool),
		tabs:        make(map[string]map[string]any),
		tabURLs:     make(map[string]string),
		tabCounter:  0,
	}

	// Create initial default tab
	mbs.tabs["tab-1"] = map[string]any{
		"id":       "tab-1",
		"url":      urlAboutBlank,
		"title":    "New Tab",
		"isActive": true,
	}
	mbs.activeTabID = "tab-1"
	mbs.tabURLs["tab-1"] = urlAboutBlank

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
			case methodNavigate:
				var params struct {
					URL   string `json:"url"`
					TabID string `json:"tabId"`
				}
				if paramData, err := json.Marshal(req.Params); err == nil {
					_ = json.Unmarshal(paramData, &params)
				}

				// Track URL for tab
				tabID := params.TabID
				if tabID == "" {
					mbs.mu.Lock()
					tabID = mbs.activeTabID
					mbs.mu.Unlock()
				}

				mbs.mu.Lock()
				if tabID != "" {
					mbs.tabURLs[tabID] = params.URL
					if tab, exists := mbs.tabs[tabID]; exists {
						tab["url"] = params.URL
						mbs.tabs[tabID] = tab
					}
				}
				mbs.mu.Unlock()

				response.Result = map[string]any{
					"frameId": "frame-123",
				}

			case "Page.screenshot":
				var params struct {
					TabID string `json:"tabId"`
				}
				if paramData, err := json.Marshal(req.Params); err == nil {
					_ = json.Unmarshal(paramData, &params)
				}

				tabID := params.TabID
				if tabID == "" {
					mbs.mu.Lock()
					tabID = mbs.activeTabID
					mbs.mu.Unlock()
				}

				// Get URL for this tab
				mbs.mu.Lock()
				tabURL := mbs.tabURLs[tabID]
				isBlank := mbs.blankPageMode || tabURL == "" || tabURL == urlAboutBlank
				mbs.mu.Unlock()

				if isBlank {
					// Return screenshot with nil RawNodes and RawViewport for blank pages
					response.Result = map[string]any{
						"data":        "iVBORw0KGgoAAAANSUhEUgAAAAEAAAABCAYAAAAfFcSJAAAADUlEQVR42mNk+M9QDwADhgGAWjR9awAAAABJRU5ErkJggg==",
						"format":      "png",
						"rawNodes":    nil,
						"rawViewport": nil,
					}
				} else {
					// Return screenshot with valid data
					response.Result = map[string]any{
						"data":   "iVBORw0KGgoAAAANSUhEUgAAAAEAAAABCAYAAAAfFcSJAAAADUlEQVR42mNk+M9QDwADhgGAWjR9awAAAABJRU5ErkJggg==",
						"format": "png",
						"rawNodes": []map[string]any{
							{
								"role":      "button",
								"name":      fmt.Sprintf("Button on %s", tabURL),
								"backendId": 123,
								"bounds": map[string]any{
									"x":      100.0,
									"y":      200.0,
									"width":  80.0,
									"height": 40.0,
								},
							},
							{
								"role":      "link",
								"name":      fmt.Sprintf("Link on %s", tabURL),
								"backendId": 124,
								"bounds": map[string]any{
									"x":      200.0,
									"y":      300.0,
									"width":  60.0,
									"height": 30.0,
								},
							},
						},
						"rawViewport": map[string]any{
							"x":      0.0,
							"y":      0.0,
							"width":  1920.0,
							"height": 1080.0,
						},
					}
				}

			case "Page.click":
				response.Result = map[string]any{
					"success": true,
				}

			case "Page.clickByBackendID":
				response.Result = map[string]any{
					"success": true,
				}

			case "Tab.create":
				var params struct {
					URL string `json:"url"`
				}
				if paramData, err := json.Marshal(req.Params); err == nil {
					_ = json.Unmarshal(paramData, &params)
				}

				mbs.mu.Lock()
				mbs.tabCounter++
				newTabID := fmt.Sprintf("tab-%d", mbs.tabCounter+1)
				url := params.URL
				if url == "" {
					url = urlAboutBlank
				}
				mbs.tabs[newTabID] = map[string]any{
					"id":       newTabID,
					"url":      url,
					"title":    "New Tab",
					"isActive": true, // New tabs become active
				}
				mbs.tabURLs[newTabID] = url

				// Update old active tab
				if mbs.activeTabID != "" {
					if oldTab, exists := mbs.tabs[mbs.activeTabID]; exists {
						oldTab["isActive"] = false
						mbs.tabs[mbs.activeTabID] = oldTab
					}
				}
				mbs.activeTabID = newTabID
				mbs.mu.Unlock()

				response.Result = map[string]any{
					"tab": map[string]any{
						"id":       newTabID,
						"url":      url,
						"title":    "New Tab",
						"isActive": true,
					},
				}

			case "Tab.list":
				mbs.mu.Lock()
				tabs := make([]map[string]any, 0, len(mbs.tabs))
				for _, tab := range mbs.tabs {
					tabs = append(tabs, tab)
				}
				activeTabID := mbs.activeTabID
				mbs.mu.Unlock()

				response.Result = map[string]any{
					"tabs":        tabs,
					"activeTabId": activeTabID,
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
					delete(mbs.tabURLs, params.TabID)

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

func (mbs *mockBrowserServerBlankPage) Close() {
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

func (mbs *mockBrowserServerBlankPage) setBlankPageMode(blank bool) {
	mbs.mu.Lock()
	defer mbs.mu.Unlock()
	mbs.blankPageMode = blank
}

// Test 1: Blank page detection and cache clearing
func TestBrowserTool_BlankPageDetection(t *testing.T) {
	skipIfIntegrationTestsDisabled(t)

	mockServer := startMockBrowserServerBlankPage(t)
	defer mockServer.Close()

	mockPermissionService := &MockPermissionService{}
	sessionConfig := session.DefaultConfig()
	tool := NewBrowserTool(mockPermissionService, &MockSessionService{}, mockServer.wsURL, sessionConfig, "", mockClientFactory, nil, nil, "")

	tempDir := t.TempDir()
	ctx := createBrowserTestContext("test-session", "test-message", tempDir)

	mockPermissionService.On("Request", mock.Anything).Return(true)

	// Set mock to return blank page (nil RawNodes)
	mockServer.setBlankPageMode(true)

	// Navigate and screenshot - should detect blank page
	t.Run("detect blank page", func(t *testing.T) {
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
		assert.Contains(t, response.Content, "Page appears blank or not fully loaded")
	})

	// Attempt click - should fail with cache error
	t.Run("click fails without cache", func(t *testing.T) {
		clickCall := interfaces.ToolCall{
			ID:    "call-click",
			Name:  BrowserToolName,
			Input: `{"action": "click", "index": 0}`,
		}
		response, err := tool.Run(ctx, clickCall)
		require.NoError(t, err)
		assert.True(t, response.IsError)
		assert.Contains(t, response.Content, "no element cache found")
	})

	// Set mock to return valid page with elements
	mockServer.setBlankPageMode(false)

	// Take screenshot - cache should populate
	t.Run("screenshot populates cache", func(t *testing.T) {
		screenshotCall := interfaces.ToolCall{
			ID:    "call-screenshot-2",
			Name:  BrowserToolName,
			Input: `{"action": "screenshot"}`,
		}
		response, err := tool.Run(ctx, screenshotCall)
		require.NoError(t, err)
		assert.False(t, response.IsError)
		assert.NotContains(t, response.Content, "Page appears blank")
	})

	// Click should now succeed
	t.Run("click succeeds with cache", func(t *testing.T) {
		clickCall := interfaces.ToolCall{
			ID:    "call-click-2",
			Name:  BrowserToolName,
			Input: `{"action": "click", "index": 0}`,
		}
		response, err := tool.Run(ctx, clickCall)
		require.NoError(t, err)
		assert.False(t, response.IsError)
		assert.Contains(t, response.Content, "Successfully clicked element 0")
	})
}

// Test 2: Cache invalidation on tab switch
func TestBrowserTool_CacheInvalidationOnTabSwitch(t *testing.T) {
	skipIfIntegrationTestsDisabled(t)

	mockServer := startMockBrowserServerBlankPage(t)
	defer mockServer.Close()

	mockPermissionService := &MockPermissionService{}
	sessionConfig := session.DefaultConfig()
	tool := NewBrowserTool(mockPermissionService, &MockSessionService{}, mockServer.wsURL, sessionConfig, "", mockClientFactory, nil, nil, "")

	tempDir := t.TempDir()
	ctx := createBrowserTestContext("test-session", "test-message", tempDir)

	mockPermissionService.On("Request", mock.Anything).Return(true)
	mockServer.setBlankPageMode(false)

	// Create tab A and take screenshot
	t.Run("populate cache for tab A", func(t *testing.T) {
		openCall := interfaces.ToolCall{
			ID:    "call-open",
			Name:  BrowserToolName,
			Input: `{"action": "open", "url": "https://example-a.com"}`,
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
	})

	// Create tab B without URL (starts blank)
	var tabBID string
	t.Run("create tab B", func(t *testing.T) {
		createCall := interfaces.ToolCall{
			ID:    "call-create-tab",
			Name:  BrowserToolName,
			Input: `{"action": "tab_create"}`,
		}
		response, err := tool.Run(ctx, createCall)
		require.NoError(t, err)
		assert.False(t, response.IsError)
		// Extract tab ID from response (format: "Created new tab: tab-2...")
		parts := strings.Split(response.Content, ":")
		if len(parts) > 1 {
			tabBID = strings.TrimSpace(strings.Split(parts[1], " ")[0])
		}
	})

	// Switch to tab A
	t.Run("switch to tab A clears cache", func(t *testing.T) {
		switchCall := interfaces.ToolCall{
			ID:    "call-switch",
			Name:  BrowserToolName,
			Input: `{"action": "tab_switch", "tabId": "tab-1"}`,
		}
		response, err := tool.Run(ctx, switchCall)
		require.NoError(t, err)
		assert.False(t, response.IsError)
		assert.Contains(t, response.Content, "Cache cleared for previous tab")
	})

	// Click on tab A - should still work because cache is still valid for tab A
	t.Run("click works on tab A (cache preserved)", func(t *testing.T) {
		clickCall := interfaces.ToolCall{
			ID:    "call-click",
			Name:  BrowserToolName,
			Input: `{"action": "click", "index": 0}`,
		}
		response, err := tool.Run(ctx, clickCall)
		require.NoError(t, err)
		assert.False(t, response.IsError)
		assert.Contains(t, response.Content, "Successfully clicked")
	})

	// Note: Tab cache isolation is tested more thoroughly in TestBrowserTool_ConcurrentTabScreenshots
	_ = tabBID // Created for testing tab switching
}

// Test 3: Active tab tracking
func TestBrowserTool_ActiveTabTracking(t *testing.T) {
	skipIfIntegrationTestsDisabled(t)

	mockServer := startMockBrowserServerBlankPage(t)
	defer mockServer.Close()

	mockPermissionService := &MockPermissionService{}
	sessionConfig := session.DefaultConfig()
	tool := NewBrowserTool(mockPermissionService, &MockSessionService{}, mockServer.wsURL, sessionConfig, "", mockClientFactory, nil, nil, "")

	tempDir := t.TempDir()
	ctx := createBrowserTestContext("test-session", "test-message", tempDir)

	mockPermissionService.On("Request", mock.Anything).Return(true)
	mockServer.setBlankPageMode(false)

	// Navigate without explicit tabID - should use active tab
	t.Run("navigate uses active tab", func(t *testing.T) {
		openCall := interfaces.ToolCall{
			ID:    "call-open",
			Name:  BrowserToolName,
			Input: `{"action": "open", "url": "https://example.com"}`,
		}
		response, err := tool.Run(ctx, openCall)
		require.NoError(t, err)
		assert.False(t, response.IsError)
	})

	// Screenshot without tabID - should use active tab
	t.Run("screenshot uses active tab", func(t *testing.T) {
		screenshotCall := interfaces.ToolCall{
			ID:    "call-screenshot",
			Name:  BrowserToolName,
			Input: `{"action": "screenshot"}`,
		}
		response, err := tool.Run(ctx, screenshotCall)
		require.NoError(t, err)
		assert.False(t, response.IsError)
	})

	// Click should succeed on active tab
	t.Run("click uses active tab cache", func(t *testing.T) {
		clickCall := interfaces.ToolCall{
			ID:    "call-click",
			Name:  BrowserToolName,
			Input: `{"action": "click", "index": 0}`,
		}
		response, err := tool.Run(ctx, clickCall)
		require.NoError(t, err)
		assert.False(t, response.IsError)
	})

	// Create second tab - becomes active
	t.Run("new tab becomes active", func(t *testing.T) {
		createCall := interfaces.ToolCall{
			ID:    "call-create",
			Name:  BrowserToolName,
			Input: `{"action": "tab_create", "url": "https://example2.com"}`,
		}
		response, err := tool.Run(ctx, createCall)
		require.NoError(t, err)
		assert.False(t, response.IsError)
	})

	// Screenshot without tabID - should use new active tab
	t.Run("screenshot uses new active tab", func(t *testing.T) {
		screenshotCall := interfaces.ToolCall{
			ID:    "call-screenshot-2",
			Name:  BrowserToolName,
			Input: `{"action": "screenshot"}`,
		}
		response, err := tool.Run(ctx, screenshotCall)
		require.NoError(t, err)
		assert.False(t, response.IsError)
	})
}

// Test 4: Backend ID cache validation
func TestBrowserTool_BackendIDCacheValidation(t *testing.T) {
	skipIfIntegrationTestsDisabled(t)

	mockServer := startMockBrowserServerBlankPage(t)
	defer mockServer.Close()

	mockPermissionService := &MockPermissionService{}
	sessionConfig := session.DefaultConfig()
	tool := NewBrowserTool(mockPermissionService, &MockSessionService{}, mockServer.wsURL, sessionConfig, "", mockClientFactory, nil, nil, "")

	tempDir := t.TempDir()
	ctx := createBrowserTestContext("test-session", "test-message", tempDir)

	mockPermissionService.On("Request", mock.Anything).Return(true)
	mockServer.setBlankPageMode(false)

	// Navigate without screenshot
	t.Run("navigate without screenshot", func(t *testing.T) {
		openCall := interfaces.ToolCall{
			ID:    "call-open",
			Name:  BrowserToolName,
			Input: `{"action": "open", "url": "https://example.com"}`,
		}
		_, err := tool.Run(ctx, openCall)
		require.NoError(t, err)
	})

	// Click should fail - no cache
	t.Run("click fails without cache", func(t *testing.T) {
		clickCall := interfaces.ToolCall{
			ID:    "call-click",
			Name:  BrowserToolName,
			Input: `{"action": "click", "index": 0}`,
		}
		response, err := tool.Run(ctx, clickCall)
		require.NoError(t, err)
		assert.True(t, response.IsError)
		assert.Contains(t, response.Content, "no element cache found")
	})

	// Screenshot - cache populated
	t.Run("screenshot populates cache", func(t *testing.T) {
		screenshotCall := interfaces.ToolCall{
			ID:    "call-screenshot",
			Name:  BrowserToolName,
			Input: `{"action": "screenshot"}`,
		}
		response, err := tool.Run(ctx, screenshotCall)
		require.NoError(t, err)
		assert.False(t, response.IsError)
	})

	// Click succeeds
	t.Run("click succeeds with cache", func(t *testing.T) {
		clickCall := interfaces.ToolCall{
			ID:    "call-click-2",
			Name:  BrowserToolName,
			Input: `{"action": "click", "index": 0}`,
		}
		response, err := tool.Run(ctx, clickCall)
		require.NoError(t, err)
		assert.False(t, response.IsError)
	})

	// Navigate to new page
	t.Run("navigate clears cache", func(t *testing.T) {
		openCall := interfaces.ToolCall{
			ID:    "call-open-2",
			Name:  BrowserToolName,
			Input: `{"action": "open", "url": "https://different.com"}`,
		}
		_, err := tool.Run(ctx, openCall)
		require.NoError(t, err)
	})

	// Click fails again
	t.Run("click fails after navigation", func(t *testing.T) {
		clickCall := interfaces.ToolCall{
			ID:    "call-click-3",
			Name:  BrowserToolName,
			Input: `{"action": "click", "index": 0}`,
		}
		response, err := tool.Run(ctx, clickCall)
		require.NoError(t, err)
		assert.True(t, response.IsError)
		assert.Contains(t, response.Content, "no element cache found")
	})
}

// Test 5: Empty URL handling
func TestBrowserTool_EmptyURLHandling(t *testing.T) {
	skipIfIntegrationTestsDisabled(t)

	mockServer := startMockBrowserServerBlankPage(t)
	defer mockServer.Close()

	mockPermissionService := &MockPermissionService{}
	sessionConfig := session.DefaultConfig()
	tool := NewBrowserTool(mockPermissionService, &MockSessionService{}, mockServer.wsURL, sessionConfig, "", mockClientFactory, nil, nil, "")

	tempDir := t.TempDir()
	ctx := createBrowserTestContext("test-session", "test-message", tempDir)

	mockPermissionService.On("Request", mock.Anything).Return(true)

	// List tabs - default tab should have about:blank URL
	t.Run("list tabs shows about:blank", func(t *testing.T) {
		listCall := interfaces.ToolCall{
			ID:    "call-list",
			Name:  BrowserToolName,
			Input: `{"action": "tab_list"}`,
		}
		response, err := tool.Run(ctx, listCall)
		require.NoError(t, err)
		assert.False(t, response.IsError)
		assert.Contains(t, response.Content, "about:blank")
	})

	// Screenshot on tab with empty/blank URL
	t.Run("screenshot detects blank page", func(t *testing.T) {
		screenshotCall := interfaces.ToolCall{
			ID:    "call-screenshot",
			Name:  BrowserToolName,
			Input: `{"action": "screenshot"}`,
		}
		response, err := tool.Run(ctx, screenshotCall)
		require.NoError(t, err)
		assert.False(t, response.IsError)
		assert.Contains(t, response.Content, "Page appears blank or not fully loaded")
	})
}

// Test 6: Cache staleness across page navigation
func TestBrowserTool_CacheStalenessCrossPage(t *testing.T) {
	skipIfIntegrationTestsDisabled(t)

	mockServer := startMockBrowserServerBlankPage(t)
	defer mockServer.Close()

	mockPermissionService := &MockPermissionService{}
	sessionConfig := session.DefaultConfig()
	tool := NewBrowserTool(mockPermissionService, &MockSessionService{}, mockServer.wsURL, sessionConfig, "", mockClientFactory, nil, nil, "")

	tempDir := t.TempDir()
	ctx := createBrowserTestContext("test-session", "test-message", tempDir)

	mockPermissionService.On("Request", mock.Anything).Return(true)
	mockServer.setBlankPageMode(false)

	// Navigate to example.com
	t.Run("navigate to example.com", func(t *testing.T) {
		openCall := interfaces.ToolCall{
			ID:    "call-open",
			Name:  BrowserToolName,
			Input: `{"action": "open", "url": "https://example.com"}`,
		}
		_, err := tool.Run(ctx, openCall)
		require.NoError(t, err)
	})

	// Screenshot - cache populated
	t.Run("screenshot populates cache", func(t *testing.T) {
		screenshotCall := interfaces.ToolCall{
			ID:    "call-screenshot",
			Name:  BrowserToolName,
			Input: `{"action": "screenshot"}`,
		}
		response, err := tool.Run(ctx, screenshotCall)
		require.NoError(t, err)
		assert.False(t, response.IsError)
	})

	// Navigate same tab to different.com
	t.Run("navigate to different.com", func(t *testing.T) {
		openCall := interfaces.ToolCall{
			ID:    "call-open-2",
			Name:  BrowserToolName,
			Input: `{"action": "open", "url": "https://different.com"}`,
		}
		_, err := tool.Run(ctx, openCall)
		require.NoError(t, err)
	})

	// Click should fail - cache cleared
	t.Run("click fails after navigation", func(t *testing.T) {
		clickCall := interfaces.ToolCall{
			ID:    "call-click",
			Name:  BrowserToolName,
			Input: `{"action": "click", "index": 0}`,
		}
		response, err := tool.Run(ctx, clickCall)
		require.NoError(t, err)
		assert.True(t, response.IsError)
		assert.Contains(t, response.Content, "no element cache found")
	})

	// Screenshot on different.com
	t.Run("screenshot on new page", func(t *testing.T) {
		screenshotCall := interfaces.ToolCall{
			ID:    "call-screenshot-2",
			Name:  BrowserToolName,
			Input: `{"action": "screenshot"}`,
		}
		response, err := tool.Run(ctx, screenshotCall)
		require.NoError(t, err)
		assert.False(t, response.IsError)
	})

	// Click succeeds with new elements
	t.Run("click succeeds on new page", func(t *testing.T) {
		clickCall := interfaces.ToolCall{
			ID:    "call-click-2",
			Name:  BrowserToolName,
			Input: `{"action": "click", "index": 0}`,
		}
		response, err := tool.Run(ctx, clickCall)
		require.NoError(t, err)
		assert.False(t, response.IsError)
	})
}

// Test 7: Rapid navigation sequence
func TestBrowserTool_RapidNavigationSequence(t *testing.T) {
	skipIfIntegrationTestsDisabled(t)

	mockServer := startMockBrowserServerBlankPage(t)
	defer mockServer.Close()

	mockPermissionService := &MockPermissionService{}
	sessionConfig := session.DefaultConfig()
	tool := NewBrowserTool(mockPermissionService, &MockSessionService{}, mockServer.wsURL, sessionConfig, "", mockClientFactory, nil, nil, "")

	tempDir := t.TempDir()
	ctx := createBrowserTestContext("test-session", "test-message", tempDir)

	mockPermissionService.On("Request", mock.Anything).Return(true)
	mockServer.setBlankPageMode(false)

	// Navigate to URL1
	t.Run("navigate to URL1", func(t *testing.T) {
		openCall := interfaces.ToolCall{
			ID:    "call-open-1",
			Name:  BrowserToolName,
			Input: `{"action": "open", "url": "https://url1.com"}`,
		}
		_, err := tool.Run(ctx, openCall)
		require.NoError(t, err)
	})

	// Immediately navigate to URL2
	t.Run("navigate to URL2", func(t *testing.T) {
		openCall := interfaces.ToolCall{
			ID:    "call-open-2",
			Name:  BrowserToolName,
			Input: `{"action": "open", "url": "https://url2.com"}`,
		}
		_, err := tool.Run(ctx, openCall)
		require.NoError(t, err)
	})

	// Screenshot
	t.Run("screenshot after rapid navigation", func(t *testing.T) {
		screenshotCall := interfaces.ToolCall{
			ID:    "call-screenshot",
			Name:  BrowserToolName,
			Input: `{"action": "screenshot"}`,
		}
		response, err := tool.Run(ctx, screenshotCall)
		require.NoError(t, err)
		assert.False(t, response.IsError)
	})

	// Verify cache has URL2 elements (click should work)
	t.Run("click works with URL2 elements", func(t *testing.T) {
		clickCall := interfaces.ToolCall{
			ID:    "call-click",
			Name:  BrowserToolName,
			Input: `{"action": "click", "index": 0}`,
		}
		response, err := tool.Run(ctx, clickCall)
		require.NoError(t, err)
		assert.False(t, response.IsError)
	})

	// Navigate to URL3
	t.Run("navigate to URL3", func(t *testing.T) {
		openCall := interfaces.ToolCall{
			ID:    "call-open-3",
			Name:  BrowserToolName,
			Input: `{"action": "open", "url": "https://url3.com"}`,
		}
		_, err := tool.Run(ctx, openCall)
		require.NoError(t, err)
	})

	// Screenshot
	t.Run("screenshot on URL3", func(t *testing.T) {
		screenshotCall := interfaces.ToolCall{
			ID:    "call-screenshot-2",
			Name:  BrowserToolName,
			Input: `{"action": "screenshot"}`,
		}
		response, err := tool.Run(ctx, screenshotCall)
		require.NoError(t, err)
		assert.False(t, response.IsError)
	})

	// Verify cache updated
	t.Run("click works with URL3 elements", func(t *testing.T) {
		clickCall := interfaces.ToolCall{
			ID:    "call-click-2",
			Name:  BrowserToolName,
			Input: `{"action": "click", "index": 0}`,
		}
		response, err := tool.Run(ctx, clickCall)
		require.NoError(t, err)
		assert.False(t, response.IsError)
	})
}

// Test 8: Concurrent tab screenshots (thread safety)
func TestBrowserTool_ConcurrentTabScreenshots(t *testing.T) {
	skipIfIntegrationTestsDisabled(t)

	mockServer := startMockBrowserServerBlankPage(t)
	defer mockServer.Close()

	mockPermissionService := &MockPermissionService{}
	sessionConfig := session.DefaultConfig()
	tool := NewBrowserTool(mockPermissionService, &MockSessionService{}, mockServer.wsURL, sessionConfig, "", mockClientFactory, nil, nil, "")

	tempDir := t.TempDir()
	ctx := createBrowserTestContext("test-session", "test-message", tempDir)

	mockPermissionService.On("Request", mock.Anything).Return(true)
	mockServer.setBlankPageMode(false)

	// Navigate default tab to a URL first
	openCall := interfaces.ToolCall{
		ID:    "call-open-default",
		Name:  BrowserToolName,
		Input: `{"action": "open", "url": "https://example1.com", "tabId": "tab-1"}`,
	}
	_, err := tool.Run(ctx, openCall)
	require.NoError(t, err)

	// Create 3 tabs
	tabIDs := []string{"tab-1"}
	for i := 2; i <= 3; i++ {
		createCall := interfaces.ToolCall{
			ID:    fmt.Sprintf("call-create-%d", i),
			Name:  BrowserToolName,
			Input: fmt.Sprintf(`{"action": "tab_create", "url": "https://example%d.com"}`, i),
		}
		response, err := tool.Run(ctx, createCall)
		require.NoError(t, err)
		assert.False(t, response.IsError)
		// Extract tab ID
		parts := strings.Split(response.Content, ":")
		if len(parts) > 1 {
			tabID := strings.TrimSpace(strings.Split(parts[1], " ")[0])
			tabIDs = append(tabIDs, tabID)
		}
	}

	// Screenshot all tabs concurrently
	t.Run("concurrent screenshots", func(t *testing.T) {
		var wg sync.WaitGroup
		errors := make([]error, len(tabIDs))

		for i, tabID := range tabIDs {
			wg.Add(1)
			go func(idx int, tid string) {
				defer wg.Done()

				screenshotCall := interfaces.ToolCall{
					ID:    fmt.Sprintf("call-screenshot-%d", idx),
					Name:  BrowserToolName,
					Input: fmt.Sprintf(`{"action": "screenshot", "tabId": %q}`, tid),
				}
				response, err := tool.Run(ctx, screenshotCall)
				if err != nil {
					errors[idx] = err
					return
				}
				if response.IsError {
					errors[idx] = fmt.Errorf("screenshot error: %s", response.Content)
				}
			}(i, tabID)
		}

		wg.Wait()

		// Check no errors
		for i, err := range errors {
			assert.NoError(t, err, "Screenshot %d failed", i)
		}
	})

	// Verify each tab has different cache by clicking
	t.Run("verify separate caches", func(t *testing.T) {
		for i, tabID := range tabIDs {
			clickCall := interfaces.ToolCall{
				ID:    fmt.Sprintf("call-click-%d", i),
				Name:  BrowserToolName,
				Input: fmt.Sprintf(`{"action": "click", "index": 0, "tabId": %q}`, tabID),
			}
			response, err := tool.Run(ctx, clickCall)
			require.NoError(t, err, "Click in tab %d failed", i)
			assert.False(t, response.IsError, "Click in tab %d returned error", i)
		}
	})

	// Click in each tab concurrently
	t.Run("concurrent clicks", func(t *testing.T) {
		var wg sync.WaitGroup
		errors := make([]error, len(tabIDs))

		for i, tabID := range tabIDs {
			wg.Add(1)
			go func(idx int, tid string) {
				defer wg.Done()

				clickCall := interfaces.ToolCall{
					ID:    fmt.Sprintf("call-click-concurrent-%d", idx),
					Name:  BrowserToolName,
					Input: fmt.Sprintf(`{"action": "click", "index": 1, "tabId": %q}`, tid),
				}
				response, err := tool.Run(ctx, clickCall)
				if err != nil {
					errors[idx] = err
					return
				}
				if response.IsError {
					errors[idx] = fmt.Errorf("click error: %s", response.Content)
				}
			}(i, tabID)
		}

		wg.Wait()

		// Check all clicks succeeded
		for i, err := range errors {
			assert.NoError(t, err, "Concurrent click %d failed", i)
		}
	})
}
