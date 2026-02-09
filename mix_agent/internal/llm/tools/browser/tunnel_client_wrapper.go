package browser

import (
	"context"
	"encoding/json"
	"fmt"
	"reflect"
	"sort"
	"strings"
	"sync"
	"sync/atomic"

	"mix/internal/llm/tools/browser/cdp"
	"mix/internal/logging"

	browserprotocol "github.com/sarathmenon/browser-service/pkg/protocol"
)

// tabInfo stores CDP-level identifiers for a tab
type tabInfo struct {
	friendlyID   string // "tab-1", "tab-2" (friendly ID for LLM)
	targetID     string // CDP target ID from Target.createTarget
	cdpSessionID string // CDP session ID from Target.attachToTarget
}

// elementInfo stores element data for click/type operations
type elementInfo struct {
	Role      string
	Name      string
	Bounds    browserprotocol.BoundingBox
	BackendID int64
	FrameID   string
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

	// Element cache (per tab)
	cacheMu      sync.RWMutex
	elementCache map[string][]elementInfo // tabID → elements
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
		elementCache:   make(map[string][]elementInfo),
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

	// Create typed command
	command := cdp.Command{
		ID:     requestID,
		Method: method,
		Params: params,
	}
	// Add CDP session ID if provided (required for tab-specific commands)
	if len(cdpSessionID) > 0 && cdpSessionID[0] != "" {
		command.SessionID = cdpSessionID[0]
	}

	// Convert command to JSON to match the CDPRequest type in internal/http
	commandJSON, err := json.Marshal(command)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal command: %w", err)
	}

	// Debug log all outgoing CDP commands
	logging.Debug("Sending CDP command via tunnel", "method", method, "sessionId", command.SessionID, "requestId", requestID, "command", string(commandJSON))

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

	// Convert response to typed CDP response
	responseJSON, err := json.Marshal(responsePtr)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal response: %w", err)
	}

	var response cdp.Response
	if err := json.Unmarshal(responseJSON, &response); err != nil {
		return nil, fmt.Errorf("failed to unmarshal response: %w", err)
	}

	// Check for error in response
	if response.Error != nil {
		return nil, fmt.Errorf("CDP error for %s: %s", method, response.Error.Message)
	}

	// Return result
	return response.Result, nil
}

// Navigate navigates to a URL
func (t *TunnelClientWrapper) Navigate(ctx context.Context, url string, tabID ...string) (*browserprotocol.NavigateResult, error) {
	// Get CDP session ID for the target tab
	cdpSessionID, err := t.getTabCDPSessionID(tabID...)
	if err != nil {
		return nil, fmt.Errorf("failed to get tab CDP session: %w", err)
	}

	params := cdp.PageNavigateParams{
		URL: url,
	}

	result, err := t.sendCommand(ctx, "Page.navigate", params, cdpSessionID)
	if err != nil {
		return nil, err
	}

	// Convert result to typed response
	resultJSON, err := json.Marshal(result)
	if err != nil {
		return &browserprotocol.NavigateResult{}, err
	}

	var navResult cdp.PageNavigateResult
	if err := json.Unmarshal(resultJSON, &navResult); err != nil {
		return &browserprotocol.NavigateResult{}, err
	}

	return &browserprotocol.NavigateResult{
		FrameID:  navResult.FrameID,
		LoaderID: navResult.LoaderID,
	}, nil
}

// GoBack navigates back in history
func (t *TunnelClientWrapper) GoBack(ctx context.Context, tabID ...string) (string, error) {
	cdpSessionID, err := t.getTabCDPSessionID(tabID...)
	if err != nil {
		return "", fmt.Errorf("failed to get tab CDP session: %w", err)
	}

	params := cdp.PageNavigateToHistoryParams{
		EntryID: -1, // Go back
	}

	_, err = t.sendCommand(ctx, "Page.navigateToHistoryEntry", params, cdpSessionID)
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

	params := cdp.PageNavigateToHistoryParams{
		EntryID: 1, // Go forward
	}

	_, err = t.sendCommand(ctx, "Page.navigateToHistoryEntry", params, cdpSessionID)
	if err != nil {
		return "", err
	}

	// TODO: Return actual URL from navigation
	return "", nil
}

// Screenshot captures a screenshot
func (t *TunnelClientWrapper) Screenshot(ctx context.Context, params browserprotocol.ScreenshotParams) (*browserprotocol.ScreenshotResult, error) {
	// Convert *string to ...string for getTabCDPSessionID
	var tabIDSlice []string
	if params.TabID != nil {
		tabIDSlice = []string{*params.TabID}
	}
	cdpSessionID, err := t.getTabCDPSessionID(tabIDSlice...)
	if err != nil {
		return nil, fmt.Errorf("failed to get tab CDP session: %w", err)
	}

	// Capture screenshot using Page.captureScreenshot
	screenshotParams := cdp.PageCaptureScreenshotParams{
		Format:      "png",
		FromSurface: true,
	}

	if params.Quality > 0 {
		screenshotParams.Quality = params.Quality
	}

	if params.Format == "jpeg" {
		screenshotParams.Format = "jpeg"
	}

	result, err := t.sendCommand(ctx, "Page.captureScreenshot", screenshotParams, cdpSessionID)
	if err != nil {
		return nil, fmt.Errorf("screenshot failed: %w", err)
	}

	// Convert result to typed response
	resultJSON, err := json.Marshal(result)
	if err != nil {
		return nil, fmt.Errorf("unexpected screenshot result type: %T", result)
	}

	var screenshotResult cdp.PageCaptureScreenshotResult
	if err := json.Unmarshal(resultJSON, &screenshotResult); err != nil {
		return nil, fmt.Errorf("failed to unmarshal screenshot result: %w", err)
	}

	format := params.Format
	if format == "" {
		format = "png"
	}

	result2 := &browserprotocol.ScreenshotResult{
		Data:   screenshotResult.Data,
		Format: format,
	}

	// Raw mode: return accessibility tree and viewport
	if params.Raw {
		// Get full accessibility tree
		axResult, err := t.sendCommand(ctx, "Accessibility.getFullAXTree", nil, cdpSessionID)
		if err != nil {
			return nil, fmt.Errorf("failed to get accessibility tree: %w", err)
		}

		// Convert to typed result
		axJSON, err := json.Marshal(axResult)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal AX result: %w", err)
		}

		var axTreeResult cdp.AccessibilityGetFullAXTreeResult
		if err := json.Unmarshal(axJSON, &axTreeResult); err != nil {
			return nil, fmt.Errorf("failed to unmarshal AX result: %w", err)
		}

		// Convert nodes to raw accessibility nodes
		rawNodes := t.convertAXNodesToRaw(axTreeResult.Nodes)

		// Get viewport bounds
		viewport, err := t.getViewportBounds(ctx, cdpSessionID)
		if err != nil {
			return nil, fmt.Errorf("failed to get viewport: %w", err)
		}

		result2.RawNodes = rawNodes
		result2.RawViewport = viewport
	}

	return result2, nil
}

// ReadPage reads the accessibility tree
func (t *TunnelClientWrapper) ReadPage(ctx context.Context, interactiveOnly bool, tabID ...string) (*browserprotocol.ReadPageResult, error) {
	cdpSessionID, err := t.getTabCDPSessionID(tabID...)
	if err != nil {
		return nil, fmt.Errorf("failed to get tab CDP session: %w", err)
	}

	// Determine which tab we're reading
	targetTabID := t.activeTabID
	if len(tabID) > 0 && tabID[0] != "" {
		targetTabID = tabID[0]
	}

	// Get viewport bounds
	viewport, err := t.getViewportBounds(ctx, cdpSessionID)
	if err != nil {
		return nil, fmt.Errorf("failed to get viewport: %w", err)
	}

	// Get full accessibility tree
	axResult, err := t.sendCommand(ctx, "Accessibility.getFullAXTree", nil, cdpSessionID)
	if err != nil {
		return nil, fmt.Errorf("failed to get accessibility tree: %w", err)
	}

	// Convert to typed result
	axJSON, err := json.Marshal(axResult)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal AX result: %w", err)
	}

	var axTreeResult cdp.AccessibilityGetFullAXTreeResult
	if err := json.Unmarshal(axJSON, &axTreeResult); err != nil {
		return nil, fmt.Errorf("failed to unmarshal AX result: %w", err)
	}

	// Convert nodes
	rawNodes := t.convertAXNodesToRaw(axTreeResult.Nodes)

	// Filter nodes based on interactiveOnly and viewport visibility
	filteredNodes := make([]browserprotocol.RawAccessibilityNode, 0)
	elements := make([]elementInfo, 0)

	for _, node := range rawNodes {
		// Skip nodes with zero size
		if node.Bounds.Width <= 0 || node.Bounds.Height <= 0 {
			continue
		}

		// Check viewport visibility
		if !isInViewport(&node.Bounds, viewport) {
			continue
		}

		// Filter interactive only if requested
		if interactiveOnly && !isInteractiveRole(node.Role) {
			continue
		}

		filteredNodes = append(filteredNodes, node)

		// Add to element cache for click/type operations
		elements = append(elements, elementInfo{
			Role:      node.Role,
			Name:      node.Name,
			Bounds:    node.Bounds,
			BackendID: node.BackendID,
			FrameID:   node.FrameID,
		})
	}

	// Update element cache
	t.cacheMu.Lock()
	t.elementCache[targetTabID] = elements
	t.cacheMu.Unlock()

	logging.Info("ReadPage completed", "tabID", targetTabID, "totalNodes", len(rawNodes), "filteredNodes", len(filteredNodes), "cached", len(elements))

	return &browserprotocol.ReadPageResult{
		Elements: filteredNodes,
		Viewport: browserprotocol.BoundingBox{
			X:      viewport.X,
			Y:      viewport.Y,
			Width:  viewport.Width,
			Height: viewport.Height,
		},
	}, nil
}

// GetText extracts text from the page
func (t *TunnelClientWrapper) GetText(ctx context.Context, strategy string, tabID ...string) (*browserprotocol.GetTextResult, error) {
	cdpSessionID, err := t.getTabCDPSessionID(tabID...)
	if err != nil {
		return nil, fmt.Errorf("failed to get tab CDP session: %w", err)
	}

	// Default to auto strategy
	if strategy == "" {
		strategy = "auto"
	}

	// Build JavaScript based on strategy
	var js string
	switch strategy {
	case "auto":
		js = `(function() {
			let elem = document.querySelector('article');
			if (elem) return { text: elem.innerText, source: 'article' };

			elem = document.querySelector('main, [role="main"]');
			if (elem) return { text: elem.innerText, source: 'main' };

			return { text: document.body.innerText, source: 'body' };
		})()`
	case "article":
		js = `(function() {
			const elem = document.querySelector('article');
			if (!elem) return { text: '', source: 'article' };
			return { text: elem.innerText, source: 'article' };
		})()`
	case "main":
		js = `(function() {
			const elem = document.querySelector('main, [role="main"]');
			if (!elem) return { text: '', source: 'main' };
			return { text: elem.innerText, source: 'main' };
		})()`
	case "body":
		js = `(function() {
			return { text: document.body.innerText, source: 'body' };
		})()`
	default:
		return nil, fmt.Errorf("invalid strategy: %s", strategy)
	}

	// Execute JavaScript using Runtime.evaluate
	evaluateParams := cdp.RuntimeEvaluateParams{
		Expression:    js,
		ReturnByValue: true,
	}

	result, err := t.sendCommand(ctx, "Runtime.evaluate", evaluateParams, cdpSessionID)
	if err != nil {
		return nil, fmt.Errorf("failed to evaluate JavaScript: %w", err)
	}

	// Convert to typed result
	resultJSON, err := json.Marshal(result)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal result: %w", err)
	}

	var evalResult cdp.RuntimeEvaluateResult
	if err := json.Unmarshal(resultJSON, &evalResult); err != nil {
		return nil, fmt.Errorf("failed to unmarshal result: %w", err)
	}

	// Check for exception
	if evalResult.ExceptionDetails != nil {
		return nil, fmt.Errorf("JavaScript execution error")
	}

	// Extract result value - the value field contains our {text, source} object
	valueMap, ok := evalResult.Result.Value.(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("unexpected value type: %T", evalResult.Result.Value)
	}

	text, _ := valueMap["text"].(string)
	source, _ := valueMap["source"].(string)

	return &browserprotocol.GetTextResult{
		Text:   text,
		Source: source,
	}, nil
}

// Find searches for elements
func (t *TunnelClientWrapper) Find(ctx context.Context, query string, limit int, tabID ...string) (*browserprotocol.FindResult, error) {
	cdpSessionID, err := t.getTabCDPSessionID(tabID...)
	if err != nil {
		return nil, fmt.Errorf("failed to get tab CDP session: %w", err)
	}

	if query == "" {
		return nil, fmt.Errorf("query cannot be empty")
	}

	// Default limit
	if limit <= 0 {
		limit = 10
	}

	// Get full accessibility tree
	axResult, err := t.sendCommand(ctx, "Accessibility.getFullAXTree", nil, cdpSessionID)
	if err != nil {
		return nil, fmt.Errorf("failed to get accessibility tree: %w", err)
	}

	// Convert to typed result
	axJSON, err := json.Marshal(axResult)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal AX result: %w", err)
	}

	var axTreeResult cdp.AccessibilityGetFullAXTreeResult
	if err := json.Unmarshal(axJSON, &axTreeResult); err != nil {
		return nil, fmt.Errorf("failed to unmarshal AX result: %w", err)
	}

	// Convert nodes
	rawNodes := t.convertAXNodesToRaw(axTreeResult.Nodes)

	// Parse query for role and keywords
	queryLower := strings.ToLower(query)
	words := strings.Fields(queryLower)

	var targetRole string
	keywords := make([]string, 0)

	// Check if first word is a role
	if len(words) > 0 {
		knownRoles := map[string]bool{
			"button": true, "link": true, "input": true, "textbox": true,
			"checkbox": true, "radio": true, "menu": true, "tab": true,
		}

		if knownRoles[words[0]] {
			targetRole = words[0]
			keywords = words[1:]
		} else {
			keywords = words
		}
	}

	// Find matching elements
	type match struct {
		element browserprotocol.RawAccessibilityNode
		score   int
	}

	matches := make([]match, 0)

	for _, node := range rawNodes {
		// Skip zero-size elements
		if node.Bounds.Width <= 0 || node.Bounds.Height <= 0 {
			continue
		}

		// Check role match if specified
		if targetRole != "" && !strings.EqualFold(node.Role, targetRole) {
			continue
		}

		// Calculate match score based on keywords
		score := 0
		nameLower := strings.ToLower(node.Name)

		for _, keyword := range keywords {
			if strings.Contains(nameLower, keyword) {
				score++
			}
		}

		// If no keywords or at least one match
		if len(keywords) == 0 || score > 0 {
			matches = append(matches, match{
				element: node,
				score:   score,
			})
		}
	}

	// Sort by score (descending)
	sort.Slice(matches, func(i, j int) bool {
		return matches[i].score > matches[j].score
	})

	// Apply limit
	if len(matches) > limit {
		matches = matches[:limit]
	}

	// Extract elements
	elements := make([]browserprotocol.RawAccessibilityNode, len(matches))
	for i, m := range matches {
		elements[i] = m.element
	}

	return &browserprotocol.FindResult{
		Elements:  elements,
		Total:     len(elements),
		Truncated: len(matches) >= limit,
	}, nil
}

// Click clicks an element by index
func (t *TunnelClientWrapper) Click(ctx context.Context, index int, tabID ...string) error {
	// Get element from cache
	targetTabID := t.activeTabID
	if len(tabID) > 0 && tabID[0] != "" {
		targetTabID = tabID[0]
	}

	t.cacheMu.RLock()
	elements, exists := t.elementCache[targetTabID]
	t.cacheMu.RUnlock()

	if !exists || len(elements) == 0 {
		return fmt.Errorf("no elements cached for tab %s, call read_page first", targetTabID)
	}

	if index < 0 || index >= len(elements) {
		return fmt.Errorf("index %d out of range (0-%d)", index, len(elements)-1)
	}

	// Delegate to ClickByBackendID
	return t.ClickByBackendID(ctx, elements[index].BackendID, tabID...)
}

// ClickByBackendID clicks an element by backend ID
func (t *TunnelClientWrapper) ClickByBackendID(ctx context.Context, backendID int64, tabID ...string) error {
	cdpSessionID, err := t.getTabCDPSessionID(tabID...)
	if err != nil {
		return fmt.Errorf("failed to get tab CDP session: %w", err)
	}

	params := cdp.DOMClickParams{
		BackendNodeID: backendID,
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
	cdpSessionID, err := t.getTabCDPSessionID(tabID...)
	if err != nil {
		return fmt.Errorf("failed to get tab CDP session: %w", err)
	}

	// Get element from cache
	targetTabID := t.activeTabID
	if len(tabID) > 0 && tabID[0] != "" {
		targetTabID = tabID[0]
	}

	t.cacheMu.RLock()
	elements, exists := t.elementCache[targetTabID]
	t.cacheMu.RUnlock()

	if !exists || len(elements) == 0 {
		return fmt.Errorf("no elements cached for tab %s, call read_page first", targetTabID)
	}

	if index < 0 || index >= len(elements) {
		return fmt.Errorf("index %d out of range (0-%d)", index, len(elements)-1)
	}

	elem := elements[index]

	// Click to focus the element first
	x := elem.Bounds.X + elem.Bounds.Width/2
	y := elem.Bounds.Y + elem.Bounds.Height/2

	// Move mouse and click
	moveParams := cdp.InputDispatchMouseEventParams{
		Type:   "mouseMoved",
		X:      x,
		Y:      y,
		Button: "left",
	}
	_, err = t.sendCommand(ctx, "Input.dispatchMouseEvent", moveParams, cdpSessionID)
	if err != nil {
		return fmt.Errorf("failed to move mouse: %w", err)
	}

	pressParams := cdp.InputDispatchMouseEventParams{
		Type:       "mousePressed",
		X:          x,
		Y:          y,
		Button:     "left",
		ClickCount: 1,
	}
	_, err = t.sendCommand(ctx, "Input.dispatchMouseEvent", pressParams, cdpSessionID)
	if err != nil {
		return fmt.Errorf("failed to press mouse: %w", err)
	}

	releaseParams := cdp.InputDispatchMouseEventParams{
		Type:       "mouseReleased",
		X:          x,
		Y:          y,
		Button:     "left",
		ClickCount: 1,
	}
	_, err = t.sendCommand(ctx, "Input.dispatchMouseEvent", releaseParams, cdpSessionID)
	if err != nil {
		return fmt.Errorf("failed to release mouse: %w", err)
	}

	// Type text using Input.insertText
	insertParams := cdp.InputInsertTextParams{
		Text: text,
	}
	_, err = t.sendCommand(ctx, "Input.insertText", insertParams, cdpSessionID)
	if err != nil {
		return fmt.Errorf("failed to insert text: %w", err)
	}

	return nil
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
	cdpSessionID, err := t.getTabCDPSessionID(tabID...)
	if err != nil {
		return fmt.Errorf("failed to get tab CDP session: %w", err)
	}

	var deltaX, deltaY float64

	switch direction {
	case "up":
		deltaY = -float64(amount)
	case "down":
		deltaY = float64(amount)
	case "left":
		deltaX = -float64(amount)
	case "right":
		deltaX = float64(amount)
	default:
		return fmt.Errorf("invalid scroll direction: %s (must be up, down, left, or right)", direction)
	}

	// Use Input.dispatchMouseEvent with type=mouseWheel
	scrollParams := cdp.InputDispatchMouseEventParams{
		Type:   "mouseWheel",
		X:      0,
		Y:      0,
		DeltaX: deltaX,
		DeltaY: deltaY,
	}
	_, err = t.sendCommand(ctx, "Input.dispatchMouseEvent", scrollParams, cdpSessionID)

	return err
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

	createParams := cdp.TargetCreateParams{
		URL: targetURL,
	}

	// Send Target.createTarget (no CDP session ID needed for Target commands)
	createResult, err := t.sendCommand(ctx, "Target.createTarget", createParams)
	if err != nil {
		return nil, fmt.Errorf("failed to create target: %w", err)
	}

	// Convert to typed result
	createJSON, err := json.Marshal(createResult)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal create result: %w", err)
	}

	var createRes cdp.TargetCreateResult
	if err := json.Unmarshal(createJSON, &createRes); err != nil {
		return nil, fmt.Errorf("failed to unmarshal create result: %w", err)
	}

	// Step 2: Attach to target to get CDP session ID
	attachParams := cdp.TargetAttachParams{
		TargetID: createRes.TargetID,
		Flatten:  true, // Flatten session for easier command routing
	}

	attachResult, err := t.sendCommand(ctx, "Target.attachToTarget", attachParams)
	if err != nil {
		return nil, fmt.Errorf("failed to attach to target: %w", err)
	}

	// Convert to typed result
	attachJSON, err := json.Marshal(attachResult)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal attach result: %w", err)
	}

	var attachRes cdp.TargetAttachResult
	if err := json.Unmarshal(attachJSON, &attachRes); err != nil {
		return nil, fmt.Errorf("failed to unmarshal attach result: %w", err)
	}

	// Step 3: Generate friendly tab ID and store mapping
	friendlyTabID := t.generateFriendlyTabID()

	tab := &tabInfo{
		friendlyID:   friendlyTabID,
		targetID:     createRes.TargetID,
		cdpSessionID: attachRes.SessionID,
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
	closeParams := cdp.TargetCloseParams{
		TargetID: targetID,
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

// Helper methods

// getViewportBounds returns the current viewport dimensions
func (t *TunnelClientWrapper) getViewportBounds(ctx context.Context, cdpSessionID string) (*browserprotocol.ViewportBounds, error) {
	result, err := t.sendCommand(ctx, "Page.getLayoutMetrics", nil, cdpSessionID)
	if err != nil {
		return nil, err
	}

	// Convert to typed result
	resultJSON, err := json.Marshal(result)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal layout metrics: %w", err)
	}

	var layoutMetrics cdp.PageGetLayoutMetricsResult
	if err := json.Unmarshal(resultJSON, &layoutMetrics); err != nil {
		return nil, fmt.Errorf("failed to unmarshal layout metrics: %w", err)
	}

	return &browserprotocol.ViewportBounds{
		X:      layoutMetrics.VisualViewport.PageX,
		Y:      layoutMetrics.VisualViewport.PageY,
		Width:  layoutMetrics.VisualViewport.ClientWidth,
		Height: layoutMetrics.VisualViewport.ClientHeight,
	}, nil
}

// convertAXNodesToRaw converts CDP accessibility nodes to protocol raw nodes
func (t *TunnelClientWrapper) convertAXNodesToRaw(nodes []cdp.AccessibilityNode) []browserprotocol.RawAccessibilityNode {
	rawNodes := make([]browserprotocol.RawAccessibilityNode, 0, len(nodes))

	for _, node := range nodes {
		// Get bounds
		var bounds browserprotocol.BoundingBox
		if node.BoundingBox != nil {
			bounds = browserprotocol.BoundingBox{
				X:      node.BoundingBox.X,
				Y:      node.BoundingBox.Y,
				Width:  node.BoundingBox.Width,
				Height: node.BoundingBox.Height,
			}
		}

		// Get attributes from properties
		attributes := make(map[string]string)
		for _, prop := range node.Properties {
			if prop.Name != "" {
				attributes[prop.Name] = prop.Value.Value
			}
		}

		rawNodes = append(rawNodes, browserprotocol.RawAccessibilityNode{
			Role:       node.Role.Value,
			Name:       node.Name.Value,
			Bounds:     bounds,
			BackendID:  node.BackendDOMNodeID,
			FrameID:    node.FrameID,
			Attributes: attributes,
		})
	}

	return rawNodes
}

// isInViewport checks if an element's bounds overlap with the viewport
func isInViewport(bounds *browserprotocol.BoundingBox, viewport *browserprotocol.ViewportBounds) bool {
	// Check if element overlaps with viewport
	return bounds.X+bounds.Width > viewport.X &&
		bounds.X < viewport.X+viewport.Width &&
		bounds.Y+bounds.Height > viewport.Y &&
		bounds.Y < viewport.Y+viewport.Height
}

// isInteractiveRole checks if a role is considered interactive
func isInteractiveRole(role string) bool {
	interactiveRoles := map[string]bool{
		"button":           true,
		"link":             true,
		"textbox":          true,
		"searchbox":        true,
		"combobox":         true,
		"listbox":          true,
		"menu":             true,
		"menuitem":         true,
		"menuitemcheckbox": true,
		"menuitemradio":    true,
		"tab":              true,
		"checkbox":         true,
		"radio":            true,
		"slider":           true,
		"spinbutton":       true,
		"switch":           true,
	}
	return interactiveRoles[strings.ToLower(role)]
}
