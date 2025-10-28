package agent

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"

	"mix/internal/llm/interfaces"
	"mix/internal/llm/tools"
	"mix/internal/logging"
	"mix/internal/message"
	"mix/internal/permission"
	"mix/internal/pubsub"
)

// executeToolsWithDependencies executes all tool calls with dependency awareness
func (a *agent) executeToolsWithDependencies(ctx context.Context, sessionID string, toolCalls []message.ToolCall, assistantMsg message.Message) ([]message.ContentPart, error) {
	if len(toolCalls) == 0 {
		return nil, nil
	}

	// For now, implement a simple approach that executes all tools in parallel
	// unless we detect obvious dependencies (same file operations)
	// This is a pragmatic first step that will give us immediate performance gains

	var results []message.ContentPart
	var resultsMutex sync.Mutex
	var wg sync.WaitGroup
	var firstError error

	// Filter tools based on plan mode
	availableTools := a.tools
	if ctx.Value(interfaces.PlanModeContextKey) != nil {
		availableTools = filterToolsForPlanMode(a.tools)
	}

	// Execute tools concurrently
	for i, toolCall := range toolCalls {
		wg.Add(1)
		go func(index int, tc message.ToolCall) {
			defer wg.Done()

			result := a.executeToolCall(ctx, sessionID, tc, availableTools, assistantMsg)

			resultsMutex.Lock()
			// Ensure results are in the same order as toolCalls
			if len(results) <= index {
				// Extend slice if needed
				newResults := make([]message.ContentPart, index+1)
				copy(newResults, results)
				results = newResults
			}
			results[index] = result
			resultsMutex.Unlock()
		}(i, toolCall)
	}

	// Wait for all tools to complete
	wg.Wait()

	// Ensure results slice is properly sized
	if len(results) < len(toolCalls) {
		newResults := make([]message.ContentPart, len(toolCalls))
		copy(newResults, results)
		results = newResults
	}

	// Check for any errors in results
	for _, result := range results {
		if toolResult, ok := result.(message.ToolResult); ok && toolResult.IsError {
			// If any tool failed, we still return all results but log the error
			if firstError == nil {
				firstError = fmt.Errorf("tool execution failed: %s", toolResult.Content)
			}
		}
	}

	return results, firstError
}

// executeToolCall executes a single tool call with proper error handling and events
func (a *agent) executeToolCall(ctx context.Context, sessionID string, toolCall message.ToolCall, availableTools []tools.BaseTool, assistantMsg message.Message) message.ToolResult {
	// Check for context cancellation
	select {
	case <-ctx.Done():
		return message.ToolResult{
			ToolCallID: toolCall.ID,
			Content:    "Tool execution cancelled",
			IsError:    true,
		}
	default:
	}

	// Find tool
	var tool tools.BaseTool
	for _, availableTool := range availableTools {
		if availableTool.Info().Name == toolCall.Name {
			tool = availableTool
			break
		}
	}

	// Tool not found
	if tool == nil {
		return message.ToolResult{
			ToolCallID: toolCall.ID,
			Content:    fmt.Sprintf("Tool not found: %s", toolCall.Name),
			IsError:    true,
		}
	}

	// Check if tool is available in plan mode
	if ctx.Value("plan_mode") != nil && !isToolAllowedInPlanMode(tool) {
		return message.ToolResult{
			ToolCallID: toolCall.ID,
			Content:    "Tool not available in plan mode. Use exit_plan_mode to proceed with execution.",
			IsError:    true,
		}
	}

	// Publish tool execution start event
	err := a.Publish(ctx, pubsub.CreatedEvent, AgentEvent{
		Type:       AgentEventTypeToolExecutionStart,
		Message:    assistantMsg,
		SessionID:  sessionID,
		Progress:   fmt.Sprintf("Executing %s tool", toolCall.Name),
		ToolCallID: toolCall.ID,
	})
	if err != nil {
		logging.Error("Failed to publish tool execution start event", "error", err)
	}

	toolStartTime := time.Now()
	toolResult, toolErr := tool.Run(ctx, tools.ToolCall{
		ID:    toolCall.ID,
		Name:  toolCall.Name,
		Input: toolCall.Input,
	})
	toolDuration := time.Since(toolStartTime)

	// Publish tool execution completion event
	completionProgress := fmt.Sprintf("Completed %s tool in %v", toolCall.Name, toolDuration)
	if toolErr != nil {
		completionProgress = fmt.Sprintf("Failed %s tool after %v: %v", toolCall.Name, toolDuration, toolErr)
	}
	err = a.Publish(ctx, pubsub.CreatedEvent, AgentEvent{
		Type:       AgentEventTypeToolExecutionComplete,
		Message:    assistantMsg,
		SessionID:  sessionID,
		Progress:   completionProgress,
		ToolCallID: toolCall.ID,
	})
	if err != nil {
		logging.Error("Failed to publish tool execution completion event", "error", err)
	}

	// Handle permission denied - special case for security
	if toolErr != nil {
		if errors.Is(toolErr, permission.ErrPermissionDenied) {
			return message.ToolResult{
				ToolCallID: toolCall.ID,
				Content:    "The user doesn't want to proceed with this tool use. The tool use was rejected (eg. if it was a file edit, the new_string was NOT written to the file). STOP what you are doing and wait for the user to tell you how to proceed.",
				IsError:    false, // Not a technical error - it's a security boundary
			}
		}

		// Enhanced error logging with context information
		logFields := []any{"toolName", toolCall.Name, "sessionID", sessionID, "toolCallID", toolCall.ID, "error", toolErr}

		// Check if error is due to context timeout or cancellation
		switch {
		case errors.Is(toolErr, context.DeadlineExceeded):
			logFields = append(logFields, "cause", "timeout_exceeded")
			logging.Error("[Agent] Tool execution failed: timeout exceeded", logFields...)
		case errors.Is(toolErr, context.Canceled):
			logFields = append(logFields, "cause", "context_cancelled")
			logging.Error("[Agent] Tool execution failed: context cancelled", logFields...)
		default:
			logging.Error("[Agent] Tool execution failed", logFields...)
		}
	}

	// Create tool result
	result := message.ToolResult{
		ToolCallID: toolCall.ID,
		Content:    toolResult.Content,
		Metadata:   toolResult.Metadata,
		IsError:    toolResult.IsError,
	}

	return result
}

// getSessionCallbacks loads and filters callbacks from session configuration
func (a *agent) getSessionCallbacks(ctx context.Context, sessionID, toolName string) ([]interfaces.CallbackConfig, error) {
	session, err := a.sessions.Get(ctx, sessionID)
	if err != nil {
		return nil, fmt.Errorf("failed to get session: %w", err)
	}

	allCallbacks, err := session.GetCallbacks()
	if err != nil {
		return nil, fmt.Errorf("failed to parse session callbacks: %w", err)
	}

	if len(allCallbacks) == 0 {
		return []interfaces.CallbackConfig{}, nil
	}

	// Filter callbacks for this tool (exact match or wildcard)
	var filtered []interfaces.CallbackConfig
	for i := range allCallbacks {
		if allCallbacks[i].MatchesTool(toolName) {
			filtered = append(filtered, allCallbacks[i])
		}
	}

	return filtered, nil
}
