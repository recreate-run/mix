# Database-Level Provider Authentication System Migration Plan

## Overview

This plan outlines the migration from the current hardcoded 'default_user' authentication system to a proper database-level provider authentication system with consistent client ID management. This will facilitate enhanced OpenRouter integration with improved analytics tracking while ensuring consistent user identification across different authentication flows.

## Current State Analysis

### Existing Authentication System

The current authentication system uses:

1. **Hardcoded User ID**: All database operations use a hardcoded 'default_user' ID in:
   - `api_credentials` table for API key storage
   - `user_preferences` table for provider preferences

2. **Local Storage Mechanism**: 
   - OAuth credentials stored in `~/.mix/credentials/` directory
   - Machine-specific encrypted storage using `CredentialStorage`

3. **Provider Authentication**:
   - API keys stored in database with 'default_user' ID
   - OAuth tokens stored separately with no explicit user association

4. **Analytics Tracking**:
   - Uses a generic 'anonymous_user' for all analytics events
   - No consistent identification across sessions

### Current OpenRouter Integration

1. **API Usage**: 
   - OpenRouter uses OpenAI-compatible client with custom base URL
   - Authentication managed via API key stored in database

2. **Analytics**: 
   - Enhanced tracking for OpenRouter in `message/tracking_service.go`
   - Special tracking for thinking detection

3. **Client Headers**:
   - Custom headers for OpenRouter API calls (HTTP-Referer, X-Title)
   - No user-specific identifiers

## Client ID Consistency Strategy

### 1. Machine ID Generation

```go
// user/machine_id.go
func GetOrCreateMachineID() (string, error) {
    homeDir, err := os.UserHomeDir()
    if err != nil {
        return "", err
    }
    
    mixDir := filepath.Join(homeDir, ".mix")
    idFile := filepath.Join(mixDir, "machine_id")
    
    // Check if file exists
    if data, err := os.ReadFile(idFile); err == nil && len(data) > 0 {
        return string(data), nil
    }
    
    // Generate new machine ID
    id := make([]byte, 32)
    if _, err := rand.Read(id); err != nil {
        return "", err
    }
    
    machineID := base64.URLEncoding.EncodeToString(id)
    
    // Ensure directory exists
    if err := os.MkdirAll(mixDir, 0700); err != nil {
        return "", err
    }
    
    // Save to file
    if err := os.WriteFile(idFile, []byte(machineID), 0600); err != nil {
        return "", err
    }
    
    return machineID, nil
}
```

### 2. User Identity Layer

```go
// user/service.go
type UserService struct {
    queries *db.Queries
}

func NewUserService(database *sql.DB) *UserService {
    return &UserService{
        queries: db.New(database),
    }
}

// GetOrCreateUser retrieves or creates a user based on machine ID
func (us *UserService) GetOrCreateUser(ctx context.Context) (*db.User, error) {
    // Get machine ID
    machineID, err := GetOrCreateMachineID()
    if err != nil {
        return nil, fmt.Errorf("failed to get machine ID: %w", err)
    }
    
    // Try to get user by machine ID
    user, err := us.queries.GetUserByMachineID(ctx, machineID)
    if err == nil {
        return &user, nil
    }
    
    if err != sql.ErrNoRows {
        return nil, fmt.Errorf("database error: %w", err)
    }
    
    // Create new user if not found
    userID := generateUserID(machineID)
    now := time.Now().Unix() * 1000
    
    newUser, err := us.queries.CreateUser(ctx, db.CreateUserParams{
        ID:        userID,
        MachineID: machineID,
        CreatedAt: now,
        UpdatedAt: now,
    })
    
    if err != nil {
        return nil, fmt.Errorf("failed to create user: %w", err)
    }
    
    return &newUser, nil
}

// generateUserID creates a unique user ID based on machine ID
func generateUserID(machineID string) string {
    h := sha256.New()
    h.Write([]byte(machineID))
    h.Write([]byte(time.Now().String()))
    return fmt.Sprintf("user_%x", h.Sum(nil)[:8])
}
```

### 3. Request Context Enhancement

```go
// http/middleware.go
func WithUserContext(userService *user.UserService) func(http.Handler) http.Handler {
    return func(next http.Handler) http.Handler {
        return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
            // Get or create user
            user, err := userService.GetOrCreateUser(r.Context())
            if err != nil {
                logging.Error("Failed to get or create user", "error", err)
                http.Error(w, "Internal server error", http.StatusInternalServerError)
                return
            }
            
            // Add user to context
            ctx := context.WithValue(r.Context(), userContextKey, user)
            
            // Call the next handler with enhanced context
            next.ServeHTTP(w, r.WithContext(ctx))
        })
    }
}

// GetUserFromContext extracts user from request context
func GetUserFromContext(ctx context.Context) (*db.User, error) {
    user, ok := ctx.Value(userContextKey).(*db.User)
    if !ok || user == nil {
        return nil, errors.New("user not found in context")
    }
    return user, nil
}
```

## Database Schema Changes

### 1. Users Table Schema

```sql
-- Create users table
CREATE TABLE users (
    id TEXT PRIMARY KEY,
    machine_id TEXT NOT NULL UNIQUE,
    name TEXT NULL,
    email TEXT NULL,
    created_at INTEGER NOT NULL,
    updated_at INTEGER NOT NULL
);

-- Create index for machine_id lookups
CREATE UNIQUE INDEX idx_users_machine_id ON users(machine_id);
```

### 2. API Credentials Table Migration

```sql
-- Backup existing credentials
CREATE TABLE api_credentials_backup AS SELECT * FROM api_credentials;

-- Drop existing table
DROP TABLE api_credentials;

-- Create new table with user_id foreign key
CREATE TABLE api_credentials (
    user_id TEXT NOT NULL,
    provider TEXT NOT NULL,
    api_key TEXT NOT NULL,
    created_at INTEGER NOT NULL,
    updated_at INTEGER NOT NULL,
    PRIMARY KEY (user_id, provider),
    FOREIGN KEY (user_id) REFERENCES users(id)
);

-- Create index for provider lookups
CREATE INDEX idx_api_credentials_provider ON api_credentials(provider);
```

### 3. User Preferences Table Migration

```sql
-- Backup existing preferences
CREATE TABLE user_preferences_backup AS SELECT * FROM user_preferences;

-- Drop existing table
DROP TABLE user_preferences;

-- Create new table with user_id foreign key
CREATE TABLE user_preferences (
    user_id TEXT PRIMARY KEY,
    preferred_provider TEXT NULL,
    main_agent_model TEXT NULL,
    main_agent_max_tokens INTEGER NULL,
    main_agent_reasoning_effort TEXT NULL,
    sub_agent_model TEXT NULL,
    sub_agent_max_tokens INTEGER NULL,
    sub_agent_reasoning_effort TEXT NULL,
    created_at INTEGER NOT NULL,
    updated_at INTEGER NOT NULL,
    FOREIGN KEY (user_id) REFERENCES users(id)
);

-- Create index for preferred_provider lookups
CREATE INDEX idx_user_preferences_provider ON user_preferences(preferred_provider);
```

### 4. Data Migration Queries

```sql
-- Create default user to migrate existing data
INSERT INTO users (id, machine_id, created_at, updated_at)
VALUES ('default_user', 'legacy_machine_id', strftime('%s', 'now') * 1000, strftime('%s', 'now') * 1000);

-- Migrate existing api credentials
INSERT INTO api_credentials (user_id, provider, api_key, created_at, updated_at)
SELECT 'default_user', provider, api_key, created_at, updated_at
FROM api_credentials_backup;

-- Migrate existing preferences
INSERT INTO user_preferences 
    (user_id, preferred_provider, main_agent_model, main_agent_max_tokens, 
     main_agent_reasoning_effort, sub_agent_model, sub_agent_max_tokens,
     sub_agent_reasoning_effort, created_at, updated_at)
SELECT 
    'default_user', preferred_provider, main_agent_model, main_agent_max_tokens, 
    main_agent_reasoning_effort, sub_agent_model, sub_agent_max_tokens,
    sub_agent_reasoning_effort, created_at, updated_at
FROM user_preferences_backup;
```

## Code Changes Required

### 1. API Credentials Service Updates

```go
// credentials/service.go

// APICredentialsService handles encrypted API key storage and retrieval
type APICredentialsService struct {
    queries       *db.Queries
    encryptionKey []byte
}

// StoreAPIKey stores an encrypted API key for a provider
func (acs *APICredentialsService) StoreAPIKey(ctx context.Context, provider models.ModelProvider, apiKey string) error {
    // Get user from context
    user, err := http.GetUserFromContext(ctx)
    if err != nil {
        return fmt.Errorf("failed to get user from context: %w", err)
    }

    encryptedKey, err := acs.encrypt(apiKey)
    if err != nil {
        return fmt.Errorf("failed to encrypt API key: %w", err)
    }

    _, err = acs.queries.UpsertAPICredential(ctx, db.UpsertAPICredentialParams{
        UserID:   user.ID,          // Use user ID from context
        Provider: string(provider),
        ApiKey:   encryptedKey,
    })
    if err != nil {
        return fmt.Errorf("failed to store API credential: %w", err)
    }

    logging.Info("API key stored successfully", "provider", provider, "user", user.ID)
    return nil
}

// GetAPIKey retrieves and decrypts an API key for a provider
func (acs *APICredentialsService) GetAPIKey(ctx context.Context, provider models.ModelProvider) (string, error) {
    // Get user from context
    user, err := http.GetUserFromContext(ctx)
    if err != nil {
        return "", fmt.Errorf("failed to get user from context: %w", err)
    }

    logging.Info("Getting API key from database", "provider", provider, "user", user.ID)
    credential, err := acs.queries.GetAPICredential(ctx, db.GetAPICredentialParams{
        UserID:   user.ID,
        Provider: string(provider),
    })
    
    if err != nil {
        if err == sql.ErrNoRows {
            // Try fallback to default_user for backward compatibility
            logging.Info("No API key found for user, trying default_user", "provider", provider)
            credential, err = acs.queries.GetAPICredential(ctx, db.GetAPICredentialParams{
                UserID:   "default_user",
                Provider: string(provider),
            })
            if err != nil {
                if err == sql.ErrNoRows {
                    logging.Info("No API key found in database", "provider", provider)
                    return "", nil // No credential found
                }
                return "", fmt.Errorf("failed to get API credential: %w", err)
            }
        } else {
            return "", fmt.Errorf("failed to get API credential: %w", err)
        }
    }

    decryptedKey, err := acs.decrypt(credential.ApiKey)
    if err != nil {
        return "", fmt.Errorf("failed to decrypt API key: %w", err)
    }

    return decryptedKey, nil
}

// Similar modifications needed for other methods:
// - HasAPIKey
// - DeleteAPIKey
// - ListCredentials
// - DeleteAllCredentials
```

### 2. User Preferences Service Updates

```go
// preferences/service.go

// GetOrCreateUserPreferences gets user preferences from database or creates default ones
func (ups *UserPreferencesService) GetOrCreateUserPreferences(ctx context.Context) (*db.UserPreference, error) {
    // Get user from context
    user, err := http.GetUserFromContext(ctx)
    if err != nil {
        return nil, fmt.Errorf("failed to get user from context: %w", err)
    }
    
    // Try to get existing preferences
    prefs, err := ups.queries.GetUserPreferences(ctx, user.ID)
    if err == nil {
        return &prefs, nil
    }
    
    // If not found, try fallback to default_user for backward compatibility
    if err == sql.ErrNoRows {
        defaultPrefs, defaultErr := ups.queries.GetUserPreferences(ctx, "default_user")
        if defaultErr == nil {
            // Clone default preferences to user-specific preferences
            return ups.createUserPreferencesFromDefault(ctx, user.ID, defaultPrefs)
        }
    }
    
    // If not found or error, create default preferences
    if err == sql.ErrNoRows {
        logging.Info("Creating default user preferences", "user_id", user.ID)
        defaultPrefs := db.CreateUserPreferencesParams{
            UserID:                 user.ID,
            PreferredProvider:      sql.NullString{String: "anthropic", Valid: true},
            MainAgentModel:         sql.NullString{String: "claude-4-sonnet", Valid: true},
            MainAgentMaxTokens:     sql.NullInt64{Int64: 4096, Valid: true},
            MainAgentReasoningEffort: sql.NullString{String: "", Valid: false},
            SubAgentModel:          sql.NullString{String: "claude-4-sonnet", Valid: true},
            SubAgentMaxTokens:      sql.NullInt64{Int64: 2048, Valid: true},
            SubAgentReasoningEffort: sql.NullString{String: "", Valid: false},
        }
        
        createdPrefs, createErr := ups.queries.CreateUserPreferences(ctx, defaultPrefs)
        if createErr != nil {
            return nil, fmt.Errorf("failed to create default user preferences: %w", createErr)
        }
        return &createdPrefs, nil
    }
    
    return nil, fmt.Errorf("failed to get user preferences: %w", err)
}

// Similar modifications needed for other methods:
// - UpdateMainAgentPreferences
// - UpdateSubAgentPreferences
// - UpdatePreferredProvider
// - GetAgentConfig
// - GetPreferredProvider
// - MigrateFromConfig
```

### 3. OAuth Credential Storage Integration

```go
// provider/oauth.go

// Enhanced CredentialStore with user ID
type CredentialStore struct {
    AnthropicCredentials map[string]map[string]OAuthCredentials  `json:"anthropic,omitempty"`
    OpenAICredentials    map[string]map[string]OpenAICredentials `json:"openai,omitempty"`
}

// StoreOAuthCredentials stores OAuth credentials for a user
func (cs *CredentialStorage) StoreOAuthCredentials(ctx context.Context, provider string, accessToken, refreshToken string, expiresAt int64, clientID string) error {
    // Get user from context or fallback to machine ID
    var userID string
    if user, err := http.GetUserFromContext(ctx); err == nil {
        userID = user.ID
    } else {
        // Fallback to machine ID for backward compatibility
        machineID, err := user.GetOrCreateMachineID()
        if err != nil {
            return fmt.Errorf("failed to get machine ID: %w", err)
        }
        userID = machineID
    }
    
    cs.mu.Lock()
    defer cs.mu.Unlock()

    store, err := cs.loadCredentialStore()
    if err != nil {
        return fmt.Errorf("failed to load credential store: %w", err)
    }

    // Ensure user map exists
    if store.AnthropicCredentials[userID] == nil {
        store.AnthropicCredentials[userID] = make(map[string]OAuthCredentials)
    }

    // Add/update credentials for this provider and user
    store.AnthropicCredentials[userID][provider] = OAuthCredentials{
        AccessToken:  accessToken,
        RefreshToken: refreshToken,
        ExpiresAt:    expiresAt,
        ClientID:     clientID,
        Provider:     provider,
    }

    if err := cs.saveCredentialStore(store); err != nil {
        return fmt.Errorf("failed to save credential store: %w", err)
    }

    logging.Info("OAuth credentials stored", "provider", provider, "user", userID)
    return nil
}

// Similar updates needed for:
// - GetOAuthCredentials
// - ClearOAuthCredentials
// - StoreOpenAICredentials 
// - GetOpenAICredentials
```

### 4. Analytics Service Integration

```go
// analytics/analytics.go

// NewAnalyticsService creates a new analytics service with the provided API key
func NewAnalyticsService(apiKey string) Service {
    enabled := apiKey != ""
    var client posthog.Client
    var err error

    if enabled {
        client, err = posthog.NewWithConfig(
            apiKey,
            posthog.Config{
                Endpoint: "https://eu.posthog.com",
            },
        )

        if err != nil {
            logging.Error("Failed to create PostHog client: %v", err)
            enabled = false
        }
    }

    // Create a service with empty distinct ID - will be set per-request
    return &analyticsService{
        client:   client,
        apiKey:   apiKey,
        enabled:  enabled,
        distinct: "",
    }
}

// TrackUserMessage tracks a user message with user context
func (s *analyticsService) TrackUserMessage(ctx context.Context, sessionID, messageID, content string, model string) error {
    if !s.enabled {
        return nil
    }

    // Get user ID from context for analytics
    distinctID := "anonymous_user" // Default fallback
    if user, err := http.GetUserFromContext(ctx); err == nil {
        distinctID = user.ID
    }

    // Track with user-specific ID
    props := posthog.NewProperties().
        Set(PropSessionID, sessionID).
        Set(PropMessageID, messageID).
        Set(PropContent, content).
        Set(PropModel, model)

    err := s.client.Enqueue(posthog.Capture{
        DistinctId: distinctID,
        Event:      EventUserMessage,
        Properties: props,
    })

    if err != nil {
        logging.Error("Failed to track user message: %v", err)
        return fmt.Errorf("failed to track user message: %w", err)
    }

    return nil
}

// Similar updates for all other tracking methods
```

### 5. OpenRouter Integration with Client ID

```go
// provider/provider.go - OpenRouter client initialization

case models.ProviderOpenRouter:
    // Get user ID from context for OpenRouter client headers
    userID := "anonymous"
    if user, err := http.GetUserFromContext(ctx); err == nil {
        userID = user.ID
    }

    clientOptions.openaiOptions = append(clientOptions.openaiOptions,
        WithOpenAIBaseURL("https://openrouter.ai/api/v1"),
        WithOpenAIExtraHeaders(map[string]string{
            "HTTP-Referer": "mix.ai",
            "X-Title":      "Mix",
            "X-Client-ID":  userID, // Add user ID as client ID
        }),
    )
    return &baseProvider[OpenAIClient]{
        options: clientOptions,
        client:  newOpenAIClient(clientOptions),
    }, nil
```

## Implementation Steps

### Phase 1: Database Schema Preparation

1. **Create Database Migration Script**
   - Add users table
   - Update api_credentials table schema
   - Update user_preferences table schema
   - Add necessary indexes

2. **Create Data Migration Logic**
   - Backup existing data
   - Create default user
   - Migrate credentials and preferences to new schema
   - Add data integrity verification steps

3. **Update sqlc Queries**
   - Add new SQL queries to `db/query.sql`
   - Generate new Go code with `sqlc generate`
   - Verify generated code integrity

### Phase 2: Core User Service Implementation

1. **Create Machine ID Service**
   - Implement machine ID generation and storage
   - Add proper error handling and logging
   - Create tests for ID generation

2. **Build User Service**
   - Implement `GetOrCreateUser` functionality
   - Create user context utilities
   - Add unit tests for user management

3. **Add User Context Middleware**
   - Implement HTTP middleware to inject user context
   - Add context extraction utilities
   - Update router configuration to use middleware

### Phase 3: Service Layer Updates

1. **Update API Credentials Service**
   - Modify all methods to use user ID from context
   - Add backward compatibility for default_user
   - Add unit tests for user-specific credentials

2. **Update User Preferences Service**
   - Modify all methods to use user ID from context
   - Add backward compatibility for default_user
   - Add unit tests for user-specific preferences

3. **Update OAuth Credential Storage**
   - Modify credential store structure to support user-specific credentials
   - Update storage and retrieval methods
   - Add migration logic for existing OAuth credentials

### Phase 4: Integration Updates

1. **Update OpenRouter Integration**
   - Add client ID to request headers
   - Ensure consistent user identification
   - Update tracking functionality

2. **Update Analytics Service**
   - Add user ID as distinct identifier
   - Ensure consistent tracking across providers
   - Update tracking methods for user context

3. **Update HTTP Handlers**
   - Update REST handlers to extract user context
   - Ensure proper error handling for missing context
   - Test authentication flows with new user context

### Phase 5: Testing and Validation

1. **Unit Tests**
   - Test machine ID generation and storage
   - Test user creation and retrieval
   - Test credential operations with user context
   - Test preference operations with user context

2. **Integration Tests**
   - Test authentication flows with multiple users
   - Test credential persistence across sessions
   - Test OpenRouter integration with user tracking
   - Test analytics with consistent user identification

3. **Migration Validation**
   - Test data migration script
   - Verify data integrity before and after migration
   - Test backward compatibility with existing data

## Testing Strategy

### Unit Tests

1. **Machine ID Generation**
   - Test stable ID generation
   - Test ID persistence
   - Test error handling

2. **User Service**
   - Test user creation
   - Test user retrieval by machine ID
   - Test context utilities

3. **Credentials Service**
   - Test API key storage and retrieval with user context
   - Test OAuth credential storage and retrieval
   - Test backward compatibility

4. **Preferences Service**
   - Test preference storage and retrieval with user context
   - Test default preference creation
   - Test backward compatibility

### Integration Tests

1. **Authentication Flow**
   - Test API key authentication with user context
   - Test OAuth authentication with user context
   - Test credential persistence across sessions

2. **OpenRouter Integration**
   - Test API calls with client ID
   - Test analytics tracking with consistent user ID
   - Test thinking detection with user context

3. **Multi-User Scenarios**
   - Test different credentials for different users
   - Test different preferences for different users
   - Test isolation between user data

### Migration Testing

1. **Database Migration**
   - Test schema upgrade scripts
   - Test data migration integrity
   - Test rollback procedures

2. **Backward Compatibility**
   - Test existing credentials still work
   - Test existing preferences still apply
   - Test fallback to default_user when needed

## Benefits of Implementation

1. **Enhanced User Tracking**: Consistent user identification across all authentication flows

2. **Improved Analytics**: Better analytics data with proper user segmentation

3. **OpenRouter Integration**: Enhanced tracking for OpenRouter with user-specific client ID

4. **Future Multi-User Support**: Foundation for proper multi-user support

5. **Data Integrity**: Better database schema with foreign key constraints

6. **Security**: User-specific encrypted credentials with proper isolation

## Migration Risks and Mitigations

### Risks

1. **Data Loss**: Existing credentials or preferences could be lost during migration
   - **Mitigation**: Create backup tables before migration and verify data integrity

2. **User Experience Disruption**: Authentication may fail during migration
   - **Mitigation**: Add fallback to default_user for backward compatibility

3. **Performance Impact**: Additional context lookups could impact performance
   - **Mitigation**: Add proper caching for frequent user context retrievals

4. **Compatibility Issues**: OAuth tokens may not work with new user context
   - **Mitigation**: Test all authentication flows thoroughly before deployment

### Fallback Plan

1. **Schema Rollback**: Revert to original schema if issues are detected
2. **Dual-Mode Operation**: Support both user-specific and default_user modes during transition
3. **Incremental Migration**: Migrate one service at a time to isolate issues

## Conclusion

This migration plan provides a comprehensive approach to implementing a database-level provider authentication system with consistent client ID management. By following these steps, the application will gain enhanced user tracking, improved analytics, and better integration with OpenRouter while maintaining backward compatibility with existing data.

The plan carefully considers the current architecture and provides detailed changes required for each component, along with a phased implementation approach and thorough testing strategy to ensure a smooth transition.