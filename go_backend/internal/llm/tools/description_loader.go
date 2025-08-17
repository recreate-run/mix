package tools

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"mix/internal/config"
)

// LoadToolDescription loads a tool description from filesystem tools directory
func LoadToolDescription(name string) string {
	promptsDir, err := config.PromptsDirectory()
	if err != nil {
		return fmt.Sprintf("Error: failed to get prompts directory: %v", err)
	}
	
	toolPath := filepath.Join(promptsDir, "tools", name+".md")
	content, err := os.ReadFile(toolPath)
	if err != nil {
		return fmt.Sprintf("Tool description not found: %s\n\nPlease ensure the file exists in tools directory: %s", name+".md", filepath.Join(promptsDir, "tools"))
	}

	return strings.TrimSpace(string(content))
}