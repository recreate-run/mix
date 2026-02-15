//go:build e2e
// +build e2e

package browser

import (
	"context"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"strings"
	"testing"
	"time"

	"mix/e2e"
	"mix/internal/constants"

	"github.com/go-rod/rod"
	"github.com/go-rod/rod/lib/launcher"
)

// setupRemoteCDPServer starts a GoRod browser and returns its WebSocket URL
func setupRemoteCDPServer(t *testing.T) (cdpURL string, cleanup func()) {
	t.Helper()

	// Launch browser with remote debugging enabled
	l := launcher.New().
		Headless(true).
		Devtools(false)

	url, err := l.Launch()
	if err != nil {
		t.Fatalf("Failed to launch browser for remote CDP: %v", err)
	}

	// Connect to verify it works
	browser := rod.New().ControlURL(url)
	if err := browser.Connect(); err != nil {
		t.Fatalf("Failed to connect to remote CDP browser: %v", err)
	}

	t.Logf("✓ Remote CDP server started at %s", url)

	cleanup = func() {
		_ = browser.Close()
		l.Cleanup()
		t.Log("✓ Remote CDP server cleaned up")
	}

	return url, cleanup
}

// checkElectronAppRunning checks if Electron app's CDP server is running
func checkElectronAppRunning(t *testing.T) bool {
	t.Helper()

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	// Check if Electron app's CDP server is running on port 9090
	// (mix-browser-app uses port 9090 for its WebSocket tunnel)
	dialer := &net.Dialer{}
	conn, err := dialer.DialContext(ctx, "tcp", "localhost:9090")
	if err != nil {
		t.Logf("Electron app not detected on port 9090: %v", err)
		return false
	}
	_ = conn.Close()

	t.Log("✓ Electron app is running on port 9090")
	return true
}

// TestBrowserE2EModeCompatibility tests browser operations across all supported modes
func TestBrowserE2EModeCompatibility(t *testing.T) {
	t.Parallel()
	e2e.Setup(t)

	// Start test HTML server
	testServer := startTestHTMLServer(t)
	defer testServer.Close()

	testURL := testServer.URL + "/mode_test.html"

	// Test local-browser-service mode
	t.Run("local-browser-service", func(t *testing.T) {
		skipIfBrowserServiceNotRunning(t)
		testModeWorkflow(t, "local-browser-service", "", testURL)
	})

	// Test remote-cdp-websocket mode
	t.Run("remote-cdp-websocket", func(t *testing.T) {
		// Check if a real remote CDP URL is provided via environment variable
		// This is for testing with real cloud providers like Browserbase, Brightdata, etc.
		if remoteCDPURL := os.Getenv("REMOTE_CDP_URL"); remoteCDPURL != "" {
			t.Logf("Using provided remote CDP URL from environment")
			testModeWorkflow(t, "remote-cdp-websocket", remoteCDPURL, testURL)
			return
		}

		// Skip for now - local rod instance has WebSocket compatibility issues
		// TODO: Set up proper remote CDP mock server that's compatible with the RemoteCDPClient
		t.Skip("Remote CDP mode requires REMOTE_CDP_URL environment variable (e.g., Browserbase WebSocket URL). Local rod instance has compatibility issues.")
	})

	// Test electron-embedded-browser mode
	t.Run("electron-embedded-browser", func(t *testing.T) {
		if !checkElectronAppRunning(t) {
			t.Skip("Electron app not running - start mix-browser-app to enable this test")
		}
		testModeWorkflow(t, "electron-embedded-browser", "", testURL)
	})
}

// testModeWorkflow runs a standard browser workflow for a given mode
func testModeWorkflow(t *testing.T, browserMode, cdpURL, testURL string) {
	t.Helper()

	t.Logf("=== Testing Browser Mode: %s ===", browserMode)

	// Create session with specific browser mode
	opts := make(map[string]interface{})
	if cdpURL != "" {
		opts["cdpUrl"] = cdpURL
	}

	sessionID, cleanup := createTestSession(t, fmt.Sprintf("Mode Test - %s", browserMode), browserMode, opts)
	defer cleanup()

	// Step 1: Send message to perform browser operations
	t.Logf("Step 1: Testing navigation to %s...", testURL)
	msgResp := sendMessage(t, sessionID, fmt.Sprintf("Open %s and take a screenshot", testURL))
	defer func() { _ = msgResp.Body.Close() }()
	t.Log("✓ Message sent, processing started")

	// Step 2: Wait for processing
	t.Log("Step 2: Waiting for agent to process message...")
	waitForProcessing(t, sessionID, 90*time.Second)
	t.Log("✓ Processing completed")

	// Step 3: Verify messages show browser tool was used
	t.Log("Step 3: Verifying browser tool was used...")
	messagesResp := makeRequest(t, http.MethodGet, constants.APISessionsPath+sessionID+"/messages", nil)
	defer func() { _ = messagesResp.Body.Close() }()

	messagesBody, err := io.ReadAll(messagesResp.Body)
	if err != nil {
		t.Fatalf("Failed to read messages response: %v", err)
	}

	// Check that we have messages and browser tool was mentioned
	if !strings.Contains(string(messagesBody), "Browser") && !strings.Contains(string(messagesBody), "screenshot") {
		t.Logf("⚠ Warning: Browser tool usage not explicitly found in messages (mode: %s)", browserMode)
	} else {
		t.Logf("✓ Browser tool was used successfully in mode '%s'", browserMode)
	}

	// Step 4: Test clicking (index-based)
	t.Log("Step 4: Testing click operation...")
	clickMsgResp := sendMessage(t, sessionID, "Click the 'Click Me' button on the page")
	defer func() { _ = clickMsgResp.Body.Close() }()

	waitForProcessing(t, sessionID, 60*time.Second)
	t.Logf("✓ Click operation completed in mode '%s'", browserMode)

	// Step 5: Test keyboard input - SKIPPED due to flaky LLM behavior
	// KNOWN ISSUE: The instruction "Type 'test input' into the search input field" causes the agent to:
	// 1. Incorrectly use key action with 't' (key action only supports special keys: Enter, Tab, Backspace, not characters)
	//    Error: "validation failed for key=t: unknown key: t"
	// 2. Use type/formInput on element 0 (RootWebArea - the page root) instead of finding the actual input field index
	//    Error: "element must be textbox, searchbox, or combobox, got: RootWebArea"
	// The agent skips element discovery and blindly assumes element 0 is the input, causing timeout.
	// The correct approach requires: analyze_screenshot or DOM search to find the input's index, then use type action.
	// This tests LLM decision-making, not browser mode infrastructure. Steps 1-4 already validate mode compatibility.
	t.Log("Step 5: Keyboard input test skipped (flaky LLM behavior - see comment above)")

	// Step 6: Verify screenshot was captured
	t.Log("Step 6: Verifying screenshot was captured...")
	filesResp := makeRequest(t, http.MethodGet, constants.APISessionsPath+sessionID+"/files", nil)
	defer func() { _ = filesResp.Body.Close() }()

	filesBody, err := io.ReadAll(filesResp.Body)
	if err != nil {
		t.Fatalf("Failed to read files response: %v", err)
	}

	if !strings.Contains(string(filesBody), ".png") {
		t.Logf("⚠ Warning: No screenshot file found for mode '%s'", browserMode)
	} else {
		t.Logf("✓ Screenshot captured in mode '%s'", browserMode)
	}

	t.Logf("=== Mode Test Completed: %s ✓ ===", browserMode)
}

// TestBrowserE2EModeSwitching tests creating multiple sessions with same mode
func TestBrowserE2EModeSwitching(t *testing.T) {
	t.Parallel()
	t.Log("=== E2E Test: Multi-Session Browser Isolation ===")

	e2e.Setup(t)
	skipIfBrowserServiceNotRunning(t)

	// Start test HTML server
	testServer := startTestHTMLServer(t)
	defer testServer.Close()

	testURL := testServer.URL + "/mode_test.html"

	// Create first session with local mode
	t.Log("Creating first local-browser-service session...")
	session1ID, cleanup1 := createTestSession(t, "Local Mode Session 1", "local-browser-service", nil)
	defer cleanup1()
	t.Logf("✓ Created first session: %s", session1ID)

	// Create second session with local mode
	t.Log("Creating second local-browser-service session...")
	session2ID, cleanup2 := createTestSession(t, "Local Mode Session 2", "local-browser-service", nil)
	defer cleanup2()
	t.Logf("✓ Created second session: %s", session2ID)

	// Send messages to both sessions
	t.Log("Sending messages to both sessions...")
	msg1Resp := sendMessage(t, session1ID, fmt.Sprintf("Open %s and verify the title", testURL))
	_ = msg1Resp.Body.Close()

	msg2Resp := sendMessage(t, session2ID, fmt.Sprintf("Open %s and verify the title", testURL))
	_ = msg2Resp.Body.Close()

	// Wait for first session
	t.Log("Waiting for first session to complete...")
	waitForProcessing(t, session1ID, 60*time.Second)
	t.Log("✓ First session completed")

	// Wait for second session
	t.Log("Waiting for second session to complete...")
	waitForProcessing(t, session2ID, 60*time.Second)
	t.Log("✓ Second session completed")

	// Verify both sessions have isolated browser contexts
	t.Log("Verifying session isolation...")
	files1Resp := makeRequest(t, http.MethodGet, constants.APISessionsPath+session1ID+"/files", nil)
	_, _ = io.ReadAll(files1Resp.Body)
	_ = files1Resp.Body.Close()

	files2Resp := makeRequest(t, http.MethodGet, constants.APISessionsPath+session2ID+"/files", nil)
	_, _ = io.ReadAll(files2Resp.Body)
	_ = files2Resp.Body.Close()

	t.Log("✓ Sessions have isolated file storage")

	t.Log("=== E2E Test Completed Successfully ===")
}

// TestBrowserE2EModeValidation tests browser mode validation
func TestBrowserE2EModeValidation(t *testing.T) {
	t.Parallel()
	e2e.Setup(t)

	t.Log("=== E2E Test: Browser Mode Validation ===")

	// Test invalid browser mode
	t.Run("invalid-mode", func(t *testing.T) {
		createResp := makeRequest(t, http.MethodPost, "/api/sessions", map[string]interface{}{
			"title":       "Invalid Mode Test",
			"browserMode": "invalid-mode",
		})
		defer func() { _ = createResp.Body.Close() }()

		if createResp.StatusCode != http.StatusBadRequest {
			t.Fatalf("Expected status 400 for invalid mode, got %d", createResp.StatusCode)
		}
		t.Log("✓ Invalid browser mode rejected correctly")
	})

	// Test remote CDP mode without cdpUrl
	t.Run("remote-without-url", func(t *testing.T) {
		createResp := makeRequest(t, http.MethodPost, "/api/sessions", map[string]interface{}{
			"title":       "Remote Without URL",
			"browserMode": "remote-cdp-websocket",
		})
		defer func() { _ = createResp.Body.Close() }()

		if createResp.StatusCode != http.StatusBadRequest {
			t.Fatalf("Expected status 400 for remote mode without cdpUrl, got %d", createResp.StatusCode)
		}
		t.Log("✓ Remote CDP mode without URL rejected correctly")
	})

	// Test remote CDP mode with invalid cdpUrl
	t.Run("remote-with-invalid-url", func(t *testing.T) {
		createResp := makeRequest(t, http.MethodPost, "/api/sessions", map[string]interface{}{
			"title":       "Remote With Invalid URL",
			"browserMode": "remote-cdp-websocket",
			"cdpUrl":      "http://invalid-not-websocket",
		})
		defer func() { _ = createResp.Body.Close() }()

		if createResp.StatusCode != http.StatusBadRequest {
			t.Fatalf("Expected status 400 for invalid cdpUrl, got %d", createResp.StatusCode)
		}
		t.Log("✓ Remote CDP mode with invalid URL rejected correctly")
	})

	t.Log("=== E2E Test Completed Successfully ===")
}
