package http

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"mix/internal/app"
	"mix/internal/config"
	"mix/internal/llm/models"
	llmprovider "mix/internal/llm/provider"
	"mix/internal/logging"
)

// AuthHandler handles REST endpoints for authentication management
type AuthHandler struct {
	app *app.App
}

// NewAuthHandler creates a new auth handler
func NewAuthHandler(app *app.App) *AuthHandler {
	return &AuthHandler{
		app: app,
	}
}

// StoreAPIKeyRequest represents the request body for storing an API key
type StoreAPIKeyRequest struct {
	Provider string `json:"provider"`
	APIKey   string `json:"api_key"`
}

// AuthStatusResponse represents the authentication status for all providers
type AuthStatusResponse struct {
	Providers map[string]ProviderAuthStatus `json:"providers"`
}

// ProviderAuthStatus represents authentication status for a single provider
type ProviderAuthStatus struct {
	Authenticated bool   `json:"authenticated"`
	AuthMethod    string `json:"auth_method"` // "oauth", "api_key", "none"
	DisplayName   string `json:"display_name"`
}

// HandleStoreAPIKey handles POST /api/auth/api-key
func (h *AuthHandler) HandleStoreAPIKey(w http.ResponseWriter, r *http.Request) {
	setCORSHeaders(w)
	if handleCORSPreflight(w, r) {
		return
	}

	if r.Method != "POST" {
		WriteErrorResponse(w, http.StatusMethodNotAllowed, "Method not allowed", "METHOD_NOT_ALLOWED")
		return
	}

	var request StoreAPIKeyRequest
	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		WriteErrorResponse(w, http.StatusBadRequest, "Invalid JSON request", "INVALID_JSON")
		return
	}

	// Validate request
	if request.Provider == "" {
		WriteErrorResponse(w, http.StatusBadRequest, "Provider is required", "MISSING_PROVIDER")
		return
	}

	if request.APIKey == "" {
		WriteErrorResponse(w, http.StatusBadRequest, "API key is required", "MISSING_API_KEY")
		return
	}

	// Validate provider - only allow supported providers
	if _, exists := supportedProviders[request.Provider]; !exists {
		WriteErrorResponse(w, http.StatusBadRequest, "Provider not supported. Supported providers: anthropic, openai, openrouter", "INVALID_PROVIDER")
		return
	}

	provider := models.ModelProvider(request.Provider)

	// Get API credentials service
	credentialsService := config.GetAPICredentials()
	if credentialsService == nil {
		WriteErrorResponse(w, http.StatusInternalServerError, "Credentials service not available", "CREDENTIALS_SERVICE_UNAVAILABLE")
		return
	}

	// Validate API key format
	if err := credentialsService.ValidateAPIKey(provider, request.APIKey); err != nil {
		// Track failed authentication attempt
		if h.app.Analytics != nil {
			h.app.Analytics.TrackProviderAuth(r.Context(), string(provider), false, "api_key")
		}
		WriteErrorResponse(w, http.StatusBadRequest, err.Error(), "INVALID_API_KEY_FORMAT")
		return
	}

	// Store encrypted API key
	ctx := r.Context()
	if err := credentialsService.StoreAPIKey(ctx, provider, request.APIKey); err != nil {
		logging.Error("Failed to store API key", "error", err, "provider", provider)
		// Track failed storage attempt
		if h.app.Analytics != nil {
			h.app.Analytics.TrackProviderAuth(ctx, string(provider), false, "api_key")
		}
		WriteErrorResponse(w, http.StatusInternalServerError, "Failed to store API key", "STORAGE_ERROR")
		return
	}

	// Track successful authentication
	if h.app.Analytics != nil {
		h.app.Analytics.TrackProviderAuth(ctx, string(provider), true, "api_key")
		logging.Info("Tracked successful API key authentication", "provider", provider)
	}

	response := map[string]interface{}{
		"status":   "success",
		"provider": request.Provider,
		"message":  "API key stored successfully",
	}

	WriteJSONResponse(w, http.StatusOK, response)
}

// HandleDeleteCredentials handles DELETE /api/auth/{provider}
func (h *AuthHandler) HandleDeleteCredentials(w http.ResponseWriter, r *http.Request) {
	setCORSHeaders(w)
	if handleCORSPreflight(w, r) {
		return
	}

	if r.Method != "DELETE" {
		WriteErrorResponse(w, http.StatusMethodNotAllowed, "Method not allowed", "METHOD_NOT_ALLOWED")
		return
	}

	provider := r.PathValue("provider")
	if provider == "" {
		WriteErrorResponse(w, http.StatusBadRequest, "Provider is required", "MISSING_PROVIDER")
		return
	}

	// Validate provider - only allow supported providers
	if _, exists := supportedProviders[provider]; !exists {
		WriteErrorResponse(w, http.StatusBadRequest, "Provider not supported. Supported providers: anthropic, openai, openrouter", "INVALID_PROVIDER")
		return
	}

	modelProvider := models.ModelProvider(provider)

	// Get services
	credentialsService := config.GetAPICredentials()
	if credentialsService == nil {
		WriteErrorResponse(w, http.StatusInternalServerError, "Credentials service not available", "CREDENTIALS_SERVICE_UNAVAILABLE")
		return
	}

	ctx := r.Context()

	// Delete API key if it exists
	if err := credentialsService.DeleteAPIKey(ctx, modelProvider); err != nil {
		logging.Error("Failed to delete API key", "error", err, "provider", provider)
		// Track failed deletion attempt
		if h.app.Analytics != nil {
			h.app.Analytics.TrackProviderAuth(ctx, provider, false, "delete_credentials")
		}
		WriteErrorResponse(w, http.StatusInternalServerError, "Failed to delete API key", "DELETION_ERROR")
		return
	}

	// Track successful credential deletion
	if h.app.Analytics != nil {
		h.app.Analytics.TrackProviderAuth(ctx, provider, true, "delete_credentials")
	}

	// Also clear OAuth credentials if this is Anthropic
	if provider == "anthropic" {
		storage, err := llmprovider.NewCredentialStorage()
		if err == nil {
			if err := storage.ClearOAuthCredentials("anthropic"); err != nil {
				logging.Warn("Failed to clear OAuth credentials", "error", err, "provider", provider)
			}
		}
	}

	response := map[string]interface{}{
		"status":   "success",
		"provider": provider,
		"message":  "Credentials deleted successfully",
	}

	WriteJSONResponse(w, http.StatusOK, response)
}

// HandleAuthStatus handles GET /api/auth/status
func (h *AuthHandler) HandleAuthStatus(w http.ResponseWriter, r *http.Request) {
	setCORSHeaders(w)
	if handleCORSPreflight(w, r) {
		return
	}

	if r.Method != "GET" {
		WriteErrorResponse(w, http.StatusMethodNotAllowed, "Method not allowed", "METHOD_NOT_ALLOWED")
		return
	}

	ctx := r.Context()
	status := h.checkAllAuthenticationStatus(ctx)

	WriteJSONResponse(w, http.StatusOK, status)
}

// HandleStartOAuth handles POST /api/auth/oauth/{provider}
func (h *AuthHandler) HandleStartOAuth(w http.ResponseWriter, r *http.Request) {
	setCORSHeaders(w)
	if handleCORSPreflight(w, r) {
		return
	}

	if r.Method != "POST" {
		WriteErrorResponse(w, http.StatusMethodNotAllowed, "Method not allowed", "METHOD_NOT_ALLOWED")
		return
	}

	provider := r.PathValue("provider")
	if provider == "" {
		WriteErrorResponse(w, http.StatusBadRequest, "Provider is required", "MISSING_PROVIDER")
		return
	}

	// Validate provider - only allow supported providers
	if _, exists := supportedProviders[provider]; !exists {
		WriteErrorResponse(w, http.StatusBadRequest, "Provider not supported. Supported providers: anthropic, openai, openrouter", "INVALID_PROVIDER")
		return
	}

	// Currently only Anthropic supports OAuth
	if provider != "anthropic" {
		WriteErrorResponse(w, http.StatusBadRequest, "OAuth not supported for this provider", "OAUTH_NOT_SUPPORTED")
		return
	}

	// Create OAuth flow
	oauthFlow, err := llmprovider.NewOAuthFlow("")
	if err != nil {
		logging.Error("Failed to create OAuth flow", "error", err, "provider", provider)
		WriteErrorResponse(w, http.StatusInternalServerError, "Failed to create OAuth flow", "OAUTH_ERROR")
		return
	}

	// Return the authorization URL for the client to redirect to
	authURL := oauthFlow.GetAuthorizationURL()

	response := map[string]interface{}{
		"auth_url": authURL,
		"state":    oauthFlow.State,
		"message":  "Open the auth_url in your browser to complete OAuth authentication",
	}

	WriteJSONResponse(w, http.StatusOK, response)
}

// HandleValidatePreferredProvider handles GET /api/auth/validate
func (h *AuthHandler) HandleValidatePreferredProvider(w http.ResponseWriter, r *http.Request) {
	setCORSHeaders(w)
	if handleCORSPreflight(w, r) {
		return
	}

	if r.Method != "GET" {
		WriteErrorResponse(w, http.StatusMethodNotAllowed, "Method not allowed", "METHOD_NOT_ALLOWED")
		return
	}

	ctx := r.Context()

	// Get user preferences
	userPrefs := config.GetUserPreferences()
	if userPrefs == nil {
		WriteErrorResponse(w, http.StatusInternalServerError, "User preferences not available", "PREFERENCES_UNAVAILABLE")
		return
	}

	// Get preferred provider
	preferredProvider, err := userPrefs.GetPreferredProvider(ctx)
	if err != nil || preferredProvider == "" {
		response := map[string]interface{}{
			"valid":   false,
			"error":   "No preferred provider set",
			"message": "Please set a preferred provider first",
		}
		WriteJSONResponse(w, http.StatusOK, response)
		return
	}

	// Check if preferred provider is authenticated
	credentialsService := config.GetAPICredentials()
	isAuthenticated := false
	authMethod := "none"

	if credentialsService != nil {
		// Check database API key
		hasAPIKey, err := credentialsService.HasAPIKey(ctx, preferredProvider)
		if err == nil && hasAPIKey {
			isAuthenticated = true
			authMethod = "api_key"
		}

		// Check OAuth for Anthropic if no API key
		if !isAuthenticated && preferredProvider == models.ProviderAnthropic {
			if h.checkOAuthCredentials("anthropic") {
				isAuthenticated = true
				authMethod = "oauth"
			}
		}
	}

	response := map[string]interface{}{
		"valid":       isAuthenticated,
		"provider":    string(preferredProvider),
		"auth_method": authMethod,
		"message": func() string {
			if isAuthenticated {
				return fmt.Sprintf("Ready to use %s", preferredProvider)
			}
			return fmt.Sprintf("Please authenticate with %s first", preferredProvider)
		}(),
	}

	WriteJSONResponse(w, http.StatusOK, response)
}

// supportedProviders defines the limited set of providers we support
var supportedProviders = map[string]struct{}{
	"anthropic":  {},
	"openai":     {},
	"openrouter": {},
}

// HandleOAuthCallback handles POST /api/auth/oauth/callback
func (h *AuthHandler) HandleOAuthCallback(w http.ResponseWriter, r *http.Request) {
	setCORSHeaders(w)
	if handleCORSPreflight(w, r) {
		return
	}

	if r.Method != "POST" {
		WriteErrorResponse(w, http.StatusMethodNotAllowed, "Method not allowed", "METHOD_NOT_ALLOWED")
		return
	}

	// Parse the request body
	var req struct {
		Provider string `json:"provider"`
		Code     string `json:"code"`
		State    string `json:"state"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		WriteErrorResponse(w, http.StatusBadRequest, "Invalid JSON request", "INVALID_JSON")
		return
	}

	// Validate request
	if req.Provider == "" {
		WriteErrorResponse(w, http.StatusBadRequest, "Provider is required", "MISSING_PROVIDER")
		return
	}

	if req.Code == "" {
		WriteErrorResponse(w, http.StatusBadRequest, "Authorization code is required", "MISSING_CODE")
		return
	}

	// Currently only Anthropic supports OAuth
	if req.Provider != "anthropic" {
		// Track failed authentication attempt for unsupported provider
		if h.app.Analytics != nil {
			h.app.Analytics.TrackProviderAuth(r.Context(), req.Provider, false, "oauth")
		}
		logging.Error("OAuth provider not supported", "provider", req.Provider)
		WriteErrorResponse(w, http.StatusBadRequest, "OAuth not supported for this provider", "OAUTH_NOT_SUPPORTED")
		return
	}

	// Get the stored OAuth flow
	oauthFlow := llmprovider.GetOAuthFlow(req.State)
	if oauthFlow == nil {
		// Track failed authentication attempt due to invalid state
		if h.app.Analytics != nil {
			h.app.Analytics.TrackProviderAuth(r.Context(), req.Provider, false, "oauth")
		}
		logging.Error("Invalid or expired OAuth state", "state", req.State)
		WriteErrorResponse(w, http.StatusBadRequest, "Invalid or expired OAuth state", "INVALID_STATE")
		return
	}

	// Format the code with state as expected by the ExchangeCodeForTokens method
	authCode := fmt.Sprintf("%s#%s", req.Code, req.State)

	// Exchange code for tokens
	credentials, err := oauthFlow.ExchangeCodeForTokens(authCode)
	if err != nil {
		logging.Error("Failed to exchange authorization code for tokens", "error", err)
		// Track failed token exchange
		if h.app.Analytics != nil {
			h.app.Analytics.TrackProviderAuth(r.Context(), req.Provider, false, "oauth")
		}
		WriteErrorResponse(w, http.StatusInternalServerError, "Failed to exchange authorization code for tokens", "OAUTH_ERROR")
		return
	}

	// Initialize credential storage
	storage, err := llmprovider.NewCredentialStorage()
	if err != nil {
		logging.Error("Failed to initialize credential storage", "error", err)
		// Track failure due to storage initialization
		if h.app.Analytics != nil {
			h.app.Analytics.TrackProviderAuth(r.Context(), req.Provider, false, "oauth")
		}
		WriteErrorResponse(w, http.StatusInternalServerError, "Failed to initialize credential storage", "STORAGE_ERROR")
		return
	}

	// Store the credentials
	err = storage.StoreOAuthCredentials(
		"anthropic",
		credentials.AccessToken,
		credentials.RefreshToken,
		credentials.ExpiresAt,
		credentials.ClientID,
	)
	if err != nil {
		logging.Error("Failed to store OAuth credentials", "error", err)
		// Track failure due to storage error
		if h.app.Analytics != nil {
			h.app.Analytics.TrackProviderAuth(r.Context(), req.Provider, false, "oauth")
		}
		WriteErrorResponse(w, http.StatusInternalServerError, "Failed to store OAuth credentials", "STORAGE_ERROR")
		return
	}

	// Clean up the OAuth flow
	llmprovider.CleanupOAuthFlow(req.State)

	// Track successful authentication
	if h.app.Analytics != nil {
		ctx := r.Context()
		h.app.Analytics.TrackProviderAuth(ctx, req.Provider, true, "oauth")
		logging.Info("Tracked successful OAuth authentication", "provider", req.Provider)
	}

	// Return success response
	response := map[string]interface{}{
		"status":     "success",
		"provider":   req.Provider,
		"message":    "OAuth authentication successful",
		"expires_in": int64(credentials.ExpiresAt - time.Now().Unix()),
	}

	WriteJSONResponse(w, http.StatusOK, response)
}

// checkAllAuthenticationStatus checks authentication status for supported providers only
func (h *AuthHandler) checkAllAuthenticationStatus(ctx context.Context) AuthStatusResponse {
	status := AuthStatusResponse{
		Providers: make(map[string]ProviderAuthStatus),
	}

	// Get services
	credentialsService := config.GetAPICredentials()
	userPrefs := config.GetUserPreferences()

	// Get user's preferred provider if available
	var preferredProvider models.ModelProvider
	if userPrefs != nil {
		if pref, err := userPrefs.GetPreferredProvider(ctx); err == nil && pref != "" {
			preferredProvider = pref
			logging.Info("User preferred provider", "provider", preferredProvider)
		}
	}

	// Check each supported provider (database-only authentication)
	providers := []struct {
		name          string
		provider      models.ModelProvider
		displayName   string
		supportsOAuth bool
	}{
		{"anthropic", models.ProviderAnthropic, "Anthropic (Claude)", true},
		{"openai", models.ProviderOpenAI, "OpenAI (GPT)", false},
		{"openrouter", models.ProviderOpenRouter, "OpenRouter", false},
	}

	for _, p := range providers {
		var authenticated bool
		var authMethod string

		// Check OAuth first (only for Anthropic)
		hasOAuth := false
		if p.supportsOAuth {
			hasOAuth = h.checkOAuthCredentials(p.name)
		}

		// Check database API key
		hasAPIKey := false
		if credentialsService != nil {
			if hasKey, err := credentialsService.HasAPIKey(ctx, p.provider); err == nil {
				hasAPIKey = hasKey
			}
		}

		// Determine authentication status
		authenticated = hasOAuth || hasAPIKey
		authMethod = getAuthMethod(hasAPIKey, hasOAuth)

		// Mark as preferred if this matches user's preference
		displayName := p.displayName
		if p.provider == preferredProvider {
			displayName += " ⭐" // Mark preferred provider
		}

		status.Providers[p.name] = ProviderAuthStatus{
			Authenticated: authenticated,
			AuthMethod:    authMethod,
			DisplayName:   displayName,
		}
	}

	return status
}

// checkOAuthCredentials checks if OAuth credentials exist for a provider
func (h *AuthHandler) checkOAuthCredentials(provider string) bool {
	// Only Anthropic uses OAuth currently
	if provider != "anthropic" {
		return false
	}

	// Use the same credential storage system as the auth command
	storage, err := llmprovider.NewCredentialStorage()
	if err != nil {
		logging.Warn("Failed to initialize credential storage", "error", err)
		return false
	}

	// Check for Anthropic OAuth credentials
	creds, err := storage.GetOAuthCredentials("anthropic")
	if err != nil {
		return false
	}
	return creds != nil && !creds.IsTokenExpired()
}
