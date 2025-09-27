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

// Test struct JSON serialization/deserialization
func TestGlobParams_JSONSerialization(t *testing.T) {
	tests := []struct {
		name   string
		params GlobParams
		json   string
	}{
		{
			name: "complete params",
			params: GlobParams{
				Pattern: "*.go",
				Path:    "/path/to/search",
			},
			json: `{"pattern":"*.go","path":"/path/to/search"}`,
		},
		{
			name: "pattern only",
			params: GlobParams{
				Pattern: "**/*.js",
				Path:    "",
			},
			json: `{"pattern":"**/*.js","path":""}`,
		},
		{
			name: "empty params",
			params: GlobParams{
				Pattern: "",
				Path:    "",
			},
			json: `{"pattern":"","path":""}`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Test marshaling
			jsonBytes, err := json.Marshal(tt.params)
			require.NoError(t, err)
			assert.JSONEq(t, tt.json, string(jsonBytes))

			// Test unmarshaling
			var params GlobParams
			err = json.Unmarshal([]byte(tt.json), &params)
			require.NoError(t, err)
			assert.Equal(t, tt.params, params)
		})
	}
}

func TestGlobResponseMetadata_JSONSerialization(t *testing.T) {
	tests := []struct {
		name     string
		metadata GlobResponseMetadata
		json     string
	}{
		{
			name: "with files and truncated",
			metadata: GlobResponseMetadata{
				NumberOfFiles: 42,
				Truncated:     true,
			},
			json: `{"number_of_files":42,"truncated":true}`,
		},
		{
			name: "no files not truncated",
			metadata: GlobResponseMetadata{
				NumberOfFiles: 0,
				Truncated:     false,
			},
			json: `{"number_of_files":0,"truncated":false}`,
		},
		{
			name: "many files not truncated",
			metadata: GlobResponseMetadata{
				NumberOfFiles: 150,
				Truncated:     false,
			},
			json: `{"number_of_files":150,"truncated":false}`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Test marshaling
			jsonBytes, err := json.Marshal(tt.metadata)
			require.NoError(t, err)
			assert.JSONEq(t, tt.json, string(jsonBytes))

			// Test unmarshaling
			var metadata GlobResponseMetadata
			err = json.Unmarshal([]byte(tt.json), &metadata)
			require.NoError(t, err)
			assert.Equal(t, tt.metadata, metadata)
		})
	}
}

// Test NewGlobTool function and interface compliance
func TestNewGlobTool(t *testing.T) {
	tool := NewGlobTool()
	assert.NotNil(t, tool)

	// Test interface compliance
	var _ interfaces.BaseTool = tool

	// Test that it's the correct implementation
	globTool, ok := tool.(*globTool)
	assert.True(t, ok)
	assert.NotNil(t, globTool)
}

// Test Info method
func TestGlobTool_Info(t *testing.T) {
	tool := NewGlobTool()
	info := tool.Info()

	// Test basic structure
	assert.Equal(t, GlobToolName, info.Name)
	assert.Equal(t, "glob", info.Name)
	assert.NotEmpty(t, info.Description)
	assert.NotNil(t, info.Parameters)
	assert.NotNil(t, info.Required)

	// Test required parameters
	assert.Contains(t, info.Required, "pattern")
	assert.Len(t, info.Required, 1) // Only pattern is required

	// Test parameter definitions
	assert.Contains(t, info.Parameters, "pattern")
	assert.Contains(t, info.Parameters, "path")

	// Test pattern parameter structure
	patternParam, ok := info.Parameters["pattern"].(map[string]any)
	require.True(t, ok)
	assert.Equal(t, "string", patternParam["type"])
	assert.Contains(t, patternParam["description"], "glob pattern")

	// Test path parameter structure
	pathParam, ok := info.Parameters["path"].(map[string]any)
	require.True(t, ok)
	assert.Equal(t, "string", pathParam["type"])
	assert.Contains(t, pathParam["description"], "directory")
}

// Test Run method with various scenarios
func TestGlobTool_Run(t *testing.T) {
	tool := NewGlobTool()

	t.Run("invalid JSON input", func(t *testing.T) {
		call := ToolCall{
			ID:    "test-1",
			Name:  "glob",
			Input: `{"invalid": json}`,
		}

		response, err := tool.Run(context.Background(), call)
		require.NoError(t, err)
		assert.True(t, response.IsError)
		assert.Contains(t, response.Content, "error parsing parameters")
	})

	t.Run("missing pattern", func(t *testing.T) {
		call := ToolCall{
			ID:    "test-2",
			Name:  "glob",
			Input: `{"path": "/some/path"}`,
		}

		response, err := tool.Run(context.Background(), call)
		require.NoError(t, err)
		assert.True(t, response.IsError)
		assert.Contains(t, response.Content, "pattern is required")
	})

	t.Run("empty pattern", func(t *testing.T) {
		call := ToolCall{
			ID:    "test-3",
			Name:  "glob",
			Input: `{"pattern": "", "path": "/some/path"}`,
		}

		response, err := tool.Run(context.Background(), call)
		require.NoError(t, err)
		assert.True(t, response.IsError)
		assert.Contains(t, response.Content, "pattern is required")
	})

	t.Run("missing session storage in context", func(t *testing.T) {
		call := ToolCall{
			ID:    "test-4",
			Name:  "glob",
			Input: `{"pattern": "*.go"}`,
		}

		response, err := tool.Run(context.Background(), call)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "failed to get session storage directory")
		assert.Empty(t, response.Content) // response should be empty on error
	})

	t.Run("successful run with session storage context", func(t *testing.T) {
		// Create a temporary directory for testing
		tempDir, err := os.MkdirTemp("", "glob_test_*")
		require.NoError(t, err)
		defer os.RemoveAll(tempDir)

		// Create some test files
		testFiles := []string{"test1.go", "test2.go", "test.txt", "subdir/test3.go"}
		for _, file := range testFiles {
			fullPath := filepath.Join(tempDir, file)
			err := os.MkdirAll(filepath.Dir(fullPath), 0755)
			require.NoError(t, err)
			err = os.WriteFile(fullPath, []byte("test content"), 0644)
			require.NoError(t, err)
		}

		// Set up context with session storage
		ctx := context.WithValue(context.Background(), SessionStorageContextKey, tempDir)

		call := ToolCall{
			ID:    "test-5",
			Name:  "glob",
			Input: `{"pattern": "*.go"}`,
		}

		response, err := tool.Run(ctx, call)
		require.NoError(t, err)
		assert.False(t, response.IsError)
		assert.Contains(t, response.Content, "test1.go")
		assert.Contains(t, response.Content, "test2.go")
		assert.NotContains(t, response.Content, "test.txt")
	})

	t.Run("successful run with explicit path", func(t *testing.T) {
		// Create a temporary directory for testing
		tempDir, err := os.MkdirTemp("", "glob_test_*")
		require.NoError(t, err)
		defer os.RemoveAll(tempDir)

		// Create some test files
		testFile := filepath.Join(tempDir, "explicit.go")
		err = os.WriteFile(testFile, []byte("test content"), 0644)
		require.NoError(t, err)

		// Set up context (storage directory shouldn't be used)
		ctx := context.WithValue(context.Background(), SessionStorageContextKey, "/should/not/be/used")

		call := ToolCall{
			ID:    "test-6",
			Name:  "glob",
			Input: `{"pattern": "*.go", "path": "` + tempDir + `"}`,
		}

		response, err := tool.Run(ctx, call)
		require.NoError(t, err)
		assert.False(t, response.IsError)
		assert.Contains(t, response.Content, "explicit.go")
	})

	t.Run("no files found", func(t *testing.T) {
		// Create a temporary directory with no matching files
		tempDir, err := os.MkdirTemp("", "glob_test_*")
		require.NoError(t, err)
		defer os.RemoveAll(tempDir)

		// Create a non-matching file
		testFile := filepath.Join(tempDir, "test.txt")
		err = os.WriteFile(testFile, []byte("test content"), 0644)
		require.NoError(t, err)

		ctx := context.WithValue(context.Background(), SessionStorageContextKey, tempDir)

		call := ToolCall{
			ID:    "test-7",
			Name:  "glob",
			Input: `{"pattern": "*.go"}`,
		}

		response, err := tool.Run(ctx, call)
		require.NoError(t, err)
		assert.False(t, response.IsError)
		assert.Equal(t, "No files found", response.Content)
	})

	t.Run("response metadata", func(t *testing.T) {
		// Create a temporary directory for testing
		tempDir, err := os.MkdirTemp("", "glob_test_*")
		require.NoError(t, err)
		defer os.RemoveAll(tempDir)

		// Create test files
		for i := 0; i < 3; i++ {
			testFile := filepath.Join(tempDir, "test"+string(rune('1'+i))+".go")
			err = os.WriteFile(testFile, []byte("test content"), 0644)
			require.NoError(t, err)
		}

		ctx := context.WithValue(context.Background(), SessionStorageContextKey, tempDir)

		call := ToolCall{
			ID:    "test-8",
			Name:  "glob",
			Input: `{"pattern": "*.go"}`,
		}

		response, err := tool.Run(ctx, call)
		require.NoError(t, err)
		assert.False(t, response.IsError)
		assert.NotEmpty(t, response.Metadata)

		var metadata GlobResponseMetadata
		err = json.Unmarshal([]byte(response.Metadata), &metadata)
		require.NoError(t, err)
		assert.Equal(t, 3, metadata.NumberOfFiles)
		assert.False(t, metadata.Truncated)
	})
}

// Test globFiles function
func TestGlobFiles(t *testing.T) {
	t.Run("with valid directory and pattern", func(t *testing.T) {
		// Create a temporary directory for testing
		tempDir, err := os.MkdirTemp("", "glob_files_test_*")
		require.NoError(t, err)
		defer os.RemoveAll(tempDir)

		// Create test files
		testFiles := []string{"test1.go", "test2.go", "test.txt"}
		for _, file := range testFiles {
			fullPath := filepath.Join(tempDir, file)
			err = os.WriteFile(fullPath, []byte("test content"), 0644)
			require.NoError(t, err)
		}

		files, truncated, err := globFiles("*.go", tempDir, 10)
		require.NoError(t, err)
		assert.False(t, truncated)
		assert.Len(t, files, 2)

		// Check that results contain the .go files
		fileNames := make([]string, len(files))
		for i, f := range files {
			fileNames[i] = filepath.Base(f)
		}
		assert.Contains(t, fileNames, "test1.go")
		assert.Contains(t, fileNames, "test2.go")
	})

	t.Run("with limit causing truncation", func(t *testing.T) {
		// Create a temporary directory for testing
		tempDir, err := os.MkdirTemp("", "glob_files_test_*")
		require.NoError(t, err)
		defer os.RemoveAll(tempDir)

		// Create more test files than the limit
		for i := 0; i < 5; i++ {
			testFile := filepath.Join(tempDir, "test"+string(rune('1'+i))+".go")
			err = os.WriteFile(testFile, []byte("test content"), 0644)
			require.NoError(t, err)
		}

		files, truncated, err := globFiles("*.go", tempDir, 3)
		require.NoError(t, err)
		assert.True(t, truncated)
		assert.Len(t, files, 3)
	})

	t.Run("with nonexistent directory", func(t *testing.T) {
		files, truncated, err := globFiles("*.go", "/nonexistent/directory", 10)
		require.Error(t, err)
		assert.False(t, truncated)
		assert.Nil(t, files)
	})

	t.Run("with zero limit", func(t *testing.T) {
		// Create a temporary directory for testing
		tempDir, err := os.MkdirTemp("", "glob_files_test_*")
		require.NoError(t, err)
		defer os.RemoveAll(tempDir)

		// Create test files
		for i := 0; i < 3; i++ {
			testFile := filepath.Join(tempDir, "test"+string(rune('1'+i))+".go")
			err = os.WriteFile(testFile, []byte("test content"), 0644)
			require.NoError(t, err)
		}

		files, truncated, err := globFiles("*.go", tempDir, 0)
		require.NoError(t, err)
		assert.False(t, truncated) // No truncation when limit is 0
		assert.Len(t, files, 3)     // Should return all files
	})
}

// Test runRipgrep function
func TestRunRipgrep(t *testing.T) {
	t.Run("successful execution with files", func(t *testing.T) {
		// Create a temporary directory for testing
		tempDir, err := os.MkdirTemp("", "ripgrep_test_*")
		require.NoError(t, err)
		defer os.RemoveAll(tempDir)

		// Create test files
		testFiles := []string{"test1.go", "test2.go"}
		for _, file := range testFiles {
			fullPath := filepath.Join(tempDir, file)
			err = os.WriteFile(fullPath, []byte("test content"), 0644)
			require.NoError(t, err)
		}

		// Create a mock command that simulates ripgrep output
		// We'll use echo to simulate the null-separated output
		output := strings.Join(testFiles, "\x00") + "\x00"
		cmd := exec.Command("echo", "-n", output)
		cmd.Dir = tempDir

		matches, err := runRipgrep(cmd, tempDir, 10)
		require.NoError(t, err)
		assert.Len(t, matches, 2)

		// Check that paths are properly constructed
		for _, match := range matches {
			assert.True(t, filepath.IsAbs(match))
			assert.True(t, strings.HasSuffix(match, ".go"))
		}
	})

	t.Run("no files found (exit code 1)", func(t *testing.T) {
		// Create a command that exits with code 1 (no matches)
		cmd := exec.Command("sh", "-c", "exit 1")

		matches, err := runRipgrep(cmd, "/tmp", 10)
		require.NoError(t, err)
		assert.Nil(t, matches)
	})

	t.Run("command failure (non-1 exit code)", func(t *testing.T) {
		// Create a command that fails with a different exit code
		cmd := exec.Command("sh", "-c", "echo 'error message' >&2; exit 2")

		matches, err := runRipgrep(cmd, "/tmp", 10)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "ripgrep")
		assert.Nil(t, matches)
	})

	t.Run("with limit", func(t *testing.T) {
		// Create output with more files than the limit
		files := []string{"file1.go", "file2.go", "file3.go", "file4.go", "file5.go"}
		output := strings.Join(files, "\x00") + "\x00"
		cmd := exec.Command("echo", "-n", output)

		matches, err := runRipgrep(cmd, "/tmp", 3)
		require.NoError(t, err)
		assert.Len(t, matches, 3)
	})

	t.Run("with relative paths", func(t *testing.T) {
		tempDir, err := os.MkdirTemp("", "ripgrep_test_*")
		require.NoError(t, err)
		defer os.RemoveAll(tempDir)

		// Simulate ripgrep output with relative paths
		files := []string{"test1.go", "subdir/test2.go"}
		output := strings.Join(files, "\x00") + "\x00"
		cmd := exec.Command("echo", "-n", output)

		matches, err := runRipgrep(cmd, tempDir, 10)
		require.NoError(t, err)
		assert.Len(t, matches, 2)

		// All returned paths should be absolute
		for _, match := range matches {
			assert.True(t, filepath.IsAbs(match))
			assert.True(t, strings.HasPrefix(match, tempDir))
		}
	})

	t.Run("sorting by path length", func(t *testing.T) {
		// Create output with files of different path lengths
		files := []string{"very/long/path/file.go", "short.go", "medium/path.go"}
		output := strings.Join(files, "\x00") + "\x00"
		cmd := exec.Command("echo", "-n", output)

		matches, err := runRipgrep(cmd, "/tmp", 10)
		require.NoError(t, err)
		assert.Len(t, matches, 3)

		// Results should be sorted by path length (shortest first)
		prevLen := 0
		for _, match := range matches {
			assert.GreaterOrEqual(t, len(match), prevLen)
			prevLen = len(match)
		}
	})

	t.Run("empty output", func(t *testing.T) {
		cmd := exec.Command("echo", "-n", "")

		matches, err := runRipgrep(cmd, "/tmp", 10)
		require.NoError(t, err)
		assert.Empty(t, matches)
	})
}

// Test constants
func TestGlobConstants(t *testing.T) {
	assert.Equal(t, "glob", GlobToolName)
}

// Integration test combining multiple components
func TestGlobTool_Integration(t *testing.T) {
	// Create a complex directory structure for testing
	tempDir, err := os.MkdirTemp("", "glob_integration_test_*")
	require.NoError(t, err)
	defer os.RemoveAll(tempDir)

	// Create a complex file structure
	files := []string{
		"main.go",
		"utils.go",
		"src/app.go",
		"src/handler.go",
		"tests/main_test.go",
		"tests/utils_test.go",
		"docs/README.md",
		"scripts/build.sh",
		".hidden/secret.go", // Should be skipped
	}

	for _, file := range files {
		fullPath := filepath.Join(tempDir, file)
		err := os.MkdirAll(filepath.Dir(fullPath), 0755)
		require.NoError(t, err)
		err = os.WriteFile(fullPath, []byte("content"), 0644)
		require.NoError(t, err)
	}

	tool := NewGlobTool()
	ctx := context.WithValue(context.Background(), SessionStorageContextKey, tempDir)

	t.Run("find all go files", func(t *testing.T) {
		call := ToolCall{
			ID:    "integration-1",
			Name:  "glob",
			Input: `{"pattern": "**/*.go"}`,
		}

		response, err := tool.Run(ctx, call)
		require.NoError(t, err)
		assert.False(t, response.IsError)

		// Should find all .go files except hidden ones
		lines := strings.Split(strings.TrimSpace(response.Content), "\n")
		goFiles := 0
		for _, line := range lines {
			if strings.HasSuffix(line, ".go") {
				goFiles++
				// Should not include hidden files
				assert.NotContains(t, line, ".hidden")
			}
		}
		assert.Equal(t, 6, goFiles) // main.go, utils.go, app.go, handler.go, main_test.go, utils_test.go

		// Check metadata
		var metadata GlobResponseMetadata
		err = json.Unmarshal([]byte(response.Metadata), &metadata)
		require.NoError(t, err)
		assert.Equal(t, 6, metadata.NumberOfFiles)
		assert.False(t, metadata.Truncated)
	})

	t.Run("find test files only", func(t *testing.T) {
		call := ToolCall{
			ID:    "integration-2",
			Name:  "glob",
			Input: `{"pattern": "**/*_test.go"}`,
		}

		response, err := tool.Run(ctx, call)
		require.NoError(t, err)
		assert.False(t, response.IsError)

		lines := strings.Split(strings.TrimSpace(response.Content), "\n")
		assert.Len(t, lines, 2)
		for _, line := range lines {
			assert.Contains(t, line, "_test.go")
		}
	})

	t.Run("find files with truncation", func(t *testing.T) {
		// Create many files to trigger truncation
		for i := 0; i < 150; i++ {
			fileName := filepath.Join(tempDir, "generated", "file"+string(rune('0'+i%10))+".txt")
			err := os.MkdirAll(filepath.Dir(fileName), 0755)
			require.NoError(t, err)
			err = os.WriteFile(fileName, []byte("content"), 0644)
			require.NoError(t, err)
		}

		call := ToolCall{
			ID:    "integration-3",
			Name:  "glob",
			Input: `{"pattern": "**/*.txt"}`,
		}

		response, err := tool.Run(ctx, call)
		require.NoError(t, err)
		assert.False(t, response.IsError)

		// Should be truncated due to the 100 file limit in globFiles
		assert.Contains(t, response.Content, "Results are truncated")

		// Check metadata
		var metadata GlobResponseMetadata
		err = json.Unmarshal([]byte(response.Metadata), &metadata)
		require.NoError(t, err)
		assert.Equal(t, 100, metadata.NumberOfFiles) // Limited by globFiles limit
		assert.True(t, metadata.Truncated)
	})
}

// Benchmark tests
func BenchmarkGlobTool_Run(b *testing.B) {
	// Create a temporary directory with many files
	tempDir, err := os.MkdirTemp("", "glob_benchmark_*")
	require.NoError(b, err)
	defer os.RemoveAll(tempDir)

	// Create test files
	for i := 0; i < 1000; i++ {
		file := filepath.Join(tempDir, "file"+string(rune('0'+i%10))+".go")
		err = os.WriteFile(file, []byte("content"), 0644)
		require.NoError(b, err)
	}

	tool := NewGlobTool()
	ctx := context.WithValue(context.Background(), SessionStorageContextKey, tempDir)
	call := ToolCall{
		ID:    "benchmark",
		Name:  "glob",
		Input: `{"pattern": "*.go"}`,
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := tool.Run(ctx, call)
		require.NoError(b, err)
	}
}