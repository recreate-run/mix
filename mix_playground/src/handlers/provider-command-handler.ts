import { mix } from "@/lib/mix-sdk";
import { UIMessage } from "@/types/message";
import { ProviderInfo } from "@/types/provider";

/**
 * Format provider information for the provider UI
 */
function formatProviders(providers: Record<string, any>): {
  formattedProviders: ProviderInfo[];
  hasAuthenticatedProvider: boolean;
  preferredProvider?: string;
} {
  const formattedProviders: ProviderInfo[] = [];
  
  let hasAuthenticatedProvider = false;
  let preferredProvider: string | undefined = undefined;
  
  Object.entries(providers).forEach(([id, provider]) => {
    // Extract the base name without the star symbol
    const name = provider.displayName || id;
    const cleanName = name.replace(" ⭐", "");
    const isPreferred = name.includes("⭐");
    
    if (isPreferred) {
      preferredProvider = id;
    }
    
    // Format provider for UI
    formattedProviders.push({
      id,
      displayName: cleanName,
      authenticated: provider.authenticated,
      authMethod: provider.authMethod,
      isPreferred,
      authMethods: ["api_key", "oauth"] // Standard auth methods
    });
    
    if (provider.authenticated) {
      hasAuthenticatedProvider = true;
    }
  });
  
  // Sort providers - authenticated and preferred first, then alphabetically
  formattedProviders.sort((a, b) => {
    // Preferred provider first
    if (a.isPreferred !== b.isPreferred) {
      return a.isPreferred ? -1 : 1;
    }
    
    // Then authenticated providers
    if (a.authenticated !== b.authenticated) {
      return a.authenticated ? -1 : 1;
    }
    
    // Then alphabetically
    return a.displayName.localeCompare(b.displayName);
  });
  
  return {
    formattedProviders,
    hasAuthenticatedProvider,
    preferredProvider
  };
}

/**
 * Handles the provider command by fetching authenticated providers and showing UI to select one
 */
export async function handleProviderCommand(): Promise<UIMessage> {
  try {
    // Get authentication status using the SDK
    const authStatus = await mix.authentication.getAuthStatus();
    
    // Format providers for UI
    const { formattedProviders, hasAuthenticatedProvider, preferredProvider } = formatProviders(
      authStatus.providers || {}
    );
    
    if (!hasAuthenticatedProvider) {
      return {
        content: "❌ **Not authenticated with any provider**\n\nYou need to authenticate first. Please use API keys or OAuth through your provider's authentication flow.",
        from: "assistant",
        frontend_only: true
      };
    }
    
    // Prepare a message for the UI
    const authenticatedCount = formattedProviders.filter(p => p.authenticated).length;
    const content = `**Select a provider to use as default**\n\nYou have ${authenticatedCount} authenticated provider${authenticatedCount > 1 ? 's' : ''}. ${preferredProvider ? `Current default: **${formattedProviders.find(p => p.isPreferred)?.displayName}**` : 'No default provider set.'}`;
    
    // Return message with provider selection UI
    return {
      content,
      from: "assistant",
      frontend_only: true,
      provider: {
        providers: formattedProviders,
        currentProvider: preferredProvider
      }
    };
    
  } catch (error) {
    return {
      content: `Failed to get providers: ${error instanceof Error ? error.message : "Unknown error"}`,
      from: "assistant",
      frontend_only: true
    };
  }
}

/**
 * Handles the selection of a provider to set as preferred
 */
export async function handleProviderSelection(providerId: string): Promise<UIMessage> {
  try {
    if (!providerId) {
      throw new Error("No provider selected");
    }
    
    // Update preferences to use this provider as the preferred one
    await mix.preferences.update({
      preferredProvider: providerId
    });
    
    // Verify the update was successful
    const verifyPrefs = await mix.preferences.get();
    const savedProvider = verifyPrefs.preferences?.preferredProvider;
    
    if (savedProvider !== providerId) {
      throw new Error("Failed to update provider preference");
    }
    
    // Get provider display name and available models
    const authStatus = await mix.authentication.getAuthStatus();
    const providerName = authStatus.providers?.[providerId]?.displayName || providerId;
    
    // Get models for this provider
    const preferences = await mix.preferences.get();
    const providerData = preferences.availableProviders?.[providerId];
    
    if (providerData && providerData.models && providerData.models.length > 0) {
      // Format models for UI
      const currentModel = preferences.preferences?.mainAgentModel || "";
      const formattedModels = providerData.models.map((modelId: string) => ({
        id: modelId,
        displayName: modelId,
        isSelected: modelId === currentModel
      }));
      
      // Sort models - selected first, then alphabetically
      formattedModels.sort((a: any, b: any) => {
        // Selected model first
        if (a.isSelected !== b.isSelected) {
          return a.isSelected ? -1 : 1;
        }
        
        // Then alphabetically
        return a.displayName.localeCompare(b.displayName);
      });
      
      // Success message with model selection UI
      return {
        content: `✅ Successfully set **${providerName}** as your default provider\n\nNow select a model for this provider:`,
        from: "assistant",
        frontend_only: true,
        model: {
          models: formattedModels,
          currentModel,
          provider: {
            id: providerId,
            displayName: providerName
          }
        }
      };
    } else {
      // No models available, just show success message
      return {
        content: `✅ Successfully set **${providerName}** as your default provider${!providerData?.models?.length ? "\n\nNo models available for this provider." : ""}`,
        from: "assistant",
        frontend_only: true
      };
    }
  } catch (error) {
    return {
      content: `Failed to update provider preference: ${error instanceof Error ? error.message : "Unknown error"}`,
      from: "assistant",
      frontend_only: true
    };
  }
}