package testdata

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/gorilla/websocket"
)

// MockCDPServer simulates a Chrome DevTools Protocol WebSocket server for testing
type MockCDPServer struct {
	server        *httptest.Server
	upgrader      websocket.Upgrader
	mu            sync.RWMutex
	conns         []*websocket.Conn
	messageID     atomic.Int64
	targets       map[string]string        // targetID → sessionID
	crashed       atomic.Bool
	commandDelays map[string]time.Duration // command method → delay duration
	t             *testing.T
}

// CDPCommand represents an incoming CDP command
type CDPCommand struct {
	ID        int             `json:"id"`
	Method    string          `json:"method"`
	Params    json.RawMessage `json:"params,omitempty"`
	SessionID string          `json:"sessionId,omitempty"`
}

// CDPResponse represents a CDP response
type CDPResponse struct {
	ID     int         `json:"id"`
	Result interface{} `json:"result,omitempty"`
	Error  *CDPError   `json:"error,omitempty"`
}

// CDPError represents a CDP error
type CDPError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

// NewMockCDPServer creates a new mock CDP WebSocket server
func NewMockCDPServer(t *testing.T) *MockCDPServer {
	t.Helper()

	mock := &MockCDPServer{
		upgrader: websocket.Upgrader{
			CheckOrigin: func(r *http.Request) bool {
				return true // Allow all origins for testing
			},
		},
		targets:       make(map[string]string),
		commandDelays: make(map[string]time.Duration),
		t:             t,
	}

	// Create HTTP server with WebSocket handler
	mock.server = httptest.NewServer(http.HandlerFunc(mock.handleWebSocket))

	// Convert http:// to ws:// for WebSocket URL
	wsURL := "ws" + mock.server.URL[4:] + "/devtools/browser"

	t.Logf("✓ Mock CDP server started at %s", wsURL)

	return mock
}

// GetURL returns the WebSocket URL for the mock server
func (m *MockCDPServer) GetURL() string {
	return "ws" + m.server.URL[4:] + "/devtools/browser"
}

// handleWebSocket handles incoming WebSocket connections
func (m *MockCDPServer) handleWebSocket(w http.ResponseWriter, r *http.Request) {
	// Check if server is crashed
	if m.crashed.Load() {
		http.Error(w, "Server crashed", http.StatusServiceUnavailable)
		return
	}

	conn, err := m.upgrader.Upgrade(w, r, nil)
	if err != nil {
		m.t.Logf("Failed to upgrade WebSocket: %v", err)
		return
	}

	// Store connection
	m.mu.Lock()
	m.conns = append(m.conns, conn)
	m.mu.Unlock()

	m.t.Log("✓ Client connected to mock CDP server")

	// Handle messages from this connection
	go m.handleConnection(conn)
}

// handleConnection handles messages from a single WebSocket connection
func (m *MockCDPServer) handleConnection(conn *websocket.Conn) {
	defer func() {
		_ = conn.Close()
		m.removeConnection(conn)
		m.t.Log("✓ Client disconnected from mock CDP server")
	}()

	for {
		// Check if crashed
		if m.crashed.Load() {
			return
		}

		var cmd CDPCommand
		err := conn.ReadJSON(&cmd)
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseNormalClosure) {
				m.t.Logf("WebSocket read error: %v", err)
			}
			return
		}

		m.t.Logf("Received CDP command: %s (id=%d)", cmd.Method, cmd.ID)

		// Check if there's a delay configured for this command
		m.mu.RLock()
		delay, hasDelay := m.commandDelays[cmd.Method]
		m.mu.RUnlock()

		if hasDelay && delay > 0 {
			m.t.Logf("Delaying response for %s by %v", cmd.Method, delay)
			time.Sleep(delay)
		}

		// Handle command and send response
		response := m.handleCommand(&cmd)
		if err := conn.WriteJSON(response); err != nil {
			m.t.Logf("Failed to send response: %v", err)
			return
		}
	}
}

// handleCommand processes a CDP command and returns a response
func (m *MockCDPServer) handleCommand(cmd *CDPCommand) *CDPResponse {
	switch cmd.Method {
	case "Target.createTarget":
		// Parse params to get URL
		var params struct {
			URL string `json:"url"`
		}
		_ = json.Unmarshal(cmd.Params, &params)

		targetID := fmt.Sprintf("target-%d", m.messageID.Add(1))
		m.t.Logf("Creating target: %s (url=%s)", targetID, params.URL)

		return &CDPResponse{
			ID: cmd.ID,
			Result: map[string]interface{}{
				"targetId": targetID,
			},
		}

	case "Target.attachToTarget":
		// Parse params to get targetId
		var params struct {
			TargetID string `json:"targetId"`
			Flatten  bool   `json:"flatten,omitempty"`
		}
		_ = json.Unmarshal(cmd.Params, &params)

		sessionID := fmt.Sprintf("session-%d", m.messageID.Add(1))
		m.mu.Lock()
		m.targets[params.TargetID] = sessionID
		m.mu.Unlock()

		m.t.Logf("Attaching to target: %s → session: %s", params.TargetID, sessionID)

		return &CDPResponse{
			ID: cmd.ID,
			Result: map[string]interface{}{
				"sessionId": sessionID,
			},
		}

	case "Target.closeTarget":
		var params struct {
			TargetID string `json:"targetId"`
		}
		_ = json.Unmarshal(cmd.Params, &params)

		m.t.Logf("Closing target: %s", params.TargetID)

		return &CDPResponse{
			ID: cmd.ID,
			Result: map[string]interface{}{
				"success": true,
			},
		}

	case "Page.navigate":
		var params struct {
			URL string `json:"url"`
		}
		_ = json.Unmarshal(cmd.Params, &params)

		m.t.Logf("Navigating to: %s", params.URL)

		return &CDPResponse{
			ID: cmd.ID,
			Result: map[string]interface{}{
				"frameId":   "frame-1",
				"loaderId":  "loader-1",
				"networkId": "network-1",
			},
		}

	case "Page.enable", "Runtime.enable", "DOM.enable", "Accessibility.enable",
		 "Network.enable", "Target.setDiscoverTargets", "Target.setAutoAttach":
		m.t.Logf("Enabling domain: %s", cmd.Method)
		return &CDPResponse{
			ID:     cmd.ID,
			Result: map[string]interface{}{},
		}

	case "Page.captureScreenshot":
		m.t.Logf("Capturing screenshot")
		// Return a minimal PNG data (1x1 transparent pixel in base64)
		return &CDPResponse{
			ID: cmd.ID,
			Result: map[string]interface{}{
				"data": "iVBORw0KGgoAAAANSUhEUgAAAAEAAAABCAYAAAAfFcSJAAAADUlEQVR42mNk+M9QDwADhgGAWjR9awAAAABJRU5ErkJggg==",
			},
		}

	case "Accessibility.getFullAXTree", "Accessibility.getPartialAXTree":
		m.t.Logf("Getting accessibility tree")
		return &CDPResponse{
			ID: cmd.ID,
			Result: map[string]interface{}{
				"nodes": []map[string]interface{}{
					{
						"nodeId":          "1",
						"backendDOMNodeId": 1,
						"role": map[string]interface{}{
							"value": "WebArea",
						},
						"name": map[string]interface{}{
							"value": "Mock Page",
						},
						"boundingBox": map[string]interface{}{
							"x":      0.0,
							"y":      0.0,
							"width":  1024.0,
							"height": 768.0,
						},
					},
					{
						"nodeId":          "2",
						"backendDOMNodeId": 2,
						"role": map[string]interface{}{
							"value": "button",
						},
						"name": map[string]interface{}{
							"value": "Click Me",
						},
						"boundingBox": map[string]interface{}{
							"x":      100.0,
							"y":      100.0,
							"width":  80.0,
							"height": 30.0,
						},
					},
				},
			},
		}

	case "Page.getFrameTree":
		m.t.Logf("Getting frame tree")
		return &CDPResponse{
			ID: cmd.ID,
			Result: map[string]interface{}{
				"frameTree": map[string]interface{}{
					"frame": map[string]interface{}{
						"id":             "frame-1",
						"loaderId":       "loader-1",
						"url":            "http://example.com",
						"securityOrigin": "http://example.com",
						"mimeType":       "text/html",
					},
				},
			},
		}

	case "DOM.getDocument":
		m.t.Logf("Getting DOM document")
		return &CDPResponse{
			ID: cmd.ID,
			Result: map[string]interface{}{
				"root": map[string]interface{}{
					"nodeId":   1,
					"nodeType": 9,
					"nodeName": "#document",
				},
			},
		}

	default:
		m.t.Logf("Unhandled CDP command: %s", cmd.Method)
		// Return empty result for unknown commands to prevent errors
		return &CDPResponse{
			ID:     cmd.ID,
			Result: map[string]interface{}{},
		}
	}
}

// removeConnection removes a connection from the active connections list
func (m *MockCDPServer) removeConnection(conn *websocket.Conn) {
	m.mu.Lock()
	defer m.mu.Unlock()

	for i, c := range m.conns {
		if c == conn {
			m.conns = append(m.conns[:i], m.conns[i+1:]...)
			return
		}
	}
}

// Crash simulates a server crash by closing all connections
func (m *MockCDPServer) Crash() {
	m.t.Log("💥 Simulating CDP server crash...")
	m.crashed.Store(true)

	m.mu.Lock()
	conns := make([]*websocket.Conn, len(m.conns))
	copy(conns, m.conns)
	m.mu.Unlock()

	// Close all connections abruptly
	for _, conn := range conns {
		_ = conn.Close()
	}

	m.mu.Lock()
	m.conns = nil
	m.mu.Unlock()

	m.t.Log("✓ All connections closed")
}

// Restart allows the server to accept new connections after a crash
func (m *MockCDPServer) Restart() {
	m.t.Log("🔄 Restarting mock CDP server...")
	m.crashed.Store(false)
	m.t.Log("✓ Server ready to accept connections")
}

// ConnectionCount returns the number of active connections
func (m *MockCDPServer) ConnectionCount() int {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return len(m.conns)
}

// SetCommandDelay configures a delay for a specific CDP command
// This simulates slow operations (e.g., large accessibility trees)
func (m *MockCDPServer) SetCommandDelay(method string, delay time.Duration) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.commandDelays[method] = delay
	m.t.Logf("Set %v delay for command: %s", delay, method)
}

// ClearCommandDelay removes the delay for a specific CDP command
func (m *MockCDPServer) ClearCommandDelay(method string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	delete(m.commandDelays, method)
	m.t.Logf("Cleared delay for command: %s", method)
}

// ClearAllCommandDelays removes all command delays
func (m *MockCDPServer) ClearAllCommandDelays() {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.commandDelays = make(map[string]time.Duration)
	m.t.Log("Cleared all command delays")
}

// Close shuts down the mock server
func (m *MockCDPServer) Close() {
	m.Crash()
	m.server.Close()
	m.t.Log("✓ Mock CDP server shut down")
}
