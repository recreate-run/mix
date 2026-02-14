package watchdog

import (
	"context"
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/go-rod/rod"
	"github.com/go-rod/rod/lib/proto"
	"github.com/sarathmenon/browser-service/internal/browser/events"
)

// NetworkRequest tracks an in-flight network request
type NetworkRequest struct {
	RequestID    string
	StartTime    time.Time
	URL          string
	Method       string
	ResourceType string
}

// CrashWatchdog monitors browser health and detects crashes
type CrashWatchdog struct {
	browser              *rod.Browser
	eventBus             *events.Broker[events.BrowserEvent]
	networkRequests      map[string]*NetworkRequest
	networkRequestsMu    sync.RWMutex
	targetsWithListeners map[string]bool
	targetsMu            sync.RWMutex
	cdpEventCancels      []context.CancelFunc
	monitoringCancel     context.CancelFunc
	mu                   sync.Mutex
}

// NewCrashWatchdog creates a new crash watchdog
func NewCrashWatchdog(browser *rod.Browser, eventBus *events.Broker[events.BrowserEvent]) *CrashWatchdog {
	return &CrashWatchdog{
		browser:              browser,
		eventBus:             eventBus,
		networkRequests:      make(map[string]*NetworkRequest),
		targetsWithListeners: make(map[string]bool),
		cdpEventCancels:      make([]context.CancelFunc, 0),
	}
}

// Start begins monitoring for crashes and browser health
func (w *CrashWatchdog) Start(ctx context.Context) error {
	// Enable Network domain for all pages to track requests
	pages, err := w.browser.Pages()
	if err != nil {
		return fmt.Errorf("get pages: %w", err)
	}

	for _, page := range pages {
		if err := w.registerPageListeners(ctx, page); err != nil {
			log.Printf("Failed to register listeners for page: %v", err)
		}
	}

	// Start monitoring loop
	monitorCtx, cancel := context.WithCancel(ctx)
	w.monitoringCancel = cancel
	go w.monitoringLoop(monitorCtx)

	return nil
}

// RegisterPage registers crash detection listeners for a new page
func (w *CrashWatchdog) RegisterPage(ctx context.Context, page *rod.Page) error {
	return w.registerPageListeners(ctx, page)
}

// registerPageListeners sets up CDP event listeners for a page
func (w *CrashWatchdog) registerPageListeners(ctx context.Context, page *rod.Page) error {
	info, err := page.Info()
	if err != nil {
		return fmt.Errorf("get page info: %w", err)
	}

	targetID := string(info.TargetID)

	// Check if already registered
	w.targetsMu.Lock()
	if w.targetsWithListeners[targetID] {
		w.targetsMu.Unlock()
		return nil
	}
	w.targetsWithListeners[targetID] = true
	w.targetsMu.Unlock()

	// Enable Network domain
	err = proto.NetworkEnable{}.Call(page)
	if err != nil {
		return fmt.Errorf("enable network domain: %w", err)
	}

	// Enable Target domain for crash events
	err = proto.TargetSetDiscoverTargets{Discover: true}.Call(page)
	if err != nil {
		// Log but don't fail - target domain may not be critical
		log.Printf("Warning: failed to enable target domain: %v", err)
	}

	// Listen for Target.targetCrashed events
	listenerCtx, cancel := context.WithCancel(ctx)
	w.mu.Lock()
	w.cdpEventCancels = append(w.cdpEventCancels, cancel)
	w.mu.Unlock()

	go page.EachEvent(func(e *proto.TargetTargetCrashed) {
		w.handleTargetCrashed(&events.TargetCrashedEvent{
			TargetID: string(e.TargetID),
		})
	})()

	// Listen for Network.requestWillBeSent
	go page.EachEvent(func(e *proto.NetworkRequestWillBeSent) {
		w.trackNetworkRequest(e)
	})()

	// Listen for Network.responseReceived
	go page.EachEvent(func(e *proto.NetworkResponseReceived) {
		w.clearNetworkRequest(string(e.RequestID))
	})()

	// Listen for Network.loadingFailed
	go page.EachEvent(func(e *proto.NetworkLoadingFailed) {
		w.clearNetworkRequest(string(e.RequestID))
	})()

	// Clean up listeners when context is cancelled
	go func() {
		<-listenerCtx.Done()
		w.targetsMu.Lock()
		delete(w.targetsWithListeners, targetID)
		w.targetsMu.Unlock()
	}()

	return nil
}

// trackNetworkRequest records a new network request
func (w *CrashWatchdog) trackNetworkRequest(event *proto.NetworkRequestWillBeSent) {
	w.networkRequestsMu.Lock()
	defer w.networkRequestsMu.Unlock()

	requestID := string(event.RequestID)
	w.networkRequests[requestID] = &NetworkRequest{
		RequestID:    requestID,
		StartTime:    time.Now(),
		URL:          event.Request.URL,
		Method:       event.Request.Method,
		ResourceType: string(event.Type),
	}
}

// clearNetworkRequest removes a completed network request
func (w *CrashWatchdog) clearNetworkRequest(requestID string) {
	w.networkRequestsMu.Lock()
	defer w.networkRequestsMu.Unlock()

	delete(w.networkRequests, requestID)
}

// monitoringLoop runs periodic health checks
func (w *CrashWatchdog) monitoringLoop(ctx context.Context) {
	// Initial delay before monitoring begins
	time.Sleep(10 * time.Second)

	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			w.checkNetworkTimeouts(ctx)
			w.checkBrowserHealth(ctx)
		}
	}
}

// checkNetworkTimeouts detects hanging network requests
func (w *CrashWatchdog) checkNetworkTimeouts(ctx context.Context) {
	w.networkRequestsMu.Lock()
	defer w.networkRequestsMu.Unlock()

	now := time.Now()
	for id, req := range w.networkRequests {
		elapsed := now.Sub(req.StartTime)
		if elapsed > 10*time.Second {
			w.eventBus.Publish(ctx, events.BrowserErrorEvent{
				ErrorType: "NetworkTimeout",
				Details: map[string]any{
					"url":             req.URL,
					"elapsed_seconds": elapsed.Seconds(),
					"method":          req.Method,
					"resource_type":   req.ResourceType,
				},
			})
			delete(w.networkRequests, id)
		}
	}
}

// checkBrowserHealth runs a simple health check
func (w *CrashWatchdog) checkBrowserHealth(ctx context.Context) {
	// Get all pages
	pages, err := w.browser.Pages()
	if err != nil {
		log.Printf("Health check: failed to get pages: %v", err)
		return
	}

	if len(pages) == 0 {
		return
	}

	// Check first page with a simple eval
	page := pages[0]
	healthCtx, cancel := context.WithTimeout(ctx, 1*time.Second)
	defer cancel()

	_, err = page.Context(healthCtx).Eval("() => 1+1")
	if err != nil {
		// Browser is unresponsive
		w.eventBus.Publish(ctx, events.BrowserErrorEvent{
			ErrorType: "BrowserUnresponsive",
			Details: map[string]any{
				"error": err.Error(),
			},
		})
	}
}

// handleTargetCrashed handles target crash events
func (w *CrashWatchdog) handleTargetCrashed(event *events.TargetCrashedEvent) {
	w.eventBus.Publish(context.Background(), events.BrowserErrorEvent{
		ErrorType: "TargetCrash",
		Details: map[string]any{
			"target_id": event.TargetID,
		},
	})
}

// Stop stops the crash watchdog
func (w *CrashWatchdog) Stop() {
	w.mu.Lock()
	defer w.mu.Unlock()

	// Cancel monitoring loop
	if w.monitoringCancel != nil {
		w.monitoringCancel()
	}

	// Cancel all CDP event listeners
	for _, cancel := range w.cdpEventCancels {
		cancel()
	}
	w.cdpEventCancels = nil
}
