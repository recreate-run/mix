package prompt

import (
	"context"
	"fmt"
	"path/filepath"
	"regexp"
	"runtime"
	"strings"

	"mix/internal/config"
	"mix/internal/llm/tools"
)

// LoadPrompt loads a prompt from embedded filesystem markdown files
func LoadPrompt(name string) string {
	return LoadPromptWithVars(name, nil)
}

// loadEmbeddedPrompt loads a prompt from the embedded filesystem
func loadEmbeddedPrompt(name string) (string, error) {
	embeddedFS := config.GetEmbeddedPrompts()
	promptPath := filepath.Join("prompts", name+".md")
	
	content, err := embeddedFS.ReadFile(promptPath)
	if err != nil {
		return "", fmt.Errorf("failed to read embedded prompt file '%s': %w", promptPath, err)
	}
	
	return string(content), nil
}

// LoadPromptWithVars loads a prompt from embedded filesystem only and replaces $<name> placeholders
func LoadPromptWithVars(name string, vars map[string]string) string {
	// Load from embedded filesystem only
	result, err := loadEmbeddedPrompt(name)
	if err != nil {
		return fmt.Sprintf("Error: failed to load embedded prompt '%s': %v", name, err)
	}

	// Replace $<name> placeholders with values
	if vars != nil {
		for key, value := range vars {
			placeholder := "$<" + key + ">"
			result = strings.ReplaceAll(result, placeholder, value)
		}
	}

	// Resolve markdown file templates
	result = resolveMarkdownTemplates(result, vars)

	return strings.TrimSpace(result)
}

// getStandardVars returns standard variables available to all prompts
func getStandardVars(ctx context.Context) (map[string]string, error) {
	workingDir := ctx.Value(tools.WorkingDirectoryContextKey).(string)

	launchDir, err := config.LaunchDirectory()
	if err != nil {
		return nil, fmt.Errorf("failed to get launch directory: %w", err)
	}

	return map[string]string{
		"workdir":   workingDir,
		"platform":  runtime.GOOS,
		"launchdir": launchDir,
	}, nil
}

// LoadPromptWithStandardVars loads a prompt with standard environment variables plus custom vars
func LoadPromptWithStandardVars(ctx context.Context, name string, customVars map[string]string) string {
	// Merge standard vars with custom vars
	allVars, err := getStandardVars(ctx)
	if err != nil {
		return fmt.Sprintf("Error: failed to get standard vars for prompt '%s': %v", name, err)
	}
	for k, v := range customVars {
		allVars[k] = v
	}

	return LoadPromptWithVars(name, allVars)
}

// resolveMarkdownTemplates resolves {markdown:path} templates in content
func resolveMarkdownTemplates(content string, vars map[string]string) string {
	markdownRegex := regexp.MustCompile(`\{markdown:([^}]+)\}`)

	return markdownRegex.ReplaceAllStringFunc(content, func(match string) string {
		// Extract the file path from the match
		submatches := markdownRegex.FindStringSubmatch(match)
		if len(submatches) < 2 {
			return fmt.Sprintf("Error: Invalid markdown template: %s", match)
		}

		relativePath := strings.TrimSpace(submatches[1])
		if relativePath == "" {
			return fmt.Sprintf("Error: Empty path in markdown template: %s", match)
		}

		// Load from embedded filesystem only
		embeddedFS := config.GetEmbeddedPrompts()
		embeddedPath := filepath.Join("prompts", relativePath)
		fileContent, err := embeddedFS.ReadFile(embeddedPath)
		
		if err != nil {
			return fmt.Sprintf("Error: failed to load embedded markdown template '%s': %v", relativePath, err)
		}
		
		result := string(fileContent)

		// Apply variable substitution to included markdown file
		if vars != nil {
			for key, value := range vars {
				placeholder := "$<" + key + ">"
				result = strings.ReplaceAll(result, placeholder, value)
			}
		}

		// Check for unmatched template variables
		templateRegex := regexp.MustCompile(`\$<[^>]+>`)
		if matches := templateRegex.FindAllString(result, -1); len(matches) > 0 {
			return fmt.Sprintf("Error: Unmatched template variables in markdown file %s: %s", relativePath, strings.Join(matches, ", "))
		}

		return result
	})
}
