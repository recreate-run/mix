package http

import (
	"database/sql"
	"encoding/json"
	"net/http"
	"os"

	"mix/internal/app"
	"mix/internal/config"
	"mix/internal/db"
	"mix/internal/llm/models"
	llmprovider "mix/internal/llm/provider"
	"mix/internal/logging"
)

// UserPreferencesResponse represents the API response for user preferences
type UserPreferencesResponse struct {
	PreferredProvider        string `json:"preferred_provider"`
	MainAgentModel           string `json:"main_agent_model"`
	MainAgentMaxTokens       int64  `json:"main_agent_max_tokens"`
	MainAgentReasoningEffort string `json:"main_agent_reasoning_effort"`
	SubAgentModel            string `json:"sub_agent_model"`
	SubAgentMaxTokens        int64  `json:"sub_agent_max_tokens"`
	SubAgentReasoningEffort  string `json:"sub_agent_reasoning_effort"`
	CreatedAt                int64  `json:"created_at"`
	UpdatedAt                int64  `json:"updated_at"`
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
	// First check if user has any authentication configured
	authStatus := h.checkAuthenticationStatus()

	if !authStatus.HasAnyAuth {
		// No auth configured, return setup flow
		response := map[string]interface{}{
			"setup_required":      true,
			"auth_status":         authStatus,
			"available_providers": h.getAvailableProvidersForSetup(),
			"message":             "No authentication configured. Please select a provider and complete OAuth flow.",
		}
		WriteJSONResponse(w, http.StatusOK, response)
		return
	}

	// User has auth, check if preferences exist in database
	userPrefs := config.GetUserPreferences()
	if userPrefs == nil {
		WriteErrorResponse(w, http.StatusInternalServerError, "user preferences service not available", "PREFERENCES_SERVICE_UNAVAILABLE")
		return
	}

	ctx := r.Context()
	prefs, err := userPrefs.GetOrCreateUserPreferences(ctx)
	if err != nil {
		logging.Error("Failed to get user preferences", "error", err)
		WriteErrorResponse(w, http.StatusInternalServerError, "failed to get preferences", "DATABASE_ERROR")
		return
	}

	response := UserPreferencesResponse{
		PreferredProvider:        getStringValue(prefs.PreferredProvider),
		MainAgentModel:           getStringValue(prefs.MainAgentModel),
		MainAgentMaxTokens:       getInt64Value(prefs.MainAgentMaxTokens),
		MainAgentReasoningEffort: getStringValue(prefs.MainAgentReasoningEffort),
		SubAgentModel:            getStringValue(prefs.SubAgentModel),
		SubAgentMaxTokens:        getInt64Value(prefs.SubAgentMaxTokens),
		SubAgentReasoningEffort:  getStringValue(prefs.SubAgentReasoningEffort),
		CreatedAt:                prefs.CreatedAt,
		UpdatedAt:                prefs.UpdatedAt,
	}

	WriteJSONResponse(w, http.StatusOK, response)
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

	// Get current preferences
	currentPrefs, err := userPrefs.GetOrCreateUserPreferences(ctx)
	if err != nil {
		logging.Error("Failed to get current user preferences", "error", err)
		WriteErrorResponse(w, http.StatusInternalServerError, "failed to get current preferences", "DATABASE_ERROR")
		return
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
		updateParams.MainAgentMaxTokens = sql.NullInt64{Int64: *request.MainAgentMaxTokens, Valid: *request.MainAgentMaxTokens > 0}
	}

	if request.MainAgentReasoningEffort != nil {
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
		updateParams.SubAgentMaxTokens = sql.NullInt64{Int64: *request.SubAgentMaxTokens, Valid: *request.SubAgentMaxTokens > 0}
	}

	if request.SubAgentReasoningEffort != nil {
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

	response := UserPreferencesResponse{
		PreferredProvider:        getStringValue(updatedPrefs.PreferredProvider),
		MainAgentModel:           getStringValue(updatedPrefs.MainAgentModel),
		MainAgentMaxTokens:       getInt64Value(updatedPrefs.MainAgentMaxTokens),
		MainAgentReasoningEffort: getStringValue(updatedPrefs.MainAgentReasoningEffort),
		SubAgentModel:            getStringValue(updatedPrefs.SubAgentModel),
		SubAgentMaxTokens:        getInt64Value(updatedPrefs.SubAgentMaxTokens),
		SubAgentReasoningEffort:  getStringValue(updatedPrefs.SubAgentReasoningEffort),
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

	// Reset preferences to defaults via user preferences service
	// Reset main agent to defaults
	err := userPrefs.UpdateMainAgentPreferences(ctx, "claude-4-sonnet", 4096, "")
	if err != nil {
		logging.Error("Failed to reset main agent preferences", "error", err)
		WriteErrorResponse(w, http.StatusInternalServerError, "failed to reset main agent", "DATABASE_ERROR")
		return
	}
	
	// Reset sub agent to defaults
	err = userPrefs.UpdateSubAgentPreferences(ctx, "claude-4-sonnet", 2048, "")
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

	response := UserPreferencesResponse{
		PreferredProvider:        getStringValue(resetPrefs.PreferredProvider),
		MainAgentModel:           getStringValue(resetPrefs.MainAgentModel),
		MainAgentMaxTokens:       getInt64Value(resetPrefs.MainAgentMaxTokens),
		MainAgentReasoningEffort: getStringValue(resetPrefs.MainAgentReasoningEffort),
		SubAgentModel:            getStringValue(resetPrefs.SubAgentModel),
		SubAgentMaxTokens:        getInt64Value(resetPrefs.SubAgentMaxTokens),
		SubAgentReasoningEffort:  getStringValue(resetPrefs.SubAgentReasoningEffort),
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
	Authenticated bool   `json:"authenticated"`
	AuthMethod    string `json:"auth_method"` // "oauth", "api_key", "none"
	DisplayName   string `json:"display_name"`
}

// checkAuthenticationStatus checks what auth methods are available
func (h *PreferencesHandler) checkAuthenticationStatus() AuthStatus {
	status := AuthStatus{
		HasAnyAuth: false,
		Providers:  make(map[string]ProviderStatus),
	}

	// Check environment variables for API keys (Anthropic uses OAuth only)
	openaiKey := os.Getenv("OPENAI_API_KEY")
	geminiKey := os.Getenv("GEMINI_API_KEY")
	groqKey := os.Getenv("GROQ_API_KEY")
	openrouterKey := os.Getenv("OPENROUTER_API_KEY")

	// Check OAuth credentials for both Anthropic and OpenAI
	anthropicOAuth := h.checkOAuthCredentials("anthropic")
	openaiOAuth := h.checkOAuthCredentials("openai")

	status.Providers["anthropic"] = ProviderStatus{
		Authenticated: anthropicOAuth, // Anthropic: OAuth only
		AuthMethod:    getAuthMethod(false, anthropicOAuth),
		DisplayName:   "Anthropic (Claude)",
	}

	status.Providers["openai"] = ProviderStatus{
		Authenticated: openaiKey != "" || openaiOAuth, // OpenAI: API key OR OAuth
		AuthMethod:    getAuthMethod(openaiKey != "", openaiOAuth),
		DisplayName:   "OpenAI (GPT)",
	}

	status.Providers["gemini"] = ProviderStatus{
		Authenticated: geminiKey != "",
		AuthMethod:    getAuthMethod(geminiKey != "", false),
		DisplayName:   "Google Gemini",
	}

	status.Providers["groq"] = ProviderStatus{
		Authenticated: groqKey != "",
		AuthMethod:    getAuthMethod(groqKey != "", false),
		DisplayName:   "GROQ",
	}

	status.Providers["openrouter"] = ProviderStatus{
		Authenticated: openrouterKey != "",
		AuthMethod:    getAuthMethod(openrouterKey != "", false),
		DisplayName:   "OpenRouter",
	}

	// Check if any provider is authenticated
	for _, provider := range status.Providers {
		if provider.Authenticated {
			status.HasAnyAuth = true
			break
		}
	}

	return status
}

// getAvailableProvidersForSetup returns providers available for initial setup
func (h *PreferencesHandler) getAvailableProvidersForSetup() map[string]interface{} {
	return map[string]interface{}{
		"anthropic": map[string]interface{}{
			"display_name":   "Anthropic (Claude)",
			"auth_method":    "oauth",
			"supports_oauth": true,
			"models":         []string{"claude-4-sonnet", "claude-3.5-sonnet", "claude-3-haiku"},
		},
		"openai": map[string]interface{}{
			"display_name":   "OpenAI (GPT)", 
			"auth_method":    "api_key",
			"supports_oauth": false,
			"models":         []string{"gpt-4o", "gpt-4-turbo", "o1-preview"},
		},
		"gemini": map[string]interface{}{
			"display_name":   "Google Gemini",
			"auth_method":    "api_key",
			"supports_oauth": false,
			"models":         []string{"gemini-2.5-flash", "gemini-2.5-pro"},
		},
		"groq": map[string]interface{}{
			"display_name":   "GROQ",
			"auth_method":    "api_key",
			"supports_oauth": false,
			"models":         []string{"llama-3.3-70b", "qwen-qwq-32b"},
		},
		"openrouter": map[string]interface{}{
			"display_name":   "OpenRouter",
			"auth_method":    "api_key",
			"supports_oauth": false,
			"models":         []string{"claude-3.5-sonnet", "gpt-4o", "gemini-2.5-flash"},
		},
	}
}

// checkOAuthCredentials checks if OAuth credentials exist for a provider  
func (h *PreferencesHandler) checkOAuthCredentials(provider string) bool {
	// Check both Anthropic and OpenAI OAuth for now since both are working
	if provider != "anthropic" && provider != "openai" {
		return false
	}
	
	// Use the same credential storage system as the auth command
	storage, err := llmprovider.NewCredentialStorage()
	if err != nil {
		logging.Warn("Failed to initialize credential storage", "error", err)
		return false
	}
	
	switch provider {
	case "anthropic":
		// Check for Anthropic OAuth credentials
		creds, err := storage.GetOAuthCredentials("anthropic")
		if err != nil {
			return false
		}
		return creds != nil && !creds.IsTokenExpired()
		
	case "openai":
		// Check for OpenAI OAuth credentials  
		creds, err := storage.GetOpenAICredentials("openai")
		if err != nil {
			return false
		}
		return creds != nil && !creds.IsTokenExpired()
		
	default:
		return false
	}
}

// Helper function for auth method determination
func getAuthMethod(hasAPIKey, hasOAuth bool) string {
	if hasOAuth {
		return "oauth"
	}
	if hasAPIKey {
		return "api_key"
	}
	return "none"
}
