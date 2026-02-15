package test

import (
	"context"
	"fmt"
	"net"
	"strings"
	"testing"
	"time"

	"github.com/sarathmenon/browser-service/internal/server"
	"github.com/sarathmenon/browser-service/pkg/client"
	"github.com/sarathmenon/browser-service/test/testserver"
)

// TestModalBlockingInIncognitoMode tests if modal blocking works WITHOUT EnableExtensions
// If this passes, we don't actually need the shared browser context!
func TestModalBlockingInIncognitoMode(t *testing.T) {
	skipIfIntegrationTestsDisabled(t)

	ctx := context.Background()

	// Start test server
	httpServer := testserver.StartTestServer(t)
	defer httpServer.Close()

	// Get a free port for browser service
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

	// Create server with BlockModals=true but EnableExtensions=FALSE (incognito mode)
	srv, err := server.New(ctx, server.Config{
		Port:             fmt.Sprintf("%d", port),
		Headless:         true,
		BlockModals:      true,
		EnableExtensions: false, // CRITICAL: Testing incognito mode
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
	t.Log("Testing modal blocking in INCOGNITO mode (EnableExtensions=false)")
	_, err = c.Navigate(cmdCtx, httpServer.URL+"/modal-popup-page")
	if err != nil {
		t.Fatalf("Failed to navigate: %v", err)
	}

	time.Sleep(2 * time.Second)

	// Check if modal blocker is active
	activeResult, err := c.EvalJS(cmdCtx, "window.__modalBlockActive === true")
	if err != nil {
		t.Fatalf("Failed to check modal blocker: %v", err)
	}

	t.Logf("Modal blocker active in incognito: %v", activeResult.Result)

	if activeResult.Result != true {
		// Check for errors
		errorResult, _ := c.EvalJS(cmdCtx, "window.__modalBlockError")
		t.Logf("Modal blocker error (if any): %v", errorResult.Result)
		t.Errorf("Modal blocker should be active even in incognito mode")
	}

	// Check if modal is blocked
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

	t.Logf("Modal status in incognito: %v", modalCheckResult.Result)

	modalStatus, ok := modalCheckResult.Result.(string)
	if !ok {
		t.Fatalf("Unexpected modal status type: %T", modalCheckResult.Result)
	}

	if modalStatus != "hidden" && modalStatus != "not_found" {
		t.Errorf("Expected modal to be blocked in incognito mode, got: %s", modalStatus)
	}

	// Check page content
	textResult, err := c.GetText(cmdCtx, "body")
	if err != nil {
		t.Fatalf("Failed to get text: %v", err)
	}

	if !strings.Contains(textResult.Text, "Main Page Content") {
		t.Errorf("Expected main content to be visible")
	}

	t.Log("✅ Modal blocking WORKS in incognito mode - EnableExtensions NOT required!")
}
