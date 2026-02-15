package test

import (
	"context"
	"fmt"
	"net"
	"testing"
	"time"

	"github.com/sarathmenon/browser-service/internal/server"
	"github.com/sarathmenon/browser-service/pkg/client"
)

// TestAmazonModalBlockingIncognito tests Amazon.com with BlockModals but WITHOUT EnableExtensions
func TestAmazonModalBlockingIncognito(t *testing.T) {
	skipIfIntegrationTestsDisabled(t)

	if testing.Short() {
		t.Skip("Skipping Amazon integration test in short mode")
	}

	ctx := context.Background()

	// Get free port
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

	// Create server with BlockModals=true, EnableExtensions=false (incognito mode)
	srv, err := server.New(ctx, server.Config{
		Port:             fmt.Sprintf("%d", port),
		Headless:         true,
		BlockModals:      true,
		EnableExtensions: false, // Test in incognito mode
	})
	if err != nil {
		t.Fatalf("Failed to create server: %v", err)
	}

	go func() {
		_ = srv.Start()
	}()

	time.Sleep(500 * time.Millisecond)

	wsURL := fmt.Sprintf("ws://127.0.0.1:%d/ws", port)
	c, err := client.New(wsURL)
	if err != nil {
		t.Fatalf("Failed to create client: %v", err)
	}

	cmdCtx, cancel := context.WithTimeout(ctx, 60*time.Second)
	defer cancel()

	defer func() {
		_ = c.Close()
		shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer shutdownCancel()
		_ = srv.Shutdown(shutdownCtx)
	}()

	// Navigate to Amazon.com
	t.Log("Testing Amazon.com in INCOGNITO mode (EnableExtensions=false)...")
	_, err = c.Navigate(cmdCtx, "https://www.amazon.com")
	if err != nil {
		t.Fatalf("Failed to navigate: %v", err)
	}

	time.Sleep(8 * time.Second)

	// Check modal blocker active
	activeResult, err := c.EvalJS(cmdCtx, "window.__modalBlockActive === true")
	if err != nil {
		t.Fatalf("Failed to check modal blocker: %v", err)
	}

	if activeResult.Result != true {
		t.Errorf("Modal blocker should be active in incognito mode")
	}

	// Check for visible modals
	modalCheckResult, err := c.EvalJS(cmdCtx, `
		(() => {
			const selectors = [
				'[data-testid*="GLUXZipUpdate"]',
				'[aria-label*="location"]',
				'[id*="nav-global-location-popover"]',
				'#GLUXZipUpdateModal',
				'[role="dialog"]'
			];

			let visibleCount = 0;
			selectors.forEach(selector => {
				const el = document.querySelector(selector);
				if (el) {
					const styles = window.getComputedStyle(el);
					if (styles.display !== 'none' &&
					    styles.visibility !== 'hidden' &&
					    styles.opacity !== '0') {
						visibleCount++;
					}
				}
			});

			return visibleCount;
		})()
	`)
	if err != nil {
		t.Fatalf("Failed to check modals: %v", err)
	}

	modalCount := int(modalCheckResult.Result.(float64))
	t.Logf("Visible modals on Amazon (incognito): %d", modalCount)

	if modalCount > 0 {
		t.Errorf("Expected 0 visible modals in incognito mode, found %d", modalCount)
	}

	// Check search box accessible
	searchResult, err := c.EvalJS(cmdCtx, `
		document.querySelector('#twotabsearchtextbox') ? 'found' : 'not_found'
	`)
	if err != nil {
		t.Fatalf("Failed to check search box: %v", err)
	}

	if searchResult.Result != "found" {
		t.Errorf("Search box should be accessible")
	}

	t.Log("✅ Amazon modal blocking works perfectly in INCOGNITO mode - EnableExtensions NOT needed!")
}
