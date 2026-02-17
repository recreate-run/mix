//go:build e2e
// +build e2e

package browser

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"regexp"
	"strings"
	"testing"
	"time"

	"mix/internal/constants"
)

// TestBrowserE2EProductURLExtraction tests that read_page can extract product URLs
// from an Amazon-style product listing page using href attributes
func TestBrowserE2EProductURLExtraction(t *testing.T) {
	t.Parallel()
	t.Log("=== E2E Test: Amazon Product URL Extraction ===")

	// Setup test environment
	testServerURL, sessionID, cleanup := setupBrowserTest(t, "E2E Product URL Extraction Test")
	defer cleanup()

	// Step 1: Navigate to mock Amazon products page and use read_page
	t.Log("Step 1: Navigating to product listing page and extracting URLs...")
	testURL := testServerURL + "/amazon_products.html"

	msgResp := sendMessage(t, sessionID, fmt.Sprintf(
		"Navigate to %s. Once the page loads, use read_page with filter=\"links\" to get only the links on the page. "+
			"Count how many product links contain '/dp/' or '/gp/product/' in their href attribute. "+
			"Report the total count and list at least 3 example product URLs.",
		testURL,
	))
	defer func() { _ = msgResp.Body.Close() }()
	t.Log("✓ Message sent, processing started")

	// Step 2: Wait for processing
	t.Log("Step 2: Waiting for read_page processing...")
	waitForProcessing(t, sessionID, 90*time.Second)
	t.Log("✓ Processing completed")

	// Step 3: Verify URL extraction
	t.Log("Step 3: Verifying product URL extraction...")
	messagesResp := makeRequest(t, http.MethodGet, constants.APISessionsPath+sessionID+"/messages", nil)
	defer func() { _ = messagesResp.Body.Close() }()

	messagesBody, err := io.ReadAll(messagesResp.Body)
	if err != nil {
		t.Fatalf("Failed to read messages response: %v", err)
	}

	var messages []map[string]interface{}
	if err := json.Unmarshal(messagesBody, &messages); err != nil {
		t.Fatalf("Failed to parse messages: %v", err)
	}

	// Debug: print message count and roles
	t.Logf("Debug: Found %d messages", len(messages))
	for i, msg := range messages {
		if role, ok := msg["role"].(string); ok {
			t.Logf("Debug: Message %d - role: %s", i, role)
			if content, ok := msg["content"].(string); ok && len(content) > 0 {
				preview := content
				if len(preview) > 100 {
					preview = preview[:100] + "..."
				}
				t.Logf("Debug: Content preview: %s", preview)
			}
		}
	}

	// Verify the workflow
	foundBrowserTool, urlCount, urls := verifyProductURLExtraction(t, messages)

	if !foundBrowserTool {
		t.Fatal("❌ Browser tool was not used")
	}
	t.Log("✓ Browser tool was executed")

	if urlCount == 0 {
		t.Fatal("❌ No product URLs were extracted from read_page results")
	}
	t.Logf("✓ Successfully extracted %d product URLs", urlCount)

	// Log sample URLs
	if len(urls) > 0 {
		t.Log("Sample extracted URLs:")
		for i, url := range urls {
			if i >= 3 {
				break // Show first 3 URLs only
			}
			t.Logf("  - %s", url)
		}
	}

	// Verify all URLs have expected patterns
	validURLs := 0
	for _, url := range urls {
		if strings.Contains(url, "/dp/") || strings.Contains(url, "/gp/product/") {
			validURLs++
		}
	}

	if validURLs == 0 {
		t.Fatal("❌ No URLs matched expected Amazon product patterns (/dp/ or /gp/product/)")
	}
	t.Logf("✓ %d/%d URLs matched expected Amazon product patterns", validURLs, urlCount)

	// Expected: 12 products in the mock HTML (6 with /dp/, 2 with /gp/product/)
	expectedMinURLs := 8 // At least 8 product links should be found
	if urlCount < expectedMinURLs {
		t.Errorf("⚠ Expected at least %d product URLs, but found %d", expectedMinURLs, urlCount)
	} else {
		t.Log("✓ URL count meets expectations")
	}

	t.Log("=== E2E Test Completed Successfully ===")
}

// verifyProductURLExtraction verifies that product URLs were extracted using browser tools
// Returns: (foundBrowserTool, urlCount, extractedURLs)
func verifyProductURLExtraction(t *testing.T, messages []map[string]interface{}) (bool, int, []string) {
	t.Helper()

	foundBrowserTool := false
	extractedURLs := make([]string, 0)

	// Pattern to match href attributes in browser tool output
	// Format: href="/dp/B0CX23V18H" or href="/gp/product/B0CQ55W4PJ"
	hrefPattern := regexp.MustCompile(`href="([^"]*(?:/dp/|/gp/product/)[^"]*)"`)

	for _, msg := range messages {
		// Check for browser tool calls
		if toolCalls, ok := msg["toolCalls"].([]interface{}); ok {
			for _, tc := range toolCalls {
				if toolCall, ok := tc.(map[string]interface{}); ok {
					if name, ok := toolCall["name"].(string); ok && name == "Browser" {
						foundBrowserTool = true

						// Check for read_page or other actions in input
						if input, ok := toolCall["input"].(string); ok {
							inputLower := strings.ToLower(input)
							if strings.Contains(inputLower, "read_page") ||
								strings.Contains(inputLower, "readpage") ||
								strings.Contains(inputLower, "links") {
								t.Log("✓ Found read_page with filter=links in browser tool input")
							}
						}

						// Extract URLs from tool result
						if result, ok := toolCall["result"].(string); ok {
							// Debug: print result preview (show more to see href attributes)
							resultPreview := result
							if len(resultPreview) > 2000 {
								resultPreview = resultPreview[:2000] + "..."
							}
							t.Logf("Debug: Tool result preview: %s", resultPreview)

							// Find all href attributes in the result
							matches := hrefPattern.FindAllStringSubmatch(result, -1)
							for _, match := range matches {
								if len(match) > 1 {
									url := match[1]
									extractedURLs = append(extractedURLs, url)
								}
							}

							if len(matches) > 0 {
								t.Logf("✓ Found %d href attributes in browser tool result", len(matches))
							} else {
								t.Log("⚠ No href patterns matched in tool result")
							}
						}
					}
				}
			}
		}

		// Also check assistant responses for URL mentions
		if role, ok := msg["role"].(string); ok && role == "assistant" {
			if content, ok := msg["content"].(string); ok {
				contentLower := strings.ToLower(content)

				// Check if assistant mentions finding URLs
				if (strings.Contains(contentLower, "url") || strings.Contains(contentLower, "href")) &&
					(strings.Contains(contentLower, "product") || strings.Contains(contentLower, "link")) {
					t.Log("✓ Assistant mentioned finding product URLs in response")
				}

				// Also extract URLs from assistant content (in case they're mentioned)
				additionalMatches := hrefPattern.FindAllStringSubmatch(content, -1)
				for _, match := range additionalMatches {
					if len(match) > 1 {
						url := match[1]
						// Avoid duplicates
						if !contains(extractedURLs, url) {
							extractedURLs = append(extractedURLs, url)
						}
					}
				}
			}
		}
	}

	// Remove duplicates
	uniqueURLs := removeDuplicates(extractedURLs)

	return foundBrowserTool, len(uniqueURLs), uniqueURLs
}

// contains checks if a string slice contains a specific string
func contains(slice []string, item string) bool {
	for _, s := range slice {
		if s == item {
			return true
		}
	}
	return false
}

// removeDuplicates removes duplicate strings from a slice
func removeDuplicates(slice []string) []string {
	seen := make(map[string]bool)
	result := make([]string, 0)

	for _, item := range slice {
		if !seen[item] {
			seen[item] = true
			result = append(result, item)
		}
	}

	return result
}
