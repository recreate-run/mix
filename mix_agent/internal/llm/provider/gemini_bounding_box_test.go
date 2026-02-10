//go:build integration
// +build integration

package provider

import (
	"context"
	"encoding/json"
	"os"
	"testing"
	"time"

	"mix/internal/llm/models"
	"mix/internal/message"
)

// boundingBoxSchema defines the JSON schema for bounding box responses
// Coordinates are normalized to [0, 1000] range
var boundingBoxSchema = map[string]any{
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

// expectedTaxonomyButtonBox is the ground truth bounding box for the taxonomy button
// in the test image testdata/taxonomy_button.png
var expectedTaxonomyButtonBox = []map[string]interface{}{
	{"box_2d": []interface{}{float64(398), float64(37), float64(418), float64(104)}},
}

// createBoundingBoxClient creates a Gemini client configured for bounding box detection
func createBoundingBoxClient(t *testing.T, apiKey string) GeminiClient {
	t.Helper()

	temp := float32(0.0)
	opts := providerClientOptions{
		apiKey:        apiKey,
		model:         models.SupportedModels[models.Gemini3Flash],
		maxTokens:     1000,
		systemMessage: "You are a helpful AI assistant.",
		temperature:   &temp,
		geminiOptions: []GeminiOption{
			WithGeminiResponseMIMEType("application/json"),
			WithGeminiResponseJSONSchema(boundingBoxSchema),
		},
	}

	client := newGeminiClient(opts)
	if client == nil {
		t.Fatal("Failed to create Gemini client")
	}

	return client
}

// validateBoundingBoxResponse validates the structure and values of a bounding box response
func validateBoundingBoxResponse(t *testing.T, response string) []map[string]interface{} {
	t.Helper()

	// Parse the JSON response (should be clean JSON without markdown)
	var boundingBox []map[string]interface{}
	if err := json.Unmarshal([]byte(response), &boundingBox); err != nil {
		t.Fatalf("Failed to parse JSON response: %v\nResponse: %s", err, response)
	}

	// Validate structure
	if len(boundingBox) == 0 {
		t.Fatal("Response is empty array")
	}

	box2d, ok := boundingBox[0]["box_2d"]
	if !ok {
		t.Fatal("Missing box_2d key in response")
	}

	coords, ok := box2d.([]interface{})
	if !ok || len(coords) != 4 {
		t.Fatalf("box_2d does not contain 4 coordinates, got: %v", box2d)
	}

	t.Logf("✓ Response has correct JSON structure with box_2d coordinates")
	t.Logf("Coordinates: [%.0f, %.0f, %.0f, %.0f]", coords[0], coords[1], coords[2], coords[3])

	// Validate that all coordinates are in the normalized range [0, 1000]
	for i, coord := range coords {
		coordFloat, ok := coord.(float64)
		if !ok {
			t.Errorf("Coordinate %d is not a number: %v", i, coord)
			continue
		}
		if coordFloat < 0 || coordFloat > 1000 {
			t.Errorf("Coordinate %d (%.0f) is outside normalized range [0, 1000]", i, coordFloat)
		}
	}
	t.Logf("✓ All coordinates are within normalized range [0, 1000]")

	return boundingBox
}

func TestGeminiClient_RealAPI_BoundingBox(t *testing.T) {
	apiKey := os.Getenv("GEMINI_API_KEY")
	if apiKey == "" {
		t.Skip("GEMINI_API_KEY not set, skipping integration test")
	}

	client := createBoundingBoxClient(t, apiKey)

	// Load test image from testdata
	imageData, err := os.ReadFile("testdata/taxonomy_button.png")
	if err != nil {
		t.Fatalf("Failed to read test image: %v", err)
	}

	// Create message with image and prompt for bounding box
	msg := message.Message{
		Role: message.User,
		Parts: []message.ContentPart{
			message.TextContent{Text: "Give me the bounding box coordinates for the taxonomy button in the contents sidebar. Use normalized coordinates in the range [0, 1000]."},
			message.BinaryContent{
				Data:     imageData,
				MIMEType: "image/png",
			},
		},
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	response, err := client.Send(ctx, []message.Message{msg}, nil)
	if err != nil {
		t.Fatalf("Failed to send message: %v", err)
	}

	if response.Content == "" {
		t.Error("Expected non-empty response content")
	}

	t.Logf("Bounding box response: %s", response.Content)

	// Validate response structure
	actualBoundingBox := validateBoundingBoxResponse(t, response.Content)

	// Log expected vs actual
	expectedJSON, _ := json.Marshal(expectedTaxonomyButtonBox)
	actualJSON, _ := json.Marshal(actualBoundingBox)

	t.Logf("Expected: %s", expectedJSON)
	t.Logf("Actual: %s", actualJSON)

	// Verify token usage is recorded
	if response.Usage.InputTokens == 0 {
		t.Error("Expected non-zero input tokens")
	}
	if response.Usage.OutputTokens == 0 {
		t.Error("Expected non-zero output tokens")
	}
}
