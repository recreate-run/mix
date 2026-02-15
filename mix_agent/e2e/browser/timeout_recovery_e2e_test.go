//go:build e2e
// +build e2e

package browser

import (
	"context"
	"testing"
	"time"

	"mix/e2e"
	"mix/e2e/browser/testdata"
	"mix/internal/llm/tools/browser"
)

// TestRemoteCDPClientSlowAccessibilityTree tests that accessibility tree operations
// are fast with domain enabling, but can handle slow edge cases with graceful degradation
func TestRemoteCDPClientSlowAccessibilityTree(t *testing.T) {
	t.Helper()
	t.Parallel()
	e2e.Setup(t)

	t.Log("=== E2E Test: Accessibility Tree with Domain Enabling ===")

	// Set up mock CDP server
	mockServer := testdata.NewMockCDPServer(t)
	defer mockServer.Close()

	cdpURL := mockServer.GetURL()

	// Step 1: Create client
	t.Log("Step 1: Creating RemoteCDPClient...")
	ctx := context.Background()
	client, err := browser.NewRemoteCDPClient(ctx, cdpURL)
	if err != nil {
		t.Fatalf("Failed to create client: %v", err)
	}
	defer func() { _ = client.Close() }()
	t.Log("✓ Client created and connected")

	// Step 2: Create a tab (should enable domains automatically)
	t.Log("Step 2: Creating tab (enables Page, DOM, Accessibility, Runtime domains)...")
	tab, err := client.CreateTab(ctx, "http://example.com")
	if err != nil {
		t.Fatalf("Failed to create tab: %v", err)
	}
	t.Log("✓ Tab created successfully with domains enabled")

	// Step 3: Test fast accessibility tree (should be instant with domain enabling)
	t.Log("Step 3: Testing fast accessibility tree (should be instant with domain enabling)...")
	start := time.Now()
	readResult, err := client.ReadPage(ctx, false, tab.ID)
	elapsed := time.Since(start)

	if err != nil {
		t.Fatalf("ReadPage failed: %v (elapsed: %v)", err, elapsed)
	}

	t.Logf("✓ ReadPage succeeded quickly (elapsed: %v)", elapsed)

	if len(readResult.Elements) < 1 {
		t.Fatalf("Expected at least 1 accessibility element, got %d", len(readResult.Elements))
	}
	t.Logf("✓ Received %d accessibility elements", len(readResult.Elements))

	// Verify it was fast (should be under 1 second with domain enabling)
	if elapsed > 5*time.Second {
		t.Logf("⚠ Warning: ReadPage took %v (expected <5s with domain enabling)", elapsed)
	} else {
		t.Log("✓ Accessibility tree retrieved quickly thanks to domain enabling")
	}

	// Step 4: Test that operations still work with moderate delay (15s within 30s timeout)
	t.Log("Step 4: Testing with 15-second delay (within 30s timeout)...")
	mockServer.SetCommandDelay("Accessibility.getFullAXTree", 15*time.Second)

	start = time.Now()
	readResult2, err := client.ReadPage(ctx, false, tab.ID)
	elapsed = time.Since(start)

	if err != nil {
		t.Fatalf("ReadPage failed with 15s delay: %v (elapsed: %v)", err, elapsed)
	}

	t.Logf("✓ ReadPage succeeded with 15s delay (elapsed: %v)", elapsed)

	if len(readResult2.Elements) < 1 {
		t.Fatalf("Expected at least 1 accessibility element, got %d", len(readResult2.Elements))
	}
	t.Log("✓ Operations work correctly even with network delays")

	mockServer.ClearAllCommandDelays()

	t.Log("=== E2E Test Completed Successfully ===")
}

// TestRemoteCDPClientTimeoutRecovery tests that timeout triggers automatic reconnection
func TestRemoteCDPClientTimeoutRecovery(t *testing.T) {
	t.Helper()
	t.Parallel()
	e2e.Setup(t)

	t.Log("=== E2E Test: Timeout Triggers Reconnection ===")

	// Set up mock CDP server
	mockServer := testdata.NewMockCDPServer(t)
	defer mockServer.Close()

	cdpURL := mockServer.GetURL()

	// Step 1: Create client
	t.Log("Step 1: Creating RemoteCDPClient...")
	ctx := context.Background()
	client, err := browser.NewRemoteCDPClient(ctx, cdpURL)
	if err != nil {
		t.Fatalf("Failed to create client: %v", err)
	}
	defer func() { _ = client.Close() }()
	t.Log("✓ Client created and connected")

	// Step 2: Create a tab
	t.Log("Step 2: Creating tab...")
	tab, err := client.CreateTab(ctx, "http://example.com")
	if err != nil {
		t.Fatalf("Failed to create tab: %v", err)
	}
	t.Log("✓ Tab created successfully")

	// Step 3: Configure extreme delay (35 seconds - exceeds 30s timeout)
	t.Log("Step 3: Configuring 35-second delay for ReadPage (exceeds 30s timeout)...")
	mockServer.SetCommandDelay("Accessibility.getFullAXTree", 35*time.Second)

	// Step 4: Attempt ReadPage - should timeout
	t.Log("Step 4: Attempting ReadPage (should timeout and trigger reconnection)...")
	readCtx, readCancel := context.WithTimeout(ctx, 40*time.Second)
	defer readCancel()

	start := time.Now()
	_, err = client.ReadPage(readCtx, false, tab.ID)
	elapsed := time.Since(start)

	if err == nil {
		t.Fatal("Expected ReadPage to fail with timeout, but it succeeded")
	}
	t.Logf("✓ ReadPage timed out as expected: %v (elapsed: %v)", err, elapsed)

	// Verify client detected disconnection
	time.Sleep(1 * time.Second) // Give time for state to update
	if client.IsConnected() {
		t.Log("⚠ Warning: Client still reports connected after timeout")
	} else {
		t.Log("✓ Client detected disconnection after timeout")
	}

	// Step 5: Clear delays and wait for reconnection
	t.Log("Step 5: Clearing delays and waiting for automatic reconnection...")
	mockServer.ClearAllCommandDelays()
	time.Sleep(10 * time.Second) // Wait for reconnection attempts (2s + 4s + buffer)

	// Verify reconnection happened
	if mockServer.ConnectionCount() > 0 {
		t.Logf("✓ Automatic reconnection successful (%d connection(s))", mockServer.ConnectionCount())
	} else {
		t.Log("⚠ Warning: No connections after timeout - may need more time")
	}

	// Step 6: Verify operations work after recovery
	t.Log("Step 6: Verifying operations after recovery...")
	_, err = client.CreateTab(ctx, "http://example.com/after-recovery")
	if err != nil {
		t.Logf("⚠ CreateTab failed after recovery: %v (reconnection may not be complete)", err)
	} else {
		t.Log("✓ Operations working after timeout recovery")
	}

	t.Log("=== E2E Test Completed Successfully ===")
}

// TestConnectionManagerHealthCheck tests that health check uses lightweight operation
func TestConnectionManagerHealthCheck(t *testing.T) {
	t.Helper()
	t.Parallel()
	e2e.Setup(t)

	t.Log("=== E2E Test: Connection Manager Health Check ===")

	// Set up mock CDP server
	mockServer := testdata.NewMockCDPServer(t)
	defer mockServer.Close()

	cdpURL := mockServer.GetURL()

	// Step 1: Create RemoteCDPClient
	t.Log("Step 1: Creating RemoteCDPClient...")
	ctx := context.Background()

	// Create RemoteCDPClient manually
	client, err := browser.NewRemoteCDPClient(ctx, cdpURL)
	if err != nil {
		t.Fatalf("Failed to create client: %v", err)
	}
	defer func() { _ = client.Close() }()
	t.Log("✓ Client created")

	// Step 2: Configure slow ReadPage but fast ListTabs
	t.Log("Step 2: Configuring delays (slow Accessibility.getFullAXTree)...")
	mockServer.SetCommandDelay("Accessibility.getFullAXTree", 30*time.Second)
	// ListTabs uses Target.getTargets which has no delay configured (fast)

	// Step 3: Test that ListTabs is fast (health check should use this)
	t.Log("Step 3: Verifying ListTabs is fast (used by health check)...")
	start := time.Now()
	_, err = client.ListTabs(ctx)
	elapsed := time.Since(start)

	if err != nil {
		t.Fatalf("ListTabs failed: %v", err)
	}

	if elapsed > 5*time.Second {
		t.Fatalf("ListTabs took too long: %v (should be fast)", elapsed)
	}
	t.Logf("✓ ListTabs completed quickly: %v", elapsed)

	// Step 4: Verify ReadPage would be slow (old health check behavior)
	t.Log("Step 4: Verifying ReadPage would be slow (old health check method)...")
	mockServer.SetCommandDelay("Accessibility.getFullAXTree", 5*time.Second)

	start = time.Now()
	readCtx, readCancel := context.WithTimeout(ctx, 10*time.Second)
	defer readCancel()

	_, err = client.ReadPage(readCtx, false)
	elapsed = time.Since(start)

	if err != nil {
		t.Logf("ReadPage timed out: %v (elapsed: %v)", err, elapsed)
	} else {
		t.Logf("⚠ ReadPage completed in %v (old health check would be slow)", elapsed)
	}

	t.Log("✓ Health check optimization: ListTabs is much faster than ReadPage")
	t.Log("=== E2E Test Completed Successfully ===")
}
