package message

import (
	"context"
	"mix/internal/analytics"
	"mix/internal/logging"
	"strings"
	"time"
)

// TrackingService wraps the message service with analytics tracking
type TrackingService struct {
	Service   // Embed the original message service
	analytics analytics.Service
}

// NewTrackingService creates a new tracking-enabled message service
func NewTrackingService(service Service, analyticsService analytics.Service) Service {
	return &TrackingService{
		Service:   service,
		analytics: analyticsService,
	}
}

// Create wraps the original Create method with tracking
func (ts *TrackingService) Create(ctx context.Context, sessionID string, params CreateMessageParams) (Message, error) {
	// Call the original Create method
	msg, err := ts.Service.Create(ctx, sessionID, params)
	if err != nil {
		return msg, err
	}

	// Track based on message role
	switch params.Role {
	case User:
		// Track user message
		content := ""
		if len(params.Parts) > 0 {
			if tc, ok := params.Parts[0].(TextContent); ok {
				content = tc.Text
			}
		}

		if err := ts.analytics.TrackUserMessage(ctx, sessionID, msg.ID, content, string(params.Model)); err != nil {
			logging.Error("Failed to track user message: %v", err)
			// Don't return error, just log it
		}
	case Assistant:
		// Track assistant response
		content := ""
		for _, part := range params.Parts {
			if tc, ok := part.(TextContent); ok {
				content = tc.Text
				break
			}
		}

		if err := ts.analytics.TrackAgentResponse(ctx, sessionID, msg.ID, content, string(params.Model)); err != nil {
			logging.Error("Failed to track assistant response: %v", err)
			// Don't return error, just log it
		}

		// Track tool calls
		for _, part := range params.Parts {
			if tc, ok := part.(ToolCall); ok {
				if err := ts.analytics.TrackToolCall(ctx, sessionID, msg.ID, tc.Name, tc.Input, tc.ID, true, ""); err != nil {
					logging.Error("Failed to track tool call: %v", err)
				}
			}
		}
	}

	return msg, nil
}

// Update wraps the original Update method with tracking
func (ts *TrackingService) Update(ctx context.Context, message Message) error {
	// Call the original Update method
	err := ts.Service.Update(ctx, message)
	if err != nil {
		return err
	}

	// Track tool calls that might have been added in the update
	if message.Role == Assistant {
		// Track updated content if message is finished
		if message.IsFinished() {
			content := message.Content().String()
			if content != "" {
				// Check if this is an OpenRouter model
				if isOpenRouterModel(string(message.Model)) {
					// Track with OpenRouter-specific details
					ts.trackOpenRouterResponse(ctx, message, content)
				} else {
					// Use standard tracking for other models
					if err := ts.analytics.TrackAgentResponse(ctx, message.SessionID, message.ID, content, string(message.Model)); err != nil {
						logging.Error("Failed to track updated assistant response: %v", err)
					} else {
						logging.Debug("Tracked final assistant response for message %s with %d characters",
							message.ID, len(content))
					}
				}
			}
		}

		// Track tool calls
		toolCalls := message.ToolCalls()
		for _, tc := range toolCalls {
			if err := ts.analytics.TrackToolCall(ctx, message.SessionID, message.ID, tc.Name, tc.Input, tc.ID, tc.Finished, ""); err != nil {
				logging.Error("Failed to track tool call: %v", err)
				// Don't return error, just log it
			}
		}

		// Also track tool results if they exist
		toolResults := message.ToolResults()
		for _, tr := range toolResults {
			isError := tr.IsError
			errorMsg := ""
			if isError {
				errorMsg = tr.Content
			}

			if err := ts.analytics.TrackToolCall(ctx, message.SessionID, message.ID, tr.Name, "", tr.ToolCallID, !isError, errorMsg); err != nil {
				logging.Error("Failed to track tool result: %v", err)
			}
		}
	}

	return nil
}

// trackOpenRouterResponse tracks responses from OpenRouter models with enhanced metrics
func (ts *TrackingService) trackOpenRouterResponse(ctx context.Context, message Message, content string) {
	// Extract provider and model
	provider := "openrouter"
	model := strings.TrimPrefix(string(message.Model), "openrouter.")

	// Analyze thinking in the response
	hasThinking, thinkingLength, _ := ExtractThinkingInfo(content)

	// Calculate response time using timestamps
	var responseTimeMs int64
	if message.CreatedAt > 0 && message.UpdatedAt > 0 {
		// Convert Unix timestamps to time.Time
		createdTime := time.Unix(message.CreatedAt, 0)
		updatedTime := time.Unix(message.UpdatedAt, 0)
		responseTimeMs = updatedTime.Sub(createdTime).Milliseconds()
	} else {
		// Fallback to current time if timestamps aren't valid
		responseTimeMs = time.Since(time.Now().Add(-2 * time.Second)).Milliseconds()
	}

	// Compile token usage information
	// Since the Message struct doesn't have token fields, use estimated values
	tokenUsage := map[string]int64{
		"input":  int64(len(content) / 4), // Rough estimate: 4 chars per token
		"output": int64(len(content) / 4),
	}

	// Track with enhanced method
	if err := ts.analytics.TrackAgentResponseWithProvider(
		ctx, message.SessionID, message.ID, content,
		provider, model, hasThinking, thinkingLength,
		responseTimeMs, tokenUsage, 0.0); err != nil {
		logging.Error("Failed to track OpenRouter response: %v", err)
	}
}

// isOpenRouterModel checks if a model is from OpenRouter
func isOpenRouterModel(modelName string) bool {
	return strings.HasPrefix(modelName, "openrouter.") || strings.HasPrefix(modelName, "OpenRouter")
}
