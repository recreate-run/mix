package tools

import (
	"context"
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"mix/internal/permission"
	"mix/internal/pubsub"
)

// Mock permission service for testing
type pythonMockPermissionService struct {
	mock.Mock
}

func (m *pythonMockPermissionService) Subscribe(ctx context.Context) <-chan pubsub.Event[permission.PermissionRequest] {
	args := m.Called(ctx)
	return args.Get(0).(<-chan pubsub.Event[permission.PermissionRequest])
}

func (m *pythonMockPermissionService) GrantPersistant(permission permission.PermissionRequest) {
	m.Called(permission)
}

func (m *pythonMockPermissionService) Grant(permission permission.PermissionRequest) {
	m.Called(permission)
}

func (m *pythonMockPermissionService) Deny(permission permission.PermissionRequest) {
	m.Called(permission)
}

func (m *pythonMockPermissionService) Request(opts permission.CreatePermissionRequest) bool {
	args := m.Called(opts)
	return args.Bool(0)
}

// Helper function to create context with session and message IDs
func createPythonTestContext(sessionID, messageID string) context.Context {
	ctx := context.Background()
	if sessionID != "" {
		ctx = context.WithValue(ctx, SessionIDContextKey, sessionID)
	}
	if messageID != "" {
		ctx = context.WithValue(ctx, MessageIDContextKey, messageID)
	}
	return ctx
}


// Test PythonExecutionParams struct
func TestPythonExecutionParams(t *testing.T) {
	t.Run("JSON serialization and deserialization", func(t *testing.T) {
		params := PythonExecutionParams{
			Code: "print('hello world')",
		}

		// Test marshaling
		data, err := json.Marshal(params)
		require.NoError(t, err)
		assert.Contains(t, string(data), "print('hello world')")

		// Test unmarshaling
		var unmarshaled PythonExecutionParams
		err = json.Unmarshal(data, &unmarshaled)
		require.NoError(t, err)
		assert.Equal(t, params.Code, unmarshaled.Code)
	})

	t.Run("JSON with empty code", func(t *testing.T) {
		params := PythonExecutionParams{Code: ""}
		data, err := json.Marshal(params)
		require.NoError(t, err)

		var unmarshaled PythonExecutionParams
		err = json.Unmarshal(data, &unmarshaled)
		require.NoError(t, err)
		assert.Equal(t, "", unmarshaled.Code)
	})

	t.Run("JSON with special characters", func(t *testing.T) {
		params := PythonExecutionParams{
			Code: "print('hello \"world\" with \\n newlines')",
		}
		data, err := json.Marshal(params)
		require.NoError(t, err)

		var unmarshaled PythonExecutionParams
		err = json.Unmarshal(data, &unmarshaled)
		require.NoError(t, err)
		assert.Equal(t, params.Code, unmarshaled.Code)
	})
}

// Test PythonExecutionResult struct
func TestPythonExecutionResult(t *testing.T) {
	t.Run("JSON serialization and deserialization", func(t *testing.T) {
		result := PythonExecutionResult{
			Type:       "code_execution_result",
			Stdout:     "hello world\n",
			Stderr:     "",
			ReturnCode: 0,
		}

		// Test marshaling
		data, err := json.Marshal(result)
		require.NoError(t, err)
		assert.Contains(t, string(data), "code_execution_result")
		assert.Contains(t, string(data), "hello world")

		// Test unmarshaling
		var unmarshaled PythonExecutionResult
		err = json.Unmarshal(data, &unmarshaled)
		require.NoError(t, err)
		assert.Equal(t, result.Type, unmarshaled.Type)
		assert.Equal(t, result.Stdout, unmarshaled.Stdout)
		assert.Equal(t, result.Stderr, unmarshaled.Stderr)
		assert.Equal(t, result.ReturnCode, unmarshaled.ReturnCode)
	})

	t.Run("JSON with error result", func(t *testing.T) {
		result := PythonExecutionResult{
			Type:       "code_execution_result",
			Stdout:     "",
			Stderr:     "SyntaxError: invalid syntax",
			ReturnCode: 1,
		}

		data, err := json.Marshal(result)
		require.NoError(t, err)

		var unmarshaled PythonExecutionResult
		err = json.Unmarshal(data, &unmarshaled)
		require.NoError(t, err)
		assert.Equal(t, result.Type, unmarshaled.Type)
		assert.Equal(t, result.Stdout, unmarshaled.Stdout)
		assert.Equal(t, result.Stderr, unmarshaled.Stderr)
		assert.Equal(t, result.ReturnCode, unmarshaled.ReturnCode)
	})
}

// Test NewPythonExecutionTool constructor
func TestNewPythonExecutionTool(t *testing.T) {
	t.Run("creates tool with valid permission service", func(t *testing.T) {
		mockPermission := &pythonMockPermissionService{}
		tool := NewPythonExecutionTool(mockPermission)

		assert.NotNil(t, tool)
		assert.Implements(t, (*BaseTool)(nil), tool)

		// Verify it's the correct type
		pythonTool, ok := tool.(*pythonExecutionTool)
		assert.True(t, ok)
		assert.Equal(t, mockPermission, pythonTool.permissions)
	})

	t.Run("creates tool with nil permission service", func(t *testing.T) {
		tool := NewPythonExecutionTool(nil)

		assert.NotNil(t, tool)
		pythonTool, ok := tool.(*pythonExecutionTool)
		assert.True(t, ok)
		assert.Nil(t, pythonTool.permissions)
	})
}

// Test Info method
func TestPythonExecutionTool_Info(t *testing.T) {
	mockPermission := &pythonMockPermissionService{}
	tool := NewPythonExecutionTool(mockPermission)

	info := tool.Info()

	assert.Equal(t, PythonExecutionToolName, info.Name)
	assert.NotEmpty(t, info.Description)
	assert.NotNil(t, info.Parameters)
	assert.Contains(t, info.Required, "code")

	// Check parameters structure
	params, ok := info.Parameters["code"]
	assert.True(t, ok)
	paramMap, ok := params.(map[string]any)
	assert.True(t, ok)
	assert.Equal(t, "string", paramMap["type"])
	assert.NotEmpty(t, paramMap["description"])
}

// Test Run method
func TestPythonExecutionTool_Run(t *testing.T) {
	t.Run("invalid JSON parameters", func(t *testing.T) {
		mockPermission := &pythonMockPermissionService{}
		tool := NewPythonExecutionTool(mockPermission)
		ctx := createPythonTestContext("session-123", "message-456")

		call := ToolCall{
			ID:    "call-1",
			Name:  PythonExecutionToolName,
			Input: "invalid json",
		}

		response, err := tool.Run(ctx, call)
		require.NoError(t, err)
		assert.Equal(t, "text", string(response.Type))
		assert.True(t, response.IsError)
		assert.Contains(t, response.Content, "invalid parameters")
	})

	t.Run("empty code parameter", func(t *testing.T) {
		mockPermission := &pythonMockPermissionService{}
		tool := NewPythonExecutionTool(mockPermission)
		ctx := createPythonTestContext("session-123", "message-456")

		params := PythonExecutionParams{Code: ""}
		inputJSON, _ := json.Marshal(params)

		call := ToolCall{
			ID:    "call-1",
			Name:  PythonExecutionToolName,
			Input: string(inputJSON),
		}

		response, err := tool.Run(ctx, call)
		require.NoError(t, err)
		assert.Equal(t, "text", string(response.Type))
		assert.True(t, response.IsError)
		assert.Contains(t, response.Content, "missing code parameter")
	})

	t.Run("unsafe code detection - subprocess", func(t *testing.T) {
		mockPermission := &pythonMockPermissionService{}
		tool := NewPythonExecutionTool(mockPermission)
		ctx := createPythonTestContext("session-123", "message-456")

		params := PythonExecutionParams{Code: "import subprocess\nsubprocess.run(['ls'])"}
		inputJSON, _ := json.Marshal(params)

		call := ToolCall{
			ID:    "call-1",
			Name:  PythonExecutionToolName,
			Input: string(inputJSON),
		}

		response, err := tool.Run(ctx, call)
		require.NoError(t, err)
		assert.Equal(t, "text", string(response.Type))
		assert.True(t, response.IsError)
		assert.Contains(t, response.Content, "potentially unsafe code detected")
	})

	t.Run("unsafe code detection - os import", func(t *testing.T) {
		mockPermission := &pythonMockPermissionService{}
		tool := NewPythonExecutionTool(mockPermission)
		ctx := createPythonTestContext("session-123", "message-456")

		params := PythonExecutionParams{Code: "import os\nos.system('rm -rf /')"}
		inputJSON, _ := json.Marshal(params)

		call := ToolCall{
			ID:    "call-1",
			Name:  PythonExecutionToolName,
			Input: string(inputJSON),
		}

		response, err := tool.Run(ctx, call)
		require.NoError(t, err)
		assert.Equal(t, "text", string(response.Type))
		assert.True(t, response.IsError)
		assert.Contains(t, response.Content, "potentially unsafe code detected")
	})

	t.Run("unsafe code detection - exec", func(t *testing.T) {
		mockPermission := &pythonMockPermissionService{}
		tool := NewPythonExecutionTool(mockPermission)
		ctx := createPythonTestContext("session-123", "message-456")

		params := PythonExecutionParams{Code: "exec('print(\"hello\")')"}
		inputJSON, _ := json.Marshal(params)

		call := ToolCall{
			ID:    "call-1",
			Name:  PythonExecutionToolName,
			Input: string(inputJSON),
		}

		response, err := tool.Run(ctx, call)
		require.NoError(t, err)
		assert.Equal(t, "text", string(response.Type))
		assert.True(t, response.IsError)
		assert.Contains(t, response.Content, "potentially unsafe code detected")
	})

	t.Run("unsafe code detection - eval", func(t *testing.T) {
		mockPermission := &pythonMockPermissionService{}
		tool := NewPythonExecutionTool(mockPermission)
		ctx := createPythonTestContext("session-123", "message-456")

		params := PythonExecutionParams{Code: "result = eval('1 + 1')"}
		inputJSON, _ := json.Marshal(params)

		call := ToolCall{
			ID:    "call-1",
			Name:  PythonExecutionToolName,
			Input: string(inputJSON),
		}

		response, err := tool.Run(ctx, call)
		require.NoError(t, err)
		assert.Equal(t, "text", string(response.Type))
		assert.True(t, response.IsError)
		assert.Contains(t, response.Content, "potentially unsafe code detected")
	})

	t.Run("unsafe code detection - __import__", func(t *testing.T) {
		mockPermission := &pythonMockPermissionService{}
		tool := NewPythonExecutionTool(mockPermission)
		ctx := createPythonTestContext("session-123", "message-456")

		params := PythonExecutionParams{Code: "module = __import__('os')"}
		inputJSON, _ := json.Marshal(params)

		call := ToolCall{
			ID:    "call-1",
			Name:  PythonExecutionToolName,
			Input: string(inputJSON),
		}

		response, err := tool.Run(ctx, call)
		require.NoError(t, err)
		assert.Equal(t, "text", string(response.Type))
		assert.True(t, response.IsError)
		assert.Contains(t, response.Content, "potentially unsafe code detected")
	})

	t.Run("missing session ID", func(t *testing.T) {
		mockPermission := &pythonMockPermissionService{}
		tool := NewPythonExecutionTool(mockPermission)
		ctx := createPythonTestContext("", "message-456") // Empty session ID

		params := PythonExecutionParams{Code: "print('hello')"}
		inputJSON, _ := json.Marshal(params)

		call := ToolCall{
			ID:    "call-1",
			Name:  PythonExecutionToolName,
			Input: string(inputJSON),
		}

		_, err := tool.Run(ctx, call)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "session ID and message ID are required")
	})

	t.Run("missing message ID", func(t *testing.T) {
		mockPermission := &pythonMockPermissionService{}
		tool := NewPythonExecutionTool(mockPermission)
		ctx := createPythonTestContext("session-123", "") // Empty message ID

		params := PythonExecutionParams{Code: "print('hello')"}
		inputJSON, _ := json.Marshal(params)

		call := ToolCall{
			ID:    "call-1",
			Name:  PythonExecutionToolName,
			Input: string(inputJSON),
		}

		_, err := tool.Run(ctx, call)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "session ID and message ID are required")
	})

	t.Run("valid simple code execution", func(t *testing.T) {
		mockPermission := &pythonMockPermissionService{}
		tool := NewPythonExecutionTool(mockPermission)
		ctx := createPythonTestContext("session-123", "message-456")

		params := PythonExecutionParams{Code: "print('hello world')"}
		inputJSON, _ := json.Marshal(params)

		call := ToolCall{
			ID:    "call-1",
			Name:  PythonExecutionToolName,
			Input: string(inputJSON),
		}

		response, err := tool.Run(ctx, call)
		require.NoError(t, err)
		assert.Equal(t, "text", string(response.Type))
		assert.False(t, response.IsError)

		// Parse the JSON response
		var result PythonExecutionResult
		err = json.Unmarshal([]byte(response.Content), &result)
		require.NoError(t, err)
		assert.Equal(t, "code_execution_result", result.Type)
		assert.Contains(t, result.Stdout, "hello world")
		assert.Equal(t, 0, result.ReturnCode)
	})

	t.Run("valid code with syntax error", func(t *testing.T) {
		mockPermission := &pythonMockPermissionService{}
		tool := NewPythonExecutionTool(mockPermission)
		ctx := createPythonTestContext("session-123", "message-456")

		params := PythonExecutionParams{Code: "print('hello world'"}  // Missing closing quote
		inputJSON, _ := json.Marshal(params)

		call := ToolCall{
			ID:    "call-1",
			Name:  PythonExecutionToolName,
			Input: string(inputJSON),
		}

		response, err := tool.Run(ctx, call)
		require.NoError(t, err)
		assert.Equal(t, "text", string(response.Type))
		assert.False(t, response.IsError)

		// Parse the JSON response
		var result PythonExecutionResult
		err = json.Unmarshal([]byte(response.Content), &result)
		require.NoError(t, err)
		assert.Equal(t, "code_execution_result", result.Type)
		assert.NotEqual(t, 0, result.ReturnCode) // Should have non-zero exit code
		assert.NotEmpty(t, result.Stderr)        // Should have error output
	})

	t.Run("valid code with numpy import", func(t *testing.T) {
		mockPermission := &pythonMockPermissionService{}
		tool := NewPythonExecutionTool(mockPermission)
		ctx := createPythonTestContext("session-123", "message-456")

		params := PythonExecutionParams{Code: "import numpy as np\nprint(np.array([1, 2, 3]))"}
		inputJSON, _ := json.Marshal(params)

		call := ToolCall{
			ID:    "call-1",
			Name:  PythonExecutionToolName,
			Input: string(inputJSON),
		}

		response, err := tool.Run(ctx, call)
		require.NoError(t, err)
		assert.Equal(t, "text", string(response.Type))
		assert.False(t, response.IsError)

		// Parse the JSON response
		var result PythonExecutionResult
		err = json.Unmarshal([]byte(response.Content), &result)
		require.NoError(t, err)
		assert.Equal(t, "code_execution_result", result.Type)
		assert.Contains(t, result.Stdout, "[1 2 3]")
		assert.Equal(t, 0, result.ReturnCode)
	})
}

// Test executePythonCode method directly
func TestPythonExecutionTool_ExecutePythonCode(t *testing.T) {
	t.Run("simple print statement", func(t *testing.T) {
		mockPermission := &pythonMockPermissionService{}
		tool := &pythonExecutionTool{permissions: mockPermission}
		ctx := context.Background()

		result, err := tool.executePythonCode(ctx, "print('hello world')")
		require.NoError(t, err)
		assert.Equal(t, "code_execution_result", result.Type)
		assert.Contains(t, result.Stdout, "hello world")
		assert.Empty(t, result.Stderr)
		assert.Equal(t, 0, result.ReturnCode)
	})

	t.Run("code with error", func(t *testing.T) {
		mockPermission := &pythonMockPermissionService{}
		tool := &pythonExecutionTool{permissions: mockPermission}
		ctx := context.Background()

		result, err := tool.executePythonCode(ctx, "undefined_variable")
		require.NoError(t, err)
		assert.Equal(t, "code_execution_result", result.Type)
		assert.Empty(t, result.Stdout)
		assert.NotEmpty(t, result.Stderr)
		assert.NotEqual(t, 0, result.ReturnCode)
	})

	t.Run("code with timeout context", func(t *testing.T) {
		mockPermission := &pythonMockPermissionService{}
		tool := &pythonExecutionTool{permissions: mockPermission}

		// Create a context with a very short timeout
		ctx, cancel := context.WithTimeout(context.Background(), 1*time.Millisecond)
		defer cancel()

		result, err := tool.executePythonCode(ctx, "import time; time.sleep(1)")

		// The execution should either succeed quickly or be cancelled due to timeout
		// We can't guarantee the exact behavior due to timing, so we just check it doesn't panic
		if err != nil {
			assert.Contains(t, err.Error(), "context")
		} else {
			assert.NotNil(t, result)
		}
	})

	t.Run("code with large output", func(t *testing.T) {
		mockPermission := &pythonMockPermissionService{}
		tool := &pythonExecutionTool{permissions: mockPermission}
		ctx := context.Background()

		// Generate output larger than PythonMaxOutputLength
		largeCode := "for i in range(1000):\n    print('x' * 100)"

		result, err := tool.executePythonCode(ctx, largeCode)
		require.NoError(t, err)
		assert.Equal(t, "code_execution_result", result.Type)

		// Output should be truncated to PythonMaxOutputLength
		assert.True(t, len(result.Stdout) <= PythonMaxOutputLength)
		assert.Equal(t, 0, result.ReturnCode)
	})

	t.Run("mathematical computation", func(t *testing.T) {
		mockPermission := &pythonMockPermissionService{}
		tool := &pythonExecutionTool{permissions: mockPermission}
		ctx := context.Background()

		code := `
result = 2 + 2
print(f"2 + 2 = {result}")
print(f"10 ** 2 = {10 ** 2}")
`

		result, err := tool.executePythonCode(ctx, code)
		require.NoError(t, err)
		assert.Equal(t, "code_execution_result", result.Type)
		assert.Contains(t, result.Stdout, "2 + 2 = 4")
		assert.Contains(t, result.Stdout, "10 ** 2 = 100")
		assert.Equal(t, 0, result.ReturnCode)
	})
}

// Test constants
func TestPythonConstants(t *testing.T) {
	assert.Equal(t, "python_execution", PythonExecutionToolName)
	assert.Equal(t, 30*1000, PythonDefaultTimeout)
	assert.Equal(t, 120*1000, PythonMaxTimeout)
	assert.Equal(t, 30000, PythonMaxOutputLength)
}

// Test pythonExecutionDescription function
func TestPythonExecutionDescription(t *testing.T) {
	description := pythonExecutionDescription()
	assert.NotEmpty(t, description)
	// The description comes from LoadToolDescription, so we just verify it's not empty
	// and doesn't contain error text
	assert.NotContains(t, description, "Error:")
}

// Test interface compliance
func TestPythonExecutionTool_InterfaceCompliance(t *testing.T) {
	mockPermission := &pythonMockPermissionService{}
	tool := NewPythonExecutionTool(mockPermission)

	// Verify it implements BaseTool interface
	assert.Implements(t, (*BaseTool)(nil), tool)

	// Test that all required methods exist and work
	info := tool.Info()
	assert.NotEmpty(t, info.Name)

	ctx := createPythonTestContext("session-123", "message-456")
	params := PythonExecutionParams{Code: "print('interface test')"}
	inputJSON, _ := json.Marshal(params)

	call := ToolCall{
		ID:    "interface-test",
		Name:  PythonExecutionToolName,
		Input: string(inputJSON),
	}

	response, err := tool.Run(ctx, call)
	require.NoError(t, err)
	assert.NotEmpty(t, response.Content)
}

// Test edge cases and boundary conditions
func TestPythonExecutionTool_EdgeCases(t *testing.T) {
	t.Run("empty context", func(t *testing.T) {
		mockPermission := &pythonMockPermissionService{}
		tool := NewPythonExecutionTool(mockPermission)

		params := PythonExecutionParams{Code: "print('hello')"}
		inputJSON, _ := json.Marshal(params)

		call := ToolCall{
			ID:    "call-1",
			Name:  PythonExecutionToolName,
			Input: string(inputJSON),
		}

		_, err := tool.Run(context.Background(), call)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "session ID and message ID are required")
	})

	t.Run("very long code", func(t *testing.T) {
		mockPermission := &pythonMockPermissionService{}
		tool := NewPythonExecutionTool(mockPermission)
		ctx := createPythonTestContext("session-123", "message-456")

		// Create very long code string
		longCode := strings.Repeat("# This is a comment\n", 1000) + "print('done')"

		params := PythonExecutionParams{Code: longCode}
		inputJSON, _ := json.Marshal(params)

		call := ToolCall{
			ID:    "call-1",
			Name:  PythonExecutionToolName,
			Input: string(inputJSON),
		}

		response, err := tool.Run(ctx, call)
		require.NoError(t, err)
		assert.Equal(t, "text", string(response.Type))
		assert.False(t, response.IsError)

		// Parse the response
		var result PythonExecutionResult
		err = json.Unmarshal([]byte(response.Content), &result)
		require.NoError(t, err)
		assert.Contains(t, result.Stdout, "done")
	})

	t.Run("code with unicode characters", func(t *testing.T) {
		mockPermission := &pythonMockPermissionService{}
		tool := NewPythonExecutionTool(mockPermission)
		ctx := createPythonTestContext("session-123", "message-456")

		params := PythonExecutionParams{Code: "print('Hello 世界! 🐍')"}
		inputJSON, _ := json.Marshal(params)

		call := ToolCall{
			ID:    "call-1",
			Name:  PythonExecutionToolName,
			Input: string(inputJSON),
		}

		response, err := tool.Run(ctx, call)
		require.NoError(t, err)
		assert.Equal(t, "text", string(response.Type))
		assert.False(t, response.IsError)

		var result PythonExecutionResult
		err = json.Unmarshal([]byte(response.Content), &result)
		require.NoError(t, err)
		assert.Contains(t, result.Stdout, "Hello 世界! 🐍")
	})

	t.Run("multiple statements", func(t *testing.T) {
		mockPermission := &pythonMockPermissionService{}
		tool := NewPythonExecutionTool(mockPermission)
		ctx := createPythonTestContext("session-123", "message-456")

		code := `
x = 5
y = 10
print(f"x = {x}")
print(f"y = {y}")
print(f"x + y = {x + y}")
`

		params := PythonExecutionParams{Code: code}
		inputJSON, _ := json.Marshal(params)

		call := ToolCall{
			ID:    "call-1",
			Name:  PythonExecutionToolName,
			Input: string(inputJSON),
		}

		response, err := tool.Run(ctx, call)
		require.NoError(t, err)
		assert.Equal(t, "text", string(response.Type))
		assert.False(t, response.IsError)

		var result PythonExecutionResult
		err = json.Unmarshal([]byte(response.Content), &result)
		require.NoError(t, err)
		assert.Contains(t, result.Stdout, "x = 5")
		assert.Contains(t, result.Stdout, "y = 10")
		assert.Contains(t, result.Stdout, "x + y = 15")
	})
}

// Test malformed JSON handling
func TestPythonExecutionTool_MalformedJSON(t *testing.T) {
	testCases := []struct {
		name  string
		input string
	}{
		{"completely invalid", "not json at all"},
		{"incomplete object", `{"code":`},
		{"wrong field type", `{"code": 123}`},
		{"missing quotes", `{code: "print('hello')"}`},
		{"extra comma", `{"code": "print('hello')",}`},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			mockPermission := &pythonMockPermissionService{}
			tool := NewPythonExecutionTool(mockPermission)
			ctx := createPythonTestContext("session-123", "message-456")

			call := ToolCall{
				ID:    "call-1",
				Name:  PythonExecutionToolName,
				Input: tc.input,
			}

			response, err := tool.Run(ctx, call)
			require.NoError(t, err)
			assert.Equal(t, "text", string(response.Type))
			assert.True(t, response.IsError)
			assert.Contains(t, response.Content, "invalid parameters")
		})
	}
}

// Benchmark tests
func BenchmarkPythonExecutionTool_SimpleCode(b *testing.B) {
	mockPermission := &pythonMockPermissionService{}
	tool := NewPythonExecutionTool(mockPermission)
	ctx := createPythonTestContext("session-123", "message-456")

	params := PythonExecutionParams{Code: "print('benchmark test')"}
	inputJSON, _ := json.Marshal(params)

	call := ToolCall{
		ID:    "benchmark-call",
		Name:  PythonExecutionToolName,
		Input: string(inputJSON),
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := tool.Run(ctx, call)
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkPythonExecutionTool_JSONSerialization(b *testing.B) {
	result := PythonExecutionResult{
		Type:       "code_execution_result",
		Stdout:     "Hello world\n",
		Stderr:     "",
		ReturnCode: 0,
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := json.Marshal(result)
		if err != nil {
			b.Fatal(err)
		}
	}
}