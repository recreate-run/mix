package app

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"mix/internal/analytics"
	"mix/internal/config"
	"mix/internal/db"
	"mix/internal/format"
	"mix/internal/history"
	"mix/internal/llm/agent"
	"mix/internal/logging"
	"mix/internal/message"
	"mix/internal/permission"
	"mix/internal/session"
	"mix/internal/video"
)

type App struct {
	Sessions    session.Service
	Messages    message.Service
	History     history.Service
	Permissions permission.Service
	Analytics   analytics.Service
	Video       *video.ExportService

	CoderAgent agent.Service

	// Current session tracking for API session selection
	currentSessionID string
}

func New(ctx context.Context, conn *sql.DB) (*App, error) {
	q := db.New(conn)
	sessions := session.NewService(q)

	// Create base message service
	baseMessageService := message.NewService(q)

	files := history.NewService(q, conn)

	// Initialize analytics service with PostHog API key
	posthogAPIKey := "phc_9QLQI8n19fRg1vsqvYRaQVXFRpMRXTGQK4i2DaYqWRU"

	cfg := config.Get()
	analyticsEnabled := cfg.AnalyticsEnabled

	if !analyticsEnabled {
		logging.Info("PostHog analytics disabled: analyticsEnabled config set to false")
		posthogAPIKey = "" // Empty API key disables analytics
	} else {
		logging.Info("PostHog analytics enabled")
	}
	analyticsService := analytics.NewAnalyticsService(posthogAPIKey)

	// Initialize video export service
	videoService, err := video.NewExportService()
	if err != nil {
		logging.Error("Failed to initialize video export service", "error", err)
		return nil, fmt.Errorf("failed to initialize video export service: %w", err)
	}

	// Wrap message service with tracking
	messages := message.NewTrackingService(baseMessageService, analyticsService)

	app := &App{
		Sessions:    sessions,
		Messages:    messages,
		History:     files,
		Permissions: permission.NewPermissionService(sessions),
		Analytics:   analyticsService,
		Video:       videoService,
	}

	// Create MCP manager for this agent
	mcpManager := agent.NewMCPClientManager()

	app.CoderAgent, err = agent.NewAgent(
		config.AgentMain,
		app.Sessions,
		app.Messages,
		agent.CoderAgentTools(
			app.Permissions,
			app.Sessions,
			app.Messages,
			app.History,
			mcpManager,
		),
	)
	if err != nil {
		logging.Error("Failed to create coder agent", err)
		return nil, err
	}

	return app, nil
}

// Removed theme initialization for embedded binary

// RunNonInteractive handles the execution flow when a prompt is provided via CLI flag.
func (a *App) RunNonInteractive(ctx context.Context, prompt string, outputFormat string, quiet bool) error {
	logging.Info("Running in non-interactive mode")

	// Processing message for non-interactive mode
	if !quiet {
		fmt.Println("Processing...")
	}

	const maxPromptLengthForTitle = 100
	titlePrefix := "Non-interactive: "
	var titleSuffix string

	if len(prompt) > maxPromptLengthForTitle {
		titleSuffix = prompt[:maxPromptLengthForTitle] + "..."
	} else {
		titleSuffix = prompt
	}
	title := titlePrefix + titleSuffix

	launchDir, err := config.LaunchDirectory()
	if err != nil {
		return fmt.Errorf("failed to get launch directory: %w", err)
	}

	sess, err := a.Sessions.Create(ctx, title, launchDir)
	if err != nil {
		return fmt.Errorf("failed to create session for non-interactive mode: %w", err)
	}
	logging.Info("Created session for non-interactive run", "session_id", sess.ID)

	done, err := a.CoderAgent.Run(ctx, sess.ID, prompt)
	if err != nil {
		return fmt.Errorf("failed to start agent processing stream: %w", err)
	}

	result := <-done
	if result.Error != nil {
		if errors.Is(result.Error, context.Canceled) || errors.Is(result.Error, agent.ErrRequestCancelled) {
			logging.Info("Agent processing cancelled", "session_id", sess.ID)
			return nil
		}
		return fmt.Errorf("agent processing failed: %w", result.Error)
	}

	// Get the text content from the response
	content := "No content available"
	if result.Message.Content().String() != "" {
		content = result.Message.Content().String()
	}

	fmt.Println(format.FormatOutput(content, outputFormat))

	logging.Info("Non-interactive run completed", "session_id", sess.ID)

	return nil
}

// SetCurrentSession sets the current session ID for API operations
func (a *App) SetCurrentSession(sessionID string) error {
	if sessionID == "" {
		a.currentSessionID = ""
		return nil
	}

	// Verify session exists
	_, err := a.Sessions.Get(context.Background(), sessionID)
	if err != nil {
		return fmt.Errorf("session not found: %w", err)
	}

	a.currentSessionID = sessionID
	return nil
}

// GetCurrentSession returns the currently selected session, or nil if none selected
func (a *App) GetCurrentSession(ctx context.Context) (*session.Session, error) {
	if a.currentSessionID == "" {
		return nil, nil
	}

	sess, err := a.Sessions.Get(ctx, a.currentSessionID)
	if err != nil {
		// Reset current session if it no longer exists
		a.currentSessionID = ""
		return nil, fmt.Errorf("current session no longer exists: %w", err)
	}

	return &sess, nil
}

// GetCurrentSessionID returns the current session ID (may be empty)
func (a *App) GetCurrentSessionID() string {
	return a.currentSessionID
}

// Shutdown performs a clean shutdown of the application
func (app *App) Shutdown() {
	if app.CoderAgent != nil {
		app.CoderAgent.Shutdown()
	}

	// Clean up analytics service
	if app.Analytics != nil {
		if err := app.Analytics.Close(); err != nil {
			logging.Error("Failed to close analytics service: %v", err)
		}
	}

	logging.Info("Application shutdown completed")
}
