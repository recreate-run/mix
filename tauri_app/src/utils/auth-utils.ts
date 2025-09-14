import { mix } from '@/lib/mix-sdk';
import { UIMessage } from '@/types/message';

/**
 * Format provider information in a simpler way
 */
function formatProviders(providers: Record<string, any>): {
  authenticatedProviders: string[];
  supportedProviders: string[];
  preferredProvider: string | null;
} {
  const authenticatedProviders: string[] = [];
  const supportedProviders: string[] = [];
  let preferredProvider: string | null = null;
  
  Object.entries(providers).forEach(([id, provider]) => {
    const name = provider.displayName || id;
    const cleanName = name.replace(" ⭐", "");
    
    if (provider.authenticated) {
      authenticatedProviders.push(cleanName);
      
      if (name.includes("⭐")) {
        preferredProvider = cleanName;
      }
    } else {
      supportedProviders.push(cleanName);
    }
  });
  
  return {
    authenticatedProviders,
    supportedProviders,
    preferredProvider
  };
}

/**
 * Handles the status command by using the SDK to check authentication status
 * and returns a message for display
 */
export async function handleStatusCommand(): Promise<UIMessage> {
  try {
    // Get authentication status using the SDK
    const authStatus = await mix.authentication.getAuthStatus();
    
    // Check if we have any authenticated provider
    const anyAuthenticated = Object.values(authStatus.providers || {}).some(
      provider => provider.authenticated
    );
    
    if (anyAuthenticated) {
      // Find the first authenticated provider for a friendly message
      const firstAuthProvider = Object.values(authStatus.providers || {}).find(
        provider => provider.authenticated
      );
      
      const { authenticatedProviders, supportedProviders, preferredProvider } = formatProviders(authStatus.providers || {});
      
      // Get authentication method for the authenticated provider
      let authMethod = "Unknown";
      if (firstAuthProvider?.authMethod === "api_key") {
        authMethod = "API Key";
      } else if (firstAuthProvider?.authMethod === "oauth") {
        authMethod = "OAuth";
      }
      
      // Format the primary provider that's authenticated (likely OpenRouter)
      const primaryProviderName = preferredProvider || authenticatedProviders[0] || "Unknown";
      
      let content = `✅ **Authenticated with ${primaryProviderName}** (via ${authMethod})`;
      
      // List other supported providers if any
      if (supportedProviders.length > 0) {
        content += `\n\nOther supported providers: ${supportedProviders.join(", ")}`;
      }
      
      return {
        content,
        from: "assistant",
        frontend_only: true,
      };
    } else {
      const { supportedProviders } = formatProviders(authStatus.providers || {});
      
      let content = "❌ **Not authenticated with any provider**";
      
      // List supported providers if any
      if (supportedProviders.length > 0) {
        content += `\n\nAvailable providers: ${supportedProviders.join(", ")}`;
      }
      
      content += "\n\nUse `/login` to authenticate.";
      
      return {
        content,
        from: "assistant",
        frontend_only: true,
      };
    }
  } catch (error) {
    return {
      content: `Failed to check authentication status: ${error instanceof Error ? error.message : "Unknown error"}`,
      from: "assistant",
      frontend_only: true,
    };
  }
}