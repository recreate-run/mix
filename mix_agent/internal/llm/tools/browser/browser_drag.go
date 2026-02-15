package browser

import (
	"context"
	"fmt"

	"mix/internal/llm/interfaces"
)

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

	// Set duration with default of 500ms if not specified
	duration := 500
	if params.Duration > 0 {
		duration = params.Duration
	}

	// Perform drag (tabID is always required and validated)
	err = client.Drag(ctx, params.FromIndex, params.ToIndex, params.FromX, params.FromY, params.ToX, params.ToY, &duration, params.TabID)
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
