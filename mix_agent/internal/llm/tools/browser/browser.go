package browser

import (
	"context"
	"encoding/json"
	"fmt"
	"slices"
	"sync"
	"time"

	browserpkg "mix/internal/browser"
	"mix/internal/llm/interfaces"
	"mix/internal/permission"
	"mix/internal/session"
	"mix/internal/storage"
)

const (
	BrowserToolName = "Browser"

	// DefaultRequestTimeout is the default timeout for browser operations
	DefaultRequestTimeout = 30 * time.Second
)

// ClientFactory creates browser clients for sessions
type ClientFactory func(sessionID string) (browserpkg.Client, error)

// clientEntry holds a client with sync.Once for guaranteed single creation
type clientEntry struct {
	client BrowserClient
	once   sync.Once
	err    error
}

// browserTool implements the Browser tool for LLM-driven browser automation
type browserTool struct {
	permissions          permission.Service
	sessions             session.Service // Session service to fetch per-session browser config
	connectionManager    *ConnectionManager
	sessionConfig        session.Config
	baseURL              string
	browserMode          string                          // Global browser mode (fallback for legacy sessions)
	clientFactory        ClientFactory                   // Factory for creating browser clients
	tunnelRegistryGetter func() interface{}              // Getter for tunnel registry (allows late initialization)
	browserServiceURL    string                          // URL for browser-service
	tunnelClients        map[string]*TunnelClientWrapper // Cache tunnel clients per session
	tunnelClientsMu      sync.RWMutex                    // Protect tunnel clients cache
	remoteCDPClients     map[string]*clientEntry         // Cache remote CDP clients per session
	remoteCDPClientsMu   sync.RWMutex                    // Protect remote CDP clients cache
	screenshotStorage    storage.ScreenshotStorage       // Storage for analyze_screenshot screenshots
}

// NewBrowserTool creates a new browser tool instance
func NewBrowserTool(permissions permission.Service, sessions session.Service, browserServiceURL string, sessionConfig session.Config, browserMode string, clientFactory ClientFactory, connectionManager interface{}, tunnelRegistryGetter func() interface{}, baseURL string) interfaces.BaseTool {
	// Default to local-browser-service mode if not specified
	if browserMode == "" {
		browserMode = browserpkg.ModeLocalBrowserService
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
		sessions:             sessions,
		connectionManager:    connMgr,
		sessionConfig:        sessionConfig,
		baseURL:              baseURL,
		browserMode:          browserMode,
		clientFactory:        clientFactory,
		tunnelRegistryGetter: tunnelRegistryGetter,
		browserServiceURL:    browserServiceURL,
		tunnelClients:        make(map[string]*TunnelClientWrapper),
		remoteCDPClients:     make(map[string]*clientEntry),
		screenshotStorage:    storage.NewStorage(sessionConfig.BasePath, baseURL),
	}
}

// getClient creates a browser client using the factory pattern
// Supports both tunnel and service modes based on configuration
// Returns BrowserClient interface that works with both modes
func (b *browserTool) getClient(ctx context.Context, sessionID string) (BrowserClient, error) {
	// Try to get session from context first (preferred - no DB query)
	var sess session.Session
	if sessionVal := ctx.Value(interfaces.SessionContextKey); sessionVal != nil {
		if s, ok := sessionVal.(session.Session); ok {
			sess = s
		}
	}

	// Fall back to DB query if not in context (backward compatibility)
	if sess.ID == "" {
		var err error
		sess, err = b.sessions.Get(ctx, sessionID)
		if err != nil {
			return nil, fmt.Errorf("failed to get session: %w", err)
		}
	}

	// Use session's browser mode (falls back to tool's global mode if empty)
	browserMode := sess.BrowserMode
	if browserMode == "" {
		browserMode = b.browserMode
	}

	// Route to appropriate client based on browser mode
	switch browserMode {
	case browserpkg.ModeLocalBrowserService:
		if b.connectionManager == nil {
			return nil, fmt.Errorf("BROWSER_SERVICE_URL environment variable is required for local-browser-service mode (format: http://localhost:PORT)")
		}
		return b.connectionManager.GetOrCreate(ctx, sessionID)

	case browserpkg.ModeRemoteCDP:
		// Remote CDP mode: get or create cached client using sync.Once pattern
		b.remoteCDPClientsMu.Lock()
		entry, exists := b.remoteCDPClients[sessionID]
		if !exists {
			entry = &clientEntry{}
			b.remoteCDPClients[sessionID] = entry
		}
		b.remoteCDPClientsMu.Unlock()

		// Use sync.Once to guarantee exactly one creation
		entry.once.Do(func() {
			// Validate CDP URL
			if sess.CdpUrl == "" {
				entry.err = fmt.Errorf("CDP URL is required for remote-cdp-websocket mode")
				return
			}

			// Create new client
			client, err := NewRemoteCDPClient(ctx, sess.CdpUrl)
			if err != nil {
				entry.err = fmt.Errorf("failed to create remote CDP client: %w", err)
				return
			}

			entry.client = client
		})

		return entry.client, entry.err

	case browserpkg.ModeElectronEmbedded:
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

	default:
		return nil, fmt.Errorf("unsupported browser mode: %s", browserMode)
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
				"enum":        []string{ActionOpen /* ActionScreenshot, */, ActionReadPage, ActionLeftClick, ActionType, ActionScroll, ActionUpload, ActionGetText, ActionFind, ActionClose, ActionRightClick, ActionDoubleClick, ActionTripleClick, ActionLeftClickDrag, ActionFormInput, ActionGoBack, ActionGoForward, ActionTabCreate, ActionTabList, ActionTabSwitch, ActionTabClose, ActionWait, ActionScrollTo, ActionSequence, ActionAnalyzeScreenshot},
			},
			"description": map[string]any{
				"type":        "string",
				"description": "A 2-6 word description of what the action does",
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
			"ref": map[string]any{
				"type":        "string",
				"description": "Element reference from read_page (e.g. f0_ref_1) for click actions",
			},
			"coordinate": map[string]any{
				"type":        "array",
				"description": "Coordinate in screenshot space [x, y] for click actions",
				"items": map[string]any{
					"type": "number",
				},
			},
			"text": map[string]any{
				"type":        "string",
				"description": "Text to type (for type action). Supports {key} syntax for special keys. Examples: 'hello{Enter}', 'search query{Tab}', '{cmd+a}{Delete}', '{Backspace}{Backspace}'. Use {{}} to escape literal braces.",
			},
			"direction": map[string]any{
				"type":        "string",
				"description": "Scroll direction (for scroll action)",
				"enum":        []string{DirectionUp, DirectionDown, DirectionLeft, DirectionRight},
			},
			"scroll_amount": map[string]any{
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
				"description": "Tab ID to operate on. Required for all tab-specific actions (open, screenshot, read_page, click, type, scroll, upload, get_text, find, form_input, go_back, go_forward, scroll_to, sequence, wait, tab_switch, tab_close). Not required for tab_create, tab_list, or close actions.",
			},
			"tab_id": map[string]any{
				"type":        "string",
				"description": "Tab ID to operate on (reference-style field).",
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
			"actions": map[string]any{
				"type":        "array",
				"description": "Array of actions to execute in sequence (for action batching)",
				"items": map[string]any{
					"type": "object",
				},
			},
			"prompt": map[string]any{
				"type":        "string",
				"description": "Analysis prompt for screenshot (for analyze_screenshot action). For bounding boxes, include keywords like 'bounding box' or 'coordinates'",
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
	params.TabID = params.EffectiveTabID()

	// Validate tabId requirement for tab-interaction actions
	requiresTabID := []string{
		ActionOpen /* ActionScreenshot, */, ActionReadPage, ActionLeftClick, ActionType,
		ActionScroll, ActionUpload, ActionGetText, ActionFind, ActionRightClick,
		ActionDoubleClick, ActionTripleClick, ActionLeftClickDrag, ActionFormInput,
		ActionGoBack, ActionGoForward, ActionScrollTo, ActionSequence, ActionWait,
		ActionTabSwitch, ActionTabClose, ActionAnalyzeScreenshot,
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
	case ActionReadPage:
		return b.handleReadPage(ctx, params, sessionID), nil
	case ActionLeftClick:
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
	case ActionLeftClickDrag:
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
	case ActionScrollTo:
		return b.handleScrollTo(ctx, params, sessionID), nil
	case ActionSequence:
		return b.handleActionSequence(ctx, params, sessionID, sessionStorageDir), nil
	case ActionAnalyzeScreenshot:
		return b.handleAnalyzeScreenshot(ctx, params, sessionID), nil
	default:
		return interfaces.NewTextErrorResponse(fmt.Sprintf("unknown action: %s", params.Action)), nil
	}
}
