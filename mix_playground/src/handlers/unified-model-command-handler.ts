import { toast } from "sonner";
import { mix } from "@/lib/mix-sdk";
import type { ProviderWithModels, UIMessage } from "@/types";

/**
 * Format providers with their models for the hierarchical UI
 */
function formatProvidersWithModels(
	authProviders: Record<string, any>,
	availableProviders: Record<string, any>,
	currentModel?: string,
): {
	providers: ProviderWithModels[];
	hasAuthenticatedProvider: boolean;
	preferredProvider?: string;
} {
	const providers: ProviderWithModels[] = [];
	let hasAuthenticatedProvider = false;
	let preferredProvider: string | undefined;

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
			isSelected: modelId === currentModel,
		}));

		// Sort models - selected first, then alphabetically
		formattedModels.sort((a: any, b: any) => {
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
			authMethods: ["api_key"], // Default auth methods, could be enhanced to read from config
			isPreferred,
			models: formattedModels,
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
		preferredProvider,
	};
}

/**
 * Handles the unified model command - returns hierarchical data for CMDK
 */
export async function handleUnifiedModelCommand(): Promise<UIMessage> {
	try {
		// Get authentication status and preferences
		const [authStatus, preferences] = await Promise.all([
			mix.authentication.getAuthStatus(),
			mix.preferences.get(),
		]);

		// Format providers with their models
		const { providers, hasAuthenticatedProvider, preferredProvider } =
			formatProvidersWithModels(
				authStatus.providers || {},
				preferences.availableProviders || {},
				preferences.preferences?.mainAgentModel,
			);

		if (!hasAuthenticatedProvider) {
			return {
				content:
					"Not authenticated with any provider. Please authenticate using API keys or OAuth through your provider's authentication flow.",
				from: "assistant",
				frontend_only: true,
			};
		}

		// Return hierarchical data for CMDK wrapped in UIMessage
		return {
			content: "",
			from: "assistant",
			frontend_only: true,
			hierarchicalModel: {
				providers,
				currentProvider: preferredProvider,
				currentModel: preferences.preferences?.mainAgentModel,
			},
		};
	} catch (error) {
		return {
			content: `Failed to get providers and models: ${error instanceof Error ? error.message : "Unknown error"}`,
			from: "assistant",
			frontend_only: true,
			suppressChatMessage: true,
		};
	}
}

/**
 * Handles the selection of a model (second level of hierarchy)
 */
export async function handleModelSelectionInHierarchy(
	providerId: string,
	modelId: string,
): Promise<UIMessage> {
	try {
		if (!(modelId && providerId)) {
			throw new Error("No model or provider selected");
		}

		// Update preferences with the selected model and provider
		await mix.preferences.update({
			preferredProvider: providerId,
			mainAgentModel: modelId,
			subAgentModel: modelId, // Also update the sub agent model for consistency
		});

		// Verify the update was successful
		const verifyPrefs = await mix.preferences.get();
		const savedModel = verifyPrefs.preferences?.mainAgentModel;

		if (savedModel !== modelId) {
			throw new Error("Failed to update model preference");
		}

		// Get provider display name and clean it
		const authStatus = await mix.authentication.getAuthStatus();
		const rawProviderName =
			authStatus.providers?.[providerId]?.displayName || providerId;
		const providerName = rawProviderName.replace(" ⭐", "");

		// This function is called directly from the chat-app component
		// The caller needs to invalidate the preferences cache to show updated model info

		// Show success toast notification
		try {
			// Then try the more complex version
			toast.success("Model updated", {
				description: `${modelId} is now your default model for ${providerName}`,
				duration: 3000,
			});
			console.log("Toast notifications triggered for model update");
		} catch (toastError) {
			console.error("Failed to show toast:", toastError);
		}

		return {
			content: `✅ Successfully set ${modelId} as your default model for ${providerName}`,
			from: "assistant",
			frontend_only: true,
			shouldInvalidatePreferencesCache: true, // Signal to invalidate preferences cache
			suppressChatMessage: true, // Hide this success message from the chat UI
		};
	} catch (error) {
		return {
			content: `Failed to update model preference: ${error instanceof Error ? error.message : "Unknown error"}`,
			from: "assistant",
			frontend_only: true,
			suppressChatMessage: true,
		};
	}
}
