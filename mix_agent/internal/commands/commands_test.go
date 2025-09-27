package commands

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Test helper functions
func setupTempDir(t *testing.T) (string, func()) {
	tempDir, err := os.MkdirTemp("", "commands_test_*")
	require.NoError(t, err)

	cleanup := func() {
		os.RemoveAll(tempDir)
	}

	return tempDir, cleanup
}

func createTestCommandFile(t *testing.T, dir, filename, content string) string {
	filePath := filepath.Join(dir, filename)
	err := os.WriteFile(filePath, []byte(content), 0644)
	require.NoError(t, err)
	return filePath
}

// Test FileCommand creation and parsing
func TestNewFileCommand(t *testing.T) {
	tempDir, cleanup := setupTempDir(t)
	defer cleanup()

	content := `---
description: Test command
argument-hint: "<test-arg>"
allowed-tools: ["tool1", "tool2"]
---

This is a test command with $ARGUMENTS placeholder.`

	filePath := createTestCommandFile(t, tempDir, "test.md", content)

	cmd, err := NewFileCommand("test", filePath)

	assert.NoError(t, err)
	assert.NotNil(t, cmd)
	assert.Equal(t, "test", cmd.Name())
	assert.Equal(t, "Test command", cmd.Description())
	assert.Equal(t, "This is a test command with $ARGUMENTS placeholder.", cmd.content)
	assert.Equal(t, "Test command", cmd.metadata.Description)
	assert.Equal(t, "<test-arg>", cmd.metadata.ArgumentHint)
	assert.Equal(t, []string{"tool1", "tool2"}, cmd.metadata.AllowedTools)
}

func TestNewFileCommandWithoutFrontmatter(t *testing.T) {
	tempDir, cleanup := setupTempDir(t)
	defer cleanup()

	content := "Simple command without frontmatter"
	filePath := createTestCommandFile(t, tempDir, "simple.md", content)

	cmd, err := NewFileCommand("simple", filePath)

	assert.NoError(t, err)
	assert.Equal(t, "simple", cmd.Name())
	assert.Contains(t, cmd.Description(), "Custom command from simple.md")
	assert.Equal(t, content, cmd.content)
	assert.Empty(t, cmd.metadata.Description)
	assert.Empty(t, cmd.metadata.ArgumentHint)
	assert.Empty(t, cmd.metadata.AllowedTools)
}

func TestNewFileCommandWithIncompleteFrontmatter(t *testing.T) {
	tempDir, cleanup := setupTempDir(t)
	defer cleanup()

	content := `---
description: Test command
No closing frontmatter marker

This should be treated as regular content.`

	filePath := createTestCommandFile(t, tempDir, "incomplete.md", content)

	cmd, err := NewFileCommand("incomplete", filePath)

	assert.NoError(t, err)
	assert.Equal(t, content, cmd.content) // Should treat entire content as prompt
}

func TestNewFileCommandInvalidYAML(t *testing.T) {
	tempDir, cleanup := setupTempDir(t)
	defer cleanup()

	content := `---
description: Test command
invalid: yaml: content: here
---

Command content`

	filePath := createTestCommandFile(t, tempDir, "invalid.md", content)

	_, err := NewFileCommand("invalid", filePath)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to parse YAML frontmatter")
}

func TestNewFileCommandFileNotFound(t *testing.T) {
	_, err := NewFileCommand("nonexistent", "/nonexistent/file.md")

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to read command file")
}

// Test FileCommand execution
func TestFileCommandExecute(t *testing.T) {
	tempDir, cleanup := setupTempDir(t)
	defer cleanup()

	content := "Execute command with $ARGUMENTS here"
	filePath := createTestCommandFile(t, tempDir, "exec.md", content)

	cmd, err := NewFileCommand("exec", filePath)
	require.NoError(t, err)

	result, err := cmd.Execute(context.Background(), "test args")

	assert.NoError(t, err)
	assert.Equal(t, "Execute command with test args here", result)
}

func TestFileCommandExecuteNoArgs(t *testing.T) {
	tempDir, cleanup := setupTempDir(t)
	defer cleanup()

	content := "Execute command with $ARGUMENTS here"
	filePath := createTestCommandFile(t, tempDir, "exec.md", content)

	cmd, err := NewFileCommand("exec", filePath)
	require.NoError(t, err)

	result, err := cmd.Execute(context.Background(), "")

	assert.NoError(t, err)
	assert.Equal(t, "Execute command with  here", result)
}

// Test LoadCommandsFromDirectory
func TestLoadCommandsFromDirectory(t *testing.T) {
	tempDir, cleanup := setupTempDir(t)
	defer cleanup()

	// Create test command files
	createTestCommandFile(t, tempDir, "cmd1.md", "Command 1 content")
	createTestCommandFile(t, tempDir, "cmd2.md", "Command 2 content")

	// Create subdirectory with command
	subDir := filepath.Join(tempDir, "sub")
	err := os.MkdirAll(subDir, 0755)
	require.NoError(t, err)
	createTestCommandFile(t, subDir, "subcmd.md", "Sub command content")

	// Create non-command file (should be ignored)
	createTestCommandFile(t, tempDir, "README.txt", "Not a command")

	commands, err := LoadCommandsFromDirectory(tempDir)

	assert.NoError(t, err)
	assert.Len(t, commands, 3)
	assert.Contains(t, commands, "cmd1")
	assert.Contains(t, commands, "cmd2")
	assert.Contains(t, commands, "sub:subcmd") // Namespaced
	assert.NotContains(t, commands, "README")
}

func TestLoadCommandsFromDirectoryNonexistent(t *testing.T) {
	commands, err := LoadCommandsFromDirectory("/nonexistent/directory")

	assert.NoError(t, err)
	assert.Empty(t, commands)
}

func TestLoadCommandsFromDirectoryInvalidCommand(t *testing.T) {
	tempDir, cleanup := setupTempDir(t)
	defer cleanup()

	// Create invalid command file
	invalidContent := `---
invalid: yaml: content: here
---`
	createTestCommandFile(t, tempDir, "invalid.md", invalidContent)

	_, err := LoadCommandsFromDirectory(tempDir)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to load command invalid")
}

// Test command parsing functionality
func TestParseFileWithEmptyContent(t *testing.T) {
	cmd := &FileCommand{
		name:     "empty",
		filePath: "empty.md",
	}

	err := cmd.parseFile("")

	assert.NoError(t, err)
	assert.Empty(t, cmd.content)
	assert.Contains(t, cmd.description, "Custom command from empty.md")
}

func TestParseFileWithOnlyFrontmatter(t *testing.T) {
	cmd := &FileCommand{
		name:     "frontmatter",
		filePath: "frontmatter.md",
	}

	content := `---
description: Only frontmatter
---`

	err := cmd.parseFile(content)

	assert.NoError(t, err)
	assert.Empty(t, cmd.content)
	assert.Equal(t, "Only frontmatter", cmd.description)
}

func TestParseFileWithWhitespace(t *testing.T) {
	cmd := &FileCommand{
		name:     "whitespace",
		filePath: "whitespace.md",
	}

	content := `---
description: Test whitespace
---

   Content with leading whitespace
`

	err := cmd.parseFile(content)

	assert.NoError(t, err)
	assert.Equal(t, "Content with leading whitespace", cmd.content)
}