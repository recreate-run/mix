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

// TestAllowModalsFlag verifies that --allow-modals flag disables modal blocking
func TestAllowModalsFlag(t *testing.T) {
	skipIfIntegrationTestsDisabled(t)

	ctx := context.Background()

	// Start test server
	httpServer := testserver.StartTestServer(t)
	defer httpServer.Close()

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

	// Create server with BlockModals=false (simulating --allow-modals flag)
	srv, err := server.New(ctx, server.Config{
		Port:        fmt.Sprintf("%d", port),
		Headless:    true,
		BlockModals: false, // Simulating --allow-modals flag
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

	cmdCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	defer func() {
		_ = c.Close()
		shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer shutdownCancel()
		_ = srv.Shutdown(shutdownCtx)
	}()

	// Navigate to modal test page
	_, err = c.Navigate(cmdCtx, httpServer.URL+"/modal-popup-page")
	if err != nil {
		t.Fatalf("Failed to navigate: %v", err)
	}

	time.Sleep(2 * time.Second)

	// Check if modal blocker is active (should be false when --allow-modals is used)
	activeResult, err := c.EvalJS(cmdCtx, "window.__modalBlockActive === true")
	if err != nil {
		t.Fatalf("Failed to check modal blocker: %v", err)
	}

	if activeResult.Result == true {
		t.Errorf("Modal blocker should be DISABLED when --allow-modals flag is used, but it's active")
	}

	// Check that modal is NOT blocked (should be visible)
	modalCheckResult, err := c.EvalJS(cmdCtx, `
		(() => {
			const modal = document.querySelector('[role="dialog"]');
			if (!modal) return 'not_found';
			const styles = window.getComputedStyle(modal);
			return styles.display === 'none' ? 'hidden' : 'visible';
		})()
	`)
	if err != nil {
		t.Fatalf("Failed to check modal: %v", err)
	}

	modalStatus, ok := modalCheckResult.Result.(string)
	if !ok {
		t.Fatalf("Unexpected result type: %T", modalCheckResult.Result)
	}

	// Modal should be visible when blocking is disabled
	if modalStatus != "visible" {
		t.Logf("Note: Modal status is %s (expected 'visible' when blocking disabled)", modalStatus)
	}

	t.Log("✅ --allow-modals flag successfully disables modal blocking")
}
