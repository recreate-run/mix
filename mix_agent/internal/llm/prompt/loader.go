package prompt

import (
	"context"
	"fmt"
	"path/filepath"
	"regexp"
	"runtime"
	"strings"
	"time"

	"mix/internal/config"
	"mix/internal/llm/tools"
)

// LoadPrompt loads a prompt from embedded filesystem with automatic standard variables
func LoadPrompt(ctx context.Context, name string, customVars map[string]string) (string, error) {
	// Load from embedded filesystem
	embeddedFS := config.GetEmbeddedPrompts()
	promptPath := filepath.Join("prompts", name+".md")

	content, err := embeddedFS.ReadFile(promptPath)
	if err != nil {
		return "", fmt.Errorf("failed to read embedded prompt file '%s': %w", promptPath, err)
	}

	result := string(content)

	// Build variables starting with standard ones (if context available)
	allVars := make(map[string]string)

	// Add standard variables if context is available
	if ctx != nil {
		// Add platform and date (always available)
		allVars["platform"] = runtime.GOOS
		allVars["today_date"] = time.Now().Format("2006-01-02")

		// Add session working directory if available
		if sessionStorageDir := ctx.Value(tools.SessionStorageContextKey); sessionStorageDir != nil {
			if workdir, ok := sessionStorageDir.(string); ok {
				allVars["workdir"] = workdir
			}
		}
	}

	// Merge with custom vars (custom vars override standard ones)
	for k, v := range customVars {
		allVars[k] = v
	}

	// Replace $<name> placeholders with values
	for key, value := range allVars {
		placeholder := "$<" + key + ">"
		result = strings.ReplaceAll(result, placeholder, value)
	}

	// Resolve markdown file templates
	result, err = resolveMarkdownTemplates(result, allVars)
	if err != nil {
		return "", err
	}

	return strings.TrimSpace(result), nil
}

// resolveMarkdownTemplates resolves {markdown:path} templates in content
func resolveMarkdownTemplates(content string, vars map[string]string) (string, error) {
	markdownRegex := regexp.MustCompile(`\{markdown:([^}]+)\}`)
	var resolveErr error

	result := markdownRegex.ReplaceAllStringFunc(content, func(match string) string {
		// Extract the file path from the match
		submatches := markdownRegex.FindStringSubmatch(match)
		if len(submatches) < 2 {
			resolveErr = fmt.Errorf("invalid markdown template: %s", match)
			return match
		}

		relativePath := strings.TrimSpace(submatches[1])
		if relativePath == "" {
			resolveErr = fmt.Errorf("empty path in markdown template: %s", match)
			return match
		}

		// Load from embedded filesystem only
		embeddedFS := config.GetEmbeddedPrompts()
		embeddedPath := filepath.Join("prompts", relativePath)
		fileContent, err := embeddedFS.ReadFile(embeddedPath)

		if err != nil {
			resolveErr = fmt.Errorf("failed to load embedded markdown template '%s': %w", relativePath, err)
			return match
		}

		fileResult := string(fileContent)

		// Apply variable substitution to included markdown file
		if vars != nil {
			for key, value := range vars {
				placeholder := "$<" + key + ">"
				fileResult = strings.ReplaceAll(fileResult, placeholder, value)
			}
		}

		// Check for unmatched template variables
		templateRegex := regexp.MustCompile(`\$<[^>]+>`)
		if matches := templateRegex.FindAllString(fileResult, -1); len(matches) > 0 {
			resolveErr = fmt.Errorf("unmatched template variables in markdown file %s: %s", relativePath, strings.Join(matches, ", "))
			return match
		}

		return fileResult
	})

	if resolveErr != nil {
		return "", resolveErr
	}

	return result, nil
}
