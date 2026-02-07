package http

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Test ErrorResponse structure
func TestErrorResponse(t *testing.T) {
	err := &RESTError{
		Code:    400,
		Message: "Bad request",
		Type:    ErrorTypeBadRequest,
	}

	response := &ErrorResponse{
		Error: err,
	}

	assert.Equal(t, 400, response.Error.Code)
	assert.Equal(t, "Bad request", response.Error.Message)
	assert.Equal(t, ErrorTypeBadRequest, response.Error.Type)
}

// Test RESTError structure
func TestRESTError(t *testing.T) {
	err := &RESTError{
		Code:    404,
		Message: "Resource not found",
		Type:    ErrorTypeNotFound,
	}

	assert.Equal(t, 404, err.Code)
	assert.Equal(t, "Resource not found", err.Message)
	assert.Equal(t, ErrorTypeNotFound, err.Type)
}

// Test error type constants
func TestErrorTypeConstants(t *testing.T) {
	assert.Equal(t, "bad_request", ErrorTypeBadRequest)
	assert.Equal(t, "not_found", ErrorTypeNotFound)
	assert.Equal(t, "internal_error", ErrorTypeInternalError)
	assert.Equal(t, "unauthorized", ErrorTypeUnauthorized)
	assert.Equal(t, "validation_error", ErrorTypeValidation)
}

// Test error status mapping
func TestErrorStatusMapping(t *testing.T) {
	expectedMappings := map[string]int{
		ErrorTypeBadRequest:    http.StatusBadRequest,
		ErrorTypeNotFound:      http.StatusNotFound,
		ErrorTypeInternalError: http.StatusInternalServerError,
		ErrorTypeUnauthorized:  http.StatusUnauthorized,
		ErrorTypeValidation:    http.StatusBadRequest,
	}

	for errorType, expectedStatus := range expectedMappings {
		actualStatus, exists := errorStatusMap[errorType]
		assert.True(t, exists, "Error type %s should exist in status map", errorType)
		assert.Equal(t, expectedStatus, actualStatus, "Status for %s should match", errorType)
	}
}

// Test sendJSONResponse function
func TestSendJSONResponse(t *testing.T) {
	tests := []struct {
		name           string
		status         int
		data           interface{}
		expectedStatus int
		expectedBody   interface{}
	}{
		{
			name:           "success response",
			status:         http.StatusOK,
			data:           map[string]string{"message": "success"},
			expectedStatus: http.StatusOK,
			expectedBody:   map[string]interface{}{"message": "success"},
		},
		{
			name:           "error response",
			status:         http.StatusBadRequest,
			data:           ErrorResponse{Error: &RESTError{Code: 400, Message: "Bad request", Type: ErrorTypeBadRequest}},
			expectedStatus: http.StatusBadRequest,
			expectedBody: map[string]interface{}{
				"error": map[string]interface{}{
					"code":    float64(400), // JSON unmarshaling converts numbers to float64
					"message": "Bad request",
					"type":    "bad_request",
				},
			},
		},
		{
			name:           "array response",
			status:         http.StatusOK,
			data:           []string{"item1", "item2"},
			expectedStatus: http.StatusOK,
			expectedBody:   []interface{}{"item1", "item2"},
		},
		{
			name:           "nil response",
			status:         http.StatusNoContent,
			data:           nil,
			expectedStatus: http.StatusNoContent,
			expectedBody:   nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create response recorder
			w := httptest.NewRecorder()

			// Call function
			sendJSONResponse(w, tt.status, tt.data)

			// Check status code
			assert.Equal(t, tt.expectedStatus, w.Code)

			// Check content type
			assert.Equal(t, "application/json", w.Header().Get("Content-Type"))

			// Check response body
			if tt.expectedBody != nil {
				var actualBody interface{}
				err := json.Unmarshal(w.Body.Bytes(), &actualBody)
				require.NoError(t, err)
				assert.Equal(t, tt.expectedBody, actualBody)
			} else {
				// For nil responses, expect "null" in JSON
				assert.Equal(t, "null\n", w.Body.String())
			}
		})
	}
}

// Test sendErrorResponse function
func TestSendErrorResponse(t *testing.T) {
	tests := []struct {
		name           string
		errorType      string
		message        string
		expectedStatus int
		expectedCode   int
	}{
		{
			name:           "bad request error",
			errorType:      ErrorTypeBadRequest,
			message:        "Invalid input",
			expectedStatus: http.StatusBadRequest,
			expectedCode:   http.StatusBadRequest,
		},
		{
			name:           "not found error",
			errorType:      ErrorTypeNotFound,
			message:        "Resource not found",
			expectedStatus: http.StatusNotFound,
			expectedCode:   http.StatusNotFound,
		},
		{
			name:           "internal error",
			errorType:      ErrorTypeInternalError,
			message:        "Server error",
			expectedStatus: http.StatusInternalServerError,
			expectedCode:   http.StatusInternalServerError,
		},
		{
			name:           "unauthorized error",
			errorType:      ErrorTypeUnauthorized,
			message:        "Access denied",
			expectedStatus: http.StatusUnauthorized,
			expectedCode:   http.StatusUnauthorized,
		},
		{
			name:           "validation error",
			errorType:      ErrorTypeValidation,
			message:        "Validation failed",
			expectedStatus: http.StatusBadRequest,
			expectedCode:   http.StatusBadRequest,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w := httptest.NewRecorder()

			sendErrorResponse(w, tt.errorType, tt.message)

			// Check status code
			assert.Equal(t, tt.expectedStatus, w.Code)

			// Check content type
			assert.Equal(t, "application/json", w.Header().Get("Content-Type"))

			// Parse response body
			var response ErrorResponse
			err := json.Unmarshal(w.Body.Bytes(), &response)
			require.NoError(t, err)

			// Check error structure
			assert.NotNil(t, response.Error)
			assert.Equal(t, tt.expectedCode, response.Error.Code)
			assert.Equal(t, tt.message, response.Error.Message)
			assert.Equal(t, tt.errorType, response.Error.Type)
		})
	}
}

// Test sendErrorResponse with unknown error type
func TestSendErrorResponseUnknownType(t *testing.T) {
	w := httptest.NewRecorder()

	sendErrorResponse(w, "unknown_error", "Unknown error")

	// Should default to internal server error
	assert.Equal(t, http.StatusInternalServerError, w.Code)

	var response ErrorResponse
	err := json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)

	assert.Equal(t, http.StatusInternalServerError, response.Error.Code)
	assert.Equal(t, "Unknown error", response.Error.Message)
	assert.Equal(t, "unknown_error", response.Error.Type)
}

// Test sendValidationError function
func TestSendValidationError(t *testing.T) {
	w := httptest.NewRecorder()

	sendValidationError(w, "email", "must be a valid email address")

	assert.Equal(t, http.StatusBadRequest, w.Code)
	assert.Equal(t, "application/json", w.Header().Get("Content-Type"))

	var response ErrorResponse
	err := json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)

	assert.Equal(t, http.StatusBadRequest, response.Error.Code)
	assert.Contains(t, response.Error.Message, "Validation error for field 'email'")
	assert.Contains(t, response.Error.Message, "must be a valid email address")
	assert.Equal(t, ErrorTypeValidation, response.Error.Type)
}

// Test sendNotFoundError function
func TestSendNotFoundError(t *testing.T) {
	w := httptest.NewRecorder()

	sendNotFoundError(w, "User", "123")

	assert.Equal(t, http.StatusNotFound, w.Code)
	assert.Equal(t, "application/json", w.Header().Get("Content-Type"))

	var response ErrorResponse
	err := json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)

	assert.Equal(t, http.StatusNotFound, response.Error.Code)
	assert.Equal(t, "User with ID '123' not found", response.Error.Message)
	assert.Equal(t, ErrorTypeNotFound, response.Error.Type)
}

// Test sendInternalError function
func TestSendInternalError(t *testing.T) {
	w := httptest.NewRecorder()

	sendInternalError(w, "database operation", assert.AnError)

	assert.Equal(t, http.StatusInternalServerError, w.Code)
	assert.Equal(t, "application/json", w.Header().Get("Content-Type"))

	var response ErrorResponse
	err := json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)

	assert.Equal(t, http.StatusInternalServerError, response.Error.Code)
	assert.Contains(t, response.Error.Message, "Internal error during database operation")
	assert.Equal(t, ErrorTypeInternalError, response.Error.Type)
}

// Test sendUnauthorizedError function
func TestSendUnauthorizedError(t *testing.T) {
	w := httptest.NewRecorder()

	sendUnauthorizedError(w, "Invalid API key")

	assert.Equal(t, http.StatusUnauthorized, w.Code)
	assert.Equal(t, "application/json", w.Header().Get("Content-Type"))

	var response ErrorResponse
	err := json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)

	assert.Equal(t, http.StatusUnauthorized, response.Error.Code)
	assert.Equal(t, "Invalid API key", response.Error.Message)
	assert.Equal(t, ErrorTypeUnauthorized, response.Error.Type)
}

// Test parseIntParam function
func TestParseIntParam(t *testing.T) {
	tests := []struct {
		name        string
		param       string
		paramName   string
		expected    int64
		expectError bool
		errorMsg    string
	}{
		{
			name:        "valid positive integer",
			param:       "123",
			paramName:   "id",
			expected:    123,
			expectError: false,
		},
		{
			name:        "valid negative integer",
			param:       "-456",
			paramName:   "offset",
			expected:    -456,
			expectError: false,
		},
		{
			name:        "zero value",
			param:       "0",
			paramName:   "count",
			expected:    0,
			expectError: false,
		},
		{
			name:        "empty parameter",
			param:       "",
			paramName:   "id",
			expected:    0,
			expectError: true,
			errorMsg:    "id is required",
		},
		{
			name:        "invalid integer",
			param:       "abc",
			paramName:   "id",
			expected:    0,
			expectError: true,
			errorMsg:    "id must be a valid integer",
		},
		{
			name:        "float value",
			param:       "123.45",
			paramName:   "id",
			expected:    0,
			expectError: true,
			errorMsg:    "id must be a valid integer",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := parseIntParam(tt.param, tt.paramName)

			if tt.expectError {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.errorMsg)
				assert.Equal(t, int64(0), result)
			} else {
				require.NoError(t, err)
				assert.Equal(t, tt.expected, result)
			}
		})
	}
}

// Test parseJSONBody function
func TestParseJSONBody(t *testing.T) {
	tests := []struct {
		name        string
		body        interface{}
		target      interface{}
		expectError bool
	}{
		{
			name:        "valid JSON object",
			body:        map[string]string{"key": "value"},
			target:      &map[string]string{},
			expectError: false,
		},
		{
			name:        "valid JSON array",
			body:        []string{"item1", "item2"},
			target:      &[]string{},
			expectError: false,
		},
		{
			name:        "nil body",
			body:        nil,
			target:      &map[string]string{},
			expectError: true,
		},
		{
			name:        "invalid JSON",
			body:        `{"key": invalid}`,
			target:      &map[string]string{},
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Prepare request body
			var req *http.Request
			if tt.body == nil {
				req = httptest.NewRequest(http.MethodPost, "/test", http.NoBody)
			} else if str, ok := tt.body.(string); ok {
				req = httptest.NewRequest(http.MethodPost, "/test", bytes.NewReader([]byte(str)))
			} else {
				bodyBytes, err := json.Marshal(tt.body)
				require.NoError(t, err)
				req = httptest.NewRequest(http.MethodPost, "/test", bytes.NewReader(bodyBytes))
			}
			req.Header.Set("Content-Type", "application/json")

			// Call function
			err := parseJSONBody(req, tt.target)

			if tt.expectError {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
				// Verify data was parsed correctly
				switch v := tt.body.(type) {
				case map[string]string:
					actualMap := tt.target.(*map[string]string)
					assert.Equal(t, v, *actualMap)
				case []string:
					actualSlice := tt.target.(*[]string)
					assert.Equal(t, v, *actualSlice)
				}
			}
		})
	}
}

// Test setCORSHeaders function
func TestSetCORSHeaders(t *testing.T) {
	w := httptest.NewRecorder()

	setCORSHeaders(w)

	assert.Equal(t, "*", w.Header().Get("Access-Control-Allow-Origin"))
	assert.Equal(t, "GET, POST, PUT, PATCH, DELETE, OPTIONS", w.Header().Get("Access-Control-Allow-Methods"))
	assert.Equal(t, "Content-Type, Authorization", w.Header().Get("Access-Control-Allow-Headers"))
}

// Test handleCORSPreflight function
func TestHandleCORSPreflight(t *testing.T) {
	tests := []struct {
		name           string
		method         string
		expectedResult bool
		expectedStatus int
	}{
		{
			name:           "OPTIONS request",
			method:         "OPTIONS",
			expectedResult: true,
			expectedStatus: http.StatusOK,
		},
		{
			name:           "GET request",
			method:         "GET",
			expectedResult: false,
			expectedStatus: 0, // No status set
		},
		{
			name:           "POST request",
			method:         "POST",
			expectedResult: false,
			expectedStatus: 0, // No status set
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w := httptest.NewRecorder()
			req := httptest.NewRequest(tt.method, "/test", http.NoBody)

			result := handleCORSPreflight(w, req)

			assert.Equal(t, tt.expectedResult, result)

			if tt.expectedResult {
				assert.Equal(t, tt.expectedStatus, w.Code)
				// Check CORS headers were set
				assert.Equal(t, "*", w.Header().Get("Access-Control-Allow-Origin"))
			}
		})
	}
}

// Test WriteJSONResponse function
func TestWriteJSONResponse(t *testing.T) {
	w := httptest.NewRecorder()

	data := map[string]string{"message": "success"}
	WriteJSONResponse(w, http.StatusOK, data)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, "application/json", w.Header().Get("Content-Type"))

	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)
	assert.Equal(t, "success", response["message"])
}

// Test WriteErrorResponse function
func TestWriteErrorResponse(t *testing.T) {
	w := httptest.NewRecorder()

	WriteErrorResponse(w, http.StatusBadRequest, "Invalid input", ErrorTypeBadRequest)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	assert.Equal(t, "application/json", w.Header().Get("Content-Type"))

	var response ErrorResponse
	err := json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)

	assert.Equal(t, http.StatusBadRequest, response.Error.Code)
	assert.Equal(t, "Invalid input", response.Error.Message)
	assert.Equal(t, ErrorTypeBadRequest, response.Error.Type)
}
