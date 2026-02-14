package client

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"sync/atomic"
	"time"

	"github.com/gorilla/websocket"
	"github.com/sarathmenon/browser-service/internal/constants"
	"github.com/sarathmenon/browser-service/pkg/protocol"
)

// Client is a Go client for the browser service WebSocket API
type Client struct {
	conn           *websocket.Conn
	endpoint       string
	requestID      uint64
	mu             sync.Mutex
	pending        map[string]chan protocol.Response
	eventListeners []chan protocol.Event
	readDone       chan struct{}
	connected      bool
	closeOnce      sync.Once
}

// New creates a new client and connects to the WebSocket endpoint
func New(endpoint string) (*Client, error) {
	conn, _, err := websocket.DefaultDialer.Dial(endpoint, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to %s: %w", endpoint, err)
	}

	client := &Client{
		conn:      conn,
		endpoint:  endpoint,
		pending:   make(map[string]chan protocol.Response),
		readDone:  make(chan struct{}),
		connected: true,
	}

	// Start read loop in background
	go client.readLoop()

	return client, nil
}

// Navigate navigates to a URL
func (c *Client) Navigate(ctx context.Context, url string, tabID ...string) (*protocol.NavigateResult, error) {
	params := protocol.NavigateParams{
		URL: url,
	}
	if len(tabID) > 0 {
		params.TabID = &tabID[0]
	}

	resp, err := c.sendRequest(ctx, constants.MethodPageNavigate, params)
	if err != nil {
		return nil, err
	}

	if resp.Error != nil {
		return nil, fmt.Errorf("navigate error: %s", resp.Error.Message)
	}

	// Parse result
	data, err := json.Marshal(resp.Result)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal result: %w", err)
	}

	var result protocol.NavigateResult
	if err := json.Unmarshal(data, &result); err != nil {
		return nil, fmt.Errorf("failed to unmarshal result: %w", err)
	}

	return &result, nil
}

// Screenshot captures a screenshot with the given parameters
func (c *Client) Screenshot(ctx context.Context, params protocol.ScreenshotParams) (*protocol.ScreenshotResult, error) {
	resp, err := c.sendRequest(ctx, constants.MethodPageScreenshot, params)
	if err != nil {
		return nil, err
	}

	if resp.Error != nil {
		return nil, fmt.Errorf("screenshot error: %s", resp.Error.Message)
	}

	// Parse result
	data, err := json.Marshal(resp.Result)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal result: %w", err)
	}

	var result protocol.ScreenshotResult
	if err := json.Unmarshal(data, &result); err != nil {
		return nil, fmt.Errorf("failed to unmarshal result: %w", err)
	}

	return &result, nil
}

// GetElements returns interactive elements on the page
func (c *Client) GetElements(ctx context.Context, tabID ...string) ([]protocol.RawAccessibilityNode, error) {
	var params *protocol.GetElementsParams
	if len(tabID) > 0 {
		params = &protocol.GetElementsParams{
			TabID: &tabID[0],
		}
	}

	resp, err := c.sendRequest(ctx, constants.MethodPageGetElements, params)
	if err != nil {
		return nil, err
	}

	if resp.Error != nil {
		return nil, fmt.Errorf("getElements error: %s", resp.Error.Message)
	}

	// Parse result
	data, err := json.Marshal(resp.Result)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal result: %w", err)
	}

	var result protocol.GetElementsResult
	if err := json.Unmarshal(data, &result); err != nil {
		return nil, fmt.Errorf("failed to unmarshal result: %w", err)
	}

	return result.Elements, nil
}

// ReadPage returns accessibility tree for visible viewport elements
func (c *Client) ReadPage(ctx context.Context, interactiveOnly bool, tabID ...string) (*protocol.ReadPageResult, error) {
	params := protocol.ReadPageParams{
		InteractiveOnly: interactiveOnly,
	}
	if len(tabID) > 0 {
		params.TabID = &tabID[0]
	}

	resp, err := c.sendRequest(ctx, constants.MethodPageReadPage, params)
	if err != nil {
		return nil, err
	}

	if resp.Error != nil {
		return nil, fmt.Errorf("readPage error: %s", resp.Error.Message)
	}

	data, err := json.Marshal(resp.Result)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal result: %w", err)
	}

	var result protocol.ReadPageResult
	if err := json.Unmarshal(data, &result); err != nil {
		return nil, fmt.Errorf("failed to unmarshal result: %w", err)
	}

	return &result, nil
}

// CreateTab creates a new browser tab
// CreateTab creates a new browser tab, optionally navigating to a URL
func (c *Client) CreateTab(ctx context.Context, url ...string) (*protocol.TabInfo, error) {
	var params any
	if len(url) > 0 && url[0] != "" {
		params = &protocol.TabCreateParams{URL: &url[0]}
	}

	resp, err := c.sendRequest(ctx, constants.MethodTabCreate, params)
	if err != nil {
		return nil, err
	}

	if resp.Error != nil {
		return nil, fmt.Errorf("createTab error: %s", resp.Error.Message)
	}

	// Parse result
	data, err := json.Marshal(resp.Result)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal result: %w", err)
	}

	var result protocol.TabCreateResult
	if err := json.Unmarshal(data, &result); err != nil {
		return nil, fmt.Errorf("failed to unmarshal result: %w", err)
	}

	return &result.Tab, nil
}

// ListTabs lists all browser tabs
func (c *Client) ListTabs(ctx context.Context) (*protocol.TabListResult, error) {
	resp, err := c.sendRequest(ctx, constants.MethodTabList, nil)
	if err != nil {
		return nil, err
	}

	if resp.Error != nil {
		return nil, fmt.Errorf("listTabs error: %s", resp.Error.Message)
	}

	// Parse result
	data, err := json.Marshal(resp.Result)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal result: %w", err)
	}

	var result protocol.TabListResult
	if err := json.Unmarshal(data, &result); err != nil {
		return nil, fmt.Errorf("failed to unmarshal result: %w", err)
	}

	return &result, nil
}

// SwitchTab switches to the specified tab
func (c *Client) SwitchTab(ctx context.Context, tabID string) error {
	params := protocol.TabSwitchParams{TabID: tabID}

	resp, err := c.sendRequest(ctx, constants.MethodTabSwitch, params)
	if err != nil {
		return err
	}

	if resp.Error != nil {
		return fmt.Errorf("switchTab error: %s", resp.Error.Message)
	}

	return nil
}

// CloseTab closes the specified tab
func (c *Client) CloseTab(ctx context.Context, tabID string) error {
	params := protocol.TabCloseParams{TabID: tabID}

	resp, err := c.sendRequest(ctx, constants.MethodTabClose, params)
	if err != nil {
		return err
	}

	if resp.Error != nil {
		return fmt.Errorf("closeTab error: %s", resp.Error.Message)
	}

	return nil
}

// Click clicks an element by index
func (c *Client) Click(ctx context.Context, index int, tabID ...string) error {
	params := protocol.ClickParams{
		Index: index,
	}
	if len(tabID) > 0 {
		params.TabID = &tabID[0]
	}

	resp, err := c.sendRequest(ctx, constants.MethodPageClick, params)
	if err != nil {
		return err
	}

	if resp.Error != nil {
		return fmt.Errorf("click error: %s", resp.Error.Message)
	}

	return nil
}

// RightClick right-clicks an element
func (c *Client) RightClick(ctx context.Context, index int, tabID ...string) error {
	params := protocol.RightClickParams{Index: index}
	if len(tabID) > 0 {
		params.TabID = &tabID[0]
	}

	resp, err := c.sendRequest(ctx, constants.MethodPageRightClick, params)
	if err != nil {
		return err
	}

	if resp.Error != nil {
		return fmt.Errorf("rightClick error: %s", resp.Error.Message)
	}

	return nil
}

// DoubleClick double-clicks an element
func (c *Client) DoubleClick(ctx context.Context, index int, tabID ...string) error {
	params := protocol.DoubleClickParams{Index: index}
	if len(tabID) > 0 {
		params.TabID = &tabID[0]
	}

	resp, err := c.sendRequest(ctx, constants.MethodPageDoubleClick, params)
	if err != nil {
		return err
	}

	if resp.Error != nil {
		return fmt.Errorf("doubleClick error: %s", resp.Error.Message)
	}

	return nil
}

// TripleClick triple-clicks an element
func (c *Client) TripleClick(ctx context.Context, index int, tabID ...string) error {
	params := protocol.TripleClickParams{Index: index}
	if len(tabID) > 0 {
		params.TabID = &tabID[0]
	}

	resp, err := c.sendRequest(ctx, constants.MethodPageTripleClick, params)
	if err != nil {
		return err
	}

	if resp.Error != nil {
		return fmt.Errorf("tripleClick error: %s", resp.Error.Message)
	}

	return nil
}

// ClickByBackendID clicks an element by its CDP backend node ID
func (c *Client) ClickByBackendID(ctx context.Context, backendID int64, tabID ...string) error {
	params := protocol.ClickByBackendIDParams{
		BackendID: backendID,
	}
	if len(tabID) > 0 {
		params.TabID = &tabID[0]
	}

	resp, err := c.sendRequest(ctx, constants.MethodPageClickByBackendID, params)
	if err != nil {
		return err
	}

	if resp.Error != nil {
		return fmt.Errorf("clickByBackendID error: %s", resp.Error.Message)
	}

	return nil
}

// RightClickByBackendID right-clicks an element by its CDP backend node ID
func (c *Client) RightClickByBackendID(ctx context.Context, backendID int64, tabID ...string) error {
	params := protocol.RightClickByBackendIDParams{
		BackendID: backendID,
	}
	if len(tabID) > 0 {
		params.TabID = &tabID[0]
	}

	resp, err := c.sendRequest(ctx, constants.MethodPageRightClickByBackendID, params)
	if err != nil {
		return err
	}

	if resp.Error != nil {
		return fmt.Errorf("rightClickByBackendID error: %s", resp.Error.Message)
	}

	return nil
}

// DoubleClickByBackendID double-clicks an element by its CDP backend node ID
func (c *Client) DoubleClickByBackendID(ctx context.Context, backendID int64, tabID ...string) error {
	params := protocol.DoubleClickByBackendIDParams{
		BackendID: backendID,
	}
	if len(tabID) > 0 {
		params.TabID = &tabID[0]
	}

	resp, err := c.sendRequest(ctx, constants.MethodPageDoubleClickByBackendID, params)
	if err != nil {
		return err
	}

	if resp.Error != nil {
		return fmt.Errorf("doubleClickByBackendID error: %s", resp.Error.Message)
	}

	return nil
}

// TripleClickByBackendID triple-clicks an element by its CDP backend node ID
func (c *Client) TripleClickByBackendID(ctx context.Context, backendID int64, tabID ...string) error {
	params := protocol.TripleClickByBackendIDParams{
		BackendID: backendID,
	}
	if len(tabID) > 0 {
		params.TabID = &tabID[0]
	}

	resp, err := c.sendRequest(ctx, constants.MethodPageTripleClickByBackendID, params)
	if err != nil {
		return err
	}

	if resp.Error != nil {
		return fmt.Errorf("tripleClickByBackendID error: %s", resp.Error.Message)
	}

	return nil
}

// ClickAt clicks at specific coordinates
func (c *Client) ClickAt(ctx context.Context, x, y float64, button *string, clickCount *int, duration *int, tabID ...string) error {
	params := protocol.ClickAtParams{
		X:          x,
		Y:          y,
		Button:     button,
		ClickCount: clickCount,
		Duration:   duration,
	}
	if len(tabID) > 0 {
		params.TabID = &tabID[0]
	}

	resp, err := c.sendRequest(ctx, constants.MethodPageClickAt, params)
	if err != nil {
		return err
	}

	if resp.Error != nil {
		return fmt.Errorf("clickAt error: %s", resp.Error.Message)
	}

	return nil
}

// RightClickAt right-clicks at specific coordinates
func (c *Client) RightClickAt(ctx context.Context, x, y float64, duration *int, tabID ...string) error {
	params := protocol.RightClickAtParams{
		X:        x,
		Y:        y,
		Duration: duration,
	}
	if len(tabID) > 0 {
		params.TabID = &tabID[0]
	}

	resp, err := c.sendRequest(ctx, constants.MethodPageRightClickAt, params)
	if err != nil {
		return err
	}

	if resp.Error != nil {
		return fmt.Errorf("rightClickAt error: %s", resp.Error.Message)
	}

	return nil
}

// DoubleClickAt double-clicks at specific coordinates
func (c *Client) DoubleClickAt(ctx context.Context, x, y float64, button *string, duration *int, tabID ...string) error {
	params := protocol.DoubleClickAtParams{
		X:        x,
		Y:        y,
		Button:   button,
		Duration: duration,
	}
	if len(tabID) > 0 {
		params.TabID = &tabID[0]
	}

	resp, err := c.sendRequest(ctx, constants.MethodPageDoubleClickAt, params)
	if err != nil {
		return err
	}

	if resp.Error != nil {
		return fmt.Errorf("doubleClickAt error: %s", resp.Error.Message)
	}

	return nil
}

// TripleClickAt triple-clicks at specific coordinates
func (c *Client) TripleClickAt(ctx context.Context, x, y float64, button *string, duration *int, tabID ...string) error {
	params := protocol.TripleClickAtParams{
		X:        x,
		Y:        y,
		Button:   button,
		Duration: duration,
	}
	if len(tabID) > 0 {
		params.TabID = &tabID[0]
	}

	resp, err := c.sendRequest(ctx, constants.MethodPageTripleClickAt, params)
	if err != nil {
		return err
	}

	if resp.Error != nil {
		return fmt.Errorf("tripleClickAt error: %s", resp.Error.Message)
	}

	return nil
}

// Drag performs a drag operation either by index or coordinates
func (c *Client) Drag(ctx context.Context, fromIndex, toIndex *int, fromX, fromY, toX, toY *float64, duration *int, tabID ...string) error {
	params := protocol.DragParams{
		FromIndex: fromIndex,
		ToIndex:   toIndex,
		FromX:     fromX,
		FromY:     fromY,
		ToX:       toX,
		ToY:       toY,
		Duration:  duration,
	}
	if len(tabID) > 0 {
		params.TabID = &tabID[0]
	}

	resp, err := c.sendRequest(ctx, constants.MethodPageDrag, params)
	if err != nil {
		return err
	}

	if resp.Error != nil {
		return fmt.Errorf("drag error: %s", resp.Error.Message)
	}

	return nil
}

// FormInput sets form input value directly
func (c *Client) FormInput(ctx context.Context, index int, value string, tabID ...string) error {
	params := protocol.FormInputParams{
		Index: index,
		Value: value,
	}
	if len(tabID) > 0 {
		params.TabID = &tabID[0]
	}

	resp, err := c.sendRequest(ctx, constants.MethodPageFormInput, params)
	if err != nil {
		return err
	}

	if resp.Error != nil {
		return fmt.Errorf("formInput error: %s", resp.Error.Message)
	}

	return nil
}

// GoBack navigates backward in browser history
func (c *Client) GoBack(ctx context.Context, tabID ...string) (string, error) {
	var params *protocol.GoBackParams
	if len(tabID) > 0 {
		params = &protocol.GoBackParams{
			TabID: &tabID[0],
		}
	}

	resp, err := c.sendRequest(ctx, constants.MethodPageGoBack, params)
	if err != nil {
		return "", err
	}

	if resp.Error != nil {
		return "", fmt.Errorf("goBack error: %s", resp.Error.Message)
	}

	// Parse result
	data, err := json.Marshal(resp.Result)
	if err != nil {
		return "", fmt.Errorf("failed to marshal result: %w", err)
	}

	var result protocol.GoBackResult
	if err := json.Unmarshal(data, &result); err != nil {
		return "", fmt.Errorf("failed to unmarshal goBack result: %w", err)
	}

	return result.URL, nil
}

// GoForward navigates forward in browser history
func (c *Client) GoForward(ctx context.Context, tabID ...string) (string, error) {
	var params *protocol.GoForwardParams
	if len(tabID) > 0 {
		params = &protocol.GoForwardParams{
			TabID: &tabID[0],
		}
	}

	resp, err := c.sendRequest(ctx, constants.MethodPageGoForward, params)
	if err != nil {
		return "", err
	}

	if resp.Error != nil {
		return "", fmt.Errorf("goForward error: %s", resp.Error.Message)
	}

	// Parse result
	data, err := json.Marshal(resp.Result)
	if err != nil {
		return "", fmt.Errorf("failed to marshal result: %w", err)
	}

	var result protocol.GoForwardResult
	if err := json.Unmarshal(data, &result); err != nil {
		return "", fmt.Errorf("failed to unmarshal goForward result: %w", err)
	}

	return result.URL, nil
}

// Wait pauses execution for the specified duration in milliseconds
func (c *Client) Wait(ctx context.Context, duration int, tabID ...string) error {
	params := protocol.WaitParams{Duration: duration}
	if len(tabID) > 0 {
		params.TabID = &tabID[0]
	}

	resp, err := c.sendRequest(ctx, constants.MethodPageWait, params)
	if err != nil {
		return err
	}
	if resp.Error != nil {
		return fmt.Errorf("wait error: %s", resp.Error.Message)
	}
	return nil
}

// Type types text into an element
func (c *Client) Type(ctx context.Context, index *int, text string, tabID ...string) error {
	params := protocol.TypeParams{
		Index: index,
		Text:  text,
	}
	if len(tabID) > 0 {
		params.TabID = &tabID[0]
	}

	resp, err := c.sendRequest(ctx, constants.MethodPageType, params)
	if err != nil {
		return err
	}

	if resp.Error != nil {
		return fmt.Errorf("type error: %s", resp.Error.Message)
	}

	return nil
}

// Scroll scrolls the page in a direction
func (c *Client) Scroll(ctx context.Context, direction string, amount int, tabID ...string) error {
	params := protocol.ScrollParams{
		Direction: direction,
		Amount:    amount,
	}
	if len(tabID) > 0 {
		params.TabID = &tabID[0]
	}

	resp, err := c.sendRequest(ctx, constants.MethodPageScroll, params)
	if err != nil {
		return err
	}

	if resp.Error != nil {
		return fmt.Errorf("scroll error: %s", resp.Error.Message)
	}

	return nil
}

// Close closes the browser context
// SubscribeToEvents creates a new event subscription channel
// The channel will receive all browser events (crashes, timeouts, etc.)
// The channel is buffered with size 100 to prevent blocking
// The channel will be closed when the client connection is closed
func (c *Client) SubscribeToEvents() <-chan protocol.Event {
	c.mu.Lock()
	defer c.mu.Unlock()

	ch := make(chan protocol.Event, 100)
	c.eventListeners = append(c.eventListeners, ch)
	return ch
}

func (c *Client) Close() error {
	resp, err := c.sendRequest(context.Background(), constants.MethodBrowserClose, nil)
	if err != nil {
		return err
	}

	if resp.Error != nil {
		return fmt.Errorf("close error: %s", resp.Error.Message)
	}

	// Close WebSocket connection
	c.mu.Lock()
	c.connected = false
	c.mu.Unlock()

	c.closeOnce.Do(func() {
		close(c.readDone)
	})
	return c.conn.Close()
}

// ImportCookies imports cookies from a browser
func (c *Client) ImportCookies(ctx context.Context, browser, profile string) (*protocol.ImportCookiesResult, error) {
	params := protocol.ImportCookiesParams{
		Browser: browser,
		Profile: profile,
	}

	resp, err := c.sendRequest(ctx, constants.MethodBrowserImportCookies, params)
	if err != nil {
		return nil, err
	}

	if resp.Error != nil {
		return nil, fmt.Errorf("importCookies error: %s", resp.Error.Message)
	}

	// Parse result
	data, err := json.Marshal(resp.Result)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal result: %w", err)
	}

	var result protocol.ImportCookiesResult
	if err := json.Unmarshal(data, &result); err != nil {
		return nil, fmt.Errorf("failed to unmarshal result: %w", err)
	}

	return &result, nil
}

// SetUserAgent sets the user agent
func (c *Client) SetUserAgent(ctx context.Context, userAgent string) error {
	params := protocol.SetUserAgentParams{
		UserAgent: userAgent,
	}

	resp, err := c.sendRequest(ctx, constants.MethodBrowserSetUserAgent, params)
	if err != nil {
		return err
	}

	if resp.Error != nil {
		return fmt.Errorf("setUserAgent error: %s", resp.Error.Message)
	}

	return nil
}

// UploadFile uploads files to a file input element
func (c *Client) UploadFile(ctx context.Context, index int, filePaths []string, tabID ...string) (*protocol.UploadFileResult, error) {
	params := protocol.UploadFileParams{
		Index:     index,
		FilePaths: filePaths,
	}
	if len(tabID) > 0 {
		params.TabID = &tabID[0]
	}

	resp, err := c.sendRequest(ctx, constants.MethodPageUploadFile, params)
	if err != nil {
		return nil, err
	}

	if resp.Error != nil {
		return nil, fmt.Errorf("uploadFile error: %s", resp.Error.Message)
	}

	// Parse result
	data, err := json.Marshal(resp.Result)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal result: %w", err)
	}

	var result protocol.UploadFileResult
	if err := json.Unmarshal(data, &result); err != nil {
		return nil, fmt.Errorf("failed to unmarshal result: %w", err)
	}

	return &result, nil
}

// GetText extracts text content from the page
func (c *Client) GetText(ctx context.Context, strategy string, tabID ...string) (*protocol.GetTextResult, error) {
	params := protocol.GetTextParams{
		Strategy: strategy,
	}
	if len(tabID) > 0 {
		params.TabID = &tabID[0]
	}

	resp, err := c.sendRequest(ctx, constants.MethodPageGetText, params)
	if err != nil {
		return nil, err
	}

	if resp.Error != nil {
		return nil, fmt.Errorf("getText error: %s", resp.Error.Message)
	}

	// Parse result
	data, err := json.Marshal(resp.Result)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal result: %w", err)
	}

	var result protocol.GetTextResult
	if err := json.Unmarshal(data, &result); err != nil {
		return nil, fmt.Errorf("failed to unmarshal result: %w", err)
	}

	return &result, nil
}

// Find searches for elements matching a query
func (c *Client) Find(ctx context.Context, query string, limit int, tabID ...string) (*protocol.FindResult, error) {
	params := protocol.FindParams{
		Query: query,
		Limit: limit,
	}
	if len(tabID) > 0 {
		params.TabID = &tabID[0]
	}

	resp, err := c.sendRequest(ctx, constants.MethodPageFind, params)
	if err != nil {
		return nil, err
	}

	if resp.Error != nil {
		return nil, fmt.Errorf("find error: %s", resp.Error.Message)
	}

	// Parse result
	data, err := json.Marshal(resp.Result)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal result: %w", err)
	}

	var result protocol.FindResult
	if err := json.Unmarshal(data, &result); err != nil {
		return nil, fmt.Errorf("failed to unmarshal result: %w", err)
	}

	return &result, nil
}

// sendRequest sends a request and waits for the response
func (c *Client) sendRequest(ctx context.Context, method string, params interface{}) (protocol.Response, error) {
	// Generate request ID
	reqID := fmt.Sprintf("%d", atomic.AddUint64(&c.requestID, 1))

	// Marshal params
	var paramsJSON json.RawMessage
	if params != nil {
		data, err := json.Marshal(params)
		if err != nil {
			return protocol.Response{}, fmt.Errorf("failed to marshal params: %w", err)
		}
		paramsJSON = data
	}

	// Create request
	req := protocol.Request{
		ID:     reqID,
		Method: method,
		Params: paramsJSON,
	}

	// Create response channel
	respChan := make(chan protocol.Response, 1)

	c.mu.Lock()
	if !c.connected {
		c.mu.Unlock()
		return protocol.Response{}, fmt.Errorf("client is not connected")
	}
	c.pending[reqID] = respChan
	c.mu.Unlock()

	// Clean up on exit
	defer func() {
		c.mu.Lock()
		delete(c.pending, reqID)
		c.mu.Unlock()
	}()

	// Send request
	c.mu.Lock()
	err := c.conn.WriteJSON(req)
	c.mu.Unlock()

	if err != nil {
		return protocol.Response{}, fmt.Errorf("failed to send request: %w", err)
	}

	// Wait for response or timeout
	select {
	case resp := <-respChan:
		return resp, nil
	case <-ctx.Done():
		return protocol.Response{}, fmt.Errorf("request timeout: %w", ctx.Err())
	case <-c.readDone:
		return protocol.Response{}, fmt.Errorf("connection closed")
	}
}

// readLoop reads responses from the WebSocket connection
func (c *Client) readLoop() {
	defer c.closeOnce.Do(func() {
		close(c.readDone)
		// Close all event listener channels
		c.mu.Lock()
		for _, ch := range c.eventListeners {
			close(ch)
		}
		c.eventListeners = nil
		c.mu.Unlock()
	})

	for {
		// Read raw message to determine if it's a response or event
		_, data, err := c.conn.ReadMessage()
		if err != nil {
			// Connection closed
			c.mu.Lock()
			c.connected = false
			// Notify all pending requests
			for _, ch := range c.pending {
				close(ch)
			}
			c.pending = make(map[string]chan protocol.Response)
			c.mu.Unlock()
			return
		}

		// Try to parse as response first (has ID field)
		var resp protocol.Response
		if err := json.Unmarshal(data, &resp); err == nil && resp.ID != "" {
			// It's a response
			c.mu.Lock()
			if ch, ok := c.pending[resp.ID]; ok {
				select {
				case ch <- resp:
				default:
					// Channel full or closed, ignore
				}
			}
			c.mu.Unlock()
			continue
		}

		// Try to parse as event (has method field, no ID)
		var event protocol.Event
		if err := json.Unmarshal(data, &event); err == nil && event.Method != "" {
			// It's an event - broadcast to all listeners
			c.mu.Lock()
			for _, ch := range c.eventListeners {
				select {
				case ch <- event:
				default:
					// Channel full, drop event
				}
			}
			c.mu.Unlock()
		}
	}
}

// SetDeadline sets the read and write deadlines on the underlying connection
func (c *Client) SetDeadline(t time.Time) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.conn.UnderlyingConn().SetDeadline(t)
}

// SetReadDeadline sets the read deadline on the underlying connection
func (c *Client) SetReadDeadline(t time.Time) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.conn.UnderlyingConn().SetReadDeadline(t)
}

// SetWriteDeadline sets the write deadline on the underlying connection
func (c *Client) SetWriteDeadline(t time.Time) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.conn.UnderlyingConn().SetWriteDeadline(t)
}

// PressKey presses keyboard keys or key combinations
func (c *Client) PressKey(ctx context.Context, keys string, tabID ...string) error {
	params := protocol.PressKeyParams{Keys: keys}
	if len(tabID) > 0 {
		params.TabID = &tabID[0]
	}

	resp, err := c.sendRequest(ctx, constants.MethodPagePressKey, params)
	if err != nil {
		return err
	}

	if resp.Error != nil {
		return fmt.Errorf("pressKey error: %s", resp.Error.Message)
	}

	return nil
}

// ScrollIntoView scrolls an element into the visible viewport
func (c *Client) ScrollIntoView(ctx context.Context, index *int, backendID *int64, tabID ...string) error {
	params := protocol.ScrollIntoViewParams{
		Index:     index,
		BackendID: backendID,
	}
	if len(tabID) > 0 {
		params.TabID = &tabID[0]
	}

	resp, err := c.sendRequest(ctx, constants.MethodPageScrollIntoView, params)
	if err != nil {
		return err
	}

	if resp.Error != nil {
		return fmt.Errorf("scrollIntoView error: %s", resp.Error.Message)
	}

	return nil
}

// ScrollIntoViewByIndex scrolls an element into view by index
func (c *Client) ScrollIntoViewByIndex(ctx context.Context, index int, tabID ...string) error {
	return c.ScrollIntoView(ctx, &index, nil, tabID...)
}

// ScrollIntoViewByBackendID scrolls an element into view by backend ID
func (c *Client) ScrollIntoViewByBackendID(ctx context.Context, backendID int64, tabID ...string) error {
	return c.ScrollIntoView(ctx, nil, &backendID, tabID...)
}

// GetCookies retrieves all cookies from the browser
func (c *Client) GetCookies(ctx context.Context, tabID ...string) (*protocol.GetCookiesResult, error) {
	var params *protocol.GetCookiesParams
	if len(tabID) > 0 {
		params = &protocol.GetCookiesParams{TabID: &tabID[0]}
	}

	resp, err := c.sendRequest(ctx, constants.MethodBrowserGetCookies, params)
	if err != nil {
		return nil, err
	}

	if resp.Error != nil {
		return nil, fmt.Errorf("getCookies error: %s", resp.Error.Message)
	}

	data, err := json.Marshal(resp.Result)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal result: %w", err)
	}

	var result protocol.GetCookiesResult
	if err := json.Unmarshal(data, &result); err != nil {
		return nil, fmt.Errorf("failed to unmarshal result: %w", err)
	}

	return &result, nil
}

// SetCookies sets cookies in the browser
func (c *Client) SetCookies(ctx context.Context, cookies []protocol.Cookie, tabID ...string) (*protocol.SetCookiesResult, error) {
	params := protocol.SetCookiesParams{Cookies: cookies}
	if len(tabID) > 0 {
		params.TabID = &tabID[0]
	}

	resp, err := c.sendRequest(ctx, constants.MethodBrowserSetCookies, params)
	if err != nil {
		return nil, err
	}

	if resp.Error != nil {
		return nil, fmt.Errorf("setCookies error: %s", resp.Error.Message)
	}

	data, err := json.Marshal(resp.Result)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal result: %w", err)
	}

	var result protocol.SetCookiesResult
	if err := json.Unmarshal(data, &result); err != nil {
		return nil, fmt.Errorf("failed to unmarshal result: %w", err)
	}

	return &result, nil
}

// ClearCookies clears all cookies from the browser
func (c *Client) ClearCookies(ctx context.Context, tabID ...string) (*protocol.ClearCookiesResult, error) {
	var params *protocol.ClearCookiesParams
	if len(tabID) > 0 {
		params = &protocol.ClearCookiesParams{TabID: &tabID[0]}
	}

	resp, err := c.sendRequest(ctx, constants.MethodBrowserClearCookies, params)
	if err != nil {
		return nil, err
	}

	if resp.Error != nil {
		return nil, fmt.Errorf("clearCookies error: %s", resp.Error.Message)
	}

	data, err := json.Marshal(resp.Result)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal result: %w", err)
	}

	var result protocol.ClearCookiesResult
	if err := json.Unmarshal(data, &result); err != nil {
		return nil, fmt.Errorf("failed to unmarshal result: %w", err)
	}

	return &result, nil
}

// SaveStorageState saves the current storage state (cookies + localStorage)
func (c *Client) SaveStorageState(ctx context.Context, tabID ...string) (*protocol.SaveStorageStateResult, error) {
	var params *protocol.SaveStorageStateParams
	if len(tabID) > 0 {
		params = &protocol.SaveStorageStateParams{TabID: &tabID[0]}
	}

	resp, err := c.sendRequest(ctx, constants.MethodBrowserSaveStorageState, params)
	if err != nil {
		return nil, err
	}

	if resp.Error != nil {
		return nil, fmt.Errorf("saveStorageState error: %s", resp.Error.Message)
	}

	data, err := json.Marshal(resp.Result)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal result: %w", err)
	}

	var result protocol.SaveStorageStateResult
	if err := json.Unmarshal(data, &result); err != nil {
		return nil, fmt.Errorf("failed to unmarshal result: %w", err)
	}

	return &result, nil
}

// LoadStorageState loads a storage state (cookies + localStorage)
func (c *Client) LoadStorageState(ctx context.Context, state protocol.StorageState, tabID ...string) (*protocol.LoadStorageStateResult, error) {
	params := protocol.LoadStorageStateParams{State: state}
	if len(tabID) > 0 {
		params.TabID = &tabID[0]
	}

	resp, err := c.sendRequest(ctx, constants.MethodBrowserLoadStorageState, params)
	if err != nil {
		return nil, err
	}

	if resp.Error != nil {
		return nil, fmt.Errorf("loadStorageState error: %s", resp.Error.Message)
	}

	data, err := json.Marshal(resp.Result)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal result: %w", err)
	}

	var result protocol.LoadStorageStateResult
	if err := json.Unmarshal(data, &result); err != nil {
		return nil, fmt.Errorf("failed to unmarshal result: %w", err)
	}

	return &result, nil
}

// SetLocalStorage sets localStorage items for the current page
func (c *Client) SetLocalStorage(ctx context.Context, items map[string]string, tabID ...string) (*protocol.SetLocalStorageResult, error) {
	params := protocol.SetLocalStorageParams{Items: items}
	if len(tabID) > 0 {
		params.TabID = &tabID[0]
	}

	resp, err := c.sendRequest(ctx, constants.MethodPageSetLocalStorage, params)
	if err != nil {
		return nil, err
	}

	if resp.Error != nil {
		return nil, fmt.Errorf("setLocalStorage error: %s", resp.Error.Message)
	}

	data, err := json.Marshal(resp.Result)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal result: %w", err)
	}

	var result protocol.SetLocalStorageResult
	if err := json.Unmarshal(data, &result); err != nil {
		return nil, fmt.Errorf("failed to unmarshal result: %w", err)
	}

	return &result, nil
}

// GetLocalStorage gets all localStorage items from the current page
func (c *Client) GetLocalStorage(ctx context.Context, tabID ...string) (*protocol.GetLocalStorageResult, error) {
	var params *protocol.GetLocalStorageParams
	if len(tabID) > 0 {
		params = &protocol.GetLocalStorageParams{TabID: &tabID[0]}
	}

	resp, err := c.sendRequest(ctx, constants.MethodPageGetLocalStorage, params)
	if err != nil {
		return nil, err
	}

	if resp.Error != nil {
		return nil, fmt.Errorf("getLocalStorage error: %s", resp.Error.Message)
	}

	data, err := json.Marshal(resp.Result)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal result: %w", err)
	}

	var result protocol.GetLocalStorageResult
	if err := json.Unmarshal(data, &result); err != nil {
		return nil, fmt.Errorf("failed to unmarshal result: %w", err)
	}

	return &result, nil
}

// SetDownloadBehavior configures download behavior for the browser
func (c *Client) SetDownloadBehavior(ctx context.Context, path string, accept bool, tabID ...string) (*protocol.SetDownloadBehaviorResult, error) {
	params := protocol.SetDownloadBehaviorParams{
		Path:   path,
		Accept: accept,
	}
	if len(tabID) > 0 {
		params.TabID = &tabID[0]
	}

	resp, err := c.sendRequest(ctx, constants.MethodBrowserSetDownloadBehavior, params)
	if err != nil {
		return nil, err
	}

	if resp.Error != nil {
		return nil, fmt.Errorf("setDownloadBehavior error: %s", resp.Error.Message)
	}

	data, err := json.Marshal(resp.Result)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal result: %w", err)
	}

	var result protocol.SetDownloadBehaviorResult
	if err := json.Unmarshal(data, &result); err != nil {
		return nil, fmt.Errorf("failed to unmarshal result: %w", err)
	}

	return &result, nil
}

// GetDownloads returns the list of downloads for the current tab
func (c *Client) GetDownloads(ctx context.Context, tabID ...string) (*protocol.GetDownloadsResult, error) {
	var params *protocol.GetDownloadsParams
	if len(tabID) > 0 {
		params = &protocol.GetDownloadsParams{TabID: &tabID[0]}
	}

	resp, err := c.sendRequest(ctx, constants.MethodPageGetDownloads, params)
	if err != nil {
		return nil, err
	}

	if resp.Error != nil {
		return nil, fmt.Errorf("getDownloads error: %s", resp.Error.Message)
	}

	data, err := json.Marshal(resp.Result)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal result: %w", err)
	}

	var result protocol.GetDownloadsResult
	if err := json.Unmarshal(data, &result); err != nil {
		return nil, fmt.Errorf("failed to unmarshal result: %w", err)
	}

	return &result, nil
}

// WaitForDownload blocks until a download completes or timeout occurs
func (c *Client) WaitForDownload(ctx context.Context, timeoutMs int, tabID ...string) (*protocol.WaitForDownloadResult, error) {
	params := protocol.WaitForDownloadParams{Timeout: timeoutMs}
	if len(tabID) > 0 {
		params.TabID = &tabID[0]
	}

	resp, err := c.sendRequest(ctx, constants.MethodPageWaitForDownload, params)
	if err != nil {
		return nil, err
	}

	if resp.Error != nil {
		return nil, fmt.Errorf("waitForDownload error: %s", resp.Error.Message)
	}

	data, err := json.Marshal(resp.Result)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal result: %w", err)
	}

	var result protocol.WaitForDownloadResult
	if err := json.Unmarshal(data, &result); err != nil {
		return nil, fmt.Errorf("failed to unmarshal result: %w", err)
	}

	return &result, nil
}

// GetClosedPopupMessages returns all closed popup messages
func (c *Client) GetClosedPopupMessages(ctx context.Context) (*protocol.GetClosedPopupMessagesResult, error) {
	resp, err := c.sendRequest(ctx, constants.MethodBrowserGetClosedPopupMessages, nil)
	if err != nil {
		return nil, err
	}

	if resp.Error != nil {
		return nil, fmt.Errorf("getClosedPopupMessages error: %s", resp.Error.Message)
	}

	data, err := json.Marshal(resp.Result)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal result: %w", err)
	}

	var result protocol.GetClosedPopupMessagesResult
	if err := json.Unmarshal(data, &result); err != nil {
		return nil, fmt.Errorf("failed to unmarshal result: %w", err)
	}

	return &result, nil
}

// LoadTaskCredentials loads credentials for a task from Convex
func (c *Client) LoadTaskCredentials(ctx context.Context, testCaseName, taskID string) (*protocol.LoadTaskCredentialsResult, error) {
	params := protocol.LoadTaskCredentialsParams{
		TestCaseName: testCaseName,
		TaskID:       taskID,
	}

	resp, err := c.sendRequest(ctx, constants.MethodBrowserLoadTaskCredentials, params)
	if err != nil {
		return nil, err
	}

	if resp.Error != nil {
		return nil, fmt.Errorf("loadTaskCredentials error: %s", resp.Error.Message)
	}

	data, err := json.Marshal(resp.Result)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal result: %w", err)
	}

	var result protocol.LoadTaskCredentialsResult
	if err := json.Unmarshal(data, &result); err != nil {
		return nil, fmt.Errorf("failed to unmarshal result: %w", err)
	}

	return &result, nil
}

// EvalJS evaluates JavaScript code on the page
func (c *Client) EvalJS(ctx context.Context, expression string, tabID ...string) (*protocol.EvalJSResult, error) {
	params := protocol.EvalJSParams{Expression: expression}
	if len(tabID) > 0 {
		params.TabID = &tabID[0]
	}

	resp, err := c.sendRequest(ctx, constants.MethodPageEvalJS, params)
	if err != nil {
		return nil, err
	}

	if resp.Error != nil {
		return nil, fmt.Errorf("evalJS error: %s", resp.Error.Message)
	}

	data, err := json.Marshal(resp.Result)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal result: %w", err)
	}

	var result protocol.EvalJSResult
	if err := json.Unmarshal(data, &result); err != nil {
		return nil, fmt.Errorf("failed to unmarshal result: %w", err)
	}

	return &result, nil
}
