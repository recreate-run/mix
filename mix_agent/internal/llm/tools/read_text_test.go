package tools

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"mix/internal/permission"
	"mix/internal/pubsub"
)

// Mock permission service for testing read_text
type mockReadTextPermissionService struct {
	requestResponse bool
	requestCalled   bool
	lastRequest     permission.CreatePermissionRequest
}

func (m *mockReadTextPermissionService) Request(opts permission.CreatePermissionRequest) bool {
	m.requestCalled = true
	m.lastRequest = opts
	return m.requestResponse
}

func (m *mockReadTextPermissionService) GrantPersistant(permission.PermissionRequest) {}
func (m *mockReadTextPermissionService) Grant(permission.PermissionRequest)           {}
func (m *mockReadTextPermissionService) Deny(permission.PermissionRequest)            {}
func (m *mockReadTextPermissionService) Subscribe(context.Context) <-chan pubsub.Event[permission.PermissionRequest] {
	return make(<-chan pubsub.Event[permission.PermissionRequest])
}

// TestReadTextParams tests JSON serialization/deserialization of ReadTextParams
func TestReadTextParams(t *testing.T) {
	tests := []struct {
		name     string
		params   ReadTextParams
		expected string
	}{
		{
			name: "all fields set",
			params: ReadTextParams{
				FilePath: "/test/file.txt",
				Offset:   10,
				Limit:    50,
			},
			expected: `{"file_path":"/test/file.txt","offset":10,"limit":50}`,
		},
		{
			name: "minimal fields",
			params: ReadTextParams{
				FilePath: "/test/file.txt",
			},
			expected: `{"file_path":"/test/file.txt","offset":0,"limit":0}`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Test marshaling
			data, err := json.Marshal(tt.params)
			require.NoError(t, err)
			assert.JSONEq(t, tt.expected, string(data))

			// Test unmarshaling
			var params ReadTextParams
			err = json.Unmarshal(data, &params)
			require.NoError(t, err)
			assert.Equal(t, tt.params, params)
		})
	}
}

// TestReadTextResponseMetadata tests JSON serialization/deserialization of ReadTextResponseMetadata
func TestReadTextResponseMetadata(t *testing.T) {
	tests := []struct {
		name     string
		metadata ReadTextResponseMetadata
		expected string
	}{
		{
			name: "with content",
			metadata: ReadTextResponseMetadata{
				FilePath: "/test/file.txt",
				Content:  "line 1\nline 2",
			},
			expected: `{"file_path":"/test/file.txt","content":"line 1\nline 2"}`,
		},
		{
			name: "empty content",
			metadata: ReadTextResponseMetadata{
				FilePath: "/test/empty.txt",
				Content:  "",
			},
			expected: `{"file_path":"/test/empty.txt","content":""}`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Test marshaling
			data, err := json.Marshal(tt.metadata)
			require.NoError(t, err)
			assert.JSONEq(t, tt.expected, string(data))

			// Test unmarshaling
			var metadata ReadTextResponseMetadata
			err = json.Unmarshal(data, &metadata)
			require.NoError(t, err)
			assert.Equal(t, tt.metadata, metadata)
		})
	}
}

// TestNewReadTextTool tests the constructor
func TestNewReadTextTool(t *testing.T) {
	mockPerms := &mockReadTextPermissionService{}
	tool := NewReadTextTool(mockPerms)

	require.NotNil(t, tool)

	// Verify it implements BaseTool interface
	var _ BaseTool = tool

	readTextTool, ok := tool.(*readTextTool)
	require.True(t, ok)
	assert.Equal(t, mockPerms, readTextTool.permissions)
}

// TestReadTextTool_Info tests the Info method
func TestReadTextTool_Info(t *testing.T) {
	mockPerms := &mockReadTextPermissionService{}
	tool := NewReadTextTool(mockPerms).(*readTextTool)

	info := tool.Info()

	assert.Equal(t, ReadTextToolName, info.Name)
	assert.NotEmpty(t, info.Description)

	// Verify parameters structure
	assert.Contains(t, info.Parameters, "file_path")
	assert.Contains(t, info.Parameters, "offset")
	assert.Contains(t, info.Parameters, "limit")

	// Verify required fields
	assert.Contains(t, info.Required, "file_path")
	assert.Len(t, info.Required, 1)

	// Verify parameter types
	filePathParam := info.Parameters["file_path"].(map[string]any)
	assert.Equal(t, "string", filePathParam["type"])
	assert.NotEmpty(t, filePathParam["description"])

	offsetParam := info.Parameters["offset"].(map[string]any)
	assert.Equal(t, "integer", offsetParam["type"])
	assert.NotEmpty(t, offsetParam["description"])

	limitParam := info.Parameters["limit"].(map[string]any)
	assert.Equal(t, "integer", limitParam["type"])
	assert.NotEmpty(t, limitParam["description"])
}

// TestReadTextTool_Run tests the main Run method
func TestReadTextTool_Run(t *testing.T) {
	// Create a temporary test file
	tempDir := t.TempDir()
	testFile := filepath.Join(tempDir, "test.txt")
	testContent := "line 1\nline 2\nline 3\nline 4\nline 5"
	err := os.WriteFile(testFile, []byte(testContent), 0644)
	require.NoError(t, err)

	mockPerms := &mockReadTextPermissionService{requestResponse: true}
	tool := NewReadTextTool(mockPerms).(*readTextTool)

	// Create context with session and message IDs
	ctx := context.Background()
	ctx = context.WithValue(ctx, SessionIDContextKey, "test-session")
	ctx = context.WithValue(ctx, MessageIDContextKey, "test-message")

	t.Run("successful read", func(t *testing.T) {
		mockPerms.requestCalled = false

		call := ToolCall{
			ID:   "test-call",
			Name: ReadTextToolName,
			Input: `{"file_path":"` + testFile + `"}`,
		}

		response, err := tool.Run(ctx, call)
		require.NoError(t, err)

		assert.False(t, response.IsError)
		assert.Contains(t, response.Content, "line 1")
		assert.Contains(t, response.Content, "line 2")
		assert.True(t, mockPerms.requestCalled)
		assert.Equal(t, testFile, mockPerms.lastRequest.Path)
		assert.Equal(t, ReadTextToolName, mockPerms.lastRequest.ToolName)
	})

	t.Run("read with offset and limit", func(t *testing.T) {
		mockPerms.requestCalled = false

		call := ToolCall{
			ID:   "test-call",
			Name: ReadTextToolName,
			Input: `{"file_path":"` + testFile + `","offset":1,"limit":2}`,
		}

		response, err := tool.Run(ctx, call)
		require.NoError(t, err)

		assert.False(t, response.IsError)
		assert.Contains(t, response.Content, "line 2")
		assert.Contains(t, response.Content, "line 3")
		assert.NotContains(t, response.Content, "line 1")
		assert.True(t, mockPerms.requestCalled)
	})

	t.Run("invalid JSON input", func(t *testing.T) {
		call := ToolCall{
			ID:   "test-call",
			Name: ReadTextToolName,
			Input: `{"invalid": json}`,
		}

		response, err := tool.Run(ctx, call)
		require.NoError(t, err)

		assert.True(t, response.IsError)
		assert.Contains(t, response.Content, "error parsing parameters")
	})

	t.Run("missing file path", func(t *testing.T) {
		call := ToolCall{
			ID:   "test-call",
			Name: ReadTextToolName,
			Input: `{"offset":0,"limit":10}`,
		}

		response, err := tool.Run(ctx, call)
		require.NoError(t, err)

		assert.True(t, response.IsError)
		assert.Contains(t, response.Content, "file_path is required")
	})

	t.Run("relative path error", func(t *testing.T) {
		call := ToolCall{
			ID:   "test-call",
			Name: ReadTextToolName,
			Input: `{"file_path":"relative/path.txt"}`,
		}

		response, err := tool.Run(ctx, call)
		require.NoError(t, err)

		assert.True(t, response.IsError)
		assert.Contains(t, response.Content, "must be an absolute path")
	})

	t.Run("missing context values", func(t *testing.T) {
		emptyCtx := context.Background()

		call := ToolCall{
			ID:   "test-call",
			Name: ReadTextToolName,
			Input: `{"file_path":"` + testFile + `"}`,
		}

		_, err := tool.Run(emptyCtx, call)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "session ID and message ID are required")
	})

	t.Run("permission denied", func(t *testing.T) {
		mockPerms.requestResponse = false

		call := ToolCall{
			ID:   "test-call",
			Name: ReadTextToolName,
			Input: `{"file_path":"` + testFile + `"}`,
		}

		_, err := tool.Run(ctx, call)
		require.Error(t, err)
		assert.Equal(t, permission.ErrorPermissionDenied, err)
	})

	t.Run("file not found", func(t *testing.T) {
		mockPerms.requestResponse = true
		nonExistentFile := filepath.Join(tempDir, "nonexistent.txt")

		call := ToolCall{
			ID:   "test-call",
			Name: ReadTextToolName,
			Input: `{"file_path":"` + nonExistentFile + `"}`,
		}

		response, err := tool.Run(ctx, call)
		require.NoError(t, err)

		assert.True(t, response.IsError)
		assert.Contains(t, response.Content, "File not found")
	})

	t.Run("file not found with suggestions", func(t *testing.T) {
		mockPerms.requestResponse = true

		// Create some similar files
		similarFile1 := filepath.Join(tempDir, "test_similar.txt")
		similarFile2 := filepath.Join(tempDir, "test_another.txt")
		err := os.WriteFile(similarFile1, []byte("content"), 0644)
		require.NoError(t, err)
		err = os.WriteFile(similarFile2, []byte("content"), 0644)
		require.NoError(t, err)

		nonExistentFile := filepath.Join(tempDir, "test_missing.txt")

		call := ToolCall{
			ID:   "test-call",
			Name: ReadTextToolName,
			Input: `{"file_path":"` + nonExistentFile + `"}`,
		}

		response, err := tool.Run(ctx, call)
		require.NoError(t, err)

		assert.True(t, response.IsError)
		assert.Contains(t, response.Content, "File not found")
		assert.Contains(t, response.Content, "Did you mean one of these?")
	})

	t.Run("directory instead of file", func(t *testing.T) {
		mockPerms.requestResponse = true

		call := ToolCall{
			ID:   "test-call",
			Name: ReadTextToolName,
			Input: `{"file_path":"` + tempDir + `"}`,
		}

		response, err := tool.Run(ctx, call)
		require.NoError(t, err)

		assert.True(t, response.IsError)
		assert.Contains(t, response.Content, "Path is a directory, not a file")
	})

	t.Run("empty file", func(t *testing.T) {
		mockPerms.requestResponse = true
		emptyFile := filepath.Join(tempDir, "empty.txt")
		err := os.WriteFile(emptyFile, []byte(""), 0644)
		require.NoError(t, err)

		call := ToolCall{
			ID:   "test-call",
			Name: ReadTextToolName,
			Input: `{"file_path":"` + emptyFile + `"}`,
		}

		response, err := tool.Run(ctx, call)
		require.NoError(t, err)

		assert.False(t, response.IsError)
		assert.Contains(t, response.Content, "File exists but has empty contents")
	})
}

// TestAddLineNumbers tests the addLineNumbers function
func TestAddLineNumbers(t *testing.T) {
	tests := []struct {
		name      string
		content   string
		startLine int
		expected  string
	}{
		{
			name:      "empty content",
			content:   "",
			startLine: 1,
			expected:  "",
		},
		{
			name:      "single line",
			content:   "hello world",
			startLine: 1,
			expected:  "     1\thello world",
		},
		{
			name:      "multiple lines",
			content:   "line 1\nline 2\nline 3",
			startLine: 1,
			expected:  "     1\tline 1\n     2\tline 2\n     3\tline 3",
		},
		{
			name:      "with carriage returns",
			content:   "line 1\r\nline 2\r",
			startLine: 1,
			expected:  "     1\tline 1\n     2\tline 2",
		},
		{
			name:      "starting from different line number",
			content:   "first\nsecond",
			startLine: 10,
			expected:  "    10\tfirst\n    11\tsecond",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := addLineNumbers(tt.content, tt.startLine)
			assert.Equal(t, tt.expected, result)
		})
	}
}

// TestReadTextFile tests the readTextFile function
func TestReadTextFile(t *testing.T) {
	tempDir := t.TempDir()

	t.Run("read full file", func(t *testing.T) {
		testFile := filepath.Join(tempDir, "test.txt")
		content := "line 1\nline 2\nline 3\nline 4\nline 5"
		err := os.WriteFile(testFile, []byte(content), 0644)
		require.NoError(t, err)

		result, lineCount, err := readTextFile(testFile, 0, 1000)
		require.NoError(t, err)

		assert.Equal(t, content, result)
		assert.Equal(t, 5, lineCount)
	})

	t.Run("read with offset", func(t *testing.T) {
		testFile := filepath.Join(tempDir, "test_offset.txt")
		content := "line 1\nline 2\nline 3\nline 4\nline 5"
		err := os.WriteFile(testFile, []byte(content), 0644)
		require.NoError(t, err)

		result, lineCount, err := readTextFile(testFile, 2, 1000)
		require.NoError(t, err)

		assert.Equal(t, "line 3\nline 4\nline 5", result)
		assert.Equal(t, 5, lineCount)
	})

	t.Run("read with limit", func(t *testing.T) {
		testFile := filepath.Join(tempDir, "test_limit.txt")
		content := "line 1\nline 2\nline 3\nline 4\nline 5"
		err := os.WriteFile(testFile, []byte(content), 0644)
		require.NoError(t, err)

		result, lineCount, err := readTextFile(testFile, 0, 3)
		require.NoError(t, err)

		assert.Equal(t, "line 1\nline 2\nline 3", result)
		assert.Equal(t, 5, lineCount) // Total line count
	})

	t.Run("read with offset and limit", func(t *testing.T) {
		testFile := filepath.Join(tempDir, "test_offset_limit.txt")
		content := "line 1\nline 2\nline 3\nline 4\nline 5"
		err := os.WriteFile(testFile, []byte(content), 0644)
		require.NoError(t, err)

		result, lineCount, err := readTextFile(testFile, 1, 2)
		require.NoError(t, err)

		assert.Equal(t, "line 2\nline 3", result)
		assert.Equal(t, 5, lineCount)
	})

	t.Run("long line truncation", func(t *testing.T) {
		testFile := filepath.Join(tempDir, "test_long.txt")
		longLine := strings.Repeat("a", MaxLineLength+100)
		err := os.WriteFile(testFile, []byte(longLine), 0644)
		require.NoError(t, err)

		result, _, err := readTextFile(testFile, 0, 1)
		require.NoError(t, err)

		assert.Len(t, result, MaxLineLength+3) // +3 for "..."
		assert.True(t, strings.HasSuffix(result, "..."))
	})

	t.Run("empty file", func(t *testing.T) {
		testFile := filepath.Join(tempDir, "empty.txt")
		err := os.WriteFile(testFile, []byte(""), 0644)
		require.NoError(t, err)

		result, lineCount, err := readTextFile(testFile, 0, 1000)
		require.NoError(t, err)

		assert.Equal(t, "", result)
		assert.Equal(t, 0, lineCount)
	})

	t.Run("file not found", func(t *testing.T) {
		nonExistentFile := filepath.Join(tempDir, "nonexistent.txt")

		_, _, err := readTextFile(nonExistentFile, 0, 1000)
		require.Error(t, err)
	})
}

// TestLineScanner tests the LineScanner type
func TestLineScanner(t *testing.T) {
	t.Run("scan lines", func(t *testing.T) {
		content := "line 1\nline 2\nline 3"
		reader := strings.NewReader(content)
		scanner := NewLineScanner(reader)

		var lines []string
		for scanner.Scan() {
			lines = append(lines, scanner.Text())
		}

		require.NoError(t, scanner.Err())
		assert.Equal(t, []string{"line 1", "line 2", "line 3"}, lines)
	})

	t.Run("empty content", func(t *testing.T) {
		reader := strings.NewReader("")
		scanner := NewLineScanner(reader)

		assert.False(t, scanner.Scan())
		require.NoError(t, scanner.Err())
	})

	t.Run("single line no newline", func(t *testing.T) {
		reader := strings.NewReader("single line")
		scanner := NewLineScanner(reader)

		assert.True(t, scanner.Scan())
		assert.Equal(t, "single line", scanner.Text())
		assert.False(t, scanner.Scan())
		require.NoError(t, scanner.Err())
	})
}

// TestIsBinaryFile tests the isBinaryFile function
func TestIsBinaryFile(t *testing.T) {
	tempDir := t.TempDir()

	binaryExtensions := []string{
		".jpg", ".png", ".pdf", ".exe", ".zip", ".mp4", ".mp3",
	}

	for _, ext := range binaryExtensions {
		t.Run("binary extension "+ext, func(t *testing.T) {
			testFile := filepath.Join(tempDir, "test"+ext)
			err := os.WriteFile(testFile, []byte("dummy content"), 0644)
			require.NoError(t, err)

			assert.True(t, isBinaryFile(testFile))
		})
	}

	textExtensions := []string{
		".txt", ".go", ".js", ".html", ".css", ".md", ".json",
	}

	for _, ext := range textExtensions {
		t.Run("text extension "+ext, func(t *testing.T) {
			testFile := filepath.Join(tempDir, "test"+ext)
			err := os.WriteFile(testFile, []byte("text content"), 0644)
			require.NoError(t, err)

			assert.False(t, isBinaryFile(testFile))
		})
	}

	t.Run("unknown extension with text content", func(t *testing.T) {
		testFile := filepath.Join(tempDir, "test.unknown")
		err := os.WriteFile(testFile, []byte("this is text content"), 0644)
		require.NoError(t, err)

		assert.False(t, isBinaryFile(testFile))
	})

	t.Run("unknown extension with binary content", func(t *testing.T) {
		testFile := filepath.Join(tempDir, "test.unknown")
		binaryContent := make([]byte, 100)
		for i := range binaryContent {
			binaryContent[i] = 0 // null bytes
		}
		err := os.WriteFile(testFile, binaryContent, 0644)
		require.NoError(t, err)

		assert.True(t, isBinaryFile(testFile))
	})
}

// TestIsBinaryContent tests the isBinaryContent function
func TestIsBinaryContent(t *testing.T) {
	tempDir := t.TempDir()

	t.Run("text content", func(t *testing.T) {
		testFile := filepath.Join(tempDir, "text.dat")
		err := os.WriteFile(testFile, []byte("Hello world\nThis is text content"), 0644)
		require.NoError(t, err)

		assert.False(t, isBinaryContent(testFile))
	})

	t.Run("binary content with null bytes", func(t *testing.T) {
		testFile := filepath.Join(tempDir, "binary.dat")
		content := []byte{72, 101, 108, 108, 111, 0, 87, 111, 114, 108, 100} // "Hello\0World"
		err := os.WriteFile(testFile, content, 0644)
		require.NoError(t, err)

		assert.True(t, isBinaryContent(testFile))
	})

	t.Run("high percentage of non-printable characters", func(t *testing.T) {
		testFile := filepath.Join(tempDir, "nonprintable.dat")
		content := make([]byte, 100)
		// Fill with non-printable characters (except a few)
		for i := range content {
			if i < 80 {
				content[i] = 1 // non-printable
			} else {
				content[i] = 65 // 'A'
			}
		}
		err := os.WriteFile(testFile, content, 0644)
		require.NoError(t, err)

		assert.True(t, isBinaryContent(testFile))
	})

	t.Run("printable characters with tabs and newlines", func(t *testing.T) {
		testFile := filepath.Join(tempDir, "printable.dat")
		content := "Hello\tWorld\nThis is a test\rWith carriage return"
		err := os.WriteFile(testFile, []byte(content), 0644)
		require.NoError(t, err)

		assert.False(t, isBinaryContent(testFile))
	})

	t.Run("file not found", func(t *testing.T) {
		nonExistentFile := filepath.Join(tempDir, "nonexistent.dat")

		// Should return false when file can't be opened
		assert.False(t, isBinaryContent(nonExistentFile))
	})

	t.Run("empty file", func(t *testing.T) {
		testFile := filepath.Join(tempDir, "empty.dat")
		err := os.WriteFile(testFile, []byte(""), 0644)
		require.NoError(t, err)

		assert.False(t, isBinaryContent(testFile))
	})
}

// TestReadTextConstants tests the defined constants
func TestReadTextConstants(t *testing.T) {
	assert.Equal(t, "ReadText", ReadTextToolName)
	assert.Equal(t, 2000, DefaultReadLimit)
	assert.Equal(t, 2000, MaxLineLength)
}

// TestInterfaceCompliance tests that readTextTool implements BaseTool
func TestInterfaceCompliance(t *testing.T) {
	mockPerms := &mockReadTextPermissionService{}
	tool := NewReadTextTool(mockPerms)

	// Verify it implements BaseTool interface
	var _ BaseTool = tool

	// Verify all required methods are available
	info := tool.Info()
	assert.NotEmpty(t, info.Name)

	// Test Run method signature
	ctx := context.Background()
	call := ToolCall{ID: "test", Name: "test", Input: "{}"}
	_, err := tool.Run(ctx, call)
	// We expect an error due to missing context values, but the method should exist
	assert.Error(t, err)
}

// TestErrorHandling tests various error conditions
func TestErrorHandling(t *testing.T) {
	tempDir := t.TempDir()
	mockPerms := &mockReadTextPermissionService{requestResponse: true}
	tool := NewReadTextTool(mockPerms).(*readTextTool)

	ctx := context.Background()
	ctx = context.WithValue(ctx, SessionIDContextKey, "test-session")
	ctx = context.WithValue(ctx, MessageIDContextKey, "test-message")

	t.Run("file permission error", func(t *testing.T) {
		// Create a file with no read permissions
		restrictedFile := filepath.Join(tempDir, "restricted.txt")
		err := os.WriteFile(restrictedFile, []byte("content"), 0000)
		require.NoError(t, err)

		call := ToolCall{
			ID:   "test-call",
			Name: ReadTextToolName,
			Input: `{"file_path":"` + restrictedFile + `"}`,
		}

		_, err = tool.Run(ctx, call)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "error accessing file")

		// Clean up by restoring permissions
		os.Chmod(restrictedFile, 0644)
	})

	t.Run("binary file rejection", func(t *testing.T) {
		// Create a binary file
		binaryFile := filepath.Join(tempDir, "binary.exe")
		err := os.WriteFile(binaryFile, []byte("dummy"), 0644)
		require.NoError(t, err)

		call := ToolCall{
			ID:   "test-call",
			Name: ReadTextToolName,
			Input: `{"file_path":"` + binaryFile + `"}`,
		}

		response, err := tool.Run(ctx, call)
		require.NoError(t, err)

		assert.True(t, response.IsError)
		assert.Contains(t, response.Content, "Cannot read binary file")
	})
}

// TestWithResponseMetadata tests the metadata functionality
func TestWithResponseMetadata(t *testing.T) {
	tempDir := t.TempDir()
	testFile := filepath.Join(tempDir, "test.txt")
	testContent := "test content"
	err := os.WriteFile(testFile, []byte(testContent), 0644)
	require.NoError(t, err)

	mockPerms := &mockReadTextPermissionService{requestResponse: true}
	tool := NewReadTextTool(mockPerms).(*readTextTool)

	ctx := context.Background()
	ctx = context.WithValue(ctx, SessionIDContextKey, "test-session")
	ctx = context.WithValue(ctx, MessageIDContextKey, "test-message")

	call := ToolCall{
		ID:   "test-call",
		Name: ReadTextToolName,
		Input: `{"file_path":"` + testFile + `"}`,
	}

	response, err := tool.Run(ctx, call)
	require.NoError(t, err)

	// Verify metadata is included
	assert.NotEmpty(t, response.Metadata)

	// Parse metadata
	var metadata ReadTextResponseMetadata
	err = json.Unmarshal([]byte(response.Metadata), &metadata)
	require.NoError(t, err)

	assert.Equal(t, testFile, metadata.FilePath)
	assert.Equal(t, testContent, metadata.Content)
}

// BenchmarkReadTextFile benchmarks the readTextFile function
func BenchmarkReadTextFile(b *testing.B) {
	tempDir := b.TempDir()
	testFile := filepath.Join(tempDir, "benchmark.txt")

	// Create a file with many lines
	var lines []string
	for i := 0; i < 1000; i++ {
		lines = append(lines, "This is line number "+string(rune(i)))
	}
	content := strings.Join(lines, "\n")
	err := os.WriteFile(testFile, []byte(content), 0644)
	require.NoError(b, err)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _, err := readTextFile(testFile, 0, 100)
		if err != nil {
			b.Fatal(err)
		}
	}
}

// BenchmarkIsBinaryFile benchmarks the isBinaryFile function
func BenchmarkIsBinaryFile(b *testing.B) {
	tempDir := b.TempDir()
	testFile := filepath.Join(tempDir, "benchmark.txt")

	content := strings.Repeat("This is a text file with some content. ", 100)
	err := os.WriteFile(testFile, []byte(content), 0644)
	require.NoError(b, err)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		isBinaryFile(testFile)
	}
}