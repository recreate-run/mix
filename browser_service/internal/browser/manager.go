package browser

import (
	"context"
	"fmt"
	"sync"

	"github.com/go-rod/rod"
	"github.com/go-rod/rod/lib/launcher"
	"github.com/go-rod/rod/lib/proto"
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
		cfg.WindowWidth = 1280
	}
	if cfg.WindowHeight == 0 {
		cfg.WindowHeight = 720
	}

	// Configure browser launcher
	l := launcher.New().
		Headless(cfg.Headless).
		Devtools(false).
		Set("ignore-certificate-errors").      // Ignore SSL certificate errors for testing
		Set("allow-insecure-localhost").       // Allow insecure localhost connections
		Set("disable-web-security")            // Disable web security for testing

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

	// Create initial tab with ID "tab-1"
	initialTab := &tabContext{
		id:           "tab-1",
		page:         page,
		elements:     make([]elementInfo, 0),
		downloads:    make([]protocol.Download, 0),
		downloadChan: make(chan protocol.Download, 10),
	}

	return &Context{
		browser:      browserCtx,
		tabs:         map[string]*tabContext{"tab-1": initialTab},
		activeTabID:  "tab-1",
		tabIDCounter: 1, // Start counter at 1 since we already created tab-1
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
