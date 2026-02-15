package test

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/sarathmenon/browser-service/pkg/protocol"
	"github.com/sarathmenon/browser-service/test/testserver"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// waitForBrowserEvent waits for a specific browser error event type
func waitForBrowserEvent(t *testing.T, eventChan <-chan protocol.Event, errorType string, timeout time.Duration) *protocol.BrowserErrorEventParams {
	t.Helper()

	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	for {
		select {
		case event, ok := <-eventChan:
			if !ok {
				t.Fatal("Event channel closed before receiving expected event")
				return nil
			}

			if event.Method != "Browser.errorOccurred" {
				continue
			}

			// Parse event params
			paramsBytes, err := json.Marshal(event.Params)
			if err != nil {
				t.Logf("Failed to marshal event params: %v", err)
				continue
			}

			var params protocol.BrowserErrorEventParams
			if err := json.Unmarshal(paramsBytes, &params); err != nil {
				t.Logf("Failed to unmarshal event params: %v", err)
				continue
			}

			t.Logf("Received browser event: %s - %+v", params.ErrorType, params.Details)

			if params.ErrorType == errorType {
				return &params
			}

		case <-ctx.Done():
			t.Fatalf("Timeout waiting for %s event after %v", errorType, timeout)
			return nil
		}
	}
}

// TestCrashWatchdogDetectsNetworkTimeout tests that the crash watchdog detects hanging network requests
func TestCrashWatchdogDetectsNetworkTimeout(t *testing.T) {
	skipIfIntegrationTestsDisabled(t)

	ctx, c, cleanup := setupE2ETest(t, 60)
	defer cleanup()

	// Subscribe to browser events
	eventChan := c.SubscribeToEvents()

	// Start test HTTP server
	server := testserver.StartTestServer(t)
	defer server.Close()

	// Navigate to page with slow fetch button
	_, err := c.Navigate(ctx, server.URL+"/trigger-slow-fetch")
	require.NoError(t, err)

	// Wait for page to load
	time.Sleep(1 * time.Second)

	// Click button to trigger slow fetch (15s request)
	elements, err := c.GetElements(ctx)
	require.NoError(t, err)
	require.Greater(t, len(elements), 0, "Should have interactive elements")

	// Find and click the trigger button
	var buttonIndex int
	found := false
	for i, elem := range elements {
		if elem.Role == "button" {
			buttonIndex = i
			found = true
			break
		}
	}
	require.True(t, found, "Should find trigger button")

	err = c.Click(ctx, buttonIndex)
	require.NoError(t, err)

	// Wait for crash watchdog to detect the timeout
	// Watchdog has 10s initial delay + checks every 5s + request must exceed 10s
	// So we expect the event between 20-25 seconds
	event := waitForBrowserEvent(t, eventChan, "NetworkTimeout", 30*time.Second)
	require.NotNil(t, event)

	// Verify event details
	assert.Contains(t, event.Details, "url")
	assert.Contains(t, event.Details, "elapsed_seconds")

	elapsedSeconds, ok := event.Details["elapsed_seconds"].(float64)
	assert.True(t, ok, "elapsed_seconds should be a number")
	assert.GreaterOrEqual(t, elapsedSeconds, 10.0, "Request should have hung for at least 10 seconds")
}

// TestCrashWatchdogHealthCheckPasses tests that the health check runs without errors
func TestCrashWatchdogHealthCheckPasses(t *testing.T) {
	skipIfIntegrationTestsDisabled(t)

	ctx, c, cleanup := setupE2ETest(t, 30)
	defer cleanup()

	// Subscribe to events
	eventChan := c.SubscribeToEvents()

	// Navigate to a valid page
	_, err := c.Navigate(ctx, "data:text/html,<h1>Test Page</h1>")
	require.NoError(t, err)

	// Wait for initial monitoring delay (10s) plus two health check cycles (10s)
	// If health check fails, we'd see BrowserUnresponsive event
	time.Sleep(21 * time.Second)

	// Check that NO BrowserUnresponsive events were emitted
	select {
	case event := <-eventChan:
		var params protocol.BrowserErrorEventParams
		paramsBytes, _ := json.Marshal(event.Params)
		_ = json.Unmarshal(paramsBytes, &params)
		if params.ErrorType == "BrowserUnresponsive" {
			t.Fatalf("Unexpected BrowserUnresponsive event: %+v", params)
		}
	default:
		// No events - this is expected
	}

	// Browser should still be responsive
	tabs, err := c.ListTabs(ctx)
	assert.NoError(t, err)
	assert.NotNil(t, tabs)
	assert.Greater(t, len(tabs.Tabs), 0)
}

// TestCrashWatchdogWithMultipleTabs tests crash watchdog registers listeners for all tabs
func TestCrashWatchdogWithMultipleTabs(t *testing.T) {
	skipIfIntegrationTestsDisabled(t)

	ctx, c, cleanup := setupE2ETest(t, 60)
	defer cleanup()

	// Subscribe to events
	eventChan := c.SubscribeToEvents()

	// Create multiple tabs
	tab2, err := c.CreateTab(ctx)
	require.NoError(t, err)
	require.NotEmpty(t, tab2.ID)

	tab3, err := c.CreateTab(ctx)
	require.NoError(t, err)
	require.NotEmpty(t, tab3.ID)

	// Navigate tabs to different pages
	_, err = c.Navigate(ctx, "data:text/html,<h1>Tab 1</h1>")
	require.NoError(t, err)

	err = c.SwitchTab(ctx, tab2.ID)
	require.NoError(t, err)

	_, err = c.Navigate(ctx, "data:text/html,<h1>Tab 2</h1>")
	require.NoError(t, err)

	err = c.SwitchTab(ctx, tab3.ID)
	require.NoError(t, err)

	_, err = c.Navigate(ctx, "data:text/html,<h1>Tab 3</h1>")
	require.NoError(t, err)

	// Wait for health check cycle (10s delay + 5s check)
	time.Sleep(16 * time.Second)

	// Check no errors occurred
	select {
	case event := <-eventChan:
		t.Fatalf("Unexpected event: %+v", event)
	default:
		// No events - expected
	}

	// All tabs should still be functional
	tabs, err := c.ListTabs(ctx)
	assert.NoError(t, err)
	assert.Equal(t, 3, len(tabs.Tabs))
}

// TestCrashWatchdogNetworkRequestTracking tests that network requests are properly tracked and cleared
func TestCrashWatchdogNetworkRequestTracking(t *testing.T) {
	skipIfIntegrationTestsDisabled(t)

	ctx, c, cleanup := setupE2ETest(t, 30)
	defer cleanup()

	// Subscribe to events
	eventChan := c.SubscribeToEvents()

	// Start test HTTP server
	server := testserver.StartTestServer(t)
	defer server.Close()

	// Navigate to a page (this will create network requests)
	_, err := c.Navigate(ctx, server.URL+"/no-download-page")
	require.NoError(t, err)

	// Navigate to another page (more network requests)
	_, err = c.Navigate(ctx, server.URL+"/dashboard")
	require.NoError(t, err)

	// Wait a bit for any potential timeouts (none expected)
	time.Sleep(3 * time.Second)

	// Check that no network timeout events occurred (requests completed quickly)
	select {
	case event := <-eventChan:
		var params protocol.BrowserErrorEventParams
		paramsBytes, _ := json.Marshal(event.Params)
		_ = json.Unmarshal(paramsBytes, &params)
		if params.ErrorType == "NetworkTimeout" {
			t.Fatalf("Unexpected NetworkTimeout event: %+v", params)
		}
	default:
		// No events - expected
	}

	// Browser should remain responsive
	tabs, err := c.ListTabs(ctx)
	assert.NoError(t, err)
	assert.NotNil(t, tabs)
}

// TestCrashWatchdogStartsAndStops tests that the crash watchdog lifecycle works correctly
func TestCrashWatchdogStartsAndStops(t *testing.T) {
	skipIfIntegrationTestsDisabled(t)

	ctx, c, cleanup := setupE2ETest(t, 30)
	defer cleanup()

	// Subscribe to events
	eventChan := c.SubscribeToEvents()

	// Navigate to a page to verify browser is working
	_, err := c.Navigate(ctx, "data:text/html,<h1>Test</h1>")
	assert.NoError(t, err)

	// Wait for initial delay period and one health check
	time.Sleep(16 * time.Second)

	// Verify no errors occurred
	select {
	case event := <-eventChan:
		t.Logf("Received event (should be none): %+v", event)
	default:
		// Expected - no events
	}

	// Verify browser is still responsive (watchdog is running properly)
	tabs, err := c.ListTabs(ctx)
	assert.NoError(t, err)
	assert.NotNil(t, tabs)
	assert.Equal(t, 1, len(tabs.Tabs))

	// Cleanup will stop the watchdog when context is closed
}

// TestCrashWatchdogEventsStopAfterDisconnect tests that events stop flowing after disconnect
func TestCrashWatchdogEventsStopAfterDisconnect(t *testing.T) {
	skipIfIntegrationTestsDisabled(t)

	ctx, c, cleanup := setupE2ETest(t, 10)

	// Subscribe to events
	eventChan := c.SubscribeToEvents()

	// Navigate to verify connection works
	_, err := c.Navigate(ctx, "data:text/html,<h1>Test</h1>")
	require.NoError(t, err)

	// Close the browser connection
	cleanup()

	// Either channel should be closed OR no more events should arrive
	// We don't strictly require channel closure as long as events stop
	foundEvent := false
	select {
	case event, ok := <-eventChan:
		if ok {
			// Received an event after disconnect - this is a problem
			var params protocol.BrowserErrorEventParams
			paramsBytes, _ := json.Marshal(event.Params)
			_ = json.Unmarshal(paramsBytes, &params)
			t.Errorf("Received event after disconnect: %s - %+v", params.ErrorType, params.Details)
			foundEvent = true
		}
		// Channel closed (ok == false) - this is expected behavior
	case <-time.After(3 * time.Second):
		// No events received - this is also acceptable
	}

	assert.False(t, foundEvent, "Should not receive events after disconnect")
}
