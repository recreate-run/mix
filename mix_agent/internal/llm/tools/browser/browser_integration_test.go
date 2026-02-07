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
	server       *httptest.Server
	wsURL        string
	connections  map[*websocket.Conn]bool
	requestCount int
	tabCounter   int
	tabs         map[string]map[string]any
	activeTabID  string
	mu           sync.Mutex
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
				// Return a minimal base64-encoded PNG with raw accessibility tree
				response.Result = map[string]any{
					"data":   "iVBORw0KGgoAAAANSUhEUgAAAAEAAAABCAYAAAAfFcSJAAAADUlEQVR42mNk+M9QDwADhgGAWjR9awAAAABJRU5ErkJggg==",
					"format": "png",
					"rawNodes": []map[string]any{
						{
							"role":      "button",
							"name":      "Click me",
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
							"name":      "Home",
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

			case "Page.click":
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

// Test full browser workflow: open -> screenshot -> click -> type -> scroll -> close
func TestBrowserToolIntegrationFullWorkflow(t *testing.T) {
	skipIfIntegrationTestsDisabled(t)

	// Start mock browser service
	mockServer := startMockBrowserServer(t)
	defer mockServer.Close()

	// Create tool
	mockPermissionService := &MockPermissionService{}
	sessionConfig := session.DefaultConfig()
	tool := NewBrowserTool(mockPermissionService, mockServer.wsURL, sessionConfig)

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
	tool := NewBrowserTool(mockPermissionService, mockServer.wsURL, sessionConfig)

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

	// Screenshot with overlay (default)
	t.Run("with overlay", func(t *testing.T) {
		call := interfaces.ToolCall{
			ID:    "call-screenshot-1",
			Name:  BrowserToolName,
			Input: `{"action": "screenshot", "withOverlay": true}`,
		}

		response, err := tool.Run(ctx, call)
		require.NoError(t, err)
		assert.False(t, response.IsError)
		assert.Contains(t, response.Content, "interactive elements")
		assert.Contains(t, response.Content, "Element details:")
	})

	// Screenshot without overlay
	t.Run("without overlay", func(t *testing.T) {
		call := interfaces.ToolCall{
			ID:    "call-screenshot-2",
			Name:  BrowserToolName,
			Input: `{"action": "screenshot", "withOverlay": false}`,
		}

		response, err := tool.Run(ctx, call)
		require.NoError(t, err)
		assert.False(t, response.IsError)
		assert.Contains(t, response.Content, "Screenshot captured successfully")
		assert.NotContains(t, response.Content, "Element details:")
	})
}

// Test error handling when browser service is down
func TestBrowserToolIntegrationServiceDown(t *testing.T) {
	skipIfIntegrationTestsDisabled(t)

	mockPermissionService := &MockPermissionService{}
	sessionConfig := session.DefaultConfig()

	// Use invalid endpoint
	tool := NewBrowserTool(mockPermissionService, "ws://localhost:99999", sessionConfig)

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
	assert.Contains(t, response.Content, "Failed to connect to browser service")
}

// Test session isolation - multiple sessions should have separate connections
func TestBrowserToolIntegrationSessionIsolation(t *testing.T) {
	skipIfIntegrationTestsDisabled(t)

	mockServer := startMockBrowserServer(t)
	defer mockServer.Close()

	mockPermissionService := &MockPermissionService{}
	sessionConfig := session.DefaultConfig()
	tool := NewBrowserTool(mockPermissionService, mockServer.wsURL, sessionConfig)

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
	tool := NewBrowserTool(mockPermissionService, mockServer.wsURL, sessionConfig)

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
	for i := 0; i < 3; i++ {
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
	tool := NewBrowserTool(mockPermissionService, mockServer.wsURL, sessionConfig)

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
	tool := NewBrowserTool(mockPermissionService, mockServer.wsURL, sessionConfig)

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
	tool := NewBrowserTool(mockPermissionService, mockServer.wsURL, sessionConfig)

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

	tool := NewBrowserTool(mockPermissionService, mockServer.wsURL, sessionConfig)

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
	tool := NewBrowserTool(mockPermissionService, mockServer.wsURL, sessionConfig)

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
	tool := NewBrowserTool(mockPermissionService, mockServer.wsURL, sessionConfig)

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
	tool := NewBrowserTool(mockPermissionService, mockServer.wsURL, sessionConfig)

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
	tool := NewBrowserTool(mockPermissionService, wsURL, sessionConfig)

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
