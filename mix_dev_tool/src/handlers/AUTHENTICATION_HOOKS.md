# Authentication Hooks

This document outlines the available authentication business logic hooks. These hooks provide a clean foundation for implementing authentication interfaces.

## Core Authentication Functions

### API Key Authentication
```typescript
import { authenticateWithApiKey } from '@/handlers/login-command-handler';

// Authenticate using an API key
const result = await authenticateWithApiKey(provider: string, apiKey: string);
```
- Stores API key using Mix SDK
- Validates authentication by checking status
- Updates user preferences to set as preferred provider
- Shows success toast notification
- Handles cleanup on authentication failure

### OAuth Flow Initiation
```typescript
import { startOAuthFlow } from '@/handlers/login-command-handler';

// Start OAuth authentication flow
const result = await startOAuthFlow(provider: string);
```
- Initiates OAuth with `mix.authentication.startOAuthFlow()`
- Returns authorization URL for browser redirect
- Includes OAuth state parameter for security
- Returns `loginData` with provider information and OAuth state

### OAuth Flow Completion
```typescript
import { handleOAuthCallback } from '@/handlers/login-command-handler';

// Complete OAuth authentication with authorization code
const result = await handleOAuthCallback(provider: string, code: string, state: string);
```
- Handles OAuth completion with authorization code
- Calls `mix.authentication.handleOAuthCallback()`
- Updates user preferences with authenticated provider
- Shows success toast notification

## State Management Hooks

### Authentication Flow Hook
```typescript
import { useAuthFlow } from '@/hooks/useAuthFlow';

const {
  authCode,
  setAuthCode,
  apiKey,
  setApiKey,
  authMode,
  setAuthMode,
  oauthState,
  setOauthState,
  isLoading,
  showSuccess,
  handleSubmit,
} = useAuthFlow();
```
- Manages authentication state (API key vs OAuth code)
- Handles form submission logic
- Provides loading and success states
- Integrates with TanStack Query for data management

### Command Handlers (Partial)
```typescript
import {
  handleApiKeySubmitSpecial,
  handleOAuthCodeSubmitSpecial,
  handleLoginProviderSelectionSpecial
} from '@/hooks/command-slash/useCommandHandlers';
```
- `handleApiKeySubmitSpecial` - Submits API key for authentication
- `handleOAuthCodeSubmitSpecial` - Submits OAuth code with state parameter
- `handleLoginProviderSelectionSpecial` - Initiates OAuth flow for provider

## Type Definitions

### Core Types (Preserved)
```typescript
// Provider information for login flows
interface LoginProviderInfo {
  id: string;
  displayName: string;
  authMethods: ("api_key" | "oauth")[];
  authenticated: boolean;
  apiKeyFormat?: string;
  isPreferred?: boolean;
}

// Login data structure for hooks
interface LoginData {
  providers: LoginProviderInfo[];
  hasExistingPreferences?: boolean;
  oauthState?: string;
}

// Authentication methods
type AuthMethod = 'api_key' | 'oauth' | 'oauth_code';
```

### Message Interface (Preserved)
```typescript
interface UIMessage {
  // ... other fields
  loginData?: {
    providers: LoginProviderInfo[];
    hasExistingPreferences?: boolean;
    oauthState?: string;
  };
}
```

## Backend Integration

All authentication functions integrate with the Mix SDK:

- **API Key Storage**: `mix.authentication.storeApiKey()`
- **OAuth Initiation**: `mix.authentication.startOAuthFlow()`
- **OAuth Completion**: `mix.authentication.handleOAuthCallback()`
- **Status Checking**: `mix.authentication.getAuthStatus()`
- **Preference Management**: `mix.preferences.get()` and `mix.preferences.update()`

## Usage Examples

### Direct API Key Authentication
```typescript
try {
  const result = await authenticateWithApiKey('anthropic', 'sk-ant-...');
  if (result.content.includes('✅')) {
    // Authentication successful
    console.log('Authenticated successfully');
  }
} catch (error) {
  console.error('Authentication failed:', error);
}
```

### OAuth Flow Implementation
```typescript
// Step 1: Start OAuth flow
const oauthResult = await startOAuthFlow('anthropic');
const authUrl = oauthResult.loginData?.oauthState; // Get auth URL from response
const oauthState = oauthResult.loginData?.oauthState;

// Step 2: User authorizes in browser and gets code
// Step 3: Complete OAuth with code
const result = await handleOAuthCallback('anthropic', authorizationCode, oauthState);
```

### Using the Auth Flow Hook
```typescript
function CustomLoginForm() {
  const {
    apiKey,
    setApiKey,
    authMode,
    setAuthMode,
    isLoading,
    handleSubmit
  } = useAuthFlow();

  return (
    <form onSubmit={handleSubmit}>
      <input
        type="password"
        value={apiKey}
        onChange={(e) => setApiKey(e.target.value)}
        disabled={isLoading}
      />
      <button type="submit" disabled={isLoading}>
        {isLoading ? 'Authenticating...' : 'Login'}
      </button>
    </form>
  );
}
```

## Notes for Future Implementations

1. **Error Handling**: All functions include comprehensive error handling with cleanup
2. **Toast Notifications**: Success/error notifications are automatically shown
3. **Preference Management**: Authenticated providers are automatically set as preferred
4. **State Security**: OAuth flows include proper state parameter handling
5. **Clean Separation**: Business logic is completely separated from UI concerns

## Architecture

The authentication system is built with clean separation between business logic and UI concerns. All authentication functionality is available through the documented hooks and handlers, allowing for flexible UI implementations while maintaining consistent backend integration.