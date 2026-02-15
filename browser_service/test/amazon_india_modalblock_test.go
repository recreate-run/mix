package test

import (
	"testing"
	"time"
)

// TestAmazonIndiaModalBlocking tests modal blocking on Amazon India (mentioned by user)
func TestAmazonIndiaModalBlocking(t *testing.T) {
	skipIfIntegrationTestsDisabled(t)

	if testing.Short() {
		t.Skip("Skipping Amazon India integration test in short mode")
	}

	ctx, client, cleanup := setupE2ETestWithBlockModals(t, 60)
	defer cleanup()

	// Navigate to Amazon India
	t.Log("Navigating to Amazon.in...")
	_, err := client.Navigate(ctx, "https://www.amazon.in")
	if err != nil {
		t.Fatalf("Failed to navigate to Amazon India: %v", err)
	}

	// Wait for page load and modal processing
	t.Log("Waiting for page load and modal blocker...")
	time.Sleep(8 * time.Second)

	// Check modal blocker is active
	activeResult, err := client.EvalJS(ctx, "window.__modalBlockActive === true")
	if err != nil {
		t.Fatalf("Failed to check modal blocker: %v", err)
	}

	if activeResult.Result != true {
		t.Errorf("Modal blocker should be active")
	}

	// Check for delivery location modal (Amazon India specific)
	modalCheckResult, err := client.EvalJS(ctx, `
		(() => {
			const selectors = [
				'[data-testid*="GLUXZipUpdate"]',
				'[aria-label*="location"]',
				'[aria-label*="delivery"]',
				'[id*="nav-global-location-popover"]',
				'#GLUXZipUpdateModal',
				'[role="dialog"]',
				'[data-action="GLUXPostalInputAction"]'
			];

			let visibleModals = 0;
			selectors.forEach(selector => {
				const el = document.querySelector(selector);
				if (el) {
					const styles = window.getComputedStyle(el);
					if (styles.display !== 'none' &&
					    styles.visibility !== 'hidden' &&
					    styles.opacity !== '0') {
						visibleModals++;
					}
				}
			});

			return visibleModals;
		})()
	`)
	if err != nil {
		t.Fatalf("Failed to check modals: %v", err)
	}

	modalCount, ok := modalCheckResult.Result.(float64)
	if !ok {
		// Try int type
		if intCount, ok := modalCheckResult.Result.(int); ok {
			modalCount = float64(intCount)
		} else {
			t.Fatalf("Unexpected modal count type: %T, value: %v", modalCheckResult.Result, modalCheckResult.Result)
		}
	}

	t.Logf("Visible modals on Amazon India: %.0f", modalCount)

	if modalCount > 0 {
		t.Errorf("Expected 0 visible modals, found %.0f", modalCount)
	}

	// Verify search box is accessible
	searchResult, err := client.EvalJS(ctx, `
		document.querySelector('#twotabsearchtextbox') ? 'found' : 'not_found'
	`)
	if err != nil {
		t.Fatalf("Failed to check search box: %v", err)
	}

	if searchResult.Result != "found" {
		t.Logf("Warning: Search box not found (Amazon India may have different page structure)")
	} else {
		t.Logf("Search box is accessible")
	}

	t.Log("✅ Amazon India modal blocking successful")
}
