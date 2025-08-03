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

type EditParams struct {
	FilePath  string `json:"file_path"`
	OldString string `json:"old_string"`
	NewString string `json:"new_string"`
}

type EditPermissionsParams struct {
	FilePath string `json:"file_path"`
	Diff     string `json:"diff"`
}

type EditResponseMetadata struct {
	Diff      string `json:"diff"`
	Additions int    `json:"additions"`
	Removals  int    `json:"removals"`
}

type editTool struct {
	permissions permission.Service
	files       history.Service
}

const (
	EditToolName = "edit"
)

func NewEditTool(permissions permission.Service, files history.Service) BaseTool {
	return &editTool{
		permissions: permissions,
		files:       files,
	}
}

func (e *editTool) Info() ToolInfo {
	return ToolInfo{
		Name:        EditToolName,
		Description: LoadToolDescription("edit"),
		Parameters: map[string]any{
			"file_path": map[string]any{
				"type":        "string",
				"description": "The absolute path to the file to modify",
			},
			"old_string": map[string]any{
				"type":        "string",
				"description": "The text to replace",
			},
			"new_string": map[string]any{
				"type":        "string",
				"description": "The text to replace it with",
			},
		},
		Required: []string{"file_path", "old_string", "new_string"},
	}
}

func (e *editTool) Run(ctx context.Context, call ToolCall) (ToolResponse, error) {
	var params EditParams
	if err := json.Unmarshal([]byte(call.Input), &params); err != nil {
		return NewTextErrorResponse("invalid parameters"), nil
	}

	if params.FilePath == "" {
		return NewTextErrorResponse("file_path is required"), nil
	}

	if !filepath.IsAbs(params.FilePath) {
		workingDir, err := GetWorkingDirectory(ctx)
		if err != nil {
			return ToolResponse{}, fmt.Errorf("failed to get working directory: %w", err)
		}
		params.FilePath = filepath.Join(workingDir, params.FilePath)
	}

	var response ToolResponse
	var err error

	if params.OldString == "" {
		response, err = e.createNewFile(ctx, params.FilePath, params.NewString)
		if err != nil {
			return response, err
		}
	}

	if params.NewString == "" {
		response, err = e.deleteContent(ctx, params.FilePath, params.OldString)
		if err != nil {
			return response, err
		}
	}

	response, err = e.replaceContent(ctx, params.FilePath, params.OldString, params.NewString)
	if err != nil {
		return response, err
	}
	if response.IsError {
		// Return early if there was an error during content replacement
		// This prevents unnecessary LSP diagnostics processing
		return response, nil
	}

	// LSP diagnostics functionality removed
	text := fmt.Sprintf("<result>\n%s\n</result>\n", response.Content)
	response.Content = text
	return response, nil
}

func (e *editTool) createNewFile(ctx context.Context, filePath, content string) (ToolResponse, error) {
	fileInfo, err := os.Stat(filePath)
	if err == nil {
		if fileInfo.IsDir() {
			return NewTextErrorResponse(fmt.Sprintf("path is a directory, not a file: %s", filePath)), nil
		}
		return NewTextErrorResponse(fmt.Sprintf("file already exists: %s", filePath)), nil
	} else if !os.IsNotExist(err) {
		return ToolResponse{}, fmt.Errorf("failed to access file: %w", err)
	}

	dir := filepath.Dir(filePath)
	if err = os.MkdirAll(dir, 0o755); err != nil {
		return ToolResponse{}, fmt.Errorf("failed to create parent directories: %w", err)
	}

	sessionID, messageID := GetContextValues(ctx)
	if sessionID == "" || messageID == "" {
		return ToolResponse{}, fmt.Errorf("session ID and message ID are required for creating a new file")
	}

	// Simple diff replacement for new file creation
	diffText := fmt.Sprintf("--- /dev/null\n+++ %s\n", filePath)
	lines := strings.Split(content, "\n")
	for i, line := range lines {
		diffText += fmt.Sprintf("@@ -%d,0 +%d,1 @@\n+%s\n", i+1, i+1, line)
	}
	additions := len(lines)
	removals := 0
	rootDir, err := GetWorkingDirectory(ctx)
	if err != nil {
		return ToolResponse{}, fmt.Errorf("failed to get working directory: %w", err)
	}
	permissionPath := filepath.Dir(filePath)
	if strings.HasPrefix(filePath, rootDir) {
		permissionPath = rootDir
	}
	p := e.permissions.Request(
		permission.CreatePermissionRequest{
			SessionID:   sessionID,
			Path:        permissionPath,
			ToolName:    EditToolName,
			Action:      "write",
			Description: fmt.Sprintf("Create file %s", filePath),
			Params: EditPermissionsParams{
				FilePath: filePath,
				Diff:     diffText,
			},
		},
	)
	if !p {
		return ToolResponse{}, permission.ErrorPermissionDenied
	}

	err = os.WriteFile(filePath, []byte(content), 0o644)
	if err != nil {
		return ToolResponse{}, fmt.Errorf("failed to write file: %w", err)
	}

	// File can't be in the history so we create a new file history
	_, err = e.files.Create(ctx, sessionID, filePath, "")
	if err != nil {
		// Log error but don't fail the operation
		return ToolResponse{}, fmt.Errorf("error creating file history: %w", err)
	}

	// Add the new content to the file history
	_, err = e.files.CreateVersion(ctx, sessionID, filePath, content)
	if err != nil {
		// Log error but don't fail the operation
		logging.Debug("Error creating file history version", "error", err)
	}

	recordFileWrite(filePath)
	recordFileRead(filePath)

	return WithResponseMetadata(
		NewTextResponse("File created: "+filePath),
		EditResponseMetadata{
			Diff:      diffText,
			Additions: additions,
			Removals:  removals,
		},
	), nil
}

func (e *editTool) deleteContent(ctx context.Context, filePath, oldString string) (ToolResponse, error) {
	fileInfo, err := os.Stat(filePath)
	if err != nil {
		if os.IsNotExist(err) {
			return NewTextErrorResponse(fmt.Sprintf("file not found: %s", filePath)), nil
		}
		return ToolResponse{}, fmt.Errorf("failed to access file: %w", err)
	}

	if fileInfo.IsDir() {
		return NewTextErrorResponse(fmt.Sprintf("path is a directory, not a file: %s", filePath)), nil
	}

	if getLastReadTime(filePath).IsZero() {
		return NewTextErrorResponse("you must read the file before editing it. Use the View tool first"), nil
	}

	modTime := fileInfo.ModTime()
	lastRead := getLastReadTime(filePath)
	if modTime.After(lastRead) {
		return NewTextErrorResponse(
			fmt.Sprintf("file %s has been modified since it was last read (mod time: %s, last read: %s)",
				filePath, modTime.Format(time.RFC3339), lastRead.Format(time.RFC3339),
			)), nil
	}

	content, err := os.ReadFile(filePath)
	if err != nil {
		return ToolResponse{}, fmt.Errorf("failed to read file: %w", err)
	}

	oldContent := string(content)

	index := strings.Index(oldContent, oldString)
	if index == -1 {
		return NewTextErrorResponse("old_string not found in file. Make sure it matches exactly, including whitespace and line breaks"), nil
	}

	lastIndex := strings.LastIndex(oldContent, oldString)
	if index != lastIndex {
		return NewTextErrorResponse("old_string appears multiple times in the file. Please provide more context to ensure a unique match"), nil
	}

	newContent := oldContent[:index] + oldContent[index+len(oldString):]

	sessionID, messageID := GetContextValues(ctx)

	if sessionID == "" || messageID == "" {
		return ToolResponse{}, fmt.Errorf("session ID and message ID are required for creating a new file")
	}

	// Simple diff replacement for content editing
	diffText := fmt.Sprintf("--- %s\n+++ %s\n", filePath, filePath)
	oldLines := strings.Split(oldContent, "\n")
	newLines := strings.Split(newContent, "\n")
	additions := len(newLines)
	removals := len(oldLines)

	rootDir, err := GetWorkingDirectory(ctx)
	if err != nil {
		return ToolResponse{}, fmt.Errorf("failed to get working directory: %w", err)
	}
	permissionPath := filepath.Dir(filePath)
	if strings.HasPrefix(filePath, rootDir) {
		permissionPath = rootDir
	}
	p := e.permissions.Request(
		permission.CreatePermissionRequest{
			SessionID:   sessionID,
			Path:        permissionPath,
			ToolName:    EditToolName,
			Action:      "write",
			Description: fmt.Sprintf("Delete content from file %s", filePath),
			Params: EditPermissionsParams{
				FilePath: filePath,
				Diff:     diffText,
			},
		},
	)
	if !p {
		return ToolResponse{}, permission.ErrorPermissionDenied
	}

	err = os.WriteFile(filePath, []byte(newContent), 0o644)
	if err != nil {
		return ToolResponse{}, fmt.Errorf("failed to write file: %w", err)
	}

	// Check if file exists in history
	file, err := e.files.GetByPathAndSession(ctx, filePath, sessionID)
	if err != nil {
		_, err = e.files.Create(ctx, sessionID, filePath, oldContent)
		if err != nil {
			// Log error but don't fail the operation
			return ToolResponse{}, fmt.Errorf("error creating file history: %w", err)
		}
	}
	if file.Content != oldContent {
		// User Manually changed the content store an intermediate version
		_, err = e.files.CreateVersion(ctx, sessionID, filePath, oldContent)
		if err != nil {
			logging.Debug("Error creating file history version", "error", err)
		}
	}
	// Store the new version
	_, err = e.files.CreateVersion(ctx, sessionID, filePath, "")
	if err != nil {
		logging.Debug("Error creating file history version", "error", err)
	}

	recordFileWrite(filePath)
	recordFileRead(filePath)

	return WithResponseMetadata(
		NewTextResponse("Content deleted from file: "+filePath),
		EditResponseMetadata{
			Diff:      diffText,
			Additions: additions,
			Removals:  removals,
		},
	), nil
}

func (e *editTool) replaceContent(ctx context.Context, filePath, oldString, newString string) (ToolResponse, error) {
	fileInfo, err := os.Stat(filePath)
	if err != nil {
		if os.IsNotExist(err) {
			return NewTextErrorResponse(fmt.Sprintf("file not found: %s", filePath)), nil
		}
		return ToolResponse{}, fmt.Errorf("failed to access file: %w", err)
	}

	if fileInfo.IsDir() {
		return NewTextErrorResponse(fmt.Sprintf("path is a directory, not a file: %s", filePath)), nil
	}

	if getLastReadTime(filePath).IsZero() {
		return NewTextErrorResponse("you must read the file before editing it. Use the View tool first"), nil
	}

	modTime := fileInfo.ModTime()
	lastRead := getLastReadTime(filePath)
	if modTime.After(lastRead) {
		return NewTextErrorResponse(
			fmt.Sprintf("file %s has been modified since it was last read (mod time: %s, last read: %s)",
				filePath, modTime.Format(time.RFC3339), lastRead.Format(time.RFC3339),
			)), nil
	}

	content, err := os.ReadFile(filePath)
	if err != nil {
		return ToolResponse{}, fmt.Errorf("failed to read file: %w", err)
	}

	oldContent := string(content)

	index := strings.Index(oldContent, oldString)
	if index == -1 {
		return NewTextErrorResponse("old_string not found in file. Make sure it matches exactly, including whitespace and line breaks"), nil
	}

	lastIndex := strings.LastIndex(oldContent, oldString)
	if index != lastIndex {
		return NewTextErrorResponse("old_string appears multiple times in the file. Please provide more context to ensure a unique match"), nil
	}

	newContent := oldContent[:index] + newString + oldContent[index+len(oldString):]

	if oldContent == newContent {
		return NewTextErrorResponse("new content is the same as old content. No changes made."), nil
	}
	sessionID, messageID := GetContextValues(ctx)

	if sessionID == "" || messageID == "" {
		return ToolResponse{}, fmt.Errorf("session ID and message ID are required for creating a new file")
	}
	// Simple diff replacement for content editing
	diffText := fmt.Sprintf("--- %s\n+++ %s\n", filePath, filePath)
	oldLines := strings.Split(oldContent, "\n")
	newLines := strings.Split(newContent, "\n")
	additions := len(newLines)
	removals := len(oldLines)
	rootDir, err := GetWorkingDirectory(ctx)
	if err != nil {
		return ToolResponse{}, fmt.Errorf("failed to get working directory: %w", err)
	}
	permissionPath := filepath.Dir(filePath)
	if strings.HasPrefix(filePath, rootDir) {
		permissionPath = rootDir
	}
	p := e.permissions.Request(
		permission.CreatePermissionRequest{
			SessionID:   sessionID,
			Path:        permissionPath,
			ToolName:    EditToolName,
			Action:      "write",
			Description: fmt.Sprintf("Replace content in file %s", filePath),
			Params: EditPermissionsParams{
				FilePath: filePath,
				Diff:     diffText,
			},
		},
	)
	if !p {
		return ToolResponse{}, permission.ErrorPermissionDenied
	}

	err = os.WriteFile(filePath, []byte(newContent), 0o644)
	if err != nil {
		return ToolResponse{}, fmt.Errorf("failed to write file: %w", err)
	}

	// Check if file exists in history
	file, err := e.files.GetByPathAndSession(ctx, filePath, sessionID)
	if err != nil {
		_, err = e.files.Create(ctx, sessionID, filePath, oldContent)
		if err != nil {
			// Log error but don't fail the operation
			return ToolResponse{}, fmt.Errorf("error creating file history: %w", err)
		}
	}
	if file.Content != oldContent {
		// User Manually changed the content store an intermediate version
		_, err = e.files.CreateVersion(ctx, sessionID, filePath, oldContent)
		if err != nil {
			logging.Debug("Error creating file history version", "error", err)
		}
	}
	// Store the new version
	_, err = e.files.CreateVersion(ctx, sessionID, filePath, newContent)
	if err != nil {
		logging.Debug("Error creating file history version", "error", err)
	}

	recordFileWrite(filePath)
	recordFileRead(filePath)

	return WithResponseMetadata(
		NewTextResponse("Content replaced in file: "+filePath),
		EditResponseMetadata{
			Diff:      diffText,
			Additions: additions,
			Removals:  removals,
		}), nil
}
