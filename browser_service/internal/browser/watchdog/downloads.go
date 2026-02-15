package watchdog

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"

	"github.com/go-rod/rod"
	"github.com/go-rod/rod/lib/proto"
	"github.com/google/uuid"
	"github.com/sarathmenon/browser-service/internal/browser/events"
)

// DownloadsWatchdog monitors network responses for auto-downloadable files
type DownloadsWatchdog struct {
	page                    *rod.Page
	eventBus                *events.Broker[events.BrowserEvent]
	downloadCallbacks       []DownloadCallback
	detectedDownloads       map[string]bool   // URL → detected
	sessionPDFURLs          map[string]string // URL → path (prevent re-downloads)
	networkMonitoredTargets map[string]bool
	downloadPath            string
	mu                      sync.RWMutex
	callbacksMu             sync.RWMutex
}

// DownloadCallback is called when a download completes
type DownloadCallback func(download events.Download)

// NewDownloadsWatchdog creates a new downloads watchdog
func NewDownloadsWatchdog(page *rod.Page, eventBus *events.Broker[events.BrowserEvent], downloadPath string) *DownloadsWatchdog {
	return &DownloadsWatchdog{
		page:                    page,
		eventBus:                eventBus,
		downloadCallbacks:       []DownloadCallback{},
		detectedDownloads:       make(map[string]bool),
		sessionPDFURLs:          make(map[string]string),
		networkMonitoredTargets: make(map[string]bool),
		downloadPath:            downloadPath,
	}
}

// Start begins monitoring for network-based downloads
func (w *DownloadsWatchdog) Start(ctx context.Context) error {
	// Enable Network domain
	err := proto.NetworkEnable{}.Call(w.page)
	if err != nil {
		return fmt.Errorf("enable network domain: %w", err)
	}

	// Listen for network response events
	go w.page.EachEvent(func(e *proto.NetworkResponseReceived) {
		w.handleNetworkResponse(ctx, e)
	})()

	log.Printf("Downloads watchdog started for page")
	return nil
}

// handleNetworkResponse processes network responses to detect downloadable files
func (w *DownloadsWatchdog) handleNetworkResponse(ctx context.Context, event *proto.NetworkResponseReceived) {
	// Use MIMEType field which is pre-parsed by the browser
	contentType := event.Response.MIMEType

	// Get content-disposition header
	contentDisposition := ""
	if val, ok := event.Response.Headers["content-disposition"]; ok {
		contentDisposition = val.String()
	}
	if contentDisposition == "" {
		if val, ok := event.Response.Headers["Content-Disposition"]; ok {
			contentDisposition = val.String()
		}
	}

	// Skip unwanted types
	unwanted := []string{"image/", "video/", "audio/", "text/css", "text/javascript", "application/json", "font/"}
	for _, prefix := range unwanted {
		if strings.HasPrefix(contentType, prefix) {
			return
		}
	}

	// Detect PDFs
	isPDF := strings.Contains(contentType, "application/pdf")

	// Detect attachments
	isAttachment := strings.Contains(contentDisposition, "attachment")

	if !isPDF && !isAttachment {
		return
	}

	// Check if already detected
	w.mu.Lock()
	if w.detectedDownloads[event.Response.URL] {
		w.mu.Unlock()
		return
	}
	w.detectedDownloads[event.Response.URL] = true

	// Check session cache (prevent re-download)
	if _, exists := w.sessionPDFURLs[event.Response.URL]; exists {
		w.mu.Unlock()
		return
	}
	w.mu.Unlock()

	// Download via JS fetch
	go func() {
		if err := w.downloadViaFetch(ctx, event.Response.URL); err != nil {
			log.Printf("Downloads watchdog: failed to download %s: %v", event.Response.URL, err)
		}
	}()
}

// downloadViaFetch downloads a file using JavaScript fetch
func (w *DownloadsWatchdog) downloadViaFetch(ctx context.Context, url string) error {
	// JavaScript to download file using XMLHttpRequest (synchronous)
	script := fmt.Sprintf(`
		() => {
			const xhr = new XMLHttpRequest();
			xhr.open('GET', %s, false);  // synchronous request
			xhr.overrideMimeType('text/plain; charset=x-user-defined');
			xhr.send(null);

			if (xhr.status !== 200) {
				throw new Error('HTTP error ' + xhr.status);
			}

			const responseText = xhr.responseText;
			const bytes = new Uint8Array(responseText.length);
			for (let i = 0; i < responseText.length; i++) {
				bytes[i] = responseText.charCodeAt(i) & 0xff;
			}

			// Convert to base64
			let binary = '';
			for (let i = 0; i < bytes.length; i++) {
				binary += String.fromCharCode(bytes[i]);
			}

			return {
				data: btoa(binary),
				responseSize: bytes.length
			};
		}
	`, strconv.Quote(url))

	result, err := w.page.Eval(script)
	if err != nil {
		return fmt.Errorf("fetch failed: %w", err)
	}

	// Parse result - marshal then unmarshal to get proper struct
	var fetchResult struct {
		Data         string `json:"data"`
		ResponseSize int    `json:"responseSize"`
	}

	resultBytes, err := result.Value.MarshalJSON()
	if err != nil {
		return fmt.Errorf("marshal result: %w", err)
	}

	if err := json.Unmarshal(resultBytes, &fetchResult); err != nil {
		return fmt.Errorf("unmarshal result: %w", err)
	}

	// Decode base64 data
	fileData, err := base64.StdEncoding.DecodeString(fetchResult.Data)
	if err != nil {
		return fmt.Errorf("decode base64: %w", err)
	}

	// Generate filename
	filename := filepath.Base(url)
	if filename == "" || filename == "." {
		filename = "download"
	}

	// Write to file
	downloadPath := filepath.Join(w.downloadPath, filename)
	if err := os.WriteFile(downloadPath, fileData, 0644); err != nil {
		return fmt.Errorf("write file: %w", err)
	}

	// Cache in session
	w.mu.Lock()
	w.sessionPDFURLs[url] = downloadPath
	w.mu.Unlock()

	// Create download record
	download := events.Download{
		GUID:              uuid.New().String(),
		URL:               url,
		SuggestedFilename: filename,
		TotalBytes:        int64(fetchResult.ResponseSize),
		State:             "completed",
		Path:              downloadPath,
	}

	// Call direct callbacks
	w.callbacksMu.RLock()
	callbacks := make([]DownloadCallback, len(w.downloadCallbacks))
	copy(callbacks, w.downloadCallbacks)
	w.callbacksMu.RUnlock()

	for _, cb := range callbacks {
		cb(download)
	}

	// Emit event
	w.eventBus.Publish(ctx, events.FileDownloadedEvent{
		Download:     download,
		AutoDownload: true,
		FileType:     filepath.Ext(filename),
	})

	log.Printf("Downloads watchdog: downloaded %s to %s (%d bytes)", filename, downloadPath, fetchResult.ResponseSize)
	return nil
}

// OnDownloadComplete registers a callback for download completion
func (w *DownloadsWatchdog) OnDownloadComplete(cb DownloadCallback) {
	w.callbacksMu.Lock()
	defer w.callbacksMu.Unlock()
	w.downloadCallbacks = append(w.downloadCallbacks, cb)
}

// Stop stops the downloads watchdog
func (w *DownloadsWatchdog) Stop() {
	// Disable Network domain
	_ = proto.NetworkDisable{}.Call(w.page)
	log.Printf("Downloads watchdog stopped")
}
