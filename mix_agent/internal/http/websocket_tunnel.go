package http

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"sync"
	"time"

	"mix/internal/browser_logger"
	"mix/internal/logging"
	"mix/internal/session"

	"github.com/coder/websocket"
)

// Sentinel errors for tunnel operations
var (
	ErrBrowserNotConnected = errors.New("browser not connected")
	ErrSendTimeout         = errors.New("send timeout - browser not consuming messages")
	ErrResponseTimeout     = errors.New("response timeout - browser did not respond")
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

// TunnelConnection represents a browser connection
type TunnelConnection struct {
	SessionID       string
	Token           string
	Conn            *websocket.Conn
	SendChan        chan []byte
	Context         context.Context
	Cancel          context.CancelFunc
	pendingRequests sync.Map // requestId (string) -> chan CDPResponse
}

// normalizeID converts any ID type to a consistent string for map lookups
func normalizeID(id interface{}) string {
	if id == nil {
		return ""
	}
	return fmt.Sprintf("%v", id)
}

// TunnelRegistry manages all active tunnel connections
type TunnelRegistry struct {
	mu            sync.RWMutex
	connections   map[string]*TunnelConnection // sessionId -> connection
	storageConfig session.Config
}

// NewTunnelRegistry creates a new tunnel registry
func NewTunnelRegistry(storageConfig session.Config) *TunnelRegistry {
	return &TunnelRegistry{
		connections:   make(map[string]*TunnelConnection),
		storageConfig: storageConfig,
	}
}

// HandleTunnelConnection handles WebSocket tunnel connections from Electron browsers
func (registry *TunnelRegistry) HandleTunnelConnection(w http.ResponseWriter, r *http.Request) {
	// Extract sessionId from URL path: /api/v1/tunnel/cdp/session/{sessionId}
	sessionID := r.PathValue("sessionId")
	if sessionID == "" {
		http.Error(w, "Missing sessionId", http.StatusBadRequest)
		return
	}

	// Extract token from query parameter
	token := r.URL.Query().Get("token")
	if token == "" {
		http.Error(w, "Missing token", http.StatusUnauthorized)
		return
	}

	// TODO: Validate token against session
	// For now, accept any token

	// Upgrade connection to WebSocket
	conn, err := websocket.Accept(w, r, &websocket.AcceptOptions{
		CompressionMode: websocket.CompressionContextTakeover,
	})
	if err != nil {
		logging.Error("Failed to upgrade connection to WebSocket", "error", err)
		return
	}

	// Set read limit to 10MB for large screenshots and accessibility trees
	conn.SetReadLimit(10 * 1024 * 1024)

	ctx, cancel := context.WithCancel(r.Context())
	defer cancel()

	tunnelConn := &TunnelConnection{
		SessionID: sessionID,
		Token:     token,
		Conn:      conn,
		SendChan:  make(chan []byte, 50),
		Context:   ctx,
		Cancel:    cancel,
	}

	// Register connection
	registry.mu.Lock()
	registry.connections[sessionID] = tunnelConn
	registry.mu.Unlock()

	// Cleanup on disconnect
	defer func() {
		registry.mu.Lock()
		delete(registry.connections, sessionID)
		registry.mu.Unlock()
		_ = conn.Close(websocket.StatusNormalClosure, "")
		logging.Info("Browser disconnected from tunnel", "session", sessionID)
	}()

	// Start send goroutine
	go tunnelConn.sendLoop()

	// Start ping goroutine
	go tunnelConn.pingLoop()

	// Read messages from browser (blocks until connection closes)
	tunnelConn.readLoop(registry.storageConfig)
}

// readLoop reads messages from the browser
func (tc *TunnelConnection) readLoop(storageConfig session.Config) {
	for {
		_, message, err := tc.Conn.Read(tc.Context)
		if err != nil {
			if websocket.CloseStatus(err) != websocket.StatusNormalClosure {
				logging.Error("Tunnel read error", "error", err, "session", tc.SessionID)
			}
			return
		}

		// Check if this is a browser_log message (one-way event)
		// Browser logs have {"type": "browser_log", ...} while CDP responses have {"id": ..., "result": ...}
		var msgType struct {
			Type string `json:"type"`
		}
		if err := json.Unmarshal(message, &msgType); err == nil && msgType.Type == "browser_log" {
			// Handle browser log message
			var logMessage browser_logger.BrowserLogMessage
			if err := json.Unmarshal(message, &logMessage); err != nil {
				logging.Error("Failed to parse browser log message", "error", err, "session", tc.SessionID)
				continue
			}

			// Log received browser log message for debugging/verification
			logging.Debug("Browser log received", "session", logMessage.SessionID, "tabId", logMessage.TabID, "logType", logMessage.LogType, "timestamp", logMessage.Timestamp)

			// Write log (fire-and-forget, errors logged but don't crash handler)
			if err := browser_logger.AppendLog(logMessage, storageConfig); err != nil {
				logging.Error("Failed to write browser log", "error", err, "session", tc.SessionID)
			}
			continue
		}

		// Parse CDP response from browser
		var response CDPResponse
		if err := json.Unmarshal(message, &response); err != nil {
			logging.Error("Failed to parse CDP response", "error", err, "session", tc.SessionID)
			continue
		}

		// Route to waiting request using normalized ID
		normalizedID := normalizeID(response.ID)
		if ch, ok := tc.pendingRequests.LoadAndDelete(normalizedID); ok {
			responseChan := ch.(chan CDPResponse)
			select {
			case responseChan <- response:
			case <-time.After(time.Second):
				logging.Warn("Response channel full, dropping response", "id", response.ID, "session", tc.SessionID)
			}
		}
	}
}

// sendLoop sends messages to the browser
func (tc *TunnelConnection) sendLoop() {
	for {
		select {
		case <-tc.Context.Done():
			return
		case message := <-tc.SendChan:
			ctx, cancel := context.WithTimeout(tc.Context, 10*time.Second)
			err := tc.Conn.Write(ctx, websocket.MessageText, message)
			cancel()
			if err != nil {
				logging.Error("Tunnel write error", "error", err, "session", tc.SessionID)
				return
			}
		}
	}
}

// pingLoop sends periodic pings
func (tc *TunnelConnection) pingLoop() {
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-tc.Context.Done():
			return
		case <-ticker.C:
			ctx, cancel := context.WithTimeout(tc.Context, 5*time.Second)
			err := tc.Conn.Ping(ctx)
			cancel()
			if err != nil {
				logging.Warn("Ping failed, closing connection", "error", err, "session", tc.SessionID)
				tc.Cancel()
				return
			}
		}
	}
}

// SendCommandToTunnel sends a CDP command to the browser and waits for response
func (registry *TunnelRegistry) SendCommandToTunnel(sessionID string, command CDPRequest) (*CDPResponse, error) {
	registry.mu.RLock()
	conn := registry.connections[sessionID]
	registry.mu.RUnlock()

	if conn == nil {
		return nil, fmt.Errorf("%w for session %s", ErrBrowserNotConnected, sessionID)
	}

	// Create response channel and store with normalized ID
	responseChan := make(chan CDPResponse, 1)
	normalizedID := normalizeID(command.ID)
	conn.pendingRequests.Store(normalizedID, responseChan)
	defer conn.pendingRequests.Delete(normalizedID)

	// Marshal command
	message, err := json.Marshal(command)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal command: %w", err)
	}


	// Send command (non-blocking with timeout)
	select {
	case conn.SendChan <- message:
		// Command sent successfully
	case <-time.After(5 * time.Second):
		return nil, ErrSendTimeout
	}

	// Wait for response with timeout
	select {
	case response := <-responseChan:
		return &response, nil
	case <-time.After(30 * time.Second):
		return nil, ErrResponseTimeout
	}
}

// GetActiveTunnels returns list of active session IDs
func (registry *TunnelRegistry) GetActiveTunnels() []string {
	registry.mu.RLock()
	defer registry.mu.RUnlock()

	sessions := make([]string, 0, len(registry.connections))
	for sessionID := range registry.connections {
		sessions = append(sessions, sessionID)
	}
	return sessions
}
