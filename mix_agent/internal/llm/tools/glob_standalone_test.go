//go:build standalone
// +build standalone

package tools

import (
	"context"
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"mix/internal/llm/interfaces"
)

// Standalone test for glob functionality that can be run independently
func TestGlobToolStandalone(t *testing.T) {
	// Test NewGlobTool
	tool := NewGlobTool()
	assert.NotNil(t, tool)

	// Test interface compliance
	var _ interfaces.BaseTool = tool

	// Test Info method
	info := tool.Info()
	assert.Equal(t, "glob", info.Name)
	assert.NotEmpty(t, info.Description)
	assert.Contains(t, info.Required, "pattern")

	// Test JSON serialization of structs
	params := GlobParams{Pattern: "*.go", Path: "/test"}
	jsonData, err := json.Marshal(params)
	require.NoError(t, err)

	var unmarshaledParams GlobParams
	err = json.Unmarshal(jsonData, &unmarshaledParams)
	require.NoError(t, err)
	assert.Equal(t, params, unmarshaledParams)

	// Test metadata serialization
	metadata := GlobResponseMetadata{NumberOfFiles: 5, Truncated: true}
	metaData, err := json.Marshal(metadata)
	require.NoError(t, err)

	var unmarshaledMeta GlobResponseMetadata
	err = json.Unmarshal(metaData, &unmarshaledMeta)
	require.NoError(t, err)
	assert.Equal(t, metadata, unmarshaledMeta)

	// Test Run method with various inputs
	t.Run("invalid JSON", func(t *testing.T) {
		call := ToolCall{ID: "1", Name: "glob", Input: "invalid"}
		response, err := tool.Run(context.Background(), call)
		require.NoError(t, err)
		assert.True(t, response.IsError)
		assert.Contains(t, response.Content, "error parsing parameters")
	})

	t.Run("missing pattern", func(t *testing.T) {
		call := ToolCall{ID: "2", Name: "glob", Input: `{"path": "/test"}`}
		response, err := tool.Run(context.Background(), call)
		require.NoError(t, err)
		assert.True(t, response.IsError)
		assert.Contains(t, response.Content, "pattern is required")
	})

	t.Run("no session storage context", func(t *testing.T) {
		call := ToolCall{ID: "3", Name: "glob", Input: `{"pattern": "*.go"}`}
		response, err := tool.Run(context.Background(), call)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "failed to get session storage directory")
		assert.Empty(t, response.Content) // Should be empty on error
	})

	t.Run("with valid context", func(t *testing.T) {
		// Create temp directory
		tempDir, err := os.MkdirTemp("", "glob_test_*")
		require.NoError(t, err)
		defer os.RemoveAll(tempDir)

		// Create test files
		err = os.WriteFile(filepath.Join(tempDir, "test.go"), []byte("test"), 0o600)
		require.NoError(t, err)
		err = os.WriteFile(filepath.Join(tempDir, "test.txt"), []byte("test"), 0o600)
		require.NoError(t, err)

		ctx := context.WithValue(context.Background(), SessionStorageContextKey, tempDir)
		call := ToolCall{ID: "4", Name: "glob", Input: `{"pattern": "*.go"}`}

		response, err := tool.Run(ctx, call)
		require.NoError(t, err)
		assert.False(t, response.IsError)
		assert.Contains(t, response.Content, "test.go")
		assert.NotContains(t, response.Content, "test.txt")
	})
}

// Test globFiles function
func TestGlobFilesStandalone(t *testing.T) {
	// Create temp directory with test files
	tempDir, err := os.MkdirTemp("", "glob_files_test_*")
	require.NoError(t, err)
	defer os.RemoveAll(tempDir)

	// Create test files
	testFiles := []string{"test1.go", "test2.go", "test.txt"}
	for _, file := range testFiles {
		err = os.WriteFile(filepath.Join(tempDir, file), []byte("content"), 0o600)
		require.NoError(t, err)
	}

	files, truncated, err := globFiles("*.go", tempDir, 10)
	require.NoError(t, err)
	assert.False(t, truncated)
	assert.Len(t, files, 2)

	// Check file paths
	for _, file := range files {
		assert.True(t, strings.HasSuffix(file, ".go"))
		assert.True(t, filepath.IsAbs(file))
	}
}

// Test runRipgrep function
func TestRunRipgrepStandalone(t *testing.T) {
	t.Run("successful command", func(t *testing.T) {
		// Use printf to simulate ripgrep output (echo might not handle null bytes well)
		files := []string{"test1.go", "test2.go"}
		output := strings.Join(files, "\\0") + "\\0"
		cmd := exec.CommandContext(context.Background(), "printf", output)

		matches, err := runRipgrep(cmd, "/tmp", 10)
		require.NoError(t, err)
		assert.Len(t, matches, 2)
	})

	t.Run("no matches (exit code 1)", func(t *testing.T) {
		cmd := exec.CommandContext(context.Background(), "sh", "-c", "exit 1")
		matches, err := runRipgrep(cmd, "/tmp", 10)
		require.NoError(t, err)
		assert.Nil(t, matches)
	})

	t.Run("command failure", func(t *testing.T) {
		cmd := exec.CommandContext(context.Background(), "sh", "-c", "exit 2")
		matches, err := runRipgrep(cmd, "/tmp", 10)
		require.Error(t, err)
		assert.Nil(t, matches)
	})
}

func TestGlobConstantsStandalone(t *testing.T) {
	assert.Equal(t, "glob", GlobToolName)
}
