package browser

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/go-rod/rod"
	"github.com/go-rod/rod/lib/input"
	"github.com/go-rod/rod/lib/proto"
	"github.com/sarathmenon/browser-service/internal/browser/events"
	"github.com/sarathmenon/browser-service/internal/browser/watchdog"
	"github.com/sarathmenon/browser-service/internal/constants"
	"github.com/sarathmenon/browser-service/internal/errors"
	"github.com/sarathmenon/browser-service/pkg/convex"
	"github.com/sarathmenon/browser-service/pkg/protocol"
)

// Context represents an isolated browser context for a single client
type Context struct {
	browser             *rod.Browser
	tabs                map[string]*tabContext // tabID → tab context
	activeTabID         string                 // Current active tab
	tabIDCounter        uint64                 // Atomic counter for tab IDs
	mu                  sync.RWMutex
	popupsWatchdog      *watchdog.PopupsWatchdog
	permissionsWatchdog *watchdog.PermissionsWatchdog
	crashWatchdog       *watchdog.CrashWatchdog
	storageWatchdog     *watchdog.StorageStateWatchdog
	eventBus            *events.Broker[events.BrowserEvent]
}

// tabContext represents a single browser tab
type tabContext struct {
	id                string
	page              *rod.Page
	elements          []elementInfo
	mu                sync.RWMutex // Per-tab element cache lock
	currentURL        string       // Cached current URL
	currentTitle      string       // Cached current title
	navigationTimeout time.Duration // Navigation timeout (not wrapped page)
	downloads         []protocol.Download
	downloadsMu       sync.RWMutex
	downloadChan      chan protocol.Download
	downloadsWatchdog *watchdog.DownloadsWatchdog
}

// elementInfo stores element data for click/type operations
type elementInfo struct {
	Role      string
	Name      string
	Bounds    protocol.BoundingBox
	BackendID int64
}

// interactiveRoles defines the set of accessibility roles that are interactive
var interactiveRoles = map[string]bool{
	constants.RoleButton:           true,
	constants.RoleLink:             true,
	constants.RoleTextbox:          true,
	constants.RoleSearchbox:        true,
	constants.RoleCombobox:         true,
	constants.RoleListbox:          true,
	constants.RoleMenu:             true,
	constants.RoleMenuItem:         true,
	constants.RoleMenuItemCheckbox: true,
	constants.RoleMenuItemRadio:    true,
	constants.RoleTab:              true,
	constants.RoleCheckbox:         true,
	constants.RoleRadio:            true,
	constants.RoleSlider:           true,
	constants.RoleSpinbutton:       true,
	constants.RoleSwitch:           true,
}

// getTab returns the tab context for the given tabID, or the active tab if tabID is nil
func (c *Context) getTab(tabID *string) (*tabContext, error) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	var targetID string
	if tabID == nil || *tabID == "" {
		targetID = c.activeTabID
	} else {
		targetID = *tabID
	}

	tab, exists := c.tabs[targetID]
	if !exists {
		return nil, errors.NewValidationError("tabID", targetID, fmt.Errorf("tab not found"))
	}

	return tab, nil
}

// CreateTab creates a new tab and returns its info
func (c *Context) CreateTab(ctx context.Context) (*protocol.TabInfo, error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	// Generate new tab ID using atomic counter
	counter := atomic.AddUint64(&c.tabIDCounter, 1)
	tabID := fmt.Sprintf("tab-%d", counter)

	// Create new page
	page, err := c.browser.Page(proto.TargetCreateTarget{})
	if err != nil {
		return nil, errors.NewContextError("create_tab", err)
	}

	// Navigate to blank page with proper HTML to ensure valid URL
	err = page.Navigate("data:text/html,<html><body></body></html>")
	if err != nil {
		return nil, errors.NewContextError("initialize_tab", err)
	}
	err = page.WaitLoad()
	if err != nil {
		return nil, errors.NewContextError("wait_tab_load", err)
	}

	// Get page info after initialization
	info, err := page.Info()
	if err != nil {
		return nil, errors.NewContextError("get_page_info", err)
	}

	// Create tab context with cached URL and title
	tab := &tabContext{
		id:                tabID,
		page:              page,
		elements:          make([]elementInfo, 0),
		currentURL:        info.URL,
		currentTitle:      info.Title,
		navigationTimeout: constants.DefaultNavigationTimeout,
		downloads:         make([]protocol.Download, 0),
		downloadChan:      make(chan protocol.Download, 10), // Buffered channel for download events
	}

	c.tabs[tabID] = tab

	// Register page with watchdogs
	if c.popupsWatchdog != nil {
		if err := c.popupsWatchdog.RegisterPage(ctx, page); err != nil {
			// Log but don't fail - watchdog registration is non-critical
			_ = err
		}
	}

	if c.crashWatchdog != nil {
		if err := c.crashWatchdog.RegisterPage(ctx, page); err != nil {
			// Log but don't fail - watchdog registration is non-critical
			_ = err
		}
	}

	return &protocol.TabInfo{
		ID:       tabID,
		URL:      info.URL,
		Title:    info.Title,
		IsActive: false,
	}, nil
}

// ListTabs returns information about all tabs
func (c *Context) ListTabs(ctx context.Context) ([]protocol.TabInfo, string, error) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	tabs := make([]protocol.TabInfo, 0, len(c.tabs))

	for _, tab := range c.tabs {
		tab.mu.RLock()
		// Use cached URL and title to avoid issues with wrapped pages
		url := tab.currentURL
		title := tab.currentTitle

		// Fallback to page.Info() only if cached values are empty
		if url == "" || title == "" {
			info, err := tab.page.Info()
			if err == nil {
				url = info.URL
				title = info.Title
			}
		}
		tab.mu.RUnlock()

		tabs = append(tabs, protocol.TabInfo{
			ID:       tab.id,
			URL:      url,
			Title:    title,
			IsActive: tab.id == c.activeTabID,
		})
	}

	// Sort tabs by ID for consistent ordering
	sort.Slice(tabs, func(i, j int) bool {
		return tabs[i].ID < tabs[j].ID
	})

	return tabs, c.activeTabID, nil
}

// SwitchTab switches the active tab to the specified tabID
func (c *Context) SwitchTab(ctx context.Context, tabID string) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if _, exists := c.tabs[tabID]; !exists {
		return errors.NewValidationError("tabID", tabID, fmt.Errorf("tab not found"))
	}

	c.activeTabID = tabID
	return nil
}

// CloseTab closes the specified tab
func (c *Context) CloseTab(ctx context.Context, tabID string) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	// Cannot close last tab
	if len(c.tabs) == 1 {
		return errors.NewValidationError("tabID", tabID, fmt.Errorf("cannot close last tab"))
	}

	tab, exists := c.tabs[tabID]
	if !exists {
		return errors.NewValidationError("tabID", tabID, fmt.Errorf("tab not found"))
	}

	// Close the page
	tab.mu.Lock()
	err := tab.page.Close()
	tab.mu.Unlock()

	if err != nil {
		return errors.NewContextError("close_tab", err)
	}

	// Remove from map
	delete(c.tabs, tabID)

	// If we closed the active tab, switch to first remaining tab (lexicographic order)
	if c.activeTabID == tabID {
		// Find first tab (lexicographically)
		var firstTabID string
		for id := range c.tabs {
			if firstTabID == "" || id < firstTabID {
				firstTabID = id
			}
		}
		c.activeTabID = firstTabID
	}

	return nil
}

// Navigate navigates to a URL
func (c *Context) Navigate(ctx context.Context, url string, timeoutMs int, tabID *string) (*protocol.NavigateResult, error) {
	tab, err := c.getTab(tabID)
	if err != nil {
		return nil, err
	}

	tab.mu.Lock()
	defer tab.mu.Unlock()

	// Clear element cache BEFORE navigation (even if navigation fails)
	tab.elements = nil

	if timeoutMs <= 0 {
		timeoutMs = int(constants.DefaultNavigationTimeout.Milliseconds())
	}

	// Store timeout in tab context instead of wrapping page
	tab.navigationTimeout = time.Duration(timeoutMs) * time.Millisecond

	// Store previous URL for error recovery verification
	previousURL := tab.currentURL

	// Create timeout context for navigation
	navCtx, cancel := context.WithTimeout(ctx, tab.navigationTimeout)
	defer cancel()

	// Navigate with timeout context using rod.Try for error handling
	var navErr error
	_ = rod.Try(func() {
		navErr = tab.page.Context(navCtx).Navigate(url)
		if navErr != nil {
			return
		}

		// Wait for page load
		navErr = tab.page.Context(navCtx).WaitLoad()
	})

	if navErr != nil {
		// Verify page state on error
		info, infoErr := tab.page.Info()
		if infoErr != nil || info.URL == "" || (info.URL != url && info.URL != previousURL) {
			// Page is in corrupted state - log warning but don't try to recover
			// The important fix is that we didn't wrap the page, so it's still usable
		}
		return nil, errors.NewNavigationError(url, navErr)
	}

	// Update cached URL and title on successful navigation
	info, err := tab.page.Info()
	if err != nil {
		return nil, errors.NewNavigationError(url, fmt.Errorf("failed to get page info after navigation: %w", err))
	}

	tab.currentURL = info.URL
	tab.currentTitle = info.Title

	return &protocol.NavigateResult{
		FrameID: string(tab.page.FrameID),
	}, nil
}

// Screenshot captures a screenshot
func (c *Context) Screenshot(ctx context.Context, params protocol.ScreenshotParams) (*protocol.ScreenshotResult, error) {
	tab, err := c.getTab(params.TabID)
	if err != nil {
		return nil, err
	}

	tab.mu.Lock()
	defer tab.mu.Unlock()

	var data []byte

	if params.FullPage {
		data, err = tab.page.Screenshot(true, &proto.PageCaptureScreenshot{
			Format: proto.PageCaptureScreenshotFormatPng,
		})
	} else {
		data, err = tab.page.Screenshot(false, &proto.PageCaptureScreenshot{
			Format: proto.PageCaptureScreenshotFormatPng,
		})
	}

	if err != nil {
		return nil, errors.NewBrowserError("screenshot", err)
	}

	format := params.Format
	if format == "" {
		format = constants.DefaultImageFormat
	}

	result := &protocol.ScreenshotResult{
		Data:   base64.StdEncoding.EncodeToString(data),
		Format: format,
	}

	// Raw mode: return accessibility tree without processing
	if params.Raw {
		// Get full accessibility tree
		tree, err := proto.AccessibilityGetFullAXTree{}.Call(tab.page)
		if err != nil {
			return nil, fmt.Errorf("failed to get accessibility tree: %w", err)
		}

		// Get viewport bounds
		viewport, err := tab.getViewportBounds()
		if err != nil {
			return nil, fmt.Errorf("failed to get viewport: %w", err)
		}

		// Convert CDP nodes to raw protocol nodes (no filtering)
		rawNodes := make([]protocol.RawAccessibilityNode, 0, len(tree.Nodes))
		for _, node := range tree.Nodes {
			// Get bounds via DOM
			bounds, err := tab.getNodeBounds(node)
			if err != nil {
				continue // Skip nodes without bounds
			}

			rawNodes = append(rawNodes, protocol.RawAccessibilityNode{
				Role:      getNodeRole(node),
				Name:      getNodeName(node),
				Bounds:    *bounds,
				BackendID: int64(node.BackendDOMNodeID),
			})
		}

		result.RawNodes = rawNodes
		viewportBounds := protocol.ViewportBounds(*viewport)
		result.RawViewport = &viewportBounds
	}

	return result, nil
}

// EvalJS evaluates JavaScript in the page context
func (c *Context) EvalJS(ctx context.Context, js string, tabID *string) (*proto.RuntimeRemoteObject, error) {
	tab, err := c.getTab(tabID)
	if err != nil {
		return nil, err
	}

	tab.mu.RLock()
	defer tab.mu.RUnlock()

	// Wrap expression in arrow function for Rod's Eval
	wrappedJS := fmt.Sprintf("() => %s", js)
	result, err := tab.page.Eval(wrappedJS)
	if err != nil {
		return nil, errors.NewBrowserError("eval_js", err)
	}
	return result, nil
}

// GetElements returns interactive elements on the page
func (c *Context) GetElements(ctx context.Context, tabID *string) ([]protocol.RawAccessibilityNode, error) {
	tab, err := c.getTab(tabID)
	if err != nil {
		return nil, err
	}

	tab.mu.Lock()
	defer tab.mu.Unlock()

	return tab.extractElements()
}

// ReadPage returns accessibility tree for visible viewport elements only
func (c *Context) ReadPage(ctx context.Context, interactiveOnly bool, tabID *string) ([]protocol.RawAccessibilityNode, *protocol.BoundingBox, error) {
	tab, err := c.getTab(tabID)
	if err != nil {
		return nil, nil, err
	}

	tab.mu.Lock()
	defer tab.mu.Unlock()

	// Get current viewport bounds
	viewport, err := tab.getViewportBounds()
	if err != nil {
		return nil, nil, fmt.Errorf("failed to get viewport: %w", err)
	}

	// Get full accessibility tree
	tree, err := proto.AccessibilityGetFullAXTree{}.Call(tab.page)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to get accessibility tree: %w", err)
	}

	// First pass: collect all backend IDs that pass filtering
	type candidateElement struct {
		node   *proto.AccessibilityAXNode
		bounds *protocol.BoundingBox
		name   string
		role   string
	}

	var candidates []candidateElement
	var candidateBackendIDs []int64

	for _, node := range tree.Nodes {
		// Filter for interactive elements if requested
		if interactiveOnly && !isInteractive(node) {
			continue
		}

		// Get element bounds
		bounds, err := tab.getNodeBounds(node)
		if err != nil || bounds == nil {
			continue
		}

		// Skip zero-size elements
		if bounds.Width <= 0 || bounds.Height <= 0 {
			continue
		}

		// CRITICAL: Filter by viewport visibility
		if !isInViewport(bounds, viewport) {
			continue
		}

		name := getNodeName(node)
		role := getNodeRole(node)
		backendID := int64(node.BackendDOMNodeID)

		candidates = append(candidates, candidateElement{
			node:   node,
			bounds: bounds,
			name:   name,
			role:   role,
		})
		candidateBackendIDs = append(candidateBackendIDs, backendID)
	}

	// Batch extract attributes in parallel
	attributesMap := tab.batchExtractAttributes(candidateBackendIDs)

	// Second pass: build final elements with attributes
	frameIDStr := string(tab.page.FrameID)
	elements := make([]protocol.RawAccessibilityNode, 0, len(candidates))

	for i, candidate := range candidates {
		backendID := candidateBackendIDs[i]
		attrs := attributesMap[backendID] // nil if extraction failed

		elem := protocol.RawAccessibilityNode{
			Role:       candidate.role,
			Name:       candidate.name,
			Bounds:     *candidate.bounds,
			BackendID:  backendID,
			FrameID:    frameIDStr,
			Attributes: attrs,
		}
		elements = append(elements, elem)
	}

	// CRITICAL: Populate element cache so ClickByBackendID can find these elements
	// Without this, parallel sessions experience "element not found in cache" errors
	tab.elements = make([]elementInfo, len(elements))
	for i, elem := range elements {
		tab.elements[i] = elementInfo{
			Role:      elem.Role,
			Name:      elem.Name,
			Bounds:    elem.Bounds,
			BackendID: elem.BackendID,
		}
	}

	return elements, viewport, nil
}

// extractElementAttributes extracts HTML attributes from a DOM node using CDP
// Pattern follows UploadFile method for BackendNodeID → ObjectID → JavaScript execution
func (t *tabContext) extractElementAttributes(backendID int64) (map[string]string, error) {
	// Step 1: Resolve BackendNodeID to ObjectID (required for JavaScript execution)
	nodeResult, err := proto.DOMResolveNode{
		BackendNodeID: proto.DOMBackendNodeID(backendID),
	}.Call(t.page)
	if err != nil {
		return nil, err
	}

	if nodeResult.Object.ObjectID == "" {
		return nil, fmt.Errorf("no object ID for element")
	}

	// Step 2: Execute JavaScript to extract common attributes
	// Priority: href, id, type, placeholder, name, aria-label
	jsExtract := `(function() {
		const elem = this;
		const attrs = {};

		// Extract priority attributes via getAttribute (works for all)
		const priorityAttrs = ['href', 'id', 'type', 'placeholder', 'name', 'aria-label'];
		priorityAttrs.forEach(attr => {
			const val = elem.getAttribute(attr);
			if (val !== null && val !== '') {
				attrs[attr] = val;
			}
		});

		// Special cases: properties that aren't attributes
		if (elem.value !== undefined && elem.value !== '') {
			attrs['value'] = elem.value;
		}

		return attrs;
	})()`

	// Step 3: Call JavaScript on the element
	result, err := proto.RuntimeCallFunctionOn{
		ObjectID:            nodeResult.Object.ObjectID,
		FunctionDeclaration: jsExtract,
		ReturnByValue:       true, // CRITICAL: Get actual JSON value, not object reference
	}.Call(t.page)
	if err != nil {
		return nil, err
	}

	// Step 4: Parse JSON result
	var attrs map[string]string
	resultBytes, _ := json.Marshal(result.Result.Value)
	if err := json.Unmarshal(resultBytes, &attrs); err != nil {
		return nil, err
	}

	return attrs, nil
}

// batchExtractAttributes extracts attributes in parallel for all elements
func (t *tabContext) batchExtractAttributes(backendIDs []int64) map[int64]map[string]string {
	results := make(map[int64]map[string]string)
	var mu sync.Mutex
	var wg sync.WaitGroup

	// Extract attributes concurrently (limited parallelism to avoid overwhelming CDP)
	sem := make(chan struct{}, 10) // Max 10 concurrent extractions

	for _, backendID := range backendIDs {
		wg.Add(1)
		go func(bid int64) {
			defer wg.Done()
			sem <- struct{}{}        // Acquire semaphore
			defer func() { <-sem }() // Release semaphore

			attrs, err := t.extractElementAttributes(bid)
			if err == nil && attrs != nil && len(attrs) > 0 {
				mu.Lock()
				results[bid] = attrs
				mu.Unlock()
			}
		}(backendID)
	}

	wg.Wait()
	return results
}

// extractElements extracts interactive elements from the accessibility tree (tab-level method)
func (t *tabContext) extractElements() ([]protocol.RawAccessibilityNode, error) {
	// Get accessibility tree
	tree, err := proto.AccessibilityGetFullAXTree{}.Call(t.page)
	if err != nil {
		return nil, errors.NewBrowserError("get_accessibility_tree", err)
	}

	elements := make([]protocol.RawAccessibilityNode, 0)
	t.elements = make([]elementInfo, 0)

	for _, node := range tree.Nodes {
		// Filter for interactive elements
		if !isInteractive(node) {
			continue
		}

		// Get element bounds
		bounds, err := t.getNodeBounds(node)
		if err != nil || bounds == nil {
			continue
		}

		// Skip elements with zero size
		if bounds.Width <= 0 || bounds.Height <= 0 {
			continue
		}

		name := getNodeName(node)
		role := getNodeRole(node)

		elem := protocol.RawAccessibilityNode{
			Role:      role,
			Name:      name,
			Bounds:    *bounds,
			BackendID: int64(node.BackendDOMNodeID),
		}
		elements = append(elements, elem)

		// Store for later interaction
		t.elements = append(t.elements, elementInfo{
			Role:      role,
			Name:      name,
			Bounds:    *bounds,
			BackendID: int64(node.BackendDOMNodeID),
		})
	}

	return elements, nil
}

// isInteractive checks if an accessibility node is interactive
func isInteractive(node *proto.AccessibilityAXNode) bool {
	if node.Role == nil {
		return false
	}

	role := getNodeRole(node)
	return interactiveRoles[role]
}

// getNodeRole extracts the role from a node
func getNodeRole(node *proto.AccessibilityAXNode) string {
	if node.Role == nil {
		return ""
	}
	// Try to extract string from Value (which is a gson.JSON)
	return node.Role.Value.String()
}

// getNodeName extracts the accessible name from a node
func getNodeName(node *proto.AccessibilityAXNode) string {
	if node.Name == nil {
		return ""
	}
	// Try to extract string from Value (which is a gson.JSON)
	return node.Name.Value.String()
}

// getNodeBounds gets the bounding box for an accessibility node
func (t *tabContext) getNodeBounds(node *proto.AccessibilityAXNode) (*protocol.BoundingBox, error) {
	if node.BackendDOMNodeID == 0 {
		return nil, errors.NewBrowserError("get_node_bounds", fmt.Errorf("node has no backend DOM node ID"))
	}

	// Get box model for the DOM node
	boxModel, err := proto.DOMGetBoxModel{
		BackendNodeID: node.BackendDOMNodeID,
	}.Call(t.page)

	if err != nil {
		return nil, err
	}

	if boxModel == nil || boxModel.Model == nil {
		return nil, errors.NewBrowserError("get_node_bounds", fmt.Errorf("box model is nil"))
	}

	content := boxModel.Model.Content
	if len(content) < 8 {
		return nil, errors.NewBrowserError("get_node_bounds", fmt.Errorf("invalid box model content"))
	}

	// Content is [x1, y1, x2, y2, x3, y3, x4, y4] - a quad
	x := content[0]
	y := content[1]
	width := content[2] - content[0]
	height := content[5] - content[1]

	return &protocol.BoundingBox{
		X:      x,
		Y:      y,
		Width:  width,
		Height: height,
	}, nil
}

// getViewportBounds returns the visible viewport dimensions and scroll position
func (t *tabContext) getViewportBounds() (*protocol.BoundingBox, error) {
	// Use CDP Runtime.evaluate directly for more control
	result, err := proto.RuntimeEvaluate{
		Expression:            `({x: window.scrollX || window.pageXOffset, y: window.scrollY || window.pageYOffset, width: window.innerWidth, height: window.innerHeight})`,
		ReturnByValue:         true,
		AwaitPromise:          false,
		UserGesture:           false,
		IncludeCommandLineAPI: false,
	}.Call(t.page)

	if err != nil {
		return nil, fmt.Errorf("failed to evaluate viewport script: %w", err)
	}

	if result.ExceptionDetails != nil {
		return nil, fmt.Errorf("viewport script exception: %s", result.ExceptionDetails.Text)
	}

	// Parse JSON result into BoundingBox struct
	var viewport struct {
		X      float64 `json:"x"`
		Y      float64 `json:"y"`
		Width  float64 `json:"width"`
		Height float64 `json:"height"`
	}

	// Marshal the result.Value first, then unmarshal
	resultBytes, err := json.Marshal(result.Result.Value)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal viewport bounds: %w", err)
	}

	if err := json.Unmarshal(resultBytes, &viewport); err != nil {
		return nil, fmt.Errorf("failed to parse viewport bounds: %w", err)
	}

	return &protocol.BoundingBox{
		X:      viewport.X,
		Y:      viewport.Y,
		Width:  viewport.Width,
		Height: viewport.Height,
	}, nil
}

// isInViewport checks if element bounds intersect with viewport bounds
func isInViewport(elem, viewport *protocol.BoundingBox) bool {
	// Element is visible if it overlaps with viewport
	// Use standard rectangle intersection test
	return !(elem.X+elem.Width < viewport.X ||
		elem.X > viewport.X+viewport.Width ||
		elem.Y+elem.Height < viewport.Y ||
		elem.Y > viewport.Y+viewport.Height)
}

// Click clicks an element by index
func (c *Context) Click(ctx context.Context, index int, tabID *string) error {
	tab, err := c.getTab(tabID)
	if err != nil {
		return err
	}

	tab.mu.Lock()
	defer tab.mu.Unlock()

	// Lazy load elements if cache is empty
	if len(tab.elements) == 0 {
		_, err := tab.extractElements()
		if err != nil {
			return fmt.Errorf("failed to auto-extract elements for click: %w", err)
		}
	}

	if index < 0 || index >= len(tab.elements) {
		return errors.NewElementError(index, "click", errors.NewValidationError("index", index, nil))
	}

	elem := tab.elements[index]

	// Click at center of element
	x := elem.Bounds.X + elem.Bounds.Width/2
	y := elem.Bounds.Y + elem.Bounds.Height/2

	// Move to position and click
	if err := tab.page.Mouse.MoveTo(proto.Point{X: x, Y: y}); err != nil {
		return errors.NewElementError(index, "click", err)
	}

	if err := tab.page.Mouse.Click(proto.InputMouseButtonLeft, 1); err != nil {
		return errors.NewElementError(index, "click", err)
	}

	return nil
}

// RightClick right-clicks an element at its center
func (c *Context) RightClick(ctx context.Context, index int, tabID *string) error {
	tab, err := c.getTab(tabID)
	if err != nil {
		return err
	}

	tab.mu.Lock()
	defer tab.mu.Unlock()

	// Lazy load elements if cache is empty
	if len(tab.elements) == 0 {
		_, err := tab.extractElements()
		if err != nil {
			return fmt.Errorf("failed to auto-extract elements for rightClick: %w", err)
		}
	}

	if index < 0 || index >= len(tab.elements) {
		return errors.NewElementError(index, "rightClick", errors.NewValidationError("index", index, nil))
	}

	elem := tab.elements[index]

	// Right-click at center of element
	x := elem.Bounds.X + elem.Bounds.Width/2
	y := elem.Bounds.Y + elem.Bounds.Height/2

	// Move to position and right-click
	if err := tab.page.Mouse.MoveTo(proto.Point{X: x, Y: y}); err != nil {
		return errors.NewElementError(index, "rightClick", err)
	}

	if err := tab.page.Mouse.Click(proto.InputMouseButtonRight, 1); err != nil {
		return errors.NewElementError(index, "rightClick", err)
	}

	return nil
}

// DoubleClick double-clicks an element at its center
func (c *Context) DoubleClick(ctx context.Context, index int, tabID *string) error {
	tab, err := c.getTab(tabID)
	if err != nil {
		return err
	}

	tab.mu.Lock()
	defer tab.mu.Unlock()

	// Lazy load elements if cache is empty
	if len(tab.elements) == 0 {
		_, err := tab.extractElements()
		if err != nil {
			return fmt.Errorf("failed to auto-extract elements for doubleClick: %w", err)
		}
	}

	if index < 0 || index >= len(tab.elements) {
		return errors.NewElementError(index, "doubleClick", errors.NewValidationError("index", index, nil))
	}

	elem := tab.elements[index]

	// Double-click at center of element
	x := elem.Bounds.X + elem.Bounds.Width/2
	y := elem.Bounds.Y + elem.Bounds.Height/2

	// Move to position and double-click
	if err := tab.page.Mouse.MoveTo(proto.Point{X: x, Y: y}); err != nil {
		return errors.NewElementError(index, "doubleClick", err)
	}

	if err := tab.page.Mouse.Click(proto.InputMouseButtonLeft, 2); err != nil {
		return errors.NewElementError(index, "doubleClick", err)
	}

	return nil
}

// TripleClick triple-clicks an element at its center
func (c *Context) TripleClick(ctx context.Context, index int, tabID *string) error {
	tab, err := c.getTab(tabID)
	if err != nil {
		return err
	}

	tab.mu.Lock()
	defer tab.mu.Unlock()

	// Lazy load elements if cache is empty
	if len(tab.elements) == 0 {
		_, err := tab.extractElements()
		if err != nil {
			return fmt.Errorf("failed to auto-extract elements for tripleClick: %w", err)
		}
	}

	if index < 0 || index >= len(tab.elements) {
		return errors.NewElementError(index, "tripleClick", errors.NewValidationError("index", index, nil))
	}

	elem := tab.elements[index]

	// Triple-click at center of element
	x := elem.Bounds.X + elem.Bounds.Width/2
	y := elem.Bounds.Y + elem.Bounds.Height/2

	// Move to position and triple-click
	if err := tab.page.Mouse.MoveTo(proto.Point{X: x, Y: y}); err != nil {
		return errors.NewElementError(index, "tripleClick", err)
	}

	if err := tab.page.Mouse.Click(proto.InputMouseButtonLeft, 3); err != nil {
		return errors.NewElementError(index, "tripleClick", err)
	}

	return nil
}

// ClickByBackendID clicks an element by its CDP backend node ID
func (c *Context) ClickByBackendID(ctx context.Context, backendID int64, tabID *string) error {
	tab, err := c.getTab(tabID)
	if err != nil {
		return err
	}

	tab.mu.Lock()
	defer tab.mu.Unlock()

	// Search for element with matching BackendID in cached elements
	var elem *elementInfo
	for i := range tab.elements {
		if tab.elements[i].BackendID == backendID {
			elem = &tab.elements[i]
			break
		}
	}

	if elem == nil {
		return fmt.Errorf("element with backendID %d not found in cache", backendID)
	}

	// Click at center of element
	x := elem.Bounds.X + elem.Bounds.Width/2
	y := elem.Bounds.Y + elem.Bounds.Height/2

	// Move to position and click
	if err := tab.page.Mouse.MoveTo(proto.Point{X: x, Y: y}); err != nil {
		return fmt.Errorf("failed to move mouse for backendID %d: %w", backendID, err)
	}

	if err := tab.page.Mouse.Click(proto.InputMouseButtonLeft, 1); err != nil {
		return fmt.Errorf("failed to click backendID %d: %w", backendID, err)
	}

	return nil
}

// RightClickByBackendID right-clicks an element by its CDP backend node ID
func (c *Context) RightClickByBackendID(ctx context.Context, backendID int64, tabID *string) error {
	tab, err := c.getTab(tabID)
	if err != nil {
		return err
	}

	tab.mu.Lock()
	defer tab.mu.Unlock()

	// Search for element with matching BackendID in cached elements
	var elem *elementInfo
	for i := range tab.elements {
		if tab.elements[i].BackendID == backendID {
			elem = &tab.elements[i]
			break
		}
	}

	if elem == nil {
		return fmt.Errorf("element with backendID %d not found in cache", backendID)
	}

	// Right-click at center of element
	x := elem.Bounds.X + elem.Bounds.Width/2
	y := elem.Bounds.Y + elem.Bounds.Height/2

	// Move to position and right-click
	if err := tab.page.Mouse.MoveTo(proto.Point{X: x, Y: y}); err != nil {
		return fmt.Errorf("failed to move mouse for backendID %d: %w", backendID, err)
	}

	if err := tab.page.Mouse.Click(proto.InputMouseButtonRight, 1); err != nil {
		return fmt.Errorf("failed to right-click backendID %d: %w", backendID, err)
	}

	return nil
}

// DoubleClickByBackendID double-clicks an element by its CDP backend node ID
func (c *Context) DoubleClickByBackendID(ctx context.Context, backendID int64, tabID *string) error {
	tab, err := c.getTab(tabID)
	if err != nil {
		return err
	}

	tab.mu.Lock()
	defer tab.mu.Unlock()

	// Search for element with matching BackendID in cached elements
	var elem *elementInfo
	for i := range tab.elements {
		if tab.elements[i].BackendID == backendID {
			elem = &tab.elements[i]
			break
		}
	}

	if elem == nil {
		return fmt.Errorf("element with backendID %d not found in cache", backendID)
	}

	// Double-click at center of element
	x := elem.Bounds.X + elem.Bounds.Width/2
	y := elem.Bounds.Y + elem.Bounds.Height/2

	// Move to position and double-click
	if err := tab.page.Mouse.MoveTo(proto.Point{X: x, Y: y}); err != nil {
		return fmt.Errorf("failed to move mouse for backendID %d: %w", backendID, err)
	}

	if err := tab.page.Mouse.Click(proto.InputMouseButtonLeft, 2); err != nil {
		return fmt.Errorf("failed to double-click backendID %d: %w", backendID, err)
	}

	return nil
}

// TripleClickByBackendID triple-clicks an element by its CDP backend node ID
func (c *Context) TripleClickByBackendID(ctx context.Context, backendID int64, tabID *string) error {
	tab, err := c.getTab(tabID)
	if err != nil {
		return err
	}

	tab.mu.Lock()
	defer tab.mu.Unlock()

	// Search for element with matching BackendID in cached elements
	var elem *elementInfo
	for i := range tab.elements {
		if tab.elements[i].BackendID == backendID {
			elem = &tab.elements[i]
			break
		}
	}

	if elem == nil {
		return fmt.Errorf("element with backendID %d not found in cache", backendID)
	}

	// Triple-click at center of element
	x := elem.Bounds.X + elem.Bounds.Width/2
	y := elem.Bounds.Y + elem.Bounds.Height/2

	// Move to position and triple-click
	if err := tab.page.Mouse.MoveTo(proto.Point{X: x, Y: y}); err != nil {
		return fmt.Errorf("failed to move mouse for backendID %d: %w", backendID, err)
	}

	if err := tab.page.Mouse.Click(proto.InputMouseButtonLeft, 3); err != nil {
		return fmt.Errorf("failed to triple-click backendID %d: %w", backendID, err)
	}

	return nil
}

// Drag performs a drag operation either by index or coordinates
func (c *Context) Drag(ctx context.Context, fromIndex, toIndex *int, fromX, fromY, toX, toY *float64, duration *int, tabID *string) error {
	tab, err := c.getTab(tabID)
	if err != nil {
		return err
	}

	tab.mu.Lock()
	defer tab.mu.Unlock()

	// Validate: must have either (fromIndex AND toIndex) OR (fromX AND fromY AND toX AND toY)
	hasIndexMode := fromIndex != nil && toIndex != nil
	hasCoordMode := fromX != nil && fromY != nil && toX != nil && toY != nil

	if !hasIndexMode && !hasCoordMode {
		return errors.NewValidationError("drag", "parameters",
			fmt.Errorf("must provide either (fromIndex and toIndex) or (fromX, fromY, toX, toY)"))
	}

	if hasIndexMode && hasCoordMode {
		return errors.NewValidationError("drag", "parameters",
			fmt.Errorf("cannot mix index mode and coordinate mode"))
	}

	var startX, startY, endX, endY float64

	// Index-based mode
	if hasIndexMode {
		// Lazy load elements if cache is empty
		if len(tab.elements) == 0 {
			_, err := tab.extractElements()
			if err != nil {
				return fmt.Errorf("failed to auto-extract elements for drag: %w", err)
			}
		}

		// Validate indices
		if *fromIndex < 0 || *fromIndex >= len(tab.elements) {
			return errors.NewElementError(*fromIndex, "drag", errors.NewValidationError("fromIndex", *fromIndex, nil))
		}
		if *toIndex < 0 || *toIndex >= len(tab.elements) {
			return errors.NewElementError(*toIndex, "drag", errors.NewValidationError("toIndex", *toIndex, nil))
		}

		fromElem := tab.elements[*fromIndex]
		toElem := tab.elements[*toIndex]

		// Calculate center points
		startX = fromElem.Bounds.X + fromElem.Bounds.Width/2
		startY = fromElem.Bounds.Y + fromElem.Bounds.Height/2
		endX = toElem.Bounds.X + toElem.Bounds.Width/2
		endY = toElem.Bounds.Y + toElem.Bounds.Height/2
	} else {
		// Coordinate-based mode
		startX = *fromX
		startY = *fromY
		endX = *toX
		endY = *toY
	}

	// Default duration: 500ms
	dragDuration := 500
	if duration != nil && *duration > 0 {
		dragDuration = *duration
	}

	// Perform drag using CDP mouse events
	// 1. Move to start position
	if err := tab.page.Mouse.MoveTo(proto.Point{X: startX, Y: startY}); err != nil {
		return errors.NewBrowserError("drag", fmt.Errorf("failed to move to start position: %w", err))
	}

	// 2. Mouse down at start
	if err := tab.page.Mouse.Down(proto.InputMouseButtonLeft, 1); err != nil {
		return errors.NewBrowserError("drag", fmt.Errorf("failed to press mouse down: %w", err))
	}

	// 3. Move to end position (with small delay to simulate dragging)
	time.Sleep(time.Duration(dragDuration) * time.Millisecond)
	if err := tab.page.Mouse.MoveTo(proto.Point{X: endX, Y: endY}); err != nil {
		return errors.NewBrowserError("drag", fmt.Errorf("failed to move to end position: %w", err))
	}

	// 4. Mouse up at end
	if err := tab.page.Mouse.Up(proto.InputMouseButtonLeft, 1); err != nil {
		return errors.NewBrowserError("drag", fmt.Errorf("failed to release mouse: %w", err))
	}

	return nil
}

// FormInput sets form input value directly via JavaScript (for React/Vue apps)
func (c *Context) FormInput(ctx context.Context, index int, value string, tabID *string) error {
	tab, err := c.getTab(tabID)
	if err != nil {
		return err
	}

	tab.mu.Lock()
	defer tab.mu.Unlock()

	// Lazy load elements if cache is empty
	if len(tab.elements) == 0 {
		_, err := tab.extractElements()
		if err != nil {
			return fmt.Errorf("failed to auto-extract elements for formInput: %w", err)
		}
	}

	if index < 0 || index >= len(tab.elements) {
		return errors.NewElementError(index, "formInput", errors.NewValidationError("index", index, nil))
	}

	elem := tab.elements[index]

	// Validate element is an input field
	if elem.Role != constants.RoleTextbox && elem.Role != constants.RoleSearchbox && elem.Role != constants.RoleCombobox {
		return errors.NewElementError(index, "formInput",
			fmt.Errorf("element must be textbox, searchbox, or combobox, got: %s", elem.Role))
	}

	// Resolve BackendNodeID to ObjectID
	resolveResp, err := proto.DOMResolveNode{BackendNodeID: proto.DOMBackendNodeID(elem.BackendID)}.Call(tab.page)
	if err != nil {
		return errors.NewElementError(index, "formInput", fmt.Errorf("failed to resolve node: %w", err))
	}

	// Set value and dispatch events via JavaScript
	script := fmt.Sprintf(`
		this.value = %s;
		this.dispatchEvent(new Event('input', { bubbles: true }));
		this.dispatchEvent(new Event('change', { bubbles: true }));
	`, strconv.Quote(value))

	_, err = proto.RuntimeCallFunctionOn{
		FunctionDeclaration: script,
		ObjectID:            resolveResp.Object.ObjectID,
	}.Call(tab.page)
	if err != nil {
		return errors.NewElementError(index, "formInput", fmt.Errorf("failed to set value: %w", err))
	}

	return nil
}

// GoBack navigates backward in browser history
func (c *Context) GoBack(ctx context.Context, tabID *string) (string, error) {
	tab, err := c.getTab(tabID)
	if err != nil {
		return "", err
	}

	tab.mu.Lock()
	defer tab.mu.Unlock()

	// Clear element cache before navigation
	tab.elements = nil

	if err := tab.page.NavigateBack(); err != nil {
		return "", fmt.Errorf("failed to navigate back: %w", err)
	}

	if err := tab.page.WaitLoad(); err != nil {
		return "", fmt.Errorf("failed to wait for page load after going back: %w", err)
	}

	info, err := tab.page.Info()
	if err != nil {
		return "", fmt.Errorf("failed to get page info: %w", err)
	}

	// Update cached URL and title
	tab.currentURL = info.URL
	tab.currentTitle = info.Title

	return info.URL, nil
}

// GoForward navigates forward in browser history
func (c *Context) GoForward(ctx context.Context, tabID *string) (string, error) {
	tab, err := c.getTab(tabID)
	if err != nil {
		return "", err
	}

	tab.mu.Lock()
	defer tab.mu.Unlock()

	// Clear element cache before navigation
	tab.elements = nil

	if err := tab.page.NavigateForward(); err != nil {
		return "", fmt.Errorf("failed to navigate forward: %w", err)
	}

	if err := tab.page.WaitLoad(); err != nil {
		return "", fmt.Errorf("failed to wait for page load after going forward: %w", err)
	}

	info, err := tab.page.Info()
	if err != nil {
		return "", fmt.Errorf("failed to get page info: %w", err)
	}

	// Update cached URL and title
	tab.currentURL = info.URL
	tab.currentTitle = info.Title

	return info.URL, nil
}

// Wait pauses execution for the specified duration in milliseconds
func (c *Context) Wait(ctx context.Context, duration int, tabID *string) error {
	// Validate tab exists if tabID provided
	if tabID != nil {
		_, err := c.getTab(tabID)
		if err != nil {
			return err
		}
	}

	// Sleep for specified duration
	time.Sleep(time.Duration(duration) * time.Millisecond)
	return nil
}

// Type types text into an element
func (c *Context) Type(ctx context.Context, index int, text string, tabID *string) error {
	tab, err := c.getTab(tabID)
	if err != nil {
		return err
	}

	tab.mu.Lock()
	defer tab.mu.Unlock()

	// Lazy load elements if cache is empty
	if len(tab.elements) == 0 {
		_, err := tab.extractElements()
		if err != nil {
			return fmt.Errorf("failed to auto-extract elements for type: %w", err)
		}
	}

	if index < 0 || index >= len(tab.elements) {
		return errors.NewElementError(index, "type", errors.NewValidationError("index", index, nil))
	}

	elem := tab.elements[index]

	// Click to focus
	x := elem.Bounds.X + elem.Bounds.Width/2
	y := elem.Bounds.Y + elem.Bounds.Height/2

	if err := tab.page.Mouse.MoveTo(proto.Point{X: x, Y: y}); err != nil {
		return errors.NewElementError(index, "type", err)
	}

	if err := tab.page.Mouse.Click(proto.InputMouseButtonLeft, 1); err != nil {
		return errors.NewElementError(index, "type", err)
	}

	// Type text character by character
	for _, ch := range text {
		key := input.Key(ch)
		if err := tab.page.Keyboard.Type(key); err != nil {
			return errors.NewElementError(index, "type", err)
		}
	}

	return nil
}

// Scroll scrolls the page
func (c *Context) Scroll(ctx context.Context, direction string, amount int, tabID *string) error {
	tab, err := c.getTab(tabID)
	if err != nil {
		return err
	}

	tab.mu.Lock()
	defer tab.mu.Unlock()

	var deltaX, deltaY float64

	switch direction {
	case constants.ScrollUp:
		deltaY = -float64(amount)
	case constants.ScrollDown:
		deltaY = float64(amount)
	case constants.ScrollLeft:
		deltaX = -float64(amount)
	case constants.ScrollRight:
		deltaX = float64(amount)
	default:
		return errors.NewValidationError("direction", direction, nil)
	}

	if err := tab.page.Mouse.Scroll(deltaX, deltaY, 1); err != nil {
		return errors.NewBrowserError("scroll", err)
	}

	return nil
}

// UploadFile uploads files to a file input element
func (c *Context) UploadFile(ctx context.Context, index int, filePaths []string, tabID *string) (*protocol.UploadFileResult, error) {
	tab, err := c.getTab(tabID)
	if err != nil {
		return nil, err
	}

	tab.mu.Lock()
	defer tab.mu.Unlock()

	// Lazy load elements if cache is empty
	if len(tab.elements) == 0 {
		_, err := tab.extractElements()
		if err != nil {
			return nil, fmt.Errorf("failed to auto-extract elements for upload: %w", err)
		}
	}

	// Validate index
	if index < 0 || index >= len(tab.elements) {
		return nil, errors.NewElementError(index, "upload", errors.NewValidationError("index", index, nil))
	}

	elem := tab.elements[index]

	// Validate all files exist
	fileNames := make([]string, 0, len(filePaths))
	for _, path := range filePaths {
		// Check file exists
		info, err := os.Stat(path)
		if err != nil {
			return nil, errors.NewFileError(path, "stat", err)
		}

		// Ensure it's not a directory
		if info.IsDir() {
			return nil, errors.NewFileError(path, "upload", fmt.Errorf("path is a directory"))
		}

		fileNames = append(fileNames, filepath.Base(path))
	}

	// Resolve DOM node from backend node ID
	nodeResult, err := proto.DOMResolveNode{
		BackendNodeID: proto.DOMBackendNodeID(elem.BackendID),
	}.Call(tab.page)
	if err != nil {
		return nil, errors.NewElementError(index, "upload", fmt.Errorf("failed to resolve node: %w", err))
	}

	if nodeResult.Object.ObjectID == "" {
		return nil, errors.NewElementError(index, "upload", fmt.Errorf("no object ID for element"))
	}

	// Verify it's a file input using JavaScript
	jsCheck := fmt.Sprintf(`(function() {
		const elem = arguments[0];
		return {
			tagName: elem.tagName.toLowerCase(),
			type: elem.type || ''
		};
	})`)

	checkResult, err := proto.RuntimeCallFunctionOn{
		ObjectID:  nodeResult.Object.ObjectID,
		FunctionDeclaration: jsCheck,
	}.Call(tab.page)
	if err != nil {
		return nil, errors.NewElementError(index, "upload", fmt.Errorf("failed to check element: %w", err))
	}

	var elemInfo struct {
		TagName string `json:"tagName"`
		Type    string `json:"type"`
	}

	resultBytes, _ := json.Marshal(checkResult.Result.Value)
	if err := json.Unmarshal(resultBytes, &elemInfo); err != nil {
		return nil, errors.NewElementError(index, "upload", fmt.Errorf("failed to parse element info: %w", err))
	}

	if elemInfo.TagName != "input" {
		return nil, errors.NewElementError(index, "upload", fmt.Errorf("element is not an input (got %s)", elemInfo.TagName))
	}

	if elemInfo.Type != "file" {
		return nil, errors.NewElementError(index, "upload", fmt.Errorf("element is not a file input (got type %s)", elemInfo.Type))
	}

	// Upload files using CDP SetFileInputFiles
	backendNodeID := proto.DOMBackendNodeID(elem.BackendID)
	err = proto.DOMSetFileInputFiles{
		Files:         filePaths,
		BackendNodeID: backendNodeID,
	}.Call(tab.page)
	if err != nil {
		return nil, errors.NewElementError(index, "upload", err)
	}

	return &protocol.UploadFileResult{
		FilesUploaded: len(filePaths),
		FileNames:     fileNames,
	}, nil
}

// GetText extracts text content from the page
func (c *Context) GetText(ctx context.Context, strategy string, tabID *string) (*protocol.GetTextResult, error) {
	tab, err := c.getTab(tabID)
	if err != nil {
		return nil, err
	}

	tab.mu.RLock()
	defer tab.mu.RUnlock()

	// Default to auto strategy
	if strategy == "" {
		strategy = constants.TextStrategyAuto
	}

	// Build JavaScript based on strategy
	var js string
	switch strategy {
	case constants.TextStrategyAuto:
		js = `() => {
			let elem = document.querySelector('article');
			if (elem) return { text: elem.innerText, source: 'article' };

			elem = document.querySelector('main, [role="main"]');
			if (elem) return { text: elem.innerText, source: 'main' };

			return { text: document.body.innerText, source: 'body' };
		}`
	case constants.TextStrategyArticle:
		js = `() => {
			const elem = document.querySelector('article');
			if (!elem) return { text: '', source: 'article' };
			return { text: elem.innerText, source: 'article' };
		}`
	case constants.TextStrategyMain:
		js = `() => {
			const elem = document.querySelector('main, [role="main"]');
			if (!elem) return { text: '', source: 'main' };
			return { text: elem.innerText, source: 'main' };
		}`
	case constants.TextStrategyBody:
		js = `() => {
			return { text: document.body.innerText, source: 'body' };
		}`
	default:
		return nil, errors.NewValidationError("strategy", strategy, fmt.Errorf("invalid strategy"))
	}

	// Execute JavaScript
	result, err := tab.page.Eval(js)
	if err != nil {
		return nil, errors.NewBrowserError("get_text", err)
	}

	// Parse result
	var extracted struct {
		Text   string `json:"text"`
		Source string `json:"source"`
	}

	resultBytes, err := json.Marshal(result.Value)
	if err != nil {
		return nil, errors.NewBrowserError("get_text", fmt.Errorf("failed to marshal result: %w", err))
	}

	if err := json.Unmarshal(resultBytes, &extracted); err != nil {
		return nil, errors.NewBrowserError("get_text", fmt.Errorf("failed to parse result: %w", err))
	}

	text := extracted.Text
	truncated := false

	// Check size limit
	if len(text) > constants.MaxTextSize {
		text = text[:constants.MaxTextSize]
		truncated = true
	}

	return &protocol.GetTextResult{
		Text:      text,
		Length:    len(text),
		Source:    extracted.Source,
		Truncated: truncated,
	}, nil
}

// Find searches for elements matching a keyword query
func (c *Context) Find(ctx context.Context, query string, limit int, tabID *string) (*protocol.FindResult, error) {
	tab, err := c.getTab(tabID)
	if err != nil {
		return nil, err
	}

	tab.mu.Lock()
	defer tab.mu.Unlock()

	if query == "" {
		return nil, errors.NewValidationError("query", query, fmt.Errorf("query cannot be empty"))
	}

	// Default limit
	if limit <= 0 {
		limit = constants.DefaultFindLimit
	}

	// Get full accessibility tree (no filtering)
	tree, err := proto.AccessibilityGetFullAXTree{}.Call(tab.page)
	if err != nil {
		return nil, errors.NewBrowserError("find", err)
	}

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
		element protocol.RawAccessibilityNode
		score   int
	}

	matches := make([]match, 0)

	for _, node := range tree.Nodes {
		role := getNodeRole(node)
		name := getNodeName(node)
		nameLower := strings.ToLower(name)

		// Filter by role if specified
		if targetRole != "" && role != targetRole {
			continue
		}

		// Score based on keyword matching
		score := 0
		for _, keyword := range keywords {
			if keyword == "" {
				continue
			}

			// Exact match
			if nameLower == keyword {
				score += 10
			} else if strings.HasPrefix(nameLower, keyword) {
				score += 5
			} else if strings.Contains(nameLower, keyword) {
				score += 1
			}
		}

		// If no keywords, match all elements with the role (or all if no role)
		if len(keywords) == 0 && name != "" {
			score = 1
		}

		if score == 0 {
			continue
		}

		// Get bounds
		bounds, err := tab.getNodeBounds(node)
		if err != nil || bounds == nil {
			continue
		}

		matches = append(matches, match{
			element: protocol.RawAccessibilityNode{
				Role:      role,
				Name:      name,
				Bounds:    *bounds,
				BackendID: int64(node.BackendDOMNodeID),
			},
			score: score,
		})
	}

	// Sort by score (descending), then Y position, then X position
	sort.Slice(matches, func(i, j int) bool {
		if matches[i].score != matches[j].score {
			return matches[i].score > matches[j].score
		}
		if matches[i].element.Bounds.Y != matches[j].element.Bounds.Y {
			return matches[i].element.Bounds.Y < matches[j].element.Bounds.Y
		}
		return matches[i].element.Bounds.X < matches[j].element.Bounds.X
	})

	// Extract elements from matches (no re-indexing)
	elements := make([]protocol.RawAccessibilityNode, 0, len(matches))
	for _, m := range matches {
		elements = append(elements, m.element)
	}

	total := len(elements)
	truncated := false

	// Apply limit
	if total > limit {
		elements = elements[:limit]
		truncated = true
	}

	return &protocol.FindResult{
		Elements:  elements,
		Total:     total,
		Truncated: truncated,
	}, nil
}

// SetUserAgent sets the user agent for this context
func (c *Context) SetUserAgent(ctx context.Context, ua string, tabID *string) error {
	tab, err := c.getTab(tabID)
	if err != nil {
		return err
	}

	tab.mu.Lock()
	defer tab.mu.Unlock()

	err = proto.NetworkSetUserAgentOverride{
		UserAgent: ua,
	}.Call(tab.page)

	if err != nil {
		return errors.NewBrowserError("set_user_agent", err)
	}

	return nil
}

// ImportCookies imports cookies from a browser (stub - implemented in cookies package)
func (c *Context) ImportCookies(ctx context.Context, browser, profile string, tabID *string) (*protocol.ImportCookiesResult, error) {
	// TODO: Implement in Phase 4 using cookies package
	return &protocol.ImportCookiesResult{
		Imported: 0,
		Domains:  []string{},
	}, errors.NewNotImplementedError("import_cookies")
}

// parseKeyName converts string key names to go-rod input.Key constants
func parseKeyName(name string) (input.Key, error) {
	switch strings.ToLower(name) {
	case "enter":
		return input.Enter, nil
	case "escape", "esc":
		return input.Escape, nil
	case "backspace":
		return input.Backspace, nil
	case "delete", "del":
		return input.Delete, nil
	case "tab":
		return input.Tab, nil
	case "arrowup", "arrow_up":
		return input.ArrowUp, nil
	case "arrowdown", "arrow_down":
		return input.ArrowDown, nil
	case "arrowleft", "arrow_left":
		return input.ArrowLeft, nil
	case "arrowright", "arrow_right":
		return input.ArrowRight, nil
	case "space":
		return input.Space, nil
	case "home":
		return input.Home, nil
	case "end":
		return input.End, nil
	case "pageup":
		return input.PageUp, nil
	case "pagedown":
		return input.PageDown, nil
	default:
		return 0, fmt.Errorf("unknown key: %s", name)
	}
}

// parseModifier converts modifier names to input.Key constants
func parseModifier(name string) (input.Key, error) {
	switch strings.ToLower(name) {
	case "cmd", "meta":
		return input.MetaLeft, nil
	case "ctrl", "control":
		return input.ControlLeft, nil
	case "alt":
		return input.AltLeft, nil
	case "shift":
		return input.ShiftLeft, nil
	default:
		return 0, fmt.Errorf("unknown modifier: %s", name)
	}
}

// PressKey presses keyboard keys or key combinations
func (c *Context) PressKey(ctx context.Context, keys string, tabID *string) error {
	tab, err := c.getTab(tabID)
	if err != nil {
		return err
	}

	tab.mu.Lock()
	defer tab.mu.Unlock()

	// Parse space-separated key sequence
	keyTokens := strings.Split(keys, " ")

	for _, token := range keyTokens {
		if strings.Contains(token, "+") {
			// Key combination (e.g., "cmd+a")
			parts := strings.Split(token, "+")
			if len(parts) != 2 {
				return errors.NewValidationError("key", token, fmt.Errorf("invalid key combination: %s", token))
			}

			modifier, err := parseModifier(parts[0])
			if err != nil {
				return errors.NewValidationError("key", token, err)
			}

			key, err := parseKeyName(parts[1])
			if err != nil {
				return errors.NewValidationError("key", token, err)
			}

			// Press modifier, type key, release modifier
			if err := tab.page.Keyboard.Press(modifier); err != nil {
				return errors.NewBrowserError("press_key", fmt.Errorf("failed to press modifier: %w", err))
			}
			if err := tab.page.Keyboard.Type(key); err != nil {
				return errors.NewBrowserError("press_key", fmt.Errorf("failed to type key: %w", err))
			}
			if err := tab.page.Keyboard.Release(modifier); err != nil {
				return errors.NewBrowserError("press_key", fmt.Errorf("failed to release modifier: %w", err))
			}
		} else {
			// Single key press
			key, err := parseKeyName(token)
			if err != nil {
				return errors.NewValidationError("key", token, err)
			}

			if err := tab.page.Keyboard.Type(key); err != nil {
				return errors.NewBrowserError("press_key", fmt.Errorf("failed to type key: %w", err))
			}
		}
	}

	return nil
}

// ScrollIntoView scrolls an element into the visible viewport
func (c *Context) ScrollIntoView(ctx context.Context, index *int, backendID *int64, tabID *string) error {
	tab, err := c.getTab(tabID)
	if err != nil {
		return err
	}

	tab.mu.Lock()
	defer tab.mu.Unlock()

	var nodeBackendID int64

	if backendID != nil {
		nodeBackendID = *backendID
	} else if index != nil {
		// Lazy load elements if needed
		if len(tab.elements) == 0 {
			_, err := tab.extractElements()
			if err != nil {
				return errors.NewBrowserError("scroll_into_view", fmt.Errorf("failed to extract elements: %w", err))
			}
		}

		if *index < 0 || *index >= len(tab.elements) {
			return errors.NewElementError(*index, "scroll_into_view", fmt.Errorf("index out of range"))
		}

		nodeBackendID = tab.elements[*index].BackendID
	} else {
		return errors.NewValidationError("scroll_into_view", "index or backendId", fmt.Errorf("either index or backendId must be provided"))
	}

	// Use CDP DOM.scrollIntoViewIfNeeded
	err = proto.DOMScrollIntoViewIfNeeded{
		BackendNodeID: proto.DOMBackendNodeID(nodeBackendID),
	}.Call(tab.page)

	if err != nil {
		return errors.NewBrowserError("scroll_into_view", fmt.Errorf("scrollIntoView failed: %w", err))
	}

	return nil
}

// GetCookies retrieves all cookies from the browser
func (c *Context) GetCookies(ctx context.Context, tabID *string) (*protocol.GetCookiesResult, error) {
	tab, err := c.getTab(tabID)
	if err != nil {
		return nil, err
	}

	// Use CDP Network.getAllCookies to get all cookies regardless of current page path
	cookiesProto, err := proto.NetworkGetAllCookies{}.Call(tab.page)
	if err != nil {
		return nil, errors.NewBrowserError("get_cookies", fmt.Errorf("failed to get cookies: %w", err))
	}

	// Convert CDP cookies to protocol cookies
	cookies := make([]protocol.Cookie, len(cookiesProto.Cookies))
	for i, c := range cookiesProto.Cookies {
		cookies[i] = protocol.Cookie{
			Name:     c.Name,
			Value:    c.Value,
			Domain:   c.Domain,
			Path:     c.Path,
			Expires:  float64(c.Expires),
			HTTPOnly: c.HTTPOnly,
			Secure:   c.Secure,
			SameSite: string(c.SameSite),
		}
	}

	return &protocol.GetCookiesResult{Cookies: cookies}, nil
}

// SetCookies sets cookies in the browser
func (c *Context) SetCookies(ctx context.Context, cookies []protocol.Cookie, tabID *string) (*protocol.SetCookiesResult, error) {
	tab, err := c.getTab(tabID)
	if err != nil {
		return nil, err
	}

	count := 0
	for _, cookie := range cookies {
		// Convert protocol cookie to CDP cookie
		sameSite := proto.NetworkCookieSameSiteLax
		switch cookie.SameSite {
		case "Strict":
			sameSite = proto.NetworkCookieSameSiteStrict
		case "None":
			sameSite = proto.NetworkCookieSameSiteNone
		case "Lax":
			sameSite = proto.NetworkCookieSameSiteLax
		}

		expires := proto.TimeSinceEpoch(cookie.Expires)
		result, err := proto.NetworkSetCookie{
			Name:     cookie.Name,
			Value:    cookie.Value,
			Domain:   cookie.Domain,
			Path:     cookie.Path,
			Expires:  expires,
			HTTPOnly: cookie.HTTPOnly,
			Secure:   cookie.Secure,
			SameSite: sameSite,
		}.Call(tab.page)

		if err != nil {
			return nil, errors.NewBrowserError("set_cookies", fmt.Errorf("failed to set cookie %s: %w", cookie.Name, err))
		}

		if result != nil && !result.Success {
			return nil, errors.NewBrowserError("set_cookies", fmt.Errorf("cookie %s was rejected by browser (domain/path mismatch?)", cookie.Name))
		}
		count++
	}

	return &protocol.SetCookiesResult{Set: count}, nil
}

// ClearCookies clears all cookies from the browser
func (c *Context) ClearCookies(ctx context.Context, tabID *string) (*protocol.ClearCookiesResult, error) {
	tab, err := c.getTab(tabID)
	if err != nil {
		return nil, err
	}

	// First get all cookies to count them
	cookiesProto, err := proto.NetworkGetCookies{}.Call(tab.page)
	if err != nil {
		return nil, errors.NewBrowserError("clear_cookies", fmt.Errorf("failed to get cookies: %w", err))
	}

	count := len(cookiesProto.Cookies)

	// Clear all cookies
	err = proto.NetworkClearBrowserCookies{}.Call(tab.page)
	if err != nil {
		return nil, errors.NewBrowserError("clear_cookies", fmt.Errorf("failed to clear cookies: %w", err))
	}

	return &protocol.ClearCookiesResult{Cleared: count}, nil
}

// SaveStorageState saves the current storage state (cookies + localStorage)
func (c *Context) SaveStorageState(ctx context.Context, tabID *string) (*protocol.SaveStorageStateResult, error) {
	tab, err := c.getTab(tabID)
	if err != nil {
		return nil, err
	}

	// Get all cookies
	cookiesProto, err := proto.NetworkGetAllCookies{}.Call(tab.page)
	if err != nil {
		return nil, errors.NewBrowserError("save_storage_state", fmt.Errorf("failed to get cookies: %w", err))
	}

	// Convert CDP cookies to protocol cookies
	cookies := make([]protocol.Cookie, len(cookiesProto.Cookies))
	for i, c := range cookiesProto.Cookies {
		cookies[i] = protocol.Cookie{
			Name:     c.Name,
			Value:    c.Value,
			Domain:   c.Domain,
			Path:     c.Path,
			Expires:  float64(c.Expires),
			HTTPOnly: c.HTTPOnly,
			Secure:   c.Secure,
			SameSite: string(c.SameSite),
		}
	}

	// Get unique origins from cookies
	originMap := make(map[string]bool)
	for _, cookie := range cookies {
		// Create origin from cookie domain
		origin := "http://" + strings.TrimPrefix(cookie.Domain, ".")
		if cookie.Secure {
			origin = "https://" + strings.TrimPrefix(cookie.Domain, ".")
		}
		originMap[origin] = true
	}

	// Also add the current page origin
	currentURL := ""
	if info, err := tab.page.Info(); err == nil && info != nil {
		currentURL = info.URL
		if currentURL != "" && currentURL != "about:blank" {
			// Extract origin from URL
			if idx := strings.Index(currentURL, "://"); idx != -1 {
				rest := currentURL[idx+3:]
				if slashIdx := strings.Index(rest, "/"); slashIdx != -1 {
					origin := currentURL[:idx+3+slashIdx]
					originMap[origin] = true
				} else {
					originMap[currentURL] = true
				}
			}
		}
	}

	// Get localStorage for each origin
	origins := make([]protocol.OriginState, 0)
	for origin := range originMap {
		// Navigate to origin to read localStorage
		err = tab.page.Navigate(origin)
		if err != nil {
			// Skip origins that can't be navigated to
			continue
		}
		tab.page.WaitLoad()

		// Get localStorage via JavaScript
		localStorage, err := tab.page.Eval("() => { return JSON.stringify(localStorage) }")
		if err != nil {
			// Skip if localStorage is not accessible
			continue
		}

		var items map[string]string
		if err := json.Unmarshal([]byte(localStorage.Value.String()), &items); err != nil {
			continue
		}

		if len(items) > 0 {
			origins = append(origins, protocol.OriginState{
				Origin:       origin,
				LocalStorage: items,
			})
		}
	}

	state := protocol.StorageState{
		Cookies: cookies,
		Origins: origins,
	}

	return &protocol.SaveStorageStateResult{State: state}, nil
}

// LoadStorageState loads a storage state (cookies + localStorage)
func (c *Context) LoadStorageState(ctx context.Context, state protocol.StorageState, tabID *string) (*protocol.LoadStorageStateResult, error) {
	tab, err := c.getTab(tabID)
	if err != nil {
		return nil, err
	}

	// Save current URL
	currentURL := ""
	if info, err := tab.page.Info(); err == nil && info != nil {
		currentURL = info.URL
	}

	// Set cookies (can be done cross-origin)
	for _, cookie := range state.Cookies {
		sameSite := proto.NetworkCookieSameSiteLax
		switch cookie.SameSite {
		case "Strict":
			sameSite = proto.NetworkCookieSameSiteStrict
		case "None":
			sameSite = proto.NetworkCookieSameSiteNone
		case "Lax":
			sameSite = proto.NetworkCookieSameSiteLax
		}

		expires := proto.TimeSinceEpoch(cookie.Expires)
		result, err := proto.NetworkSetCookie{
			Name:     cookie.Name,
			Value:    cookie.Value,
			Domain:   cookie.Domain,
			Path:     cookie.Path,
			Expires:  expires,
			HTTPOnly: cookie.HTTPOnly,
			Secure:   cookie.Secure,
			SameSite: sameSite,
		}.Call(tab.page)

		if err != nil {
			return nil, errors.NewBrowserError("load_storage_state", fmt.Errorf("failed to set cookie %s: %w", cookie.Name, err))
		}

		if result != nil && !result.Success {
			return nil, errors.NewBrowserError("load_storage_state", fmt.Errorf("cookie %s was rejected by browser (domain/path mismatch?)", cookie.Name))
		}
	}

	// Set localStorage for each origin (requires navigation due to same-origin policy)
	for _, originState := range state.Origins {
		if len(originState.LocalStorage) == 0 {
			continue
		}

		// Navigate to origin
		err = tab.page.Navigate(originState.Origin)
		if err != nil {
			return nil, errors.NewBrowserError("load_storage_state", fmt.Errorf("failed to navigate to origin %s: %w", originState.Origin, err))
		}
		tab.page.WaitLoad()

		// Set each localStorage item via JavaScript
		for key, value := range originState.LocalStorage {
			// Escape quotes in key and value
			escapedKey := strings.ReplaceAll(key, "\\", "\\\\")
			escapedKey = strings.ReplaceAll(escapedKey, "'", "\\'")
			escapedValue := strings.ReplaceAll(value, "\\", "\\\\")
			escapedValue = strings.ReplaceAll(escapedValue, "'", "\\'")

			script := fmt.Sprintf("() => localStorage.setItem('%s', '%s')", escapedKey, escapedValue)
			_, err := tab.page.Eval(script)
			if err != nil {
				return nil, errors.NewBrowserError("load_storage_state", fmt.Errorf("failed to set localStorage for %s: %w", originState.Origin, err))
			}
		}
	}

	// Navigate back to original URL or about:blank
	if currentURL != "" && currentURL != "about:blank" {
		err = tab.page.Navigate(currentURL)
		if err == nil {
			tab.page.WaitLoad()
		}
		// Ignore navigation errors on return
	} else {
		_ = tab.page.Navigate("about:blank")
	}

	return &protocol.LoadStorageStateResult{Loaded: true}, nil
}

// SetLocalStorage sets localStorage items for the current page
func (c *Context) SetLocalStorage(ctx context.Context, items map[string]string, tabID *string) (*protocol.SetLocalStorageResult, error) {
	tab, err := c.getTab(tabID)
	if err != nil {
		return nil, err
	}

	count := 0
	for key, value := range items {
		// Escape quotes in key and value
		escapedKey := strings.ReplaceAll(key, "\\", "\\\\")
		escapedKey = strings.ReplaceAll(escapedKey, "'", "\\'")
		escapedValue := strings.ReplaceAll(value, "\\", "\\\\")
		escapedValue = strings.ReplaceAll(escapedValue, "'", "\\'")

		script := fmt.Sprintf("() => localStorage.setItem('%s', '%s')", escapedKey, escapedValue)
		_, err := tab.page.Eval(script)
		if err != nil {
			return nil, errors.NewBrowserError("set_local_storage", fmt.Errorf("failed to set item %s: %w", key, err))
		}
		count++
	}

	return &protocol.SetLocalStorageResult{Set: count}, nil
}

// GetLocalStorage gets all localStorage items from the current page
func (c *Context) GetLocalStorage(ctx context.Context, tabID *string) (*protocol.GetLocalStorageResult, error) {
	tab, err := c.getTab(tabID)
	if err != nil {
		return nil, err
	}

	// Get localStorage via JavaScript
	localStorage, err := tab.page.Eval("() => { return JSON.stringify(localStorage) }")
	if err != nil {
		return nil, errors.NewBrowserError("get_local_storage", fmt.Errorf("failed to get localStorage: %w", err))
	}

	var items map[string]string
	if err := json.Unmarshal([]byte(localStorage.Value.String()), &items); err != nil {
		return nil, errors.NewBrowserError("get_local_storage", fmt.Errorf("failed to parse localStorage: %w", err))
	}

	return &protocol.GetLocalStorageResult{Items: items}, nil
}

// LoadTaskCredentials fetches task credentials from Convex and loads them into the browser
func (c *Context) LoadTaskCredentials(ctx context.Context, testCaseName, taskID string) (*protocol.LoadTaskCredentialsResult, error) {
	// Get Convex credentials from environment
	convexURL := os.Getenv("CONVEX_URL")
	convexSecretKey := os.Getenv("CONVEX_SECRET_KEY")

	if convexURL == "" || convexSecretKey == "" {
		return nil, errors.NewValidationError("credentials", "CONVEX_URL and CONVEX_SECRET_KEY",
			fmt.Errorf("environment variables not set"))
	}

	// Create Convex client
	convexClient := convex.NewClient(convexURL, convexSecretKey)

	// Fetch task from Convex
	task, err := convexClient.FetchTask(ctx, testCaseName, taskID)
	if err != nil {
		return nil, errors.NewBrowserError("load_task_credentials",
			fmt.Errorf("failed to fetch task from Convex: %w", err))
	}

	cookiesLoaded := 0

	// Load credentials from loginCookie field (legacy support)
	if task.LoginCookie != "" {
		// Parse storage state JSON from loginCookie
		var storageState protocol.StorageState
		if err := json.Unmarshal([]byte(task.LoginCookie), &storageState); err != nil {
			return nil, errors.NewBrowserError("load_task_credentials",
				fmt.Errorf("failed to parse loginCookie: %w", err))
		}

		// Load storage state (cookies + localStorage)
		_, err := c.LoadStorageState(ctx, storageState, nil)
		if err != nil {
			return nil, errors.NewBrowserError("load_task_credentials",
				fmt.Errorf("failed to load storage state: %w", err))
		}

		cookiesLoaded = len(storageState.Cookies)
	}

	// Load credentials from credentials field (new format)
	if task.Credentials != nil {
		// Handle cookies
		if cookiesRaw, ok := task.Credentials["cookies"]; ok {
			cookiesJSON, err := json.Marshal(cookiesRaw)
			if err == nil {
				var cookies []protocol.Cookie
				if err := json.Unmarshal(cookiesJSON, &cookies); err == nil {
					_, err := c.SetCookies(ctx, cookies, nil)
					if err != nil {
						return nil, errors.NewBrowserError("load_task_credentials",
							fmt.Errorf("failed to set cookies: %w", err))
					}
					cookiesLoaded += len(cookies)
				}
			}
		}

		// Handle localStorage
		if localStorageRaw, ok := task.Credentials["localStorage"]; ok {
			localStorageJSON, err := json.Marshal(localStorageRaw)
			if err == nil {
				var localStorage map[string]string
				if err := json.Unmarshal(localStorageJSON, &localStorage); err == nil {
					_, err := c.SetLocalStorage(ctx, localStorage, nil)
					if err != nil {
						return nil, errors.NewBrowserError("load_task_credentials",
							fmt.Errorf("failed to set localStorage: %w", err))
					}
				}
			}
		}
	}

	return &protocol.LoadTaskCredentialsResult{
		Loaded:       true,
		CookiesCount: cookiesLoaded,
		TaskID:       taskID,
	}, nil
}

// SetDownloadBehavior configures download behavior for the browser
func (c *Context) SetDownloadBehavior(ctx context.Context, path string, accept bool, tabID *string) (*protocol.SetDownloadBehaviorResult, error) {
	tab, err := c.getTab(tabID)
	if err != nil {
		return nil, err
	}

	// Enable Page domain for download events
	err = proto.PageEnable{}.Call(tab.page)
	if err != nil {
		return nil, errors.NewBrowserError("set_download_behavior", fmt.Errorf("failed to enable Page domain: %w", err))
	}

	// Configure download behavior using CDP
	behavior := "deny"
	if accept {
		behavior = "allow"
	}

	err = proto.PageSetDownloadBehavior{
		Behavior:     proto.PageSetDownloadBehaviorBehavior(behavior),
		DownloadPath: path,
	}.Call(tab.page)

	if err != nil {
		return nil, errors.NewBrowserError("set_download_behavior", fmt.Errorf("failed to configure downloads: %w", err))
	}

	// Start listening for download events if accepting downloads
	if accept {
		c.listenForDownloadEvents(tab, path)

		// Initialize and start downloads watchdog
		if tab.downloadsWatchdog == nil {
			tab.downloadsWatchdog = watchdog.NewDownloadsWatchdog(tab.page, c.eventBus, path)
			if err := tab.downloadsWatchdog.Start(ctx); err != nil {
				return nil, errors.NewBrowserError("set_download_behavior", fmt.Errorf("failed to start downloads watchdog: %w", err))
			}
		}
	}

	return &protocol.SetDownloadBehaviorResult{Configured: true}, nil
}

// GetDownloads returns the list of downloads for the current tab
func (c *Context) GetDownloads(ctx context.Context, tabID *string) (*protocol.GetDownloadsResult, error) {
	tab, err := c.getTab(tabID)
	if err != nil {
		return nil, err
	}

	tab.downloadsMu.RLock()
	defer tab.downloadsMu.RUnlock()

	// Return a copy of downloads to avoid data races
	downloadsCopy := make([]protocol.Download, len(tab.downloads))
	copy(downloadsCopy, tab.downloads)

	return &protocol.GetDownloadsResult{Downloads: downloadsCopy}, nil
}

// WaitForDownload blocks until a download completes or timeout occurs
func (c *Context) WaitForDownload(ctx context.Context, timeoutMs int, tabID *string) (*protocol.WaitForDownloadResult, error) {
	tab, err := c.getTab(tabID)
	if err != nil {
		return nil, err
	}

	timeout := time.Duration(timeoutMs) * time.Millisecond
	select {
	case download := <-tab.downloadChan:
		return &protocol.WaitForDownloadResult{Download: download}, nil
	case <-time.After(timeout):
		return nil, fmt.Errorf("timeout waiting for download")
	case <-ctx.Done():
		return nil, errors.NewBrowserError("wait_for_download", ctx.Err())
	}
}

// listenForDownloadEvents listens for CDP download events in goroutines
func (c *Context) listenForDownloadEvents(tab *tabContext, downloadPath string) {
	// Track downloads by GUID
	downloadsByGUID := make(map[string]int) // GUID -> index in tab.downloads
	var guidMu sync.Mutex

	// Listen for downloadWillBegin events in a goroutine
	go tab.page.EachEvent(func(e *proto.PageDownloadWillBegin) {
		// Generate GUID for download
		guid := e.GUID
		if guid == "" {
			guid = fmt.Sprintf("download-%d", time.Now().UnixNano())
		}

		// Construct the full download path
		fullPath := filepath.Join(downloadPath, e.SuggestedFilename)

		download := protocol.Download{
			GUID:              guid,
			URL:               e.URL,
			SuggestedFilename: e.SuggestedFilename,
			State:             "inProgress",
			TotalBytes:        0,
			Path:              fullPath,
		}

		tab.downloadsMu.Lock()
		tab.downloads = append(tab.downloads, download)
		downloadIndex := len(tab.downloads) - 1
		tab.downloadsMu.Unlock()

		guidMu.Lock()
		downloadsByGUID[guid] = downloadIndex
		guidMu.Unlock()
	})()

	// Listen for downloadProgress events in a separate goroutine
	go tab.page.EachEvent(func(e *proto.PageDownloadProgress) {
		// Handle download completion or cancellation (in headless mode, downloads often show as "canceled" even when successful)
		// We treat both "completed" and "canceled" with TotalBytes > 0 as successful downloads
		isFinished := e.State == proto.PageDownloadProgressStateCompleted ||
			(e.State == proto.PageDownloadProgressState("canceled") && e.TotalBytes > 0)

		if isFinished {
			guidMu.Lock()
			downloadIndex, exists := downloadsByGUID[e.GUID]
			guidMu.Unlock()

			if !exists {
				return
			}

			tab.downloadsMu.Lock()
			if downloadIndex < len(tab.downloads) {
				tab.downloads[downloadIndex].State = "completed"
				tab.downloads[downloadIndex].TotalBytes = int64(e.TotalBytes)

				completedDownload := tab.downloads[downloadIndex]
				tab.downloadsMu.Unlock()

				// Send completed download to channel (blocking)
				// The channel is buffered, so this won't block unless buffer is full
				tab.downloadChan <- completedDownload
			} else {
				tab.downloadsMu.Unlock()
			}
		}
	})()
}

// GetClosedPopupMessages returns all closed popup messages
func (c *Context) GetClosedPopupMessages(ctx context.Context) (*protocol.GetClosedPopupMessagesResult, error) {
	if c.popupsWatchdog == nil {
		return &protocol.GetClosedPopupMessagesResult{
			Messages: []string{},
		}, nil
	}

	messages := c.popupsWatchdog.GetClosedPopupMessages()
	return &protocol.GetClosedPopupMessagesResult{
		Messages: messages,
	}, nil
}

// SubscribeToBrowserEvents returns a channel that receives browser events
func (c *Context) SubscribeToBrowserEvents(ctx context.Context) <-chan events.BrowserEvent {
	if c.eventBus == nil {
		// Return a closed channel if event bus is not available
		ch := make(chan events.BrowserEvent)
		close(ch)
		return ch
	}
	return c.eventBus.Subscribe(ctx)
}

// Close closes the browser context
func (c *Context) Close(ctx context.Context) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	// Stop watchdogs
	if c.popupsWatchdog != nil {
		c.popupsWatchdog.Stop()
	}

	if c.crashWatchdog != nil {
		c.crashWatchdog.Stop()
	}

	if c.storageWatchdog != nil {
		c.storageWatchdog.Stop()
	}

	// Close event bus
	if c.eventBus != nil {
		c.eventBus.Close()
	}

	if c.browser != nil {
		if err := c.browser.Close(); err != nil {
			return errors.NewContextError("close", err)
		}
	}
	return nil
}
