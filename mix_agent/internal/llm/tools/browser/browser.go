package browser

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"

	browserprotocol "github.com/sarathmenon/browser-service/pkg/protocol"

	"mix/internal/llm/interfaces"
	"mix/internal/llm/tools"
	"mix/internal/llm/tools/browser/vision"
	"mix/internal/permission"
	"mix/internal/session"
)

const (
	BrowserToolName = "Browser"

	// DefaultRequestTimeout is the default timeout for browser operations
	DefaultRequestTimeout = 30 * time.Second

	// URL schemes
	schemeHTTP  = "http"
	schemeHTTPS = "https"
	schemeFile  = "file"
)

// browserTool implements the Browser tool for LLM-driven browser automation
type browserTool struct {
	permissions       permission.Service
	connectionManager *ConnectionManager
	sessionConfig     session.Config
	baseURL           string
	elementCache      map[string]map[int]int64 // sessionID_tabID → visualIndex → backendID
	cacheMu           sync.RWMutex             // Protect element cache
	activeTabIDs      map[string]string        // sessionID → activeTabID
	activeTabMu       sync.RWMutex             // Protect active tab tracking
}

// NewBrowserTool creates a new browser tool instance
func NewBrowserTool(permissions permission.Service, browserServiceURL string, sessionConfig session.Config) interfaces.BaseTool {
	return &browserTool{
		permissions:       permissions,
		connectionManager: NewConnectionManager(browserServiceURL),
		sessionConfig:     sessionConfig,
		baseURL:           getBaseURL(),
		elementCache:      make(map[string]map[int]int64),
		activeTabIDs:      make(map[string]string),
	}
}

// Info returns tool metadata for the LLM
func (b *browserTool) Info() interfaces.ToolInfo {
	return interfaces.ToolInfo{
		Name:        BrowserToolName,
		Description: loadBrowserDescription(),
		Parameters: map[string]any{
			"action": map[string]any{
				"type":        "string",
				"description": "The action to perform",
				"enum":        []string{ActionOpen, ActionScreenshot, ActionReadPage, ActionClick, ActionType, ActionScroll, ActionUpload, ActionGetText, ActionFind, ActionClose, ActionRightClick, ActionDoubleClick, ActionTripleClick, ActionDrag, ActionFormInput, ActionGoBack, ActionGoForward, ActionTabCreate, ActionTabList, ActionTabSwitch, ActionTabClose, ActionWait},
			},
			"url": map[string]any{
				"type":        "string",
				"description": "URL to navigate to (for open action or optional for tab_create). Supports http://, https://, and file:// schemes. For file:// URLs, path must be within session storage directory.",
			},
			"interactiveOnly": map[string]any{
				"type":        "boolean",
				"description": "Filter to interactive elements only (for read_page action, default: false)",
			},
			"index": map[string]any{
				"type":        "integer",
				"description": "Element index (for click/type/upload actions)",
			},
			"text": map[string]any{
				"type":        "string",
				"description": "Text to type (for type action)",
			},
			"direction": map[string]any{
				"type":        "string",
				"description": "Scroll direction (for scroll action)",
				"enum":        []string{DirectionUp, DirectionDown, DirectionLeft, DirectionRight},
			},
			"amount": map[string]any{
				"type":        "integer",
				"description": "Scroll amount in pixels (for scroll action)",
			},
			"filePath": map[string]any{
				"type":        "string",
				"description": "File path to upload (for upload action) - can be absolute or session-relative",
			},
			"strategy": map[string]any{
				"type":        "string",
				"description": "Text extraction strategy (for get_text action): auto, article, main, body",
				"enum":        []string{"auto", "article", "main", "body"},
			},
			"query": map[string]any{
				"type":        "string",
				"description": "Keyword query to find elements (for find action)",
			},
			"value": map[string]any{
				"type":        "string",
				"description": "Value to set in form input (for form_input action)",
			},
			"tabId": map[string]any{
				"type":        "string",
				"description": "Tab ID to operate on (optional - defaults to active tab). Required for tab_switch and tab_close actions.",
			},
			"duration": map[string]any{
				"type":        "integer",
				"description": "Wait duration in milliseconds (for wait action) or drag duration in milliseconds (for drag action, default: 500)",
			},
			"fromIndex": map[string]any{
				"type":        "integer",
				"description": "Element index to drag from (for drag action in index mode)",
			},
			"toIndex": map[string]any{
				"type":        "integer",
				"description": "Element index to drag to (for drag action in index mode)",
			},
			"fromX": map[string]any{
				"type":        "number",
				"description": "X coordinate to drag from (for drag action in coordinate mode)",
			},
			"fromY": map[string]any{
				"type":        "number",
				"description": "Y coordinate to drag from (for drag action in coordinate mode)",
			},
			"toX": map[string]any{
				"type":        "number",
				"description": "X coordinate to drag to (for drag action in coordinate mode)",
			},
			"toY": map[string]any{
				"type":        "number",
				"description": "Y coordinate to drag to (for drag action in coordinate mode)",
			},
		},
		Required: []string{"action"},
	}
}

// Run executes the browser tool action
func (b *browserTool) Run(ctx context.Context, call interfaces.ToolCall) (interfaces.ToolResponse, error) {
	var params BrowserParams
	if err := json.Unmarshal([]byte(call.Input), &params); err != nil {
		return interfaces.NewTextErrorResponse("invalid parameters"), fmt.Errorf("failed to unmarshal browser parameters: %w", err)
	}

	// Validate action
	if params.Action == "" {
		return interfaces.NewTextErrorResponse("missing action parameter"), nil
	}

	// Get session context
	sessionID, _, sessionStorageDir, err := b.getContextInfo(ctx)
	if err != nil {
		return interfaces.ToolResponse{}, err
	}

	// Add timeout to context
	ctx, cancel := context.WithTimeout(ctx, DefaultRequestTimeout)
	defer cancel()

	// Route to appropriate action handler
	switch params.Action {
	case ActionOpen:
		return b.handleOpen(ctx, params, sessionID), nil
	case ActionScreenshot:
		return b.handleScreenshot(ctx, params, sessionID, sessionStorageDir), nil
	case ActionReadPage:
		return b.handleReadPage(ctx, params, sessionID), nil
	case ActionClick:
		return b.handleClick(ctx, params, sessionID), nil
	case ActionType:
		return b.handleType(ctx, params, sessionID), nil
	case ActionScroll:
		return b.handleScroll(ctx, params, sessionID), nil
	case ActionUpload:
		return b.handleUpload(ctx, params, sessionID, sessionStorageDir), nil
	case ActionGetText:
		return b.handleGetText(ctx, params, sessionID), nil
	case ActionFind:
		return b.handleFind(ctx, params, sessionID), nil
	case ActionClose:
		return b.handleClose(ctx, sessionID), nil
	case ActionRightClick:
		return b.handleRightClick(ctx, params, sessionID), nil
	case ActionDoubleClick:
		return b.handleDoubleClick(ctx, params, sessionID), nil
	case ActionTripleClick:
		return b.handleTripleClick(ctx, params, sessionID), nil
	case ActionDrag:
		return b.handleDrag(ctx, params, sessionID), nil
	case ActionFormInput:
		return b.handleFormInput(ctx, params, sessionID), nil
	case ActionGoBack:
		return b.handleGoBack(ctx, params, sessionID), nil
	case ActionGoForward:
		return b.handleGoForward(ctx, params, sessionID), nil
	case ActionWait:
		return b.handleWait(ctx, params, sessionID), nil
	case ActionTabCreate:
		return b.handleTabCreate(ctx, params, sessionID), nil
	case ActionTabList:
		return b.handleTabList(ctx, sessionID), nil
	case ActionTabSwitch:
		return b.handleTabSwitch(ctx, params, sessionID), nil
	case ActionTabClose:
		return b.handleTabClose(ctx, params, sessionID), nil
	default:
		return interfaces.NewTextErrorResponse(fmt.Sprintf("unknown action: %s", params.Action)), nil
	}
}

// handleOpen navigates to a URL
func (b *browserTool) handleOpen(ctx context.Context, params BrowserParams, sessionID string) interfaces.ToolResponse {
	// Validate URL
	if params.URL == "" {
		return interfaces.NewTextErrorResponse("missing url parameter for open action")
	}

	// Parse and validate URL
	parsedURL, err := url.Parse(params.URL)
	if err != nil {
		return interfaces.NewTextErrorResponse(fmt.Sprintf("invalid URL %s: %v", params.URL, err))
	}
	if parsedURL.Scheme != schemeHTTP && parsedURL.Scheme != schemeHTTPS && parsedURL.Scheme != schemeFile {
		return interfaces.NewTextErrorResponse(fmt.Sprintf("invalid URL scheme: %s (must be http, https, or file)", params.URL))
	}

	// Security validation for file:// URLs
	if parsedURL.Scheme == schemeFile {
		filePath := parsedURL.Path
		absFilePath, err := filepath.Abs(filePath)
		if err != nil {
			return interfaces.NewTextErrorResponse(fmt.Sprintf("invalid file path: %v", err))
		}

		// Get the session storage directory
		sessionStorageDir := session.GetSessionStoragePath(sessionID, b.sessionConfig)
		absSessionDir, err := filepath.Abs(sessionStorageDir)
		if err != nil {
			return interfaces.NewTextErrorResponse(fmt.Sprintf("failed to resolve session directory: %v", err))
		}

		// Security: Only allow files within session directory
		if !strings.HasPrefix(absFilePath, absSessionDir+string(filepath.Separator)) {
			return interfaces.NewTextErrorResponse("file:// URLs must reference files within session storage directory")
		}

		// Verify file exists
		if _, err := os.Stat(absFilePath); os.IsNotExist(err) {
			return interfaces.NewTextErrorResponse(fmt.Sprintf("file not found: %s", absFilePath))
		}
	}

	// Permission check temporarily disabled for testing
	// TODO: Re-enable permissions later

	// Get or create browser connection
	client, err := b.connectionManager.GetOrCreate(ctx, sessionID)
	if err != nil {
		return interfaces.NewTextErrorResponse(fmt.Sprintf("Failed to connect to browser service: %v", err))
	}

	// Navigate
	var result *browserprotocol.NavigateResult
	if params.TabID != "" {
		result, err = client.Navigate(ctx, params.URL, params.TabID)
	} else {
		result, err = client.Navigate(ctx, params.URL)
	}
	if err != nil {
		return interfaces.NewTextErrorResponse(fmt.Sprintf("Navigation failed: %v", err))
	}

	// Clear element cache on navigation
	tabID := params.TabID
	if tabID == "" {
		// Get active tab ID for cache clearing
		activeTabID, err := b.getActiveTabID(ctx, sessionID)
		if err == nil {
			tabID = activeTabID
		}
	}
	if tabID != "" {
		b.clearCacheForTab(sessionID, tabID)
	}

	return interfaces.NewTextResponse(fmt.Sprintf("Successfully navigated to %s (Frame ID: %s)", params.URL, result.FrameID))
}

// handleScreenshot captures a screenshot
func (b *browserTool) handleScreenshot(ctx context.Context, params BrowserParams, sessionID, sessionStorageDir string) interfaces.ToolResponse {
	// Get or create browser connection
	client, err := b.connectionManager.GetOrCreate(ctx, sessionID)
	if err != nil {
		return interfaces.NewTextErrorResponse(fmt.Sprintf("Failed to connect to browser service: %v", err))
	}

	// Request raw screenshot with accessibility data
	screenshotParams := browserprotocol.ScreenshotParams{
		Format:   "png",
		FullPage: false,
		Raw:      true, // Request raw accessibility tree
	}
	if params.TabID != "" {
		screenshotParams.TabID = &params.TabID
	}

	result, err := client.Screenshot(ctx, screenshotParams)
	if err != nil {
		return interfaces.NewTextErrorResponse(fmt.Sprintf("Screenshot failed: %v", err))
	}

	// Save screenshot directly to session storage
	filename, err := saveScreenshot(result.Data, sessionStorageDir)
	if err != nil {
		return interfaces.NewTextErrorResponse(fmt.Sprintf("Failed to save screenshot: %v", err))
	}

	// Check if page is blank (Fix 2: Validate Screenshot Results)
	if result.RawNodes == nil || result.RawViewport == nil {
		// Clear cache for blank/unloaded pages
		tabID := params.TabID
		if tabID == "" {
			activeTabID, err := b.getActiveTabID(ctx, sessionID)
			if err != nil {
				return interfaces.NewTextErrorResponse(fmt.Sprintf("Failed to get active tab: %v", err))
			}
			tabID = activeTabID
		}
		b.clearCacheForTab(sessionID, tabID)

		// Format response with warning
		response := formatScreenshotResponse(filename, sessionID, b.baseURL)
		response += "\n\n⚠️  Warning: Page appears blank or not fully loaded. Element cache cleared."
		return interfaces.NewTextResponse(response)
	}

	// Cache BackendID mappings if raw data is available
	b.cacheElementMapping(ctx, sessionID, params.TabID, result.RawNodes, *result.RawViewport)

	// Format response
	response := formatScreenshotResponse(filename, sessionID, b.baseURL)

	return interfaces.NewTextResponse(response)
}

// cacheElementMapping filters raw accessibility nodes and caches BackendID mappings
func (b *browserTool) cacheElementMapping(ctx context.Context, sessionID, tabID string, rawNodes []browserprotocol.RawAccessibilityNode, viewport browserprotocol.ViewportBounds) {
	// Fix 1: Always use explicit tabID
	explicitTabID := tabID
	if explicitTabID == "" {
		// Fetch active tab ID from browser-service
		activeTabID, err := b.getActiveTabID(ctx, sessionID)
		if err != nil {
			// Cannot cache without explicit tab ID
			return
		}
		explicitTabID = activeTabID
	}

	// Convert protocol types to vision types
	visionNodes := make([]vision.RawAccessibilityNode, len(rawNodes))
	for i, node := range rawNodes {
		visionNodes[i] = vision.RawAccessibilityNode{
			Role:      node.Role,
			Name:      node.Name,
			Bounds:    vision.BoundingBox(node.Bounds),
			BackendID: node.BackendID,
		}
	}

	visionViewport := vision.ViewportBounds(viewport)

	// Filter to interactive elements in viewport
	filteredElements := vision.FilterInteractiveElements(visionNodes, visionViewport)

	// Create cache key using explicit tabID
	cacheKey := sessionID + "_" + explicitTabID

	// Build mapping: visualIndex → backendID
	mapping := make(map[int]int64)
	for _, elem := range filteredElements {
		mapping[elem.Index] = elem.BackendID
	}

	// Store in cache
	b.cacheMu.Lock()
	b.elementCache[cacheKey] = mapping
	b.cacheMu.Unlock()
}

// getBackendIDFromCache looks up the BackendID for a given visual index
func (b *browserTool) getBackendIDFromCache(ctx context.Context, sessionID, tabID string, visualIndex int) (int64, error) {
	// Fix 1: Always use explicit tabID
	explicitTabID := tabID
	if explicitTabID == "" {
		// Fetch active tab ID from browser-service
		activeTabID, err := b.getActiveTabID(ctx, sessionID)
		if err != nil {
			return 0, fmt.Errorf("failed to get active tab: %w", err)
		}
		explicitTabID = activeTabID
	}

	// Create cache key using explicit tabID
	cacheKey := sessionID + "_" + explicitTabID

	// Fix 4: Validate cache exists before lookup
	b.cacheMu.RLock()
	mapping, exists := b.elementCache[cacheKey]
	b.cacheMu.RUnlock()

	if !exists {
		return 0, fmt.Errorf("no element cache found for this tab")
	}

	if len(mapping) == 0 {
		return 0, fmt.Errorf("element cache is empty for this tab")
	}

	backendID, found := mapping[visualIndex]
	if !found {
		return 0, fmt.Errorf("index %d not found in cache", visualIndex)
	}

	return backendID, nil
}

// clearCacheForTab clears the element cache for a specific tab
func (b *browserTool) clearCacheForTab(sessionID, tabID string) {
	// Always use explicit tabID for cache key
	if tabID == "" {
		// If no tabID provided, cannot clear cache reliably
		return
	}

	cacheKey := sessionID + "_" + tabID

	b.cacheMu.Lock()
	delete(b.elementCache, cacheKey)
	b.cacheMu.Unlock()
}

// getActiveTabID retrieves the active tab ID from browser-service
func (b *browserTool) getActiveTabID(ctx context.Context, sessionID string) (string, error) {
	// Check cached active tab ID first
	b.activeTabMu.RLock()
	cachedTabID, exists := b.activeTabIDs[sessionID]
	b.activeTabMu.RUnlock()

	if exists && cachedTabID != "" {
		return cachedTabID, nil
	}

	// Fetch from browser-service
	client, err := b.connectionManager.GetOrCreate(ctx, sessionID)
	if err != nil {
		return "", fmt.Errorf("failed to connect to browser service: %w", err)
	}

	result, err := client.ListTabs(ctx)
	if err != nil {
		return "", fmt.Errorf("failed to list tabs: %w", err)
	}

	// Cache the active tab ID
	b.activeTabMu.Lock()
	b.activeTabIDs[sessionID] = result.ActiveTabID
	b.activeTabMu.Unlock()

	return result.ActiveTabID, nil
}

// updateActiveTabID updates the cached active tab ID for a session
func (b *browserTool) updateActiveTabID(sessionID, tabID string) {
	b.activeTabMu.Lock()
	b.activeTabIDs[sessionID] = tabID
	b.activeTabMu.Unlock()
}

// clearActiveTabID clears the cached active tab ID for a session
func (b *browserTool) clearActiveTabID(sessionID string) {
	b.activeTabMu.Lock()
	delete(b.activeTabIDs, sessionID)
	b.activeTabMu.Unlock()
}

// handleClick clicks an element
func (b *browserTool) handleClick(ctx context.Context, params BrowserParams, sessionID string) interfaces.ToolResponse {
	// Get browser connection
	client, err := b.connectionManager.GetOrCreate(ctx, sessionID)
	if err != nil {
		return interfaces.NewTextErrorResponse(fmt.Sprintf("Failed to connect to browser service: %v", err))
	}

	// Fix 4: Look up BackendID from cache with validation
	backendID, err := b.getBackendIDFromCache(ctx, sessionID, params.TabID, params.Index)
	if err != nil {
		return interfaces.NewTextErrorResponse(fmt.Sprintf("Element not found in cache: %v. Please take a screenshot first.", err))
	}

	// Click by BackendID
	if params.TabID != "" {
		err = client.ClickByBackendID(ctx, backendID, params.TabID)
	} else {
		err = client.ClickByBackendID(ctx, backendID)
	}
	if err != nil {
		return interfaces.NewTextErrorResponse(fmt.Sprintf("Click failed: %v", err))
	}

	return interfaces.NewTextResponse(fmt.Sprintf("Successfully clicked element %d", params.Index))
}

// handleRightClick right-clicks an element
func (b *browserTool) handleRightClick(ctx context.Context, params BrowserParams, sessionID string) interfaces.ToolResponse {
	// Get browser connection
	client, err := b.connectionManager.GetOrCreate(ctx, sessionID)
	if err != nil {
		return interfaces.NewTextErrorResponse(fmt.Sprintf("Failed to connect to browser service: %v", err))
	}

	// Fix 4: Look up BackendID from cache with validation
	backendID, err := b.getBackendIDFromCache(ctx, sessionID, params.TabID, params.Index)
	if err != nil {
		return interfaces.NewTextErrorResponse(fmt.Sprintf("Element not found in cache: %v. Please take a screenshot first.", err))
	}

	// Right-click by BackendID
	if params.TabID != "" {
		err = client.RightClickByBackendID(ctx, backendID, params.TabID)
	} else {
		err = client.RightClickByBackendID(ctx, backendID)
	}
	if err != nil {
		return interfaces.NewTextErrorResponse(fmt.Sprintf("Right-click failed: %v", err))
	}

	return interfaces.NewTextResponse(fmt.Sprintf("Successfully right-clicked element %d", params.Index))
}

// handleDoubleClick double-clicks an element
func (b *browserTool) handleDoubleClick(ctx context.Context, params BrowserParams, sessionID string) interfaces.ToolResponse {
	// Get browser connection
	client, err := b.connectionManager.GetOrCreate(ctx, sessionID)
	if err != nil {
		return interfaces.NewTextErrorResponse(fmt.Sprintf("Failed to connect to browser service: %v", err))
	}

	// Fix 4: Look up BackendID from cache with validation
	backendID, err := b.getBackendIDFromCache(ctx, sessionID, params.TabID, params.Index)
	if err != nil {
		return interfaces.NewTextErrorResponse(fmt.Sprintf("Element not found in cache: %v. Please take a screenshot first.", err))
	}

	// Double-click by BackendID
	if params.TabID != "" {
		err = client.DoubleClickByBackendID(ctx, backendID, params.TabID)
	} else {
		err = client.DoubleClickByBackendID(ctx, backendID)
	}
	if err != nil {
		return interfaces.NewTextErrorResponse(fmt.Sprintf("Double-click failed: %v", err))
	}

	return interfaces.NewTextResponse(fmt.Sprintf("Successfully double-clicked element %d", params.Index))
}

// handleTripleClick triple-clicks an element
func (b *browserTool) handleTripleClick(ctx context.Context, params BrowserParams, sessionID string) interfaces.ToolResponse {
	// Get browser connection
	client, err := b.connectionManager.GetOrCreate(ctx, sessionID)
	if err != nil {
		return interfaces.NewTextErrorResponse(fmt.Sprintf("Failed to connect to browser service: %v", err))
	}

	// Fix 4: Look up BackendID from cache with validation
	backendID, err := b.getBackendIDFromCache(ctx, sessionID, params.TabID, params.Index)
	if err != nil {
		return interfaces.NewTextErrorResponse(fmt.Sprintf("Element not found in cache: %v. Please take a screenshot first.", err))
	}

	// Triple-click by BackendID
	if params.TabID != "" {
		err = client.TripleClickByBackendID(ctx, backendID, params.TabID)
	} else {
		err = client.TripleClickByBackendID(ctx, backendID)
	}
	if err != nil {
		return interfaces.NewTextErrorResponse(fmt.Sprintf("Triple-click failed: %v", err))
	}

	return interfaces.NewTextResponse(fmt.Sprintf("Successfully triple-clicked element %d", params.Index))
}

// handleDrag performs a drag operation
func (b *browserTool) handleDrag(ctx context.Context, params BrowserParams, sessionID string) interfaces.ToolResponse {
	// Get browser connection
	client, err := b.connectionManager.GetOrCreate(ctx, sessionID)
	if err != nil {
		return interfaces.NewTextErrorResponse(fmt.Sprintf("Failed to connect to browser service: %v", err))
	}

	// Validate parameters - must have either index mode or coordinate mode
	hasIndexMode := params.FromIndex != nil && params.ToIndex != nil
	hasCoordMode := params.FromX != nil && params.FromY != nil && params.ToX != nil && params.ToY != nil

	if !hasIndexMode && !hasCoordMode {
		return interfaces.NewTextErrorResponse("drag action requires either (fromIndex and toIndex) or (fromX, fromY, toX, toY)")
	}

	if hasIndexMode && hasCoordMode {
		return interfaces.NewTextErrorResponse("drag action cannot mix index mode and coordinate mode")
	}

	// Set duration pointer
	var duration *int
	if params.Duration > 0 {
		duration = &params.Duration
	}

	// Perform drag
	if params.TabID != "" {
		err = client.Drag(ctx, params.FromIndex, params.ToIndex, params.FromX, params.FromY, params.ToX, params.ToY, duration, params.TabID)
	} else {
		err = client.Drag(ctx, params.FromIndex, params.ToIndex, params.FromX, params.FromY, params.ToX, params.ToY, duration)
	}
	if err != nil {
		return interfaces.NewTextErrorResponse(fmt.Sprintf("Drag failed: %v", err))
	}

	// Format response message
	var responseMsg string
	if hasIndexMode {
		responseMsg = fmt.Sprintf("Successfully dragged element %d to element %d", *params.FromIndex, *params.ToIndex)
	} else {
		responseMsg = fmt.Sprintf("Successfully dragged from (%.0f, %.0f) to (%.0f, %.0f)", *params.FromX, *params.FromY, *params.ToX, *params.ToY)
	}

	return interfaces.NewTextResponse(responseMsg)
}

// handleFormInput sets form input value directly
func (b *browserTool) handleFormInput(ctx context.Context, params BrowserParams, sessionID string) interfaces.ToolResponse {
	// Validate value parameter
	if params.Value == "" {
		return interfaces.NewTextErrorResponse("missing value parameter for form_input action")
	}

	// Get browser connection
	client, err := b.connectionManager.GetOrCreate(ctx, sessionID)
	if err != nil {
		return interfaces.NewTextErrorResponse(fmt.Sprintf("Failed to connect to browser service: %v", err))
	}

	// Set form input value
	if params.TabID != "" {
		err = client.FormInput(ctx, params.Index, params.Value, params.TabID)
	} else {
		err = client.FormInput(ctx, params.Index, params.Value)
	}
	if err != nil {
		return interfaces.NewTextErrorResponse(fmt.Sprintf("Form input failed: %v", err))
	}

	return interfaces.NewTextResponse(fmt.Sprintf("Successfully set value in element %d", params.Index))
}

// handleGoBack navigates backward in browser history
func (b *browserTool) handleGoBack(ctx context.Context, params BrowserParams, sessionID string) interfaces.ToolResponse {
	// Get browser connection
	client, err := b.connectionManager.GetOrCreate(ctx, sessionID)
	if err != nil {
		return interfaces.NewTextErrorResponse(fmt.Sprintf("Failed to connect to browser service: %v", err))
	}

	// Navigate back
	var resultURL string
	if params.TabID != "" {
		resultURL, err = client.GoBack(ctx, params.TabID)
	} else {
		resultURL, err = client.GoBack(ctx)
	}
	if err != nil {
		return interfaces.NewTextErrorResponse(fmt.Sprintf("Go back failed: %v", err))
	}

	return interfaces.NewTextResponse(fmt.Sprintf("Successfully navigated back to: %s", resultURL))
}

// handleGoForward navigates forward in browser history
func (b *browserTool) handleGoForward(ctx context.Context, params BrowserParams, sessionID string) interfaces.ToolResponse {
	// Get browser connection
	client, err := b.connectionManager.GetOrCreate(ctx, sessionID)
	if err != nil {
		return interfaces.NewTextErrorResponse(fmt.Sprintf("Failed to connect to browser service: %v", err))
	}

	// Navigate forward
	var resultURL string
	if params.TabID != "" {
		resultURL, err = client.GoForward(ctx, params.TabID)
	} else {
		resultURL, err = client.GoForward(ctx)
	}
	if err != nil {
		return interfaces.NewTextErrorResponse(fmt.Sprintf("Go forward failed: %v", err))
	}

	return interfaces.NewTextResponse(fmt.Sprintf("Successfully navigated forward to: %s", resultURL))
}

// handleWait pauses execution for specified milliseconds
func (b *browserTool) handleWait(ctx context.Context, params BrowserParams, sessionID string) interfaces.ToolResponse {
	if params.Duration <= 0 {
		return interfaces.NewTextErrorResponse("missing or invalid duration parameter for wait action")
	}

	client, err := b.connectionManager.GetOrCreate(ctx, sessionID)
	if err != nil {
		return interfaces.NewTextErrorResponse(fmt.Sprintf("Failed to connect to browser: %v", err))
	}

	if params.TabID != "" {
		err = client.Wait(ctx, params.Duration, params.TabID)
	} else {
		err = client.Wait(ctx, params.Duration)
	}

	if err != nil {
		return interfaces.NewTextErrorResponse(fmt.Sprintf("Wait failed: %v", err))
	}

	return interfaces.NewTextResponse(fmt.Sprintf("Waited %d milliseconds", params.Duration))
}

// handleType types text into an element
func (b *browserTool) handleType(ctx context.Context, params BrowserParams, sessionID string) interfaces.ToolResponse {
	// Validate text parameter
	if params.Text == "" {
		return interfaces.NewTextErrorResponse("missing text parameter for type action")
	}

	// Get browser connection
	client, err := b.connectionManager.GetOrCreate(ctx, sessionID)
	if err != nil {
		return interfaces.NewTextErrorResponse(fmt.Sprintf("Failed to connect to browser service: %v", err))
	}

	// Type text
	if params.TabID != "" {
		err = client.Type(ctx, params.Index, params.Text, params.TabID)
	} else {
		err = client.Type(ctx, params.Index, params.Text)
	}
	if err != nil {
		return interfaces.NewTextErrorResponse(fmt.Sprintf("Type failed: %v", err))
	}

	return interfaces.NewTextResponse(fmt.Sprintf("Successfully typed text into element %d", params.Index))
}

// handleScroll scrolls the page
func (b *browserTool) handleScroll(ctx context.Context, params BrowserParams, sessionID string) interfaces.ToolResponse {
	// Validate direction
	if params.Direction == "" {
		return interfaces.NewTextErrorResponse("missing direction parameter for scroll action")
	}

	// Validate direction value
	validDirections := map[string]bool{
		DirectionUp:    true,
		DirectionDown:  true,
		DirectionLeft:  true,
		DirectionRight: true,
	}
	if !validDirections[params.Direction] {
		return interfaces.NewTextErrorResponse(fmt.Sprintf("invalid direction: %s (must be up/down/left/right)", params.Direction))
	}

	// Default amount to 100 pixels if not specified
	amount := params.Amount
	if amount == 0 {
		amount = 100
	}

	// Get browser connection
	client, err := b.connectionManager.GetOrCreate(ctx, sessionID)
	if err != nil {
		return interfaces.NewTextErrorResponse(fmt.Sprintf("Failed to connect to browser service: %v", err))
	}

	// Scroll
	if params.TabID != "" {
		err = client.Scroll(ctx, params.Direction, amount, params.TabID)
	} else {
		err = client.Scroll(ctx, params.Direction, amount)
	}
	if err != nil {
		return interfaces.NewTextErrorResponse(fmt.Sprintf("Scroll failed: %v", err))
	}

	return interfaces.NewTextResponse(fmt.Sprintf("Successfully scrolled %s by %d pixels", params.Direction, amount))
}

// handleUpload uploads a file to a file input element
func (b *browserTool) handleUpload(ctx context.Context, params BrowserParams, sessionID, sessionStorageDir string) interfaces.ToolResponse {
	// Validate parameters
	if params.FilePath == "" {
		return interfaces.NewTextErrorResponse("missing filePath parameter for upload action")
	}

	// Resolve file path
	var absolutePath string
	if filepath.IsAbs(params.FilePath) {
		// Absolute path - use as is
		absolutePath = params.FilePath
	} else {
		// Relative path - try session storage first, then uploads directory
		sessionPath := filepath.Join(sessionStorageDir, params.FilePath)
		if _, err := os.Stat(sessionPath); err == nil {
			absolutePath = sessionPath
		} else {
			// Try uploads directory
			uploadsPath := filepath.Join(b.sessionConfig.BasePath, "uploads", params.FilePath)
			if _, err := os.Stat(uploadsPath); err == nil {
				absolutePath = uploadsPath
			} else {
				return interfaces.NewTextErrorResponse(fmt.Sprintf("File not found: %s (tried session storage and uploads directory)", params.FilePath))
			}
		}
	}

	// Security check - ensure path is within allowed directories
	allowedDirs := []string{sessionStorageDir, filepath.Join(b.sessionConfig.BasePath, "uploads")}
	isAllowed := false
	for _, dir := range allowedDirs {
		if strings.HasPrefix(absolutePath, dir) {
			isAllowed = true
			break
		}
	}
	if !isAllowed {
		return interfaces.NewTextErrorResponse(fmt.Sprintf("File path outside allowed directories: %s", params.FilePath))
	}

	// Verify file exists and is not a directory
	info, err := os.Stat(absolutePath)
	if err != nil {
		return interfaces.NewTextErrorResponse(fmt.Sprintf("File not found: %s", absolutePath))
	}
	if info.IsDir() {
		return interfaces.NewTextErrorResponse(fmt.Sprintf("Path is a directory, not a file: %s", absolutePath))
	}

	// Get browser connection
	client, err := b.connectionManager.GetOrCreate(ctx, sessionID)
	if err != nil {
		return interfaces.NewTextErrorResponse(fmt.Sprintf("Failed to connect to browser service: %v", err))
	}

	// Upload file
	var result *browserprotocol.UploadFileResult
	if params.TabID != "" {
		result, err = client.UploadFile(ctx, params.Index, []string{absolutePath}, params.TabID)
	} else {
		result, err = client.UploadFile(ctx, params.Index, []string{absolutePath})
	}
	if err != nil {
		return interfaces.NewTextErrorResponse(fmt.Sprintf("Upload failed: %v", err))
	}

	return interfaces.NewTextResponse(fmt.Sprintf("Successfully uploaded %d file(s): %s", result.FilesUploaded, strings.Join(result.FileNames, ", ")))
}

// handleGetText extracts text content from the page
func (b *browserTool) handleGetText(ctx context.Context, params BrowserParams, sessionID string) interfaces.ToolResponse {
	// Default strategy to auto
	strategy := params.Strategy
	if strategy == "" {
		strategy = "auto"
	}

	// Get browser connection
	client, err := b.connectionManager.GetOrCreate(ctx, sessionID)
	if err != nil {
		return interfaces.NewTextErrorResponse(fmt.Sprintf("Failed to connect to browser service: %v", err))
	}

	// Extract text
	var result *browserprotocol.GetTextResult
	if params.TabID != "" {
		result, err = client.GetText(ctx, strategy, params.TabID)
	} else {
		result, err = client.GetText(ctx, strategy)
	}
	if err != nil {
		return interfaces.NewTextErrorResponse(fmt.Sprintf("Text extraction failed: %v", err))
	}

	// Format response
	var response strings.Builder
	response.WriteString(fmt.Sprintf("Extracted %d characters from page (%s section)\n\n", result.Length, result.Source))
	if result.Truncated {
		response.WriteString("⚠️  Text was truncated to 1MB limit\n\n")
	}
	response.WriteString("=== Page Text ===\n")
	response.WriteString(result.Text)

	return interfaces.NewTextResponse(response.String())
}

// handleFind searches for elements matching a query
func (b *browserTool) handleFind(ctx context.Context, params BrowserParams, sessionID string) interfaces.ToolResponse {
	// Validate query
	if params.Query == "" {
		return interfaces.NewTextErrorResponse("missing query parameter for find action")
	}

	// Get browser connection
	client, err := b.connectionManager.GetOrCreate(ctx, sessionID)
	if err != nil {
		return interfaces.NewTextErrorResponse(fmt.Sprintf("Failed to connect to browser service: %v", err))
	}

	// Search for elements
	var result *browserprotocol.FindResult
	if params.TabID != "" {
		result, err = client.Find(ctx, params.Query, 100, params.TabID)
	} else {
		result, err = client.Find(ctx, params.Query, 100)
	}
	if err != nil {
		return interfaces.NewTextErrorResponse(fmt.Sprintf("Search failed: %v", err))
	}

	// No results
	if result.Total == 0 {
		return interfaces.NewTextResponse(fmt.Sprintf("No elements found matching query: %s", params.Query))
	}

	// Format response
	var response strings.Builder
	response.WriteString(fmt.Sprintf("Found %d element(s) matching query: %s\n\n", result.Total, params.Query))

	if result.Truncated {
		response.WriteString(fmt.Sprintf("⚠️  Showing first %d of %d results\n\n", len(result.Elements), result.Total))
	}

	// Show elements
	for i, elem := range result.Elements {
		response.WriteString(fmt.Sprintf("[%d] %s: %s (x:%.0f, y:%.0f)\n", i, elem.Role, elem.Name, elem.Bounds.X, elem.Bounds.Y))
	}

	return interfaces.NewTextResponse(response.String())
}

// handleClose closes the browser
func (b *browserTool) handleClose(_ context.Context, sessionID string) interfaces.ToolResponse {
	// Close connection
	if err := b.connectionManager.Close(sessionID); err != nil {
		return interfaces.NewTextErrorResponse(fmt.Sprintf("Failed to close browser: %v", err))
	}

	// Clear all caches for this session
	b.cacheMu.Lock()
	for key := range b.elementCache {
		if strings.HasPrefix(key, sessionID+"_") {
			delete(b.elementCache, key)
		}
	}
	b.cacheMu.Unlock()

	// Clear active tab tracking
	b.clearActiveTabID(sessionID)

	return interfaces.NewTextResponse("Browser closed successfully")
}

// handleTabCreate creates a new tab, optionally navigating to a URL
func (b *browserTool) handleTabCreate(ctx context.Context, params BrowserParams, sessionID string) interfaces.ToolResponse {
	// Get or create browser connection
	client, err := b.connectionManager.GetOrCreate(ctx, sessionID)
	if err != nil {
		return interfaces.NewTextErrorResponse(fmt.Sprintf("Failed to connect to browser service: %v", err))
	}

	var tab *browserprotocol.TabInfo
	if params.URL != "" {
		// Validate URL (reuse validation from handleOpen)
		parsedURL, err := url.Parse(params.URL)
		if err != nil {
			return interfaces.NewTextErrorResponse(fmt.Sprintf("invalid URL %s: %v", params.URL, err))
		}
		if parsedURL.Scheme != schemeHTTP && parsedURL.Scheme != schemeHTTPS && parsedURL.Scheme != schemeFile {
			return interfaces.NewTextErrorResponse(fmt.Sprintf("invalid URL scheme: %s (must be http, https, or file)", params.URL))
		}

		// Create tab with URL
		tab, err = client.CreateTab(ctx, params.URL)
		if err != nil {
			return interfaces.NewTextErrorResponse(fmt.Sprintf("Failed to create tab: %v", err))
		}

		// Fix 5: Update active tab tracking (new tabs become active)
		b.updateActiveTabID(sessionID, tab.ID)

		return interfaces.NewTextResponse(fmt.Sprintf("Created new tab: %s and navigated to %s (Title: %s)", tab.ID, tab.URL, tab.Title))
	}

	// Create tab without URL
	tab, err = client.CreateTab(ctx)
	if err != nil {
		return interfaces.NewTextErrorResponse(fmt.Sprintf("Failed to create tab: %v", err))
	}

	// Fix 5: Update active tab tracking (new tabs become active)
	b.updateActiveTabID(sessionID, tab.ID)

	return interfaces.NewTextResponse(fmt.Sprintf("Created new tab: %s (URL: %s, Title: %s)", tab.ID, tab.URL, tab.Title))
}

// handleTabList lists all tabs
func (b *browserTool) handleTabList(ctx context.Context, sessionID string) interfaces.ToolResponse {
	// Get or create browser connection
	client, err := b.connectionManager.GetOrCreate(ctx, sessionID)
	if err != nil {
		return interfaces.NewTextErrorResponse(fmt.Sprintf("Failed to connect to browser service: %v", err))
	}

	// List tabs
	result, err := client.ListTabs(ctx)
	if err != nil {
		return interfaces.NewTextErrorResponse(fmt.Sprintf("Failed to list tabs: %v", err))
	}

	// Format response
	var response strings.Builder
	response.WriteString(fmt.Sprintf("Total tabs: %d\n", len(result.Tabs)))
	response.WriteString(fmt.Sprintf("Active tab: %s\n\n", result.ActiveTabID))

	for _, tab := range result.Tabs {
		activeMarker := ""
		if tab.IsActive {
			activeMarker = " [ACTIVE]"
		}
		response.WriteString(fmt.Sprintf("%s%s\n", tab.ID, activeMarker))
		response.WriteString(fmt.Sprintf("  URL: %s\n", tab.URL))
		response.WriteString(fmt.Sprintf("  Title: %s\n\n", tab.Title))
	}

	return interfaces.NewTextResponse(response.String())
}

// handleTabSwitch switches to a different tab
func (b *browserTool) handleTabSwitch(ctx context.Context, params BrowserParams, sessionID string) interfaces.ToolResponse {
	// Validate tabId parameter
	if params.TabID == "" {
		return interfaces.NewTextErrorResponse("missing tabId parameter for tab_switch action")
	}

	// Get or create browser connection
	client, err := b.connectionManager.GetOrCreate(ctx, sessionID)
	if err != nil {
		return interfaces.NewTextErrorResponse(fmt.Sprintf("Failed to connect to browser service: %v", err))
	}

	// Fix 3: Get old active tab ID before switching
	oldActiveTabID, err := b.getActiveTabID(ctx, sessionID)
	if err == nil && oldActiveTabID != "" {
		// Clear cache for the old active tab
		b.clearCacheForTab(sessionID, oldActiveTabID)
	}

	// Switch tab
	if err := client.SwitchTab(ctx, params.TabID); err != nil {
		return interfaces.NewTextErrorResponse(fmt.Sprintf("Failed to switch tab: %v", err))
	}

	// Fix 3: Update active tab tracking
	b.updateActiveTabID(sessionID, params.TabID)

	return interfaces.NewTextResponse(fmt.Sprintf("Switched to tab: %s. Cache cleared for previous tab. Take a screenshot to interact with this tab.", params.TabID))
}

// handleTabClose closes a tab
func (b *browserTool) handleTabClose(ctx context.Context, params BrowserParams, sessionID string) interfaces.ToolResponse {
	// Validate tabId parameter
	if params.TabID == "" {
		return interfaces.NewTextErrorResponse("missing tabId parameter for tab_close action")
	}

	// Get or create browser connection
	client, err := b.connectionManager.GetOrCreate(ctx, sessionID)
	if err != nil {
		return interfaces.NewTextErrorResponse(fmt.Sprintf("Failed to connect to browser service: %v", err))
	}

	// Close tab
	if err := client.CloseTab(ctx, params.TabID); err != nil {
		return interfaces.NewTextErrorResponse(fmt.Sprintf("Failed to close tab: %v", err))
	}

	// Clear element cache for closed tab
	b.clearCacheForTab(sessionID, params.TabID)

	// Fix 5: Clear active tab tracking if this was the active tab
	b.activeTabMu.RLock()
	activeTabID := b.activeTabIDs[sessionID]
	b.activeTabMu.RUnlock()

	if activeTabID == params.TabID {
		b.clearActiveTabID(sessionID)
	}

	return interfaces.NewTextResponse(fmt.Sprintf("Closed tab: %s", params.TabID))
}

// handleReadPage returns accessibility tree for visible viewport elements
func (b *browserTool) handleReadPage(ctx context.Context, params BrowserParams, sessionID string) interfaces.ToolResponse {
	client, err := b.connectionManager.GetOrCreate(ctx, sessionID)
	if err != nil {
		return interfaces.NewTextErrorResponse(fmt.Sprintf("Failed to connect to browser: %v", err))
	}

	// Default to false if not specified
	interactiveOnly := false
	if params.InteractiveOnly != nil {
		interactiveOnly = *params.InteractiveOnly
	}

	// Call browser service
	var result *browserprotocol.ReadPageResult
	if params.TabID != "" {
		result, err = client.ReadPage(ctx, interactiveOnly, params.TabID)
	} else {
		result, err = client.ReadPage(ctx, interactiveOnly)
	}
	if err != nil {
		return interfaces.NewTextErrorResponse(fmt.Sprintf("Read page failed: %v", err))
	}

	// Format response
	response := formatReadPageResponse(result.Elements, result.Viewport, interactiveOnly)
	return interfaces.NewTextResponse(response)
}

// buildFrameNumberMap creates consistent frame number mapping
// Maps CDP FrameID strings to integers (f0, f1, f2, ...)
func buildFrameNumberMap(elements []browserprotocol.RawAccessibilityNode) map[string]int {
	frameMap := make(map[string]int)
	frameCounter := 0

	for _, elem := range elements {
		if elem.FrameID == "" {
			continue
		}
		if _, exists := frameMap[elem.FrameID]; !exists {
			frameMap[elem.FrameID] = frameCounter
			frameCounter++
		}
	}

	return frameMap
}

// formatAttributes converts attribute map to inline string
// Priority order: href, id, type, placeholder, name, aria-label
// Then remaining attributes alphabetically
func formatAttributes(attrs map[string]string) string {
	if len(attrs) == 0 {
		return ""
	}

	parts := make([]string, 0, len(attrs))

	// Priority attributes in specific order
	priority := []string{"href", "id", "type", "placeholder", "name", "aria-label"}
	for _, key := range priority {
		if val, exists := attrs[key]; exists {
			parts = append(parts, fmt.Sprintf(`%s=%q`, key, val))
		}
	}

	// Remaining attributes (alphabetically sorted)
	var otherKeys []string
	for key := range attrs {
		isPriority := false
		for _, pk := range priority {
			if key == pk {
				isPriority = true
				break
			}
		}
		if !isPriority {
			otherKeys = append(otherKeys, key)
		}
	}
	sort.Strings(otherKeys)

	for _, key := range otherKeys {
		parts = append(parts, fmt.Sprintf(`%s=%q`, key, attrs[key]))
	}

	return " " + strings.Join(parts, " ")
}

// formatReadPageResponse formats the read_page response for the LLM
func formatReadPageResponse(elements []browserprotocol.RawAccessibilityNode, viewport browserprotocol.BoundingBox, interactiveOnly bool) string {
	var sb strings.Builder

	filter := "all"
	if interactiveOnly {
		filter = "interactive"
	}

	sb.WriteString(fmt.Sprintf("Accessibility tree (%s elements in viewport)\n", filter))
	sb.WriteString(fmt.Sprintf("Viewport: %.0fx%.0f at scroll position (%.0f, %.0f)\n\n",
		viewport.Width, viewport.Height, viewport.X, viewport.Y))
	sb.WriteString(fmt.Sprintf("Found %d element(s):\n\n", len(elements)))

	// Build frame number map for consistent reference IDs
	frameMap := buildFrameNumberMap(elements)

	for _, elem := range elements {
		// Format: - role "name" [ref=fX_ref_Y] (x=X,y=Y) attrs...

		// 1. Role
		sb.WriteString(fmt.Sprintf("- %s", elem.Role))

		// 2. Name (quoted, if present)
		if elem.Name != "" {
			sb.WriteString(fmt.Sprintf(" %q", elem.Name))
		}

		// 3. Reference ID [ref=f{frameNum}_ref_{backendID}]
		frameNum := 0
		if elem.FrameID != "" {
			if num, exists := frameMap[elem.FrameID]; exists {
				frameNum = num
			}
		}
		sb.WriteString(fmt.Sprintf(" [ref=f%d_ref_%d]", frameNum, elem.BackendID))

		// 4. Coordinates (x=X,y=Y)
		sb.WriteString(fmt.Sprintf(" (x=%.0f,y=%.0f)", elem.Bounds.X, elem.Bounds.Y))

		// 5. Attributes (inline, space-separated)
		if len(elem.Attributes) > 0 {
			sb.WriteString(formatAttributes(elem.Attributes))
		}

		sb.WriteString("\n")
	}

	return sb.String()
}

// getContextInfo extracts context information needed for tool execution
func (b *browserTool) getContextInfo(ctx context.Context) (sessionID, messageID, sessionStorageDir string, err error) {
	sessionIDVal := ctx.Value(interfaces.SessionIDContextKey)
	messageIDVal := ctx.Value(interfaces.MessageIDContextKey)
	sessionStorageDirVal := ctx.Value(interfaces.SessionStorageContextKey)

	if sessionIDVal == nil {
		return "", "", "", fmt.Errorf("session ID not found in context")
	}
	if messageIDVal == nil {
		return "", "", "", fmt.Errorf("message ID not found in context")
	}
	if sessionStorageDirVal == nil {
		return "", "", "", fmt.Errorf("session storage directory not found in context")
	}

	sessionID, ok := sessionIDVal.(string)
	if !ok {
		return "", "", "", fmt.Errorf("session ID context value is not a string")
	}

	messageID, ok = messageIDVal.(string)
	if !ok {
		return "", "", "", fmt.Errorf("message ID context value is not a string")
	}

	sessionStorageDir, ok = sessionStorageDirVal.(string)
	if !ok {
		return "", "", "", fmt.Errorf("session storage directory context value is not a string")
	}

	return sessionID, messageID, sessionStorageDir, nil
}

// loadBrowserDescription loads the browser tool description
func loadBrowserDescription() string {
	return tools.LoadToolDescription("browser")
}

// getBaseURL returns the base URL for file access
func getBaseURL() string {
	// Frontend URL for serving session files (screenshots, etc)
	// Try FRONTEND_URL first, then BASE_URL, then default to localhost:3020
	baseURL := os.Getenv("FRONTEND_URL")
	if baseURL == "" {
		baseURL = os.Getenv("BASE_URL")
	}
	if baseURL == "" {
		baseURL = "http://localhost:3020"
	}
	return baseURL
}
