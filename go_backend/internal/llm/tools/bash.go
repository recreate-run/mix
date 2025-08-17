package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"mix/internal/llm/tools/shell"
	"mix/internal/permission"
)

type BashParams struct {
	Command string `json:"command"`
	Timeout int    `json:"timeout"`
}

type BashPermissionsParams struct {
	Command string `json:"command"`
	Timeout int    `json:"timeout"`
}

type BashResponseMetadata struct {
	StartTime int64 `json:"start_time"`
	EndTime   int64 `json:"end_time"`
}
type bashTool struct {
	permissions permission.Service
}

const (
	BashToolName = "bash"

	DefaultTimeout  = 1 * 60 * 1000  // 1 minutes in milliseconds
	MaxTimeout      = 10 * 60 * 1000 // 10 minutes in milliseconds
	MaxOutputLength = 30000
)

var bannedCommands = []string{
	"alias", "curl", "curlie", "wget", "axel", "aria2c",
	"nc", "telnet", "lynx", "w3m", "links", "httpie", "xh",
	"http-prompt", "chrome", "firefox", "safari",
}

var safeReadOnlyCommands = []string{
	"ls", "echo", "pwd", "date", "cal", "uptime", "whoami", "id", "groups", "env", "printenv", "set", "unset", "which", "type", "whereis",
	"whatis", "uname", "hostname", "df", "du", "free", "top", "ps", "kill", "killall", "nice", "nohup", "time", "timeout",

	"git status", "git log", "git diff", "git show", "git branch", "git tag", "git remote", "git ls-files", "git ls-remote",
	"git rev-parse", "git config --get", "git config --list", "git describe", "git blame", "git grep", "git shortlog",

	"go version", "go help", "go list", "go env", "go doc", "go vet", "go fmt", "go mod", "go test", "go build", "go run", "go install", "go clean",
}

func bashDescription() string {
	return LoadToolDescription("bash")
}

func NewBashTool(permission permission.Service) BaseTool {
	return &bashTool{
		permissions: permission,
	}
}

func (b *bashTool) Info() ToolInfo {
	return ToolInfo{
		Name:        BashToolName,
		Description: bashDescription(),
		Parameters: map[string]any{
			"command": map[string]any{
				"type":        "string",
				"description": "The command to execute",
			},
			"timeout": map[string]any{
				"type":        "number",
				"description": "Optional timeout in milliseconds (max 600000)",
			},
		},
		Required: []string{"command"},
	}
}

func (b *bashTool) Run(ctx context.Context, call ToolCall) (ToolResponse, error) {
	var params BashParams
	if err := json.Unmarshal([]byte(call.Input), &params); err != nil {
		return NewTextErrorResponse("invalid parameters"), nil
	}

	if params.Timeout > MaxTimeout {
		params.Timeout = MaxTimeout
	} else if params.Timeout <= 0 {
		params.Timeout = DefaultTimeout
	}

	if params.Command == "" {
		return NewTextErrorResponse("missing command"), nil
	}

	baseCmd := strings.Fields(params.Command)[0]
	for _, banned := range bannedCommands {
		if strings.EqualFold(baseCmd, banned) {
			return NewTextErrorResponse(fmt.Sprintf("command '%s' is not allowed", baseCmd)), nil
		}
	}

	isSafeReadOnly := false
	cmdLower := strings.ToLower(params.Command)

	for _, safe := range safeReadOnlyCommands {
		if strings.HasPrefix(cmdLower, strings.ToLower(safe)) {
			if len(cmdLower) == len(safe) || cmdLower[len(safe)] == ' ' || cmdLower[len(safe)] == '-' {
				isSafeReadOnly = true
				break
			}
		}
	}

	sessionID, messageID := GetContextValues(ctx)
	if sessionID == "" || messageID == "" {
		return ToolResponse{}, fmt.Errorf("session ID and message ID are required for creating a new file")
	}
	startTime := time.Now()
	workingDir, err := GetWorkingDirectory(ctx)
	if err != nil {
		return ToolResponse{}, fmt.Errorf("failed to get working directory: %w", err)
	}
	
	if !isSafeReadOnly {
		p := b.permissions.Request(
			permission.CreatePermissionRequest{
				SessionID:   sessionID,
				Path:        workingDir,
				ToolName:    BashToolName,
				Action:      fmt.Sprintf("Execute command: %s", params.Command),
				Description: fmt.Sprintf("Execute command: %s", params.Command),
				Params: BashPermissionsParams{
					Command: params.Command,
				},
			},
		)
		if !p {
			return ToolResponse{}, permission.ErrorPermissionDenied
		}
	}
	
	shell := shell.GetPersistentShell(workingDir)
	stdout, stderr, exitCode, interrupted, err := shell.Exec(ctx, params.Command, params.Timeout)
	if err != nil {
		return ToolResponse{}, fmt.Errorf("error executing command: %w", err)
	}

	stdout = truncateOutput(stdout)
	stderr = truncateOutput(stderr)

	errorMessage := stderr
	if interrupted {
		if errorMessage != "" {
			errorMessage += "\n"
		}
		errorMessage += "Command was aborted before completion"
	} else if exitCode != 0 {
		if errorMessage != "" {
			errorMessage += "\n"
		}
		errorMessage += fmt.Sprintf("Exit code %d", exitCode)
	}

	hasBothOutputs := stdout != "" && stderr != ""

	if hasBothOutputs {
		stdout += "\n"
	}

	if errorMessage != "" {
		stdout += "\n" + errorMessage
	}

	metadata := BashResponseMetadata{
		StartTime: startTime.UnixMilli(),
		EndTime:   time.Now().UnixMilli(),
	}
	if stdout == "" {
		return WithResponseMetadata(NewTextResponse("no output"), metadata), nil
	}
	return WithResponseMetadata(NewTextResponse(stdout), metadata), nil
}

func truncateOutput(content string) string {
	if len(content) <= MaxOutputLength {
		return content
	}

	halfLength := MaxOutputLength / 2
	start := content[:halfLength]
	end := content[len(content)-halfLength:]

	truncatedLinesCount := countLines(content[halfLength : len(content)-halfLength])
	return fmt.Sprintf("%s\n\n... [%d lines truncated] ...\n\n%s", start, truncatedLinesCount, end)
}

func countLines(s string) int {
	if s == "" {
		return 0
	}
	return len(strings.Split(s, "\n"))
}
