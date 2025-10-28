package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"mix/internal/history"
	"mix/internal/permission"
	"mix/internal/pubsub"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

// Mock implementations for dependencies
type mockEditPermissionService struct {
	mock.Mock
}

func (m *mockEditPermissionService) Request(req permission.CreatePermissionRequest) bool {
	args := m.Called(req)
	return args.Bool(0)
}

func (m *mockEditPermissionService) GrantPersistant(req permission.PermissionRequest) {
	m.Called(req)
}

func (m *mockEditPermissionService) Grant(req permission.PermissionRequest) {
	m.Called(req)
}

func (m *mockEditPermissionService) Deny(req permission.PermissionRequest) {
	m.Called(req)
}

func (m *mockEditPermissionService) Subscribe(ctx context.Context) <-chan pubsub.Event[permission.PermissionRequest] {
	args := m.Called(ctx)
	return args.Get(0).(<-chan pubsub.Event[permission.PermissionRequest])
}

func (m *mockEditPermissionService) Unsubscribe(callback func(permission.PermissionRequest)) {
	m.Called(callback)
}

type mockEditHistoryService struct {
	mock.Mock
}

func (m *mockEditHistoryService) Create(ctx context.Context, sessionID, path, content string) (history.File, error) {
	args := m.Called(ctx, sessionID, path, content)
	return args.Get(0).(history.File), args.Error(1)
}

func (m *mockEditHistoryService) CreateVersion(ctx context.Context, sessionID, path, content string) (history.File, error) {
	args := m.Called(ctx, sessionID, path, content)
	return args.Get(0).(history.File), args.Error(1)
}

func (m *mockEditHistoryService) Get(ctx context.Context, id string) (history.File, error) {
	args := m.Called(ctx, id)
	return args.Get(0).(history.File), args.Error(1)
}

func (m *mockEditHistoryService) GetByPathAndSession(ctx context.Context, path, sessionID string) (history.File, error) {
	args := m.Called(ctx, path, sessionID)
	return args.Get(0).(history.File), args.Error(1)
}

func (m *mockEditHistoryService) ListBySession(ctx context.Context, sessionID string) ([]history.File, error) {
	args := m.Called(ctx, sessionID)
	return args.Get(0).([]history.File), args.Error(1)
}

func (m *mockEditHistoryService) ListLatestSessionFiles(ctx context.Context, sessionID string) ([]history.File, error) {
	args := m.Called(ctx, sessionID)
	return args.Get(0).([]history.File), args.Error(1)
}

func (m *mockEditHistoryService) Update(ctx context.Context, file history.File) (history.File, error) {
	args := m.Called(ctx, file)
	return args.Get(0).(history.File), args.Error(1)
}

func (m *mockEditHistoryService) Delete(ctx context.Context, id string) error {
	args := m.Called(ctx, id)
	return args.Error(0)
}

func (m *mockEditHistoryService) Subscribe(ctx context.Context) <-chan pubsub.Event[history.File] {
	args := m.Called(ctx)
	return args.Get(0).(<-chan pubsub.Event[history.File])
}

func (m *mockEditHistoryService) Unsubscribe(callback func(history.File)) {
	m.Called(callback)
}

// Note: clearFileRecords function is defined in file_test.go

// Helper functions for tests
func setupEditTestContext() context.Context {
	ctx := context.Background()
	ctx = context.WithValue(ctx, SessionIDContextKey, "test-session-id")
	ctx = context.WithValue(ctx, MessageIDContextKey, "test-message-id")
	ctx = context.WithValue(ctx, SessionStorageContextKey, "/tmp/test-session")
	return ctx
}

func createTempFile(t *testing.T, content string) string {
	tempFile, err := os.CreateTemp("", "edit_test_*.txt")
	require.NoError(t, err)

	if content != "" {
		_, err = tempFile.WriteString(content)
		require.NoError(t, err)
	}

	_ = tempFile.Close()
	return tempFile.Name()
}

func cleanupTempFile(filePath string) {
	_ = os.Remove(filePath)
}

// Test struct creation and validation
func TestEditParams_JSONSerialization(t *testing.T) {
	params := EditParams{
		FilePath:  "/test/path.txt",
		OldString: "old content",
		NewString: "new content",
	}

	// Test marshalling
	data, err := json.Marshal(params)
	assert.NoError(t, err)

	// Test unmarshalling
	var unmarshalled EditParams
	err = json.Unmarshal(data, &unmarshalled)
	assert.NoError(t, err)
	assert.Equal(t, params, unmarshalled)
}

func TestEditPermissionsParams_JSONSerialization(t *testing.T) {
	params := EditPermissionsParams{
		FilePath: "/test/path.txt",
		Diff:     "--- a/file\n+++ b/file\n@@ -1,1 +1,1 @@\n-old\n+new",
	}

	// Test marshalling
	data, err := json.Marshal(params)
	assert.NoError(t, err)

	// Test unmarshalling
	var unmarshalled EditPermissionsParams
	err = json.Unmarshal(data, &unmarshalled)
	assert.NoError(t, err)
	assert.Equal(t, params, unmarshalled)
}

func TestEditResponseMetadata_JSONSerialization(t *testing.T) {
	metadata := EditResponseMetadata{
		Diff:      "--- a/file\n+++ b/file\n@@ -1,1 +1,1 @@\n-old\n+new",
		Additions: 5,
		Removals:  3,
	}

	// Test marshalling
	data, err := json.Marshal(metadata)
	assert.NoError(t, err)

	// Test unmarshalling
	var unmarshalled EditResponseMetadata
	err = json.Unmarshal(data, &unmarshalled)
	assert.NoError(t, err)
	assert.Equal(t, metadata, unmarshalled)
}

// Test tool creation and interface compliance
func TestNewEditTool(t *testing.T) {
	mockPermissions := &mockEditPermissionService{}
	mockFiles := &mockEditHistoryService{}

	tool := NewEditTool(mockPermissions, mockFiles)

	// Test that it implements BaseTool interface
	assert.Implements(t, (*BaseTool)(nil), tool)

	// Test that it's the correct type
	editTool, ok := tool.(*editTool)
	assert.True(t, ok)
	assert.Equal(t, mockPermissions, editTool.permissions)
	assert.Equal(t, mockFiles, editTool.files)
}

func TestEditTool_Info(t *testing.T) {
	mockPermissions := &mockEditPermissionService{}
	mockFiles := &mockEditHistoryService{}
	tool := NewEditTool(mockPermissions, mockFiles)

	info := tool.Info()

	assert.Equal(t, EditToolName, info.Name)
	assert.Equal(t, "edit", info.Name)
	assert.NotEmpty(t, info.Description)
	assert.Len(t, info.Required, 3)
	assert.Contains(t, info.Required, "file_path")
	assert.Contains(t, info.Required, "old_string")
	assert.Contains(t, info.Required, "new_string")

	// Check parameters structure
	assert.Contains(t, info.Parameters, "file_path")
	assert.Contains(t, info.Parameters, "old_string")
	assert.Contains(t, info.Parameters, "new_string")

	filePathParam := info.Parameters["file_path"].(map[string]any)
	assert.Equal(t, "string", filePathParam["type"])
	assert.Contains(t, filePathParam["description"], "absolute path")
}

// Test Run method with invalid parameters
func TestEditTool_Run_InvalidJSON(t *testing.T) {
	mockPermissions := &mockEditPermissionService{}
	mockFiles := &mockEditHistoryService{}
	tool := NewEditTool(mockPermissions, mockFiles)

	ctx := setupEditTestContext()
	call := ToolCall{
		ID:    "test-call",
		Name:  "edit",
		Input: `{"invalid": json}`,
	}

	response, err := tool.Run(ctx, call)

	assert.NoError(t, err)
	assert.True(t, response.IsError)
	assert.Equal(t, "invalid parameters", response.Content)
}

func TestEditTool_Run_MissingFilePath(t *testing.T) {
	mockPermissions := &mockEditPermissionService{}
	mockFiles := &mockEditHistoryService{}
	tool := NewEditTool(mockPermissions, mockFiles)

	ctx := setupEditTestContext()
	params := EditParams{
		FilePath:  "",
		OldString: "old",
		NewString: "new",
	}
	paramsJSON, _ := json.Marshal(params)
	call := ToolCall{
		ID:    "test-call",
		Name:  "edit",
		Input: string(paramsJSON),
	}

	response, err := tool.Run(ctx, call)

	assert.NoError(t, err)
	assert.True(t, response.IsError)
	assert.Equal(t, "file_path is required", response.Content)
}

func TestEditTool_Run_RelativePathConversion(t *testing.T) {
	mockPermissions := &mockEditPermissionService{}
	mockFiles := &mockEditHistoryService{}
	tool := NewEditTool(mockPermissions, mockFiles)

	// Create context without session storage
	ctx := context.Background()
	ctx = context.WithValue(ctx, SessionIDContextKey, "test-session-id")
	ctx = context.WithValue(ctx, MessageIDContextKey, "test-message-id")

	params := EditParams{
		FilePath:  "relative/path.txt",
		OldString: "",
		NewString: "content",
	}
	paramsJSON, _ := json.Marshal(params)
	call := ToolCall{
		ID:    "test-call",
		Name:  "edit",
		Input: string(paramsJSON),
	}

	_, err := tool.Run(ctx, call)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "session storage directory")
}

// Test createNewFile method
func TestEditTool_CreateNewFile_Success(t *testing.T) {
	clearFileRecords() // Clear any previous file records

	mockPermissions := &mockEditPermissionService{}
	mockFiles := &mockEditHistoryService{}
	tool := NewEditTool(mockPermissions, mockFiles).(*editTool)

	ctx := setupEditTestContext()

	// Create a temporary directory for the test
	tempDir := t.TempDir()
	filePath := filepath.Join(tempDir, "new_file.txt")
	content := "Hello, World!"

	// Setup mocks
	mockPermissions.On("Request", mock.MatchedBy(func(req permission.CreatePermissionRequest) bool {
		return req.SessionID == "test-session-id" &&
			req.ToolName == EditToolName &&
			req.Action == "write"
	})).Return(true)

	mockFiles.On("Create", ctx, "test-session-id", filePath, "").Return(history.File{}, nil)
	mockFiles.On("CreateVersion", ctx, "test-session-id", filePath, content).Return(history.File{}, nil)

	response, err := tool.createNewFile(ctx, filePath, content)

	assert.NoError(t, err)
	assert.False(t, response.IsError)
	assert.Contains(t, response.Content, "File created:")
	assert.Contains(t, response.Content, filePath)

	// Verify file was actually created
	createdContent, err := os.ReadFile(filePath)
	assert.NoError(t, err)
	assert.Equal(t, content, string(createdContent))

	// Verify metadata
	assert.NotEmpty(t, response.Metadata)
	var metadata EditResponseMetadata
	err = json.Unmarshal([]byte(response.Metadata), &metadata)
	assert.NoError(t, err)
	assert.Greater(t, metadata.Additions, 0)
	assert.Equal(t, 0, metadata.Removals)
	assert.NotEmpty(t, metadata.Diff)

	mockPermissions.AssertExpectations(t)
	mockFiles.AssertExpectations(t)
}

func TestEditTool_CreateNewFile_FileExists(t *testing.T) {
	clearFileRecords()

	mockPermissions := &mockEditPermissionService{}
	mockFiles := &mockEditHistoryService{}
	tool := NewEditTool(mockPermissions, mockFiles).(*editTool)

	ctx := setupEditTestContext()

	// Create a temporary file that already exists
	existingFile := createTempFile(t, "existing content")
	defer cleanupTempFile(existingFile)

	response, err := tool.createNewFile(ctx, existingFile, "new content")

	assert.NoError(t, err)
	assert.True(t, response.IsError)
	assert.Contains(t, response.Content, "file already exists")
}

func TestEditTool_CreateNewFile_DirectoryExists(t *testing.T) {
	clearFileRecords()

	mockPermissions := &mockEditPermissionService{}
	mockFiles := &mockEditHistoryService{}
	tool := NewEditTool(mockPermissions, mockFiles).(*editTool)

	ctx := setupEditTestContext()

	// Create a temporary directory
	tempDir := t.TempDir()

	response, err := tool.createNewFile(ctx, tempDir, "content")

	assert.NoError(t, err)
	assert.True(t, response.IsError)
	assert.Contains(t, response.Content, "path is a directory")
}

func TestEditTool_CreateNewFile_PermissionDenied(t *testing.T) {
	clearFileRecords()

	mockPermissions := &mockEditPermissionService{}
	mockFiles := &mockEditHistoryService{}
	tool := NewEditTool(mockPermissions, mockFiles).(*editTool)

	ctx := setupEditTestContext()
	tempDir := t.TempDir()
	filePath := filepath.Join(tempDir, "denied_file.txt")

	// Setup mocks - permission denied
	mockPermissions.On("Request", mock.AnythingOfType("permission.CreatePermissionRequest")).Return(false)

	_, err := tool.createNewFile(ctx, filePath, "content")

	assert.Error(t, err)
	assert.Equal(t, permission.ErrorPermissionDenied, err)

	mockPermissions.AssertExpectations(t)
}

func TestEditTool_CreateNewFile_MissingSessionInfo(t *testing.T) {
	clearFileRecords()

	mockPermissions := &mockEditPermissionService{}
	mockFiles := &mockEditHistoryService{}
	tool := NewEditTool(mockPermissions, mockFiles).(*editTool)

	// Context without session info
	ctx := context.Background()
	tempDir := t.TempDir()
	filePath := filepath.Join(tempDir, "file.txt")

	_, err := tool.createNewFile(ctx, filePath, "content")

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "session ID and message ID are required")
}

// Test replaceContent method
func TestEditTool_ReplaceContent_Success(t *testing.T) {
	clearFileRecords()

	mockPermissions := &mockEditPermissionService{}
	mockFiles := &mockEditHistoryService{}
	tool := NewEditTool(mockPermissions, mockFiles).(*editTool)

	ctx := setupEditTestContext()

	// Create a temporary file with content
	originalContent := "Hello, World!\nThis is a test file.\nGoodbye!"
	tempFile := createTempFile(t, originalContent)
	defer cleanupTempFile(tempFile)

	// Record that file was read
	recordFileRead(tempFile)

	oldString := "World"
	newString := "Universe"

	// Setup mocks
	mockPermissions.On("Request", mock.MatchedBy(func(req permission.CreatePermissionRequest) bool {
		return req.SessionID == "test-session-id" &&
			req.ToolName == EditToolName &&
			req.Action == "write"
	})).Return(true)

	mockFiles.On("GetByPathAndSession", ctx, tempFile, "test-session-id").Return(
		history.File{Content: originalContent}, nil)
	mockFiles.On("CreateVersion", ctx, "test-session-id", tempFile, mock.AnythingOfType("string")).Return(
		history.File{}, nil)

	response, err := tool.replaceContent(ctx, tempFile, oldString, newString, false)

	assert.NoError(t, err)
	assert.False(t, response.IsError)
	assert.Contains(t, response.Content, "Content replaced in file:")

	// Verify file content was changed
	newContent, err := os.ReadFile(tempFile)
	assert.NoError(t, err)
	expectedContent := strings.Replace(originalContent, oldString, newString, 1)
	assert.Equal(t, expectedContent, string(newContent))

	// Verify metadata
	assert.NotEmpty(t, response.Metadata)
	var metadata EditResponseMetadata
	err = json.Unmarshal([]byte(response.Metadata), &metadata)
	assert.NoError(t, err)
	assert.NotEmpty(t, metadata.Diff)

	mockPermissions.AssertExpectations(t)
	mockFiles.AssertExpectations(t)
}

func TestEditTool_ReplaceContent_FileNotFound(t *testing.T) {
	clearFileRecords()

	mockPermissions := &mockEditPermissionService{}
	mockFiles := &mockEditHistoryService{}
	tool := NewEditTool(mockPermissions, mockFiles).(*editTool)

	ctx := setupEditTestContext()
	nonExistentFile := "/tmp/nonexistent_file.txt"

	response, err := tool.replaceContent(ctx, nonExistentFile, "old", "new", false)

	assert.NoError(t, err)
	assert.True(t, response.IsError)
	assert.Contains(t, response.Content, "file not found")
}

func TestEditTool_ReplaceContent_FileNotRead(t *testing.T) {
	clearFileRecords()

	mockPermissions := &mockEditPermissionService{}
	mockFiles := &mockEditHistoryService{}
	tool := NewEditTool(mockPermissions, mockFiles).(*editTool)

	ctx := setupEditTestContext()
	tempFile := createTempFile(t, "content")
	defer cleanupTempFile(tempFile)

	response, err := tool.replaceContent(ctx, tempFile, "old", "new", false)

	assert.NoError(t, err)
	assert.True(t, response.IsError)
	assert.Contains(t, response.Content, "This tool will error if you attempt an edit without reading the file")
}

func TestEditTool_ReplaceContent_OldStringNotFound(t *testing.T) {
	clearFileRecords()

	mockPermissions := &mockEditPermissionService{}
	mockFiles := &mockEditHistoryService{}
	tool := NewEditTool(mockPermissions, mockFiles).(*editTool)

	ctx := setupEditTestContext()
	tempFile := createTempFile(t, "Hello, World!")
	defer cleanupTempFile(tempFile)

	recordFileRead(tempFile)

	response, err := tool.replaceContent(ctx, tempFile, "nonexistent", "new", false)

	assert.NoError(t, err)
	assert.True(t, response.IsError)
	assert.Contains(t, response.Content, "old_string not found in file")
}

func TestEditTool_ReplaceContent_MultipleOccurrences(t *testing.T) {
	clearFileRecords()

	mockPermissions := &mockEditPermissionService{}
	mockFiles := &mockEditHistoryService{}
	tool := NewEditTool(mockPermissions, mockFiles).(*editTool)

	ctx := setupEditTestContext()
	tempFile := createTempFile(t, "Hello, World! Hello, World!")
	defer cleanupTempFile(tempFile)

	recordFileRead(tempFile)

	response, err := tool.replaceContent(ctx, tempFile, "Hello", "Hi", false)

	assert.NoError(t, err)
	assert.True(t, response.IsError)
	assert.Contains(t, response.Content, "old_string is not unique")
}

func TestEditTool_ReplaceContent_NoChange(t *testing.T) {
	clearFileRecords()

	mockPermissions := &mockEditPermissionService{}
	mockFiles := &mockEditHistoryService{}
	tool := NewEditTool(mockPermissions, mockFiles).(*editTool)

	ctx := setupEditTestContext()
	content := "Hello, World!"
	tempFile := createTempFile(t, content)
	defer cleanupTempFile(tempFile)

	recordFileRead(tempFile)

	response, err := tool.replaceContent(ctx, tempFile, "World", "World", false)

	assert.NoError(t, err)
	assert.True(t, response.IsError)
	assert.Contains(t, response.Content, "new content is the same as old content")
}

func TestEditTool_ReplaceContent_FileModifiedSinceRead(t *testing.T) {
	clearFileRecords()

	mockPermissions := &mockEditPermissionService{}
	mockFiles := &mockEditHistoryService{}
	tool := NewEditTool(mockPermissions, mockFiles).(*editTool)

	ctx := setupEditTestContext()
	tempFile := createTempFile(t, "original content")
	defer cleanupTempFile(tempFile)

	// Record file read
	recordFileRead(tempFile)

	// Wait a moment and modify the file
	time.Sleep(10 * time.Millisecond)
	err := os.WriteFile(tempFile, []byte("modified content"), 0644)
	require.NoError(t, err)

	response, err := tool.replaceContent(ctx, tempFile, "original", "new", false)

	assert.NoError(t, err)
	assert.True(t, response.IsError)
	assert.Contains(t, response.Content, "file")
	assert.Contains(t, response.Content, "has been modified since it was last read")
}

// Test deleteContent method
func TestEditTool_DeleteContent_Success(t *testing.T) {
	clearFileRecords()

	mockPermissions := &mockEditPermissionService{}
	mockFiles := &mockEditHistoryService{}
	tool := NewEditTool(mockPermissions, mockFiles).(*editTool)

	ctx := setupEditTestContext()

	originalContent := "Hello, World!\nThis will be deleted.\nGoodbye!"
	tempFile := createTempFile(t, originalContent)
	defer cleanupTempFile(tempFile)

	recordFileRead(tempFile)

	deleteString := "This will be deleted.\n"

	// Setup mocks
	mockPermissions.On("Request", mock.AnythingOfType("permission.CreatePermissionRequest")).Return(true)
	mockFiles.On("GetByPathAndSession", ctx, tempFile, "test-session-id").Return(
		history.File{Content: originalContent}, nil)
	mockFiles.On("CreateVersion", ctx, "test-session-id", tempFile, "").Return(
		history.File{}, nil)

	response, err := tool.deleteContent(ctx, tempFile, deleteString)

	assert.NoError(t, err)
	assert.False(t, response.IsError)
	assert.Contains(t, response.Content, "Content deleted from file:")

	// Verify content was deleted
	newContent, err := os.ReadFile(tempFile)
	assert.NoError(t, err)
	expectedContent := strings.Replace(originalContent, deleteString, "", 1)
	assert.Equal(t, expectedContent, string(newContent))

	mockPermissions.AssertExpectations(t)
	mockFiles.AssertExpectations(t)
}

// Test Run method integration scenarios
func TestEditTool_Run_CreateNewFileIntegration(t *testing.T) {
	clearFileRecords()

	mockPermissions := &mockEditPermissionService{}
	mockFiles := &mockEditHistoryService{}
	tool := NewEditTool(mockPermissions, mockFiles)

	ctx := setupEditTestContext()
	tempDir := t.TempDir()
	filePath := filepath.Join(tempDir, "new_file.txt")

	params := EditParams{
		FilePath:  filePath,
		OldString: "", // Empty old string triggers new file creation
		NewString: "Hello, World!",
	}
	paramsJSON, _ := json.Marshal(params)
	call := ToolCall{
		ID:    "test-call",
		Name:  "edit",
		Input: string(paramsJSON),
	}

	// Setup mocks
	mockPermissions.On("Request", mock.AnythingOfType("permission.CreatePermissionRequest")).Return(true)
	mockFiles.On("Create", ctx, "test-session-id", filePath, "").Return(history.File{}, nil)
	mockFiles.On("CreateVersion", ctx, "test-session-id", filePath, "Hello, World!").Return(
		history.File{}, nil)

	response, err := tool.Run(ctx, call)

	assert.NoError(t, err)
	assert.False(t, response.IsError)
	assert.Contains(t, response.Content, "<result>")
	assert.Contains(t, response.Content, "File created:")

	// Verify file exists
	_, err = os.Stat(filePath)
	assert.NoError(t, err)

	mockPermissions.AssertExpectations(t)
	mockFiles.AssertExpectations(t)
}

func TestEditTool_Run_DeleteContentIntegration(t *testing.T) {
	clearFileRecords()

	mockPermissions := &mockEditPermissionService{}
	mockFiles := &mockEditHistoryService{}
	tool := NewEditTool(mockPermissions, mockFiles)

	ctx := setupEditTestContext()
	tempFile := createTempFile(t, "Hello, World!")
	defer cleanupTempFile(tempFile)

	recordFileRead(tempFile)

	params := EditParams{
		FilePath:  tempFile,
		OldString: "Hello, ",
		NewString: "", // Empty new string triggers deletion
	}
	paramsJSON, _ := json.Marshal(params)
	call := ToolCall{
		ID:    "test-call",
		Name:  "edit",
		Input: string(paramsJSON),
	}

	// Setup mocks
	mockPermissions.On("Request", mock.AnythingOfType("permission.CreatePermissionRequest")).Return(true)
	mockFiles.On("GetByPathAndSession", ctx, tempFile, "test-session-id").Return(
		history.File{Content: "Hello, World!"}, nil)
	mockFiles.On("CreateVersion", ctx, "test-session-id", tempFile, "").Return(
		history.File{}, nil)

	response, err := tool.Run(ctx, call)

	assert.NoError(t, err)
	assert.False(t, response.IsError)
	assert.Contains(t, response.Content, "<result>")
	assert.Contains(t, response.Content, "Content deleted from file:")

	mockPermissions.AssertExpectations(t)
	mockFiles.AssertExpectations(t)
}

func TestEditTool_Run_ReplaceContentIntegration(t *testing.T) {
	clearFileRecords()

	mockPermissions := &mockEditPermissionService{}
	mockFiles := &mockEditHistoryService{}
	tool := NewEditTool(mockPermissions, mockFiles)

	ctx := setupEditTestContext()
	tempFile := createTempFile(t, "Hello, World!")
	defer cleanupTempFile(tempFile)

	recordFileRead(tempFile)

	params := EditParams{
		FilePath:  tempFile,
		OldString: "World",
		NewString: "Universe",
	}
	paramsJSON, _ := json.Marshal(params)
	call := ToolCall{
		ID:    "test-call",
		Name:  "edit",
		Input: string(paramsJSON),
	}

	// Setup mocks
	mockPermissions.On("Request", mock.AnythingOfType("permission.CreatePermissionRequest")).Return(true)
	mockFiles.On("GetByPathAndSession", ctx, tempFile, "test-session-id").Return(
		history.File{Content: "Hello, World!"}, nil)
	mockFiles.On("CreateVersion", ctx, "test-session-id", tempFile, "Hello, Universe!").Return(
		history.File{}, nil)

	response, err := tool.Run(ctx, call)

	assert.NoError(t, err)
	assert.False(t, response.IsError)
	assert.Contains(t, response.Content, "<result>")
	assert.Contains(t, response.Content, "Content replaced in file:")

	mockPermissions.AssertExpectations(t)
	mockFiles.AssertExpectations(t)
}

// Test constants
func TestEditToolName(t *testing.T) {
	assert.Equal(t, "edit", EditToolName)
}

// Test edge cases and error scenarios
func TestEditTool_EdgeCases(t *testing.T) {
	t.Run("empty strings", func(t *testing.T) {
		clearFileRecords()

		mockPermissions := &mockEditPermissionService{}
		mockFiles := &mockEditHistoryService{}
		tool := NewEditTool(mockPermissions, mockFiles)

		ctx := setupEditTestContext()
		tempFile := createTempFile(t, "content")
		defer cleanupTempFile(tempFile)

		recordFileRead(tempFile)

		params := EditParams{
			FilePath:  tempFile,
			OldString: "",
			NewString: "",
		}
		paramsJSON, _ := json.Marshal(params)
		call := ToolCall{
			ID:    "test-call",
			Name:  "edit",
			Input: string(paramsJSON),
		}

		// This should trigger new file creation logic but file exists
		response, err := tool.Run(ctx, call)
		assert.NoError(t, err)
		assert.True(t, response.IsError)
		assert.Contains(t, response.Content, "file already exists")
	})

	t.Run("unicode content", func(t *testing.T) {
		clearFileRecords()

		mockPermissions := &mockEditPermissionService{}
		mockFiles := &mockEditHistoryService{}
		tool := NewEditTool(mockPermissions, mockFiles)

		ctx := setupEditTestContext()
		unicodeContent := "Hello, 世界! 🌍"
		tempFile := createTempFile(t, unicodeContent)
		defer cleanupTempFile(tempFile)

		recordFileRead(tempFile)

		params := EditParams{
			FilePath:  tempFile,
			OldString: "世界",
			NewString: "宇宙",
		}
		paramsJSON, _ := json.Marshal(params)
		call := ToolCall{
			ID:    "test-call",
			Name:  "edit",
			Input: string(paramsJSON),
		}

		mockPermissions.On("Request", mock.AnythingOfType("permission.CreatePermissionRequest")).Return(true)
		mockFiles.On("GetByPathAndSession", ctx, tempFile, "test-session-id").Return(
			history.File{Content: unicodeContent}, nil)
		mockFiles.On("CreateVersion", ctx, "test-session-id", tempFile, "Hello, 宇宙! 🌍").Return(
			history.File{}, nil)

		response, err := tool.Run(ctx, call)
		assert.NoError(t, err)
		assert.False(t, response.IsError)

		mockPermissions.AssertExpectations(t)
		mockFiles.AssertExpectations(t)
	})
}

// Test context handling
func TestEditTool_ContextHandling(t *testing.T) {
	t.Run("missing session storage context", func(t *testing.T) {
		mockPermissions := &mockEditPermissionService{}
		mockFiles := &mockEditHistoryService{}
		tool := NewEditTool(mockPermissions, mockFiles)

		ctx := context.Background()
		ctx = context.WithValue(ctx, SessionIDContextKey, "test-session-id")
		ctx = context.WithValue(ctx, MessageIDContextKey, "test-message-id")
		// Missing SessionStorageContextKey

		params := EditParams{
			FilePath:  "relative/path.txt",
			OldString: "",
			NewString: "content",
		}
		paramsJSON, _ := json.Marshal(params)
		call := ToolCall{
			ID:    "test-call",
			Name:  "edit",
			Input: string(paramsJSON),
		}

		_, err := tool.Run(ctx, call)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "session storage directory")
	})
}

func TestEditTool_HistoryServiceErrors(t *testing.T) {
	clearFileRecords()

	mockPermissions := &mockEditPermissionService{}
	mockFiles := &mockEditHistoryService{}
	tool := NewEditTool(mockPermissions, mockFiles).(*editTool)

	ctx := setupEditTestContext()
	tempDir := t.TempDir()
	filePath := filepath.Join(tempDir, "test_file.txt")

	// Setup permission mock to allow operation
	mockPermissions.On("Request", mock.AnythingOfType("permission.CreatePermissionRequest")).Return(true)

	// Setup history service to return error
	mockFiles.On("Create", ctx, "test-session-id", filePath, "").Return(history.File{}, fmt.Errorf("database error"))

	_, err := tool.createNewFile(ctx, filePath, "content")

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "error creating file history")

	mockPermissions.AssertExpectations(t)
	mockFiles.AssertExpectations(t)
}
