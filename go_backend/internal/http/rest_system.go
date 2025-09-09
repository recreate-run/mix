package http

import (
	"net/http"
	"os"
	"sort"
	"strings"
	"time"

	"mix/internal/app"
	"mix/internal/commands"
	"mix/internal/config"
	"mix/internal/llm/agent"
	"mix/internal/llm/provider"
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

// HandleAuthLogin handles POST /api/auth/login
func (h *SystemHandler) HandleAuthLogin(w http.ResponseWriter, r *http.Request) {
	setCORSHeaders(w)
	if handleCORSPreflight(w, r) {
		return
	}

	if r.Method != "POST" {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req AuthLoginRequest
	if err := parseJSONBody(r, &req); err != nil {
		sendValidationError(w, "body", err.Error())
		return
	}

	// Check if this is a manual API key submission
	if req.APIKey != "" {
		// Set environment variable
		os.Setenv("ANTHROPIC_API_KEY", req.APIKey)

		result := map[string]interface{}{
			"status":  "success",
			"message": "API key set successfully. You can now use the application.",
		}

		sendJSONResponse(w, http.StatusOK, result)
		return
	}

	if req.AuthCode == "" {
		sendValidationError(w, "authCode", "authCode or apiKey is required")
		return
	}

	storage, err := provider.NewCredentialStorage()
	if err != nil {
		sendInternalError(w, "initializing credential storage", err)
		return
	}

	// Extract state from auth code to retrieve the correct OAuth flow
	authCodeParts := strings.Split(req.AuthCode, "#")
	var oauthFlow *provider.OAuthFlow

	if len(authCodeParts) == 2 {
		// Auth code format: code#state
		state := authCodeParts[1]
		oauthFlow = provider.GetOAuthFlow(state)

		if oauthFlow == nil {
			sendErrorResponse(w, ErrorTypeValidation, "OAuth flow not found for this session. Please restart the authentication process.")
			return
		}
	} else {
		// Fallback: create new OAuth flow (for backwards compatibility)
		oauthFlow, err = provider.NewOAuthFlow("")
		if err != nil {
			sendInternalError(w, "creating OAuth flow", err)
			return
		}
	}

	// For manual token entry (from UI), check if this is an API key (starts with sk-ant-)
	if req.Manual && strings.HasPrefix(req.AuthCode, "sk-ant-") {
		// This is a direct API key, not an auth code
		os.Setenv("ANTHROPIC_API_KEY", req.AuthCode)

		result := map[string]interface{}{
			"status":  "success",
			"message": "API key set successfully. You can now use the application.",
		}

		sendJSONResponse(w, http.StatusOK, result)
		return
	}

	// Exchange the authorization code for tokens
	credentials, err := oauthFlow.ExchangeCodeForTokens(req.AuthCode)
	if err != nil {
		// Check if this is the Cloudflare protection error
		if strings.Contains(err.Error(), "Cloudflare") || strings.Contains(err.Error(), "manual token extraction") {
			result := map[string]interface{}{
				"status":  "error",
				"step":    "manual_fallback",
				"message": "OAuth flow completed but token exchange was blocked by Cloudflare protection. Please try one of these methods:\n\n1. Try again with the exact code format: code#state\n\n2. Or create an API key manually:\n   - Visit: https://console.anthropic.com/settings/keys\n   - Create a new API key\n   - Enter the API key in the form below\n\nNote: Terminal authentication may still work via `mix auth add anthropic-claude-pro-max`",
			}

			sendJSONResponse(w, http.StatusOK, result)
			return
		}

		// For other OAuth exchange failures, guide user to manual API key approach
		sendInternalError(w, "exchanging authorization code", err)
		return
	}

	// Store the credentials
	err = storage.StoreOAuthCredentials("anthropic", credentials.AccessToken, credentials.RefreshToken, credentials.ExpiresAt, credentials.ClientID)
	if err != nil {
		sendInternalError(w, "storing credentials", err)
		return
	}

	// Clean up the OAuth flow from memory after successful authentication
	if len(authCodeParts) == 2 {
		provider.CleanupOAuthFlow(authCodeParts[1])
	}

	result := map[string]interface{}{
		"status":    "success",
		"message":   "Successfully authenticated with Claude Code OAuth! You can now use the application.",
		"expiresIn": (credentials.ExpiresAt - time.Now().Unix()) / 60, // minutes
	}

	sendJSONResponse(w, http.StatusOK, result)
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
	os.Setenv("ANTHROPIC_API_KEY", req.APIKey)

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