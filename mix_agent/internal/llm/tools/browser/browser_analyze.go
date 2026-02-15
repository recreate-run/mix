package browser

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"image"
	"image/color"
	"image/draw"
	"image/jpeg"
	"image/png"
	"os"
	"path/filepath"
	"strings"
	"time"

	browserprotocol "github.com/sarathmenon/browser-service/pkg/protocol"

	"mix/internal/config"
	"mix/internal/constants"
	"mix/internal/llm/interfaces"
	"mix/internal/llm/models"
	"mix/internal/llm/provider"
	"mix/internal/message"
)

// handleAnalyzeScreenshot analyzes a screenshot using Gemini
func (b *browserTool) handleAnalyzeScreenshot(ctx context.Context, params BrowserParams, sessionID string) interfaces.ToolResponse {
	// Validate prompt parameter
	if params.Prompt == "" {
		return interfaces.NewTextErrorResponse("missing prompt parameter for analyze_screenshot action")
	}

	// Get browser client
	client, err := b.getClient(ctx, sessionID)
	if err != nil {
		return interfaces.NewTextErrorResponse(fmt.Sprintf("Failed to get browser client: %v", err))
	}

	// Capture screenshot with viewport dimensions for coordinate conversion
	screenshotParams := browserprotocol.ScreenshotParams{
		Format:   "png",
		FullPage: false,
		Raw:      true, // Need viewport dimensions for bounding box coordinate conversion
		TabID:    &params.TabID,
	}

	screenshotStart := time.Now()
	result, err := client.Screenshot(ctx, screenshotParams)
	screenshotDuration := time.Since(screenshotStart)
	if shouldEnableDebugVisualization() {
		fmt.Printf("[DEBUG] Screenshot capture took: %v\n", screenshotDuration)
	}
	if err != nil {
		return interfaces.NewTextErrorResponse(fmt.Sprintf("Screenshot failed: %v", err))
	}

	// Decode base64 screenshot data
	imageData, err := base64.StdEncoding.DecodeString(result.Data)
	if err != nil {
		return interfaces.NewTextErrorResponse(fmt.Sprintf("Failed to decode screenshot: %v", err))
	}

	// Save screenshot and get URL
	screenshotURL, err := b.screenshotStorage.Save(ctx, sessionID, imageData)
	if err != nil {
		// Log error but continue (screenshot saving is non-critical)
		fmt.Printf("[WARN] Failed to save screenshot: %v\n", err)
	}

	// Get actual image dimensions (required for coordinate conversion)
	imageWidth, imageHeight, err := getImageDimensions(imageData)
	if err != nil {
		return interfaces.NewTextErrorResponse(fmt.Sprintf("Failed to get image dimensions: %v", err))
	}

	// Create Gemini provider for analysis
	geminiProvider, useBoundingBox, err := b.createGeminiProviderForAnalysis(params.Prompt)
	if err != nil {
		return interfaces.NewTextErrorResponse(fmt.Sprintf("Failed to create Gemini provider: %v", err))
	}

	// Create message with screenshot and prompt
	userMessage := message.Message{
		Role: message.User,
		Parts: []message.ContentPart{
			message.TextContent{Text: params.Prompt},
			message.BinaryContent{
				Data:     imageData,
				MIMEType: "image/png",
			},
		},
	}

	// Send to Gemini for analysis
	geminiStart := time.Now()
	response, err := geminiProvider.SendMessages(ctx, []message.Message{userMessage}, nil)
	geminiDuration := time.Since(geminiStart)
	if shouldEnableDebugVisualization() {
		fmt.Printf("[DEBUG] Gemini API call took: %v\n", geminiDuration)
	}
	if err != nil {
		return interfaces.NewTextErrorResponse(fmt.Sprintf("Gemini analysis failed: %v", err))
	}

	if response == nil || response.Content == "" {
		return interfaces.NewTextErrorResponse("No response received from Gemini")
	}

	// Build screenshot URLs array (only if save succeeded)
	var screenshotUrls []string
	if screenshotURL != "" {
		screenshotUrls = []string{screenshotURL}
	}

	// For bounding box responses, validate and format in read_page style
	if useBoundingBox {
		formattedResponse, err := formatBoundingBoxResponse(response.Content, imageWidth, imageHeight)
		if err != nil {
			return interfaces.NewTextErrorResponse(fmt.Sprintf("Invalid bounding box response: %v", err))
		}

		// Launch async visualization for debugging (fire-and-forget) - only in dev mode
		if shouldEnableDebugVisualization() {
			go visualizeBoundingBoxes(imageData, response.Content, imageWidth, imageHeight, params.Prompt)
		}

		return interfaces.ToolResponse{
			Type:           interfaces.ToolResponseTypeText,
			Content:        formattedResponse,
			ScreenshotUrls: screenshotUrls,
		}
	}

	return interfaces.ToolResponse{
		Type:           interfaces.ToolResponseTypeText,
		Content:        response.Content,
		ScreenshotUrls: screenshotUrls,
	}
}

// createGeminiProviderForAnalysis creates a Gemini provider for screenshot analysis
// Returns the provider and a boolean indicating if bounding box mode is enabled
func (b *browserTool) createGeminiProviderForAnalysis(prompt string) (interfaces.Provider, bool, error) {
	// Get API credentials
	credentialsService := config.GetAPICredentials()
	if credentialsService == nil {
		return nil, false, fmt.Errorf("API credentials service not available")
	}

	ctx := context.Background()
	apiKey, err := credentialsService.GetAPIKey(ctx, models.ProviderGemini)
	if err != nil {
		return nil, false, fmt.Errorf("failed to get Gemini API key: %w", err)
	}

	if apiKey == "" {
		return nil, false, fmt.Errorf("gemini API key not configured")
	}

	// Detect if this is a bounding box request
	promptLower := strings.ToLower(prompt)
	useBoundingBox := strings.Contains(promptLower, "bounding box") ||
		strings.Contains(promptLower, "coordinates") ||
		strings.Contains(promptLower, "box_2d")

	// Build provider options
	providerOpts := []provider.ProviderClientOption{
		provider.WithAPIKey(apiKey),
		provider.WithModel(models.GeminiModels[models.Gemini3Flash]),
		provider.WithMaxTokens(4096),
	}

	// Add JSON schema for bounding box requests
	if useBoundingBox {
		temp := float32(0.0)
		providerOpts = append(providerOpts,
			provider.WithTemperature(temp),
			provider.WithGeminiOptions(
				provider.WithGeminiResponseMIMEType("application/json"),
				provider.WithGeminiResponseJSONSchema(getBoundingBoxSchema()),
			),
		)
	}

	// Create Gemini provider
	geminiProvider, err := provider.NewProvider(models.ProviderGemini, providerOpts...)
	if err != nil {
		return nil, false, fmt.Errorf("failed to create Gemini provider: %w", err)
	}

	return geminiProvider, useBoundingBox, nil
}

// getBoundingBoxSchema returns the JSON schema for bounding box responses
func getBoundingBoxSchema() map[string]any {
	return map[string]any{
		"type": "array",
		"items": map[string]any{
			"type": "object",
			"properties": map[string]any{
				"box_2d": map[string]any{
					"type": "array",
					"items": map[string]any{
						"type":    "integer",
						"minimum": 0,
						"maximum": 1000,
					},
					"minItems":    4,
					"maxItems":    4,
					"description": "Bounding box coordinates [x1, y1, x2, y2] in normalized range [0, 1000]",
				},
			},
			"required": []string{"box_2d"},
		},
	}
}

// getImageDimensions decodes image data and returns its width and height
func getImageDimensions(imageData []byte) (width, height int, err error) {
	img, _, err := image.Decode(bytes.NewReader(imageData))
	if err != nil {
		return 0, 0, fmt.Errorf("failed to decode image: %w", err)
	}
	bounds := img.Bounds()
	return bounds.Dx(), bounds.Dy(), nil
}

// formatBoundingBoxResponse formats bounding box JSON into read_page style text
// Converts Gemini's normalized [0, 1000] coordinates to actual pixel coordinates
// Uses actual image dimensions (not viewport) since Gemini normalizes against the PNG image size
func formatBoundingBoxResponse(responseContent string, imageWidth, imageHeight int) (string, error) {
	var boundingBoxes []map[string]interface{}
	if err := json.Unmarshal([]byte(responseContent), &boundingBoxes); err != nil {
		return "", fmt.Errorf("invalid JSON structure: %w", err)
	}

	// Handle empty results
	if len(boundingBoxes) == 0 {
		return "Found 0 element(s).\n\nNo elements detected in screenshot.", nil
	}

	var result strings.Builder
	fmt.Fprintf(&result, "Found %d element(s):\n\n", len(boundingBoxes))

	// Calculate scaling factors from normalized [0, 1000] to actual image pixel space
	// CRITICAL: Use image dimensions, NOT viewport dimensions
	// (Image may be 2x or 3x viewport size due to device pixel ratio)
	scaleX := float64(imageWidth) / 1000.0
	scaleY := float64(imageHeight) / 1000.0

	for i, box := range boundingBoxes {
		// Extract box_2d coordinates
		box2d, ok := box["box_2d"]
		if !ok {
			return "", fmt.Errorf("missing box_2d key in element %d", i)
		}

		coords, ok := box2d.([]interface{})
		if !ok || len(coords) != 4 {
			return "", fmt.Errorf("box_2d must contain 4 coordinates in element %d", i)
		}

		// Parse normalized coordinates [0, 1000]
		x1Norm, ok1 := coords[0].(float64)
		y1Norm, ok2 := coords[1].(float64)
		x2Norm, ok3 := coords[2].(float64)
		y2Norm, ok4 := coords[3].(float64)

		if !ok1 || !ok2 || !ok3 || !ok4 {
			return "", fmt.Errorf("invalid coordinate values in element %d", i)
		}

		// Calculate center in normalized space
		centerXNorm := (x1Norm + x2Norm) / 2
		centerYNorm := (y1Norm + y2Norm) / 2

		// Convert from normalized [0, 1000] to pixel space
		centerXPixel := centerXNorm * scaleX
		centerYPixel := centerYNorm * scaleY

		// Get element type if available (optional field)
		elementType := "element"
		if name, ok := box["name"].(string); ok && name != "" {
			elementType = name
		}

		// Format in read_page style: - element_type (x=centerX,y=centerY)
		// Now using actual pixel coordinates
		fmt.Fprintf(&result, "- %s (x=%.0f,y=%.0f)\n", elementType, centerXPixel, centerYPixel)
	}

	return result.String(), nil
}

// visualizeBoundingBoxes draws bounding boxes and center points on the image and saves it to disk
// This function runs asynchronously for debugging purposes and fails silently on errors
func visualizeBoundingBoxes(imageData []byte, responseJSON string, imageWidth, imageHeight int, prompt string) {
	var err error

	// Parse bounding box JSON
	var boundingBoxes []map[string]interface{}
	if err = json.Unmarshal([]byte(responseJSON), &boundingBoxes); err != nil {
		return // Silent failure for debug feature
	}

	// Create timestamp-based subfolder first
	timestamp := time.Now().Format("20060102_150405")
	debugSubfolder := filepath.Join("debug_screenshots", timestamp)
	_ = os.MkdirAll(debugSubfolder, 0o755)

	// Save raw screenshot as PNG before any modifications
	rawFilename := filepath.Join(debugSubfolder, "screenshot_raw.png")
	rawFile, err := os.Create(rawFilename)
	if err != nil {
		return
	}
	_, _ = rawFile.Write(imageData)
	_ = rawFile.Close()

	// Decode the original image for overlay drawing
	var img image.Image
	img, err = png.Decode(bytes.NewReader(imageData))
	if err != nil {
		return
	}

	// Create a new RGBA image for drawing overlays
	bounds := img.Bounds()
	rgba := image.NewRGBA(bounds)
	draw.Draw(rgba, bounds, img, bounds.Min, draw.Src)

	// Calculate scaling factors from normalized [0, 1000] to actual pixel space
	scaleX := float64(imageWidth) / 1000.0
	scaleY := float64(imageHeight) / 1000.0

	// Colors for visualization
	red := color.RGBA{R: 255, G: 0, B: 0, A: 255}
	green := color.RGBA{R: 0, G: 255, B: 0, A: 255}

	// Draw each bounding box and center point
	for _, box := range boundingBoxes {
		box2d, ok := box["box_2d"].([]interface{})
		if !ok || len(box2d) != 4 {
			continue
		}

		// Parse normalized coordinates
		x1Norm, _ := box2d[0].(float64)
		y1Norm, _ := box2d[1].(float64)
		x2Norm, _ := box2d[2].(float64)
		y2Norm, _ := box2d[3].(float64)

		// Convert to pixel coordinates
		x1 := int(x1Norm * scaleX)
		y1 := int(y1Norm * scaleY)
		x2 := int(x2Norm * scaleX)
		y2 := int(y2Norm * scaleY)

		// Calculate center point
		centerX := (x1 + x2) / 2
		centerY := (y1 + y2) / 2

		// Draw bounding box rectangle (red, 2px thickness)
		drawRect(rgba, x1, y1, x2, y2, red, 2)

		// Draw center point (green circle, 5px radius)
		drawCircle(rgba, centerX, centerY, 5, green)
	}

	// Save overlayed screenshot as JPEG
	imageFilename := filepath.Join(debugSubfolder, "screenshot_overlayed.jpg")
	var outFile *os.File
	outFile, err = os.Create(imageFilename)
	if err != nil {
		return
	}
	defer func() { _ = outFile.Close() }()

	_ = jpeg.Encode(outFile, rgba, &jpeg.Options{Quality: 90})

	// Save prompt to prompt.txt
	promptFilename := filepath.Join(debugSubfolder, "prompt.txt")
	_ = os.WriteFile(promptFilename, []byte(prompt), 0o644)
}

// drawRect draws a rectangle outline with specified thickness
func drawRect(img *image.RGBA, x1, y1, x2, y2 int, col color.RGBA, thickness int) {
	// Draw horizontal lines (top and bottom)
	for t := 0; t < thickness; t++ {
		for x := x1; x <= x2; x++ {
			img.Set(x, y1+t, col) // Top
			img.Set(x, y2-t, col) // Bottom
		}
	}

	// Draw vertical lines (left and right)
	for t := 0; t < thickness; t++ {
		for y := y1; y <= y2; y++ {
			img.Set(x1+t, y, col) // Left
			img.Set(x2-t, y, col) // Right
		}
	}
}

// drawCircle draws a filled circle at the specified center point
func drawCircle(img *image.RGBA, cx, cy, radius int, col color.RGBA) {
	for y := cy - radius; y <= cy+radius; y++ {
		for x := cx - radius; x <= cx+radius; x++ {
			dx := x - cx
			dy := y - cy
			if dx*dx+dy*dy <= radius*radius {
				img.Set(x, y, col)
			}
		}
	}
}

// shouldEnableDebugVisualization checks if debug visualization should be enabled
// Returns true only when _DEV_DEBUG environment variable is set to "true"
func shouldEnableDebugVisualization() bool {
	return os.Getenv(constants.DevDebugEnv) == "true"
}
