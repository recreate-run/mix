package agent

import (
	"context"
	"encoding/json"
	"fmt"

	"mix/internal/llm/tools"
	"mix/internal/message"
	"mix/internal/session"
)

type agentTool struct {
	sessions session.Service
	messages message.Service
}

const (
	AgentToolName = "agent"
)

type AgentParams struct {
	Prompt string `json:"prompt"`
}

func (b *agentTool) Info() tools.ToolInfo {
	return tools.ToolInfo{
		Name:        AgentToolName,
		Description: tools.LoadToolDescription("agent_tool"),
		Parameters: map[string]any{
			"prompt": map[string]any{
				"type":        "string",
				"description": "The task for the agent to perform",
			},
		},
		Required: []string{"prompt"},
	}
}

func (b *agentTool) Run(ctx context.Context, call tools.ToolCall) (tools.ToolResponse, error) {
	var params AgentParams
	if err := json.Unmarshal([]byte(call.Input), &params); err != nil {
		return tools.NewTextErrorResponse(fmt.Sprintf("error parsing parameters: %s", err)), nil
	}
	if params.Prompt == "" {
		return tools.NewTextErrorResponse("prompt is required"), nil
	}

	sessionID, messageID := tools.GetContextValues(ctx)
	if sessionID == "" || messageID == "" {
		return tools.ToolResponse{}, fmt.Errorf("session_id and message_id are required")
	}

	agent, err := NewAgent("sub", b.sessions, b.messages, TaskAgentTools())
	if err != nil {
		return tools.ToolResponse{}, fmt.Errorf("error creating agent: %s", err)
	}
	defer agent.Shutdown()

	session, err := b.sessions.Create(ctx, "New Agent Session", ctx.Value(tools.WorkingDirectoryContextKey).(string))
	if err != nil {
		return tools.ToolResponse{}, fmt.Errorf("error creating session: %s", err)
	}

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
	fmt.Printf("[AGENT TOOL] Sub-agent returned %d characters: %q\n", len(content), preview)

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
	return tools.NewTextResponse(content), nil
}

func NewAgentTool(
	Sessions session.Service,
	Messages message.Service,
) tools.BaseTool {
	return &agentTool{
		sessions: Sessions,
		messages: Messages,
	}
}
