package browser

import (
	"context"
	"fmt"
	"sync"

	"github.com/go-rod/rod"
	"github.com/go-rod/rod/lib/launcher"
	"github.com/go-rod/rod/lib/proto"
	"github.com/sarathmenon/browser-service/internal/browser/events"
	"github.com/sarathmenon/browser-service/internal/browser/watchdog"
	"github.com/sarathmenon/browser-service/internal/errors"
	"github.com/sarathmenon/browser-service/pkg/protocol"
)

// Config holds browser configuration
type Config struct {
	Headless     bool
	Stealth      bool
	WindowWidth  int
	WindowHeight int
}

// Manager manages the browser instance and contexts
type Manager struct {
	browser *rod.Browser
	config  Config
	mu      sync.Mutex
}

// NewManager creates a new browser manager
func NewManager(ctx context.Context, cfg Config) (*Manager, error) {
	// Set defaults
	if cfg.WindowWidth == 0 {
		cfg.WindowWidth = 1920
	}
	if cfg.WindowHeight == 0 {
		cfg.WindowHeight = 1080
	}

	// Configure browser launcher
	l := launcher.New().
		Headless(cfg.Headless).
		Devtools(false).
		Set("ignore-certificate-errors").      // Ignore SSL certificate errors for testing
		Set("allow-insecure-localhost").       // Allow insecure localhost connections
		Set("disable-web-security").           // Disable web security for testing
		Set("disable-pdf-viewer")              // Auto-download PDFs instead of displaying in browser

	// Add stealth arguments if enabled
	if cfg.Stealth {
		l = l.
			Set("disable-blink-features", "AutomationControlled").
			Set("disable-sync").
			Set("no-first-run").
			Set("disable-client-side-phishing-detection").
			Set("silent-debugger-extension-api").
			Set("disable-component-extensions-with-background-pages").
			Set("no-default-browser-check").
			Set("disable-background-networking")
	}

	// Set window size
	l = l.Set("window-size", fmt.Sprintf("%d,%d", cfg.WindowWidth, cfg.WindowHeight))

	// Launch browser
	url, err := l.Launch()
	if err != nil {
		return nil, errors.NewBrowserError("launch", err)
	}

	browser := rod.New().ControlURL(url)
	if err := browser.Connect(); err != nil {
		return nil, errors.NewBrowserError("connect", err)
	}

	return &Manager{
		browser: browser,
		config:  cfg,
	}, nil
}

// NewContext creates a new isolated browser context with initial tab
func (m *Manager) NewContext(ctx context.Context) (*Context, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	// Create incognito context for isolation
	browserCtx, err := m.browser.Incognito()
	if err != nil {
		return nil, errors.NewContextError("create_incognito", err)
	}

	// Create initial page
	page, err := browserCtx.Page(proto.TargetCreateTarget{})
	if err != nil {
		browserCtx.MustClose()
		return nil, errors.NewContextError("create_page", err)
	}

	// Set viewport to match window size
	err = page.SetViewport(&proto.EmulationSetDeviceMetricsOverride{
		Width:             m.config.WindowWidth,
		Height:            m.config.WindowHeight,
		DeviceScaleFactor: 1,
		Mobile:            false,
	})
	if err != nil {
		browserCtx.MustClose()
		return nil, errors.NewContextError("set_viewport", err)
	}

	// Create initial tab with ID "tab-1"
	initialTab := &tabContext{
		id:           "tab-1",
		page:         page,
		downloads:    make([]protocol.Download, 0),
		downloadChan: make(chan protocol.Download, 10),
	}

	// Create event bus
	eventBus := events.NewBroker[events.BrowserEvent]()

	// Create and start watchdogs
	popupsWd := watchdog.NewPopupsWatchdog(browserCtx)
	permissionsWd := watchdog.NewPermissionsWatchdog(browserCtx)
	crashWd := watchdog.NewCrashWatchdog(browserCtx, eventBus)

	// Start watchdogs
	if err := popupsWd.Start(ctx); err != nil {
		browserCtx.MustClose()
		return nil, errors.NewContextError("start_popups_watchdog", err)
	}

	if err := permissionsWd.Start(ctx); err != nil {
		browserCtx.MustClose()
		return nil, errors.NewContextError("start_permissions_watchdog", err)
	}

	if err := crashWd.Start(ctx); err != nil {
		browserCtx.MustClose()
		return nil, errors.NewContextError("start_crash_watchdog", err)
	}

	// Register initial page with watchdogs
	if err := popupsWd.RegisterPage(ctx, page); err != nil {
		browserCtx.MustClose()
		return nil, errors.NewContextError("register_initial_page_popups", err)
	}

	if err := crashWd.RegisterPage(ctx, page); err != nil {
		browserCtx.MustClose()
		return nil, errors.NewContextError("register_initial_page_crash", err)
	}

	return &Context{
		browser:             browserCtx,
		tabs:                map[string]*tabContext{"tab-1": initialTab},
		activeTabID:         "tab-1",
		tabIDCounter:        1, // Start counter at 1 since we already created tab-1
		popupsWatchdog:      popupsWd,
		permissionsWatchdog: permissionsWd,
		crashWatchdog:       crashWd,
		eventBus:            eventBus,
		windowWidth:         m.config.WindowWidth,
		windowHeight:        m.config.WindowHeight,
	}, nil
}

// Close closes the browser
func (m *Manager) Close(ctx context.Context) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.browser != nil {
		if err := m.browser.Close(); err != nil {
			return errors.NewBrowserError("close", err)
		}
	}
	return nil
}
