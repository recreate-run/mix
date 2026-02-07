package agent

import (
	"os"

	"mix/internal/history"
	"mix/internal/llm/tools"
	"mix/internal/llm/tools/browser"
	"mix/internal/message"
	"mix/internal/notification"
	"mix/internal/permission"
	"mix/internal/session"
)

func CoderAgentTools(
	permissions permission.Service,
	notifications notification.Service,
	sessions session.Service,
	messages message.Service,
	historySvc history.Service,
	manager *MCPClientManager,
) []tools.BaseTool {
	// Don't block on MCP tools during initialization - they will be loaded in the background
	// and available when first needed (lazy loading happens in GetClient)
	bashTool := tools.NewBashTool(permissions)

	// Get browser service URL from environment - required
	browserServiceURL := os.Getenv("BROWSER_SERVICE_URL")
	if browserServiceURL == "" {
		panic("BROWSER_SERVICE_URL environment variable is required but not set")
	}

	return []tools.BaseTool{
		bashTool,
		tools.NewEditTool(permissions, historySvc),
		tools.NewGlobTool(),
		tools.NewGrepTool(permissions),
		tools.NewReadTextTool(),
		tools.NewWebFetchTool(permissions),
		tools.NewWebSearchTool(permissions),
		tools.NewWriteTool(permissions, historySvc),
		// tools.NewPythonExecutionTool(permissions),
		// tools.NewReadMediaTool(),
		tools.NewTodoWriteTool(),
		tools.NewExitPlanModeTool(),
		// tools.NewShowTool(),
		tools.NewNotifyTool(notifications),
		browser.NewBrowserTool(permissions, browserServiceURL, session.DefaultConfig()),
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
