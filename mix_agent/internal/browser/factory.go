package browser

import (
	"context"
	"fmt"
)

const (
	// ModeElectronEmbedded uses Electron app with embedded Chromium browser
	ModeElectronEmbedded = "electron-embedded-browser"

	// ModeLocalBrowserService uses local browser-service (GoRod-based)
	ModeLocalBrowserService = "local-browser-service"

	// ModeRemoteCDP connects to remote CDP WebSocket URL (cloud browser providers)
	ModeRemoteCDP = "remote-cdp-websocket"

	// DefaultMode is the default browser mode
	DefaultMode = ModeLocalBrowserService
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
// Supports Electron embedded browser, local browser-service, and remote CDP
type Client interface {
	// SendCommand sends a CDP command and waits for response
	SendCommand(ctx context.Context, method string, params interface{}) (interface{}, error)

	// Close closes the browser client connection
	Close() error
}

// FactoryConfig contains configuration for creating browser clients
type FactoryConfig struct {
	Mode              string      // "electron-embedded-browser", "local-browser-service", or "remote-cdp-websocket"
	BrowserServiceURL string      // URL for browser-service (used in local-browser-service mode)
	TunnelRegistry    interface{} // Tunnel registry (used in electron-embedded-browser mode)
	ConnectionManager interface{} // Connection manager (used in local-browser-service mode)
	CDPURL            string      // CDP WebSocket URL (used in remote-cdp-websocket mode)
}

// NewClient creates a browser client based on the provided mode and configuration
func NewClient(config FactoryConfig, sessionID string) (Client, error) {
	switch config.Mode {
	case ModeElectronEmbedded:
		if config.TunnelRegistry == nil {
			return nil, fmt.Errorf("tunnel registry required for electron-embedded-browser mode")
		}
		return newTunnelClient(config.TunnelRegistry, sessionID), nil

	case ModeLocalBrowserService:
		// Local browser-service mode uses connection manager directly, not factory pattern
		// The browser tool calls connectionManager.GetOrCreate() directly for this mode
		return nil, fmt.Errorf("local-browser-service mode should use connection manager directly, not factory")

	case ModeRemoteCDP:
		if config.CDPURL == "" {
			return nil, fmt.Errorf("CDP URL required for remote-cdp-websocket mode")
		}
		return nil, fmt.Errorf("remote CDP mode not yet implemented")

	default:
		return nil, fmt.Errorf("unknown browser mode: %s (must be 'electron-embedded-browser', 'local-browser-service', or 'remote-cdp-websocket')", config.Mode)
	}
}
