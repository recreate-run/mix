package protocol

import (
	"encoding/json"
	"testing"
)

func TestRequestSerialization(t *testing.T) {
	tests := []struct {
		name     string
		request  Request
		expected string
	}{
		{
			name: "request with params",
			//nolint:musttag // Test struct for JSON serialization testing
			request: Request{
				ID:     "1",
				Method: "Page.navigate",
				Params: json.RawMessage(`{"url":"https://example.com"}`),
			},
			expected: `{"id":"1","method":"Page.navigate","params":{"url":"https://example.com"}}`,
		},
		{
			name: "request without params",
			//nolint:musttag // Test struct for JSON serialization testing
			request: Request{
				ID:     "2",
				Method: "Page.screenshot",
			},
			expected: `{"id":"2","method":"Page.screenshot"}`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Marshal to JSON
			//nolint:musttag // Test struct for JSON serialization testing
			data, err := json.Marshal(tt.request)
			if err != nil {
				t.Fatalf("Failed to marshal request: %v", err)
			}

			// Verify JSON matches expected
			if string(data) != tt.expected {
				t.Errorf("JSON mismatch:\ngot:  %s\nwant: %s", string(data), tt.expected)
			}

			// Unmarshal and verify round-trip
			var decoded Request
			if err := json.Unmarshal(data, &decoded); err != nil {
				t.Fatalf("Failed to unmarshal request: %v", err)
			}

			if decoded.ID != tt.request.ID {
				t.Errorf("ID mismatch: got %s, want %s", decoded.ID, tt.request.ID)
			}
			if decoded.Method != tt.request.Method {
				t.Errorf("Method mismatch: got %s, want %s", decoded.Method, tt.request.Method)
			}
		})
	}
}

func TestResponseSerialization(t *testing.T) {
	tests := []struct {
		name     string
		response Response
		wantErr  bool
	}{
		{
			name: "response with result",
			//nolint:musttag // Test struct for JSON serialization testing
			response: Response{
				ID: "1",
				Result: NavigateResult{
					FrameID:  "frame-123",
					LoaderID: "loader-456",
				},
			},
			wantErr: false,
		},
		{
			name: "response with error",
			//nolint:musttag // Test struct for JSON serialization testing
			response: Response{
				ID: "2",
				Error: &Error{
					Code:    ErrCodeInvalidParams,
					Message: "Missing URL parameter",
				},
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Marshal to JSON
			//nolint:musttag // Test struct for JSON serialization testing
			data, err := json.Marshal(tt.response)
			if err != nil {
				t.Fatalf("Failed to marshal response: %v", err)
			}

			// Unmarshal and verify
			var decoded Response
			if err := json.Unmarshal(data, &decoded); err != nil {
				t.Fatalf("Failed to unmarshal response: %v", err)
			}

			if decoded.ID != tt.response.ID {
				t.Errorf("ID mismatch: got %s, want %s", decoded.ID, tt.response.ID)
			}

			if tt.response.Error != nil {
				if decoded.Error == nil {
					t.Error("Expected error in response, got nil")
				} else {
					if decoded.Error.Code != tt.response.Error.Code {
						t.Errorf("Error code mismatch: got %d, want %d", decoded.Error.Code, tt.response.Error.Code)
					}
					if decoded.Error.Message != tt.response.Error.Message {
						t.Errorf("Error message mismatch: got %s, want %s", decoded.Error.Message, tt.response.Error.Message)
					}
				}
			}
		})
	}
}

func TestErrorResponseSerialization(t *testing.T) {
	err := NewError(ErrCodeMethodNotFound, "Method not found")
	resp := NewErrorResponse("test-id", err)

	//nolint:musttag // Test struct for JSON serialization testing
	data, marshalErr := json.Marshal(resp)
	if marshalErr != nil {
		t.Fatalf("Failed to marshal error response: %v", marshalErr)
	}

	var decoded Response
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("Failed to unmarshal error response: %v", err)
	}

	if decoded.ID != "test-id" {
		t.Errorf("ID mismatch: got %s, want test-id", decoded.ID)
	}

	if decoded.Error == nil {
		t.Fatal("Expected error in response, got nil")
	}

	if decoded.Error.Code != ErrCodeMethodNotFound {
		t.Errorf("Error code mismatch: got %d, want %d", decoded.Error.Code, ErrCodeMethodNotFound)
	}

	if decoded.Result != nil {
		t.Error("Expected nil result in error response")
	}
}

func TestNavigateParamsSerialization(t *testing.T) {
	tests := []struct {
		name     string
		params   NavigateParams
		expected string
	}{
		{
			name: "with timeout",
			//nolint:musttag // Test struct for JSON serialization testing
			params: NavigateParams{
				URL:     "https://example.com",
				Timeout: 5000,
			},
			expected: `{"url":"https://example.com","timeout":5000}`,
		},
		{
			name: "without timeout",
			//nolint:musttag // Test struct for JSON serialization testing
			params: NavigateParams{
				URL: "https://example.com",
			},
			expected: `{"url":"https://example.com"}`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			//nolint:musttag // Test struct for JSON serialization testing
			data, err := json.Marshal(tt.params)
			if err != nil {
				t.Fatalf("Failed to marshal params: %v", err)
			}

			if string(data) != tt.expected {
				t.Errorf("JSON mismatch:\ngot:  %s\nwant: %s", string(data), tt.expected)
			}

			var decoded NavigateParams
			if err := json.Unmarshal(data, &decoded); err != nil {
				t.Fatalf("Failed to unmarshal params: %v", err)
			}

			if decoded.URL != tt.params.URL {
				t.Errorf("URL mismatch: got %s, want %s", decoded.URL, tt.params.URL)
			}
			if decoded.Timeout != tt.params.Timeout {
				t.Errorf("Timeout mismatch: got %d, want %d", decoded.Timeout, tt.params.Timeout)
			}
		})
	}
}

func TestScreenshotParamsSerialization(t *testing.T) {
	tests := []struct {
		name   string
		params ScreenshotParams
	}{
		{
			name:   "default params",
			params: ScreenshotParams{},
		},
		{
			name: "png full page",
			//nolint:musttag // Test struct for JSON serialization testing
			params: ScreenshotParams{
				Format:   "png",
				FullPage: true,
			},
		},
		{
			name: "jpeg with quality",
			//nolint:musttag // Test struct for JSON serialization testing
			params: ScreenshotParams{
				Format:  "jpeg",
				Quality: 80,
			},
		},
		{
			name: "with raw mode",
			//nolint:musttag // Test struct for JSON serialization testing
			params: ScreenshotParams{
				Format: "png",
				Raw:    true,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			//nolint:musttag // Test struct for JSON serialization testing
			data, err := json.Marshal(tt.params)
			if err != nil {
				t.Fatalf("Failed to marshal params: %v", err)
			}

			var decoded ScreenshotParams
			if err := json.Unmarshal(data, &decoded); err != nil {
				t.Fatalf("Failed to unmarshal params: %v", err)
			}

			if decoded.Format != tt.params.Format {
				t.Errorf("Format mismatch: got %s, want %s", decoded.Format, tt.params.Format)
			}
			if decoded.Quality != tt.params.Quality {
				t.Errorf("Quality mismatch: got %d, want %d", decoded.Quality, tt.params.Quality)
			}
			if decoded.FullPage != tt.params.FullPage {
				t.Errorf("FullPage mismatch: got %v, want %v", decoded.FullPage, tt.params.FullPage)
			}
			if decoded.Raw != tt.params.Raw {
				t.Errorf("Raw mismatch: got %v, want %v", decoded.Raw, tt.params.Raw)
			}
		})
	}
}

func TestElementSerialization(t *testing.T) {
	element := Element{
		Index: 1,
		Role:  "button",
		Name:  "Click Me",
		Bounds: BoundingBox{
			X:      100.5,
			Y:      200.75,
			Width:  50.25,
			Height: 30.0,
		},
	}

	//nolint:musttag // Test struct for JSON serialization testing
	data, err := json.Marshal(element)
	if err != nil {
		t.Fatalf("Failed to marshal element: %v", err)
	}

	var decoded Element
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("Failed to unmarshal element: %v", err)
	}

	if decoded.Index != element.Index {
		t.Errorf("Index mismatch: got %d, want %d", decoded.Index, element.Index)
	}
	if decoded.Role != element.Role {
		t.Errorf("Role mismatch: got %s, want %s", decoded.Role, element.Role)
	}
	if decoded.Name != element.Name {
		t.Errorf("Name mismatch: got %s, want %s", decoded.Name, element.Name)
	}
	if decoded.Bounds.X != element.Bounds.X {
		t.Errorf("X mismatch: got %f, want %f", decoded.Bounds.X, element.Bounds.X)
	}
	if decoded.Bounds.Y != element.Bounds.Y {
		t.Errorf("Y mismatch: got %f, want %f", decoded.Bounds.Y, element.Bounds.Y)
	}
	if decoded.Bounds.Width != element.Bounds.Width {
		t.Errorf("Width mismatch: got %f, want %f", decoded.Bounds.Width, element.Bounds.Width)
	}
	if decoded.Bounds.Height != element.Bounds.Height {
		t.Errorf("Height mismatch: got %f, want %f", decoded.Bounds.Height, element.Bounds.Height)
	}
}

func TestErrorCodes(t *testing.T) {
	tests := []struct {
		name string
		code int
	}{
		{"InvalidRequest", ErrCodeInvalidRequest},
		{"MethodNotFound", ErrCodeMethodNotFound},
		{"InvalidParams", ErrCodeInvalidParams},
		{"InternalError", ErrCodeInternalError},
		{"BrowserError", ErrCodeBrowserError},
		{"NavigationError", ErrCodeNavigationError},
		{"ElementNotFound", ErrCodeElementNotFound},
		{"Timeout", ErrCodeTimeout},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.code == 0 {
				t.Errorf("Error code %s is zero", tt.name)
			}
			if tt.code > 0 {
				t.Errorf("Error code %s is positive (should be negative)", tt.name)
			}
		})
	}
}

func TestNewError(t *testing.T) {
	err := NewError(ErrCodeInvalidParams, "test error")

	if err.Code != ErrCodeInvalidParams {
		t.Errorf("Code mismatch: got %d, want %d", err.Code, ErrCodeInvalidParams)
	}
	if err.Message != "test error" {
		t.Errorf("Message mismatch: got %s, want 'test error'", err.Message)
	}
}

func TestNewResponse(t *testing.T) {
	result := NavigateResult{FrameID: "test-frame"}
	resp := NewResponse("test-id", result)

	if resp.ID != "test-id" {
		t.Errorf("ID mismatch: got %s, want test-id", resp.ID)
	}
	if resp.Result == nil {
		t.Error("Expected non-nil result")
	}
	if resp.Error != nil {
		t.Error("Expected nil error")
	}
}
