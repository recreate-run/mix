package http

// REST API response types
// These typed structs replace map[string]interface{} for type safety and API contract clarity

// StoreToolAPIKeyResponse represents the success response when storing a tool API key
type StoreToolAPIKeyResponse struct {
	Status   string `json:"status"`
	ToolType string `json:"tool_type"`
	Provider string `json:"provider"`
	Message  string `json:"message"`
}

// DeleteToolCredentialResponse represents the success response when deleting a tool credential
type DeleteToolCredentialResponse struct {
	Status  string `json:"status"`
	Message string `json:"message"`
}

// SystemHealthResponse represents the main system health check endpoint response
type SystemHealthResponse struct {
	Status      string         `json:"status"`
	Timestamp   string         `json:"timestamp"`
	Version     string         `json:"version"`
	Environment string         `json:"environment"`
	Services    HealthServices `json:"services"`
}

// HealthServices represents the status of backend services
type HealthServices struct {
	Backend  string `json:"backend"`
	Database string `json:"database"`
}

// ActiveTunnelsResponse represents the list of active WebSocket tunnels
type ActiveTunnelsResponse struct {
	ActiveTunnels []string `json:"active_tunnels"`
	Count         int      `json:"count"`
}

// SetAPIKeyResponse represents the success response when setting an API key
type SetAPIKeyResponse struct {
	Status  string `json:"status"`
	Message string `json:"message"`
}

// PermissionResponse represents the response for permission grant/deny operations
type PermissionResponse struct {
	Status  string `json:"status"`
	ID      string `json:"id"`
	Message string `json:"message"`
}


// ProviderInfo represents metadata about an LLM provider
type ProviderInfo struct {
	DisplayName string      `json:"display_name"`
	Models      interface{} `json:"models"` // Dynamic structure from models package
}
