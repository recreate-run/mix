package message

import (
	"context"
	"mix/internal/analytics"
	"mix/internal/logging"
)

// TrackingService wraps the message service with analytics tracking
type TrackingService struct {
	Service          // Embed the original message service
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