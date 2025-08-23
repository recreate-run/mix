package api

import (
	"context"
	"encoding/json"
	"fmt"
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

// JSON-RPC Request
type QueryRequest struct {
	Method string          `json:"method"`
	Params json.RawMessage `json:"params,omitempty"`
	ID     interface{}     `json:"id"`
}

// JSON-RPC Response
type QueryResponse struct {
	Result interface{} `json:"result,omitempty"`
	Error  *QueryError `json:"error,omitempty"`
	ID     interface{} `json:"id"`
}

// JSON-RPC Error
type QueryError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

// Structured data types
type SessionData struct {
	ID                    string    `json:"id"`
	Title                 string    `json:"title"`
	UserMessageCount      int64     `json:"userMessageCount"`
	AssistantMessageCount int64     `json:"assistantMessageCount"`
	ToolCallCount         int64     `json:"toolCallCount"`
	PromptTokens          int64     `json:"promptTokens"`
	CompletionTokens      int64     `json:"completionTokens"`
	Cost                  float64   `json:"cost"`
	CreatedAt             time.Time `json:"createdAt"`
	WorkingDirectory      string    `json:"workingDirectory,omitempty"`
	FirstUserMessage      string    `json:"firstUserMessage,omitempty"`
}

type ToolData struct {
	Name        string `json:"name"`
	Description string `json:"description"`
}

type MCPServerData struct {
	Name      string     `json:"name"`
	Connected bool       `json:"connected"`
	Status    string     `json:"status"`
	Tools     []ToolData `json:"tools"`
}

type CommandData struct {
	Name        string `json:"name"`
	Description string `json:"description"`
	Type        string `json:"type"` // "builtin" or "file"
}

type ToolCallData struct {
	ID       string `json:"id"`
	Name     string `json:"name"`
	Input    string `json:"input"`
	Type     string `json:"type"`
	Finished bool   `json:"finished"`
}

type MessageData struct {
	ID        string         `json:"id"`
	SessionID string         `json:"sessionId"`
	Role      string         `json:"role"`
	Content   string         `json:"content"`
	Response  string         `json:"response,omitempty"`
	ToolCalls []ToolCallData `json:"toolCalls,omitempty"`
}

// Error response helper functions

// newErrorResponse creates a standardized QueryResponse with error
func newErrorResponse(req *QueryRequest, code int, message string) *QueryResponse {
	return &QueryResponse{
		Error: &QueryError{
			Code:    code,
			Message: message,
		},
		ID: req.ID,
	}
}

// newInvalidParamsError creates a -32602 Invalid params error response
func newInvalidParamsError(req *QueryRequest, err error) *QueryResponse {
	return newErrorResponse(req, -32602, "Invalid params: "+err.Error())
}

// newMissingParamError creates a -32602 Missing required parameter error response
func newMissingParamError(req *QueryRequest, param string) *QueryResponse {
	return newErrorResponse(req, -32602, "Missing required parameter: "+param)
}

// newInternalError creates a -32603 Internal error response
func newInternalError(req *QueryRequest, err error) *QueryResponse {
	return newErrorResponse(req, -32603, "Internal error: "+err.Error())
}

// newMethodNotFoundError creates a -32601 Method not found error response
func newMethodNotFoundError(req *QueryRequest, method string) *QueryResponse {
	return newErrorResponse(req, -32601, "Method not found: "+method)
}

// newApplicationError creates a -32000 Application-specific error response
func newApplicationError(req *QueryRequest, message string) *QueryResponse {
	return newErrorResponse(req, -32000, message)
}

// Query handler
type QueryHandler struct {
	app             *app.App
	commandRegistry *commands.Registry
}

func NewQueryHandler(app *app.App) *QueryHandler {
	// Create command registry
	registry := commands.NewRegistry()
	if err := registry.LoadCommands(app); err != nil {
		logging.Error("Failed to load commands", "error", err)
		// Continue with empty registry - API will return proper errors
	}
	return &QueryHandler{
		app:             app,
		commandRegistry: registry,
	}
}

// GetApp returns the app instance for external use
func (h *QueryHandler) GetApp() *app.App {
	return h.app
}

// Helper function to get command names for logging
func getCommandNames(commands map[string]commands.Command) []string {
	names := make([]string, 0, len(commands))
	for name := range commands {
		names = append(names, name)
	}
	return names
}

func (h *QueryHandler) Handle(ctx context.Context, req *QueryRequest) *QueryResponse {
	switch req.Method {
	case "sessions.list":
		return h.handleSessionsList(ctx, req)
	case "sessions.get":
		return h.handleSessionsGet(ctx, req)
	case "sessions.current":
		return h.handleSessionsCurrent(ctx, req)
	case "sessions.select":
		return h.handleSessionsSelect(ctx, req)
	case "sessions.create":
		return h.handleSessionsCreate(ctx, req)
	case "sessions.fork":
		return h.handleSessionsFork(ctx, req)
	case "sessions.delete":
		return h.handleSessionsDelete(ctx, req)
	case "messages.send":
		return h.handleMessagesSend(ctx, req)
	case "messages.history":
		return h.handleMessagesHistory(ctx, req)
	case "messages.list":
		return h.handleMessagesList(ctx, req)
	case "mcp.list":
		return h.handleMCPList(ctx, req)
	case "commands.list":
		return h.handleCommandsList(ctx, req)
	case "commands.get":
		return h.handleCommandsGet(ctx, req)
	case "agent.cancel":
		return h.handleAgentCancel(ctx, req)
	case "auth.login":
		return h.handleAuthLogin(ctx, req)
	case "auth.apikey":
		return h.handleSetAPIKey(ctx, req)
	case "permission.grant":
		return h.handlePermissionGrant(ctx, req)
	case "permission.deny":
		return h.handlePermissionDeny(ctx, req)
	default:
		return newMethodNotFoundError(req, req.Method)
	}
}

// HandleQueryType handles a query by type, mapping to appropriate JSON-RPC method
func (h *QueryHandler) HandleQueryType(ctx context.Context, queryType string) *QueryResponse {
	// Check if queryType is supported
	supportedTypes := h.GetSupportedQueryTypes()
	for _, supported := range supportedTypes {
		if queryType == supported {
			// Construct method using pattern
			method := queryType + ".list"
			req := &QueryRequest{Method: method, ID: 1}
			return h.Handle(ctx, req)
		}
	}

	// Invalid query type
	req := &QueryRequest{ID: 1} // Create temporary request for error response
	return newErrorResponse(req, -32602, "Invalid query type: " + queryType + ". Supported: " + strings.Join(supportedTypes, ", "))
}

// GetSupportedQueryTypes returns all supported query types
func (h *QueryHandler) GetSupportedQueryTypes() []string {
	return []string{"sessions", "tools", "mcp", "commands"}
}

func (h *QueryHandler) handleSetAPIKey(ctx context.Context, req *QueryRequest) *QueryResponse {
	var params struct {
		APIKey string `json:"apiKey"`
	}

	if err := json.Unmarshal(req.Params, &params); err != nil {
		return newInvalidParamsError(req, err)
	}

	if params.APIKey == "" {
		return newMissingParamError(req, "apiKey")
	}

	// Set environment variable
	os.Setenv("ANTHROPIC_API_KEY", params.APIKey)

	// No need to reload providers, they'll pick up the new API key from env automatically
	// on the next request

	return &QueryResponse{
		Result: map[string]interface{}{
			"status":  "success",
			"message": "API key set successfully. You can now use the application.",
		},
		ID: req.ID,
	}
}

func (h *QueryHandler) handleAuthLogin(ctx context.Context, req *QueryRequest) *QueryResponse {
	var params struct {
		AuthCode string `json:"authCode"`
		APIKey   string `json:"apiKey"` // Allow direct API key submission
		Manual   bool   `json:"manual"` // Flag for manual token input
	}

	if err := json.Unmarshal(req.Params, &params); err != nil {
		return newInvalidParamsError(req, err)
	}

	// Check if this is a manual API key submission
	if params.APIKey != "" {
		// Set environment variable
		os.Setenv("ANTHROPIC_API_KEY", params.APIKey)

		return &QueryResponse{
			Result: map[string]interface{}{
				"status":  "success",
				"message": "API key set successfully. You can now use the application.",
			},
			ID: req.ID,
		}
	}

	if params.AuthCode == "" {
		return newMissingParamError(req, "authCode or apiKey")
	}

	storage, err := provider.NewCredentialStorage()
	if err != nil {
		return newErrorResponse(req, -32603, "Failed to initialize credential storage: " + err.Error())
	}

	// Extract state from auth code to retrieve the correct OAuth flow
	authCodeParts := strings.Split(params.AuthCode, "#")
	var oauthFlow *provider.OAuthFlow

	if len(authCodeParts) == 2 {
		// Auth code format: code#state
		state := authCodeParts[1]
		oauthFlow = provider.GetOAuthFlow(state)

		if oauthFlow == nil {
			return newErrorResponse(req, -32603, "OAuth flow not found for this session. Please restart the authentication process.")
		}
	} else {
		// Fallback: create new OAuth flow (for backwards compatibility)
		var err error
		oauthFlow, err = provider.NewOAuthFlow("")
		if err != nil {
			return newErrorResponse(req, -32603, "Failed to create OAuth flow: " + err.Error())
		}
	}

	// For manual token entry (from UI), check if this is an API key (starts with sk-ant-)
	if params.Manual && strings.HasPrefix(params.AuthCode, "sk-ant-") {
		// This is a direct API key, not an auth code
		os.Setenv("ANTHROPIC_API_KEY", params.AuthCode)

		return &QueryResponse{
			Result: map[string]interface{}{
				"status":  "success",
				"message": "API key set successfully. You can now use the application.",
			},
			ID: req.ID,
		}
	}

	// Exchange the authorization code for tokens
	credentials, err := oauthFlow.ExchangeCodeForTokens(params.AuthCode)
	if err != nil {
		// Check if this is the Cloudflare protection error
		if strings.Contains(err.Error(), "Cloudflare") || strings.Contains(err.Error(), "manual token extraction") {
			return &QueryResponse{
				Result: map[string]interface{}{
					"status":  "error",
					"step":    "manual_fallback",
					"message": "OAuth flow completed but token exchange was blocked by Cloudflare protection. Please try one of these methods:\n\n1. Try again with the exact code format: code#state\n\n2. Or create an API key manually:\n   - Visit: https://console.anthropic.com/settings/keys\n   - Create a new API key\n   - Enter the API key in the form below\n\nNote: Terminal authentication may still work via `mix auth add anthropic-claude-pro-max`",
				},
				ID: req.ID,
			}
		}

		// For other OAuth exchange failures, guide user to manual API key approach
		return newErrorResponse(req, -32603, "Failed to exchange authorization code: " + err.Error())
	}

	// Store the credentials
	err = storage.StoreOAuthCredentials("anthropic", credentials.AccessToken, credentials.RefreshToken, credentials.ExpiresAt, credentials.ClientID)
	if err != nil {
		return newErrorResponse(req, -32603, "Failed to store credentials: " + err.Error())
	}

	// Clean up the OAuth flow from memory after successful authentication
	if len(authCodeParts) == 2 {
		provider.CleanupOAuthFlow(authCodeParts[1])
	}

	return &QueryResponse{
		Result: map[string]interface{}{
			"status":    "success",
			"message":   "Successfully authenticated with Claude Code OAuth! You can now use the application.",
			"expiresIn": (credentials.ExpiresAt - time.Now().Unix()) / 60, // minutes
		},
		ID: req.ID,
	}
}

func (h *QueryHandler) handleSessionsList(ctx context.Context, req *QueryRequest) *QueryResponse {
	sessions, err := h.app.Sessions.ListWithContent(ctx)
	if err != nil {
		return newApplicationError(req, "Failed to list sessions: " + err.Error())
	}

	var result []SessionData
	for _, s := range sessions {
		workingDir := ""
		if s.WorkingDirectory.Valid {
			workingDir = s.WorkingDirectory.String
		}

		result = append(result, SessionData{
			ID:                    s.ID,
			Title:                 s.Title,
			UserMessageCount:      s.UserMessageCount,
			AssistantMessageCount: s.AssistantMessageCount,
			ToolCallCount:         s.ToolCallCount,
			PromptTokens:          s.PromptTokens,
			CompletionTokens:      s.CompletionTokens,
			Cost:                  s.Cost,
			CreatedAt:             time.Unix(s.CreatedAt, 0),
			WorkingDirectory:      workingDir,
			FirstUserMessage:      s.FirstUserMessage,
		})
	}

	return &QueryResponse{
		Result: result,
		ID:     req.ID,
	}
}

func (h *QueryHandler) handleSessionsGet(ctx context.Context, req *QueryRequest) *QueryResponse {
	var params struct {
		ID string `json:"id"`
	}

	if err := json.Unmarshal(req.Params, &params); err != nil {
		return newInvalidParamsError(req, err)
	}

	if params.ID == "" {
		return newMissingParamError(req, "id")
	}

	session, err := h.app.Sessions.Get(ctx, params.ID)
	if err != nil {
		return newApplicationError(req, "Failed to get session: " + err.Error())
	}

	result := SessionData{
		ID:               session.ID,
		Title:            session.Title,
		UserMessageCount:      session.UserMessageCount,
		AssistantMessageCount: session.AssistantMessageCount,
		ToolCallCount:         session.ToolCallCount,
		PromptTokens:     session.PromptTokens,
		CompletionTokens: session.CompletionTokens,
		Cost:             session.Cost,
		CreatedAt:        time.Unix(session.CreatedAt, 0),
		WorkingDirectory: session.WorkingDirectory,
	}

	return &QueryResponse{
		Result: result,
		ID:     req.ID,
	}
}

func (h *QueryHandler) handleSessionsCurrent(ctx context.Context, req *QueryRequest) *QueryResponse {
	currentSession, err := h.app.GetCurrentSession(ctx)
	if err != nil {
		return newApplicationError(req, "Failed to get current session: " + err.Error())
	}

	if currentSession == nil {
		return newApplicationError(req, "No current session selected")
	}

	result := SessionData{
		ID:               currentSession.ID,
		Title:            currentSession.Title,
		UserMessageCount:      currentSession.UserMessageCount,
		AssistantMessageCount: currentSession.AssistantMessageCount,
		ToolCallCount:         currentSession.ToolCallCount,
		PromptTokens:     currentSession.PromptTokens,
		CompletionTokens: currentSession.CompletionTokens,
		Cost:             currentSession.Cost,
		CreatedAt:        time.Unix(currentSession.CreatedAt, 0),
	}

	return &QueryResponse{
		Result: result,
		ID:     req.ID,
	}
}

func (h *QueryHandler) handleSessionsSelect(ctx context.Context, req *QueryRequest) *QueryResponse {
	var params struct {
		ID string `json:"id"`
	}

	if err := json.Unmarshal(req.Params, &params); err != nil {
		return newInvalidParamsError(req, err)
	}

	if params.ID == "" {
		return newMissingParamError(req, "id")
	}

	// Check if already on this session
	currentSessionID := h.app.GetCurrentSessionID()
	if params.ID == currentSessionID {
		return &QueryResponse{
			Result: map[string]string{"message": "Already on session: " + params.ID},
			ID:     req.ID,
		}
	}

	// Set current session
	err := h.app.SetCurrentSession(params.ID)
	if err != nil {
		return newApplicationError(req, "Failed to select session: " + err.Error())
	}

	return &QueryResponse{
		Result: map[string]string{"message": "Session selected: " + params.ID},
		ID:     req.ID,
	}
}

func (h *QueryHandler) handleSessionsCreate(ctx context.Context, req *QueryRequest) *QueryResponse {
	var params struct {
		Title            string `json:"title"`
		SetCurrent       bool   `json:"setCurrent,omitempty"`
		WorkingDirectory string `json:"workingDirectory,omitempty"`
	}

	if err := json.Unmarshal(req.Params, &params); err != nil {
		return newInvalidParamsError(req, err)
	}

	if params.Title == "" {
		return newMissingParamError(req, "title")
	}

	// Create session
	session, err := h.app.Sessions.Create(ctx, params.Title, params.WorkingDirectory)
	if err != nil {
		return newApplicationError(req, "Failed to create session: " + err.Error())
	}

	// Optionally set as current
	if params.SetCurrent {
		err = h.app.SetCurrentSession(session.ID)
		if err != nil {
			return newApplicationError(req, "Session created but failed to set as current: " + err.Error())
		}
	}

	result := SessionData{
		ID:               session.ID,
		Title:            session.Title,
		UserMessageCount:      session.UserMessageCount,
		AssistantMessageCount: session.AssistantMessageCount,
		ToolCallCount:         session.ToolCallCount,
		PromptTokens:     session.PromptTokens,
		CompletionTokens: session.CompletionTokens,
		Cost:             session.Cost,
		CreatedAt:        time.Unix(session.CreatedAt, 0),
		WorkingDirectory: session.WorkingDirectory,
	}

	return &QueryResponse{
		Result: result,
		ID:     req.ID,
	}
}

func (h *QueryHandler) handleSessionsFork(ctx context.Context, req *QueryRequest) *QueryResponse {
	var params struct {
		SourceSessionID string `json:"sourceSessionId"`
		MessageIndex    int64  `json:"messageIndex"`
		Title           string `json:"title,omitempty"`
	}

	if err := json.Unmarshal(req.Params, &params); err != nil {
		return newInvalidParamsError(req, err)
	}

	if params.SourceSessionID == "" {
		return newMissingParamError(req, "sourceSessionId")
	}

	if params.MessageIndex <= 0 {
		return newMissingParamError(req, "messageIndex must be > 0")
	}

	// Use a default title if not provided
	title := params.Title
	if title == "" {
		title = "Forked Session"
	}

	// Create the forked session
	newSession, err := h.app.Sessions.Fork(ctx, params.SourceSessionID, title)
	if err != nil {
		return newApplicationError(req, "Failed to fork session: " + err.Error())
	}

	// Copy messages to the new session
	err = h.app.Messages.CopyMessagesToSession(ctx, params.SourceSessionID, newSession.ID, params.MessageIndex)
	if err != nil {
		return newApplicationError(req, "Failed to copy messages: " + err.Error())
	}

	result := SessionData{
		ID:               newSession.ID,
		Title:            newSession.Title,
		UserMessageCount:      newSession.UserMessageCount,
		AssistantMessageCount: newSession.AssistantMessageCount,
		ToolCallCount:         newSession.ToolCallCount,
		PromptTokens:     newSession.PromptTokens,
		CompletionTokens: newSession.CompletionTokens,
		Cost:             newSession.Cost,
		CreatedAt:        time.Unix(newSession.CreatedAt, 0),
		WorkingDirectory: newSession.WorkingDirectory,
	}

	return &QueryResponse{
		Result: result,
		ID:     req.ID,
	}
}

func (h *QueryHandler) handleMCPList(ctx context.Context, req *QueryRequest) *QueryResponse {
	cfg := config.Get()

	var result []MCPServerData

	if len(cfg.MCPServers) == 0 {
		return &QueryResponse{
			Result: result, // Empty array
			ID:     req.ID,
		}
	}

	// Get MCP tools to check connection status and group by server
	// Create temporary manager for informational listing
	tempManager2 := agent.NewMCPClientManager()
	defer tempManager2.Close()
	mcpTools := agent.GetMcpTools(ctx, h.app.Permissions, tempManager2)

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

	return &QueryResponse{
		Result: result,
		ID:     req.ID,
	}
}

func (h *QueryHandler) handleCommandsList(ctx context.Context, req *QueryRequest) *QueryResponse {
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

	return &QueryResponse{
		Result: result,
		ID:     req.ID,
	}
}

func (h *QueryHandler) handleCommandsGet(ctx context.Context, req *QueryRequest) *QueryResponse {
	var params struct {
		Name string `json:"name"`
	}

	if err := json.Unmarshal(req.Params, &params); err != nil {
		return newInvalidParamsError(req, err)
	}

	if params.Name == "" {
		return newMissingParamError(req, "name")
	}

	cmd, exists := h.commandRegistry.GetCommand(params.Name)
	if !exists {
		return newApplicationError(req, "Command not found: " + params.Name)
	}

	builtins := map[string]bool{
		"help": true, "clear": true, "session": true,
		"sessions": true, "tools": true, "mcp": true,
	}

	cmdType := "file"
	if builtins[params.Name] {
		cmdType = "builtin"
	}

	result := CommandData{
		Name:        cmd.Name(),
		Description: cmd.Description(),
		Type:        cmdType,
	}

	return &QueryResponse{
		Result: result,
		ID:     req.ID,
	}
}

func (h *QueryHandler) handleMessagesSend(ctx context.Context, req *QueryRequest) *QueryResponse {
	var params struct {
		SessionID string `json:"sessionId"`
		Content   string `json:"content"`
	}

	if err := json.Unmarshal(req.Params, &params); err != nil {
		return newInvalidParamsError(req, err)
	}

	if params.SessionID == "" {
		return newMissingParamError(req, "sessionId")
	}

	if params.Content == "" {
		return newMissingParamError(req, "content")
	}

	// Check authentication status before processing the message using the centralized function
	authenticated, _, authErr := provider.IsAuthenticated()
	if authErr != nil {
		return newApplicationError(req, fmt.Sprintf("Error checking authentication: %s", authErr.Error()))
	}

	// If not authenticated, show a clear error message
	if !authenticated {
		helpfulMsg := "⚠️ Authentication required. Please use /login command to authenticate with Claude using an API key.\n\n" +
			"To login:\n" +
			"1. Visit https://console.anthropic.com/settings/keys\n" +
			"2. Create an API key\n" +
			"3. Use the /login command to authenticate"

		return &QueryResponse{
			Result: map[string]interface{}{
				"id":       "system-auth-prompt",
				"role":     "assistant",
				"content":  params.Content,
				"response": helpfulMsg,
			},
			ID: req.ID,
		}
	}

	// Set the session as current
	setSessionErr := h.app.SetCurrentSession(params.SessionID)
	if setSessionErr != nil {
		return newApplicationError(req, "Failed to set session: " + setSessionErr.Error())
	}

	// Check if this is a slash command and handle it immediately
	if commands.IsSlashCommand(params.Content) {
		parsed, parseErr := commands.ParseCommand(params.Content)
		if parseErr != nil {
			return newErrorResponse(req, -32602, "Invalid slash command: " + parseErr.Error())
		}

		logging.Info("Executing command", "name", parsed.Name, "args", parsed.Arguments)

		commandResult, execErr := h.commandRegistry.ExecuteCommand(ctx, parsed.Name, parsed.Arguments)
		if execErr != nil {
			logging.Error("Command execution failed", "name", parsed.Name, "error", execErr)

			// Check if it's a "command not found" error
			if strings.Contains(execErr.Error(), "command not found") {
				// List available commands for debugging
				allCommands := h.commandRegistry.GetAllCommands()
				commandNames := getCommandNames(allCommands)
				logging.Info("Available commands", "commands", commandNames)

				return newApplicationError(req, fmt.Sprintf("Command '%s' not found. Available commands: %v", parsed.Name, commandNames))
			}

			return newApplicationError(req, "Command execution failed: " + execErr.Error())
		}

		logging.Info("Command executed successfully", "name", parsed.Name, "result_length", len(commandResult))

		// Return the command result immediately as a message
		return &QueryResponse{
			Result: map[string]interface{}{
				"id":       "cmd-" + parsed.Name,
				"role":     "assistant",
				"content":  params.Content,
				"response": commandResult,
			},
			ID: req.ID,
		}
	}

	// Send message to agent
	done, err := h.app.CoderAgent.Run(ctx, params.SessionID, params.Content)
	if err != nil {
		return newApplicationError(req, "Failed to send message: " + err.Error())
	}

	// Wait for response
	result := <-done

	// Check for processing errors
	if result.Error != nil {
		// Convert error to user-friendly message
		errorMessage := result.Error.Error()

		// Special handling for auth errors
		if strings.Contains(errorMessage, "401") || strings.Contains(errorMessage, "authentication") {
			return &QueryResponse{
				Result: map[string]interface{}{
					"id":       "system-auth-prompt",
					"role":     "assistant",
					"content":  params.Content,
					"response": "⚠️ Authentication required. Please use the /login command to authenticate with Claude API key.",
				},
				ID: req.ID,
			}
		}

		return newApplicationError(req, "Agent processing failed: " + errorMessage)
	}

	// Extract text content from the response message
	response := ""
	if result.Message.Content().String() != "" {
		response = result.Message.Content().String()
	}

	messageData := MessageData{
		ID:       result.Message.ID,
		Role:     "user",
		Content:  params.Content,
		Response: response,
	}

	return &QueryResponse{
		Result: messageData,
		ID:     req.ID,
	}
}

func (h *QueryHandler) handleMessagesHistory(ctx context.Context, req *QueryRequest) *QueryResponse {
	var params struct {
		Limit  int64 `json:"limit,omitempty"`
		Offset int64 `json:"offset,omitempty"`
	}

	if err := json.Unmarshal(req.Params, &params); err != nil {
		return newInvalidParamsError(req, err)
	}

	// Set default limit if not provided
	if params.Limit <= 0 {
		params.Limit = 50
	}

	messages, err := h.app.Messages.ListUserMessageHistory(ctx, params.Limit, params.Offset)
	if err != nil {
		return newApplicationError(req, "Failed to get message history: " + err.Error())
	}

	var result []MessageData
	for _, msg := range messages {
		// Extract tool calls
		toolCalls := msg.ToolCalls()
		toolCallsData := make([]ToolCallData, len(toolCalls))
		for i, tc := range toolCalls {
			toolCallsData[i] = ToolCallData{
				ID:       tc.ID,
				Name:     tc.Name,
				Input:    tc.Input,
				Type:     tc.Type,
				Finished: tc.Finished,
			}
		}

		result = append(result, MessageData{
			ID:        msg.ID,
			SessionID: msg.SessionID,
			Role:      string(msg.Role),
			Content:   msg.Content().String(),
			ToolCalls: toolCallsData,
		})
	}

	return &QueryResponse{
		Result: result,
		ID:     req.ID,
	}
}

func (h *QueryHandler) handleMessagesList(ctx context.Context, req *QueryRequest) *QueryResponse {
	var params struct {
		SessionID string `json:"sessionId"`
	}

	if err := json.Unmarshal(req.Params, &params); err != nil {
		return newInvalidParamsError(req, err)
	}

	if params.SessionID == "" {
		return newMissingParamError(req, "sessionId")
	}

	messages, err := h.app.Messages.List(ctx, params.SessionID)
	if err != nil {
		return newApplicationError(req, "Failed to get messages: " + err.Error())
	}

	var result []MessageData
	for _, msg := range messages {
		// Extract tool calls
		toolCalls := msg.ToolCalls()
		toolCallsData := make([]ToolCallData, len(toolCalls))
		for i, tc := range toolCalls {
			toolCallsData[i] = ToolCallData{
				ID:       tc.ID,
				Name:     tc.Name,
				Input:    tc.Input,
				Type:     tc.Type,
				Finished: tc.Finished,
			}
		}

		result = append(result, MessageData{
			ID:        msg.ID,
			SessionID: msg.SessionID,
			Role:      string(msg.Role),
			Content:   msg.Content().String(),
			ToolCalls: toolCallsData,
		})
	}

	return &QueryResponse{
		Result: result,
		ID:     req.ID,
	}
}

func (h *QueryHandler) handleAgentCancel(ctx context.Context, req *QueryRequest) *QueryResponse {
	var params struct {
		SessionID string `json:"sessionId"`
	}

	if err := json.Unmarshal(req.Params, &params); err != nil {
		return newInvalidParamsError(req, err)
	}

	if params.SessionID == "" {
		return newMissingParamError(req, "sessionId")
	}

	// Cancel the agent processing for this session
	h.app.CoderAgent.Cancel(params.SessionID)

	return &QueryResponse{
		Result: map[string]string{
			"status":    "cancelled",
			"sessionId": params.SessionID,
		},
		ID: req.ID,
	}
}

func (h *QueryHandler) handleSessionsDelete(ctx context.Context, req *QueryRequest) *QueryResponse {
	var params struct {
		ID string `json:"id"`
	}

	if err := json.Unmarshal(req.Params, &params); err != nil {
		return newInvalidParamsError(req, err)
	}

	if params.ID == "" {
		return newMissingParamError(req, "id")
	}

	// Check if this is the current session
	currentSessionID := h.app.GetCurrentSessionID()
	if params.ID == currentSessionID {
		return newApplicationError(req, "Cannot delete the currently active session")
	}

	// Delete the session
	err := h.app.Sessions.Delete(ctx, params.ID)
	if err != nil {
		return newApplicationError(req, "Failed to delete session: " + err.Error())
	}

	return &QueryResponse{
		Result: map[string]string{"message": "Session deleted: " + params.ID},
		ID:     req.ID,
	}
}

func (h *QueryHandler) handlePermissionGrant(ctx context.Context, req *QueryRequest) *QueryResponse {
	var params struct {
		ID string `json:"id"`
	}

	if err := json.Unmarshal(req.Params, &params); err != nil {
		return newInvalidParamsError(req, err)
	}

	if params.ID == "" {
		return newMissingParamError(req, "id")
	}

	// Grant the permission using the existing service
	h.app.Permissions.Grant(permission.PermissionRequest{ID: params.ID})

	return &QueryResponse{
		Result: map[string]string{
			"status":  "granted",
			"id":      params.ID,
			"message": "Permission granted successfully",
		},
		ID: req.ID,
	}
}

func (h *QueryHandler) handlePermissionDeny(ctx context.Context, req *QueryRequest) *QueryResponse {
	var params struct {
		ID string `json:"id"`
	}

	if err := json.Unmarshal(req.Params, &params); err != nil {
		return newInvalidParamsError(req, err)
	}

	if params.ID == "" {
		return newMissingParamError(req, "id")
	}

	// Deny the permission using the existing service
	h.app.Permissions.Deny(permission.PermissionRequest{ID: params.ID})

	return &QueryResponse{
		Result: map[string]string{
			"status":  "denied",
			"id":      params.ID,
			"message": "Permission denied successfully",
		},
		ID: req.ID,
	}
}
