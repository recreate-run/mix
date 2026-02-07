//go:build integration
// +build integration

package provider

import (
	"context"
	"encoding/base64"
	"os"
	"testing"
	"time"

	"mix/internal/llm/interfaces"
	"mix/internal/llm/models"
	"mix/internal/message"
)

// Integration tests for Gemini provider with real API calls
// Run with: go test -tags=integration

func TestGeminiClient_RealAPI_ImageUnderstanding(t *testing.T) {
	apiKey := os.Getenv("GEMINI_API_KEY")
	if apiKey == "" {
		t.Skip("GEMINI_API_KEY not set, skipping integration test")
	}

	opts := providerClientOptions{
		apiKey:        apiKey,
		model:         models.SupportedModels[models.Gemini3Pro],
		maxTokens:     1000,
		systemMessage: "You are a helpful AI assistant that can analyze images.",
	}

	client := newGeminiClient(opts)
	if client == nil {
		t.Fatal("Failed to create Gemini client")
	}

	// Create a simple test image (red 2x2 PNG)
	testImageBase64 := "iVBORw0KGgoAAAANSUhEUgAAAAIAAAACCAYAAABytg0kAAAAFElEQVR42mNkYGBgZGBgYGQkGQAALAALAAE+fPkAAAAASUVORK5CYII="
	imageData, err := base64.StdEncoding.DecodeString(testImageBase64)
	if err != nil {
		t.Fatalf("Failed to decode test image: %v", err)
	}

	// Create message with image
	msg := message.Message{
		Role: message.User,
		Parts: []message.MessagePart{
			{Content: "What do you see in this image? Describe its colors and dimensions."},
			{BinaryContent: &message.BinaryContent{
				Data:     imageData,
				MIMEType: "image/png",
			}},
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

	t.Logf("Image analysis response: %s", response.Content)

	// Verify token usage is recorded
	if response.Usage.InputTokens == 0 {
		t.Error("Expected non-zero input tokens")
	}
	if response.Usage.OutputTokens == 0 {
		t.Error("Expected non-zero output tokens")
	}
}

func TestGeminiClient_RealAPI_StreamImageUnderstanding(t *testing.T) {
	apiKey := os.Getenv("GEMINI_API_KEY")
	if apiKey == "" {
		t.Skip("GEMINI_API_KEY not set, skipping integration test")
	}

	opts := providerClientOptions{
		apiKey:        apiKey,
		model:         models.SupportedModels[models.Gemini3Pro],
		maxTokens:     500,
		systemMessage: "You are a helpful AI assistant.",
	}

	client := newGeminiClient(opts)
	if client == nil {
		t.Fatal("Failed to create Gemini client")
	}

	// Create a simple test image
	testImageBase64 := "iVBORw0KGgoAAAANSUhEUgAAAAIAAAACCAYAAABytg0kAAAAFElEQVR42mNkYGBgZGBgYGQkGQAALAALAAE+fPkAAAAASUVORK5CYII="
	imageData, err := base64.StdEncoding.DecodeString(testImageBase64)
	if err != nil {
		t.Fatalf("Failed to decode test image: %v", err)
	}

	msg := message.Message{
		Role: message.User,
		Parts: []message.MessagePart{
			{Content: "Briefly describe this image in one sentence."},
			{BinaryContent: &message.BinaryContent{
				Data:     imageData,
				MIMEType: "image/png",
			}},
		},
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	eventChan := client.Stream(ctx, []message.Message{msg}, nil)

	var finalResponse *interfaces.ProviderResponse
	var contentReceived string
	eventCount := 0

	for event := range eventChan {
		eventCount++
		switch event.Type {
		case interfaces.EventContentDelta:
			contentReceived += event.Content
			t.Logf("Content delta: %q", event.Content)
		case interfaces.EventComplete:
			finalResponse = event.Response
			t.Logf("Stream completed")
		case interfaces.EventError:
			t.Fatalf("Stream error: %v", event.Error)
		}
	}

	if eventCount == 0 {
		t.Error("Expected to receive events from stream")
	}

	if finalResponse == nil {
		t.Error("Expected final response")
	} else {
		if finalResponse.Content == "" {
			t.Error("Expected non-empty final response content")
		}
		t.Logf("Final response: %s", finalResponse.Content)
	}

	if contentReceived == "" {
		t.Error("Expected to receive content deltas")
	}
}

func TestGeminiClient_RealAPI_MultipleImageFormats(t *testing.T) {
	apiKey := os.Getenv("GEMINI_API_KEY")
	if apiKey == "" {
		t.Skip("GEMINI_API_KEY not set, skipping integration test")
	}

	opts := providerClientOptions{
		apiKey:        apiKey,
		model:         models.SupportedModels[models.Gemini3Pro],
		maxTokens:     200,
		systemMessage: "You are a helpful AI assistant.",
	}

	client := newGeminiClient(opts)
	if client == nil {
		t.Fatal("Failed to create Gemini client")
	}

	// Test different image formats
	testCases := []struct {
		name     string
		mimeType string
		base64   string
	}{
		{
			name:     "PNG",
			mimeType: "image/png",
			base64:   "iVBORw0KGgoAAAANSUhEUgAAAAIAAAACCAYAAABytg0kAAAAFElEQVR42mNkYGBgZGBgYGQkGQAALAALAAE+fPkAAAAASUVORK5CYII=",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			imageData, err := base64.StdEncoding.DecodeString(tc.base64)
			if err != nil {
				t.Fatalf("Failed to decode %s image: %v", tc.name, err)
			}

			msg := message.Message{
				Role: message.User,
				Parts: []message.MessagePart{
					{Content: "What format is this image?"},
					{BinaryContent: &message.BinaryContent{
						Data:     imageData,
						MIMEType: tc.mimeType,
					}},
				},
			}

			ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
			defer cancel()

			response, err := client.Send(ctx, []message.Message{msg}, nil)
			if err != nil {
				t.Fatalf("Failed to send %s image: %v", tc.name, err)
			}

			if response.Content == "" {
				t.Errorf("Expected non-empty response for %s image", tc.name)
			}

			t.Logf("%s image response: %s", tc.name, response.Content)
		})
	}
}

func TestGeminiClient_RealAPI_ToolUsage(t *testing.T) {
	apiKey := os.Getenv("GEMINI_API_KEY")
	if apiKey == "" {
		t.Skip("GEMINI_API_KEY not set, skipping integration test")
	}

	opts := providerClientOptions{
		apiKey:        apiKey,
		model:         models.SupportedModels[models.Gemini3Pro],
		maxTokens:     1000,
		systemMessage: "You are a helpful AI assistant with access to tools.",
	}

	client := newGeminiClient(opts)
	if client == nil {
		t.Fatal("Failed to create Gemini client")
	}

	// Create a simple calculator tool
	calculatorTool := &mockTool{
		name:        "calculator",
		description: "Performs basic arithmetic calculations",
		parameters: map[string]interface{}{
			"expression": map[string]interface{}{
				"type":        "string",
				"description": "Mathematical expression to evaluate (e.g., '2+2')",
			},
		},
		required: []string{"expression"},
	}

	msg := message.Message{
		Role: message.User,
		Parts: []message.MessagePart{
			{Content: "What is 15 multiplied by 7? Use the calculator tool."},
		},
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	response, err := client.Send(ctx, []message.Message{msg}, []interfaces.BaseTool{calculatorTool})
	if err != nil {
		t.Fatalf("Failed to send message with tools: %v", err)
	}

	// We expect either a response with tool calls or regular text
	// Since this is a real API call, the model might choose not to use tools
	if response.Content == "" && len(response.ToolCalls) == 0 {
		t.Error("Expected either content or tool calls in response")
	}

	if len(response.ToolCalls) > 0 {
		t.Logf("Tool calls made: %d", len(response.ToolCalls))
		for _, toolCall := range response.ToolCalls {
			t.Logf("Tool call: %s with input: %s", toolCall.Name, toolCall.Input)
		}
	}

	if response.Content != "" {
		t.Logf("Response content: %s", response.Content)
	}
}

func TestGeminiClient_RealAPI_ErrorHandling(t *testing.T) {
	// Test with invalid API key
	opts := providerClientOptions{
		apiKey:        "invalid-key",
		model:         models.SupportedModels[models.Gemini3Pro],
		maxTokens:     100,
		systemMessage: "You are a helpful AI assistant.",
	}

	client := newGeminiClient(opts)
	if client == nil {
		t.Fatal("Failed to create Gemini client")
	}

	msg := message.Message{
		Role: message.User,
		Parts: []message.MessagePart{
			{Content: "Hello"},
		},
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	_, err := client.Send(ctx, []message.Message{msg}, nil)
	if err == nil {
		t.Error("Expected error with invalid API key")
	} else {
		t.Logf("Expected error received: %v", err)
	}
}
