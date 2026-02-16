package browser

import (
	"context"
	"fmt"

	"mix/internal/llm/interfaces"
)

func (b *browserTool) clickByBackendID(ctx context.Context, client BrowserClient, backendID int64, tabID, button string, clickCount int) error {
	const (
		mouseButtonLeft  = "left"
		mouseButtonRight = "right"
	)

	switch button {
	case mouseButtonLeft:
		switch clickCount {
		case 1:
			return client.ClickByBackendID(ctx, backendID, tabID)
		case 2:
			return client.DoubleClickByBackendID(ctx, backendID, tabID)
		case 3:
			return client.TripleClickByBackendID(ctx, backendID, tabID)
		default:
			return fmt.Errorf("unsupported click count: %d", clickCount)
		}
	case mouseButtonRight:
		if clickCount != 1 {
			return fmt.Errorf("right click only supports single click")
		}
		return client.RightClickByBackendID(ctx, backendID, tabID)
	default:
		return fmt.Errorf("unsupported button: %s", button)
	}
}

// handleClick clicks an element
func (b *browserTool) handleClick(ctx context.Context, params BrowserParams, sessionID string) interfaces.ToolResponse {
	// Get browser connection
	client, err := b.getClient(ctx, sessionID)
	if err != nil {
		return interfaces.NewTextErrorResponse(fmt.Sprintf("Failed to get browser client: %v", err))
	}

	if params.Ref != "" {
		backendID, err := b.backendIDFromRef(params.Ref)
		if err != nil {
			return interfaces.NewTextErrorResponse(fmt.Sprintf("Invalid ref: %v", err))
		}
		// Tunnel mode: convert backendID to coordinate for clicking
		if tunnelClient, ok := client.(*TunnelClientWrapper); ok {
			coordinate, err := b.coordinateFromBackendID(ctx, sessionID, params.TabID, backendID)
			if err != nil {
				return interfaces.NewTextErrorResponse(fmt.Sprintf("Click failed: %v", err))
			}
			if err := tunnelClient.ClickAt(ctx, coordinate.X, coordinate.Y, "left", 1, nil, params.TabID); err != nil {
				return interfaces.NewTextErrorResponse(fmt.Sprintf("Click failed: %v", err))
			}
			return interfaces.NewTextResponse("Successfully clicked element by ref")
		}
		// Service/RemoteCDP mode: use backendID directly
		if err := b.clickByBackendID(ctx, client, backendID, params.TabID, "left", 1); err != nil {
			return interfaces.NewTextErrorResponse(fmt.Sprintf("Click failed: %v", err))
		}
		return interfaces.NewTextResponse("Successfully clicked element by ref")
	}

	if params.Coordinate != nil {
		// ServiceClientAdapter supports ClickAt with pointer parameters
		if adapter, ok := client.(*ServiceClientAdapter); ok {
			button := mouseButtonLeft
			clickCount := 1
			if err := adapter.ClickAt(ctx, params.Coordinate.X, params.Coordinate.Y, &button, &clickCount, nil, params.TabID); err != nil {
				return interfaces.NewTextErrorResponse(fmt.Sprintf("Click failed: %v", err))
			}
			return interfaces.NewTextResponse("Successfully clicked coordinate")
		}

		// TunnelClientWrapper supports ClickAt with non-pointer parameters
		if tunnel, ok := client.(*TunnelClientWrapper); ok {
			if err := tunnel.ClickAt(ctx, params.Coordinate.X, params.Coordinate.Y, "left", 1, nil, params.TabID); err != nil {
				return interfaces.NewTextErrorResponse(fmt.Sprintf("Click failed: %v", err))
			}
			return interfaces.NewTextResponse("Successfully clicked coordinate")
		}

		return interfaces.NewTextErrorResponse("Coordinate-based clicking not supported for this client type")
	}

	// Validate index parameter (required if no ref or coordinate provided)
	if params.Index == nil {
		return interfaces.NewTextErrorResponse("missing index, ref, or coordinate parameter for left_click action")
	}

	// browser-service mode: use Click(index) directly to avoid cache synchronization issues
	// browser-service's ReadPage doesn't populate the backendID click cache, so we use the index-based Click API instead
	if adapter, ok := client.(*ServiceClientAdapter); ok {
		// Call ReadPage to ensure element list is fresh
		_, readErr := adapter.ReadPage(ctx, true, params.TabID)
		if readErr != nil {
			return interfaces.NewTextErrorResponse(fmt.Sprintf("Failed to read page elements: %v", readErr))
		}

		// Use index-based Click directly
		if err := adapter.Click(ctx, *params.Index, params.TabID); err != nil {
			return interfaces.NewTextErrorResponse(fmt.Sprintf("Click failed: %v", err))
		}
		return interfaces.NewTextResponse(fmt.Sprintf("Successfully clicked element %d", *params.Index))
	}

	// Other modes (RemoteCDP): use backendID-based clicking
	backendID, err := b.backendIDFromIndex(ctx, sessionID, params.TabID, *params.Index)
	if err != nil {
		return interfaces.NewTextErrorResponse(fmt.Sprintf("Element not found: %v", err))
	}

	if err := b.clickByBackendID(ctx, client, backendID, params.TabID, "left", 1); err != nil {
		return interfaces.NewTextErrorResponse(fmt.Sprintf("Click failed: %v", err))
	}

	return interfaces.NewTextResponse(fmt.Sprintf("Successfully clicked element %d", *params.Index))
}

// handleRightClick right-clicks an element
func (b *browserTool) handleRightClick(ctx context.Context, params BrowserParams, sessionID string) interfaces.ToolResponse {
	// Get browser connection
	client, err := b.getClient(ctx, sessionID)
	if err != nil {
		return interfaces.NewTextErrorResponse(fmt.Sprintf("Failed to get browser client: %v", err))
	}

	if params.Ref != "" {
		backendID, err := b.backendIDFromRef(params.Ref)
		if err != nil {
			return interfaces.NewTextErrorResponse(fmt.Sprintf("Invalid ref: %v", err))
		}
		// Tunnel mode: convert backendID to coordinate for clicking
		if tunnelClient, ok := client.(*TunnelClientWrapper); ok {
			coordinate, err := b.coordinateFromBackendID(ctx, sessionID, params.TabID, backendID)
			if err != nil {
				return interfaces.NewTextErrorResponse(fmt.Sprintf("Right-click failed: %v", err))
			}
			if err := tunnelClient.ClickAt(ctx, coordinate.X, coordinate.Y, "right", 1, nil, params.TabID); err != nil {
				return interfaces.NewTextErrorResponse(fmt.Sprintf("Right-click failed: %v", err))
			}
			return interfaces.NewTextResponse("Successfully right-clicked element by ref")
		}
		// Service/RemoteCDP mode: use backendID directly
		if err := b.clickByBackendID(ctx, client, backendID, params.TabID, "right", 1); err != nil {
			return interfaces.NewTextErrorResponse(fmt.Sprintf("Right-click failed: %v", err))
		}
		return interfaces.NewTextResponse("Successfully right-clicked element by ref")
	}

	if params.Coordinate != nil {
		// ServiceClientAdapter supports RightClickAt
		if adapter, ok := client.(*ServiceClientAdapter); ok {
			if err := adapter.RightClickAt(ctx, params.Coordinate.X, params.Coordinate.Y, nil, params.TabID); err != nil {
				return interfaces.NewTextErrorResponse(fmt.Sprintf("Right-click failed: %v", err))
			}
			return interfaces.NewTextResponse("Successfully right-clicked coordinate")
		}

		// TunnelClientWrapper supports ClickAt with right button
		if tunnel, ok := client.(*TunnelClientWrapper); ok {
			if err := tunnel.ClickAt(ctx, params.Coordinate.X, params.Coordinate.Y, "right", 1, nil, params.TabID); err != nil {
				return interfaces.NewTextErrorResponse(fmt.Sprintf("Right-click failed: %v", err))
			}
			return interfaces.NewTextResponse("Successfully right-clicked coordinate")
		}

		return interfaces.NewTextErrorResponse("Coordinate-based right-clicking not supported for this client type")
	}

	// Validate index parameter (required if no ref or coordinate provided)
	if params.Index == nil {
		return interfaces.NewTextErrorResponse("missing index, ref, or coordinate parameter for right_click action")
	}

	// Service mode: support index-based clicking
	backendID, err := b.backendIDFromIndex(ctx, sessionID, params.TabID, *params.Index)
	if err != nil {
		return interfaces.NewTextErrorResponse(fmt.Sprintf("Element not found: %v", err))
	}

	if err := b.clickByBackendID(ctx, client, backendID, params.TabID, "right", 1); err != nil {
		return interfaces.NewTextErrorResponse(fmt.Sprintf("Right-click failed: %v", err))
	}

	return interfaces.NewTextResponse(fmt.Sprintf("Successfully right-clicked element %d", *params.Index))
}

// handleDoubleClick double-clicks an element
func (b *browserTool) handleDoubleClick(ctx context.Context, params BrowserParams, sessionID string) interfaces.ToolResponse {
	// Get browser connection
	client, err := b.getClient(ctx, sessionID)
	if err != nil {
		return interfaces.NewTextErrorResponse(fmt.Sprintf("Failed to get browser client: %v", err))
	}

	if params.Ref != "" {
		backendID, err := b.backendIDFromRef(params.Ref)
		if err != nil {
			return interfaces.NewTextErrorResponse(fmt.Sprintf("Invalid ref: %v", err))
		}
		// Tunnel mode: convert backendID to coordinate for clicking
		if tunnelClient, ok := client.(*TunnelClientWrapper); ok {
			coordinate, err := b.coordinateFromBackendID(ctx, sessionID, params.TabID, backendID)
			if err != nil {
				return interfaces.NewTextErrorResponse(fmt.Sprintf("Double-click failed: %v", err))
			}
			if err := tunnelClient.ClickAt(ctx, coordinate.X, coordinate.Y, "left", 2, nil, params.TabID); err != nil {
				return interfaces.NewTextErrorResponse(fmt.Sprintf("Double-click failed: %v", err))
			}
			return interfaces.NewTextResponse("Successfully double-clicked element by ref")
		}
		// Service/RemoteCDP mode: use backendID directly
		if err := b.clickByBackendID(ctx, client, backendID, params.TabID, "left", 2); err != nil {
			return interfaces.NewTextErrorResponse(fmt.Sprintf("Double-click failed: %v", err))
		}
		return interfaces.NewTextResponse("Successfully double-clicked element by ref")
	}

	if params.Coordinate != nil {
		// ServiceClientAdapter supports DoubleClickAt
		if adapter, ok := client.(*ServiceClientAdapter); ok {
			button := mouseButtonLeft
			if err := adapter.DoubleClickAt(ctx, params.Coordinate.X, params.Coordinate.Y, &button, nil, params.TabID); err != nil {
				return interfaces.NewTextErrorResponse(fmt.Sprintf("Double-click failed: %v", err))
			}
			return interfaces.NewTextResponse("Successfully double-clicked coordinate")
		}

		// TunnelClientWrapper supports ClickAt with clickCount=2
		if tunnel, ok := client.(*TunnelClientWrapper); ok {
			if err := tunnel.ClickAt(ctx, params.Coordinate.X, params.Coordinate.Y, "left", 2, nil, params.TabID); err != nil {
				return interfaces.NewTextErrorResponse(fmt.Sprintf("Double-click failed: %v", err))
			}
			return interfaces.NewTextResponse("Successfully double-clicked coordinate")
		}

		return interfaces.NewTextErrorResponse("Coordinate-based double-clicking not supported for this client type")
	}

	// Validate index parameter (required if no ref or coordinate provided)
	if params.Index == nil {
		return interfaces.NewTextErrorResponse("missing index, ref, or coordinate parameter for double_click action")
	}

	// Service mode: support index-based clicking
	backendID, err := b.backendIDFromIndex(ctx, sessionID, params.TabID, *params.Index)
	if err != nil {
		return interfaces.NewTextErrorResponse(fmt.Sprintf("Element not found: %v", err))
	}

	if err := b.clickByBackendID(ctx, client, backendID, params.TabID, "left", 2); err != nil {
		return interfaces.NewTextErrorResponse(fmt.Sprintf("Double-click failed: %v", err))
	}

	return interfaces.NewTextResponse(fmt.Sprintf("Successfully double-clicked element %d", *params.Index))
}

// handleTripleClick triple-clicks an element
func (b *browserTool) handleTripleClick(ctx context.Context, params BrowserParams, sessionID string) interfaces.ToolResponse {
	// Get browser connection
	client, err := b.getClient(ctx, sessionID)
	if err != nil {
		return interfaces.NewTextErrorResponse(fmt.Sprintf("Failed to get browser client: %v", err))
	}

	if params.Ref != "" {
		backendID, err := b.backendIDFromRef(params.Ref)
		if err != nil {
			return interfaces.NewTextErrorResponse(fmt.Sprintf("Invalid ref: %v", err))
		}
		// Tunnel mode: convert backendID to coordinate for clicking
		if tunnelClient, ok := client.(*TunnelClientWrapper); ok {
			coordinate, err := b.coordinateFromBackendID(ctx, sessionID, params.TabID, backendID)
			if err != nil {
				return interfaces.NewTextErrorResponse(fmt.Sprintf("Triple-click failed: %v", err))
			}
			if err := tunnelClient.ClickAt(ctx, coordinate.X, coordinate.Y, "left", 3, nil, params.TabID); err != nil {
				return interfaces.NewTextErrorResponse(fmt.Sprintf("Triple-click failed: %v", err))
			}
			return interfaces.NewTextResponse("Successfully triple-clicked element by ref")
		}
		// Service/RemoteCDP mode: use backendID directly
		if err := b.clickByBackendID(ctx, client, backendID, params.TabID, "left", 3); err != nil {
			return interfaces.NewTextErrorResponse(fmt.Sprintf("Triple-click failed: %v", err))
		}
		return interfaces.NewTextResponse("Successfully triple-clicked element by ref")
	}

	if params.Coordinate != nil {
		// ServiceClientAdapter supports TripleClickAt
		if adapter, ok := client.(*ServiceClientAdapter); ok {
			button := mouseButtonLeft
			if err := adapter.TripleClickAt(ctx, params.Coordinate.X, params.Coordinate.Y, &button, nil, params.TabID); err != nil {
				return interfaces.NewTextErrorResponse(fmt.Sprintf("Triple-click failed: %v", err))
			}
			return interfaces.NewTextResponse("Successfully triple-clicked coordinate")
		}

		// TunnelClientWrapper supports ClickAt with clickCount=3
		if tunnel, ok := client.(*TunnelClientWrapper); ok {
			if err := tunnel.ClickAt(ctx, params.Coordinate.X, params.Coordinate.Y, "left", 3, nil, params.TabID); err != nil {
				return interfaces.NewTextErrorResponse(fmt.Sprintf("Triple-click failed: %v", err))
			}
			return interfaces.NewTextResponse("Successfully triple-clicked coordinate")
		}

		return interfaces.NewTextErrorResponse("Coordinate-based triple-clicking not supported for this client type")
	}

	// Validate index parameter (required if no ref or coordinate provided)
	if params.Index == nil {
		return interfaces.NewTextErrorResponse("missing index, ref, or coordinate parameter for triple_click action")
	}

	// Service mode: support index-based clicking
	backendID, err := b.backendIDFromIndex(ctx, sessionID, params.TabID, *params.Index)
	if err != nil {
		return interfaces.NewTextErrorResponse(fmt.Sprintf("Element not found: %v", err))
	}

	if err := b.clickByBackendID(ctx, client, backendID, params.TabID, "left", 3); err != nil {
		return interfaces.NewTextErrorResponse(fmt.Sprintf("Triple-click failed: %v", err))
	}

	return interfaces.NewTextResponse(fmt.Sprintf("Successfully triple-clicked element %d", *params.Index))
}

// executeClick executes a click sub-action
func (b *browserTool) executeClick(ctx context.Context, client BrowserClient, action SubAction, sessionID, tabID string) error {
	if action.Ref != "" {
		backendID, err := b.backendIDFromRef(action.Ref)
		if err != nil {
			return err
		}
		repeat := action.Repeat
		if repeat <= 0 {
			repeat = 1
		}
		// Tunnel mode: convert backendID to coordinate for clicking
		if tunnelClient, ok := client.(*TunnelClientWrapper); ok {
			coordinate, err := b.coordinateFromBackendID(ctx, sessionID, tabID, backendID)
			if err != nil {
				return err
			}
			for i := 0; i < repeat; i++ {
				if err := tunnelClient.ClickAt(ctx, coordinate.X, coordinate.Y, "left", 1, nil, tabID); err != nil {
					return err
				}
			}
			return nil
		}
		// Service/RemoteCDP mode: use backendID directly
		for i := 0; i < repeat; i++ {
			if err := b.clickByBackendID(ctx, client, backendID, tabID, "left", 1); err != nil {
				return err
			}
		}
		return nil
	}

	if action.Coordinate != nil {
		// ServiceClientAdapter supports ClickAt
		if adapter, ok := client.(*ServiceClientAdapter); ok {
			button := mouseButtonLeft
			clickCount := 1
			duration := action.Duration
			var durationPtr *int
			if duration > 0 {
				durationPtr = &duration
			}
			for i := 0; i < action.Repeat || (action.Repeat == 0 && i == 0); i++ {
				if err := adapter.ClickAt(ctx, action.Coordinate.X, action.Coordinate.Y, &button, &clickCount, durationPtr, tabID); err != nil {
					return err
				}
			}
			return nil
		}

		// TunnelClientWrapper supports ClickAt
		if tunnel, ok := client.(*TunnelClientWrapper); ok {
			duration := action.Duration
			var durationPtr *int
			if duration > 0 {
				durationPtr = &duration
			}
			for i := 0; i < action.Repeat || (action.Repeat == 0 && i == 0); i++ {
				if err := tunnel.ClickAt(ctx, action.Coordinate.X, action.Coordinate.Y, "left", 1, durationPtr, tabID); err != nil {
					return err
				}
			}
			return nil
		}

		return fmt.Errorf("coordinate-based clicking not supported for this client type")
	}

	if action.Index == nil {
		return fmt.Errorf("index or ref required for click action")
	}

	// Service mode: support index-based clicking
	backendID, err := b.backendIDFromIndex(ctx, sessionID, tabID, *action.Index)
	if err != nil {
		return err
	}
	repeat := action.Repeat
	if repeat <= 0 {
		repeat = 1
	}
	for i := 0; i < repeat; i++ {
		if err := b.clickByBackendID(ctx, client, backendID, tabID, "left", 1); err != nil {
			return err
		}
	}
	return nil
}

// executeRightClick executes a right-click sub-action
func (b *browserTool) executeRightClick(ctx context.Context, client BrowserClient, action SubAction, sessionID, tabID string) error {
	if action.Ref != "" {
		backendID, err := b.backendIDFromRef(action.Ref)
		if err != nil {
			return err
		}
		// Tunnel mode: convert backendID to coordinate for clicking
		if tunnelClient, ok := client.(*TunnelClientWrapper); ok {
			coordinate, err := b.coordinateFromBackendID(ctx, sessionID, tabID, backendID)
			if err != nil {
				return err
			}
			return tunnelClient.ClickAt(ctx, coordinate.X, coordinate.Y, "right", 1, nil, tabID)
		}
		// Service/RemoteCDP mode: use backendID directly
		return b.clickByBackendID(ctx, client, backendID, tabID, "right", 1)
	}

	if action.Coordinate != nil {
		// ServiceClientAdapter supports RightClickAt
		if adapter, ok := client.(*ServiceClientAdapter); ok {
			duration := action.Duration
			var durationPtr *int
			if duration > 0 {
				durationPtr = &duration
			}
			return adapter.RightClickAt(ctx, action.Coordinate.X, action.Coordinate.Y, durationPtr, tabID)
		}

		// TunnelClientWrapper supports ClickAt with right button
		if tunnel, ok := client.(*TunnelClientWrapper); ok {
			duration := action.Duration
			var durationPtr *int
			if duration > 0 {
				durationPtr = &duration
			}
			return tunnel.ClickAt(ctx, action.Coordinate.X, action.Coordinate.Y, "right", 1, durationPtr, tabID)
		}

		return fmt.Errorf("coordinate-based right-clicking not supported for this client type")
	}

	if action.Index == nil {
		return fmt.Errorf("index or ref required for right_click action")
	}

	// Service mode: support index-based clicking
	backendID, err := b.backendIDFromIndex(ctx, sessionID, tabID, *action.Index)
	if err != nil {
		return err
	}
	return b.clickByBackendID(ctx, client, backendID, tabID, "right", 1)
}

// executeDoubleClick executes a double-click sub-action
func (b *browserTool) executeDoubleClick(ctx context.Context, client BrowserClient, action SubAction, sessionID, tabID string) error {
	if action.Ref != "" {
		backendID, err := b.backendIDFromRef(action.Ref)
		if err != nil {
			return err
		}
		// Tunnel mode: convert backendID to coordinate for clicking
		if tunnelClient, ok := client.(*TunnelClientWrapper); ok {
			coordinate, err := b.coordinateFromBackendID(ctx, sessionID, tabID, backendID)
			if err != nil {
				return err
			}
			return tunnelClient.ClickAt(ctx, coordinate.X, coordinate.Y, "left", 2, nil, tabID)
		}
		// Service/RemoteCDP mode: use backendID directly
		return b.clickByBackendID(ctx, client, backendID, tabID, "left", 2)
	}

	if action.Coordinate != nil {
		// ServiceClientAdapter supports DoubleClickAt
		if adapter, ok := client.(*ServiceClientAdapter); ok {
			button := mouseButtonLeft
			duration := action.Duration
			var durationPtr *int
			if duration > 0 {
				durationPtr = &duration
			}
			return adapter.DoubleClickAt(ctx, action.Coordinate.X, action.Coordinate.Y, &button, durationPtr, tabID)
		}

		// TunnelClientWrapper supports ClickAt with clickCount=2
		if tunnel, ok := client.(*TunnelClientWrapper); ok {
			duration := action.Duration
			var durationPtr *int
			if duration > 0 {
				durationPtr = &duration
			}
			return tunnel.ClickAt(ctx, action.Coordinate.X, action.Coordinate.Y, "left", 2, durationPtr, tabID)
		}

		return fmt.Errorf("coordinate-based double-clicking not supported for this client type")
	}

	if action.Index == nil {
		return fmt.Errorf("index or ref required for double_click action")
	}

	// Service mode: support index-based clicking
	backendID, err := b.backendIDFromIndex(ctx, sessionID, tabID, *action.Index)
	if err != nil {
		return err
	}
	return b.clickByBackendID(ctx, client, backendID, tabID, "left", 2)
}

// executeTripleClick executes a triple-click sub-action
func (b *browserTool) executeTripleClick(ctx context.Context, client BrowserClient, action SubAction, sessionID, tabID string) error {
	if action.Ref != "" {
		backendID, err := b.backendIDFromRef(action.Ref)
		if err != nil {
			return err
		}
		// Tunnel mode: convert backendID to coordinate for clicking
		if tunnelClient, ok := client.(*TunnelClientWrapper); ok {
			coordinate, err := b.coordinateFromBackendID(ctx, sessionID, tabID, backendID)
			if err != nil {
				return err
			}
			return tunnelClient.ClickAt(ctx, coordinate.X, coordinate.Y, "left", 3, nil, tabID)
		}
		// Service/RemoteCDP mode: use backendID directly
		return b.clickByBackendID(ctx, client, backendID, tabID, "left", 3)
	}

	if action.Coordinate != nil {
		// ServiceClientAdapter supports TripleClickAt
		if adapter, ok := client.(*ServiceClientAdapter); ok {
			button := mouseButtonLeft
			duration := action.Duration
			var durationPtr *int
			if duration > 0 {
				durationPtr = &duration
			}
			return adapter.TripleClickAt(ctx, action.Coordinate.X, action.Coordinate.Y, &button, durationPtr, tabID)
		}

		// TunnelClientWrapper supports ClickAt with clickCount=3
		if tunnel, ok := client.(*TunnelClientWrapper); ok {
			duration := action.Duration
			var durationPtr *int
			if duration > 0 {
				durationPtr = &duration
			}
			return tunnel.ClickAt(ctx, action.Coordinate.X, action.Coordinate.Y, "left", 3, durationPtr, tabID)
		}

		return fmt.Errorf("coordinate-based triple-clicking not supported for this client type")
	}

	if action.Index == nil {
		return fmt.Errorf("index or ref required for triple_click action")
	}

	// Service mode: support index-based clicking
	backendID, err := b.backendIDFromIndex(ctx, sessionID, tabID, *action.Index)
	if err != nil {
		return err
	}
	return b.clickByBackendID(ctx, client, backendID, tabID, "left", 3)
}
