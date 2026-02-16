import { useQuery } from "@tanstack/react-query";

function getApiKeyFormat(providerId: string): string {
	const formats: Record<string, string> = {
		anthropic: "sk-ant-...",
		openai: "sk-...",
		openrouter: "sk-or-...",
	};
	return formats[providerId] || "API Key";
}

export interface Provider {
	id: string;
	displayName: string;
	authenticated: boolean;
	authMethods: ("api_key" | "oauth")[];
	authMethod: string;
	isPreferred?: boolean;
	apiKeyFormat?: string;
}

async function getProviders(): Promise<Provider[]> {
	const baseUrl = import.meta.env.VITE_BACKEND_URL || "http://localhost:8081";
	const response = await fetch(`${baseUrl}/api/auth/status`);

	if (!response.ok) {
		throw new Error("Failed to fetch auth status");
	}

	const data = await response.json();

	if (!data.providers) {
		return [];
	}

	// Transform the auth status response into provider list
	return Object.entries(data.providers).map(
		([id, status]: [string, unknown]) => {
			const providerStatus = status as {
				display_name?: string;
				authenticated?: boolean;
				auth_method?: string;
			};
			// Determine available auth methods based on provider
			const authMethods: ("api_key" | "oauth")[] = [];
			if (id === "anthropic") {
				authMethods.push("oauth", "api_key");
			} else {
				authMethods.push("api_key");
			}

			return {
				id,
				displayName: providerStatus.display_name || id,
				authenticated: providerStatus.authenticated || false,
				authMethods,
				authMethod: providerStatus.auth_method || "none",
				isPreferred: id === "anthropic", // Hardcoded default provider
				apiKeyFormat: getApiKeyFormat(id),
			};
		},
	);
}

export function useProviders() {
	return useQuery({
		queryKey: ["auth", "providers"],
		queryFn: getProviders,
		retry: 2,
	});
}
