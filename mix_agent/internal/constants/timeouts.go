package constants

import "time"

// Database operation timeouts
const (
	// DatabasePingTimeout is the timeout for database ping operations
	DatabasePingTimeout = 10 * time.Second

	// DatabaseConnectionTimeout is the timeout for establishing database connections
	DatabaseConnectionTimeout = 30 * time.Second

	// DatabaseMigrationTimeout is the timeout for running database migrations
	DatabaseMigrationTimeout = 5 * time.Minute
)

// WebSocket operation timeouts
const (
	// WebSocketWriteTimeout is the timeout for WebSocket write operations
	WebSocketWriteTimeout = 10 * time.Second

	// WebSocketPongWait is the timeout for receiving pong responses
	WebSocketPongWait = 5 * time.Second

	// WebSocketPingInterval is the interval between sending pings
	WebSocketPingInterval = 30 * time.Second
)

// MCP (Model Context Protocol) operation timeouts
const (
	// MCPToolTimeout is the timeout for MCP tool operations
	MCPToolTimeout = 3 * time.Second

	// MCPInitTimeout is the timeout for initial MCP tools initialization
	MCPInitTimeout = 30 * time.Second
)

// Message processing timeouts
const (
	// MessageCleanupDelay is the delay before cleaning up accumulated messages
	MessageCleanupDelay = 5 * time.Second
)

// Query operation timeouts
const (
	// QueryExecutionTimeout is the timeout for executing queries
	QueryExecutionTimeout = 30 * time.Second
)
