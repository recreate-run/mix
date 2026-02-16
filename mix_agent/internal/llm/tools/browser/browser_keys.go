package browser

import (
	"context"
	"fmt"

	"mix/internal/llm/interfaces"
)

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
