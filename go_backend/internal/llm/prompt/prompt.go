package prompt

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"mix/internal/config"
	"mix/internal/llm/models"
	"mix/internal/llm/tools"
	"mix/internal/logging"
)

func GetAgentPromptWithVars(ctx context.Context, agentName config.AgentName, provider models.ModelProvider, sessionVars map[string]string) string {
	var basePrompt string

	if agentName == config.AgentSub {
		// Load task agent system prompt
		basePrompt = LoadPromptWithStandardVars(ctx, "task_agent", sessionVars)
	} else {
		// Load main agent prompt with standard environment variables
		basePrompt = LoadPromptWithStandardVars(ctx, "system", sessionVars)

		if agentName == config.AgentMain {
			// Add context from project-specific instruction files if they exist
			contextContent, err := getContextFromPaths(ctx)
			if err != nil {
				logging.Error("Failed to load context files", "error", err)
				return fmt.Sprintf("%s\n\n# Context Loading Error\nError loading project context files: %s", basePrompt, err.Error())
			}
			logging.Debug("Context content", "Context", contextContent)
			if contextContent != "" {
				return fmt.Sprintf("%s\n\n# Project-Specific Context\n Make sure to follow the instructions in the context below\n%s", basePrompt, contextContent)
			}
		}
	}

	return basePrompt
}


func getContextFromPaths(ctx context.Context) (string, error) {
	workingDir, ok := ctx.Value(tools.WorkingDirectoryContextKey).(string)
	if !ok {
		return "", fmt.Errorf("no working directory found in context")
	}

	cfg := config.Get()
	contextPaths := cfg.ContextPaths

	return processContextPaths(workingDir, contextPaths)
}

func processContextPaths(workDir string, paths []string) (string, error) {
	processedFiles := make(map[string]bool)
	results := make([]string, 0)
	var foundCount, loadedCount int

	for _, path := range paths {
		if strings.HasSuffix(path, "/") {
			err := filepath.WalkDir(filepath.Join(workDir, path), func(filePath string, d os.DirEntry, err error) error {
				if err != nil {
					return err
				}
				if !d.IsDir() {
					lowerPath := strings.ToLower(filePath)
					if processedFiles[lowerPath] {
						return nil
					}
					processedFiles[lowerPath] = true

					result, found, err := processFile(filePath)
					if err != nil {
						return err
					}
					if found {
						foundCount++
						if result != "" {
							loadedCount++
							results = append(results, result)
						}
					}
				}
				return nil
			})
			if err != nil {
				return "", err
			}
		} else {
			fullPath := filepath.Join(workDir, path)
			lowerPath := strings.ToLower(fullPath)
			if processedFiles[lowerPath] {
				continue
			}
			processedFiles[lowerPath] = true

			result, found, err := processFile(fullPath)
			if err != nil {
				return "", err
			}
			if found {
				foundCount++
				if result != "" {
					loadedCount++
					results = append(results, result)
				}
			}
		}
	}

	content := strings.Join(results, "\n")
	logging.Info("Context file loading completed",
		"files_found", foundCount,
		"files_loaded", loadedCount,
		"content_length", len(content))

	return content, nil
}

func processFile(filePath string) (string, bool, error) {
	content, err := os.ReadFile(filePath)
	if err != nil {
		if os.IsNotExist(err) {
			logging.Info("Context file not found", "path", filePath)
			return "", false, nil // Not found, not an error
		}
		logging.Error("Failed to read context file", "path", filePath, "error", err)
		return "", false, fmt.Errorf("failed to read context file %s: %w", filePath, err)
	}
	
	if len(content) == 0 {
		logging.Info("Context file is empty", "path", filePath)
		return "", true, nil // Found but empty
	}
	
	logging.Info("Successfully loaded context file", "path", filePath, "size", len(content))
	return "# From:" + filePath + "\n" + string(content), true, nil
}
