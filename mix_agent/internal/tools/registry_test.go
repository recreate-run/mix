package tools

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Test ToolType constants
func TestToolTypeConstants(t *testing.T) {
	assert.Equal(t, ToolType("web_search"), ToolTypeWebSearch)
	assert.Equal(t, ToolType("multimodal_analyzer"), ToolTypeMultimodalAnalyzer)
}

// Test ToolProvider constants
func TestToolProviderConstants(t *testing.T) {
	assert.Equal(t, ToolProvider("brave"), WebSearchBrave)
	assert.Equal(t, ToolProvider("gemini"), MultimodalGemini)
	assert.Equal(t, ToolProvider("openai"), MultimodalOpenAI)
}

// Test ToolInfo structure
func TestToolInfo(t *testing.T) {
	tool := ToolInfo{
		Type:         ToolTypeWebSearch,
		Provider:     WebSearchBrave,
		DisplayName:  "Test Tool",
		Description:  "A test tool",
		APIKeyFormat: "TEST...",
		Enabled:      true,
		RequiresKey:  true,
	}

	assert.Equal(t, ToolTypeWebSearch, tool.Type)
	assert.Equal(t, WebSearchBrave, tool.Provider)
	assert.Equal(t, "Test Tool", tool.DisplayName)
	assert.Equal(t, "A test tool", tool.Description)
	assert.Equal(t, "TEST...", tool.APIKeyFormat)
	assert.True(t, tool.Enabled)
	assert.True(t, tool.RequiresKey)
}

// Test ToolCategory structure
func TestToolCategory(t *testing.T) {
	tools := []ToolInfo{
		{
			Type:        ToolTypeWebSearch,
			Provider:    WebSearchBrave,
			DisplayName: "Tool 1",
			Enabled:     true,
		},
	}

	category := ToolCategory{
		Type:        ToolTypeWebSearch,
		DisplayName: "Test Category",
		Description: "A test category",
		Icon:        "🔍",
		Tools:       tools,
	}

	assert.Equal(t, ToolTypeWebSearch, category.Type)
	assert.Equal(t, "Test Category", category.DisplayName)
	assert.Equal(t, "A test category", category.Description)
	assert.Equal(t, "🔍", category.Icon)
	assert.Len(t, category.Tools, 1)
}

// Test NewToolRegistry
func TestNewToolRegistry(t *testing.T) {
	registry := NewToolRegistry()

	assert.NotNil(t, registry)
	assert.NotNil(t, registry.tools)
	assert.NotNil(t, registry.categories)

	// Verify that tools are initialized
	braveSearch, exists := registry.GetTool(ToolTypeWebSearch, WebSearchBrave)
	assert.True(t, exists)
	assert.Equal(t, "Brave Search", braveSearch.DisplayName)
	assert.True(t, braveSearch.Enabled)
	assert.True(t, braveSearch.RequiresKey)

	geminiVision, exists := registry.GetTool(ToolTypeMultimodalAnalyzer, MultimodalGemini)
	assert.True(t, exists)
	assert.Equal(t, "Gemini Vision", geminiVision.DisplayName)
	assert.True(t, geminiVision.Enabled)
	assert.True(t, geminiVision.RequiresKey)

	gpt4Vision, exists := registry.GetTool(ToolTypeMultimodalAnalyzer, MultimodalOpenAI)
	assert.True(t, exists)
	assert.Equal(t, "GPT-4 Vision", gpt4Vision.DisplayName)
	assert.False(t, gpt4Vision.Enabled) // Should be disabled
	assert.False(t, gpt4Vision.RequiresKey)
}

// Test registerTool
func TestRegisterTool(t *testing.T) {
	registry := &ToolRegistry{
		tools:      make(map[ToolType]map[ToolProvider]ToolInfo),
		categories: make(map[ToolType]ToolCategory),
	}

	tool := ToolInfo{
		Type:        ToolTypeWebSearch,
		Provider:    WebSearchBrave,
		DisplayName: "Test Tool",
		Enabled:     true,
	}

	registry.registerTool(tool)

	retrievedTool, exists := registry.GetTool(ToolTypeWebSearch, WebSearchBrave)
	assert.True(t, exists)
	assert.Equal(t, tool, retrievedTool)
}

// Test GetTool
func TestGetTool(t *testing.T) {
	registry := NewToolRegistry()

	// Test existing tool
	tool, exists := registry.GetTool(ToolTypeWebSearch, WebSearchBrave)
	assert.True(t, exists)
	assert.Equal(t, "Brave Search", tool.DisplayName)

	// Test non-existing tool type
	_, exists = registry.GetTool(ToolType("non_existent"), WebSearchBrave)
	assert.False(t, exists)

	// Test non-existing provider
	_, exists = registry.GetTool(ToolTypeWebSearch, ToolProvider("non_existent"))
	assert.False(t, exists)
}

// Test getToolsByType
func TestGetToolsByType(t *testing.T) {
	registry := NewToolRegistry()

	// Test web search tools
	webSearchTools := registry.getToolsByType(ToolTypeWebSearch)
	assert.Len(t, webSearchTools, 1) // Only enabled tools
	assert.Equal(t, "Brave Search", webSearchTools[0].DisplayName)

	// Test multimodal tools
	multimodalTools := registry.getToolsByType(ToolTypeMultimodalAnalyzer)
	assert.Len(t, multimodalTools, 1) // Only Gemini should be enabled
	assert.Equal(t, "Gemini Vision", multimodalTools[0].DisplayName)

	// Test non-existent type
	nonExistentTools := registry.getToolsByType(ToolType("non_existent"))
	assert.Empty(t, nonExistentTools)
}

// Test GetAllCategories
func TestGetAllCategories(t *testing.T) {
	registry := NewToolRegistry()

	categories := registry.GetAllCategories()

	// Should only include categories with enabled tools
	assert.Greater(t, len(categories), 0)

	// Verify web search category exists
	var webSearchCategory *ToolCategory
	for _, category := range categories {
		if category.Type == ToolTypeWebSearch {
			webSearchCategory = &category
			break
		}
	}

	require.NotNil(t, webSearchCategory)
	assert.Equal(t, "Web Search", webSearchCategory.DisplayName)
	assert.Equal(t, "🔍", webSearchCategory.Icon)
	assert.Len(t, webSearchCategory.Tools, 1)
}

// Test GetCategory
func TestGetCategory(t *testing.T) {
	registry := NewToolRegistry()

	// Test existing category
	category, exists := registry.GetCategory(ToolTypeWebSearch)
	assert.True(t, exists)
	assert.Equal(t, "Web Search", category.DisplayName)
	assert.Len(t, category.Tools, 1)

	// Test non-existing category
	_, exists = registry.GetCategory(ToolType("non_existent"))
	assert.False(t, exists)
}

// Test ValidateAPIKey
func TestValidateAPIKey(t *testing.T) {
	registry := NewToolRegistry()

	tests := []struct {
		name        string
		toolType    ToolType
		provider    ToolProvider
		apiKey      string
		expectError bool
		errorMsg    string
	}{
		{
			name:        "valid Brave API key",
			toolType:    ToolTypeWebSearch,
			provider:    WebSearchBrave,
			apiKey:      "BSA123456789",
			expectError: false,
		},
		{
			name:        "invalid Brave API key format",
			toolType:    ToolTypeWebSearch,
			provider:    WebSearchBrave,
			apiKey:      "INVALID123",
			expectError: true,
			errorMsg:    "brave API key must start with 'BSA'",
		},
		{
			name:        "empty API key for required tool",
			toolType:    ToolTypeWebSearch,
			provider:    WebSearchBrave,
			apiKey:      "",
			expectError: true,
			errorMsg:    "API key is required for Brave Search",
		},
		{
			name:        "valid Gemini API key",
			toolType:    ToolTypeMultimodalAnalyzer,
			provider:    MultimodalGemini,
			apiKey:      "AI123456789",
			expectError: false,
		},
		{
			name:        "invalid Gemini API key format",
			toolType:    ToolTypeMultimodalAnalyzer,
			provider:    MultimodalGemini,
			apiKey:      "INVALID123",
			expectError: true,
			errorMsg:    "Gemini API key must start with 'AI'",
		},
		{
			name:        "tool not found",
			toolType:    ToolType("non_existent"),
			provider:    ToolProvider("non_existent"),
			apiKey:      "any_key",
			expectError: true,
			errorMsg:    "tool not found",
		},
		{
			name:        "tool doesn't require key",
			toolType:    ToolTypeMultimodalAnalyzer,
			provider:    MultimodalOpenAI,
			apiKey:      "",
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := registry.ValidateAPIKey(tt.toolType, tt.provider, tt.apiKey)

			if tt.expectError {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tt.errorMsg)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

// Test ValidateAPIKey with generic validation
func TestValidateAPIKeyGeneric(t *testing.T) {
	registry := &ToolRegistry{
		tools: make(map[ToolType]map[ToolProvider]ToolInfo),
	}

	// Register a custom tool for generic validation
	customTool := ToolInfo{
		Type:        ToolType("custom"),
		Provider:    ToolProvider("custom_provider"),
		DisplayName: "Custom Tool",
		RequiresKey: true,
	}
	registry.registerTool(customTool)

	// Test generic validation (too short)
	err := registry.ValidateAPIKey(ToolType("custom"), ToolProvider("custom_provider"), "short")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "API key is too short")

	// Test generic validation (valid length)
	err = registry.ValidateAPIKey(ToolType("custom"), ToolProvider("custom_provider"), "long_enough_key")
	assert.NoError(t, err)
}

// Test GetProviderIdentifier
func TestGetProviderIdentifier(t *testing.T) {
	tests := []struct {
		toolType ToolType
		provider ToolProvider
		expected string
	}{
		{
			toolType: ToolTypeWebSearch,
			provider: WebSearchBrave,
			expected: "web_search_brave",
		},
		{
			toolType: ToolTypeMultimodalAnalyzer,
			provider: MultimodalGemini,
			expected: "multimodal_analyzer_gemini",
		},
	}

	for _, tt := range tests {
		result := GetProviderIdentifier(tt.toolType, tt.provider)
		assert.Equal(t, tt.expected, result)
	}
}

// Test ParseProviderIdentifier
func TestParseProviderIdentifier(t *testing.T) {
	tests := []struct {
		name        string
		identifier  string
		expectedType ToolType
		expectedProv ToolProvider
		expectError bool
	}{
		{
			name:         "valid identifier",
			identifier:   "web_search_brave",
			expectedType: ToolType("web"),
			expectedProv: ToolProvider("search_brave"),
			expectError:  false,
		},
		{
			name:         "valid identifier with multiple underscores",
			identifier:   "multimodal_analyzer_gemini",
			expectedType: ToolType("multimodal"),
			expectedProv: ToolProvider("analyzer_gemini"),
			expectError:  false,
		},
		{
			name:        "simple valid identifier",
			identifier:  "type_provider",
			expectedType: ToolType("type"),
			expectedProv: ToolProvider("provider"),
			expectError: false,
		},
		{
			name:        "invalid identifier - no underscore",
			identifier:  "invalid",
			expectError: true,
		},
		{
			name:        "invalid identifier - empty",
			identifier:  "",
			expectError: true,
		},
		{
			name:        "edge case - only underscore",
			identifier:  "_",
			expectedType: ToolType(""),
			expectedProv: ToolProvider(""),
			expectError: false, // Function doesn't validate empty parts
		},
		{
			name:        "edge case - ends with underscore",
			identifier:  "type_",
			expectedType: ToolType("type"),
			expectedProv: ToolProvider(""),
			expectError: false, // Function doesn't validate empty parts
		},
		{
			name:        "edge case - starts with underscore",
			identifier:  "_provider",
			expectedType: ToolType(""),
			expectedProv: ToolProvider("provider"),
			expectError: false, // Function doesn't validate empty parts
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			toolType, provider, err := ParseProviderIdentifier(tt.identifier)

			if tt.expectError {
				assert.Error(t, err)
				if err != nil {
					assert.Contains(t, err.Error(), "invalid provider identifier")
				}
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.expectedType, toolType)
				assert.Equal(t, tt.expectedProv, provider)
			}
		})
	}
}

// Test GetRegistry global function
func TestGetRegistry(t *testing.T) {
	// Clear the global registry first
	defaultRegistry = nil

	registry1 := GetRegistry()
	assert.NotNil(t, registry1)

	// Should return the same instance
	registry2 := GetRegistry()
	assert.Same(t, registry1, registry2)

	// Verify it has the expected tools
	tool, exists := registry1.GetTool(ToolTypeWebSearch, WebSearchBrave)
	assert.True(t, exists)
	assert.Equal(t, "Brave Search", tool.DisplayName)
}

// Test initialization of all tool categories
func TestInitializeCategories(t *testing.T) {
	registry := NewToolRegistry()

	// Test all expected categories are initialized
	expectedCategories := []ToolType{
		ToolTypeWebSearch,
		ToolTypeMultimodalAnalyzer,
	}

	for _, expectedType := range expectedCategories {
		category, exists := registry.GetCategory(expectedType)
		assert.True(t, exists, "Category %s should exist", expectedType)
		assert.NotEmpty(t, category.DisplayName)
		assert.NotEmpty(t, category.Description)
		assert.NotEmpty(t, category.Icon)
	}
}

// Test tool registry state isolation
func TestToolRegistryStateIsolation(t *testing.T) {
	registry1 := NewToolRegistry()
	registry2 := NewToolRegistry()

	// Both should have the same initial state
	tool1, exists1 := registry1.GetTool(ToolTypeWebSearch, WebSearchBrave)
	tool2, exists2 := registry2.GetTool(ToolTypeWebSearch, WebSearchBrave)

	assert.True(t, exists1)
	assert.True(t, exists2)
	assert.Equal(t, tool1, tool2)

	// Register a tool in one registry
	customTool := ToolInfo{
		Type:        ToolType("custom"),
		Provider:    ToolProvider("test"),
		DisplayName: "Custom Test Tool",
		Enabled:     true,
	}

	registry1.registerTool(customTool)

	// Should exist in registry1 but not in registry2
	_, exists1 = registry1.GetTool(ToolType("custom"), ToolProvider("test"))
	_, exists2 = registry2.GetTool(ToolType("custom"), ToolProvider("test"))

	assert.True(t, exists1)
	assert.False(t, exists2)
}

// Test edge cases for category filtering
func TestCategoryFiltering(t *testing.T) {
	registry := &ToolRegistry{
		tools:      make(map[ToolType]map[ToolProvider]ToolInfo),
		categories: make(map[ToolType]ToolCategory),
	}

	// Add a category with no enabled tools
	registry.categories[ToolType("empty")] = ToolCategory{
		Type:        ToolType("empty"),
		DisplayName: "Empty Category",
		Tools:       []ToolInfo{},
	}

	// Add a category with only disabled tools
	disabledTool := ToolInfo{
		Type:     ToolType("disabled"),
		Provider: ToolProvider("test"),
		Enabled:  false,
	}
	registry.registerTool(disabledTool)
	registry.categories[ToolType("disabled")] = ToolCategory{
		Type:        ToolType("disabled"),
		DisplayName: "Disabled Category",
		Tools:       []ToolInfo{},
	}

	// GetAllCategories should filter out categories with no enabled tools
	categories := registry.GetAllCategories()
	assert.Empty(t, categories)
}