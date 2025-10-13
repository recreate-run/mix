package callbacks

import (
	"context"
	"fmt"
	"strings"

	"mix/internal/llm/interfaces"
	"mix/internal/llm/tools/shell"
	"mix/internal/logging"
	"mix/internal/permission"
	"mix/internal/session"
)

type executor struct {
	sessions    session.Service
	permissions permission.Service
}

func NewExecutor(
	sessions session.Service,
	permissions permission.Service,
) interfaces.CallbackExecutor {
	return &executor{
		sessions:    sessions,
		permissions: permissions,
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
	// Build prompt with tool context
	prompt := config.SubAgentPrompt
	if prompt == "" {
		return interfaces.CallbackResult{Success: false, Error: "sub_agent_prompt is required"}, nil
	}

	// Sub-agent callbacks are not yet implemented due to circular dependency issues
	// To implement this feature, we would need to:
	// 1. Extract agent creation into a separate factory package
	// 2. Use dependency injection to pass agent factory to callback executor
	// 3. Get message history if config.IncludeFullHistory is true
	// 4. Create sub-agent with appropriate tools based on config.SubAgentType
	// 5. Run sub-agent and return its output

	return interfaces.CallbackResult{
		Success: false,
		Error:   "sub_agent callbacks not yet implemented - requires architecture refactoring to avoid circular dependencies",
	}, nil
}

// shellQuote safely quotes a string for use in bash
// It escapes single quotes and wraps the string in single quotes
func shellQuote(s string) string {
	// Replace single quotes with '\'' (end quote, escaped quote, start quote)
	escaped := strings.ReplaceAll(s, "'", "'\\''")
	return fmt.Sprintf("'%s'", escaped)
}
