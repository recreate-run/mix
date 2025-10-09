import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { toast } from "sonner";
import { CACHE_KEYS } from "@/lib/cache-keys";
import { mix } from "@/lib/mix-sdk";

interface ToolInfo {
	provider: string;
	displayName: string;

	description: string;
	authenticated: boolean;
	apiKeyRequired: boolean;
}

interface ToolCategory {
	displayName: string;
	tools: ToolInfo[];
}

interface ToolsStatus {
	categories: Record<string, ToolCategory>;
}

interface StoreCredentialsRequest {
	provider: string;
	api_key: string;
}

async function fetchToolsStatus(): Promise<ToolsStatus> {
	try {
		// Return empty structure if tools module not available
		if (!mix.tools?.getToolsStatus) {
			console.warn("Tools API not available in current SDK version");
			return { categories: {} };
		}

		try {
			const response = await mix.tools.getToolsStatus();
			// Transform SDK response to match our interface
			const transformedCategories: Record<string, ToolCategory> = {};

			if (response.categories) {
				Object.entries(response.categories).forEach(([key, category]) => {
					transformedCategories[key] = {
						displayName: category.displayName || key,
						tools: (category.tools || []).map((tool) => ({
							provider: tool.provider || "",
							displayName: tool.displayName || tool.provider || "",
							description: tool.description || "",
							authenticated: tool.authenticated ?? false,
							apiKeyRequired: tool.apiKeyRequired ?? false,
						})),
					};
				});
			}

			return {
				categories: transformedCategories,
			};
		} catch (sdkError) {
			// SDK validation error - likely a mismatch between SDK schema and API response
			console.warn(
				"SDK validation error, returning empty structure:",
				sdkError,
			);
			return { categories: {} };
		}
	} catch (error) {
		console.error("Failed to fetch tools status:", error);
		throw new Error("Failed to fetch tools status");
	}
}

export function useToolsStatus() {
	return useQuery({
		queryKey: CACHE_KEYS.toolsStatus,
		queryFn: fetchToolsStatus,
		retry: 2,
	});
}

export function useStoreCredentials() {
	const queryClient = useQueryClient();

	return useMutation({
		mutationFn: async (request: StoreCredentialsRequest) => {
			try {
				// Use the SDK authentication API for storing credentials
				await mix.authentication.storeApiKey({
					provider: request.provider as any,
					apiKey: request.api_key,
				});
				return { status: "success", provider: request.provider };
			} catch (error) {
				console.error("Failed to store credentials:", error);
				throw error;
			}
		},
		onSuccess: (_, variables) => {
			// Invalidate tools status to refresh authentication state
			queryClient.invalidateQueries({ queryKey: CACHE_KEYS.toolsStatus });

			toast.success(`${variables.provider} API key stored successfully`);
		},
		onError: (error: any) => {
			const message = error?.message || "Failed to store API key";
			toast.error(message);
		},
	});
}

export function useDeleteCredentials() {
	const queryClient = useQueryClient();

	return useMutation({
		mutationFn: async ({ provider }: { provider: string }) => {
			try {
				// Use the SDK authentication API for deleting credentials
				await mix.authentication.deleteCredentials({ provider });
				return { status: "success", provider };
			} catch (error) {
				console.error("Failed to delete credentials:", error);
				throw error;
			}
		},
		onSuccess: (_, variables) => {
			// Invalidate tools status to refresh authentication state
			queryClient.invalidateQueries({ queryKey: CACHE_KEYS.toolsStatus });

			toast.success(`${variables.provider} API key removed successfully`);
		},
		onError: (error: any) => {
			const message = error?.message || "Failed to remove API key";
			toast.error(message);
		},
	});
}
