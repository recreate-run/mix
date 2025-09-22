# Authentication System Documentation

## Overview

This document describes the complete authentication system implemented for the Mix AI assistant. The system provides secure, database-first credential management with support for multiple authentication methods and AI providers.

## Architecture

### Database-First Approach

- **Primary Storage**: Encrypted SQLite database
- **Encryption**: AES-256-GCM for API key storage
- **User Model**: Single-user system with `default_user` ID
- **No Environment Dependencies**: Pure database approach, no fallback to environment variables

### Supported Providers

The system is limited to 3 carefully selected AI providers:

- **Anthropic**: Claude models with OAuth + API key support
- **OpenAI**: GPT models with API key authentication
- **OpenRouter**: Multi-model gateway with API key authentication

## Authentication Methods

### 1. API Key Authentication

- **Storage**: Encrypted in database using AES-256-GCM encryption
- **Validation**: Format validation per provider
- **Lifecycle**: Full CRUD operations via REST API

### 2. OAuth Authentication (Anthropic Only)

- **PKCE Flow**: Secure OAuth with PKCE challenge/response
- **Token Management**: Automatic refresh and expiration handling
- **State Validation**: CSRF protection with state parameter

## Database Schema

### API Credentials Table

```sql
CREATE TABLE IF NOT EXISTS api_credentials (
    id TEXT PRIMARY KEY DEFAULT 'default_user',  -- Single user system
    provider TEXT NOT NULL,                      -- anthropic, openai, openrouter
    api_key TEXT,                               -- Encrypted API key
    created_at INTEGER NOT NULL,               -- Unix timestamp in milliseconds
    updated_at INTEGER NOT NULL,               -- Unix timestamp in milliseconds
    UNIQUE(id, provider)                       -- One credential per provider per user
);
```

### Key Features

- **Encryption**: All API keys encrypted before storage
- **Uniqueness**: One credential per provider per user
- **Timestamps**: Track creation and update times
- **Indexing**: Optimized for provider lookups

## REST API Endpoints

### Authentication Management

#### Store API Key

```http
POST /api/auth/api-key
Content-Type: application/json

{
  "provider": "openai",
  "api_key": "sk-..."
}
```

#### Get Authentication Status

```http
GET /api/auth/status
```

**Response:**

```json
{
  "data": {
    "providers": {
      "anthropic": {
        "authenticated": false,
        "auth_method": "none",
        "display_name": "Anthropic"
      },
      "openai": {
        "authenticated": true,
        "auth_method": "api_key", 
        "display_name": "OpenAI (GPT) ⭐"
      },
      "openrouter": {
        "authenticated": false,
        "auth_method": "none",
        "display_name": "OpenRouter"
      }
    }
  }
}
```

#### Validate Preferred Provider

```http
GET /api/auth/validate
```

**Response:**

```json
{
  "data": {
    "valid": true,
    "provider": "openai",
    "auth_method": "api_key",
    "message": "Ready to use openai"
  }
}
```

#### Delete Credentials

```http
DELETE /api/auth/{provider}
```

#### Initiate OAuth Flow

```http
POST /api/auth/oauth/anthropic
```

**Response:**

```json
{
  "data": {
    "auth_url": "https://claude.ai/oauth/authorize?client_id=...",
    "state": "...",
    "message": "Open the auth_url in your browser to complete OAuth authentication"
  }
}
```

## User Preferences Integration

### Preferred Provider

- **Selection**: Users can set their preferred AI provider
- **Visual Indicator**: Preferred provider marked with ⭐ star in status
- **Validation**: System validates authentication for preferred provider
- **Database Storage**: Preferences stored in database, not config files

### Agent Configuration

- **Model Selection**: Configurable main and sub-agent models
- **Token Limits**: Per-agent token limit configuration
- **Reasoning Effort**: OpenAI reasoning model parameters
- **Database-First**: All configuration stored in database

## Security Features

### Encryption

- **Algorithm**: AES-256-GCM (Galois/Counter Mode)
- **Key Management**: Dynamic encryption key generation per session
- **Nonce**: Unique nonce per encryption operation
- **Authentication**: Built-in authentication tag verification

### OAuth Security

- **PKCE**: Proof Key for Code Exchange prevents authorization code interception
- **State Parameter**: CSRF protection for OAuth flows
- **Token Refresh**: Automatic handling of expired tokens
- **Secure Storage**: OAuth tokens stored in encrypted credential files

### API Validation

- **Provider Validation**: Rejects unsupported providers
- **Format Validation**: Per-provider API key format checks
- **CORS Protection**: Proper CORS headers for web requests
- **Method Validation**: HTTP method validation on all endpoints

## Implementation Details

### Core Services

#### APICredentialsService

```go
type APICredentialsService struct {
    queries       *db.Queries
    encryptionKey []byte
}
```

**Key Methods:**

- `StoreAPIKey(ctx, provider, apiKey)` - Encrypt and store API key
- `GetAPIKey(ctx, provider)` - Retrieve and decrypt API key  
- `HasAPIKey(ctx, provider)` - Check if provider has stored key
- `DeleteAPIKey(ctx, provider)` - Remove stored credentials
- `ValidateAPIKey(provider, apiKey)` - Validate API key format

#### Provider Integration

The provider system has been updated to use database credentials:

```go
// Database-only approach: only use credentials from database
apiKey := ""
credentialsService := config.GetAPICredentials()
if credentialsService != nil {
    dbKey, err := credentialsService.GetAPIKey(ctx, model.Provider)
    if err == nil && dbKey != "" {
        apiKey = dbKey
        logging.Info("Using database-stored API key", "provider", model.Provider)
    } else {
        // No database key = not authenticated
        logging.Warn("No API key found in database", "provider", model.Provider)
    }
}
```

### Database Migration

- **Version**: `20250911000000_add_api_credentials.sql`
- **Auto-Timestamps**: Trigger for automatic `updated_at` management
- **Indexing**: Optimized indexes for performance
- **Rollback**: Clean rollback support for schema changes

## Error Handling

### API Error Codes

- `INVALID_PROVIDER` - Provider not supported (only anthropic, openai, openrouter allowed)
- `INVALID_API_KEY_FORMAT` - API key doesn't match expected format for provider
- `MISSING_PROVIDER` - Provider parameter required but not provided
- `MISSING_API_KEY` - API key parameter required but not provided
- `STORAGE_ERROR` - Failed to store credentials in database
- `DELETION_ERROR` - Failed to delete credentials from database
- `CREDENTIALS_SERVICE_UNAVAILABLE` - Database service not initialized
- `OAUTH_NOT_SUPPORTED` - OAuth only supported for Anthropic
- `OAUTH_ERROR` - Failed to create OAuth authorization flow

### Validation Rules

#### Provider Validation

- Only `anthropic`, `openai`, `openrouter` accepted
- Case-sensitive string matching
- Clear error messages with supported provider list

#### API Key Format Validation

- **Anthropic**: Must start with `sk-ant-` and be at least 40 characters
- **OpenAI**: Must start with `sk-` and be at least 40 characters  
- **OpenRouter**: Must be at least 40 characters
- **Length**: Minimum length validation prevents obviously invalid keys

## Frontend Integration

### TypeScript SDK

All authentication endpoints are available through the existing TypeScript SDK:

```typescript
// Store API key
await mix.auth.setApiKey({ 
  provider: "openai", 
  api_key: "sk-..." 
});

// Check authentication status
const status = await mix.auth.getStatus();

// Validate preferred provider
const validation = await mix.auth.validate();

// Delete credentials  
await mix.auth.deleteCredentials({ provider: "openai" });
```

### React Integration

- **TanStack Query**: All API calls wrapped in React Query hooks
- **Error Handling**: Structured error responses with user-friendly messages
- **State Management**: Automatic caching and invalidation
- **Real-time Updates**: SSE for live authentication status updates

## Migration from Environment Variables

### Before (Environment Variable Approach)

```bash
export OPENAI_API_KEY="sk-..."
export ANTHROPIC_API_KEY="sk-ant-..."
export OPENROUTER_API_KEY="sk-..."
```

### After (Database Approach)

```bash
# No environment variables needed
# All credentials managed through web UI or API
```

### Migration Benefits

1. **Security**: Encrypted storage vs. plain text environment variables
2. **User Control**: Web UI management vs. terminal environment setup
3. **Portability**: Database travels with application data
4. **Auditability**: Creation and modification timestamps
5. **Validation**: Format validation and provider restrictions
6. **Flexibility**: OAuth + API key hybrid authentication

## Development Workflow

### Adding New Provider Support

1. Add provider to `supportedProviders` map in `rest_auth.go`
2. Add validation rules to `ValidateAPIKey()` method
3. Update provider list in authentication status endpoint
4. Add OAuth support if needed (follow Anthropic pattern)
5. Update documentation

### Testing Authentication

```bash
# Check status (should show all providers unauthenticated)
curl -X GET "http://localhost:8088/api/auth/status"

# Store API key
curl -X POST "http://localhost:8088/api/auth/api-key" \
  -H "Content-Type: application/json" \
  -d '{"provider": "openai", "api_key": "sk-test..."}'

# Validate preferred provider setup
curl -X GET "http://localhost:8088/api/auth/validate"

# Delete credentials
curl -X DELETE "http://localhost:8088/api/auth/openai"
```

## Production Considerations

### Security

- **Encryption Key Management**: Ensure proper key rotation in production
- **Database Security**: SQLite file permissions and backup encryption
- **HTTPS**: All authentication endpoints must use HTTPS in production
- **Rate Limiting**: Consider rate limiting on authentication endpoints

### Monitoring

- **Authentication Events**: Log all authentication successes/failures
- **Credential Usage**: Monitor API key usage patterns
- **Error Tracking**: Alert on repeated authentication failures
- **Performance**: Monitor database query performance for credential operations

### Backup and Recovery

- **Database Backups**: Regular encrypted backups of credential database
- **Key Recovery**: Secure procedure for encryption key recovery
- **Disaster Recovery**: Test restoration procedures regularly

## Changelog

### v1.0.0 - Initial Implementation (2025-09-11)

- ✅ Database-first authentication system
- ✅ AES-256-GCM encrypted credential storage
- ✅ Support for 3 providers: anthropic, openai, openrouter
- ✅ Complete REST API for credential management
- ✅ OAuth integration for Anthropic Claude
- ✅ User preferences integration with preferred provider
- ✅ Provider validation and API key format checking
- ✅ Database migration and schema setup
- ✅ Frontend TypeScript SDK integration
- ✅ Comprehensive error handling and validation
- ✅ Security hardening with no environment variable fallbacks

### Previous Implementation

- Database-first model preferences system
- User preferences service with SQLite storage
- Migration from .mix.json config files to database
- Agent configuration management
- Model and provider preference handling

## Future Enhancements

### Planned Features

- **Multi-User Support**: Extend single-user to multi-user system
- **API Key Rotation**: Automatic API key rotation for supported providers
- **Usage Analytics**: Track API usage per provider and model
- **Backup Integration**: Cloud backup for credential database
- **Advanced OAuth**: Support for more OAuth providers
- **SSO Integration**: Enterprise single sign-on support

### Provider Expansion

- **Azure OpenAI**: Enterprise OpenAI integration
- **Google Vertex AI**: Google Cloud AI platform
- **AWS Bedrock**: Amazon's AI service platform
- **Local Models**: Self-hosted model support

## Support and Troubleshooting

### Common Issues

1. **"Credentials service not available"**: Database service not initialized properly
2. **"Invalid provider"**: Using unsupported provider (only anthropic/openai/openrouter allowed)
3. **"Invalid API key format"**: API key doesn't match expected format for provider
4. **OAuth failures**: Check network connectivity and Anthropic service status

### Debug Mode

Enable debug logging to see credential management operations:

```bash
export _DEV_DEBUG=true
make dev
```

### Database Inspection

```bash
# Check credential database directly
sqlite3 .mix/mix.db "SELECT provider, created_at, updated_at FROM api_credentials;"
```

---

**Note**: This authentication system represents a significant security and usability improvement over environment variable-based credential management. All credentials are encrypted at rest and managed through a user-friendly web interface with comprehensive validation and error handling.
