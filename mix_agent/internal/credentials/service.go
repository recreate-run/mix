package credentials

import (
	"context"
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"database/sql"
	"encoding/base64"
	"errors"
	"fmt"
	"io"
	"strings"
	"sync"
	"time"

	"mix/internal/db"
	"mix/internal/llm/models"
	"mix/internal/logging"
)

// ErrOAuthCredentialNotFound is returned when OAuth credentials are not found for a provider
var ErrOAuthCredentialNotFound = errors.New("OAuth credential not found")

// APICredentialsService handles encrypted API key and OAuth credential storage and retrieval
type APICredentialsService struct {
	queries          db.Querier
	encryptionKey    []byte
	credentialsCache sync.Map // Provider -> API Key cache
	oauthCache       sync.Map // Provider -> OAuth Credentials cache
}

// OAuthCredentials holds OAuth token information (matches provider/oauth.go structs)
type OAuthCredentials struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token,omitempty"`
	IDToken      string `json:"id_token,omitempty"`   // For OpenAI
	APIKey       string `json:"api_key,omitempty"`    // Generated API key (for OpenAI)
	AccountID    string `json:"account_id,omitempty"` // For OpenAI
	ExpiresAt    int64  `json:"expires_at"`
	ClientID     string `json:"client_id"`
	Provider     string `json:"provider"`
	LastRefresh  string `json:"last_refresh,omitempty"`
}

// IsTokenExpired checks if the OAuth token is expired or will expire soon (5 minutes buffer)
// This is used for runtime checks during API calls - only marks tokens as expired when truly about to expire
// The background refresh service uses a separate 35-minute buffer to refresh tokens well before this threshold
func (cred *OAuthCredentials) IsTokenExpired() bool {
	if cred.ExpiresAt == 0 {
		return false // No expiry time set
	}
	return time.Now().Unix() >= (cred.ExpiresAt - 300) // 5 minute buffer (5 * 60 = 300 seconds)
}

// NewAPICredentialsService creates a new API credentials service
func NewAPICredentialsService(database *sql.DB, encryptionKey []byte) *APICredentialsService {
	return NewAPICredentialsServiceWithQuerier(db.New(database), encryptionKey)
}

// NewAPICredentialsServiceWithQuerier creates a new API credentials service with a custom querier (for testing)
func NewAPICredentialsServiceWithQuerier(querier db.Querier, encryptionKey []byte) *APICredentialsService {
	return NewAPICredentialsServiceWithQuerierAndPreload(querier, encryptionKey, true)
}

// NewAPICredentialsServiceWithQuerierAndPreload creates a new API credentials service with optional preloading
func NewAPICredentialsServiceWithQuerierAndPreload(querier db.Querier, encryptionKey []byte, enablePreload bool) *APICredentialsService {
	service := &APICredentialsService{
		queries:          querier,
		encryptionKey:    encryptionKey,
		credentialsCache: sync.Map{},
		oauthCache:       sync.Map{},
	}

	if enablePreload {
		// Preload credentials in the background
		go service.PreloadCredentials(context.Background())
		go service.PreloadOAuthCredentials(context.Background())
	}

	return service
}

// encrypt encrypts plaintext using AES-GCM
func (acs *APICredentialsService) encrypt(plaintext string) (string, error) {
	if plaintext == "" {
		return "", nil
	}

	block, err := aes.NewCipher(acs.encryptionKey)
	if err != nil {
		return "", fmt.Errorf("failed to create cipher: %w", err)
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", fmt.Errorf("failed to create GCM: %w", err)
	}

	nonce := make([]byte, gcm.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return "", fmt.Errorf("failed to generate nonce: %w", err)
	}

	ciphertext := gcm.Seal(nonce, nonce, []byte(plaintext), nil)
	return base64.StdEncoding.EncodeToString(ciphertext), nil
}

// decrypt decrypts ciphertext using AES-GCM
func (acs *APICredentialsService) decrypt(ciphertext string) (string, error) {
	// Attempting to decrypt API key
	if ciphertext == "" {
		return "", nil
	}

	data, err := base64.StdEncoding.DecodeString(ciphertext)
	if err != nil {
		return "", fmt.Errorf("failed to decode base64: %w", err)
	}

	if acs.encryptionKey == nil {
		return "", fmt.Errorf("encryption key is nil")
	}

	block, err := aes.NewCipher(acs.encryptionKey)
	if err != nil {
		return "", fmt.Errorf("failed to create cipher: %w", err)
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", fmt.Errorf("failed to create GCM: %w", err)
	}

	if len(data) < gcm.NonceSize() {
		return "", fmt.Errorf("invalid ciphertext: too short for nonce")
	}

	nonce := data[:gcm.NonceSize()]
	cipherBytes := data[gcm.NonceSize():]

	plaintext, err := gcm.Open(nil, nonce, cipherBytes, nil)
	if err != nil {
		return "", fmt.Errorf("failed to decrypt: %w", err)
	}

	// API key successfully decrypted
	return string(plaintext), nil
}

// StoreAPIKey stores an encrypted API key for a provider
func (acs *APICredentialsService) StoreAPIKey(ctx context.Context, provider models.ModelProvider, apiKey string) error {
	encryptedKey, err := acs.encrypt(apiKey)
	if err != nil {
		return fmt.Errorf("failed to encrypt API key: %w", err)
	}

	_, err = acs.queries.UpsertAPICredential(ctx, db.UpsertAPICredentialParams{
		Provider: string(provider),
		ApiKey:   sql.NullString{String: encryptedKey, Valid: true},
	})
	if err != nil {
		return fmt.Errorf("failed to store API credential: %w", err)
	}

	// Update the cache with the new API key
	acs.credentialsCache.Store(provider, apiKey)

	// API key stored and cached
	return nil
}

// GetAPIKey retrieves and decrypts an API key for a provider
func (acs *APICredentialsService) GetAPIKey(ctx context.Context, provider models.ModelProvider) (string, error) {
	// Check the cache first
	if cachedValue, found := acs.credentialsCache.Load(provider); found {
		return cachedValue.(string), nil
	}

	// API key not in cache, retrieving from database
	credential, err := acs.queries.GetAPICredential(ctx, string(provider))
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			// No API key found in database
			// Cache the empty result to avoid repeated database lookups
			acs.credentialsCache.Store(provider, "")
			return "", nil // No credential found
		}
		logging.Error("Failed to get API credential from database", "provider", provider, "error", err)
		return "", fmt.Errorf("failed to get API credential: %w", err)
	}

	// API key found in database, attempting decryption
	decryptedKey, err := acs.decrypt(credential.ApiKey.String)
	if err != nil {
		logging.Error("Failed to decrypt API key", "provider", provider, "error", err)
		return "", fmt.Errorf("failed to decrypt API key: %w", err)
	}

	// API key decrypted and cached
	// Store the decrypted key in the cache
	acs.credentialsCache.Store(provider, decryptedKey)
	return decryptedKey, nil
}

// HasAPIKey checks if a provider has a stored API key
func (acs *APICredentialsService) HasAPIKey(ctx context.Context, provider models.ModelProvider) (bool, error) {
	// Check the cache first
	if cachedValue, found := acs.credentialsCache.Load(provider); found {
		// Using cached value for HasAPIKey check
		// If we have a non-empty string in cache, the key exists
		return cachedValue.(string) != "", nil
	}

	// Not in cache, check the database
	count, err := acs.queries.HasAPICredential(ctx, string(provider))
	if err != nil {
		return false, fmt.Errorf("failed to check API credential: %w", err)
	}

	// Don't update the cache here - GetAPIKey will do that when the actual key is needed
	return count > 0, nil
}

// DeleteAPIKey removes the API key for a provider
func (acs *APICredentialsService) DeleteAPIKey(ctx context.Context, provider models.ModelProvider) error {
	err := acs.queries.DeleteAPICredential(ctx, string(provider))
	if err != nil {
		return fmt.Errorf("failed to delete API credential: %w", err)
	}

	// Remove from cache
	acs.credentialsCache.Delete(provider)

	// API key deleted and cache updated
	return nil
}

// ListCredentials returns a list of providers that have stored API keys (without the actual keys)
func (acs *APICredentialsService) ListCredentials(ctx context.Context) ([]models.ModelProvider, error) {
	credentials, err := acs.queries.ListAPICredentials(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to list API credentials: %w", err)
	}

	providers := make([]models.ModelProvider, 0) // Initialize as empty slice instead of nil
	for _, cred := range credentials {
		if cred.ApiKey.Valid && cred.ApiKey.String != "" {
			providers = append(providers, models.ModelProvider(cred.Provider))
		}
	}

	return providers, nil
}

// DeleteAllCredentials removes all stored API keys
func (acs *APICredentialsService) DeleteAllCredentials(ctx context.Context) error {
	err := acs.queries.DeleteAllAPICredentials(ctx)
	if err != nil {
		return fmt.Errorf("failed to delete all API credentials: %w", err)
	}

	// Clear the entire cache
	acs.ClearCache()

	// All API credentials deleted and cache cleared
	return nil
}

// supportedProviders defines the providers we support for API key storage
var supportedProviders = map[models.ModelProvider]struct{}{
	models.ProviderAnthropic:  {},
	models.ProviderOpenAI:     {},
	models.ProviderOpenRouter: {},
	models.ProviderGemini:     {},
	"brave":                   {}, // External tool provider
}

// ClearCache removes all entries from the credentials cache
func (acs *APICredentialsService) ClearCache() {
	// Create a new empty map to replace the existing one
	acs.credentialsCache = sync.Map{}
	// API credentials cache cleared
}

// ClearProviderCache removes a specific provider's credentials from the cache
func (acs *APICredentialsService) ClearProviderCache(provider models.ModelProvider) {
	acs.credentialsCache.Delete(provider)
	// Provider cache cleared
}

// PreloadCredentials loads all credentials into the cache to avoid database hits
func (acs *APICredentialsService) PreloadCredentials(ctx context.Context) {
	// List all credentials from the database
	credentials, err := acs.queries.ListAPICredentials(ctx)
	if err != nil {
		logging.Error("Failed to preload API credentials", "error", err)
		return
	}

	count := 0
	for _, cred := range credentials {
		if !cred.ApiKey.Valid || cred.ApiKey.String == "" {
			continue
		}

		// Handle provider credentials (simplified schema with no tool_type)
		provider := models.ModelProvider(cred.Provider)
		decryptedKey, err := acs.decrypt(cred.ApiKey.String)
		if err != nil {
			logging.Error("Failed to decrypt API key during preload", "provider", provider, "error", err)
			continue
		}
		// Store in cache
		acs.credentialsCache.Store(provider, decryptedKey)
		count++
	}
}

// ValidateAPIKey performs basic validation on an API key for a provider
func (acs *APICredentialsService) ValidateAPIKey(provider models.ModelProvider, apiKey string) error {
	if apiKey == "" {
		return fmt.Errorf("API key cannot be empty")
	}

	// Check if provider is supported
	if _, exists := supportedProviders[provider]; !exists {
		return fmt.Errorf("provider %s not supported. Supported providers: anthropic, openai, openrouter, gemini", provider)
	}

	// Basic format validation per provider
	switch provider {
	case models.ProviderAnthropic:
		if len(apiKey) < 40 || !strings.HasPrefix(apiKey, "sk-ant-") {
			return fmt.Errorf("invalid Anthropic API key format")
		}
	case models.ProviderOpenAI:
		if len(apiKey) < 40 || !strings.HasPrefix(apiKey, "sk-") {
			return fmt.Errorf("invalid OpenAI API key format")
		}
	case models.ProviderOpenRouter:
		if len(apiKey) < 40 {
			return fmt.Errorf("invalid OpenRouter API key format")
		}
	case models.ProviderGemini:
		if len(apiKey) < 30 {
			return fmt.Errorf("invalid Gemini API key format")
		}
	case "brave":
		if len(apiKey) < 30 {
			return fmt.Errorf("invalid Brave API key format")
		}
	}

	return nil
}

// GenerateEncryptionKey returns a fixed 32-byte encryption key for AES-256
// This ensures the same key is used across application restarts for consistent encryption/decryption
func GenerateEncryptionKey() ([]byte, error) {
	// Fixed key for consistent encryption/decryption
	// Note: In production, this should ideally be stored securely and loaded from persistent storage
	key := []byte{
		0x0a, 0x1b, 0x2c, 0x3d, 0x4e, 0x5f, 0x6a, 0x7b,
		0x8c, 0x9d, 0xae, 0xbf, 0xc0, 0xd1, 0xe2, 0xf3,
		0x0a, 0x1b, 0x2c, 0x3d, 0x4e, 0x5f, 0x6a, 0x7b,
		0x8c, 0x9d, 0xae, 0xbf, 0xc0, 0xd1, 0xe2, 0xf3,
	}
	return key, nil
}

// OAuth credential methods

// StoreOAuthCredentials stores encrypted OAuth credentials for a provider
func (acs *APICredentialsService) StoreOAuthCredentials(ctx context.Context, provider string, creds *OAuthCredentials) error {
	encryptedAccessToken, err := acs.encrypt(creds.AccessToken)
	if err != nil {
		return fmt.Errorf("failed to encrypt access token: %w", err)
	}

	encryptedRefreshToken, err := acs.encrypt(creds.RefreshToken)
	if err != nil {
		return fmt.Errorf("failed to encrypt refresh token: %w", err)
	}

	encryptedIDToken, err := acs.encrypt(creds.IDToken)
	if err != nil {
		return fmt.Errorf("failed to encrypt ID token: %w", err)
	}

	encryptedAPIKey, err := acs.encrypt(creds.APIKey)
	if err != nil {
		return fmt.Errorf("failed to encrypt API key: %w", err)
	}

	_, err = acs.queries.UpsertOAuthCredential(ctx, db.UpsertOAuthCredentialParams{
		Provider:     provider,
		AccessToken:  sql.NullString{String: encryptedAccessToken, Valid: encryptedAccessToken != ""},
		RefreshToken: sql.NullString{String: encryptedRefreshToken, Valid: encryptedRefreshToken != ""},
		IDToken:      sql.NullString{String: encryptedIDToken, Valid: encryptedIDToken != ""},
		ApiKey:       sql.NullString{String: encryptedAPIKey, Valid: encryptedAPIKey != ""},
		AccountID:    sql.NullString{String: creds.AccountID, Valid: creds.AccountID != ""},
		ClientID:     creds.ClientID,
		ExpiresAt:    sql.NullInt64{Int64: creds.ExpiresAt, Valid: creds.ExpiresAt != 0},
		LastRefresh:  sql.NullString{String: creds.LastRefresh, Valid: creds.LastRefresh != ""},
	})
	if err != nil {
		return fmt.Errorf("failed to store OAuth credential: %w", err)
	}

	// Update the cache with the new credentials
	acs.oauthCache.Store(provider, creds)

	return nil
}

// GetOAuthCredentials retrieves and decrypts OAuth credentials for a provider
func (acs *APICredentialsService) GetOAuthCredentials(ctx context.Context, provider string) (*OAuthCredentials, error) {
	// Check the cache first
	if cachedValue, found := acs.oauthCache.Load(provider); found {
		return cachedValue.(*OAuthCredentials), nil
	}

	// OAuth credentials not in cache, retrieving from database
	credential, err := acs.queries.GetOAuthCredential(ctx, provider)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			// No OAuth credentials found in database
			return nil, ErrOAuthCredentialNotFound
		}
		return nil, fmt.Errorf("failed to get OAuth credential: %w", err)
	}

	// OAuth credentials found in database, attempting decryption
	accessToken, err := acs.decrypt(credential.AccessToken.String)
	if err != nil {
		return nil, fmt.Errorf("failed to decrypt access token: %w", err)
	}

	refreshToken, err := acs.decrypt(credential.RefreshToken.String)
	if err != nil {
		return nil, fmt.Errorf("failed to decrypt refresh token: %w", err)
	}

	idToken, err := acs.decrypt(credential.IDToken.String)
	if err != nil {
		return nil, fmt.Errorf("failed to decrypt ID token: %w", err)
	}

	apiKey, err := acs.decrypt(credential.ApiKey.String)
	if err != nil {
		return nil, fmt.Errorf("failed to decrypt API key: %w", err)
	}

	creds := &OAuthCredentials{
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
		IDToken:      idToken,
		APIKey:       apiKey,
		AccountID:    credential.AccountID.String,
		ExpiresAt:    credential.ExpiresAt.Int64,
		ClientID:     credential.ClientID,
		Provider:     provider,
		LastRefresh:  credential.LastRefresh.String,
	}

	// Store in cache
	acs.oauthCache.Store(provider, creds)
	return creds, nil
}

// HasOAuthCredentials checks if a provider has stored OAuth credentials
func (acs *APICredentialsService) HasOAuthCredentials(ctx context.Context, provider string) (bool, error) {
	// Check the cache first
	if _, found := acs.oauthCache.Load(provider); found {
		return true, nil
	}

	// Not in cache, check the database
	count, err := acs.queries.HasOAuthCredential(ctx, provider)
	if err != nil {
		return false, fmt.Errorf("failed to check OAuth credential: %w", err)
	}

	return count > 0, nil
}

// DeleteOAuthCredentials removes the OAuth credentials for a provider
func (acs *APICredentialsService) DeleteOAuthCredentials(ctx context.Context, provider string) error {
	err := acs.queries.DeleteOAuthCredential(ctx, provider)
	if err != nil {
		return fmt.Errorf("failed to delete OAuth credential: %w", err)
	}

	// Remove from cache
	acs.oauthCache.Delete(provider)

	return nil
}

// ListOAuthCredentials returns a list of providers that have stored OAuth credentials (without the actual tokens)
func (acs *APICredentialsService) ListOAuthCredentials(ctx context.Context) ([]string, error) {
	credentials, err := acs.queries.ListOAuthCredentials(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to list OAuth credentials: %w", err)
	}

	providers := make([]string, 0) // Initialize as empty slice instead of nil
	for i := range credentials {
		if credentials[i].AccessToken.Valid && credentials[i].AccessToken.String != "" {
			providers = append(providers, credentials[i].Provider)
		}
	}

	return providers, nil
}

// DeleteAllOAuthCredentials removes all stored OAuth credentials
func (acs *APICredentialsService) DeleteAllOAuthCredentials(ctx context.Context) error {
	err := acs.queries.DeleteAllOAuthCredentials(ctx)
	if err != nil {
		return fmt.Errorf("failed to delete all OAuth credentials: %w", err)
	}

	// Clear the OAuth cache
	acs.oauthCache = sync.Map{}

	return nil
}

// PreloadOAuthCredentials loads all OAuth credentials into the cache to avoid database hits
func (acs *APICredentialsService) PreloadOAuthCredentials(ctx context.Context) {
	// List all OAuth credentials from the database
	credentials, err := acs.queries.ListOAuthCredentials(ctx)
	if err != nil {
		logging.Error("Failed to preload OAuth credentials", "error", err)
		return
	}

	count := 0
	for i := range credentials {
		if !credentials[i].AccessToken.Valid || credentials[i].AccessToken.String == "" {
			continue
		}

		// Decrypt credentials
		accessToken, err := acs.decrypt(credentials[i].AccessToken.String)
		if err != nil {
			logging.Error("Failed to decrypt access token during preload", "provider", credentials[i].Provider, "error", err)
			continue
		}

		refreshToken, err := acs.decrypt(credentials[i].RefreshToken.String)
		if err != nil {
			logging.Error("Failed to decrypt refresh token during preload", "provider", credentials[i].Provider, "error", err)
			continue
		}

		idToken, err := acs.decrypt(credentials[i].IDToken.String)
		if err != nil {
			logging.Error("Failed to decrypt ID token during preload", "provider", credentials[i].Provider, "error", err)
			continue
		}

		apiKey, err := acs.decrypt(credentials[i].ApiKey.String)
		if err != nil {
			logging.Error("Failed to decrypt API key during preload", "provider", credentials[i].Provider, "error", err)
			continue
		}

		oauthCreds := &OAuthCredentials{
			AccessToken:  accessToken,
			RefreshToken: refreshToken,
			IDToken:      idToken,
			APIKey:       apiKey,
			AccountID:    credentials[i].AccountID.String,
			ExpiresAt:    credentials[i].ExpiresAt.Int64,
			ClientID:     credentials[i].ClientID,
			Provider:     credentials[i].Provider,
			LastRefresh:  credentials[i].LastRefresh.String,
		}

		// Store in cache
		acs.oauthCache.Store(credentials[i].Provider, oauthCreds)
		count++
	}
}

// GetExpiredOAuthCredentials returns OAuth credentials that are expired or will expire soon
func (acs *APICredentialsService) GetExpiredOAuthCredentials(ctx context.Context) ([]*OAuthCredentials, error) {
	credentials, err := acs.queries.ListExpiredOAuthCredentials(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to list expired OAuth credentials: %w", err)
	}

	expiredCreds := make([]*OAuthCredentials, 0, len(credentials))
	for i := range credentials {
		if !credentials[i].AccessToken.Valid || credentials[i].AccessToken.String == "" {
			continue
		}

		// Decrypt credentials
		accessToken, err := acs.decrypt(credentials[i].AccessToken.String)
		if err != nil {
			logging.Error("Failed to decrypt access token for expired credential", "provider", credentials[i].Provider, "error", err)
			continue
		}

		refreshToken, err := acs.decrypt(credentials[i].RefreshToken.String)
		if err != nil {
			logging.Error("Failed to decrypt refresh token for expired credential", "provider", credentials[i].Provider, "error", err)
			continue
		}

		idToken, err := acs.decrypt(credentials[i].IDToken.String)
		if err != nil {
			logging.Error("Failed to decrypt ID token for expired credential", "provider", credentials[i].Provider, "error", err)
			continue
		}

		apiKey, err := acs.decrypt(credentials[i].ApiKey.String)
		if err != nil {
			logging.Error("Failed to decrypt API key for expired credential", "provider", credentials[i].Provider, "error", err)
			continue
		}

		oauthCreds := &OAuthCredentials{
			AccessToken:  accessToken,
			RefreshToken: refreshToken,
			IDToken:      idToken,
			APIKey:       apiKey,
			AccountID:    credentials[i].AccountID.String,
			ExpiresAt:    credentials[i].ExpiresAt.Int64,
			ClientID:     credentials[i].ClientID,
			Provider:     credentials[i].Provider,
			LastRefresh:  credentials[i].LastRefresh.String,
		}

		expiredCreds = append(expiredCreds, oauthCreds)
	}

	return expiredCreds, nil
}
