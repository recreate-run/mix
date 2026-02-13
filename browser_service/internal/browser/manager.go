package browser

import (
	"context"
	"sync"

	"github.com/go-rod/rod"
	"github.com/go-rod/rod/lib/launcher"
	"github.com/go-rod/rod/lib/proto"
	"github.com/sarathmenon/browser-service/internal/errors"
)

// Manager manages the browser instance and contexts
type Manager struct {
	browser  *rod.Browser
	headless bool
	mu       sync.Mutex
}

// NewManager creates a new browser manager
func NewManager(ctx context.Context, headless bool) (*Manager, error) {
	// Configure browser launcher
	l := launcher.New().
		Headless(headless).
		Devtools(false).
		Set("ignore-certificate-errors").      // Ignore SSL certificate errors for testing
		Set("allow-insecure-localhost").       // Allow insecure localhost connections
		Set("disable-web-security")            // Disable web security for testing

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
		browser:  browser,
		headless: headless,
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
		id:       "tab-1",
		page:     page,
		elements: make([]elementInfo, 0),
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
