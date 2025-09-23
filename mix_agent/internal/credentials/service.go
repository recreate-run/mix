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
	"sync"

	"mix/internal/db"
	"mix/internal/llm/models"
	"mix/internal/logging"
	"mix/internal/tools"
)

// APICredentialsService handles encrypted API key storage and retrieval
type APICredentialsService struct {
	queries          *db.Queries
	encryptionKey    []byte
	credentialsCache sync.Map // Provider -> API Key cache
}

// NewAPICredentialsService creates a new API credentials service
func NewAPICredentialsService(database *sql.DB, encryptionKey []byte) *APICredentialsService {
	service := &APICredentialsService{
		queries:          db.New(database),
		encryptionKey:    encryptionKey,
		credentialsCache: sync.Map{},
	}

	// Preload credentials in the background
	go service.PreloadCredentials(context.Background())

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
		logging.Warn("Empty ciphertext provided for decryption")
		return "", nil
	}

	logging.Debug("Decoding base64 ciphertext")
	data, err := base64.StdEncoding.DecodeString(ciphertext)
	if err != nil {
		logging.Error("Failed to decode base64", "error", err)
		return "", fmt.Errorf("failed to decode base64: %w", err)
	}
	logging.Debug("Base64 decoding successful", "decodedLength", len(data))

	if acs.encryptionKey == nil {
		logging.Error("Encryption key is nil")
		return "", fmt.Errorf("encryption key is nil")
	}
	logging.Debug("Creating cipher with encryption key", "keyLength", len(acs.encryptionKey))

	block, err := aes.NewCipher(acs.encryptionKey)
	if err != nil {
		logging.Error("Failed to create cipher", "error", err)
		return "", fmt.Errorf("failed to create cipher: %w", err)
	}

	logging.Debug("Creating GCM from cipher")
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		logging.Error("Failed to create GCM", "error", err)
		return "", fmt.Errorf("failed to create GCM: %w", err)
	}

	if len(data) < gcm.NonceSize() {
		logging.Error("Invalid ciphertext: too short", "dataLength", len(data), "requiredNonceSize", gcm.NonceSize())
		return "", fmt.Errorf("invalid ciphertext: too short for nonce")
	}

	nonce := data[:gcm.NonceSize()]
	cipherBytes := data[gcm.NonceSize():]
	logging.Debug("Extracted nonce and ciphertext", "nonceSize", len(nonce), "cipherBytesSize", len(cipherBytes))

	logging.Debug("Attempting GCM Open operation")
	plaintext, err := gcm.Open(nil, nonce, cipherBytes, nil)
	if err != nil {
		logging.Error("Failed to decrypt using GCM", "error", err)
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
		Column3:  "provider",
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
	credential, err := acs.queries.GetAPICredential(ctx, db.GetAPICredentialParams{
		Provider: string(provider),
		ToolType: "provider",
	})
	if err != nil {
		if err == sql.ErrNoRows {
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
	count, err := acs.queries.HasAPICredential(ctx, db.HasAPICredentialParams{
		Provider: string(provider),
		ToolType: "provider",
	})
	if err != nil {
		return false, fmt.Errorf("failed to check API credential: %w", err)
	}

	// Don't update the cache here - GetAPIKey will do that when the actual key is needed
	return count > 0, nil
}

// DeleteAPIKey removes the API key for a provider
func (acs *APICredentialsService) DeleteAPIKey(ctx context.Context, provider models.ModelProvider) error {
	err := acs.queries.DeleteAPICredential(ctx, db.DeleteAPICredentialParams{
		Provider: string(provider),
		ToolType: "provider",
	})
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

	var providers []models.ModelProvider
	for _, cred := range credentials {
		if cred.ApiKey.Valid && cred.ApiKey.String != "" && cred.ToolType == "provider" {
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

		// Handle both provider and tool credentials
		if cred.ToolType == "provider" {
			provider := models.ModelProvider(cred.Provider)
			decryptedKey, err := acs.decrypt(cred.ApiKey.String)
			if err != nil {
				logging.Error("Failed to decrypt API key during preload", "provider", provider, "error", err)
				continue
			}
			// Store in cache
			acs.credentialsCache.Store(provider, decryptedKey)
			count++
		} else {
			// Handle tool credentials
			cacheKey := fmt.Sprintf("tool_%s_%s", cred.ToolType, cred.Provider)
			decryptedKey, err := acs.decrypt(cred.ApiKey.String)
			if err != nil {
				logging.Error("Failed to decrypt tool API key during preload", "tool", cred.Provider, "error", err)
				continue
			}
			// Store in cache
			acs.credentialsCache.Store(cacheKey, decryptedKey)
			count++
		}
	}
}

// Tool-specific credential management methods

// SetToolAPIKey stores an encrypted API key for a tool
func (acs *APICredentialsService) SetToolAPIKey(ctx context.Context, toolType tools.ToolType, provider tools.ToolProvider, apiKey string) error {
	// Validate the API key using the tool registry
	registry := tools.GetRegistry()
	if err := registry.ValidateAPIKey(toolType, provider, apiKey); err != nil {
		return fmt.Errorf("API key validation failed: %w", err)
	}

	// Encrypt the API key
	encryptedKey, err := acs.encrypt(apiKey)
	if err != nil {
		return fmt.Errorf("failed to encrypt API key: %w", err)
	}

	// Store in database
	providerID := tools.GetProviderIdentifier(toolType, provider)
	_, err = acs.queries.UpsertAPICredential(ctx, db.UpsertAPICredentialParams{
		Provider: providerID,
		ApiKey:   sql.NullString{String: encryptedKey, Valid: true},
		Column3:  string(toolType),
	})
	if err != nil {
		return fmt.Errorf("failed to store tool API credential: %w", err)
	}

	// Update cache
	cacheKey := fmt.Sprintf("tool_%s_%s", toolType, provider)
	acs.credentialsCache.Store(cacheKey, apiKey)

	return nil
}

// GetToolAPIKey retrieves a decrypted API key for a tool
func (acs *APICredentialsService) GetToolAPIKey(ctx context.Context, toolType tools.ToolType, provider tools.ToolProvider) (string, error) {
	cacheKey := fmt.Sprintf("tool_%s_%s", toolType, provider)
	
	// Check cache first
	if cachedValue, exists := acs.credentialsCache.Load(cacheKey); exists {
		return cachedValue.(string), nil
	}

	// Get from database
	providerID := tools.GetProviderIdentifier(toolType, provider)
	credential, err := acs.queries.GetToolCredential(ctx, db.GetToolCredentialParams{
		Provider: providerID,
		ToolType: string(toolType),
	})
	if err != nil {
		if err == sql.ErrNoRows {
			return "", fmt.Errorf("no API key found for tool %s/%s", toolType, provider)
		}
		return "", fmt.Errorf("failed to get tool API credential: %w", err)
	}

	if !credential.ApiKey.Valid || credential.ApiKey.String == "" {
		return "", fmt.Errorf("API key is empty for tool %s/%s", toolType, provider)
	}

	// Decrypt the API key
	decryptedKey, err := acs.decrypt(credential.ApiKey.String)
	if err != nil {
		return "", fmt.Errorf("failed to decrypt tool API key: %w", err)
	}

	// Cache the decrypted key
	acs.credentialsCache.Store(cacheKey, decryptedKey)

	return decryptedKey, nil
}

// HasToolAPIKey checks if a tool has an API key stored
func (acs *APICredentialsService) HasToolAPIKey(ctx context.Context, toolType tools.ToolType, provider tools.ToolProvider) (bool, error) {
	cacheKey := fmt.Sprintf("tool_%s_%s", toolType, provider)
	
	// Check cache first
	if cachedValue, exists := acs.credentialsCache.Load(cacheKey); exists {
		return cachedValue.(string) != "", nil
	}

	// Check database
	providerID := tools.GetProviderIdentifier(toolType, provider)
	count, err := acs.queries.HasAPICredential(ctx, db.HasAPICredentialParams{
		Provider: providerID,
		ToolType: string(toolType),
	})
	if err != nil {
		return false, fmt.Errorf("failed to check tool API credential: %w", err)
	}

	return count > 0, nil
}

// DeleteToolAPIKey removes the API key for a tool
func (acs *APICredentialsService) DeleteToolAPIKey(ctx context.Context, toolType tools.ToolType, provider tools.ToolProvider) error {
	providerID := tools.GetProviderIdentifier(toolType, provider)
	err := acs.queries.DeleteAPICredential(ctx, db.DeleteAPICredentialParams{
		Provider: providerID,
		ToolType: string(toolType),
	})
	if err != nil {
		return fmt.Errorf("failed to delete tool API credential: %w", err)
	}

	// Remove from cache
	cacheKey := fmt.Sprintf("tool_%s_%s", toolType, provider)
	acs.credentialsCache.Delete(cacheKey)

	return nil
}

// ListToolCredentials returns a list of tools that have stored API keys
func (acs *APICredentialsService) ListToolCredentials(ctx context.Context, toolType tools.ToolType) ([]tools.ToolProvider, error) {
	credentials, err := acs.queries.ListToolCredentials(ctx, string(toolType))
	if err != nil {
		return nil, fmt.Errorf("failed to list tool credentials: %w", err)
	}

	var providers []tools.ToolProvider
	for _, cred := range credentials {
		if cred.ApiKey.Valid && cred.ApiKey.String != "" {
			// Extract provider from the stored identifier
			_, provider, err := tools.ParseProviderIdentifier(cred.Provider)
			if err != nil {
				logging.Error("Failed to parse provider identifier", "provider", cred.Provider, "error", err)
				continue
			}
			providers = append(providers, provider)
		}
	}

	return providers, nil
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
