package prompt

import (
	"context"
	"fmt"

	"mix/internal/config"
	"mix/internal/llm/models"
)

func GetAgentPromptWithVars(ctx context.Context, agentName config.AgentName, provider models.ModelProvider, sessionVars map[string]string) (string, error) {
	var basePrompt string
	var err error

	if agentName == config.AgentSub {
		// Load task agent system prompt (uses same system.md as main agent)
		basePrompt, err = LoadPrompt(ctx, "system", sessionVars)
		if err != nil {
			return "", fmt.Errorf("failed to load system prompt for sub agent: %w", err)
		}
	} else {
		// Load main agent prompt with standard environment variables
		basePrompt, err = LoadPrompt(ctx, "system", sessionVars)
		if err != nil {
			return "", fmt.Errorf("failed to load system prompt for main agent: %w", err)
		}

		// Main agent uses base prompt without context files
		// For code exploration, use subagents via the Task tool instead
	}

	return basePrompt, nil
}

