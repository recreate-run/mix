package credentials

import (
	"context"
	"database/sql"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"mix/internal/db"
	"mix/internal/llm/models"
)

// Test helper functions
func CreateTestService(t *testing.T, mockQueries *db.MockQuerier) *APICredentialsService {
	t.Helper()
	encryptionKey, err := GenerateEncryptionKey()
	require.NoError(t, err)
	return NewAPICredentialsServiceWithQuerierAndPreload(mockQueries, encryptionKey, false)
}

func CreateTestAPIKey(provider models.ModelProvider) string {
	switch provider {
	case models.ProviderAnthropic:
		return "sk-ant-api03-testkey1234567890abcdef1234567890abcdef1234567890abcdef1234567890abcdef12"
	case models.ProviderOpenAI:
		return "sk-1234567890abcdef1234567890abcdef1234567890abcdef1234567890abcdef12"
	case models.ProviderOpenRouter:
		return "sk-or-v1-1234567890abcdef1234567890abcdef1234567890abcdef1234567890abcdef12"
	case models.ProviderGemini:
		return "AIzaSyTestKey1234567890abcdef123456"
	default:
		return "test-api-key-1234567890abcdef1234567890abcdef"
	}
}

func CreateTestOAuthCreds(provider string) *OAuthCredentials {
	return &OAuthCredentials{
		AccessToken:  "access_token_" + provider,
		RefreshToken: "refresh_token_" + provider,
		IDToken:      "id_token_" + provider,
		APIKey:       "api_key_" + provider,
		AccountID:    "account_" + provider,
		ExpiresAt:    time.Now().Add(1 * time.Hour).Unix(),
		ClientID:     "client_" + provider,
		Provider:     provider,
		LastRefresh:  time.Now().Format(time.RFC3339),
	}
}

// Test encryption/decryption functionality
func TestEncryptDecrypt(t *testing.T) {
	service := CreateTestService(t, &db.MockQuerier{})

	plaintext := "test-api-key-12345"
	encrypted, err := service.encrypt(plaintext)
	require.NoError(t, err)
	assert.NotEmpty(t, encrypted)

	decrypted, err := service.decrypt(encrypted)
	require.NoError(t, err)
	assert.Equal(t, plaintext, decrypted)
}

func TestEncryptDecryptEmpty(t *testing.T) {
	service := CreateTestService(t, &db.MockQuerier{})

	encrypted, err := service.encrypt("")
	require.NoError(t, err)
	assert.Empty(t, encrypted)

	decrypted, err := service.decrypt("")
	require.NoError(t, err)
	assert.Empty(t, decrypted)
}

// Test API key management
func TestStoreAPIKey(t *testing.T) {
	mockQueries := &db.MockQuerier{}
	service := CreateTestService(t, mockQueries)

	mockQueries.On("UpsertAPICredential", mock.Anything, mock.AnythingOfType("db.UpsertAPICredentialParams")).
		Return(db.ApiCredential{Provider: "anthropic"}, nil)

	err := service.StoreAPIKey(context.Background(), models.ProviderAnthropic, CreateTestAPIKey(models.ProviderAnthropic))
	require.NoError(t, err)

	// Check cache
	cached, found := service.credentialsCache.Load(models.ProviderAnthropic)
	assert.True(t, found)
	assert.Equal(t, CreateTestAPIKey(models.ProviderAnthropic), cached)

	mockQueries.AssertExpectations(t)
}

func TestGetAPIKeyFromCache(t *testing.T) {
	mockQueries := &db.MockQuerier{}
	service := CreateTestService(t, mockQueries)

	// Pre-populate cache
	service.credentialsCache.Store(models.ProviderAnthropic, "cached-key")

	key, err := service.GetAPIKey(context.Background(), models.ProviderAnthropic)
	require.NoError(t, err)
	assert.Equal(t, "cached-key", key)

	// Should not call DB
	mockQueries.AssertExpectations(t)
}

func TestGetAPIKeyFromDB(t *testing.T) {
	mockQueries := &db.MockQuerier{}
	service := CreateTestService(t, mockQueries)

	mockQueries.On("GetAPICredential", mock.Anything, "anthropic").
		Return(db.ApiCredential{
			Provider: "anthropic",
			ApiKey:   sql.NullString{String: "", Valid: false}, // Empty key
		}, nil)

	key, err := service.GetAPIKey(context.Background(), models.ProviderAnthropic)
	require.NoError(t, err)
	assert.Empty(t, key)

	mockQueries.AssertExpectations(t)
}

func TestGetAPIKeyNotFound(t *testing.T) {
	mockQueries := &db.MockQuerier{}
	service := CreateTestService(t, mockQueries)

	mockQueries.On("GetAPICredential", mock.Anything, "anthropic").
		Return(db.ApiCredential{}, sql.ErrNoRows)

	key, err := service.GetAPIKey(context.Background(), models.ProviderAnthropic)
	require.NoError(t, err)
	assert.Empty(t, key)

	mockQueries.AssertExpectations(t)
}

func TestHasAPIKey(t *testing.T) {
	mockQueries := &db.MockQuerier{}
	service := CreateTestService(t, mockQueries)

	// Test cache hit
	service.credentialsCache.Store(models.ProviderAnthropic, "some-key")
	has, err := service.HasAPIKey(context.Background(), models.ProviderAnthropic)
	require.NoError(t, err)
	assert.True(t, has)

	// Test DB call
	mockQueries.On("HasAPICredential", mock.Anything, "openai").Return(int64(1), nil)
	has, err = service.HasAPIKey(context.Background(), models.ProviderOpenAI)
	require.NoError(t, err)
	assert.True(t, has)

	mockQueries.AssertExpectations(t)
}

func TestDeleteAPIKey(t *testing.T) {
	mockQueries := &db.MockQuerier{}
	service := CreateTestService(t, mockQueries)

	// Pre-populate cache
	service.credentialsCache.Store(models.ProviderAnthropic, "some-key")

	mockQueries.On("DeleteAPICredential", mock.Anything, "anthropic").Return(nil)

	err := service.DeleteAPIKey(context.Background(), models.ProviderAnthropic)
	require.NoError(t, err)

	// Check cache is cleared
	_, found := service.credentialsCache.Load(models.ProviderAnthropic)
	assert.False(t, found)

	mockQueries.AssertExpectations(t)
}

func TestListCredentials(t *testing.T) {
	mockQueries := &db.MockQuerier{}
	service := CreateTestService(t, mockQueries)

	mockQueries.On("ListAPICredentials", mock.Anything).
		Return([]db.ApiCredential{
			{Provider: "anthropic", ApiKey: sql.NullString{String: "key1", Valid: true}},
			{Provider: "openai", ApiKey: sql.NullString{String: "key2", Valid: true}},
			{Provider: "gemini", ApiKey: sql.NullString{String: "", Valid: false}}, // Should be filtered out
		}, nil)

	providers, err := service.ListCredentials(context.Background())
	require.NoError(t, err)
	assert.Len(t, providers, 2)
	assert.Contains(t, providers, models.ModelProvider("anthropic"))
	assert.Contains(t, providers, models.ModelProvider("openai"))

	mockQueries.AssertExpectations(t)
}

func TestDeleteAllCredentials(t *testing.T) {
	mockQueries := &db.MockQuerier{}
	service := CreateTestService(t, mockQueries)

	// Pre-populate cache
	service.credentialsCache.Store(models.ProviderAnthropic, "key1")
	service.credentialsCache.Store(models.ProviderOpenAI, "key2")

	mockQueries.On("DeleteAllAPICredentials", mock.Anything).Return(nil)

	err := service.DeleteAllCredentials(context.Background())
	require.NoError(t, err)

	// Check cache is cleared
	_, found1 := service.credentialsCache.Load(models.ProviderAnthropic)
	_, found2 := service.credentialsCache.Load(models.ProviderOpenAI)
	assert.False(t, found1)
	assert.False(t, found2)

	mockQueries.AssertExpectations(t)
}

// Test API key validation
func TestValidateAPIKey(t *testing.T) {
	service := CreateTestService(t, &db.MockQuerier{})

	// Valid keys
	assert.NoError(t, service.ValidateAPIKey(models.ProviderAnthropic, CreateTestAPIKey(models.ProviderAnthropic)))
	assert.NoError(t, service.ValidateAPIKey(models.ProviderOpenAI, CreateTestAPIKey(models.ProviderOpenAI)))

	// Invalid keys
	require.Error(t, service.ValidateAPIKey(models.ProviderAnthropic, ""))
	require.Error(t, service.ValidateAPIKey(models.ProviderAnthropic, "short"))
	require.Error(t, service.ValidateAPIKey("unsupported", "any-key"))
}

// Test OAuth credentials
func TestStoreOAuthCredentials(t *testing.T) {
	mockQueries := &db.MockQuerier{}
	service := CreateTestService(t, mockQueries)

	mockQueries.On("UpsertOAuthCredential", mock.Anything, mock.AnythingOfType("db.UpsertOAuthCredentialParams")).
		Return(db.OauthCredential{Provider: "test_provider"}, nil)

	creds := CreateTestOAuthCreds("test_provider")
	err := service.StoreOAuthCredentials(context.Background(), "test_provider", creds)
	require.NoError(t, err)

	// Check cache
	cached, found := service.oauthCache.Load("test_provider")
	assert.True(t, found)
	assert.Equal(t, creds, cached)

	mockQueries.AssertExpectations(t)
}

func TestGetOAuthCredentialsFromCache(t *testing.T) {
	mockQueries := &db.MockQuerier{}
	service := CreateTestService(t, mockQueries)

	expectedCreds := CreateTestOAuthCreds("test_provider")
	service.oauthCache.Store("test_provider", expectedCreds)

	creds, err := service.GetOAuthCredentials(context.Background(), "test_provider")
	require.NoError(t, err)
	assert.Equal(t, expectedCreds, creds)

	// Should not call DB
	mockQueries.AssertExpectations(t)
}

func TestGetOAuthCredentialsNotFound(t *testing.T) {
	mockQueries := &db.MockQuerier{}
	service := CreateTestService(t, mockQueries)

	mockQueries.On("GetOAuthCredential", mock.Anything, "missing_provider").
		Return(db.OauthCredential{}, sql.ErrNoRows)

	creds, err := service.GetOAuthCredentials(context.Background(), "missing_provider")
	require.NoError(t, err)
	assert.Nil(t, creds)

	mockQueries.AssertExpectations(t)
}

func TestHasOAuthCredentials(t *testing.T) {
	mockQueries := &db.MockQuerier{}
	service := CreateTestService(t, mockQueries)

	// Test cache hit
	service.oauthCache.Store("cached_provider", CreateTestOAuthCreds("cached_provider"))
	has, err := service.HasOAuthCredentials(context.Background(), "cached_provider")
	require.NoError(t, err)
	assert.True(t, has)

	// Test DB call
	mockQueries.On("HasOAuthCredential", mock.Anything, "db_provider").Return(int64(1), nil)
	has, err = service.HasOAuthCredentials(context.Background(), "db_provider")
	require.NoError(t, err)
	assert.True(t, has)

	mockQueries.AssertExpectations(t)
}

func TestDeleteOAuthCredentials(t *testing.T) {
	mockQueries := &db.MockQuerier{}
	service := CreateTestService(t, mockQueries)

	// Pre-populate cache
	service.oauthCache.Store("test_provider", CreateTestOAuthCreds("test_provider"))

	mockQueries.On("DeleteOAuthCredential", mock.Anything, "test_provider").Return(nil)

	err := service.DeleteOAuthCredentials(context.Background(), "test_provider")
	require.NoError(t, err)

	// Check cache is cleared
	_, found := service.oauthCache.Load("test_provider")
	assert.False(t, found)

	mockQueries.AssertExpectations(t)
}

func TestListOAuthCredentials(t *testing.T) {
	mockQueries := &db.MockQuerier{}
	service := CreateTestService(t, mockQueries)

	mockQueries.On("ListOAuthCredentials", mock.Anything).
		Return([]db.OauthCredential{
			{Provider: "provider1", AccessToken: sql.NullString{String: "token1", Valid: true}},
			{Provider: "provider2", AccessToken: sql.NullString{String: "token2", Valid: true}},
			{Provider: "provider3", AccessToken: sql.NullString{String: "", Valid: false}}, // Should be filtered out
		}, nil)

	providers, err := service.ListOAuthCredentials(context.Background())
	require.NoError(t, err)
	assert.Len(t, providers, 2)
	assert.Contains(t, providers, "provider1")
	assert.Contains(t, providers, "provider2")

	mockQueries.AssertExpectations(t)
}

func TestDeleteAllOAuthCredentials(t *testing.T) {
	mockQueries := &db.MockQuerier{}
	service := CreateTestService(t, mockQueries)

	// Pre-populate cache
	service.oauthCache.Store("provider1", CreateTestOAuthCreds("provider1"))
	service.oauthCache.Store("provider2", CreateTestOAuthCreds("provider2"))

	mockQueries.On("DeleteAllOAuthCredentials", mock.Anything).Return(nil)

	err := service.DeleteAllOAuthCredentials(context.Background())
	require.NoError(t, err)

	// Check cache is cleared
	_, found1 := service.oauthCache.Load("provider1")
	_, found2 := service.oauthCache.Load("provider2")
	assert.False(t, found1)
	assert.False(t, found2)

	mockQueries.AssertExpectations(t)
}

func TestGetExpiredOAuthCredentials(t *testing.T) {
	mockQueries := &db.MockQuerier{}
	service := CreateTestService(t, mockQueries)

	mockQueries.On("ListExpiredOAuthCredentials", mock.Anything).
		Return([]db.OauthCredential{}, nil) // Return empty list

	creds, err := service.GetExpiredOAuthCredentials(context.Background())
	require.NoError(t, err)
	assert.Empty(t, creds)

	mockQueries.AssertExpectations(t)
}

// Test token expiration
func TestIsTokenExpired(t *testing.T) {
	// Not expired
	creds := &OAuthCredentials{ExpiresAt: time.Now().Add(10 * time.Minute).Unix()}
	assert.False(t, creds.IsTokenExpired())

	// Expired
	creds = &OAuthCredentials{ExpiresAt: time.Now().Add(-10 * time.Minute).Unix()}
	assert.True(t, creds.IsTokenExpired())

	// Expiring soon (within 5 min buffer)
	creds = &OAuthCredentials{ExpiresAt: time.Now().Add(2 * time.Minute).Unix()}
	assert.True(t, creds.IsTokenExpired())

	// No expiry set
	creds = &OAuthCredentials{ExpiresAt: 0}
	assert.False(t, creds.IsTokenExpired())
}

// Test cache operations
func TestClearCache(t *testing.T) {
	service := CreateTestService(t, &db.MockQuerier{})

	service.credentialsCache.Store(models.ProviderAnthropic, "key1")
	service.credentialsCache.Store(models.ProviderOpenAI, "key2")

	service.ClearCache()

	_, found1 := service.credentialsCache.Load(models.ProviderAnthropic)
	_, found2 := service.credentialsCache.Load(models.ProviderOpenAI)
	assert.False(t, found1)
	assert.False(t, found2)
}

func TestClearProviderCache(t *testing.T) {
	service := CreateTestService(t, &db.MockQuerier{})

	service.credentialsCache.Store(models.ProviderAnthropic, "key1")
	service.credentialsCache.Store(models.ProviderOpenAI, "key2")

	service.ClearProviderCache(models.ProviderAnthropic)

	_, found1 := service.credentialsCache.Load(models.ProviderAnthropic)
	_, found2 := service.credentialsCache.Load(models.ProviderOpenAI)
	assert.False(t, found1)
	assert.True(t, found2)
}

// Test preloading functions
func TestPreloadCredentials(t *testing.T) {
	mockQueries := &db.MockQuerier{}
	service := CreateTestService(t, mockQueries)

	mockQueries.On("ListAPICredentials", mock.Anything).
		Return([]db.ApiCredential{
			{Provider: "anthropic", ApiKey: sql.NullString{String: "", Valid: true}},
		}, nil)

	service.PreloadCredentials(context.Background())

	mockQueries.AssertExpectations(t)
}

func TestPreloadCredentialsError(t *testing.T) {
	mockQueries := &db.MockQuerier{}
	service := CreateTestService(t, mockQueries)

	mockQueries.On("ListAPICredentials", mock.Anything).
		Return([]db.ApiCredential{}, errors.New("database error"))

	// Should not panic, just log error
	service.PreloadCredentials(context.Background())

	mockQueries.AssertExpectations(t)
}

func TestPreloadOAuthCredentials(t *testing.T) {
	mockQueries := &db.MockQuerier{}
	service := CreateTestService(t, mockQueries)

	mockQueries.On("ListOAuthCredentials", mock.Anything).
		Return([]db.OauthCredential{
			{Provider: "provider1", AccessToken: sql.NullString{String: "", Valid: true}},
		}, nil)

	service.PreloadOAuthCredentials(context.Background())

	mockQueries.AssertExpectations(t)
}

// Test encryption key generation
func TestGenerateEncryptionKey(t *testing.T) {
	key, err := GenerateEncryptionKey()
	require.NoError(t, err)
	assert.Len(t, key, 32) // AES-256 requires 32-byte key

	// Test deterministic
	key2, err := GenerateEncryptionKey()
	require.NoError(t, err)
	assert.Equal(t, key, key2)
}

// Test error handling
func TestEncryptWithNilKey(t *testing.T) {
	service := NewAPICredentialsServiceWithQuerierAndPreload(&db.MockQuerier{}, nil, false)
	_, err := service.encrypt("test")
	require.Error(t, err)
}

func TestDecryptInvalidData(t *testing.T) {
	service := CreateTestService(t, &db.MockQuerier{})

	// Invalid base64
	_, err := service.decrypt("invalid-base64!@#")
	require.Error(t, err)

	// Too short ciphertext
	_, err = service.decrypt("YWJjZA==") // "abcd" in base64
	require.Error(t, err)
}
