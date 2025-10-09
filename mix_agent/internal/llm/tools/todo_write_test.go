package tools

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"mix/internal/config"
)

// Test constants
func TestTodoConstants(t *testing.T) {
	// Test TodoStatus constants
	assert.Equal(t, TodoStatus("pending"), TodoStatusPending)
	assert.Equal(t, TodoStatus("in_progress"), TodoStatusInProgress)
	assert.Equal(t, TodoStatus("completed"), TodoStatusCompleted)

	// Test TodoPriority constants
	assert.Equal(t, TodoPriority("low"), TodoPriorityLow)
	assert.Equal(t, TodoPriority("medium"), TodoPriorityMedium)
	assert.Equal(t, TodoPriority("high"), TodoPriorityHigh)
}

// Test Todo struct
func TestTodoStruct(t *testing.T) {
	todo := Todo{
		ID:       "test-id",
		Content:  "Test todo content",
		Status:   TodoStatusPending,
		Priority: TodoPriorityHigh,
	}

	assert.Equal(t, "test-id", todo.ID)
	assert.Equal(t, "Test todo content", todo.Content)
	assert.Equal(t, TodoStatusPending, todo.Status)
	assert.Equal(t, TodoPriorityHigh, todo.Priority)
}

// Test TodoWriteParams struct
func TestTodoWriteParams(t *testing.T) {
	todos := []Todo{
		{ID: "1", Content: "First todo", Status: TodoStatusPending, Priority: TodoPriorityHigh},
		{ID: "2", Content: "Second todo", Status: TodoStatusCompleted, Priority: TodoPriorityLow},
	}

	params := TodoWriteParams{Todos: todos}
	assert.Len(t, params.Todos, 2)
	assert.Equal(t, "1", params.Todos[0].ID)
	assert.Equal(t, "2", params.Todos[1].ID)
}

// Test NewTodoWriteTool constructor
func TestNewTodoWriteTool(t *testing.T) {
	tool := NewTodoWriteTool()
	assert.NotNil(t, tool)
	assert.IsType(t, &todoWriteTool{}, tool)

	// Should implement BaseTool interface
	var _ = tool
}

// Test todoWriteTool Info method
func TestTodoWriteTool_Info(t *testing.T) {
	tool := NewTodoWriteTool()
	info := tool.Info()

	assert.Equal(t, "todo_write", info.Name)
	assert.NotEmpty(t, info.Description)
	assert.Equal(t, []string{"todos"}, info.Required)

	// Verify parameters structure
	assert.Contains(t, info.Parameters, "todos")
	todosParam, ok := info.Parameters["todos"].(map[string]any)
	require.True(t, ok)

	assert.Equal(t, "array", todosParam["type"])
	assert.NotEmpty(t, todosParam["description"])

	// Verify items structure
	items, ok := todosParam["items"].(map[string]any)
	require.True(t, ok)
	assert.Equal(t, "object", items["type"])

	// Verify properties
	properties, ok := items["properties"].(map[string]any)
	require.True(t, ok)
	assert.Contains(t, properties, "id")
	assert.Contains(t, properties, "content")
	assert.Contains(t, properties, "status")
	assert.Contains(t, properties, "priority")

	// Verify status enum
	statusProp, ok := properties["status"].(map[string]any)
	require.True(t, ok)
	statusEnum, ok := statusProp["enum"].([]string)
	require.True(t, ok)
	assert.Contains(t, statusEnum, "pending")
	assert.Contains(t, statusEnum, "in_progress")
	assert.Contains(t, statusEnum, "completed")

	// Verify priority enum
	priorityProp, ok := properties["priority"].(map[string]any)
	require.True(t, ok)
	priorityEnum, ok := priorityProp["enum"].([]string)
	require.True(t, ok)
	assert.Contains(t, priorityEnum, "high")
	assert.Contains(t, priorityEnum, "medium")
	assert.Contains(t, priorityEnum, "low")
}

// Test todoWriteTool Run method with valid input
func TestTodoWriteTool_Run_Success(t *testing.T) {
	// Create temporary directory for test
	tempDir, err := os.MkdirTemp("", "todo_test_*")
	require.NoError(t, err)
	defer func() {
		_ = os.RemoveAll(tempDir)
	}()

	// Mock config to use temp directory
	originalCfg := config.Get()
	// Can't easily restore global config, but test isolation should handle this
	_ = originalCfg

	// Load a test config
	testConfig, err := config.Load(tempDir, false, false)
	require.NoError(t, err)
	testConfig.Data.Directory = tempDir

	tool := NewTodoWriteTool()
	ctx := context.Background()

	tests := []struct {
		name          string
		todos         []Todo
		expectedCount int
	}{
		{
			name: "single todo",
			todos: []Todo{
				{ID: "1", Content: "Test todo", Status: TodoStatusPending, Priority: TodoPriorityHigh},
			},
			expectedCount: 1,
		},
		{
			name: "multiple todos",
			todos: []Todo{
				{ID: "1", Content: "First todo", Status: TodoStatusPending, Priority: TodoPriorityHigh},
				{ID: "2", Content: "Second todo", Status: TodoStatusInProgress, Priority: TodoPriorityMedium},
				{ID: "3", Content: "Third todo", Status: TodoStatusCompleted, Priority: TodoPriorityLow},
			},
			expectedCount: 3,
		},
		{
			name:          "empty todos list",
			todos:         []Todo{},
			expectedCount: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			params := TodoWriteParams{Todos: tt.todos}
			paramsJSON, err := json.Marshal(params)
			require.NoError(t, err)

			call := ToolCall{
				ID:    "test-id",
				Name:  "todo_write",
				Input: string(paramsJSON),
			}

			response, err := tool.Run(ctx, call)
			assert.NoError(t, err)
			assert.False(t, response.IsError)
			assert.Contains(t, response.Content, "Successfully updated")
			assert.Contains(t, response.Content, "todos")

			// Verify file was created and contains expected data
			todosFile := filepath.Join(tempDir, "todos", "todos.json")
			assert.FileExists(t, todosFile)

			fileData, err := os.ReadFile(todosFile)
			assert.NoError(t, err)

			var savedTodos []Todo
			err = json.Unmarshal(fileData, &savedTodos)
			assert.NoError(t, err)
			assert.Len(t, savedTodos, tt.expectedCount)

			if tt.expectedCount > 0 {
				assert.Equal(t, tt.todos, savedTodos)
			}
		})
	}
}

// Test todoWriteTool Run method with invalid JSON
func TestTodoWriteTool_Run_InvalidJSON(t *testing.T) {
	tool := NewTodoWriteTool()
	ctx := context.Background()

	tests := []struct {
		name  string
		input string
	}{
		{"malformed JSON", `{"todos":}`},
		{"invalid JSON structure", `{todos:[]}`},
		{"empty input", ""},
		{"non-JSON input", "not json at all"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			call := ToolCall{
				ID:    "test-id",
				Name:  "todo_write",
				Input: tt.input,
			}

			response, err := tool.Run(ctx, call)
			assert.NoError(t, err)
			assert.True(t, response.IsError)
			assert.Contains(t, response.Content, "Invalid parameters")
		})
	}
}

// Test todoWriteTool Run method with validation errors
func TestTodoWriteTool_Run_ValidationErrors(t *testing.T) {
	tool := NewTodoWriteTool()
	ctx := context.Background()

	tests := []struct {
		name        string
		todos       []Todo
		expectedErr string
	}{
		{
			name: "missing ID",
			todos: []Todo{
				{Content: "Test todo", Status: TodoStatusPending, Priority: TodoPriorityHigh},
			},
			expectedErr: "Todo 0 missing ID",
		},
		{
			name: "missing content",
			todos: []Todo{
				{ID: "1", Status: TodoStatusPending, Priority: TodoPriorityHigh},
			},
			expectedErr: "Todo 0 missing content",
		},
		{
			name: "invalid status",
			todos: []Todo{
				{ID: "1", Content: "Test todo", Status: TodoStatus("invalid"), Priority: TodoPriorityHigh},
			},
			expectedErr: "Invalid status 'invalid' for todo 0",
		},
		{
			name: "invalid priority",
			todos: []Todo{
				{ID: "1", Content: "Test todo", Status: TodoStatusPending, Priority: TodoPriority("invalid")},
			},
			expectedErr: "Invalid priority 'invalid' for todo 0",
		},
		{
			name: "multiple validation errors - first one reported",
			todos: []Todo{
				{Content: "Test todo", Status: TodoStatus("invalid"), Priority: TodoPriority("invalid")},
			},
			expectedErr: "Todo 0 missing ID", // First validation error
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			params := TodoWriteParams{Todos: tt.todos}
			paramsJSON, err := json.Marshal(params)
			require.NoError(t, err)

			call := ToolCall{
				ID:    "test-id",
				Name:  "todo_write",
				Input: string(paramsJSON),
			}

			response, err := tool.Run(ctx, call)
			assert.NoError(t, err)
			assert.True(t, response.IsError)
			assert.Contains(t, response.Content, tt.expectedErr)
		})
	}
}

// Test isValidStatus function
func TestIsValidStatus(t *testing.T) {
	tests := []struct {
		status   TodoStatus
		expected bool
	}{
		{TodoStatusPending, true},
		{TodoStatusInProgress, true},
		{TodoStatusCompleted, true},
		{TodoStatus("invalid"), false},
		{TodoStatus(""), false},
		{TodoStatus("PENDING"), false}, // Case sensitive
	}

	for _, tt := range tests {
		t.Run(string(tt.status), func(t *testing.T) {
			result := isValidStatus(tt.status)
			assert.Equal(t, tt.expected, result)
		})
	}
}

// Test isValidPriority function
func TestIsValidPriority(t *testing.T) {
	tests := []struct {
		priority TodoPriority
		expected bool
	}{
		{TodoPriorityLow, true},
		{TodoPriorityMedium, true},
		{TodoPriorityHigh, true},
		{TodoPriority("invalid"), false},
		{TodoPriority(""), false},
		{TodoPriority("HIGH"), false}, // Case sensitive
	}

	for _, tt := range tests {
		t.Run(string(tt.priority), func(t *testing.T) {
			result := isValidPriority(tt.priority)
			assert.Equal(t, tt.expected, result)
		})
	}
}

// Test JSON marshaling/unmarshaling of Todo structs
func TestTodoJSONSerialization(t *testing.T) {
	originalTodo := Todo{
		ID:       "test-123",
		Content:  "Test todo with special chars: áéíóú",
		Status:   TodoStatusInProgress,
		Priority: TodoPriorityMedium,
	}

	// Marshal to JSON
	jsonData, err := json.Marshal(originalTodo)
	require.NoError(t, err)

	// Unmarshal back
	var deserializedTodo Todo
	err = json.Unmarshal(jsonData, &deserializedTodo)
	require.NoError(t, err)

	// Should be identical
	assert.Equal(t, originalTodo, deserializedTodo)
}

// Test TodoWriteParams JSON serialization
func TestTodoWriteParamsJSONSerialization(t *testing.T) {
	originalParams := TodoWriteParams{
		Todos: []Todo{
			{ID: "1", Content: "First", Status: TodoStatusPending, Priority: TodoPriorityHigh},
			{ID: "2", Content: "Second", Status: TodoStatusCompleted, Priority: TodoPriorityLow},
		},
	}

	// Marshal to JSON
	jsonData, err := json.Marshal(originalParams)
	require.NoError(t, err)

	// Unmarshal back
	var deserializedParams TodoWriteParams
	err = json.Unmarshal(jsonData, &deserializedParams)
	require.NoError(t, err)

	// Should be identical
	assert.Equal(t, originalParams, deserializedParams)
}

// Test edge cases and special scenarios
func TestTodoWriteTool_EdgeCases(t *testing.T) {
	tool := NewTodoWriteTool()
	ctx := context.Background()

	t.Run("todo with special characters", func(t *testing.T) {
		todos := []Todo{
			{
				ID:       "special-123",
				Content:  "Todo with special chars: áéíóú, 中文, 🚀",
				Status:   TodoStatusPending,
				Priority: TodoPriorityMedium,
			},
		}

		params := TodoWriteParams{Todos: todos}
		paramsJSON, err := json.Marshal(params)
		require.NoError(t, err)

		call := ToolCall{
			ID:    "test-id",
			Name:  "todo_write",
			Input: string(paramsJSON),
		}

		response, err := tool.Run(ctx, call)
		assert.NoError(t, err)
		assert.False(t, response.IsError)
	})

	t.Run("todo with very long content", func(t *testing.T) {
		longContent := string(make([]byte, 1000)) // 1000 null bytes
		for i := range longContent {
			longContent = longContent[:i] + "a" + longContent[i+1:]
		}

		todos := []Todo{
			{
				ID:       "long-content",
				Content:  longContent,
				Status:   TodoStatusPending,
				Priority: TodoPriorityLow,
			},
		}

		params := TodoWriteParams{Todos: todos}
		paramsJSON, err := json.Marshal(params)
		require.NoError(t, err)

		call := ToolCall{
			ID:    "test-id",
			Name:  "todo_write",
			Input: string(paramsJSON),
		}

		response, err := tool.Run(ctx, call)
		assert.NoError(t, err)
		assert.False(t, response.IsError)
	})
}

// Test type safety
func TestTodoTypes(t *testing.T) {
	// Test that TodoStatus and TodoPriority are distinct types
	var status TodoStatus = "pending"
	var priority TodoPriority = "high"

	// These should be different types and not directly assignable
	assert.IsType(t, TodoStatus(""), status)
	assert.IsType(t, TodoPriority(""), priority)

	// But underlying values should be comparable
	assert.Equal(t, string(status), "pending")
	assert.Equal(t, string(priority), "high")
}