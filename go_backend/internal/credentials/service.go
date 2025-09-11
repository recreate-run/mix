package credentials

import (
	"context"
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"database/sql"
	"encoding/base64"
	"fmt"
	"io"
	"strings"

	"mix/internal/db"
	"mix/internal/llm/models"
	"mix/internal/logging"
	llmprovider "mix/internal/llm/provider"
)

// APICredentialsService handles encrypted API key storage and retrieval
type APICredentialsService struct {
	queries       *db.Queries
	encryptionKey []byte
}

// NewAPICredentialsService creates a new API credentials service
func NewAPICredentialsService(database *sql.DB, encryptionKey []byte) *APICredentialsService {
	return &APICredentialsService{
		queries:       db.New(database),
		encryptionKey: encryptionKey,
	}
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
	if ciphertext == "" {
		return "", nil
	}

	data, err := base64.StdEncoding.DecodeString(ciphertext)
	if err != nil {
		return "", fmt.Errorf("failed to decode base64: %w", err)
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
		return "", fmt.Errorf("invalid ciphertext")
	}

	nonce := data[:gcm.NonceSize()]
	cipherBytes := data[gcm.NonceSize():]

	plaintext, err := gcm.Open(nil, nonce, cipherBytes, nil)
	if err != nil {
		return "", fmt.Errorf("failed to decrypt: %w", err)
	}

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
		ApiKey:   encryptedKey,
	})
	if err != nil {
		return fmt.Errorf("failed to store API credential: %w", err)
	}

	logging.Info("API key stored successfully", "provider", provider)
	return nil
}

// GetAPIKey retrieves and decrypts an API key for a provider
func (acs *APICredentialsService) GetAPIKey(ctx context.Context, provider models.ModelProvider) (string, error) {
	credential, err := acs.queries.GetAPICredential(ctx, string(provider))
	if err != nil {
		if err == sql.ErrNoRows {
			return "", nil // No credential found
		}
		return "", fmt.Errorf("failed to get API credential: %w", err)
	}

	decryptedKey, err := acs.decrypt(credential.ApiKey)
	if err != nil {
		return "", fmt.Errorf("failed to decrypt API key: %w", err)
	}

	return decryptedKey, nil
}

// HasAPIKey checks if a provider has a stored API key
func (acs *APICredentialsService) HasAPIKey(ctx context.Context, provider models.ModelProvider) (bool, error) {
	count, err := acs.queries.HasAPICredential(ctx, string(provider))
	if err != nil {
		return false, fmt.Errorf("failed to check API credential: %w", err)
	}
	return count > 0, nil
}

// DeleteAPIKey removes the API key for a provider
func (acs *APICredentialsService) DeleteAPIKey(ctx context.Context, provider models.ModelProvider) error {
	err := acs.queries.DeleteAPICredential(ctx, string(provider))
	if err != nil {
		return fmt.Errorf("failed to delete API credential: %w", err)
	}

	logging.Info("API key deleted successfully", "provider", provider)
	return nil
}

// ListCredentials returns a list of providers that have stored API keys (without the actual keys)
func (acs *APICredentialsService) ListCredentials(ctx context.Context) ([]models.ModelProvider, error) {
	credentials, err := acs.queries.ListAPICredentials(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to list API credentials: %w", err)
	}

	var providers []models.ModelProvider
	for _, cred := range credentials {
		if cred.ApiKey != "" {
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

	logging.Info("All API credentials deleted successfully")
	return nil
}

// supportedProviders defines the providers we support for API key storage
var supportedProviders = map[models.ModelProvider]struct{}{
	models.ProviderAnthropic:  {},
	models.ProviderOpenAI:     {},
	models.ProviderOpenRouter: {},
}

// ValidateAPIKey performs basic validation on an API key for a provider
func (acs *APICredentialsService) ValidateAPIKey(provider models.ModelProvider, apiKey string) error {
	if apiKey == "" {
		return fmt.Errorf("API key cannot be empty")
	}

	// Check if provider is supported
	if _, exists := supportedProviders[provider]; !exists {
		return fmt.Errorf("provider %s not supported. Supported providers: anthropic, openai, openrouter", provider)
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
	}

	return nil
}

// GenerateEncryptionKey generates a new 32-byte encryption key for AES-256
func GenerateEncryptionKey() ([]byte, error) {
	key := make([]byte, 32) // AES-256
	if _, err := rand.Read(key); err != nil {
		return nil, fmt.Errorf("failed to generate encryption key: %w", err)
	}
	return key, nil
}

// IsAuthenticated checks if a provider has valid credentials in the database.
// It returns:
// - isAuthenticated: true if valid credentials exist
// - authMethod: "api_key", "oauth", or "none"
// - error: any error encountered during credential checking
//
// If provider is empty, it will try to use the user's preferred provider from database.
func (acs *APICredentialsService) IsAuthenticated(ctx context.Context, provider models.ModelProvider) (bool, string, error) {
	// If provider is empty, try to get the user's preferred provider
	if provider == "" {
		// Get preferred provider from user preferences
		// Since we can't import config package (to avoid circular dependency),
		// we'll have to check if there are any API keys available
		providers, err := acs.ListCredentials(ctx)
		if err == nil && len(providers) > 0 {
			// Use the first available provider as a fallback
			provider = providers[0]
			logging.Info("Using first available provider from credentials", "provider", provider)
		} else {
			// Default to Anthropic if no providers found
			provider = models.ProviderAnthropic
			logging.Info("No provider specified and none found in database, defaulting to Anthropic")
		}
	}

	// Check if the provider is supported
	if _, exists := supportedProviders[provider]; !exists {
		return false, "none", fmt.Errorf("provider %s not supported", provider)
	}

	// First check for API key in database
	hasAPIKey, err := acs.HasAPIKey(ctx, provider)
	if err != nil {
		return false, "none", fmt.Errorf("failed to check API credential: %w", err)
	}
	if hasAPIKey {
		return true, "api_key", nil
	}

	// Check for OAuth credentials for supported providers
	if provider == models.ProviderAnthropic || provider == models.ProviderOpenAI {
		// Try to initialize credential storage
		storage, err := llmprovider.NewCredentialStorage()
		if err != nil {
			logging.Warn("Failed to initialize credential storage", "error", err)
			return false, "none", nil
		}

		// Different handling based on provider
		switch provider {
		case models.ProviderAnthropic:
			// Check for valid Anthropic OAuth credentials
			creds, err := storage.GetOAuthCredentials("anthropic")
			if err == nil && creds != nil && !creds.IsTokenExpired() {
				return true, "oauth", nil
			}

		case models.ProviderOpenAI:
			// Check for valid OpenAI OAuth credentials
			creds, err := storage.GetOpenAICredentials("openai")
			if err == nil && creds != nil && !creds.IsTokenExpired() {
				return true, "oauth", nil
			}
		}
	}

	// No valid credentials found
	return false, "none", nil
}