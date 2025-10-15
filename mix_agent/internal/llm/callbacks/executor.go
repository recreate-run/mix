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
	sessions        session.Service
	permissions     permission.Service
	messages        message.Service
	createSubAgent  SubAgentFactory
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

	return interfaces.CallbackResult{
		Success: true,
		Output:  output.String(),
	}, nil
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
