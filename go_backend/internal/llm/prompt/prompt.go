package prompt

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"mix/internal/config"
	"mix/internal/llm/models"
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
			contextContent := getContextFromPaths()
			logging.Debug("Context content", "Context", contextContent)
			if contextContent != "" {
				return fmt.Sprintf("%s\n\n# Project-Specific Context\n Make sure to follow the instructions in the context below\n%s", basePrompt, contextContent)
			}
		}
	}

	return basePrompt
}

var (
	onceContext    sync.Once
	contextContent string
)

func getContextFromPaths() string {
	onceContext.Do(func() {
		var (
			cfg          = config.Get()
			workDir      = cfg.WorkingDir
			contextPaths = []string{} // Context paths removed for embedded binary
		)

		contextContent = processContextPaths(workDir, contextPaths)
	})

	return contextContent
}

func processContextPaths(workDir string, paths []string) string {
	var (
		wg       sync.WaitGroup
		resultCh = make(chan string)
	)

	// Track processed files to avoid duplicates
	processedFiles := make(map[string]bool)
	var processedMutex sync.Mutex

	for _, path := range paths {
		wg.Add(1)
		go func(p string) {
			defer wg.Done()

			if strings.HasSuffix(p, "/") {
				filepath.WalkDir(filepath.Join(workDir, p), func(path string, d os.DirEntry, err error) error {
					if err != nil {
						return err
					}
					if !d.IsDir() {
						// Check if we've already processed this file (case-insensitive)
						processedMutex.Lock()
						lowerPath := strings.ToLower(path)
						if !processedFiles[lowerPath] {
							processedFiles[lowerPath] = true
							processedMutex.Unlock()

							if result := processFile(path); result != "" {
								resultCh <- result
							}
						} else {
							processedMutex.Unlock()
						}
					}
					return nil
				})
			} else {
				fullPath := filepath.Join(workDir, p)

				// Check if we've already processed this file (case-insensitive)
				processedMutex.Lock()
				lowerPath := strings.ToLower(fullPath)
				if !processedFiles[lowerPath] {
					processedFiles[lowerPath] = true
					processedMutex.Unlock()

					result := processFile(fullPath)
					if result != "" {
						resultCh <- result
					}
				} else {
					processedMutex.Unlock()
				}
			}
		}(path)
	}

	go func() {
		wg.Wait()
		close(resultCh)
	}()

	results := make([]string, 0)
	for result := range resultCh {
		results = append(results, result)
	}

	return strings.Join(results, "\n")
}

func processFile(filePath string) string {
	content, err := os.ReadFile(filePath)
	if err != nil {
		return ""
	}
	return "# From:" + filePath + "\n" + string(content)
}
