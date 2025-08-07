package commands

import (
	"context"
	"encoding/json"
	"fmt"
	"sort"
	"strings"

	"mix/internal/app"
	"mix/internal/config"
	"mix/internal/llm/agent"
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
