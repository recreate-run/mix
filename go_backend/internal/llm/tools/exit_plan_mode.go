package tools

import (
	"context"
	"encoding/json"
	"fmt"
)

type ExitPlanModeTool struct{}

func NewExitPlanModeTool() *ExitPlanModeTool {
	return &ExitPlanModeTool{}
}

func (t *ExitPlanModeTool) Info() ToolInfo {
	return ToolInfo{
		Name:        "exit_plan_mode",
		Description: "Use this tool when you are in plan mode and have finished presenting your plan and are ready to execute. This will prompt the user to exit plan mode.",
		Parameters: map[string]any{
			"plan": map[string]any{
				"type":        "string",
				"description": "The plan you came up with, that you want to run by the user for approval. Supports markdown. The plan should be pretty concise.",
			},
		},
		Required: []string{"plan"},
	}
}

type ExitPlanModeParams struct {
	Plan string `json:"plan"`
}

func (t *ExitPlanModeTool) Run(ctx context.Context, params ToolCall) (ToolResponse, error) {
	var p ExitPlanModeParams
	if err := json.Unmarshal([]byte(params.Input), &p); err != nil {
		return NewTextErrorResponse("Failed to parse parameters"), err
	}

	if p.Plan == "" {
		return NewTextErrorResponse("Plan is required"), nil
	}

	response := fmt.Sprintf("# Plan Ready for Approval\n\n%s\n\n---\n\n✅ Ready to proceed when you confirm.", p.Plan)
	return NewTextResponse(response), nil
}
