package prompt

import (
	"context"
	"embed"
	"fmt"
	"os"
	"path"
	"path/filepath"
	"regexp"
	"runtime"
	"strings"

	"mix/internal/config"
	"mix/internal/llm/tools"
)

//go:embed prompts/*.md
var promptFiles embed.FS

// LoadPrompt loads a prompt from embedded markdown files
func LoadPrompt(name string) string {
	return LoadPromptWithVars(name, nil)
}

// LoadPromptWithVars loads a prompt from embedded markdown files and replaces $<name> placeholders
func LoadPromptWithVars(name string, vars map[string]string) string {
	content, err := promptFiles.ReadFile(path.Join("prompts", name+".md"))
	if err != nil {
		// This should not happen with embedded files, but provide minimal fallback
		return "Error loading prompt: " + name
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
		panic(fmt.Sprintf("failed to get standard vars for prompt '%s': %v", name, err))
	}
	for k, v := range customVars {
		allVars[k] = v
	}

	return LoadPromptWithVars(name, allVars)
}

// resolveMarkdownTemplates resolves {markdown:path} templates in content
func resolveMarkdownTemplates(content string, vars map[string]string) string {
	markdownRegex := regexp.MustCompile(`\{markdown:([^}]+)\}`)
	workspaceRoot, err := config.LaunchDirectory()
	if err != nil {
		panic(fmt.Sprintf("failed to get launch directory for markdown templates: %v", err))
	}

	return markdownRegex.ReplaceAllStringFunc(content, func(match string) string {
		// Extract the file path from the match
		submatches := markdownRegex.FindStringSubmatch(match)
		if len(submatches) < 2 {
			panic("Invalid markdown template: " + match)
		}

		relativePath := strings.TrimSpace(submatches[1])
		if relativePath == "" {
			panic("Empty path in markdown template: " + match)
		}

		// Construct absolute path relative to workspace
		fullPath := filepath.Join(workspaceRoot, relativePath)

		// Read the file content
		fileContent, err := os.ReadFile(fullPath)
		if err != nil {
			panic("Failed to load markdown file: " + relativePath + " - " + err.Error())
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
			panic("Unmatched template variables in markdown file " + relativePath + ": " + strings.Join(matches, ", "))
		}

		return result
	})
}
