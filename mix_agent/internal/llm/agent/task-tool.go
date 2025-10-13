package agent

import (
	"context"
	"encoding/json"
	"fmt"

	"mix/internal/llm/tools"
	"mix/internal/message"
	"mix/internal/permission"
	"mix/internal/session"
)

type taskTool struct {
	sessions    session.Service
	messages    message.Service
	permissions permission.Service
}

const (
	TaskToolName = "task"
)

type TaskParams struct {
	Description  string `json:"description"`
	Prompt       string `json:"prompt"`
	SubagentType string `json:"subagent_type"`
}

func (b *taskTool) getToolsForSubagentType(subagentType string) []tools.BaseTool {
	switch subagentType {
	case "general-purpose":
		// General-purpose agent with full access to all tools except MCP
		return []tools.BaseTool{
			tools.NewGlobTool(),
			tools.NewGrepTool(b.permissions),
			tools.NewReadTextTool(),
			tools.NewEditTool(b.permissions, nil), // history not needed for sub-agents
			tools.NewWriteTool(b.permissions, nil), // history not needed for sub-agents
			tools.NewWebFetchTool(b.permissions),
			tools.NewWebSearchTool(b.permissions),
			tools.NewReadMediaTool(),
			tools.NewTodoWriteTool(),
			NewTaskTool(b.sessions, b.messages, b.permissions),
		}
	default:
		// Default to limited read-only tools for safety
		return TaskAgentTools(b.permissions, b.sessions, b.messages)
	}
}

func (b *taskTool) Info() tools.ToolInfo {
	return tools.ToolInfo{
		Name:        TaskToolName,
		Description: tools.LoadToolDescription("task_tool"),
		Parameters: map[string]any{
			"description": map[string]any{
				"description": "A short (3-5 word) description of the task",
				"type":        "string",
			},
			"prompt": map[string]any{
				"description": "The task for the agent to perform",
				"type":        "string",
			},
			"subagent_type": map[string]any{
				"description": "The type of specialized agent to use for this task",
				"type":        "string",
			},
		},
		Required: []string{"description", "prompt", "subagent_type"},
	}
}

func (b *taskTool) Run(ctx context.Context, call tools.ToolCall) (tools.ToolResponse, error) {
	var params TaskParams
	if err := json.Unmarshal([]byte(call.Input), &params); err != nil {
		return tools.NewTextErrorResponse(fmt.Sprintf("error parsing parameters: %s", err)), nil
	}
	if params.Description == "" {
		return tools.NewTextErrorResponse("description is required"), nil
	}
	if params.Prompt == "" {
		return tools.NewTextErrorResponse("prompt is required"), nil
	}
	if params.SubagentType == "" {
		return tools.NewTextErrorResponse("subagent_type is required"), nil
	}

	sessionID, messageID := tools.GetContextValues(ctx)
	if sessionID == "" || messageID == "" {
		return tools.ToolResponse{}, fmt.Errorf("session_id and message_id are required")
	}

	agentTools := b.getToolsForSubagentType(params.SubagentType)
	agent, err := NewAgent("sub", b.sessions, b.messages, agentTools, session.DefaultConfig(), b.permissions)
	if err != nil {
		return tools.ToolResponse{}, fmt.Errorf("error creating agent: %s", err)
	}
	defer agent.Shutdown()

	// Use suppress-publish context to prevent SSE broadcasting of internal sub-agent session
	suppressCtx := session.WithSuppressPublish(ctx)
	session, err := b.sessions.Create(suppressCtx, "New Agent Session", "", "default")
	if err != nil {
		return tools.ToolResponse{}, fmt.Errorf("error creating session: %s", err)
	}

	// Clean up the temporary sub-agent session even if errors occur
	defer func() {
		if err := b.sessions.Delete(suppressCtx, session.ID); err != nil {
			fmt.Printf("[TASK TOOL] Warning: failed to delete sub-agent session %s: %v\n", session.ID, err)
		}
	}()

	done, err := agent.Run(ctx, session.ID, params.Prompt)
	if err != nil {
		return tools.ToolResponse{}, fmt.Errorf("error generating agent: %s", err)
	}

	// Wait for the final message with end_turn finish reason
	var finalResult AgentEvent
	for result := range done {
		if result.Error != nil {
			return tools.ToolResponse{}, fmt.Errorf("error generating agent: %s", result.Error)
		}

		// Check if this is the final message
		if result.Message.FinishReason() == message.FinishReasonEndTurn {
			finalResult = result
			break
		}

		// Continue processing intermediate messages (like tool_use)
	}

	// Verify we got a final result
	if finalResult.Message.Role == "" {
		return tools.ToolResponse{}, fmt.Errorf("no final message received from sub-agent")
	}

	response := finalResult.Message
	if response.Role != message.Assistant {
		return tools.NewTextErrorResponse("no response"), nil
	}

	// Get content from the final response
	content := response.Content().String()

	// Log the final output returned by the sub-agent
	previewLen := 100
	if len(content) < previewLen {
		previewLen = len(content)
	}
	preview := content
	if len(content) > previewLen {
		preview = content[:previewLen] + "..."
	}
	fmt.Printf("[TASK TOOL] Sub-agent returned %d characters: %q\n", len(content), preview)

	updatedSession, err := b.sessions.Get(ctx, session.ID)
	if err != nil {
		return tools.ToolResponse{}, fmt.Errorf("error getting session: %s", err)
	}
	parentSession, err := b.sessions.Get(ctx, sessionID)
	if err != nil {
		return tools.ToolResponse{}, fmt.Errorf("error getting parent session: %s", err)
	}

	parentSession.Cost += updatedSession.Cost

	_, err = b.sessions.Save(ctx, parentSession)
	if err != nil {
		return tools.ToolResponse{}, fmt.Errorf("error saving parent session: %s", err)
	}

	// Session cleanup handled by defer at function start
	return tools.NewTextResponse(content), nil
}

func NewTaskTool(
	Sessions session.Service,
	Messages message.Service,
	Permissions permission.Service,
) tools.BaseTool {
	return &taskTool{
		sessions:    Sessions,
		messages:    Messages,
		permissions: Permissions,
	}
}
