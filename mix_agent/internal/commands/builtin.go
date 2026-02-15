package commands

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"sort"
	"strings"
	"time"

	"mix/internal/app"
	"mix/internal/config"
	"mix/internal/llm/agent"
	"mix/internal/llm/provider"
	"mix/internal/llm/tools"
	session2 "mix/internal/session"
)

// HelpResponse represents the JSON response for the /help command
type HelpResponse struct {
	Type     string        `json:"type"`
	Commands []HelpCommand `json:"commands"`
}

// HelpCommand represents a command in the help response
type HelpCommand struct {
	Name        string `json:"name"`
	Description string `json:"description"`
	Usage       string `json:"usage"`
}

// SessionResponse represents the JSON response for the /session command
type SessionResponse struct {
	Type                  string  `json:"type"`
	ID                    string  `json:"id"`
	Title                 string  `json:"title"`
	UserMessageCount      int64   `json:"userMessageCount"`
	AssistantMessageCount int64   `json:"assistantMessageCount"`
	ToolCallCount         int64   `json:"toolCallCount"`
	TotalTokens           int64   `json:"totalTokens"`
	PromptTokens          int64   `json:"promptTokens"`
	CompletionTokens      int64   `json:"completionTokens"`
	Cost                  float64 `json:"cost"`
	CreatedAt             int64   `json:"createdAt"`
	UpdatedAt             int64   `json:"updatedAt"`
	ParentSessionID       string  `json:"parentSessionId,omitempty"`
}

// McpResponse represents the JSON response for the /mcp command
type McpResponse struct {
	Type    string      `json:"type"`
	Servers []McpServer `json:"servers"`
}

// McpServer represents an MCP server in the response
type McpServer struct {
	Name      string    `json:"name"`
	Status    string    `json:"status"`
	Connected bool      `json:"connected"`
	ToolCount int       `json:"toolCount"`
	Tools     []McpTool `json:"tools"`
}

// McpTool represents a tool available from an MCP server
type McpTool struct {
	Name        string `json:"name"`
	Description string `json:"description"`
}

// SessionsResponse represents the JSON response for the /sessions command
type SessionsResponse struct {
	Type     string           `json:"type"`
	Sessions []SessionSummary `json:"sessions"`
}

// SessionSummary represents a session summary in the sessions list
type SessionSummary struct {
	ID                    string  `json:"id"`
	Title                 string  `json:"title"`
	UserMessageCount      int64   `json:"userMessageCount"`
	AssistantMessageCount int64   `json:"assistantMessageCount"`
	ToolCallCount         int64   `json:"toolCallCount"`
	TotalTokens           int64   `json:"totalTokens"`
	Cost                  float64 `json:"cost"`
	CreatedAt             int64   `json:"createdAt"`
	UpdatedAt             int64   `json:"updatedAt"`
	ParentSessionID       string  `json:"parentSessionId,omitempty"`
}

// ErrorResponse represents error responses from commands
type ErrorResponse struct {
	Type    string `json:"type"`
	Error   string `json:"error"`
	Command string `json:"command,omitempty"`
}

// MessageResponse represents informational messages from commands
type MessageResponse struct {
	Type    string `json:"type"`
	Message string `json:"message"`
	Command string `json:"command,omitempty"`
}

// AuthStatusResponse represents authentication status
type AuthStatusResponse struct {
	Type      string `json:"type"`
	Status    string `json:"status"`    // "authenticated" | "not_authenticated"
	Provider  string `json:"provider"`  // "anthropic"
	ExpiresIn int64  `json:"expiresIn"` // minutes until expiry
	Message   string `json:"message"`
}

// AuthLoginResponse represents login flow responses
type AuthLoginResponse struct {
	Type    string `json:"type"`
	Status  string `json:"status"` // "success" | "pending" | "error"
	Message string `json:"message"`
	AuthURL string `json:"authUrl,omitempty"` // for OAuth flow
	Step    string `json:"step,omitempty"`    // current step in flow
}

// SessionSwitchResponse represents responses that create a session and instruct client to switch
type SessionSwitchResponse struct {
	Type         string `json:"type"`
	Action       string `json:"action"` // "switch"
	Message      string `json:"message"`
	Command      string `json:"command,omitempty"`
	SessionID    string `json:"sessionId"`
	SessionTitle string `json:"sessionTitle"`
}

// BuiltinCommand represents a built-in command
type BuiltinCommand struct {
	name        string
	description string
	handler     func(ctx context.Context, args string) (string, error)
}

func (c *BuiltinCommand) Name() string {
	return c.name
}

func (c *BuiltinCommand) Description() string {
	return c.description
}

func (c *BuiltinCommand) Execute(ctx context.Context, args string) (string, error) {
	return c.handler(ctx, args)
}

// Helper functions for structured responses

// returnError creates a structured error response
func returnError(command, errorMsg string) (string, error) {
	response := ErrorResponse{
		Type:    "error",
		Error:   errorMsg,
		Command: command,
	}
	jsonData, _ := json.Marshal(response)
	return string(jsonData), nil
}

// returnMessage creates a structured informational message response
func returnMessage(command, message string) string {
	response := MessageResponse{
		Type:    "message",
		Message: message,
		Command: command,
	}
	jsonData, _ := json.Marshal(response)
	return string(jsonData)
}

// returnSessionSwitch creates a structured session switch response
func returnSessionSwitch(command, message, sessionID, sessionTitle string) (string, error) {
	response := SessionSwitchResponse{
		Type:         "session_switch",
		Action:       "switch",
		Message:      message,
		Command:      command,
		SessionID:    sessionID,
		SessionTitle: sessionTitle,
	}
	jsonData, _ := json.Marshal(response)
	return string(jsonData), nil
}

// GetBuiltinCommands returns all built-in commands
func GetBuiltinCommands(registry *Registry, application *app.App) map[string]Command {
	return map[string]Command{
		"help": &BuiltinCommand{
			name:        "help",
			description: "Show available commands",
			handler:     createHelpHandler(registry),
		},
		"clear": &BuiltinCommand{
			name:        "clear",
			description: "Start new session",
			handler:     createClearHandler(application),
		},
		"sessions": &BuiltinCommand{
			name:        "sessions",
			description: "List all available sessions",
			handler:     createSessionsHandler(application),
		},
		"mcp": &BuiltinCommand{
			name:        "mcp",
			description: "List configured MCP servers",
			handler:     createMcpHandler(),
		},
		"login": &BuiltinCommand{
			name:        "login",
			description: "Authenticate with Claude Code OAuth",
			handler:     createLoginHandler(),
		},
		"logout": &BuiltinCommand{
			name:        "logout",
			description: "Sign out from Claude Code",
			handler:     createLogoutHandler(),
		},
		"status": &BuiltinCommand{
			name:        "status",
			description: "Check Claude Code authentication status",
			handler:     createAuthStatusHandler(),
		},
		"auth-code": &BuiltinCommand{
			name:        "auth-code",
			description: "Exchange authorization code for OAuth tokens",
			handler:     createAuthCodeHandler(),
		},
	}
}

func createHelpHandler(registry *Registry) func(ctx context.Context, args string) (string, error) {
	return func(ctx context.Context, args string) (string, error) {
		// Get all commands from registry
		commands := registry.GetAllCommands()

		// Build commands slice
		var helpCommands []HelpCommand
		for name, cmd := range commands {
			helpCommands = append(helpCommands, HelpCommand{
				Name:        name,
				Description: cmd.Description(),
				Usage:       fmt.Sprintf("/%s", name),
			})
		}

		// Sort commands alphabetically by name
		sort.Slice(helpCommands, func(i, j int) bool {
			return helpCommands[i].Name < helpCommands[j].Name
		})

		// Create structured response
		response := HelpResponse{
			Type:     "help",
			Commands: helpCommands,
		}

		// Convert to JSON
		jsonData, err := json.Marshal(response)
		if err != nil {
			return returnError("help", fmt.Sprintf("Error marshaling help data: %v", err))
		}

		return string(jsonData), nil
	}
}

func createClearHandler(application *app.App) func(ctx context.Context, args string) (string, error) {
	return func(ctx context.Context, args string) (string, error) {
		// Extract session ID from context - required for clear command
		sessionID, ok := ctx.Value(tools.SessionIDContextKey).(string)
		if !ok || sessionID == "" {
			return returnError("clear", "Session context required")
		}

		// Create new session
		session, err := application.Sessions.Create(ctx, "New Session", "", "default", session2.SessionTypeMain, "", "", "", "local-browser-service", "")
		if err != nil {
			return returnError("clear", fmt.Sprintf("Failed to create new session: %v", err))
		}

		// Return session switch response
		return returnSessionSwitch("clear", fmt.Sprintf("Started new session: %s", session.Title), session.ID, session.Title)
	}
}

func createSessionsHandler(application *app.App) func(ctx context.Context, args string) (string, error) {
	return func(ctx context.Context, args string) (string, error) {
		// Get all sessions from the database
		sessions, err := application.Sessions.List(ctx)
		if err != nil {
			return returnError("sessions", fmt.Sprintf("Error retrieving sessions: %v", err))
		}

		// Build session summaries
		var sessionSummaries []SessionSummary
		for i := range sessions {
			sessionSummaries = append(sessionSummaries, SessionSummary{
				ID:                    sessions[i].ID,
				Title:                 sessions[i].Title,
				UserMessageCount:      sessions[i].UserMessageCount,
				AssistantMessageCount: sessions[i].AssistantMessageCount,
				ToolCallCount:         sessions[i].ToolCallCount,
				TotalTokens:           sessions[i].PromptTokens + sessions[i].CompletionTokens,
				Cost:                  sessions[i].Cost,
				CreatedAt:             sessions[i].CreatedAt,
				UpdatedAt:             sessions[i].UpdatedAt,
				ParentSessionID:       sessions[i].ParentSessionID,
			})
		}

		// Create structured response
		response := SessionsResponse{
			Type:     "sessions",
			Sessions: sessionSummaries,
		}

		// Convert to JSON
		jsonData, err := json.Marshal(response)
		if err != nil {
			return returnError("sessions", fmt.Sprintf("Error marshaling sessions data: %v", err))
		}

		return string(jsonData), nil
	}
}

func createMcpHandler() func(ctx context.Context, args string) (string, error) {
	return func(ctx context.Context, args string) (string, error) {
		cfg := config.Get()

		if len(cfg.MCPServers) == 0 {
			return returnMessage("mcp", "No MCP servers configured.\n\nTo configure MCP servers, add them to your configuration file under 'mcpServers'."), nil
		}

		// Sort server names for consistent output
		var serverNames []string
		for name := range cfg.MCPServers {
			serverNames = append(serverNames, name)
		}
		sort.Strings(serverNames)

		// Get MCP tools to check connection status and group by server
		// Create temporary manager for informational listing
		tempManager := agent.NewMCPClientManager()
		defer tempManager.Close() // No error returned
		mcpTools := agent.GetMcpTools(ctx, nil, tempManager)

		// Group tools by server name
		serverTools := make(map[string][]tools.BaseTool)
		for _, tool := range mcpTools {
			if toolInfo := tool.Info(); strings.Contains(toolInfo.Name, "_") {
				serverName := strings.Split(toolInfo.Name, "_")[0]
				serverTools[serverName] = append(serverTools[serverName], tool)
			}
		}

		// Build server data
		var servers []McpServer
		for _, name := range serverNames {
			serverToolList := serverTools[name]

			// Determine connection status
			var statusText string
			connected := len(serverToolList) > 0
			if connected {
				statusText = "connected"
			} else {
				statusText = "failed"
			}

			// Build tool list
			var mcpTools []McpTool
			if len(serverToolList) > 0 {
				// Sort tools by name for consistent output
				sort.Slice(serverToolList, func(i, j int) bool {
					return serverToolList[i].Info().Name < serverToolList[j].Info().Name
				})

				for _, tool := range serverToolList {
					info := tool.Info()
					// Remove server prefix from tool name for cleaner display
					toolName := info.Name
					if strings.Contains(toolName, "_") {
						parts := strings.SplitN(toolName, "_", 2)
						if len(parts) > 1 {
							toolName = parts[1]
						}
					}
					mcpTools = append(mcpTools, McpTool{
						Name:        toolName,
						Description: info.Description,
					})
				}
			}

			servers = append(servers, McpServer{
				Name:      name,
				Status:    statusText,
				Connected: connected,
				ToolCount: len(serverToolList),
				Tools:     mcpTools,
			})
		}

		// Create structured response
		response := McpResponse{
			Type:    "mcp",
			Servers: servers,
		}

		// Convert to JSON
		jsonData, err := json.Marshal(response)
		if err != nil {
			return returnError("mcp", fmt.Sprintf("Error marshaling MCP data: %v", err))
		}

		return string(jsonData), nil
	}
}

// Authentication command handlers

func createAuthStatusHandler() func(ctx context.Context, args string) (string, error) {
	return func(ctx context.Context, args string) (string, error) {
		storage, err := provider.NewCredentialStorage()
		if err != nil {
			return returnError("status", fmt.Sprintf("Failed to initialize credential storage: %v", err))
		}

		// Check Anthropic OAuth credentials
		creds, err := storage.GetOAuthCredentials("anthropic")
		if err != nil && !errors.Is(err, provider.ErrOAuthCredentialNotFound) {
			return returnError("status", fmt.Sprintf("Error checking credentials: %v", err))
		}

		// Check if API key is set in environment
		hasAPIKey := os.Getenv("ANTHROPIC_API_KEY") != ""

		response := AuthStatusResponse{
			Type:     "auth_status",
			Provider: "anthropic",
		}

		// OAuth takes precedence over API key
		switch {
		case err == nil && !creds.IsTokenExpired():
			response.Status = "authenticated"
			response.ExpiresIn = (creds.ExpiresAt - time.Now().Unix()) / 60 // minutes
			response.Message = "✅ Authenticated with Claude Code OAuth"
		case hasAPIKey:
			response.Status = "authenticated"
			response.ExpiresIn = 0 // API keys don't expire
			response.Message = "✅ Authenticated with Anthropic API Key"
		default:
			response.Status = "not_authenticated"
			response.ExpiresIn = 0
			if err == nil && creds.IsTokenExpired() {
				response.Message = "❌ Token expired. Please login again."
			} else {
				response.Message = "❌ Not authenticated. Use /login to authenticate."
			}
		}

		jsonData, err := json.Marshal(response)
		if err != nil {
			return returnError("status", fmt.Sprintf("Error marshaling status data: %v", err))
		}

		return string(jsonData), nil
	}
}

func createLoginHandler() func(ctx context.Context, args string) (string, error) {
	return func(ctx context.Context, args string) (string, error) {
		// Check if already authenticated
		storage, err := provider.NewCredentialStorage()
		if err != nil {
			return returnError("login", fmt.Sprintf("Failed to initialize credential storage: %v", err))
		}

		existingCreds, err := storage.GetOAuthCredentials("anthropic")
		switch {
		case errors.Is(err, provider.ErrOAuthCredentialNotFound):
			// Not authenticated, continue with login flow
		case err != nil:
			return returnError("login", fmt.Sprintf("Failed to check existing credentials: %v", err))
		case !existingCreds.IsTokenExpired():
			response := AuthLoginResponse{
				Type:    "auth_login",
				Status:  "success",
				Message: "✅ Already authenticated with Claude Code OAuth!",
			}
			jsonData, _ := json.Marshal(response)
			return string(jsonData), nil
		}

		// Check if API key is set in environment
		if apiKey := os.Getenv("ANTHROPIC_API_KEY"); apiKey != "" {
			response := AuthLoginResponse{
				Type:    "auth_login",
				Status:  "success",
				Step:    "api_key",
				Message: "✅ Using Anthropic API key from environment variables. OAuth not needed.",
			}
			jsonData, _ := json.Marshal(response)
			return string(jsonData), nil
		}

		// Check if user provided authorization code as argument
		args = strings.TrimSpace(args)
		if args != "" {
			// Handle authorization code exchange
			return handleAuthCodeExchange(args, storage)
		}

		// Create OAuth flow and initiate login
		oauthFlow, err := provider.NewOAuthFlow("")
		if err != nil {
			return returnError("login", fmt.Sprintf("Failed to create OAuth flow: %v", err))
		}

		authURL := oauthFlow.GetAuthorizationURL()

		// Try to open browser automatically
		if err := oauthFlow.OpenBrowser(); err != nil {
			// If browser opening fails, just provide the URL
			response := AuthLoginResponse{
				Type:    "auth_login",
				Status:  "pending",
				AuthURL: authURL,
				Step:    "authorization",
				Message: "🔐 Failed to open browser automatically. Please manually visit the URL above and complete OAuth authentication. Then run: /login <authorization_code>",
			}
			jsonData, _ := json.Marshal(response)
			return string(jsonData), fmt.Errorf("failed to open browser: %w", err)
		}

		response := AuthLoginResponse{
			Type:    "auth_login",
			Status:  "pending",
			AuthURL: authURL,
			Step:    "authorization",
			Message: "🔐 Browser opened for authentication. Complete OAuth in your browser, then copy the authorization code and paste it.",
		}

		jsonData, err := json.Marshal(response)
		if err != nil {
			return returnError("login", fmt.Sprintf("Error marshaling login data: %v", err))
		}

		return string(jsonData), nil
	}
}

// handleAuthCodeExchange handles the authorization code exchange for tokens
func handleAuthCodeExchange(authCode string, storage *provider.CredentialStorage) (string, error) {
	// Create new OAuth flow for token exchange
	oauthFlow, err := provider.NewOAuthFlow("")
	if err != nil {
		return returnError("login", fmt.Sprintf("Failed to create OAuth flow: %v", err))
	}

	// Exchange authorization code for tokens
	creds, err := oauthFlow.ExchangeCodeForTokens(authCode)
	if err != nil {
		// Check for errors, but suggest API key as an alternative for all OAuth exchange failures
		response := AuthLoginResponse{
			Type:    "auth_login",
			Status:  "error",
			Step:    "manual_api_key",
			Message: "OAuth flow could not be completed automatically due to Cloudflare protection. \n\nPlease use an API key instead:\n\n1. Visit: https://console.anthropic.com/settings/keys\n2. Create a new API key\n3. Set the environment variable: export ANTHROPIC_API_KEY=your_api_key\n4. Restart the application\n\nThis will be fixed in a future update.",
		}
		jsonData, _ := json.Marshal(response)
		return string(jsonData), fmt.Errorf("oauth token exchange failed: %w", err)
	}

	// Store the credentials
	err = storage.StoreOAuthCredentials("anthropic", creds.AccessToken, creds.RefreshToken, creds.ExpiresAt, creds.ClientID)
	if err != nil {
		return returnError("login", fmt.Sprintf("Failed to store credentials: %v", err))
	}

	response := AuthLoginResponse{
		Type:    "auth_login",
		Status:  "success",
		Step:    "completed",
		Message: "✅ Successfully authenticated with Claude Code OAuth!",
	}

	jsonData, err := json.Marshal(response)
	if err != nil {
		return returnError("login", fmt.Sprintf("Error marshaling success response: %v", err))
	}

	return string(jsonData), nil
}

func createLogoutHandler() func(ctx context.Context, args string) (string, error) {
	return func(ctx context.Context, args string) (string, error) {
		storage, err := provider.NewCredentialStorage()
		if err != nil {
			return returnError("logout", fmt.Sprintf("Failed to initialize credential storage: %v", err))
		}

		// Check if authenticated with OAuth
		_, err = storage.GetOAuthCredentials("anthropic")
		hasOAuth := err == nil

		// Check if API key is set in environment
		hasAPIKey := os.Getenv("ANTHROPIC_API_KEY") != ""

		// If neither authentication method is active, we're already logged out
		if !hasOAuth && !hasAPIKey {
			response := AuthStatusResponse{
				Type:     "auth_status",
				Status:   "not_authenticated",
				Provider: "anthropic",
				Message:  "❌ Already logged out",
			}
			jsonData, _ := json.Marshal(response)
			return string(jsonData), nil
		}

		// Clear OAuth credentials if present
		if hasOAuth {
			err = storage.ClearOAuthCredentials("anthropic")
			if err != nil {
				return returnError("logout", fmt.Sprintf("Failed to clear credentials: %v", err))
			}
		}

		// Clear API key from environment if present
		if hasAPIKey {
			_ = os.Unsetenv("ANTHROPIC_API_KEY")
		}

		response := AuthStatusResponse{
			Type:     "auth_status",
			Status:   "not_authenticated",
			Provider: "anthropic",
			Message:  "✅ Successfully logged out from Claude Code",
		}

		jsonData, err := json.Marshal(response)
		if err != nil {
			return returnError("logout", fmt.Sprintf("Error marshaling logout data: %v", err))
		}

		return string(jsonData), nil
	}
}

func createAuthCodeHandler() func(ctx context.Context, args string) (string, error) {
	return func(ctx context.Context, args string) (string, error) {
		authCode := strings.TrimSpace(args)
		if authCode == "" {
			return returnError("auth-code", "Authorization code is required. Usage: /auth-code <code#state>")
		}

		// Check if there's a '/login ' prefix and remove it - this happens when users copy the whole command
		if strings.HasPrefix(strings.ToLower(authCode), "/login ") {
			authCode = strings.TrimSpace(authCode[7:])
		}

		storage, err := provider.NewCredentialStorage()
		if err != nil {
			return returnError("auth-code", fmt.Sprintf("Failed to initialize credential storage: %v", err))
		}

		// Create OAuth flow (we need this to exchange the code)
		oauthFlow, err := provider.NewOAuthFlow("")
		if err != nil {
			return returnError("auth-code", fmt.Sprintf("Failed to create OAuth flow: %v", err))
		}

		// Exchange the authorization code for tokens
		credentials, err := oauthFlow.ExchangeCodeForTokens(authCode)
		if err != nil {
			// For Cloudflare protection or other errors, guide the user to use API key
			response := AuthLoginResponse{
				Type:    "auth_login",
				Status:  "error",
				Step:    "manual_api_key",
				Message: "OAuth flow could not be completed automatically due to Cloudflare protection. \n\nPlease use an API key instead:\n\n1. Visit: https://console.anthropic.com/settings/keys\n2. Create a new API key\n3. Set the environment variable: export ANTHROPIC_API_KEY=your_api_key\n4. Restart the application\n\nThis will be fixed in a future update.",
			}
			jsonData, _ := json.Marshal(response)
			return string(jsonData), fmt.Errorf("oauth token exchange failed: %w", err)
		}

		// Store the credentials
		err = storage.StoreOAuthCredentials("anthropic", credentials.AccessToken, credentials.RefreshToken, credentials.ExpiresAt, credentials.ClientID)
		if err != nil {
			return returnError("auth-code", fmt.Sprintf("Failed to store credentials: %v", err))
		}

		response := AuthLoginResponse{
			Type:    "auth_login",
			Status:  "success",
			Message: "✅ Successfully authenticated with Claude Code OAuth! You can now use the application.",
		}

		jsonData, err := json.Marshal(response)
		if err != nil {
			return returnError("auth-code", fmt.Sprintf("Error marshaling success response: %v", err))
		}

		return string(jsonData), nil
	}
}
