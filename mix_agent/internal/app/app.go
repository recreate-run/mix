package app

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"os"

	"mix/internal/analytics"
	browserpkg "mix/internal/browser"
	"mix/internal/config"
	"mix/internal/db"
	"mix/internal/format"
	"mix/internal/history"
	"mix/internal/llm/agent"
	"mix/internal/llm/tools/browser"
	"mix/internal/logging"
	"mix/internal/message"
	storage "mix/internal/mix_storage"
	"mix/internal/notification"
	"mix/internal/permission"
	"mix/internal/session"
)

type App struct {
	Sessions        session.Service
	Messages        message.Service
	History         history.Service
	Permissions     permission.Service
	Notifications   notification.Service
	Analytics       analytics.Service
	StorageConfig   session.Config
	StorageProvider storage.Provider
	BaseURL         string // Base URL for constructing file URLs

	CoderAgent     agent.Service
	TunnelRegistry interface{} // *http.TunnelRegistry (avoid circular import)
}

func New(ctx context.Context, conn *sql.DB) (*App, error) {
	q := db.New(conn)

	// Initialize storage system
	storageConfig := session.DefaultConfig()
	if err := session.Initialize(storageConfig); err != nil {
		return nil, fmt.Errorf("failed to initialize storage system: %w", err)
	}

	// Initialize storage provider
	storageProviderConfig := storage.LoadConfigFromEnv()

	storageProvider, err := storage.NewProvider(storageProviderConfig)
	if err != nil {
		logging.Warn("Failed to initialize storage provider, falling back to local storage", "error", err)
		// Fallback to local storage
		storageProviderConfig = storage.Config{
			Type:     storage.ProviderTypeLocal,
			Endpoint: storageConfig.BasePath,
		}
		storageProvider, err = storage.NewProvider(storageProviderConfig)
		if err != nil {
			return nil, fmt.Errorf("failed to initialize local storage provider: %w", err)
		}
	}

	// Create session service with storage configuration
	sessions := session.NewService(q, storageConfig)

	// Initialize API credentials service with database connection
	if err := config.InitAPICredentials(conn); err != nil {
		logging.Error("Failed to initialize API credentials", "error", err)
		return nil, fmt.Errorf("failed to initialize API credentials: %w", err)
	}

	// Create base message service
	baseMessageService := message.NewService(q)

	files := history.NewService(q, conn)

	// Initialize analytics service with PostHog API key
	posthogAPIKey := "phc_9QLQI8n19fRg1vsqvYRaQVXFRpMRXTGQK4i2DaYqWRU"

	cfg := config.Get()
	analyticsEnabled := cfg.AnalyticsEnabled

	if !analyticsEnabled {
		posthogAPIKey = "" // Empty API key disables analytics
	}
	analyticsService := analytics.NewAnalyticsService(posthogAPIKey)

	// Wrap message service with tracking
	messages := message.NewTrackingService(baseMessageService, analyticsService)

	// Get browser service URL from environment (required for local-browser-service mode)
	browserServiceURL := os.Getenv("BROWSER_SERVICE_URL")

	app := &App{
		Sessions:        sessions,
		Messages:        messages,
		History:         files,
		Permissions:     permission.NewPermissionService(sessions, storageConfig),
		Notifications:   notification.NewService(sessions),
		Analytics:       analyticsService,
		StorageConfig:   storageConfig,
		StorageProvider: storageProvider,
		BaseURL:         cfg.BaseURL,
	}

	// Create MCP manager for this agent
	mcpManager := agent.NewMCPClientManager()

	// Create browser connection manager (for local-browser-service mode)
	var browserConnectionManager interface{}
	if browserServiceURL != "" {
		browserConnectionManager = createBrowserConnectionManager(browserServiceURL)
	}

	// Create browser client factory with connection manager
	browserClientFactory := app.createBrowserClientFactory(browserConnectionManager, browserServiceURL)

	app.CoderAgent, err = agent.NewAgent(
		config.AgentMain,
		app.Sessions,
		app.Messages,
		agent.CoderAgentTools(
			app.Permissions,
			app.Notifications,
			app.Sessions,
			app.Messages,
			app.History,
			mcpManager,
			browserpkg.ModeLocalBrowserService, // Default fallback for legacy sessions without browser mode
			browserServiceURL,
			browserClientFactory,
			browserConnectionManager,
			func() interface{} { return app.TunnelRegistry }, // Closure that looks up current value
			app.BaseURL,
		),
		storageConfig,
		app.Permissions, // Pass permissions for callback executor
	)
	if err != nil {
		logging.Error("Failed to create coder agent", err)
		return nil, err
	}

	return app, nil
}

// createBrowserClientFactory creates a factory function for browser clients
func (app *App) createBrowserClientFactory(connectionManager interface{}, browserServiceURL string) func(sessionID string) (browserpkg.Client, error) {
	return func(sessionID string) (browserpkg.Client, error) {
		// Session's browser mode is used by NewClient - no need to pass default here
		factoryConfig := browserpkg.FactoryConfig{
			Mode:              "", // Will be set from session by NewClient
			BrowserServiceURL: browserServiceURL,
			TunnelRegistry:    app.TunnelRegistry,
			ConnectionManager: connectionManager,
		}

		return browserpkg.NewClient(factoryConfig, sessionID)
	}
}

// createBrowserConnectionManager creates a connection manager for browser-service
func createBrowserConnectionManager(serviceURL string) interface{} {
	return browser.NewConnectionManager(serviceURL)
}

// Removed theme initialization for embedded binary

// RunNonInteractive handles the execution flow when a prompt is provided via CLI flag.
func (a *App) RunNonInteractive(ctx context.Context, prompt, outputFormat, browserMode, cdpURL string, quiet bool) error {
	// Processing message for non-interactive mode
	if !quiet {
		fmt.Println("Processing...")
	}

	titlePrefix := "Non-interactive: "
	title := session.TruncateTitle(titlePrefix + prompt)

	sess, err := a.Sessions.Create(ctx, title, "", "default", session.SessionTypeMain, "", "", "", browserMode, cdpURL)
	if err != nil {
		return fmt.Errorf("failed to create session for non-interactive mode: %w", err)
	}

	done, err := a.CoderAgent.Run(ctx, sess.ID, prompt)
	if err != nil {
		return fmt.Errorf("failed to start agent processing stream: %w", err)
	}

	// Consume all events from the channel (like HTTP handler does)
	var lastEvent agent.AgentEvent
	for event := range done {
		lastEvent = event
		// Print streaming content in text mode (quiet flag only hides "Processing..." spinner)
		if event.Content != "" && outputFormat == format.Text.String() {
			fmt.Print(event.Content)
		}
	}

	// Check for errors after all events are processed
	if lastEvent.Error != nil {
		if errors.Is(lastEvent.Error, context.Canceled) || errors.Is(lastEvent.Error, agent.ErrRequestCancelled) {
			return nil
		}
		return fmt.Errorf("agent processing failed: %w", lastEvent.Error)
	}

	// Print final newline after streaming content in text mode
	if outputFormat == format.Text.String() {
		fmt.Println()
	}

	// For JSON output format, print the final message
	if outputFormat != format.Text.String() {
		content := "No content available"
		if lastEvent.Message.Content().String() != "" {
			content = lastEvent.Message.Content().String()
		}
		fmt.Println(format.FormatOutput(content, outputFormat))
	}

	return nil
}

// Shutdown performs a clean shutdown of the application
func (app *App) Shutdown() {
	if app.CoderAgent != nil {
		app.CoderAgent.Shutdown()
	}

	// Clean up storage provider
	if app.StorageProvider != nil {
		if err := app.StorageProvider.Close(); err != nil {
			logging.Error("Failed to close storage provider", "error", err)
		}
	}

	// Clean up analytics service
	if app.Analytics != nil {
		if err := app.Analytics.Close(); err != nil {
			logging.Error("Failed to close analytics service: %v", err)
		}
	}
}
