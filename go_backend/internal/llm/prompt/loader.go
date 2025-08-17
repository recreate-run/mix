package prompt

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"runtime"
	"strings"

	"mix/internal/config"
	"mix/internal/llm/tools"
)

// LoadPrompt loads a prompt from filesystem markdown files
func LoadPrompt(name string) string {
	return LoadPromptWithVars(name, nil)
}

// LoadPromptWithVars loads a prompt from filesystem markdown files and replaces $<name> placeholders
func LoadPromptWithVars(name string, vars map[string]string) string {
	promptsDir, err := config.PromptsDirectory()
	if err != nil {
		return fmt.Sprintf("Error: failed to get prompts directory: %v", err)
	}
	
	promptPath := filepath.Join(promptsDir, name+".md")
	content, err := os.ReadFile(promptPath)
	if err != nil {
		if os.IsNotExist(err) {
			return fmt.Sprintf("Prompt file not found: %s\n\nPlease ensure the prompts directory exists at: %s\nand contains the required prompt files, or use --prompts-dir to specify a different location", promptPath, promptsDir)
		}
		return fmt.Sprintf("Failed to read prompt file '%s': %v", promptPath, err)
	}

	result := string(content)

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
	
	promptsDir, err := config.PromptsDirectory()
	if err != nil {
		return fmt.Sprintf("Error: failed to get prompts directory for markdown templates: %v", err)
	}

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

		// Load file relative to prompts directory
		promptPath := filepath.Join(promptsDir, relativePath)
		fileContent, err := os.ReadFile(promptPath)
		if err != nil {
			if os.IsNotExist(err) {
				return fmt.Sprintf("Markdown template file not found: %s\n\nPlease ensure the file exists in prompts directory: %s", relativePath, promptsDir)
			}
			return fmt.Sprintf("Failed to read markdown template file %s: %v", promptPath, err)
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
