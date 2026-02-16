package browser

import (
	"context"
	"fmt"
	"regexp"
	"strings"

	"mix/internal/llm/interfaces"
)

// Valid keyboard keys
var validKeys = map[string]bool{
	// Navigation
	"Enter": true, "Tab": true, "Escape": true, "Space": true,
	"ArrowUp": true, "ArrowDown": true, "ArrowLeft": true, "ArrowRight": true,
	"Home": true, "End": true, "PageUp": true, "PageDown": true,

	// Editing
	"Backspace": true, "Delete": true, "Insert": true,

	// Function keys
	"F1": true, "F2": true, "F3": true, "F4": true, "F5": true, "F6": true,
	"F7": true, "F8": true, "F9": true, "F10": true, "F11": true, "F12": true,
}

// Modifier pattern: cmd+x, ctrl+x, shift+x, alt+x
var modifierPattern = regexp.MustCompile(`^(cmd|ctrl|shift|alt)\+.+$`)

// keyboardSegment represents either text to type or a key to press
type keyboardSegment struct {
	isKey bool
	value string
}

// validateKeyboardInput validates the input string for proper syntax
func validateKeyboardInput(text string) error {
	i := 0
	for i < len(text) {
		switch text[i] {
		case '{':
			// Check for escape sequence {{
			if i+1 < len(text) && text[i+1] == '{' {
				i += 2
				continue
			}

			// Find closing brace
			closeIdx := strings.IndexByte(text[i+1:], '}')
			if closeIdx == -1 {
				return fmt.Errorf("unclosed brace at position %d", i)
			}
			closeIdx += i + 1

			// Extract key name
			keyName := text[i+1 : closeIdx]
			if keyName == "" {
				return fmt.Errorf("empty key sequence at position %d", i)
			}

			// Validate key name
			if !isValidKey(keyName) {
				return fmt.Errorf("unknown key: %s (valid keys: Enter, Tab, Backspace, Escape, ArrowUp, ArrowDown, ArrowLeft, ArrowRight, Delete, Home, End, PageUp, PageDown, Insert, Space, F1-F12, or modifiers like cmd+a, ctrl+c)", keyName)
			}

			i = closeIdx + 1
		case '}':
			// Check for escape sequence }}
			if i+1 < len(text) && text[i+1] == '}' {
				i += 2
				continue
			}
			// Unmatched closing brace
			return fmt.Errorf("unmatched closing brace at position %d", i)
		default:
			i++
		}
	}
	return nil
}

// isValidKey checks if a key name is valid
func isValidKey(key string) bool {
	// Check if it's a simple valid key
	if validKeys[key] {
		return true
	}

	// Check if it's a modifier combination (cmd+a, ctrl+c, etc.)
	if modifierPattern.MatchString(key) {
		return true
	}

	return false
}

// parseKeyboardInput parses text with {key} syntax into segments
func parseKeyboardInput(text string) []keyboardSegment {
	var segments []keyboardSegment
	var currentText strings.Builder
	i := 0

	for i < len(text) {
		switch {
		case text[i] == '{':
			// Check for escape sequence {{
			if i+1 < len(text) && text[i+1] == '{' {
				currentText.WriteByte('{')
				i += 2
				continue
			}

			// Save accumulated text
			if currentText.Len() > 0 {
				segments = append(segments, keyboardSegment{
					isKey: false,
					value: currentText.String(),
				})
				currentText.Reset()
			}

			// Find closing brace
			closeIdx := strings.IndexByte(text[i+1:], '}')
			closeIdx += i + 1

			// Extract and save key
			keyName := text[i+1 : closeIdx]
			segments = append(segments, keyboardSegment{
				isKey: true,
				value: keyName,
			})

			i = closeIdx + 1
		case text[i] == '}' && i+1 < len(text) && text[i+1] == '}':
			// Handle escaped closing brace }}
			currentText.WriteByte('}')
			i += 2
		default:
			currentText.WriteByte(text[i])
			i++
		}
	}

	// Save any remaining text
	if currentText.Len() > 0 {
		segments = append(segments, keyboardSegment{
			isKey: false,
			value: currentText.String(),
		})
	}

	return segments
}

// handleType types text and/or presses keys using {key} syntax
func (b *browserTool) handleType(ctx context.Context, params BrowserParams, sessionID string) interfaces.ToolResponse {
	// Validate text parameter
	if params.Text == "" {
		return interfaces.NewTextErrorResponse("missing text parameter for type action")
	}

	// Validate keyboard input syntax
	if err := validateKeyboardInput(params.Text); err != nil {
		return interfaces.NewTextErrorResponse(err.Error())
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

	// Parse keyboard input into segments
	segments := parseKeyboardInput(params.Text)

	// If there's an index, click the element first (only once)
	if params.Index != nil {
		// This will be handled by the first Type call if it's text,
		// or we need to handle it specially if first segment is a key
		if len(segments) > 0 && segments[0].isKey {
			// Click element before pressing key
			backendID, err := b.backendIDFromIndex(ctx, sessionID, params.TabID, *params.Index)
			if err != nil {
				return interfaces.NewTextErrorResponse(fmt.Sprintf("Failed to resolve element index: %v", err))
			}
			if err := client.ClickByBackendID(ctx, backendID, params.TabID); err != nil {
				return interfaces.NewTextErrorResponse(fmt.Sprintf("Failed to click element: %v", err))
			}
		}
	}

	// Execute each segment
	for i, segment := range segments {
		if segment.isKey {
			// Press key
			err = client.PressKey(ctx, segment.value, params.TabID)
			if err != nil {
				return interfaces.NewTextErrorResponse(fmt.Sprintf("Key press failed for '%s': %v", segment.value, err))
			}
		} else {
			// Type text - only pass index for the very first text segment
			var indexToUse *int
			if i == 0 && params.Index != nil {
				indexToUse = params.Index
			}
			err = client.Type(ctx, indexToUse, segment.value, params.TabID)
			if err != nil {
				return interfaces.NewTextErrorResponse(fmt.Sprintf("Type failed: %v", err))
			}
		}
	}

	if params.Index != nil {
		return interfaces.NewTextResponse(fmt.Sprintf("Successfully processed keyboard input into element %d", *params.Index))
	}
	return interfaces.NewTextResponse("Successfully processed keyboard input into focused element")
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

	// Validate index parameter (required for form_input)
	if params.Index == nil {
		return interfaces.NewTextErrorResponse("missing index parameter for form_input action")
	}

	// Convert value to string (handles string, number, boolean)
	valueStr := fmt.Sprintf("%v", params.Value)

	// Set form input value (tabID is always required and validated)
	err = client.FormInput(ctx, *params.Index, valueStr, params.TabID)
	if err != nil {
		return interfaces.NewTextErrorResponse(fmt.Sprintf("Form input failed: %v", err))
	}

	return interfaces.NewTextResponse(fmt.Sprintf("Successfully set value in element %d", *params.Index))
}

// executeType executes a type sub-action with {key} syntax support
func (b *browserTool) executeType(ctx context.Context, client BrowserClient, action SubAction, sessionID, tabID string) error {
	// Validate keyboard input syntax
	if err := validateKeyboardInput(action.Text); err != nil {
		return err
	}

	// Parse keyboard input into segments
	segments := parseKeyboardInput(action.Text)

	// If there's an index, click the element first (only if first segment is a key)
	if action.Index != nil && len(segments) > 0 && segments[0].isKey {
		// Need to click element before pressing key
		backendID, err := b.backendIDFromIndex(ctx, sessionID, tabID, *action.Index)
		if err != nil {
			return fmt.Errorf("failed to resolve element index: %w", err)
		}
		if err := client.ClickByBackendID(ctx, backendID, tabID); err != nil {
			return fmt.Errorf("failed to click element: %w", err)
		}
	}

	// Execute each segment
	for i, segment := range segments {
		if segment.isKey {
			// Press key
			if err := client.PressKey(ctx, segment.value, tabID); err != nil {
				return fmt.Errorf("key press failed for '%s': %w", segment.value, err)
			}
		} else {
			// Type text - only pass index for the very first text segment
			var indexToUse *int
			if i == 0 && action.Index != nil {
				indexToUse = action.Index
			}
			if err := client.Type(ctx, indexToUse, segment.value, tabID); err != nil {
				return err
			}
		}
	}

	return nil
}
