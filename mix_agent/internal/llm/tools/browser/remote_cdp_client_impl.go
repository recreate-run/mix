package browser

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"image"
	"sort"
	"strings"
	"time"

	browserprotocol "github.com/sarathmenon/browser-service/pkg/protocol"

	"mix/internal/llm/tools/browser/cdp"
	"mix/internal/logging"
)

// Navigate navigates to a URL
func (c *RemoteCDPClient) Navigate(ctx context.Context, url string, tabID ...string) (*browserprotocol.NavigateResult, error) {
	cdpSessionID, err := c.getTabCDPSessionID(tabID...)
	if err != nil {
		return nil, fmt.Errorf("failed to get tab CDP session: %w", err)
	}

	params := cdp.PageNavigateParams{
		URL: url,
	}

	// Start waiting for load event BEFORE sending navigate command
	loadTimeout := 15 * time.Second
	loadErrChan := make(chan error, 1)
	go func() {
		loadErrChan <- c.waitForEvent(ctx, "Page.loadEventFired", loadTimeout)
	}()

	result, err := c.sendCommand(ctx, "Page.navigate", params, cdpSessionID)
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

	// Wait for page load completion
	if loadErr := <-loadErrChan; loadErr != nil {
		logging.Warn("Page load event timeout during navigation",
			"url", url,
			"error", loadErr)
		// Don't fail - page might still be usable
	}

	return &browserprotocol.NavigateResult{
		FrameID:  navResult.FrameID,
		LoaderID: navResult.LoaderID,
	}, nil
}

// GoBack navigates back in history
func (c *RemoteCDPClient) GoBack(ctx context.Context, tabID ...string) (string, error) {
	cdpSessionID, err := c.getTabCDPSessionID(tabID...)
	if err != nil {
		return "", fmt.Errorf("failed to get tab CDP session: %w", err)
	}

	// Get navigation history
	histResult, err := c.sendCommand(ctx, "Page.getNavigationHistory", nil, cdpSessionID)
	if err != nil {
		return "", fmt.Errorf("failed to get navigation history: %w", err)
	}

	histJSON, err := json.Marshal(histResult)
	if err != nil {
		return "", fmt.Errorf("failed to marshal navigation history: %w", err)
	}

	var navHistory cdp.PageGetNavigationHistoryResult
	if err := json.Unmarshal(histJSON, &navHistory); err != nil {
		return "", fmt.Errorf("failed to unmarshal navigation history: %w", err)
	}

	if navHistory.CurrentIndex <= 0 {
		return "", fmt.Errorf("cannot go back: already at first entry")
	}

	previousEntry := navHistory.Entries[navHistory.CurrentIndex-1]
	params := cdp.PageNavigateToHistoryParams{
		EntryID: previousEntry.ID,
	}

	_, err = c.sendCommand(ctx, "Page.navigateToHistoryEntry", params, cdpSessionID)
	if err != nil {
		return "", fmt.Errorf("failed to navigate back: %w", err)
	}

	return previousEntry.URL, nil
}

// GoForward navigates forward in history
func (c *RemoteCDPClient) GoForward(ctx context.Context, tabID ...string) (string, error) {
	cdpSessionID, err := c.getTabCDPSessionID(tabID...)
	if err != nil {
		return "", fmt.Errorf("failed to get tab CDP session: %w", err)
	}

	histResult, err := c.sendCommand(ctx, "Page.getNavigationHistory", nil, cdpSessionID)
	if err != nil {
		return "", fmt.Errorf("failed to get navigation history: %w", err)
	}

	histJSON, err := json.Marshal(histResult)
	if err != nil {
		return "", fmt.Errorf("failed to marshal navigation history: %w", err)
	}

	var navHistory cdp.PageGetNavigationHistoryResult
	if err := json.Unmarshal(histJSON, &navHistory); err != nil {
		return "", fmt.Errorf("failed to unmarshal navigation history: %w", err)
	}

	if navHistory.CurrentIndex >= len(navHistory.Entries)-1 {
		return "", fmt.Errorf("cannot go forward: already at last entry")
	}

	nextEntry := navHistory.Entries[navHistory.CurrentIndex+1]
	params := cdp.PageNavigateToHistoryParams{
		EntryID: nextEntry.ID,
	}

	_, err = c.sendCommand(ctx, "Page.navigateToHistoryEntry", params, cdpSessionID)
	if err != nil {
		return "", fmt.Errorf("failed to navigate forward: %w", err)
	}

	return nextEntry.URL, nil
}

// Screenshot captures a screenshot
func (c *RemoteCDPClient) Screenshot(ctx context.Context, params browserprotocol.ScreenshotParams) (*browserprotocol.ScreenshotResult, error) {
	var tabIDSlice []string
	if params.TabID != nil {
		tabIDSlice = []string{*params.TabID}
	}
	cdpSessionID, err := c.getTabCDPSessionID(tabIDSlice...)
	if err != nil {
		return nil, fmt.Errorf("failed to get tab CDP session: %w", err)
	}

	targetTabID := c.activeTabID
	if params.TabID != nil && *params.TabID != "" {
		targetTabID = *params.TabID
	}

	// Capture screenshot
	screenshotParams := cdp.PageCaptureScreenshotParams{
		Format:      imageFormatPNG,
		FromSurface: true,
	}

	if params.Quality > 0 {
		screenshotParams.Quality = params.Quality
	}

	if params.Format == imageFormatJPEG {
		screenshotParams.Format = imageFormatJPEG
	}

	result, err := c.sendCommand(ctx, "Page.captureScreenshot", screenshotParams, cdpSessionID)
	if err != nil {
		return nil, fmt.Errorf("screenshot failed: %w", err)
	}

	resultJSON, err := json.Marshal(result)
	if err != nil {
		return nil, fmt.Errorf("unexpected screenshot result type: %T", result)
	}

	var screenshotResult cdp.PageCaptureScreenshotResult
	if err := json.Unmarshal(resultJSON, &screenshotResult); err != nil {
		return nil, fmt.Errorf("failed to unmarshal screenshot result: %w", err)
	}

	decodedImage, err := base64.StdEncoding.DecodeString(screenshotResult.Data)
	if err != nil {
		return nil, fmt.Errorf("failed to decode screenshot data: %w", err)
	}
	imageConfig, _, err := image.DecodeConfig(bytes.NewReader(decodedImage))
	if err != nil {
		return nil, fmt.Errorf("failed to decode screenshot config: %w", err)
	}
	c.setLastScreenshotSize(targetTabID, imageConfig.Width, imageConfig.Height)

	format := params.Format
	if format == "" {
		format = imageFormatPNG
	}

	// Get accessibility tree if requested (with graceful degradation)
	var rawNodes []browserprotocol.RawAccessibilityNode
	if params.Raw {
		// Use separate context with timeout for accessibility tree
		// With domain enabling, this should be nearly instant, but allow time for network latency
		axCtx, axCancel := context.WithTimeout(ctx, 30*time.Second)
		defer axCancel()

		axResult, err := c.sendCommand(axCtx, "Accessibility.getFullAXTree", nil, cdpSessionID)
		if err != nil {
			// Graceful degradation: return screenshot without accessibility tree
			logging.Warn("Failed to get accessibility tree for screenshot, continuing without raw nodes",
				"error", err,
				"tabID", targetTabID)
			// rawNodes remains nil - screenshot will be returned without element data
		} else {
			axJSON, err := json.Marshal(axResult)
			if err != nil {
				logging.Warn("Failed to marshal accessibility result, continuing without raw nodes", "error", err)
			} else {
				var axTree cdp.AccessibilityGetFullAXTreeResult
				if err := json.Unmarshal(axJSON, &axTree); err != nil {
					logging.Warn("Failed to unmarshal accessibility tree, continuing without raw nodes", "error", err)
				} else {
					// Convert accessibility nodes to browser protocol elements
					rawNodes = c.convertAccessibilityNodesToRawNodes(axTree.Nodes, targetTabID)
				}
			}
		}
	}

	return &browserprotocol.ScreenshotResult{
		Data:     screenshotResult.Data,
		Format:   format,
		RawNodes: rawNodes,
	}, nil
}

// convertAccessibilityNodesToRawNodes converts CDP accessibility nodes to browser protocol raw nodes
func (c *RemoteCDPClient) convertAccessibilityNodesToRawNodes(nodes []cdp.AccessibilityNode, tabID string) []browserprotocol.RawAccessibilityNode {
	rawNodes := make([]browserprotocol.RawAccessibilityNode, 0, len(nodes))
	elementCache := make([]elementInfo, 0, len(nodes))

	for _, node := range nodes {
		// Skip nodes without bounding boxes
		if node.BoundingBox == nil {
			continue
		}

		// Extract role and name
		var role, name string
		if node.Role.Value != nil {
			role = fmt.Sprintf("%v", node.Role.Value)
		}
		if node.Name.Value != nil {
			name = fmt.Sprintf("%v", node.Name.Value)
		}

		// Skip non-interactive elements
		if role == "" || name == "" {
			continue
		}

		bounds := browserprotocol.BoundingBox{
			X:      node.BoundingBox.X,
			Y:      node.BoundingBox.Y,
			Width:  node.BoundingBox.Width,
			Height: node.BoundingBox.Height,
		}

		rawNode := browserprotocol.RawAccessibilityNode{
			Role:      role,
			Name:      name,
			Bounds:    bounds,
			BackendID: node.BackendDOMNodeID,
			FrameID:   node.FrameID,
		}

		rawNodes = append(rawNodes, rawNode)

		// Cache element info for clicks
		elementCache = append(elementCache, elementInfo{
			Role:      role,
			Name:      name,
			Bounds:    bounds,
			BackendID: node.BackendDOMNodeID,
			FrameID:   node.FrameID,
		})
	}

	// Store element cache for this tab
	c.cacheMu.Lock()
	c.elementCache[tabID] = elementCache
	c.cacheMu.Unlock()

	return rawNodes
}

// ReadPage reads the accessibility tree
func (c *RemoteCDPClient) ReadPage(ctx context.Context, interactiveOnly bool, tabID ...string) (*browserprotocol.ReadPageResult, error) {
	cdpSessionID, err := c.getTabCDPSessionID(tabID...)
	if err != nil {
		return nil, fmt.Errorf("failed to get tab CDP session: %w", err)
	}

	targetTabID := c.activeTabID
	if len(tabID) > 0 && tabID[0] != "" {
		targetTabID = tabID[0]
	}

	// Get accessibility tree
	// With domain enabling, this should be nearly instant
	axCtx, axCancel := context.WithTimeout(ctx, 30*time.Second)
	defer axCancel()

	axResult, err := c.sendCommand(axCtx, "Accessibility.getFullAXTree", nil, cdpSessionID)
	if err != nil {
		return nil, fmt.Errorf("failed to get accessibility tree: %w", err)
	}

	axJSON, err := json.Marshal(axResult)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal accessibility result: %w", err)
	}

	var axTree cdp.AccessibilityGetFullAXTreeResult
	if err := json.Unmarshal(axJSON, &axTree); err != nil {
		return nil, fmt.Errorf("failed to unmarshal accessibility tree: %w", err)
	}

	// Convert to browser protocol elements
	rawNodes := c.convertAccessibilityNodesToRawNodes(axTree.Nodes, targetTabID)

	// Filter interactive only if requested
	if interactiveOnly {
		filtered := make([]browserprotocol.RawAccessibilityNode, 0, len(rawNodes))
		for _, node := range rawNodes {
			if isInteractiveRoleCDP(node.Role) {
				filtered = append(filtered, node)
			}
		}
		rawNodes = filtered
	}

	return &browserprotocol.ReadPageResult{
		Elements: rawNodes,
	}, nil
}

// isInteractiveRoleCDP checks if a role is interactive (CDP client version)
func isInteractiveRoleCDP(role string) bool {
	interactiveRoles := map[string]bool{
		"button":   true,
		"link":     true,
		"textbox":  true,
		"checkbox": true,
		"radio":    true,
		"combobox": true,
		"listbox":  true,
		"menuitem": true,
		"tab":      true,
		"switch":   true,
		"slider":   true,
	}
	return interactiveRoles[strings.ToLower(role)]
}

// GetText extracts text from the page
func (c *RemoteCDPClient) GetText(ctx context.Context, strategy string, tabID ...string) (*browserprotocol.GetTextResult, error) {
	cdpSessionID, err := c.getTabCDPSessionID(tabID...)
	if err != nil {
		return nil, fmt.Errorf("failed to get tab CDP session: %w", err)
	}

	// Use JavaScript to extract text based on strategy
	var jsExpression string
	switch strategy {
	case "article":
		jsExpression = "document.querySelector('article')?.innerText || ''"
	case "main":
		jsExpression = "document.querySelector('main')?.innerText || ''"
	case "body":
		jsExpression = "document.body?.innerText || ''"
	default:
		jsExpression = "document.body?.innerText || ''"
	}

	params := cdp.RuntimeEvaluateParams{
		Expression:    jsExpression,
		ReturnByValue: true,
	}

	result, err := c.sendCommand(ctx, "Runtime.evaluate", params, cdpSessionID)
	if err != nil {
		return nil, fmt.Errorf("failed to evaluate text extraction: %w", err)
	}

	resultJSON, err := json.Marshal(result)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal runtime result: %w", err)
	}

	var evalResult cdp.RuntimeEvaluateResult
	if err := json.Unmarshal(resultJSON, &evalResult); err != nil {
		return nil, fmt.Errorf("failed to unmarshal runtime result: %w", err)
	}

	text := ""
	if evalResult.Result.Value != nil {
		text = fmt.Sprintf("%v", evalResult.Result.Value)
	}

	return &browserprotocol.GetTextResult{
		Text: text,
	}, nil
}

// Find searches for elements
func (c *RemoteCDPClient) Find(ctx context.Context, query string, limit int, tabID ...string) (*browserprotocol.FindResult, error) {
	cdpSessionID, err := c.getTabCDPSessionID(tabID...)
	if err != nil {
		return nil, fmt.Errorf("failed to get tab CDP session: %w", err)
	}

	targetTabID := c.activeTabID
	if len(tabID) > 0 && tabID[0] != "" {
		targetTabID = tabID[0]
	}

	// Get accessibility tree
	// With domain enabling, this should be nearly instant
	axCtx, axCancel := context.WithTimeout(ctx, 30*time.Second)
	defer axCancel()

	axResult, err := c.sendCommand(axCtx, "Accessibility.getFullAXTree", nil, cdpSessionID)
	if err != nil {
		return nil, fmt.Errorf("failed to get accessibility tree: %w", err)
	}

	axJSON, err := json.Marshal(axResult)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal accessibility result: %w", err)
	}

	var axTree cdp.AccessibilityGetFullAXTreeResult
	if err := json.Unmarshal(axJSON, &axTree); err != nil {
		return nil, fmt.Errorf("failed to unmarshal accessibility tree: %w", err)
	}

	// Convert and search
	allNodes := c.convertAccessibilityNodesToRawNodes(axTree.Nodes, targetTabID)

	// Filter by query
	queryLower := strings.ToLower(query)
	matches := make([]browserprotocol.RawAccessibilityNode, 0)
	for _, node := range allNodes {
		if strings.Contains(strings.ToLower(node.Name), queryLower) ||
			strings.Contains(strings.ToLower(node.Role), queryLower) {
			matches = append(matches, node)
			if limit > 0 && len(matches) >= limit {
				break
			}
		}
	}

	return &browserprotocol.FindResult{
		Elements: matches,
		Total:    len(matches),
	}, nil
}

// Click clicks an element by index
func (c *RemoteCDPClient) Click(ctx context.Context, index int, tabID ...string) error {
	return c.clickByIndexWithButton(ctx, index, "left", 1, tabID...)
}

// clickByIndexWithButton performs a click with specified button and count
func (c *RemoteCDPClient) clickByIndexWithButton(ctx context.Context, index int, button string, clickCount int, tabID ...string) error {
	cdpSessionID, err := c.getTabCDPSessionID(tabID...)
	if err != nil {
		return fmt.Errorf("failed to get tab CDP session: %w", err)
	}

	targetTabID := c.activeTabID
	if len(tabID) > 0 && tabID[0] != "" {
		targetTabID = tabID[0]
	}

	// Get element from cache
	c.cacheMu.RLock()
	elements, exists := c.elementCache[targetTabID]
	c.cacheMu.RUnlock()

	if !exists || index >= len(elements) {
		return fmt.Errorf("element at index %d not found in cache", index)
	}

	elem := elements[index]

	// Calculate click position (center of element)
	x := elem.Bounds.X + elem.Bounds.Width/2
	y := elem.Bounds.Y + elem.Bounds.Height/2

	// Dispatch mouse events
	moveParams := cdp.InputDispatchMouseEventParams{
		Type: "mouseMoved",
		X:    x,
		Y:    y,
	}
	if _, err := c.sendCommand(ctx, "Input.dispatchMouseEvent", moveParams, cdpSessionID); err != nil {
		return fmt.Errorf("failed to move mouse: %w", err)
	}

	pressParams := cdp.InputDispatchMouseEventParams{
		Type:       "mousePressed",
		X:          x,
		Y:          y,
		Button:     button,
		ClickCount: clickCount,
	}
	if _, err := c.sendCommand(ctx, "Input.dispatchMouseEvent", pressParams, cdpSessionID); err != nil {
		return fmt.Errorf("failed to press mouse: %w", err)
	}

	releaseParams := cdp.InputDispatchMouseEventParams{
		Type:       "mouseReleased",
		X:          x,
		Y:          y,
		Button:     button,
		ClickCount: clickCount,
	}
	if _, err := c.sendCommand(ctx, "Input.dispatchMouseEvent", releaseParams, cdpSessionID); err != nil {
		return fmt.Errorf("failed to release mouse: %w", err)
	}

	return nil
}

// ClickByBackendID clicks an element by backend ID
func (c *RemoteCDPClient) ClickByBackendID(ctx context.Context, backendID int64, tabID ...string) error {
	return c.clickByBackendIDWithButton(ctx, backendID, "left", 1, tabID...)
}

// clickByBackendIDWithButton performs a click by backend ID with specified button and count
func (c *RemoteCDPClient) clickByBackendIDWithButton(ctx context.Context, backendID int64, button string, clickCount int, tabID ...string) error {
	cdpSessionID, err := c.getTabCDPSessionID(tabID...)
	if err != nil {
		return fmt.Errorf("failed to get tab CDP session: %w", err)
	}

	// Get element bounds using DOM.getBoxModel (on-demand lookup, no cache dependency)
	boxModelParams := cdp.DOMGetBoxModelParams{
		BackendNodeID: backendID,
	}
	result, err := c.sendCommand(ctx, "DOM.getBoxModel", boxModelParams, cdpSessionID)
	if err != nil {
		return fmt.Errorf("failed to get element box model for backendID %d: %w", backendID, err)
	}

	resultJSON, err := json.Marshal(result)
	if err != nil {
		return fmt.Errorf("unexpected box model result type: %T", result)
	}

	var boxModelResult cdp.DOMGetBoxModelResult
	if err := json.Unmarshal(resultJSON, &boxModelResult); err != nil {
		return fmt.Errorf("failed to unmarshal box model result: %w", err)
	}

	// Calculate click position from content quad (center of bounding box)
	// Content quad: [x1, y1, x2, y2, x3, y3, x4, y4]
	if len(boxModelResult.Model.Content) < 8 {
		return fmt.Errorf("invalid box model content quad for backendID %d", backendID)
	}

	x := (boxModelResult.Model.Content[0] + boxModelResult.Model.Content[4]) / 2
	y := (boxModelResult.Model.Content[1] + boxModelResult.Model.Content[5]) / 2

	// Dispatch mouse events
	moveParams := cdp.InputDispatchMouseEventParams{
		Type: "mouseMoved",
		X:    x,
		Y:    y,
	}
	if _, err := c.sendCommand(ctx, "Input.dispatchMouseEvent", moveParams, cdpSessionID); err != nil {
		return fmt.Errorf("failed to move mouse: %w", err)
	}

	pressParams := cdp.InputDispatchMouseEventParams{
		Type:       "mousePressed",
		X:          x,
		Y:          y,
		Button:     button,
		ClickCount: clickCount,
	}
	if _, err := c.sendCommand(ctx, "Input.dispatchMouseEvent", pressParams, cdpSessionID); err != nil {
		return fmt.Errorf("failed to press mouse: %w", err)
	}

	releaseParams := cdp.InputDispatchMouseEventParams{
		Type:       "mouseReleased",
		X:          x,
		Y:          y,
		Button:     button,
		ClickCount: clickCount,
	}
	if _, err := c.sendCommand(ctx, "Input.dispatchMouseEvent", releaseParams, cdpSessionID); err != nil {
		return fmt.Errorf("failed to release mouse: %w", err)
	}

	return nil
}

// RightClick right-clicks an element by index
func (c *RemoteCDPClient) RightClick(ctx context.Context, index int, tabID ...string) error {
	return c.clickByIndexWithButton(ctx, index, "right", 1, tabID...)
}

// RightClickByBackendID right-clicks an element by backend ID
func (c *RemoteCDPClient) RightClickByBackendID(ctx context.Context, backendID int64, tabID ...string) error {
	return c.clickByBackendIDWithButton(ctx, backendID, "right", 1, tabID...)
}

// DoubleClick double-clicks an element by index
func (c *RemoteCDPClient) DoubleClick(ctx context.Context, index int, tabID ...string) error {
	return c.clickByIndexWithButton(ctx, index, "left", 2, tabID...)
}

// DoubleClickByBackendID double-clicks an element by backend ID
func (c *RemoteCDPClient) DoubleClickByBackendID(ctx context.Context, backendID int64, tabID ...string) error {
	return c.clickByBackendIDWithButton(ctx, backendID, "left", 2, tabID...)
}

// TripleClick triple-clicks an element by index
func (c *RemoteCDPClient) TripleClick(ctx context.Context, index int, tabID ...string) error {
	return c.clickByIndexWithButton(ctx, index, "left", 3, tabID...)
}

// TripleClickByBackendID triple-clicks an element by backend ID
func (c *RemoteCDPClient) TripleClickByBackendID(ctx context.Context, backendID int64, tabID ...string) error {
	return c.clickByBackendIDWithButton(ctx, backendID, "left", 3, tabID...)
}

// Drag performs a drag operation
func (c *RemoteCDPClient) Drag(ctx context.Context, fromIndex, toIndex *int, fromX, fromY, toX, toY *float64, duration *int, tabID ...string) error {
	cdpSessionID, err := c.getTabCDPSessionID(tabID...)
	if err != nil {
		return fmt.Errorf("failed to get tab CDP session: %w", err)
	}

	targetTabID := c.activeTabID
	if len(tabID) > 0 && tabID[0] != "" {
		targetTabID = tabID[0]
	}

	// Determine start and end coordinates
	var startX, startY, endX, endY float64

	switch {
	case fromX != nil && fromY != nil:
		startX, startY = *fromX, *fromY
	case fromIndex != nil:
		c.cacheMu.RLock()
		elements, exists := c.elementCache[targetTabID]
		c.cacheMu.RUnlock()

		if !exists || *fromIndex >= len(elements) {
			return fmt.Errorf("from element at index %d not found", *fromIndex)
		}

		elem := elements[*fromIndex]
		startX = elem.Bounds.X + elem.Bounds.Width/2
		startY = elem.Bounds.Y + elem.Bounds.Height/2
	default:
		return fmt.Errorf("either fromIndex or fromX/fromY must be provided")
	}

	switch {
	case toX != nil && toY != nil:
		endX, endY = *toX, *toY
	case toIndex != nil:
		c.cacheMu.RLock()
		elements, exists := c.elementCache[targetTabID]
		c.cacheMu.RUnlock()

		if !exists || *toIndex >= len(elements) {
			return fmt.Errorf("to element at index %d not found", *toIndex)
		}

		elem := elements[*toIndex]
		endX = elem.Bounds.X + elem.Bounds.Width/2
		endY = elem.Bounds.Y + elem.Bounds.Height/2
	default:
		return fmt.Errorf("either toIndex or toX/toY must be provided")
	}

	// Perform drag
	moveParams := cdp.InputDispatchMouseEventParams{
		Type: "mouseMoved",
		X:    startX,
		Y:    startY,
	}
	if _, err := c.sendCommand(ctx, "Input.dispatchMouseEvent", moveParams, cdpSessionID); err != nil {
		return fmt.Errorf("failed to move to start: %w", err)
	}

	pressParams := cdp.InputDispatchMouseEventParams{
		Type:       "mousePressed",
		X:          startX,
		Y:          startY,
		Button:     "left",
		ClickCount: 1,
	}
	if _, err := c.sendCommand(ctx, "Input.dispatchMouseEvent", pressParams, cdpSessionID); err != nil {
		return fmt.Errorf("failed to press mouse: %w", err)
	}

	// Move to end position
	dragParams := cdp.InputDispatchMouseEventParams{
		Type: "mouseMoved",
		X:    endX,
		Y:    endY,
	}
	if _, err := c.sendCommand(ctx, "Input.dispatchMouseEvent", dragParams, cdpSessionID); err != nil {
		return fmt.Errorf("failed to drag: %w", err)
	}

	releaseParams := cdp.InputDispatchMouseEventParams{
		Type:       "mouseReleased",
		X:          endX,
		Y:          endY,
		Button:     "left",
		ClickCount: 1,
	}
	if _, err := c.sendCommand(ctx, "Input.dispatchMouseEvent", releaseParams, cdpSessionID); err != nil {
		return fmt.Errorf("failed to release mouse: %w", err)
	}

	return nil
}

// Type types text into an element
func (c *RemoteCDPClient) Type(ctx context.Context, index int, text string, tabID ...string) error {
	// First click the element to focus it
	if err := c.Click(ctx, index, tabID...); err != nil {
		return fmt.Errorf("failed to focus element: %w", err)
	}

	// Small delay to ensure focus
	time.Sleep(100 * time.Millisecond)

	cdpSessionID, err := c.getTabCDPSessionID(tabID...)
	if err != nil {
		return fmt.Errorf("failed to get tab CDP session: %w", err)
	}

	// Type text
	params := cdp.InputInsertTextParams{
		Text: text,
	}

	_, err = c.sendCommand(ctx, "Input.insertText", params, cdpSessionID)
	if err != nil {
		return fmt.Errorf("failed to type text: %w", err)
	}

	return nil
}

// PressKey presses keyboard keys
func (c *RemoteCDPClient) PressKey(ctx context.Context, keys string, tabID ...string) error {
	cdpSessionID, err := c.getTabCDPSessionID(tabID...)
	if err != nil {
		return fmt.Errorf("failed to get tab CDP session: %w", err)
	}

	// Send key down
	downParams := cdp.InputDispatchKeyEventParams{
		Type: "keyDown",
		Key:  keys,
	}
	if _, err := c.sendCommand(ctx, "Input.dispatchKeyEvent", downParams, cdpSessionID); err != nil {
		return fmt.Errorf("failed to press key down: %w", err)
	}

	// Send key up
	upParams := cdp.InputDispatchKeyEventParams{
		Type: "keyUp",
		Key:  keys,
	}
	if _, err := c.sendCommand(ctx, "Input.dispatchKeyEvent", upParams, cdpSessionID); err != nil {
		return fmt.Errorf("failed to press key up: %w", err)
	}

	return nil
}

// FormInput sets form input value
func (c *RemoteCDPClient) FormInput(ctx context.Context, index int, value string, tabID ...string) error {
	return c.Type(ctx, index, value, tabID...)
}

// Scroll scrolls the page
func (c *RemoteCDPClient) Scroll(ctx context.Context, direction string, amount int, tabID ...string) error {
	cdpSessionID, err := c.getTabCDPSessionID(tabID...)
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
		return fmt.Errorf("invalid scroll direction: %s", direction)
	}

	params := cdp.InputDispatchMouseWheelParams{
		Type:   "mouseWheel",
		X:      0,
		Y:      0,
		DeltaX: deltaX,
		DeltaY: deltaY,
	}

	_, err = c.sendCommand(ctx, "Input.dispatchMouseEvent", params, cdpSessionID)
	if err != nil {
		return fmt.Errorf("failed to scroll: %w", err)
	}

	return nil
}

// ScrollIntoView scrolls an element into view
func (c *RemoteCDPClient) ScrollIntoView(ctx context.Context, index *int, backendID *int64, tabID ...string) error {
	if backendID != nil {
		return c.ScrollIntoViewByBackendID(ctx, *backendID, tabID...)
	}
	if index != nil {
		return c.ScrollIntoViewByIndex(ctx, *index, tabID...)
	}
	return fmt.Errorf("either index or backendID must be provided")
}

// ScrollIntoViewByIndex scrolls an element into view by index
func (c *RemoteCDPClient) ScrollIntoViewByIndex(ctx context.Context, index int, tabID ...string) error {
	targetTabID := c.activeTabID
	if len(tabID) > 0 && tabID[0] != "" {
		targetTabID = tabID[0]
	}

	c.cacheMu.RLock()
	elements, exists := c.elementCache[targetTabID]
	c.cacheMu.RUnlock()

	if !exists || index >= len(elements) {
		return fmt.Errorf("element at index %d not found", index)
	}

	return c.ScrollIntoViewByBackendID(ctx, elements[index].BackendID, tabID...)
}

// ScrollIntoViewByBackendID scrolls an element into view by backend ID
func (c *RemoteCDPClient) ScrollIntoViewByBackendID(ctx context.Context, backendID int64, tabID ...string) error {
	cdpSessionID, err := c.getTabCDPSessionID(tabID...)
	if err != nil {
		return fmt.Errorf("failed to get tab CDP session: %w", err)
	}

	params := cdp.DOMScrollIntoViewIfNeededParams{
		BackendNodeID: backendID,
	}

	_, err = c.sendCommand(ctx, "DOM.scrollIntoViewIfNeeded", params, cdpSessionID)
	if err != nil {
		return fmt.Errorf("failed to scroll into view: %w", err)
	}

	return nil
}

// UploadFile uploads files to a file input element
func (c *RemoteCDPClient) UploadFile(ctx context.Context, index int, filePaths []string, tabID ...string) (*browserprotocol.UploadFileResult, error) {
	cdpSessionID, err := c.getTabCDPSessionID(tabID...)
	if err != nil {
		return nil, fmt.Errorf("failed to get tab CDP session: %w", err)
	}

	targetTabID := c.activeTabID
	if len(tabID) > 0 && tabID[0] != "" {
		targetTabID = tabID[0]
	}

	// Get element from cache
	c.cacheMu.RLock()
	elements, exists := c.elementCache[targetTabID]
	c.cacheMu.RUnlock()

	if !exists || index >= len(elements) {
		return nil, fmt.Errorf("element at index %d not found", index)
	}

	params := cdp.DOMSetFileInputFilesParams{
		Files:         filePaths,
		BackendNodeID: elements[index].BackendID,
	}

	_, err = c.sendCommand(ctx, "DOM.setFileInputFiles", params, cdpSessionID)
	if err != nil {
		return nil, fmt.Errorf("failed to upload file: %w", err)
	}

	return &browserprotocol.UploadFileResult{
		FilesUploaded: len(filePaths),
		FileNames:     filePaths,
	}, nil
}

// CreateTab creates a new tab
func (c *RemoteCDPClient) CreateTab(ctx context.Context, url ...string) (*browserprotocol.TabInfo, error) {
	userURL := ""
	if len(url) > 0 && url[0] != "" {
		userURL = url[0]
	}

	// Always create target with a data URL (not about:blank)
	// about:blank has special browser handling that can cause target detachment
	// Match local browser service behavior using an empty HTML data URL
	createParams := cdp.TargetCreateParams{
		URL: "data:text/html,<html><body></body></html>",
	}

	result, err := c.sendCommand(ctx, "Target.createTarget", createParams)
	if err != nil {
		return nil, fmt.Errorf("failed to create target: %w", err)
	}

	resultJSON, err := json.Marshal(result)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal create result: %w", err)
	}

	var createResult cdp.TargetCreateResult
	if err := json.Unmarshal(resultJSON, &createResult); err != nil {
		return nil, fmt.Errorf("failed to unmarshal create result: %w", err)
	}

	// Attach to target
	attachParams := cdp.TargetAttachParams{
		TargetID: createResult.TargetID,
		Flatten:  true,
	}

	attachResult, err := c.sendCommand(ctx, "Target.attachToTarget", attachParams)
	if err != nil {
		return nil, fmt.Errorf("failed to attach to target: %w", err)
	}

	attachJSON, err := json.Marshal(attachResult)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal attach result: %w", err)
	}

	var attach cdp.TargetAttachResult
	if err := json.Unmarshal(attachJSON, &attach); err != nil {
		return nil, fmt.Errorf("failed to unmarshal attach result: %w", err)
	}

	// Generate friendly tab ID
	friendlyID := c.generateFriendlyTabID()

	// Store tab info
	c.tabsMu.Lock()
	c.tabs[friendlyID] = &tabInfo{
		friendlyID:   friendlyID,
		targetID:     createResult.TargetID,
		cdpSessionID: attach.SessionID,
	}
	c.activeTabID = friendlyID
	c.tabsMu.Unlock()

	// Enable required CDP domains for this tab
	// This is critical - without enabling, browser builds accessibility tree from scratch (30+ seconds)
	// With enabling, browser maintains tree incrementally (instant)
	sessionID := attach.SessionID
	_, _ = c.sendCommand(ctx, "Page.enable", nil, sessionID)
	_, _ = c.sendCommand(ctx, "DOM.enable", nil, sessionID)
	_, _ = c.sendCommand(ctx, "Accessibility.enable", nil, sessionID)
	_, _ = c.sendCommand(ctx, "Runtime.enable", nil, sessionID)

	// Determine final URL to return
	finalURL := "data:text/html,<html><body></body></html>"
	if userURL == urlAboutBlank {
		// User explicitly requested about:blank - return that
		// (but we already have blank page via data URL, no need to navigate)
		finalURL = urlAboutBlank
	} else if userURL != "" {
		// User provided a real URL - navigate explicitly and wait for page load
		navParams := cdp.PageNavigateParams{
			URL: userURL,
		}

		// Start waiting for load event BEFORE sending navigate command
		loadTimeout := 15 * time.Second
		loadErrChan := make(chan error, 1)
		go func() {
			loadErrChan <- c.waitForEvent(ctx, "Page.loadEventFired", loadTimeout)
		}()

		_, err = c.sendCommand(ctx, "Page.navigate", navParams, sessionID)
		if err != nil {
			return nil, fmt.Errorf("failed to navigate to %s: %w", userURL, err)
		}

		// Wait for page load completion
		if loadErr := <-loadErrChan; loadErr != nil {
			logging.Warn("Page load event timeout - page may not be fully loaded",
				"url", userURL,
				"tabID", friendlyID,
				"error", loadErr)
			// Don't fail - page might still be usable
		}

		finalURL = userURL
	}

	logging.Info("Created new tab", "tabID", friendlyID, "targetID", createResult.TargetID, "sessionID", attach.SessionID, "url", finalURL)

	return &browserprotocol.TabInfo{
		ID:  friendlyID,
		URL: finalURL,
	}, nil
}

// ListTabs lists all tabs
func (c *RemoteCDPClient) ListTabs(ctx context.Context) (*browserprotocol.TabListResult, error) {
	c.tabsMu.RLock()
	defer c.tabsMu.RUnlock()

	tabs := make([]browserprotocol.TabInfo, 0, len(c.tabs))
	for _, tab := range c.tabs {
		tabs = append(tabs, browserprotocol.TabInfo{
			ID: tab.friendlyID,
		})
	}

	// Sort by tab ID for consistent ordering
	sort.Slice(tabs, func(i, j int) bool {
		return tabs[i].ID < tabs[j].ID
	})

	return &browserprotocol.TabListResult{
		Tabs:        tabs,
		ActiveTabID: c.activeTabID,
	}, nil
}

// SwitchTab switches to a different tab
func (c *RemoteCDPClient) SwitchTab(ctx context.Context, tabID string) error {
	c.tabsMu.Lock()
	defer c.tabsMu.Unlock()

	if _, exists := c.tabs[tabID]; !exists {
		return fmt.Errorf("tab not found: %s", tabID)
	}

	c.activeTabID = tabID
	return nil
}

// CloseTab closes a tab
func (c *RemoteCDPClient) CloseTab(ctx context.Context, tabID string) error {
	c.tabsMu.Lock()
	tab, exists := c.tabs[tabID]
	if !exists {
		c.tabsMu.Unlock()
		return fmt.Errorf("tab not found: %s", tabID)
	}

	targetID := tab.targetID
	delete(c.tabs, tabID)

	// Update active tab if we closed it
	if c.activeTabID == tabID {
		c.activeTabID = ""
		// Set active tab to first remaining tab
		for id := range c.tabs {
			c.activeTabID = id
			break
		}
	}
	c.tabsMu.Unlock()

	// Close target
	closeParams := cdp.TargetCloseParams{
		TargetID: targetID,
	}

	_, err := c.sendCommand(ctx, "Target.closeTarget", closeParams)
	if err != nil {
		return fmt.Errorf("failed to close target: %w", err)
	}

	// Clean up caches
	c.cacheMu.Lock()
	delete(c.elementCache, tabID)
	c.cacheMu.Unlock()

	c.screenshotMu.Lock()
	delete(c.screenshotSize, tabID)
	c.screenshotMu.Unlock()

	return nil
}

// Wait pauses execution
func (c *RemoteCDPClient) Wait(ctx context.Context, duration int, tabID ...string) error {
	time.Sleep(time.Duration(duration) * time.Millisecond)
	return nil
}
