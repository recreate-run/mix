package browser

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	browserprotocol "github.com/sarathmenon/browser-service/pkg/protocol"

	"mix/internal/llm/interfaces"
	"mix/internal/llm/tools"
)

// readPageElements fetches elements using read_page without caching (Phase 11 cacheless design)
func (b *browserTool) readPageElements(ctx context.Context, sessionID, tabID string, interactiveOnly bool) ([]browserprotocol.RawAccessibilityNode, error) {
	client, err := b.getClient(ctx, sessionID)
	if err != nil {
		return nil, fmt.Errorf("failed to get browser client: %w", err)
	}

	result, err := client.ReadPage(ctx, interactiveOnly, tabID)
	if err != nil {
		return nil, fmt.Errorf("read page failed: %w", err)
	}

	return result.Elements, nil
}

// parseRefBackendID parses fX_ref_Y and returns backend ID
func parseRefBackendID(ref string) (int64, error) {
	const refMarker = "_ref_"
	refIndex := strings.LastIndex(ref, refMarker)
	if refIndex == -1 {
		return 0, fmt.Errorf("invalid ref format: %s", ref)
	}

	backendIDStr := ref[refIndex+len(refMarker):]
	backendID, err := strconv.ParseInt(backendIDStr, 10, 64)
	if err != nil {
		return 0, fmt.Errorf("invalid ref backend ID: %w", err)
	}

	return backendID, nil
}

func (b *browserTool) backendIDFromIndex(ctx context.Context, sessionID, tabID string, index int) (int64, error) {
	// Cacheless design (Phase 11): fetch elements on-demand
	elements, err := b.readPageElements(ctx, sessionID, tabID, true)
	if err != nil {
		return 0, err
	}

	if index < 0 || index >= len(elements) {
		return 0, fmt.Errorf("index %d out of range (0-%d)", index, len(elements)-1)
	}

	return elements[index].BackendID, nil
}

func (b *browserTool) backendIDFromRef(ref string) (int64, error) {
	if ref == "" {
		return 0, fmt.Errorf("ref is required")
	}
	return parseRefBackendID(ref)
}

func (b *browserTool) coordinateFromBackendID(ctx context.Context, sessionID, tabID string, backendID int64) (Coordinate, error) {
	elements, err := b.readPageElements(ctx, sessionID, tabID, false)
	if err != nil {
		return Coordinate{}, err
	}

	for _, elem := range elements {
		if elem.BackendID == backendID {
			return Coordinate{
				X: elem.Bounds.X + elem.Bounds.Width/2,
				Y: elem.Bounds.Y + elem.Bounds.Height/2,
			}, nil
		}
	}

	return Coordinate{}, fmt.Errorf("backend ID %d not found in read_page results", backendID)
}

// getContextInfo extracts context information needed for tool execution
func (b *browserTool) getContextInfo(ctx context.Context) (sessionID, sessionStorageDir string, err error) {
	sessionIDVal := ctx.Value(interfaces.SessionIDContextKey)
	sessionStorageDirVal := ctx.Value(interfaces.SessionStorageContextKey)

	if sessionIDVal == nil {
		return "", "", fmt.Errorf("session ID not found in context")
	}
	if sessionStorageDirVal == nil {
		return "", "", fmt.Errorf("session storage directory not found in context")
	}

	sessionID, ok := sessionIDVal.(string)
	if !ok {
		return "", "", fmt.Errorf("session ID context value is not a string")
	}

	sessionStorageDir, ok = sessionStorageDirVal.(string)
	if !ok {
		return "", "", fmt.Errorf("session storage directory context value is not a string")
	}

	return sessionID, sessionStorageDir, nil
}

// loadBrowserDescription loads the browser tool description
func loadBrowserDescription() string {
	return tools.LoadToolDescription("browser")
}
