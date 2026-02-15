package browser

import (
	"context"
	"fmt"
	"sync"
	"time"

	browserclient "github.com/sarathmenon/browser-service/pkg/client"
)

// ConnectionManager manages WebSocket connections to browser service per session
type ConnectionManager struct {
	mu          sync.RWMutex
	connections map[string]BrowserClient // sessionID → client (wrapped in adapter)
	endpoint    string
}

// NewConnectionManager creates a new connection manager
func NewConnectionManager(endpoint string) *ConnectionManager {
	return &ConnectionManager{
		connections: make(map[string]BrowserClient),
		endpoint:    endpoint,
	}
}

// GetOrCreate returns an existing connection or creates a new one for the session
func (cm *ConnectionManager) GetOrCreate(ctx context.Context, sessionID string) (BrowserClient, error) {
	// First try to get existing connection with read lock
	cm.mu.RLock()
	client, exists := cm.connections[sessionID]
	cm.mu.RUnlock()

	if exists && cm.isConnected(ctx, client) {
		return client, nil
	}

	// Need to create or recreate connection - acquire write lock
	cm.mu.Lock()
	defer cm.mu.Unlock()

	// Double-check after acquiring write lock (another goroutine may have created it)
	if client, exists := cm.connections[sessionID]; exists && cm.isConnected(ctx, client) {
		return client, nil
	}

	// Remove stale connection if exists
	if client != nil {
		_ = client.Close()
		delete(cm.connections, sessionID)
	}

	// Create new external client and wrap in adapter
	externalClient, err := browserclient.New(cm.endpoint)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to browser service: %w", err)
	}

	// Wrap external client in adapter to implement BrowserClient interface
	adapter := NewServiceClientAdapter(externalClient)
	cm.connections[sessionID] = adapter
	return adapter, nil
}

// Close closes and removes the connection for a session
func (cm *ConnectionManager) Close(sessionID string) error {
	cm.mu.Lock()
	defer cm.mu.Unlock()

	client, exists := cm.connections[sessionID]
	if !exists {
		return nil
	}

	delete(cm.connections, sessionID)
	return client.Close()
}

// isConnected tests if a client connection is still alive
// Uses a lightweight ListTabs call to test the connection
func (cm *ConnectionManager) isConnected(ctx context.Context, client BrowserClient) bool {
	if client == nil {
		return false
	}

	// Check if client has IsConnected method (for RemoteCDPClient)
	type connectedChecker interface {
		IsConnected() bool
	}
	if checker, ok := client.(connectedChecker); ok {
		if !checker.IsConnected() {
			return false
		}
	}

	// Quick context with short timeout for liveness check
	checkCtx, cancel := context.WithTimeout(ctx, 2*time.Second)
	defer cancel()

	// Try a lightweight operation to check if connection is alive
	// ListTabs is much faster than ReadPage (no accessibility tree traversal)
	_, err := client.ListTabs(checkCtx)
	return err == nil
}
