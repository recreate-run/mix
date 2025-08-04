package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"time"

	"mix/internal/permission"
)

type PythonExecutionParams struct {
	Code string `json:"code"`
}

type PythonExecutionResult struct {
	Type       string `json:"type"`
	Stdout     string `json:"stdout"`
	Stderr     string `json:"stderr"`
	ReturnCode int    `json:"return_code"`
}

type pythonExecutionTool struct {
	permissions permission.Service
}

const (
	PythonExecutionToolName = "python_execution"
	PythonDefaultTimeout    = 30 * 1000  // 30 seconds in milliseconds
	PythonMaxTimeout        = 120 * 1000 // 2 minutes in milliseconds
	PythonMaxOutputLength   = 30000
)

func pythonExecutionDescription() string {
	return LoadToolDescription("python_execution")
}

func NewPythonExecutionTool(permission permission.Service) BaseTool {
	return &pythonExecutionTool{
		permissions: permission,
	}
}

func (p *pythonExecutionTool) Info() ToolInfo {
	return ToolInfo{
		Name:        PythonExecutionToolName,
		Description: pythonExecutionDescription(),
		Parameters: map[string]any{
			"code": map[string]any{
				"type":        "string",
				"description": "The Python code to execute",
			},
		},
		Required: []string{"code"},
	}
}

func (p *pythonExecutionTool) Run(ctx context.Context, call ToolCall) (ToolResponse, error) {
	var params PythonExecutionParams
	if err := json.Unmarshal([]byte(call.Input), &params); err != nil {
		return NewTextErrorResponse("invalid parameters"), nil
	}

	if params.Code == "" {
		return NewTextErrorResponse("missing code parameter"), nil
	}

	if strings.Contains(params.Code, "import subprocess") ||
		strings.Contains(params.Code, "import os") ||
		strings.Contains(params.Code, "exec(") ||
		strings.Contains(params.Code, "eval(") ||
		strings.Contains(params.Code, "__import__") {
		return NewTextErrorResponse("potentially unsafe code detected"), nil
	}

	sessionID, messageID := GetContextValues(ctx)
	if sessionID == "" || messageID == "" {
		return ToolResponse{}, fmt.Errorf("session ID and message ID are required for Python execution")
	}

	p.permissions.Request(
		permission.CreatePermissionRequest{
			SessionID:   sessionID,
			ToolName:    PythonExecutionToolName,
			Action:      "execute",
			Description: "Execute Python code in isolated environment",
			Params:      params,
		},
	)

	result, err := p.executePythonCode(ctx, params.Code)
	if err != nil {
		return NewTextErrorResponse(fmt.Sprintf("execution failed: %v", err)), nil
	}

	resultJSON, err := json.Marshal(result)
	if err != nil {
		return NewTextErrorResponse("failed to format result"), nil
	}

	return NewTextResponse(string(resultJSON)), nil
}

func (p *pythonExecutionTool) executePythonCode(ctx context.Context, code string) (*PythonExecutionResult, error) {
	timeoutCtx, cancel := context.WithTimeout(ctx, time.Duration(PythonDefaultTimeout)*time.Millisecond)
	defer cancel()

	tempDir, err := os.MkdirTemp("", "python_exec_*")
	if err != nil {
		return nil, fmt.Errorf("failed to create temp directory: %w", err)
	}
	defer os.RemoveAll(tempDir)

	// Create temporary Python file
	tempFile, err := os.CreateTemp(tempDir, "script_*.py")
	if err != nil {
		return nil, fmt.Errorf("failed to create temp script file: %w", err)
	}
	defer tempFile.Close()

	// Write the code to the temporary file
	if _, err := tempFile.WriteString(code); err != nil {
		return nil, fmt.Errorf("failed to write code to temp file: %w", err)
	}

	cmd := exec.CommandContext(timeoutCtx, "uv", "run", "--isolated", "--with", "numpy", tempFile.Name())
	cmd.Dir = tempDir

	var stdout, stderr strings.Builder
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err = cmd.Run()
	
	returnCode := 0
	if err != nil {
		if exitError, ok := err.(*exec.ExitError); ok {
			returnCode = exitError.ExitCode()
		} else {
			returnCode = 1
		}
	}

	stdoutStr := truncateOutput(stdout.String())
	stderrStr := truncateOutput(stderr.String())

	return &PythonExecutionResult{
		Type:       "code_execution_result",
		Stdout:     stdoutStr,
		Stderr:     stderrStr,
		ReturnCode: returnCode,
	}, nil
}