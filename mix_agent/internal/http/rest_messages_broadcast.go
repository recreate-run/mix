package http

import (
	"fmt"
	"strings"

	"mix/internal/llm/agent"
)

// BroadcastAgentEventToSSE converts an AgentEvent and broadcasts it to all SSE connections for the session
func BroadcastAgentEventToSSE(sessionID string, event agent.AgentEvent) {
	// Determine routing target from event
	targetSessionID := event.RouteTo
	if targetSessionID == "" {
		targetSessionID = sessionID // Fallback: use function parameter
	}
	parentToolCallID := event.ParentToolCallID

	switch event.Type {
	case agent.AgentEventTypeThinking:
		// Send thinking delta event
		registry.BroadcastEvent(targetSessionID, "thinking", ThinkingEvent{
			Type:             "thinking",
			Content:          event.Thinking,
			ParentToolCallID: parentToolCallID,
		})

	case agent.AgentEventTypeContentDelta:
		// Stream content deltas for text between tool calls
		if event.Content != "" {
			contentEvent := ContentEvent{
				Type:             "content",
				Content:          event.Content,
				ParentToolCallID: parentToolCallID,
			}
			registry.BroadcastEvent(targetSessionID, "content", contentEvent)
		}

	case agent.AgentEventTypeToolParameterDelta:
		// Stream tool parameter deltas for real-time parameter visibility
		registry.BroadcastEvent(targetSessionID, "tool_parameter_delta", ToolParameterDeltaEvent{
			Type:             "tool_parameter_delta",
			ToolCallID:       event.ToolCallID,
			Input:            event.Content, // Delta is stored in Content field
			ParentToolCallID: parentToolCallID,
		})

	case agent.AgentEventTypeResponse:
		// Stream tool calls - detect new tool calls by checking completion status
		toolCalls := event.Message.ToolCalls()
		for _, toolCall := range toolCalls {
			// Determine tool status - tools start with complete parameters
			status := "running"
			if toolCall.Finished {
				status = "completed"
			}

			registry.BroadcastEvent(targetSessionID, "tool", ToolEvent{
				Type:             "tool",
				Name:             toolCall.Name,
				Input:            toolCall.Input,
				ID:               toolCall.ID,
				Status:           status,
				ParentToolCallID: parentToolCallID,
			})
		}

		// Send completion event only for final events, include final content
		if event.Done {
			// Check if this is a permission denied error
			if event.Message.FinishReason() == "permission_denied" {
				registry.BroadcastEvent(targetSessionID, "error", ErrorEvent{
					Error:            "Permission denied",
					ParentToolCallID: parentToolCallID,
				})
			} else {
				content := event.Message.Content().String()
				reasoningContent := event.Message.ReasoningContent()
				reasoning := reasoningContent.String()
				reasoningDuration := reasoningContent.Duration
				registry.BroadcastEvent(targetSessionID, "complete", CompleteEvent{
					Type:              "complete",
					Content:           content,
					MessageID:         event.Message.ID,
					Done:              true,
					Reasoning:         reasoning,
					ReasoningDuration: reasoningDuration,
					ParentToolCallID:  parentToolCallID,
				})
			}
		}

	case agent.AgentEventTypeError:
		errMsg := event.Error.Error()

		// Special handling for rate limit errors
		if strings.Contains(errMsg, "rate_limit_error") {
			// Extract retry information if available
			retryAfter := 60 // Default retry after 60 seconds
			attempt := 1
			maxAttempts := 8

			// Try to extract retry info from error message
			if strings.Contains(errMsg, "Retrying due to rate limit") {
				var currentAttempt, totalAttempts int
				_, err := fmt.Sscanf(errMsg, "Retrying due to rate limit... attempt %d of %d", &currentAttempt, &totalAttempts)
				if err == nil && currentAttempt > 0 && totalAttempts > 0 {
					attempt = currentAttempt
					maxAttempts = totalAttempts
				}
			}

			errorEvent := ErrorEvent{
				Error:            "This request would exceed your account's rate limit. The application will automatically retry.",
				Type:             "rate_limit_error",
				RetryAfter:       retryAfter,
				Attempt:          attempt,
				MaxAttempts:      maxAttempts,
				ParentToolCallID: parentToolCallID,
			}

			registry.BroadcastEvent(targetSessionID, "rate_limit_error", errorEvent)
		} else {
			registry.BroadcastEvent(targetSessionID, "error", ErrorEvent{
				Error:            event.Error.Error(),
				ParentToolCallID: parentToolCallID,
			})
		}

	case agent.AgentEventTypeSummarize:
		registry.BroadcastEvent(targetSessionID, "summarize", SummarizeEvent{
			Type:             "summarize",
			Progress:         event.Progress,
			Done:             event.Done,
			ParentToolCallID: parentToolCallID,
		})

	case agent.AgentEventTypeToolExecutionStart:
		toolName := extractToolNameFromProgress(event.Progress)
		registry.BroadcastEvent(targetSessionID, "tool_execution_start", ToolExecutionStartEvent{
			Type:             "tool_execution_start",
			ToolName:         toolName,
			Progress:         event.Progress,
			ToolCallID:       event.ToolCallID,
			ParentToolCallID: parentToolCallID,
		})

	case agent.AgentEventTypeToolExecutionComplete:
		toolName := extractToolNameFromProgress(event.Progress)
		success := !strings.Contains(strings.ToLower(event.Progress), "error") && !strings.Contains(strings.ToLower(event.Progress), "failed")

		registry.BroadcastEvent(targetSessionID, "tool_execution_complete", ToolExecutionCompleteEvent{
			Type:             "tool_execution_complete",
			ToolName:         toolName,
			Progress:         event.Progress,
			Success:          success,
			ToolCallID:       event.ToolCallID,
			ParentToolCallID: parentToolCallID,
		})
	}
}

// extractToolNameFromProgress extracts tool name from progress string
func extractToolNameFromProgress(progress string) string {
	// Common progress formats:
	// "Executing tool: Bash"
	// "Tool Bash execution started"
	// etc.

	if strings.Contains(progress, "Executing tool: ") {
		parts := strings.Split(progress, "Executing tool: ")
		if len(parts) > 1 {
			return strings.TrimSpace(parts[1])
		}
	}

	if strings.Contains(progress, "Tool ") && strings.Contains(progress, " execution") {
		parts := strings.Split(progress, "Tool ")
		if len(parts) > 1 {
			toolPart := parts[1]
			if idx := strings.Index(toolPart, " execution"); idx > 0 {
				return strings.TrimSpace(toolPart[:idx])
			}
		}
	}

	return ""
}
