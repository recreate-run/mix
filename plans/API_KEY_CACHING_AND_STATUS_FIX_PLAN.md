# API Key Caching and Authentication Status Fix Plan

## Current Issues Identified

Based on the analysis of the logs and code review, two key issues have been identified:

### 1. Excessive Database Calls for API Keys

The current implementation fetches API keys from the database for every LLM provider initialization:

```
level=INFO msg="Getting API key from database" provider=openrouter
level=INFO msg="API key found in database, attempting to decrypt" provider=openrouter keyLength=136
level=INFO msg="Attempting to decrypt API key" ciphertextLength=136
level=INFO msg="Successfully decrypted API key" plaintextLength=73
level=INFO msg="API key successfully decrypted" provider=openrouter keyLength=73
```

These database calls occur repeatedly during conversations, even for the same provider within seconds of each other:

- In `provider.go`, each call to `NewProvider()` results in a fresh database lookup
- No caching mechanism exists for API keys
- Each message processing creates a new provider instance, causing multiple DB calls

### 2. Authentication Status Discrepancy

There is a discrepancy between the REST API authentication status and the Tauri `/status` command:

- REST API at `/api/auth/status` correctly reports OpenRouter as authenticated
- Tauri app's `/status` command reports "❌ Not authenticated. Use /login to authenticate"

## Root Causes

### Excessive Database Calls

1. **No API Key Caching**: The `credentials/service.go` implementation doesn't maintain any in-memory cache of fetched API keys.

2. **Provider Instance Recreation**: Each message processing creates new provider instances via `NewProvider()`, resulting in fresh database lookups.

3. **Redundant Decryption**: Every API key access requires database retrieval and decryption, which is inefficient.

### Authentication Status Discrepancy

1. **Different Auth Check Mechanisms**: The REST endpoint and built-in commands likely use different methods to check authentication status:
   - REST endpoint uses direct database checks
   - Built-in command may use cached or stale information

2. **Command Implementation**: The built-in command `/status` implementation could be:
   - Only checking for Anthropic authentication specifically
   - Not checking all available providers
   - Using a different code path for authentication verification

## Solution Plan

### 1. Implement API Key Caching

#### A. Create Credentials Cache in Credentials Service

```go
// credentials/service.go

// Add a cache structure to APICredentialsService
type APICredentialsService struct {
    queries       *db.Queries
    encryptionKey []byte
    cache         *credentialsCache
    mu            sync.RWMutex
}

// Create a credentials cache
type credentialsCache struct {
    apiKeys          map[models.ModelProvider]string
    lastRefreshed    map[models.ModelProvider]time.Time
    cacheExpiration  time.Duration
}

// Initialize cache in constructor
func NewAPICredentialsService(database *sql.DB, encryptionKey []byte) *APICredentialsService {
    return &APICredentialsService{
        queries:       db.New(database),
        encryptionKey: encryptionKey,
        cache: &credentialsCache{
            apiKeys:         make(map[models.ModelProvider]string),
            lastRefreshed:   make(map[models.ModelProvider]time.Time),
            cacheExpiration: 10 * time.Minute, // Set reasonable expiration
        },
    }
}
```

#### B. Update GetAPIKey Method to Use Cache

```go
// GetAPIKey retrieves and decrypts an API key for a provider with caching
func (acs *APICredentialsService) GetAPIKey(ctx context.Context, provider models.ModelProvider) (string, error) {
    // First check cache with read lock
    acs.mu.RLock()
    if apiKey, found := acs.cache.apiKeys[provider]; found {
        lastRefresh := acs.cache.lastRefreshed[provider]
        if time.Since(lastRefresh) < acs.cache.cacheExpiration {
            acs.mu.RUnlock()
            return apiKey, nil
        }
    }
    acs.mu.RUnlock()
    
    // Cache miss or expired, acquire write lock and fetch from database
    acs.mu.Lock()
    defer acs.mu.Unlock()
    
    // Double check after acquiring write lock
    if apiKey, found := acs.cache.apiKeys[provider]; found {
        lastRefresh := acs.cache.lastRefreshed[provider]
        if time.Since(lastRefresh) < acs.cache.cacheExpiration {
            return apiKey, nil
        }
    }
    
    // Log at debug level instead of info to reduce noise
    logging.Debug("Cache miss: Getting API key from database", "provider", provider)
    
    // Fetch from database (existing code)
    credential, err := acs.queries.GetAPICredential(ctx, string(provider))
    if err != nil {
        if err == sql.ErrNoRows {
            logging.Debug("No API key found in database", "provider", provider)
            return "", nil
        }
        logging.Error("Failed to get API credential from database", "provider", provider, "error", err)
        return "", fmt.Errorf("failed to get API credential: %w", err)
    }
    
    // Decrypt (existing code)
    decryptedKey, err := acs.decrypt(credential.ApiKey)
    if err != nil {
        logging.Error("Failed to decrypt API key", "provider", provider, "error", err)
        return "", fmt.Errorf("failed to decrypt API key: %w", err)
    }
    
    // Update cache
    acs.cache.apiKeys[provider] = decryptedKey
    acs.cache.lastRefreshed[provider] = time.Now()
    
    logging.Debug("API key fetched and cached", "provider", provider)
    return decryptedKey, nil
}
```

#### C. Invalidate Cache on Updates

```go
// StoreAPIKey with cache invalidation
func (acs *APICredentialsService) StoreAPIKey(ctx context.Context, provider models.ModelProvider, apiKey string) error {
    // Existing implementation
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

    // Invalidate cache
    acs.mu.Lock()
    delete(acs.cache.apiKeys, provider)
    delete(acs.cache.lastRefreshed, provider)
    acs.mu.Unlock()

    logging.Info("API key stored successfully and cache invalidated", "provider", provider)
    return nil
}

// Similar cache invalidation for DeleteAPIKey
func (acs *APICredentialsService) DeleteAPIKey(ctx context.Context, provider models.ModelProvider) error {
    err := acs.queries.DeleteAPICredential(ctx, string(provider))
    if err != nil {
        return fmt.Errorf("failed to delete API credential: %w", err)
    }

    // Invalidate cache
    acs.mu.Lock()
    delete(acs.cache.apiKeys, provider)
    delete(acs.cache.lastRefreshed, provider)
    acs.mu.Unlock()

    logging.Info("API key deleted successfully and cache invalidated", "provider", provider)
    return nil
}
```

#### D. Use Cached Status for HasAPIKey

```go
// HasAPIKey checks if a provider has a stored API key (with cache check)
func (acs *APICredentialsService) HasAPIKey(ctx context.Context, provider models.ModelProvider) (bool, error) {
    // Check cache first
    acs.mu.RLock()
    if _, found := acs.cache.apiKeys[provider]; found {
        lastRefresh := acs.cache.lastRefreshed[provider]
        if time.Since(lastRefresh) < acs.cache.cacheExpiration {
            acs.mu.RUnlock()
            return true, nil
        }
    }
    acs.mu.RUnlock()
    
    // Fall back to database check
    count, err := acs.queries.HasAPICredential(ctx, string(provider))
    if err != nil {
        return false, fmt.Errorf("failed to check API credential: %w", err)
    }
    return count > 0, nil
}
```

### 2. Fix Authentication Status Discrepancy

#### A. Locate and Fix Status Command Implementation

The `/status` command likely exists in a command file that wasn't found in our search. To fix this:

1. **Identify the Status Command**:
   - Search for the implementation of `/status` command within the codebase
   - This could be in a custom commands directory or another location

2. **Update Status Command to Match REST Behavior**:

   ```go
   // status command handler
   func handleStatusCommand(ctx context.Context, args string) (string, error) {
       // Check auth status for all providers
       providers := []models.ModelProvider{
           models.ProviderAnthropic,
           models.ProviderOpenAI,
           models.ProviderOpenRouter,
       }
       
       credentialsService := config.GetAPICredentials()
       if credentialsService == nil {
           return returnError("status", "Credentials service unavailable"), nil
       }
       
       // Check if any provider is authenticated
       for _, provider := range providers {
           authenticated, _ := credentialsService.HasAPIKey(ctx, provider)
           if authenticated {
               return returnMessage("status", fmt.Sprintf("✅ Authenticated with %s", provider)), nil
           }
           
           // Also check OAuth for supported providers
           if provider == models.ProviderAnthropic || provider == models.ProviderOpenAI {
               storage, err := llmprovider.NewCredentialStorage()
               if err == nil {
                   var hasOAuth bool
                   if provider == models.ProviderAnthropic {
                       creds, _ := storage.GetOAuthCredentials("anthropic")
                       hasOAuth = creds != nil && !creds.IsTokenExpired()
                   } else {
                       creds, _ := storage.GetOpenAICredentials("openai")
                       hasOAuth = creds != nil && !creds.IsTokenExpired()
                   }
                   
                   if hasOAuth {
                       return returnMessage("status", fmt.Sprintf("✅ Authenticated with %s via OAuth", provider)), nil
                   }
               }
           }
       }
       
       // No provider authenticated
       return returnMessage("status", "❌ Not authenticated. Use /login to authenticate."), nil
   }
   ```

#### B. Add Additional Debugging to REST Auth Status

```go
// Add debug logging to checkAllAuthenticationStatus
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
        {"anthropic", models.ProviderAnthropic, "Anthropic", true},
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
            logging.Debug("Auth status check - OAuth", "provider", p.name, "hasOAuth", hasOAuth)
        }

        // Check database API key
        hasAPIKey := false
        if credentialsService != nil {
            if hasKey, err := credentialsService.HasAPIKey(ctx, p.provider); err == nil {
                hasAPIKey = hasKey
                logging.Debug("Auth status check - API Key", "provider", p.name, "hasAPIKey", hasAPIKey)
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
        
        // Log complete status
        logging.Info("Provider auth status", 
            "provider", p.name, 
            "authenticated", authenticated, 
            "method", authMethod,
            "isPreferred", p.provider == preferredProvider)
    }

    return status
}
```

## Implementation Strategy

### Phase 1: API Key Caching

1. **Add Cache Structure**:
   - Update `credentials/service.go` to include caching infrastructure
   - Add expiration time configuration
   - Implement thread-safe cache access

2. **Update API Key Methods**:
   - Modify `GetAPIKey` to check cache before database
   - Update `StoreAPIKey` and `DeleteAPIKey` to invalidate cache
   - Add cache awareness to `HasAPIKey`

3. **Testing**:
   - Add unit tests for cache behavior
   - Verify expiration works correctly
   - Test thread safety with concurrent access

### Phase 2: Fix Authentication Status Command

1. **Locate Status Command**:
   - Find the implementation of the `/status` built-in command
   - This is likely in a commands directory or handler

2. **Update Command Logic**:
   - Modify to check all providers, not just one
   - Ensure consistency with REST endpoint behavior
   - Add proper debugging logs

3. **Testing**:
   - Verify REST API and command behavior match
   - Test with different provider authentications

### Phase 3: Performance Monitoring

1. **Add Performance Metrics**:
   - Add hit/miss statistics to the cache
   - Track API key retrieval latency
   - Monitor cache effectiveness

2. **Cache Tuning**:
   - Adjust expiration times based on usage patterns
   - Consider adding TTL (time-to-live) configuration

## Benefits of Implementation

1. **Reduced Database Load**: Minimizing database calls by caching API keys

2. **Improved Performance**: Faster response times for LLM interactions

3. **Consistent User Experience**: Ensuring authentication status is reported consistently across different interfaces

4. **Reduced Log Noise**: Fewer repeated log entries for API key retrieval

## Risks and Mitigations

1. **Cache Staleness**:
   - **Risk**: Cached API keys could become invalid
   - **Mitigation**: Implement reasonable expiration time and cache invalidation

2. **Memory Usage**:
   - **Risk**: Large number of providers could consume memory
   - **Mitigation**: API key data is small; memory impact will be minimal

3. **Thread Safety**:
   - **Risk**: Concurrent access could cause race conditions
   - **Mitigation**: Proper mutex usage with read/write locks

4. **Command Discovery**:
   - **Risk**: Unable to locate status command implementation
   - **Mitigation**: Fallback to update REST endpoint with more complete logging
