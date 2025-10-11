package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"mix/internal/llm/interfaces"
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
		Name:        "show_media",
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
							"description": "Absolute path to the media file (required for image/video/audio, optional for gsap_animation)",
						},
						"type": map[string]any{
							"type":        "string",
							"description": "Media type",
							"enum":        []string{"image", "video", "audio", "gsap_animation", "pdf", "csv"},
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
							"description": "Configuration data for gsap_animation type (JSON object with animation settings)",
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

		// Path is only required for physical file types (not gsap_animation)
		if output.Type != "gsap_animation" && output.Path == "" {
			return NewTextErrorResponse(fmt.Sprintf("Output %d missing path", i)), nil
		}

		// Validate media type
		validTypes := map[string]bool{
			"image":          true,
			"video":          true,
			"audio":          true,
			"gsap_animation": true,
			"pdf":            true,
			"csv":            true,
		}
		if !validTypes[output.Type] {
			return NewTextErrorResponse(fmt.Sprintf("Invalid media type '%s' for output %d", output.Type, i)), nil
		}

		// Require HTTP/HTTPS URLs for all types except gsap_animation
		if output.Type != "gsap_animation" && !isURL(output.Path) {
			return NewTextErrorResponse(fmt.Sprintf("For the show_media tool ,path must be a valid HTTP/HTTPS URL for output %d: %s", i, output.Path)), nil
		}

		// For gsap_animation, validate that config is provided
		if output.Type == "gsap_animation" {
			if output.Config == nil {
				return NewTextErrorResponse(fmt.Sprintf("gsap_animation type requires config parameter for output %d", i)), nil
			}

			configMap, ok := output.Config.(map[string]interface{})
			if !ok {
				return NewTextErrorResponse(fmt.Sprintf("gsap_animation config must be a JSON object for output %d", i)), nil
			}

			url, exists := configMap["url"]
			if !exists || url == nil {
				return NewTextErrorResponse(fmt.Sprintf("gsap_animation requires config.url field for output %d", i)), nil
			}

			urlStr, ok := url.(string)
			if !ok || urlStr == "" {
				return NewTextErrorResponse(fmt.Sprintf("gsap_animation config.url must be a non-empty string for output %d", i)), nil
			}

			// Basic URL validation
			if !strings.HasPrefix(urlStr, "http://") && !strings.HasPrefix(urlStr, "https://") {
				return NewTextErrorResponse(fmt.Sprintf("gsap_animation config.url must be a valid HTTP/HTTPS URL for output %d", i)), nil
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

// GetCallbacks implements the CallbackTool interface
// This allows post-processing of showcased media files
func (t *mediaShowcaseTool) GetCallbacks() []interfaces.CallbackConfig {
	// Callbacks are disabled by default
	// To enable, return a configured callback array like:
	//
	// return []interfaces.CallbackConfig{
	// 	{
	// 		Type: interfaces.CallbackTypeBashScript,
	// 		BashCommand: `
	// 			echo "Processing media: $CALLBACK_TOOL_NAME" >> media_callback.log
	// 			echo "Session: $CALLBACK_SESSION_ID" >> media_callback.log
	// 			echo "Result: $CALLBACK_TOOL_RESULT" >> media_callback.log
	// 		`,
	// 		BashTimeout: 30000, // 30 seconds
	// 		NonBlocking: true,
	// 	},
	// }

	return []interfaces.CallbackConfig{}
}
