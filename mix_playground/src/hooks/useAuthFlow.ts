import { useState, useCallback } from "react";
import { useMutation, useQueryClient } from "@tanstack/react-query";
import { mix } from "@/lib/mix-sdk";

type AuthMode = "code" | "apikey";

interface AuthParams {
	input: string;
	mode: AuthMode;
	oauthState?: string;
}

interface AuthResponse {
	success?: boolean;
	authUrl?: string;
}

async function authenticate(params: AuthParams): Promise<AuthResponse> {
	const { input, mode, oauthState } = params;

	if (mode === "apikey" || input.startsWith("sk-ant-")) {
		return await mix.auth.setApiKey({ apiKey: input });
	}

	try {
		await mix.authentication.handleOAuthCallback({
			provider: "anthropic",
			code: input,
			state: oauthState || ""
		});
		return { success: true };
	} catch {
		return await mix.authentication.login();
	}
}

interface UseAuthFlowReturn {
	authCode: string;
	setAuthCode: (value: string) => void;
	apiKey: string;
	setApiKey: (value: string) => void;
	authMode: AuthMode;
	setAuthMode: (mode: AuthMode) => void;
	oauthState: string;
	setOauthState: (state: string) => void;
	isLoading: boolean;
	showSuccess: boolean;
	handleSubmit: () => void;
}

export function useAuthFlow(): UseAuthFlowReturn {
	const queryClient = useQueryClient();
	const [authCode, setAuthCode] = useState("");
	const [apiKey, setApiKey] = useState("");
	const [authMode, setAuthMode] = useState<AuthMode>("code");
	const [oauthState, setOauthState] = useState("");
	const [showSuccess, setShowSuccess] = useState(false);

	const authMutation = useMutation({
		mutationFn: authenticate,
		onSuccess: (data) => {
			if (data.success) {
				setAuthCode("");
				setApiKey("");
				setShowSuccess(true);
				setTimeout(() => setShowSuccess(false), 3000);
				queryClient.invalidateQueries();
			} else if (!data.authUrl) {
				setAuthMode("apikey");
			}
		},
		onError: (error: Error) => {
			const msg = error.message;
			if (msg?.includes("Cloudflare") || msg?.includes("API key") || msg?.includes("OAuth")) {
				setAuthMode("apikey");
			}
		}
	});

	const handleSubmit = useCallback(() => {
		const input = (authMode === "code" ? authCode : apiKey).trim();
		if (!input || authMutation.isPending) return;

		authMutation.mutate({ input, mode: authMode, oauthState });
	}, [authCode, apiKey, authMode, oauthState, authMutation]);

	const handleModeChange = useCallback((mode: AuthMode) => {
		authMutation.reset();
		setShowSuccess(false);
		setAuthMode(mode);
	}, [authMutation]);

	return {
		authCode,
		setAuthCode,
		apiKey,
		setApiKey,
		authMode,
		setAuthMode: handleModeChange,
		oauthState,
		setOauthState,
		isLoading: authMutation.isPending,
		showSuccess,
		handleSubmit,
	};
}