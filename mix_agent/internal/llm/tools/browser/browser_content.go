package browser

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"sort"
	"strings"
	"time"

	browserprotocol "github.com/sarathmenon/browser-service/pkg/protocol"

	"mix/internal/llm/interfaces"
)

// handleReadPage returns accessibility tree for visible viewport elements
func (b *browserTool) handleReadPage(ctx context.Context, params BrowserParams, sessionID string) interfaces.ToolResponse {
	client, err := b.getClient(ctx, sessionID)
	if err != nil {
		return interfaces.NewTextErrorResponse(fmt.Sprintf("Failed to get browser client: %v", err))
	}

	// Use interactiveOnly optimization only for "interactive" filter
	interactiveOnly := params.Filter == FilterInteractive

	// Call browser service (tabID is always required and validated)
	result, err := client.ReadPage(ctx, interactiveOnly, params.TabID)
	if err != nil {
		return interfaces.NewTextErrorResponse(fmt.Sprintf("Read page failed: %v", err))
	}

	// Apply additional element filtering
	elements := applyElementFilter(result.Elements, params.Filter)

	response := formatReadPageResponse(elements, result.Viewport, params.Filter)
	return interfaces.NewTextResponse(response)
}

// applyElementFilter filters accessibility nodes by the requested filter type
func applyElementFilter(elements []browserprotocol.RawAccessibilityNode, filter string) []browserprotocol.RawAccessibilityNode {
	switch filter {
	case FilterInteractive, "":
		return elements // interactive already filtered by client; empty means all
	case FilterLinks:
		return filterByRole(elements, "link")
	case FilterButtons:
		return filterByRole(elements, "button")
	case FilterHeadings:
		return filterByRole(elements, "heading")
	case FilterText:
		return filterTextElements(elements)
	default:
		return elements
	}
}

// filterByRole returns only elements matching the given role (case-insensitive)
func filterByRole(elements []browserprotocol.RawAccessibilityNode, role string) []browserprotocol.RawAccessibilityNode {
	filtered := make([]browserprotocol.RawAccessibilityNode, 0)
	for _, elem := range elements {
		if strings.EqualFold(elem.Role, role) {
			filtered = append(filtered, elem)
		}
	}
	return filtered
}

var textRoles = map[string]bool{
	"statictext": true,
	"paragraph":  true,
	"heading":    true,
	"label":      true,
	"caption":    true,
	"figure":     true,
}

// filterTextElements returns only text-content elements
func filterTextElements(elements []browserprotocol.RawAccessibilityNode) []browserprotocol.RawAccessibilityNode {
	filtered := make([]browserprotocol.RawAccessibilityNode, 0)
	for _, elem := range elements {
		if textRoles[strings.ToLower(elem.Role)] {
			filtered = append(filtered, elem)
		}
	}
	return filtered
}

// buildFrameNumberMap creates consistent frame number mapping
// Maps CDP FrameID strings to integers (f0, f1, f2, ...)
func buildFrameNumberMap(elements []browserprotocol.RawAccessibilityNode) map[string]int {
	frameMap := make(map[string]int)
	frameCounter := 0

	for _, elem := range elements {
		if elem.FrameID == "" {
			continue
		}
		if _, exists := frameMap[elem.FrameID]; !exists {
			frameMap[elem.FrameID] = frameCounter
			frameCounter++
		}
	}

	return frameMap
}

// formatAttributes converts attribute map to inline string
// Priority order: href, id, type, placeholder, name, aria-label
// Then remaining attributes alphabetically
func formatAttributes(attrs map[string]string) string {
	if len(attrs) == 0 {
		return ""
	}

	parts := make([]string, 0, len(attrs))

	// Priority attributes in specific order
	priority := []string{"href", "id", "type", "placeholder", "name", "aria-label"}
	for _, key := range priority {
		if val, exists := attrs[key]; exists {
			parts = append(parts, fmt.Sprintf(`%s=%q`, key, val))
		}
	}

	// Remaining attributes (alphabetically sorted)
	var otherKeys []string
	for key := range attrs {
		if !slices.Contains(priority, key) {
			otherKeys = append(otherKeys, key)
		}
	}
	sort.Strings(otherKeys)

	for _, key := range otherKeys {
		parts = append(parts, fmt.Sprintf(`%s=%q`, key, attrs[key]))
	}

	return " " + strings.Join(parts, " ")
}

// formatReadPageResponse formats the read_page response for the LLM
func formatReadPageResponse(elements []browserprotocol.RawAccessibilityNode, viewport browserprotocol.BoundingBox, filter string) string {
	var sb strings.Builder

	filterLabel := "all"
	if filter != "" {
		filterLabel = filter
	}

	fmt.Fprintf(&sb, "Accessibility tree (%s elements in viewport)\n", filterLabel)
	fmt.Fprintf(&sb, "Viewport: %.0fx%.0f at scroll position (%.0f, %.0f)\n\n",
		viewport.Width, viewport.Height, viewport.X, viewport.Y)
	fmt.Fprintf(&sb, "Found %d element(s):\n\n", len(elements))

	// Build frame number map for consistent reference IDs
	frameMap := buildFrameNumberMap(elements)

	for _, elem := range elements {
		// Format: - role "name" [ref=fX_ref_Y] (x=X,y=Y) attrs...

		// 1. Role
		fmt.Fprintf(&sb, "- %s", elem.Role)

		// 2. Name (quoted, if present)
		if elem.Name != "" {
			fmt.Fprintf(&sb, " %q", elem.Name)
		}

		// 3. Reference ID [ref=f{frameNum}_ref_{backendID}]
		frameNum := 0
		if elem.FrameID != "" {
			if num, exists := frameMap[elem.FrameID]; exists {
				frameNum = num
			}
		}
		fmt.Fprintf(&sb, " [ref=f%d_ref_%d]", frameNum, elem.BackendID)

		// 4. Coordinates (x=X,y=Y) - center of element for clicking
		centerX := elem.Bounds.X + elem.Bounds.Width/2
		centerY := elem.Bounds.Y + elem.Bounds.Height/2
		fmt.Fprintf(&sb, " (x=%.0f,y=%.0f)", centerX, centerY)

		// 5. Attributes (inline, space-separated)
		if len(elem.Attributes) > 0 {
			sb.WriteString(formatAttributes(elem.Attributes))
		}

		sb.WriteString("\n")
	}

	return sb.String()
}

// handleFind searches for elements matching a query
func (b *browserTool) handleFind(ctx context.Context, params BrowserParams, sessionID string) interfaces.ToolResponse {
	// Validate query
	if params.Query == "" {
		return interfaces.NewTextErrorResponse("missing query parameter for find action")
	}

	// Get session storage dir for potential file writing
	_, sessionStorageDir, err := b.getContextInfo(ctx)
	if err != nil {
		return interfaces.NewTextErrorResponse(fmt.Sprintf("Failed to get session info: %v", err))
	}

	// Get browser connection
	client, err := b.getClient(ctx, sessionID)
	if err != nil {
		return interfaces.NewTextErrorResponse(fmt.Sprintf("Failed to get browser client: %v", err))
	}

	// Search for elements (tabID is always required and validated)
	result, err := client.Find(ctx, params.Query, 100, params.TabID)
	if err != nil {
		return interfaces.NewTextErrorResponse(fmt.Sprintf("Search failed: %v", err))
	}

	// No results
	if result.Total == 0 {
		return interfaces.NewTextResponse(fmt.Sprintf("No elements found matching query: %s", params.Query))
	}

	// If more than 100 results, write to file
	if result.Total > 100 {
		// Format all results for file
		var fileContent strings.Builder
		fmt.Fprintf(&fileContent, "Find Results for query: %s\n", params.Query)
		fmt.Fprintf(&fileContent, "Total matches: %d\n\n", result.Total)

		for i, elem := range result.Elements {
			fmt.Fprintf(&fileContent, "[%d] %s: %s (x:%.0f, y:%.0f)\n", i+1, elem.Role, elem.Name, elem.Bounds.X, elem.Bounds.Y)
		}

		// Write to session storage
		timestamp := time.Now().Format("20060102_150405")
		filename := fmt.Sprintf("find_results_%s.txt", timestamp)
		filePath := filepath.Join(sessionStorageDir, filename)

		if err := os.WriteFile(filePath, []byte(fileContent.String()), 0o644); err != nil {
			return interfaces.NewTextErrorResponse(fmt.Sprintf("Failed to save results to file: %v", err))
		}

		// Return response with file reference
		return interfaces.NewTextResponse(
			fmt.Sprintf("Found %d elements matching query: %s\n\n⚠️  Results exceed 100 items. Full results saved to: %s\n\nUse the Read tool to view complete results.",
				result.Total, params.Query, filename))
	}

	// Format inline response for <= 100 results
	var response strings.Builder
	fmt.Fprintf(&response, "Found %d element(s) matching query: %s\n\n", result.Total, params.Query)

	if result.Truncated {
		fmt.Fprintf(&response, "⚠️  Showing first %d of %d results\n\n", len(result.Elements), result.Total)
	}

	// Show elements
	for i, elem := range result.Elements {
		fmt.Fprintf(&response, "[%d] %s: %s (x:%.0f, y:%.0f)\n", i+1, elem.Role, elem.Name, elem.Bounds.X, elem.Bounds.Y)
	}

	return interfaces.NewTextResponse(response.String())
}
