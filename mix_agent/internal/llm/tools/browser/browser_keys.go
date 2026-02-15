package browser

import (
	"context"
	"fmt"

	"mix/internal/llm/interfaces"
)

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

// handleWait pauses execution for specified milliseconds
func (b *browserTool) handleWait(ctx context.Context, params BrowserParams, sessionID string) interfaces.ToolResponse {
	if params.Duration <= 0 || params.Duration > MaxWaitDuration {
		return interfaces.NewTextErrorResponse(fmt.Sprintf("invalid duration: must be between 1-%d milliseconds", MaxWaitDuration))
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
