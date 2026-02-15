package browser

import (
	"context"
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"strings"

	"mix/internal/llm/interfaces"
	"mix/internal/session"
)

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
	if parsedURL.Scheme != "http" && parsedURL.Scheme != "https" && parsedURL.Scheme != "file" {
		return interfaces.NewTextErrorResponse(fmt.Sprintf("invalid URL scheme: %s (must be http, https, or file)", params.URL))
	}

	// Security validation for file:// URLs
	if parsedURL.Scheme == "file" {
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

	return interfaces.NewTextResponse(fmt.Sprintf("Successfully navigated to %s (Frame ID: %s)", params.URL, result.FrameID))
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

// handleClose closes the browser
func (b *browserTool) handleClose(_ context.Context, sessionID string) interfaces.ToolResponse {
	// Close connection
	if err := b.connectionManager.Close(sessionID); err != nil {
		return interfaces.NewTextErrorResponse(fmt.Sprintf("Failed to close browser: %v", err))
	}

	return interfaces.NewTextResponse("Browser closed successfully")
}
