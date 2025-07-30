package prompt

import (
	"context"
	"embed"
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
	result = resolveMarkdownTemplates(result)

	return strings.TrimSpace(result)
}

// getStandardVars returns standard variables available to all prompts
func getStandardVars() map[string]string {
	cwd := config.WorkingDirectory()
	isGit := isGitRepo(cwd)
	platform := runtime.GOOS
	ls := tools.NewLsTool()
	r, _ := ls.Run(context.Background(), tools.ToolCall{
		Input: `{"path":"."}`,
	})

	return map[string]string{
		"workdir":     cwd,
		"platform":    platform,
		"is_git_repo": boolToYesNo(isGit),
		"project_ls":  r.Content,
	}
}

// LoadPromptWithStandardVars loads a prompt with standard environment variables plus custom vars
func LoadPromptWithStandardVars(name string, customVars map[string]string) string {
	// Merge standard vars with custom vars
	allVars := getStandardVars()
	for k, v := range customVars {
		allVars[k] = v
	}

	return LoadPromptWithVars(name, allVars)
}

func isGitRepo(dir string) bool {
	_, err := os.Stat(filepath.Join(dir, ".git"))
	return err == nil
}

func boolToYesNo(b bool) string {
	if b {
		return "Yes"
	}
	return "No"
}

// resolveMarkdownTemplates resolves {markdown:path} templates in content
func resolveMarkdownTemplates(content string) string {
	markdownRegex := regexp.MustCompile(`\{markdown:([^}]+)\}`)
	workspaceRoot := config.WorkingDirectory()

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

		return string(fileContent)
	})
}
