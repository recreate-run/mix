import { mix } from "@/lib/mix-sdk";
import { UIMessage } from "@/types/message";
import { toast } from "sonner";

// Provider info structure for logout UI
export interface LogoutProviderInfo {
  id: string;
  displayName: string;
  authenticated: boolean;
  authMethod?: "api_key" | "oauth";
  isPreferred?: boolean;
}

/**
 * Format provider information for the logout UI
 */
function formatLogoutProviders(providers: Record<string, any>): {
  formattedProviders: LogoutProviderInfo[];
  hasAuthenticatedProvider: boolean;
} {
  const formattedProviders: LogoutProviderInfo[] = [];
  
  let hasAuthenticatedProvider = false;
  
  Object.entries(providers).forEach(([id, provider]) => {
    // Extract the base name without the star symbol
    const name = provider.displayName || id;
    const cleanName = name.replace(" ⭐", "");
    const isPreferred = name.includes("⭐");
    
    // Only include authenticated providers for logout
    if (provider.authenticated) {
      formattedProviders.push({
        id,
        displayName: cleanName,
        authenticated: true,
        authMethod: provider.auth_method,
        isPreferred
      });
      
      hasAuthenticatedProvider = true;
    }
  });
  
  // Sort providers - preferred first, then alphabetically
  formattedProviders.sort((a, b) => {
    // Preferred provider first
    if (a.isPreferred !== b.isPreferred) {
      return a.isPreferred ? -1 : 1;
    }
    
    // Then alphabetically
    return a.displayName.localeCompare(b.displayName);
  });
  
  return {
    formattedProviders,
    hasAuthenticatedProvider
  };
}

/**
 * Handles the logout command by fetching authenticated providers
 * and returns a message with logout UI
 */
export async function handleLogoutCommand(): Promise<UIMessage> {
  try {
    // Get authentication status using the SDK
    const authStatus = await mix.authentication.getAuthStatus();
    
    // Format providers for UI - only include authenticated providers
    const { formattedProviders, hasAuthenticatedProvider } = formatLogoutProviders(
      authStatus.providers || {}
    );
    
    if (hasAuthenticatedProvider) {
      return {
        content: "Select a provider to log out from:",
        from: "assistant",
        frontend_only: true,
        logoutData: {
          providers: formattedProviders
        }
      };
    } else {
      return {
        content: "❌ **Not authenticated with any provider**\n\nUse `/login` to authenticate first.",
        from: "assistant",
        frontend_only: true
      };
    }
  } catch (error) {
    return {
      content: `Failed to check authentication status: ${error instanceof Error ? error.message : "Unknown error"}`,
      from: "assistant",
      frontend_only: true,
      suppressChatMessage: true
    };
  }
}

/**
 * Logs out from the specified provider by deleting credentials
 */
export async function logoutProvider(provider: string): Promise<UIMessage> {
  try {
    // Delete credentials using the SDK
    await mix.authentication.deleteCredentials({ provider });
    
    // Get updated auth status to confirm logout
    const authStatus = await mix.authentication.getAuthStatus();
    const isStillAuthenticated = authStatus.providers?.[provider]?.authenticated || false;
    
    if (isStillAuthenticated) {
      return {
        content: `❌ Failed to log out from ${provider}. Please try again.`,
        from: "assistant",
        frontend_only: true,
        suppressChatMessage: true
      };
    }
    
    // Show success toast notification
    toast.success("Logged out successfully", {
      description: `You have been logged out from ${provider}`,
      duration: 3000
    });
    
    return {
      content: `✅ Successfully logged out from ${provider}`,
      from: "assistant",
      frontend_only: true,
      suppressChatMessage: true  // Hide this success message from the chat UI
    };
  } catch (error) {
    return {
      content: `❌ Failed to log out: ${error instanceof Error ? error.message : "Unknown error"}`,
      from: "assistant",
      frontend_only: true,
      suppressChatMessage: true
    };
  }
}