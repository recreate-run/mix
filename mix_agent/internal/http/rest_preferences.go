package http

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"net/http"

	"mix/internal/app"
	"mix/internal/config"
	"mix/internal/db"
	"mix/internal/llm/models"
	"mix/internal/logging"
	"mix/internal/preferences"
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
func NewPreferencesHandler(a *app.App) *PreferencesHandler {
	return &PreferencesHandler{
		app: a,
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
		if errors.Is(err, sql.ErrNoRows) {
			WriteJSONResponse(w, http.StatusOK, PreferencesWithProviders{
				Preferences:        nil,
				AvailableProviders: convertProvidersToInfo(models.GetProviders()),
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
	responseWithProviders := PreferencesWithProviders{
		Preferences:        &response,
		AvailableProviders: convertProvidersToInfo(models.GetProviders()),
	}

	WriteJSONResponse(w, http.StatusOK, responseWithProviders)
}

// HandleUpdatePreferences handles POST /api/preferences
func (h *PreferencesHandler) HandleUpdatePreferences(w http.ResponseWriter, r *http.Request) {
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

	if err := h.validateAndUpdatePreferences(ctx, w, userPrefs, &request); err != nil {
		return
	}

	updatedPrefs, err := userPrefs.GetOrCreateUserPreferences(ctx)
	if err != nil {
		logging.Error("Failed to update user preferences", "error", err)
		WriteErrorResponse(w, http.StatusInternalServerError, "failed to update preferences", "DATABASE_ERROR")
		return
	}

	h.trackPreferencesUpdate(ctx, &request)
	h.clearSessionProviderCache()

	response := buildPreferencesResponse(*updatedPrefs)
	WriteJSONResponse(w, http.StatusOK, response)
}

// HandleGetAvailableProviders handles GET /api/preferences/providers
func (h *PreferencesHandler) HandleGetAvailableProviders(w http.ResponseWriter, r *http.Request) {
	providers := models.GetProviders()
	availableProviders := convertProvidersToInfo(providers)
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
	Authenticated bool              `json:"authenticated"`
	AuthMethod    models.AuthMethod `json:"auth_method"`
	DisplayName   string            `json:"display_name"`
}

// Deprecated: getAuthMethod is replaced by functionality in AuthHandler.
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

func (h *PreferencesHandler) validateAndUpdatePreferences(ctx context.Context, w http.ResponseWriter, userPrefs preferences.Service, request *UpdatePreferencesRequest) error {
	if err := h.validateRequest(w, request); err != nil {
		return err
	}
	return h.applyPreferenceUpdates(ctx, w, userPrefs, request)
}

func (h *PreferencesHandler) validateRequest(w http.ResponseWriter, request *UpdatePreferencesRequest) error {
	if request.PreferredProvider != nil && *request.PreferredProvider != "" {
		if _, exists := models.GetProviders()[models.ModelProvider(*request.PreferredProvider)]; !exists {
			WriteErrorResponse(w, http.StatusBadRequest, "invalid provider", "INVALID_PROVIDER")
			return errors.New("invalid provider")
		}
	}

	if err := validateModel(w, request.MainAgentModel, "main agent"); err != nil {
		return err
	}
	if err := validateModel(w, request.SubAgentModel, "sub agent"); err != nil {
		return err
	}
	if err := validateMaxTokens(w, request.MainAgentMaxTokens, "main agent"); err != nil {
		return err
	}
	if err := validateMaxTokens(w, request.SubAgentMaxTokens, "sub agent"); err != nil {
		return err
	}
	if err := validateReasoningEffort(w, request.MainAgentReasoningEffort, "main agent"); err != nil {
		return err
	}
	return validateReasoningEffort(w, request.SubAgentReasoningEffort, "sub agent")
}

func validateModel(w http.ResponseWriter, model *string, agentType string) error {
	if model != nil && *model != "" {
		if _, exists := models.SupportedModels[models.ModelID(*model)]; !exists {
			WriteErrorResponse(w, http.StatusBadRequest, "invalid "+agentType+" model", "INVALID_MODEL")
			return errors.New("invalid model")
		}
	}
	return nil
}

func validateMaxTokens(w http.ResponseWriter, maxTokens *int64, agentType string) error {
	if maxTokens != nil && *maxTokens <= 0 {
		WriteErrorResponse(w, http.StatusBadRequest, agentType+" max tokens must be positive", "INVALID_TOKEN_COUNT")
		return errors.New("invalid token count")
	}
	return nil
}

func validateReasoningEffort(w http.ResponseWriter, effort *string, agentType string) error {
	if effort != nil && *effort != "" {
		validEfforts := map[string]bool{"low": true, "medium": true, "high": true}
		if !validEfforts[*effort] {
			WriteErrorResponse(w, http.StatusBadRequest, "invalid "+agentType+" reasoning effort: must be 'low', 'medium', 'high', or empty", "INVALID_REASONING_EFFORT")
			return errors.New("invalid reasoning effort")
		}
	}
	return nil
}

func (h *PreferencesHandler) applyPreferenceUpdates(ctx context.Context, w http.ResponseWriter, userPrefs preferences.Service, request *UpdatePreferencesRequest) error {
	var err error

	if request.MainAgentModel != nil {
		err = h.updateAgentPreferences(ctx, userPrefs, request.MainAgentModel, request.MainAgentMaxTokens, request.MainAgentReasoningEffort, true)
		if err != nil {
			logging.Error("Failed to update main agent preferences", "error", err)
			WriteErrorResponse(w, http.StatusInternalServerError, "failed to update main agent preferences", "DATABASE_ERROR")
			return err
		}
	}

	if request.SubAgentModel != nil {
		err = h.updateAgentPreferences(ctx, userPrefs, request.SubAgentModel, request.SubAgentMaxTokens, request.SubAgentReasoningEffort, false)
		if err != nil {
			logging.Error("Failed to update sub agent preferences", "error", err)
			WriteErrorResponse(w, http.StatusInternalServerError, "failed to update sub agent preferences", "DATABASE_ERROR")
			return err
		}
	}

	if request.PreferredProvider != nil {
		err = userPrefs.UpdatePreferredProvider(ctx, models.ModelProvider(*request.PreferredProvider))
		if err != nil {
			logging.Error("Failed to update preferred provider", "error", err)
			WriteErrorResponse(w, http.StatusInternalServerError, "failed to update preferred provider", "DATABASE_ERROR")
			return err
		}
	}

	return nil
}

func (h *PreferencesHandler) updateAgentPreferences(ctx context.Context, userPrefs preferences.Service, model *string, maxTokens *int64, reasoningEffort *string, isMainAgent bool) error {
	modelID := models.ModelID(*model)
	defaultTokens := int64(2048)
	if isMainAgent {
		defaultTokens = 4096
	}

	tokens := defaultTokens
	if maxTokens != nil {
		tokens = *maxTokens
	}

	effort := ""
	if reasoningEffort != nil {
		effort = *reasoningEffort
	}

	if isMainAgent {
		return userPrefs.UpdateMainAgentPreferences(ctx, modelID, tokens, effort)
	}
	return userPrefs.UpdateSubAgentPreferences(ctx, modelID, tokens, effort)
}

func (h *PreferencesHandler) trackPreferencesUpdate(ctx context.Context, request *UpdatePreferencesRequest) {
	if h.app.Analytics == nil {
		return
	}

	fieldsChanged := []string{}
	updates := make(map[string]interface{})

	addFieldUpdate := func(field *string, name string, value interface{}) {
		if field != nil {
			fieldsChanged = append(fieldsChanged, name)
			updates[name] = value
		}
	}

	addFieldUpdate(request.PreferredProvider, "preferred_provider", request.PreferredProvider)
	addFieldUpdate(request.MainAgentModel, "main_agent_model", request.MainAgentModel)
	if request.MainAgentMaxTokens != nil {
		fieldsChanged = append(fieldsChanged, "main_agent_max_tokens")
		updates["main_agent_max_tokens"] = *request.MainAgentMaxTokens
	}
	addFieldUpdate(request.MainAgentReasoningEffort, "main_agent_reasoning_effort", request.MainAgentReasoningEffort)
	addFieldUpdate(request.SubAgentModel, "sub_agent_model", request.SubAgentModel)
	if request.SubAgentMaxTokens != nil {
		fieldsChanged = append(fieldsChanged, "sub_agent_max_tokens")
		updates["sub_agent_max_tokens"] = *request.SubAgentMaxTokens
	}
	addFieldUpdate(request.SubAgentReasoningEffort, "sub_agent_reasoning_effort", request.SubAgentReasoningEffort)

	if len(fieldsChanged) > 0 {
		_ = h.app.Analytics.TrackPreferencesUpdated(ctx, fieldsChanged, updates)
	}
}

func (h *PreferencesHandler) clearSessionProviderCache() {
	if h.app.CoderAgent != nil {
		h.app.CoderAgent.ClearAllSessionProviders()
	} else {
		logging.Warn("Could not clear session provider cache: coderAgent is nil")
	}
}

func buildPreferencesResponse(prefs db.UserPreference) UserPreferencesResponse {
	return UserPreferencesResponse{
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
}

// convertProvidersToInfo converts the models package provider info to API response format
func convertProvidersToInfo(providers map[models.ModelProvider]models.ProviderInfo) map[string]ProviderInfo {
	result := make(map[string]ProviderInfo, len(providers))
	for providerName, providerInfo := range providers {
		result[string(providerName)] = ProviderInfo{
			DisplayName: providerInfo.DisplayName,
			Models:      providerInfo.Models,
		}
	}
	return result
}
