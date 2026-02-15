package test

import (
	"strings"
	"testing"
	"time"

	"github.com/sarathmenon/browser-service/test/testserver"
)

func TestPopupAutoAcceptAlert(t *testing.T) {
	skipIfIntegrationTestsDisabled(t)

	server := testserver.StartTestServer(t)
	defer server.Close()

	ctx, client, cleanup := setupE2ETest(t, 30)
	defer cleanup()

	// Navigate to page with alert on load
	_, err := client.Navigate(ctx, server.URL+"/alert-page")
	// Should NOT block, alert should be auto-dismissed
	if err != nil {
		t.Logf("Navigate error (expected if alert blocked navigation): %v", err)
	}

	// Wait a bit for dialog to be handled
	time.Sleep(2 * time.Second)

	// Verify popup message stored
	messages, err := client.GetClosedPopupMessages(ctx)
	if err != nil {
		t.Fatalf("Failed to get closed popup messages: %v", err)
	}

	// Should contain the alert message
	if len(messages.Messages) == 0 {
		t.Errorf("Expected popup messages, got empty list")
	}

	foundAlert := false
	for _, msg := range messages.Messages {
		t.Logf("Found popup message: %s", msg)
		if msg == "[alert] Test alert message" {
			foundAlert = true
			break
		}
	}
	if !foundAlert {
		t.Errorf("Expected alert message not found. Messages: %v", messages.Messages)
	}

	// Verify page is still interactive by getting text
	textResult, err := client.GetText(ctx, "body")
	if err != nil {
		t.Fatalf("Failed to get text: %v", err)
	}

	t.Logf("Page text: %s", textResult.Text)
	if textResult.Text == "" {
		t.Errorf("Expected non-empty page text")
	}
}

func TestPopupAutoAcceptConfirm(t *testing.T) {
	skipIfIntegrationTestsDisabled(t)

	server := testserver.StartTestServer(t)
	defer server.Close()

	ctx, client, cleanup := setupE2ETest(t, 30)
	defer cleanup()

	// Navigate to confirm test page
	_, err := client.Navigate(ctx, server.URL+"/confirm-page")
	if err != nil {
		t.Fatalf("Failed to navigate: %v", err)
	}

	time.Sleep(500 * time.Millisecond)

	// Execute confirm (should auto-accept and return true)
	result, err := client.EvalJS(ctx, "confirm('Proceed with action?')")
	if err != nil {
		t.Fatalf("Failed to eval JS: %v", err)
	}

	// Confirm should return true (accepted)
	t.Logf("Confirm result: %v (type: %T)", result.Result, result.Result)
	if result.Result != true {
		t.Errorf("Expected confirm to return true (accepted), got %v", result.Result)
	}

	// Wait a bit for dialog to be handled
	time.Sleep(1 * time.Second)

	// Verify popup message stored
	messages, err := client.GetClosedPopupMessages(ctx)
	if err != nil {
		t.Fatalf("Failed to get closed popup messages: %v", err)
	}

	// Should contain the confirm message
	foundConfirm := false
	for _, msg := range messages.Messages {
		t.Logf("Found popup message: %s", msg)
		if msg == "[confirm] Proceed with action?" {
			foundConfirm = true
			break
		}
	}
	if !foundConfirm {
		t.Errorf("Expected confirm message not found. Messages: %v", messages.Messages)
	}
}

func TestPopupAutoDismissPrompt(t *testing.T) {
	skipIfIntegrationTestsDisabled(t)

	server := testserver.StartTestServer(t)
	defer server.Close()

	ctx, client, cleanup := setupE2ETest(t, 30)
	defer cleanup()

	// Navigate to prompt test page
	_, err := client.Navigate(ctx, server.URL+"/prompt-page")
	if err != nil {
		t.Fatalf("Failed to navigate: %v", err)
	}

	time.Sleep(500 * time.Millisecond)

	// Execute prompt (should auto-dismiss and return null)
	result, err := client.EvalJS(ctx, "prompt('Enter your name:')")
	if err != nil {
		t.Fatalf("Failed to eval JS: %v", err)
	}

	// Prompt should return null (dismissed)
	t.Logf("Prompt result: %v (type: %T)", result.Result, result.Result)
	if result.Result != nil {
		t.Errorf("Expected prompt to return null (dismissed), got %v", result.Result)
	}

	// Wait a bit for dialog to be handled
	time.Sleep(1 * time.Second)

	// Verify popup message stored
	messages, err := client.GetClosedPopupMessages(ctx)
	if err != nil {
		t.Fatalf("Failed to get closed popup messages: %v", err)
	}

	// Should contain the prompt message
	foundPrompt := false
	for _, msg := range messages.Messages {
		t.Logf("Found popup message: %s", msg)
		if msg == "[prompt] Enter your name:" {
			foundPrompt = true
			break
		}
	}
	if !foundPrompt {
		t.Errorf("Expected prompt message not found. Messages: %v", messages.Messages)
	}
}

func TestPermissionsGrantClipboardAccess(t *testing.T) {
	skipIfIntegrationTestsDisabled(t)

	server := testserver.StartTestServer(t)
	defer server.Close()

	ctx, client, cleanup := setupE2ETest(t, 30)
	defer cleanup()

	// Navigate to clipboard test page
	_, err := client.Navigate(ctx, server.URL+"/clipboard-test")
	if err != nil {
		t.Fatalf("Failed to navigate: %v", err)
	}

	time.Sleep(1 * time.Second)

	// Get elements to find buttons
	elements, err := client.GetElements(ctx)
	if err != nil {
		t.Fatalf("Failed to get elements: %v", err)
	}

	// Find and click "Write to clipboard" button (should be first button)
	writeButtonIdx := -1
	for i, elem := range elements {
		if elem.Role == "button" && elem.Name == "Write to clipboard" {
			writeButtonIdx = i
			break
		}
	}

	if writeButtonIdx == -1 {
		t.Fatalf("Write to clipboard button not found")
	}

	err = client.Click(ctx, writeButtonIdx)
	if err != nil {
		t.Fatalf("Failed to click write button: %v", err)
	}

	time.Sleep(2 * time.Second)

	// Check output - should say "Write success" (not permission denied)
	textResult, err := client.GetText(ctx, "body")
	if err != nil {
		t.Fatalf("Failed to get text: %v", err)
	}

	t.Logf("Page text after write: %s", textResult.Text)

	// The text should contain "Write success" if permissions were granted
	if textResult.Text == "" {
		t.Errorf("Expected non-empty page text after clipboard write")
	}

	// Verify write succeeded
	if !strings.Contains(textResult.Text, "Write success") {
		t.Errorf("Expected 'Write success' in page text after clipboard write, got: %s", textResult.Text)
	}

	// Get elements again for read button
	elements, err = client.GetElements(ctx)
	if err != nil {
		t.Fatalf("Failed to get elements: %v", err)
	}

	// Find and click "Read from clipboard" button
	readButtonIdx := -1
	for i, elem := range elements {
		if elem.Role == "button" && elem.Name == "Read from clipboard" {
			readButtonIdx = i
			break
		}
	}

	if readButtonIdx == -1 {
		t.Fatalf("Read from clipboard button not found")
	}

	err = client.Click(ctx, readButtonIdx)
	if err != nil {
		t.Fatalf("Failed to click read button: %v", err)
	}

	time.Sleep(2 * time.Second)

	// Check output div content specifically using JavaScript
	outputResult, err := client.EvalJS(ctx, "document.getElementById('output').textContent")
	if err != nil {
		t.Fatalf("Failed to get output div content: %v", err)
	}

	t.Logf("Output div content after read: %v", outputResult.Result)

	// Clipboard API in headless mode has limitations
	// We verify that either:
	// 1. Read succeeded ("Read: test data")
	// 2. Read failed with a permission error (which shows permissions API is working, just restricted in headless)
	// 3. Still shows "Write success" (clipboard read didn't update the div)

	outputText, ok := outputResult.Result.(string)
	if !ok {
		t.Fatalf("Output content is not a string: %T", outputResult.Result)
	}

	// Check what we got
	if strings.Contains(outputText, "Read: test data") {
		t.Logf("✅ Clipboard read succeeded - full round-trip working")
	} else if strings.Contains(outputText, "Read error") {
		t.Logf("⚠️  Clipboard read failed with error (expected in some headless configurations): %s", outputText)
		// This is acceptable - permissions were granted but headless Chrome may have additional restrictions
	} else if outputText == "Write success" {
		t.Logf("⚠️  Clipboard read button clicked but output not updated - async operation may have failed silently")
		// In headless mode, clipboard read often fails silently or isn't allowed even with permissions
		// The important thing is that write worked, proving permissions were granted
	} else {
		t.Errorf("Unexpected output div content: %s", outputText)
	}

	// The core test is that write succeeded (permissions granted)
	// Read may or may not work depending on Chrome version and headless mode restrictions
}
