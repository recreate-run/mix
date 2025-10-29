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
	agent       Service // Parent agent for re-publishing subagent events
}

const (
	TaskToolName = "Task"
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
			tools.NewEditTool(b.permissions, nil),  // history not needed for sub-agents
			tools.NewWriteTool(b.permissions, nil), // history not needed for sub-agents
			tools.NewWebFetchTool(b.permissions),
			tools.NewWebSearchTool(b.permissions),
			tools.NewReadMediaTool(),
			tools.NewTodoWriteTool(),
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

// validateTaskParams validates the task parameters
func (b *taskTool) validateTaskParams(params TaskParams) error {
	if params.Description == "" {
		return fmt.Errorf("description is required")
	}
	if params.Prompt == "" {
		return fmt.Errorf("prompt is required")
	}
	if params.SubagentType == "" {
		return fmt.Errorf("subagent_type is required")
	}
	return nil
}

// createSubagentAndSession creates the subagent and its session
func (b *taskTool) createSubagentAndSession(ctx context.Context, params TaskParams, sessionID, toolCallID string) (Service, *session.Session, error) {
	agentTools := b.getToolsForSubagentType(params.SubagentType)
	parentBroker := b.agent.GetBroker()

	agent, err := NewAgentWithBroker("sub", b.sessions, b.messages, agentTools, session.DefaultConfig(), parentBroker, b.permissions)
	if err != nil {
		return nil, nil, fmt.Errorf("error creating agent: %w", err)
	}

	subSession, err := b.sessions.Create(ctx, "Subagent: "+params.Description, "", "default", session.SessionTypeSubagent, session.SubagentType(params.SubagentType), sessionID, toolCallID)
	if err != nil {
		agent.Shutdown()
		return nil, nil, fmt.Errorf("error creating session for tool call %s: %w", toolCallID, err)
	}

	return agent, &subSession, nil
}

// waitForFinalResult waits for the subagent to complete and returns the final message
func (b *taskTool) waitForFinalResult(done <-chan AgentEvent) (message.Message, error) {
	var finalResult AgentEvent
	for result := range done {
		if result.Error != nil {
			return message.Message{}, fmt.Errorf("error generating agent: %w", result.Error)
		}

		if result.Message.FinishReason() == message.FinishReasonEndTurn {
			finalResult = result
			break
		}
	}

	if finalResult.Message.Role == "" {
		return message.Message{}, fmt.Errorf("no final message received from sub-agent")
	}

	if finalResult.Message.Role != message.Assistant {
		return message.Message{}, fmt.Errorf("expected assistant response, got %s", finalResult.Message.Role)
	}

	return finalResult.Message, nil
}

// rollupCostToParent increments the parent session cost with the subagent's cost
func (b *taskTool) rollupCostToParent(ctx context.Context, sessionID, subSessionID string) error {
	updatedSubSession, err := b.sessions.Get(ctx, subSessionID)
	if err != nil {
		return fmt.Errorf("error getting subagent session: %w", err)
	}

	if _, err := b.sessions.Get(ctx, sessionID); err != nil {
		return fmt.Errorf("parent session %s not found during cost rollup: %w", sessionID, err)
	}

	if err := b.sessions.IncrementCost(ctx, sessionID, updatedSubSession.Cost); err != nil {
		return fmt.Errorf("failed to increment parent session %s cost: %w", sessionID, err)
	}

	return nil
}

func (b *taskTool) Run(ctx context.Context, call tools.ToolCall) (tools.ToolResponse, error) {
	var params TaskParams
	if err := json.Unmarshal([]byte(call.Input), &params); err != nil {
		return tools.NewTextErrorResponse(fmt.Sprintf("error parsing parameters: %s", err)), nil
	}

	if err := b.validateTaskParams(params); err != nil {
		return tools.NewTextErrorResponse(err.Error()), nil
	}

	sessionID, messageID := tools.GetContextValues(ctx)
	if sessionID == "" || messageID == "" {
		return tools.ToolResponse{}, fmt.Errorf("session_id and message_id are required")
	}

	agent, subSession, err := b.createSubagentAndSession(ctx, params, sessionID, call.ID)
	if err != nil {
		return tools.ToolResponse{}, err
	}
	defer agent.Shutdown()

	toolCtx := withToolContext(ctx, call.ID)
	done, err := agent.Run(toolCtx, subSession.ID, params.Prompt)
	if err != nil {
		return tools.ToolResponse{}, fmt.Errorf("error running agent: %w", err)
	}

	finalMessage, err := b.waitForFinalResult(done)
	if err != nil {
		return tools.ToolResponse{}, err
	}

	if err := b.rollupCostToParent(ctx, sessionID, subSession.ID); err != nil {
		return tools.ToolResponse{}, err
	}

	return tools.NewTextResponse(finalMessage.Content().String()), nil
}

func NewTaskTool(
	sessions session.Service,
	messages message.Service,
	permissions permission.Service,
) tools.BaseTool {
	return &taskTool{
		sessions:    sessions,
		messages:    messages,
		permissions: permissions,
	}
}

// SetAgent injects the parent agent reference after tool creation
// This resolves the circular dependency where tools need the agent but agent needs tools
func (b *taskTool) SetAgent(agent Service) {
	b.agent = agent
}
