package preferences

import (
	"context"

	"mix/internal/db"
	"mix/internal/llm/models"
)

// Service defines the interface for user preferences operations
type Service interface {
	// GetUserPreferences retrieves user preferences from cache or database
	GetUserPreferences(ctx context.Context) (*db.UserPreference, error)

	// CreateDefaultUserPreferences creates default user preferences in the database
	CreateDefaultUserPreferences(ctx context.Context) (*db.UserPreference, error)

	// GetOrCreateUserPreferences gets existing preferences or creates defaults
	GetOrCreateUserPreferences(ctx context.Context) (*db.UserPreference, error)

	// UpdateMainAgentPreferences updates main agent preferences
	UpdateMainAgentPreferences(ctx context.Context, modelID models.ModelID, maxTokens int64, reasoningEffort string) error

	// UpdateSubAgentPreferences updates sub-agent preferences
	UpdateSubAgentPreferences(ctx context.Context, modelID models.ModelID, maxTokens int64, reasoningEffort string) error

	// UpdatePreferredProvider updates the preferred provider
	UpdatePreferredProvider(ctx context.Context, provider models.ModelProvider) error

	// GetAgentConfig retrieves agent configuration by name
	GetAgentConfig(ctx context.Context, agentName AgentName) (Agent, error)

	// GetPreferredProvider gets the user's preferred provider
	GetPreferredProvider(ctx context.Context) (models.ModelProvider, error)

	// PreloadPreferences preloads preferences into cache
	PreloadPreferences(ctx context.Context)

	// ClearCache clears the preferences cache
	ClearCache()
}

// Ensure UserPreferencesService implements Service interface
var _ Service = (*UserPreferencesService)(nil)