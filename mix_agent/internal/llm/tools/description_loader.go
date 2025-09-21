package tools

import (
	"fmt"
	"path/filepath"
	"strings"

	"mix/internal/config"
)

// LoadToolDescription loads a tool description from embedded filesystem
func LoadToolDescription(name string) string {
	embeddedFS := config.GetEmbeddedPrompts()
	toolPath := filepath.Join("prompts", "tools", name+".md")
	
	content, err := embeddedFS.ReadFile(toolPath)
	if err != nil {
		return fmt.Sprintf("Error: failed to load embedded tool description '%s': %v", name, err)
	}

	return strings.TrimSpace(string(content))
}