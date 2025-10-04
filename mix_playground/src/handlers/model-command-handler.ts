import { mix } from '@/lib/mix-sdk';
import type { UIMessage } from '@/types/message';

/**
 * Handles the selection of a model to set for the current provider
 */
export async function handleModelSelection(
  modelId: string
): Promise<UIMessage> {
  try {
    if (!modelId) {
      throw new Error('No model selected');
    }

    // Get current preferences
    const preferences = await mix.preferences.get();
    if (!preferences.preferences?.preferredProvider) {
      throw new Error('No default provider set');
    }

    const providerId = preferences.preferences.preferredProvider;

    // Update preferences with the selected model
    await mix.preferences.update({
      preferredProvider: providerId, // Keep the same provider
      mainAgentModel: modelId, // Update the model
      subAgentModel: modelId, // Also update the sub agent model for consistency
    });

    // Verify the update was successful
    const verifyPrefs = await mix.preferences.get();
    const savedModel = verifyPrefs.preferences?.mainAgentModel;

    if (savedModel !== modelId) {
      throw new Error('Failed to update model preference');
    }

    // Get provider display name
    const providerData = preferences.availableProviders?.[providerId];
    const providerName = providerData?.displayName || providerId;

    return {
      content: `✅ Successfully set **${modelId}** as your default model for ${providerName}`,
      from: 'assistant',
      frontend_only: true,
    };
  } catch (error) {
    return {
      content: `Failed to update model preference: ${error instanceof Error ? error.message : 'Unknown error'}`,
      from: 'assistant',
      frontend_only: true,
    };
  }
}
