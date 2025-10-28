package tools

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Test PythonExecutionParams struct standalone
func TestPythonExecutionParams_Standalone(t *testing.T) {
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
}

// Test PythonExecutionResult struct standalone
func TestPythonExecutionResult_Standalone(t *testing.T) {
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
}

// Test NewPythonExecutionTool constructor standalone
func TestNewPythonExecutionTool_Standalone(t *testing.T) {
	t.Run("creates tool with nil permission service", func(t *testing.T) {
		tool := NewPythonExecutionTool(nil)

		assert.NotNil(t, tool)
		pythonTool, ok := tool.(*pythonExecutionTool)
		assert.True(t, ok)
		assert.Nil(t, pythonTool.permissions)
	})
}

// Test Info method standalone
func TestPythonExecutionTool_Info_Standalone(t *testing.T) {
	tool := NewPythonExecutionTool(nil)

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

// Test constants standalone
func TestPythonConstants_Standalone(t *testing.T) {
	assert.Equal(t, "python_execution", PythonExecutionToolName)
	assert.Equal(t, 30*1000, PythonDefaultTimeout)
	assert.Equal(t, 120*1000, PythonMaxTimeout)
	assert.Equal(t, 30000, PythonMaxOutputLength)
}

// Test pythonExecutionDescription function standalone
func TestPythonExecutionDescription_Standalone(t *testing.T) {
	description := pythonExecutionDescription()
	assert.NotEmpty(t, description)
	// The description comes from LoadToolDescription, so we just verify it's not empty
	// and doesn't contain error text
	assert.NotContains(t, description, "Error:")
}

// Test interface compliance standalone
func TestPythonExecutionTool_InterfaceCompliance_Standalone(t *testing.T) {
	tool := NewPythonExecutionTool(nil)

	// Verify it implements BaseTool interface
	assert.Implements(t, (*BaseTool)(nil), tool)

	// Test that all required methods exist and work
	info := tool.Info()
	assert.NotEmpty(t, info.Name)
}

// Test Run method basic validation without execution
func TestPythonExecutionTool_Run_BasicValidation(t *testing.T) {
	tool := NewPythonExecutionTool(nil)

	t.Run("invalid JSON parameters", func(t *testing.T) {
		ctx := context.Background()
		ctx = context.WithValue(ctx, SessionIDContextKey, "session-123")
		ctx = context.WithValue(ctx, MessageIDContextKey, "message-456")

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
		ctx := context.Background()
		ctx = context.WithValue(ctx, SessionIDContextKey, "session-123")
		ctx = context.WithValue(ctx, MessageIDContextKey, "message-456")

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

	t.Run("unsafe code detection", func(t *testing.T) {
		ctx := context.Background()
		ctx = context.WithValue(ctx, SessionIDContextKey, "session-123")
		ctx = context.WithValue(ctx, MessageIDContextKey, "message-456")

		unsafeCodes := []string{
			"import subprocess\nsubprocess.run(['ls'])",
			"import os\nos.system('rm -rf /')",
			"exec('print(\"hello\")')",
			"result = eval('1 + 1')",
			"module = __import__('os')",
		}

		for _, unsafeCode := range unsafeCodes {
			params := PythonExecutionParams{Code: unsafeCode}
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
		}
	})

	t.Run("missing session ID or message ID", func(t *testing.T) {
		params := PythonExecutionParams{Code: "print('hello')"}
		inputJSON, _ := json.Marshal(params)

		call := ToolCall{
			ID:    "call-1",
			Name:  PythonExecutionToolName,
			Input: string(inputJSON),
		}

		// Test with empty context
		_, err := tool.Run(context.Background(), call)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "session ID and message ID are required")

		// Test with only session ID
		ctx := context.WithValue(context.Background(), SessionIDContextKey, "session-123")
		_, err = tool.Run(ctx, call)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "session ID and message ID are required")

		// Test with only message ID
		ctx = context.WithValue(context.Background(), MessageIDContextKey, "message-456")
		_, err = tool.Run(ctx, call)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "session ID and message ID are required")
	})
}
