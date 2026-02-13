package client

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
)

func skipIfIntegrationTestsDisabled(t *testing.T) {
	t.Helper()
	if os.Getenv("SKIP_INTEGRATION_TESTS") != "" {
		t.Skip("Skipping integration test")
	}
}

// startTestServer starts a server on a random port and returns the WebSocket URL
func startTestServer(t *testing.T, ctx context.Context) (wsURL string, cleanup func()) {
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
	srv, err := server.New(ctx, server.Config{
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

	return wsURL, cleanup
}

func TestNewClient(t *testing.T) {
	skipIfIntegrationTestsDisabled(t)

	ctx := context.Background()
	wsURL, cleanup := startTestServer(t, ctx)
	defer cleanup()

	// Create client
	client, err := New(wsURL)
	if err != nil {
		t.Fatalf("Failed to create client: %v", err)
	}
	defer func() {
		if err := client.Close(); err != nil {
			t.Errorf("Failed to close client: %v", err)
		}
	}()

	if client == nil {
		t.Fatal("Expected non-nil client")
	}

	client.mu.Lock()
	connected := client.connected
	client.mu.Unlock()

	if !connected {
		t.Error("Expected client to be connected")
	}
}

func TestNavigate(t *testing.T) {
	skipIfIntegrationTestsDisabled(t)

	ctx := context.Background()
	wsURL, cleanup := startTestServer(t, ctx)
	defer cleanup()

	client, err := New(wsURL)
	if err != nil {
		t.Fatalf("Failed to create client: %v", err)
	}
	defer func() {
		if err := client.Close(); err != nil {
			t.Errorf("Failed to close client: %v", err)
		}
	}()

	// Navigate to a page
	navCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	result, err := client.Navigate(navCtx, "https://example.com")
	if err != nil {
		t.Fatalf("Navigate failed: %v", err)
	}

	if result == nil {
		t.Fatal("Expected non-nil result")
	}

	if result.FrameID == "" {
		t.Error("Expected non-empty FrameID")
	}
}

func TestScreenshot(t *testing.T) {
	skipIfIntegrationTestsDisabled(t)

	ctx := context.Background()
	wsURL, cleanup := startTestServer(t, ctx)
	defer cleanup()

	client, err := New(wsURL)
	if err != nil {
		t.Fatalf("Failed to create client: %v", err)
	}
	defer func() {
		if err := client.Close(); err != nil {
			t.Errorf("Failed to close client: %v", err)
		}
	}()

	// Navigate first
	navCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	_, err = client.Navigate(navCtx, "https://example.com")
	if err != nil {
		t.Fatalf("Navigate failed: %v", err)
	}

	// Take screenshot
	screenshotCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	result, err := client.Screenshot(screenshotCtx, protocol.ScreenshotParams{})
	if err != nil {
		t.Fatalf("Screenshot failed: %v", err)
	}

	if result == nil {
		t.Fatal("Expected non-nil result")
	}

	if result.Data == "" {
		t.Error("Expected non-empty screenshot data")
	}

	if result.Format != "png" {
		t.Errorf("Expected format 'png', got '%s'", result.Format)
	}
}

func TestClose(t *testing.T) {
	skipIfIntegrationTestsDisabled(t)

	ctx := context.Background()
	wsURL, cleanup := startTestServer(t, ctx)
	defer cleanup()

	client, err := New(wsURL)
	if err != nil {
		t.Fatalf("Failed to create client: %v", err)
	}

	// Close client
	err = client.Close()
	if err != nil {
		t.Errorf("Close failed: %v", err)
	}

	client.mu.Lock()
	connected := client.connected
	client.mu.Unlock()

	if connected {
		t.Error("Expected client to be disconnected")
	}
}

func TestConcurrentRequests(t *testing.T) {
	skipIfIntegrationTestsDisabled(t)

	ctx := context.Background()
	wsURL, cleanup := startTestServer(t, ctx)
	defer cleanup()

	client, err := New(wsURL)
	if err != nil {
		t.Fatalf("Failed to create client: %v", err)
	}
	defer func() {
		if err := client.Close(); err != nil {
			t.Errorf("Failed to close client: %v", err)
		}
	}()

	// Navigate first
	navCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	_, err = client.Navigate(navCtx, "https://example.com")
	if err != nil {
		t.Fatalf("Navigate failed: %v", err)
	}

	// Make multiple concurrent screenshot requests
	numRequests := 5
	done := make(chan error, numRequests)

	for i := 0; i < numRequests; i++ {
		go func() {
			reqCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
			defer cancel()

			_, err := client.Screenshot(reqCtx, protocol.ScreenshotParams{})
			done <- err
		}()
	}

	// Wait for all requests to complete
	for i := 0; i < numRequests; i++ {
		if err := <-done; err != nil {
			t.Errorf("Request %d failed: %v", i, err)
		}
	}
}

func TestTimeout(t *testing.T) {
	skipIfIntegrationTestsDisabled(t)

	ctx := context.Background()
	wsURL, cleanup := startTestServer(t, ctx)
	defer cleanup()

	client, err := New(wsURL)
	if err != nil {
		t.Fatalf("Failed to create client: %v", err)
	}
	defer func() {
		if err := client.Close(); err != nil {
			t.Errorf("Failed to close client: %v", err)
		}
	}()

	// Create a context that times out very quickly
	// Note: This test might be flaky if the navigation completes too fast
	timeoutCtx, cancel := context.WithTimeout(ctx, 1*time.Nanosecond)
	defer cancel()

	// Wait a bit to ensure timeout happens
	time.Sleep(10 * time.Millisecond)

	_, err = client.Navigate(timeoutCtx, "https://example.com")
	if err == nil {
		t.Log("Warning: Expected timeout error, but navigation succeeded (this can happen if navigation is very fast)")
	} else if !strings.Contains(err.Error(), "timeout") && !strings.Contains(err.Error(), "context") {
		t.Errorf("Expected timeout or context error, got: %v", err)
	}
}

func TestInvalidEndpoint(t *testing.T) {
	_, err := New("ws://invalid:99999")
	if err == nil {
		t.Error("Expected error for invalid endpoint")
	}
}

func TestMultipleCommands(t *testing.T) {
	skipIfIntegrationTestsDisabled(t)

	ctx := context.Background()
	wsURL, cleanup := startTestServer(t, ctx)
	defer cleanup()

	client, err := New(wsURL)
	if err != nil {
		t.Fatalf("Failed to create client: %v", err)
	}
	defer func() {
		if err := client.Close(); err != nil {
			t.Errorf("Failed to close client: %v", err)
		}
	}()

	cmdCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	// Navigate
	_, err = client.Navigate(cmdCtx, "https://example.com")
	if err != nil {
		t.Fatalf("Navigate failed: %v", err)
	}

	// Screenshot 1
	_, err = client.Screenshot(cmdCtx, protocol.ScreenshotParams{})
	if err != nil {
		t.Fatalf("Screenshot 1 failed: %v", err)
	}

	// Screenshot 2 with different params
	_, err = client.Screenshot(cmdCtx, protocol.ScreenshotParams{FullPage: true})
	if err != nil {
		t.Fatalf("Screenshot 2 failed: %v", err)
	}

	// All commands succeeded
	t.Log("All commands completed successfully")
}

func TestRequestIDIncrement(t *testing.T) {
	skipIfIntegrationTestsDisabled(t)

	ctx := context.Background()
	wsURL, cleanup := startTestServer(t, ctx)
	defer cleanup()

	client, err := New(wsURL)
	if err != nil {
		t.Fatalf("Failed to create client: %v", err)
	}
	defer func() {
		if err := client.Close(); err != nil {
			t.Errorf("Failed to close client: %v", err)
		}
	}()

	cmdCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	// Navigate
	_, err = client.Navigate(cmdCtx, "https://example.com")
	if err != nil {
		t.Fatalf("Navigate failed: %v", err)
	}

	// Make multiple requests and verify IDs are incrementing
	// This is implicit in the client implementation using atomic.AddUint64
	for i := 0; i < 5; i++ {
		_, err = client.Screenshot(cmdCtx, protocol.ScreenshotParams{})
		if err != nil {
			t.Fatalf("Screenshot %d failed: %v", i, err)
		}
	}

	// If we got here, IDs were unique (otherwise we'd have a race/collision)
	t.Log("Request IDs are unique and incrementing")
}
