package tools

import (
	"context"
	"encoding/json"
	"testing"

	"mix/internal/notification"
	"mix/internal/pubsub"
)

const (
	responseTypeText = "text"
)

// mockNotificationService for testing
type mockNotificationService struct {
	requestFunc  func(notification.CreateNotificationRequest) (notification.NotificationResponse, error)
	respondFunc  func(string, notification.NotificationResponse) error
}

func (m *mockNotificationService) Request(opts notification.CreateNotificationRequest) (notification.NotificationResponse, error) {
	if m.requestFunc != nil {
		return m.requestFunc(opts)
	}
	return notification.NotificationResponse{}, nil
}

func (m *mockNotificationService) Respond(notificationID string, response notification.NotificationResponse) error {
	if m.respondFunc != nil {
		return m.respondFunc(notificationID, response)
	}
	return nil
}

func (m *mockNotificationService) Subscribe(ctx context.Context) <-chan pubsub.Event[notification.NotificationRequest] {
	ch := make(chan pubsub.Event[notification.NotificationRequest])
	return ch
}

func TestNotifyTool_Info(t *testing.T) {
	t.Helper()
	tool := NewNotifyTool(&mockNotificationService{})
	
	info := tool.Info()
	
	if info.Name != "Notify" {
		t.Errorf("Expected name 'Notify', got '%s'", info.Name)
	}
	
	if len(info.Required) != 4 {
		t.Errorf("Expected 4 required fields, got %d", len(info.Required))
	}
}

func TestNotifyTool_Run_Acknowledge(t *testing.T) {
	t.Helper()
	
	mockSvc := &mockNotificationService{
		requestFunc: func(req notification.CreateNotificationRequest) (notification.NotificationResponse, error) {
			// Validate request
			if req.Type != notification.NotificationTypeInfo {
				t.Errorf("Expected type 'info', got '%s'", req.Type)
			}
			if req.ResponseType != notification.ResponseTypeAcknowledge {
				t.Errorf("Expected response type 'acknowledge', got '%s'", req.ResponseType)
			}
			
			return notification.NotificationResponse{
				ID:   "test-notif-id",
				Type: "acknowledge",
			}, nil
		},
	}
	
	tool := NewNotifyTool(mockSvc)
	
	params := NotifyParams{
		Type:         "info",
		Title:        "Test Notification",
		Message:      "Please acknowledge",
		ResponseType: "acknowledge",
	}
	
	paramsJSON, _ := json.Marshal(params)
	
	ctx := context.WithValue(context.Background(), SessionIDContextKey, "test-session")
	
	resp, err := tool.Run(ctx, ToolCall{
		Input: string(paramsJSON),
	})
	
	if err != nil {
		t.Fatalf("Run failed: %v", err)
	}
	
	if resp.Type != responseTypeText {
		t.Errorf("Expected response type 'text', got '%s'", resp.Type)
	}
	
	expectedContent := "User acknowledged the notification"
	if resp.Content != expectedContent {
		t.Errorf("Expected content '%s', got '%s'", expectedContent, resp.Content)
	}
}

func TestNotifyTool_Run_Text(t *testing.T) {
	t.Helper()
	
	expectedText := "User's response text"
	
	mockSvc := &mockNotificationService{
		requestFunc: func(req notification.CreateNotificationRequest) (notification.NotificationResponse, error) {
			return notification.NotificationResponse{
				ID:    "test-notif-id",
				Type:  "text",
				Value: expectedText,
			}, nil
		},
	}
	
	tool := NewNotifyTool(mockSvc)
	
	params := NotifyParams{
		Type:         "question",
		Title:        "Enter your name",
		Message:      "Please provide your name",
		ResponseType: "text",
	}
	
	paramsJSON, _ := json.Marshal(params)
	
	ctx := context.WithValue(context.Background(), SessionIDContextKey, "test-session")
	
	resp, err := tool.Run(ctx, ToolCall{
		Input: string(paramsJSON),
	})
	
	if err != nil {
		t.Fatalf("Run failed: %v", err)
	}
	
	expectedContent := "User response: " + expectedText
	if resp.Content != expectedContent {
		t.Errorf("Expected content '%s', got '%s'", expectedContent, resp.Content)
	}
}

func TestNotifyTool_Run_Choice(t *testing.T) {
	t.Helper()
	
	choices := []string{"Option A", "Option B", "Option C"}
	selectedChoice := "Option B"
	
	mockSvc := &mockNotificationService{
		requestFunc: func(req notification.CreateNotificationRequest) (notification.NotificationResponse, error) {
			// Validate choices
			if len(req.Choices) != 3 {
				t.Errorf("Expected 3 choices, got %d", len(req.Choices))
			}
			
			return notification.NotificationResponse{
				ID:    "test-notif-id",
				Type:  "choice",
				Value: selectedChoice,
			}, nil
		},
	}
	
	tool := NewNotifyTool(mockSvc)
	
	params := NotifyParams{
		Type:         "question",
		Title:        "Choose an option",
		Message:      "Please select one",
		ResponseType: "choice",
		Choices:      choices,
	}
	
	paramsJSON, _ := json.Marshal(params)
	
	ctx := context.WithValue(context.Background(), SessionIDContextKey, "test-session")
	
	resp, err := tool.Run(ctx, ToolCall{
		Input: string(paramsJSON),
	})
	
	if err != nil {
		t.Fatalf("Run failed: %v", err)
	}
	
	expectedContent := "User selected: " + selectedChoice
	if resp.Content != expectedContent {
		t.Errorf("Expected content '%s', got '%s'", expectedContent, resp.Content)
	}
}

func TestNotifyTool_Run_Timeout(t *testing.T) {
	t.Helper()
	
	mockSvc := &mockNotificationService{
		requestFunc: func(req notification.CreateNotificationRequest) (notification.NotificationResponse, error) {
			return notification.NotificationResponse{}, notification.ErrNotificationTimeout
		},
	}
	
	tool := NewNotifyTool(mockSvc)
	
	params := NotifyParams{
		Type:         "info",
		Title:        "Test Timeout",
		Message:      "This will timeout",
		ResponseType: "acknowledge",
		Timeout:      1,
	}
	
	paramsJSON, _ := json.Marshal(params)
	
	ctx := context.WithValue(context.Background(), SessionIDContextKey, "test-session")
	
	resp, err := tool.Run(ctx, ToolCall{
		Input: string(paramsJSON),
	})
	
	if err != nil {
		t.Fatalf("Run should not return error, got: %v", err)
	}
	
	if resp.Type != responseTypeText {
		t.Errorf("Expected response type 'text', got '%s'", resp.Type)
	}
	
	expectedContent := "Notification request timed out - user did not respond in time"
	if resp.Content != expectedContent {
		t.Errorf("Expected timeout message, got '%s'", resp.Content)
	}
}

func TestNotifyTool_Run_InvalidJSON(t *testing.T) {
	t.Helper()
	
	tool := NewNotifyTool(&mockNotificationService{})
	
	ctx := context.WithValue(context.Background(), SessionIDContextKey, "test-session")
	
	resp, err := tool.Run(ctx, ToolCall{
		Input: "invalid json {{{",
	})
	
	if err != nil {
		t.Fatalf("Run should not return error for invalid JSON, got: %v", err)
	}
	
	if resp.Type != responseTypeText {
		t.Errorf("Expected response type 'text', got '%s'", resp.Type)
	}
}

func TestNotifyTool_Run_MissingRequiredFields(t *testing.T) {
	t.Helper()
	
	tests := []struct {
		name   string
		params NotifyParams
		errMsg string
	}{
		{
			name: "missing type",
			params: NotifyParams{
				Title:        "Test",
				Message:      "Test message",
				ResponseType: "acknowledge",
			},
			errMsg: "Missing required field: type",
		},
		{
			name: "missing title",
			params: NotifyParams{
				Type:         "info",
				Message:      "Test message",
				ResponseType: "acknowledge",
			},
			errMsg: "Missing required field: title",
		},
		{
			name: "missing message",
			params: NotifyParams{
				Type:         "info",
				Title:        "Test",
				ResponseType: "acknowledge",
			},
			errMsg: "Missing required field: message",
		},
		{
			name: "missing responseType",
			params: NotifyParams{
				Type:    "info",
				Title:   "Test",
				Message: "Test message",
			},
			errMsg: "Missing required field: responseType",
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Helper()
			
			tool := NewNotifyTool(&mockNotificationService{})
			
			paramsJSON, _ := json.Marshal(tt.params)
			
			ctx := context.WithValue(context.Background(), SessionIDContextKey, "test-session")
			
			resp, err := tool.Run(ctx, ToolCall{
				Input: string(paramsJSON),
			})
			
			if err != nil {
				t.Fatalf("Run should not return error, got: %v", err)
			}
			
			if resp.Content != tt.errMsg {
				t.Errorf("Expected error message '%s', got '%s'", tt.errMsg, resp.Content)
			}
		})
	}
}

func TestNotifyTool_Run_InvalidType(t *testing.T) {
	t.Helper()
	
	tool := NewNotifyTool(&mockNotificationService{})
	
	params := NotifyParams{
		Type:         "invalid",
		Title:        "Test",
		Message:      "Test message",
		ResponseType: "acknowledge",
	}
	
	paramsJSON, _ := json.Marshal(params)
	
	ctx := context.WithValue(context.Background(), SessionIDContextKey, "test-session")
	
	resp, err := tool.Run(ctx, ToolCall{
		Input: string(paramsJSON),
	})
	
	if err != nil {
		t.Fatalf("Run should not return error, got: %v", err)
	}
	
	if resp.Type != responseTypeText {
		t.Errorf("Expected response type 'text', got '%s'", resp.Type)
	}
}

func TestNotifyTool_Run_InvalidResponseType(t *testing.T) {
	t.Helper()
	
	tool := NewNotifyTool(&mockNotificationService{})
	
	params := NotifyParams{
		Type:         "info",
		Title:        "Test",
		Message:      "Test message",
		ResponseType: "invalid",
	}
	
	paramsJSON, _ := json.Marshal(params)
	
	ctx := context.WithValue(context.Background(), SessionIDContextKey, "test-session")
	
	resp, err := tool.Run(ctx, ToolCall{
		Input: string(paramsJSON),
	})
	
	if err != nil {
		t.Fatalf("Run should not return error, got: %v", err)
	}
	
	if resp.Type != responseTypeText {
		t.Errorf("Expected response type 'text', got '%s'", resp.Type)
	}
}

func TestNotifyTool_Run_MissingChoices(t *testing.T) {
	t.Helper()
	
	tool := NewNotifyTool(&mockNotificationService{})
	
	params := NotifyParams{
		Type:         "question",
		Title:        "Choose",
		Message:      "Please select",
		ResponseType: "choice",
		Choices:      []string{},
	}
	
	paramsJSON, _ := json.Marshal(params)
	
	ctx := context.WithValue(context.Background(), SessionIDContextKey, "test-session")
	
	resp, err := tool.Run(ctx, ToolCall{
		Input: string(paramsJSON),
	})
	
	if err != nil {
		t.Fatalf("Run should not return error, got: %v", err)
	}
	
	expectedContent := "choices array is required when responseType is 'choice'"
	if resp.Content != expectedContent {
		t.Errorf("Expected error message '%s', got '%s'", expectedContent, resp.Content)
	}
}

func TestNotifyTool_Run_InsufficientChoices(t *testing.T) {
	t.Helper()
	
	tool := NewNotifyTool(&mockNotificationService{})
	
	params := NotifyParams{
		Type:         "question",
		Title:        "Choose",
		Message:      "Please select",
		ResponseType: "choice",
		Choices:      []string{"Only one"},
	}
	
	paramsJSON, _ := json.Marshal(params)
	
	ctx := context.WithValue(context.Background(), SessionIDContextKey, "test-session")
	
	resp, err := tool.Run(ctx, ToolCall{
		Input: string(paramsJSON),
	})
	
	if err != nil {
		t.Fatalf("Run should not return error, got: %v", err)
	}
	
	expectedContent := "choices array must contain at least 2 options"
	if resp.Content != expectedContent {
		t.Errorf("Expected error message '%s', got '%s'", expectedContent, resp.Content)
	}
}

func TestNotifyTool_Run_InvalidTimeout(t *testing.T) {
	t.Helper()
	
	tool := NewNotifyTool(&mockNotificationService{})
	
	params := NotifyParams{
		Type:         "info",
		Title:        "Test",
		Message:      "Test message",
		ResponseType: "acknowledge",
		Timeout:      500,
	}
	
	paramsJSON, _ := json.Marshal(params)
	
	ctx := context.WithValue(context.Background(), SessionIDContextKey, "test-session")
	
	resp, err := tool.Run(ctx, ToolCall{
		Input: string(paramsJSON),
	})
	
	if err != nil {
		t.Fatalf("Run should not return error, got: %v", err)
	}
	
	expectedContent := "timeout must be between 0 and 300 seconds"
	if resp.Content != expectedContent {
		t.Errorf("Expected error message '%s', got '%s'", expectedContent, resp.Content)
	}
}

func TestNotifyTool_Run_MissingSessionID(t *testing.T) {
	t.Helper()
	
	tool := NewNotifyTool(&mockNotificationService{})
	
	params := NotifyParams{
		Type:         "info",
		Title:        "Test",
		Message:      "Test message",
		ResponseType: "acknowledge",
	}
	
	paramsJSON, _ := json.Marshal(params)
	
	// Context without session ID
	ctx := context.Background()
	
	resp, err := tool.Run(ctx, ToolCall{
		Input: string(paramsJSON),
	})
	
	if err != nil {
		t.Fatalf("Run should not return error, got: %v", err)
	}
	
	expectedContent := "Failed to get session ID from context"
	if resp.Content != expectedContent {
		t.Errorf("Expected error message '%s', got '%s'", expectedContent, resp.Content)
	}
}

func TestNotifyTool_Run_WithCustomTimeout(t *testing.T) {
	t.Helper()
	
	mockSvc := &mockNotificationService{
		requestFunc: func(req notification.CreateNotificationRequest) (notification.NotificationResponse, error) {
			if req.Timeout != 30 {
				t.Errorf("Expected timeout 30, got %d", req.Timeout)
			}
			
			return notification.NotificationResponse{
				ID:   "test-notif-id",
				Type: "acknowledge",
			}, nil
		},
	}
	
	tool := NewNotifyTool(mockSvc)
	
	params := NotifyParams{
		Type:         "info",
		Title:        "Test Timeout",
		Message:      "Custom timeout",
		ResponseType: "acknowledge",
		Timeout:      30,
	}
	
	paramsJSON, _ := json.Marshal(params)
	
	ctx := context.WithValue(context.Background(), SessionIDContextKey, "test-session")
	
	_, err := tool.Run(ctx, ToolCall{
		Input: string(paramsJSON),
	})
	
	if err != nil {
		t.Fatalf("Run failed: %v", err)
	}
}
