package browser

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
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
}

// startMockBrowserServer starts a mock browser service server
func startMockBrowserServer(t *testing.T) *mockBrowserServer {
	t.Helper()

	mbs := &mockBrowserServer{
		connections: make(map[*websocket.Conn]bool),
	}

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		conn, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			t.Logf("Failed to upgrade connection: %v", err)
			return
		}

		mbs.connections[conn] = true
		defer func() {
			delete(mbs.connections, conn)
			_ = conn.Close()
		}()

		// Handle WebSocket messages
		for {
			_, message, err := conn.ReadMessage()
			if err != nil {
				return
			}

			mbs.requestCount++

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
				// Return a minimal base64-encoded PNG
				response.Result = map[string]any{
					"data":   "iVBORw0KGgoAAAANSUhEUgAAAAEAAAABCAYAAAAfFcSJAAAADUlEQVR42mNk+M9QDwADhgGAWjR9awAAAABJRU5ErkJggg==",
					"format": "png",
					"elements": []map[string]any{
						{
							"index": 1,
							"role":  "button",
							"name":  "Test Button",
							"bounds": map[string]any{
								"x":      100.0,
								"y":      200.0,
								"width":  80.0,
								"height": 40.0,
							},
						},
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
	for conn := range mbs.connections {
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
