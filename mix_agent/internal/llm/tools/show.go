package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"mix/internal/logging"
)

const (
	contentTypeMarkdown = "markdown"
	contentTypeStatus   = "status"
	contentTypeJSON     = "json"
)

type showTool struct{}

type ShowParams struct {
	Outputs []MediaOutput `json:"outputs"`
}

type MediaOutput struct {
	Path      string `json:"path,omitempty"`
	Data      string `json:"data,omitempty"` // Inline content for markdown/json/status types
	Type      string `json:"type"`
	Title     string `json:"title"`
	StartTime *int   `json:"startTime,omitempty"` // Optional: start time in seconds for video/audio segments
	Duration  *int   `json:"duration,omitempty"`  // Optional: duration in seconds for video/audio segments
}

func NewShowTool() BaseTool {
	return &showTool{}
}

func (t *showTool) Info() ToolInfo {
	return ToolInfo{
		Name:        "Show",
		Description: LoadToolDescription("show"),
		Parameters: map[string]any{
			"outputs": map[string]any{
				"type":        "array",
				"description": "REQUIRED: Must be an array of media output objects. Each object represents one piece of content to display. Example: [{\"type\": \"image\", \"title\": \"Screenshot\", \"path\": \"https://example.com/image.png\"}]",
				"minItems":    1,
				"items": map[string]any{
					"type": "object",
					"properties": map[string]any{
						"path": map[string]any{
							"type":        "string",
							"description": "URL to file (for image/video/audio/pdf/csv). Not used for markdown, json, or status types.",
						},
						"data": map[string]any{
							"type":        "string",
							"description": "Inline content (for markdown/json/status types). For json type, this should be a JSON string. For status type, this is the status message.",
						},
						"type": map[string]any{
							"type":        "string",
							"description": "Content type",
							"enum":        []string{"image", "video", "audio", "pdf", "csv", "markdown", "json", "status"},
						},
						"title": map[string]any{
							"type":        "string",
							"description": "Display title for the content.",
						},
						"startTime": map[string]any{
							"type":        "integer",
							"description": "Optional: start time in seconds for video/audio segments",
							"minimum":     0,
						},
						"duration": map[string]any{
							"type":        "integer",
							"description": "Optional: duration in seconds for video/audio segments",
							"minimum":     1,
						},
					},
					"required": []string{"type", "title"},
				},
			},
		},
		Required: []string{"outputs"},
	}
}

func (t *showTool) Run(ctx context.Context, call ToolCall) (ToolResponse, error) {
	logging.Debug("Show tool received input", "input", call.Input)

	var params ShowParams
	if err := json.Unmarshal([]byte(call.Input), &params); err != nil {
		// Log the malformed input for debugging
		logging.Error("Show tool failed to parse parameters",
			"error", err,
			"raw_input", call.Input)

		// Check if outputs is not an array to provide better error message
		var raw map[string]any
		if parseErr := json.Unmarshal([]byte(call.Input), &raw); parseErr == nil {
			if outputs, exists := raw["outputs"]; exists {
				logging.Error("Show tool outputs type mismatch",
					"outputs_type", fmt.Sprintf("%T", outputs),
					"outputs_value", outputs)
			}
		}

		return NewTextErrorResponse(fmt.Sprintf("Invalid parameters: %v. The 'outputs' parameter must be an array of objects, e.g., [{\"type\": \"image\", \"title\": \"My Image\", \"path\": \"https://...\"}]", err)), nil
	}

	if len(params.Outputs) == 0 {
		return NewTextErrorResponse("No media outputs provided"), nil
	}

	// Validate each media output
	for i, output := range params.Outputs {
		if output.Type == "" {
			return NewTextErrorResponse(fmt.Sprintf("Output %d missing type", i)), nil
		}

		// All types require title
		if output.Title == "" {
			return NewTextErrorResponse(fmt.Sprintf("Output %d missing title", i)), nil
		}

		// Validate content type
		validTypes := map[string]bool{
			"image":    true,
			"video":    true,
			"audio":    true,
			"pdf":      true,
			"csv":      true,
			"markdown": true,
			"json":     true,
			"status":   true,
		}
		if !validTypes[output.Type] {
			return NewTextErrorResponse(fmt.Sprintf("Invalid content type '%s' for output %d", output.Type, i)), nil
		}

		// Type-specific validation
		switch output.Type {
		case contentTypeStatus:
			// Status requires data field
			if output.Data == "" {
				return NewTextErrorResponse(fmt.Sprintf("Status output %d missing data field", i)), nil
			}
		case contentTypeMarkdown:
			// Markdown requires data field for inline content
			if output.Data == "" {
				return NewTextErrorResponse(fmt.Sprintf("Markdown output %d missing data field", i)), nil
			}
		case contentTypeJSON:
			// JSON requires data field
			if output.Data == "" {
				return NewTextErrorResponse(fmt.Sprintf("JSON output %d missing data field", i)), nil
			}
			// Validate that data contains valid JSON
			var jsonData any
			if err := json.Unmarshal([]byte(output.Data), &jsonData); err != nil {
				return NewTextErrorResponse(fmt.Sprintf("JSON output %d has invalid JSON in data field: %v", i, err)), nil
			}
		default:
			// All file-based types (image, video, audio, pdf, csv) require path with URL
			if output.Path == "" {
				return NewTextErrorResponse(fmt.Sprintf("Output %d missing path field", i)), nil
			}
			if !isURL(output.Path) {
				return NewTextErrorResponse(fmt.Sprintf("For the show tool, path must be a valid HTTP/HTTPS URL for output %d: %s", i, output.Path)), nil
			}
		}

		// Validate timing fields if provided
		if output.StartTime != nil {
			if *output.StartTime < 0 {
				return NewTextErrorResponse(fmt.Sprintf("startTime must be >= 0 for output %d", i)), nil
			}
		}
		if output.Duration != nil {
			if *output.Duration <= 0 {
				return NewTextErrorResponse(fmt.Sprintf("duration must be > 0 for output %d", i)), nil
			}
		}
	}

	// Create success message
	titles := make([]string, len(params.Outputs))
	for i, output := range params.Outputs {
		if output.Type == contentTypeStatus {
			titles[i] = "status"
		} else {
			titles[i] = output.Title
		}
	}

	message := fmt.Sprintf("Successfully displaying %d item(s): %s",
		len(params.Outputs),
		strings.Join(titles, ", "))

	return ToolResponse{
		Type:    "text",
		Content: message,
	}, nil
}
