package agent

import (
	"mix/internal/history"
	"mix/internal/llm/tools"
	"mix/internal/llm/tools/browser"
	"mix/internal/message"
	"mix/internal/notification"
	"mix/internal/permission"
	"mix/internal/session"
)

// CoderAgentTools returns the tools available to the coder agent
func CoderAgentTools(
	permissions permission.Service,
	notifications notification.Service,
	sessions session.Service,
	messages message.Service,
	historySvc history.Service,
	manager *MCPClientManager,
	browserMode string,
	browserServiceURL string,
	clientFactory browser.ClientFactory,
	connectionManager interface{},
	tunnelRegistryGetter func() interface{},
) []tools.BaseTool {
	// Don't block on MCP tools during initialization - they will be loaded in the background
	// and available when first needed (lazy loading happens in GetClient)
	bashTool := tools.NewBashTool(permissions)

	return []tools.BaseTool{
		bashTool,
		tools.NewEditTool(permissions, historySvc),
		// tools.NewGlobTool(),
		// tools.NewGrepTool(permissions),
		tools.NewReadTextTool(),
		// tools.NewWebFetchTool(permissions),
		tools.NewWebSearchTool(permissions),
		tools.NewWriteTool(permissions, historySvc),
		// tools.NewPythonExecutionTool(permissions),
		// tools.NewReadMediaTool(),
		tools.NewTodoWriteTool(),
		tools.NewExitPlanModeTool(),
		// tools.NewShowTool(),
		tools.NewNotifyTool(notifications),
		browser.NewBrowserTool(permissions, browserServiceURL, session.DefaultConfig(), browserMode, clientFactory, connectionManager, tunnelRegistryGetter),
		NewTaskTool(sessions, messages, permissions),
	}
}

func TaskAgentTools(
	permissions permission.Service,
	sessions session.Service,
	messages message.Service,
) []tools.BaseTool {
	return []tools.BaseTool{
		tools.NewGlobTool(),
		tools.NewGrepTool(permissions),
		tools.NewReadTextTool(),
		NewTaskTool(sessions, messages, permissions),
	}
}
