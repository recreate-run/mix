package prompt

import (
	"context"
	"fmt"
	"regexp"
	"runtime"
	"strings"
	"time"

	"mix/internal/config"
	"mix/internal/llm/models"
)

func GetAgentPromptWithVars(ctx context.Context, agentName config.AgentName, provider models.ModelProvider, sessionVars map[string]string, customPrompt, mode string) (string, error) {
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

	// Separate base prompt content from system-controlled env section
	baseContent, envSection := extractEnvSection(basePrompt)

	// Handle custom system prompt customization on content only (not env)
	var finalContent string
	if customPrompt != "" {
		switch mode {
		case "replace":
			// Replace the entire base content with custom prompt
			finalContent = applyVariableSubstitution(customPrompt, sessionVars)
		case "append":
			// Append custom prompt to base content
			customWithVars := applyVariableSubstitution(customPrompt, sessionVars)
			finalContent = baseContent + "\n\n" + customWithVars
		default:
			// Default mode, use base content as-is (customPrompt ignored)
			finalContent = baseContent
		}
	} else {
		// No custom prompt, use base content
		finalContent = baseContent
	}

	// Always append system-controlled env section at the end
	finalPrompt := finalContent
	if envSection != "" {
		// Apply variable substitution to env section
		envWithVars := applyVariableSubstitution(envSection, sessionVars)
		finalPrompt = finalContent + "\n\n" + envWithVars
	}

	return finalPrompt, nil
}

// extractEnvSection separates the <env> section from the base prompt content
// Returns (baseContent, envSection) where envSection includes the <env> tags
func extractEnvSection(prompt string) (baseContent, envSection string) {
	envRegex := regexp.MustCompile(`(?s)<env>.*?</env>`)
	envMatch := envRegex.FindString(prompt)

	if envMatch == "" {
		// No env section found, return original prompt and empty env
		return prompt, ""
	}

	// Remove env section from base content
	baseContent = envRegex.ReplaceAllString(prompt, "")
	baseContent = strings.TrimSpace(baseContent)

	return baseContent, envMatch
}

// applyVariableSubstitution applies variable substitution to custom prompts using the same logic as LoadPrompt
func applyVariableSubstitution(content string, sessionVars map[string]string) string {
	result := content

	// Build variables starting with standard ones
	allVars := make(map[string]string)

	// Add platform and date (always available)
	allVars["platform"] = runtime.GOOS
	allVars["today_date"] = time.Now().Format("2006-01-02")

	// Merge with session vars (session vars override standard ones)
	for k, v := range sessionVars {
		allVars[k] = v
	}

	// Replace $<name> placeholders with values
	for key, value := range allVars {
		placeholder := "$<" + key + ">"
		result = strings.ReplaceAll(result, placeholder, value)
	}

	return strings.TrimSpace(result)
}
