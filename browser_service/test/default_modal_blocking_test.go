package test

import (
	"testing"
	"time"

	"github.com/sarathmenon/browser-service/test/testserver"
)

// TestModalBlockingEnabledByDefault verifies modal blocking is active without any flags
func TestModalBlockingEnabledByDefault(t *testing.T) {
	skipIfIntegrationTestsDisabled(t)

	server := testserver.StartTestServer(t)
	defer server.Close()

	// Use standard setup (no special flags) - modal blocking should be ON by default
	ctx, client, cleanup := setupE2ETest(t, 30)
	defer cleanup()

	// Navigate to modal test page
	_, err := client.Navigate(ctx, server.URL+"/modal-popup-page")
	if err != nil {
		t.Fatalf("Failed to navigate: %v", err)
	}

	time.Sleep(2 * time.Second)

	// Check if modal blocker is active (should be true by default)
	activeResult, err := client.EvalJS(ctx, "window.__modalBlockActive === true")
	if err != nil {
		t.Fatalf("Failed to check modal blocker: %v", err)
	}

	if activeResult.Result != true {
		t.Errorf("Modal blocker should be ENABLED by default, but got: %v", activeResult.Result)
	}

	// Check modal is blocked
	modalCheckResult, err := client.EvalJS(ctx, `
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

	modalStatus, ok := modalCheckResult.Result.(string)
	if !ok {
		t.Fatalf("Unexpected result type: %T", modalCheckResult.Result)
	}

	if modalStatus != "hidden" && modalStatus != "not_found" {
		t.Errorf("Modal should be blocked by default, got status: %s", modalStatus)
	}

	t.Log("✅ Modal blocking is ENABLED by default - no flags needed!")
}
