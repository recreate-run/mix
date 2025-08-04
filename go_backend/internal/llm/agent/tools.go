package agent

import (
	"context"
	"time"

	"mix/internal/history"
	"mix/internal/llm/tools"
	"mix/internal/message"
	"mix/internal/permission"
	"mix/internal/session"
)

func CoderAgentTools(
	permissions permission.Service,
	sessions session.Service,
	messages message.Service,
	history history.Service,
	manager *MCPClientManager,
) []tools.BaseTool {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	otherTools := GetMcpTools(ctx, permissions, manager)
	bashTool := tools.NewBashTool(permissions)
	return append(
		[]tools.BaseTool{
			bashTool,
			tools.NewEditTool(permissions, history),
			tools.NewFetchTool(permissions),
			tools.NewGlobTool(),
			tools.NewGrepTool(),
			tools.NewLsTool(),
			tools.NewViewTool(),
			tools.NewWriteTool(permissions, history),
			tools.NewPythonExecutionTool(permissions),
			tools.NewTodoWriteTool(),
			tools.NewExitPlanModeTool(),
			// tools.NewPixelmatorTool(permissions, bashTool),
			// tools.NewNotesTool(permissions, bashTool),
			NewAgentTool(sessions, messages),
		}, otherTools...,
	)
}

func TaskAgentTools() []tools.BaseTool {
	return []tools.BaseTool{
		tools.NewGlobTool(),
		tools.NewGrepTool(),
		tools.NewLsTool(),
		tools.NewViewTool(),
	}
}
