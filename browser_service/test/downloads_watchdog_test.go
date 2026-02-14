//go:build e2e
// +build e2e

package test

import (
	"bytes"
	"context"
	"os"
	"testing"
	"time"

	"github.com/sarathmenon/browser-service/pkg/client"
	"github.com/sarathmenon/browser-service/test/testserver"
)

func TestDownloadsWatchdogNetworkDetection(t *testing.T) {
	server := testserver.StartTestServer(t)
	defer server.Close()

	tmpDir := t.TempDir()

	// Connect to browser_service
	c, err := client.New("ws://localhost:8081/ws")
	if err != nil {
		t.Fatalf("Failed to connect to browser service: %v", err)
	}
	defer c.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Enable downloads
	_, err = c.SetDownloadBehavior(ctx, tmpDir, true)
	if err != nil {
		t.Fatalf("Failed to set download behavior: %v", err)
	}

	// Navigate to CSV with Content-Disposition: attachment
	// This should trigger automatic download via network watchdog
	_, err = c.Navigate(ctx, server.URL+"/data.csv")
	if err != nil {
		t.Logf("Navigate error (may be expected): %v", err)
	}

	// Wait a bit for network watchdog to detect and download
	time.Sleep(3 * time.Second)

	// Verify file was downloaded
	files, err := os.ReadDir(tmpDir)
	if err != nil {
		t.Fatalf("Failed to read download directory: %v", err)
	}

	if len(files) == 0 {
		t.Fatal("Expected download file, but directory is empty")
	}

	// Verify CSV file exists
	csvPath := tmpDir + "/data.csv"
	if _, err := os.Stat(csvPath); os.IsNotExist(err) {
		t.Fatalf("Expected file data.csv not found at %s", csvPath)
	}

	// Verify content
	content, err := os.ReadFile(csvPath)
	if err != nil {
		t.Fatalf("Failed to read downloaded CSV: %v", err)
	}

	expectedContent := "col1,col2\nval1,val2"
	if string(content) != expectedContent {
		t.Errorf("Expected CSV content '%s', got '%s'", expectedContent, string(content))
	}
}

func TestDownloadsWatchdogPDFContentType(t *testing.T) {
	server := testserver.StartTestServer(t)
	defer server.Close()

	tmpDir := t.TempDir()

	// Connect to browser_service
	c, err := client.New("ws://localhost:8081/ws")
	if err != nil {
		t.Fatalf("Failed to connect to browser service: %v", err)
	}
	defer c.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Enable downloads
	_, err = c.SetDownloadBehavior(ctx, tmpDir, true)
	if err != nil {
		t.Fatalf("Failed to set download behavior: %v", err)
	}

	// Navigate to page that fetches PDF via JavaScript fetch()
	// This triggers network monitoring and should auto-download the PDF
	// based on Content-Type: application/pdf (NO Content-Disposition header)
	_, err = c.Navigate(ctx, server.URL+"/pdf-fetch-page")
	if err != nil {
		t.Fatalf("Failed to navigate to PDF fetch page: %v", err)
	}

	// Wait for watchdog to detect and download the PDF
	time.Sleep(5 * time.Second)

	// Verify file was downloaded
	files, err := os.ReadDir(tmpDir)
	if err != nil {
		t.Fatalf("Failed to read download directory: %v", err)
	}

	if len(files) == 0 {
		t.Fatal("Expected download file, but directory is empty")
	}

	// Find PDF file
	pdfPath := tmpDir + "/report.pdf"
	if _, err := os.Stat(pdfPath); os.IsNotExist(err) {
		t.Fatalf("Expected file report.pdf not found at %s", pdfPath)
	}

	// Verify PDF magic bytes
	data, err := os.ReadFile(pdfPath)
	if err != nil {
		t.Fatalf("Failed to read downloaded PDF: %v", err)
	}

	if !bytes.HasPrefix(data, []byte("%PDF")) {
		t.Errorf("Expected PDF magic bytes '%%PDF', got '%s'", string(data[0:4]))
	}
}

func TestDownloadsWatchdogPreventsDuplicates(t *testing.T) {
	server := testserver.StartTestServer(t)
	defer server.Close()

	tmpDir := t.TempDir()

	// Connect to browser_service
	c, err := client.New("ws://localhost:8081/ws")
	if err != nil {
		t.Fatalf("Failed to connect to browser service: %v", err)
	}
	defer c.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Enable downloads
	_, err = c.SetDownloadBehavior(ctx, tmpDir, true)
	if err != nil {
		t.Fatalf("Failed to set download behavior: %v", err)
	}

	// Navigate to PDF first time
	_, err = c.Navigate(ctx, server.URL+"/document.pdf")
	if err != nil {
		t.Logf("Navigate error (may be expected): %v", err)
	}

	// Wait for first download
	time.Sleep(3 * time.Second)

	// Count files after first download
	files1, err := os.ReadDir(tmpDir)
	if err != nil {
		t.Fatalf("Failed to read download directory: %v", err)
	}

	if len(files1) != 1 {
		t.Fatalf("Expected 1 file after first download, got %d", len(files1))
	}

	// Navigate to SAME URL again
	_, err = c.Navigate(ctx, server.URL+"/document.pdf")
	if err != nil {
		t.Logf("Navigate error (may be expected): %v", err)
	}

	// Wait to see if second download happens
	time.Sleep(3 * time.Second)

	// Verify only one file exists (no duplicate download)
	files2, err := os.ReadDir(tmpDir)
	if err != nil {
		t.Fatalf("Failed to read download directory: %v", err)
	}

	if len(files2) != 1 {
		t.Errorf("Expected 1 file (no duplicate), got %d files", len(files2))
	}
}
