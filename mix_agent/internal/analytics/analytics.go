package analytics

import (
	"context"
	"errors"
	"fmt"
	"os"
	"sync"

	"mix/internal/logging"

	"github.com/posthog/posthog-go"
)

const (
	// Event types
	EventUserMessage    = "user_message"
	EventAgentResponse  = "agent_response"
	EventToolCall       = "tool_call"
	EventProviderAuth   = "provider_auth"

	// Session lifecycle events
	EventSessionCreated = "session_created"
	EventSessionDeleted = "session_deleted"
	EventSessionRewound = "session_rewound"

	// File operation events
	EventFileUploaded = "file_uploaded"
	EventFileDeleted  = "file_deleted"

	// Export events
	EventSessionExported = "session_exported"
	EventVideoExported   = "video_exported"

	// Preferences events
	EventPreferencesUpdated = "preferences_updated"
	EventPreferencesReset   = "preferences_reset"

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

	// Provider-specific properties
	PropProvider           = "provider"
	PropProviderModel      = "provider_model"
	PropThinkingEnabled    = "thinking_enabled"
	PropThinkingLength     = "thinking_length"
	PropResponseTime       = "response_time_ms"
	PropTokenUsageInput    = "token_usage_input"
	PropTokenUsageOutput   = "token_usage_output"
	PropTokenUsageCached   = "token_usage_cached"
	PropCost               = "cost"
	PropAuthMethod         = "auth_method"

	// Session-specific properties
	PropTitle              = "title"
	PropHasCustomPrompt    = "has_custom_prompt"
	PropPromptMode         = "prompt_mode"
	PropCustomPromptLength = "custom_prompt_length"
	PropSessionAgeSeconds  = "session_age_seconds"
	PropMessageCount       = "message_count"
	PropRewindToMessageID  = "rewind_to_message_id"
	PropMessagesDeleted    = "messages_deleted_count"
	PropCleanupMedia       = "cleanup_media"

	// File-specific properties
	PropFileSizeBytes     = "file_size_bytes"
	PropFileType          = "file_type"
	PropFileNameSanitized = "file_name_sanitized"
	PropIsMedia           = "is_media"
	PropFileName          = "file_name"
	PropFileExisted       = "file_existed"

	// Export-specific properties
	PropExportFormat      = "export_format"
	PropTotalTokens       = "total_tokens"
	PropURL               = "url"
	PropFPS               = "fps"
	PropAspectRatio       = "aspect_ratio"
	PropHeight            = "height"
	PropDuration          = "duration"
	PropUploadedToS3      = "uploaded_to_s3"
	PropExportDurationMs  = "export_duration_ms"

	// Preferences-specific properties
	PropFieldsChanged              = "fields_changed"
	PropPreferredProvider          = "preferred_provider"
	PropMainAgentModel             = "main_agent_model"
	PropMainAgentMaxTokens         = "main_agent_max_tokens"
	PropMainAgentReasoningEffort   = "main_agent_reasoning_effort"
	PropSubAgentModel              = "sub_agent_model"
	PropSubAgentMaxTokens          = "sub_agent_max_tokens"
	PropSubAgentReasoningEffort    = "sub_agent_reasoning_effort"
	PropPreviousProvider           = "previous_provider"
	PropPreviousModel              = "previous_model"
)

// Service defines the analytics tracking interface
type Service interface {
	// TrackUserMessage tracks a user's message/prompt
	TrackUserMessage(ctx context.Context, sessionID, messageID, content string, model string) error
	
	// TrackAgentResponse tracks an assistant's response
	TrackAgentResponse(ctx context.Context, sessionID, messageID, content string, model string) error
	
	// TrackToolCall tracks a tool call
	TrackToolCall(ctx context.Context, sessionID, messageID, toolName, toolInput, toolID string, success bool, errorMsg string) error
	
	// TrackAgentResponseWithProvider tracks an assistant's response with detailed provider information
	TrackAgentResponseWithProvider(ctx context.Context, sessionID, messageID, content string, 
		provider string, model string, thinkingEnabled bool, thinkingLength int, 
		responseTimeMs int64, tokenUsage map[string]int64, cost float64) error
	
	// TrackProviderAuth tracks authentication events for providers
	TrackProviderAuth(ctx context.Context, provider string, success bool, authMethod string) error

	// TrackSessionCreated tracks session creation events
	TrackSessionCreated(ctx context.Context, sessionID, title string, hasCustomPrompt bool, promptMode string, customPromptLength int) error

	// TrackSessionDeleted tracks session deletion events
	TrackSessionDeleted(ctx context.Context, sessionID string, ageSeconds int64, messageCount int, cost float64) error

	// TrackSessionRewound tracks session rewind events
	TrackSessionRewound(ctx context.Context, sessionID, messageID string, messagesDeleted int, cleanupMedia bool) error

	// TrackFileUploaded tracks file upload events
	TrackFileUploaded(ctx context.Context, sessionID string, fileSizeBytes int64, fileType string, fileNameSanitized bool, isMedia bool) error

	// TrackFileDeleted tracks file deletion events
	TrackFileDeleted(ctx context.Context, sessionID, fileName string, fileExisted bool) error

	// TrackSessionExported tracks session export events
	TrackSessionExported(ctx context.Context, sessionID string, messageCount int, cost float64, totalTokens int64) error

	// TrackVideoExported tracks video export events
	TrackVideoExported(ctx context.Context, url string, fps int, aspectRatio string, height int, duration float64, uploadedToS3 bool, exportDurationMs int64) error

	// TrackPreferencesUpdated tracks user preferences update events
	TrackPreferencesUpdated(ctx context.Context, fieldsChanged []string, updates map[string]interface{}) error

	// TrackPreferencesReset tracks user preferences reset events
	TrackPreferencesReset(ctx context.Context, previousProvider, previousModel string) error

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

	// Check for USER_NAME environment variable, fall back to anonymous_user
	distinct := os.Getenv("POSTHON_USER_NAME")
	if distinct == "" {
		distinct = "anonymous_user"
	}

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

	// Don't track empty content
	if content == "" {
		return nil
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	// For longer content, truncate to a reasonable size for analytics
	// but preserve enough for meaningful analysis
	trackedContent := content
	if len(trackedContent) > 10000 {
		trackedContent = trackedContent[:10000] + "... [truncated]"
	}

	props := posthog.NewProperties().
		Set(PropSessionID, sessionID).
		Set(PropMessageID, messageID).
		Set(PropContent, trackedContent).
		Set(PropModel, model).
		Set("content_length", len(content)).
		Set("is_truncated", len(trackedContent) < len(content))

	err := s.client.Enqueue(posthog.Capture{
		DistinctId: s.distinct,
		Event:      EventUserMessage,
		Properties: props,
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

	// Don't track empty content
	if content == "" {
		return nil
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	// For longer content, truncate to a reasonable size for analytics
	// but preserve enough for meaningful analysis
	trackedContent := content
	if len(trackedContent) > 10000 {
		trackedContent = trackedContent[:10000] + "... [truncated]"
	}

	props := posthog.NewProperties().
		Set(PropSessionID, sessionID).
		Set(PropMessageID, messageID).
		Set(PropContent, trackedContent).
		Set(PropModel, model).
		Set("content_length", len(content)).
		Set("is_truncated", len(trackedContent) < len(content))

	err := s.client.Enqueue(posthog.Capture{
		DistinctId: s.distinct,
		Event:      EventAgentResponse,
		Properties: props,
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

// TrackAgentResponseWithProvider tracks an assistant's response with detailed provider information
func (s *analyticsService) TrackAgentResponseWithProvider(ctx context.Context, 
	sessionID, messageID, content string, provider string, model string,
	thinkingEnabled bool, thinkingLength int, responseTimeMs int64,
	tokenUsage map[string]int64, cost float64) error {
	
	if !s.enabled {
		return nil
	}
	
	// Don't track empty content
	if content == "" {
		return nil
	}
	
	s.mu.Lock()
	defer s.mu.Unlock()
	
	// Truncate long content
	trackedContent := content
	if len(trackedContent) > 10000 {
		trackedContent = trackedContent[:10000] + "... [truncated]"
	}
	
	props := posthog.NewProperties().
		Set(PropSessionID, sessionID).
		Set(PropMessageID, messageID).
		Set(PropContent, trackedContent).
		Set(PropModel, model).
		Set(PropProvider, provider).
		Set(PropProviderModel, model).
		Set(PropThinkingEnabled, thinkingEnabled).
		Set(PropThinkingLength, thinkingLength).
		Set(PropResponseTime, responseTimeMs).
		Set("content_length", len(content)).
		Set("is_truncated", len(trackedContent) < len(content))
	
	// Add token usage if available
	if tokenUsage != nil {
		if input, ok := tokenUsage["input"]; ok {
			props = props.Set(PropTokenUsageInput, input)
		}
		if output, ok := tokenUsage["output"]; ok {
			props = props.Set(PropTokenUsageOutput, output)
		}
		if cached, ok := tokenUsage["cached"]; ok {
			props = props.Set(PropTokenUsageCached, cached)
		}
	}
	
	// Add cost if available
	if cost > 0 {
		props = props.Set(PropCost, cost)
	}
	
	err := s.client.Enqueue(posthog.Capture{
		DistinctId: s.distinct,
		Event:      EventAgentResponse,
		Properties: props,
	})
	
	if err != nil {
		logging.Error("Failed to track agent response with provider details: %v", err)
		return fmt.Errorf("failed to track agent response with provider details: %w", err)
	}
	
	return nil
}

// TrackProviderAuth tracks authentication events for providers
func (s *analyticsService) TrackProviderAuth(ctx context.Context, 
	provider string, success bool, authMethod string) error {
	
	if !s.enabled {
		return nil
	}
	
	s.mu.Lock()
	defer s.mu.Unlock()
	
	props := posthog.NewProperties().
		Set(PropProvider, provider).
		Set(PropSuccess, success).
		Set(PropAuthMethod, authMethod)
	
	err := s.client.Enqueue(posthog.Capture{
		DistinctId: s.distinct,
		Event:      EventProviderAuth,
		Properties: props,
	})
	
	if err != nil {
		logging.Error("Failed to track provider auth: %v", err)
		return fmt.Errorf("failed to track provider auth: %w", err)
	}
	
	return nil
}

// TrackSessionCreated tracks session creation events
func (s *analyticsService) TrackSessionCreated(ctx context.Context, sessionID, title string, hasCustomPrompt bool, promptMode string, customPromptLength int) error {
	if !s.enabled {
		return nil
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	props := posthog.NewProperties().
		Set(PropSessionID, sessionID).
		Set(PropTitle, title).
		Set(PropHasCustomPrompt, hasCustomPrompt).
		Set(PropPromptMode, promptMode)

	if hasCustomPrompt {
		props = props.Set(PropCustomPromptLength, customPromptLength)
	}

	err := s.client.Enqueue(posthog.Capture{
		DistinctId: s.distinct,
		Event:      EventSessionCreated,
		Properties: props,
	})

	if err != nil {
		logging.Error("Failed to track session created: %v", err)
		return fmt.Errorf("failed to track session created: %w", err)
	}

	return nil
}

// TrackSessionDeleted tracks session deletion events
func (s *analyticsService) TrackSessionDeleted(ctx context.Context, sessionID string, ageSeconds int64, messageCount int, cost float64) error {
	if !s.enabled {
		return nil
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	props := posthog.NewProperties().
		Set(PropSessionID, sessionID).
		Set(PropSessionAgeSeconds, ageSeconds).
		Set(PropMessageCount, messageCount).
		Set(PropCost, cost)

	err := s.client.Enqueue(posthog.Capture{
		DistinctId: s.distinct,
		Event:      EventSessionDeleted,
		Properties: props,
	})

	if err != nil {
		logging.Error("Failed to track session deleted: %v", err)
		return fmt.Errorf("failed to track session deleted: %w", err)
	}

	return nil
}

// TrackSessionRewound tracks session rewind events
func (s *analyticsService) TrackSessionRewound(ctx context.Context, sessionID, messageID string, messagesDeleted int, cleanupMedia bool) error {
	if !s.enabled {
		return nil
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	props := posthog.NewProperties().
		Set(PropSessionID, sessionID).
		Set(PropRewindToMessageID, messageID).
		Set(PropMessagesDeleted, messagesDeleted).
		Set(PropCleanupMedia, cleanupMedia)

	err := s.client.Enqueue(posthog.Capture{
		DistinctId: s.distinct,
		Event:      EventSessionRewound,
		Properties: props,
	})

	if err != nil {
		logging.Error("Failed to track session rewound: %v", err)
		return fmt.Errorf("failed to track session rewound: %w", err)
	}

	return nil
}

// TrackFileUploaded tracks file upload events
func (s *analyticsService) TrackFileUploaded(ctx context.Context, sessionID string, fileSizeBytes int64, fileType string, fileNameSanitized bool, isMedia bool) error {
	if !s.enabled {
		return nil
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	props := posthog.NewProperties().
		Set(PropSessionID, sessionID).
		Set(PropFileSizeBytes, fileSizeBytes).
		Set(PropFileType, fileType).
		Set(PropFileNameSanitized, fileNameSanitized).
		Set(PropIsMedia, isMedia)

	err := s.client.Enqueue(posthog.Capture{
		DistinctId: s.distinct,
		Event:      EventFileUploaded,
		Properties: props,
	})

	if err != nil {
		logging.Error("Failed to track file uploaded: %v", err)
		return fmt.Errorf("failed to track file uploaded: %w", err)
	}

	return nil
}

// TrackFileDeleted tracks file deletion events
func (s *analyticsService) TrackFileDeleted(ctx context.Context, sessionID, fileName string, fileExisted bool) error {
	if !s.enabled {
		return nil
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	props := posthog.NewProperties().
		Set(PropSessionID, sessionID).
		Set(PropFileName, fileName).
		Set(PropFileExisted, fileExisted)

	err := s.client.Enqueue(posthog.Capture{
		DistinctId: s.distinct,
		Event:      EventFileDeleted,
		Properties: props,
	})

	if err != nil {
		logging.Error("Failed to track file deleted: %v", err)
		return fmt.Errorf("failed to track file deleted: %w", err)
	}

	return nil
}

// TrackSessionExported tracks session export events
func (s *analyticsService) TrackSessionExported(ctx context.Context, sessionID string, messageCount int, cost float64, totalTokens int64) error {
	if !s.enabled {
		return nil
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	props := posthog.NewProperties().
		Set(PropSessionID, sessionID).
		Set(PropMessageCount, messageCount).
		Set(PropCost, cost).
		Set(PropTotalTokens, totalTokens).
		Set(PropExportFormat, "json")

	err := s.client.Enqueue(posthog.Capture{
		DistinctId: s.distinct,
		Event:      EventSessionExported,
		Properties: props,
	})

	if err != nil {
		logging.Error("Failed to track session exported: %v", err)
		return fmt.Errorf("failed to track session exported: %w", err)
	}

	return nil
}

// TrackVideoExported tracks video export events
func (s *analyticsService) TrackVideoExported(ctx context.Context, url string, fps int, aspectRatio string, height int, duration float64, uploadedToS3 bool, exportDurationMs int64) error {
	if !s.enabled {
		return nil
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	props := posthog.NewProperties().
		Set(PropURL, url).
		Set(PropFPS, fps).
		Set(PropAspectRatio, aspectRatio).
		Set(PropHeight, height).
		Set(PropDuration, duration).
		Set(PropUploadedToS3, uploadedToS3).
		Set(PropExportDurationMs, exportDurationMs)

	err := s.client.Enqueue(posthog.Capture{
		DistinctId: s.distinct,
		Event:      EventVideoExported,
		Properties: props,
	})

	if err != nil {
		logging.Error("Failed to track video exported: %v", err)
		return fmt.Errorf("failed to track video exported: %w", err)
	}

	return nil
}

// TrackPreferencesUpdated tracks user preferences update events
func (s *analyticsService) TrackPreferencesUpdated(ctx context.Context, fieldsChanged []string, updates map[string]interface{}) error {
	if !s.enabled {
		return nil
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	props := posthog.NewProperties().
		Set(PropFieldsChanged, fieldsChanged)

	// Add each update field to properties
	for key, value := range updates {
		props = props.Set(key, value)
	}

	err := s.client.Enqueue(posthog.Capture{
		DistinctId: s.distinct,
		Event:      EventPreferencesUpdated,
		Properties: props,
	})

	if err != nil {
		logging.Error("Failed to track preferences updated: %v", err)
		return fmt.Errorf("failed to track preferences updated: %w", err)
	}

	return nil
}

// TrackPreferencesReset tracks user preferences reset events
func (s *analyticsService) TrackPreferencesReset(ctx context.Context, previousProvider, previousModel string) error {
	if !s.enabled {
		return nil
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	props := posthog.NewProperties().
		Set(PropPreviousProvider, previousProvider).
		Set(PropPreviousModel, previousModel)

	err := s.client.Enqueue(posthog.Capture{
		DistinctId: s.distinct,
		Event:      EventPreferencesReset,
		Properties: props,
	})

	if err != nil {
		logging.Error("Failed to track preferences reset: %v", err)
		return fmt.Errorf("failed to track preferences reset: %w", err)
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