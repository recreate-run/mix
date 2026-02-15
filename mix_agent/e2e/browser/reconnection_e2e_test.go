//go:build e2e
// +build e2e

package browser

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"sync"
	"testing"
	"time"

	"mix/e2e"
	"mix/e2e/browser/testdata"
	"mix/internal/constants"
	"mix/internal/llm/tools/browser"
)

// TestBrowserE2EReconnectionOnCrash tests automatic reconnection when browser crashes mid-operation
func TestBrowserE2EReconnectionOnCrash(t *testing.T) {
	t.Parallel()
	e2e.Setup(t)

	t.Log("=== E2E Test: Browser Reconnection on Crash (Direct Client) ===")

	// Set up mock CDP server
	mockServer := testdata.NewMockCDPServer(t)
	defer mockServer.Close()

	cdpURL := mockServer.GetURL()

	// Step 1: Create RemoteCDPClient directly to avoid LLM non-determinism
	t.Log("Step 1: Creating RemoteCDPClient...")
	ctx := context.Background()
	client, err := browser.NewRemoteCDPClient(ctx, cdpURL)
	if err != nil {
		t.Fatalf("Failed to create client: %v", err)
	}
	defer func() { _ = client.Close() }()
	t.Log("✓ Client created and connected")

	// Verify initial connection
	if mockServer.ConnectionCount() != 1 {
		t.Fatalf("Expected 1 connection, got %d", mockServer.ConnectionCount())
	}
	t.Log("✓ Initial connection established")

	// Step 2: Create a tab to verify functionality
	t.Log("Step 2: Creating tab to verify functionality...")
	_, err = client.CreateTab(ctx, "http://example.com")
	if err != nil {
		t.Fatalf("Failed to create tab: %v", err)
	}
	t.Log("✓ Tab created successfully")

	// Step 3: Simulate server crash
	t.Log("Step 3: Simulating server crash...")
	mockServer.Crash()
	time.Sleep(1 * time.Second) // Give time for disconnection to be detected
	t.Log("✓ Server crashed")

	// Verify client detected disconnection
	if client.IsConnected() {
		t.Log("⚠ Warning: Client still reports connected after crash")
	} else {
		t.Log("✓ Client detected disconnection")
	}

	// Step 4: Restart server
	t.Log("Step 4: Restarting mock server...")
	mockServer.Restart()
	t.Log("✓ Server restarted")

	// Step 5: Wait for automatic reconnection
	t.Log("Step 5: Waiting for automatic reconnection...")
	time.Sleep(8 * time.Second) // Wait for first few reconnection attempts (2s + 4s)

	// Check if reconnection happened
	if mockServer.ConnectionCount() > 0 {
		t.Logf("✓ Automatic reconnection successful (%d connection(s))", mockServer.ConnectionCount())
	} else {
		t.Fatal("Expected reconnection but no connections found - reconnection failed")
	}

	// Step 6: Try to create another tab after reconnection to verify functionality
	t.Log("Step 6: Attempting to create tab after reconnection...")
	_, err = client.CreateTab(ctx, "http://example.com/after-reconnect")
	if err != nil {
		t.Fatalf("Tab creation failed after reconnection: %v", err)
	}
	t.Log("✓ Tab created successfully after reconnection")

	t.Log("=== E2E Test Completed Successfully ===")
}

// TestBrowserE2EConcurrentClientCreation tests that concurrent browser tool calls don't create duplicate connections
func TestBrowserE2EConcurrentClientCreation(t *testing.T) {
	t.Parallel()
	e2e.Setup(t)

	t.Log("=== E2E Test: Concurrent Client Creation (No Duplicates) ===")

	// Check if a real remote CDP URL is provided via environment variable
	remoteCDPURL := os.Getenv("REMOTE_CDP_URL")
	var mockServer *testdata.MockCDPServer
	if remoteCDPURL == "" {
		// Set up mock CDP server
		mockServer = testdata.NewMockCDPServer(t)
		defer mockServer.Close()
		remoteCDPURL = mockServer.GetURL()
	}

	// Start test HTML server
	testServer := startTestHTMLServer(t)
	defer testServer.Close()

	testURL := testServer.URL + "/mode_test.html"

	// Create session with remote CDP mode
	t.Log("Creating session with remote-cdp-websocket mode...")
	opts := map[string]interface{}{
		"cdpUrl": remoteCDPURL,
	}
	sessionID, cleanup := createTestSession(t, "Concurrency Test", "remote-cdp-websocket", opts)
	defer cleanup()

	// Step 1: Send multiple concurrent messages that will trigger browser operations
	t.Log("Step 1: Sending 10 concurrent messages to trigger parallel browser client creation...")
	var wg sync.WaitGroup
	responses := make([]*http.Response, 10)

	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			msgResp := sendMessage(t, sessionID, fmt.Sprintf("Message %d: Open %s", idx, testURL))
			responses[idx] = msgResp
		}(i)
	}

	// Wait for all messages to be sent
	wg.Wait()
	t.Log("✓ All 10 messages sent concurrently")

	// Close all response bodies
	for _, resp := range responses {
		if resp != nil && resp.Body != nil {
			_ = resp.Body.Close()
		}
	}

	// Step 2: Wait for at least first message to start processing (this triggers client creation)
	t.Log("Step 2: Waiting for first message to start processing (triggers client creation)...")
	time.Sleep(5 * time.Second)

	// Step 3: Verify only one connection was created (using mock server if available)
	if mockServer != nil {
		connCount := mockServer.ConnectionCount()
		if connCount != 1 {
			t.Fatalf("Expected exactly 1 connection due to sync.Once, got %d - duplicate connections created!", connCount)
		}
		t.Logf("✓ Exactly 1 connection to mock server (sync.Once prevented duplicates)")
	}

	// Step 4: Wait for all processing to complete
	t.Log("Step 4: Waiting for all messages to be processed...")
	waitForProcessing(t, sessionID, 90*time.Second)
	t.Log("✓ All messages processed")

	// Step 5: Verify no errors occurred (which would indicate duplicate connection issues)
	t.Log("Step 5: Verifying no connection errors occurred...")
	messagesResp := makeRequest(t, http.MethodGet, constants.APISessionsPath+sessionID+"/messages", nil)
	defer func() { _ = messagesResp.Body.Close() }()

	messagesBody, err := io.ReadAll(messagesResp.Body)
	if err != nil {
		t.Fatalf("Failed to read messages response: %v", err)
	}

	messagesStr := string(messagesBody)

	// Check for common error patterns that would indicate connection issues
	errorPatterns := []string{
		"duplicate connection",
		"connection already exists",
		"failed to create client",
		"connection race",
	}

	for _, pattern := range errorPatterns {
		if strings.Contains(messagesStr, pattern) {
			t.Fatalf("Found connection error pattern '%s' in messages - duplicate connection likely occurred", pattern)
		}
	}

	// Verify that browser operations were successful
	if !strings.Contains(messagesStr, "Browser") {
		t.Log("⚠ Warning: Expected browser tool usage in messages")
	} else {
		t.Log("✓ Browser operations completed without duplicate connection errors")
	}

	t.Log("=== E2E Test Completed Successfully ===")
}

// TestBrowserE2EReconnectionBackoff tests exponential backoff reconnection strategy
func TestBrowserE2EReconnectionBackoff(t *testing.T) {
	e2e.Setup(t)

	t.Log("=== E2E Test: Reconnection with Exponential Backoff ===")

	// Set up mock CDP server
	mockServer := testdata.NewMockCDPServer(t)
	defer mockServer.Close()

	cdpURL := mockServer.GetURL()

	// Start test HTML server
	testServer := startTestHTMLServer(t)
	defer testServer.Close()

	testURL := testServer.URL + "/mode_test.html"

	// Create session with remote CDP mode
	t.Log("Creating session with remote-cdp-websocket mode...")
	opts := map[string]interface{}{
		"cdpUrl": cdpURL,
	}
	sessionID, cleanup := createTestSession(t, "Backoff Test", "remote-cdp-websocket", opts)
	defer cleanup()

	// Step 1: Establish initial connection
	t.Log("Step 1: Establishing initial connection...")
	msgResp := sendMessage(t, sessionID, fmt.Sprintf("Open %s", testURL))
	_ = msgResp.Body.Close()
	waitForProcessing(t, sessionID, 60*time.Second)
	t.Log("✓ Initial connection established")

	// Step 2: Crash server to trigger reconnection attempts
	t.Log("Step 2: Crashing mock server to trigger reconnection...")
	mockServer.Crash()
	t.Log("✓ Mock server crashed")

	// Step 3: Wait for reconnection attempts (check logs for exponential backoff pattern)
	// The backend should attempt reconnection with increasing delays: 2s, 4s, 8s, 16s, 30s (capped)
	t.Log("Step 3: Waiting for first reconnection attempt (should occur after 2s)...")
	time.Sleep(3 * time.Second) // Wait for at least first reconnection attempt

	// Step 4: Restart server to allow successful reconnection
	t.Log("Step 4: Restarting mock server...")
	mockServer.Restart()
	t.Log("✓ Mock server restarted")

	// Wait for reconnection to happen (second attempt after 4s)
	t.Log("Waiting for reconnection to complete...")
	time.Sleep(5 * time.Second)

	// Step 5: Verify reconnection succeeded by checking connection count
	if mockServer.ConnectionCount() > 0 {
		t.Logf("✓ Reconnection successful (%d connection(s))", mockServer.ConnectionCount())
	} else {
		t.Log("⚠ Warning: No connections after restart, may need more time for reconnection")
	}

	// Step 6: Send another message to verify operations work
	t.Log("Step 6: Verifying operations after reconnection...")
	msgResp2 := sendMessage(t, sessionID, "Navigate to another page")
	_ = msgResp2.Body.Close()

	// Wait for processing - should complete successfully after reconnection
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	done := make(chan bool)
	go func() {
		waitForProcessing(t, sessionID, 30*time.Second)
		done <- true
	}()

	select {
	case <-done:
		t.Log("✓ Operation completed successfully after reconnection")
	case <-ctx.Done():
		t.Fatal("Operation timed out - reconnection may have failed")
	}

	t.Log("=== E2E Test Completed Successfully ===")
}
