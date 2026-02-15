package browser

import (
	"context"
	"fmt"

	"mix/internal/llm/interfaces"
)

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
	amount := params.ScrollAmount
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

// handleScrollTo scrolls an element into view
func (b *browserTool) handleScrollTo(ctx context.Context, params BrowserParams, sessionID string) interfaces.ToolResponse {
	client, err := b.getClient(ctx, sessionID)
	if err != nil {
		return interfaces.NewTextErrorResponse(fmt.Sprintf("Failed to get browser client: %v", err))
	}

	// browser-service mode: refresh element cache to avoid stale element errors
	if adapter, ok := client.(*ServiceClientAdapter); ok {
		_, readErr := adapter.ReadPage(ctx, true, params.TabID)
		if readErr != nil {
			return interfaces.NewTextErrorResponse(fmt.Sprintf("Failed to read page elements: %v", readErr))
		}
	}

	// Get BackendID from cache or read_page
	backendID, err := b.backendIDFromIndex(ctx, sessionID, params.TabID, params.Index)
	if err != nil {
		return interfaces.NewTextErrorResponse(fmt.Sprintf("Element not found: %v", err))
	}

	// Scroll element into view (tabID is always required and validated)
	err = client.ScrollIntoViewByBackendID(ctx, backendID, params.TabID)
	if err != nil {
		return interfaces.NewTextErrorResponse(fmt.Sprintf("Scroll to element failed: %v", err))
	}

	return interfaces.NewTextResponse(fmt.Sprintf("Successfully scrolled element %d into view", params.Index))
}
