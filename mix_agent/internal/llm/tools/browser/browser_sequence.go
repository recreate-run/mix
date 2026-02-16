package browser

import (
	"context"
	"encoding/base64"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	browserprotocol "github.com/sarathmenon/browser-service/pkg/protocol"

	"mix/internal/llm/interfaces"
)

// handleActionSequence executes multiple actions in sequence
func (b *browserTool) handleActionSequence(ctx context.Context, params BrowserParams, sessionID, sessionStorageDir string) interfaces.ToolResponse {
	if len(params.Actions) == 0 {
		return interfaces.NewTextErrorResponse("missing actions array for action sequence")
	}

	client, err := b.getClient(ctx, sessionID)
	if err != nil {
		return interfaces.NewTextErrorResponse(fmt.Sprintf("Failed to get browser client: %v", err))
	}

	results := make([]SubActionResult, len(params.Actions))
	successCount := 0
	var lastErr error

	// Execute each action in sequence
	for i := range params.Actions {
		subAction := &params.Actions[i]
		result := SubActionResult{
			Index: i,
			Type:  subAction.Type,
		}

		// Execute sub-action
		screenshotFile, err := b.executeSubAction(ctx, client, *subAction, sessionID, params.TabID, sessionStorageDir)
		if err != nil {
			result.Success = false
			result.Error = err.Error()
			lastErr = err
			results[i] = result
			// Fail-fast: stop on first error
			break
		}

		result.Success = true
		result.ScreenshotFile = screenshotFile
		successCount++
		results[i] = result

		// Inter-action delay
		if i < len(params.Actions)-1 {
			delay := getDelayForAction(subAction.Type)
			time.Sleep(delay)
		}
	}

	// Check if any sub-action was a screenshot
	hasScreenshotSubAction := false
	for _, result := range results {
		if result.Type == SubActionScreenshot && result.Success {
			hasScreenshotSubAction = true
			break
		}
	}

	// Take automatic screenshot after successful sequence (only if no screenshot sub-action)
	var screenshotFile string
	if lastErr == nil && !hasScreenshotSubAction {
		screenshotParams := browserprotocol.ScreenshotParams{
			Format:   "png",
			FullPage: false,
			Raw:      true,
			TabID:    &params.TabID, // TabID is always required and validated
		}

		screenshotResult, err := client.Screenshot(ctx, screenshotParams)
		if err == nil && screenshotResult != nil {
			filename, err := saveScreenshot(screenshotResult.Data, sessionStorageDir)
			if err == nil {
				screenshotFile = filename
			}
		}
	}

	// Format response
	var response strings.Builder
	fmt.Fprintf(&response, "Action Sequence Results (%d/%d successful)\n\n", successCount, len(params.Actions))

	for _, result := range results {
		if result.Type == "" {
			continue // Skipped action
		}

		status := "✓ Success"
		if !result.Success {
			status = fmt.Sprintf("✗ Failed: %s", result.Error)
		}

		fmt.Fprintf(&response, "[%d] %s %s\n", result.Index, result.Type, status)

		// Include screenshot file from sub-action if present
		if result.ScreenshotFile != "" {
			fmt.Fprintf(&response, "    Screenshot: %s\n", formatScreenshotResponse(result.ScreenshotFile, sessionID, b.baseURL))
		}
	}

	if screenshotFile != "" {
		fmt.Fprintf(&response, "\nFinal Screenshot: %s\n", formatScreenshotResponse(screenshotFile, sessionID, b.baseURL))
	}

	if lastErr != nil {
		response.WriteString("\nSequence stopped early due to error.\n")
	}

	return interfaces.NewTextResponse(response.String())
}

// executeSubAction executes a single sub-action
func (b *browserTool) executeSubAction(ctx context.Context, client BrowserClient, action SubAction, sessionID, tabID, sessionStorageDir string) (screenshotFile string, err error) {
	switch action.Type {
	case SubActionLeftClick:
		return "", b.executeClick(ctx, client, action, sessionID, tabID)
	case SubActionRightClick:
		return "", b.executeRightClick(ctx, client, action, sessionID, tabID)
	case SubActionDoubleClick:
		return "", b.executeDoubleClick(ctx, client, action, sessionID, tabID)
	case SubActionTripleClick:
		return "", b.executeTripleClick(ctx, client, action, sessionID, tabID)
	case SubActionType:
		return "", b.executeType(ctx, client, action, sessionID, tabID)
	case SubActionScroll:
		amount := action.ScrollAmount
		if amount == 0 {
			amount = 100
		}
		return "", client.Scroll(ctx, action.Direction, amount, tabID)
	case SubActionScrollTo:
		if action.Index == nil {
			return "", fmt.Errorf("index required for scroll_to action")
		}
		backendID, err := b.backendIDFromIndex(ctx, sessionID, tabID, *action.Index)
		if err != nil {
			return "", err
		}
		return "", client.ScrollIntoViewByBackendID(ctx, backendID, tabID)
	case SubActionFormInput:
		if action.Index == nil {
			return "", fmt.Errorf("index required for form_input action")
		}
		// Convert value to string (handles string, number, boolean)
		valueStr := fmt.Sprintf("%v", action.Value)
		return "", client.FormInput(ctx, *action.Index, valueStr, tabID)
	case SubActionWait:
		return "", client.Wait(ctx, action.Duration, tabID)
	case SubActionLeftClickDrag:
		// Support both index mode and coordinate mode
		hasIndexMode := action.FromIndex != nil && action.ToIndex != nil
		hasCoordMode := action.FromX != nil && action.FromY != nil && action.ToX != nil && action.ToY != nil

		if !hasIndexMode && !hasCoordMode {
			return "", fmt.Errorf("left_click_drag requires either (fromIndex and toIndex) or (fromX, fromY, toX, toY)")
		}
		if hasIndexMode && hasCoordMode {
			return "", fmt.Errorf("left_click_drag cannot mix index mode and coordinate mode")
		}

		// Set duration with default of 500ms if not specified
		duration := 500
		if action.Duration > 0 {
			duration = action.Duration
		}

		if hasIndexMode {
			// Index mode: drag from element to element
			return "", client.Drag(ctx, action.FromIndex, action.ToIndex, nil, nil, nil, nil, &duration, tabID)
		}

		// Coordinate mode: drag from coordinate to coordinate
		return "", client.Drag(ctx, nil, nil, action.FromX, action.FromY, action.ToX, action.ToY, &duration, tabID)
	case SubActionScreenshot:
		// Take screenshot and save to session storage
		screenshotParams := browserprotocol.ScreenshotParams{
			Format:   "png",
			FullPage: false,
			Raw:      true,
			TabID:    &tabID,
		}
		result, err := client.Screenshot(ctx, screenshotParams)
		if err != nil {
			return "", fmt.Errorf("screenshot failed: %w", err)
		}

		// Save screenshot - use custom file_path if provided
		var filename string
		if action.FilePath != "" {
			// Custom filename specified - decode base64 data
			imgData, err := base64.StdEncoding.DecodeString(result.Data)
			if err != nil {
				return "", fmt.Errorf("failed to decode screenshot: %w", err)
			}
			filename = action.FilePath
			fullPath := filepath.Join(sessionStorageDir, filename)
			if err := os.WriteFile(fullPath, imgData, 0o644); err != nil {
				return "", fmt.Errorf("failed to save screenshot: %w", err)
			}
		} else {
			// Auto-generate filename
			filename, err = saveScreenshot(result.Data, sessionStorageDir)
			if err != nil {
				return "", fmt.Errorf("failed to save screenshot: %w", err)
			}
		}

		return filename, nil
	default:
		return "", fmt.Errorf("unknown sub-action type: %s", action.Type)
	}
}

// getDelayForAction returns appropriate inter-action delay
func getDelayForAction(actionType string) time.Duration {
	switch actionType {
	case SubActionLeftClick, SubActionRightClick, SubActionDoubleClick, SubActionTripleClick:
		return 100 * time.Millisecond
	case SubActionType, SubActionFormInput:
		return 50 * time.Millisecond
	default:
		return 50 * time.Millisecond
	}
}
