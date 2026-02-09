package browser

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"slices"
	"sort"
	"strings"
	"sync"
	"time"

	browserprotocol "github.com/sarathmenon/browser-service/pkg/protocol"

	browserpkg "mix/internal/browser"
	"mix/internal/llm/interfaces"
	"mix/internal/llm/tools"
	"mix/internal/llm/tools/browser/vision"
	"mix/internal/logging"
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

// ClientFactory creates browser clients for sessions
type ClientFactory func(sessionID string) (browserpkg.Client, error)

// browserTool implements the Browser tool for LLM-driven browser automation
type browserTool struct {
	permissions          permission.Service
	connectionManager    *ConnectionManager
	sessionConfig        session.Config
	baseURL              string
	elementCache         map[string]map[int]int64 // sessionID_tabID → visualIndex → backendID
	cacheMu              sync.RWMutex             // Protect element cache
	browserMode          string                   // "tunnel" or "service"
	clientFactory        ClientFactory            // Factory for creating browser clients
	tunnelRegistryGetter func() interface{}       // Getter for tunnel registry (allows late initialization)
	browserServiceURL    string                   // URL for browser-service
	tunnelClients        map[string]*TunnelClientWrapper // Cache tunnel clients per session
	tunnelClientsMu      sync.RWMutex                    // Protect tunnel clients cache
}

// NewBrowserTool creates a new browser tool instance
func NewBrowserTool(permissions permission.Service, browserServiceURL string, sessionConfig session.Config, browserMode string, clientFactory ClientFactory, connectionManager interface{}, tunnelRegistryGetter func() interface{}) interfaces.BaseTool {
	// Default to service mode if not specified
	if browserMode == "" {
		browserMode = browserpkg.ModeService
	}

	// Type assert connection manager if provided
	var connMgr *ConnectionManager
	if connectionManager != nil {
		if cm, ok := connectionManager.(*ConnectionManager); ok {
			connMgr = cm
		}
	}

	// If no connection manager provided, create one (backward compatibility)
	if connMgr == nil && browserServiceURL != "" {
		connMgr = NewConnectionManager(browserServiceURL)
	}

	return &browserTool{
		permissions:          permissions,
		connectionManager:    connMgr,
		sessionConfig:        sessionConfig,
		baseURL:              getBaseURL(),
		elementCache:         make(map[string]map[int]int64),
		browserMode:          browserMode,
		clientFactory:        clientFactory,
		tunnelRegistryGetter: tunnelRegistryGetter,
		browserServiceURL:    browserServiceURL,
		tunnelClients:        make(map[string]*TunnelClientWrapper),
	}
}

// getClient creates a browser client using the factory pattern
// Supports both tunnel and service modes based on configuration
// Returns BrowserClient interface that works with both modes
func (b *browserTool) getClient(ctx context.Context, sessionID string) (BrowserClient, error) {
	if b.browserMode == browserpkg.ModeService {
		return b.connectionManager.GetOrCreate(ctx, sessionID)
	}

	// Tunnel mode: get or create cached client
	b.tunnelClientsMu.Lock()
	defer b.tunnelClientsMu.Unlock()

	// Return existing client if cached
	if client, exists := b.tunnelClients[sessionID]; exists {
		return client, nil
	}

	// Create new client and cache it
	var tunnelRegistry interface{}
	if b.tunnelRegistryGetter != nil {
		tunnelRegistry = b.tunnelRegistryGetter()
	}

	client := NewTunnelClientWrapper(tunnelRegistry, sessionID)
	b.tunnelClients[sessionID] = client
	return client, nil
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
				"enum":        []string{ActionOpen, ActionScreenshot, ActionReadPage, ActionClick, ActionType, ActionScroll, ActionUpload, ActionGetText, ActionFind, ActionClose, ActionRightClick, ActionDoubleClick, ActionTripleClick, ActionDrag, ActionFormInput, ActionGoBack, ActionGoForward, ActionTabCreate, ActionTabList, ActionTabSwitch, ActionTabClose, ActionWait, ActionKey, ActionScrollTo, ActionSequence},
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
				"description": "Tab ID to operate on. Required for all tab-specific actions (open, screenshot, read_page, click, type, scroll, upload, get_text, find, form_input, go_back, go_forward, key, scroll_to, sequence, wait, tab_switch, tab_close). Not required for tab_create, tab_list, or close actions.",
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
			"key": map[string]any{
				"type":        "string",
				"description": "Keyboard key(s) to press (for key action). Space-separated sequence. Examples: 'Enter', 'cmd+a', 'Backspace Backspace'",
			},
			"actions": map[string]any{
				"type":        "array",
				"description": "Array of actions to execute in sequence (for action batching)",
				"items": map[string]any{
					"type": "object",
				},
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

	// Log all browser tool invocations with key parameters
	sessionID, _, err := b.getContextInfo(ctx)
	if err == nil {
		logging.Debug("Browser tool invoked",
			"action", params.Action,
			"tabID", params.TabID,
			"sessionID", sessionID,
			"browserMode", b.browserMode)
	}

	// Validate tabId requirement for tab-interaction actions
	requiresTabID := []string{
		ActionOpen, ActionScreenshot, ActionReadPage, ActionClick, ActionType,
		ActionScroll, ActionUpload, ActionGetText, ActionFind, ActionRightClick,
		ActionDoubleClick, ActionTripleClick, ActionDrag, ActionFormInput,
		ActionGoBack, ActionGoForward, ActionKey, ActionScrollTo, ActionSequence, ActionWait,
		ActionTabSwitch, ActionTabClose,
	}
	if slices.Contains(requiresTabID, params.Action) && params.TabID == "" {
		return interfaces.NewTextErrorResponse(fmt.Sprintf("%s action requires tabId parameter", params.Action)), nil
	}

	// Get session context
	sessionID, sessionStorageDir, err := b.getContextInfo(ctx)
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
	case ActionKey:
		return b.handleKey(ctx, params, sessionID), nil
	case ActionScrollTo:
		return b.handleScrollTo(ctx, params, sessionID), nil
	case ActionSequence:
		return b.handleActionSequence(ctx, params, sessionID, sessionStorageDir), nil
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
	client, err := b.getClient(ctx, sessionID)
	if err != nil {
		return interfaces.NewTextErrorResponse(fmt.Sprintf("Failed to get browser client: %v", err))
	}

	// Navigate (tabID is always required and validated)
	result, err := client.Navigate(ctx, params.URL, params.TabID)
	if err != nil {
		return interfaces.NewTextErrorResponse(fmt.Sprintf("Navigation failed: %v", err))
	}

	// Clear element cache on navigation
	b.clearCacheForTab(sessionID, params.TabID)

	return interfaces.NewTextResponse(fmt.Sprintf("Successfully navigated to %s (Frame ID: %s)", params.URL, result.FrameID))
}

// handleScreenshot captures a screenshot
func (b *browserTool) handleScreenshot(ctx context.Context, params BrowserParams, sessionID, sessionStorageDir string) interfaces.ToolResponse {
	// Get or create browser connection
	client, err := b.getClient(ctx, sessionID)
	if err != nil {
		return interfaces.NewTextErrorResponse(fmt.Sprintf("Failed to get browser client: %v", err))
	}

	// Request raw screenshot with accessibility data (tabID is always required and validated)
	screenshotParams := browserprotocol.ScreenshotParams{
		Format:   "png",
		FullPage: false,
		Raw:      true,   // Request raw accessibility tree
		TabID:    &params.TabID,
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
		// Clear cache for blank/unloaded pages (tabID is always provided)
		b.clearCacheForTab(sessionID, params.TabID)

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
func (b *browserTool) cacheElementMapping(_ context.Context, sessionID, tabID string, rawNodes []browserprotocol.RawAccessibilityNode, viewport browserprotocol.ViewportBounds) {
	// tabID is required - cannot cache without it
	if tabID == "" {
		return
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

	// Create cache key using tabID
	cacheKey := sessionID + "_" + tabID

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
func (b *browserTool) getBackendIDFromCache(_ context.Context, sessionID, tabID string, visualIndex int) (int64, error) {
	// tabID is required
	if tabID == "" {
		return 0, fmt.Errorf("tabId is required for element lookup")
	}

	// Create cache key using tabID
	cacheKey := sessionID + "_" + tabID

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


// handleClick clicks an element
func (b *browserTool) handleClick(ctx context.Context, params BrowserParams, sessionID string) interfaces.ToolResponse {
	// Get browser connection
	client, err := b.getClient(ctx, sessionID)
	if err != nil {
		return interfaces.NewTextErrorResponse(fmt.Sprintf("Failed to get browser client: %v", err))
	}

	// Fix 4: Look up BackendID from cache with validation
	backendID, err := b.getBackendIDFromCache(ctx, sessionID, params.TabID, params.Index)
	if err != nil {
		return interfaces.NewTextErrorResponse(fmt.Sprintf("Element not found in cache: %v. Please take a screenshot first.", err))
	}

	// Click by BackendID (tabID is always required and validated)
	err = client.ClickByBackendID(ctx, backendID, params.TabID)
	if err != nil {
		return interfaces.NewTextErrorResponse(fmt.Sprintf("Click failed: %v", err))
	}

	return interfaces.NewTextResponse(fmt.Sprintf("Successfully clicked element %d", params.Index))
}

// handleRightClick right-clicks an element
func (b *browserTool) handleRightClick(ctx context.Context, params BrowserParams, sessionID string) interfaces.ToolResponse {
	// Get browser connection
	client, err := b.getClient(ctx, sessionID)
	if err != nil {
		return interfaces.NewTextErrorResponse(fmt.Sprintf("Failed to get browser client: %v", err))
	}

	// Fix 4: Look up BackendID from cache with validation
	backendID, err := b.getBackendIDFromCache(ctx, sessionID, params.TabID, params.Index)
	if err != nil {
		return interfaces.NewTextErrorResponse(fmt.Sprintf("Element not found in cache: %v. Please take a screenshot first.", err))
	}

	// Right-click by BackendID (tabID is always required and validated)
	err = client.RightClickByBackendID(ctx, backendID, params.TabID)
	if err != nil {
		return interfaces.NewTextErrorResponse(fmt.Sprintf("Right-click failed: %v", err))
	}

	return interfaces.NewTextResponse(fmt.Sprintf("Successfully right-clicked element %d", params.Index))
}

// handleDoubleClick double-clicks an element
func (b *browserTool) handleDoubleClick(ctx context.Context, params BrowserParams, sessionID string) interfaces.ToolResponse {
	// Get browser connection
	client, err := b.getClient(ctx, sessionID)
	if err != nil {
		return interfaces.NewTextErrorResponse(fmt.Sprintf("Failed to get browser client: %v", err))
	}

	// Fix 4: Look up BackendID from cache with validation
	backendID, err := b.getBackendIDFromCache(ctx, sessionID, params.TabID, params.Index)
	if err != nil {
		return interfaces.NewTextErrorResponse(fmt.Sprintf("Element not found in cache: %v. Please take a screenshot first.", err))
	}

	// Double-click by BackendID (tabID is always required and validated)
	err = client.DoubleClickByBackendID(ctx, backendID, params.TabID)
	if err != nil {
		return interfaces.NewTextErrorResponse(fmt.Sprintf("Double-click failed: %v", err))
	}

	return interfaces.NewTextResponse(fmt.Sprintf("Successfully double-clicked element %d", params.Index))
}

// handleTripleClick triple-clicks an element
func (b *browserTool) handleTripleClick(ctx context.Context, params BrowserParams, sessionID string) interfaces.ToolResponse {
	// Get browser connection
	client, err := b.getClient(ctx, sessionID)
	if err != nil {
		return interfaces.NewTextErrorResponse(fmt.Sprintf("Failed to get browser client: %v", err))
	}

	// Fix 4: Look up BackendID from cache with validation
	backendID, err := b.getBackendIDFromCache(ctx, sessionID, params.TabID, params.Index)
	if err != nil {
		return interfaces.NewTextErrorResponse(fmt.Sprintf("Element not found in cache: %v. Please take a screenshot first.", err))
	}

	// Triple-click by BackendID (tabID is always required and validated)
	err = client.TripleClickByBackendID(ctx, backendID, params.TabID)
	if err != nil {
		return interfaces.NewTextErrorResponse(fmt.Sprintf("Triple-click failed: %v", err))
	}

	return interfaces.NewTextResponse(fmt.Sprintf("Successfully triple-clicked element %d", params.Index))
}

// handleDrag performs a drag operation
func (b *browserTool) handleDrag(ctx context.Context, params BrowserParams, sessionID string) interfaces.ToolResponse {
	// Get browser connection
	client, err := b.getClient(ctx, sessionID)
	if err != nil {
		return interfaces.NewTextErrorResponse(fmt.Sprintf("Failed to get browser client: %v", err))
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

	// Perform drag (tabID is always required and validated)
	err = client.Drag(ctx, params.FromIndex, params.ToIndex, params.FromX, params.FromY, params.ToX, params.ToY, duration, params.TabID)
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
	client, err := b.getClient(ctx, sessionID)
	if err != nil {
		return interfaces.NewTextErrorResponse(fmt.Sprintf("Failed to get browser client: %v", err))
	}

	// Set form input value (tabID is always required and validated)
	err = client.FormInput(ctx, params.Index, params.Value, params.TabID)
	if err != nil {
		return interfaces.NewTextErrorResponse(fmt.Sprintf("Form input failed: %v", err))
	}

	return interfaces.NewTextResponse(fmt.Sprintf("Successfully set value in element %d", params.Index))
}

// handleGoBack navigates backward in browser history
func (b *browserTool) handleGoBack(ctx context.Context, params BrowserParams, sessionID string) interfaces.ToolResponse {
	// Get browser connection
	client, err := b.getClient(ctx, sessionID)
	if err != nil {
		return interfaces.NewTextErrorResponse(fmt.Sprintf("Failed to get browser client: %v", err))
	}

	// Navigate back (tabID is always required and validated)
	resultURL, err := client.GoBack(ctx, params.TabID)
	if err != nil {
		return interfaces.NewTextErrorResponse(fmt.Sprintf("Go back failed: %v", err))
	}

	return interfaces.NewTextResponse(fmt.Sprintf("Successfully navigated back to: %s", resultURL))
}

// handleGoForward navigates forward in browser history
func (b *browserTool) handleGoForward(ctx context.Context, params BrowserParams, sessionID string) interfaces.ToolResponse {
	// Get browser connection
	client, err := b.getClient(ctx, sessionID)
	if err != nil {
		return interfaces.NewTextErrorResponse(fmt.Sprintf("Failed to get browser client: %v", err))
	}

	// Navigate forward (tabID is always required and validated)
	resultURL, err := client.GoForward(ctx, params.TabID)
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

	client, err := b.getClient(ctx, sessionID)
	if err != nil {
		return interfaces.NewTextErrorResponse(fmt.Sprintf("Failed to get browser client: %v", err))
	}

	// Wait (tabID is always required and validated)
	err = client.Wait(ctx, params.Duration, params.TabID)
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
	client, err := b.getClient(ctx, sessionID)
	if err != nil {
		return interfaces.NewTextErrorResponse(fmt.Sprintf("Failed to get browser client: %v", err))
	}

	// Type text (tabID is always required and validated)
	err = client.Type(ctx, params.Index, params.Text, params.TabID)
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
	client, err := b.getClient(ctx, sessionID)
	if err != nil {
		return interfaces.NewTextErrorResponse(fmt.Sprintf("Failed to get browser client: %v", err))
	}

	// Scroll (tabID is always required and validated)
	err = client.Scroll(ctx, params.Direction, amount, params.TabID)
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
	client, err := b.getClient(ctx, sessionID)
	if err != nil {
		return interfaces.NewTextErrorResponse(fmt.Sprintf("Failed to get browser client: %v", err))
	}

	// Upload file (tabID is always required and validated)
	result, err := client.UploadFile(ctx, params.Index, []string{absolutePath}, params.TabID)
	if err != nil {
		return interfaces.NewTextErrorResponse(fmt.Sprintf("Upload failed: %v", err))
	}

	return interfaces.NewTextResponse(fmt.Sprintf("Successfully uploaded %d file(s): %s", result.FilesUploaded, strings.Join(result.FileNames, ", ")))
}

const defaultTextStrategy = "auto"

// handleGetText extracts text content from the page
func (b *browserTool) handleGetText(ctx context.Context, params BrowserParams, sessionID string) interfaces.ToolResponse {
	// Default strategy to auto
	strategy := params.Strategy
	if strategy == "" {
		strategy = defaultTextStrategy
	}

	// Get browser connection
	client, err := b.getClient(ctx, sessionID)
	if err != nil {
		return interfaces.NewTextErrorResponse(fmt.Sprintf("Failed to get browser client: %v", err))
	}

	// Extract text (tabID is always required and validated)
	result, err := client.GetText(ctx, strategy, params.TabID)
	if err != nil {
		return interfaces.NewTextErrorResponse(fmt.Sprintf("Text extraction failed: %v", err))
	}

	// Format response
	var response strings.Builder
	fmt.Fprintf(&response, "Extracted %d characters from page (%s section)\n\n", result.Length, result.Source)
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
	client, err := b.getClient(ctx, sessionID)
	if err != nil {
		return interfaces.NewTextErrorResponse(fmt.Sprintf("Failed to get browser client: %v", err))
	}

	// Search for elements (tabID is always required and validated)
	result, err := client.Find(ctx, params.Query, 100, params.TabID)
	if err != nil {
		return interfaces.NewTextErrorResponse(fmt.Sprintf("Search failed: %v", err))
	}

	// No results
	if result.Total == 0 {
		return interfaces.NewTextResponse(fmt.Sprintf("No elements found matching query: %s", params.Query))
	}

	// Format response
	var response strings.Builder
	fmt.Fprintf(&response, "Found %d element(s) matching query: %s\n\n", result.Total, params.Query)

	if result.Truncated {
		fmt.Fprintf(&response, "⚠️  Showing first %d of %d results\n\n", len(result.Elements), result.Total)
	}

	// Show elements
	for i, elem := range result.Elements {
		fmt.Fprintf(&response, "[%d] %s: %s (x:%.0f, y:%.0f)\n", i+1, elem.Role, elem.Name, elem.Bounds.X, elem.Bounds.Y)
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

	return interfaces.NewTextResponse("Browser closed successfully")
}

// handleTabCreate creates a new tab, optionally navigating to a URL
func (b *browserTool) handleTabCreate(ctx context.Context, params BrowserParams, sessionID string) interfaces.ToolResponse {
	// Get or create browser connection
	client, err := b.getClient(ctx, sessionID)
	if err != nil {
		return interfaces.NewTextErrorResponse(fmt.Sprintf("Failed to get browser client: %v", err))
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

		return interfaces.NewTextResponse(fmt.Sprintf("Created new tab: %s and navigated to %s (Title: %s)", tab.ID, tab.URL, tab.Title))
	}

	// Create tab without URL
	tab, err = client.CreateTab(ctx)
	if err != nil {
		return interfaces.NewTextErrorResponse(fmt.Sprintf("Failed to create tab: %v", err))
	}

	return interfaces.NewTextResponse(fmt.Sprintf("Created new tab: %s (URL: %s, Title: %s)", tab.ID, tab.URL, tab.Title))
}

// handleTabList lists all tabs
func (b *browserTool) handleTabList(ctx context.Context, sessionID string) interfaces.ToolResponse {
	// Get or create browser connection
	client, err := b.getClient(ctx, sessionID)
	if err != nil {
		return interfaces.NewTextErrorResponse(fmt.Sprintf("Failed to get browser client: %v", err))
	}

	// List tabs
	result, err := client.ListTabs(ctx)
	if err != nil {
		return interfaces.NewTextErrorResponse(fmt.Sprintf("Failed to list tabs: %v", err))
	}

	// Format response
	var response strings.Builder
	fmt.Fprintf(&response, "Total tabs: %d\n", len(result.Tabs))
	fmt.Fprintf(&response, "Active tab: %s\n\n", result.ActiveTabID)

	for _, tab := range result.Tabs {
		activeMarker := ""
		if tab.IsActive {
			activeMarker = " [ACTIVE]"
		}
		fmt.Fprintf(&response, "%s%s\n", tab.ID, activeMarker)
		fmt.Fprintf(&response, "  URL: %s\n", tab.URL)
		fmt.Fprintf(&response, "  Title: %s\n\n", tab.Title)
	}

	return interfaces.NewTextResponse(response.String())
}

// handleTabSwitch switches to a different tab
func (b *browserTool) handleTabSwitch(ctx context.Context, params BrowserParams, sessionID string) interfaces.ToolResponse {
	// tabId already validated in Run()

	// Get or create browser connection
	client, err := b.getClient(ctx, sessionID)
	if err != nil {
		return interfaces.NewTextErrorResponse(fmt.Sprintf("Failed to get browser client: %v", err))
	}

	// Switch tab
	if err := client.SwitchTab(ctx, params.TabID); err != nil {
		return interfaces.NewTextErrorResponse(fmt.Sprintf("Failed to switch tab: %v", err))
	}

	return interfaces.NewTextResponse(fmt.Sprintf("Switched to tab: %s. Take a screenshot to interact with this tab.", params.TabID))
}

// handleTabClose closes a tab
func (b *browserTool) handleTabClose(ctx context.Context, params BrowserParams, sessionID string) interfaces.ToolResponse {
	// tabId already validated in Run()

	// Get or create browser connection
	client, err := b.getClient(ctx, sessionID)
	if err != nil {
		return interfaces.NewTextErrorResponse(fmt.Sprintf("Failed to get browser client: %v", err))
	}

	// Close tab
	if err := client.CloseTab(ctx, params.TabID); err != nil {
		return interfaces.NewTextErrorResponse(fmt.Sprintf("Failed to close tab: %v", err))
	}

	// Clear element cache for closed tab
	b.clearCacheForTab(sessionID, params.TabID)

	return interfaces.NewTextResponse(fmt.Sprintf("Closed tab: %s", params.TabID))
}

// handleReadPage returns accessibility tree for visible viewport elements
func (b *browserTool) handleReadPage(ctx context.Context, params BrowserParams, sessionID string) interfaces.ToolResponse {
	client, err := b.getClient(ctx, sessionID)
	if err != nil {
		return interfaces.NewTextErrorResponse(fmt.Sprintf("Failed to get browser client: %v", err))
	}

	// Default to false if not specified
	interactiveOnly := false
	if params.InteractiveOnly != nil {
		interactiveOnly = *params.InteractiveOnly
	}

	// Call browser service (tabID is always required and validated)
	result, err := client.ReadPage(ctx, interactiveOnly, params.TabID)
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
		if !slices.Contains(priority, key) {
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

	fmt.Fprintf(&sb, "Accessibility tree (%s elements in viewport)\n", filter)
	fmt.Fprintf(&sb, "Viewport: %.0fx%.0f at scroll position (%.0f, %.0f)\n\n",
		viewport.Width, viewport.Height, viewport.X, viewport.Y)
	fmt.Fprintf(&sb, "Found %d element(s):\n\n", len(elements))

	// Build frame number map for consistent reference IDs
	frameMap := buildFrameNumberMap(elements)

	for _, elem := range elements {
		// Format: - role "name" [ref=fX_ref_Y] (x=X,y=Y) attrs...

		// 1. Role
		fmt.Fprintf(&sb, "- %s", elem.Role)

		// 2. Name (quoted, if present)
		if elem.Name != "" {
			fmt.Fprintf(&sb, " %q", elem.Name)
		}

		// 3. Reference ID [ref=f{frameNum}_ref_{backendID}]
		frameNum := 0
		if elem.FrameID != "" {
			if num, exists := frameMap[elem.FrameID]; exists {
				frameNum = num
			}
		}
		fmt.Fprintf(&sb, " [ref=f%d_ref_%d]", frameNum, elem.BackendID)

		// 4. Coordinates (x=X,y=Y)
		fmt.Fprintf(&sb, " (x=%.0f,y=%.0f)", elem.Bounds.X, elem.Bounds.Y)

		// 5. Attributes (inline, space-separated)
		if len(elem.Attributes) > 0 {
			sb.WriteString(formatAttributes(elem.Attributes))
		}

		sb.WriteString("\n")
	}

	return sb.String()
}

// handleKey presses keyboard keys
func (b *browserTool) handleKey(ctx context.Context, params BrowserParams, sessionID string) interfaces.ToolResponse {
	if params.Key == "" {
		return interfaces.NewTextErrorResponse("missing key parameter for key action")
	}

	client, err := b.getClient(ctx, sessionID)
	if err != nil {
		return interfaces.NewTextErrorResponse(fmt.Sprintf("Failed to get browser client: %v", err))
	}

	// Press key (tabID is always required and validated)
	err = client.PressKey(ctx, params.Key, params.TabID)
	if err != nil {
		return interfaces.NewTextErrorResponse(fmt.Sprintf("Key press failed: %v", err))
	}

	return interfaces.NewTextResponse(fmt.Sprintf("Successfully pressed key(s): %s", params.Key))
}

// handleScrollTo scrolls an element into view
func (b *browserTool) handleScrollTo(ctx context.Context, params BrowserParams, sessionID string) interfaces.ToolResponse {
	client, err := b.getClient(ctx, sessionID)
	if err != nil {
		return interfaces.NewTextErrorResponse(fmt.Sprintf("Failed to get browser client: %v", err))
	}

	// Get BackendID from cache
	backendID, err := b.getBackendIDFromCache(ctx, sessionID, params.TabID, params.Index)
	if err != nil {
		return interfaces.NewTextErrorResponse(fmt.Sprintf("Element not found in cache: %v. Please take a screenshot first.", err))
	}

	// Scroll element into view (tabID is always required and validated)
	err = client.ScrollIntoViewByBackendID(ctx, backendID, params.TabID)
	if err != nil {
		return interfaces.NewTextErrorResponse(fmt.Sprintf("Scroll to element failed: %v", err))
	}

	return interfaces.NewTextResponse(fmt.Sprintf("Successfully scrolled element %d into view", params.Index))
}

// handleActionSequence executes multiple actions in sequence
func (b *browserTool) handleActionSequence(ctx context.Context, params BrowserParams, sessionID, sessionStorageDir string) interfaces.ToolResponse {
	if len(params.Actions) == 0 {
		return interfaces.NewTextErrorResponse("missing actions array for action sequence")
	}

	client, err := b.getClient(ctx, sessionID)
	if err != nil {
		return interfaces.NewTextErrorResponse(fmt.Sprintf("Failed to get browser client: %v", err))
	}

	results := make([]SubActionResult, len(params.Actions))
	successCount := 0
	var lastErr error
	containsScreenshot := false

	// Execute each action in sequence
	for i := range params.Actions {
		subAction := &params.Actions[i]
		result := SubActionResult{
			Index: i,
			Type:  subAction.Type,
		}

		// Check if sequence contains screenshot
		if subAction.Type == "screenshot" {
			containsScreenshot = true
		}

		// Execute sub-action
		err := b.executeSubAction(ctx, client, *subAction, sessionID, params.TabID)
		if err != nil {
			result.Success = false
			result.Error = err.Error()
			lastErr = err
			results[i] = result
			// Fail-fast: stop on first error
			break
		}

		result.Success = true
		successCount++
		results[i] = result

		// Inter-action delay
		if i < len(params.Actions)-1 {
			delay := getDelayForAction(subAction.Type)
			time.Sleep(delay)
		}
	}

	// Take automatic screenshot if sequence doesn't contain one
	var screenshotFile string
	if !containsScreenshot && lastErr == nil {
		screenshotParams := browserprotocol.ScreenshotParams{
			Format:   "png",
			FullPage: false,
			Raw:      true,
			TabID:    &params.TabID, // TabID is always required and validated
		}

		screenshotResult, err := client.Screenshot(ctx, screenshotParams)
		if err == nil && screenshotResult != nil {
			filename, err := saveScreenshot(screenshotResult.Data, sessionStorageDir)
			if err == nil {
				screenshotFile = filename
				// Cache element mappings
				if screenshotResult.RawNodes != nil && screenshotResult.RawViewport != nil {
					b.cacheElementMapping(ctx, sessionID, params.TabID, screenshotResult.RawNodes, *screenshotResult.RawViewport)
				}
			}
		}
	}

	// Format response
	var response strings.Builder
	fmt.Fprintf(&response, "Action Sequence Results (%d/%d successful)\n\n", successCount, len(params.Actions))

	for _, result := range results {
		if result.Type == "" {
			continue // Skipped action
		}

		status := "✓ Success"
		if !result.Success {
			status = fmt.Sprintf("✗ Failed: %s", result.Error)
		}

		fmt.Fprintf(&response, "[%d] %s %s\n", result.Index, result.Type, status)
	}

	if screenshotFile != "" {
		fmt.Fprintf(&response, "\nScreenshot: %s\n", formatScreenshotResponse(screenshotFile, sessionID, b.baseURL))
	}

	if lastErr != nil {
		response.WriteString("\nSequence stopped early due to error.\n")
	}

	return interfaces.NewTextResponse(response.String())
}

// executeSubAction executes a single sub-action
func (b *browserTool) executeSubAction(ctx context.Context, client BrowserClient, action SubAction, sessionID, tabID string) error {
	switch action.Type {
	case "left_click":
		return b.executeClick(ctx, client, action, sessionID, tabID)
	case "right_click":
		return b.executeRightClick(ctx, client, action, sessionID, tabID)
	case "double_click":
		return b.executeDoubleClick(ctx, client, action, sessionID, tabID)
	case "triple_click":
		return b.executeTripleClick(ctx, client, action, sessionID, tabID)
	case "type":
		return b.executeType(ctx, client, action, sessionID, tabID)
	case "key":
		return client.PressKey(ctx, action.Key, tabID)
	case "scroll":
		amount := action.ScrollAmount
		if amount == 0 {
			amount = 100
		}
		return client.Scroll(ctx, action.Direction, amount, tabID)
	case "scroll_to":
		if action.Index == nil {
			return fmt.Errorf("index required for scroll_to action")
		}
		backendID, err := b.getBackendIDFromCache(ctx, sessionID, tabID, *action.Index)
		if err != nil {
			return err
		}
		return client.ScrollIntoViewByBackendID(ctx, backendID, tabID)
	case "form_input":
		if action.Index == nil {
			return fmt.Errorf("index required for form_input action")
		}
		return client.FormInput(ctx, *action.Index, action.Value, tabID)
	case "wait":
		return client.Wait(ctx, action.Duration, tabID)
	case "left_click_drag":
		return client.Drag(ctx, action.FromIndex, action.ToIndex, action.FromX, action.FromY, action.ToX, action.ToY, nil, tabID)
	case "screenshot":
		// Skip screenshot in sequence - handled at end
		return nil
	default:
		return fmt.Errorf("unknown sub-action type: %s", action.Type)
	}
}

// executeClick executes a click sub-action
func (b *browserTool) executeClick(ctx context.Context, client BrowserClient, action SubAction, sessionID, tabID string) error {
	if action.Index == nil {
		return fmt.Errorf("index required for click action")
	}
	backendID, err := b.getBackendIDFromCache(ctx, sessionID, tabID, *action.Index)
	if err != nil {
		return err
	}
	return client.ClickByBackendID(ctx, backendID, tabID)
}

// executeRightClick executes a right-click sub-action
func (b *browserTool) executeRightClick(ctx context.Context, client BrowserClient, action SubAction, sessionID, tabID string) error {
	if action.Index == nil {
		return fmt.Errorf("index required for right_click action")
	}
	backendID, err := b.getBackendIDFromCache(ctx, sessionID, tabID, *action.Index)
	if err != nil {
		return err
	}
	return client.RightClickByBackendID(ctx, backendID, tabID)
}

// executeDoubleClick executes a double-click sub-action
func (b *browserTool) executeDoubleClick(ctx context.Context, client BrowserClient, action SubAction, sessionID, tabID string) error {
	if action.Index == nil {
		return fmt.Errorf("index required for double_click action")
	}
	backendID, err := b.getBackendIDFromCache(ctx, sessionID, tabID, *action.Index)
	if err != nil {
		return err
	}
	return client.DoubleClickByBackendID(ctx, backendID, tabID)
}

// executeTripleClick executes a triple-click sub-action
func (b *browserTool) executeTripleClick(ctx context.Context, client BrowserClient, action SubAction, sessionID, tabID string) error {
	if action.Index == nil {
		return fmt.Errorf("index required for triple_click action")
	}
	backendID, err := b.getBackendIDFromCache(ctx, sessionID, tabID, *action.Index)
	if err != nil {
		return err
	}
	return client.TripleClickByBackendID(ctx, backendID, tabID)
}

// executeType executes a type sub-action
func (b *browserTool) executeType(ctx context.Context, client BrowserClient, action SubAction, _, tabID string) error {
	if action.Index == nil {
		return fmt.Errorf("index required for type action")
	}
	return client.Type(ctx, *action.Index, action.Text, tabID)
}

// getDelayForAction returns appropriate inter-action delay
func getDelayForAction(actionType string) time.Duration {
	switch actionType {
	case "left_click", "right_click", "double_click", "triple_click":
		return 100 * time.Millisecond
	case "type", "form_input":
		return 50 * time.Millisecond
	default:
		return 50 * time.Millisecond
	}
}

// getContextInfo extracts context information needed for tool execution
func (b *browserTool) getContextInfo(ctx context.Context) (sessionID, sessionStorageDir string, err error) {
	sessionIDVal := ctx.Value(interfaces.SessionIDContextKey)
	sessionStorageDirVal := ctx.Value(interfaces.SessionStorageContextKey)

	if sessionIDVal == nil {
		return "", "", fmt.Errorf("session ID not found in context")
	}
	if sessionStorageDirVal == nil {
		return "", "", fmt.Errorf("session storage directory not found in context")
	}

	sessionID, ok := sessionIDVal.(string)
	if !ok {
		return "", "", fmt.Errorf("session ID context value is not a string")
	}

	sessionStorageDir, ok = sessionStorageDirVal.(string)
	if !ok {
		return "", "", fmt.Errorf("session storage directory context value is not a string")
	}

	return sessionID, sessionStorageDir, nil
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
