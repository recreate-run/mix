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
	logging.Info("Attempting to decrypt API key", "ciphertextLength", len(ciphertext))
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

	logging.Info("Successfully decrypted API key", "plaintextLength", len(plaintext))
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
	logging.Info("Getting API key from database", "provider", provider)
	credential, err := acs.queries.GetAPICredential(ctx, string(provider))
	if err != nil {
		if err == sql.ErrNoRows {
			logging.Info("No API key found in database", "provider", provider, "error", "sql.ErrNoRows")
			return "", nil // No credential found
		}
		logging.Error("Failed to get API credential from database", "provider", provider, "error", err)
		return "", fmt.Errorf("failed to get API credential: %w", err)
	}

	logging.Info("API key found in database, attempting to decrypt", "provider", provider, "keyLength", len(credential.ApiKey))
	decryptedKey, err := acs.decrypt(credential.ApiKey)
	if err != nil {
		logging.Error("Failed to decrypt API key", "provider", provider, "error", err)
		return "", fmt.Errorf("failed to decrypt API key: %w", err)
	}

	logging.Info("API key successfully decrypted", "provider", provider, "keyLength", len(decryptedKey))
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

