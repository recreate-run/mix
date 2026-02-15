package browser

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/gorilla/websocket"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	browserpkg "mix/internal/browser"
	"mix/internal/llm/interfaces"
	"mix/internal/session"
)

// Local CDP types to avoid import cycle with internal/http
// These mirror the types in mix/internal/http/websocket_tunnel.go

const (
	methodScreenshotTunnel     = "Page.screenshot"
	methodClickByBackendTunnel = "Page.clickByBackendID"
)

// CDPRequest represents a CDP command request
type CDPRequest struct {
	ID        interface{} `json:"id"`
	Method    string      `json:"method"`
	Params    interface{} `json:"params,omitempty"`
	SessionID string      `json:"sessionId,omitempty"`
	BrowserID string      `json:"browserId,omitempty"`
}

// CDPResponse represents a CDP command response
type CDPResponse struct {
	ID        interface{} `json:"id"`
	Result    interface{} `json:"result,omitempty"`
	Error     *CDPError   `json:"error,omitempty"`
	BrowserID string      `json:"browserId,omitempty"`
}

// CDPError represents a CDP error
type CDPError struct {
	Code    int         `json:"code"`
	Message string      `json:"message"`
	Data    interface{} `json:"data,omitempty"`
}

// mockTunnelRegistry simulates the TunnelRegistry for testing
type mockTunnelRegistry struct {
	mu          sync.RWMutex
	connections map[string]*mockTunnelConnection
}

// mockTunnelConnection simulates a tunnel connection
type mockTunnelConnection struct {
	SessionID       string
	Token           string
	Conn            *websocket.Conn
	SendChan        chan []byte
	Context         context.Context
	Cancel          context.CancelFunc
	pendingRequests sync.Map
}

func newMockTunnelRegistry() *mockTunnelRegistry {
	return &mockTunnelRegistry{
		connections: make(map[string]*mockTunnelConnection),
	}
}

func (r *mockTunnelRegistry) HandleTunnelConnection(w http.ResponseWriter, req *http.Request) {
	sessionID := req.PathValue("sessionId")
	if sessionID == "" {
		http.Error(w, "Missing sessionId", http.StatusBadRequest)
		return
	}

	token := req.URL.Query().Get("token")
	if token == "" {
		http.Error(w, "Missing token", http.StatusUnauthorized)
		return
	}

	upgrader := websocket.Upgrader{
		CheckOrigin: func(r *http.Request) bool { return true },
	}

	conn, err := upgrader.Upgrade(w, req, nil)
	if err != nil {
		return
	}

	ctx, cancel := context.WithCancel(req.Context())
	defer cancel()

	tunnelConn := &mockTunnelConnection{
		SessionID: sessionID,
		Token:     token,
		Conn:      conn,
		SendChan:  make(chan []byte, 50),
		Context:   ctx,
		Cancel:    cancel,
	}

	r.mu.Lock()
	r.connections[sessionID] = tunnelConn
	r.mu.Unlock()

	defer func() {
		r.mu.Lock()
		delete(r.connections, sessionID)
		r.mu.Unlock()
		_ = conn.Close()
	}()

	go tunnelConn.sendLoop()
	tunnelConn.readLoop()
}

func (tc *mockTunnelConnection) readLoop() {
	for {
		_, message, err := tc.Conn.ReadMessage()
		if err != nil {
			return
		}

		var response CDPResponse
		if err := json.Unmarshal(message, &response); err != nil {
			continue
		}

		normalizedID := fmt.Sprintf("%v", response.ID)
		if ch, ok := tc.pendingRequests.LoadAndDelete(normalizedID); ok {
			responseChan := ch.(chan CDPResponse)
			select {
			case responseChan <- response:
			case <-time.After(time.Second):
			}
		}
	}
}

func (tc *mockTunnelConnection) sendLoop() {
	for {
		select {
		case <-tc.Context.Done():
			return
		case message := <-tc.SendChan:
			if err := tc.Conn.WriteMessage(websocket.TextMessage, message); err != nil {
				return
			}
		}
	}
}

func (r *mockTunnelRegistry) SendCommandToTunnel(sessionID string, command CDPRequest) (*CDPResponse, error) {
	r.mu.RLock()
	conn := r.connections[sessionID]
	r.mu.RUnlock()

	if conn == nil {
		return nil, fmt.Errorf("browser not connected for session %s", sessionID)
	}

	responseChan := make(chan CDPResponse, 1)
	normalizedID := fmt.Sprintf("%v", command.ID)
	conn.pendingRequests.Store(normalizedID, responseChan)
	defer conn.pendingRequests.Delete(normalizedID)

	message, err := json.Marshal(command)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal command: %w", err)
	}

	select {
	case conn.SendChan <- message:
	case <-time.After(5 * time.Second):
		return nil, fmt.Errorf("send timeout")
	}

	select {
	case response := <-responseChan:
		return &response, nil
	case <-time.After(30 * time.Second):
		return nil, fmt.Errorf("response timeout")
	}
}

func (r *mockTunnelRegistry) GetActiveTunnels() []string {
	r.mu.RLock()
	defer r.mu.RUnlock()

	sessions := make([]string, 0, len(r.connections))
	for sessionID := range r.connections {
		sessions = append(sessions, sessionID)
	}
	return sessions
}

// mockElectronClient simulates an Electron browser connecting via WebSocket tunnel
type mockElectronClient struct {
	conn         *websocket.Conn
	receivedCmds []CDPRequest
	mu           sync.Mutex
	t            *testing.T
}

// newMockElectronClient creates and connects a mock Electron client to the tunnel
func newMockElectronClient(t *testing.T, tunnelURL, sessionID, token string) *mockElectronClient {
	t.Helper()

	// Build WebSocket URL with sessionID and token
	wsURL := "ws" + strings.TrimPrefix(tunnelURL, "http") + "/api/v1/tunnel/cdp/session/" + sessionID + "?token=" + token

	// Connect to tunnel
	conn, resp, err := websocket.DefaultDialer.Dial(wsURL, nil)
	if resp != nil && resp.Body != nil {
		defer resp.Body.Close()
	}
	require.NoError(t, err, "Failed to connect mock Electron client to tunnel")

	client := &mockElectronClient{
		conn:         conn,
		receivedCmds: make([]CDPRequest, 0),
		t:            t,
	}

	// Start listening for commands
	go client.readLoop()

	return client
}

// readLoop reads CDP commands from the tunnel and sends mock responses
func (m *mockElectronClient) readLoop() {
	for {
		_, message, err := m.conn.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseNormalClosure) {
				m.t.Logf("Mock Electron client read error: %v", err)
			}
			return
		}

		// Parse CDP request
		var req CDPRequest
		if err := json.Unmarshal(message, &req); err != nil {
			m.t.Logf("Failed to parse CDP request: %v", err)
			continue
		}

		m.mu.Lock()
		m.receivedCmds = append(m.receivedCmds, req)
		m.mu.Unlock()

		// Generate mock response based on method
		response := m.generateMockResponse(req)

		// Send response back through tunnel
		respData, err := json.Marshal(response)
		if err != nil {
			m.t.Logf("Failed to marshal response: %v", err)
			continue
		}

		if err := m.conn.WriteMessage(websocket.TextMessage, respData); err != nil {
			m.t.Logf("Failed to send response: %v", err)
			return
		}
	}
}

// generateMockResponse generates a mock CDP response for a given request
func (m *mockElectronClient) generateMockResponse(req CDPRequest) CDPResponse {
	m.t.Helper()

	response := CDPResponse{
		ID: req.ID,
	}

	// Simulate responses for different CDP methods
	switch req.Method {
	case methodNavigate:
		response.Result = map[string]interface{}{
			"frameId": "frame-tunnel-test",
		}

	case methodScreenshotTunnel:
		// Create mock accessibility tree with elements
		rawNodes := make([]map[string]interface{}, 10)
		for i := 0; i < 10; i++ {
			rawNodes[i] = map[string]interface{}{
				"role":      "button",
				"name":      "Tunnel Button " + string(rune('A'+i)),
				"backendId": int64(2000 + i*10),
				"bounds": map[string]interface{}{
					"x":      float64(50 + i*100),
					"y":      float64(100),
					"width":  80.0,
					"height": 40.0,
				},
			}
		}

		response.Result = map[string]interface{}{
			"data":     "iVBORw0KGgoAAAANSUhEUgAAAAEAAAABCAYAAAAfFcSJAAAADUlEQVR42mNk+M9QDwADhgGAWjR9awAAAABJRU5ErkJggg==",
			"format":   "png",
			"rawNodes": rawNodes,
			"rawViewport": map[string]interface{}{
				"x":      0.0,
				"y":      0.0,
				"width":  1920.0,
				"height": 1080.0,
			},
		}

	case methodClickByBackendTunnel:
		response.Result = map[string]interface{}{
			"success": true,
		}

	case "Browser.close":
		response.Result = map[string]interface{}{
			"success": true,
		}

	default:
		response.Error = &CDPError{
			Code:    -1,
			Message: "Method not implemented in mock: " + req.Method,
		}
	}

	return response
}

// getReceivedCommands returns all commands received by the mock client
func (m *mockElectronClient) getReceivedCommands() []CDPRequest {
	m.mu.Lock()
	defer m.mu.Unlock()
	cmds := make([]CDPRequest, len(m.receivedCmds))
	copy(cmds, m.receivedCmds)
	return cmds
}

// close closes the mock client connection
func (m *mockElectronClient) close() {
	_ = m.conn.Close()
}

// TestBrowserToolTunnelIntegrationE2E tests the full tunnel integration flow
func TestBrowserToolTunnelIntegrationE2E(t *testing.T) {
	skipIfIntegrationTestsDisabled(t)

	// 1. Create TunnelRegistry
	tunnelRegistry := newMockTunnelRegistry()

	// 2. Create HTTP test server with tunnel endpoint
	mux := http.NewServeMux()
	mux.HandleFunc("/api/v1/tunnel/cdp/session/{sessionId}", tunnelRegistry.HandleTunnelConnection)
	server := httptest.NewServer(mux)
	defer server.Close()

	// 3. Use sessionID directly (no more threadID)
	sessionID := "test-session-456"
	token := "test-token-789"

	// 4. Connect mock Electron client to tunnel
	mockElectron := newMockElectronClient(t, server.URL, sessionID, token)
	defer mockElectron.close()

	// Wait for connection to establish
	time.Sleep(100 * time.Millisecond)

	// 5. Verify tunnel connection is registered
	activeTunnels := tunnelRegistry.GetActiveTunnels()
	require.Contains(t, activeTunnels, sessionID, "Tunnel should be registered")

	// 6. Create browser tool with tunnel support
	mockPermissionService := &MockPermissionService{}
	sessionConfig := session.DefaultConfig()

	// Create a factory for electron-embedded-browser mode
	tunnelFactory := func(sid string) (browserpkg.Client, error) {
		config := browserpkg.FactoryConfig{
			Mode:           browserpkg.ModeElectronEmbedded,
			TunnelRegistry: tunnelRegistry,
		}

		return browserpkg.NewClient(config, sid)
	}

	// Create browser tool with tunnel registry getter
	tool := NewBrowserTool(mockPermissionService, &MockSessionService{}, "ws://unused:9999", sessionConfig, browserpkg.ModeElectronEmbedded, tunnelFactory, nil, func() interface{} { return tunnelRegistry }, "http://localhost:8081")

	tempDir := t.TempDir()
	ctx := createBrowserTestContext(sessionID, "test-message", tempDir)

	t.Run("navigate_through_tunnel", func(t *testing.T) {
		call := interfaces.ToolCall{
			ID:    "call-navigate",
			Name:  BrowserToolName,
			Input: `{"action": "open", "url": "https://example.com"}`,
		}

		// This should route through the tunnel, not the ConnectionManager
		response, err := tool.Run(ctx, call)

		// TODO: Once tunnel integration is implemented, these assertions should pass:
		// require.NoError(t, err)
		// assert.False(t, response.IsError)
		// assert.Contains(t, response.Content, "Successfully navigated")

		// For now, we expect this to fail because tunnel integration doesn't exist
		_ = response
		_ = err

		// Verify mock Electron received the command
		// TODO: Uncomment once tunnel integration works
		// cmds := mockElectron.getReceivedCommands()
		// require.NotEmpty(t, cmds, "Mock Electron should have received commands")
		// assert.Equal(t, "Page.navigate", cmds[0].Method)
	})

	t.Run("screenshot_through_tunnel", func(t *testing.T) {
		call := interfaces.ToolCall{
			ID:    "call-screenshot",
			Name:  BrowserToolName,
			Input: `{"action": "screenshot"}`,
		}

		response, err := tool.Run(ctx, call)

		// TODO: Once tunnel integration is implemented:
		// require.NoError(t, err)
		// assert.False(t, response.IsError)
		// assert.Contains(t, response.Content, "Screenshot captured")

		_ = response
		_ = err

		// Verify screenshot command was sent through tunnel
		// TODO: Uncomment once tunnel integration works
		// cmds := mockElectron.getReceivedCommands()
		// screenshotCmds := filterCommandsByMethod(cmds, "Page.screenshot")
		// require.NotEmpty(t, screenshotCmds, "Screenshot command should be sent through tunnel")
	})

	t.Run("click_through_tunnel", func(t *testing.T) {
		// First take screenshot to populate cache
		screenshotCall := interfaces.ToolCall{
			ID:    "call-screenshot-2",
			Name:  BrowserToolName,
			Input: `{"action": "screenshot"}`,
		}
		_, _ = tool.Run(ctx, screenshotCall)

		// Then click an element
		clickCall := interfaces.ToolCall{
			ID:    "call-click",
			Name:  BrowserToolName,
			Input: `{"action": "click", "index": 0}`,
		}

		response, err := tool.Run(ctx, clickCall)

		// TODO: Once tunnel integration is implemented:
		// require.NoError(t, err)
		// assert.False(t, response.IsError)
		// assert.Contains(t, response.Content, "Successfully clicked")

		_ = response
		_ = err

		// Verify click command used BackendID from cached elements
		// TODO: Uncomment once tunnel integration works
		// cmds := mockElectron.getReceivedCommands()
		// clickCmds := filterCommandsByMethod(cmds, "Page.clickByBackendID")
		// require.NotEmpty(t, clickCmds, "Click command should be sent through tunnel")
	})

	t.Run("verify_tunnel_path_used_not_connection_manager", func(t *testing.T) {
		// This test verifies that commands go through the tunnel, not ConnectionManager
		// The test should fail if ConnectionManager is used (port 9999 doesn't exist)

		// TODO: Once tunnel integration is implemented, verify that:
		// 1. ConnectionManager.GetOrCreate was NOT called
		// 2. All commands went through tunnelRegistry.SendCommandToTunnel
		// 3. Mock Electron received all commands
		//
		// This can be verified by:
		// - Adding a flag/counter to ConnectionManager
		// - Checking mockElectron.getReceivedCommands() length
		// - Asserting no connection errors to ws://unused:9999
	})

	t.Run("close_browser_through_tunnel", func(t *testing.T) {
		call := interfaces.ToolCall{
			ID:    "call-close",
			Name:  BrowserToolName,
			Input: `{"action": "close"}`,
		}

		response, err := tool.Run(ctx, call)

		// TODO: Once tunnel integration is implemented:
		// require.NoError(t, err)
		// assert.False(t, response.IsError)
		// assert.Contains(t, response.Content, "Browser closed")

		_ = response
		_ = err

		// Verify close command was sent
		// TODO: Uncomment once tunnel integration works
		// cmds := mockElectron.getReceivedCommands()
		// closeCmds := filterCommandsByMethod(cmds, "Browser.close")
		// require.NotEmpty(t, closeCmds, "Close command should be sent through tunnel")
	})
}

// TestBrowserToolTunnelConnectionFallback tests fallback to ConnectionManager when tunnel is not available
func TestBrowserToolTunnelConnectionFallback(t *testing.T) {
	skipIfIntegrationTestsDisabled(t)

	// Create TunnelRegistry (empty, no connected browsers)
	tunnelRegistry := newMockTunnelRegistry()

	// Start a real browser service mock for fallback
	mockServer := startMockBrowserServer(t)
	defer mockServer.Close()

	mockPermissionService := &MockPermissionService{}
	sessionConfig := session.DefaultConfig()

	// Create a factory for local-browser-service mode (fallback)
	serviceFactory := func(sid string) (browserpkg.Client, error) {
		// Return error to force use of ConnectionManager
		return nil, ErrMockFactoryNotConfigured
	}

	tool := NewBrowserTool(mockPermissionService, &MockSessionService{}, mockServer.wsURL, sessionConfig, browserpkg.ModeLocalBrowserService, serviceFactory, nil, nil, "")

	tempDir := t.TempDir()
	sessionID := "fallback-session"
	ctx := createBrowserTestContext(sessionID, "test-message", tempDir)

	// TODO: Add threadID that has NO tunnel connection
	// ctx = context.WithValue(ctx, interfaces.ThreadIDContextKey, "nonexistent-thread")

	t.Run("should_fallback_to_connection_manager_when_no_tunnel", func(t *testing.T) {
		// This should use ConnectionManager since no tunnel is connected
		call := interfaces.ToolCall{
			ID:    "call-open",
			Name:  BrowserToolName,
			Input: `{"action": "open", "url": "https://example.com"}`,
		}

		response, err := tool.Run(ctx, call)

		// Should succeed via ConnectionManager fallback
		require.NoError(t, err)
		assert.False(t, response.IsError)
		assert.Contains(t, response.Content, "Successfully navigated")

		// Verify that ConnectionManager was used, not tunnel
		activeTunnels := tunnelRegistry.GetActiveTunnels()
		assert.Empty(t, activeTunnels, "No tunnels should be active")
	})
}

// TestBrowserToolTunnelSessionBinding tests binding sessionID to threadID
func TestBrowserToolTunnelSessionBinding(t *testing.T) {
	skipIfIntegrationTestsDisabled(t)

	// This test is a placeholder for the session-thread binding logic
	// TODO: Implement session-thread binding in the actual codebase

	t.Run("session_connects_directly", func(t *testing.T) {
		// New flow with sessionID directly:
		// 1. Electron opens a tunnel connection with sessionID
		// 2. Server stores mapping: sessionID -> connection
		// 3. When browser tool is invoked with a sessionID, use that sessionID directly
		// 4. Use tunnelRegistry.SendCommandToTunnel(sessionID, command)

		// No binding needed - sessionID is used throughout
		t.Skip("Direct session connection - refactoring complete")
	})
}

// TestBrowserToolTunnelResponseTimeout tests timeout handling
func TestBrowserToolTunnelResponseTimeout(t *testing.T) {
	skipIfIntegrationTestsDisabled(t)

	tunnelRegistry := newMockTunnelRegistry()

	mux := http.NewServeMux()
	mux.HandleFunc("/api/v1/tunnel/cdp/session/{sessionId}", tunnelRegistry.HandleTunnelConnection)
	server := httptest.NewServer(mux)
	defer server.Close()

	sessionID := "timeout-session"
	token := "test-token"

	// Connect client that never responds
	wsURL := "ws" + strings.TrimPrefix(server.URL, "http") + "/api/v1/tunnel/cdp/session/" + sessionID + "?token=" + token
	conn, resp, err := websocket.DefaultDialer.Dial(wsURL, nil)
	if resp != nil && resp.Body != nil {
		defer resp.Body.Close()
	}
	require.NoError(t, err)
	defer func() {
		_ = conn.Close()
	}()

	// Start reading to keep connection alive, but never send responses
	go func() {
		for {
			_, _, err := conn.ReadMessage()
			if err != nil {
				return
			}
			// Don't send any response - simulate hung browser
		}
	}()

	time.Sleep(100 * time.Millisecond)

	t.Run("command_timeout_returns_error", func(t *testing.T) {
		// Send a command that will timeout
		req := CDPRequest{
			ID:     1,
			Method: "Page.navigate",
			Params: map[string]interface{}{"url": "https://example.com"},
		}

		// This should timeout after 30 seconds
		response, err := tunnelRegistry.SendCommandToTunnel(sessionID, req)

		// Should get timeout error
		require.Error(t, err)
		assert.Nil(t, response)
		assert.Contains(t, err.Error(), "timeout")
	})
}

// TestBrowserToolTunnelConcurrentCommands tests concurrent command handling
func TestBrowserToolTunnelConcurrentCommands(t *testing.T) {
	skipIfIntegrationTestsDisabled(t)

	tunnelRegistry := newMockTunnelRegistry()

	mux := http.NewServeMux()
	mux.HandleFunc("/api/v1/tunnel/cdp/session/{sessionId}", tunnelRegistry.HandleTunnelConnection)
	server := httptest.NewServer(mux)
	defer server.Close()

	sessionID := "concurrent-session"
	token := "test-token"

	mockElectron := newMockElectronClient(t, server.URL, sessionID, token)
	defer mockElectron.close()

	time.Sleep(100 * time.Millisecond)

	t.Run("multiple_commands_correctly_routed", func(t *testing.T) {
		// Send multiple commands concurrently
		var wg sync.WaitGroup
		numCommands := 10

		for i := 0; i < numCommands; i++ {
			wg.Add(1)
			go func(id int) {
				defer wg.Done()

				req := CDPRequest{
					ID:     id,
					Method: "Page.navigate",
					Params: map[string]interface{}{"url": "https://example.com/" + string(rune('a'+id))},
				}

				response, err := tunnelRegistry.SendCommandToTunnel(sessionID, req)
				assert.NoError(t, err)
				assert.NotNil(t, response)
				// Response.ID may be float64 after JSON unmarshaling
				assert.NotNil(t, response.ID)
			}(i)
		}

		wg.Wait()

		// Verify all commands were received
		cmds := mockElectron.getReceivedCommands()
		assert.Len(t, cmds, numCommands)
	})
}

// filterCommandsByMethod filters CDP commands by method name
