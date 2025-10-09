package http

import (
	"net/http"
	"os"
	"sort"
	"strings"

	"mix/internal/app"
	"mix/internal/commands"
	"mix/internal/config"
	"mix/internal/llm/agent"
	"mix/internal/llm/tools"
	"mix/internal/logging"
	"mix/internal/permission"
)

// ToolData represents tool information for REST API
type ToolData struct {
	Name        string `json:"name"`
	Description string `json:"description"`
}

// MCPServerData represents MCP server information for REST API
type MCPServerData struct {
	Name      string     `json:"name"`
	Connected bool       `json:"connected"`
	Status    string     `json:"status"`
	Tools     []ToolData `json:"tools"`
}

// CommandData represents command information for REST API
type CommandData struct {
	Name        string `json:"name"`
	Description string `json:"description"`
	Type        string `json:"type"` // "builtin" or "file"
}

// SystemHandler handles REST endpoints for system operations
type SystemHandler struct {
	app             *app.App
	commandRegistry *commands.Registry
}

// NewSystemHandler creates a new system handler
func NewSystemHandler(app *app.App) *SystemHandler {
	// Create command registry
	registry := commands.NewRegistry()
	if err := registry.LoadCommands(app); err != nil {
		logging.Error("Failed to load commands", "error", err)
		// Continue with empty registry - API will return proper errors
	}

	return &SystemHandler{
		app:             app,
		commandRegistry: registry,
	}
}

// AuthLoginRequest represents the request body for authentication
type AuthLoginRequest struct {
	AuthCode string `json:"authCode"`
	APIKey   string `json:"apiKey"` // Allow direct API key submission
	Manual   bool   `json:"manual"` // Flag for manual token input
}

// SetAPIKeyRequest represents the request body for setting API key
type SetAPIKeyRequest struct {
	APIKey string `json:"apiKey"`
}

// HandleSetAPIKey handles POST /api/auth/apikey
func (h *SystemHandler) HandleSetAPIKey(w http.ResponseWriter, r *http.Request) {
	setCORSHeaders(w)
	if handleCORSPreflight(w, r) {
		return
	}

	if r.Method != "POST" {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req SetAPIKeyRequest
	if err := parseJSONBody(r, &req); err != nil {
		sendValidationError(w, "body", err.Error())
		return
	}

	if req.APIKey == "" {
		sendValidationError(w, "apiKey", "API key is required")
		return
	}

	// Set environment variable
	_ = os.Setenv("ANTHROPIC_API_KEY", req.APIKey)

	result := map[string]interface{}{
		"status":  "success",
		"message": "API key set successfully. You can now use the application.",
	}

	sendJSONResponse(w, http.StatusOK, result)
}

// HandleListMCPServers handles GET /api/mcp
func (h *SystemHandler) HandleListMCPServers(w http.ResponseWriter, r *http.Request) {
	setCORSHeaders(w)
	if handleCORSPreflight(w, r) {
		return
	}

	if r.Method != "GET" {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	ctx := r.Context()
	cfg := config.Get()

	result := make([]MCPServerData, 0)

	if len(cfg.MCPServers) == 0 {
		sendJSONResponse(w, http.StatusOK, result) // Empty array
		return
	}

	// Get MCP tools to check connection status and group by server
	// Create temporary manager for informational listing
	tempManager := agent.NewMCPClientManager()
	defer tempManager.Close()
	mcpTools := agent.GetMcpTools(ctx, h.app.Permissions, tempManager)

	// Group tools by server name
	serverTools := make(map[string][]tools.BaseTool)
	for _, tool := range mcpTools {
		if toolInfo := tool.Info(); strings.Contains(toolInfo.Name, "_") {
			serverName := strings.Split(toolInfo.Name, "_")[0]
			serverTools[serverName] = append(serverTools[serverName], tool)
		}
	}

	// Sort server names for consistent output
	var serverNames []string
	for name := range cfg.MCPServers {
		serverNames = append(serverNames, name)
	}
	sort.Strings(serverNames)

	for _, name := range serverNames {
		tools := serverTools[name]

		// Determine connection status
		connected := len(tools) > 0
		status := "connected"
		if !connected {
			status = "failed"
		}

		// Convert tools to ToolData
		var toolsData []ToolData
		for _, tool := range tools {
			info := tool.Info()
			// Remove server prefix from tool name for cleaner display
			toolName := info.Name
			if strings.Contains(toolName, "_") {
				parts := strings.SplitN(toolName, "_", 2)
				if len(parts) > 1 {
					toolName = parts[1]
				}
			}
			toolsData = append(toolsData, ToolData{
				Name:        toolName,
				Description: info.Description,
			})
		}

		// Sort tools by name
		sort.Slice(toolsData, func(i, j int) bool {
			return toolsData[i].Name < toolsData[j].Name
		})

		result = append(result, MCPServerData{
			Name:      name,
			Connected: connected,
			Status:    status,
			Tools:     toolsData,
		})
	}

	sendJSONResponse(w, http.StatusOK, result)
}

// HandleListCommands handles GET /api/commands
func (h *SystemHandler) HandleListCommands(w http.ResponseWriter, r *http.Request) {
	setCORSHeaders(w)
	if handleCORSPreflight(w, r) {
		return
	}

	if r.Method != "GET" {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	allCommands := h.commandRegistry.GetAllCommands()

	var result []CommandData
	builtins := map[string]bool{
		"help": true, "clear": true, "session": true,
		"sessions": true, "tools": true, "mcp": true,
	}

	for name, cmd := range allCommands {
		cmdType := "file"
		if builtins[name] {
			cmdType = "builtin"
		}

		result = append(result, CommandData{
			Name:        name,
			Description: cmd.Description(),
			Type:        cmdType,
		})
	}

	// Sort by name
	sort.Slice(result, func(i, j int) bool {
		return result[i].Name < result[j].Name
	})

	sendJSONResponse(w, http.StatusOK, result)
}

// HandleGetCommand handles GET /api/commands/{name}
func (h *SystemHandler) HandleGetCommand(w http.ResponseWriter, r *http.Request) {
	setCORSHeaders(w)
	if handleCORSPreflight(w, r) {
		return
	}

	if r.Method != "GET" {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	commandName := r.PathValue("name")
	if commandName == "" {
		sendValidationError(w, "name", "command name is required")
		return
	}

	cmd, exists := h.commandRegistry.GetCommand(commandName)
	if !exists {
		sendNotFoundError(w, "Command", commandName)
		return
	}

	builtins := map[string]bool{
		"help": true, "clear": true, "session": true,
		"sessions": true, "tools": true, "mcp": true,
	}

	cmdType := "file"
	if builtins[commandName] {
		cmdType = "builtin"
	}

	result := CommandData{
		Name:        cmd.Name(),
		Description: cmd.Description(),
		Type:        cmdType,
	}

	sendJSONResponse(w, http.StatusOK, result)
}

// PermissionRequest represents the request body for permission operations
type PermissionRequest struct {
	ID string `json:"id"`
}

// HandleGrantPermission handles POST /api/permissions/{id}/grant
func (h *SystemHandler) HandleGrantPermission(w http.ResponseWriter, r *http.Request) {
	setCORSHeaders(w)
	if handleCORSPreflight(w, r) {
		return
	}

	if r.Method != "POST" {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	permissionID := r.PathValue("id")
	if permissionID == "" {
		sendValidationError(w, "id", "permission ID is required")
		return
	}

	// Grant the permission using the existing service
	h.app.Permissions.Grant(permission.PermissionRequest{ID: permissionID})

	result := map[string]string{
		"status":  "granted",
		"id":      permissionID,
		"message": "Permission granted successfully",
	}

	sendJSONResponse(w, http.StatusOK, result)
}

// HandleDenyPermission handles POST /api/permissions/{id}/deny
func (h *SystemHandler) HandleDenyPermission(w http.ResponseWriter, r *http.Request) {
	setCORSHeaders(w)
	if handleCORSPreflight(w, r) {
		return
	}

	if r.Method != "POST" {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	permissionID := r.PathValue("id")
	if permissionID == "" {
		sendValidationError(w, "id", "permission ID is required")
		return
	}

	// Deny the permission using the existing service
	h.app.Permissions.Deny(permission.PermissionRequest{ID: permissionID})

	result := map[string]string{
		"status":  "denied",
		"id":      permissionID,
		"message": "Permission denied successfully",
	}

	sendJSONResponse(w, http.StatusOK, result)
}
