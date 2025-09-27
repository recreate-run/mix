package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"

	"mix/internal/history"
	"mix/internal/llm/interfaces"
	"mix/internal/permission"
	"mix/internal/pubsub"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// Mock implementations for testing write tool specifically
type writeToolMockPermissionService struct {
	mock.Mock
}

func (m *writeToolMockPermissionService) Subscribe(ctx context.Context) <-chan pubsub.Event[permission.PermissionRequest] {
	args := m.Called(ctx)
	return args.Get(0).(<-chan pubsub.Event[permission.PermissionRequest])
}

func (m *writeToolMockPermissionService) GrantPersistant(permission permission.PermissionRequest) {
	m.Called(permission)
}

func (m *writeToolMockPermissionService) Grant(permission permission.PermissionRequest) {
	m.Called(permission)
}

func (m *writeToolMockPermissionService) Deny(permission permission.PermissionRequest) {
	m.Called(permission)
}

func (m *writeToolMockPermissionService) Request(opts permission.CreatePermissionRequest) bool {
	args := m.Called(opts)
	return args.Bool(0)
}

type writeToolMockHistoryService struct {
	mock.Mock
}

func (m *writeToolMockHistoryService) Subscribe(ctx context.Context) <-chan pubsub.Event[history.File] {
	args := m.Called(ctx)
	return args.Get(0).(<-chan pubsub.Event[history.File])
}

func (m *writeToolMockHistoryService) Create(ctx context.Context, sessionID, path, content string) (history.File, error) {
	args := m.Called(ctx, sessionID, path, content)
	return args.Get(0).(history.File), args.Error(1)
}

func (m *writeToolMockHistoryService) CreateVersion(ctx context.Context, sessionID, path, content string) (history.File, error) {
	args := m.Called(ctx, sessionID, path, content)
	return args.Get(0).(history.File), args.Error(1)
}

func (m *writeToolMockHistoryService) Get(ctx context.Context, id string) (history.File, error) {
	args := m.Called(ctx, id)
	return args.Get(0).(history.File), args.Error(1)
}

func (m *writeToolMockHistoryService) GetByPathAndSession(ctx context.Context, path, sessionID string) (history.File, error) {
	args := m.Called(ctx, path, sessionID)
	return args.Get(0).(history.File), args.Error(1)
}

func (m *writeToolMockHistoryService) ListBySession(ctx context.Context, sessionID string) ([]history.File, error) {
	args := m.Called(ctx, sessionID)
	return args.Get(0).([]history.File), args.Error(1)
}

func (m *writeToolMockHistoryService) ListLatestSessionFiles(ctx context.Context, sessionID string) ([]history.File, error) {
	args := m.Called(ctx, sessionID)
	return args.Get(0).([]history.File), args.Error(1)
}

func (m *writeToolMockHistoryService) Update(ctx context.Context, file history.File) (history.File, error) {
	args := m.Called(ctx, file)
	return args.Get(0).(history.File), args.Error(1)
}

func (m *writeToolMockHistoryService) Delete(ctx context.Context, id string) error {
	args := m.Called(ctx, id)
	return args.Error(0)
}

// Test setup helpers for write tool
func setupWriteToolTestDir(t *testing.T) string {
	tempDir, err := os.MkdirTemp("", "write_tool_test_*")
	assert.NoError(t, err)
	t.Cleanup(func() {
		os.RemoveAll(tempDir)
	})
	return tempDir
}

func createWriteToolTestContext(sessionID, messageID, storageDir string) context.Context {
	ctx := context.Background()
	ctx = context.WithValue(ctx, interfaces.SessionIDContextKey, sessionID)
	ctx = context.WithValue(ctx, interfaces.MessageIDContextKey, messageID)
	ctx = context.WithValue(ctx, interfaces.SessionStorageContextKey, storageDir)
	return ctx
}

// Note: clearFileRecords function is defined in file_test.go

// Test WriteParams struct
func TestWriteParams(t *testing.T) {
	params := WriteParams{
		FilePath: "/test/path.txt",
		Content:  "test content",
	}

	assert.Equal(t, "/test/path.txt", params.FilePath)
	assert.Equal(t, "test content", params.Content)
}

// Test WritePermissionsParams struct
func TestWritePermissionsParams(t *testing.T) {
	params := WritePermissionsParams{
		FilePath: "/test/path.txt",
		Diff:     "--- old\n+++ new",
	}

	assert.Equal(t, "/test/path.txt", params.FilePath)
	assert.Equal(t, "--- old\n+++ new", params.Diff)
}

// Test WriteResponseMetadata struct
func TestWriteResponseMetadata(t *testing.T) {
	metadata := WriteResponseMetadata{
		Diff:      "--- old\n+++ new",
		Additions: 5,
		Removals:  3,
	}

	assert.Equal(t, "--- old\n+++ new", metadata.Diff)
	assert.Equal(t, 5, metadata.Additions)
	assert.Equal(t, 3, metadata.Removals)
}

// Test JSON serialization/deserialization
func TestWriteParamsJSONSerialization(t *testing.T) {
	original := WriteParams{
		FilePath: "/test/path.txt",
		Content:  "test content with\nmultiple lines",
	}

	// Serialize
	data, err := json.Marshal(original)
	assert.NoError(t, err)

	// Deserialize
	var deserialized WriteParams
	err = json.Unmarshal(data, &deserialized)
	assert.NoError(t, err)

	assert.Equal(t, original, deserialized)
}

func TestWritePermissionsParamsJSONSerialization(t *testing.T) {
	original := WritePermissionsParams{
		FilePath: "/test/path.txt",
		Diff:     "--- old\n+++ new\n@@ -1,1 +1,1 @@\n-old\n+new",
	}

	// Serialize
	data, err := json.Marshal(original)
	assert.NoError(t, err)

	// Deserialize
	var deserialized WritePermissionsParams
	err = json.Unmarshal(data, &deserialized)
	assert.NoError(t, err)

	assert.Equal(t, original, deserialized)
}

func TestWriteResponseMetadataJSONSerialization(t *testing.T) {
	original := WriteResponseMetadata{
		Diff:      "--- old\n+++ new",
		Additions: 10,
		Removals:  5,
	}

	// Serialize
	data, err := json.Marshal(original)
	assert.NoError(t, err)

	// Deserialize
	var deserialized WriteResponseMetadata
	err = json.Unmarshal(data, &deserialized)
	assert.NoError(t, err)

	assert.Equal(t, original, deserialized)
}

// Test NewWriteTool constructor
func TestNewWriteTool(t *testing.T) {
	mockPermissions := &writeToolMockPermissionService{}
	mockFiles := &writeToolMockHistoryService{}

	tool := NewWriteTool(mockPermissions, mockFiles)
	assert.NotNil(t, tool)

	// Verify it implements BaseTool interface
	var _ interfaces.BaseTool = tool
}

// Test tool Info method
func TestWriteTool_Info(t *testing.T) {
	mockPermissions := &writeToolMockPermissionService{}
	mockFiles := &writeToolMockHistoryService{}

	tool := NewWriteTool(mockPermissions, mockFiles)
	info := tool.Info()

	assert.Equal(t, WriteToolName, info.Name)
	assert.Equal(t, "write", info.Name)
	assert.NotEmpty(t, info.Description)
	assert.Contains(t, info.Required, "file_path")
	assert.Contains(t, info.Required, "content")

	// Check parameters structure
	assert.Contains(t, info.Parameters, "file_path")
	assert.Contains(t, info.Parameters, "content")

	filePathParam := info.Parameters["file_path"].(map[string]any)
	assert.Equal(t, "string", filePathParam["type"])
	assert.Contains(t, filePathParam["description"], "path")

	contentParam := info.Parameters["content"].(map[string]any)
	assert.Equal(t, "string", contentParam["type"])
	assert.Contains(t, contentParam["description"], "content")
}

// Test Run method with invalid JSON
func TestWriteTool_Run_InvalidJSON(t *testing.T) {
	mockPermissions := &writeToolMockPermissionService{}
	mockFiles := &writeToolMockHistoryService{}
	tool := NewWriteTool(mockPermissions, mockFiles)

	ctx := createWriteToolTestContext("session123", "message456", "/tmp/test")
	call := interfaces.ToolCall{
		ID:    "call123",
		Name:  "write",
		Input: "invalid json",
	}

	response, err := tool.Run(ctx, call)
	assert.NoError(t, err)
	assert.True(t, response.IsError)
	assert.Contains(t, response.Content, "error parsing parameters")
}

// Test Run method with missing file_path
func TestWriteTool_Run_MissingFilePath(t *testing.T) {
	mockPermissions := &writeToolMockPermissionService{}
	mockFiles := &writeToolMockHistoryService{}
	tool := NewWriteTool(mockPermissions, mockFiles)

	ctx := createWriteToolTestContext("session123", "message456", "/tmp/test")
	params := WriteParams{
		Content: "test content",
	}
	input, _ := json.Marshal(params)
	call := interfaces.ToolCall{
		ID:    "call123",
		Name:  "write",
		Input: string(input),
	}

	response, err := tool.Run(ctx, call)
	assert.NoError(t, err)
	assert.True(t, response.IsError)
	assert.Contains(t, response.Content, "file_path is required")
}

// Test Run method with missing content
func TestWriteTool_Run_MissingContent(t *testing.T) {
	mockPermissions := &writeToolMockPermissionService{}
	mockFiles := &writeToolMockHistoryService{}
	tool := NewWriteTool(mockPermissions, mockFiles)

	ctx := createWriteToolTestContext("session123", "message456", "/tmp/test")
	params := WriteParams{
		FilePath: "/test/file.txt",
	}
	input, _ := json.Marshal(params)
	call := interfaces.ToolCall{
		ID:    "call123",
		Name:  "write",
		Input: string(input),
	}

	response, err := tool.Run(ctx, call)
	assert.NoError(t, err)
	assert.True(t, response.IsError)
	assert.Contains(t, response.Content, "content is required")
}

// Test Run method with missing session storage context
func TestWriteTool_Run_MissingSessionStorage(t *testing.T) {
	mockPermissions := &writeToolMockPermissionService{}
	mockFiles := &writeToolMockHistoryService{}
	tool := NewWriteTool(mockPermissions, mockFiles)

	ctx := context.Background()
	ctx = context.WithValue(ctx, interfaces.SessionIDContextKey, "session123")
	ctx = context.WithValue(ctx, interfaces.MessageIDContextKey, "message456")

	params := WriteParams{
		FilePath: "/test/file.txt",
		Content:  "test content",
	}
	input, _ := json.Marshal(params)
	call := interfaces.ToolCall{
		ID:    "call123",
		Name:  "write",
		Input: string(input),
	}

	_, err := tool.Run(ctx, call)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "session storage directory not found")
}

// Test Run method with missing session ID
func TestWriteTool_Run_MissingSessionID(t *testing.T) {
	mockPermissions := &writeToolMockPermissionService{}
	mockFiles := &writeToolMockHistoryService{}
	tool := NewWriteTool(mockPermissions, mockFiles)

	tempDir := setupWriteToolTestDir(t)
	ctx := context.Background()
	ctx = context.WithValue(ctx, interfaces.MessageIDContextKey, "message456")
	ctx = context.WithValue(ctx, interfaces.SessionStorageContextKey, tempDir)

	params := WriteParams{
		FilePath: "test.txt",
		Content:  "test content",
	}
	input, _ := json.Marshal(params)
	call := interfaces.ToolCall{
		ID:    "call123",
		Name:  "write",
		Input: string(input),
	}

	_, err := tool.Run(ctx, call)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "session_id and message_id are required")
}

// Test Run method attempting to write to directory
func TestWriteTool_Run_WriteToDirectory(t *testing.T) {
	mockPermissions := &writeToolMockPermissionService{}
	mockFiles := &writeToolMockHistoryService{}
	tool := NewWriteTool(mockPermissions, mockFiles)

	tempDir := setupWriteToolTestDir(t)
	subDir := filepath.Join(tempDir, "subdir")
	err := os.MkdirAll(subDir, 0755)
	assert.NoError(t, err)

	ctx := createWriteToolTestContext("session123", "message456", tempDir)
	params := WriteParams{
		FilePath: subDir,
		Content:  "test content",
	}
	input, _ := json.Marshal(params)
	call := interfaces.ToolCall{
		ID:    "call123",
		Name:  "write",
		Input: string(input),
	}

	response, err := tool.Run(ctx, call)
	assert.NoError(t, err)
	assert.True(t, response.IsError)
	assert.Contains(t, response.Content, "Path is a directory, not a file")
}

// Test Run method with file modified after last read
func TestWriteTool_Run_FileModifiedAfterRead(t *testing.T) {
	clearFileRecords() // Clear any existing records

	mockPermissions := &writeToolMockPermissionService{}
	mockFiles := &writeToolMockHistoryService{}
	tool := NewWriteTool(mockPermissions, mockFiles)

	tempDir := setupWriteToolTestDir(t)
	filePath := filepath.Join(tempDir, "test.txt")

	// Create a file
	err := os.WriteFile(filePath, []byte("original content"), 0644)
	assert.NoError(t, err)

	// Record a read time in the past
	recordFileRead(filePath)
	time.Sleep(10 * time.Millisecond) // Ensure some time passes

	// Modify the file after the read
	err = os.WriteFile(filePath, []byte("modified content"), 0644)
	assert.NoError(t, err)

	ctx := createWriteToolTestContext("session123", "message456", tempDir)
	params := WriteParams{
		FilePath: filePath,
		Content:  "new content",
	}
	input, _ := json.Marshal(params)
	call := interfaces.ToolCall{
		ID:    "call123",
		Name:  "write",
		Input: string(input),
	}

	response, err := tool.Run(ctx, call)
	assert.NoError(t, err)
	assert.True(t, response.IsError)
	assert.Contains(t, response.Content, "has been modified since it was last read")
}

// Test Run method with same content (no changes)
func TestWriteTool_Run_SameContent(t *testing.T) {
	clearFileRecords()

	mockPermissions := &writeToolMockPermissionService{}
	mockFiles := &writeToolMockHistoryService{}
	tool := NewWriteTool(mockPermissions, mockFiles)

	tempDir := setupWriteToolTestDir(t)
	filePath := filepath.Join(tempDir, "test.txt")
	content := "test content"

	// Create a file with content
	err := os.WriteFile(filePath, []byte(content), 0644)
	assert.NoError(t, err)

	// Record recent read
	recordFileRead(filePath)

	ctx := createWriteToolTestContext("session123", "message456", tempDir)
	params := WriteParams{
		FilePath: filePath,
		Content:  content, // Same content
	}
	input, _ := json.Marshal(params)
	call := interfaces.ToolCall{
		ID:    "call123",
		Name:  "write",
		Input: string(input),
	}

	response, err := tool.Run(ctx, call)
	assert.NoError(t, err)
	assert.True(t, response.IsError)
	assert.Contains(t, response.Content, "already contains the exact content")
}

// Test Run method with permission denied
func TestWriteTool_Run_PermissionDenied(t *testing.T) {
	clearFileRecords()

	mockPermissions := &writeToolMockPermissionService{}
	mockFiles := &writeToolMockHistoryService{}
	tool := NewWriteTool(mockPermissions, mockFiles)

	tempDir := setupWriteToolTestDir(t)
	filePath := filepath.Join(tempDir, "test.txt")

	// Mock permission denial
	mockPermissions.On("Request", mock.AnythingOfType("permission.CreatePermissionRequest")).Return(false)

	ctx := createWriteToolTestContext("session123", "message456", tempDir)
	params := WriteParams{
		FilePath: filePath,
		Content:  "test content",
	}
	input, _ := json.Marshal(params)
	call := interfaces.ToolCall{
		ID:    "call123",
		Name:  "write",
		Input: string(input),
	}

	_, err := tool.Run(ctx, call)
	assert.Error(t, err)
	assert.Equal(t, permission.ErrorPermissionDenied, err)

	mockPermissions.AssertExpectations(t)
}

// Test successful file write with new file
func TestWriteTool_Run_SuccessfulNewFile(t *testing.T) {
	clearFileRecords()

	mockPermissions := &writeToolMockPermissionService{}
	mockFiles := &writeToolMockHistoryService{}
	tool := NewWriteTool(mockPermissions, mockFiles)

	tempDir := setupWriteToolTestDir(t)
	filePath := filepath.Join(tempDir, "test.txt")
	content := "test content"

	// Mock permission granted
	mockPermissions.On("Request", mock.AnythingOfType("permission.CreatePermissionRequest")).Return(true)

	// Mock history service calls for new file
	mockFiles.On("GetByPathAndSession", mock.Anything, filePath, "session123").Return(history.File{}, fmt.Errorf("not found"))
	mockFiles.On("Create", mock.Anything, "session123", filePath, "").Return(history.File{ID: "file123"}, nil)
	mockFiles.On("CreateVersion", mock.Anything, "session123", filePath, content).Return(history.File{ID: "version123"}, nil)

	ctx := createWriteToolTestContext("session123", "message456", tempDir)
	params := WriteParams{
		FilePath: filePath,
		Content:  content,
	}
	input, _ := json.Marshal(params)
	call := interfaces.ToolCall{
		ID:    "call123",
		Name:  "write",
		Input: string(input),
	}

	response, err := tool.Run(ctx, call)
	assert.NoError(t, err)
	assert.False(t, response.IsError)
	assert.Contains(t, response.Content, "File successfully written")

	// Verify file was actually written
	written, err := os.ReadFile(filePath)
	assert.NoError(t, err)
	assert.Equal(t, content, string(written))

	// Verify metadata
	assert.NotEmpty(t, response.Metadata)
	var metadata WriteResponseMetadata
	err = json.Unmarshal([]byte(response.Metadata), &metadata)
	assert.NoError(t, err)
	assert.Greater(t, metadata.Additions, 0)

	mockPermissions.AssertExpectations(t)
	mockFiles.AssertExpectations(t)
}

// Test successful file write with existing file
func TestWriteTool_Run_SuccessfulExistingFile(t *testing.T) {
	clearFileRecords()

	mockPermissions := &writeToolMockPermissionService{}
	mockFiles := &writeToolMockHistoryService{}
	tool := NewWriteTool(mockPermissions, mockFiles)

	tempDir := setupWriteToolTestDir(t)
	filePath := filepath.Join(tempDir, "test.txt")
	oldContent := "old content"
	newContent := "new content"

	// Create existing file
	err := os.WriteFile(filePath, []byte(oldContent), 0644)
	assert.NoError(t, err)
	recordFileRead(filePath) // Record that file was read

	// Mock permission granted
	mockPermissions.On("Request", mock.AnythingOfType("permission.CreatePermissionRequest")).Return(true)

	// Mock history service calls for existing file
	existingFile := history.File{
		ID:      "file123",
		Content: oldContent,
	}
	mockFiles.On("GetByPathAndSession", mock.Anything, filePath, "session123").Return(existingFile, nil)
	mockFiles.On("CreateVersion", mock.Anything, "session123", filePath, newContent).Return(history.File{ID: "version123"}, nil)

	ctx := createWriteToolTestContext("session123", "message456", tempDir)
	params := WriteParams{
		FilePath: filePath,
		Content:  newContent,
	}
	input, _ := json.Marshal(params)
	call := interfaces.ToolCall{
		ID:    "call123",
		Name:  "write",
		Input: string(input),
	}

	response, err := tool.Run(ctx, call)
	assert.NoError(t, err)
	assert.False(t, response.IsError)
	assert.Contains(t, response.Content, "File successfully written")

	// Verify file was actually written
	written, err := os.ReadFile(filePath)
	assert.NoError(t, err)
	assert.Equal(t, newContent, string(written))

	mockPermissions.AssertExpectations(t)
	mockFiles.AssertExpectations(t)
}

// Test relative path handling
func TestWriteTool_Run_RelativePath(t *testing.T) {
	clearFileRecords()

	mockPermissions := &writeToolMockPermissionService{}
	mockFiles := &writeToolMockHistoryService{}
	tool := NewWriteTool(mockPermissions, mockFiles)

	tempDir := setupWriteToolTestDir(t)
	relativePath := "test.txt"
	content := "test content"

	// Mock permission granted
	mockPermissions.On("Request", mock.AnythingOfType("permission.CreatePermissionRequest")).Return(true)

	// Mock history service calls
	absolutePath := filepath.Join(tempDir, relativePath)
	mockFiles.On("GetByPathAndSession", mock.Anything, absolutePath, "session123").Return(history.File{}, fmt.Errorf("not found"))
	mockFiles.On("Create", mock.Anything, "session123", absolutePath, "").Return(history.File{ID: "file123"}, nil)
	mockFiles.On("CreateVersion", mock.Anything, "session123", absolutePath, content).Return(history.File{ID: "version123"}, nil)

	ctx := createWriteToolTestContext("session123", "message456", tempDir)
	params := WriteParams{
		FilePath: relativePath, // Relative path
		Content:  content,
	}
	input, _ := json.Marshal(params)
	call := interfaces.ToolCall{
		ID:    "call123",
		Name:  "write",
		Input: string(input),
	}

	response, err := tool.Run(ctx, call)
	assert.NoError(t, err)
	assert.False(t, response.IsError)

	// Verify file was written at the correct absolute path
	written, err := os.ReadFile(absolutePath)
	assert.NoError(t, err)
	assert.Equal(t, content, string(written))

	mockPermissions.AssertExpectations(t)
	mockFiles.AssertExpectations(t)
}

// Test directory creation
func TestWriteTool_Run_CreateDirectories(t *testing.T) {
	clearFileRecords()

	mockPermissions := &writeToolMockPermissionService{}
	mockFiles := &writeToolMockHistoryService{}
	tool := NewWriteTool(mockPermissions, mockFiles)

	tempDir := setupWriteToolTestDir(t)
	filePath := filepath.Join(tempDir, "deep", "nested", "path", "test.txt")
	content := "test content"

	// Mock permission granted
	mockPermissions.On("Request", mock.AnythingOfType("permission.CreatePermissionRequest")).Return(true)

	// Mock history service calls
	mockFiles.On("GetByPathAndSession", mock.Anything, filePath, "session123").Return(history.File{}, fmt.Errorf("not found"))
	mockFiles.On("Create", mock.Anything, "session123", filePath, "").Return(history.File{ID: "file123"}, nil)
	mockFiles.On("CreateVersion", mock.Anything, "session123", filePath, content).Return(history.File{ID: "version123"}, nil)

	ctx := createWriteToolTestContext("session123", "message456", tempDir)
	params := WriteParams{
		FilePath: filePath,
		Content:  content,
	}
	input, _ := json.Marshal(params)
	call := interfaces.ToolCall{
		ID:    "call123",
		Name:  "write",
		Input: string(input),
	}

	response, err := tool.Run(ctx, call)
	assert.NoError(t, err)
	assert.False(t, response.IsError)

	// Verify directories were created
	assert.DirExists(t, filepath.Dir(filePath))

	// Verify file was written
	written, err := os.ReadFile(filePath)
	assert.NoError(t, err)
	assert.Equal(t, content, string(written))

	mockPermissions.AssertExpectations(t)
	mockFiles.AssertExpectations(t)
}

// Test with history service errors
func TestWriteTool_Run_HistoryServiceErrors(t *testing.T) {
	clearFileRecords()

	mockPermissions := &writeToolMockPermissionService{}
	mockFiles := &writeToolMockHistoryService{}
	tool := NewWriteTool(mockPermissions, mockFiles)

	tempDir := setupWriteToolTestDir(t)
	filePath := filepath.Join(tempDir, "test.txt")
	content := "test content"

	// Mock permission granted
	mockPermissions.On("Request", mock.AnythingOfType("permission.CreatePermissionRequest")).Return(true)

	// Mock history service calls with error on Create
	mockFiles.On("GetByPathAndSession", mock.Anything, filePath, "session123").Return(history.File{}, fmt.Errorf("not found"))
	mockFiles.On("Create", mock.Anything, "session123", filePath, "").Return(history.File{}, fmt.Errorf("create error"))

	ctx := createWriteToolTestContext("session123", "message456", tempDir)
	params := WriteParams{
		FilePath: filePath,
		Content:  content,
	}
	input, _ := json.Marshal(params)
	call := interfaces.ToolCall{
		ID:    "call123",
		Name:  "write",
		Input: string(input),
	}

	_, err := tool.Run(ctx, call)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "error creating file history")

	mockPermissions.AssertExpectations(t)
	mockFiles.AssertExpectations(t)
}

// Test file manually changed scenario
func TestWriteTool_Run_FileManuallyChanged(t *testing.T) {
	clearFileRecords()

	mockPermissions := &writeToolMockPermissionService{}
	mockFiles := &writeToolMockHistoryService{}
	tool := NewWriteTool(mockPermissions, mockFiles)

	tempDir := setupWriteToolTestDir(t)
	filePath := filepath.Join(tempDir, "test.txt")
	oldContent := "old content"
	manualContent := "manually changed"
	newContent := "new content"

	// Create existing file
	err := os.WriteFile(filePath, []byte(manualContent), 0644)
	assert.NoError(t, err)
	recordFileRead(filePath)

	// Mock permission granted
	mockPermissions.On("Request", mock.AnythingOfType("permission.CreatePermissionRequest")).Return(true)

	// Mock history service - file exists but content differs (manual change)
	existingFile := history.File{
		ID:      "file123",
		Content: oldContent, // Different from what's on disk
	}
	mockFiles.On("GetByPathAndSession", mock.Anything, filePath, "session123").Return(existingFile, nil)
	// Should create intermediate version for manual change
	mockFiles.On("CreateVersion", mock.Anything, "session123", filePath, manualContent).Return(history.File{ID: "intermediate123"}, nil)
	// Should create version for new content
	mockFiles.On("CreateVersion", mock.Anything, "session123", filePath, newContent).Return(history.File{ID: "new123"}, nil)

	ctx := createWriteToolTestContext("session123", "message456", tempDir)
	params := WriteParams{
		FilePath: filePath,
		Content:  newContent,
	}
	input, _ := json.Marshal(params)
	call := interfaces.ToolCall{
		ID:    "call123",
		Name:  "write",
		Input: string(input),
	}

	response, err := tool.Run(ctx, call)
	assert.NoError(t, err)
	assert.False(t, response.IsError)

	// Verify file was written with new content
	written, err := os.ReadFile(filePath)
	assert.NoError(t, err)
	assert.Equal(t, newContent, string(written))

	mockPermissions.AssertExpectations(t)
	mockFiles.AssertExpectations(t)
}

// Test edge cases
func TestWriteTool_Run_EdgeCases(t *testing.T) {
	tests := []struct {
		name     string
		content  string
		expected string
	}{
		{
			name:     "empty content",
			content:  "",
			expected: "",
		},
		{
			name:     "unicode content",
			content:  "Hello 世界 🌍",
			expected: "Hello 世界 🌍",
		},
		{
			name:     "multiline content",
			content:  "line 1\nline 2\nline 3",
			expected: "line 1\nline 2\nline 3",
		},
		{
			name:     "content with special characters",
			content:  "特殊字符\t制表符\r\n换行符",
			expected: "特殊字符\t制表符\r\n换行符",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			clearFileRecords()

			mockPermissions := &writeToolMockPermissionService{}
			mockFiles := &writeToolMockHistoryService{}
			tool := NewWriteTool(mockPermissions, mockFiles)

			tempDir := setupWriteToolTestDir(t)
			filePath := filepath.Join(tempDir, "test.txt")

			// Mock permission granted
			mockPermissions.On("Request", mock.AnythingOfType("permission.CreatePermissionRequest")).Return(true)

			// Mock history service calls
			mockFiles.On("GetByPathAndSession", mock.Anything, filePath, "session123").Return(history.File{}, fmt.Errorf("not found"))
			mockFiles.On("Create", mock.Anything, "session123", filePath, "").Return(history.File{ID: "file123"}, nil)
			mockFiles.On("CreateVersion", mock.Anything, "session123", filePath, tt.content).Return(history.File{ID: "version123"}, nil)

			ctx := createWriteToolTestContext("session123", "message456", tempDir)
			params := WriteParams{
				FilePath: filePath,
				Content:  tt.content,
			}
			input, _ := json.Marshal(params)
			call := interfaces.ToolCall{
				ID:    "call123",
				Name:  "write",
				Input: string(input),
			}

			response, err := tool.Run(ctx, call)
			assert.NoError(t, err)
			assert.False(t, response.IsError)

			// Verify file content
			written, err := os.ReadFile(filePath)
			assert.NoError(t, err)
			assert.Equal(t, tt.expected, string(written))

			mockPermissions.AssertExpectations(t)
			mockFiles.AssertExpectations(t)
		})
	}
}

// Test interface compliance
func TestWriteTool_InterfaceCompliance(t *testing.T) {
	mockPermissions := &writeToolMockPermissionService{}
	mockFiles := &writeToolMockHistoryService{}
	tool := NewWriteTool(mockPermissions, mockFiles)

	// Verify it implements BaseTool interface
	var _ interfaces.BaseTool = tool

	// Verify Info method returns correct structure
	info := tool.Info()
	assert.NotEmpty(t, info.Name)
	assert.NotEmpty(t, info.Description)
	assert.NotNil(t, info.Parameters)
	assert.NotEmpty(t, info.Required)
}

// Test WriteToolName constant
func TestWriteToolName(t *testing.T) {
	assert.Equal(t, "write", WriteToolName)
}

// Test diff generation in metadata
func TestWriteTool_DiffGeneration(t *testing.T) {
	clearFileRecords()

	mockPermissions := &writeToolMockPermissionService{}
	mockFiles := &writeToolMockHistoryService{}
	tool := NewWriteTool(mockPermissions, mockFiles)

	tempDir := setupWriteToolTestDir(t)
	filePath := filepath.Join(tempDir, "test.txt")
	content := "line1\nline2\nline3"

	// Mock permission granted
	mockPermissions.On("Request", mock.AnythingOfType("permission.CreatePermissionRequest")).Return(true)

	// Mock history service calls
	mockFiles.On("GetByPathAndSession", mock.Anything, filePath, "session123").Return(history.File{}, fmt.Errorf("not found"))
	mockFiles.On("Create", mock.Anything, "session123", filePath, "").Return(history.File{ID: "file123"}, nil)
	mockFiles.On("CreateVersion", mock.Anything, "session123", filePath, content).Return(history.File{ID: "version123"}, nil)

	ctx := createWriteToolTestContext("session123", "message456", tempDir)
	params := WriteParams{
		FilePath: filePath,
		Content:  content,
	}
	input, _ := json.Marshal(params)
	call := interfaces.ToolCall{
		ID:    "call123",
		Name:  "write",
		Input: string(input),
	}

	response, err := tool.Run(ctx, call)
	assert.NoError(t, err)
	assert.False(t, response.IsError)

	// Verify metadata contains diff
	assert.NotEmpty(t, response.Metadata)
	var metadata WriteResponseMetadata
	err = json.Unmarshal([]byte(response.Metadata), &metadata)
	assert.NoError(t, err)

	assert.Contains(t, metadata.Diff, "---")
	assert.Contains(t, metadata.Diff, "+++")
	assert.Equal(t, 3, metadata.Additions) // 3 lines added
	assert.Equal(t, 0, metadata.Removals)  // 0 lines removed (new file)

	mockPermissions.AssertExpectations(t)
	mockFiles.AssertExpectations(t)
}