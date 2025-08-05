package api

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"sort"
	"strings"
	"time"

	"mix/internal/app"
	"mix/internal/commands"
	"mix/internal/config"
	"mix/internal/llm/agent"
	"mix/internal/llm/tools"
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
	ID               string    `json:"id"`
	Title            string    `json:"title"`
	MessageCount     int64     `json:"messageCount"`
	PromptTokens     int64     `json:"promptTokens"`
	CompletionTokens int64     `json:"completionTokens"`
	Cost             float64   `json:"cost"`
	CreatedAt        time.Time `json:"createdAt"`
	WorkingDirectory string    `json:"workingDirectory,omitempty"`
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

type MessageData struct {
	ID        string `json:"id"`
	SessionID string `json:"sessionId"`
	Role      string `json:"role"`
	Content   string `json:"content"`
	Response  string `json:"response,omitempty"`
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
		log.Printf("ERROR: Failed to load commands: %v", err)
		// Continue with empty registry - API will return proper errors
	} else {
		// Log successful command loading
		allCommands := registry.GetAllCommands()
		log.Printf("Successfully loaded %d commands: %v", len(allCommands), getCommandNames(allCommands))
	}

	return &QueryHandler{
		app:             app,
		commandRegistry: registry,
	}
}

// GetApp returns the app instance for external access
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
	case "messages.send":
		return h.handleMessagesSend(ctx, req)
	case "messages.history":
		return h.handleMessagesHistory(ctx, req)
	case "mcp.list":
		return h.handleMCPList(ctx, req)
	case "commands.list":
		return h.handleCommandsList(ctx, req)
	case "commands.get":
		return h.handleCommandsGet(ctx, req)
	default:
		return &QueryResponse{
			Error: &QueryError{
				Code:    -32601,
				Message: "Method not found: " + req.Method,
			},
			ID: req.ID,
		}
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
	return &QueryResponse{
		Error: &QueryError{
			Code:    -32602,
			Message: "Invalid query type: " + queryType + ". Supported: " + strings.Join(supportedTypes, ", "),
		},
	}
}

// GetSupportedQueryTypes returns all supported query types
func (h *QueryHandler) GetSupportedQueryTypes() []string {
	return []string{"sessions", "tools", "mcp", "commands"}
}

func (h *QueryHandler) handleSessionsList(ctx context.Context, req *QueryRequest) *QueryResponse {
	sessions, err := h.app.Sessions.List(ctx)
	if err != nil {
		return &QueryResponse{
			Error: &QueryError{
				Code:    -32000,
				Message: "Failed to list sessions: " + err.Error(),
			},
			ID: req.ID,
		}
	}

	var result []SessionData
	for _, s := range sessions {
		result = append(result, SessionData{
			ID:               s.ID,
			Title:            s.Title,
			MessageCount:     s.MessageCount,
			PromptTokens:     s.PromptTokens,
			CompletionTokens: s.CompletionTokens,
			Cost:             s.Cost,
			CreatedAt:        time.Unix(s.CreatedAt, 0),
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
		return &QueryResponse{
			Error: &QueryError{
				Code:    -32602,
				Message: "Invalid params: " + err.Error(),
			},
			ID: req.ID,
		}
	}

	if params.ID == "" {
		return &QueryResponse{
			Error: &QueryError{
				Code:    -32602,
				Message: "Missing required parameter: id",
			},
			ID: req.ID,
		}
	}

	session, err := h.app.Sessions.Get(ctx, params.ID)
	if err != nil {
		return &QueryResponse{
			Error: &QueryError{
				Code:    -32000,
				Message: "Failed to get session: " + err.Error(),
			},
			ID: req.ID,
		}
	}

	result := SessionData{
		ID:               session.ID,
		Title:            session.Title,
		MessageCount:     session.MessageCount,
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
		return &QueryResponse{
			Error: &QueryError{
				Code:    -32000,
				Message: "Failed to get current session: " + err.Error(),
			},
			ID: req.ID,
		}
	}

	if currentSession == nil {
		return &QueryResponse{
			Error: &QueryError{
				Code:    -32000,
				Message: "No current session selected",
			},
			ID: req.ID,
		}
	}

	result := SessionData{
		ID:               currentSession.ID,
		Title:            currentSession.Title,
		MessageCount:     currentSession.MessageCount,
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
		return &QueryResponse{
			Error: &QueryError{
				Code:    -32602,
				Message: "Invalid params: " + err.Error(),
			},
			ID: req.ID,
		}
	}

	if params.ID == "" {
		return &QueryResponse{
			Error: &QueryError{
				Code:    -32602,
				Message: "Missing required parameter: id",
			},
			ID: req.ID,
		}
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
		return &QueryResponse{
			Error: &QueryError{
				Code:    -32000,
				Message: "Failed to select session: " + err.Error(),
			},
			ID: req.ID,
		}
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
		return &QueryResponse{
			Error: &QueryError{
				Code:    -32602,
				Message: "Invalid params: " + err.Error(),
			},
			ID: req.ID,
		}
	}

	if params.Title == "" {
		return &QueryResponse{
			Error: &QueryError{
				Code:    -32602,
				Message: "Missing required parameter: title",
			},
			ID: req.ID,
		}
	}


	// Create session
	session, err := h.app.Sessions.Create(ctx, params.Title, params.WorkingDirectory)
	if err != nil {
		return &QueryResponse{
			Error: &QueryError{
				Code:    -32000,
				Message: "Failed to create session: " + err.Error(),
			},
			ID: req.ID,
		}
	}

	// Optionally set as current
	if params.SetCurrent {
		err = h.app.SetCurrentSession(session.ID)
		if err != nil {
			return &QueryResponse{
				Error: &QueryError{
					Code:    -32000,
					Message: "Session created but failed to set as current: " + err.Error(),
				},
				ID: req.ID,
			}
		}
	}

	result := SessionData{
		ID:               session.ID,
		Title:            session.Title,
		MessageCount:     session.MessageCount,
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
		return &QueryResponse{
			Error: &QueryError{
				Code:    -32602,
				Message: "Invalid params: " + err.Error(),
			},
			ID: req.ID,
		}
	}

	if params.Name == "" {
		return &QueryResponse{
			Error: &QueryError{
				Code:    -32602,
				Message: "Missing required parameter: name",
			},
			ID: req.ID,
		}
	}

	cmd, exists := h.commandRegistry.GetCommand(params.Name)
	if !exists {
		return &QueryResponse{
			Error: &QueryError{
				Code:    -32000,
				Message: "Command not found: " + params.Name,
			},
			ID: req.ID,
		}
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
		return &QueryResponse{
			Error: &QueryError{
				Code:    -32602,
				Message: "Invalid params: " + err.Error(),
			},
			ID: req.ID,
		}
	}

	if params.SessionID == "" {
		return &QueryResponse{
			Error: &QueryError{
				Code:    -32602,
				Message: "Missing required parameter: sessionId",
			},
			ID: req.ID,
		}
	}

	if params.Content == "" {
		return &QueryResponse{
			Error: &QueryError{
				Code:    -32602,
				Message: "Missing required parameter: content",
			},
			ID: req.ID,
		}
	}

	// Set the session as current
	err := h.app.SetCurrentSession(params.SessionID)
	if err != nil {
		return &QueryResponse{
			Error: &QueryError{
				Code:    -32000,
				Message: "Failed to set session: " + err.Error(),
			},
			ID: req.ID,
		}
	}

	// Check if this is a slash command and handle it immediately
	if commands.IsSlashCommand(params.Content) {
		parsed, parseErr := commands.ParseCommand(params.Content)
		if parseErr != nil {
			return &QueryResponse{
				Error: &QueryError{
					Code:    -32602,
					Message: "Invalid slash command: " + parseErr.Error(),
				},
				ID: req.ID,
			}
		}

		log.Printf("Executing command: '%s' with args: '%s'", parsed.Name, parsed.Arguments)

		commandResult, execErr := h.commandRegistry.ExecuteCommand(ctx, parsed.Name, parsed.Arguments)
		if execErr != nil {
			log.Printf("Command execution failed for '%s': %v", parsed.Name, execErr)

			// Check if it's a "command not found" error
			if strings.Contains(execErr.Error(), "command not found") {
				// List available commands for debugging
				allCommands := h.commandRegistry.GetAllCommands()
				commandNames := getCommandNames(allCommands)
				log.Printf("Available commands: %v", commandNames)

				return &QueryResponse{
					Error: &QueryError{
						Code:    -32000,
						Message: fmt.Sprintf("Command '%s' not found. Available commands: %v", parsed.Name, commandNames),
					},
					ID: req.ID,
				}
			}

			return &QueryResponse{
				Error: &QueryError{
					Code:    -32000,
					Message: "Command execution failed: " + execErr.Error(),
				},
				ID: req.ID,
			}
		}

		log.Printf("Command '%s' executed successfully, result length: %d", parsed.Name, len(commandResult))

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
		return &QueryResponse{
			Error: &QueryError{
				Code:    -32000,
				Message: "Failed to send message: " + err.Error(),
			},
			ID: req.ID,
		}
	}

	// Wait for response
	result := <-done

	// Check for processing errors
	if result.Error != nil {
		return &QueryResponse{
			Error: &QueryError{
				Code:    -32000,
				Message: "Agent processing failed: " + result.Error.Error(),
			},
			ID: req.ID,
		}
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
		return &QueryResponse{
			Error: &QueryError{
				Code:    -32602,
				Message: "Invalid params: " + err.Error(),
			},
			ID: req.ID,
		}
	}

	// Set default limit if not provided
	if params.Limit <= 0 {
		params.Limit = 50
	}

	messages, err := h.app.Messages.ListUserMessageHistory(ctx, params.Limit, params.Offset)
	if err != nil {
		return &QueryResponse{
			Error: &QueryError{
				Code:    -32000,
				Message: "Failed to get message history: " + err.Error(),
			},
			ID: req.ID,
		}
	}

	var result []MessageData
	for _, msg := range messages {
		result = append(result, MessageData{
			ID:        msg.ID,
			SessionID: msg.SessionID,
			Role:      string(msg.Role),
			Content:   msg.Content().String(),
		})
	}

	return &QueryResponse{
		Result: result,
		ID:     req.ID,
	}
}

