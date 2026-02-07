package provider

import (
	"context"
	"errors"
	"strings"
	"sync"
	"testing"
	"time"

	"mix/internal/llm/interfaces"
	"mix/internal/llm/models"
	"mix/internal/message"
)

func TestFakeProvider_SendMessages_SimpleText(t *testing.T) {
	t.Helper()
	config := NewFakeTextResponse("Hello, world!")
	mockModel := models.Model{ID: models.MockClaudeSonnet, Name: "Test Model"}
	provider := NewFakeProvider(mockModel, config)

	ctx := context.Background()
	response, err := provider.SendMessages(ctx, nil, nil)

	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	if response.Content != "Hello, world!" {
		t.Errorf("Expected content 'Hello, world!', got: %s", response.Content)
	}

	if response.FinishReason != message.FinishReasonEndTurn {
		t.Errorf("Expected finish reason end_turn, got: %s", response.FinishReason)
	}
}

func TestFakeProvider_SendMessages_ToolCall(t *testing.T) {
	t.Helper()
	config := NewFakeToolCallResponse("Bash", `{"command":"echo test"}`)
	mockModel := models.Model{ID: models.MockClaudeSonnet, Name: "Test Model"}
	provider := NewFakeProvider(mockModel, config)

	ctx := context.Background()
	response, err := provider.SendMessages(ctx, nil, nil)

	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	if len(response.ToolCalls) != 1 {
		t.Fatalf("Expected 1 tool call, got: %d", len(response.ToolCalls))
	}

	toolCall := response.ToolCalls[0]
	if toolCall.Name != "Bash" {
		t.Errorf("Expected tool name 'Bash', got: %s", toolCall.Name)
	}

	if !strings.Contains(toolCall.Input, "echo test") {
		t.Errorf("Expected tool input to contain 'echo test', got: %s", toolCall.Input)
	}

	if response.FinishReason != message.FinishReasonToolUse {
		t.Errorf("Expected finish reason tool_use, got: %s", response.FinishReason)
	}
}

func TestFakeProvider_SendMessages_ContextCancellation(t *testing.T) {
	t.Helper()
	config := NewFakeTextResponse("Should not complete")
	mockModel := models.Model{ID: models.MockClaudeSonnet, Name: "Test Model"}
	provider := NewFakeProvider(mockModel, config)

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately

	_, err := provider.SendMessages(ctx, nil, nil)

	if err == nil {
		t.Fatal("Expected context cancellation error, got nil")
	}

	if !errors.Is(err, context.Canceled) {
		t.Errorf("Expected context.Canceled error, got: %v", err)
	}
}

func TestFakeProvider_StreamResponse_TextContent(t *testing.T) {
	t.Helper()
	config := NewFakeTextResponse("Hello streaming world!")
	mockModel := models.Model{ID: models.MockClaudeSonnet, Name: "Test Model"}
	provider := NewFakeProvider(mockModel, config)

	ctx := context.Background()
	eventChan := provider.StreamResponse(ctx, nil, nil)

	events := make([]interfaces.ProviderEvent, 0, 10)
	for event := range eventChan {
		events = append(events, event)
	}

	// Verify event sequence
	if len(events) < 3 {
		t.Fatalf("Expected at least 3 events (start, delta, stop, complete), got %d", len(events))
	}

	// First event should be content start
	if events[0].Type != interfaces.EventContentStart {
		t.Errorf("Expected first event to be content_start, got: %s", events[0].Type)
	}

	// Middle events should be content deltas
	foundDelta := false
	for i := 1; i < len(events)-2; i++ {
		if events[i].Type == interfaces.EventContentDelta {
			foundDelta = true
			break
		}
	}
	if !foundDelta {
		t.Error("Expected at least one content_delta event")
	}

	// Second to last should be content stop
	if events[len(events)-2].Type != interfaces.EventContentStop {
		t.Errorf("Expected second to last event to be content_stop, got: %s", events[len(events)-2].Type)
	}

	// Last event should be complete
	lastEvent := events[len(events)-1]
	if lastEvent.Type != interfaces.EventComplete {
		t.Errorf("Expected last event to be complete, got: %s", lastEvent.Type)
	}

	if lastEvent.Response == nil {
		t.Fatal("Expected complete event to have response")
	}

	if lastEvent.Response.Content != "Hello streaming world!" {
		t.Errorf("Expected complete response content 'Hello streaming world!', got: %s", lastEvent.Response.Content)
	}
}

func TestFakeProvider_StreamResponse_ToolCall(t *testing.T) {
	t.Helper()
	config := NewFakeToolCallResponse("Read", `{"file_path":"/test/file.txt"}`)
	mockModel := models.Model{ID: models.MockClaudeSonnet, Name: "Test Model"}
	provider := NewFakeProvider(mockModel, config)

	ctx := context.Background()
	eventChan := provider.StreamResponse(ctx, nil, nil)

	events := make([]interfaces.ProviderEvent, 0, 10)
	for event := range eventChan {
		events = append(events, event)
	}

	// Verify tool call event sequence
	foundToolStart := false
	foundToolDelta := false
	foundToolStop := false

	for _, event := range events {
		switch event.Type {
		case interfaces.EventToolUseStart:
			foundToolStart = true
			if event.ToolCall == nil {
				t.Error("Expected tool_use_start to have ToolCall")
			}
		case interfaces.EventToolUseDelta:
			foundToolDelta = true
			if event.ToolCall == nil {
				t.Error("Expected tool_use_delta to have ToolCall")
			}
		case interfaces.EventToolUseStop:
			foundToolStop = true
			if event.ToolCall == nil {
				t.Error("Expected tool_use_stop to have ToolCall")
			}
		}
	}

	if !foundToolStart {
		t.Error("Expected tool_use_start event")
	}
	if !foundToolDelta {
		t.Error("Expected tool_use_delta event")
	}
	if !foundToolStop {
		t.Error("Expected tool_use_stop event")
	}

	// Verify complete event
	lastEvent := events[len(events)-1]
	if lastEvent.Type != interfaces.EventComplete {
		t.Errorf("Expected last event to be complete, got: %s", lastEvent.Type)
	}

	if len(lastEvent.Response.ToolCalls) != 1 {
		t.Fatalf("Expected 1 tool call in response, got: %d", len(lastEvent.Response.ToolCalls))
	}
}

func TestFakeProvider_StreamResponse_ContextCancellation(t *testing.T) {
	t.Helper()
	config := NewFakeTextResponse("Should not complete")
	mockModel := models.Model{ID: models.MockClaudeSonnet, Name: "Test Model"}
	provider := NewFakeProvider(mockModel, config)

	ctx, cancel := context.WithCancel(context.Background())
	eventChan := provider.StreamResponse(ctx, nil, nil)

	// Cancel after receiving first event
	<-eventChan
	cancel()

	// Check remaining events for error
	for event := range eventChan {
		// Should get an error event due to cancellation
		if event.Type == interfaces.EventError && event.Error != nil {
			return // Test passes
		}
	}

	t.Error("Expected error event due to context cancellation")
}

func TestFakeProvider_MultipleResponses_Sequential(t *testing.T) {
	t.Helper()
	config := NewFakeSequence(
		FakeResponse{
			Content:      "First response",
			FinishReason: message.FinishReasonEndTurn,
			Usage:        interfaces.TokenUsage{InputTokens: 10, OutputTokens: 20},
		},
		FakeResponse{
			Content:      "Second response",
			FinishReason: message.FinishReasonEndTurn,
			Usage:        interfaces.TokenUsage{InputTokens: 10, OutputTokens: 20},
		},
		FakeResponse{
			Content:      "Third response",
			FinishReason: message.FinishReasonEndTurn,
			Usage:        interfaces.TokenUsage{InputTokens: 10, OutputTokens: 20},
		},
	)
	mockModel := models.Model{ID: models.MockClaudeSonnet, Name: "Test Model"}
	provider := NewFakeProvider(mockModel, config)

	ctx := context.Background()

	// First call
	response1, err := provider.SendMessages(ctx, nil, nil)
	if err != nil {
		t.Fatalf("First call failed: %v", err)
	}
	if response1.Content != "First response" {
		t.Errorf("Expected 'First response', got: %s", response1.Content)
	}

	// Second call
	response2, err := provider.SendMessages(ctx, nil, nil)
	if err != nil {
		t.Fatalf("Second call failed: %v", err)
	}
	if response2.Content != "Second response" {
		t.Errorf("Expected 'Second response', got: %s", response2.Content)
	}

	// Third call
	response3, err := provider.SendMessages(ctx, nil, nil)
	if err != nil {
		t.Fatalf("Third call failed: %v", err)
	}
	if response3.Content != "Third response" {
		t.Errorf("Expected 'Third response', got: %s", response3.Content)
	}

	// Fourth call (should cycle back to first)
	response4, err := provider.SendMessages(ctx, nil, nil)
	if err != nil {
		t.Fatalf("Fourth call failed: %v", err)
	}
	if response4.Content != "First response" {
		t.Errorf("Expected to cycle back to 'First response', got: %s", response4.Content)
	}
}

func TestFakeProvider_ConcurrentAccess(t *testing.T) {
	t.Helper()
	config := NewFakeTextResponse("Concurrent response")
	mockModel := models.Model{ID: models.MockClaudeSonnet, Name: "Test Model"}
	provider := NewFakeProvider(mockModel, config)

	ctx := context.Background()
	var wg sync.WaitGroup
	const numGoroutines = 10

	// Launch multiple concurrent requests
	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_, err := provider.SendMessages(ctx, nil, nil)
			if err != nil {
				t.Errorf("Concurrent call failed: %v", err)
			}
		}()
	}

	wg.Wait()
}

func TestFakeProvider_Model(t *testing.T) {
	t.Helper()
	config := NewFakeTextResponse("Test")
	mockModel := models.Model{
		ID:   models.MockClaudeSonnet,
		Name: "Test Mock Model",
	}
	provider := NewFakeProvider(mockModel, config)

	returnedModel := provider.Model()

	if returnedModel.ID != mockModel.ID {
		t.Errorf("Expected model ID %s, got: %s", mockModel.ID, returnedModel.ID)
	}

	if returnedModel.Name != mockModel.Name {
		t.Errorf("Expected model name %s, got: %s", mockModel.Name, returnedModel.Name)
	}
}

func TestFakeProvider_EmptyConfig_UsesDefault(t *testing.T) {
	t.Helper()
	mockModel := models.Model{ID: models.MockClaudeSonnet, Name: "Test Model"}
	provider := NewFakeProvider(mockModel, nil) // nil config

	ctx := context.Background()
	response, err := provider.SendMessages(ctx, nil, nil)

	if err != nil {
		t.Fatalf("Expected no error with nil config, got: %v", err)
	}

	if response.Content == "" {
		t.Error("Expected default response content, got empty string")
	}

	if response.FinishReason != message.FinishReasonEndTurn {
		t.Errorf("Expected default finish reason end_turn, got: %s", response.FinishReason)
	}
}

func TestFakeProvider_StreamDelay(t *testing.T) {
	t.Helper()
	delay := 100 * time.Millisecond
	config := &FakeResponseConfig{
		Responses: []FakeResponse{{
			Content:      "Delayed response",
			FinishReason: message.FinishReasonEndTurn,
			StreamDelay:  delay,
			Usage:        interfaces.TokenUsage{InputTokens: 10, OutputTokens: 20},
		}},
	}
	mockModel := models.Model{ID: models.MockClaudeSonnet, Name: "Test Model"}
	provider := NewFakeProvider(mockModel, config)

	ctx := context.Background()
	start := time.Now()
	_, err := provider.SendMessages(ctx, nil, nil)
	elapsed := time.Since(start)

	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	if elapsed < delay {
		t.Errorf("Expected delay of at least %v, got: %v", delay, elapsed)
	}
}

func TestFakeProvider_ToolCallID_Generation(t *testing.T) {
	t.Helper()
	// Create tool call without ID
	config := &FakeResponseConfig{
		Responses: []FakeResponse{{
			ToolCalls: []message.ToolCall{{
				Name:  "TestTool",
				Input: `{"test":"data"}`,
			}},
			FinishReason: message.FinishReasonToolUse,
			Usage:        interfaces.TokenUsage{InputTokens: 10, OutputTokens: 15},
		}},
	}
	mockModel := models.Model{ID: models.MockClaudeSonnet, Name: "Test Model"}
	provider := NewFakeProvider(mockModel, config)

	ctx := context.Background()
	eventChan := provider.StreamResponse(ctx, nil, nil)

	var toolCallID string
	for event := range eventChan {
		if event.Type == interfaces.EventToolUseStart && event.ToolCall != nil {
			toolCallID = event.ToolCall.ID
			break
		}
	}

	if toolCallID == "" {
		t.Error("Expected tool call ID to be generated")
	}

	if !strings.HasPrefix(toolCallID, "toolu_fake_") {
		t.Errorf("Expected tool call ID to start with 'toolu_fake_', got: %s", toolCallID)
	}
}
