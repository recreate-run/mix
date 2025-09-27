# Provider Authentication Improvement Plan

## Problem Statement

Currently, Mix defaults to using Anthropic/Claude as the preferred provider. If a user adds an OpenRouter API key but doesn't manually change their preferred provider from Anthropic, they still receive an authentication error message prompting them to authenticate with Anthropic or change their preference.

This creates a confusing user experience, since the user has already provided valid authentication for a supported provider (OpenRouter) but is unable to send messages without taking additional steps.

## Current Implementation Analysis

The issue stems from the following architectural decisions in the codebase:

1. **Default Provider Setting**: Anthropic/Claude is set as the default provider in multiple places.

2. **Authentication Check Logic**: In `oauth.go`, the `IsAuthenticated` function checks only the preferred provider's authentication status (or defaults to Anthropic if no preference is set).

3. **Message Handling Flow**: In `rest_messages.go`, the `HandleSendMessage` function blocks any message sending if `IsAuthenticated` returns false, showing a provider-specific error message.

4. **Error Message Generation**: The `getAuthenticationErrorMessage` function in `auth_utils.go` creates error messages specifically mentioning the preferred provider, not acknowledging other authenticated providers.

## Proposed Solutions

We've evaluated multiple approaches to address this issue:

### Approach 1: Message Fallback - Try All Authenticated Providers

Modify the message processing flow to automatically use any available authenticated provider if the preferred one isn't authenticated.

### Approach 2: Auto-Update Preferred Provider on Authentication

Automatically update the user's preferred provider setting when they add a new API key.

### Approach 3: Check All Authenticated Providers Before Showing Errors

Update the authentication check to verify if any provider is authenticated before blocking message sending.

### Approach 4: Smart Provider Selection with Prioritization

Implement a sophisticated provider selection system with fallback logic and provider prioritization.

### Approach 5: Improved User Interface for Provider Status

Enhance the UI to provide clearer information about authentication status and provider selection.

## Recommended Solution: Check All Authenticated Providers

After evaluating all approaches, **Approach 3** (Check All Authenticated Providers Before Showing Errors) offers the best balance between user experience, simplicity, and maintaining user control.

## Implementation Details

### 1. Modify IsAuthenticated Function in oauth.go

Update the function to support checking for any authenticated provider, not just the preferred one:

```go
// IsAuthenticated checks if there are valid authentication credentials available.
// If checkAnyProvider is true, it will return true if ANY provider is authenticated.
// Otherwise, it will only check the specified provider (or preferred provider if empty).
func IsAuthenticated(ctx context.Context, provider models.ModelProvider, checkAnyProvider bool) (bool, string, error) {
    // Get API credentials service from config
    credentialsService := config.GetAPICredentials()
    if credentialsService == nil {
        return false, "none", fmt.Errorf("credentials service not available")
    }

    // If provider is empty, try to get the user's preferred provider
    if provider == "" {
        // Try to get user preferences service
        userPrefs := config.GetUserPreferences()
        if userPrefs != nil {
            // Get preferred provider
            if pref, err := userPrefs.GetPreferredProvider(ctx); err == nil && pref != "" {
                provider = pref
            }
        }

        // If still empty, check if there are any API keys available
        if provider == "" {
            providers, err := credentialsService.ListCredentials(ctx)
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
    }

    // First check the specified/preferred provider
    hasAPIKey, err := credentialsService.HasAPIKey(ctx, provider)
    if err != nil {
        logging.Warn("Failed to check API credential", "error", err)
    } else if hasAPIKey {
        return true, "api_key", nil
    }

    // Check OAuth for the specified/preferred provider
    if provider == models.ProviderAnthropic || provider == models.ProviderOpenAI {
        switch provider {
        case models.ProviderAnthropic:
            creds, err := credentialsService.GetOAuthCredentials(ctx, "anthropic")
            if err == nil && creds != nil && !creds.IsTokenExpired() {
                return true, "oauth", nil
            }
        case models.ProviderOpenAI:
            creds, err := credentialsService.GetOAuthCredentials(ctx, "openai")
            if err == nil && creds != nil && !creds.IsTokenExpired() {
                return true, "oauth", nil
            }
        }
    }

    // If checkAnyProvider flag is set and preferred provider isn't authenticated,
    // check if ANY provider is authenticated
    if checkAnyProvider {
        providers, err := credentialsService.ListCredentials(ctx)
        if err == nil {
            // Check each provider for valid API key
            for _, p := range providers {
                if p == provider {
                    continue // Skip the preferred provider we already checked
                }
                
                hasAPIKey, err := credentialsService.HasAPIKey(ctx, p)
                if err == nil && hasAPIKey {
                    return true, "api_key", nil
                }
                
                // Check OAuth for supported providers
                if p == models.ProviderAnthropic || p == models.ProviderOpenAI {
                    var creds *OAuthCredentials
                    var err error
                    
                    switch p {
                    case models.ProviderAnthropic:
                        creds, err = credentialsService.GetOAuthCredentials(ctx, "anthropic")
                    case models.ProviderOpenAI:
                        creds, err = credentialsService.GetOAuthCredentials(ctx, "openai")
                    }
                    
                    if err == nil && creds != nil && !creds.IsTokenExpired() {
                        return true, "oauth", nil
                    }
                }
            }
        }
    }

    // No valid credentials found
    return false, "none", nil
}
```

### 2. Update HandleSendMessage in rest_messages.go

Modify the message handling to check for any authenticated provider:

```go
// In HandleSendMessage function
// Check authentication status before processing the message
authenticated, _, authErr := provider.IsAuthenticated(ctx, "", true) // true = check any provider
if authErr != nil {
    sendInternalError(w, "checking authentication", authErr)
    return
}

// If not authenticated, show a provider-specific error message
if !authenticated {
    helpfulMsg := getAuthenticationErrorMessage(ctx)

    result := MessageData{
        ID:                "system-auth-prompt",
        Role:              "assistant",
        UserInput:         req.Text,
        AssistantResponse: helpfulMsg,
    }

    sendJSONResponse(w, http.StatusOK, result)
    return
}
```

### 3. Update getAuthenticationErrorMessage in auth_utils.go

Make the error message more helpful by suggesting checking for any available authenticated providers:

```go
// getAuthenticationErrorMessage returns a provider-specific authentication error message
func getAuthenticationErrorMessage(ctx context.Context) string {
    // Get user preferences to determine their preferred provider
    userPrefs := config.GetUserPreferences()
    if userPrefs == nil {
        return "⚠️ Authentication required. Please go to settings and authenticate"
    }

    preferredProvider, err := userPrefs.GetPreferredProvider(ctx)
    if err != nil || preferredProvider == "" {
        return "⚠️ Authentication required. Please go to settings and authenticate"
    }

    // Check if user has any authenticated providers
    credentialsService := config.GetAPICredentials()
    if credentialsService != nil {
        providers, err := credentialsService.ListCredentials(ctx)
        if err == nil && len(providers) > 0 {
            // User has other authenticated providers but not their preferred one
            providerName := getProviderDisplayName(string(preferredProvider))
            authenticatedProviders := make([]string, 0, len(providers))
            
            for _, p := range providers {
                authenticatedProviders = append(authenticatedProviders, getProviderDisplayName(string(p)))
            }
            
            return fmt.Sprintf("⚠️ Not authenticated with %s (your preferred provider)\n\n"+
                "You have authenticated providers: %s\n\n"+
                "Choose one option:\n"+
                "•Authentication to connect your %s account\n"+
                "•Change your preferred provider to one of your authenticated providers",
                providerName, strings.Join(authenticatedProviders, ", "), providerName)
        }
    }

    // Get a user-friendly name for the provider
    providerName := getProviderDisplayName(string(preferredProvider))

    // Create provider-specific message with helpful instructions
    return fmt.Sprintf("⚠️ Not authenticated with %s (your preferred provider)\n\n"+
        "Choose one option:\n"+
        "•Authentication to connect your %s account\n"+
        "•change your preferred provider",
        providerName, providerName)
}
```

### 4. Update Agent Provider Creation Logic

Ensure that the agent creation logic in `agent.go` also attempts to use any authenticated provider if the preferred one isn't authenticated:

```go
// In createAgentProvider or related function
// Check if preferred provider is authenticated
authenticated, _, _ := provider.IsAuthenticated(ctx, preferredProvider, false)
if !authenticated {
    // Try to find any authenticated provider
    anyAuthenticated, _, _ := provider.IsAuthenticated(ctx, "", true)
    if anyAuthenticated {
        // Get the first authenticated provider
        providers, _ := credentialsService.ListCredentials(ctx)
        if len(providers) > 0 {
            // Use the first available provider instead
            logging.Warn("Preferred provider not authenticated, using alternative provider", 
                "preferred", preferredProvider, "using", providers[0])
            preferredProvider = providers[0]
        }
    }
}
```

## Benefits of This Approach

1. **Better User Experience**: Users can send messages as long as any provider is authenticated.

2. **Maintains User Control**: User preferences are preserved while ensuring functionality.

3. **Clear Messaging**: Updated error messages inform users about their authenticated providers.

4. **Minimal Changes**: Implementation requires focused changes to specific functions without major architectural modifications.

5. **Forward Compatibility**: This approach works well if more providers are added in the future.

## Future Enhancements

After implementing this initial solution, consider these potential enhancements:

1. **Provider Preference UI**: Make it clearer which providers are authenticated in the UI.

2. **Per-Session Provider Selection**: Allow users to select different providers for different sessions.

3. **Smart Provider Fallback**: Add more sophisticated provider selection logic based on model capabilities.

4. **Authentication Status Indicators**: Show clear visual indicators of which providers are authenticated.

## Implementation Timeline

1. **Day 1**: Modify `IsAuthenticated` function and update `HandleSendMessage`.
2. **Day 2**: Update error message generation and agent provider creation logic.
3. **Day 3**: Testing and fixes.
4. **Day 4**: UI enhancements for provider selection (optional).

## Conclusion

By implementing the "Check All Authenticated Providers" approach, we'll ensure that users with any valid provider authentication can use the system without manual reconfiguration, while still maintaining their preferences for when their preferred provider becomes available.