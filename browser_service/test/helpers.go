package test

import (
	"context"
	"fmt"
	"net"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/sarathmenon/browser-service/pkg/protocol"
	"github.com/sarathmenon/browser-service/internal/server"
	"github.com/sarathmenon/browser-service/pkg/client"
)

// skipIfIntegrationTestsDisabled skips integration tests if SKIP_INTEGRATION_TESTS env var is set
func skipIfIntegrationTestsDisabled(t *testing.T) {
	t.Helper()
	if os.Getenv("SKIP_INTEGRATION_TESTS") != "" {
		t.Skip("Skipping integration test")
	}
}

// startTestServer starts a test server on a random port and returns the server, WebSocket URL, and cleanup function.
// This helper is shared between e2e tests and can be used by other test packages that need a running server.
func startTestServer(t *testing.T, ctx context.Context) (srv *server.Server, wsURL string, cleanup func()) {
	t.Helper()
	// Get a free port
	lc := net.ListenConfig{}
	listener, err := lc.Listen(ctx, "tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("Failed to get free port: %v", err)
	}
	tcpAddr, ok := listener.Addr().(*net.TCPAddr)
	if !ok {
		t.Fatalf("Failed to get TCP address")
	}
	port := tcpAddr.Port
	if err := listener.Close(); err != nil {
		t.Fatalf("Failed to close listener: %v", err)
	}

	// Create server
	srv, err = server.New(ctx, server.Config{
		Port:     fmt.Sprintf("%d", port),
		Headless: true,
	})
	if err != nil {
		t.Fatalf("Failed to create server: %v", err)
	}

	// Start server in background
	go func() {
		_ = srv.Start()
	}()

	// Wait for server to start
	time.Sleep(500 * time.Millisecond)

	wsURL = fmt.Sprintf("ws://127.0.0.1:%d/ws", port)

	cleanup = func() {
		if err := srv.Shutdown(ctx); err != nil {
			t.Errorf("Failed to shutdown server: %v", err)
		}
	}

	return srv, wsURL, cleanup
}

// setupE2ETest creates server, client, and context for E2E tests
// Returns command context, client, and cleanup function
func setupE2ETest(t *testing.T, timeoutSec int) (context.Context, *client.Client, func()) {
	t.Helper()
	ctx := context.Background()

	_, wsURL, serverCleanup := startTestServer(t, ctx)

	c, err := client.New(wsURL)
	if err != nil {
		serverCleanup()
		t.Fatalf("Failed to create client: %v", err)
	}

	cmdCtx, cancel := context.WithTimeout(ctx, time.Duration(timeoutSec)*time.Second)

	cleanup := func() {
		cancel()
		if err := c.Close(); err != nil {
			// Ignore "not connected" errors - expected during shutdown tests
			if !strings.Contains(err.Error(), "not connected") {
				t.Errorf("Failed to close client: %v", err)
			}
		}
		serverCleanup()
	}

	return cmdCtx, c, cleanup
}

// setupE2ETestWithServer returns server instance for shutdown tests
func setupE2ETestWithServer(t *testing.T, timeoutSec int) (context.Context, *server.Server, *client.Client, func()) {
	t.Helper()
	ctx := context.Background()

	srv, wsURL, serverCleanup := startTestServer(t, ctx)

	c, err := client.New(wsURL)
	if err != nil {
		serverCleanup()
		t.Fatalf("Failed to create client: %v", err)
	}

	cmdCtx, cancel := context.WithTimeout(ctx, time.Duration(timeoutSec)*time.Second)

	cleanup := func() {
		cancel()
		if err := c.Close(); err != nil {
			// Ignore "not connected" errors - expected during shutdown tests
			if !strings.Contains(err.Error(), "not connected") {
				t.Errorf("Failed to close client: %v", err)
			}
		}
		serverCleanup()
	}

	return cmdCtx, srv, c, cleanup
}

// findElementByRole finds the first element matching any of the given role(s)
// Returns array position (to use as index parameter) and true if found, -1 and false otherwise
func findElementByRole(elements []protocol.RawAccessibilityNode, roles ...string) (int, bool) {
	roleMap := make(map[string]bool)
	for _, r := range roles {
		roleMap[strings.ToLower(r)] = true
	}

	for i, elem := range elements {
		if roleMap[strings.ToLower(elem.Role)] {
			return i, true
		}
	}
	return -1, false
}
