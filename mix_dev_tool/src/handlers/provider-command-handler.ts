import { mix } from "@/lib/mix-sdk";
import type { UIMessage } from "@/types/message";

/**
 * Handles the selection of a provider to set as preferred
 */
export async function handleProviderSelection(
	providerId: string,
): Promise<UIMessage> {
	try {
		if (!providerId) {
			throw new Error("No provider selected");
		}

		// Update preferences to use this provider as the preferred one
		await mix.preferences.update({
			preferredProvider: providerId,
		});

		// Verify the update was successful
		const verifyPrefs = await mix.preferences.get();
		const savedProvider = verifyPrefs.preferences?.preferredProvider;

		if (savedProvider !== providerId) {
			throw new Error("Failed to update provider preference");
		}

		// Get provider display name and available models
		const authStatus = await mix.authentication.getAuthStatus();
		const providerName =
			authStatus.providers?.[providerId]?.displayName || providerId;

		// Get models for this provider
		const preferences = await mix.preferences.get();
		const providerData = preferences.availableProviders?.[providerId];

		if (providerData && providerData.models && providerData.models.length > 0) {
			// Format models for UI
			const currentModel = preferences.preferences?.mainAgentModel || "";
			const formattedModels = providerData.models.map((modelId: string) => ({
				id: modelId,
				displayName: modelId,
				isSelected: modelId === currentModel,
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
						displayName: providerName,
					},
				},
			};
		}
		// No models available, just show success message
		return {
			content: `✅ Successfully set **${providerName}** as your default provider${providerData?.models?.length ? "" : "\n\nNo models available for this provider."}`,
			from: "assistant",
			frontend_only: true,
		};
	} catch (error) {
		return {
			content: `Failed to update provider preference: ${error instanceof Error ? error.message : "Unknown error"}`,
			from: "assistant",
			frontend_only: true,
		};
	}
}
