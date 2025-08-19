package commands

import (
	"context"
	"encoding/json"
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
)

// ContextResponse represents the JSON response for the /context command
type ContextResponse struct {
	Model          string               `json:"model"`
	MaxTokens      int64                `json:"maxTokens"`
	TotalTokens    int64                `json:"totalTokens"`
	UsagePercent   float64              `json:"usagePercent"`
	Components     []ComponentBreakdown `json:"components"`
	WarningLevel   string               `json:"warningLevel,omitempty"`
	WarningMessage string               `json:"warningMessage,omitempty"`
}

// ComponentBreakdown represents individual context component usage
type ComponentBreakdown struct {
	Name       string  `json:"name"`
	Tokens     int64   `json:"tokens"`
	Percentage float64 `json:"percentage"`
	IsTotal    bool    `json:"isTotal,omitempty"`
}

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
	Type             string  `json:"type"`
	ID               string  `json:"id"`
	Title            string  `json:"title"`
	MessageCount     int64   `json:"messageCount"`
	TotalTokens      int64   `json:"totalTokens"`
	PromptTokens     int64   `json:"promptTokens"`
	CompletionTokens int64   `json:"completionTokens"`
	Cost             float64 `json:"cost"`
	CreatedAt        int64   `json:"createdAt"`
	UpdatedAt        int64   `json:"updatedAt"`
	ParentSessionID  string  `json:"parentSessionId,omitempty"`
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
	Type           string           `json:"type"`
	CurrentSession string           `json:"currentSession,omitempty"`
	Sessions       []SessionSummary `json:"sessions"`
}

// SessionSummary represents a session summary in the sessions list
type SessionSummary struct {
	ID              string  `json:"id"`
	Title           string  `json:"title"`
	MessageCount    int64   `json:"messageCount"`
	TotalTokens     int64   `json:"totalTokens"`
	Cost            float64 `json:"cost"`
	CreatedAt       int64   `json:"createdAt"`
	UpdatedAt       int64   `json:"updatedAt"`
	ParentSessionID string  `json:"parentSessionId,omitempty"`
	IsCurrent       bool    `json:"isCurrent"`
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
func returnMessage(command, message string) (string, error) {
	response := MessageResponse{
		Type:    "message",
		Message: message,
		Command: command,
	}
	jsonData, _ := json.Marshal(response)
	return string(jsonData), nil
}

// GetBuiltinCommands returns all built-in commands
func GetBuiltinCommands(registry *Registry, app *app.App) map[string]Command {
	return map[string]Command{
		"help": &BuiltinCommand{
			name:        "help",
			description: "Show available commands",
			handler:     createHelpHandler(registry),
		},
		"clear": &BuiltinCommand{
			name:        "clear",
			description: "Start new session",
			handler:     createClearHandler(app),
		},
		"session": &BuiltinCommand{
			name:        "session",
			description: "Show session information or switch sessions",
			handler:     createSessionHandler(app),
		},
		"sessions": &BuiltinCommand{
			name:        "sessions",
			description: "List all available sessions",
			handler:     createSessionsHandler(app),
		},
		"mcp": &BuiltinCommand{
			name:        "mcp",
			description: "List configured MCP servers",
			handler:     createMcpHandler(),
		},
		"context": &BuiltinCommand{
			name:        "context",
			description: "Show context usage breakdown with percentages",
			handler:     createContextHandler(app),
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

func createClearHandler(app *app.App) func(ctx context.Context, args string) (string, error) {
	return func(ctx context.Context, args string) (string, error) {
		// Create a new session with a default title
		title := "New Session"
		workingDirectory := ""

		// Get current session for working directory context
		if currentSession, err := app.GetCurrentSession(ctx); err == nil && currentSession != nil {
			workingDirectory = currentSession.WorkingDirectory
		}

		// Create the new session
		session, err := app.Sessions.Create(ctx, title, workingDirectory)
		if err != nil {
			return returnError("clear", fmt.Sprintf("Failed to create new session: %v", err))
		}

		// Set the new session as current
		if err := app.SetCurrentSession(session.ID); err != nil {
			return returnError("clear", fmt.Sprintf("Failed to set new session as current: %v", err))
		}

		// Return success message with session info
		return returnMessage("clear", fmt.Sprintf("Started new session: %s (ID: %s)", session.Title, session.ID[:8]))
	}
}

func createSessionHandler(app *app.App) func(ctx context.Context, args string) (string, error) {
	return func(ctx context.Context, args string) (string, error) {
		args = strings.TrimSpace(args)
		if args == "" {
			// Show current session info
			currentSession, err := app.GetCurrentSession(ctx)
			if err != nil {
				return returnError("session", fmt.Sprintf("Error retrieving current session: %v", err))
			}

			if currentSession == nil {
				return returnMessage("session", "No active session. Use /sessions to list available sessions.")
			}

			// Create structured response
			response := SessionResponse{
				Type:             "session",
				ID:               currentSession.ID,
				Title:            currentSession.Title,
				MessageCount:     currentSession.MessageCount,
				TotalTokens:      currentSession.PromptTokens + currentSession.CompletionTokens,
				PromptTokens:     currentSession.PromptTokens,
				CompletionTokens: currentSession.CompletionTokens,
				Cost:             currentSession.Cost,
				CreatedAt:        currentSession.CreatedAt,
				UpdatedAt:        currentSession.UpdatedAt,
				ParentSessionID:  currentSession.ParentSessionID,
			}

			// Convert to JSON
			jsonData, err := json.Marshal(response)
			if err != nil {
				return returnError("session", fmt.Sprintf("Error marshaling session data: %v", err))
			}

			return string(jsonData), nil
		} else {
			// Switch to specific session
			return returnMessage("session", fmt.Sprintf("Session switching to '%s' is available via the HTTP API.", args))
		}
	}
}

func createSessionsHandler(app *app.App) func(ctx context.Context, args string) (string, error) {
	return func(ctx context.Context, args string) (string, error) {
		// Get all sessions from the database
		sessions, err := app.Sessions.List(ctx)
		if err != nil {
			return returnError("sessions", fmt.Sprintf("Error retrieving sessions: %v", err))
		}

		// Get current session ID for comparison
		currentSessionID := app.GetCurrentSessionID()

		// Build session summaries
		var sessionSummaries []SessionSummary
		for _, session := range sessions {
			sessionSummaries = append(sessionSummaries, SessionSummary{
				ID:              session.ID,
				Title:           session.Title,
				MessageCount:    session.MessageCount,
				TotalTokens:     session.PromptTokens + session.CompletionTokens,
				Cost:            session.Cost,
				CreatedAt:       session.CreatedAt,
				UpdatedAt:       session.UpdatedAt,
				ParentSessionID: session.ParentSessionID,
				IsCurrent:       session.ID == currentSessionID,
			})
		}

		// Create structured response
		response := SessionsResponse{
			Type:           "sessions",
			CurrentSession: currentSessionID,
			Sessions:       sessionSummaries,
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
			return returnMessage("mcp", "No MCP servers configured.\n\nTo configure MCP servers, add them to your configuration file under 'mcpServers'.")
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
		defer tempManager.Close()
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
			tools := serverTools[name]

			// Determine connection status
			var statusText string
			connected := len(tools) > 0
			if connected {
				statusText = "connected"
			} else {
				statusText = "failed"
			}

			// Build tool list
			var mcpTools []McpTool
			if len(tools) > 0 {
				// Sort tools by name for consistent output
				sort.Slice(tools, func(i, j int) bool {
					return tools[i].Info().Name < tools[j].Info().Name
				})

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
				ToolCount: len(tools),
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

func createContextHandler(app *app.App) func(ctx context.Context, args string) (string, error) {
	return func(ctx context.Context, args string) (string, error) {
		currentSession, err := app.GetCurrentSession(ctx)
		if err != nil {
			return returnError("context", fmt.Sprintf("Error retrieving current session: %v", err))
		}

		if currentSession == nil {
			return returnMessage("context", "No active session. Use /sessions to list available sessions.")
		}

		// Get current model's context window from agent
		currentModel := app.CoderAgent.Model()
		maxContextTokens := int64(currentModel.ContextWindow)

		// System prompt estimation (rough approximation)
		systemPromptTokens := int64(5000) // Typical system prompt size
		systemPromptPercent := float64(systemPromptTokens) / float64(maxContextTokens) * 100

		// Tool descriptions estimation
		toolTokens := int64(15000) // Typical tool descriptions size
		toolPercent := float64(toolTokens) / float64(maxContextTokens) * 100

		// Calculate conversation tokens (excluding system overhead)
		conversationTokens := currentSession.PromptTokens + currentSession.CompletionTokens

		// User and assistant message breakdown
		userTokens := currentSession.PromptTokens
		userPercent := float64(userTokens) / float64(maxContextTokens) * 100

		assistantTokens := currentSession.CompletionTokens
		assistantPercent := float64(assistantTokens) / float64(maxContextTokens) * 100

		// Calculate total tokens including baseline system context
		baselineTokens := systemPromptTokens + toolTokens
		totalTokens := baselineTokens + conversationTokens
		contextUsagePercent := float64(totalTokens) / float64(maxContextTokens) * 100

		// Determine warning level
		warningLevel := "none"
		warningMessage := ""
		if contextUsagePercent > 80 {
			warningLevel = "high"
			warningMessage = "Context usage above 80% - consider starting a new session"
		} else if contextUsagePercent > 60 {
			warningLevel = "medium"
			warningMessage = "Context usage above 60% - monitor usage"
		}

		// Create structured response
		response := ContextResponse{
			Model:          currentModel.Name,
			MaxTokens:      maxContextTokens,
			TotalTokens:    totalTokens,
			UsagePercent:   contextUsagePercent,
			WarningLevel:   warningLevel,
			WarningMessage: warningMessage,
			Components: []ComponentBreakdown{
				{
					Name:       "System Prompt",
					Tokens:     systemPromptTokens,
					Percentage: systemPromptPercent,
				},
				{
					Name:       "Tool Descriptions",
					Tokens:     toolTokens,
					Percentage: toolPercent,
				},
				{
					Name:       "User Messages",
					Tokens:     userTokens,
					Percentage: userPercent,
				},
				{
					Name:       "Assistant Responses",
					Tokens:     assistantTokens,
					Percentage: assistantPercent,
				},
				{
					Name:       "Total",
					Tokens:     totalTokens,
					Percentage: contextUsagePercent,
					IsTotal:    true,
				},
			},
		}

		// Convert to JSON
		jsonData, err := json.Marshal(response)
		if err != nil {
			return returnError("context", fmt.Sprintf("Error marshaling context data: %v", err))
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
		if err != nil {
			return returnError("status", fmt.Sprintf("Error checking credentials: %v", err))
		}

		// Check if API key is set in environment
		hasAPIKey := os.Getenv("ANTHROPIC_API_KEY") != ""

		response := AuthStatusResponse{
			Type:     "auth_status",
			Provider: "anthropic",
		}

		// OAuth takes precedence over API key
		if creds != nil && !creds.IsTokenExpired() {
			response.Status = "authenticated"
			response.ExpiresIn = (creds.ExpiresAt - time.Now().Unix()) / 60 // minutes
			response.Message = "✅ Authenticated with Claude Code OAuth"
		} else if hasAPIKey {
			response.Status = "authenticated"
			response.ExpiresIn = 0 // API keys don't expire
			response.Message = "✅ Authenticated with Anthropic API Key"
		} else {
			response.Status = "not_authenticated"
			response.ExpiresIn = 0
			if creds != nil && creds.IsTokenExpired() {
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
		if err == nil && existingCreds != nil && !existingCreds.IsTokenExpired() {
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
			return string(jsonData), nil
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
		return string(jsonData), nil
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
		creds, err := storage.GetOAuthCredentials("anthropic")
		hasOAuth := err == nil && creds != nil

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
			os.Unsetenv("ANTHROPIC_API_KEY")
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
			return string(jsonData), nil
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
