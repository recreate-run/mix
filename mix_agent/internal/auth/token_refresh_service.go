package auth

import (
	"context"
	"errors"
	"time"

	"mix/internal/credentials"
	"mix/internal/llm/provider"
	"mix/internal/logging"
)

// TokenRefreshService handles automatic background refresh of OAuth tokens
type TokenRefreshService struct {
	credentialsService *credentials.APICredentialsService
	checkInterval      time.Duration
	stopChan           chan struct{}
}

// NewTokenRefreshService creates a new token refresh service
func NewTokenRefreshService(credentialsService *credentials.APICredentialsService, checkInterval time.Duration) *TokenRefreshService {
	return &TokenRefreshService{
		credentialsService: credentialsService,
		checkInterval:      checkInterval,
		stopChan:           make(chan struct{}),
	}
}

// Start begins the background token refresh service
func (s *TokenRefreshService) Start(ctx context.Context) {
	// Run initial refresh check immediately
	s.RefreshExpiredTokens(ctx)

	ticker := time.NewTicker(s.checkInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			s.RefreshExpiredTokens(ctx)
		case <-ctx.Done():
			logging.Info("Token refresh service stopped due to context cancellation")
			return
		case <-s.stopChan:
			logging.Info("Token refresh service stopped")
			return
		}
	}
}

// Stop stops the background token refresh service
func (s *TokenRefreshService) Stop() {
	close(s.stopChan)
}

// RefreshExpiredTokens refreshes all OAuth tokens that are expiring soon
// Uses a 35-minute buffer to ensure tokens are refreshed before they're considered expired by IsTokenExpired()
// With 30-minute background checks + 5-minute safety margin = zero downtime
func (s *TokenRefreshService) RefreshExpiredTokens(ctx context.Context) {
	// Get tokens expiring within 35 minutes (database query buffer matches IsTokenExpired buffer)
	expiredCreds, err := s.credentialsService.GetExpiredOAuthCredentials(ctx)
	if err != nil {
		logging.Error("Failed to get expired credentials", "error", err)
		return
	}

	if len(expiredCreds) == 0 {
		return
	}

	logging.Info("Found tokens expiring soon, refreshing now", "count", len(expiredCreds))

	for _, cred := range expiredCreds {
		s.refreshSingleToken(ctx, cred)
	}
}

// refreshSingleToken refreshes a single OAuth token
func (s *TokenRefreshService) refreshSingleToken(ctx context.Context, cred *credentials.OAuthCredentials) {
	if cred.RefreshToken == "" {
		logging.Warn("No refresh token available for provider - manual re-authentication required", "provider", cred.Provider)
		// TODO: Add alerting mechanism here (e.g., webhook, email, Slack)
		return
	}

	// Convert to provider format for refresh
	oauthCred := &provider.OAuthCredentials{
		AccessToken:  cred.AccessToken,
		RefreshToken: cred.RefreshToken,
		ExpiresAt:    cred.ExpiresAt,
		ClientID:     cred.ClientID,
		Provider:     cred.Provider,
	}

	// Attempt refresh
	refreshed, err := provider.RefreshAccessToken(oauthCred)
	if err != nil {
		logging.Error("Failed to refresh OAuth token", "provider", cred.Provider, "error", err)
		// TODO: Add alerting mechanism here (e.g., webhook, email, Slack)
		return
	}

	// Store refreshed credentials
	newCreds := &credentials.OAuthCredentials{
		AccessToken:  refreshed.AccessToken,
		RefreshToken: refreshed.RefreshToken,
		ExpiresAt:    refreshed.ExpiresAt,
		ClientID:     refreshed.ClientID,
		Provider:     cred.Provider,
		LastRefresh:  time.Now().Format(time.RFC3339),
	}

	if err := s.credentialsService.StoreOAuthCredentials(ctx, cred.Provider, newCreds); err != nil {
		logging.Error("Failed to store refreshed OAuth credentials", "provider", cred.Provider, "error", err)
		return
	}

	logging.Info("Successfully refreshed OAuth token", "provider", cred.Provider, "expiresAt", time.Unix(refreshed.ExpiresAt, 0).Format(time.RFC3339))
}

// GetStatus returns the current status of all OAuth credentials
func (s *TokenRefreshService) GetStatus(ctx context.Context) (map[string]TokenStatus, error) {
	providers, err := s.credentialsService.ListOAuthCredentials(ctx)
	if err != nil {
		return nil, err
	}

	status := make(map[string]TokenStatus)
	for _, providerName := range providers {
		cred, err := s.credentialsService.GetOAuthCredentials(ctx, providerName)
		if errors.Is(err, credentials.ErrOAuthCredentialNotFound) {
			status[providerName] = TokenStatus{
				Provider: providerName,
				Status:   "not_found",
			}
			continue
		}
		if err != nil {
			status[providerName] = TokenStatus{
				Provider: providerName,
				Status:   "error",
				Error:    err.Error(),
			}
			continue
		}

		tokenStatus := TokenStatus{
			Provider:    providerName,
			ExpiresAt:   time.Unix(cred.ExpiresAt, 0),
			LastRefresh: cred.LastRefresh,
		}

		if cred.IsTokenExpired() {
			tokenStatus.Status = "expired"
			if cred.RefreshToken == "" {
				tokenStatus.Status = "expired_no_refresh"
			}
		} else {
			tokenStatus.Status = "active"
			// Calculate time until expiry
			expiresIn := time.Until(time.Unix(cred.ExpiresAt, 0))
			tokenStatus.ExpiresIn = expiresIn.String()
		}

		status[providerName] = tokenStatus
	}

	return status, nil
}

// TokenStatus represents the status of an OAuth token
type TokenStatus struct {
	Provider    string    `json:"provider"`
	Status      string    `json:"status"` // "active", "expired", "expired_no_refresh", "error", "not_found"
	ExpiresAt   time.Time `json:"expires_at,omitempty"`
	ExpiresIn   string    `json:"expires_in,omitempty"`
	LastRefresh string    `json:"last_refresh,omitempty"`
	Error       string    `json:"error,omitempty"`
}
