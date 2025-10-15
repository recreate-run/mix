import { useCallback } from "react";
import { toast } from "sonner";
import {
	authenticateWithApiKey,
	handleOAuthCallback,
	startOAuthFlow,
} from "@/handlers/login-command-handler";
import {
	handleProviderSelection,
	handleStatusCommand,
} from "@/handlers/status-command-handler";
import {
	handleModelSelectionInHierarchy,
	handleUnifiedModelCommand,
} from "@/handlers/unified-model-command-handler";
import type {
	CommandSlashProps,
	HelpData,
	StatusData,
	ViewState,
} from "@/types/command-slash";
import type { HierarchicalModelData } from "@/types/provider";

interface UseCommandHandlersProps {
	onQueryClientInvalidate?: CommandSlashProps["onQueryClientInvalidate"];
	onClose: CommandSlashProps["onClose"];
	setStatusData: (data: StatusData) => void;
	setHierarchicalModelData: (data: HierarchicalModelData) => void;
	setHelpData: (data: HelpData) => void;
	goToView: (view: ViewState) => void;
}

export function useCommandHandlers({
	onQueryClientInvalidate,
	onClose,
	setStatusData,
	setHierarchicalModelData,
	setHelpData,
	goToView,
}: UseCommandHandlersProps) {
	// Handle the status command
	const handleStatusCommandSpecial = useCallback(async () => {
		try {
			const statusResult = await handleStatusCommand();

			if (statusResult.statusData) {
				setStatusData({
					providers: statusResult.statusData.providers,
				});
				goToView("status");
			} else if (statusResult.suppressChatMessage) {
				if (
					statusResult.content.includes("Failed") ||
					statusResult.content.includes("❌")
				) {
					toast.error(statusResult.content.replace("❌", "").trim());
				} else if (statusResult.content.includes("✅")) {
					toast.success(statusResult.content.replace("✅", "").trim());
				}
			}
		} catch (error) {
			console.error("Status command failed:", error);
			toast.error("Failed to check authentication status");
		}
	}, [setStatusData, goToView]);

	// Handle the unified model command
	const handleUnifiedModelCommandSpecial = useCallback(async () => {
		try {
			const modelResult = await handleUnifiedModelCommand();

			if (modelResult.hierarchicalModel) {
				setHierarchicalModelData(modelResult.hierarchicalModel);
				goToView("hierarchical-model");
			}
		} catch (error) {
			console.error("Model command failed:", error);
			toast.error("Failed to load model selection");
		}
	}, [setHierarchicalModelData, goToView]);

	// Handle the help command
	const handleHelpCommandSpecial = useCallback(async () => {
		try {
			const helpData = {
				menuItems: [
					{
						id: "documentation",
						name: "Documentation",
						description: "View Mix documentation",
						action: "link",
						url: "https://docs.mix.com",
					},
					{
						id: "commands",
						name: "Available Commands",
						description: "Show list of available slash commands",
						action: "commands",
					},
					{
						id: "support",
						name: "Support",
						description: "Get help and support",
						action: "link",
						url: "https://support.mix.com",
					},
				],
			};

			setHelpData(helpData);
			goToView("help");
		} catch (error) {
			console.error("Help command failed:", error);
			toast.error("Failed to load help menu");
		}
	}, [setHelpData, goToView]);

	// Handle provider selection from status view
	const handleProviderSelectionSpecial = useCallback(
		async (providerId: string) => {
			try {
				await handleProviderSelection(providerId);
				onClose();
			} catch (error) {
				console.error("Provider selection failed:", error);
				toast.error("Failed to select provider");
			}
		},
		[onClose],
	);

	// Handle model selection from unified model view
	const handleModelSelectionSpecial = useCallback(
		async (providerId: string, modelId: string) => {
			try {
				const result = await handleModelSelectionInHierarchy(
					providerId,
					modelId,
				);

				// Check if we need to invalidate the preferences cache
				if (result.shouldInvalidatePreferencesCache) {
					onQueryClientInvalidate?.(["preferences"]);
				}

				onClose();
			} catch (error) {
				console.error("Model selection failed:", error);
				toast.error("Failed to select model");
			}
		},
		[onClose, onQueryClientInvalidate],
	);

	// Handle login provider selection
	const handleLoginProviderSelectionSpecial = useCallback(
		async (providerId: string, authMethod: "api_key" | "oauth") => {
			try {
				if (authMethod === "oauth") {
					await startOAuthFlow(providerId);
					// OAuth state is handled by the individual auth functions
				}
				// For API key method, the component will handle showing the input form
			} catch (error) {
				console.error("Login provider selection failed:", error);
				toast.error("Failed to start authentication");
			}
		},
		[],
	);

	// Handle API key submission
	const handleApiKeySubmitSpecial = useCallback(
		async (providerId: string, apiKey: string) => {
			try {
				await authenticateWithApiKey(providerId, apiKey);
				onQueryClientInvalidate?.(["providers"]);
				onClose();
			} catch (error) {
				console.error("API key submission failed:", error);
				toast.error("Failed to authenticate with API key");
			}
		},
		[onQueryClientInvalidate, onClose],
	);

	// Handle OAuth code submission
	const handleOAuthCodeSubmitSpecial = useCallback(
		async (providerId: string, code: string, state?: string) => {
			try {
				await handleOAuthCallback(providerId, code, state || "");
				onQueryClientInvalidate?.(["providers"]);
				onClose();
			} catch (error) {
				console.error("OAuth code submission failed:", error);
				toast.error("Failed to complete OAuth authentication");
			}
		},
		[onQueryClientInvalidate, onClose],
	);

	return {
		handleStatusCommandSpecial,
		handleUnifiedModelCommandSpecial,
		handleHelpCommandSpecial,
		handleProviderSelectionSpecial,
		handleModelSelectionSpecial,
		handleLoginProviderSelectionSpecial,
		handleApiKeySubmitSpecial,
		handleOAuthCodeSubmitSpecial,
	};
}
