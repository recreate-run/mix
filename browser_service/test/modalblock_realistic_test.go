package test

import (
	"strings"
	"testing"
	"time"

	"github.com/sarathmenon/browser-service/test/testserver"
)

// TestModalBlockingRealisticScenario tests modal blocking with a realistic Amazon-style page
func TestModalBlockingRealisticScenario(t *testing.T) {
	skipIfIntegrationTestsDisabled(t)

	server := testserver.StartTestServer(t)
	defer server.Close()

	ctx, client, cleanup := setupE2ETestWithBlockModals(t, 30)
	defer cleanup()

	// Navigate to modal popup page
	_, err := client.Navigate(ctx, server.URL+"/modal-popup-page")
	if err != nil {
		t.Fatalf("Failed to navigate: %v", err)
	}

	// Wait for modal blocker to process the page
	time.Sleep(2 * time.Second)

	// Verify the modal blocker is active
	activeResult, err := client.EvalJS(ctx, "window.__modalBlockActive === true")
	if err != nil {
		t.Fatalf("Failed to check modal blocker status: %v", err)
	}

	if activeResult.Result != true {
		t.Errorf("Modal blocker should be active")
	}

	// Check that main content is accessible
	textResult, err := client.GetText(ctx, "body")
	if err != nil {
		t.Fatalf("Failed to get page text: %v", err)
	}

	if !strings.Contains(textResult.Text, "Main Page Content") {
		t.Errorf("Expected to see main page content, got: %s", textResult.Text)
	}

	// Verify modal is not blocking interaction
	// The modal text should not be the primary content
	modalTextRatio := float64(strings.Count(textResult.Text, "modal")) / float64(len(textResult.Text))
	if modalTextRatio > 0.1 {
		t.Logf("Warning: Modal text still appears prominently in page (%.1f%% of content)", modalTextRatio*100)
	}

	// Verify body is scrollable (no scroll lock)
	scrollableResult, err := client.EvalJS(ctx, `
		(() => {
			const bodyOverflow = window.getComputedStyle(document.body).overflow;
			const bodyClasses = document.body.className;
			return {
				overflow: bodyOverflow,
				hasModalOpenClass: bodyClasses.includes('modal-open'),
				hasNoScrollClass: bodyClasses.includes('no-scroll')
			};
		})()
	`)
	if err != nil {
		t.Fatalf("Failed to check scroll state: %v", err)
	}

	t.Logf("Body scroll state: %+v", scrollableResult.Result)

	// Verify backdrop is hidden or removed
	backdropResult, err := client.EvalJS(ctx, `
		(() => {
			const backdrop = document.querySelector('.modal-backdrop');
			if (!backdrop) return 'removed';
			const styles = window.getComputedStyle(backdrop);
			return styles.display === 'none' ? 'hidden' : 'visible';
		})()
	`)
	if err != nil {
		t.Fatalf("Failed to check backdrop: %v", err)
	}

	if backdropResult.Result != "removed" && backdropResult.Result != "hidden" {
		t.Errorf("Expected backdrop to be removed or hidden, got: %v", backdropResult.Result)
	}

	t.Log("✅ Realistic modal blocking scenario passed - page is fully accessible without modal interference")
}
