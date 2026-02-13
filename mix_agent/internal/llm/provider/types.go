package provider

// LLM provider-specific types
// These typed structs replace map[string]interface{} for type safety where field structure is known

// ToolResultResponse represents a structured tool execution result
type ToolResultResponse struct {
	Result string `json:"result"`
}
