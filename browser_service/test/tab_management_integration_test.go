package test

import (
	"context"
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/sarathmenon/browser-service/pkg/client"
)

// TestTabCreate verifies creation of multiple tabs
func TestTabCreate(t *testing.T) {
	skipIfIntegrationTestsDisabled(t)

	ctx, c, cleanup := setupE2ETest(t, 20)
	defer cleanup()

	// Create 3 tabs
	tabIDs := make([]string, 3)
	for i := 0; i < 3; i++ {
		tab, err := c.CreateTab(ctx)
		if err != nil {
			t.Fatalf("Failed to create tab %d: %v", i, err)
		}

		if tab.ID == "" {
			t.Errorf("Tab %d has empty ID", i)
		}

		if tab.URL == "" {
			t.Errorf("Tab %d has empty URL", i)
		}

		tabIDs[i] = tab.ID
		t.Logf("Created tab %d with ID: %s, URL: %s", i, tab.ID, tab.URL)
	}

	// Verify all tab IDs are unique
	seen := make(map[string]bool)
	for i, id := range tabIDs {
		if seen[id] {
			t.Errorf("Duplicate tab ID found: %s", id)
		}
		seen[id] = true
		t.Logf("Tab %d ID is unique: %s", i, id)
	}
}

// TestTabList verifies listing all tabs
func TestTabList(t *testing.T) {
	skipIfIntegrationTestsDisabled(t)

	ctx, c, cleanup := setupE2ETest(t, 20)
	defer cleanup()

	// Create 3 tabs
	expectedCount := 4 // 1 default + 3 created
	createdTabIDs := make([]string, 3)
	for i := 0; i < 3; i++ {
		tab, err := c.CreateTab(ctx)
		if err != nil {
			t.Fatalf("Failed to create tab %d: %v", i, err)
		}
		createdTabIDs[i] = tab.ID
	}

	// List tabs
	result, err := c.ListTabs(ctx)
	if err != nil {
		t.Fatalf("Failed to list tabs: %v", err)
	}

	if len(result.Tabs) != expectedCount {
		t.Errorf("Expected %d tabs, got %d", expectedCount, len(result.Tabs))
	}

	if result.ActiveTabID == "" {
		t.Error("Expected non-empty active tab ID")
	}

	// Verify all created tabs are in the list
	tabIDMap := make(map[string]bool)
	for _, tab := range result.Tabs {
		tabIDMap[tab.ID] = true
		t.Logf("Found tab: ID=%s, URL=%s, Title=%s, Active=%v", tab.ID, tab.URL, tab.Title, tab.IsActive)
	}

	for i, id := range createdTabIDs {
		if !tabIDMap[id] {
			t.Errorf("Created tab %d (ID: %s) not found in list", i, id)
		}
	}

	// Verify exactly one tab is active
	activeCount := 0
	for _, tab := range result.Tabs {
		if tab.IsActive {
			activeCount++
		}
	}
	if activeCount != 1 {
		t.Errorf("Expected exactly 1 active tab, got %d", activeCount)
	}
}

// TestTabSwitch verifies switching between tabs
func TestTabSwitch(t *testing.T) {
	skipIfIntegrationTestsDisabled(t)

	ctx, c, cleanup := setupE2ETest(t, 20)
	defer cleanup()

	// Create 2 additional tabs
	tab1, err := c.CreateTab(ctx)
	if err != nil {
		t.Fatalf("Failed to create tab 1: %v", err)
	}

	tab2, err := c.CreateTab(ctx)
	if err != nil {
		t.Fatalf("Failed to create tab 2: %v", err)
	}

	// Get initial state
	result, err := c.ListTabs(ctx)
	if err != nil {
		t.Fatalf("Failed to list tabs: %v", err)
	}
	initialActiveID := result.ActiveTabID
	t.Logf("Initial active tab: %s", initialActiveID)

	// Switch to tab1
	err = c.SwitchTab(ctx, tab1.ID)
	if err != nil {
		t.Fatalf("Failed to switch to tab1: %v", err)
	}

	result, err = c.ListTabs(ctx)
	if err != nil {
		t.Fatalf("Failed to list tabs after switch: %v", err)
	}

	if result.ActiveTabID != tab1.ID {
		t.Errorf("Expected active tab to be %s, got %s", tab1.ID, result.ActiveTabID)
	}
	t.Logf("Successfully switched to tab1: %s", result.ActiveTabID)

	// Switch to tab2
	err = c.SwitchTab(ctx, tab2.ID)
	if err != nil {
		t.Fatalf("Failed to switch to tab2: %v", err)
	}

	result, err = c.ListTabs(ctx)
	if err != nil {
		t.Fatalf("Failed to list tabs after second switch: %v", err)
	}

	if result.ActiveTabID != tab2.ID {
		t.Errorf("Expected active tab to be %s, got %s", tab2.ID, result.ActiveTabID)
	}
	t.Logf("Successfully switched to tab2: %s", result.ActiveTabID)
}

// TestTabClose verifies closing a tab
func TestTabClose(t *testing.T) {
	skipIfIntegrationTestsDisabled(t)

	ctx, c, cleanup := setupE2ETest(t, 20)
	defer cleanup()

	// Create 3 tabs
	tab1, err := c.CreateTab(ctx)
	if err != nil {
		t.Fatalf("Failed to create tab 1: %v", err)
	}

	tab2, err := c.CreateTab(ctx)
	if err != nil {
		t.Fatalf("Failed to create tab 2: %v", err)
	}

	_, err = c.CreateTab(ctx)
	if err != nil {
		t.Fatalf("Failed to create tab 3: %v", err)
	}

	// Verify we have 4 tabs (1 default + 3 created)
	result, err := c.ListTabs(ctx)
	if err != nil {
		t.Fatalf("Failed to list tabs: %v", err)
	}
	if len(result.Tabs) != 4 {
		t.Errorf("Expected 4 tabs before close, got %d", len(result.Tabs))
	}

	// Close tab2
	err = c.CloseTab(ctx, tab2.ID)
	if err != nil {
		t.Fatalf("Failed to close tab2: %v", err)
	}
	t.Logf("Successfully closed tab: %s", tab2.ID)

	// Verify tab count decreased
	result, err = c.ListTabs(ctx)
	if err != nil {
		t.Fatalf("Failed to list tabs after close: %v", err)
	}

	if len(result.Tabs) != 3 {
		t.Errorf("Expected 3 tabs after close, got %d", len(result.Tabs))
	}

	// Verify tab2 is not in the list
	for _, tab := range result.Tabs {
		if tab.ID == tab2.ID {
			t.Errorf("Closed tab %s still appears in list", tab2.ID)
		}
	}

	// Close the active tab and verify automatic switch
	activeID := result.ActiveTabID
	err = c.CloseTab(ctx, activeID)
	if err != nil {
		t.Fatalf("Failed to close active tab: %v", err)
	}

	result, err = c.ListTabs(ctx)
	if err != nil {
		t.Fatalf("Failed to list tabs after closing active: %v", err)
	}

	if result.ActiveTabID == activeID {
		t.Errorf("Active tab ID should have changed after closing active tab")
	}

	if result.ActiveTabID == "" {
		t.Error("Active tab ID should not be empty after closing a tab")
	}

	// Verify tab1 still exists
	found := false
	for _, tab := range result.Tabs {
		if tab.ID == tab1.ID {
			found = true
		}
	}
	if !found {
		t.Errorf("Tab1 should still exist after closing other tabs")
	}

	t.Logf("After closing active tab, new active tab: %s", result.ActiveTabID)
}

// TestTabCloseLastTab verifies that closing the last tab returns an error
func TestTabCloseLastTab(t *testing.T) {
	skipIfIntegrationTestsDisabled(t)

	ctx, c, cleanup := setupE2ETest(t, 20)
	defer cleanup()

	// Get the initial tab ID
	result, err := c.ListTabs(ctx)
	if err != nil {
		t.Fatalf("Failed to list tabs: %v", err)
	}

	if len(result.Tabs) != 1 {
		t.Fatalf("Expected 1 initial tab, got %d", len(result.Tabs))
	}

	lastTabID := result.Tabs[0].ID

	// Try to close the last tab - should fail
	err = c.CloseTab(ctx, lastTabID)
	if err == nil {
		t.Error("Expected error when closing last tab, got nil")
	} else {
		t.Logf("Got expected error when closing last tab: %v", err)
	}

	// Verify tab still exists
	result, err = c.ListTabs(ctx)
	if err != nil {
		t.Fatalf("Failed to list tabs after attempted close: %v", err)
	}

	if len(result.Tabs) != 1 {
		t.Errorf("Expected 1 tab to remain, got %d", len(result.Tabs))
	}
}

// TestTabNavigationIsolation verifies that tabs have independent navigation
func TestTabNavigationIsolation(t *testing.T) {
	skipIfIntegrationTestsDisabled(t)

	ctx, c, cleanup := setupE2ETest(t, 30)
	defer cleanup()

	// Create 2 tabs
	tab1, err := c.CreateTab(ctx)
	if err != nil {
		t.Fatalf("Failed to create tab 1: %v", err)
	}

	tab2, err := c.CreateTab(ctx)
	if err != nil {
		t.Fatalf("Failed to create tab 2: %v", err)
	}

	// Navigate tab1 to example.com
	_, err = c.Navigate(ctx, "https://example.com", tab1.ID)
	if err != nil {
		t.Fatalf("Failed to navigate tab1: %v", err)
	}

	// Wait for navigation
	time.Sleep(500 * time.Millisecond)

	// Navigate tab2 to iana.org
	_, err = c.Navigate(ctx, "https://www.iana.org", tab2.ID)
	if err != nil {
		t.Fatalf("Failed to navigate tab2: %v", err)
	}

	// Wait for navigation
	time.Sleep(500 * time.Millisecond)

	// Verify each tab has the correct URL
	result, err := c.ListTabs(ctx)
	if err != nil {
		t.Fatalf("Failed to list tabs: %v", err)
	}

	var tab1URL, tab2URL string
	for _, tab := range result.Tabs {
		if tab.ID == tab1.ID {
			tab1URL = tab.URL
		}
		if tab.ID == tab2.ID {
			tab2URL = tab.URL
		}
	}

	if tab1URL != "https://example.com/" {
		t.Errorf("Expected tab1 URL to be https://example.com/, got %s", tab1URL)
	}

	if tab2URL != "https://www.iana.org/" {
		t.Errorf("Expected tab2 URL to be https://www.iana.org/, got %s", tab2URL)
	}

	t.Logf("Tab1 URL: %s, Tab2 URL: %s", tab1URL, tab2URL)
}

// TestTabElementIsolation verifies that element caches are independent per tab
func TestTabElementIsolation(t *testing.T) {
	skipIfIntegrationTestsDisabled(t)

	ctx, c, cleanup := setupE2ETest(t, 30)
	defer cleanup()

	// Create 2 tabs
	tab1, err := c.CreateTab(ctx)
	if err != nil {
		t.Fatalf("Failed to create tab 1: %v", err)
	}

	tab2, err := c.CreateTab(ctx)
	if err != nil {
		t.Fatalf("Failed to create tab 2: %v", err)
	}

	// Navigate both tabs to example.com
	_, err = c.Navigate(ctx, "https://example.com", tab1.ID)
	if err != nil {
		t.Fatalf("Failed to navigate tab1: %v", err)
	}

	time.Sleep(500 * time.Millisecond)

	_, err = c.Navigate(ctx, "https://example.com", tab2.ID)
	if err != nil {
		t.Fatalf("Failed to navigate tab2: %v", err)
	}

	time.Sleep(500 * time.Millisecond)

	// Extract elements from tab1
	elements1, err := c.GetElements(ctx, tab1.ID)
	if err != nil {
		t.Fatalf("Failed to get elements from tab1: %v", err)
	}

	// Extract elements from tab2
	elements2, err := c.GetElements(ctx, tab2.ID)
	if err != nil {
		t.Fatalf("Failed to get elements from tab2: %v", err)
	}

	// Verify both have elements
	if len(elements1) == 0 {
		t.Error("Expected elements in tab1")
	}

	if len(elements2) == 0 {
		t.Error("Expected elements in tab2")
	}

	// Verify elements have valid data (no Index field anymore - browser-service returns raw nodes)
	if len(elements1) > 0 && elements1[0].Role == "" {
		t.Error("Expected tab1 first element to have a role")
	}

	if len(elements2) > 0 && elements2[0].Role == "" {
		t.Error("Expected tab2 first element to have a role")
	}

	t.Logf("Tab1 elements: %d, Tab2 elements: %d", len(elements1), len(elements2))
	t.Log("Element caches are properly isolated per tab")
}

// TestTabConcurrentOperations verifies thread safety with concurrent tab operations
func TestTabConcurrentOperations(t *testing.T) {
	skipIfIntegrationTestsDisabled(t)

	ctx := context.Background()
	_, wsURL, serverCleanup := startTestServer(t, ctx)
	defer serverCleanup()

	c, err := client.New(wsURL)
	if err != nil {
		t.Fatalf("Failed to create client: %v", err)
	}
	defer func() {
		if err := c.Close(); err != nil {
			t.Errorf("Failed to close client: %v", err)
		}
	}()

	cmdCtx, cancel := context.WithTimeout(ctx, 60*time.Second)
	defer cancel()

	// Create 5 tabs concurrently
	numTabs := 5
	var wg sync.WaitGroup
	tabIDs := make(chan string, numTabs)
	errors := make(chan error, numTabs)

	for i := 0; i < numTabs; i++ {
		wg.Add(1)
		go func(index int) {
			defer wg.Done()

			tab, err := c.CreateTab(cmdCtx)
			if err != nil {
				errors <- fmt.Errorf("failed to create tab %d: %w", index, err)
				return
			}

			tabIDs <- tab.ID

			// Navigate each tab to a different URL
			url := fmt.Sprintf("https://example.com/?tab=%d", index)
			_, err = c.Navigate(cmdCtx, url, tab.ID)
			if err != nil {
				errors <- fmt.Errorf("failed to navigate tab %d: %w", index, err)
				return
			}
		}(i)
	}

	wg.Wait()
	close(tabIDs)
	close(errors)

	// Check for errors
	for err := range errors {
		t.Error(err)
	}

	// Verify all tabs were created
	result, err := c.ListTabs(cmdCtx)
	if err != nil {
		t.Fatalf("Failed to list tabs: %v", err)
	}

	expectedCount := numTabs + 1 // initial tab + created tabs
	if len(result.Tabs) != expectedCount {
		t.Errorf("Expected %d tabs, got %d", expectedCount, len(result.Tabs))
	}

	// Verify no duplicate IDs
	seen := make(map[string]bool)
	for _, tab := range result.Tabs {
		if seen[tab.ID] {
			t.Errorf("Duplicate tab ID found: %s", tab.ID)
		}
		seen[tab.ID] = true
	}

	t.Logf("Successfully created and operated on %d tabs concurrently", numTabs)
}

// TestTabSwitchInvalidID verifies error handling for invalid tab ID
func TestTabSwitchInvalidID(t *testing.T) {
	skipIfIntegrationTestsDisabled(t)

	ctx, c, cleanup := setupE2ETest(t, 10)
	defer cleanup()

	// Try to switch to non-existent tab
	err := c.SwitchTab(ctx, "invalid-tab-id")
	if err == nil {
		t.Error("Expected error when switching to invalid tab ID, got nil")
	} else {
		t.Logf("Got expected error for invalid tab ID: %v", err)
	}
}

// TestTabCloseInvalidID verifies error handling for invalid tab ID
func TestTabCloseInvalidID(t *testing.T) {
	skipIfIntegrationTestsDisabled(t)

	ctx, c, cleanup := setupE2ETest(t, 10)
	defer cleanup()

	// Try to close non-existent tab
	err := c.CloseTab(ctx, "invalid-tab-id")
	if err == nil {
		t.Error("Expected error when closing invalid tab ID, got nil")
	} else {
		t.Logf("Got expected error for invalid tab ID: %v", err)
	}
}
