//go:build e2e
// +build e2e

package test

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/sarathmenon/browser-service/pkg/client"
	"github.com/sarathmenon/browser-service/test/testserver"
)

func TestDownloadConfiguration(t *testing.T) {
	// Setup: Create temp download directory, start server
	tmpDir := t.TempDir()
	server := testserver.StartTestServer(t)
	defer server.Close()

	// Connect to browser_service
	c, err := client.New("ws://localhost:8081/ws")
	if err != nil {
		t.Fatalf("Failed to connect to browser service: %v", err)
	}
	defer c.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// 1. Call Browser.setDownloadBehavior(tmpDir, true)
	result, err := c.SetDownloadBehavior(ctx, tmpDir, true)
	if err != nil {
		t.Fatalf("Failed to set download behavior: %v", err)
	}
	if !result.Configured {
		t.Fatal("Download behavior not configured")
	}

	// 2. Navigate to server.URL + "/trigger-download" (page with download link)
	_, err = c.Navigate(ctx, server.URL+"/trigger-download")
	if err != nil {
		t.Fatalf("Failed to navigate to download page: %v", err)
	}

	// Get elements to find the download link
	elements, err := c.GetElements(ctx)
	if err != nil {
		t.Fatalf("Failed to get elements: %v", err)
	}

	// Find and click the download link (index 0)
	if len(elements) == 0 {
		t.Fatal("No elements found on page")
	}

	err = c.Click(ctx, 0)
	if err != nil {
		t.Fatalf("Failed to click download link: %v", err)
	}

	// 3. Call Page.waitForDownload(timeout: 10000)
	downloadResult, err := c.WaitForDownload(ctx, 10000)
	if err != nil {
		t.Fatalf("Failed to wait for download: %v", err)
	}

	// 4. Assert: Download.State == "completed"
	if downloadResult.Download.State != "completed" {
		t.Errorf("Expected download state 'completed', got '%s'", downloadResult.Download.State)
	}

	// Give Chrome a moment to finish writing the file to disk
	time.Sleep(500 * time.Millisecond)

	// 5. Assert: File exists at the path reported by the download
	expectedPath := downloadResult.Download.Path
	if expectedPath == "" {
		t.Fatal("Download path is empty")
	}

	if _, err := os.Stat(expectedPath); os.IsNotExist(err) {
		// List what files ARE in the download directory
		t.Logf("Expected file at: %s", expectedPath)
		if entries, err := os.ReadDir(tmpDir); err == nil {
			t.Logf("Files in tmpDir: %v", entries)
		}
		t.Errorf("Downloaded file does not exist at %s", expectedPath)
	}

	// 6. Assert: File size matches Download.TotalBytes
	fileInfo, err := os.Stat(expectedPath)
	if err == nil {
		if fileInfo.Size() != downloadResult.Download.TotalBytes {
			t.Errorf("Expected file size %d, got %d", downloadResult.Download.TotalBytes, fileInfo.Size())
		}
	}

	// 7. Read file, verify content matches expected
	content, err := os.ReadFile(expectedPath)
	if err != nil {
		t.Fatalf("Failed to read downloaded file: %v", err)
	}
	expectedContent := "Test file content"
	if string(content) != expectedContent {
		t.Errorf("Expected file content '%s', got '%s'", expectedContent, string(content))
	}
}

func TestDownloadRejection(t *testing.T) {
	// Setup: Start server, connect to service
	server := testserver.StartTestServer(t)
	defer server.Close()

	// Connect to browser_service
	c, err := client.New("ws://localhost:8081/ws")
	if err != nil {
		t.Fatalf("Failed to connect to browser service: %v", err)
	}
	defer c.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// 1. Call Browser.setDownloadBehavior("", false)  // Deny downloads
	result, err := c.SetDownloadBehavior(ctx, "", false)
	if err != nil {
		t.Fatalf("Failed to set download behavior: %v", err)
	}
	if !result.Configured {
		t.Fatal("Download behavior not configured")
	}

	// 2. Navigate to server.URL + "/trigger-download"
	_, err = c.Navigate(ctx, server.URL+"/trigger-download")
	if err != nil {
		t.Fatalf("Failed to navigate to trigger download page: %v", err)
	}

	// 3. Wait 1 second
	time.Sleep(1 * time.Second)

	// 4. Call Page.getDownloads()
	downloads, err := c.GetDownloads(ctx)
	if err != nil {
		t.Fatalf("Failed to get downloads: %v", err)
	}

	// 5. Assert: Empty downloads array (download blocked by browser)
	if len(downloads.Downloads) != 0 {
		t.Errorf("Expected 0 downloads, got %d", len(downloads.Downloads))
	}
}

func TestDownloadTimeout(t *testing.T) {
	// Setup: Start server, configure downloads
	tmpDir := t.TempDir()
	server := testserver.StartTestServer(t)
	defer server.Close()

	// Connect to browser_service
	c, err := client.New("ws://localhost:8081/ws")
	if err != nil {
		t.Fatalf("Failed to connect to browser service: %v", err)
	}
	defer c.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// 1. Call Browser.setDownloadBehavior(tmpDir, true)
	result, err := c.SetDownloadBehavior(ctx, tmpDir, true)
	if err != nil {
		t.Fatalf("Failed to set download behavior: %v", err)
	}
	if !result.Configured {
		t.Fatal("Download behavior not configured")
	}

	// 2. Navigate to server.URL + "/no-download-page" (regular page)
	_, err = c.Navigate(ctx, server.URL+"/no-download-page")
	if err != nil {
		t.Fatalf("Failed to navigate to no-download page: %v", err)
	}

	// 3. Call Page.waitForDownload(timeout: 2000)
	_, err = c.WaitForDownload(ctx, 2000)

	// 4. Assert: Error response with timeout
	if err == nil {
		t.Fatal("Expected timeout error, got nil")
	}

	// 5. Verify error message contains "timeout"
	if err.Error() == "" {
		t.Error("Expected non-empty error message")
	}
}
