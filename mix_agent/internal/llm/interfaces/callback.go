package interfaces

import "context"

// CallbackType defines the type of callback to execute
type CallbackType string

const (
	CallbackTypeBashScript CallbackType = "bash_script"
	CallbackTypeSubAgent   CallbackType = "sub_agent"
)

// CallbackConfig defines configuration for a tool callback
type CallbackConfig struct {
	// Tool to attach callback to (e.g., "show_media", "bash", "*" for all tools)
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

	// Common options
	NonBlocking bool `json:"nonBlocking,omitempty"` // Run async without waiting
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

// CallbackTool is an optional interface that tools can implement
// to register post-execution callbacks
type CallbackTool interface {
	BaseTool
	// GetCallbacks returns callback configurations for this tool
	GetCallbacks() []CallbackConfig
}
