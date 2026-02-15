package test

import (
	"context"
	"encoding/json"
	"fmt"
	"net"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/sarathmenon/browser-service/internal/server"
	"github.com/sarathmenon/browser-service/pkg/client"
	"github.com/sarathmenon/browser-service/pkg/protocol"
	"github.com/sarathmenon/browser-service/test/testserver"
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

	// Create server with default settings (modal blocking enabled by default)
	srv, err = server.New(ctx, server.Config{
		Port:        fmt.Sprintf("%d", port),
		Headless:    true,
		BlockModals: true, // Default: modal blocking enabled
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

// startTestServerWithStorageState starts a test server with a custom storage state path
func startTestServerWithStorageState(t *testing.T, ctx context.Context, storageStatePath string) (srv *server.Server, wsURL string, cleanup func()) {
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

	// Create server with storage state path (modal blocking enabled by default)
	srv, err = server.New(ctx, server.Config{
		Port:             fmt.Sprintf("%d", port),
		Headless:         true,
		StorageStatePath: storageStatePath,
		BlockModals:      true, // Default: modal blocking enabled
	})
	if err != nil {
		t.Fatalf("Failed to create server: %v", err)
	}

	// Start server in background
	go func() {
		_ = srv.Start()
	}()

	// Give server time to start
	time.Sleep(500 * time.Millisecond)

	wsURL = fmt.Sprintf("ws://127.0.0.1:%d/ws", port)

	cleanup = func() {
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		_ = srv.Shutdown(shutdownCtx)
	}

	return srv, wsURL, cleanup
}

// startTestServerWithBlockModals starts a test server with modal blocking enabled
func startTestServerWithBlockModals(t *testing.T, ctx context.Context) (srv *server.Server, wsURL string, cleanup func()) {
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

	// Create server with BlockModals enabled
	srv, err = server.New(ctx, server.Config{
		Port:        fmt.Sprintf("%d", port),
		Headless:    true,
		BlockModals: true,
	})
	if err != nil {
		t.Fatalf("Failed to create server: %v", err)
	}

	// Start server in background
	go func() {
		_ = srv.Start()
	}()

	// Give server time to start
	time.Sleep(500 * time.Millisecond)

	wsURL = fmt.Sprintf("ws://127.0.0.1:%d/ws", port)

	cleanup = func() {
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		_ = srv.Shutdown(shutdownCtx)
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

// setupE2ETestWithBlockModals creates server with modal blocking, client, and context for E2E tests
func setupE2ETestWithBlockModals(t *testing.T, timeoutSec int) (context.Context, *client.Client, func()) {
	t.Helper()
	ctx := context.Background()

	_, wsURL, serverCleanup := startTestServerWithBlockModals(t, ctx)

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

// findElementByRoleAndName finds the first element matching role and name
// Returns array position and true if found, -1 and false otherwise
func findElementByRoleAndName(elements []protocol.RawAccessibilityNode, role, name string) (int, bool) {
	roleLower := strings.ToLower(role)
	nameLower := strings.ToLower(name)

	for i, elem := range elements {
		if strings.ToLower(elem.Role) == roleLower && strings.ToLower(elem.Name) == nameLower {
			return i, true
		}
	}
	return -1, false
}

// setupTestServerAndBrowser starts both HTTP test server and browser service
// Returns context, HTTP server, client, and cleanup function
func setupTestServerAndBrowser(t *testing.T, timeoutSec int) (context.Context, *httptest.Server, *client.Client, func()) {
	t.Helper()

	server := testserver.StartTestServer(t)
	ctx, c, browserCleanup := setupE2ETest(t, timeoutSec)

	cleanup := func() {
		browserCleanup()
		server.Close()
	}

	return ctx, server, c, cleanup
}

// navigateAndWait navigates to URL and waits for page load
// Default wait is 500ms, can be overridden with waitMs parameter (0 uses default)
func navigateAndWait(ctx context.Context, c *client.Client, url string, waitMs int, tabID ...string) (*protocol.NavigateResult, error) {
	result, err := c.Navigate(ctx, url, tabID...)
	if err != nil {
		return nil, err
	}

	if waitMs == 0 {
		waitMs = 500
	}
	time.Sleep(time.Duration(waitMs) * time.Millisecond)

	return result, nil
}

// findCookie finds a cookie by name in the cookies list
// Returns the cookie and true if found, empty cookie and false otherwise
func findCookie(cookies []protocol.Cookie, name string) (protocol.Cookie, bool) {
	for _, cookie := range cookies {
		if cookie.Name == name {
			return cookie, true
		}
	}
	return protocol.Cookie{}, false
}

// assertCookieExists verifies that a cookie with given name and value exists
func assertCookieExists(t *testing.T, c *client.Client, ctx context.Context, name, expectedValue string) {
	t.Helper()

	cookies, err := c.GetCookies(ctx)
	if err != nil {
		t.Fatalf("Failed to get cookies: %v", err)
	}

	cookie, found := findCookie(cookies.Cookies, name)
	if !found {
		t.Errorf("Cookie '%s' not found", name)
		return
	}

	if cookie.Value != expectedValue {
		t.Errorf("Cookie '%s' has value '%s', expected '%s'", name, cookie.Value, expectedValue)
	}
}

// assertCookieNotExists verifies that a cookie with given name does not exist
func assertCookieNotExists(t *testing.T, c *client.Client, ctx context.Context, name string) {
	t.Helper()

	cookies, err := c.GetCookies(ctx)
	if err != nil {
		t.Fatalf("Failed to get cookies: %v", err)
	}

	if _, found := findCookie(cookies.Cookies, name); found {
		t.Errorf("Cookie '%s' should not exist", name)
	}
}

// setupFreshSession creates a new isolated browser session
// Useful for multi-session tests where you need to start from scratch
func setupFreshSession(t *testing.T, timeoutSec int) (context.Context, *client.Client, func()) {
	t.Helper()
	return setupE2ETest(t, timeoutSec)
}

// waitForEvent waits for a specific event type from the event channel
// Returns the event params as a map if found, or fails the test on timeout
func waitForEvent(t *testing.T, eventChan <-chan protocol.Event, eventType string, timeout time.Duration) map[string]interface{} {
	t.Helper()

	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	for {
		select {
		case event, ok := <-eventChan:
			if !ok {
				t.Fatal("Event channel closed before receiving expected event")
				return nil
			}

			if event.Method == eventType {
				// Convert params to map
				paramsBytes, err := json.Marshal(event.Params)
				if err != nil {
					t.Logf("Failed to marshal event params: %v", err)
					continue
				}

				var params map[string]interface{}
				if err := json.Unmarshal(paramsBytes, &params); err != nil {
					t.Logf("Failed to unmarshal event params: %v", err)
					continue
				}

				return params
			}

		case <-ctx.Done():
			t.Fatalf("Timeout waiting for %s event after %v", eventType, timeout)
			return nil
		}
	}
}

// waitForBrowserErrorEvent waits for a Browser.errorOccurred event with specific error type
// Returns the parsed event params or fails the test on timeout
func waitForBrowserErrorEvent(t *testing.T, eventChan <-chan protocol.Event, errorType string, timeout time.Duration) *protocol.BrowserErrorEventParams {
	t.Helper()

	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	for {
		select {
		case event, ok := <-eventChan:
			if !ok {
				t.Fatal("Event channel closed before receiving expected event")
				return nil
			}

			if event.Method != "Browser.errorOccurred" {
				continue
			}

			// Parse event params
			paramsBytes, err := json.Marshal(event.Params)
			if err != nil {
				t.Logf("Failed to marshal event params: %v", err)
				continue
			}

			var params protocol.BrowserErrorEventParams
			if err := json.Unmarshal(paramsBytes, &params); err != nil {
				t.Logf("Failed to unmarshal event params: %v", err)
				continue
			}

			t.Logf("Received browser event: %s - %+v", params.ErrorType, params.Details)

			if params.ErrorType == errorType {
				return &params
			}

		case <-ctx.Done():
			t.Fatalf("Timeout waiting for %s event after %v", errorType, timeout)
			return nil
		}
	}
}

// saveAndRestoreStorageState tests storage state round-trip: save, clear, and reload
// Returns the saved storage state for further verification
func saveAndRestoreStorageState(t *testing.T, c *client.Client, ctx context.Context) protocol.StorageState {
	t.Helper()

	// Save storage state
	saved, err := c.SaveStorageState(ctx)
	if err != nil {
		t.Fatalf("Failed to save storage state: %v", err)
	}

	// Clear cookies
	_, err = c.ClearCookies(ctx)
	if err != nil {
		t.Fatalf("Failed to clear cookies: %v", err)
	}

	// Load storage state back
	_, err = c.LoadStorageState(ctx, saved.State)
	if err != nil {
		t.Fatalf("Failed to load storage state: %v", err)
	}

	return saved.State
}
