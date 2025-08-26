package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
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
	Config      interface{} `json:"config,omitempty"`    // For remotion configuration data
	StartTime   *int        `json:"startTime,omitempty"` // Optional: start time in seconds for video/audio segments
	Duration    *int        `json:"duration,omitempty"`  // Optional: duration in seconds for video/audio segments
}

func NewMediaShowcaseTool() BaseTool {
	return &mediaShowcaseTool{}
}

func (t *mediaShowcaseTool) Info() ToolInfo {
	return ToolInfo{
		Name:        "media_showcase",
		Description: LoadToolDescription("media_showcase"),
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
							"enum":        []string{"image", "video", "audio", "gsap_animation"},
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
		}
		if !validTypes[output.Type] {
			return NewTextErrorResponse(fmt.Sprintf("Invalid media type '%s' for output %d", output.Type, i)), nil
		}

		// Check if file exists (skip for gsap_animation which doesn't require physical files)
		if output.Type != "gsap_animation" {
			if !filepath.IsAbs(output.Path) {
				return NewTextErrorResponse(fmt.Sprintf("Path must be absolute for output %d: %s", i, output.Path)), nil
			}

			if _, err := os.Stat(output.Path); err != nil {
				return NewTextErrorResponse(fmt.Sprintf("Media file not found for output %d: %s", i, output.Path)), nil
			}
		}

		// Validate file extension matches type (skip for gsap_animation which doesn't require physical files)
		if output.Type != "gsap_animation" {
			ext := strings.ToLower(filepath.Ext(output.Path))
			switch output.Type {
			case "image":
				if !isImageExtension(ext) {
					return NewTextErrorResponse(fmt.Sprintf("File extension '%s' doesn't match image type for output %d", ext, i)), nil
				}
			case "video":
				if !isVideoExtension(ext) {
					return NewTextErrorResponse(fmt.Sprintf("File extension '%s' doesn't match video type for output %d", ext, i)), nil
				}
			case "audio":
				if !isAudioExtension(ext) {
					return NewTextErrorResponse(fmt.Sprintf("File extension '%s' doesn't match audio type for output %d", ext, i)), nil
				}
			}
		} else {
			// For gsap_animation, validate that config is provided
			if output.Config == nil {
				return NewTextErrorResponse(fmt.Sprintf("%s type requires config parameter for output %d", output.Type, i)), nil
			}

			// Validate that config.url is provided for gsap_animation
			if output.Type == "gsap_animation" {
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

func isImageExtension(ext string) bool {
	imageExts := map[string]bool{
		".jpg": true, ".jpeg": true, ".png": true, ".gif": true,
		".bmp": true, ".tiff": true, ".tif": true, ".webp": true,
		".svg": true, ".ico": true, ".heic": true, ".heif": true,
	}
	return imageExts[ext]
}

func isVideoExtension(ext string) bool {
	videoExts := map[string]bool{
		".mp4": true, ".avi": true, ".mov": true, ".wmv": true,
		".flv": true, ".webm": true, ".mkv": true, ".m4v": true,
		".3gp": true, ".ogv": true, ".ts": true, ".mts": true,
	}
	return videoExts[ext]
}

func isAudioExtension(ext string) bool {
	audioExts := map[string]bool{
		".mp3": true, ".wav": true, ".flac": true, ".aac": true,
		".ogg": true, ".wma": true, ".m4a": true, ".opus": true,
		".aiff": true, ".au": true, ".ra": true,
	}
	return audioExts[ext]
}
