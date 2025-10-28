package interfaces

import (
	"context"

	"mix/internal/llm/models"
	"mix/internal/message"
)

type EventType string

const (
	EventContentStart  EventType = "content_start"
	EventToolUseStart  EventType = "tool_use_start"
	EventToolUseDelta  EventType = "tool_use_delta"
	EventToolUseStop   EventType = "tool_use_stop"
	EventContentDelta  EventType = "content_delta"
	EventThinkingDelta EventType = "thinking_delta"
	EventContentStop   EventType = "content_stop"
	EventComplete      EventType = "complete"
	EventError         EventType = "error"
	EventWarning       EventType = "warning"
)

type TokenUsage struct {
	InputTokens         int64
	OutputTokens        int64
	CacheCreationTokens int64
	CacheReadTokens     int64
}

type ProviderResponse struct {
	Content                string
	ToolCalls              []message.ToolCall
	Usage                  TokenUsage
	FinishReason           message.FinishReason
	ThinkingBlocks         []message.ThinkingBlockContent
	RedactedThinkingBlocks []message.RedactedThinkingContent
}

type ProviderEvent struct {
	Type EventType

	Content  string
	Thinking string
	Response *ProviderResponse
	ToolCall *message.ToolCall
	Error    error
}

type Provider interface {
	SendMessages(ctx context.Context, messages []message.Message, tools []BaseTool) (*ProviderResponse, error)
	StreamResponse(ctx context.Context, messages []message.Message, tools []BaseTool) <-chan ProviderEvent
	Model() models.Model
}

type ProviderClient interface {
	Send(ctx context.Context, messages []message.Message, tools []BaseTool) (*ProviderResponse, error)
	Stream(ctx context.Context, messages []message.Message, tools []BaseTool) <-chan ProviderEvent
}
