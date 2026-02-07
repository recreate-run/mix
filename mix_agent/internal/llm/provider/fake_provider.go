package provider

import (
	"context"
	"fmt"
	"sync"
	"time"

	"mix/internal/llm/interfaces"
	"mix/internal/llm/models"
	"mix/internal/message"

	"github.com/google/uuid"
)

// FakeProvider implements the Provider interface with configurable canned responses
// for use in integration tests. It provides realistic streaming behavior without
// making actual LLM API calls.
type FakeProvider struct {
	model          models.Model
	responseConfig *FakeResponseConfig
}

// FakeResponseConfig holds configuration for fake responses
type FakeResponseConfig struct {
	Responses    []FakeResponse
	currentIndex int
	mu           sync.Mutex
}

// FakeResponse represents a single canned response
type FakeResponse struct {
	Content      string
	ToolCalls    []message.ToolCall
	FinishReason message.FinishReason
	Usage        interfaces.TokenUsage
	StreamDelay  time.Duration // Delay between events (for timeout testing)
}

// NewFakeProvider creates a new fake provider with the given model and response config
func NewFakeProvider(model models.Model, config *FakeResponseConfig) *FakeProvider {
	if config == nil {
		// Default config with a simple text response
		config = &FakeResponseConfig{
			Responses: []FakeResponse{{
				Content:      "This is a fake response for testing",
				FinishReason: message.FinishReasonEndTurn,
				Usage: interfaces.TokenUsage{
					InputTokens:  10,
					OutputTokens: 20,
				},
			}},
		}
	}
	return &FakeProvider{
		model:          model,
		responseConfig: config,
	}
}

// SendMessages returns the next canned response immediately
func (f *FakeProvider) SendMessages(ctx context.Context, messages []message.Message, tools []interfaces.BaseTool) (*interfaces.ProviderResponse, error) {
	// Check context cancellation
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	default:
	}

	response := f.getNextResponse()

	// Apply stream delay if configured (for timeout testing)
	if response.StreamDelay > 0 {
		select {
		case <-time.After(response.StreamDelay):
		case <-ctx.Done():
			return nil, ctx.Err()
		}
	}

	return &interfaces.ProviderResponse{
		Content:      response.Content,
		ToolCalls:    response.ToolCalls,
		Usage:        response.Usage,
		FinishReason: response.FinishReason,
	}, nil
}

// StreamResponse emits a realistic event sequence for the next canned response
func (f *FakeProvider) StreamResponse(ctx context.Context, messages []message.Message, tools []interfaces.BaseTool) <-chan interfaces.ProviderEvent {
	eventChan := make(chan interfaces.ProviderEvent)

	go func() {
		defer close(eventChan)

		// Check context cancellation
		select {
		case <-ctx.Done():
			eventChan <- interfaces.ProviderEvent{
				Type:  interfaces.EventError,
				Error: ctx.Err(),
			}
			return
		default:
		}

		response := f.getNextResponse()

		// Apply stream delay if configured
		if response.StreamDelay > 0 {
			select {
			case <-time.After(response.StreamDelay):
			case <-ctx.Done():
				eventChan <- interfaces.ProviderEvent{
					Type:  interfaces.EventError,
					Error: ctx.Err(),
				}
				return
			}
		}

		// Emit content events if there's text content
		if response.Content != "" {
			// Content start
			eventChan <- interfaces.ProviderEvent{
				Type: interfaces.EventContentStart,
			}

			// Check context
			select {
			case <-ctx.Done():
				eventChan <- interfaces.ProviderEvent{
					Type:  interfaces.EventError,
					Error: ctx.Err(),
				}
				return
			default:
			}

			// Content delta (emit in chunks for more realistic streaming)
			chunkSize := 10
			for i := 0; i < len(response.Content); i += chunkSize {
				end := i + chunkSize
				if end > len(response.Content) {
					end = len(response.Content)
				}
				chunk := response.Content[i:end]

				eventChan <- interfaces.ProviderEvent{
					Type:    interfaces.EventContentDelta,
					Content: chunk,
				}

				// Check context
				select {
				case <-ctx.Done():
					eventChan <- interfaces.ProviderEvent{
						Type:  interfaces.EventError,
						Error: ctx.Err(),
					}
					return
				default:
				}
			}

			// Content stop
			eventChan <- interfaces.ProviderEvent{
				Type: interfaces.EventContentStop,
			}
		}

		// Emit tool call events if there are tool calls
		for _, toolCall := range response.ToolCalls {
			// Generate a fake tool call ID if not provided
			if toolCall.ID == "" {
				toolCall.ID = fmt.Sprintf("toolu_fake_%s", uuid.New().String())
			}

			// Tool use start
			eventChan <- interfaces.ProviderEvent{
				Type: interfaces.EventToolUseStart,
				ToolCall: &message.ToolCall{
					ID:       toolCall.ID,
					Name:     toolCall.Name,
					Finished: false,
				},
			}

			// Check context
			select {
			case <-ctx.Done():
				eventChan <- interfaces.ProviderEvent{
					Type:  interfaces.EventError,
					Error: ctx.Err(),
				}
				return
			default:
			}

			// Tool use delta (emit input in chunks)
			chunkSize := 20
			for i := 0; i < len(toolCall.Input); i += chunkSize {
				end := i + chunkSize
				if end > len(toolCall.Input) {
					end = len(toolCall.Input)
				}
				chunk := toolCall.Input[i:end]

				eventChan <- interfaces.ProviderEvent{
					Type: interfaces.EventToolUseDelta,
					ToolCall: &message.ToolCall{
						ID:    toolCall.ID,
						Input: chunk,
					},
				}

				// Check context
				select {
				case <-ctx.Done():
					eventChan <- interfaces.ProviderEvent{
						Type:  interfaces.EventError,
						Error: ctx.Err(),
					}
					return
				default:
				}
			}

			// Tool use stop
			eventChan <- interfaces.ProviderEvent{
				Type: interfaces.EventToolUseStop,
				ToolCall: &message.ToolCall{
					ID:       toolCall.ID,
					Name:     toolCall.Name,
					Input:    toolCall.Input,
					Finished: true,
				},
			}
		}

		// Final complete event with full response
		eventChan <- interfaces.ProviderEvent{
			Type: interfaces.EventComplete,
			Response: &interfaces.ProviderResponse{
				Content:      response.Content,
				ToolCalls:    response.ToolCalls,
				Usage:        response.Usage,
				FinishReason: response.FinishReason,
			},
		}
	}()

	return eventChan
}

// Model returns the mock model this provider is configured for
func (f *FakeProvider) Model() models.Model {
	return f.model
}

// getNextResponse returns the next response in the sequence (thread-safe)
func (f *FakeProvider) getNextResponse() FakeResponse {
	f.responseConfig.mu.Lock()
	defer f.responseConfig.mu.Unlock()

	if len(f.responseConfig.Responses) == 0 {
		// Return default response if none configured
		return FakeResponse{
			Content:      "Default fake response",
			FinishReason: message.FinishReasonEndTurn,
			Usage: interfaces.TokenUsage{
				InputTokens:  10,
				OutputTokens: 20,
			},
		}
	}

	// Get current response
	response := f.responseConfig.Responses[f.responseConfig.currentIndex]

	// Advance to next response (cycle back to start if at end)
	f.responseConfig.currentIndex = (f.responseConfig.currentIndex + 1) % len(f.responseConfig.Responses)

	return response
}

// Helper functions for building common fake responses

// NewFakeTextResponse creates a simple text response
func NewFakeTextResponse(content string) *FakeResponseConfig {
	return &FakeResponseConfig{
		Responses: []FakeResponse{{
			Content:      content,
			FinishReason: message.FinishReasonEndTurn,
			Usage: interfaces.TokenUsage{
				InputTokens:  10,
				OutputTokens: 20,
			},
		}},
	}
}

// NewFakeToolCallResponse creates a response with a tool call
func NewFakeToolCallResponse(toolName, input string) *FakeResponseConfig {
	return &FakeResponseConfig{
		Responses: []FakeResponse{{
			ToolCalls: []message.ToolCall{{
				Name:  toolName,
				Input: input,
				Type:  "tool_use",
			}},
			FinishReason: message.FinishReasonToolUse,
			Usage: interfaces.TokenUsage{
				InputTokens:  10,
				OutputTokens: 15,
			},
		}},
	}
}

// NewFakeSequence creates a multi-response sequence
func NewFakeSequence(responses ...FakeResponse) *FakeResponseConfig {
	return &FakeResponseConfig{
		Responses: responses,
	}
}

// NewFakeErrorResponse creates an error response
func NewFakeErrorResponse(errorMsg string) *FakeResponseConfig {
	return &FakeResponseConfig{
		Responses: []FakeResponse{{
			Content:      errorMsg,
			FinishReason: message.FinishReasonError,
			Usage: interfaces.TokenUsage{
				InputTokens: 10,
			},
		}},
	}
}
