package tunnel

import (
	"context"
	"encoding/json"
	"fmt"
	"reflect"
	"time"
)

// CDPRequest represents a Chrome DevTools Protocol command request
type CDPRequest struct {
	ID        interface{} `json:"id"`
	Method    string      `json:"method"`
	Params    interface{} `json:"params,omitempty"`
	SessionID string      `json:"sessionId,omitempty"`
	BrowserID string      `json:"browserId,omitempty"`
}

// CDPResponse represents a Chrome DevTools Protocol command response
type CDPResponse struct {
	ID        interface{} `json:"id"`
	Result    interface{} `json:"result,omitempty"`
	Error     *CDPError   `json:"error,omitempty"`
	BrowserID string      `json:"browserId,omitempty"`
}

// CDPError represents a Chrome DevTools Protocol error
type CDPError struct {
	Code    int         `json:"code"`
	Message string      `json:"message"`
	Data    interface{} `json:"data,omitempty"`
}

// Client provides Electron tunnel CDP communication using reflection
// Uses reflection to avoid circular import with internal/http
// Implements browser.Client interface via duck typing
type Client struct {
	registry  interface{} // *http.TunnelRegistry (via interface{} to avoid import cycle)
	sessionID string
}

// NewClient creates a new tunnel client
func NewClient(registry interface{}, sessionID string) *Client {
	return &Client{
		registry:  registry,
		sessionID: sessionID,
	}
}

// SendCommand sends a CDP command through the Electron tunnel
func (c *Client) SendCommand(ctx context.Context, method string, params interface{}) (interface{}, error) {
	if c.registry == nil {
		return nil, fmt.Errorf("tunnel registry not available")
	}

	// Use reflection to call SendCommandToTunnel method on registry to avoid circular import
	registryValue := reflect.ValueOf(c.registry)
	sendMethod := registryValue.MethodByName("SendCommandToTunnel")

	if !sendMethod.IsValid() {
		return nil, fmt.Errorf("tunnelRegistry does not have SendCommandToTunnel method")
	}

	// Generate request ID
	requestID := time.Now().UnixNano()

	// Build CDPRequest using reflection
	cdpRequest := map[string]interface{}{
		"id":     requestID,
		"method": method,
		"params": params,
	}

	// Convert map to the actual CDPRequest type via JSON marshaling/unmarshaling
	requestData, err := json.Marshal(cdpRequest)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal CDP request: %w", err)
	}

	// Get the CDPRequest type from the method signature
	requestType := sendMethod.Type().In(1)
	requestValue := reflect.New(requestType).Interface()

	if err := json.Unmarshal(requestData, requestValue); err != nil {
		return nil, fmt.Errorf("failed to unmarshal CDP request: %w", err)
	}

	// Call SendCommandToTunnel(sessionID, request)
	args := []reflect.Value{
		reflect.ValueOf(c.sessionID),
		reflect.ValueOf(requestValue).Elem(),
	}

	results := sendMethod.Call(args)

	// Check for error (second return value)
	if len(results) != 2 {
		return nil, fmt.Errorf("unexpected return values from SendCommandToTunnel")
	}

	if !results[1].IsNil() {
		err := results[1].Interface().(error)
		return nil, fmt.Errorf("tunnel command failed: %w", err)
	}

	// Extract result from CDPResponse
	response := results[0].Interface()
	responseValue := reflect.ValueOf(response)

	// Get the Result field
	if responseValue.Kind() == reflect.Ptr {
		responseValue = responseValue.Elem()
	}

	resultField := responseValue.FieldByName("Result")
	if !resultField.IsValid() {
		return nil, fmt.Errorf("CDPResponse missing Result field")
	}

	// Check for CDP error
	errorField := responseValue.FieldByName("Error")
	if errorField.IsValid() && !errorField.IsNil() {
		errorValue := errorField.Interface()
		errorReflect := reflect.ValueOf(errorValue)
		if errorReflect.Kind() == reflect.Ptr {
			errorReflect = errorReflect.Elem()
		}
		messageField := errorReflect.FieldByName("Message")
		if messageField.IsValid() {
			return nil, fmt.Errorf("CDP error: %s", messageField.String())
		}
		return nil, fmt.Errorf("CDP error occurred")
	}

	return resultField.Interface(), nil
}

// Close closes the tunnel client
// Note: Actual tunnel cleanup is handled by the tunnel registry
func (c *Client) Close() error {
	// No-op for now as tunnel cleanup is handled by registry
	return nil
}
