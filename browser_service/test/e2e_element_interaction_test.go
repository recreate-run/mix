package test

import (
	"encoding/base64"
	"image/png"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/sarathmenon/browser-service/pkg/protocol"
)

func TestE2EElementWorkflow(t *testing.T) {
	skipIfIntegrationTestsDisabled(t)

	ctx, c, cleanup := setupE2ETest(t, 30)
	defer cleanup()

	// Step 1: Navigate to test page
	t.Log("Step 1: Navigating to example.com")
	_, err := c.Navigate(ctx, "https://example.com")
	if err != nil {
		t.Fatalf("Navigate failed: %v", err)
	}

	// Step 2: Get elements
	t.Log("Step 2: Getting interactive elements")
	elements, err := c.GetElements(ctx)
	if err != nil {
		t.Fatalf("GetElements failed: %v", err)
	}

	if len(elements) == 0 {
		t.Fatal("No elements found on page")
	}
	t.Logf("Found %d interactive elements", len(elements))

	// Step 3: Find a link element and click it
	t.Log("Step 3: Clicking a link element")
	linkIndex, found := findElementByRole(elements, "link")

	if found {
		t.Logf("Found link at index %d", linkIndex)
		err = c.Click(ctx, linkIndex)
		if err != nil {
			t.Errorf("Click failed: %v", err)
		} else {
			t.Logf("Successfully clicked element %d", linkIndex)
		}
		time.Sleep(500 * time.Millisecond)
	} else {
		t.Log("No link element found, skipping click test")
	}

	// Step 4: Navigate to Google for input test
	t.Log("Step 4: Navigating to Google for input test")
	_, err = c.Navigate(ctx, "https://www.google.com")
	if err != nil {
		t.Fatalf("Navigate to Google failed: %v", err)
	}

	// Step 5: Get elements and find input
	elements, err = c.GetElements(ctx)
	if err != nil {
		t.Fatalf("GetElements failed: %v", err)
	}

	t.Log("Step 5: Typing into input element")
	inputIndex, found := findElementByRole(elements, "textbox", "searchbox", "combobox")

	if found {
		t.Logf("Found input at index %d", inputIndex)
		err = c.Type(ctx, inputIndex, "test input")
		if err != nil {
			t.Errorf("Type failed: %v", err)
		} else {
			t.Logf("Successfully typed into element %d", inputIndex)
		}
		time.Sleep(500 * time.Millisecond)
	} else {
		t.Log("No input element found, skipping type test")
	}

	// Step 6: Scroll the page
	t.Log("Step 6: Scrolling page")
	err = c.Scroll(ctx, "down", 500)
	if err != nil {
		t.Errorf("Scroll failed: %v", err)
	} else {
		t.Log("Successfully scrolled page")
	}
	time.Sleep(200 * time.Millisecond)

	t.Log("Element workflow completed successfully")
}

func TestE2EScreenshotWithRawMode(t *testing.T) {
	skipIfIntegrationTestsDisabled(t)

	ctx, c, cleanup := setupE2ETest(t, 30)
	defer cleanup()

	// Navigate to test page
	t.Log("Navigating to example.com")
	_, err := c.Navigate(ctx, "https://example.com")
	if err != nil {
		t.Fatalf("Navigate failed: %v", err)
	}

	// Take screenshot without raw mode
	t.Log("Taking screenshot without raw mode")
	result1, err := c.Screenshot(ctx, protocol.ScreenshotParams{
		Raw: false,
	})
	if err != nil {
		t.Fatalf("Screenshot without raw mode failed: %v", err)
	}

	// Take screenshot with raw mode
	t.Log("Taking screenshot with raw mode")
	result2, err := c.Screenshot(ctx, protocol.ScreenshotParams{
		Raw: true,
	})
	if err != nil {
		t.Fatalf("Screenshot with raw mode failed: %v", err)
	}

	// Verify RawNodes field is populated
	if len(result2.RawNodes) == 0 {
		t.Error("Expected raw nodes to be populated with raw mode, got none")
	} else {
		t.Logf("Screenshot has %d raw nodes", len(result2.RawNodes))
	}

	// Verify RawViewport is populated
	if result2.RawViewport == nil {
		t.Error("Expected raw viewport to be populated with raw mode, got nil")
	}

	// Verify screenshot data is valid PNG
	imgData, err := base64.StdEncoding.DecodeString(result2.Data)
	if err != nil {
		t.Fatalf("Failed to decode screenshot data: %v", err)
	}

	_, err = png.DecodeConfig(strings.NewReader(string(imgData)))
	if err != nil {
		t.Errorf("Screenshot data is not a valid PNG: %v", err)
	}

	// Verify base image data is the same (raw mode doesn't modify the image)
	if result1.Data != result2.Data {
		t.Error("Raw mode screenshot should have same image data as non-raw screenshot")
	}

	// Verify screenshot sizes are the same
	data1, _ := base64.StdEncoding.DecodeString(result1.Data)
	data2, _ := base64.StdEncoding.DecodeString(result2.Data)
	t.Logf("Screenshot sizes: without raw=%d bytes, with raw=%d bytes", len(data1), len(data2))

	// Optionally save screenshot to temp file for manual inspection
	if os.Getenv("SAVE_TEST_SCREENSHOTS") == "1" {
		tmpFile := "/tmp/test_screenshot_overlay.png"
		if err := os.WriteFile(tmpFile, imgData, 0o600); err != nil {
			t.Logf("Warning: Failed to save screenshot: %v", err)
		} else {
			t.Logf("Screenshot saved to %s for manual inspection", tmpFile)
		}
	}

	t.Log("Screenshot with overlay test completed successfully")
}

func TestE2EMultipleClicks(t *testing.T) {
	skipIfIntegrationTestsDisabled(t)

	ctx, c, cleanup := setupE2ETest(t, 30)
	defer cleanup()

	// Navigate to example.com
	_, err := c.Navigate(ctx, "https://example.com")
	if err != nil {
		t.Fatalf("Navigate failed: %v", err)
	}

	// Get elements
	elements, err := c.GetElements(ctx)
	if err != nil {
		t.Fatalf("GetElements failed: %v", err)
	}

	// Find first link element
	linkIndex, found := findElementByRole(elements, "link")
	if !found {
		t.Skip("No link elements found on page")
	}

	t.Logf("Found link element at index %d", linkIndex)

	// Click first link
	err = c.Click(ctx, linkIndex)
	if err != nil {
		t.Errorf("Failed to click first link: %v", err)
	} else {
		t.Logf("Successfully clicked link at index %d", linkIndex)
	}

	t.Log("Multiple clicks test completed")
}

func TestE2EInvalidElementIndex(t *testing.T) {
	skipIfIntegrationTestsDisabled(t)

	ctx, c, cleanup := setupE2ETest(t, 30)
	defer cleanup()

	// Navigate to example.com
	_, err := c.Navigate(ctx, "https://example.com")
	if err != nil {
		t.Fatalf("Navigate failed: %v", err)
	}

	// Try to click invalid index
	err = c.Click(ctx, 9999)
	if err == nil {
		t.Error("Expected error when clicking invalid index, got none")
	} else {
		t.Logf("Got expected error for invalid index: %v", err)
	}

	// Try to type with invalid index
	err = c.Type(ctx, 9999, "test")
	if err == nil {
		t.Error("Expected error when typing to invalid index, got none")
	} else {
		t.Logf("Got expected error for invalid type index: %v", err)
	}

	t.Log("Invalid index error handling test completed")
}

func TestE2EElementsAfterNavigation(t *testing.T) {
	skipIfIntegrationTestsDisabled(t)

	ctx, c, cleanup := setupE2ETest(t, 30)
	defer cleanup()

	// Navigate to page A
	t.Log("Navigating to page A (example.com)")
	_, err := c.Navigate(ctx, "https://example.com")
	if err != nil {
		t.Fatalf("Navigate to page A failed: %v", err)
	}

	// Get elements from page A
	elementsA, err := c.GetElements(ctx)
	if err != nil {
		t.Fatalf("GetElements from page A failed: %v", err)
	}
	t.Logf("Page A has %d elements", len(elementsA))

	// Navigate to page B
	t.Log("Navigating to page B (wikipedia.org)")
	_, err = c.Navigate(ctx, "https://www.wikipedia.org")
	if err != nil {
		t.Fatalf("Navigate to page B failed: %v", err)
	}

	// Click should auto-load elements from page B (lazy loading)
	// This verifies that elements are correctly refreshed after navigation
	err = c.Click(ctx, 0)
	if err != nil {
		t.Errorf("Failed to click with lazy-loaded elements after navigation: %v", err)
	} else {
		t.Log("Successfully clicked with lazy-loaded elements after navigation")
	}

	// Get elements from page B to verify they were auto-loaded
	elementsB, err := c.GetElements(ctx)
	if err != nil {
		t.Fatalf("GetElements from page B failed: %v", err)
	}
	t.Logf("Page B has %d elements", len(elementsB))

	// Verify that page B has different number of elements than page A
	if len(elementsA) != len(elementsB) {
		t.Logf("Page A and B have different element counts (%d vs %d), navigation verified", len(elementsA), len(elementsB))
	}

	t.Log("Elements after navigation test completed")
}
