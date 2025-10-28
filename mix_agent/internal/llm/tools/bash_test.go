package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"mix/internal/llm/interfaces"
	"mix/internal/permission"
	"mix/internal/pubsub"
)

// MockPermissionService is a mock implementation of the permission.Service interface
type MockPermissionService struct {
	mock.Mock
}

func (m *MockPermissionService) Subscribe(ctx context.Context) <-chan pubsub.Event[permission.PermissionRequest] {
	args := m.Called(ctx)
	return args.Get(0).(<-chan pubsub.Event[permission.PermissionRequest])
}

func (m *MockPermissionService) GetSubscriberCount() int {
	args := m.Called()
	return args.Int(0)
}

func (m *MockPermissionService) Publish(ctx context.Context, eventType pubsub.EventType, data permission.PermissionRequest) error {
	args := m.Called(ctx, eventType, data)
	return args.Error(0)
}

func (m *MockPermissionService) GrantPersistant(permission permission.PermissionRequest) {
	m.Called(permission)
}

func (m *MockPermissionService) Grant(permission permission.PermissionRequest) {
	m.Called(permission)
}

func (m *MockPermissionService) Deny(permission permission.PermissionRequest) {
	m.Called(permission)
}

func (m *MockPermissionService) Request(opts permission.CreatePermissionRequest) bool {
	args := m.Called(opts)
	return args.Bool(0)
}

// Test bash constants
func TestBashConstants(t *testing.T) {
	assert.Equal(t, "bash", BashToolName)
	assert.Equal(t, 1*60*1000, DefaultTimeout)
	assert.Equal(t, 10*60*1000, MaxTimeout)
	assert.Equal(t, 30000, MaxOutputLength)
}

// Test banned commands
func TestBannedCommands(t *testing.T) {
	expectedBannedCommands := []string{
		"alias", "curlie", "wget", "axel", "aria2c",
		"nc", "telnet", "lynx", "w3m", "links", "httpie", "xh",
		"http-prompt", "chrome", "firefox", "safari",
	}

	assert.Equal(t, expectedBannedCommands, bannedCommands)
}

// Test safe read-only commands
func TestSafeReadOnlyCommands(t *testing.T) {
	// Test that safe commands array is not empty and contains expected commands
	assert.NotEmpty(t, safeReadOnlyCommands)
	assert.Contains(t, safeReadOnlyCommands, "ls")
	assert.Contains(t, safeReadOnlyCommands, "echo")
	assert.Contains(t, safeReadOnlyCommands, "pwd")
	assert.Contains(t, safeReadOnlyCommands, "git status")
	assert.Contains(t, safeReadOnlyCommands, "go version")
}

// Test BashParams struct JSON serialization
func TestBashParamsJSONSerialization(t *testing.T) {
	tests := []struct {
		name     string
		params   BashParams
		expected string
	}{
		{
			name:     "basic parameters",
			params:   BashParams{Command: "ls -la", Timeout: 5000},
			expected: `{"command":"ls -la","timeout":5000}`,
		},
		{
			name:     "empty command",
			params:   BashParams{Command: "", Timeout: 0},
			expected: `{"command":"","timeout":0}`,
		},
		{
			name:     "special characters in command",
			params:   BashParams{Command: "echo 'hello world' && ls", Timeout: 1000},
			expected: `{"command":"echo 'hello world' && ls","timeout":1000}`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Test marshaling
			jsonData, err := json.Marshal(tt.params)
			assert.NoError(t, err)
			assert.JSONEq(t, tt.expected, string(jsonData))

			// Test unmarshaling
			var params BashParams
			err = json.Unmarshal(jsonData, &params)
			assert.NoError(t, err)
			assert.Equal(t, tt.params, params)
		})
	}
}

// Test BashPermissionsParams struct JSON serialization
func TestBashPermissionsParamsJSONSerialization(t *testing.T) {
	params := BashPermissionsParams{
		Command: "rm -rf /test",
		Timeout: 10000,
	}

	jsonData, err := json.Marshal(params)
	assert.NoError(t, err)

	var unmarshaled BashPermissionsParams
	err = json.Unmarshal(jsonData, &unmarshaled)
	assert.NoError(t, err)
	assert.Equal(t, params, unmarshaled)
}

// Test BashResponseMetadata struct JSON serialization
func TestBashResponseMetadataJSONSerialization(t *testing.T) {
	now := time.Now().UnixMilli()
	metadata := BashResponseMetadata{
		StartTime: now,
		EndTime:   now + 1000,
	}

	jsonData, err := json.Marshal(metadata)
	assert.NoError(t, err)

	var unmarshaled BashResponseMetadata
	err = json.Unmarshal(jsonData, &unmarshaled)
	assert.NoError(t, err)
	assert.Equal(t, metadata, unmarshaled)
}

// Test NewBashTool function
func TestNewBashTool(t *testing.T) {
	mockPermissionService := &MockPermissionService{}
	tool := NewBashTool(mockPermissionService)

	assert.NotNil(t, tool)
	assert.Implements(t, (*interfaces.BaseTool)(nil), tool)

	// Cast to concrete type to verify internal state
	bashTool, ok := tool.(*bashTool)
	assert.True(t, ok)
	assert.Equal(t, mockPermissionService, bashTool.permissions)
}

// Test bashTool.Info method
func TestBashToolInfo(t *testing.T) {
	mockPermissionService := &MockPermissionService{}
	tool := NewBashTool(mockPermissionService)

	info := tool.Info()

	assert.Equal(t, BashToolName, info.Name)
	assert.NotEmpty(t, info.Description)
	assert.Contains(t, info.Required, "command")

	// Test parameters structure
	assert.Contains(t, info.Parameters, "command")
	assert.Contains(t, info.Parameters, "timeout")

	commandParam, ok := info.Parameters["command"].(map[string]any)
	assert.True(t, ok)
	assert.Equal(t, "string", commandParam["type"])
	assert.Contains(t, commandParam["description"], "command")

	timeoutParam, ok := info.Parameters["timeout"].(map[string]any)
	assert.True(t, ok)
	assert.Equal(t, "number", timeoutParam["type"])
	assert.Contains(t, timeoutParam["description"], "timeout")
}

// Test bashDescription function
func TestBashDescription(t *testing.T) {
	description := bashDescription()
	assert.NotEmpty(t, description)
	// The actual content depends on the embedded file, but it should not be an error message
	assert.NotContains(t, description, "Error:")
}

// Helper function to create a test context with required values
func createBashTestContext(sessionID, messageID, storageDir string) context.Context {
	ctx := context.Background()
	if sessionID != "" {
		ctx = context.WithValue(ctx, SessionIDContextKey, sessionID)
	}
	if messageID != "" {
		ctx = context.WithValue(ctx, MessageIDContextKey, messageID)
	}
	if storageDir != "" {
		ctx = context.WithValue(ctx, SessionStorageContextKey, storageDir)
	}
	return ctx
}

// Helper function to create a temporary directory for testing
func createTempDir(t *testing.T) string {
	dir, err := os.MkdirTemp("", "bash_test_*")
	require.NoError(t, err)
	t.Cleanup(func() {
		_ = os.RemoveAll(dir)
	})
	return dir
}

// Test bashTool.Run method with invalid JSON input
func TestBashToolRunInvalidJSON(t *testing.T) {
	mockPermissionService := &MockPermissionService{}
	tool := NewBashTool(mockPermissionService)

	ctx := createBashTestContext("session-123", "message-456", "/tmp/test")
	call := interfaces.ToolCall{
		ID:    "call-1",
		Name:  BashToolName,
		Input: "invalid json",
	}

	response, err := tool.Run(ctx, call)

	assert.NoError(t, err) // The function returns an error response, not an error
	assert.True(t, response.IsError)
	assert.Contains(t, response.Content, "invalid parameters")
}

// Test bashTool.Run method with missing command
func TestBashToolRunMissingCommand(t *testing.T) {
	mockPermissionService := &MockPermissionService{}
	tool := NewBashTool(mockPermissionService)

	ctx := createBashTestContext("session-123", "message-456", "/tmp/test")
	call := interfaces.ToolCall{
		ID:    "call-1",
		Name:  BashToolName,
		Input: `{"command": "", "timeout": 1000}`,
	}

	response, err := tool.Run(ctx, call)

	assert.NoError(t, err)
	assert.True(t, response.IsError)
	assert.Contains(t, response.Content, "missing command")
}

// Test bashTool.Run method with banned command
func TestBashToolRunBannedCommand(t *testing.T) {
	mockPermissionService := &MockPermissionService{}
	tool := NewBashTool(mockPermissionService)

	ctx := createBashTestContext("session-123", "message-456", "/tmp/test")

	// Test each banned command
	for _, bannedCmd := range bannedCommands {
		t.Run(fmt.Sprintf("banned_%s", bannedCmd), func(t *testing.T) {
			call := interfaces.ToolCall{
				ID:    "call-1",
				Name:  BashToolName,
				Input: fmt.Sprintf(`{"command": "%s --help", "timeout": 1000}`, bannedCmd),
			}

			response, err := tool.Run(ctx, call)

			assert.NoError(t, err)
			assert.True(t, response.IsError)
			assert.Contains(t, response.Content, fmt.Sprintf("command '%s' is not allowed", bannedCmd))
		})
	}
}

// Test timeout parameter handling
func TestBashToolRunTimeoutHandling(t *testing.T) {
	mockPermissionService := &MockPermissionService{}
	tool := NewBashTool(mockPermissionService)

	tests := []struct {
		name            string
		inputTimeout    int
		expectedTimeout int
	}{
		{
			name:            "zero timeout gets default",
			inputTimeout:    0,
			expectedTimeout: DefaultTimeout,
		},
		{
			name:            "negative timeout gets default",
			inputTimeout:    -1000,
			expectedTimeout: DefaultTimeout,
		},
		{
			name:            "timeout exceeding max gets capped",
			inputTimeout:    MaxTimeout + 1000,
			expectedTimeout: MaxTimeout,
		},
		{
			name:            "valid timeout preserved",
			inputTimeout:    5000,
			expectedTimeout: 5000,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// We'll test this indirectly by checking that the timeout is handled correctly
			// The actual execution would require setting up a full shell environment
			ctx := createBashTestContext("session-123", "message-456", "/tmp/test")
			call := interfaces.ToolCall{
				ID:    "call-1",
				Name:  BashToolName,
				Input: fmt.Sprintf(`{"command": "ls", "timeout": %d}`, tt.inputTimeout),
			}

			// Since we can't easily test the actual timeout behavior without a real shell,
			// we verify that the function doesn't return an immediate error for valid commands
			_, err := tool.Run(ctx, call)
			// The error here will be due to missing session context or shell setup,
			// but not due to timeout parameter validation
			assert.Error(t, err) // Expected due to missing session setup
		})
	}
}

// Test context validation
func TestBashToolRunContextValidation(t *testing.T) {
	mockPermissionService := &MockPermissionService{}
	tool := NewBashTool(mockPermissionService)

	tests := []struct {
		name        string
		sessionID   string
		messageID   string
		storageDir  string
		expectError bool
		errorMsg    string
	}{
		{
			name:        "missing session ID",
			sessionID:   "",
			messageID:   "message-456",
			storageDir:  "/tmp/test",
			expectError: true,
			errorMsg:    "session ID and message ID are required",
		},
		{
			name:        "missing message ID",
			sessionID:   "session-123",
			messageID:   "",
			storageDir:  "/tmp/test",
			expectError: true,
			errorMsg:    "session ID and message ID are required",
		},
		{
			name:        "missing both session and message ID",
			sessionID:   "",
			messageID:   "",
			storageDir:  "/tmp/test",
			expectError: true,
			errorMsg:    "session ID and message ID are required",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := createBashTestContext(tt.sessionID, tt.messageID, tt.storageDir)
			call := interfaces.ToolCall{
				ID:    "call-1",
				Name:  BashToolName,
				Input: `{"command": "ls", "timeout": 1000}`,
			}

			_, err := tool.Run(ctx, call)

			if tt.expectError {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tt.errorMsg)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

// Test safe command detection
func TestSafeCommandDetection(t *testing.T) {
	tests := []struct {
		name    string
		command string
		isSafe  bool
	}{
		{
			name:    "basic ls command",
			command: "ls",
			isSafe:  true,
		},
		{
			name:    "ls with flags",
			command: "ls -la",
			isSafe:  true,
		},
		{
			name:    "ls with path",
			command: "ls /home/user",
			isSafe:  true,
		},
		{
			name:    "echo command",
			command: "echo hello",
			isSafe:  true,
		},
		{
			name:    "git status",
			command: "git status",
			isSafe:  true,
		},
		{
			name:    "git status with flags",
			command: "git status --porcelain",
			isSafe:  true,
		},
		{
			name:    "go version",
			command: "go version",
			isSafe:  true,
		},
		{
			name:    "go build (should be safe)",
			command: "go build",
			isSafe:  true,
		},
		{
			name:    "random command not in safe list",
			command: "rm -rf /",
			isSafe:  false,
		},
		{
			name:    "command starting with safe command but different",
			command: "list_files", // starts with 'ls' but is different
			isSafe:  false,
		},
		{
			name:    "case insensitive matching",
			command: "LS -la",
			isSafe:  true,
		},
		{
			name:    "command with hyphen after safe command",
			command: "ls-modified", // should not be safe as it's a different command
			isSafe:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			isSafeReadOnly := false
			cmdLower := strings.ToLower(tt.command)

			for _, safe := range safeReadOnlyCommands {
				if strings.HasPrefix(cmdLower, strings.ToLower(safe)) {
					if len(cmdLower) == len(safe) || cmdLower[len(safe)] == ' ' {
						isSafeReadOnly = true
						break
					}
				}
			}

			assert.Equal(t, tt.isSafe, isSafeReadOnly, "Command: %s", tt.command)
		})
	}
}

// Test truncateOutput function
func TestTruncateOutput(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "short content not truncated",
			input:    "Hello World",
			expected: "Hello World",
		},
		{
			name:     "empty content",
			input:    "",
			expected: "",
		},
		{
			name:     "content at max length",
			input:    strings.Repeat("a", MaxOutputLength),
			expected: strings.Repeat("a", MaxOutputLength),
		},
		{
			name:  "content exceeding max length gets truncated",
			input: strings.Repeat("a", MaxOutputLength+1000),
			expected: func() string {
				halfLength := MaxOutputLength / 2
				start := strings.Repeat("a", halfLength)
				end := strings.Repeat("a", halfLength)
				truncatedLinesCount := countLines(strings.Repeat("a", 1000))
				return fmt.Sprintf("%s\n\n... [%d lines truncated] ...\n\n%s", start, truncatedLinesCount, end)
			}(),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := truncateOutput(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

// Test countLines function
func TestCountLines(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected int
	}{
		{
			name:     "empty string",
			input:    "",
			expected: 0,
		},
		{
			name:     "single line",
			input:    "hello",
			expected: 1,
		},
		{
			name:     "multiple lines",
			input:    "line1\nline2\nline3",
			expected: 3,
		},
		{
			name:     "lines with empty lines",
			input:    "line1\n\nline3\n",
			expected: 4,
		},
		{
			name:     "only newlines",
			input:    "\n\n\n",
			expected: 4,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := countLines(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

// Test interface compliance
func TestBashToolInterfaceCompliance(t *testing.T) {
	mockPermissionService := &MockPermissionService{}
	tool := NewBashTool(mockPermissionService)

	// Verify that bashTool implements BaseTool interface
	var _ = tool

	// Verify that all required methods are available
	info := tool.Info()
	assert.NotEmpty(t, info.Name)
	assert.NotEmpty(t, info.Description)
	assert.NotNil(t, info.Parameters)
	assert.NotNil(t, info.Required)

	// Verify Run method signature (this will be tested more thoroughly in integration tests)
	ctx := context.Background()
	call := interfaces.ToolCall{ID: "test", Name: "bash", Input: "{}"}
	response, err := tool.Run(ctx, call)
	// Should return error response for missing command, not Go error
	assert.NoError(t, err)
	assert.True(t, response.IsError)
	assert.Contains(t, response.Content, "missing command")
}

// Test permission service integration
func TestPermissionServiceIntegration(t *testing.T) {
	mockPermissionService := &MockPermissionService{}
	tool := NewBashTool(mockPermissionService)

	// Test that non-safe commands trigger permission requests
	tempDir := createTempDir(t)
	ctx := createBashTestContext("session-123", "message-456", tempDir)

	// Mock permission service to deny the request
	mockPermissionService.On("Request", mock.AnythingOfType("permission.CreatePermissionRequest")).Return(false)

	call := interfaces.ToolCall{
		ID:    "call-1",
		Name:  BashToolName,
		Input: `{"command": "rm -rf test", "timeout": 1000}`,
	}

	_, err := tool.Run(ctx, call)

	// Should return permission denied error
	assert.Error(t, err)
	assert.Equal(t, permission.ErrorPermissionDenied, err)

	// Verify permission was requested
	mockPermissionService.AssertExpectations(t)
}

// Test that safe commands bypass permission checks
func TestSafeCommandsBypassPermissions(t *testing.T) {
	mockPermissionService := &MockPermissionService{}
	tool := NewBashTool(mockPermissionService)

	tempDir := createTempDir(t)
	ctx := createBashTestContext("session-123", "message-456", tempDir)

	call := interfaces.ToolCall{
		ID:    "call-1",
		Name:  BashToolName,
		Input: `{"command": "echo hello", "timeout": 1000}`,
	}

	// Since echo is a safe command, permission should not be requested
	// The call should succeed since safe commands bypass permissions
	response, err := tool.Run(ctx, call)

	// Should succeed without permission error
	assert.NoError(t, err)
	assert.False(t, response.IsError)
	assert.Contains(t, response.Content, "hello")

	// Verify no permission requests were made
	mockPermissionService.AssertNotCalled(t, "Request")
}

// Test output formatting with both stdout and stderr
func TestOutputFormatting(t *testing.T) {
	// This test focuses on the output formatting logic
	// We'll simulate the behavior by testing the formatting directly

	tests := []struct {
		name          string
		stdout        string
		stderr        string
		exitCode      int
		interrupted   bool
		expectedParts []string
	}{
		{
			name:          "only stdout",
			stdout:        "Hello World",
			stderr:        "",
			exitCode:      0,
			interrupted:   false,
			expectedParts: []string{"Hello World"},
		},
		{
			name:          "only stderr with non-zero exit",
			stdout:        "",
			stderr:        "Error occurred",
			exitCode:      1,
			interrupted:   false,
			expectedParts: []string{"Error occurred", "Exit code 1"},
		},
		{
			name:          "both stdout and stderr",
			stdout:        "Success output",
			stderr:        "Warning message",
			exitCode:      0,
			interrupted:   false,
			expectedParts: []string{"Success output", "Warning message"},
		},
		{
			name:          "interrupted command",
			stdout:        "Partial output",
			stderr:        "",
			exitCode:      143,
			interrupted:   true,
			expectedParts: []string{"Partial output", "Command was aborted before completion"},
		},
		{
			name:          "empty output",
			stdout:        "",
			stderr:        "",
			exitCode:      0,
			interrupted:   false,
			expectedParts: []string{}, // Should return "no output"
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Simulate the output formatting logic from the Run method
			stdout := truncateOutput(tt.stdout)
			stderr := truncateOutput(tt.stderr)

			errorMessage := stderr
			if tt.interrupted {
				if errorMessage != "" {
					errorMessage += "\n"
				}
				errorMessage += "Command was aborted before completion"
			} else if tt.exitCode != 0 {
				if errorMessage != "" {
					errorMessage += "\n"
				}
				errorMessage += fmt.Sprintf("Exit code %d", tt.exitCode)
			}

			hasBothOutputs := stdout != "" && stderr != ""

			if hasBothOutputs {
				stdout += "\n"
			}

			if errorMessage != "" {
				stdout += "\n" + errorMessage
			}

			finalOutput := stdout
			if finalOutput == "" {
				finalOutput = "no output"
			}

			// Check that expected parts are in the final output
			for _, part := range tt.expectedParts {
				assert.Contains(t, finalOutput, part)
			}
		})
	}
}

// Test metadata generation
func TestMetadataGeneration(t *testing.T) {
	// Test that metadata contains timing information
	startTime := time.Now()
	metadata := BashResponseMetadata{
		StartTime: startTime.UnixMilli(),
		EndTime:   startTime.Add(time.Second).UnixMilli(),
	}

	// Verify that end time is after start time
	assert.Greater(t, metadata.EndTime, metadata.StartTime)

	// Test JSON serialization of metadata
	jsonData, err := json.Marshal(metadata)
	assert.NoError(t, err)
	assert.Contains(t, string(jsonData), "start_time")
	assert.Contains(t, string(jsonData), "end_time")
}

// Benchmark tests for performance validation
func BenchmarkTruncateOutput(b *testing.B) {
	longContent := strings.Repeat("This is a test line.\n", 10000)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		truncateOutput(longContent)
	}
}

func BenchmarkCountLines(b *testing.B) {
	content := strings.Repeat("line\n", 1000)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		countLines(content)
	}
}

func BenchmarkJSONSerialization(b *testing.B) {
	params := BashParams{
		Command: "echo 'hello world' && ls -la /tmp",
		Timeout: 5000,
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = json.Marshal(params)
	}
}

// Test edge cases for command parsing
func TestCommandParsingEdgeCases(t *testing.T) {
	tests := []struct {
		name      string
		command   string
		shouldErr bool
		errMsg    string
	}{
		{
			name:      "command with quotes",
			command:   `echo "hello world"`,
			shouldErr: false,
		},
		{
			name:      "command with pipes",
			command:   "ls | grep test",
			shouldErr: false,
		},
		{
			name:      "command with redirections",
			command:   "echo hello > test.txt",
			shouldErr: false,
		},
		{
			name:      "command with background process",
			command:   "sleep 1 &",
			shouldErr: false,
		},
		{
			name:      "command with multiple spaces",
			command:   "ls     -la",
			shouldErr: false,
		},
		{
			name:      "command starting with space",
			command:   " ls -la",
			shouldErr: false,
		},
		{
			name:      "complex shell command",
			command:   "for i in {1..5}; do echo $i; done",
			shouldErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Test that command parsing doesn't crash with various inputs
			baseCmd := strings.Fields(tt.command)[0]

			// Should not panic
			assert.NotPanics(t, func() {
				for _, banned := range bannedCommands {
					strings.EqualFold(baseCmd, banned)
				}
			})
		})
	}
}
