package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"mix/internal/permission"
	"mix/internal/pubsub"
)

// MockGrepPermissionService implements the permission.Service interface for testing
type MockGrepPermissionService struct {
	mock.Mock
}

func (m *MockGrepPermissionService) Request(opts permission.CreatePermissionRequest) bool {
	args := m.Called(opts)
	return args.Bool(0)
}

func (m *MockGrepPermissionService) Subscribe(ctx context.Context) <-chan pubsub.Event[permission.PermissionRequest] {
	args := m.Called(ctx)
	return args.Get(0).(<-chan pubsub.Event[permission.PermissionRequest])
}

func (m *MockGrepPermissionService) GrantPersistant(permission permission.PermissionRequest) {
	m.Called(permission)
}

func (m *MockGrepPermissionService) Grant(permission permission.PermissionRequest) {
	m.Called(permission)
}

func (m *MockGrepPermissionService) Deny(permission permission.PermissionRequest) {
	m.Called(permission)
}

// Test helper functions
func createGrepTestContext(sessionID, messageID, storageDir string) context.Context {
	ctx := context.Background()
	ctx = context.WithValue(ctx, SessionIDContextKey, sessionID)
	ctx = context.WithValue(ctx, MessageIDContextKey, messageID)
	ctx = context.WithValue(ctx, SessionStorageContextKey, storageDir)
	return ctx
}

func createTestFile(t *testing.T, dir, filename, content string) string {
	filePath := filepath.Join(dir, filename)
	err := os.MkdirAll(filepath.Dir(filePath), 0755)
	require.NoError(t, err)

	err = os.WriteFile(filePath, []byte(content), 0644)
	require.NoError(t, err)

	return filePath
}

// Test GrepParams JSON serialization/deserialization
func TestGrepParams_JSONSerialization(t *testing.T) {
	tests := []struct {
		name     string
		params   GrepParams
		expected string
	}{
		{
			name: "complete params",
			params: GrepParams{
				Pattern:     "test.*pattern",
				Path:        "/path/to/search",
				Include:     "*.go",
				LiteralText: true,
			},
			expected: `{"pattern":"test.*pattern","path":"/path/to/search","include":"*.go","literal_text":true}`,
		},
		{
			name: "minimal params",
			params: GrepParams{
				Pattern: "simple",
			},
			expected: `{"pattern":"simple","path":"","include":"","literal_text":false}`,
		},
		{
			name: "with special characters",
			params: GrepParams{
				Pattern:     "test\\.\\*\\+\\?",
				Path:        "/path with spaces/",
				Include:     "*.{js,ts}",
				LiteralText: false,
			},
			expected: `{"pattern":"test\\.\\*\\+\\?","path":"/path with spaces/","include":"*.{js,ts}","literal_text":false}`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Test serialization
			jsonData, err := json.Marshal(tt.params)
			assert.NoError(t, err)
			assert.JSONEq(t, tt.expected, string(jsonData))

			// Test deserialization
			var params GrepParams
			err = json.Unmarshal(jsonData, &params)
			assert.NoError(t, err)
			assert.Equal(t, tt.params, params)
		})
	}
}

// Test GrepResponseMetadata JSON serialization
func TestGrepResponseMetadata_JSONSerialization(t *testing.T) {
	metadata := GrepResponseMetadata{
		NumberOfMatches: 42,
		Truncated:       true,
	}

	jsonData, err := json.Marshal(metadata)
	assert.NoError(t, err)

	expected := `{"number_of_matches":42,"truncated":true}`
	assert.JSONEq(t, expected, string(jsonData))

	var unmarshaled GrepResponseMetadata
	err = json.Unmarshal(jsonData, &unmarshaled)
	assert.NoError(t, err)
	assert.Equal(t, metadata, unmarshaled)
}

// Test NewGrepTool constructor and interface compliance
func TestNewGrepTool(t *testing.T) {
	mockPermission := &MockGrepPermissionService{}

	tool := NewGrepTool(mockPermission)

	// Verify interface compliance
	assert.Implements(t, (*BaseTool)(nil), tool)

	// Verify internal structure
	grepTool, ok := tool.(*grepTool)
	assert.True(t, ok)
	assert.Equal(t, mockPermission, grepTool.permissions)
}

// Test Info method
func TestGrepTool_Info(t *testing.T) {
	mockPermission := &MockGrepPermissionService{}
	tool := NewGrepTool(mockPermission)

	info := tool.Info()

	assert.Equal(t, GrepToolName, info.Name)
	assert.Equal(t, "grep", info.Name)
	assert.NotEmpty(t, info.Description)
	assert.Contains(t, info.Required, "pattern")
	assert.Len(t, info.Required, 1)

	// Check parameters
	assert.Contains(t, info.Parameters, "pattern")
	assert.Contains(t, info.Parameters, "path")
	assert.Contains(t, info.Parameters, "include")
	assert.Contains(t, info.Parameters, "literal_text")

	// Verify parameter types and descriptions
	patternParam, ok := info.Parameters["pattern"].(map[string]any)
	assert.True(t, ok)
	assert.Equal(t, "string", patternParam["type"])
	assert.NotEmpty(t, patternParam["description"])

	pathParam, ok := info.Parameters["path"].(map[string]any)
	assert.True(t, ok)
	assert.Equal(t, "string", pathParam["type"])
	assert.NotEmpty(t, pathParam["description"])

	includeParam, ok := info.Parameters["include"].(map[string]any)
	assert.True(t, ok)
	assert.Equal(t, "string", includeParam["type"])
	assert.NotEmpty(t, includeParam["description"])

	literalParam, ok := info.Parameters["literal_text"].(map[string]any)
	assert.True(t, ok)
	assert.Equal(t, "boolean", literalParam["type"])
	assert.NotEmpty(t, literalParam["description"])
}

// Test escapeRegexPattern function
func TestEscapeRegexPattern(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "no special characters",
			input:    "simple",
			expected: "simple",
		},
		{
			name:     "backslash",
			input:    "test\\path",
			expected: "test\\\\path",
		},
		{
			name:     "dot",
			input:    "test.txt",
			expected: "test\\.txt",
		},
		{
			name:     "plus and asterisk",
			input:    "test+*",
			expected: "test\\+\\*",
		},
		{
			name:     "question mark",
			input:    "test?",
			expected: "test\\?",
		},
		{
			name:     "parentheses",
			input:    "test(group)",
			expected: "test\\(group\\)",
		},
		{
			name:     "square brackets",
			input:    "test[chars]",
			expected: "test\\[chars\\]",
		},
		{
			name:     "curly braces",
			input:    "test{1,2}",
			expected: "test\\{1,2\\}",
		},
		{
			name:     "caret and dollar",
			input:    "^start$",
			expected: "\\^start\\$",
		},
		{
			name:     "pipe",
			input:    "test|pipe",
			expected: "test\\|pipe",
		},
		{
			name:     "all special characters",
			input:    "\\.+*?()[]{}^$|",
			expected: "\\\\\\.\\+\\*\\?\\(\\)\\[\\]\\{\\}\\^\\$\\|",
		},
		{
			name:     "mixed content",
			input:    "function.*test(param)",
			expected: "function\\.\\*test\\(param\\)",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := escapeRegexPattern(tt.input)
			assert.Equal(t, tt.expected, result)

			// Verify the escaped pattern can be compiled as regex
			_, err := regexp.Compile(result)
			assert.NoError(t, err, "escaped pattern should be valid regex")
		})
	}
}

// Test Run method with various scenarios
func TestGrepTool_Run(t *testing.T) {
	// Create temporary directory for testing
	tempDir, err := os.MkdirTemp("", "grep_test")
	require.NoError(t, err)
	defer os.RemoveAll(tempDir)

	// Create test files
	createTestFile(t, tempDir, "test1.go", "package main\nfunc main() {\n\tfmt.Println(\"hello world\")\n}")
	createTestFile(t, tempDir, "test2.go", "package test\nfunc TestSomething() {\n\tt.Error(\"test error\")\n}")
	createTestFile(t, tempDir, "readme.txt", "This is a readme file\nwith some content")

	tests := []struct {
		name           string
		input          string
		permissionResp bool
		expectError    bool
		expectContent  string
		setupContext   func() context.Context
	}{
		{
			name:           "successful search with permission",
			input:          `{"pattern":"func","path":"` + tempDir + `"}`,
			permissionResp: true,
			expectError:    false,
			expectContent:  "Found",
			setupContext: func() context.Context {
				return createGrepTestContext("session-123", "message-456", tempDir)
			},
		},
		{
			name:           "permission denied",
			input:          `{"pattern":"func","path":"` + tempDir + `"}`,
			permissionResp: false,
			expectError:    true,
			expectContent:  "",
			setupContext: func() context.Context {
				return createGrepTestContext("session-123", "message-456", tempDir)
			},
		},
		{
			name:        "invalid JSON input",
			input:       `{"pattern":}`,
			expectError: false, // Returns error response, not error
			expectContent: "error parsing parameters",
			setupContext: func() context.Context {
				return createGrepTestContext("session-123", "message-456", tempDir)
			},
		},
		{
			name:        "missing pattern",
			input:       `{"path":"` + tempDir + `"}`,
			expectError: false,
			expectContent: "pattern is required",
			setupContext: func() context.Context {
				return createGrepTestContext("session-123", "message-456", tempDir)
			},
		},
		{
			name:        "missing session context",
			input:       `{"pattern":"test"}`,
			expectError: true,
			expectContent: "session ID and message ID are required",
			setupContext: func() context.Context {
				return context.Background()
			},
		},
		{
			name:           "literal text search",
			input:          `{"pattern":"func()","literal_text":true,"path":"` + tempDir + `"}`,
			permissionResp: true,
			expectError:    false,
			expectContent:  "No files found", // Should not match because func() is escaped
			setupContext: func() context.Context {
				return createGrepTestContext("session-123", "message-456", tempDir)
			},
		},
		{
			name:           "search with include pattern",
			input:          `{"pattern":"package","include":"*.go","path":"` + tempDir + `"}`,
			permissionResp: true,
			expectError:    false,
			expectContent:  "Found",
			setupContext: func() context.Context {
				return createGrepTestContext("session-123", "message-456", tempDir)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockPermission := &MockGrepPermissionService{}
			if tt.permissionResp || tt.name == "permission denied" {
				mockPermission.On("Request", mock.AnythingOfType("permission.CreatePermissionRequest")).Return(tt.permissionResp)
			}

			tool := NewGrepTool(mockPermission)
			ctx := tt.setupContext()

			call := ToolCall{
				ID:    "test-call",
				Name:  "grep",
				Input: tt.input,
			}

			response, err := tool.Run(ctx, call)

			if tt.expectError {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tt.expectContent)
			} else {
				assert.NoError(t, err)
				assert.Contains(t, response.Content, tt.expectContent)

				if strings.Contains(tt.expectContent, "Found") {
					// Verify metadata is included
					assert.NotEmpty(t, response.Metadata)
					var metadata GrepResponseMetadata
					err := json.Unmarshal([]byte(response.Metadata), &metadata)
					assert.NoError(t, err)
					assert.Greater(t, metadata.NumberOfMatches, 0)
				}
			}

			mockPermission.AssertExpectations(t)
		})
	}
}

// Test searchFiles function
func TestSearchFiles(t *testing.T) {
	// Create temporary directory for testing
	tempDir, err := os.MkdirTemp("", "search_files_test")
	require.NoError(t, err)
	defer os.RemoveAll(tempDir)

	// Create test files with different patterns
	createTestFile(t, tempDir, "test1.go", "package main\nfunc main() {\n\tfmt.Println(\"hello\")\n}")
	createTestFile(t, tempDir, "test2.js", "function test() {\n\tconsole.log('test');\n}")
	createTestFile(t, tempDir, "readme.md", "# Test\nThis is a test file")

	tests := []struct {
		name         string
		pattern      string
		include      string
		limit        int
		expectMatch  bool
		expectTrunc  bool
	}{
		{
			name:        "find function keyword",
			pattern:     "func",
			include:     "",
			limit:       100,
			expectMatch: true,
			expectTrunc: false,
		},
		{
			name:        "find with file filter",
			pattern:     "function",
			include:     "*.js",
			limit:       100,
			expectMatch: true,
			expectTrunc: false,
		},
		{
			name:        "no matches",
			pattern:     "nonexistent",
			include:     "",
			limit:       100,
			expectMatch: false,
			expectTrunc: false,
		},
		{
			name:        "test truncation",
			pattern:     "test|main|function", // Should match multiple files
			include:     "",
			limit:       1,
			expectMatch: true,
			expectTrunc: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			matches, truncated, err := searchFiles(tt.pattern, tempDir, tt.include, tt.limit)

			assert.NoError(t, err)
			assert.Equal(t, tt.expectTrunc, truncated)

			if tt.expectMatch {
				assert.Greater(t, len(matches), 0)
				// Verify matches are sorted by modification time (newest first)
				for i := 1; i < len(matches); i++ {
					assert.True(t, matches[i-1].modTime.After(matches[i].modTime) || matches[i-1].modTime.Equal(matches[i].modTime))
				}
			} else {
				assert.Len(t, matches, 0)
			}
		})
	}
}

// Test searchWithRipgrep function
func TestSearchWithRipgrep(t *testing.T) {
	// Check if ripgrep is available
	_, err := exec.LookPath("rg")
	if err != nil {
		t.Skip("Ripgrep not available, skipping ripgrep tests")
	}

	// Create temporary directory for testing
	tempDir, err := os.MkdirTemp("", "ripgrep_test")
	require.NoError(t, err)
	defer os.RemoveAll(tempDir)

	// Create test files
	createTestFile(t, tempDir, "test1.go", "package main\nfunc main() {\n\tfmt.Println(\"hello\")\n}")
	createTestFile(t, tempDir, "test2.txt", "This is a test file\nwith multiple lines")

	tests := []struct {
		name        string
		pattern     string
		include     string
		expectMatch bool
	}{
		{
			name:        "find package keyword",
			pattern:     "package",
			include:     "",
			expectMatch: true,
		},
		{
			name:        "find with file filter",
			pattern:     "package",
			include:     "*.go",
			expectMatch: true,
		},
		{
			name:        "no matches",
			pattern:     "nonexistent",
			include:     "",
			expectMatch: false,
		},
		{
			name:        "regex pattern",
			pattern:     "func.*main",
			include:     "",
			expectMatch: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			matches, err := searchWithRipgrep(tt.pattern, tempDir, tt.include)

			assert.NoError(t, err)

			if tt.expectMatch {
				assert.Greater(t, len(matches), 0)
				// Verify match structure
				for _, match := range matches {
					assert.NotEmpty(t, match.path)
					assert.Greater(t, match.lineNum, 0)
					assert.NotEmpty(t, match.lineText)
					assert.False(t, match.modTime.IsZero())
				}
			} else {
				assert.Len(t, matches, 0)
			}
		})
	}
}

// Test searchFilesWithRegex function
func TestSearchFilesWithRegex(t *testing.T) {
	// Create temporary directory for testing
	tempDir, err := os.MkdirTemp("", "regex_search_test")
	require.NoError(t, err)
	defer os.RemoveAll(tempDir)

	// Create test files
	createTestFile(t, tempDir, "test1.go", "package main\nfunc main() {\n\tfmt.Println(\"hello\")\n}")
	createTestFile(t, tempDir, "test2.js", "function test() {\n\tconsole.log('test');\n}")
	createTestFile(t, tempDir, "subdir/test3.py", "def test():\n    print('hello')")

	tests := []struct {
		name        string
		pattern     string
		include     string
		expectMatch bool
		expectError bool
	}{
		{
			name:        "valid regex pattern",
			pattern:     "func.*main",
			include:     "",
			expectMatch: true,
			expectError: false,
		},
		{
			name:        "invalid regex pattern",
			pattern:     "[invalid",
			include:     "",
			expectMatch: false,
			expectError: true,
		},
		{
			name:        "search with include pattern",
			pattern:     "function",
			include:     "*.js",
			expectMatch: true,
			expectError: false,
		},
		{
			name:        "no matches",
			pattern:     "nonexistent",
			include:     "",
			expectMatch: false,
			expectError: false,
		},
		{
			name:        "invalid include pattern",
			pattern:     "test",
			include:     "[invalid",
			expectMatch: false,
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			matches, err := searchFilesWithRegex(tt.pattern, tempDir, tt.include)

			if tt.expectError {
				assert.Error(t, err)
				return
			}

			assert.NoError(t, err)

			if tt.expectMatch {
				assert.Greater(t, len(matches), 0)
				// Verify match structure
				for _, match := range matches {
					assert.NotEmpty(t, match.path)
					assert.Greater(t, match.lineNum, 0)
					assert.NotEmpty(t, match.lineText)
					assert.False(t, match.modTime.IsZero())
				}
			} else {
				assert.Len(t, matches, 0)
			}
		})
	}
}

// Test fileContainsPattern function
func TestFileContainsPattern(t *testing.T) {
	// Create temporary directory for testing
	tempDir, err := os.MkdirTemp("", "file_pattern_test")
	require.NoError(t, err)
	defer os.RemoveAll(tempDir)

	// Create test files
	testContent := "line 1: normal content\nline 2: special pattern here\nline 3: more content"
	testFile := createTestFile(t, tempDir, "test.txt", testContent)

	tests := []struct {
		name           string
		pattern        string
		expectMatch    bool
		expectLineNum  int
		expectLineText string
		expectError    bool
	}{
		{
			name:           "pattern exists",
			pattern:        "special pattern",
			expectMatch:    true,
			expectLineNum:  2,
			expectLineText: "line 2: special pattern here",
			expectError:    false,
		},
		{
			name:        "pattern does not exist",
			pattern:     "nonexistent",
			expectMatch: false,
			expectError: false,
		},
		{
			name:           "regex pattern",
			pattern:        "line \\d+:",
			expectMatch:    true,
			expectLineNum:  1,
			expectLineText: "line 1: normal content",
			expectError:    false,
		},
		{
			name:        "invalid regex",
			pattern:     "[invalid",
			expectMatch: false,
			expectError: false, // regexp.Compile will fail, but we handle that in calling function
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			regex, err := regexp.Compile(tt.pattern)
			if err != nil {
				t.Skip("Invalid regex pattern for this test")
			}

			match, lineNum, lineText, err := fileContainsPattern(testFile, regex)

			if tt.expectError {
				assert.Error(t, err)
				return
			}

			assert.NoError(t, err)
			assert.Equal(t, tt.expectMatch, match)

			if tt.expectMatch {
				assert.Equal(t, tt.expectLineNum, lineNum)
				assert.Equal(t, tt.expectLineText, lineText)
			}
		})
	}

	// Test with non-existent file
	t.Run("non-existent file", func(t *testing.T) {
		regex := regexp.MustCompile("test")
		match, lineNum, lineText, err := fileContainsPattern("/nonexistent/file.txt", regex)

		assert.Error(t, err)
		assert.False(t, match)
		assert.Equal(t, 0, lineNum)
		assert.Empty(t, lineText)
	})
}

// Test globToRegex function
func TestGlobToRegex(t *testing.T) {
	tests := []struct {
		name     string
		glob     string
		expected string
		testStr  string
		matches  bool
	}{
		{
			name:     "simple wildcard",
			glob:     "*.go",
			expected: ".*\\.go",
			testStr:  "test.go",
			matches:  true,
		},
		{
			name:     "question mark",
			glob:     "test?.go",
			expected: "test.\\.go",
			testStr:  "test1.go",
			matches:  true,
		},
		{
			name:     "braces expansion",
			glob:     "*.{js,ts}",
			expected: ".*\\.(js|ts)",
			testStr:  "test.ts",
			matches:  true,
		},
		{
			name:     "complex braces",
			glob:     "test.{js,ts,jsx}",
			expected: "test\\.(js|ts|jsx)",
			testStr:  "test.jsx",
			matches:  true,
		},
		{
			name:     "escaped dot",
			glob:     "test.file",
			expected: "test\\.file",
			testStr:  "test.file",
			matches:  true,
		},
		{
			name:     "multiple patterns",
			glob:     "*test*.{go,js}",
			expected: ".*test.*\\.(go|js)",
			testStr:  "mytestfile.go",
			matches:  true,
		},
		{
			name:     "no special characters",
			glob:     "simple",
			expected: "simple",
			testStr:  "simple",
			matches:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := globToRegex(tt.glob)
			assert.Equal(t, tt.expected, result)

			// Test that the generated regex works correctly
			regex, err := regexp.Compile(result)
			assert.NoError(t, err, "generated regex should be valid")

			matches := regex.MatchString(tt.testStr)
			assert.Equal(t, tt.matches, matches, "regex should match test string correctly")
		})
	}
}

// Test edge cases and error scenarios
func TestGrepTool_EdgeCases(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "grep_edge_cases")
	require.NoError(t, err)
	defer os.RemoveAll(tempDir)

	t.Run("empty search directory", func(t *testing.T) {
		emptyDir := filepath.Join(tempDir, "empty")
		err := os.MkdirAll(emptyDir, 0755)
		require.NoError(t, err)

		mockPermission := &MockGrepPermissionService{}
		mockPermission.On("Request", mock.AnythingOfType("permission.CreatePermissionRequest")).Return(true)

		tool := NewGrepTool(mockPermission)
		ctx := createGrepTestContext("session-123", "message-456", tempDir)

		call := ToolCall{
			ID:    "test-call",
			Name:  "grep",
			Input: fmt.Sprintf(`{"pattern":"test","path":"%s"}`, emptyDir),
		}

		response, err := tool.Run(ctx, call)
		assert.NoError(t, err)
		assert.Contains(t, response.Content, "No files found")
	})

	t.Run("malformed JSON input", func(t *testing.T) {
		mockPermission := &MockGrepPermissionService{}
		tool := NewGrepTool(mockPermission)
		ctx := createGrepTestContext("session-123", "message-456", tempDir)

		call := ToolCall{
			ID:    "test-call",
			Name:  "grep",
			Input: `{"pattern":"test",}`, // Invalid JSON
		}

		response, err := tool.Run(ctx, call)
		assert.NoError(t, err)
		assert.True(t, response.IsError)
		assert.Contains(t, response.Content, "error parsing parameters")
	})

	t.Run("default path resolution", func(t *testing.T) {
		mockPermission := &MockGrepPermissionService{}
		mockPermission.On("Request", mock.AnythingOfType("permission.CreatePermissionRequest")).Return(true)

		tool := NewGrepTool(mockPermission)
		ctx := createGrepTestContext("session-123", "message-456", tempDir)

		call := ToolCall{
			ID:    "test-call",
			Name:  "grep",
			Input: `{"pattern":"test"}`, // No path specified
		}

		_, err := tool.Run(ctx, call)
		assert.NoError(t, err)
		// Should use session storage directory as default
	})

	t.Run("context without storage directory", func(t *testing.T) {
		mockPermission := &MockGrepPermissionService{}
		tool := NewGrepTool(mockPermission)

		// Create context without storage directory
		ctx := context.Background()
		ctx = context.WithValue(ctx, SessionIDContextKey, "session-123")
		ctx = context.WithValue(ctx, MessageIDContextKey, "message-456")

		call := ToolCall{
			ID:    "test-call",
			Name:  "grep",
			Input: `{"pattern":"test"}`,
		}

		_, err := tool.Run(ctx, call)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "failed to get session storage directory")
	})
}

// Test performance and limits
func TestGrepTool_Performance(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping performance test in short mode")
	}

	tempDir, err := os.MkdirTemp("", "grep_performance")
	require.NoError(t, err)
	defer os.RemoveAll(tempDir)

	// Create many files for performance testing
	for i := 0; i < 50; i++ {
		content := fmt.Sprintf("file %d content\nwith some test data\nand more lines", i)
		createTestFile(t, tempDir, fmt.Sprintf("file%d.txt", i), content)
	}

	mockPermission := &MockGrepPermissionService{}
	mockPermission.On("Request", mock.AnythingOfType("permission.CreatePermissionRequest")).Return(true)

	tool := NewGrepTool(mockPermission)
	ctx := createGrepTestContext("session-123", "message-456", tempDir)

	start := time.Now()
	call := ToolCall{
		ID:    "test-call",
		Name:  "grep",
		Input: fmt.Sprintf(`{"pattern":"content","path":"%s"}`, tempDir),
	}

	response, err := tool.Run(ctx, call)
	duration := time.Since(start)

	assert.NoError(t, err)
	assert.Contains(t, response.Content, "Found")
	assert.Less(t, duration, 5*time.Second, "Search should complete within 5 seconds")

	// Verify truncation metadata
	assert.NotEmpty(t, response.Metadata)
	var metadata GrepResponseMetadata
	err = json.Unmarshal([]byte(response.Metadata), &metadata)
	assert.NoError(t, err)
	assert.Greater(t, metadata.NumberOfMatches, 0)
}

// Benchmark tests
func BenchmarkEscapeRegexPattern(b *testing.B) {
	pattern := "test.with+special*chars?and(more)[brackets]{and}^caret$"

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		escapeRegexPattern(pattern)
	}
}

func BenchmarkGlobToRegex(b *testing.B) {
	glob := "*.{js,ts,jsx,tsx}"

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		globToRegex(glob)
	}
}

// Test constants and exported values
func TestGrepConstants(t *testing.T) {
	assert.Equal(t, "grep", GrepToolName)
}

// Test grepMatch struct
func TestGrepMatch(t *testing.T) {
	now := time.Now()
	match := grepMatch{
		path:     "/test/path.go",
		modTime:  now,
		lineNum:  42,
		lineText: "test line content",
	}

	assert.Equal(t, "/test/path.go", match.path)
	assert.Equal(t, now, match.modTime)
	assert.Equal(t, 42, match.lineNum)
	assert.Equal(t, "test line content", match.lineText)
}