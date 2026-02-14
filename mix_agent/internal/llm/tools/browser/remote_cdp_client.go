package browser

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"sync/atomic"
	"time"

	"github.com/gorilla/websocket"

	"mix/internal/llm/tools/browser/cdp"
	"mix/internal/logging"
)

// reconnectionState tracks the current reconnection state
type reconnectionState int32

const (
	stateConnected reconnectionState = iota
	stateDisconnected
	stateReconnecting
)

// CDP command timeouts - configured generously for remote browsers with network latency
// Note: With proper domain enabling (Page.enable, Accessibility.enable, etc.),
// accessibility tree operations should be nearly instant as the browser maintains the tree.
// These timeouts handle edge cases (massive pages, slow cloud providers, network issues).
var commandTimeouts = map[string]time.Duration{
	"Accessibility.getFullAXTree": 30 * time.Second, // Should be instant with domain enabled, but allow for network latency
	"Page.captureScreenshot":      30 * time.Second, // High-res screenshots + network transfer
	"DOM.getBoxModel":              5 * time.Second,
	"default":                      10 * time.Second,
}

// RemoteCDPClient connects to remote CDP WebSocket endpoints (cloud browser providers)
// Implements BrowserClient interface for Browserbase, Brightdata, Hyperbrowser, etc.
type RemoteCDPClient struct {
	cdpURL string
	conn   atomic.Pointer[websocket.Conn] // Lock-free connection access

	// Message ID management
	messageID atomic.Int64

	// Tab management
	tabsMu      sync.RWMutex
	tabs        map[string]*tabInfo // friendlyTabID → tab info
	tabCounter  uint64
	activeTabID string

	// Element cache (per tab)
	cacheMu      sync.RWMutex
	elementCache map[string][]elementInfo // tabID → elements

	// Screenshot size cache (per tab)
	screenshotMu   sync.RWMutex
	screenshotSize map[string]screenshotSize // tabID → size in pixels

	// Response channels for synchronous request/response
	pendingMu sync.RWMutex
	pending   map[int64]chan *cdp.Response

	// Event handling
	eventListeners sync.Map // method → []func(params interface{})

	// Lifecycle management
	ctx    context.Context
	cancel context.CancelFunc
	wg     sync.WaitGroup

	// Connection failure notification
	closed   chan struct{}
	closedMu sync.Mutex

	// Reconnection management
	reconnectState       atomic.Int32 // reconnectionState
	reconnectAttempts    atomic.Int32
	maxReconnectAttempts int
}

// NewRemoteCDPClient creates a new remote CDP WebSocket client
func NewRemoteCDPClient(ctx context.Context, cdpURL string) (*RemoteCDPClient, error) {
	if cdpURL == "" {
		return nil, fmt.Errorf("CDP URL cannot be empty")
	}

	// Connect to CDP WebSocket
	conn, resp, err := websocket.DefaultDialer.DialContext(ctx, cdpURL, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to CDP endpoint %s: %w", cdpURL, err)
	}
	if resp.Body != nil {
		_ = resp.Body.Close()
	}

	clientCtx, cancel := context.WithCancel(ctx)

	client := &RemoteCDPClient{
		cdpURL:               cdpURL,
		tabs:                 make(map[string]*tabInfo),
		elementCache:         make(map[string][]elementInfo),
		screenshotSize:       make(map[string]screenshotSize),
		pending:              make(map[int64]chan *cdp.Response),
		ctx:                  clientCtx,
		cancel:               cancel,
		closed:               make(chan struct{}),
		maxReconnectAttempts: 5,
	}
	client.conn.Store(conn)
	client.reconnectState.Store(int32(stateConnected))

	// Start message reader
	client.wg.Add(1)
	go client.readMessages()

	logging.Info("Connected to remote CDP endpoint", "url", cdpURL)

	return client, nil
}

// getNextMessageID returns the next message ID
func (c *RemoteCDPClient) getNextMessageID() int64 {
	return c.messageID.Add(1)
}

// generateFriendlyTabID generates a new friendly tab ID
func (c *RemoteCDPClient) generateFriendlyTabID() string {
	counter := atomic.AddUint64(&c.tabCounter, 1)
	return fmt.Sprintf("tab-%d", counter)
}

// getTabCDPSessionID returns the CDP session ID for a given tab
func (c *RemoteCDPClient) getTabCDPSessionID(tabID ...string) (string, error) {
	c.tabsMu.RLock()
	defer c.tabsMu.RUnlock()

	var targetTabID string
	if len(tabID) > 0 && tabID[0] != "" {
		targetTabID = tabID[0]
	} else {
		targetTabID = c.activeTabID
	}

	if targetTabID == "" {
		return "", fmt.Errorf("no active tab and no tabID provided")
	}

	tab, exists := c.tabs[targetTabID]
	if !exists {
		return "", fmt.Errorf("tab not found: %s", targetTabID)
	}

	return tab.cdpSessionID, nil
}

// setLastScreenshotSize stores screenshot dimensions for coordinate scaling
func (c *RemoteCDPClient) setLastScreenshotSize(tabID string, width, height int) {
	if tabID == "" {
		return
	}
	c.screenshotMu.Lock()
	c.screenshotSize[tabID] = screenshotSize{Width: width, Height: height}
	c.screenshotMu.Unlock()
}


// getCommandTimeout returns the timeout for a specific CDP command
func getCommandTimeout(method string) time.Duration {
	if timeout, exists := commandTimeouts[method]; exists {
		return timeout
	}
	return commandTimeouts["default"]
}

// sendCommand sends a CDP command and waits for response
func (c *RemoteCDPClient) sendCommand(ctx context.Context, method string, params interface{}, sessionID ...string) (interface{}, error) {
	messageID := c.getNextMessageID()

	// Create command
	command := cdp.Command{
		ID:     int(messageID),
		Method: method,
		Params: params,
	}

	// Add session ID if provided (for tab-specific commands)
	if len(sessionID) > 0 && sessionID[0] != "" {
		command.SessionID = sessionID[0]
	}

	// Create response channel
	responseChan := make(chan *cdp.Response, 1)
	c.pendingMu.Lock()
	c.pending[messageID] = responseChan
	c.pendingMu.Unlock()

	// Clean up channel on exit
	defer func() {
		c.pendingMu.Lock()
		delete(c.pending, messageID)
		c.pendingMu.Unlock()
	}()

	// Send command
	conn := c.conn.Load()
	if conn == nil {
		return nil, fmt.Errorf("connection closed while sending CDP command %s", method)
	}
	err := conn.WriteJSON(command)
	if err != nil {
		return nil, fmt.Errorf("failed to send CDP command %s: %w", method, err)
	}

	// Get timeout for this specific command
	timeout := getCommandTimeout(method)

	// Wait for response with command-specific timeout
	select {
	case response := <-responseChan:
		if response.Error != nil {
			return nil, fmt.Errorf("CDP error for %s: %s (code: %d)", method, response.Error.Message, response.Error.Code)
		}
		return response.Result, nil
	case <-ctx.Done():
		return nil, ctx.Err()
	case <-c.closed:
		return nil, fmt.Errorf("connection closed while waiting for CDP response to %s", method)
	case <-time.After(timeout):
		// Timeout - treat as connection failure and trigger reconnection
		logging.Warn("CDP command timeout - triggering reconnection", "method", method, "timeout", timeout)
		c.reconnectState.Store(int32(stateDisconnected))
		c.notifyPendingRequests()

		// Attempt reconnection in background (only if not already reconnecting)
		if c.reconnectState.CompareAndSwap(int32(stateDisconnected), int32(stateReconnecting)) {
			go c.attemptReconnection()
		}

		return nil, fmt.Errorf("timeout waiting for CDP response to %s", method)
	}
}

// readMessages continuously reads messages from WebSocket
func (c *RemoteCDPClient) readMessages() {
	defer c.wg.Done()

	for {
		select {
		case <-c.ctx.Done():
			return
		default:
		}

		// Read message using atomic pointer (lock-free)
		conn := c.conn.Load()
		if conn == nil {
			return
		}

		var msg json.RawMessage
		err := conn.ReadJSON(&msg)

		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseNormalClosure) {
				logging.Error("WebSocket read error", "error", err)
			}

			// Mark as disconnected and notify pending requests
			c.reconnectState.Store(int32(stateDisconnected))
			c.notifyPendingRequests()

			// Attempt reconnection in background (only if not already reconnecting)
			if c.reconnectState.CompareAndSwap(int32(stateDisconnected), int32(stateReconnecting)) {
				go c.attemptReconnection()
			}
			return
		}

		// Try to parse as response (has "id" field)
		var response cdp.Response
		if err := json.Unmarshal(msg, &response); err == nil && response.ID > 0 {
			c.handleResponse(&response)
			continue
		}

		// Try to parse as event (has "method" field, no "id")
		var event struct {
			Method string          `json:"method"`
			Params json.RawMessage `json:"params"`
		}
		if err := json.Unmarshal(msg, &event); err == nil && event.Method != "" {
			c.handleEvent(event.Method, event.Params)
			continue
		}
	}
}

// handleResponse routes response to waiting request
func (c *RemoteCDPClient) handleResponse(response *cdp.Response) {
	c.pendingMu.RLock()
	ch, exists := c.pending[int64(response.ID)]
	c.pendingMu.RUnlock()

	if exists {
		select {
		case ch <- response:
		default:
			// Channel full or closed
		}
	}
}

// handleEvent handles CDP events
func (c *RemoteCDPClient) handleEvent(method string, params json.RawMessage) {
	// Log events for debugging
	logging.Debug("CDP event received", "method", method)

	// Call registered listeners
	if listeners, ok := c.eventListeners.Load(method); ok {
		for _, listener := range listeners.([]func(interface{})) {
			listener(params)
		}
	}
}

// Close closes the CDP connection
func (c *RemoteCDPClient) Close() error {
	c.cancel()

	conn := c.conn.Load()
	var err error
	if conn != nil {
		err = conn.Close()
	}

	c.wg.Wait()
	return err
}

// IsConnected returns whether the client is currently connected
func (c *RemoteCDPClient) IsConnected() bool {
	return reconnectionState(c.reconnectState.Load()) == stateConnected
}

// notifyPendingRequests closes the closed channel to notify all pending requests
func (c *RemoteCDPClient) notifyPendingRequests() {
	c.closedMu.Lock()
	defer c.closedMu.Unlock()

	select {
	case <-c.closed:
		// Already closed
	default:
		close(c.closed)
	}
}

// attemptReconnection tries to reconnect with exponential backoff
// Only one instance of this goroutine runs at a time (enforced by CompareAndSwap)
func (c *RemoteCDPClient) attemptReconnection() {
	for c.reconnectAttempts.Load() < int32(c.maxReconnectAttempts) {
		attempt := c.reconnectAttempts.Add(1)

		// Exponential backoff: 2^attempt seconds, capped at 30s
		backoffDuration := time.Duration(1<<uint(attempt)) * time.Second
		if backoffDuration > 30*time.Second {
			backoffDuration = 30 * time.Second
		}

		logging.Info("Attempting reconnection",
			"attempt", attempt,
			"maxAttempts", c.maxReconnectAttempts,
			"backoff", backoffDuration,
			"url", c.cdpURL)

		time.Sleep(backoffDuration)

		// Try to reconnect
		if err := c.reconnect(); err != nil {
			logging.Error("Reconnection attempt failed",
				"attempt", attempt,
				"error", err)
			continue
		}

		// Reconnection successful
		logging.Info("Successfully reconnected to CDP endpoint",
			"attempt", attempt,
			"url", c.cdpURL)
		c.reconnectAttempts.Store(0)
		c.reconnectState.Store(int32(stateConnected))
		return
	}

	logging.Error("Max reconnection attempts reached, giving up",
		"maxAttempts", c.maxReconnectAttempts,
		"url", c.cdpURL)
	// Stay in stateReconnecting to prevent further attempts
}

// reconnect closes the old connection and establishes a new one
func (c *RemoteCDPClient) reconnect() error {
	// Close old connection
	oldConn := c.conn.Load()
	if oldConn != nil {
		_ = oldConn.Close()
	}

	// Connect to CDP WebSocket
	conn, resp, err := websocket.DefaultDialer.DialContext(c.ctx, c.cdpURL, nil)
	if err != nil {
		return fmt.Errorf("failed to reconnect to CDP endpoint: %w", err)
	}
	if resp.Body != nil {
		_ = resp.Body.Close()
	}

	// Update connection atomically
	c.conn.Store(conn)

	// Reset closed channel
	c.closedMu.Lock()
	c.closed = make(chan struct{})
	c.closedMu.Unlock()

	// Restart message reader
	c.wg.Add(1)
	go c.readMessages()

	return nil
}
