package browser

import (
	"context"
	"encoding/json"
	"fmt"
	"reflect"
	"sync"
	"sync/atomic"

	"mix/internal/logging"

	browserprotocol "github.com/sarathmenon/browser-service/pkg/protocol"
)

// tabInfo stores CDP-level identifiers for a tab
type tabInfo struct {
	friendlyID   string // "tab-1", "tab-2" (friendly ID for LLM)
	targetID     string // CDP target ID from Target.createTarget
	cdpSessionID string // CDP session ID from Target.attachToTarget
}

// TunnelClientWrapper wraps the tunnel registry to provide the same interface as browserclient.Client
// This allows the browser tool to work transparently with both tunnel and service modes
type TunnelClientWrapper struct {
	tunnelRegistry interface{} // *httppkg.TunnelRegistry (stored as interface to avoid import cycles)
	sessionID      string       // Mix session ID (user session)
	requestID      int

	// Tab management
	tabsMu      sync.RWMutex
	tabs        map[string]*tabInfo // friendlyTabId → tab info
	tabCounter  uint64
	activeTabID string
}

// NewTunnelClientWrapper creates a new tunnel client wrapper
func NewTunnelClientWrapper(tunnelRegistry interface{}, sessionID string) *TunnelClientWrapper {
	if tunnelRegistry == nil {
		panic("tunnelRegistry cannot be nil - ensure TunnelRegistry is initialized before creating browser tool")
	}
	return &TunnelClientWrapper{
		tunnelRegistry: tunnelRegistry,
		sessionID:      sessionID,
		requestID:      1,
		tabs:           make(map[string]*tabInfo),
	}
}

// getNextRequestID returns the next request ID and increments the counter
func (t *TunnelClientWrapper) getNextRequestID() int {
	id := t.requestID
	t.requestID++
	return id
}

// getTabCDPSessionID returns the CDP session ID for a given tab
// If tabID is not provided, returns the active tab's CDP session ID
func (t *TunnelClientWrapper) getTabCDPSessionID(tabID ...string) (string, error) {
	t.tabsMu.RLock()
	defer t.tabsMu.RUnlock()

	var targetTabID string
	if len(tabID) > 0 && tabID[0] != "" {
		targetTabID = tabID[0]
	} else {
		// Use active tab
		targetTabID = t.activeTabID
	}

	if targetTabID == "" {
		return "", fmt.Errorf("no active tab and no tabID provided")
	}

	tab, exists := t.tabs[targetTabID]
	if !exists {
		return "", fmt.Errorf("tab not found: %s", targetTabID)
	}

	return tab.cdpSessionID, nil
}

// generateFriendlyTabID generates a new friendly tab ID
func (t *TunnelClientWrapper) generateFriendlyTabID() string {
	counter := atomic.AddUint64(&t.tabCounter, 1)
	return fmt.Sprintf("tab-%d", counter)
}

// sendCommand sends a CDP command through the tunnel and returns the result
// Uses reflection to call SendCommandToTunnel method without importing internal/http
// cdpSessionID is optional - if provided, it will be included in the CDP request
func (t *TunnelClientWrapper) sendCommand(_ context.Context, method string, params interface{}, cdpSessionID ...string) (interface{}, error) {
	requestID := t.getNextRequestID()

	// Create command map
	command := map[string]interface{}{
		"id":     requestID,
		"method": method,
	}
	if params != nil {
		command["params"] = params
	}
	// Add CDP session ID if provided (required for tab-specific commands)
	if len(cdpSessionID) > 0 && cdpSessionID[0] != "" {
		command["sessionId"] = cdpSessionID[0]
	}

	// Convert command map to JSON and back to match the CDPRequest type in internal/http
	commandJSON, err := json.Marshal(command)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal command: %w", err)
	}

	// Debug log all outgoing CDP commands
	logging.Debug("Sending CDP command via tunnel", "method", method, "sessionId", command["sessionId"], "requestId", requestID, "command", string(commandJSON))

	// Use reflection to call SendCommandToTunnel
	registryValue := reflect.ValueOf(t.tunnelRegistry)
	sendMethod := registryValue.MethodByName("SendCommandToTunnel")
	if !sendMethod.IsValid() {
		return nil, fmt.Errorf("tunnel registry missing SendCommandToTunnel method")
	}

	// Create CDPRequest using reflection
	// We need to create an instance of the CDPRequest type from internal/http
	cdpRequestType := sendMethod.Type().In(1) // Second parameter type
	cdpRequestValue := reflect.New(cdpRequestType).Elem()

	// Unmarshal JSON into the CDPRequest instance
	if err := json.Unmarshal(commandJSON, cdpRequestValue.Addr().Interface()); err != nil {
		return nil, fmt.Errorf("failed to create CDPRequest: %w", err)
	}

	// Call SendCommandToTunnel(sessionID, command)
	results := sendMethod.Call([]reflect.Value{
		reflect.ValueOf(t.sessionID),
		cdpRequestValue,
	})

	// Check error return value (second return)
	if !results[1].IsNil() {
		return nil, results[1].Interface().(error)
	}

	// Get response (first return value is *CDPResponse)
	responsePtr := results[0].Interface()
	if responsePtr == nil {
		return nil, fmt.Errorf("nil response from tunnel")
	}

	// Convert response to map for easier access
	responseJSON, err := json.Marshal(responsePtr)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal response: %w", err)
	}

	var responseMap map[string]interface{}
	if err := json.Unmarshal(responseJSON, &responseMap); err != nil {
		return nil, fmt.Errorf("failed to unmarshal response: %w", err)
	}

	// Check for error in response
	if errVal, hasErr := responseMap["error"]; hasErr && errVal != nil {
		if errMap, ok := errVal.(map[string]interface{}); ok {
			if msg, ok := errMap["message"].(string); ok {
				return nil, fmt.Errorf("CDP error for %s: %s", method, msg)
			}
		}
		return nil, fmt.Errorf("CDP error for %s", method)
	}

	// Return result
	return responseMap["result"], nil
}

// Navigate navigates to a URL
func (t *TunnelClientWrapper) Navigate(ctx context.Context, url string, tabID ...string) (*browserprotocol.NavigateResult, error) {
	// Get CDP session ID for the target tab
	cdpSessionID, err := t.getTabCDPSessionID(tabID...)
	if err != nil {
		return nil, fmt.Errorf("failed to get tab CDP session: %w", err)
	}

	params := map[string]interface{}{
		"url": url,
	}

	result, err := t.sendCommand(ctx, "Page.navigate", params, cdpSessionID)
	if err != nil {
		return nil, err
	}

	// Convert result to NavigateResult
	resultMap, ok := result.(map[string]interface{})
	if !ok {
		return &browserprotocol.NavigateResult{}, nil
	}

	frameID, _ := resultMap["frameId"].(string)
	loaderID, _ := resultMap["loaderId"].(string)

	return &browserprotocol.NavigateResult{
		FrameID:  frameID,
		LoaderID: loaderID,
	}, nil
}

// GoBack navigates back in history
func (t *TunnelClientWrapper) GoBack(ctx context.Context, tabID ...string) (string, error) {
	cdpSessionID, err := t.getTabCDPSessionID(tabID...)
	if err != nil {
		return "", fmt.Errorf("failed to get tab CDP session: %w", err)
	}

	_, err = t.sendCommand(ctx, "Page.navigateToHistoryEntry", map[string]interface{}{
		"entryId": -1, // Go back
	}, cdpSessionID)
	if err != nil {
		return "", err
	}

	// TODO: Return actual URL from navigation
	return "", nil
}

// GoForward navigates forward in history
func (t *TunnelClientWrapper) GoForward(ctx context.Context, tabID ...string) (string, error) {
	cdpSessionID, err := t.getTabCDPSessionID(tabID...)
	if err != nil {
		return "", fmt.Errorf("failed to get tab CDP session: %w", err)
	}

	_, err = t.sendCommand(ctx, "Page.navigateToHistoryEntry", map[string]interface{}{
		"entryId": 1, // Go forward
	}, cdpSessionID)
	if err != nil {
		return "", err
	}

	// TODO: Return actual URL from navigation
	return "", nil
}

// Screenshot captures a screenshot
func (t *TunnelClientWrapper) Screenshot(ctx context.Context, params browserprotocol.ScreenshotParams) (*browserprotocol.ScreenshotResult, error) {
	// TODO: Implement screenshot via tunnel
	return nil, fmt.Errorf("screenshot not yet implemented for tunnel mode")
}

// ReadPage reads the accessibility tree
func (t *TunnelClientWrapper) ReadPage(ctx context.Context, interactiveOnly bool, tabID ...string) (*browserprotocol.ReadPageResult, error) {
	// TODO: Implement read_page via tunnel
	return nil, fmt.Errorf("read_page not yet implemented for tunnel mode")
}

// GetText extracts text from the page
func (t *TunnelClientWrapper) GetText(ctx context.Context, strategy string, tabID ...string) (*browserprotocol.GetTextResult, error) {
	// TODO: Implement get_text via tunnel
	return nil, fmt.Errorf("get_text not yet implemented for tunnel mode")
}

// Find searches for elements
func (t *TunnelClientWrapper) Find(ctx context.Context, query string, limit int, tabID ...string) (*browserprotocol.FindResult, error) {
	// TODO: Implement find via tunnel
	return nil, fmt.Errorf("find not yet implemented for tunnel mode")
}

// Click clicks an element by index
func (t *TunnelClientWrapper) Click(ctx context.Context, index int, tabID ...string) error {
	// TODO: Implement click via tunnel
	return fmt.Errorf("click not yet implemented for tunnel mode")
}

// ClickByBackendID clicks an element by backend ID
func (t *TunnelClientWrapper) ClickByBackendID(ctx context.Context, backendID int64, tabID ...string) error {
	cdpSessionID, err := t.getTabCDPSessionID(tabID...)
	if err != nil {
		return fmt.Errorf("failed to get tab CDP session: %w", err)
	}

	params := map[string]interface{}{
		"backendNodeId": backendID,
	}

	_, err = t.sendCommand(ctx, "DOM.click", params, cdpSessionID)
	return err
}

// RightClick right-clicks an element by index
func (t *TunnelClientWrapper) RightClick(ctx context.Context, index int, tabID ...string) error {
	// TODO: Implement right_click via tunnel
	return fmt.Errorf("right_click not yet implemented for tunnel mode")
}

// RightClickByBackendID right-clicks an element by backend ID
func (t *TunnelClientWrapper) RightClickByBackendID(ctx context.Context, backendID int64, tabID ...string) error {
	// TODO: Implement right_click by backend ID via tunnel
	return fmt.Errorf("right_click by backend ID not yet implemented for tunnel mode")
}

// DoubleClick double-clicks an element by index
func (t *TunnelClientWrapper) DoubleClick(ctx context.Context, index int, tabID ...string) error {
	// TODO: Implement double_click via tunnel
	return fmt.Errorf("double_click not yet implemented for tunnel mode")
}

// DoubleClickByBackendID double-clicks an element by backend ID
func (t *TunnelClientWrapper) DoubleClickByBackendID(ctx context.Context, backendID int64, tabID ...string) error {
	// TODO: Implement double_click by backend ID via tunnel
	return fmt.Errorf("double_click by backend ID not yet implemented for tunnel mode")
}

// TripleClick triple-clicks an element by index
func (t *TunnelClientWrapper) TripleClick(ctx context.Context, index int, tabID ...string) error {
	// TODO: Implement triple_click via tunnel
	return fmt.Errorf("triple_click not yet implemented for tunnel mode")
}

// TripleClickByBackendID triple-clicks an element by backend ID
func (t *TunnelClientWrapper) TripleClickByBackendID(ctx context.Context, backendID int64, tabID ...string) error {
	// TODO: Implement triple_click by backend ID via tunnel
	return fmt.Errorf("triple_click by backend ID not yet implemented for tunnel mode")
}

// Drag performs a drag operation
func (t *TunnelClientWrapper) Drag(ctx context.Context, fromIndex, toIndex *int, fromX, fromY, toX, toY *float64, duration *int, tabID ...string) error {
	// TODO: Implement drag via tunnel
	return fmt.Errorf("drag not yet implemented for tunnel mode")
}

// Type types text into an element
func (t *TunnelClientWrapper) Type(ctx context.Context, index int, text string, tabID ...string) error {
	// TODO: Implement type via tunnel
	return fmt.Errorf("type not yet implemented for tunnel mode")
}

// PressKey presses keyboard keys
func (t *TunnelClientWrapper) PressKey(ctx context.Context, keys string, tabID ...string) error {
	// TODO: Implement press_key via tunnel
	return fmt.Errorf("press_key not yet implemented for tunnel mode")
}

// FormInput sets form input value
func (t *TunnelClientWrapper) FormInput(ctx context.Context, index int, value string, tabID ...string) error {
	// TODO: Implement form_input via tunnel
	return fmt.Errorf("form_input not yet implemented for tunnel mode")
}

// Scroll scrolls the page
func (t *TunnelClientWrapper) Scroll(ctx context.Context, direction string, amount int, tabID ...string) error {
	// TODO: Implement scroll via tunnel
	return fmt.Errorf("scroll not yet implemented for tunnel mode")
}

// ScrollIntoView scrolls an element into view
func (t *TunnelClientWrapper) ScrollIntoView(ctx context.Context, index *int, backendID *int64, tabID ...string) error {
	// TODO: Implement scroll_into_view via tunnel
	return fmt.Errorf("scroll_into_view not yet implemented for tunnel mode")
}

// ScrollIntoViewByIndex scrolls an element into view by index
func (t *TunnelClientWrapper) ScrollIntoViewByIndex(ctx context.Context, index int, tabID ...string) error {
	return t.ScrollIntoView(ctx, &index, nil, tabID...)
}

// ScrollIntoViewByBackendID scrolls an element into view by backend ID
func (t *TunnelClientWrapper) ScrollIntoViewByBackendID(ctx context.Context, backendID int64, tabID ...string) error {
	return t.ScrollIntoView(ctx, nil, &backendID, tabID...)
}

// UploadFile uploads files to a file input element
func (t *TunnelClientWrapper) UploadFile(ctx context.Context, index int, filePaths []string, tabID ...string) (*browserprotocol.UploadFileResult, error) {
	// TODO: Implement upload_file via tunnel
	return nil, fmt.Errorf("upload_file not yet implemented for tunnel mode")
}

// CreateTab creates a new tab
func (t *TunnelClientWrapper) CreateTab(ctx context.Context, url ...string) (*browserprotocol.TabInfo, error) {
	// Step 1: Create target
	targetURL := "about:blank"
	if len(url) > 0 && url[0] != "" {
		targetURL = url[0]
	}

	createParams := map[string]interface{}{
		"url": targetURL,
	}

	// Send Target.createTarget (no CDP session ID needed for Target commands)
	createResult, err := t.sendCommand(ctx, "Target.createTarget", createParams)
	if err != nil {
		return nil, fmt.Errorf("failed to create target: %w", err)
	}

	// Extract targetId from result
	resultMap, ok := createResult.(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("unexpected create target result type: %T", createResult)
	}

	targetID, ok := resultMap["targetId"].(string)
	if !ok {
		return nil, fmt.Errorf("targetId not found in create target result")
	}

	// Step 2: Attach to target to get CDP session ID
	attachParams := map[string]interface{}{
		"targetId": targetID,
		"flatten":  true, // Flatten session for easier command routing
	}

	attachResult, err := t.sendCommand(ctx, "Target.attachToTarget", attachParams)
	if err != nil {
		return nil, fmt.Errorf("failed to attach to target: %w", err)
	}

	// Extract sessionId from result
	attachMap, ok := attachResult.(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("unexpected attach target result type: %T", attachResult)
	}

	cdpSessionID, ok := attachMap["sessionId"].(string)
	if !ok {
		return nil, fmt.Errorf("sessionId not found in attach target result")
	}

	// Step 3: Generate friendly tab ID and store mapping
	friendlyTabID := t.generateFriendlyTabID()

	tab := &tabInfo{
		friendlyID:   friendlyTabID,
		targetID:     targetID,
		cdpSessionID: cdpSessionID,
	}

	t.tabsMu.Lock()
	t.tabs[friendlyTabID] = tab
	// Set as active if it's the first tab
	if t.activeTabID == "" {
		t.activeTabID = friendlyTabID
	}
	t.tabsMu.Unlock()

	// Step 4: Get tab info (URL and title)
	// For now, return the URL we navigated to
	// TODO: Query actual page info from browser
	return &browserprotocol.TabInfo{
		ID:       friendlyTabID,
		URL:      targetURL,
		Title:    "",
		IsActive: t.activeTabID == friendlyTabID,
	}, nil
}

// ListTabs lists all tabs
func (t *TunnelClientWrapper) ListTabs(ctx context.Context) (*browserprotocol.TabListResult, error) {
	t.tabsMu.RLock()
	defer t.tabsMu.RUnlock()

	tabs := make([]browserprotocol.TabInfo, 0, len(t.tabs))
	for _, tab := range t.tabs {
		// TODO: Query actual URL and title from browser
		tabs = append(tabs, browserprotocol.TabInfo{
			ID:       tab.friendlyID,
			URL:      "",
			Title:    "",
			IsActive: tab.friendlyID == t.activeTabID,
		})
	}

	return &browserprotocol.TabListResult{
		Tabs:        tabs,
		ActiveTabID: t.activeTabID,
	}, nil
}

// SwitchTab switches to a different tab
func (t *TunnelClientWrapper) SwitchTab(ctx context.Context, tabID string) error {
	t.tabsMu.Lock()
	defer t.tabsMu.Unlock()

	if _, exists := t.tabs[tabID]; !exists {
		return fmt.Errorf("tab not found: %s", tabID)
	}

	t.activeTabID = tabID
	return nil
}

// CloseTab closes a tab
func (t *TunnelClientWrapper) CloseTab(ctx context.Context, tabID string) error {
	t.tabsMu.Lock()
	tab, exists := t.tabs[tabID]
	if !exists {
		t.tabsMu.Unlock()
		return fmt.Errorf("tab not found: %s", tabID)
	}

	targetID := tab.targetID
	delete(t.tabs, tabID)

	// If closing active tab, switch to another tab
	if t.activeTabID == tabID {
		t.activeTabID = ""
		// Set first remaining tab as active
		for id := range t.tabs {
			t.activeTabID = id
			break
		}
	}
	t.tabsMu.Unlock()

	// Send Target.closeTarget command
	closeParams := map[string]interface{}{
		"targetId": targetID,
	}

	_, err := t.sendCommand(ctx, "Target.closeTarget", closeParams)
	if err != nil {
		return fmt.Errorf("failed to close target: %w", err)
	}

	return nil
}

// Wait pauses execution
func (t *TunnelClientWrapper) Wait(ctx context.Context, duration int, tabID ...string) error {
	// TODO: Implement wait via tunnel (or just use time.Sleep?)
	return fmt.Errorf("wait not yet implemented for tunnel mode")
}

// Close closes the tunnel connection
func (t *TunnelClientWrapper) Close() error {
	// Tunnel connections are managed by the registry, so nothing to do here
	return nil
}
