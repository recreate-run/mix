package tools

import (
	"fmt"
	"strings"
)

// ToolType represents different categories of tools
type ToolType string

const (
	ToolTypeWebSearch          ToolType = "web_search"
	ToolTypeMultimodalAnalyzer ToolType = "multimodal_analyzer"
)

// ToolProvider represents a specific provider within a tool category
type ToolProvider string

const (
	// Web Search providers
	WebSearchBrave ToolProvider = "brave"

	// Multimodal Analyzer providers
	MultimodalGemini ToolProvider = "gemini"
	MultimodalOpenAI ToolProvider = "openai"

	// Future providers can be added here
)

// ToolInfo represents information about a tool
type ToolInfo struct {
	Type         ToolType     `json:"type"`
	Provider     ToolProvider `json:"provider"`
	DisplayName  string       `json:"display_name"`
	Description  string       `json:"description"`
	APIKeyFormat string       `json:"api_key_format"`
	Enabled      bool         `json:"enabled"`
	RequiresKey  bool         `json:"requires_key"`
}

// ToolCategory represents a category of tools for UI display
type ToolCategory struct {
	Type        ToolType   `json:"type"`
	DisplayName string     `json:"display_name"`
	Description string     `json:"description"`
	Icon        string     `json:"icon"`
	Tools       []ToolInfo `json:"tools"`
}

// ToolRegistry manages all available tools and their configurations
type ToolRegistry struct {
	tools      map[ToolType]map[ToolProvider]ToolInfo
	categories map[ToolType]ToolCategory
}

// NewToolRegistry creates a new tool registry with predefined tools
func NewToolRegistry() *ToolRegistry {
	registry := &ToolRegistry{
		tools:      make(map[ToolType]map[ToolProvider]ToolInfo),
		categories: make(map[ToolType]ToolCategory),
	}

	registry.initializeTools()
	registry.initializeCategories()

	return registry
}

// initializeTools sets up all available tools
func (tr *ToolRegistry) initializeTools() {
	// Web Search Tools
	tr.registerTool(ToolInfo{
		Type:         ToolTypeWebSearch,
		Provider:     WebSearchBrave,
		DisplayName:  "Brave Search",
		Description:  "Privacy-focused web search with real-time results",
		APIKeyFormat: "BSA...",
		Enabled:      true,
		RequiresKey:  true,
	})

	// Multimodal Analyzer Tools
	tr.registerTool(ToolInfo{
		Type:         ToolTypeMultimodalAnalyzer,
		Provider:     MultimodalGemini,
		DisplayName:  "Gemini Vision",
		Description:  "Google's multimodal AI for image and video analysis",
		APIKeyFormat: "AI...",
		Enabled:      true,
		RequiresKey:  true,
	})

	tr.registerTool(ToolInfo{
		Type:         ToolTypeMultimodalAnalyzer,
		Provider:     MultimodalOpenAI,
		DisplayName:  "GPT-4 Vision",
		Description:  "OpenAI's vision model for image analysis",
		APIKeyFormat: "sk-...",
		Enabled:      false, // Disabled for now as it uses regular OpenAI provider
		RequiresKey:  false, // Uses existing OpenAI key
	})
}

// initializeCategories sets up tool categories for UI display
func (tr *ToolRegistry) initializeCategories() {
	tr.categories[ToolTypeWebSearch] = ToolCategory{
		Type:        ToolTypeWebSearch,
		DisplayName: "Web Search",
		Description: "Search the web for real-time information",
		Icon:        "🔍",
		Tools:       tr.getToolsByType(ToolTypeWebSearch),
	}

	tr.categories[ToolTypeMultimodalAnalyzer] = ToolCategory{
		Type:        ToolTypeMultimodalAnalyzer,
		DisplayName: "Read media",
		Description: "Analyze images, videos, and other media",
		Icon:        "👁️",
		Tools:       tr.getToolsByType(ToolTypeMultimodalAnalyzer),
	}
}

// registerTool adds a tool to the registry
func (tr *ToolRegistry) registerTool(tool ToolInfo) {
	if tr.tools[tool.Type] == nil {
		tr.tools[tool.Type] = make(map[ToolProvider]ToolInfo)
	}
	tr.tools[tool.Type][tool.Provider] = tool
}

// GetTool retrieves a specific tool by type and provider
func (tr *ToolRegistry) GetTool(toolType ToolType, provider ToolProvider) (ToolInfo, bool) {
	if typeMap, exists := tr.tools[toolType]; exists {
		if tool, exists := typeMap[provider]; exists {
			return tool, true
		}
	}
	return ToolInfo{}, false
}

// GetToolsByType returns all tools of a specific type
func (tr *ToolRegistry) getToolsByType(toolType ToolType) []ToolInfo {
	var tools []ToolInfo
	if typeMap, exists := tr.tools[toolType]; exists {
		for _, tool := range typeMap {
			if tool.Enabled {
				tools = append(tools, tool)
			}
		}
	}
	return tools
}

// GetAllCategories returns all tool categories
func (tr *ToolRegistry) GetAllCategories() []ToolCategory {
	var categories []ToolCategory
	for _, category := range tr.categories {
		// Only include categories that have enabled tools
		if len(category.Tools) > 0 {
			categories = append(categories, category)
		}
	}
	return categories
}

// GetCategory returns a specific category
func (tr *ToolRegistry) GetCategory(toolType ToolType) (ToolCategory, bool) {
	category, exists := tr.categories[toolType]
	if exists {
		// Refresh tools list
		category.Tools = tr.getToolsByType(toolType)
	}
	return category, exists
}

// ValidateAPIKey validates an API key format for a specific tool
func (tr *ToolRegistry) ValidateAPIKey(toolType ToolType, provider ToolProvider, apiKey string) error {
	tool, exists := tr.GetTool(toolType, provider)
	if !exists {
		return fmt.Errorf("tool not found: %s/%s", toolType, provider)
	}

	if !tool.RequiresKey {
		return nil // No validation needed
	}

	if apiKey == "" {
		return fmt.Errorf("API key is required for %s", tool.DisplayName)
	}

	// Validate format based on provider
	switch provider {
	case WebSearchBrave:
		if !strings.HasPrefix(apiKey, "BSA") {
			return fmt.Errorf("brave API key must start with 'BSA'")
		}
	case MultimodalGemini:
		if !strings.HasPrefix(apiKey, "AI") {
			return fmt.Errorf("gemini API key must start with 'AI'")
		}
	default:
		// Generic validation
		if len(apiKey) < 10 {
			return fmt.Errorf("API key is too short")
		}
	}

	return nil
}

// GetProviderIdentifier returns a string identifier for database storage
func GetProviderIdentifier(toolType ToolType, provider ToolProvider) string {
	return fmt.Sprintf("%s_%s", toolType, provider)
}

// ParseProviderIdentifier extracts tool type and provider from identifier
func ParseProviderIdentifier(identifier string) (ToolType, ToolProvider, error) {
	parts := strings.SplitN(identifier, "_", 2)
	if len(parts) != 2 {
		return "", "", fmt.Errorf("invalid provider identifier: %s", identifier)
	}
	return ToolType(parts[0]), ToolProvider(parts[1]), nil
}

// Global registry instance
var defaultRegistry *ToolRegistry

// GetRegistry returns the global tool registry
func GetRegistry() *ToolRegistry {
	if defaultRegistry == nil {
		defaultRegistry = NewToolRegistry()
	}
	return defaultRegistry
}
