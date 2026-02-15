package server

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/sarathmenon/browser-service/pkg/protocol"
)

// These are unit tests for message parsing and routing logic only.
// Tests that require a real browser context are in browser_integration_test.go

func TestHandleInvalidJSON(t *testing.T) {
	// We can't easily create a test client without a browser
	// This will be tested in integration tests
	handler := &MessageHandler{
		client: nil, // Will cause panic if we try to call browser methods
	}

	response := handler.Handle(context.Background(), []byte("invalid json"))

	if response.Error == nil {
		t.Fatal("Expected error for invalid JSON")
	}

	if response.Error.Code != protocol.ErrCodeInvalidRequest {
		t.Errorf("Expected error code %d, got %d", protocol.ErrCodeInvalidRequest, response.Error.Code)
	}
}

func TestHandleUnknownMethod(t *testing.T) {
	handler := &MessageHandler{
		client: &Client{ID: "test"},
	}

	req := protocol.Request{
		ID:     "1",
		Method: "Unknown.method",
	}

	//nolint:musttag // false positive for test struct
	data, _ := json.Marshal(req)
	response := handler.Handle(context.Background(), data)

	if response.Error == nil {
		t.Fatal("Expected error for unknown method")
	}

	if response.Error.Code != protocol.ErrCodeMethodNotFound {
		t.Errorf("Expected error code %d, got %d", protocol.ErrCodeMethodNotFound, response.Error.Code)
	}

	if response.ID != "1" {
		t.Errorf("Expected response ID '1', got '%s'", response.ID)
	}
}

func TestRequestSerialization(t *testing.T) {
	// Test that we can properly serialize and deserialize requests
	tests := []struct {
		name   string
		method string
		params interface{}
	}{
		{
			name:   "navigate with params",
			method: "Page.navigate",
			params: protocol.NavigateParams{URL: "https://example.com", Timeout: 5000},
		},
		{
			name:   "screenshot with params",
			method: "Page.screenshot",
			params: protocol.ScreenshotParams{Format: "png", FullPage: true},
		},
		{
			name:   "click with params",
			method: "Page.click",
			params: protocol.ClickParams{Index: 5},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			paramsJSON, err := json.Marshal(tt.params)
			if err != nil {
				t.Fatalf("Failed to marshal params: %v", err)
			}

			req := protocol.Request{
				ID:     "test-id",
				Method: tt.method,
				Params: paramsJSON,
			}

			//nolint:musttag // false positive for test struct
			data, err := json.Marshal(req)
			if err != nil {
				t.Fatalf("Failed to marshal request: %v", err)
			}

			var decoded protocol.Request
			if err := json.Unmarshal(data, &decoded); err != nil {
				t.Fatalf("Failed to unmarshal request: %v", err)
			}

			if decoded.ID != req.ID {
				t.Errorf("ID mismatch: got %s, want %s", decoded.ID, req.ID)
			}
			if decoded.Method != req.Method {
				t.Errorf("Method mismatch: got %s, want %s", decoded.Method, req.Method)
			}
		})
	}
}

func TestNavigateParamsValidation(t *testing.T) {
	tests := []struct {
		name       string
		paramsJSON string
		wantError  bool
	}{
		{
			name:       "valid params",
			paramsJSON: `{"url":"https://example.com"}`,
			wantError:  false,
		},
		{
			name:       "empty URL",
			paramsJSON: `{"url":""}`,
			wantError:  false, // Validation happens in handler, not unmarshaling
		},
		{
			name:       "with timeout",
			paramsJSON: `{"url":"https://example.com","timeout":5000}`,
			wantError:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var params protocol.NavigateParams
			err := json.Unmarshal([]byte(tt.paramsJSON), &params)

			if tt.wantError && err == nil {
				t.Error("Expected error, got nil")
			}
			if !tt.wantError && err != nil {
				t.Errorf("Expected no error, got: %v", err)
			}
		})
	}
}

func TestScreenshotParamsValidation(t *testing.T) {
	tests := []struct {
		name       string
		paramsJSON string
		wantError  bool
	}{
		{
			name:       "empty params",
			paramsJSON: `{}`,
			wantError:  false,
		},
		{
			name:       "with format",
			paramsJSON: `{"format":"png"}`,
			wantError:  false,
		},
		{
			name:       "with all fields",
			paramsJSON: `{"format":"jpeg","quality":80,"fullPage":true,"withOverlay":true}`,
			wantError:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var params protocol.ScreenshotParams
			err := json.Unmarshal([]byte(tt.paramsJSON), &params)

			if tt.wantError && err == nil {
				t.Error("Expected error, got nil")
			}
			if !tt.wantError && err != nil {
				t.Errorf("Expected no error, got: %v", err)
			}
		})
	}
}

func TestErrorResponseFormat(t *testing.T) {
	// Test that error responses have the correct format
	protoErr := protocol.NewError(protocol.ErrCodeInvalidParams, "test error")
	resp := protocol.NewErrorResponse("test-id", protoErr)

	if resp.ID != "test-id" {
		t.Errorf("Expected ID 'test-id', got '%s'", resp.ID)
	}

	if resp.Error == nil {
		t.Fatal("Expected error in response")
	}

	if resp.Error.Code != protocol.ErrCodeInvalidParams {
		t.Errorf("Expected error code %d, got %d", protocol.ErrCodeInvalidParams, resp.Error.Code)
	}

	if resp.Error.Message != "test error" {
		t.Errorf("Expected error message 'test error', got '%s'", resp.Error.Message)
	}

	if resp.Result != nil {
		t.Error("Expected nil result in error response")
	}

	// Verify it can be marshaled to JSON
	//nolint:musttag // false positive for test struct
	data, err := json.Marshal(resp)
	if err != nil {
		t.Fatalf("Failed to marshal error response: %v", err)
	}

	// Verify it can be unmarshaled
	var decoded protocol.Response
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("Failed to unmarshal error response: %v", err)
	}

	if decoded.Error == nil {
		t.Fatal("Expected error in decoded response")
	}
}

func TestSuccessResponseFormat(t *testing.T) {
	// Test that success responses have the correct format
	result := protocol.NavigateResult{
		FrameID:  "test-frame",
		LoaderID: "test-loader",
	}
	resp := protocol.NewResponse("test-id", result)

	if resp.ID != "test-id" {
		t.Errorf("Expected ID 'test-id', got '%s'", resp.ID)
	}

	if resp.Result == nil {
		t.Fatal("Expected result in response")
	}

	if resp.Error != nil {
		t.Error("Expected nil error in success response")
	}

	// Verify it can be marshaled to JSON
	//nolint:musttag // false positive for test struct
	data, err := json.Marshal(resp)
	if err != nil {
		t.Fatalf("Failed to marshal success response: %v", err)
	}

	// Verify it can be unmarshaled
	var decoded protocol.Response
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("Failed to unmarshal success response: %v", err)
	}

	if decoded.Result == nil {
		t.Fatal("Expected result in decoded response")
	}
}

func TestMessageHandlerConstruction(t *testing.T) {
	client := &Client{
		ID:   "test-client",
		Conn: nil,
		done: make(chan struct{}),
	}

	handler := NewMessageHandler(client)

	if handler == nil {
		t.Fatal("NewMessageHandler returned nil")
	}

	if handler.client != client {
		t.Error("Handler client doesn't match provided client")
	}
}
