import { mix } from "@/lib/mix-sdk";
import { UIMessage, ProviderWithModels } from "@/types";

/**
 * Format providers with their models for the hierarchical UI
 */
function formatProvidersWithModels(
  authProviders: Record<string, any>,
  availableProviders: Record<string, any>,
  currentModel?: string
): {
  providers: ProviderWithModels[];
  hasAuthenticatedProvider: boolean;
  preferredProvider?: string;
} {
  const providers: ProviderWithModels[] = [];
  let hasAuthenticatedProvider = false;
  let preferredProvider: string | undefined = undefined;

  Object.entries(authProviders).forEach(([id, authProvider]) => {
    // Extract the base name without the star symbol
    const name = authProvider.displayName || id;
    const cleanName = name.replace(" ⭐", "");
    const isPreferred = name.includes("⭐");
    
    if (isPreferred) {
      preferredProvider = id;
    }
    
    // Get models for this provider
    const providerData = availableProviders[id];
    const models = providerData?.models || [];
    
    // Format models for UI
    const formattedModels = models.map((modelId: string) => ({
      id: modelId,
      displayName: modelId,
      isSelected: modelId === currentModel
    }));
    
    // Sort models - selected first, then alphabetically
    formattedModels.sort((a, b) => {
      if (a.isSelected !== b.isSelected) {
        return a.isSelected ? -1 : 1;
      }
      return a.displayName.localeCompare(b.displayName);
    });
    
    // Format provider for UI
    providers.push({
      id,
      displayName: cleanName,
      authenticated: authProvider.authenticated,
      authMethod: authProvider.authMethod,
      isPreferred,
      models: formattedModels
    });
    
    if (authProvider.authenticated) {
      hasAuthenticatedProvider = true;
    }
  });
  
  // Sort providers - authenticated and preferred first, then alphabetically
  providers.sort((a, b) => {
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
    providers,
    hasAuthenticatedProvider,
    preferredProvider
  };
}

/**
 * Handles the unified model command - returns hierarchical data for CMDK
 */
export async function handleUnifiedModelCommand(): Promise<{
  providers: ProviderWithModels[];
  currentProvider?: string;
  currentModel?: string;
  error?: string;
}> {
  try {
    // Get authentication status and preferences
    const [authStatus, preferences] = await Promise.all([
      mix.authentication.getAuthStatus(),
      mix.preferences.getPreferences()
    ]);
    
    // Format providers with their models
    const { providers, hasAuthenticatedProvider, preferredProvider } = formatProvidersWithModels(
      authStatus.providers || {},
      preferences.availableProviders || {},
      preferences.preferences?.mainAgentModel
    );
    
    if (!hasAuthenticatedProvider) {
      return {
        providers: [],
        error: "Not authenticated with any provider. Try the /login command to authenticate with a provider."
      };
    }
    
    // Return hierarchical data for CMDK
    return {
      providers,
      currentProvider: preferredProvider,
      currentModel: preferences.preferences?.mainAgentModel
    };
    
  } catch (error) {
    return {
      providers: [],
      error: `Failed to get providers and models: ${error instanceof Error ? error.message : "Unknown error"}`
    };
  }
}

/**
 * Simple function to update provider preferences for CMDK flow
 */
export async function updateProviderPreference(providerId: string): Promise<void> {
  if (!providerId) {
    throw new Error("No provider selected");
  }
  
  // Update preferences to use this provider as the preferred one
  await mix.preferences.updatePreferences({
    preferredProvider: providerId
  });
}

/**
 * Handles the selection of a provider (first level of hierarchy) - legacy function
 */
export async function handleProviderSelectionInHierarchy(providerId: string): Promise<UIMessage> {
  try {
    await updateProviderPreference(providerId);
    
    // Get updated data to show models for selected provider
    const [authStatus, preferences] = await Promise.all([
      mix.authentication.getAuthStatus(),
      mix.preferences.getPreferences()
    ]);
    
    const providerName = authStatus.providers?.[providerId]?.displayName || providerId;
    const providerData = preferences.availableProviders?.[providerId];
    
    if (!providerData?.models?.length) {
      return {
        content: `✅ Successfully set **${providerName}** as your default provider\n\n❌ No models available for this provider.`,
        from: "assistant",
        frontend_only: true
      };
    }
    
    // Return to the hierarchical model command to show updated state
    return await handleUnifiedModelCommand();
    
  } catch (error) {
    return {
      content: `Failed to update provider preference: ${error instanceof Error ? error.message : "Unknown error"}`,
      from: "assistant",
      frontend_only: true
    };
  }
}

/**
 * Handles the selection of a model (second level of hierarchy)
 */
export async function handleModelSelectionInHierarchy(providerId: string, modelId: string): Promise<UIMessage> {
  try {
    if (!modelId || !providerId) {
      throw new Error("No model or provider selected");
    }
    
    // Update preferences with the selected model and provider
    await mix.preferences.updatePreferences({
      preferredProvider: providerId,
      mainAgentModel: modelId,
      subAgentModel: modelId  // Also update the sub agent model for consistency
    });
    
    // Verify the update was successful
    const verifyPrefs = await mix.preferences.getPreferences();
    const savedModel = verifyPrefs.preferences?.mainAgentModel;
    
    if (savedModel !== modelId) {
      throw new Error("Failed to update model preference");
    }
    
    // Get provider display name and clean it
    const authStatus = await mix.authentication.getAuthStatus();
    const rawProviderName = authStatus.providers?.[providerId]?.displayName || providerId;
    const providerName = rawProviderName.replace(" ⭐", "");
    
    return {
      content: `✅ Successfully set ${modelId} as your default model for ${providerName}`,
      from: "assistant",
      frontend_only: true
    };
  } catch (error) {
    return {
      content: `Failed to update model preference: ${error instanceof Error ? error.message : "Unknown error"}`,
      from: "assistant",
      frontend_only: true
    };
  }
}