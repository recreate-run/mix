package service

import (
	"context"
	"fmt"
)

// ConnectionManager interface to avoid circular import
// Implemented by browser.ConnectionManager
type ConnectionManager interface {
	GetOrCreate(ctx context.Context, sessionID string) (interface{}, error)
}

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

// Client provides browser-service CDP communication
// This is a stub implementation for future browser-service integration
// Implements browser.Client interface via duck typing
type Client struct {
	connectionManager ConnectionManager
	sessionID         string
}

// NewClient creates a new browser-service client
func NewClient(connectionManager ConnectionManager, sessionID string) *Client {
	return &Client{
		connectionManager: connectionManager,
		sessionID:         sessionID,
	}
}

// SendCommand sends a CDP command through browser-service
// TODO: Implement actual browser-service CDP command support
func (c *Client) SendCommand(ctx context.Context, method string, params interface{}) (interface{}, error) {
	// Get or create connection using connection manager
	_, err := c.connectionManager.GetOrCreate(ctx, c.sessionID)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to browser service: %w", err)
	}

	// TODO: Implement browser-service CDP command forwarding
	// This will require extending browser-service to support raw CDP commands
	return nil, fmt.Errorf("browser-service CDP commands not yet implemented")
}

// Close closes the browser-service client
func (c *Client) Close() error {
	// Connection is managed by ConnectionManager, no cleanup needed here
	return nil
}
