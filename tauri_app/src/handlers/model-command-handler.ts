import { mix } from "@/lib/mix-sdk";
import { UIMessage } from "@/types/message";

interface ModelInfo {
  id: string;
  displayName: string;
  isSelected?: boolean;
}

/**
 * Handles the model command by fetching available models for the current provider
 * and showing UI to select one
 */
export async function handleModelCommand(): Promise<UIMessage> {
  try {
    // Get current preferences including provider and model
    const preferences = await mix.preferences.getPreferences();
    
    // Get authentication status to check if authenticated
    const authStatus = await mix.authentication.getAuthStatus();
    
    if (!preferences.preferences?.preferredProvider) {
      return {
        content: "❌ **No default provider set**\n\nPlease set a default provider first using the `/provider` command.",
        from: "assistant",
        frontend_only: true
      };
    }
    
    const providerId = preferences.preferences.preferredProvider;
    
    // Check if the provider is authenticated
    if (!authStatus.providers?.[providerId]?.authenticated) {
      return {
        content: `❌ **Not authenticated with provider ${providerId}**\n\nPlease authenticate first using the \`/login\` command.`,
        from: "assistant",
        frontend_only: true
      };
    }
    
    // Get available models for the selected provider
    const providerData = preferences.availableProviders?.[providerId];
    if (!providerData || !providerData.models || providerData.models.length === 0) {
      return {
        content: `❌ **No models available for provider ${providerData?.displayName || providerId}**\n\nPlease contact support or try a different provider.`,
        from: "assistant",
        frontend_only: true
      };
    }
    
    // Format models for UI
    const currentModel = preferences.preferences.mainAgentModel || "";
    const formattedModels: ModelInfo[] = providerData.models.map((modelId: string) => ({
      id: modelId,
      displayName: modelId,
      isSelected: modelId === currentModel
    }));
    
    // Sort models - selected first, then alphabetically
    formattedModels.sort((a, b) => {
      // Selected model first
      if (a.isSelected !== b.isSelected) {
        return a.isSelected ? -1 : 1;
      }
      
      // Then alphabetically
      return a.displayName.localeCompare(b.displayName);
    });
    
    // Prepare a message for the UI
    const providerName = providerData.displayName || providerId;
    const content = `**Select a model for ${providerName}**\n\n${formattedModels.length} models available. ${currentModel ? `Current model: **${currentModel}**` : 'No model currently selected.'}`;
    
    // Return message with model selection UI
    return {
      content,
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
    
  } catch (error) {
    return {
      content: `Failed to get models: ${error instanceof Error ? error.message : "Unknown error"}`,
      from: "assistant",
      frontend_only: true
    };
  }
}

/**
 * Handles the selection of a model to set for the current provider
 */
export async function handleModelSelection(modelId: string): Promise<UIMessage> {
  try {
    if (!modelId) {
      throw new Error("No model selected");
    }
    
    // Get current preferences
    const preferences = await mix.preferences.getPreferences();
    if (!preferences.preferences?.preferredProvider) {
      throw new Error("No default provider set");
    }
    
    const providerId = preferences.preferences.preferredProvider;
    
    // Update preferences with the selected model
    await mix.preferences.updatePreferences({
      preferredProvider: providerId, // Keep the same provider
      mainAgentModel: modelId, // Update the model
      subAgentModel: modelId  // Also update the sub agent model for consistency
    });
    
    // Verify the update was successful
    const verifyPrefs = await mix.preferences.getPreferences();
    const savedModel = verifyPrefs.preferences?.mainAgentModel;
    
    if (savedModel !== modelId) {
      throw new Error("Failed to update model preference");
    }
    
    // Get provider display name
    const providerData = preferences.availableProviders?.[providerId];
    const providerName = providerData?.displayName || providerId;
    
    return {
      content: `✅ Successfully set **${modelId}** as your default model for ${providerName}`,
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