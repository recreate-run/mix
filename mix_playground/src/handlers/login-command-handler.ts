import { mix } from "@/lib/mix-sdk";
import { UIMessage, LoginProviderInfo } from "@/types/message";
import { toast } from "sonner"; // Import directly from package

// Note: LoginProviderInfo is now imported from @/types/message

// Map of API key formats for different providers
const API_KEY_FORMATS: Record<string, string> = {
  "anthropic": "sk-ant-...",
  "openai": "sk-...",
  "openrouter": "sk-...",
  // Add more as needed
};

// Map of supported auth methods for different providers
const AUTH_METHODS: Record<string, ("api_key" | "oauth")[]> = {
  "anthropic": ["api_key", "oauth"],
  "openai": ["api_key"],
  "openrouter": ["api_key"],
  // Add more as needed
};

/**
 * Handles the login command, returns a message with login UI or success/error message
 */
export async function handleLoginCommand(): Promise<UIMessage> {
  try {
    // Get current authentication status
    const status = await mix.authentication.getAuthStatus();
    
    // Fetch available providers from API
    const response = await mix.preferences.get();
    
    // Check if we have preferences already set
    const hasExistingPreferences = !!response.preferences;
    
    if (!response.availableProviders) {
      throw new Error("Failed to fetch available providers");
    }
    
    // Determine the preferred provider from preferences if it exists
    const preferredProvider = response.preferences?.preferredProvider;
    
    // Map available providers from API response for login
    const providers: LoginProviderInfo[] = Object.entries(response.availableProviders).map(([providerId, data]: [string, any]) => {
      return {
        id: providerId,
        displayName: data.displayName || providerId,
        authMethods: AUTH_METHODS[providerId] || ["api_key"], // Required for login
        authenticated: status.providers?.[providerId]?.authenticated || false,
        apiKeyFormat: API_KEY_FORMATS[providerId] || "API key",
        isPreferred: providerId === preferredProvider
      };
    });
    // Return message with login data for hooks
    return {
      content: hasExistingPreferences ?
        `Using preferences with preferred provider: ${preferredProvider}` :
        "No existing preferences found. Please select a provider to authenticate with.",
      from: "assistant",
      frontend_only: true,
      loginData: {
        providers,
        hasExistingPreferences: hasExistingPreferences
      }
    };
  } catch (error) {
    return {
      content: `Failed to initialize login: ${error instanceof Error ? error.message : "Unknown error"}`,
      from: "assistant",
      frontend_only: true,
      suppressChatMessage: true
    };
  }
}

/**
 * Authenticates with a provider using an API key
 */
export async function authenticateWithApiKey(
  provider: string,
  apiKey: string
): Promise<UIMessage> {
  try {
    // Store API key using the SDK
    await mix.authentication.storeApiKey({
      provider: provider as any,
      apiKey
    });
    
    // Validate the saved API key by checking auth status
    const authStatus = await mix.authentication.getAuthStatus();
    if (!authStatus.providers?.[provider]?.authenticated) {
      throw new Error(`API key was stored but provider ${provider} is not showing as authenticated. Please try again.`);
    }
    
    // Update preferences to use this provider as the preferred one
    try {
      await updateUserPreferences({ preferredProvider: provider });
    } catch (prefError) {
      console.error('Failed to update preferences:', prefError);
      // Continue even if preference update fails - the key is stored
    }
    
    // Show success toast notification
    try {
      toast.success("Authentication successful", {
        description: `Successfully authenticated with ${provider}`,
        duration: 3000
      });
      console.log("Toast notification triggered for API key login");
    } catch (toastError) {
      console.error("Failed to show toast:", toastError);
    }
    
    return {
      content: `✅ Successfully authenticated with ${provider} using API key`,
      from: "assistant",
      frontend_only: true,
      suppressChatMessage: true  // Hide this success message from the chat UI
    };
  } catch (error) {
    // Try to delete the API key if authentication failed
    try {
      await mix.authentication.deleteCredentials({ provider });
    } catch (deleteError) {
      console.error('Failed to clean up API key after authentication failure:', deleteError);
    }
    
    return {
      content: `❌ Failed to authenticate: ${error instanceof Error ? error.message : "Unknown error"}`,
      from: "assistant",
      frontend_only: true,
      suppressChatMessage: true
    };
  }
}

/**
 * Starts OAuth flow for a provider (currently only Anthropic)
 */
export async function startOAuthFlow(provider: string): Promise<UIMessage> {
  try {    
    // Start OAuth flow using the SDK
    const result = await mix.authentication.startOAuthFlow({
      provider
    });
    
    if (!result.authUrl) {
      return {
        content: "❌ Failed to start OAuth flow: No authorization URL returned",
        from: "assistant",
        frontend_only: true
      };
    }
    
    // Return success with auth URL - browser will be opened by the handleStartOAuth function in login-ui.tsx
    // Get available providers to include in response
    const preferencesResponse = await mix.preferences.get();
    if (!preferencesResponse.availableProviders) {
      throw new Error("Failed to fetch available providers for OAuth flow");
    }

    const providers: LoginProviderInfo[] = Object.entries(preferencesResponse.availableProviders).map(([providerId, data]: [string, any]) => {
      return {
        id: providerId,
        displayName: data.displayName || providerId,
        authMethods: AUTH_METHODS[providerId] || ["api_key"], // Required for login
        authenticated: false, // During OAuth flow, not yet authenticated
        apiKeyFormat: API_KEY_FORMATS[providerId] || "API key"
      };
    });

    
    return {
      content: `If the browser doesn't open automatically, you can click or copy this URL: ${result.authUrl}`,
      from: "assistant",
      frontend_only: true,
      loginData: {
        providers,
        hasExistingPreferences: false,
        oauthState: result.state
      }
    };
  } catch (error) {
    let errorMessage = error instanceof Error ? error.message : "Unknown error";
    
    // Check for structured error response
    if (typeof error === 'object' && error !== null && 'type' in error) {
      const errorObj = error as {type?: string, message?: string, code?: number};
      
      if (errorObj.type === 'OAUTH_NOT_SUPPORTED') {
        errorMessage = `OAuth is not supported for ${provider}. Please use API key authentication instead.`;
      } else if (errorObj.message) {
        errorMessage = errorObj.message;
      }
    }

    return {
      content: `❌ ${errorMessage}`,
      from: "assistant",
      frontend_only: true,
      suppressChatMessage: true
    };
  }
}

/**
 * Handles OAuth callback with authorization code
 */
export async function handleOAuthCallback(
  provider: string,
  code: string,
  state: string
): Promise<UIMessage> {
  try {
    // Handle OAuth callback using the SDK
    await mix.authentication.handleOAuthCallback({
      provider,
      code,
      state  // Pass the state parameter received from the initial OAuth response
    });
    
    // Update preferences to use this provider as the preferred one
    await updateUserPreferences({ preferredProvider: provider });
    
    // Show success toast notification
    toast.success("OAuth authentication successful", {
      description: `Successfully authenticated with ${provider}`,
      duration: 3000
    });
    
    return {
      content: `✅ Successfully authenticated with ${provider} using OAuth`,
      from: "assistant",
      frontend_only: true,
      suppressChatMessage: true  // Hide this success message from the chat UI
    };
  } catch (error) {
    return {
      content: `❌ Failed to complete OAuth: ${error instanceof Error ? error.message : "Unknown error"}`,
      from: "assistant",
      frontend_only: true,
      suppressChatMessage: true
    };
  }
}

/**
 * Updates user preferences with provided values, initializing defaults if needed
 */
export async function updateUserPreferences(
  preferences: {
    preferredProvider?: string;
    mainAgentModel?: string;
    mainAgentMaxTokens?: number;
    subAgentModel?: string;
    subAgentMaxTokens?: number;
  },
  retryCount = 3
): Promise<UIMessage> {
  try {
    // Get current preferences
    const currentPrefs = await mix.preferences.get();
    
    // Prepare update with reasonable defaults if no preferences exist
    const update = {
      preferredProvider: preferences.preferredProvider || currentPrefs.preferences?.preferredProvider || 'anthropic',
      mainAgentModel: preferences.mainAgentModel || currentPrefs.preferences?.mainAgentModel || '',
      mainAgentMaxTokens: preferences.mainAgentMaxTokens || currentPrefs.preferences?.mainAgentMaxTokens || 2000,
      subAgentModel: preferences.subAgentModel || currentPrefs.preferences?.subAgentModel || '',
      subAgentMaxTokens: preferences.subAgentMaxTokens || currentPrefs.preferences?.subAgentMaxTokens || 1000
    };
    
    // If a provider is specified but no models, try to set default models
    if (preferences.preferredProvider && !preferences.mainAgentModel && currentPrefs.availableProviders) {
      const providerData = currentPrefs.availableProviders[preferences.preferredProvider];
      if (providerData && providerData.models && providerData.models.length > 0) {
        // Use the first model as default
        update.mainAgentModel = providerData.models[0];
        update.subAgentModel = providerData.models[0];
      }
    }
    
    // Update preferences
    await mix.preferences.update(update);
    
    // Verify the update was successful by reading back preferences
    const verifyPrefs = await mix.preferences.get();
    const savedProvider = verifyPrefs.preferences?.preferredProvider;
    
    // If the preferred provider wasn't saved correctly and we have retries left, try again
    if (preferences.preferredProvider && 
        savedProvider !== preferences.preferredProvider && 
        retryCount > 0) {
      console.warn(`Preference update did not save correctly. Retrying... (${retryCount} attempts left)`);
      // Wait a bit before retrying to allow server-side operations to complete
      await new Promise(resolve => setTimeout(resolve, 500));
      return updateUserPreferences(preferences, retryCount - 1);
    }
    
    return {
      content: "✅ Preferences updated successfully",
      from: "assistant",
      frontend_only: true,
      suppressChatMessage: true  // Hide this success message from the chat UI
    };
  } catch (error) {
    if (retryCount > 0) {
      console.warn(`Error updating preferences: ${error}. Retrying... (${retryCount} attempts left)`);
      // Wait a bit before retrying
      await new Promise(resolve => setTimeout(resolve, 500));
      return updateUserPreferences(preferences, retryCount - 1);
    }
    
    return {
      content: `❌ Failed to update preferences after multiple attempts: ${error instanceof Error ? error.message : "Unknown error"}`,
      from: "assistant",
      frontend_only: true,
      suppressChatMessage: true
    };
  }
}