package preferences

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"sync"

	"mix/internal/db"
	"mix/internal/llm/models"
	"mix/internal/logging"
)

// Agent types to avoid import cycle
type AgentName string

const (
	AgentMain AgentName = "main"
	AgentSub  AgentName = "sub"
)

type Agent struct {
	Model           models.ModelID `json:"model"`
	MaxTokens       int64          `json:"maxTokens"`
	ReasoningEffort string         `json:"reasoningEffort"`
}

// UserPreferencesService handles user preferences from database
type UserPreferencesService struct {
	queries          db.Querier
	preferencesCache sync.Map // Caches user preferences to avoid database hits
}

// NewUserPreferencesService creates a new user preferences service
func NewUserPreferencesService(database *sql.DB) *UserPreferencesService {
	return NewUserPreferencesServiceWithQuerierAndPreload(db.New(database), true)
}

// NewUserPreferencesServiceWithQuerierAndPreload creates a service with custom querier and preload control
func NewUserPreferencesServiceWithQuerierAndPreload(querier db.Querier, enablePreload bool) *UserPreferencesService {
	service := &UserPreferencesService{
		queries:          querier,
		preferencesCache: sync.Map{},
	}

	// Preload preferences in the background only if enabled
	if enablePreload {
		go service.PreloadPreferences(context.Background())
	}

	return service
}

// GetUserPreferences gets user preferences from database without creating them
func (ups *UserPreferencesService) GetUserPreferences(ctx context.Context) (*db.UserPreference, error) {
	// Check cache first for fast access
	if cachedValue, found := ups.preferencesCache.Load("default_user"); found {
		return cachedValue.(*db.UserPreference), nil
	}

	// Try to get existing preferences from database
	prefs, err := ups.queries.GetUserPreferences(ctx)
	if err == nil {
		// Store in cache for future fast access
		ups.preferencesCache.Store("default_user", &prefs)
		return &prefs, nil
	}

	// Return error as is - including sql.ErrNoRows if preferences don't exist
	return nil, err
}

// CreateDefaultUserPreferences creates default user preferences
func (ups *UserPreferencesService) CreateDefaultUserPreferences(ctx context.Context) (*db.UserPreference, error) {
	// Creating default user preferences
	defaultPrefs := db.CreateUserPreferencesParams{
		PreferredProvider:        sql.NullString{String: "anthropic", Valid: true},
		MainAgentModel:           sql.NullString{String: "claude-sonnet-4-5", Valid: true},
		MainAgentMaxTokens:       sql.NullInt64{Int64: 4096, Valid: true},
		MainAgentReasoningEffort: sql.NullString{String: "medium", Valid: true},
		SubAgentModel:            sql.NullString{String: "claude-sonnet-4-5", Valid: true},
		SubAgentMaxTokens:        sql.NullInt64{Int64: 2048, Valid: true},
		SubAgentReasoningEffort:  sql.NullString{String: "medium", Valid: true},
	}

	createdPrefs, err := ups.queries.CreateUserPreferences(ctx, defaultPrefs)
	if err != nil {
		return nil, fmt.Errorf("failed to create default user preferences: %w", err)
	}

	// Store in cache for future fast access
	ups.preferencesCache.Store("default_user", &createdPrefs)
	return &createdPrefs, nil
}

// GetOrCreateUserPreferences gets user preferences from database or creates default ones
func (ups *UserPreferencesService) GetOrCreateUserPreferences(ctx context.Context) (*db.UserPreference, error) {
	// Try to get existing preferences
	prefs, err := ups.GetUserPreferences(ctx)
	if err == nil {
		return prefs, nil
	}

	// If not found, create default preferences
	if errors.Is(err, sql.ErrNoRows) {
		return ups.CreateDefaultUserPreferences(ctx)
	}

	return nil, fmt.Errorf("failed to get user preferences: %w", err)
}

// UpdateMainAgentPreferences updates the main agent model preferences
func (ups *UserPreferencesService) UpdateMainAgentPreferences(ctx context.Context, modelID models.ModelID, maxTokens int64, reasoningEffort string) error {
	params := db.UpdateMainAgentModelParams{
		MainAgentModel:           sql.NullString{String: string(modelID), Valid: true},
		MainAgentMaxTokens:       sql.NullInt64{Int64: maxTokens, Valid: true},
		MainAgentReasoningEffort: sql.NullString{String: reasoningEffort, Valid: reasoningEffort != ""},
	}

	_, err := ups.queries.UpdateMainAgentModel(ctx, params)
	if err != nil {
		return fmt.Errorf("failed to update main agent preferences: %w", err)
	}

	// Invalidate cache to force refresh on next read
	ups.preferencesCache.Delete("default_user")

	return nil
}

// UpdateSubAgentPreferences updates the sub agent model preferences
func (ups *UserPreferencesService) UpdateSubAgentPreferences(ctx context.Context, modelID models.ModelID, maxTokens int64, reasoningEffort string) error {
	params := db.UpdateSubAgentModelParams{
		SubAgentModel:           sql.NullString{String: string(modelID), Valid: true},
		SubAgentMaxTokens:       sql.NullInt64{Int64: maxTokens, Valid: true},
		SubAgentReasoningEffort: sql.NullString{String: reasoningEffort, Valid: reasoningEffort != ""},
	}

	_, err := ups.queries.UpdateSubAgentModel(ctx, params)
	if err != nil {
		return fmt.Errorf("failed to update sub agent preferences: %w", err)
	}

	// Invalidate cache to force refresh on next read
	ups.preferencesCache.Delete("default_user")

	return nil
}

// UpdatePreferredProvider updates the user's preferred provider
func (ups *UserPreferencesService) UpdatePreferredProvider(ctx context.Context, provider models.ModelProvider) error {
	_, err := ups.queries.UpdateUserPreferredProvider(ctx, sql.NullString{String: string(provider), Valid: true})
	if err != nil {
		return fmt.Errorf("failed to update preferred provider: %w", err)
	}

	// Invalidate cache to force refresh on next read
	ups.preferencesCache.Delete("default_user")

	return nil
}

// GetAgentConfig converts database preferences to Agent config format
func (ups *UserPreferencesService) GetAgentConfig(ctx context.Context, agentName AgentName) (Agent, error) {
	// Try to get existing preferences
	prefs, err := ups.GetUserPreferences(ctx) // This now checks cache first
	if err != nil {
		// If preferences don't exist, create default ones
		if errors.Is(err, sql.ErrNoRows) {
			prefs, err = ups.CreateDefaultUserPreferences(ctx) // This will cache the result
			if err != nil {
				return Agent{}, fmt.Errorf("failed to create default user preferences: %w", err)
			}
		} else {
			return Agent{}, err
		}
	}

	switch agentName {
	case AgentMain:
		return Agent{
			Model:           models.ModelID(prefs.MainAgentModel.String),
			MaxTokens:       prefs.MainAgentMaxTokens.Int64,
			ReasoningEffort: prefs.MainAgentReasoningEffort.String,
		}, nil
	case AgentSub:
		return Agent{
			Model:           models.ModelID(prefs.SubAgentModel.String),
			MaxTokens:       prefs.SubAgentMaxTokens.Int64,
			ReasoningEffort: prefs.SubAgentReasoningEffort.String,
		}, nil
	default:
		return Agent{}, fmt.Errorf("unknown agent name: %s", agentName)
	}
}

// GetPreferredProvider gets the user's preferred provider from database
func (ups *UserPreferencesService) GetPreferredProvider(ctx context.Context) (models.ModelProvider, error) {
	// Try to get existing preferences
	prefs, err := ups.GetUserPreferences(ctx) // This now checks cache first
	if err != nil {
		// If preferences don't exist, create default ones
		if errors.Is(err, sql.ErrNoRows) {
			prefs, err = ups.CreateDefaultUserPreferences(ctx) // This will cache the result
			if err != nil {
				return "", fmt.Errorf("failed to create default user preferences: %w", err)
			}
		} else {
			return "", err
		}
	}

	if !prefs.PreferredProvider.Valid || prefs.PreferredProvider.String == "" {
		return models.ProviderAnthropic, nil // Default to Anthropic
	}

	return models.ModelProvider(prefs.PreferredProvider.String), nil
}

// PreloadPreferences loads all preferences into the cache to avoid database hits
func (ups *UserPreferencesService) PreloadPreferences(ctx context.Context) {
	// Try to get existing preferences
	prefs, err := ups.queries.GetUserPreferences(ctx)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			logging.Debug("No preferences found in database during preload")
			// Don't create default preferences here, let them be created on first access
			return
		}
		logging.Error("Failed to preload user preferences", "error", err)
		return
	}

	// Store in cache
	ups.preferencesCache.Store("default_user", &prefs)
}

// ClearCache removes all entries from the preferences cache
func (ups *UserPreferencesService) ClearCache() {
	// Create a new empty map to replace the existing one
	ups.preferencesCache = sync.Map{}
	logging.Debug("User preferences cache cleared")
}
