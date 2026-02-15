package watchdog

import (
	"context"
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/go-rod/rod"
	"github.com/go-rod/rod/lib/proto"
)

// PopupsWatchdog handles automatic JavaScript dialog dismissal
type PopupsWatchdog struct {
	browser              *rod.Browser
	closedPopupMessages  []string
	closedPopupMessagesMu sync.RWMutex
	dialogListeners      map[string]bool // target_id -> registered
	dialogListenersMu    sync.RWMutex
	cancelFuncs          []context.CancelFunc
	mu                   sync.Mutex
}

// NewPopupsWatchdog creates a new popups watchdog
func NewPopupsWatchdog(browser *rod.Browser) *PopupsWatchdog {
	return &PopupsWatchdog{
		browser:             browser,
		closedPopupMessages: make([]string, 0),
		dialogListeners:     make(map[string]bool),
		cancelFuncs:         make([]context.CancelFunc, 0),
	}
}

// Start begins monitoring for JavaScript dialogs
func (w *PopupsWatchdog) Start(ctx context.Context) error {
	// Enable Page domain for all pages (required to receive dialog events)
	pages, err := w.browser.Pages()
	if err != nil {
		return fmt.Errorf("get pages: %w", err)
	}

	for _, page := range pages {
		if err := w.registerDialogListener(ctx, page); err != nil {
			log.Printf("Failed to register dialog listener for page: %v", err)
		}
	}

	return nil
}

// registerDialogListener registers a dialog event listener for a page
func (w *PopupsWatchdog) registerDialogListener(ctx context.Context, page *rod.Page) error {
	info, err := page.Info()
	if err != nil {
		return fmt.Errorf("get page info: %w", err)
	}

	targetID := string(info.TargetID)

	// Check if already registered
	w.dialogListenersMu.Lock()
	if w.dialogListeners[targetID] {
		w.dialogListenersMu.Unlock()
		return nil
	}
	w.dialogListeners[targetID] = true
	w.dialogListenersMu.Unlock()

	// Enable Page domain to receive dialog events
	err = proto.PageEnable{}.Call(page)
	if err != nil {
		return fmt.Errorf("enable page domain: %w", err)
	}

	// Create a context for this listener
	listenerCtx, cancel := context.WithCancel(ctx)
	w.mu.Lock()
	w.cancelFuncs = append(w.cancelFuncs, cancel)
	w.mu.Unlock()

	// Listen for dialog events on this specific page
	go page.EachEvent(func(e *proto.PageJavascriptDialogOpening) {
		w.handleDialog(listenerCtx, e)
	})()

	return nil
}

// handleDialog handles JavaScript dialog events
func (w *PopupsWatchdog) handleDialog(ctx context.Context, event *proto.PageJavascriptDialogOpening) {
	// Store message
	message := fmt.Sprintf("[%s] %s", event.Type, event.Message)
	w.closedPopupMessagesMu.Lock()
	w.closedPopupMessages = append(w.closedPopupMessages, message)
	w.closedPopupMessagesMu.Unlock()

	// Determine accept/dismiss based on dialog type
	accept := event.Type == proto.PageDialogTypeAlert ||
		event.Type == proto.PageDialogTypeConfirm ||
		event.Type == proto.PageDialogTypeBeforeunload

	// Try to handle dialog with timeout
	handleCtx, cancel := context.WithTimeout(ctx, 500*time.Millisecond)
	defer cancel()

	if err := w.handleDialogWithStrategies(handleCtx, accept); err != nil {
		log.Printf("Failed to handle dialog: %v", err)
	}
}

// handleDialogWithStrategies tries multiple strategies to handle the dialog
func (w *PopupsWatchdog) handleDialogWithStrategies(ctx context.Context, accept bool) error {
	// Get all pages and try to handle dialog on each
	pages, err := w.browser.Pages()
	if err != nil {
		return fmt.Errorf("get pages: %w", err)
	}

	for _, page := range pages {
		err := proto.PageHandleJavaScriptDialog{
			Accept: accept,
		}.Call(page)

		if err == nil {
			return nil
		}
	}

	return fmt.Errorf("all strategies failed")
}

// GetClosedPopupMessages returns all closed popup messages
func (w *PopupsWatchdog) GetClosedPopupMessages() []string {
	w.closedPopupMessagesMu.RLock()
	defer w.closedPopupMessagesMu.RUnlock()

	messages := make([]string, len(w.closedPopupMessages))
	copy(messages, w.closedPopupMessages)
	return messages
}

// RegisterPage registers dialog listeners for a new page
func (w *PopupsWatchdog) RegisterPage(ctx context.Context, page *rod.Page) error {
	return w.registerDialogListener(ctx, page)
}

// ClearMessages clears all stored popup messages
func (w *PopupsWatchdog) ClearMessages() {
	w.closedPopupMessagesMu.Lock()
	defer w.closedPopupMessagesMu.Unlock()
	w.closedPopupMessages = make([]string, 0)
}

// Stop stops all dialog listeners
func (w *PopupsWatchdog) Stop() {
	w.mu.Lock()
	defer w.mu.Unlock()

	for _, cancel := range w.cancelFuncs {
		cancel()
	}
	w.cancelFuncs = nil
}
