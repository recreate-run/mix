package browser

import (
	"context"
	"os"
	"strings"
	"testing"

	"github.com/sarathmenon/browser-service/pkg/protocol"
)

// skipIfIntegrationTestsDisabled skips integration tests if SKIP_INTEGRATION_TESTS env var is set
func skipIfIntegrationTestsDisabled(t *testing.T) {
	t.Helper()
	if os.Getenv("SKIP_INTEGRATION_TESTS") != "" {
		t.Skip("Skipping integration test")
	}
}

// setupBrowserTest creates a browser manager and context for integration tests
// Returns context, manager, and browser context. Cleanup is handled automatically via t.Cleanup.
func setupBrowserTest(t *testing.T) (ctx context.Context, mgr *Manager, browserCtx *Context) {
	t.Helper()
	ctx = context.Background()

	mgr, err := NewManager(ctx, Config{Headless: true})
	if err != nil {
		t.Fatalf("Failed to create browser manager: %v", err)
	}
	t.Cleanup(func() {
		if err := mgr.Close(ctx); err != nil {
			t.Errorf("Failed to close manager: %v", err)
		}
	})

	browserCtx, err = mgr.NewContext(ctx)
	if err != nil {
		t.Fatalf("Failed to create browser context: %v", err)
	}
	t.Cleanup(func() {
		if err := browserCtx.Close(ctx); err != nil {
			t.Errorf("Failed to close browser context: %v", err)
		}
	})

	return ctx, mgr, browserCtx
}

// findElementByRole finds the first element matching any of the given role(s)
// Returns array position (to use as index parameter) and true if found, -1 and false otherwise
func findElementByRole(elements []protocol.RawAccessibilityNode, roles ...string) (int, bool) {
	roleMap := make(map[string]bool)
	for _, r := range roles {
		roleMap[strings.ToLower(r)] = true
	}

	for i, elem := range elements {
		if roleMap[strings.ToLower(elem.Role)] {
			return i, true
		}
	}
	return -1, false
}
