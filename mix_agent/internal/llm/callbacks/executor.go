package callbacks

import (
	"context"
	"fmt"
	"strings"

	"mix/internal/llm/interfaces"
	"mix/internal/llm/tools/shell"
	"mix/internal/logging"
	"mix/internal/message"
	"mix/internal/permission"
	"mix/internal/session"
)

// SubAgentFactory is a function that creates a subagent for callback execution
type SubAgentFactory func(subagentType string) (SubAgent, error)

type executor struct {
	sessions       session.Service
	permissions    permission.Service
	messages       message.Service
	createSubAgent SubAgentFactory
}

// SubAgent defines the minimal interface for a subagent created by callbacks
type SubAgent interface {
	Run(ctx context.Context, sessionID string, content string) (<-chan AgentEvent, error)
	Shutdown()
}

// AgentEvent represents an event from agent execution
// This mirrors the agent.AgentEvent struct but avoids circular import
type AgentEvent struct {
	Error   error
	Content string
	Message Message
}

// Message represents a message in the agent conversation
type Message interface {
	FinishReason() string
	Role() string
	Content() MessageContent
}

// MessageContent represents message content
type MessageContent interface {
	String() string
}

func NewExecutor(
	sessions session.Service,
	permissions permission.Service,
	messages message.Service,
	createSubAgent SubAgentFactory,
) interfaces.CallbackExecutor {
	return &executor{
		sessions:       sessions,
		permissions:    permissions,
		messages:       messages,
		createSubAgent: createSubAgent,
	}
}

func (e *executor) Execute(ctx context.Context, config interfaces.CallbackConfig, callbackCtx interfaces.CallbackContext) (interfaces.CallbackResult, error) {
	switch config.Type {
	case interfaces.CallbackTypeBashScript:
		return e.executeBash(ctx, config, callbackCtx)
	case interfaces.CallbackTypeSubAgent:
		return e.executeSubAgent(ctx, config, callbackCtx)
	case interfaces.CallbackTypeSendMessage:
		return e.executeSendMessage(ctx, config, callbackCtx)
	default:
		return interfaces.CallbackResult{}, fmt.Errorf("unknown callback type: %s", config.Type)
	}
}

func (e *executor) executeBash(ctx context.Context, config interfaces.CallbackConfig, callbackCtx interfaces.CallbackContext) (interfaces.CallbackResult, error) {
	// Get persistent shell for session
	sh, err := shell.GetPersistentShell(callbackCtx.SessionStorageDir)
	if err != nil {
		logging.Error("Failed to get persistent shell for callback", "error", err)
		return interfaces.CallbackResult{Success: false, Error: err.Error()}, nil
	}

	timeout := config.BashTimeout
	if timeout <= 0 {
		timeout = 120000 // 2 minutes default
	}

	// Prepare environment variables that will be injected into the bash command
	// Since we use a persistent shell, we need to export these variables inline
	envVars := fmt.Sprintf(`
export CALLBACK_TOOL_RESULT=%s
export CALLBACK_TOOL_NAME=%s
export CALLBACK_TOOL_ID=%s
export CALLBACK_SESSION_ID=%s
`,
		shellQuote(callbackCtx.ToolResult.Content),
		shellQuote(callbackCtx.ToolCall.Name),
		shellQuote(callbackCtx.ToolCall.ID),
		shellQuote(callbackCtx.SessionID),
	)

	// Combine environment variables with the actual command
	fullCommand := envVars + "\n" + config.BashCommand

	stdout, stderr, exitCode, interrupted, err := sh.Exec(ctx, fullCommand, timeout)

	result := interfaces.CallbackResult{
		Success: exitCode == 0 && !interrupted && err == nil,
		Output:  stdout,
	}

	if stderr != "" {
		result.Error = stderr
	}
	if err != nil {
		if result.Error != "" {
			result.Error += "\n"
		}
		result.Error += err.Error()
	}

	// Save callback result as a message for display in frontend
	if err := e.saveCallbackResultMessage(ctx, callbackCtx, config, result.Success, stdout, stderr, exitCode, "", ""); err != nil {
		logging.Error("Failed to save callback result message", "error", err)
		// Annotate output with warning (primary work succeeded, but metadata persistence failed)
		result.Output = fmt.Sprintf("%s\n\n⚠️ Warning: Failed to save callback metadata: %v", result.Output, err)
	}

	return result, nil
}

func (e *executor) executeSubAgent(ctx context.Context, config interfaces.CallbackConfig, callbackCtx interfaces.CallbackContext) (interfaces.CallbackResult, error) {
	// Validate configuration
	if config.SubAgentPrompt == "" {
		return interfaces.CallbackResult{Success: false, Error: "subAgentPrompt is required"}, nil
	}
	if e.createSubAgent == nil {
		logging.Warn("Sub-agent callback skipped: factory function not provided")
		return interfaces.CallbackResult{
			Success: false,
			Error:   "subagent factory not set - sub_agent callbacks require factory function",
		}, nil
	}

	// Build prompt with tool execution context
	prompt := buildSubAgentPrompt(config.SubAgentPrompt, callbackCtx)

	logging.Info("SubAgent callback input",
		"sessionID", callbackCtx.SessionID,
		"subAgentType", config.SubAgentType,
		"prompt", prompt,
	)

	// TODO: Get message history if config.IncludeFullHistory is true
	// This is a stub for now - will be implemented later
	if config.IncludeFullHistory {
		logging.Debug("includeFullHistory parameter not yet implemented, ignoring")
	}

	// Determine subagent type (default to general-purpose)
	subagentType := config.SubAgentType
	if subagentType == "" {
		subagentType = "general-purpose"
	}

	// Create subagent via factory function
	subAgent, err := e.createSubAgent(subagentType)
	if err != nil {
		return interfaces.CallbackResult{
			Success: false,
			Error:   fmt.Sprintf("failed to create sub-agent: %v", err),
		}, nil
	}
	defer subAgent.Shutdown()

	// Create subagent session with parent tool call ID for persistent UI nesting
	subSession, err := e.sessions.Create(
		ctx,
		"Callback: "+callbackCtx.ToolCall.Name,
		"", // no custom system prompt
		"default",
		session.SessionTypeSubagent,
		session.SubagentType(subagentType),
		callbackCtx.SessionID,
		callbackCtx.ToolCall.ID,
	)
	if err != nil {
		return interfaces.CallbackResult{
			Success: false,
			Error:   fmt.Sprintf("failed to create subagent session: %v", err),
		}, nil
	}

	// Run the subagent
	done, err := subAgent.Run(ctx, subSession.ID, prompt)
	if err != nil {
		return interfaces.CallbackResult{
			Success: false,
			Error:   fmt.Sprintf("failed to run sub-agent: %v", err),
		}, nil
	}

	// Wait for completion and collect output
	var output strings.Builder
	var finalMessage Message
	for event := range done {
		if event.Error != nil {
			return interfaces.CallbackResult{
				Success: false,
				Error:   fmt.Sprintf("sub-agent error: %v", event.Error),
			}, nil
		}

		// Collect content deltas
		if event.Content != "" {
			output.WriteString(event.Content)
		}

		// Check for final message
		if event.Message != nil && event.Message.FinishReason() == "end_turn" {
			finalMessage = event.Message
		}
	}

	// Extract final response content
	if finalMessage != nil && finalMessage.Content() != nil {
		output.Reset()
		output.WriteString(finalMessage.Content().String())
	}

	// Roll up subagent cost to parent session
	updatedSubSession, err := e.sessions.Get(ctx, subSession.ID)
	if err != nil {
		logging.Warn("Failed to get subagent session for cost rollup", "error", err)
	} else {
		if err := e.sessions.IncrementCost(ctx, callbackCtx.SessionID, updatedSubSession.Cost); err != nil {
			logging.Warn("Failed to increment parent session cost", "error", err, "parentSession", callbackCtx.SessionID)
		}
	}

	outputStr := output.String()
	logging.Info("SubAgent callback output",
		"sessionID", callbackCtx.SessionID,
		"subAgentSessionID", subSession.ID,
		"output", outputStr,
		"outputLength", len(outputStr),
	)

	// Save callback result as a message for display in frontend
	successOutput := outputStr
	if err := e.saveCallbackResultMessage(ctx, callbackCtx, config, true, "", "", 0, subSession.ID, outputStr); err != nil {
		logging.Error("Failed to save callback result message", "error", err)
		// Annotate output with warning (subagent succeeded, but metadata persistence failed)
		successOutput = fmt.Sprintf("%s\n\n⚠️ Warning: Failed to save callback metadata: %v", outputStr, err)
	}

	return interfaces.CallbackResult{
		Success: true,
		Output:  successOutput,
	}, nil
}

func (e *executor) executeSendMessage(ctx context.Context, config interfaces.CallbackConfig, callbackCtx interfaces.CallbackContext) (interfaces.CallbackResult, error) {
	// Validate configuration
	if config.MessageContent == "" {
		return interfaces.CallbackResult{Success: false, Error: "messageContent is required"}, nil
	}

	logging.Info("SendMessage callback executing",
		"sessionID", callbackCtx.SessionID,
		"toolName", callbackCtx.ToolCall.Name,
		"messagePreview", truncateString(config.MessageContent, 50),
	)

	// Create a User message in the current session
	// This will be picked up by the agent's next turn naturally
	_, err := e.messages.Create(ctx, callbackCtx.SessionID, message.CreateMessageParams{
		Role:  message.User,
		Parts: []message.ContentPart{message.TextContent{Text: config.MessageContent}},
	})

	if err != nil {
		logging.Error("Failed to create message in send_message callback", "error", err)
		return interfaces.CallbackResult{
			Success: false,
			Error:   fmt.Sprintf("failed to inject message: %v", err),
		}, nil
	}

	// Save callback result for UI display
	successOutput := fmt.Sprintf("Message injected into conversation: %s", truncateString(config.MessageContent, 100))
	if err := e.saveCallbackResultMessage(ctx, callbackCtx, config, true, "", "", 0, "", config.MessageContent); err != nil {
		logging.Error("Failed to save callback result message", "error", err)
		// Annotate output with warning (message injection succeeded, but metadata persistence failed)
		successOutput = fmt.Sprintf("%s\n\n⚠️ Warning: Failed to save callback metadata: %v", successOutput, err)
	}

	return interfaces.CallbackResult{
		Success: true,
		Output:  successOutput,
	}, nil
}

// truncateString truncates a string to maxLen characters with ellipsis
func truncateString(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen] + "..."
}

// buildSubAgentPrompt enriches the prompt with tool execution context
func buildSubAgentPrompt(basePrompt string, callbackCtx interfaces.CallbackContext) string {
	return fmt.Sprintf(`%s

Tool Execution Context:
- Tool: %s
- Tool Call ID: %s
- Result: %s`,
		basePrompt,
		callbackCtx.ToolCall.Name,
		callbackCtx.ToolCall.ID,
		callbackCtx.ToolResult.Content,
	)
}

// shellQuote safely quotes a string for use in bash
// It escapes single quotes and wraps the string in single quotes
func shellQuote(s string) string {
	// Replace single quotes with '\'' (end quote, escaped quote, start quote)
	escaped := strings.ReplaceAll(s, "'", "'\\''")
	return fmt.Sprintf("'%s'", escaped)
}

// saveCallbackResultMessage creates a tool message with callback result content part
func (e *executor) saveCallbackResultMessage(
	ctx context.Context,
	callbackCtx interfaces.CallbackContext,
	config interfaces.CallbackConfig,
	success bool,
	stdout, stderr string,
	exitCode int,
	subAgentID, subAgentResult string,
) error {
	callbackResult := message.CallbackResult{
		ToolCallID:     callbackCtx.ToolCall.ID,
		ToolName:       callbackCtx.ToolCall.Name,
		CallbackName:   config.Name,
		CallbackType:   string(config.Type),
		Stdout:         stdout,
		Stderr:         stderr,
		ExitCode:       exitCode,
		SubAgentID:     subAgentID,
		SubAgentResult: subAgentResult,
		Success:        success,
	}

	// Add error message if callback failed
	if !success && stderr != "" {
		callbackResult.Error = stderr
	}

	// Create a tool message with the callback result
	_, err := e.messages.Create(ctx, callbackCtx.SessionID, message.CreateMessageParams{
		Role:  message.Tool,
		Parts: []message.ContentPart{callbackResult},
	})

	return err
}
