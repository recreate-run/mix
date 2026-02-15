package test

import (
	"strings"
	"testing"
	"time"

	"github.com/sarathmenon/browser-service/test/testserver"
)

func TestModalBlockingEnabled(t *testing.T) {
	skipIfIntegrationTestsDisabled(t)

	server := testserver.StartTestServer(t)
	defer server.Close()

	// Connect with modal blocking enabled
	ctx, client, cleanup := setupE2ETestWithBlockModals(t, 30)
	defer cleanup()

	// Navigate to modal test page (has a location/delivery popup like Amazon)
	_, err := client.Navigate(ctx, server.URL+"/modal-popup-page")
	if err != nil {
		t.Fatalf("Failed to navigate: %v", err)
	}

	// Wait for page load and modal blocker to run
	time.Sleep(2 * time.Second)

	// Check if modal blocker is active
	result, err := client.EvalJS(ctx, "window.__modalBlockActive === true")
	if err != nil {
		t.Fatalf("Failed to check modal blocker status: %v", err)
	}

	modalBlockActive, ok := result.Result.(bool)
	if !ok {
		t.Fatalf("Expected boolean result, got %T: %v", result.Result, result.Result)
	}

	if !modalBlockActive {
		t.Errorf("Expected modal blocker to be active, but window.__modalBlockActive is not true")
	}

	// Check if modal is hidden via CSS
	modalHiddenResult, err := client.EvalJS(ctx, `
		(() => {
			const modal = document.querySelector('[role="dialog"]');
			if (!modal) return 'modal_not_found';
			const styles = window.getComputedStyle(modal);
			return styles.display === 'none' ? 'hidden' : 'visible';
		})()
	`)
	if err != nil {
		t.Fatalf("Failed to check modal visibility: %v", err)
	}

	t.Logf("Modal visibility result: %v", modalHiddenResult.Result)

	modalStatus, ok := modalHiddenResult.Result.(string)
	if !ok {
		t.Fatalf("Expected string result for modal status, got %T: %v", modalHiddenResult.Result, modalHiddenResult.Result)
	}

	// Modal should be either removed, hidden, or not found (all are success states)
	if modalStatus != "hidden" && modalStatus != "removed" && modalStatus != "modal_not_found" {
		t.Errorf("Expected modal to be blocked (hidden/removed/not_found), got status: %s", modalStatus)
	}

	// Verify page content is accessible (not blocked by overlay)
	textResult, err := client.GetText(ctx, "body")
	if err != nil {
		t.Fatalf("Failed to get page text: %v", err)
	}

	t.Logf("Page text: %s", textResult.Text)

	// Should see main content, not modal content
	if !strings.Contains(textResult.Text, "Main Page Content") {
		t.Errorf("Expected to see main page content, got: %s", textResult.Text)
	}

	// Modal message should be blocked (not visible in text)
	if strings.Contains(textResult.Text, "This is a modal popup") {
		t.Logf("Warning: Modal text is visible in page text (may indicate blocker didn't hide it)")
	}

	// Verify no scroll lock on body
	scrollCheckResult, err := client.EvalJS(ctx, "document.body.style.overflow !== 'hidden'")
	if err != nil {
		t.Fatalf("Failed to check body scroll: %v", err)
	}

	canScroll, ok := scrollCheckResult.Result.(bool)
	if !ok {
		t.Fatalf("Expected boolean for scroll check, got %T", scrollCheckResult.Result)
	}

	if !canScroll {
		t.Errorf("Expected body to be scrollable (overflow not hidden)")
	}

	t.Log("✅ Modal blocker successfully blocked HTML modal popup")
}

func TestModalBlockingCookieBanner(t *testing.T) {
	skipIfIntegrationTestsDisabled(t)

	server := testserver.StartTestServer(t)
	defer server.Close()

	ctx, client, cleanup := setupE2ETestWithBlockModals(t, 30)
	defer cleanup()

	// Navigate to page with cookie consent banner
	_, err := client.Navigate(ctx, server.URL+"/cookie-banner-page")
	if err != nil {
		t.Fatalf("Failed to navigate: %v", err)
	}

	time.Sleep(2 * time.Second)

	// Check if cookie banner is hidden
	bannerHiddenResult, err := client.EvalJS(ctx, `
		(() => {
			const banner = document.querySelector('[id*="cookie"]');
			if (!banner) return 'banner_not_found';
			const styles = window.getComputedStyle(banner);
			return styles.display === 'none' ? 'hidden' : 'visible';
		})()
	`)
	if err != nil {
		t.Fatalf("Failed to check cookie banner visibility: %v", err)
	}

	bannerStatus, ok := bannerHiddenResult.Result.(string)
	if !ok {
		t.Fatalf("Expected string result, got %T", bannerHiddenResult.Result)
	}

	// Banner should be either removed, hidden, or not found (all are success states)
	if bannerStatus != "hidden" && bannerStatus != "removed" && bannerStatus != "banner_not_found" {
		t.Errorf("Expected cookie banner to be blocked (hidden/removed/not_found), got status: %s", bannerStatus)
	}

	// Verify main content is accessible
	textResult, err := client.GetText(ctx, "body")
	if err != nil {
		t.Fatalf("Failed to get page text: %v", err)
	}

	if !strings.Contains(textResult.Text, "Welcome to our site") {
		t.Errorf("Expected to see main site content, got: %s", textResult.Text)
	}

	t.Log("✅ Modal blocker successfully blocked cookie consent banner")
}

func TestModalBlockingDynamicModal(t *testing.T) {
	skipIfIntegrationTestsDisabled(t)

	server := testserver.StartTestServer(t)
	defer server.Close()

	ctx, client, cleanup := setupE2ETestWithBlockModals(t, 30)
	defer cleanup()

	// Navigate to page that will dynamically create a modal
	_, err := client.Navigate(ctx, server.URL+"/dynamic-modal-page")
	if err != nil {
		t.Fatalf("Failed to navigate: %v", err)
	}

	time.Sleep(1 * time.Second)

	// Get elements to find the trigger button
	elements, err := client.GetElements(ctx)
	if err != nil {
		t.Fatalf("Failed to get elements: %v", err)
	}

	// Find "Show Modal" button
	buttonIdx := -1
	for i, elem := range elements {
		if elem.Role == "button" && elem.Name == "Show Modal" {
			buttonIdx = i
			break
		}
	}

	if buttonIdx == -1 {
		t.Fatalf("Show Modal button not found")
	}

	// Click button to trigger dynamic modal
	err = client.Click(ctx, buttonIdx)
	if err != nil {
		t.Fatalf("Failed to click button: %v", err)
	}

	// Wait for modal to be created and removed by observer
	time.Sleep(2 * time.Second)

	// Check if dynamically created modal was removed
	modalCheckResult, err := client.EvalJS(ctx, `
		(() => {
			const modal = document.getElementById('dynamic-modal');
			if (!modal) return 'removed';
			const styles = window.getComputedStyle(modal);
			return styles.display === 'none' ? 'hidden' : 'visible';
		})()
	`)
	if err != nil {
		t.Fatalf("Failed to check dynamic modal: %v", err)
	}

	modalStatus, ok := modalCheckResult.Result.(string)
	if !ok {
		t.Fatalf("Expected string result, got %T", modalCheckResult.Result)
	}

	// Modal should either be removed or hidden
	if modalStatus != "removed" && modalStatus != "hidden" {
		t.Errorf("Expected dynamic modal to be removed or hidden, got status: %s", modalStatus)
	}

	t.Log("✅ Modal blocker successfully blocked dynamically created modal")
}
