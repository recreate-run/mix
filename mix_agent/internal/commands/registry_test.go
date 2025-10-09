package commands

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"mix/internal/app"
)

// MockCommand implements Command interface for testing
type MockCommand struct {
	name        string
	description string
	executeFunc func(ctx context.Context, args string) (string, error)
}

func (m *MockCommand) Name() string {
	return m.name
}

func (m *MockCommand) Description() string {
	return m.description
}

func (m *MockCommand) Execute(ctx context.Context, args string) (string, error) {
	if m.executeFunc != nil {
		return m.executeFunc(ctx, args)
	}
	return "mock result", nil
}

// Test NewRegistry
func TestNewRegistry(t *testing.T) {
	registry := NewRegistry()

	assert.NotNil(t, registry)
	assert.NotNil(t, registry.commands)
	assert.Empty(t, registry.commands)
}

// Test Registry command operations
func TestRegistryGetCommand(t *testing.T) {
	registry := NewRegistry()

	// Test getting non-existent command
	cmd, exists := registry.GetCommand("nonexistent")
	assert.Nil(t, cmd)
	assert.False(t, exists)

	// Add a mock command
	mockCmd := &MockCommand{
		name:        "test-cmd",
		description: "Test command",
	}
	registry.commands["test-cmd"] = mockCmd

	// Test getting existing command
	cmd, exists = registry.GetCommand("test-cmd")
	assert.NotNil(t, cmd)
	assert.True(t, exists)
	assert.Equal(t, "test-cmd", cmd.Name())
}

func TestRegistryGetAllCommands(t *testing.T) {
	registry := NewRegistry()

	// Initially empty
	commands := registry.GetAllCommands()
	assert.Empty(t, commands)

	// Add commands
	cmd1 := &MockCommand{name: "cmd1", description: "Command 1"}
	cmd2 := &MockCommand{name: "cmd2", description: "Command 2"}

	registry.commands["cmd1"] = cmd1
	registry.commands["cmd2"] = cmd2

	// Test getting all commands
	commands = registry.GetAllCommands()
	assert.Len(t, commands, 2)
	assert.Contains(t, commands, "cmd1")
	assert.Contains(t, commands, "cmd2")

	// Verify returned map is a copy (not reference)
	commands["cmd3"] = &MockCommand{name: "cmd3", description: "Command 3"}
	originalCommands := registry.GetAllCommands()
	assert.Len(t, originalCommands, 2) // Should still be 2
}

func TestRegistryExecuteCommand(t *testing.T) {
	registry := NewRegistry()

	// Test executing non-existent command
	result, err := registry.ExecuteCommand(context.Background(), "nonexistent", "args")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "command not found")
	assert.Empty(t, result)

	// Add mock command that succeeds
	successCmd := &MockCommand{
		name:        "success",
		description: "Success command",
		executeFunc: func(ctx context.Context, args string) (string, error) {
			return "success with " + args, nil
		},
	}
	registry.commands["success"] = successCmd

	// Test successful execution
	result, err = registry.ExecuteCommand(context.Background(), "success", "test args")
	assert.NoError(t, err)
	assert.Equal(t, "success with test args", result)

	// Add mock command that fails
	failCmd := &MockCommand{
		name:        "fail",
		description: "Fail command",
		executeFunc: func(ctx context.Context, args string) (string, error) {
			return "", assert.AnError
		},
	}
	registry.commands["fail"] = failCmd

	// Test failed execution
	result, err = registry.ExecuteCommand(context.Background(), "fail", "args")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "command execution failed")
	assert.Empty(t, result)
}

// Test LoadCommands with file system
func TestRegistryLoadCommandsFromDir(t *testing.T) {
	registry := NewRegistry()
	tempDir, cleanup := setupTempDir(t)
	defer cleanup()

	// Create test command files
	createTestCommandFile(t, tempDir, "test1.md", "Test command 1")
	createTestCommandFile(t, tempDir, "test2.md", "Test command 2")

	// Create subdirectory with command
	subDir := filepath.Join(tempDir, "sub")
	err := os.MkdirAll(subDir, 0755)
	require.NoError(t, err)
	createTestCommandFile(t, subDir, "subcmd.md", "Sub command")

	// Test loading commands from directory
	err = registry.loadCommandsFromDir(tempDir, "test")
	assert.NoError(t, err)

	// Verify commands were loaded with scope prefix
	cmd, exists := registry.GetCommand("test:test1")
	assert.True(t, exists)
	assert.Equal(t, "test1", cmd.Name())

	cmd, exists = registry.GetCommand("test:test2")
	assert.True(t, exists)
	assert.Equal(t, "test2", cmd.Name())

	cmd, exists = registry.GetCommand("test:sub:subcmd")
	assert.True(t, exists)
	assert.Equal(t, "sub:subcmd", cmd.Name())

	// Verify commands are also available without prefix
	cmd, exists = registry.GetCommand("test1")
	assert.True(t, exists)
	assert.Equal(t, "test1", cmd.Name())

	cmd, exists = registry.GetCommand("sub:subcmd")
	assert.True(t, exists)
	assert.Equal(t, "sub:subcmd", cmd.Name())
}

func TestRegistryLoadCommandsFromDirNonexistent(t *testing.T) {
	registry := NewRegistry()

	err := registry.loadCommandsFromDir("/nonexistent/directory", "test")
	assert.NoError(t, err) // Should not error for non-existent directory
}

func TestRegistryLoadCommandsFromDirInvalidCommand(t *testing.T) {
	registry := NewRegistry()
	tempDir, cleanup := setupTempDir(t)
	defer cleanup()

	// Create invalid command file
	invalidContent := `---
invalid: yaml: content: here
---`
	createTestCommandFile(t, tempDir, "invalid.md", invalidContent)

	err := registry.loadCommandsFromDir(tempDir, "test")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to load command")
}

// Test LoadCommands with mock app (testing the integration)
func TestRegistryLoadCommandsIntegration(t *testing.T) {
	registry := NewRegistry()

	// Create a temporary directory structure for testing
	tempHomeDir, cleanup := setupTempDir(t)
	defer cleanup()

	// Set temporary home directory for this test
	originalHome := os.Getenv("HOME")
	os.Setenv("HOME", tempHomeDir)
	defer os.Setenv("HOME", originalHome)

	// Create user commands directory
	userCommandsDir := filepath.Join(tempHomeDir, ".mix", "commands")
	err := os.MkdirAll(userCommandsDir, 0755)
	require.NoError(t, err)
	createTestCommandFile(t, userCommandsDir, "user-cmd.md", "User command")

	// Create project commands directory
	projectCommandsDir := ".mix/commands"
	err = os.MkdirAll(projectCommandsDir, 0755)
	require.NoError(t, err)
	defer os.RemoveAll(".mix")
	createTestCommandFile(t, projectCommandsDir, "project-cmd.md", "Project command")

	// Mock app
	mockApp := &app.App{} // Assuming minimal app struct for this test

	// Load commands
	err = registry.LoadCommands(mockApp)
	assert.NoError(t, err)

	// Verify project commands were loaded
	cmd, exists := registry.GetCommand("project:project-cmd")
	assert.True(t, exists)
	assert.Equal(t, "project-cmd", cmd.Name())

	// Verify user commands were loaded
	cmd, exists = registry.GetCommand("user:user-cmd")
	assert.True(t, exists)
	assert.Equal(t, "user-cmd", cmd.Name())

	// Verify builtin commands were loaded (they come from GetBuiltinCommands)
	// Note: We can't easily test this without mocking GetBuiltinCommands
	// but the integration should work
}

// Test command name conflicts and precedence
func TestRegistryCommandNamePrecedence(t *testing.T) {
	registry := NewRegistry()

	// Add commands with same name but different scopes
	cmd1 := &MockCommand{name: "cmd", description: "First command"}
	cmd2 := &MockCommand{name: "cmd", description: "Second command"}

	registry.commands["scope1:cmd"] = cmd1
	registry.commands["scope2:cmd"] = cmd2

	// Last one should win for unscoped access
	registry.commands["cmd"] = cmd2

	// Test scoped access works
	retrievedCmd1, exists := registry.GetCommand("scope1:cmd")
	assert.True(t, exists)
	assert.Equal(t, "First command", retrievedCmd1.Description())

	retrievedCmd2, exists := registry.GetCommand("scope2:cmd")
	assert.True(t, exists)
	assert.Equal(t, "Second command", retrievedCmd2.Description())

	// Test unscoped access gets the last registered
	retrievedCmd, exists := registry.GetCommand("cmd")
	assert.True(t, exists)
	assert.Equal(t, "Second command", retrievedCmd.Description())
}