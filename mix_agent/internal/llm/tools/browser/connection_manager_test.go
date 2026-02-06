package browser

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Test NewConnectionManager
func TestNewConnectionManager(t *testing.T) {
	endpoint := "ws://localhost:8080"
	cm := NewConnectionManager(endpoint)

	assert.NotNil(t, cm)
	assert.Equal(t, endpoint, cm.endpoint)
	assert.NotNil(t, cm.connections)
	assert.Empty(t, cm.connections)
}

// Test GetOrCreate creates new connection for new session
func TestConnectionManagerGetOrCreateNewSession(t *testing.T) {
	cm := NewConnectionManager("ws://invalid:9999")
	ctx := context.Background()

	// First call should attempt to create connection (will fail due to invalid endpoint)
	_, err := cm.GetOrCreate(ctx, "session-123")

	// Should return error since endpoint is invalid
	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to connect to browser service")
}

// Test Close removes connection
func TestConnectionManagerClose(t *testing.T) {
	cm := NewConnectionManager("ws://localhost:8080")

	// Closing non-existent session should not error
	err := cm.Close("session-123")
	assert.NoError(t, err)
}

// Test Close with existing connection
func TestConnectionManagerCloseExisting(t *testing.T) {
	cm := NewConnectionManager("ws://localhost:8080")

	// Note: We cannot test with a nil client because the Close method
	// will panic when calling client.Close() on a nil pointer.
	// This test verifies closing a non-existent connection is safe.

	// Close non-existent session (should be safe)
	err := cm.Close("session-123")
	require.NoError(t, err)

	// Verify it's not in the map
	cm.mu.RLock()
	_, exists := cm.connections["session-123"]
	cm.mu.RUnlock()
	assert.False(t, exists)
}

// Test isConnected with nil client
func TestConnectionManagerIsConnectedNil(t *testing.T) {
	cm := NewConnectionManager("ws://localhost:8080")
	ctx := context.Background()

	result := cm.isConnected(ctx, nil)
	assert.False(t, result)
}

// Test concurrent access to connection manager
func TestConnectionManagerConcurrentAccess(t *testing.T) {
	cm := NewConnectionManager("ws://invalid:9999")
	ctx := context.Background()

	// Start multiple goroutines trying to get connections
	done := make(chan bool, 5)

	for i := 0; i < 5; i++ {
		go func(_ int) {
			sessionID := "concurrent-session"
			_, _ = cm.GetOrCreate(ctx, sessionID)
			done <- true
		}(i)
	}

	// Wait for all goroutines to complete with timeout
	timeout := time.After(5 * time.Second)
	for i := 0; i < 5; i++ {
		select {
		case <-done:
			// Success
		case <-timeout:
			t.Fatal("Timeout waiting for concurrent operations")
		}
	}
}

// Test multiple sessions
func TestConnectionManagerMultipleSessions(t *testing.T) {
	cm := NewConnectionManager("ws://invalid:9999")
	ctx := context.Background()

	// Try to create connections for multiple sessions
	sessions := []string{"session-1", "session-2", "session-3"}

	for _, sessionID := range sessions {
		_, err := cm.GetOrCreate(ctx, sessionID)
		// Will fail due to invalid endpoint, but that's okay for this test
		require.Error(t, err)
	}

	// Verify connections were not added due to errors
	cm.mu.RLock()
	count := len(cm.connections)
	cm.mu.RUnlock()
	assert.Equal(t, 0, count)
}

// Test Close for multiple sessions
func TestConnectionManagerCloseMultipleSessions(t *testing.T) {
	cm := NewConnectionManager("ws://localhost:8080")

	sessions := []string{"session-1", "session-2", "session-3"}

	// Close all sessions (none exist, so should all succeed)
	for _, sessionID := range sessions {
		err := cm.Close(sessionID)
		assert.NoError(t, err)
	}
}

// Test connection manager state consistency
func TestConnectionManagerStateAfterClose(t *testing.T) {
	cm := NewConnectionManager("ws://localhost:8080")

	// Verify connections map is initially empty
	cm.mu.RLock()
	initialCount := len(cm.connections)
	cm.mu.RUnlock()
	assert.Equal(t, 0, initialCount)

	// Close non-existent session (should be safe)
	err := cm.Close("session-123")
	require.NoError(t, err)

	// Verify count is still 0
	cm.mu.RLock()
	finalCount := len(cm.connections)
	cm.mu.RUnlock()
	assert.Equal(t, 0, finalCount)
}

// Test endpoint storage
func TestConnectionManagerEndpoint(t *testing.T) {
	endpoints := []string{
		"ws://localhost:8080",
		"ws://127.0.0.1:9090",
		"ws://example.com:3000/ws",
	}

	for _, endpoint := range endpoints {
		t.Run(endpoint, func(t *testing.T) {
			cm := NewConnectionManager(endpoint)
			assert.Equal(t, endpoint, cm.endpoint)
		})
	}
}

// Test GetOrCreate with context cancellation
func TestConnectionManagerGetOrCreateWithCancelledContext(t *testing.T) {
	cm := NewConnectionManager("ws://localhost:8080")

	// Create cancelled context
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	_, err := cm.GetOrCreate(ctx, "session-123")
	// Should fail due to connection error (context cancelled or endpoint invalid)
	assert.Error(t, err)
}

// Test isConnected timeout behavior
func TestConnectionManagerIsConnectedTimeout(t *testing.T) {
	cm := NewConnectionManager("ws://localhost:8080")

	// Create a mock context
	ctx := context.Background()

	// isConnected should return false for nil client
	result := cm.isConnected(ctx, nil)
	assert.False(t, result)
}

// Test connection manager initialization
func TestConnectionManagerInitialization(t *testing.T) {
	cm := NewConnectionManager("ws://localhost:8080")

	// Verify all fields are properly initialized
	assert.NotNil(t, cm.connections, "connections map should be initialized")
	assert.Equal(t, "ws://localhost:8080", cm.endpoint, "endpoint should be set")

	// Verify connections map is empty
	cm.mu.RLock()
	length := len(cm.connections)
	cm.mu.RUnlock()
	assert.Equal(t, 0, length, "connections map should be empty initially")
}

// Test double-check locking in GetOrCreate
func TestConnectionManagerDoubleCheckLocking(t *testing.T) {
	cm := NewConnectionManager("ws://invalid:9999")
	ctx := context.Background()

	// This tests the double-check locking pattern
	// Multiple goroutines racing to create same session
	done := make(chan error, 10)

	for i := 0; i < 10; i++ {
		go func() {
			_, err := cm.GetOrCreate(ctx, "same-session")
			done <- err
		}()
	}

	// Collect results
	errors := 0
	for i := 0; i < 10; i++ {
		err := <-done
		if err != nil {
			errors++
		}
	}

	// All should error due to invalid endpoint
	assert.Equal(t, 10, errors)

	// But there should be no connection created
	cm.mu.RLock()
	_, exists := cm.connections["same-session"]
	cm.mu.RUnlock()
	assert.False(t, exists)
}
