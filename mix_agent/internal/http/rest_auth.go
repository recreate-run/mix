package http

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"time"

	"mix/internal/app"
	"mix/internal/auth"
	"mix/internal/config"
	"mix/internal/constants"
	"mix/internal/credentials"
	"mix/internal/llm/models"
	llmprovider "mix/internal/llm/provider"
	"mix/internal/logging"
)

// AuthHandler handles REST endpoints for authentication management
type AuthHandler struct {
	app                 *app.App
	tokenRefreshService *auth.TokenRefreshService
}

// NewAuthHandler creates a new auth handler
func NewAuthHandler(appInstance *app.App) *AuthHandler {
	return &AuthHandler{
		app: appInstance,
	}
}

// SetTokenRefreshService sets the token refresh service (called after handler creation)
func (h *AuthHandler) SetTokenRefreshService(service *auth.TokenRefreshService) {
	h.tokenRefreshService = service
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
	Authenticated bool              `json:"authenticated"`
	AuthMethod    models.AuthMethod `json:"auth_method"`
	DisplayName   string            `json:"display_name"`
}

// GenericSuccessResponse represents a generic success response with message
type GenericSuccessResponse struct {
	Status   string `json:"status"`
	Provider string `json:"provider,omitempty"`
	Message  string `json:"message"`
}

// OAuthAuthURLResponse represents the OAuth authorization URL response
type OAuthAuthURLResponse struct {
	AuthURL string `json:"auth_url"`
	State   string `json:"state"`
	Message string `json:"message"`
}

// ValidateAuthResponse represents the response for auth validation
type ValidateAuthResponse struct {
	Valid      bool              `json:"valid"`
	Provider   string            `json:"provider"`
	AuthMethod models.AuthMethod `json:"auth_method"`
	Message    string            `json:"message"`
}

// OAuthCallbackResponse represents the OAuth callback success response
type OAuthCallbackResponse struct {
	Status    string `json:"status"`
	Provider  string `json:"provider"`
	Message   string `json:"message"`
	ExpiresIn int64  `json:"expires_in"`
}

// OAuthHealthResponse represents the OAuth/auth-specific health check response
type OAuthHealthResponse struct {
	Status    string                      `json:"status"`
	Providers map[string]auth.TokenStatus `json:"providers"`
	Timestamp string                      `json:"timestamp"`
}

// HandleStoreAPIKey handles POST /api/auth/api-key
func (h *AuthHandler) HandleStoreAPIKey(w http.ResponseWriter, r *http.Request) {
	setCORSHeaders(w)
	if handleCORSPreflight(w, r) {
		return
	}

	if r.Method != http.MethodPost {
		WriteErrorResponse(w, http.StatusMethodNotAllowed, constants.MethodNotAllowed, "METHOD_NOT_ALLOWED")
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
		WriteErrorResponse(w, http.StatusBadRequest, "Provider not supported. Supported providers: anthropic, openai, openrouter, gemini", "INVALID_PROVIDER")
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
			_ = h.app.Analytics.TrackProviderAuth(r.Context(), string(provider), false, "api_key")
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
			_ = h.app.Analytics.TrackProviderAuth(ctx, string(provider), false, "api_key")
		}
		WriteErrorResponse(w, http.StatusInternalServerError, "Failed to store API key", "STORAGE_ERROR")
		return
	}

	// Track successful authentication
	if h.app.Analytics != nil {
		_ = h.app.Analytics.TrackProviderAuth(ctx, string(provider), true, "api_key")
	}

	response := GenericSuccessResponse{
		Status:   "success",
		Provider: request.Provider,
		Message:  "API key stored successfully",
	}

	WriteJSONResponse(w, http.StatusOK, response)
}

// HandleDeleteCredentials handles DELETE /api/auth/{provider}
func (h *AuthHandler) HandleDeleteCredentials(w http.ResponseWriter, r *http.Request) {
	setCORSHeaders(w)
	if handleCORSPreflight(w, r) {
		return
	}

	if r.Method != http.MethodDelete {
		WriteErrorResponse(w, http.StatusMethodNotAllowed, constants.MethodNotAllowed, "METHOD_NOT_ALLOWED")
		return
	}

	provider := r.PathValue("provider")
	if provider == "" {
		WriteErrorResponse(w, http.StatusBadRequest, "Provider is required", "MISSING_PROVIDER")
		return
	}

	// Validate provider - only allow supported providers
	if _, exists := supportedProviders[provider]; !exists {
		WriteErrorResponse(w, http.StatusBadRequest, "Provider not supported. Supported providers: anthropic, openai, openrouter, gemini", "INVALID_PROVIDER")
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
			_ = h.app.Analytics.TrackProviderAuth(ctx, provider, false, "delete_credentials")
		}
		WriteErrorResponse(w, http.StatusInternalServerError, "Failed to delete API key", "DELETION_ERROR")
		return
	}

	// Track successful credential deletion
	if h.app.Analytics != nil {
		_ = h.app.Analytics.TrackProviderAuth(ctx, provider, true, "delete_credentials")
	}

	// Also clear OAuth credentials if this provider supports OAuth
	if provider == providerAnthropic || provider == "openai" {
		if err := credentialsService.DeleteOAuthCredentials(ctx, provider); err != nil {
			logging.Warn("Failed to delete OAuth credentials", "error", err, "provider", provider)
		}
	}

	response := GenericSuccessResponse{
		Status:   "success",
		Provider: provider,
		Message:  "Credentials deleted successfully",
	}

	WriteJSONResponse(w, http.StatusOK, response)
}

// HandleAuthStatus handles GET /api/auth/status
func (h *AuthHandler) HandleAuthStatus(w http.ResponseWriter, r *http.Request) {
	setCORSHeaders(w)
	if handleCORSPreflight(w, r) {
		return
	}

	if r.Method != http.MethodGet {
		WriteErrorResponse(w, http.StatusMethodNotAllowed, constants.MethodNotAllowed, "METHOD_NOT_ALLOWED")
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

	if r.Method != http.MethodPost {
		WriteErrorResponse(w, http.StatusMethodNotAllowed, constants.MethodNotAllowed, "METHOD_NOT_ALLOWED")
		return
	}

	provider := r.PathValue("provider")
	if provider == "" {
		WriteErrorResponse(w, http.StatusBadRequest, "Provider is required", "MISSING_PROVIDER")
		return
	}

	// Validate provider - only allow supported providers
	if _, exists := supportedProviders[provider]; !exists {
		WriteErrorResponse(w, http.StatusBadRequest, "Provider not supported. Supported providers: anthropic, openai, openrouter, gemini", "INVALID_PROVIDER")
		return
	}

	// Currently only Anthropic supports OAuth
	if provider != providerAnthropic {
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

	response := OAuthAuthURLResponse{
		AuthURL: authURL,
		State:   oauthFlow.State,
		Message: "Open the auth_url in your browser to complete OAuth authentication",
	}

	WriteJSONResponse(w, http.StatusOK, response)
}

// HandleValidatePreferredProvider handles GET /api/auth/validate
func (h *AuthHandler) HandleValidatePreferredProvider(w http.ResponseWriter, r *http.Request) {
	setCORSHeaders(w)
	if handleCORSPreflight(w, r) {
		return
	}

	if r.Method != http.MethodGet {
		WriteErrorResponse(w, http.StatusMethodNotAllowed, constants.MethodNotAllowed, "METHOD_NOT_ALLOWED")
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
		// Return 400 Bad Request - user needs to set a preferred provider first
		WriteErrorResponse(w, http.StatusBadRequest, "No preferred provider set. Please set a preferred provider first", "NO_PREFERRED_PROVIDER")
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
			if h.checkOAuthCredentials(providerAnthropic) {
				isAuthenticated = true
				authMethod = "oauth"
			}
		}
	}

	message := fmt.Sprintf("Please authenticate with %s first", preferredProvider)
	if isAuthenticated {
		message = fmt.Sprintf("Ready to use %s", preferredProvider)
	}

	response := ValidateAuthResponse{
		Valid:      isAuthenticated,
		Provider:   string(preferredProvider),
		AuthMethod: models.AuthMethod(authMethod),
		Message:    message,
	}

	WriteJSONResponse(w, http.StatusOK, response)
}

// supportedProviders defines the limited set of providers we support
var supportedProviders = map[string]struct{}{
	providerAnthropic: {},
	"openai":          {},
	"openrouter":      {},
	"gemini":          {},
	"brave":           {},
}

// HandleOAuthCallback handles POST /api/auth/oauth/callback
func (h *AuthHandler) HandleOAuthCallback(w http.ResponseWriter, r *http.Request) {
	setCORSHeaders(w)
	if handleCORSPreflight(w, r) {
		return
	}

	if r.Method != http.MethodPost {
		WriteErrorResponse(w, http.StatusMethodNotAllowed, constants.MethodNotAllowed, "METHOD_NOT_ALLOWED")
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
	if req.Provider != providerAnthropic {
		// Track failed authentication attempt for unsupported provider
		if h.app.Analytics != nil {
			_ = h.app.Analytics.TrackProviderAuth(r.Context(), req.Provider, false, "oauth")
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
			_ = h.app.Analytics.TrackProviderAuth(r.Context(), req.Provider, false, "oauth")
		}
		logging.Error("Invalid or expired OAuth state", "state", req.State)
		WriteErrorResponse(w, http.StatusBadRequest, "Invalid or expired OAuth state", "INVALID_STATE")
		return
	}

	// Format the code with state as expected by the ExchangeCodeForTokens method
	authCode := fmt.Sprintf("%s#%s", req.Code, req.State)

	// Exchange code for tokens
	oauthTokens, err := oauthFlow.ExchangeCodeForTokens(authCode)
	if err != nil {
		logging.Error("Failed to exchange authorization code for tokens", "error", err)
		// Track failed token exchange
		if h.app.Analytics != nil {
			_ = h.app.Analytics.TrackProviderAuth(r.Context(), req.Provider, false, "oauth")
		}
		WriteErrorResponse(w, http.StatusInternalServerError, "Failed to exchange authorization code for tokens", "OAUTH_ERROR")
		return
	}

	// Get credentials service
	credentialsService := config.GetAPICredentials()
	if credentialsService == nil {
		logging.Error("Failed to get credentials service")
		// Track failure due to storage initialization
		if h.app.Analytics != nil {
			_ = h.app.Analytics.TrackProviderAuth(r.Context(), req.Provider, false, "oauth")
		}
		WriteErrorResponse(w, http.StatusInternalServerError, "Credentials service unavailable", "STORAGE_ERROR")
		return
	}

	// Store the OAuth credentials in database
	oauthCreds := &credentials.OAuthCredentials{
		AccessToken:  oauthTokens.AccessToken,
		RefreshToken: oauthTokens.RefreshToken,
		ExpiresAt:    oauthTokens.ExpiresAt,
		ClientID:     oauthTokens.ClientID,
		Provider:     providerAnthropic,
	}

	err = credentialsService.StoreOAuthCredentials(r.Context(), providerAnthropic, oauthCreds)
	if err != nil {
		logging.Error("Failed to store OAuth credentials", "error", err)
		// Track failure due to storage error
		if h.app.Analytics != nil {
			_ = h.app.Analytics.TrackProviderAuth(r.Context(), req.Provider, false, "oauth")
		}
		WriteErrorResponse(w, http.StatusInternalServerError, "Failed to store OAuth credentials", "STORAGE_ERROR")
		return
	}

	// Clean up the OAuth flow
	llmprovider.CleanupOAuthFlow(req.State)

	// Refresh all agent providers to pick up the new OAuth credentials
	// This is critical for title generation and summarization to work
	if h.app.CoderAgent != nil {
		h.app.CoderAgent.ClearAllSessionProviders()
		logging.Info("Refreshed agent providers after OAuth login")
	}

	// Track successful authentication
	if h.app.Analytics != nil {
		ctx := r.Context()
		_ = h.app.Analytics.TrackProviderAuth(ctx, req.Provider, true, "oauth")
	}

	// Return success response
	response := OAuthCallbackResponse{
		Status:    "success",
		Provider:  req.Provider,
		Message:   "OAuth authentication successful",
		ExpiresIn: oauthTokens.ExpiresAt - time.Now().Unix(),
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
		}
	}

	// Check each supported provider (database-only authentication)
	providers := []struct {
		name          string
		provider      models.ModelProvider
		displayName   string
		supportsOAuth bool
	}{
		{providerAnthropic, models.ProviderAnthropic, "Anthropic", true},
		{"openai", models.ProviderOpenAI, "OpenAI", false},
		{"openrouter", models.ProviderOpenRouter, "OpenRouter", false},
		// {"gemini", models.ProviderGemini, "Google Gemini", false},
	}

	for _, p := range providers {
		var authenticated bool
		var authMethod models.AuthMethod

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
		authMethod = models.AuthMethod(getAuthMethod(hasAPIKey, hasOAuth))

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
	if provider != providerAnthropic {
		return false
	}

	// Get credentials service
	credentialsService := config.GetAPICredentials()
	if credentialsService == nil {
		logging.Warn("Credentials service unavailable")
		return false
	}

	// Check for OAuth credentials in database
	creds, err := credentialsService.GetOAuthCredentials(context.Background(), provider)
	if errors.Is(err, credentials.ErrOAuthCredentialNotFound) {
		return false
	}
	if err != nil {
		logging.Warn("Failed to get OAuth credentials", "error", err)
		return false
	}
	return !creds.IsTokenExpired()
}

// HandleRefreshTokens handles POST /internal/auth/refresh-tokens
// Manually triggers OAuth token refresh for all expired tokens
func (h *AuthHandler) HandleRefreshTokens(w http.ResponseWriter, r *http.Request) {
	setCORSHeaders(w)
	if handleCORSPreflight(w, r) {
		return
	}

	if r.Method != http.MethodPost {
		WriteErrorResponse(w, http.StatusMethodNotAllowed, constants.MethodNotAllowed, "METHOD_NOT_ALLOWED")
		return
	}

	if h.tokenRefreshService == nil {
		WriteErrorResponse(w, http.StatusInternalServerError, "Token refresh service not available", "SERVICE_UNAVAILABLE")
		return
	}

	ctx := r.Context()

	// Trigger manual refresh
	h.tokenRefreshService.RefreshExpiredTokens(ctx)

	response := GenericSuccessResponse{
		Status:  "success",
		Message: "Token refresh triggered successfully",
	}

	WriteJSONResponse(w, http.StatusOK, response)
}

// HandleOAuthHealth handles GET /health/auth
// Returns the health status of all OAuth credentials
func (h *AuthHandler) HandleOAuthHealth(w http.ResponseWriter, r *http.Request) {
	setCORSHeaders(w)
	if handleCORSPreflight(w, r) {
		return
	}

	if r.Method != http.MethodGet {
		WriteErrorResponse(w, http.StatusMethodNotAllowed, constants.MethodNotAllowed, "METHOD_NOT_ALLOWED")
		return
	}

	if h.tokenRefreshService == nil {
		WriteErrorResponse(w, http.StatusInternalServerError, "Token refresh service not available", "SERVICE_UNAVAILABLE")
		return
	}

	ctx := r.Context()
	status, err := h.tokenRefreshService.GetStatus(ctx)
	if err != nil {
		logging.Error("Failed to get OAuth health status", "error", err)
		WriteErrorResponse(w, http.StatusInternalServerError, "Failed to get OAuth health status", "HEALTH_CHECK_ERROR")
		return
	}

	// Determine overall health
	overallHealth := "healthy"
	hasExpired := false
	hasExpiredNoRefresh := false

	for _, tokenStatus := range status {
		switch tokenStatus.Status {
		case "expired":
			hasExpired = true
		case "expired_no_refresh":
			hasExpiredNoRefresh = true
		}
	}

	if hasExpiredNoRefresh {
		overallHealth = "unhealthy"
	} else if hasExpired {
		overallHealth = "degraded"
	}

	response := OAuthHealthResponse{
		Status:    overallHealth,
		Providers: status,
		Timestamp: time.Now().Format(time.RFC3339),
	}

	WriteJSONResponse(w, http.StatusOK, response)
}
