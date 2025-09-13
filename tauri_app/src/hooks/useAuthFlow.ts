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
	handleSubmit: () => Promise<void>;
}

export function useAuthFlow(): UseAuthFlowReturn {
	const [authCode, setAuthCode] = useState("");
	const [apiKey, setApiKey] = useState("");
	const [authMode, setAuthMode] = useState<AuthMode>("code");
	const [isLoading, setIsLoading] = useState(false);
	const [showSuccess, setShowSuccess] = useState(false);

	const handleSubmit = async () => {
		const input = authMode === "code" ? authCode.trim() : apiKey.trim();
		if (!input) return;

		setIsLoading(true);
		try {
			let result;
			
			if (authMode === "code") {
				if (input.startsWith("sk-ant-")) {
					result = await mix.auth.setApiKey({ apiKey: input });
				} else {
					result = await mix.authentication.login();
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
	};

	return {
		authCode,
		setAuthCode,
		apiKey,
		setApiKey,
		authMode,
		setAuthMode,
		isLoading,
		showSuccess,
		handleSubmit,
	};
}