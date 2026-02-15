package test

import (
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/sarathmenon/browser-service/pkg/protocol"
	"github.com/sarathmenon/browser-service/test/testserver"
)

// TestCredentialInjectionWithAuthentication tests the full credential injection flow
// with a simulated authenticated session
func TestCredentialInjectionWithAuthentication(t *testing.T) {
	skipIfIntegrationTestsDisabled(t)

	server := testserver.StartTestServer(t)
	defer server.Close()

	// ===== PHASE 1: Simulate creating credentials (like a real Convex task would have) =====
	t.Log("\n=== PHASE 1: Creating authenticated session ===")

	ctx, client, cleanup := setupE2ETest(t, 60)
	defer cleanup()

	// Navigate to login page
	_, err := client.Navigate(ctx, server.URL+"/login-form")
	if err != nil {
		t.Fatalf("Failed to navigate to login: %v", err)
	}

	time.Sleep(500 * time.Millisecond)

	// Perform login by typing and clicking
	elements, err := client.GetElements(ctx)
	if err != nil {
		t.Fatalf("Failed to get elements: %v", err)
	}

	// Find username, password, and submit button
	var usernameIdx, passwordIdx, submitIdx int = -1, -1, -1
	for i, elem := range elements {
		if elem.Role == "textbox" && usernameIdx == -1 {
			usernameIdx = i
		} else if elem.Role == "textbox" && passwordIdx == -1 {
			passwordIdx = i
		} else if elem.Role == "button" {
			submitIdx = i
		}
	}

	if usernameIdx == -1 || passwordIdx == -1 || submitIdx == -1 {
		t.Fatalf("Failed to find login form elements")
	}

	// Type credentials
	err = client.Type(ctx, &usernameIdx, "testuser")
	if err != nil {
		t.Fatalf("Failed to type username: %v", err)
	}

	err = client.Type(ctx, &passwordIdx, "password123")
	if err != nil {
		t.Fatalf("Failed to type password: %v", err)
	}

	// Click submit
	err = client.Click(ctx, submitIdx)
	if err != nil {
		t.Fatalf("Failed to click submit: %v", err)
	}

	// Wait for redirect
	time.Sleep(2 * time.Second)

	// Verify we're logged in by checking the page
	text, err := client.GetText(ctx, "body")
	if err != nil {
		t.Fatalf("Failed to get text: %v", err)
	}

	if !strings.Contains(text.Text, "Welcome back, testuser") {
		t.Fatalf("Expected to see welcome message, got: %s", text.Text)
	}

	t.Logf("✓ Successfully logged in, page shows: %s", text.Text)

	// Save the storage state (this is what would be in Convex's loginCookie field)
	savedState, err := client.SaveStorageState(ctx)
	if err != nil {
		t.Fatalf("Failed to save storage state: %v", err)
	}

	t.Logf("✓ Saved storage state with %d cookies", len(savedState.State.Cookies))

	// Verify we have session cookie
	foundSessionCookie := false
	var sessionCookieValue string
	for _, cookie := range savedState.State.Cookies {
		if cookie.Name == "session" {
			foundSessionCookie = true
			sessionCookieValue = cookie.Value
			t.Logf("✓ Found session cookie: %s=%s", cookie.Name, cookie.Value)
		}
	}

	if !foundSessionCookie {
		t.Fatalf("Expected session cookie after login")
	}

	// Convert storage state to JSON (this is what Convex would store)
	storageStateJSON, err := json.Marshal(savedState.State)
	if err != nil {
		t.Fatalf("Failed to marshal storage state: %v", err)
	}

	t.Logf("✓ Storage state JSON size: %d bytes", len(storageStateJSON))

	// ===== PHASE 2: Simulate fresh session loading credentials (like LoadTaskCredentials would do) =====
	t.Log("\n=== PHASE 2: Fresh session with credential injection ===")

	// Close current session
	cleanup()

	// Start fresh session (simulating new browser context)
	ctx2, client2, cleanup2 := setupE2ETest(t, 60)
	defer cleanup2()

	// Simulate what LoadTaskCredentials does - load the storage state
	t.Log("Loading credentials from saved state...")
	_, err = client2.LoadStorageState(ctx2, savedState.State)
	if err != nil {
		t.Fatalf("Failed to load storage state: %v", err)
	}

	// Verify cookies were injected
	cookies, err := client2.GetCookies(ctx2)
	if err != nil {
		t.Fatalf("Failed to get cookies: %v", err)
	}

	t.Logf("✓ Loaded %d cookies into fresh session", len(cookies.Cookies))

	// Verify session cookie is present
	foundInjectedCookie := false
	for _, cookie := range cookies.Cookies {
		if cookie.Name == "session" && cookie.Value == sessionCookieValue {
			foundInjectedCookie = true
			t.Logf("✓ Session cookie successfully injected: %s=%s", cookie.Name, cookie.Value)
		}
	}

	if !foundInjectedCookie {
		t.Fatalf("Expected session cookie to be injected")
	}

	// ===== PHASE 3: Verify authentication works without re-login =====
	t.Log("\n=== PHASE 3: Verifying authentication ===")

	// Navigate directly to dashboard (should work without login)
	_, err = client2.Navigate(ctx2, server.URL+"/dashboard")
	if err != nil {
		t.Fatalf("Failed to navigate to dashboard: %v", err)
	}

	time.Sleep(500 * time.Millisecond)

	// Check if we're authenticated
	dashboardText, err := client2.GetText(ctx2, "body")
	if err != nil {
		t.Fatalf("Failed to get dashboard text: %v", err)
	}

	if !strings.Contains(dashboardText.Text, "Welcome back, testuser") {
		t.Fatalf("Expected authenticated dashboard, got: %s", dashboardText.Text)
	}

	t.Logf("✓ Successfully authenticated without re-login!")
	t.Logf("✓ Dashboard shows: %s", dashboardText.Text)

	// ===== PHASE 4: Verify credentials persist across navigation =====
	t.Log("\n=== PHASE 4: Verifying credential persistence ===")

	// Navigate to another page
	_, err = client2.Navigate(ctx2, server.URL+"/storage-test")
	if err != nil {
		t.Fatalf("Failed to navigate: %v", err)
	}

	// Navigate back to dashboard
	_, err = client2.Navigate(ctx2, server.URL+"/dashboard")
	if err != nil {
		t.Fatalf("Failed to navigate back: %v", err)
	}

	time.Sleep(500 * time.Millisecond)

	// Verify still authenticated
	finalText, err := client2.GetText(ctx2, "body")
	if err != nil {
		t.Fatalf("Failed to get final text: %v", err)
	}

	if !strings.Contains(finalText.Text, "Welcome back, testuser") {
		t.Fatalf("Lost authentication after navigation, got: %s", finalText.Text)
	}

	t.Logf("✓ Credentials persisted across navigation")

	t.Log("\n=== ✅ ALL PHASES PASSED ===")
	t.Log("This proves credential injection works exactly like Convex tasks would:")
	t.Log("1. Saved authenticated state to JSON (simulating loginCookie)")
	t.Log("2. Loaded credentials into fresh browser session")
	t.Log("3. Navigated to protected pages without re-authentication")
	t.Log("4. Credentials persisted across multiple navigations")
}

// TestCredentialInjectionFormat tests that the storage state format matches Convex expectations
func TestCredentialInjectionFormat(t *testing.T) {
	skipIfIntegrationTestsDisabled(t)

	server := testserver.StartTestServer(t)
	defer server.Close()

	ctx, client, cleanup := setupE2ETest(t, 30)
	defer cleanup()

	// Set test cookies that would come from Convex
	testCookies := []protocol.Cookie{
		{
			Name:     "auth_token",
			Value:    "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9",
			Domain:   "127.0.0.1",
			Path:     "/",
			Secure:   false,
			HTTPOnly: true,
			SameSite: "Lax",
		},
		{
			Name:     "user_id",
			Value:    "user_12345",
			Domain:   "127.0.0.1",
			Path:     "/",
			Secure:   false,
			HTTPOnly: false,
			SameSite: "Lax",
		},
	}

	_, err := client.SetCookies(ctx, testCookies)
	if err != nil {
		t.Fatalf("Failed to set cookies: %v", err)
	}

	// Navigate and set localStorage
	_, err = client.Navigate(ctx, server.URL+"/storage-test")
	if err != nil {
		t.Fatalf("Failed to navigate: %v", err)
	}

	localStorage := map[string]string{
		"session_id":   "sess_abc123",
		"user_email":   "test@example.com",
		"last_login":   time.Now().Format(time.RFC3339),
		"preferences":  `{"theme":"dark","language":"en"}`,
	}

	_, err = client.SetLocalStorage(ctx, localStorage)
	if err != nil {
		t.Fatalf("Failed to set localStorage: %v", err)
	}

	// Save storage state
	state, err := client.SaveStorageState(ctx)
	if err != nil {
		t.Fatalf("Failed to save storage state: %v", err)
	}

	// Convert to JSON (Convex loginCookie format)
	loginCookieJSON, err := json.Marshal(state.State)
	if err != nil {
		t.Fatalf("Failed to marshal: %v", err)
	}

	t.Logf("✓ Generated loginCookie JSON (%d bytes)", len(loginCookieJSON))

	// Verify JSON structure
	var parsedState protocol.StorageState
	err = json.Unmarshal(loginCookieJSON, &parsedState)
	if err != nil {
		t.Fatalf("Failed to parse loginCookie JSON: %v", err)
	}

	if len(parsedState.Cookies) != 2 {
		t.Errorf("Expected 2 cookies, got %d", len(parsedState.Cookies))
	}

	if len(parsedState.Origins) != 1 {
		t.Errorf("Expected 1 origin, got %d", len(parsedState.Origins))
	}

	if len(parsedState.Origins[0].LocalStorage) != 4 {
		t.Errorf("Expected 4 localStorage items, got %d", len(parsedState.Origins[0].LocalStorage))
	}

	t.Logf("✓ Storage state format is valid:")
	t.Logf("  - Cookies: %d", len(parsedState.Cookies))
	t.Logf("  - Origins: %d", len(parsedState.Origins))
	t.Logf("  - localStorage items: %d", len(parsedState.Origins[0].LocalStorage))

	// Test loading it back
	_, err = client.ClearCookies(ctx)
	if err != nil {
		t.Fatalf("Failed to clear cookies: %v", err)
	}

	_, err = client.LoadStorageState(ctx, parsedState)
	if err != nil {
		t.Fatalf("Failed to load storage state: %v", err)
	}

	// Verify cookies restored
	cookies, err := client.GetCookies(ctx)
	if err != nil {
		t.Fatalf("Failed to get cookies: %v", err)
	}

	if len(cookies.Cookies) < 2 {
		t.Errorf("Expected at least 2 cookies after load, got %d", len(cookies.Cookies))
	}

	t.Logf("✓ Storage state successfully loaded from loginCookie format")
}
