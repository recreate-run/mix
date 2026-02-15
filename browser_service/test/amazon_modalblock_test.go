package test

import (
	"testing"
	"time"
)

// TestAmazonModalBlocking tests modal blocking on real Amazon.com
func TestAmazonModalBlocking(t *testing.T) {
	skipIfIntegrationTestsDisabled(t)

	// This test requires internet connection
	if testing.Short() {
		t.Skip("Skipping Amazon integration test in short mode")
	}

	ctx, client, cleanup := setupE2ETestWithBlockModals(t, 60)
	defer cleanup()

	// Navigate to Amazon.com
	t.Log("Navigating to Amazon.com...")
	_, err := client.Navigate(ctx, "https://www.amazon.com")
	if err != nil {
		t.Fatalf("Failed to navigate to Amazon: %v", err)
	}

	// Wait for page to fully load and modal to potentially appear
	t.Log("Waiting for page load and modal blocker to process...")
	time.Sleep(8 * time.Second)

	// Check if modal blocker is active
	activeResult, err := client.EvalJS(ctx, "window.__modalBlockActive === true")
	if err != nil {
		t.Fatalf("Failed to check modal blocker status: %v", err)
	}

	if activeResult.Result != true {
		t.Errorf("Modal blocker should be active on Amazon")
	}

	// Check for any script errors
	errorResult, err := client.EvalJS(ctx, "window.__modalBlockError")
	if err != nil {
		t.Fatalf("Failed to check for errors: %v", err)
	}

	if errorResult.Result != nil {
		t.Errorf("Modal blocker error: %v", errorResult.Result)
	}

	// Check if Amazon's delivery location modal is blocked
	// Amazon uses various selectors for their location modal
	modalCheckResult, err := client.EvalJS(ctx, `
		(() => {
			// Check for known Amazon location modal selectors
			const selectors = [
				'[data-testid*="GLUXZipUpdate"]',
				'[aria-label*="location"]',
				'[id*="nav-global-location-popover"]',
				'#GLUXZipUpdateModal',
				'[role="dialog"]'
			];

			let foundModals = [];
			selectors.forEach(selector => {
				const element = document.querySelector(selector);
				if (element) {
					const styles = window.getComputedStyle(element);
					const isVisible = styles.display !== 'none' &&
					                  styles.visibility !== 'hidden' &&
					                  styles.opacity !== '0';
					if (isVisible) {
						foundModals.push({
							selector: selector,
							display: styles.display,
							visibility: styles.visibility,
							opacity: styles.opacity,
							zIndex: styles.zIndex
						});
					}
				}
			});

			return {
				count: foundModals.length,
				modals: foundModals
			};
		})()
	`)
	if err != nil {
		t.Fatalf("Failed to check Amazon modals: %v", err)
	}

	t.Logf("Amazon modal check result: %+v", modalCheckResult.Result)

	// Verify no visible modals
	modalData, ok := modalCheckResult.Result.(map[string]interface{})
	if !ok {
		t.Fatalf("Unexpected modal check result type: %T", modalCheckResult.Result)
	}

	modalCount := int(modalData["count"].(float64))
	if modalCount > 0 {
		t.Errorf("Found %d visible modals on Amazon (expected 0): %+v", modalCount, modalData["modals"])
	}

	// Check if search box is accessible (confirms page is interactive)
	searchBoxResult, err := client.EvalJS(ctx, `
		(() => {
			const searchBox = document.querySelector('#twotabsearchtextbox');
			return searchBox ? 'accessible' : 'not_found';
		})()
	`)
	if err != nil {
		t.Fatalf("Failed to check search box: %v", err)
	}

	if searchBoxResult.Result != "accessible" {
		t.Errorf("Amazon search box should be accessible, got: %v", searchBoxResult.Result)
	}

	// Check that body is scrollable (no scroll lock)
	scrollableResult, err := client.EvalJS(ctx, `
		(() => {
			const bodyOverflow = window.getComputedStyle(document.body).overflow;
			return bodyOverflow !== 'hidden';
		})()
	`)
	if err != nil {
		t.Fatalf("Failed to check scroll state: %v", err)
	}

	if scrollableResult.Result != true {
		t.Errorf("Body should be scrollable (overflow not hidden)")
	}

	t.Log("✅ Amazon modal blocking successful - delivery location popup blocked, page fully interactive")
}
