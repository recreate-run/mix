package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"mix/internal/config"
)

type TodoStatus string
type TodoPriority string

const (
	TodoStatusPending    TodoStatus = "pending"
	TodoStatusInProgress TodoStatus = "in_progress"
	TodoStatusCompleted  TodoStatus = "completed"
)

const (
	TodoPriorityLow    TodoPriority = "low"
	TodoPriorityMedium TodoPriority = "medium"
	TodoPriorityHigh   TodoPriority = "high"
)

type todoWriteTool struct{}

type TodoWriteParams struct {
	Todos []Todo `json:"todos"`
}

type Todo struct {
	ID       string       `json:"id"`
	Content  string       `json:"content"`
	Status   TodoStatus   `json:"status"`
	Priority TodoPriority `json:"priority"`
}

func NewTodoWriteTool() BaseTool {
	return &todoWriteTool{}
}

func (t *todoWriteTool) Info() ToolInfo {
	return ToolInfo{
		Name:        "todo_write",
		Description: LoadToolDescription("todo_write"),
		Parameters: map[string]any{
			"todos": map[string]any{
				"type":        "array",
				"description": "Array of todo items to manage",
				"items": map[string]any{
					"type": "object",
					"properties": map[string]any{
						"id": map[string]any{
							"type":        "string",
							"description": "Unique identifier for the todo item",
						},
						"content": map[string]any{
							"type":        "string",
							"description": "The todo task description",
							"minLength":   1,
						},
						"status": map[string]any{
							"type":        "string",
							"description": "Current status of the todo item",
							"enum":        []string{"pending", "in_progress", "completed"},
						},
						"priority": map[string]any{
							"type":        "string",
							"description": "Priority level of the todo item",
							"enum":        []string{"high", "medium", "low"},
						},
					},
					"required": []string{"id", "content", "status", "priority"},
				},
			},
		},
		Required: []string{"todos"},
	}
}

func (t *todoWriteTool) Run(ctx context.Context, call ToolCall) (ToolResponse, error) {
	var params TodoWriteParams
	if err := json.Unmarshal([]byte(call.Input), &params); err != nil {
		return NewTextErrorResponse(fmt.Sprintf("Invalid parameters: %v", err)), nil
	}

	// Validate todos
	for i, todo := range params.Todos {
		if todo.ID == "" {
			return NewTextErrorResponse(fmt.Sprintf("Todo %d missing ID", i)), nil
		}
		if todo.Content == "" {
			return NewTextErrorResponse(fmt.Sprintf("Todo %d missing content", i)), nil
		}
		if !isValidStatus(todo.Status) {
			return NewTextErrorResponse(fmt.Sprintf("Invalid status '%s' for todo %d", todo.Status, i)), nil
		}
		if !isValidPriority(todo.Priority) {
			return NewTextErrorResponse(fmt.Sprintf("Invalid priority '%s' for todo %d", todo.Priority, i)), nil
		}
	}

	cfg := config.Get()
	todosDir := filepath.Join(cfg.Data.Directory, "todos")
	todosFile := filepath.Join(todosDir, "todos.json")

	if err := os.MkdirAll(todosDir, 0755); err != nil {
		return NewTextErrorResponse(fmt.Sprintf("Failed to create todos directory: %v", err)), nil
	}

	data, err := json.MarshalIndent(params.Todos, "", "  ")
	if err != nil {
		return NewTextErrorResponse(fmt.Sprintf("Failed to marshal todos: %v", err)), nil
	}

	if err := os.WriteFile(todosFile, data, 0644); err != nil {
		return NewTextErrorResponse(fmt.Sprintf("Failed to write todos file: %v", err)), nil
	}

	return ToolResponse{
		Type:    "text",
		Content: fmt.Sprintf("Successfully updated %d todos", len(params.Todos)),
	}, nil
}

func isValidStatus(status TodoStatus) bool {
	return status == TodoStatusPending || status == TodoStatusInProgress || status == TodoStatusCompleted
}

func isValidPriority(priority TodoPriority) bool {
	return priority == TodoPriorityLow || priority == TodoPriorityMedium || priority == TodoPriorityHigh
}
