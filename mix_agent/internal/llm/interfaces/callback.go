package interfaces

import (
	"context"
	"fmt"
)

// CallbackType defines the type of callback to execute
type CallbackType string

const (
	CallbackTypeBashScript CallbackType = "bash_script"
	CallbackTypeSubAgent   CallbackType = "sub_agent"
	// CallbackTypeSendMessage injects a User message into the conversation.
	// The message is injected after the tool completes and will be picked up
	// by the agent on its next conversation turn.
	CallbackTypeSendMessage CallbackType = "send_message"
)

// CallbackConfig defines configuration for a tool callback
type CallbackConfig struct {
	// Human-readable name for this callback (optional, defaults to "Callback #XXXX")
	Name string `json:"name,omitempty"`

	// Tool to attach callback to (e.g., "Show", "Bash", "*" for all tools)
	ToolName string `json:"toolName"`

	// Type of callback (bash_script or sub_agent)
	Type CallbackType `json:"type"`

	// For bash_script type
	BashCommand string `json:"bashCommand,omitempty"`
	BashTimeout int    `json:"bashTimeout,omitempty"` // milliseconds

	// For sub_agent type
	SubAgentPrompt     string `json:"subAgentPrompt,omitempty"`
	SubAgentType       string `json:"subAgentType,omitempty"`
	IncludeFullHistory bool   `json:"includeFullHistory,omitempty"`

	// For send_message type
	MessageContent string `json:"messageContent,omitempty"`

	// ExcludeFromContext prevents callback results from being included in agent context.
	// Only applies to bash_script and sub_agent types. Cannot be true for send_message.
	ExcludeFromContext bool `json:"excludeFromContext,omitempty"`
}

// MatchesTool checks if this callback should be executed for the given tool name.
// Returns true if the callback's ToolName exactly matches toolName or is "*" (wildcard).
func (c CallbackConfig) MatchesTool(toolName string) bool {
	return c.ToolName == toolName || c.ToolName == "*"
}

// Validate checks if this callback configuration is valid.
// Returns an error describing any validation failures.
func (c CallbackConfig) Validate() error {
	// Check required fields
	if c.ToolName == "" {
		return fmt.Errorf("missing required field 'toolName'")
	}

	if c.Type == "" {
		return fmt.Errorf("missing required field 'type'")
	}

	// Validate callback type
	if c.Type != CallbackTypeBashScript && c.Type != CallbackTypeSubAgent && c.Type != CallbackTypeSendMessage {
		return fmt.Errorf("type must be 'bash_script', 'sub_agent', or 'send_message', got '%s'", c.Type)
	}

	// Validate type-specific required fields
	if c.Type == CallbackTypeBashScript {
		if c.BashCommand == "" {
			return fmt.Errorf("bash_script type requires 'bashCommand' field")
		}
	}

	if c.Type == CallbackTypeSubAgent {
		if c.SubAgentPrompt == "" {
			return fmt.Errorf("sub_agent type requires 'subAgentPrompt' field")
		}
	}

	if c.Type == CallbackTypeSendMessage {
		if c.MessageContent == "" {
			return fmt.Errorf("send_message type requires 'messageContent' field")
		}
		// send_message callbacks cannot be excluded from context because they create conversation content
		if c.ExcludeFromContext {
			return fmt.Errorf("send_message callbacks cannot be excluded from context")
		}
	}

	return nil
}

// CallbackContext provides context for callback execution
type CallbackContext struct {
	SessionID         string
	MessageID         string
	ToolCall          ToolCall
	ToolResult        ToolResponse
	SessionStorageDir string
	MessageHistory    []interface{} // Full chat history if needed
}

// CallbackResult contains the result of callback execution
type CallbackResult struct {
	Success bool   `json:"success"`
	Output  string `json:"output"`
	Error   string `json:"error,omitempty"`
}

// CallbackExecutor handles callback execution
type CallbackExecutor interface {
	Execute(ctx context.Context, config CallbackConfig, callbackCtx CallbackContext) (CallbackResult, error)
}
