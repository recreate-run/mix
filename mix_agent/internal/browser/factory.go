package browser

import (
	"context"
	"fmt"

	"mix/internal/browser/service"
)

const (
	// ModeTunnel uses Electron tunnel for browser communication
	ModeTunnel = "tunnel"

	// ModeService uses browser-service for browser communication
	ModeService = "service"

	// DefaultMode is the default browser mode
	DefaultMode = ModeService
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

// Client is the common interface for browser implementations
// Supports both browser-service and Electron tunnel implementations
type Client interface {
	// SendCommand sends a CDP command and waits for response
	SendCommand(ctx context.Context, method string, params interface{}) (interface{}, error)

	// Close closes the browser client connection
	Close() error
}

// FactoryConfig contains configuration for creating browser clients
type FactoryConfig struct {
	Mode              string      // "tunnel" or "service"
	BrowserServiceURL string      // URL for browser-service (used in service mode)
	TunnelRegistry    interface{} // Tunnel registry (used in tunnel mode)
	ConnectionManager interface{} // Connection manager (used in service mode)
}

// NewClient creates a browser client based on the provided mode and configuration
func NewClient(config FactoryConfig, sessionID string) (Client, error) {
	switch config.Mode {
	case ModeTunnel:
		if config.TunnelRegistry == nil {
			return nil, fmt.Errorf("tunnel registry required for tunnel mode")
		}
		return newTunnelClient(config.TunnelRegistry, sessionID), nil

	case ModeService:
		if config.ConnectionManager == nil {
			return nil, fmt.Errorf("connection manager required for service mode")
		}
		// Type assert the connection manager to the interface expected by service
		connMgr, ok := config.ConnectionManager.(service.ConnectionManager)
		if !ok {
			return nil, fmt.Errorf("invalid connection manager type")
		}
		return newServiceClient(connMgr, sessionID), nil

	default:
		return nil, fmt.Errorf("unknown browser mode: %s (must be 'tunnel' or 'service')", config.Mode)
	}
}
