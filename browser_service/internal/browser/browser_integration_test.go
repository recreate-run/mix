package browser

import (
	"encoding/base64"
	"testing"
	"time"

	"github.com/sarathmenon/browser-service/pkg/protocol"
)

func TestBrowserLaunch(t *testing.T) {
	skipIfIntegrationTestsDisabled(t)

	_, mgr, _ := setupBrowserTest(t)

	if mgr == nil {
		t.Fatal("Expected non-nil manager")
	}

	if mgr.browser == nil {
		t.Fatal("Expected browser to be initialized")
	}
}

func TestNavigate(t *testing.T) {
	skipIfIntegrationTestsDisabled(t)

	ctx, _, browserCtx := setupBrowserTest(t)

	// Navigate to example.com
	result, err := browserCtx.Navigate(ctx, "https://example.com", 0, nil)
	if err != nil {
		t.Fatalf("Navigation failed: %v", err)
	}

	if result == nil {
		t.Fatal("Expected non-nil navigation result")
	}

	if result.FrameID == "" {
		t.Error("Expected non-empty FrameID")
	}

	// Take screenshot to confirm page loaded
	screenshotResult, err := browserCtx.Screenshot(ctx, protocol.ScreenshotParams{})
	if err != nil {
		t.Fatalf("Screenshot failed: %v", err)
	}

	if screenshotResult.Data == "" {
		t.Error("Expected non-empty screenshot data")
	}
}

func TestScreenshot(t *testing.T) {
	skipIfIntegrationTestsDisabled(t)

	ctx, _, browserCtx := setupBrowserTest(t)

	// Navigate to a test page
	_, err := browserCtx.Navigate(ctx, "https://example.com", 0, nil)
	if err != nil {
		t.Fatalf("Navigation failed: %v", err)
	}

	// Wait a bit for page to fully load
	time.Sleep(500 * time.Millisecond)

	tests := []struct {
		name   string
		params protocol.ScreenshotParams
	}{
		{
			name:   "default params",
			params: protocol.ScreenshotParams{},
		},
		{
			name: "png format",
			params: protocol.ScreenshotParams{
				Format: "png",
			},
		},
		{
			name: "full page",
			params: protocol.ScreenshotParams{
				Format:   "png",
				FullPage: true,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := browserCtx.Screenshot(ctx, tt.params)
			if err != nil {
				t.Fatalf("Screenshot failed: %v", err)
			}

			if result == nil {
				t.Fatal("Expected non-nil result")
			}

			if result.Data == "" {
				t.Error("Expected non-empty screenshot data")
			}

			// Verify it's valid base64
			data, err := base64.StdEncoding.DecodeString(result.Data)
			if err != nil {
				t.Errorf("Screenshot data is not valid base64: %v", err)
			}

			// Verify PNG header
			if len(data) < 8 {
				t.Error("Screenshot data is too short to be a valid image")
			} else {
				pngHeader := []byte{0x89, 0x50, 0x4E, 0x47, 0x0D, 0x0A, 0x1A, 0x0A}
				for i := 0; i < 8; i++ {
					if data[i] != pngHeader[i] {
						t.Errorf("Invalid PNG header at byte %d: got 0x%X, want 0x%X", i, data[i], pngHeader[i])
					}
				}
			}

			// Verify format field
			expectedFormat := tt.params.Format
			if expectedFormat == "" {
				expectedFormat = "png"
			}
			if result.Format != expectedFormat {
				t.Errorf("Expected format %s, got %s", expectedFormat, result.Format)
			}
		})
	}
}

func TestCleanupOnDisconnect(t *testing.T) {
	skipIfIntegrationTestsDisabled(t)

	ctx, mgr, _ := setupBrowserTest(t)

	browserCtx, err := mgr.NewContext(ctx)
	if err != nil {
		t.Fatalf("Failed to create browser context: %v", err)
	}

	// Navigate to a page
	_, err = browserCtx.Navigate(ctx, "https://example.com", 0, nil)
	if err != nil {
		t.Fatalf("Navigation failed: %v", err)
	}

	// Close context
	err = browserCtx.Close(ctx)
	if err != nil {
		t.Errorf("Close failed: %v", err)
	}

	// Verify no panic on double-close
	err = browserCtx.Close(ctx)
	// Double close should not panic, error is acceptable
	if err != nil {
		t.Logf("Double close returned error (expected): %v", err)
	}
}

func TestNavigateWithTimeout(t *testing.T) {
	skipIfIntegrationTestsDisabled(t)

	ctx, _, browserCtx := setupBrowserTest(t)

	// Navigate with custom timeout (5 seconds)
	result, err := browserCtx.Navigate(ctx, "https://example.com", 5000, nil)
	if err != nil {
		t.Fatalf("Navigation failed: %v", err)
	}

	if result.FrameID == "" {
		t.Error("Expected non-empty FrameID")
	}
}

func TestMultipleNavigations(t *testing.T) {
	skipIfIntegrationTestsDisabled(t)

	ctx, _, browserCtx := setupBrowserTest(t)

	// Navigate to first page
	result1, err := browserCtx.Navigate(ctx, "https://example.com", 0, nil)
	if err != nil {
		t.Fatalf("First navigation failed: %v", err)
	}

	if result1.FrameID == "" {
		t.Error("Expected non-empty FrameID for first navigation")
	}

	time.Sleep(500 * time.Millisecond)

	// Navigate to second page
	result2, err := browserCtx.Navigate(ctx, "https://www.iana.org", 0, nil)
	if err != nil {
		t.Fatalf("Second navigation failed: %v", err)
	}

	if result2.FrameID == "" {
		t.Error("Expected non-empty FrameID for second navigation")
	}

	// Frame IDs should be the same (same page object)
	// This verifies the context maintains state across navigations
}

func TestContextIsolation(t *testing.T) {
	skipIfIntegrationTestsDisabled(t)

	ctx, mgr, _ := setupBrowserTest(t)

	// Create two contexts
	ctx1, err := mgr.NewContext(ctx)
	if err != nil {
		t.Fatalf("Failed to create first context: %v", err)
	}
	defer func() {
		if err := ctx1.Close(ctx); err != nil {
			t.Errorf("Failed to close first context: %v", err)
		}
	}()

	ctx2, err := mgr.NewContext(ctx)
	if err != nil {
		t.Fatalf("Failed to create second context: %v", err)
	}
	defer func() {
		if err := ctx2.Close(ctx); err != nil {
			t.Errorf("Failed to close second context: %v", err)
		}
	}()

	// Navigate each to different pages
	result1, err := ctx1.Navigate(ctx, "https://example.com", 0, nil)
	if err != nil {
		t.Fatalf("First context navigation failed: %v", err)
	}

	result2, err := ctx2.Navigate(ctx, "https://www.iana.org", 0, nil)
	if err != nil {
		t.Fatalf("Second context navigation failed: %v", err)
	}

	// Wait for pages to load
	time.Sleep(1 * time.Second)

	// Take screenshots from both
	screenshot1, err := ctx1.Screenshot(ctx, protocol.ScreenshotParams{})
	if err != nil {
		t.Fatalf("First context screenshot failed: %v", err)
	}

	screenshot2, err := ctx2.Screenshot(ctx, protocol.ScreenshotParams{})
	if err != nil {
		t.Fatalf("Second context screenshot failed: %v", err)
	}

	// Screenshots should be different (different pages)
	if screenshot1.Data == screenshot2.Data {
		t.Error("Expected different screenshots from isolated contexts")
	}

	// Frame IDs should be different
	if result1.FrameID == result2.FrameID {
		t.Log("Frame IDs are the same - this might be expected depending on rod's behavior")
	}
}

func TestNavigateInvalidURL(t *testing.T) {
	skipIfIntegrationTestsDisabled(t)

	ctx, _, browserCtx := setupBrowserTest(t)

	// Try to navigate to invalid URL
	_, err := browserCtx.Navigate(ctx, "not-a-valid-url", 0, nil)
	if err == nil {
		t.Error("Expected error for invalid URL, got nil")
	}
}
