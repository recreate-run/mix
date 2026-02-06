package browser

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"
	"os"
	"time"

	browserprotocol "github.com/sarathmenon/browser-service/pkg/protocol"

	"mix/internal/llm/interfaces"
	"mix/internal/llm/tools"
	"mix/internal/permission"
	"mix/internal/session"
)

const (
	BrowserToolName = "Browser"

	// DefaultRequestTimeout is the default timeout for browser operations
	DefaultRequestTimeout = 30 * time.Second
)

// browserTool implements the Browser tool for LLM-driven browser automation
type browserTool struct {
	permissions       permission.Service
	connectionManager *ConnectionManager
	sessionConfig     session.Config
	baseURL           string
}

// NewBrowserTool creates a new browser tool instance
func NewBrowserTool(permissions permission.Service, browserServiceURL string, sessionConfig session.Config) interfaces.BaseTool {
	return &browserTool{
		permissions:       permissions,
		connectionManager: NewConnectionManager(browserServiceURL),
		sessionConfig:     sessionConfig,
		baseURL:           getBaseURL(),
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
				"enum":        []string{ActionOpen, ActionScreenshot, ActionClick, ActionType, ActionScroll, ActionClose},
			},
			"url": map[string]any{
				"type":        "string",
				"description": "URL to navigate to (for open action)",
			},
			"withOverlay": map[string]any{
				"type":        "boolean",
				"description": "Whether to add element overlay to screenshot (default: true)",
			},
			"index": map[string]any{
				"type":        "integer",
				"description": "Element index (for click/type actions)",
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
		return b.handleOpen(ctx, params, sessionID)
	case ActionScreenshot:
		return b.handleScreenshot(ctx, params, sessionID, sessionStorageDir)
	case ActionClick:
		return b.handleClick(ctx, params, sessionID)
	case ActionType:
		return b.handleType(ctx, params, sessionID)
	case ActionScroll:
		return b.handleScroll(ctx, params, sessionID)
	case ActionClose:
		return b.handleClose(ctx, sessionID)
	default:
		return interfaces.NewTextErrorResponse(fmt.Sprintf("unknown action: %s", params.Action)), nil
	}
}

// handleOpen navigates to a URL
func (b *browserTool) handleOpen(ctx context.Context, params BrowserParams, sessionID string) (interfaces.ToolResponse, error) {
	// Validate URL
	if params.URL == "" {
		return interfaces.NewTextErrorResponse("missing url parameter for open action"), nil
	}

	// Parse and validate URL
	parsedURL, err := url.Parse(params.URL)
	if err != nil {
		return interfaces.NewTextErrorResponse(fmt.Sprintf("invalid URL %s: %v", params.URL, err)), nil
	}
	if parsedURL.Scheme != "http" && parsedURL.Scheme != "https" {
		return interfaces.NewTextErrorResponse(fmt.Sprintf("invalid URL scheme: %s (must be http or https)", params.URL)), nil
	}

	// Permission check temporarily disabled for testing
	// TODO: Re-enable permissions later

	// Get or create browser connection
	client, err := b.connectionManager.GetOrCreate(ctx, sessionID)
	if err != nil {
		return interfaces.NewTextErrorResponse(fmt.Sprintf("Failed to connect to browser service: %v", err)), nil
	}

	// Navigate
	result, err := client.Navigate(ctx, params.URL)
	if err != nil {
		return interfaces.NewTextErrorResponse(fmt.Sprintf("Navigation failed: %v", err)), nil
	}

	return interfaces.NewTextResponse(fmt.Sprintf("Successfully navigated to %s (Frame ID: %s)", params.URL, result.FrameID)), nil
}

// handleScreenshot captures a screenshot
func (b *browserTool) handleScreenshot(ctx context.Context, params BrowserParams, sessionID, sessionStorageDir string) (interfaces.ToolResponse, error) {
	// Get or create browser connection
	client, err := b.connectionManager.GetOrCreate(ctx, sessionID)
	if err != nil {
		return interfaces.NewTextErrorResponse(fmt.Sprintf("Failed to connect to browser service: %v", err)), nil
	}

	// Default withOverlay to true if not specified
	withOverlay := true
	if params.WithOverlay != nil {
		withOverlay = *params.WithOverlay
	}

	// Capture screenshot
	screenshotParams := browserprotocol.ScreenshotParams{
		Format:      "png",
		FullPage:    false,
		WithOverlay: withOverlay,
	}

	result, err := client.Screenshot(ctx, screenshotParams)
	if err != nil {
		return interfaces.NewTextErrorResponse(fmt.Sprintf("Screenshot failed: %v", err)), nil
	}

	// Save screenshot to session storage
	filename, err := saveScreenshot(result.Data, sessionStorageDir)
	if err != nil {
		return interfaces.NewTextErrorResponse(fmt.Sprintf("Failed to save screenshot: %v", err)), nil
	}

	// Format response with element list
	response := formatScreenshotResponse(filename, sessionID, b.baseURL, result.Elements, withOverlay)

	return interfaces.NewTextResponse(response), nil
}

// handleClick clicks an element
func (b *browserTool) handleClick(ctx context.Context, params BrowserParams, sessionID string) (interfaces.ToolResponse, error) {
	// Get browser connection
	client, err := b.connectionManager.GetOrCreate(ctx, sessionID)
	if err != nil {
		return interfaces.NewTextErrorResponse(fmt.Sprintf("Failed to connect to browser service: %v", err)), nil
	}

	// Click element
	if err := client.Click(ctx, params.Index); err != nil {
		return interfaces.NewTextErrorResponse(fmt.Sprintf("Click failed: %v", err)), nil
	}

	return interfaces.NewTextResponse(fmt.Sprintf("Successfully clicked element %d", params.Index)), nil
}

// handleType types text into an element
func (b *browserTool) handleType(ctx context.Context, params BrowserParams, sessionID string) (interfaces.ToolResponse, error) {
	// Validate text parameter
	if params.Text == "" {
		return interfaces.NewTextErrorResponse("missing text parameter for type action"), nil
	}

	// Get browser connection
	client, err := b.connectionManager.GetOrCreate(ctx, sessionID)
	if err != nil {
		return interfaces.NewTextErrorResponse(fmt.Sprintf("Failed to connect to browser service: %v", err)), nil
	}

	// Type text
	if err := client.Type(ctx, params.Index, params.Text); err != nil {
		return interfaces.NewTextErrorResponse(fmt.Sprintf("Type failed: %v", err)), nil
	}

	return interfaces.NewTextResponse(fmt.Sprintf("Successfully typed text into element %d", params.Index)), nil
}

// handleScroll scrolls the page
func (b *browserTool) handleScroll(ctx context.Context, params BrowserParams, sessionID string) (interfaces.ToolResponse, error) {
	// Validate direction
	if params.Direction == "" {
		return interfaces.NewTextErrorResponse("missing direction parameter for scroll action"), nil
	}

	// Validate direction value
	validDirections := map[string]bool{
		DirectionUp:    true,
		DirectionDown:  true,
		DirectionLeft:  true,
		DirectionRight: true,
	}
	if !validDirections[params.Direction] {
		return interfaces.NewTextErrorResponse(fmt.Sprintf("invalid direction: %s (must be up/down/left/right)", params.Direction)), nil
	}

	// Default amount to 100 pixels if not specified
	amount := params.Amount
	if amount == 0 {
		amount = 100
	}

	// Get browser connection
	client, err := b.connectionManager.GetOrCreate(ctx, sessionID)
	if err != nil {
		return interfaces.NewTextErrorResponse(fmt.Sprintf("Failed to connect to browser service: %v", err)), nil
	}

	// Scroll
	if err := client.Scroll(ctx, params.Direction, amount); err != nil {
		return interfaces.NewTextErrorResponse(fmt.Sprintf("Scroll failed: %v", err)), nil
	}

	return interfaces.NewTextResponse(fmt.Sprintf("Successfully scrolled %s by %d pixels", params.Direction, amount)), nil
}

// handleClose closes the browser
func (b *browserTool) handleClose(_ context.Context, sessionID string) (interfaces.ToolResponse, error) {
	// Close connection
	if err := b.connectionManager.Close(sessionID); err != nil {
		return interfaces.NewTextErrorResponse(fmt.Sprintf("Failed to close browser: %v", err)), nil
	}

	return interfaces.NewTextResponse("Browser closed successfully"), nil
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
