package browser

import (
	"context"
	"fmt"

	"mix/internal/llm/interfaces"
)

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

	// browser-service mode: refresh element cache to avoid stale element errors
	if adapter, ok := client.(*ServiceClientAdapter); ok {
		_, readErr := adapter.ReadPage(ctx, true, params.TabID)
		if readErr != nil {
			return interfaces.NewTextErrorResponse(fmt.Sprintf("Failed to read page elements: %v", readErr))
		}
	}

	// Type text (tabID is always required and validated)
	err = client.Type(ctx, &params.Index, params.Text, params.TabID)
	if err != nil {
		return interfaces.NewTextErrorResponse(fmt.Sprintf("Type failed: %v", err))
	}

	return interfaces.NewTextResponse(fmt.Sprintf("Successfully typed text into element %d", params.Index))
}

// handleFormInput sets form input value directly
func (b *browserTool) handleFormInput(ctx context.Context, params BrowserParams, sessionID string) interfaces.ToolResponse {
	// Validate value parameter
	if params.Value == nil {
		return interfaces.NewTextErrorResponse("missing value parameter for form_input action")
	}

	// Get browser connection
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

	// Convert value to string (handles string, number, boolean)
	valueStr := fmt.Sprintf("%v", params.Value)

	// Set form input value (tabID is always required and validated)
	err = client.FormInput(ctx, params.Index, valueStr, params.TabID)
	if err != nil {
		return interfaces.NewTextErrorResponse(fmt.Sprintf("Form input failed: %v", err))
	}

	return interfaces.NewTextResponse(fmt.Sprintf("Successfully set value in element %d", params.Index))
}

// executeType executes a type sub-action
func (b *browserTool) executeType(ctx context.Context, client BrowserClient, action SubAction, _, tabID string) error {
	// Index is optional - if nil, types into currently focused element
	return client.Type(ctx, action.Index, action.Text, tabID)
}
