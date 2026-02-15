//go:build e2e
// +build e2e

package test

import (
	"context"
	"fmt"
	"net"
	"testing"
	"time"

	"github.com/sarathmenon/browser-service/internal/server"
	"github.com/sarathmenon/browser-service/pkg/client"
	"github.com/sarathmenon/browser-service/test/testserver"
)

// TestUBlockActuallyBlocksAds is the definitive test that uBlock Origin works
func TestUBlockActuallyBlocksAds(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 120*time.Second)
	defer cancel()

	// Start HTTP test server with ad test page
	httpServer := testserver.StartTestServer(t)
	defer httpServer.Close()

	testPageURL := httpServer.URL + "/realistic-ad-test-page"
	t.Logf("Test page URL: %s", testPageURL)

	// Test 1: Browser WITHOUT extensions (control)
	t.Log("=== Testing WITHOUT extensions (control) ===")
	controlAds := runAdTest(t, ctx, testPageURL, false)
	t.Logf("Control (no extensions): %d visible ads", controlAds)

	// Test 2: Browser WITH extensions (uBlock enabled)
	t.Log("=== Testing WITH uBlock Origin ===")
	uBlockAds := runAdTest(t, ctx, testPageURL, true)
	t.Logf("With uBlock: %d visible ads", uBlockAds)

	// Verification: uBlock should block at least some ads
	t.Logf("=== RESULTS ===")
	t.Logf("Without extensions: %d ads visible", controlAds)
	t.Logf("With uBlock Origin: %d ads visible", uBlockAds)

	if controlAds == 0 {
		t.Fatal("Control test failed: no ads visible even without extensions. Test page may be broken.")
	}

	if uBlockAds >= controlAds {
		t.Fatalf("uBlock FAILED: Blocked %d ads out of %d (expected fewer ads with uBlock)",
			controlAds-uBlockAds, controlAds)
	}

	adsBlocked := controlAds - uBlockAds
	blockRate := float64(adsBlocked) / float64(controlAds) * 100

	t.Logf("✓ uBlock SUCCESS: Blocked %d out of %d ads (%.1f%% block rate)",
		adsBlocked, controlAds, blockRate)

	// We expect at least 50% of ads to be blocked
	if blockRate < 50 {
		t.Errorf("Warning: uBlock only blocked %.1f%% of ads, expected at least 50%%", blockRate)
	}
}

// runAdTest starts a browser server, navigates to test page, and counts visible ads
func runAdTest(t *testing.T, ctx context.Context, testPageURL string, enableExtensions bool) int {
	t.Helper()

	tmpDir := t.TempDir()

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

	// Create server with or without extensions
	// NOTE: Extensions require non-headless mode in most Chrome versions
	srv, err := server.New(ctx, server.Config{
		Port:              fmt.Sprintf("%d", port),
		Headless:          false, // Extensions don't work reliably in headless mode
		EnableExtensions:  enableExtensions,
		ExtensionCacheDir: tmpDir,
		UBlockEnabled:     true,
		CookieConsentEnabled: false, // Only test uBlock
		ClearURLsEnabled:  false,
	})
	if err != nil {
		t.Fatalf("Failed to create server: %v", err)
	}

	// Start server
	go func() {
		_ = srv.Start()
	}()
	defer func() {
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		_ = srv.Shutdown(shutdownCtx)
	}()

	// Wait for server to start
	time.Sleep(1 * time.Second)

	wsURL := fmt.Sprintf("ws://127.0.0.1:%d/ws", port)

	// Connect client
	c, err := client.New(wsURL)
	if err != nil {
		t.Fatalf("Failed to connect to server: %v", err)
	}
	defer c.Close()

	// For extensions, wait before navigating to let extension fully initialize
	if enableExtensions {
		t.Logf("Waiting 15 seconds for uBlock to initialize...")
		time.Sleep(15 * time.Second)
	}

	// Navigate to test page
	_, err = c.Navigate(ctx, testPageURL)
	if err != nil {
		t.Fatalf("Failed to navigate to test page: %v", err)
	}

	// Wait for page to load and uBlock to process
	if enableExtensions {
		t.Logf("Waiting 10 more seconds for uBlock to process page...")
		time.Sleep(10 * time.Second) // Give uBlock time to process
	} else {
		time.Sleep(2 * time.Second)
	}

	// Debug: Check if we can detect extension presence
	if enableExtensions {
		// Try to navigate to chrome://extensions to check
		extCheck, err := c.EvalJS(ctx, `
			// Check for extension-modified DOM
			document.querySelector('html').getAttribute('ublock-loaded') ||
			document.documentElement.hasAttribute('ublock') ||
			'no-ublock-attribute'
		`)
		if err == nil {
			t.Logf("Extension DOM check: %v", extCheck.Result)
		}
	}

	// Count visible ads
	result, err := c.EvalJS(ctx, "window.countVisibleTestElements()")
	if err != nil {
		t.Fatalf("Failed to count visible ads: %v", err)
	}

	visibleAds, ok := result.Result.(float64)
	if !ok {
		t.Fatalf("Unexpected result type: %T, value: %v", result.Result, result.Result)
	}

	return int(visibleAds)
}
