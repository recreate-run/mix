package config

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"

	"mix/internal/llm/models"
	"mix/internal/preferences"
)

// Test helper functions
func setupTempConfig(t *testing.T) (string, func()) {
	// Create temporary directory for testing
	tempDir, err := os.MkdirTemp("", "config_test_*")
	assert.NoError(t, err)

	// Reset global config for clean test state
	cfgMutex.Lock()
	cfg = nil
	userPreferencesService = nil
	apiCredentialsService = nil
	cfgMutex.Unlock()

	// Cleanup function
	cleanup := func() {
		cfgMutex.Lock()
		cfg = nil
		userPreferencesService = nil
		apiCredentialsService = nil
		cfgMutex.Unlock()
		_ = os.RemoveAll(tempDir)
	}

	return tempDir, cleanup
}

// Test Load function
func TestLoad(t *testing.T) {
	tempDir, cleanup := setupTempConfig(t)
	defer cleanup()

	config, err := Load(tempDir, false, false)

	assert.NoError(t, err)
	assert.NotNil(t, config)
	assert.Equal(t, tempDir, config.WorkingDir)
	assert.Equal(t, defaultDataDirectory, config.Data.Directory)
	assert.False(t, config.Debug)
	assert.False(t, config.SkipPermissions)
	assert.Contains(t, config.PromptsDir, ".mix/prompts")
	assert.NotEmpty(t, config.MCPServers)
}

// Test Load with debug mode
func TestLoadWithDebug(t *testing.T) {
	tempDir, cleanup := setupTempConfig(t)
	defer cleanup()

	config, err := Load(tempDir, true, false)

	assert.NoError(t, err)
	assert.True(t, config.Debug)
}

// Test Load with skip permissions
func TestLoadWithSkipPermissions(t *testing.T) {
	tempDir, cleanup := setupTempConfig(t)
	defer cleanup()

	config, err := Load(tempDir, false, true)

	assert.NoError(t, err)
	assert.True(t, config.SkipPermissions)
}

// Test Load called multiple times returns same config
func TestLoadIdempotent(t *testing.T) {
	tempDir, cleanup := setupTempConfig(t)
	defer cleanup()

	config1, err1 := Load(tempDir, false, false)
	config2, err2 := Load(tempDir, false, false)

	assert.NoError(t, err1)
	assert.NoError(t, err2)
	assert.Equal(t, config1, config2)
}

// Test Get function
func TestGet(t *testing.T) {
	tempDir, cleanup := setupTempConfig(t)
	defer cleanup()

	// Should return nil when not loaded
	assert.Nil(t, Get())

	// Load config
	originalConfig, err := Load(tempDir, false, false)
	assert.NoError(t, err)

	// Get should return same config
	retrievedConfig := Get()
	assert.Equal(t, originalConfig, retrievedConfig)
}

// Test InitUserPreferences
func TestInitUserPreferences(t *testing.T) {
	t.Skip("Skipping InitUserPreferences test - requires actual database connection")
}

// Test InitAPICredentials
func TestInitAPICredentials(t *testing.T) {
	t.Skip("Skipping InitAPICredentials test - requires actual database connection")
}

// Test GetUserPreferences before initialization
func TestGetUserPreferencesNotInitialized(t *testing.T) {
	_, cleanup := setupTempConfig(t)
	defer cleanup()

	assert.Nil(t, GetUserPreferences())
}

// Test GetAPICredentials before initialization
func TestGetAPICredentialsNotInitialized(t *testing.T) {
	_, cleanup := setupTempConfig(t)
	defer cleanup()

	assert.Nil(t, GetAPICredentials())
}

// Test GetAgentFromDatabase with mock service
func TestGetAgentFromDatabase(t *testing.T) {
	_, cleanup := setupTempConfig(t)
	defer cleanup()

	// Create mock preferences service
	mockService := &preferences.MockService{}
	userPreferencesService = mockService

	expectedAgent := preferences.Agent{
		Model:           models.ModelID("claude-4-sonnet"),
		MaxTokens:       4096,
		ReasoningEffort: "medium",
	}

	mockService.On("GetAgentConfig", mock.Anything, preferences.AgentMain).
		Return(expectedAgent, nil)

	agent, err := GetAgentFromDatabase(context.Background(), AgentMain)

	assert.NoError(t, err)
	assert.Equal(t, Agent{
		Model:           expectedAgent.Model,
		MaxTokens:       expectedAgent.MaxTokens,
		ReasoningEffort: expectedAgent.ReasoningEffort,
	}, agent)

	mockService.AssertExpectations(t)
}

// Test GetAgentFromDatabase with uninitialized service
func TestGetAgentFromDatabaseNotInitialized(t *testing.T) {
	_, cleanup := setupTempConfig(t)
	defer cleanup()

	_, err := GetAgentFromDatabase(context.Background(), AgentMain)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "user preferences service not initialized")
}

// Test GetAgentFromDatabase with unknown agent
func TestGetAgentFromDatabaseUnknownAgent(t *testing.T) {
	_, cleanup := setupTempConfig(t)
	defer cleanup()

	_, err := GetAgentFromDatabase(context.Background(), AgentName("unknown"))

	assert.Error(t, err)
	// Will fail at service initialization check before agent name validation
	assert.Contains(t, err.Error(), "user preferences service not initialized")
}

// Test GetAgentFromDatabase with unknown agent name but initialized service
func TestGetAgentFromDatabaseUnknownAgentWithService(t *testing.T) {
	_, cleanup := setupTempConfig(t)
	defer cleanup()

	// Create mock preferences service
	mockService := &preferences.MockService{}
	userPreferencesService = mockService

	_, err := GetAgentFromDatabase(context.Background(), AgentName("unknown"))

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "unknown agent name")
}

// Test GetEmbeddedPrompts
func TestGetEmbeddedPrompts(t *testing.T) {
	prompts := GetEmbeddedPrompts()

	assert.NotNil(t, prompts)
}

// Test PromptsDirectory
func TestPromptsDirectory(t *testing.T) {
	tempDir, cleanup := setupTempConfig(t)
	defer cleanup()

	// Should return error when config not loaded
	_, err := PromptsDirectory()
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "config not loaded")

	// Load config
	_, err = Load(tempDir, false, false)
	assert.NoError(t, err)

	// Should return prompts directory
	promptsDir, err := PromptsDirectory()
	assert.NoError(t, err)
	assert.Contains(t, promptsDir, ".mix/prompts")
}

// Test getShellPath function
func TestGetShellPath(t *testing.T) {
	// Test with SHELL environment variable set
	originalShell := os.Getenv("SHELL")
	defer func() {
		if err := os.Setenv("SHELL", originalShell); err != nil {
			t.Fatalf("failed to restore SHELL env var: %v", err)
		}
	}()

	if err := os.Setenv("SHELL", "/bin/zsh"); err != nil {
		t.Fatalf("failed to set SHELL env var: %v", err)
	}
	shellPath := getShellPath()
	assert.Equal(t, "/bin/zsh", shellPath)

	// Test with no SHELL environment variable
	if err := os.Unsetenv("SHELL"); err != nil {
		t.Fatalf("failed to unset SHELL env var: %v", err)
	}
	shellPath = getShellPath()
	assert.Equal(t, "/bin/bash", shellPath)
}

// Test getAnalyticsEnabled function
func TestGetAnalyticsEnabled(t *testing.T) {
	originalAnalytics := os.Getenv("MIX_ANALYTICS_ENABLED")
	defer func() {
		if err := os.Setenv("MIX_ANALYTICS_ENABLED", originalAnalytics); err != nil {
			t.Fatalf("failed to restore MIX_ANALYTICS_ENABLED env var: %v", err)
		}
	}()

	// Test with analytics enabled
	if err := os.Setenv("MIX_ANALYTICS_ENABLED", "true"); err != nil {
		t.Fatalf("failed to set MIX_ANALYTICS_ENABLED env var: %v", err)
	}
	enabled := getAnalyticsEnabled()
	assert.True(t, enabled)

	// Test with analytics disabled
	if err := os.Setenv("MIX_ANALYTICS_ENABLED", "false"); err != nil {
		t.Fatalf("failed to set MIX_ANALYTICS_ENABLED env var: %v", err)
	}
	enabled = getAnalyticsEnabled()
	assert.False(t, enabled)

	// Test with analytics set to "1"
	if err := os.Setenv("MIX_ANALYTICS_ENABLED", "1"); err != nil {
		t.Fatalf("failed to set MIX_ANALYTICS_ENABLED env var: %v", err)
	}
	enabled = getAnalyticsEnabled()
	assert.True(t, enabled)

	// Test with no environment variable (should default to true)
	if err := os.Unsetenv("MIX_ANALYTICS_ENABLED"); err != nil {
		t.Fatalf("failed to unset MIX_ANALYTICS_ENABLED env var: %v", err)
	}
	enabled = getAnalyticsEnabled()
	assert.True(t, enabled)
}

// Test getDefaultMCPServers function
func TestGetDefaultMCPServers(t *testing.T) {
	servers := getDefaultMCPServers()

	assert.NotEmpty(t, servers)
	assert.Contains(t, servers, "blender")

	blenderServer := servers["blender"]
	assert.Equal(t, MCPStdio, blenderServer.Type)
	assert.Equal(t, "uvx", blenderServer.Command)
	assert.Contains(t, blenderServer.Args, "blender-mcp")
	assert.Contains(t, blenderServer.AllowedTools, "execute_blender_code")
}

// Test applyDefaultValues function
func TestApplyDefaultValues(t *testing.T) {
	_, cleanup := setupTempConfig(t)
	defer cleanup()

	// Create config with MCP server without type
	cfg = &Config{
		MCPServers: map[string]MCPServer{
			"test-server": {
				Command: "test-command",
				// Type not set
			},
		},
	}

	applyDefaultValues()

	// Should have applied default MCP type
	assert.Equal(t, MCPStdio, cfg.MCPServers["test-server"].Type)
}

// Test ShouldShowInitDialog
func TestShouldShowInitDialog(t *testing.T) {
	tempDir, cleanup := setupTempConfig(t)
	defer cleanup()

	// Should return error when config not loaded
	_, err := ShouldShowInitDialog()
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "config not loaded")

	// Load config
	_, err = Load(tempDir, false, false)
	assert.NoError(t, err)

	// Set data directory to temp directory for testing
	cfg.Data.Directory = tempDir

	// Create the data directory for testing
	err = os.MkdirAll(cfg.Data.Directory, 0755)
	assert.NoError(t, err)

	// Should show dialog when init flag doesn't exist
	shouldShow, err := ShouldShowInitDialog()
	assert.NoError(t, err)
	assert.True(t, shouldShow)

	// Mark as initialized
	err = MarkProjectInitialized()
	assert.NoError(t, err)

	// Should not show dialog when init flag exists
	shouldShow, err = ShouldShowInitDialog()
	assert.NoError(t, err)
	assert.False(t, shouldShow)
}

// Test MarkProjectInitialized
func TestMarkProjectInitialized(t *testing.T) {
	tempDir, cleanup := setupTempConfig(t)
	defer cleanup()

	// Should return error when config not loaded
	err := MarkProjectInitialized()
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "config not loaded")

	// Load config
	_, err = Load(tempDir, false, false)
	assert.NoError(t, err)

	// Set data directory to temp directory for testing
	cfg.Data.Directory = tempDir

	// Create the data directory for testing
	err = os.MkdirAll(cfg.Data.Directory, 0755)
	assert.NoError(t, err)

	// Should successfully mark as initialized
	err = MarkProjectInitialized()
	assert.NoError(t, err)

	// Verify init flag file exists
	flagFilePath := filepath.Join(cfg.Data.Directory, InitFlagFilename)
	_, err = os.Stat(flagFilePath)
	assert.NoError(t, err)
}

// Test EnsurePromptsDirectory
func TestEnsurePromptsDirectory(t *testing.T) {
	tempDir, cleanup := setupTempConfig(t)
	defer cleanup()

	// Should return error when config not loaded
	err := EnsurePromptsDirectory()
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "config not loaded")

	// Load config
	_, err = Load(tempDir, false, false)
	assert.NoError(t, err)

	// Should successfully create prompts directory structure
	err = EnsurePromptsDirectory()
	assert.NoError(t, err)

	// Verify prompts directory exists
	_, err = os.Stat(cfg.PromptsDir)
	assert.NoError(t, err)

	// Verify tools subdirectory exists
	toolsDir := filepath.Join(cfg.PromptsDir, "tools")
	_, err = os.Stat(toolsDir)
	assert.NoError(t, err)
}

// Test ensureEmbeddedDataDirectory function
func TestEnsureEmbeddedDataDirectory(t *testing.T) {
	// This function creates .mix directory in home directory
	// We'll test it creates directory without errors
	err := ensureEmbeddedDataDirectory()
	assert.NoError(t, err)

	// Verify .mix directory was created in home directory
	homeDir, err := os.UserHomeDir()
	assert.NoError(t, err)

	mixDir := filepath.Join(homeDir, ".mix")
	_, err = os.Stat(mixDir)
	assert.NoError(t, err)
}
