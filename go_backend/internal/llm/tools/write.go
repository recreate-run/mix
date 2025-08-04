package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"mix/internal/history"
	"mix/internal/logging"
	"mix/internal/permission"
)

type WriteParams struct {
	FilePath string `json:"file_path"`
	Content  string `json:"content"`
}

type WritePermissionsParams struct {
	FilePath string `json:"file_path"`
	Diff     string `json:"diff"`
}

type writeTool struct {
	permissions permission.Service
	files       history.Service
}

type WriteResponseMetadata struct {
	Diff      string `json:"diff"`
	Additions int    `json:"additions"`
	Removals  int    `json:"removals"`
}

const (
	WriteToolName = "write"
)

func NewWriteTool(permissions permission.Service, files history.Service) BaseTool {
	return &writeTool{
		permissions: permissions,
		files:       files,
	}
}

func (w *writeTool) Info() ToolInfo {
	return ToolInfo{
		Name:        WriteToolName,
		Description: LoadToolDescription("write"),
		Parameters: map[string]any{
			"file_path": map[string]any{
				"type":        "string",
				"description": "The path to the file to write",
			},
			"content": map[string]any{
				"type":        "string",
				"description": "The content to write to the file",
			},
		},
		Required: []string{"file_path", "content"},
	}
}

func (w *writeTool) Run(ctx context.Context, call ToolCall) (ToolResponse, error) {
	var params WriteParams
	if err := json.Unmarshal([]byte(call.Input), &params); err != nil {
		return NewTextErrorResponse(fmt.Sprintf("error parsing parameters: %s", err)), nil
	}

	if params.FilePath == "" {
		return NewTextErrorResponse("file_path is required"), nil
	}

	if params.Content == "" {
		return NewTextErrorResponse("content is required"), nil
	}

	workingDir, err := GetWorkingDirectory(ctx)
	if err != nil {
		return ToolResponse{}, fmt.Errorf("failed to get working directory: %w", err)
	}
	filePath := params.FilePath
	if !filepath.IsAbs(filePath) {
		filePath = filepath.Join(workingDir, filePath)
	}

	fileInfo, err := os.Stat(filePath)
	if err == nil {
		if fileInfo.IsDir() {
			return NewTextErrorResponse(fmt.Sprintf("Path is a directory, not a file: %s", filePath)), nil
		}

		modTime := fileInfo.ModTime()
		lastRead := getLastReadTime(filePath)
		if modTime.After(lastRead) {
			return NewTextErrorResponse(fmt.Sprintf("File %s has been modified since it was last read.\nLast modification: %s\nLast read: %s\n\nPlease read the file again before modifying it.",
				filePath, modTime.Format(time.RFC3339), lastRead.Format(time.RFC3339))), nil
		}

		oldContent, readErr := os.ReadFile(filePath)
		if readErr == nil && string(oldContent) == params.Content {
			return NewTextErrorResponse(fmt.Sprintf("File %s already contains the exact content. No changes made.", filePath)), nil
		}
	} else if !os.IsNotExist(err) {
		return ToolResponse{}, fmt.Errorf("error checking file: %w", err)
	}

	dir := filepath.Dir(filePath)
	if err = os.MkdirAll(dir, 0o755); err != nil {
		return ToolResponse{}, fmt.Errorf("error creating directory: %w", err)
	}

	oldContent := ""
	if fileInfo != nil && !fileInfo.IsDir() {
		oldBytes, readErr := os.ReadFile(filePath)
		if readErr == nil {
			oldContent = string(oldBytes)
		}
	}

	sessionID, messageID := GetContextValues(ctx)
	if sessionID == "" || messageID == "" {
		return ToolResponse{}, fmt.Errorf("session_id and message_id are required")
	}

	// Simple diff replacement for content writing
	diffText := fmt.Sprintf("--- %s\n+++ %s\n", filePath, filePath)
	oldLines := strings.Split(oldContent, "\n")
	newLines := strings.Split(params.Content, "\n")
	additions := len(newLines)
	removals := len(oldLines)

	permissionPath := filepath.Dir(filePath)
	if strings.HasPrefix(filePath, workingDir) {
		permissionPath = workingDir
	}
	p := w.permissions.Request(
		permission.CreatePermissionRequest{
			SessionID:   sessionID,
			Path:        permissionPath,
			ToolName:    WriteToolName,
			Action:      "write",
			Description: fmt.Sprintf("Create file %s", filePath),
			Params: WritePermissionsParams{
				FilePath: filePath,
				Diff:     diffText,
			},
		},
	)
	if !p {
		return ToolResponse{}, permission.ErrorPermissionDenied
	}

	err = os.WriteFile(filePath, []byte(params.Content), 0o644)
	if err != nil {
		return ToolResponse{}, fmt.Errorf("error writing file: %w", err)
	}

	// Check if file exists in history
	file, err := w.files.GetByPathAndSession(ctx, filePath, sessionID)
	if err != nil {
		_, err = w.files.Create(ctx, sessionID, filePath, oldContent)
		if err != nil {
			// Log error but don't fail the operation
			return ToolResponse{}, fmt.Errorf("error creating file history: %w", err)
		}
	}
	if file.Content != oldContent {
		// User Manually changed the content store an intermediate version
		_, err = w.files.CreateVersion(ctx, sessionID, filePath, oldContent)
		if err != nil {
			logging.Debug("Error creating file history version", "error", err)
		}
	}
	// Store the new version
	_, err = w.files.CreateVersion(ctx, sessionID, filePath, params.Content)
	if err != nil {
		logging.Debug("Error creating file history version", "error", err)
	}

	recordFileWrite(filePath)
	recordFileRead(filePath)
	// LSP diagnostics functionality removed

	result := fmt.Sprintf("File successfully written: %s", filePath)
	result = fmt.Sprintf("<result>\n%s\n</result>", result)
	// LSP diagnostics removed
	return WithResponseMetadata(NewTextResponse(result),
		WriteResponseMetadata{
			Diff:      diffText,
			Additions: additions,
			Removals:  removals,
		},
	), nil
}
