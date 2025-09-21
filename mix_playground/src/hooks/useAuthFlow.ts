import { useState } from "react";
import { mix } from "@/lib/mix-sdk";

type AuthMode = "code" | "apikey";

interface UseAuthFlowReturn {
	authCode: string;
	setAuthCode: (value: string) => void;
	apiKey: string;
	setApiKey: (value: string) => void;
	authMode: AuthMode;
	setAuthMode: (mode: AuthMode) => void;
	isLoading: boolean;
	showSuccess: boolean;
	oauthState: string;
	setOauthState: (state: string) => void;
	handleSubmit: () => Promise<void>;
}

export function useAuthFlow(): UseAuthFlowReturn {
	const [authCode, setAuthCode] = useState("");
	const [apiKey, setApiKey] = useState("");
	const [authMode, setAuthMode] = useState<AuthMode>("code");
	const [isLoading, setIsLoading] = useState(false);
	const [showSuccess, setShowSuccess] = useState(false);
	const [oauthState, setOauthState] = useState(""); // Add state parameter for OAuth

	async function handleSubmit() {
		let input = authMode === "code" ? authCode.trim() : apiKey.trim();
		if (!input) return;

		setIsLoading(true);
		try {
			let result;
			
			if (authMode === "code") {
				// Check if it looks like an API key first
				if (input.startsWith("sk-ant-")) {
					result = await mix.auth.setApiKey({ apiKey: input });
				} else {
					// Handle OAuth code with # character - we just need to handle it properly here
					
					try {
						// Try to handle the OAuth callback with the stored state
						result = await mix.authentication.handleOAuthCallback({
							provider: "anthropic", // Default to anthropic for OAuth
							code: input,
							state: oauthState
						});
						setShowSuccess(true);
						return; // Exit early on success
					} catch (oauthError) {
						console.error("OAuth callback failed:", oauthError);
						// Fall back to regular login if OAuth fails
						result = await mix.authentication.login();
					}
				}
			} else {
				result = await mix.auth.setApiKey({ apiKey: input });
			}

			// Handle response based on the method used
			if (authMode === "apikey") {
				// For API key authentication, check success property
				const apiKeyData = result as { success?: boolean };
				if (apiKeyData.success === true) {
					setShowSuccess(true);
				}
			} else if (authMode === "code") {
				// For OAuth authentication, check if we got an authUrl
				const oauthData = result as { authUrl?: string };
				if (oauthData.authUrl) {
					// OAuth flow initiated successfully - this is handled in the UI
				} else {
					// If no authUrl, fallback to API key mode
					setAuthMode("apikey");
				}
			}
		} catch (error) {
			const errorMsg =
				error instanceof Error ? error.message : "Authentication failed";
			if (
				errorMsg.includes("Cloudflare") ||
				errorMsg.includes("manual token") ||
				errorMsg.includes("API key") ||
				errorMsg.includes("OAuth")
			) {
				setAuthMode("apikey");
			}
		} finally {
			setIsLoading(false);
			setAuthCode("");
			setApiKey("");
		}
	}

	return {
		authCode,
		setAuthCode,
		apiKey,
		setApiKey,
		authMode,
		setAuthMode,
		isLoading,
		showSuccess,
		oauthState,
		setOauthState,
		handleSubmit,
	};
}