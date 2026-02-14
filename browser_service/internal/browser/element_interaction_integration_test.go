package browser

import (
	"encoding/base64"
	"image/png"
	"strings"
	"testing"
	"time"

	"github.com/sarathmenon/browser-service/pkg/protocol"
)

func TestGetElements(t *testing.T) {
	skipIfIntegrationTestsDisabled(t)

	ctx, _, browserCtx := setupBrowserTest(t)

	// Navigate to test page
	_, err := browserCtx.Navigate(ctx, "https://example.com", 10000, nil)
	if err != nil {
		t.Fatalf("Failed to navigate: %v", err)
	}

	// Get elements
	elements, err := browserCtx.GetElements(ctx, nil)
	if err != nil {
		t.Fatalf("Failed to get elements: %v", err)
	}

	// Verify elements array is not empty
	if len(elements) == 0 {
		t.Error("Expected elements to be found, got none")
	}

	// Verify each element has valid properties
	for i, elem := range elements {
		// Non-empty role
		if elem.Role == "" {
			t.Errorf("Element at position %d has empty role", i)
		}

		// Valid bounds (width > 0, height > 0)
		if elem.Bounds.Width <= 0 {
			t.Errorf("Element at position %d has invalid width: %f", i, elem.Bounds.Width)
		}
		if elem.Bounds.Height <= 0 {
			t.Errorf("Element at position %d has invalid height: %f", i, elem.Bounds.Height)
		}

		// BackendID should be set
		if elem.BackendID == 0 {
			t.Errorf("Element at position %d has zero BackendID", i)
		}
	}

	t.Logf("Found %d interactive elements", len(elements))
}

func TestClickByIndex(t *testing.T) {
	skipIfIntegrationTestsDisabled(t)

	ctx, _, browserCtx := setupBrowserTest(t)

	// Navigate to example.com which has a link
	_, err := browserCtx.Navigate(ctx, "https://example.com", 10000, nil)
	if err != nil {
		t.Fatalf("Failed to navigate: %v", err)
	}

	// Get elements
	elements, err := browserCtx.GetElements(ctx, nil)
	if err != nil {
		t.Fatalf("Failed to get elements: %v", err)
	}

	if len(elements) == 0 {
		t.Fatal("No elements found to click")
	}

	// Find a link element to click
	linkIndex, found := findElementByRole(elements, "link")
	if !found {
		t.Skip("No link element found on page")
	}

	// Click the link
	err = browserCtx.Click(ctx, linkIndex, nil)
	if err != nil {
		t.Errorf("Failed to click element %d: %v", linkIndex, err)
	}

	// Give time for potential navigation
	time.Sleep(500 * time.Millisecond)

	t.Logf("Successfully clicked element %d", linkIndex)
}

func TestTypeByIndex(t *testing.T) {
	skipIfIntegrationTestsDisabled(t)

	ctx, _, browserCtx := setupBrowserTest(t)

	// Navigate to a page with input (Google has a search box)
	_, err := browserCtx.Navigate(ctx, "https://www.google.com", 10000, nil)
	if err != nil {
		t.Fatalf("Failed to navigate: %v", err)
	}

	// Get elements
	elements, err := browserCtx.GetElements(ctx, nil)
	if err != nil {
		t.Fatalf("Failed to get elements: %v", err)
	}

	// Find a textbox or searchbox element
	inputIndex, found := findElementByRole(elements, "textbox", "searchbox", "combobox")
	if !found {
		t.Skip("No textbox/searchbox/combobox element found on page")
	}

	// Type into the input
	testText := "hello world"
	err = browserCtx.Type(ctx, &inputIndex, testText, nil)
	if err != nil {
		t.Errorf("Failed to type into element %d: %v", inputIndex, err)
	}

	// Take screenshot to verify (could also check element value via JS)
	_, err = browserCtx.Screenshot(ctx, protocol.ScreenshotParams{})
	if err != nil {
		t.Errorf("Failed to take screenshot: %v", err)
	}

	t.Logf("Successfully typed '%s' into element %d", testText, inputIndex)
}

func TestScrollPage(t *testing.T) {
	skipIfIntegrationTestsDisabled(t)

	ctx, _, browserCtx := setupBrowserTest(t)

	// Navigate to a long page (Wikipedia has lots of content)
	_, err := browserCtx.Navigate(ctx, "https://en.wikipedia.org/wiki/Main_Page", 10000, nil)
	if err != nil {
		t.Fatalf("Failed to navigate: %v", err)
	}

	// Get initial scroll position
	initialScroll, err := browserCtx.EvalJS(ctx, "() => window.pageYOffset", nil)
	if err != nil {
		t.Fatalf("Failed to get scroll position: %v", err)
	}

	// Scroll down 500px
	err = browserCtx.Scroll(ctx, "down", 500, nil)
	if err != nil {
		t.Errorf("Failed to scroll down: %v", err)
	}

	time.Sleep(200 * time.Millisecond)

	// Get new scroll position
	afterScroll, err := browserCtx.EvalJS(ctx, "() => window.pageYOffset", nil)
	if err != nil {
		t.Fatalf("Failed to get scroll position after scroll: %v", err)
	}

	// Verify viewport changed
	initialPos := initialScroll.Value.Int()
	afterPos := afterScroll.Value.Int()
	if afterPos <= initialPos {
		t.Errorf("Scroll down failed: initial=%d, after=%d", initialPos, afterPos)
	}

	// Scroll back up
	err = browserCtx.Scroll(ctx, "up", 500, nil)
	if err != nil {
		t.Errorf("Failed to scroll up: %v", err)
	}

	time.Sleep(200 * time.Millisecond)

	// Get final scroll position
	finalScroll, err := browserCtx.EvalJS(ctx, "() => window.pageYOffset", nil)
	if err != nil {
		t.Fatalf("Failed to get final scroll position: %v", err)
	}

	finalPos := finalScroll.Value.Int()
	if finalPos >= afterPos {
		t.Errorf("Scroll up failed: after=%d, final=%d", afterPos, finalPos)
	}

	t.Logf("Scroll test passed: initial=%d, scrolled=%d, final=%d", initialPos, afterPos, finalPos)
}

func TestElementIndexConsistency(t *testing.T) {
	skipIfIntegrationTestsDisabled(t)

	ctx, _, browserCtx := setupBrowserTest(t)

	// Navigate to test page
	testURL := "https://example.com"
	_, err := browserCtx.Navigate(ctx, testURL, 10000, nil)
	if err != nil {
		t.Fatalf("Failed to navigate first time: %v", err)
	}

	// Get elements first time
	elements1, err := browserCtx.GetElements(ctx, nil)
	if err != nil {
		t.Fatalf("Failed to get elements first time: %v", err)
	}

	// Navigate to same page again
	_, err = browserCtx.Navigate(ctx, testURL, 10000, nil)
	if err != nil {
		t.Fatalf("Failed to navigate second time: %v", err)
	}

	// Get elements second time
	elements2, err := browserCtx.GetElements(ctx, nil)
	if err != nil {
		t.Fatalf("Failed to get elements second time: %v", err)
	}

	// Verify same number of elements
	if len(elements1) != len(elements2) {
		t.Errorf("Element count mismatch: first=%d, second=%d", len(elements1), len(elements2))
	}

	// Verify array positions match (roles should be identical at same positions)
	// Note: BackendIDs may change between navigations, so we only check roles
	for i := 0; i < len(elements1) && i < len(elements2); i++ {
		if elements1[i].Role != elements2[i].Role {
			t.Errorf("Role mismatch at position %d: first=%s, second=%s", i, elements1[i].Role, elements2[i].Role)
		}
	}

	t.Logf("Array position consistency verified: %d elements match", len(elements1))
}

func TestScreenshotWithRawMode(t *testing.T) {
	skipIfIntegrationTestsDisabled(t)

	ctx, _, browserCtx := setupBrowserTest(t)

	// Navigate to test page
	_, err := browserCtx.Navigate(ctx, "https://example.com", 10000, nil)
	if err != nil {
		t.Fatalf("Failed to navigate: %v", err)
	}

	// Take screenshot without raw mode
	result1, err := browserCtx.Screenshot(ctx, protocol.ScreenshotParams{
		Raw: false,
	})
	if err != nil {
		t.Fatalf("Failed to take screenshot without raw mode: %v", err)
	}

	// Take screenshot with raw mode
	result2, err := browserCtx.Screenshot(ctx, protocol.ScreenshotParams{
		Raw: true,
	})
	if err != nil {
		t.Fatalf("Failed to take screenshot with raw mode: %v", err)
	}

	// Verify RawNodes field is populated
	if len(result2.RawNodes) == 0 {
		t.Error("Expected raw nodes to be populated with raw mode, got none")
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

	t.Logf("Screenshot with raw mode successful: %d nodes returned", len(result2.RawNodes))
}

func TestInvalidElementIndex(t *testing.T) {
	skipIfIntegrationTestsDisabled(t)

	ctx, _, browserCtx := setupBrowserTest(t)

	// Navigate to test page
	_, err := browserCtx.Navigate(ctx, "https://example.com", 10000, nil)
	if err != nil {
		t.Fatalf("Failed to navigate: %v", err)
	}

	// Try to click invalid index
	err = browserCtx.Click(ctx, 9999, nil)
	if err == nil {
		t.Error("Expected error when clicking invalid index, got none")
	}

	// Verify error message mentions invalid index
	errMsg := err.Error()
	if !strings.Contains(strings.ToLower(errMsg), "index") && !strings.Contains(strings.ToLower(errMsg), "element") {
		t.Errorf("Error message should mention index or element, got: %s", errMsg)
	}

	t.Logf("Invalid index error correctly returned: %v", err)
}

func TestElementInteractionAfterNavigation(t *testing.T) {
	skipIfIntegrationTestsDisabled(t)

	ctx, _, browserCtx := setupBrowserTest(t)

	// Navigate to page A
	_, err := browserCtx.Navigate(ctx, "https://example.com", 10000, nil)
	if err != nil {
		t.Fatalf("Failed to navigate to page A: %v", err)
	}

	// Get elements from page A
	elementsA, err := browserCtx.GetElements(ctx, nil)
	if err != nil {
		t.Fatalf("Failed to get elements from page A: %v", err)
	}

	if len(elementsA) == 0 {
		t.Fatal("No elements found on page A")
	}

	// Navigate to page B (different page)
	_, err = browserCtx.Navigate(ctx, "https://www.wikipedia.org", 10000, nil)
	if err != nil {
		t.Fatalf("Failed to navigate to page B: %v", err)
	}

	// Click should auto-load elements from page B (lazy loading)
	// This verifies that elements are correctly refreshed after navigation
	err = browserCtx.Click(ctx, 0, nil)
	if err != nil {
		t.Errorf("Failed to click with lazy-loaded elements after navigation: %v", err)
	}

	// Get elements from page B to verify they were auto-loaded
	elementsB, err := browserCtx.GetElements(ctx, nil)
	if err != nil {
		t.Fatalf("Failed to get elements from page B: %v", err)
	}

	if len(elementsB) == 0 {
		t.Fatal("No elements found on page B")
	}

	// Verify that page B has different number of elements than page A
	// (to confirm navigation actually changed the page)
	if len(elementsA) == len(elementsB) {
		t.Logf("Warning: Page A and B have same number of elements (%d), might be same page", len(elementsA))
	}

	t.Logf("Element clearing after navigation verified: pageA=%d elements, pageB=%d elements", len(elementsA), len(elementsB))
}

func TestLazyLoadingClick(t *testing.T) {
	skipIfIntegrationTestsDisabled(t)

	ctx, _, browserCtx := setupBrowserTest(t)

	// Navigate to test page
	_, err := browserCtx.Navigate(ctx, "https://example.com", 10000, nil)
	if err != nil {
		t.Fatalf("Failed to navigate: %v", err)
	}

	// Click immediately without calling GetElements first
	// This should trigger lazy loading of elements
	err = browserCtx.Click(ctx, 0, nil)
	if err != nil {
		t.Errorf("Failed to click with lazy loading: %v", err)
	}

	t.Log("Lazy loading click successful")
}

func TestLazyLoadingType(t *testing.T) {
	skipIfIntegrationTestsDisabled(t)

	ctx, _, browserCtx := setupBrowserTest(t)

	// Navigate to a page with input
	_, err := browserCtx.Navigate(ctx, "https://www.google.com", 10000, nil)
	if err != nil {
		t.Fatalf("Failed to navigate: %v", err)
	}

	// Type immediately without calling GetElements first
	// This should trigger lazy loading of elements
	testText := "test"
	zeroIdx := 0
	err = browserCtx.Type(ctx, &zeroIdx, testText, nil)
	if err != nil {
		t.Errorf("Failed to type with lazy loading: %v", err)
	}

	t.Log("Lazy loading type successful")
}
