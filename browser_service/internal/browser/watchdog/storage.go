package watchdog

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"reflect"
	"sync"
	"time"

	"github.com/go-rod/rod"
	"github.com/go-rod/rod/lib/proto"
	"github.com/sarathmenon/browser-service/internal/browser/events"
	"github.com/sarathmenon/browser-service/pkg/protocol"
)

// StorageStateWatchdog monitors and auto-saves browser storage state
type StorageStateWatchdog struct {
	browser          *rod.Browser
	eventBus         *events.Broker[events.BrowserEvent]
	storageStatePath string
	lastCookieState  map[string]string // key: name+domain+path, value: cookie_value
	saveLock         sync.Mutex
	monitoringCancel context.CancelFunc
	mu               sync.Mutex
}

// NewStorageStateWatchdog creates a new storage state watchdog
func NewStorageStateWatchdog(browser *rod.Browser, eventBus *events.Broker[events.BrowserEvent], storageStatePath string) *StorageStateWatchdog {
	return &StorageStateWatchdog{
		browser:          browser,
		eventBus:         eventBus,
		storageStatePath: storageStatePath,
		lastCookieState:  make(map[string]string),
	}
}

// Start begins monitoring for storage state changes
func (w *StorageStateWatchdog) Start(ctx context.Context) error {
	// Auto-load storage state if file exists
	if err := w.autoLoadStorageState(ctx); err != nil {
		log.Printf("Warning: failed to auto-load storage state: %v", err)
	}

	// Start monitoring loop
	monitorCtx, cancel := context.WithCancel(ctx)
	w.monitoringCancel = cancel
	go w.monitoringLoop(monitorCtx)

	return nil
}

// autoLoadStorageState loads storage state from file if it exists
func (w *StorageStateWatchdog) autoLoadStorageState(ctx context.Context) error {
	// Check if storage state file exists
	if _, err := os.Stat(w.storageStatePath); os.IsNotExist(err) {
		return nil // No file to load
	}

	// Read file
	data, err := os.ReadFile(w.storageStatePath)
	if err != nil {
		return fmt.Errorf("read storage state file: %w", err)
	}

	// Parse JSON
	var state protocol.StorageState
	if err := json.Unmarshal(data, &state); err != nil {
		return fmt.Errorf("parse storage state: %w", err)
	}

	// Load storage state into browser
	if err := w.loadStorageStateIntoBrowser(ctx, state); err != nil {
		return fmt.Errorf("load storage state into browser: %w", err)
	}

	// Publish event
	w.eventBus.Publish(ctx, events.StorageStateLoadedEvent{
		Path:         w.storageStatePath,
		CookiesCount: len(state.Cookies),
		OriginsCount: len(state.Origins),
	})

	// Update last cookie state
	w.lastCookieState = w.buildCookieMapFromState(state.Cookies)

	log.Printf("Auto-loaded storage state: %d cookies, %d origins", len(state.Cookies), len(state.Origins))
	return nil
}

// loadStorageStateIntoBrowser loads a storage state into the browser
func (w *StorageStateWatchdog) loadStorageStateIntoBrowser(ctx context.Context, state protocol.StorageState) error {
	// Get first page
	pages, err := w.browser.Pages()
	if err != nil {
		return fmt.Errorf("get pages: %w", err)
	}
	if len(pages) == 0 {
		return fmt.Errorf("no pages available")
	}
	page := pages[0]

	// Save current URL
	currentURL := ""
	if info, err := page.Info(); err == nil && info != nil {
		currentURL = info.URL
	}

	// Set cookies (can be done cross-origin)
	for _, cookie := range state.Cookies {
		sameSite := proto.NetworkCookieSameSiteLax
		switch cookie.SameSite {
		case "Strict":
			sameSite = proto.NetworkCookieSameSiteStrict
		case "None":
			sameSite = proto.NetworkCookieSameSiteNone
		case "Lax":
			sameSite = proto.NetworkCookieSameSiteLax
		}

		_, err := proto.NetworkSetCookie{
			Name:     cookie.Name,
			Value:    cookie.Value,
			Domain:   cookie.Domain,
			Path:     cookie.Path,
			Expires:  proto.TimeSinceEpoch(cookie.Expires),
			HTTPOnly: cookie.HTTPOnly,
			Secure:   cookie.Secure,
			SameSite: sameSite,
		}.Call(page)

		if err != nil {
			log.Printf("Warning: failed to set cookie %s: %v", cookie.Name, err)
		}
	}

	// Set localStorage for each origin
	for _, origin := range state.Origins {
		if len(origin.LocalStorage) == 0 {
			continue
		}

		// Navigate to origin
		err := page.Navigate(origin.Origin)
		if err != nil {
			log.Printf("Warning: failed to navigate to origin %s: %v", origin.Origin, err)
			continue
		}

		// Wait for page to load
		err = page.WaitLoad()
		if err != nil {
			log.Printf("Warning: failed to wait for origin %s: %v", origin.Origin, err)
			continue
		}

		// Set localStorage items
		for key, value := range origin.LocalStorage {
			_, err := page.Eval(fmt.Sprintf("() => localStorage.setItem(%s, %s)",
				quoteString(key), quoteString(value)))
			if err != nil {
				log.Printf("Warning: failed to set localStorage %s=%s: %v", key, value, err)
			}
		}
	}

	// Navigate back to original URL or about:blank
	if currentURL != "" && currentURL != "about:blank" {
		_ = page.Navigate(currentURL)
	}

	return nil
}

// monitoringLoop runs periodic storage state checks
func (w *StorageStateWatchdog) monitoringLoop(ctx context.Context) {
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			if err := w.checkAndSave(ctx); err != nil {
				log.Printf("Storage watchdog: failed to check and save: %v", err)
			}
		}
	}
}

// checkAndSave checks for cookie changes and saves if needed
func (w *StorageStateWatchdog) checkAndSave(ctx context.Context) error {
	// Get current cookies
	cookies, err := w.getCurrentCookies()
	if err != nil {
		return fmt.Errorf("get cookies: %w", err)
	}

	// Compare with last state
	current := w.buildCookieMap(cookies)

	w.mu.Lock()
	lastState := w.lastCookieState
	w.mu.Unlock()

	if reflect.DeepEqual(current, lastState) {
		return nil // No changes
	}

	// Save to file
	if err := w.saveStorageState(ctx, cookies); err != nil {
		return fmt.Errorf("save storage state: %w", err)
	}

	// Update last state
	w.mu.Lock()
	w.lastCookieState = current
	w.mu.Unlock()

	return nil
}

// getCurrentCookies retrieves all cookies from the browser
func (w *StorageStateWatchdog) getCurrentCookies() ([]protocol.Cookie, error) {
	// Get first page
	pages, err := w.browser.Pages()
	if err != nil {
		return nil, fmt.Errorf("get pages: %w", err)
	}
	if len(pages) == 0 {
		return nil, fmt.Errorf("no pages available")
	}
	page := pages[0]

	// Get all cookies
	cookiesProto, err := proto.NetworkGetAllCookies{}.Call(page)
	if err != nil {
		return nil, fmt.Errorf("call network.getAllCookies: %w", err)
	}

	// Convert CDP cookies to protocol cookies
	cookies := make([]protocol.Cookie, len(cookiesProto.Cookies))
	for i, c := range cookiesProto.Cookies {
		cookies[i] = protocol.Cookie{
			Name:     c.Name,
			Value:    c.Value,
			Domain:   c.Domain,
			Path:     c.Path,
			Expires:  float64(c.Expires),
			HTTPOnly: c.HTTPOnly,
			Secure:   c.Secure,
			SameSite: string(c.SameSite),
		}
	}

	return cookies, nil
}

// saveStorageState saves the current storage state to file with atomic writes
func (w *StorageStateWatchdog) saveStorageState(ctx context.Context, cookies []protocol.Cookie) error {
	w.saveLock.Lock()
	defer w.saveLock.Unlock()

	// Get current storage state from browser
	state, err := w.getCurrentStorageState()
	if err != nil {
		return fmt.Errorf("get current storage state: %w", err)
	}

	// Merge with existing file if it exists
	mergedState, err := w.mergeStorageState(state)
	if err != nil {
		return fmt.Errorf("merge storage state: %w", err)
	}

	// Marshal to JSON
	data, err := json.MarshalIndent(mergedState, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal storage state: %w", err)
	}

	// Atomic write: .tmp → .bak → final
	tmpPath := w.storageStatePath + ".tmp"
	bakPath := w.storageStatePath + ".bak"

	// Write to temp file
	if err := os.WriteFile(tmpPath, data, 0644); err != nil {
		return fmt.Errorf("write temp file: %w", err)
	}

	// Backup existing file
	if _, err := os.Stat(w.storageStatePath); err == nil {
		_ = os.Rename(w.storageStatePath, bakPath)
	}

	// Move temp to final
	if err := os.Rename(tmpPath, w.storageStatePath); err != nil {
		return fmt.Errorf("rename temp to final: %w", err)
	}

	// Publish event
	w.eventBus.Publish(ctx, events.StorageStateSavedEvent{
		Path:         w.storageStatePath,
		CookiesCount: len(mergedState.Cookies),
		OriginsCount: len(mergedState.Origins),
	})

	log.Printf("Storage state saved: %d cookies, %d origins", len(mergedState.Cookies), len(mergedState.Origins))
	return nil
}

// getCurrentStorageState retrieves current storage state from browser
func (w *StorageStateWatchdog) getCurrentStorageState() (protocol.StorageState, error) {
	// Get cookies
	cookies, err := w.getCurrentCookies()
	if err != nil {
		return protocol.StorageState{}, fmt.Errorf("get cookies: %w", err)
	}

	// Get localStorage from all origins
	origins, err := w.getCurrentLocalStorage()
	if err != nil {
		log.Printf("Warning: failed to get localStorage: %v", err)
		// Continue with just cookies
	}

	return protocol.StorageState{
		Cookies: cookies,
		Origins: origins,
	}, nil
}

// getCurrentLocalStorage retrieves localStorage from all origins
func (w *StorageStateWatchdog) getCurrentLocalStorage() ([]protocol.OriginState, error) {
	// Get first page
	pages, err := w.browser.Pages()
	if err != nil {
		return nil, fmt.Errorf("get pages: %w", err)
	}
	if len(pages) == 0 {
		return nil, fmt.Errorf("no pages available")
	}
	page := pages[0]

	// Get current page origin
	info, err := page.Info()
	if err != nil {
		return nil, fmt.Errorf("get page info: %w", err)
	}

	// Only get localStorage from current origin
	// (we can't enumerate all origins without navigating)
	if info.URL == "" || info.URL == "about:blank" {
		return nil, nil
	}

	// Get localStorage
	result, err := page.Eval("() => { return JSON.stringify(localStorage) }")
	if err != nil {
		return nil, fmt.Errorf("eval localStorage: %w", err)
	}

	var localStorage map[string]string
	if err := json.Unmarshal([]byte(result.Value.String()), &localStorage); err != nil {
		return nil, fmt.Errorf("unmarshal localStorage: %w", err)
	}

	if len(localStorage) == 0 {
		return nil, nil
	}

	// Parse origin from URL
	origin := info.URL
	// Simple origin extraction (protocol://host:port)
	if idx := len(origin); idx > 0 {
		// Find third slash (after protocol://)
		slashCount := 0
		for i, c := range origin {
			if c == '/' {
				slashCount++
				if slashCount == 3 {
					origin = origin[:i]
					break
				}
			}
		}
	}

	return []protocol.OriginState{
		{
			Origin:       origin,
			LocalStorage: localStorage,
		},
	}, nil
}

// mergeStorageState merges new state with existing file (new values win)
func (w *StorageStateWatchdog) mergeStorageState(newState protocol.StorageState) (protocol.StorageState, error) {
	// Check if existing file exists
	if _, err := os.Stat(w.storageStatePath); os.IsNotExist(err) {
		return newState, nil // No existing file, return new state
	}

	// Read existing file
	data, err := os.ReadFile(w.storageStatePath)
	if err != nil {
		log.Printf("Warning: failed to read existing storage state: %v", err)
		return newState, nil // Return new state on error
	}

	// Parse existing state
	var existingState protocol.StorageState
	if err := json.Unmarshal(data, &existingState); err != nil {
		log.Printf("Warning: failed to parse existing storage state: %v", err)
		return newState, nil // Return new state on error
	}

	// Merge cookies (new values win, keyed by name+domain+path)
	cookieMap := make(map[string]protocol.Cookie)

	// Add existing cookies
	for _, cookie := range existingState.Cookies {
		key := fmt.Sprintf("%s|%s|%s", cookie.Name, cookie.Domain, cookie.Path)
		cookieMap[key] = cookie
	}

	// Overwrite with new cookies
	for _, cookie := range newState.Cookies {
		key := fmt.Sprintf("%s|%s|%s", cookie.Name, cookie.Domain, cookie.Path)
		cookieMap[key] = cookie
	}

	// Convert back to slice
	mergedCookies := make([]protocol.Cookie, 0, len(cookieMap))
	for _, cookie := range cookieMap {
		mergedCookies = append(mergedCookies, cookie)
	}

	// Merge localStorage per origin
	originMap := make(map[string]protocol.OriginState)

	// Add existing origins
	for _, origin := range existingState.Origins {
		originMap[origin.Origin] = origin
	}

	// Merge with new origins
	for _, newOrigin := range newState.Origins {
		existing, exists := originMap[newOrigin.Origin]
		if !exists {
			originMap[newOrigin.Origin] = newOrigin
			continue
		}

		// Merge localStorage (new values win)
		if existing.LocalStorage == nil {
			existing.LocalStorage = make(map[string]string)
		}
		for k, v := range newOrigin.LocalStorage {
			existing.LocalStorage[k] = v
		}
		originMap[newOrigin.Origin] = existing
	}

	// Convert back to slice
	mergedOrigins := make([]protocol.OriginState, 0, len(originMap))
	for _, origin := range originMap {
		mergedOrigins = append(mergedOrigins, origin)
	}

	return protocol.StorageState{
		Cookies: mergedCookies,
		Origins: mergedOrigins,
	}, nil
}

// buildCookieMap creates a map of cookie keys to values for comparison
func (w *StorageStateWatchdog) buildCookieMap(cookies []protocol.Cookie) map[string]string {
	m := make(map[string]string)
	for _, c := range cookies {
		key := fmt.Sprintf("%s|%s|%s", c.Name, c.Domain, c.Path)
		m[key] = c.Value
	}
	return m
}

// buildCookieMapFromState creates a map from StorageState cookies
func (w *StorageStateWatchdog) buildCookieMapFromState(cookies []protocol.Cookie) map[string]string {
	return w.buildCookieMap(cookies)
}

// Stop stops the storage state watchdog
func (w *StorageStateWatchdog) Stop() {
	w.mu.Lock()
	defer w.mu.Unlock()

	if w.monitoringCancel != nil {
		w.monitoringCancel()
		w.monitoringCancel = nil
	}
}

// quoteString quotes a string for JavaScript
func quoteString(s string) string {
	data, _ := json.Marshal(s)
	return string(data)
}
