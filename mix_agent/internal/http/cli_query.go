package http

import (
	"context"
	"fmt"
	"sort"
	"time"

	"mix/internal/app"
)

// CLIQueryHandler provides simple data access for CLI queries
type CLIQueryHandler struct {
	sessionHandler *SessionHandler
	systemHandler  *SystemHandler
}

// NewCLIQueryHandler creates a new CLI query handler
func NewCLIQueryHandler(a *app.App) *CLIQueryHandler {
	return &CLIQueryHandler{
		sessionHandler: NewSessionHandler(a),
		systemHandler:  NewSystemHandler(a),
	}
}

// HandleQueryType handles CLI queries by type
func (h *CLIQueryHandler) HandleQueryType(ctx context.Context, queryType string) (interface{}, error) {
	switch queryType {
	case "sessions":
		sessions, err := h.sessionHandler.app.Sessions.ListWithContent(ctx)
		if err != nil {
			return nil, fmt.Errorf("failed to list sessions: %w", err)
		}

		var result []SessionData
		for i := range sessions {
			result = append(result, SessionData{
				ID:                    sessions[i].ID,
				Title:                 sessions[i].Title,
				UserMessageCount:      sessions[i].UserMessageCount,
				AssistantMessageCount: sessions[i].AssistantMessageCount,
				ToolCallCount:         sessions[i].ToolCallCount,
				PromptTokens:          sessions[i].PromptTokens,
				CompletionTokens:      sessions[i].CompletionTokens,
				Cost:                  sessions[i].Cost,
				CreatedAt:             time.Unix(sessions[i].CreatedAt, 0),
				FirstUserMessage:      sessions[i].FirstUserMessage,
			})
		}
		return result, nil

	case "commands":
		allCommands := h.systemHandler.commandRegistry.GetAllCommands()

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

		return result, nil

	case "mcp":
		// Return empty for now - MCP query in CLI is not essential
		// Complex MCP logic can be added later if needed
		return []MCPServerData{}, nil

	default:
		return nil, fmt.Errorf("unsupported query type: %s. Supported types: sessions, commands, mcp", queryType)
	}
}

// GetSupportedQueryTypes returns all supported CLI query types
func (h *CLIQueryHandler) GetSupportedQueryTypes() []string {
	return []string{"sessions", "commands", "mcp"}
}
