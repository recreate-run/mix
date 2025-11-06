package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
)

const (
	mediaTypeMarkdown = "markdown"
	mediaTypeCode     = "code"
)

type mediaShowcaseTool struct{}

type MediaShowcaseParams struct {
	Outputs []MediaOutput `json:"outputs"`
}

type MediaOutput struct {
	Path        string      `json:"path"`
	Type        string      `json:"type"`
	Title       string      `json:"title"`
	Description string      `json:"description"`
	Config      interface{} `json:"config,omitempty"`    // For configuration data
	StartTime   *int        `json:"startTime,omitempty"` // Optional: start time in seconds for video/audio segments
	Duration    *int        `json:"duration,omitempty"`  // Optional: duration in seconds for video/audio segments
}

func NewMediaShowcaseTool() BaseTool {
	return &mediaShowcaseTool{}
}

func (t *mediaShowcaseTool) Info() ToolInfo {
	return ToolInfo{
		Name:        "ShowMedia",
		Description: LoadToolDescription("show_media"),
		Parameters: map[string]any{
			"outputs": map[string]any{
				"type":        "array",
				"description": "Array of final media outputs to showcase",
				"items": map[string]any{
					"type": "object",
					"properties": map[string]any{
						"path": map[string]any{
							"type":        "string",
							"description": "Absolute path to the media file (required for image/video/audio/pdf/csv) or content string (for markdown/code)",
						},
						"type": map[string]any{
							"type":        "string",
							"description": "Media type",
							"enum":        []string{"image", "video", "audio", "pdf", "csv", "markdown", "code"},
						},
						"title": map[string]any{
							"type":        "string",
							"description": "Title or name for the media output",
						},
						"description": map[string]any{
							"type":        "string",
							"description": "Optional description or context",
						},
						"config": map[string]any{
							"type":        "object",
							"description": "Configuration data for code type (JSON object with language field for syntax highlighting)",
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

func (t *mediaShowcaseTool) Run(ctx context.Context, call ToolCall) (ToolResponse, error) {
	var params MediaShowcaseParams
	if err := json.Unmarshal([]byte(call.Input), &params); err != nil {
		return NewTextErrorResponse(fmt.Sprintf("Invalid parameters: %v", err)), nil
	}

	if len(params.Outputs) == 0 {
		return NewTextErrorResponse("No media outputs provided"), nil
	}

	// Validate each media output
	for i, output := range params.Outputs {
		if output.Type == "" {
			return NewTextErrorResponse(fmt.Sprintf("Output %d missing type", i)), nil
		}
		if output.Title == "" {
			return NewTextErrorResponse(fmt.Sprintf("Output %d missing title", i)), nil
		}

		// Path is required for all types
		// For markdown and code, path contains the content
		// For other types, path contains the URL
		if output.Path == "" {
			return NewTextErrorResponse(fmt.Sprintf("Output %d missing path/content", i)), nil
		}

		// Validate media type
		validTypes := map[string]bool{
			"image":    true,
			"video":    true,
			"audio":    true,
			"pdf":      true,
			"csv":      true,
			"markdown": true,
			"code":     true,
		}
		if !validTypes[output.Type] {
			return NewTextErrorResponse(fmt.Sprintf("Invalid media type '%s' for output %d", output.Type, i)), nil
		}

		// Require HTTP/HTTPS URLs for file types (not markdown or code)
		// markdown and code types use path field to store content directly
		if output.Type != mediaTypeMarkdown &&
			output.Type != mediaTypeCode &&
			!isURL(output.Path) {
			return NewTextErrorResponse(fmt.Sprintf("For the show_media tool ,path must be a valid HTTP/HTTPS URL for output %d: %s", i, output.Path)), nil
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
		titles[i] = output.Title
	}

	message := fmt.Sprintf("Successfully showcasing %d media output(s): %s",
		len(params.Outputs),
		strings.Join(titles, ", "))

	return ToolResponse{
		Type:    "text",
		Content: message,
	}, nil
}
