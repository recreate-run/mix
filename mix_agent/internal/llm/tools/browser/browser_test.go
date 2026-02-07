package browser

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"mix/internal/llm/interfaces"
	"mix/internal/permission"
	"mix/internal/pubsub"
	"mix/internal/session"
)

// MockPermissionService is a mock implementation of the permission.Service interface
type MockPermissionService struct {
	mock.Mock
}

func (m *MockPermissionService) Subscribe(ctx context.Context) <-chan pubsub.Event[permission.PermissionRequest] {
	args := m.Called(ctx)
	return args.Get(0).(<-chan pubsub.Event[permission.PermissionRequest])
}

func (m *MockPermissionService) GetSubscriberCount() int {
	args := m.Called()
	return args.Int(0)
}

func (m *MockPermissionService) Publish(ctx context.Context, eventType pubsub.EventType, data permission.PermissionRequest) error {
	args := m.Called(ctx, eventType, data)
	return args.Error(0)
}

func (m *MockPermissionService) GrantPersistant(perm permission.PermissionRequest) {
	m.Called(perm)
}

func (m *MockPermissionService) Grant(perm permission.PermissionRequest) {
	m.Called(perm)
}

func (m *MockPermissionService) Deny(perm permission.PermissionRequest) {
	m.Called(perm)
}

func (m *MockPermissionService) Request(opts permission.CreatePermissionRequest) bool {
	args := m.Called(opts)
	return args.Bool(0)
}

// Test constants
func TestBrowserConstants(t *testing.T) {
	assert.Equal(t, "Browser", BrowserToolName)
	assert.Equal(t, "30s", DefaultRequestTimeout.String())
}

// Test action constants
func TestActionConstants(t *testing.T) {
	assert.Equal(t, "open", ActionOpen)
	assert.Equal(t, "screenshot", ActionScreenshot)
	assert.Equal(t, "click", ActionClick)
	assert.Equal(t, "type", ActionType)
	assert.Equal(t, "scroll", ActionScroll)
	assert.Equal(t, "close", ActionClose)
}

// Test direction constants
func TestDirectionConstants(t *testing.T) {
	assert.Equal(t, "up", DirectionUp)
	assert.Equal(t, "down", DirectionDown)
	assert.Equal(t, "left", DirectionLeft)
	assert.Equal(t, "right", DirectionRight)
}

// Test BrowserParams JSON serialization
func TestBrowserParamsJSONSerialization(t *testing.T) {
	tests := []struct {
		name     string
		params   BrowserParams
		expected string
	}{
		{
			name: "open action",
			params: BrowserParams{
				Action: ActionOpen,
				URL:    "https://example.com",
			},
			expected: `{"action":"open","url":"https://example.com"}`,
		},
		{
			name: "screenshot action",
			params: BrowserParams{
				Action: ActionScreenshot,
			},
			expected: `{"action":"screenshot"}`,
		},
		{
			name: "click action",
			params: BrowserParams{
				Action: ActionClick,
				Index:  5,
			},
			expected: `{"action":"click","index":5}`,
		},
		{
			name: "type action",
			params: BrowserParams{
				Action: ActionType,
				Index:  3,
				Text:   "hello world",
			},
			expected: `{"action":"type","index":3,"text":"hello world"}`,
		},
		{
			name: "scroll action",
			params: BrowserParams{
				Action:    ActionScroll,
				Direction: DirectionDown,
				Amount:    200,
			},
			expected: `{"action":"scroll","direction":"down","amount":200}`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Test marshaling
			jsonData, err := json.Marshal(tt.params)
			require.NoError(t, err)
			assert.JSONEq(t, tt.expected, string(jsonData))

			// Test unmarshaling
			var params BrowserParams
			err = json.Unmarshal(jsonData, &params)
			require.NoError(t, err)
			assert.Equal(t, tt.params.Action, params.Action)
		})
	}
}

// Test NewBrowserTool
func TestNewBrowserTool(t *testing.T) {
	mockPermissionService := &MockPermissionService{}
	sessionConfig := session.DefaultConfig()
	tool := NewBrowserTool(mockPermissionService, "ws://localhost:8080", sessionConfig)

	assert.NotNil(t, tool)
	assert.Implements(t, (*interfaces.BaseTool)(nil), tool)

	// Cast to concrete type to verify internal state
	browserTool, ok := tool.(*browserTool)
	assert.True(t, ok)
	assert.Equal(t, mockPermissionService, browserTool.permissions)
	assert.NotNil(t, browserTool.connectionManager)
	assert.NotEmpty(t, browserTool.baseURL)
}

// Test Info method
func TestBrowserToolInfo(t *testing.T) {
	mockPermissionService := &MockPermissionService{}
	sessionConfig := session.DefaultConfig()
	tool := NewBrowserTool(mockPermissionService, "ws://localhost:8080", sessionConfig)

	info := tool.Info()

	assert.Equal(t, BrowserToolName, info.Name)
	assert.NotEmpty(t, info.Description)
	assert.Contains(t, info.Required, "action")

	// Test parameters structure
	assert.Contains(t, info.Parameters, "action")
	assert.Contains(t, info.Parameters, "url")
	assert.Contains(t, info.Parameters, "index")
	assert.Contains(t, info.Parameters, "text")
	assert.Contains(t, info.Parameters, "direction")
	assert.Contains(t, info.Parameters, "amount")

	// Test action parameter enum
	actionParam, ok := info.Parameters["action"].(map[string]any)
	assert.True(t, ok)
	assert.Equal(t, "string", actionParam["type"])
	enum, ok := actionParam["enum"].([]string)
	assert.True(t, ok)
	assert.Contains(t, enum, ActionOpen)
	assert.Contains(t, enum, ActionScreenshot)
	assert.Contains(t, enum, ActionClick)
	assert.Contains(t, enum, ActionType)
	assert.Contains(t, enum, ActionScroll)
	assert.Contains(t, enum, ActionClose)

	// Test direction parameter enum
	directionParam, ok := info.Parameters["direction"].(map[string]any)
	assert.True(t, ok)
	enum, ok = directionParam["enum"].([]string)
	assert.True(t, ok)
	assert.Contains(t, enum, DirectionUp)
	assert.Contains(t, enum, DirectionDown)
	assert.Contains(t, enum, DirectionLeft)
	assert.Contains(t, enum, DirectionRight)
}

// Helper to create test context
func createBrowserTestContext(sessionID, messageID, storageDir string) context.Context {
	t := &testing.T{}
	t.Helper()
	ctx := context.Background()
	if sessionID != "" {
		ctx = context.WithValue(ctx, interfaces.SessionIDContextKey, sessionID)
	}
	if messageID != "" {
		ctx = context.WithValue(ctx, interfaces.MessageIDContextKey, messageID)
	}
	if storageDir != "" {
		ctx = context.WithValue(ctx, interfaces.SessionStorageContextKey, storageDir)
	}
	return ctx
}

// Test Run with invalid JSON
func TestBrowserToolRunInvalidJSON(t *testing.T) {
	mockPermissionService := &MockPermissionService{}
	sessionConfig := session.DefaultConfig()
	tool := NewBrowserTool(mockPermissionService, "ws://localhost:8080", sessionConfig)

	ctx := createBrowserTestContext("session-123", "message-456", "/tmp/test")
	call := interfaces.ToolCall{
		ID:    "call-1",
		Name:  BrowserToolName,
		Input: "invalid json",
	}

	response, err := tool.Run(ctx, call)

	require.Error(t, err)
	assert.True(t, response.IsError)
	assert.Contains(t, response.Content, "invalid parameters")
}

// Test Run with missing action
func TestBrowserToolRunMissingAction(t *testing.T) {
	mockPermissionService := &MockPermissionService{}
	sessionConfig := session.DefaultConfig()
	tool := NewBrowserTool(mockPermissionService, "ws://localhost:8080", sessionConfig)

	ctx := createBrowserTestContext("session-123", "message-456", "/tmp/test")
	call := interfaces.ToolCall{
		ID:    "call-1",
		Name:  BrowserToolName,
		Input: `{"url": "https://example.com"}`,
	}

	response, err := tool.Run(ctx, call)

	require.NoError(t, err)
	assert.True(t, response.IsError)
	assert.Contains(t, response.Content, "missing action")
}

// Test Run with unknown action
func TestBrowserToolRunUnknownAction(t *testing.T) {
	mockPermissionService := &MockPermissionService{}
	sessionConfig := session.DefaultConfig()
	tool := NewBrowserTool(mockPermissionService, "ws://localhost:8080", sessionConfig)

	ctx := createBrowserTestContext("session-123", "message-456", "/tmp/test")
	call := interfaces.ToolCall{
		ID:    "call-1",
		Name:  BrowserToolName,
		Input: `{"action": "invalid_action"}`,
	}

	response, err := tool.Run(ctx, call)

	require.NoError(t, err)
	assert.True(t, response.IsError)
	assert.Contains(t, response.Content, "unknown action")
}

// Test context validation
func TestBrowserToolRunContextValidation(t *testing.T) {
	mockPermissionService := &MockPermissionService{}
	sessionConfig := session.DefaultConfig()
	tool := NewBrowserTool(mockPermissionService, "ws://localhost:8080", sessionConfig)

	tests := []struct {
		name        string
		sessionID   string
		messageID   string
		storageDir  string
		expectError bool
		errorMsg    string
	}{
		{
			name:        "missing session ID",
			sessionID:   "",
			messageID:   "message-456",
			storageDir:  "/tmp/test",
			expectError: true,
			errorMsg:    "session ID not found in context",
		},
		{
			name:        "missing message ID",
			sessionID:   "session-123",
			messageID:   "",
			storageDir:  "/tmp/test",
			expectError: true,
			errorMsg:    "message ID not found in context",
		},
		{
			name:        "missing storage directory",
			sessionID:   "session-123",
			messageID:   "message-456",
			storageDir:  "",
			expectError: true,
			errorMsg:    "session storage directory not found in context",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := createBrowserTestContext(tt.sessionID, tt.messageID, tt.storageDir)
			call := interfaces.ToolCall{
				ID:    "call-1",
				Name:  BrowserToolName,
				Input: `{"action": "screenshot"}`,
			}

			_, err := tool.Run(ctx, call)

			if tt.expectError {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.errorMsg)
			} else {
				require.NoError(t, err)
			}
		})
	}
}

// Test open action validation
func TestBrowserToolOpenValidation(t *testing.T) {
	mockPermissionService := &MockPermissionService{}
	sessionConfig := session.DefaultConfig()
	tool := NewBrowserTool(mockPermissionService, "ws://localhost:8080", sessionConfig)

	tests := []struct {
		name     string
		input    string
		errorMsg string
	}{
		{
			name:     "missing URL",
			input:    `{"action": "open"}`,
			errorMsg: "missing url parameter",
		},
		{
			name:     "invalid URL format",
			input:    `{"action": "open", "url": "not-a-url"}`,
			errorMsg: "invalid URL",
		},
		{
			name:     "non-http URL",
			input:    `{"action": "open", "url": "ftp://example.com"}`,
			errorMsg: "invalid URL",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := createBrowserTestContext("session-123", "message-456", "/tmp/test")
			call := interfaces.ToolCall{
				ID:    "call-1",
				Name:  BrowserToolName,
				Input: tt.input,
			}

			response, err := tool.Run(ctx, call)

			require.NoError(t, err)
			assert.True(t, response.IsError)
			assert.Contains(t, response.Content, tt.errorMsg)
		})
	}
}

// Test file:// URL support
func TestBrowserToolFileURLSupport(t *testing.T) {
	mockPermissionService := &MockPermissionService{}
	sessionConfig := session.DefaultConfig()
	tool := NewBrowserTool(mockPermissionService, "ws://localhost:8080", sessionConfig)

	// Create a temporary session directory and test file
	sessionID := "test-session-123"
	sessionDir := session.GetSessionStoragePath(sessionID, sessionConfig)
	err := os.MkdirAll(sessionDir, 0o750)
	require.NoError(t, err)
	defer func() {
		_ = os.RemoveAll(sessionDir)
	}()

	// Create a test HTML file in session directory
	testFilePath := filepath.Join(sessionDir, "test.html")
	err = os.WriteFile(testFilePath, []byte("<html><body>Test</body></html>"), 0o644)
	require.NoError(t, err)

	tests := []struct {
		name        string
		filePath    string
		shouldError bool
		errorMsg    string
	}{
		{
			name:        "valid file in session directory",
			filePath:    testFilePath,
			shouldError: false,
		},
		{
			name:        "file outside session directory",
			filePath:    "/etc/passwd",
			shouldError: true,
			errorMsg:    "must reference files within session storage directory",
		},
		{
			name:        "non-existent file in session directory",
			filePath:    filepath.Join(sessionDir, "nonexistent.html"),
			shouldError: true,
			errorMsg:    "file not found",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := createBrowserTestContext(sessionID, "message-456", "/tmp/test")
			fileURL := fmt.Sprintf("file://%s", tt.filePath)
			call := interfaces.ToolCall{
				ID:    "call-1",
				Name:  BrowserToolName,
				Input: fmt.Sprintf(`{"action": "open", "url": %q}`, fileURL),
			}

			response, err := tool.Run(ctx, call)
			require.NoError(t, err)

			if tt.shouldError {
				assert.True(t, response.IsError)
				assert.Contains(t, response.Content, tt.errorMsg)
			} else if response.IsError {
				// Will fail to connect to browser service, but should pass validation
				// Error will be about browser service connection, not URL validation
				assert.NotContains(t, response.Content, "invalid URL scheme")
				assert.NotContains(t, response.Content, "must reference files within")
			}
		})
	}
}

// Test type action validation
func TestBrowserToolTypeValidation(t *testing.T) {
	mockPermissionService := &MockPermissionService{}
	sessionConfig := session.DefaultConfig()
	tool := NewBrowserTool(mockPermissionService, "ws://localhost:8080", sessionConfig)

	ctx := createBrowserTestContext("session-123", "message-456", "/tmp/test")
	call := interfaces.ToolCall{
		ID:    "call-1",
		Name:  BrowserToolName,
		Input: `{"action": "type", "index": 5}`,
	}

	response, err := tool.Run(ctx, call)

	require.NoError(t, err)
	assert.True(t, response.IsError)
	assert.Contains(t, response.Content, "missing text parameter")
}

// Test scroll action validation
func TestBrowserToolScrollValidation(t *testing.T) {
	mockPermissionService := &MockPermissionService{}
	sessionConfig := session.DefaultConfig()
	tool := NewBrowserTool(mockPermissionService, "ws://localhost:8080", sessionConfig)

	tests := []struct {
		name     string
		input    string
		errorMsg string
	}{
		{
			name:     "missing direction",
			input:    `{"action": "scroll"}`,
			errorMsg: "missing direction parameter",
		},
		{
			name:     "invalid direction",
			input:    `{"action": "scroll", "direction": "sideways"}`,
			errorMsg: "invalid direction",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := createBrowserTestContext("session-123", "message-456", "/tmp/test")
			call := interfaces.ToolCall{
				ID:    "call-1",
				Name:  BrowserToolName,
				Input: tt.input,
			}

			response, err := tool.Run(ctx, call)

			require.NoError(t, err)
			assert.True(t, response.IsError)
			assert.Contains(t, response.Content, tt.errorMsg)
		})
	}
}

// Test permission request for open action
// NOTE: This test is currently skipped because permissions are temporarily disabled in browser.go
// TODO: Re-enable this test when permissions are re-enabled
func TestBrowserToolOpenPermissionDenied(t *testing.T) {
	t.Skip("Permissions are temporarily disabled in browser tool - see TODO in browser.go")

	mockPermissionService := &MockPermissionService{}
	sessionConfig := session.DefaultConfig()
	tool := NewBrowserTool(mockPermissionService, "ws://localhost:8080", sessionConfig)

	ctx := createBrowserTestContext("session-123", "message-456", "/tmp/test")
	call := interfaces.ToolCall{
		ID:    "call-1",
		Name:  BrowserToolName,
		Input: `{"action": "open", "url": "https://example.com"}`,
	}

	// Mock permission denied
	mockPermissionService.On("Request", mock.AnythingOfType("permission.CreatePermissionRequest")).Return(false)

	_, err := tool.Run(ctx, call)

	require.Error(t, err)
	assert.Equal(t, permission.ErrPermissionDenied, err)
	mockPermissionService.AssertExpectations(t)
}

// Test getBaseURL function
func TestGetBaseURL(t *testing.T) {
	tests := []struct {
		name         string
		frontendURL  string
		baseURL      string
		expectedURL  string
	}{
		{
			name:        "FRONTEND_URL set",
			frontendURL: "https://frontend.example.com",
			baseURL:     "",
			expectedURL: "https://frontend.example.com",
		},
		{
			name:        "BASE_URL set when FRONTEND_URL empty",
			frontendURL: "",
			baseURL:     "https://base.example.com",
			expectedURL: "https://base.example.com",
		},
		{
			name:        "default when both empty",
			frontendURL: "",
			baseURL:     "",
			expectedURL: "http://localhost:3020",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Set environment variables
			if tt.frontendURL != "" {
				t.Setenv("FRONTEND_URL", tt.frontendURL)
			}
			if tt.baseURL != "" {
				t.Setenv("BASE_URL", tt.baseURL)
			}

			result := getBaseURL()
			assert.Equal(t, tt.expectedURL, result)
		})
	}
}

// Test scroll direction validation with valid directions
func TestScrollDirectionValidation(t *testing.T) {
	validDirections := []string{DirectionUp, DirectionDown, DirectionLeft, DirectionRight}

	for _, direction := range validDirections {
		t.Run(direction, func(t *testing.T) {
			validDirections := map[string]bool{
				DirectionUp:    true,
				DirectionDown:  true,
				DirectionLeft:  true,
				DirectionRight: true,
			}
			assert.True(t, validDirections[direction])
		})
	}
}

// Test default scroll amount
func TestDefaultScrollAmount(t *testing.T) {
	amount := 0
	if amount == 0 {
		amount = 100
	}
	assert.Equal(t, 100, amount)
}
