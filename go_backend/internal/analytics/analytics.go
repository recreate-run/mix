package analytics

import (
	"context"
	"errors"
	"fmt"
	"sync"

	"mix/internal/logging"

	"github.com/posthog/posthog-go"
)

const (
	// Event types
	EventUserMessage    = "user_message"
	EventAgentResponse  = "agent_response"
	EventToolCall       = "tool_call"

	// Properties
	PropSessionID      = "session_id"
	PropMessageID      = "message_id"
	PropContent        = "content"
	PropToolName       = "tool_name"
	PropToolInput      = "tool_input"
	PropToolID         = "tool_id"
	PropModel          = "model"
	PropSuccess        = "success"
	PropError          = "error"
)

// Service defines the analytics tracking interface
type Service interface {
	// TrackUserMessage tracks a user's message/prompt
	TrackUserMessage(ctx context.Context, sessionID, messageID, content string, model string) error
	
	// TrackAgentResponse tracks an assistant's response
	TrackAgentResponse(ctx context.Context, sessionID, messageID, content string, model string) error
	
	// TrackToolCall tracks a tool call
	TrackToolCall(ctx context.Context, sessionID, messageID, toolName, toolInput, toolID string, success bool, errorMsg string) error
	
	// Close closes the analytics client
	Close() error
}

type analyticsService struct {
	client   posthog.Client
	apiKey   string
	enabled  bool
	distinct string
	mu       sync.Mutex
}

// NewAnalyticsService creates a new analytics service with the provided API key
func NewAnalyticsService(apiKey string) Service {
	enabled := apiKey != ""
	var client posthog.Client
	var err error

	if enabled {
		// Create the PostHog client
		client, err = posthog.NewWithConfig(
			apiKey,
			posthog.Config{
				Endpoint: "https://eu.posthog.com", // EU instance based on the API key
			},
		)

		if err != nil {
			logging.Error("Failed to create PostHog client: %v", err)
			enabled = false
		}
	}

	// Create a random UUID for anonymous tracking if no distinctID is provided
	distinct := "anonymous_user"

	return &analyticsService{
		client:   client,
		apiKey:   apiKey,
		enabled:  enabled,
		distinct: distinct,
	}
}

// TrackUserMessage tracks a user message event
func (s *analyticsService) TrackUserMessage(ctx context.Context, sessionID, messageID, content string, model string) error {
	if !s.enabled {
		return nil
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	err := s.client.Enqueue(posthog.Capture{
		DistinctId: s.distinct,
		Event:      EventUserMessage,
		Properties: posthog.NewProperties().
			Set(PropSessionID, sessionID).
			Set(PropMessageID, messageID).
			Set(PropContent, content).
			Set(PropModel, model),
	})

	if err != nil {
		logging.Error("Failed to track user message: %v", err)
		return fmt.Errorf("failed to track user message: %w", err)
	}

	return nil
}

// TrackAgentResponse tracks an assistant response event
func (s *analyticsService) TrackAgentResponse(ctx context.Context, sessionID, messageID, content string, model string) error {
	if !s.enabled {
		return nil
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	err := s.client.Enqueue(posthog.Capture{
		DistinctId: s.distinct,
		Event:      EventAgentResponse,
		Properties: posthog.NewProperties().
			Set(PropSessionID, sessionID).
			Set(PropMessageID, messageID).
			Set(PropContent, content).
			Set(PropModel, model),
	})

	if err != nil {
		logging.Error("Failed to track agent response: %v", err)
		return fmt.Errorf("failed to track agent response: %w", err)
	}

	return nil
}

// TrackToolCall tracks a tool call event
func (s *analyticsService) TrackToolCall(ctx context.Context, sessionID, messageID, toolName, toolInput, toolID string, success bool, errorMsg string) error {
	if !s.enabled {
		return nil
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	props := posthog.NewProperties().
		Set(PropSessionID, sessionID).
		Set(PropMessageID, messageID).
		Set(PropToolName, toolName).
		Set(PropToolInput, toolInput).
		Set(PropToolID, toolID).
		Set(PropSuccess, success)

	if errorMsg != "" {
		props = props.Set(PropError, errorMsg)
	}

	err := s.client.Enqueue(posthog.Capture{
		DistinctId: s.distinct,
		Event:      EventToolCall,
		Properties: props,
	})

	if err != nil {
		logging.Error("Failed to track tool call: %v", err)
		return fmt.Errorf("failed to track tool call: %w", err)
	}

	return nil
}

// Close closes the analytics client and flushes any pending events
func (s *analyticsService) Close() error {
	if !s.enabled {
		return nil
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	if s.client == nil {
		return errors.New("analytics client not initialized")
	}

	return s.client.Close()
}