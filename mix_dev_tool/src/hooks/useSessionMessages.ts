import { useQuery } from "@tanstack/react-query";
import type { BackendMessage } from "mix-typescript-sdk/models";
import { CACHE_KEYS } from "@/lib/cache-keys";
import { convertBackendMessagesToUI } from "@/lib/messageUtils";
import { mix } from "@/lib/mix-sdk";
import type { UIMessage } from "@/types/message";

async function loadSessionMessages(
	sessionId: string,
): Promise<BackendMessage[]> {
	try {
		const response = await mix.messages.getSession({ id: sessionId });
		return response;
	} catch (error: unknown) {
		console.error("Failed to load messages:", error);
		console.error("Error details:", {
			name: error instanceof Error ? error.name : undefined,
			message: error instanceof Error ? error.message : String(error),
			stack: error instanceof Error ? error.stack : undefined,
		});
		throw error;
	}
}

async function loadAndConvertMessages(sessionId: string): Promise<UIMessage[]> {
	const backendMessages = await loadSessionMessages(sessionId);
	const uiMessages = await convertBackendMessagesToUI(backendMessages);
	return uiMessages;
}

export function useSessionMessages(sessionId: string | null) {
	const query = useQuery({
		queryKey: CACHE_KEYS.sessionMessages(sessionId || ""),
		queryFn: async () => {
			const result = sessionId ? await loadAndConvertMessages(sessionId) : [];
			return result;
		},
		enabled: !!sessionId,
	});

	return query;
}
