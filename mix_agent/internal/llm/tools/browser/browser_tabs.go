package browser

import (
	"context"
	"fmt"
	"net/url"
	"strings"

	browserprotocol "github.com/sarathmenon/browser-service/pkg/protocol"

	"mix/internal/llm/interfaces"
)

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

	return interfaces.NewTextResponse(fmt.Sprintf("Closed tab: %s", params.TabID))
}
