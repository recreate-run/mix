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

// RemoteCDPClient connects to remote CDP WebSocket endpoints (cloud browser providers)
// Implements BrowserClient interface for Browserbase, Brightdata, Hyperbrowser, etc.
type RemoteCDPClient struct {
	cdpURL string
	conn   *websocket.Conn
	connMu sync.RWMutex

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
		cdpURL:         cdpURL,
		conn:           conn,
		tabs:           make(map[string]*tabInfo),
		elementCache:   make(map[string][]elementInfo),
		screenshotSize: make(map[string]screenshotSize),
		pending:        make(map[int64]chan *cdp.Response),
		ctx:            clientCtx,
		cancel:         cancel,
	}

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
	c.connMu.Lock()
	err := c.conn.WriteJSON(command)
	c.connMu.Unlock()
	if err != nil {
		return nil, fmt.Errorf("failed to send CDP command %s: %w", method, err)
	}

	// Wait for response with timeout
	select {
	case response := <-responseChan:
		if response.Error != nil {
			return nil, fmt.Errorf("CDP error for %s: %s (code: %d)", method, response.Error.Message, response.Error.Code)
		}
		return response.Result, nil
	case <-ctx.Done():
		return nil, ctx.Err()
	case <-time.After(30 * time.Second):
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

		var msg json.RawMessage
		c.connMu.RLock()
		err := c.conn.ReadJSON(&msg)
		c.connMu.RUnlock()

		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseNormalClosure) {
				logging.Error("WebSocket read error", "error", err)
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

	c.connMu.Lock()
	err := c.conn.Close()
	c.connMu.Unlock()

	c.wg.Wait()
	return err
}
