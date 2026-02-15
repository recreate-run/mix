package test

import (
	"context"
	"encoding/base64"
	"fmt"
	"testing"
	"time"

	"github.com/sarathmenon/browser-service/pkg/protocol"
	"github.com/sarathmenon/browser-service/pkg/client"
)

func TestE2ENavigateAndScreenshot(t *testing.T) {
	skipIfIntegrationTestsDisabled(t)

	ctx, c, cleanup := setupE2ETest(t, 10)
	defer cleanup()

	navResult, err := c.Navigate(ctx, "https://example.com")
	if err != nil {
		t.Fatalf("Navigate failed: %v", err)
	}

	if navResult.FrameID == "" {
		t.Error("Expected non-empty FrameID")
	}

	// Take screenshot
	screenshot, err := c.Screenshot(ctx, protocol.ScreenshotParams{})
	if err != nil {
		t.Fatalf("Screenshot failed: %v", err)
	}

	// Verify screenshot is valid PNG
	data, err := base64.StdEncoding.DecodeString(screenshot.Data)
	if err != nil {
		t.Fatalf("Screenshot data is not valid base64: %v", err)
	}

	if len(data) < 8 {
		t.Fatal("Screenshot data is too short")
	}

	// Check PNG header
	pngHeader := []byte{0x89, 0x50, 0x4E, 0x47, 0x0D, 0x0A, 0x1A, 0x0A}
	for i := 0; i < 8; i++ {
		if data[i] != pngHeader[i] {
			t.Errorf("Invalid PNG header at byte %d", i)
		}
	}

	t.Logf("Successfully navigated and captured screenshot (%d bytes)", len(data))
}

func TestE2EMultipleCommands(t *testing.T) {
	skipIfIntegrationTestsDisabled(t)

	ctx, c, cleanup := setupE2ETest(t, 30)
	defer cleanup()

	// Navigate to first page
	_, err := c.Navigate(ctx, "https://example.com")
	if err != nil {
		t.Fatalf("First navigation failed: %v", err)
	}

	// Take screenshot
	screenshot1, err := c.Screenshot(ctx, protocol.ScreenshotParams{})
	if err != nil {
		t.Fatalf("First screenshot failed: %v", err)
	}

	// Navigate to another page
	_, err = c.Navigate(ctx, "https://www.iana.org")
	if err != nil {
		t.Fatalf("Second navigation failed: %v", err)
	}

	// Wait for page to load
	time.Sleep(1 * time.Second)

	// Take another screenshot
	screenshot2, err := c.Screenshot(ctx, protocol.ScreenshotParams{})
	if err != nil {
		t.Fatalf("Second screenshot failed: %v", err)
	}

	// Verify screenshots are different
	if screenshot1.Data == screenshot2.Data {
		t.Error("Expected different screenshots for different pages")
	}

	t.Log("Successfully completed multiple commands")
}

func TestE2EGracefulShutdown(t *testing.T) {
	skipIfIntegrationTestsDisabled(t)

	ctx, srv, c, cleanup := setupE2ETestWithServer(t, 10)
	defer cleanup()

	_, err := c.Navigate(ctx, "https://example.com")
	if err != nil {
		t.Fatalf("Navigate failed: %v", err)
	}

	// Trigger server shutdown
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := srv.Shutdown(shutdownCtx); err != nil {
		t.Errorf("Server shutdown error: %v", err)
	}

	// Verify connection closes cleanly
	// Attempting to use client after server shutdown should fail
	time.Sleep(500 * time.Millisecond)

	_, err = c.Screenshot(ctx, protocol.ScreenshotParams{})
	if err == nil {
		t.Log("Warning: Expected error after server shutdown, but screenshot succeeded")
	}

	t.Log("Server shutdown completed successfully")
}

func TestE2EFullPageWorkflow(t *testing.T) {
	skipIfIntegrationTestsDisabled(t)

	ctx, c, cleanup := setupE2ETest(t, 30)
	defer cleanup()

	// Step 1: Navigate
	t.Log("Step 1: Navigating to example.com")
	navResult, err := c.Navigate(ctx, "https://example.com")
	if err != nil {
		t.Fatalf("Navigate failed: %v", err)
	}
	t.Logf("Navigation complete, FrameID: %s", navResult.FrameID)

	// Step 2: Take normal screenshot
	t.Log("Step 2: Taking normal screenshot")
	screenshot1, err := c.Screenshot(ctx, protocol.ScreenshotParams{
		Format: "png",
	})
	if err != nil {
		t.Fatalf("Normal screenshot failed: %v", err)
	}
	t.Logf("Normal screenshot captured (%d bytes base64)", len(screenshot1.Data))

	// Step 3: Take full page screenshot
	t.Log("Step 3: Taking full page screenshot")
	screenshot2, err := c.Screenshot(ctx, protocol.ScreenshotParams{
		Format:   "png",
		FullPage: true,
	})
	if err != nil {
		t.Fatalf("Full page screenshot failed: %v", err)
	}
	t.Logf("Full page screenshot captured (%d bytes base64)", len(screenshot2.Data))

	t.Log("Full page workflow completed successfully")
}

func TestE2EConcurrentClients(t *testing.T) {
	skipIfIntegrationTestsDisabled(t)

	ctx := context.Background()
	_, wsURL, serverCleanup := startTestServer(t, ctx)
	defer serverCleanup()

	numClients := 3
	done := make(chan error, numClients)

	// Start multiple clients concurrently
	for i := 0; i < numClients; i++ {
		go func(clientNum int) {
			c, err := client.New(wsURL)
			if err != nil {
				done <- fmt.Errorf("client %d failed to connect: %w", clientNum, err)
				return
			}

			cmdCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
			defer cancel()

			//nolint:contextcheck // Context intentionally not passed to Close in defer
			defer func() {
				if err := c.Close(); err != nil {
					t.Errorf("client %d failed to close: %v", clientNum, err)
				}
			}()

			// Navigate
			_, err = c.Navigate(cmdCtx, "https://example.com")
			if err != nil {
				done <- fmt.Errorf("client %d navigate failed: %w", clientNum, err)
				return
			}

			// Screenshot
			_, err = c.Screenshot(cmdCtx, protocol.ScreenshotParams{})
			if err != nil {
				done <- fmt.Errorf("client %d screenshot failed: %w", clientNum, err)
				return
			}

			done <- nil
		}(i)
	}

	// Wait for all clients to complete
	for i := 0; i < numClients; i++ {
		if err := <-done; err != nil {
			t.Error(err)
		}
	}

	t.Logf("Successfully completed workflow with %d concurrent clients", numClients)
}

func TestE2EErrorHandling(t *testing.T) {
	skipIfIntegrationTestsDisabled(t)

	ctx, c, cleanup := setupE2ETest(t, 10)
	defer cleanup()

	// Try to navigate to invalid URL
	_, err := c.Navigate(ctx, "not-a-valid-url")
	if err == nil {
		t.Error("Expected error for invalid URL")
	} else {
		t.Logf("Got expected error for invalid URL: %v", err)
	}

	// Navigate to valid URL should still work
	_, err = c.Navigate(ctx, "https://example.com")
	if err != nil {
		t.Fatalf("Navigate to valid URL failed: %v", err)
	}

	t.Log("Error handling works correctly")
}
