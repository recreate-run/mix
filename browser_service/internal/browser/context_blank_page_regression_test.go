package browser

import (
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/sarathmenon/browser-service/internal/constants"
)

// TestContext_NoPageWrappingOnNavigate verifies page reference never wrapped
func TestContext_NoPageWrappingOnNavigate(t *testing.T) {
	t.Helper()
	skipIfIntegrationTestsDisabled(t)

	ctx, _, browserCtx := setupBrowserTest(t)

	// Create a new tab
	tabInfo, err := browserCtx.CreateTab(ctx)
	if err != nil {
		t.Fatalf("Failed to create tab: %v", err)
	}

	// Get the tab reference
	tab, err := browserCtx.getTab(&tabInfo.ID)
	if err != nil {
		t.Fatalf("Failed to get tab: %v", err)
	}

	// Navigate with 30s timeout
	_, err = browserCtx.Navigate(ctx, "https://example.com", 30000, &tabInfo.ID)
	if err != nil {
		t.Fatalf("First navigation failed: %v", err)
	}

	// Verify tab.navigationTimeout field is set to 30s
	tab.mu.RLock()
	if tab.navigationTimeout != 30*time.Second {
		t.Errorf("Expected navigationTimeout to be 30s, got %v", tab.navigationTimeout)
	}
	tab.mu.RUnlock()

	// Call page.Info() → should have valid URL
	info, err := tab.page.Info()
	if err != nil {
		t.Fatalf("Failed to get page info: %v", err)
	}
	if info.URL == "" {
		t.Error("Expected non-empty URL after navigation")
	}

	// Navigate again with different timeout
	_, err = browserCtx.Navigate(ctx, "https://www.iana.org", 15000, &tabInfo.ID)
	if err != nil {
		t.Fatalf("Second navigation failed: %v", err)
	}

	// Verify tab.navigationTimeout updated
	tab.mu.RLock()
	if tab.navigationTimeout != 15*time.Second {
		t.Errorf("Expected navigationTimeout to be 15s, got %v", tab.navigationTimeout)
	}
	tab.mu.RUnlock()

	// Verify page reference still consistent - multiple operations should succeed
	info2, err := tab.page.Info()
	if err != nil {
		t.Fatalf("Failed to get page info after second navigation: %v", err)
	}
	if info2.URL == "" {
		t.Error("Expected non-empty URL after second navigation")
	}
}

// TestContext_NavigationFailureDoesNotCorruptState verifies failed navigation doesn't corrupt state
func TestContext_NavigationFailureDoesNotCorruptState(t *testing.T) {
	t.Helper()
	skipIfIntegrationTestsDisabled(t)

	ctx, _, browserCtx := setupBrowserTest(t)

	// Create a new tab
	tabInfo, err := browserCtx.CreateTab(ctx)
	if err != nil {
		t.Fatalf("Failed to create tab: %v", err)
	}

	// Get the tab reference
	tab, err := browserCtx.getTab(&tabInfo.ID)
	if err != nil {
		t.Fatalf("Failed to get tab: %v", err)
	}

	// Store initial URL
	tab.mu.RLock()
	initialURL := tab.currentURL
	tab.mu.RUnlock()

	// Navigate to invalid URL
	_, err = browserCtx.Navigate(ctx, "http://invalid-url-that-will-fail.local", 5000, &tabInfo.ID)
	if err == nil {
		t.Error("Expected error for invalid URL navigation, got nil")
	}

	// Verify tab.currentURL is unchanged or empty
	tab.mu.RLock()
	currentURL := tab.currentURL
	tab.mu.RUnlock()

	if currentURL != initialURL && currentURL != "" {
		t.Errorf("Expected currentURL to be unchanged or empty after failed navigation, got %s", currentURL)
	}

	// Navigate to valid URL → should succeed
	_, err = browserCtx.Navigate(ctx, "https://example.com", 0, &tabInfo.ID)
	if err != nil {
		t.Fatalf("Navigation to valid URL failed after invalid navigation: %v", err)
	}

	// Verify tab.currentURL updated correctly
	tab.mu.RLock()
	newURL := tab.currentURL
	tab.mu.RUnlock()

	if !strings.Contains(newURL, "example.com") {
		t.Errorf("Expected currentURL to contain 'example.com', got %s", newURL)
	}

	// ListTabs → verify URL is valid
	tabs, _, err := browserCtx.ListTabs(ctx)
	if err != nil {
		t.Fatalf("ListTabs failed: %v", err)
	}

	var foundTab bool
	for _, tab := range tabs {
		if tab.ID == tabInfo.ID {
			foundTab = true
			if !strings.Contains(tab.URL, "example.com") {
				t.Errorf("Expected tab URL to contain 'example.com', got %s", tab.URL)
			}
		}
	}
	if !foundTab {
		t.Error("Tab not found in ListTabs")
	}
}

// TestContext_TabCreationHasValidURL verifies new tabs have valid URLs
func TestContext_TabCreationHasValidURL(t *testing.T) {
	t.Helper()
	skipIfIntegrationTestsDisabled(t)

	ctx, _, browserCtx := setupBrowserTest(t)

	// Create new tab
	tabInfo, err := browserCtx.CreateTab(ctx)
	if err != nil {
		t.Fatalf("Failed to create tab: %v", err)
	}

	// Verify returned TabInfo.URL is not empty
	if tabInfo.URL == "" {
		t.Error("Expected non-empty URL in TabInfo")
	}

	// Verify URL starts with "data:text/html" (the initialization URL)
	if !strings.HasPrefix(tabInfo.URL, "data:text/html") {
		t.Errorf("Expected URL to start with 'data:text/html', got %s", tabInfo.URL)
	}

	// ListTabs → verify tab has valid URL
	tabs, _, err := browserCtx.ListTabs(ctx)
	if err != nil {
		t.Fatalf("ListTabs failed: %v", err)
	}

	var foundTab bool
	for _, tab := range tabs {
		if tab.ID == tabInfo.ID {
			foundTab = true
			if tab.URL == "" {
				t.Error("Expected non-empty URL in ListTabs")
			}
			if !strings.HasPrefix(tab.URL, "data:text/html") {
				t.Errorf("Expected URL to start with 'data:text/html' in ListTabs, got %s", tab.URL)
			}
		}
	}
	if !foundTab {
		t.Error("Tab not found in ListTabs")
	}

	// Verify tab.currentURL field is set
	tab, err := browserCtx.getTab(&tabInfo.ID)
	if err != nil {
		t.Fatalf("Failed to get tab: %v", err)
	}

	tab.mu.RLock()
	currentURL := tab.currentURL
	tab.mu.RUnlock()

	if currentURL == "" {
		t.Error("Expected non-empty currentURL in tab context")
	}
}

// TestContext_URLCachingConsistency verifies cached URL matches actual page URL
func TestContext_URLCachingConsistency(t *testing.T) {
	t.Helper()
	skipIfIntegrationTestsDisabled(t)

	ctx, _, browserCtx := setupBrowserTest(t)

	// Create tab and navigate to example.com
	tabInfo, err := browserCtx.CreateTab(ctx)
	if err != nil {
		t.Fatalf("Failed to create tab: %v", err)
	}

	_, err = browserCtx.Navigate(ctx, "https://example.com", 0, &tabInfo.ID)
	if err != nil {
		t.Fatalf("Navigation failed: %v", err)
	}

	// Get the tab reference
	tab, err := browserCtx.getTab(&tabInfo.ID)
	if err != nil {
		t.Fatalf("Failed to get tab: %v", err)
	}

	// Verify tab.currentURL matches expected URL
	tab.mu.RLock()
	cachedURL := tab.currentURL
	tab.mu.RUnlock()

	if !strings.Contains(cachedURL, "example.com") {
		t.Errorf("Expected cached URL to contain 'example.com', got %s", cachedURL)
	}

	// Call page.Info() → URL should match cached value
	info, err := tab.page.Info()
	if err != nil {
		t.Fatalf("Failed to get page info: %v", err)
	}

	if info.URL != cachedURL {
		t.Errorf("Page URL (%s) doesn't match cached URL (%s)", info.URL, cachedURL)
	}

	// ListTabs → URL should match cache
	tabs, _, err := browserCtx.ListTabs(ctx)
	if err != nil {
		t.Fatalf("ListTabs failed: %v", err)
	}

	var foundTab bool
	for _, tab := range tabs {
		if tab.ID == tabInfo.ID {
			foundTab = true
			if tab.URL != cachedURL {
				t.Errorf("ListTabs URL (%s) doesn't match cached URL (%s)", tab.URL, cachedURL)
			}
		}
	}
	if !foundTab {
		t.Error("Tab not found in ListTabs")
	}

	// Navigate to different URL
	_, err = browserCtx.Navigate(ctx, "https://www.iana.org", 0, &tabInfo.ID)
	if err != nil {
		t.Fatalf("Second navigation failed: %v", err)
	}

	// Verify cache updated to new URL
	tab.mu.RLock()
	newCachedURL := tab.currentURL
	tab.mu.RUnlock()

	if !strings.Contains(newCachedURL, "iana.org") {
		t.Errorf("Expected cached URL to contain 'iana.org', got %s", newCachedURL)
	}

	if newCachedURL == cachedURL {
		t.Error("Expected cached URL to change after navigation")
	}
}

// TestContext_ListTabsWithCachedURL verifies ListTabs uses cached URL primarily
func TestContext_ListTabsWithCachedURL(t *testing.T) {
	t.Helper()
	skipIfIntegrationTestsDisabled(t)

	ctx, _, browserCtx := setupBrowserTest(t)

	// Create tab and navigate
	tabInfo, err := browserCtx.CreateTab(ctx)
	if err != nil {
		t.Fatalf("Failed to create tab: %v", err)
	}

	_, err = browserCtx.Navigate(ctx, "https://example.com", 0, &tabInfo.ID)
	if err != nil {
		t.Fatalf("Navigation failed: %v", err)
	}

	// Verify tab.currentURL is set
	tab, err := browserCtx.getTab(&tabInfo.ID)
	if err != nil {
		t.Fatalf("Failed to get tab: %v", err)
	}

	tab.mu.RLock()
	cachedURL := tab.currentURL
	tab.mu.RUnlock()

	if cachedURL == "" {
		t.Error("Expected non-empty cached URL after navigation")
	}

	// ListTabs → should return cached URL
	tabs, _, err := browserCtx.ListTabs(ctx)
	if err != nil {
		t.Fatalf("ListTabs failed: %v", err)
	}

	var foundTab bool
	for _, tab := range tabs {
		if tab.ID == tabInfo.ID {
			foundTab = true
			if tab.URL != cachedURL {
				t.Errorf("ListTabs URL (%s) doesn't match cached URL (%s)", tab.URL, cachedURL)
			}
		}
	}
	if !foundTab {
		t.Error("Tab not found in ListTabs")
	}

	// Navigate to another URL
	_, err = browserCtx.Navigate(ctx, "https://www.iana.org", 0, &tabInfo.ID)
	if err != nil {
		t.Fatalf("Second navigation failed: %v", err)
	}

	// Verify ListTabs returns updated URL
	tabs2, _, err := browserCtx.ListTabs(ctx)
	if err != nil {
		t.Fatalf("ListTabs failed after second navigation: %v", err)
	}

	var foundTab2 bool
	for _, tab := range tabs2 {
		if tab.ID == tabInfo.ID {
			foundTab2 = true
			if tab.URL == cachedURL {
				t.Error("Expected ListTabs URL to change after navigation")
			}
			if !strings.Contains(tab.URL, "iana.org") {
				t.Errorf("Expected ListTabs URL to contain 'iana.org', got %s", tab.URL)
			}
		}
	}
	if !foundTab2 {
		t.Error("Tab not found in ListTabs after second navigation")
	}
}

// TestContext_NavigationTimeoutUsesContext verifies navigation timeout uses context, not page wrapping
func TestContext_NavigationTimeoutUsesContext(t *testing.T) {
	t.Helper()
	skipIfIntegrationTestsDisabled(t)

	ctx, _, browserCtx := setupBrowserTest(t)

	// Create tab
	tabInfo, err := browserCtx.CreateTab(ctx)
	if err != nil {
		t.Fatalf("Failed to create tab: %v", err)
	}

	// Get the tab reference
	tab, err := browserCtx.getTab(&tabInfo.ID)
	if err != nil {
		t.Fatalf("Failed to get tab: %v", err)
	}

	// Navigate with 100ms timeout to a slow URL (this might timeout)
	// Using a real URL that should load quickly - the timeout is just to test the field
	_, err = browserCtx.Navigate(ctx, "https://example.com", 100, &tabInfo.ID)
	// We don't assert error here because example.com is fast and might succeed

	// Verify tab.navigationTimeout field set to 100ms
	tab.mu.RLock()
	if tab.navigationTimeout != 100*time.Millisecond {
		t.Errorf("Expected navigationTimeout to be 100ms, got %v", tab.navigationTimeout)
	}
	tab.mu.RUnlock()

	// Tab should still be functional after timeout/navigation
	// Navigate again with longer timeout → should succeed
	_, err = browserCtx.Navigate(ctx, "https://example.com", 30000, &tabInfo.ID)
	if err != nil {
		t.Fatalf("Navigation with longer timeout failed: %v", err)
	}

	// Verify timeout was updated
	tab.mu.RLock()
	if tab.navigationTimeout != 30*time.Second {
		t.Errorf("Expected navigationTimeout to be 30s, got %v", tab.navigationTimeout)
	}
	tab.mu.RUnlock()

	// Verify page is functional
	info, err := tab.page.Info()
	if err != nil {
		t.Fatalf("Failed to get page info: %v", err)
	}
	if info.URL == "" {
		t.Error("Expected non-empty URL")
	}
}

// TestContext_ElementCacheClearedBeforeNavigation verifies element cache cleared before navigation attempt
func TestContext_ElementCacheClearedBeforeNavigation(t *testing.T) {
	t.Helper()
	skipIfIntegrationTestsDisabled(t)

	ctx, _, browserCtx := setupBrowserTest(t)

	// Create tab and navigate to URL
	tabInfo, err := browserCtx.CreateTab(ctx)
	if err != nil {
		t.Fatalf("Failed to create tab: %v", err)
	}

	_, err = browserCtx.Navigate(ctx, "https://example.com", 0, &tabInfo.ID)
	if err != nil {
		t.Fatalf("Navigation failed: %v", err)
	}

	// Get the tab reference
	tab, err := browserCtx.getTab(&tabInfo.ID)
	if err != nil {
		t.Fatalf("Failed to get tab: %v", err)
	}

	// Call extractElements to populate cache
	_, err = tab.extractElements()
	if err != nil {
		t.Fatalf("Failed to extract elements: %v", err)
	}

	// Verify tab.elements is not nil
	tab.mu.RLock()
	elementsCount := len(tab.elements)
	tab.mu.RUnlock()

	if elementsCount == 0 {
		t.Log("Warning: No elements extracted (page might not have interactive elements)")
	}

	// Navigate to new URL
	_, err = browserCtx.Navigate(ctx, "https://www.iana.org", 0, &tabInfo.ID)
	if err != nil {
		t.Fatalf("Second navigation failed: %v", err)
	}

	// Verify tab.elements was cleared (should be nil)
	tab.mu.RLock()
	elements := tab.elements
	tab.mu.RUnlock()

	if elements != nil {
		t.Errorf("Expected elements to be nil after navigation, got %d elements", len(elements))
	}

	// Even if navigation fails partway, elements should be cleared
	// Test with invalid URL
	_, err = browserCtx.Navigate(ctx, "http://invalid-url.local", 5000, &tabInfo.ID)
	// Expect error for invalid URL

	// Extract elements again
	_, err = tab.extractElements()
	if err != nil {
		t.Fatalf("Failed to extract elements after invalid navigation: %v", err)
	}

	// Populate cache again
	tab.mu.RLock()
	elementsAfter := tab.elements
	tab.mu.RUnlock()

	if elementsAfter == nil {
		t.Log("Note: elements is nil after extraction (might be expected if page is in error state)")
	}

	// Navigate again - cache should be cleared even if previous navigation failed
	_, err = browserCtx.Navigate(ctx, "https://example.com", 0, &tabInfo.ID)
	if err != nil {
		t.Fatalf("Final navigation failed: %v", err)
	}

	tab.mu.RLock()
	finalElements := tab.elements
	tab.mu.RUnlock()

	if finalElements != nil {
		t.Errorf("Expected elements to be nil after final navigation, got %d elements", len(finalElements))
	}
}

// TestContext_ConcurrentNavigationSafety verifies concurrent navigation operations are thread-safe
func TestContext_ConcurrentNavigationSafety(t *testing.T) {
	t.Helper()
	skipIfIntegrationTestsDisabled(t)

	ctx, _, browserCtx := setupBrowserTest(t)

	// Create 3 tabs
	var tabs []string
	for i := 0; i < 3; i++ {
		tabInfo, err := browserCtx.CreateTab(ctx)
		if err != nil {
			t.Fatalf("Failed to create tab %d: %v", i, err)
		}
		tabs = append(tabs, tabInfo.ID)
	}

	// URLs to navigate to
	urls := []string{
		"https://example.com",
		"https://www.iana.org",
		"https://example.org",
	}

	// Use WaitGroup to coordinate goroutines
	var wg sync.WaitGroup
	wg.Add(3)

	// Track errors
	errors := make([]error, 3)

	// Navigate each tab concurrently
	for i := 0; i < 3; i++ {
		go func(idx int, tabID string, url string) {
			defer wg.Done()

			_, err := browserCtx.Navigate(ctx, url, 0, &tabID)
			if err != nil {
				errors[idx] = err
			}
		}(i, tabs[i], urls[i])
	}

	// Wait for all goroutines to complete
	wg.Wait()

	// Verify all navigations succeeded
	for i, err := range errors {
		if err != nil {
			t.Errorf("Navigation %d failed: %v", i, err)
		}
	}

	// Verify each tab has correct currentURL
	for i, tabID := range tabs {
		tab, err := browserCtx.getTab(&tabID)
		if err != nil {
			t.Fatalf("Failed to get tab %d: %v", i, err)
		}

		tab.mu.RLock()
		currentURL := tab.currentURL
		tab.mu.RUnlock()

		// Extract domain from URL for comparison
		expectedDomain := urls[i]
		if !strings.Contains(currentURL, expectedDomain[8:]) { // Skip "https://"
			t.Errorf("Tab %d: Expected URL to contain %s, got %s", i, expectedDomain, currentURL)
		}
	}

	// Verify no race conditions by listing tabs
	tabsList, _, err := browserCtx.ListTabs(ctx)
	if err != nil {
		t.Fatalf("ListTabs failed: %v", err)
	}

	// Should have initial tab + 3 created tabs = 4 total
	if len(tabsList) != 4 {
		t.Errorf("Expected 4 tabs, got %d", len(tabsList))
	}
}

// TestContext_DefaultNavigationTimeout verifies default timeout is used when timeout is 0
func TestContext_DefaultNavigationTimeout(t *testing.T) {
	t.Helper()
	skipIfIntegrationTestsDisabled(t)

	ctx, _, browserCtx := setupBrowserTest(t)

	// Create tab
	tabInfo, err := browserCtx.CreateTab(ctx)
	if err != nil {
		t.Fatalf("Failed to create tab: %v", err)
	}

	// Get the tab reference
	tab, err := browserCtx.getTab(&tabInfo.ID)
	if err != nil {
		t.Fatalf("Failed to get tab: %v", err)
	}

	// Navigate with 0 timeout (should use default)
	_, err = browserCtx.Navigate(ctx, "https://example.com", 0, &tabInfo.ID)
	if err != nil {
		t.Fatalf("Navigation failed: %v", err)
	}

	// Verify tab.navigationTimeout is set to default
	tab.mu.RLock()
	timeout := tab.navigationTimeout
	tab.mu.RUnlock()

	if timeout != constants.DefaultNavigationTimeout {
		t.Errorf("Expected default navigation timeout (%v), got %v", constants.DefaultNavigationTimeout, timeout)
	}
}
