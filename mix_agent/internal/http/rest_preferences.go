package http

import (
	"database/sql"
	"encoding/json"
	"net/http"

	"mix/internal/app"
	"mix/internal/config"
	"mix/internal/db"
	"mix/internal/llm/models"
	"mix/internal/logging"
)

// UserPreferencesResponse represents the API response for user preferences
type UserPreferencesResponse struct {
	PreferredProvider        models.ModelProvider   `json:"preferred_provider"`
	MainAgentModel           models.ModelID         `json:"main_agent_model"`
	MainAgentMaxTokens       int64                  `json:"main_agent_max_tokens"`
	MainAgentReasoningEffort models.ReasoningEffort `json:"main_agent_reasoning_effort"`
	SubAgentModel            models.ModelID         `json:"sub_agent_model"`
	SubAgentMaxTokens        int64                  `json:"sub_agent_max_tokens"`
	SubAgentReasoningEffort  models.ReasoningEffort `json:"sub_agent_reasoning_effort"`
	CreatedAt                int64                  `json:"created_at"`
	UpdatedAt                int64                  `json:"updated_at"`
}

// UpdatePreferencesRequest represents the API request for updating preferences
type UpdatePreferencesRequest struct {
	PreferredProvider        *string `json:"preferred_provider,omitempty"`
	MainAgentModel           *string `json:"main_agent_model,omitempty"`
	MainAgentMaxTokens       *int64  `json:"main_agent_max_tokens,omitempty"`
	MainAgentReasoningEffort *string `json:"main_agent_reasoning_effort,omitempty"`
	SubAgentModel            *string `json:"sub_agent_model,omitempty"`
	SubAgentMaxTokens        *int64  `json:"sub_agent_max_tokens,omitempty"`
	SubAgentReasoningEffort  *string `json:"sub_agent_reasoning_effort,omitempty"`
}

// PreferencesHandler handles REST endpoints for user preferences
type PreferencesHandler struct {
	app *app.App
}

// NewPreferencesHandler creates a new preferences handler
func NewPreferencesHandler(app *app.App) *PreferencesHandler {
	return &PreferencesHandler{
		app: app,
	}
}

// HandleGetPreferences handles GET /api/preferences
func (h *PreferencesHandler) HandleGetPreferences(w http.ResponseWriter, r *http.Request) {
	// Get request context
	ctx := r.Context()

	// Get user preferences service
	userPrefs := config.GetUserPreferences()
	if userPrefs == nil {
		WriteErrorResponse(w, http.StatusInternalServerError, "user preferences service not available", "PREFERENCES_SERVICE_UNAVAILABLE")
		return
	}

	// Only get preferences - don't create them if they don't exist
	prefs, err := userPrefs.GetUserPreferences(ctx)
	if err != nil {
		// If preferences don't exist, return an empty response with available providers
		if err == sql.ErrNoRows {
			WriteJSONResponse(w, http.StatusOK, map[string]interface{}{
				"preferences":         nil,
				"available_providers": models.GetProviders(),
			})
			return
		}

		// For any other error, log it and return an error response
		logging.Error("Failed to get user preferences", "error", err)
		WriteErrorResponse(w, http.StatusInternalServerError, "failed to get preferences", "DATABASE_ERROR")
		return
	}

	// Convert database model to response model
	response := UserPreferencesResponse{
		PreferredProvider:        models.ModelProvider(getStringValue(prefs.PreferredProvider)),
		MainAgentModel:           models.ModelID(getStringValue(prefs.MainAgentModel)),
		MainAgentMaxTokens:       getInt64Value(prefs.MainAgentMaxTokens),
		MainAgentReasoningEffort: models.ReasoningEffort(getStringValue(prefs.MainAgentReasoningEffort)),
		SubAgentModel:            models.ModelID(getStringValue(prefs.SubAgentModel)),
		SubAgentMaxTokens:        getInt64Value(prefs.SubAgentMaxTokens),
		SubAgentReasoningEffort:  models.ReasoningEffort(getStringValue(prefs.SubAgentReasoningEffort)),
		CreatedAt:                prefs.CreatedAt,
		UpdatedAt:                prefs.UpdatedAt,
	}

	// Include additional information about available providers
	responseWithProviders := map[string]interface{}{
		"preferences":         response,
		"available_providers": models.GetProviders(),
	}

	WriteJSONResponse(w, http.StatusOK, responseWithProviders)
}

// HandleUpdatePreferences handles POST /api/preferences
func (h *PreferencesHandler) HandleUpdatePreferences(w http.ResponseWriter, r *http.Request) {
	// Get main agent service to clear cached providers after update
	coderAgent := h.app.CoderAgent
	userPrefs := config.GetUserPreferences()
	if userPrefs == nil {
		WriteErrorResponse(w, http.StatusInternalServerError, "user preferences service not available", "PREFERENCES_SERVICE_UNAVAILABLE")
		return
	}

	var request UpdatePreferencesRequest
	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		WriteErrorResponse(w, http.StatusBadRequest, "invalid JSON request", "INVALID_JSON")
		return
	}

	ctx := r.Context()

	// Try to get existing preferences
	currentPrefs, err := userPrefs.GetUserPreferences(ctx)
	if err != nil {
		// If preferences don't exist, create default ones
		if err == sql.ErrNoRows {
			currentPrefs, err = userPrefs.CreateDefaultUserPreferences(ctx)
			if err != nil {
				logging.Error("Failed to create default user preferences", "error", err)
				WriteErrorResponse(w, http.StatusInternalServerError, "failed to create default preferences", "DATABASE_ERROR")
				return
			}
		} else {
			// For any other error, log it and return an error response
			logging.Error("Failed to get current user preferences", "error", err)
			WriteErrorResponse(w, http.StatusInternalServerError, "failed to get current preferences", "DATABASE_ERROR")
			return
		}
	}

	// Build update parameters using current values as defaults
	updateParams := db.UpdateUserPreferencesParams{
		PreferredProvider:        currentPrefs.PreferredProvider,
		MainAgentModel:           currentPrefs.MainAgentModel,
		MainAgentMaxTokens:       currentPrefs.MainAgentMaxTokens,
		MainAgentReasoningEffort: currentPrefs.MainAgentReasoningEffort,
		SubAgentModel:            currentPrefs.SubAgentModel,
		SubAgentMaxTokens:        currentPrefs.SubAgentMaxTokens,
		SubAgentReasoningEffort:  currentPrefs.SubAgentReasoningEffort,
	}

	// Update only provided fields
	if request.PreferredProvider != nil {
		// Validate provider
		if *request.PreferredProvider != "" {
			if _, exists := models.GetProviders()[models.ModelProvider(*request.PreferredProvider)]; !exists {
				WriteErrorResponse(w, http.StatusBadRequest, "invalid provider", "INVALID_PROVIDER")
				return
			}
		}
		updateParams.PreferredProvider = sql.NullString{String: *request.PreferredProvider, Valid: *request.PreferredProvider != ""}
	}

	if request.MainAgentModel != nil {
		// Validate model
		if *request.MainAgentModel != "" {
			if _, exists := models.SupportedModels[models.ModelID(*request.MainAgentModel)]; !exists {
				WriteErrorResponse(w, http.StatusBadRequest, "invalid main agent model", "INVALID_MODEL")
				return
			}
		}
		updateParams.MainAgentModel = sql.NullString{String: *request.MainAgentModel, Valid: *request.MainAgentModel != ""}
	}

	if request.MainAgentMaxTokens != nil {
		if *request.MainAgentMaxTokens <= 0 {
			WriteErrorResponse(w, http.StatusBadRequest, "main agent max tokens must be positive", "INVALID_TOKEN_COUNT")
			return
		}
		updateParams.MainAgentMaxTokens = sql.NullInt64{Int64: *request.MainAgentMaxTokens, Valid: true}
	}

	if request.MainAgentReasoningEffort != nil {
		// Validate reasoning effort - allow empty string or valid values
		if *request.MainAgentReasoningEffort != "" {
			validEfforts := map[string]bool{
				"low":    true,
				"medium": true,
				"high":   true,
			}
			if !validEfforts[*request.MainAgentReasoningEffort] {
				WriteErrorResponse(w, http.StatusBadRequest, "invalid main agent reasoning effort: must be 'low', 'medium', 'high', or empty", "INVALID_REASONING_EFFORT")
				return
			}
		}
		updateParams.MainAgentReasoningEffort = sql.NullString{String: *request.MainAgentReasoningEffort, Valid: *request.MainAgentReasoningEffort != ""}
	}

	if request.SubAgentModel != nil {
		// Validate model
		if *request.SubAgentModel != "" {
			if _, exists := models.SupportedModels[models.ModelID(*request.SubAgentModel)]; !exists {
				WriteErrorResponse(w, http.StatusBadRequest, "invalid sub agent model", "INVALID_MODEL")
				return
			}
		}
		updateParams.SubAgentModel = sql.NullString{String: *request.SubAgentModel, Valid: *request.SubAgentModel != ""}
	}

	if request.SubAgentMaxTokens != nil {
		if *request.SubAgentMaxTokens <= 0 {
			WriteErrorResponse(w, http.StatusBadRequest, "sub agent max tokens must be positive", "INVALID_TOKEN_COUNT")
			return
		}
		updateParams.SubAgentMaxTokens = sql.NullInt64{Int64: *request.SubAgentMaxTokens, Valid: true}
	}

	if request.SubAgentReasoningEffort != nil {
		// Validate reasoning effort - allow empty string or valid values
		if *request.SubAgentReasoningEffort != "" {
			validEfforts := map[string]bool{
				"low":    true,
				"medium": true,
				"high":   true,
			}
			if !validEfforts[*request.SubAgentReasoningEffort] {
				WriteErrorResponse(w, http.StatusBadRequest, "invalid sub agent reasoning effort: must be 'low', 'medium', 'high', or empty", "INVALID_REASONING_EFFORT")
				return
			}
		}
		updateParams.SubAgentReasoningEffort = sql.NullString{String: *request.SubAgentReasoningEffort, Valid: *request.SubAgentReasoningEffort != ""}
	}

	// For now, use simplified update approach via user preferences service
	// Handle model updates through the service
	if request.MainAgentModel != nil {
		modelID := models.ModelID(*request.MainAgentModel)
		maxTokens := int64(4096) // default
		if request.MainAgentMaxTokens != nil {
			maxTokens = *request.MainAgentMaxTokens
		}
		reasoningEffort := ""
		if request.MainAgentReasoningEffort != nil {
			reasoningEffort = *request.MainAgentReasoningEffort
		}
		err = userPrefs.UpdateMainAgentPreferences(ctx, modelID, maxTokens, reasoningEffort)
		if err != nil {
			logging.Error("Failed to update main agent preferences", "error", err)
			WriteErrorResponse(w, http.StatusInternalServerError, "failed to update main agent preferences", "DATABASE_ERROR")
			return
		}
	}

	if request.SubAgentModel != nil {
		modelID := models.ModelID(*request.SubAgentModel)
		maxTokens := int64(2048) // default
		if request.SubAgentMaxTokens != nil {
			maxTokens = *request.SubAgentMaxTokens
		}
		reasoningEffort := ""
		if request.SubAgentReasoningEffort != nil {
			reasoningEffort = *request.SubAgentReasoningEffort
		}
		err = userPrefs.UpdateSubAgentPreferences(ctx, modelID, maxTokens, reasoningEffort)
		if err != nil {
			logging.Error("Failed to update sub agent preferences", "error", err)
			WriteErrorResponse(w, http.StatusInternalServerError, "failed to update sub agent preferences", "DATABASE_ERROR")
			return
		}
	}

	if request.PreferredProvider != nil {
		err = userPrefs.UpdatePreferredProvider(ctx, models.ModelProvider(*request.PreferredProvider))
		if err != nil {
			logging.Error("Failed to update preferred provider", "error", err)
			WriteErrorResponse(w, http.StatusInternalServerError, "failed to update preferred provider", "DATABASE_ERROR")
			return
		}
	}

	// Get updated preferences to return
	updatedPrefs, err := userPrefs.GetOrCreateUserPreferences(ctx)
	if err != nil {
		logging.Error("Failed to update user preferences", "error", err)
		WriteErrorResponse(w, http.StatusInternalServerError, "failed to update preferences", "DATABASE_ERROR")
		return
	}

	// Track preferences update
	if h.app.Analytics != nil {
		fieldsChanged := []string{}
		updates := make(map[string]interface{})

		if request.PreferredProvider != nil {
			fieldsChanged = append(fieldsChanged, "preferred_provider")
			updates["preferred_provider"] = *request.PreferredProvider
		}
		if request.MainAgentModel != nil {
			fieldsChanged = append(fieldsChanged, "main_agent_model")
			updates["main_agent_model"] = *request.MainAgentModel
		}
		if request.MainAgentMaxTokens != nil {
			fieldsChanged = append(fieldsChanged, "main_agent_max_tokens")
			updates["main_agent_max_tokens"] = *request.MainAgentMaxTokens
		}
		if request.MainAgentReasoningEffort != nil {
			fieldsChanged = append(fieldsChanged, "main_agent_reasoning_effort")
			updates["main_agent_reasoning_effort"] = *request.MainAgentReasoningEffort
		}
		if request.SubAgentModel != nil {
			fieldsChanged = append(fieldsChanged, "sub_agent_model")
			updates["sub_agent_model"] = *request.SubAgentModel
		}
		if request.SubAgentMaxTokens != nil {
			fieldsChanged = append(fieldsChanged, "sub_agent_max_tokens")
			updates["sub_agent_max_tokens"] = *request.SubAgentMaxTokens
		}
		if request.SubAgentReasoningEffort != nil {
			fieldsChanged = append(fieldsChanged, "sub_agent_reasoning_effort")
			updates["sub_agent_reasoning_effort"] = *request.SubAgentReasoningEffort
		}

		if len(fieldsChanged) > 0 {
			_ = h.app.Analytics.TrackPreferencesUpdated(ctx, fieldsChanged, updates)
		}
	}

	// Clear all cached session providers to ensure new sessions use updated preferences
	if coderAgent != nil {
		coderAgent.ClearAllSessionProviders()
	} else {
		logging.Warn("Could not clear session provider cache: coderAgent is nil")
	}

	response := UserPreferencesResponse{
		PreferredProvider:        models.ModelProvider(getStringValue(updatedPrefs.PreferredProvider)),
		MainAgentModel:           models.ModelID(getStringValue(updatedPrefs.MainAgentModel)),
		MainAgentMaxTokens:       getInt64Value(updatedPrefs.MainAgentMaxTokens),
		MainAgentReasoningEffort: models.ReasoningEffort(getStringValue(updatedPrefs.MainAgentReasoningEffort)),
		SubAgentModel:            models.ModelID(getStringValue(updatedPrefs.SubAgentModel)),
		SubAgentMaxTokens:        getInt64Value(updatedPrefs.SubAgentMaxTokens),
		SubAgentReasoningEffort:  models.ReasoningEffort(getStringValue(updatedPrefs.SubAgentReasoningEffort)),
		CreatedAt:                updatedPrefs.CreatedAt,
		UpdatedAt:                updatedPrefs.UpdatedAt,
	}

	WriteJSONResponse(w, http.StatusOK, response)
}

// HandleGetAvailableProviders handles GET /api/preferences/providers
func (h *PreferencesHandler) HandleGetAvailableProviders(w http.ResponseWriter, r *http.Request) {
	providers := models.GetProviders()

	// Check which providers are authenticated
	availableProviders := make(map[string]interface{})

	for providerName, providerInfo := range providers {
		// Check if provider is authenticated
		// This would require checking OAuth credentials and API keys
		// For now, return all providers with their info
		availableProviders[string(providerName)] = map[string]interface{}{
			"display_name": providerInfo.DisplayName,
			"models":       providerInfo.Models,
		}
	}

	WriteJSONResponse(w, http.StatusOK, availableProviders)
}

// HandleResetPreferences handles POST /api/preferences/reset
func (h *PreferencesHandler) HandleResetPreferences(w http.ResponseWriter, r *http.Request) {
	userPrefs := config.GetUserPreferences()
	if userPrefs == nil {
		WriteErrorResponse(w, http.StatusInternalServerError, "user preferences service not available", "PREFERENCES_SERVICE_UNAVAILABLE")
		return
	}

	ctx := r.Context()

	// Get current preferences before resetting for analytics
	currentPrefs, err := userPrefs.GetUserPreferences(ctx)
	previousProvider := ""
	previousModel := ""
	if err == nil {
		previousProvider = getStringValue(currentPrefs.PreferredProvider)
		previousModel = getStringValue(currentPrefs.MainAgentModel)
	}

	// Reset preferences to defaults via user preferences service
	// Reset main agent to defaults
	err = userPrefs.UpdateMainAgentPreferences(ctx, "claude-sonnet-4-5", 4096, "")
	if err != nil {
		logging.Error("Failed to reset main agent preferences", "error", err)
		WriteErrorResponse(w, http.StatusInternalServerError, "failed to reset main agent", "DATABASE_ERROR")
		return
	}

	// Reset sub agent to defaults
	err = userPrefs.UpdateSubAgentPreferences(ctx, "claude-sonnet-4-5", 2048, "")
	if err != nil {
		logging.Error("Failed to reset sub agent preferences", "error", err)
		WriteErrorResponse(w, http.StatusInternalServerError, "failed to reset sub agent", "DATABASE_ERROR")
		return
	}

	// Reset preferred provider to defaults
	err = userPrefs.UpdatePreferredProvider(ctx, models.ProviderAnthropic)
	if err != nil {
		logging.Error("Failed to reset preferred provider", "error", err)
		WriteErrorResponse(w, http.StatusInternalServerError, "failed to reset provider", "DATABASE_ERROR")
		return
	}

	// Get reset preferences
	resetPrefs, err := userPrefs.GetOrCreateUserPreferences(ctx)
	if err != nil {
		logging.Error("Failed to reset user preferences", "error", err)
		WriteErrorResponse(w, http.StatusInternalServerError, "failed to reset preferences", "DATABASE_ERROR")
		return
	}

	// Track preferences reset
	if h.app.Analytics != nil {
		_ = h.app.Analytics.TrackPreferencesReset(ctx, previousProvider, previousModel)
	}

	response := UserPreferencesResponse{
		PreferredProvider:        models.ModelProvider(getStringValue(resetPrefs.PreferredProvider)),
		MainAgentModel:           models.ModelID(getStringValue(resetPrefs.MainAgentModel)),
		MainAgentMaxTokens:       getInt64Value(resetPrefs.MainAgentMaxTokens),
		MainAgentReasoningEffort: models.ReasoningEffort(getStringValue(resetPrefs.MainAgentReasoningEffort)),
		SubAgentModel:            models.ModelID(getStringValue(resetPrefs.SubAgentModel)),
		SubAgentMaxTokens:        getInt64Value(resetPrefs.SubAgentMaxTokens),
		SubAgentReasoningEffort:  models.ReasoningEffort(getStringValue(resetPrefs.SubAgentReasoningEffort)),
		CreatedAt:                resetPrefs.CreatedAt,
		UpdatedAt:                resetPrefs.UpdatedAt,
	}

	WriteJSONResponse(w, http.StatusOK, response)
}

// Helper functions to handle sql.NullString and sql.NullInt64
func getStringValue(ns sql.NullString) string {
	if ns.Valid {
		return ns.String
	}
	return ""
}

func getInt64Value(ni sql.NullInt64) int64 {
	if ni.Valid {
		return ni.Int64
	}
	return 0
}

// AuthStatus represents the current authentication status
type AuthStatus struct {
	HasAnyAuth bool                      `json:"has_any_auth"`
	Providers  map[string]ProviderStatus `json:"providers"`
}

type ProviderStatus struct {
	Authenticated bool               `json:"authenticated"`
	AuthMethod    models.AuthMethod  `json:"auth_method"`
	DisplayName   string             `json:"display_name"`
}

// DEPRECATED: getAuthMethod is replaced by functionality in AuthHandler
// This method is kept for reference and will be removed in a future update.
func getAuthMethod(hasAPIKey, hasOAuth bool) string {
	if hasOAuth {
		return "oauth"
	}
	if hasAPIKey {
		return "api_key"
	}
	return "none"
}
