package preferences

import (
	"context"
	"database/sql"
	"fmt"

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
	queries *db.Queries
}

// NewUserPreferencesService creates a new user preferences service
func NewUserPreferencesService(database *sql.DB) *UserPreferencesService {
	return &UserPreferencesService{
		queries: db.New(database),
	}
}

// GetOrCreateUserPreferences gets user preferences from database or creates default ones
func (ups *UserPreferencesService) GetOrCreateUserPreferences(ctx context.Context) (*db.UserPreference, error) {
	// Try to get existing preferences
	prefs, err := ups.queries.GetUserPreferences(ctx)
	if err == nil {
		return &prefs, nil
	}
	
	// If not found, create default preferences
	if err == sql.ErrNoRows {
		logging.Info("Creating default user preferences")
		defaultPrefs := db.CreateUserPreferencesParams{
			PreferredProvider:       sql.NullString{String: "anthropic", Valid: true},
			MainAgentModel:          sql.NullString{String: "claude-4-sonnet", Valid: true},
			MainAgentMaxTokens:      sql.NullInt64{Int64: 4096, Valid: true},
			MainAgentReasoningEffort: sql.NullString{String: "", Valid: false},
			SubAgentModel:           sql.NullString{String: "claude-4-sonnet", Valid: true},
			SubAgentMaxTokens:       sql.NullInt64{Int64: 2048, Valid: true},
			SubAgentReasoningEffort: sql.NullString{String: "", Valid: false},
		}
		
		createdPrefs, createErr := ups.queries.CreateUserPreferences(ctx, defaultPrefs)
		if createErr != nil {
			return nil, fmt.Errorf("failed to create default user preferences: %w", createErr)
		}
		return &createdPrefs, nil
	}
	
	return nil, fmt.Errorf("failed to get user preferences: %w", err)
}

// UpdateMainAgentPreferences updates the main agent model preferences
func (ups *UserPreferencesService) UpdateMainAgentPreferences(ctx context.Context, modelID models.ModelID, maxTokens int64, reasoningEffort string) error {
	params := db.UpdateMainAgentModelParams{
		MainAgentModel:          sql.NullString{String: string(modelID), Valid: true},
		MainAgentMaxTokens:      sql.NullInt64{Int64: maxTokens, Valid: true},
		MainAgentReasoningEffort: sql.NullString{String: reasoningEffort, Valid: reasoningEffort != ""},
	}
	
	_, err := ups.queries.UpdateMainAgentModel(ctx, params)
	if err != nil {
		return fmt.Errorf("failed to update main agent preferences: %w", err)
	}
	
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
	
	return nil
}

// UpdatePreferredProvider updates the user's preferred provider
func (ups *UserPreferencesService) UpdatePreferredProvider(ctx context.Context, provider models.ModelProvider) error {
	_, err := ups.queries.UpdateUserPreferredProvider(ctx, sql.NullString{String: string(provider), Valid: true})
	if err != nil {
		return fmt.Errorf("failed to update preferred provider: %w", err)
	}
	
	return nil
}

// GetAgentConfig converts database preferences to Agent config format
func (ups *UserPreferencesService) GetAgentConfig(ctx context.Context, agentName AgentName) (Agent, error) {
	prefs, err := ups.GetOrCreateUserPreferences(ctx)
	if err != nil {
		return Agent{}, err
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
	prefs, err := ups.GetOrCreateUserPreferences(ctx)
	if err != nil {
		return "", err
	}
	
	if !prefs.PreferredProvider.Valid || prefs.PreferredProvider.String == "" {
		return models.ProviderAnthropic, nil // Default to Anthropic
	}
	
	return models.ModelProvider(prefs.PreferredProvider.String), nil
}

// MigrateFromConfig migrates agent configuration from .mix.json to database
func (ups *UserPreferencesService) MigrateFromConfig(ctx context.Context, agents map[AgentName]Agent, preferredProvider models.ModelProvider) error {
	// Check if preferences already exist
	_, err := ups.queries.GetUserPreferences(ctx)
	if err == nil {
		logging.Info("User preferences already exist in database, skipping migration")
		return nil
	}
	
	if err != sql.ErrNoRows {
		return fmt.Errorf("failed to check existing preferences: %w", err)
	}
	
	// Get agent configurations
	mainAgent, hasMain := agents[AgentMain]
	subAgent, hasSub := agents[AgentSub]
	
	// Use defaults if not present
	if !hasMain {
		mainAgent = Agent{
			Model:           "claude-4-sonnet",
			MaxTokens:       4096,
			ReasoningEffort: "",
		}
	}
	
	if !hasSub {
		subAgent = Agent{
			Model:           "claude-4-sonnet", 
			MaxTokens:       2048,
			ReasoningEffort: "",
		}
	}
	
	// Create preferences from config
	params := db.CreateUserPreferencesParams{
		PreferredProvider:       sql.NullString{String: string(preferredProvider), Valid: preferredProvider != ""},
		MainAgentModel:          sql.NullString{String: string(mainAgent.Model), Valid: true},
		MainAgentMaxTokens:      sql.NullInt64{Int64: mainAgent.MaxTokens, Valid: true},
		MainAgentReasoningEffort: sql.NullString{String: mainAgent.ReasoningEffort, Valid: mainAgent.ReasoningEffort != ""},
		SubAgentModel:           sql.NullString{String: string(subAgent.Model), Valid: true},
		SubAgentMaxTokens:       sql.NullInt64{Int64: subAgent.MaxTokens, Valid: true},
		SubAgentReasoningEffort: sql.NullString{String: subAgent.ReasoningEffort, Valid: subAgent.ReasoningEffort != ""},
	}
	
	_, err = ups.queries.CreateUserPreferences(ctx, params)
	if err != nil {
		return fmt.Errorf("failed to migrate preferences to database: %w", err)
	}
	
	logging.Info("Successfully migrated user preferences from .mix.json to database")
	return nil
}