import { useQuery } from "@tanstack/react-query";
import type { SessionData as SDKSessionData } from "mix-typescript-sdk/models";
import { CACHE_KEYS } from "@/lib/cache-keys";
import { mix } from "@/lib/mix-sdk";
import type { SessionData } from "@/types/common";

export const TITLE_TRUNCATE_LENGTH = 100;

async function loadSessionsList(): Promise<SessionData[]> {
	const response = await mix.sessions.list();

	// Handle null response from API
	if (!response) {
		return [];
	}

	// Ensure response is an array
	if (!Array.isArray(response)) {
		console.error(
			"❌ API response is not an array:",
			typeof response,
			response,
		);
		return [];
	}

	// Transform SDK SessionData to match local interface (Date -> string)
	const transformedSessions = response.map((session: SDKSessionData) => ({
		...session,
		createdAt:
			session.createdAt instanceof Date
				? session.createdAt.toISOString()
				: session.createdAt,
	}));

	return transformedSessions;
}

export function useSessionsList() {
	return useQuery({
		queryKey: CACHE_KEYS.sessions,
		queryFn: loadSessionsList,
		refetchOnWindowFocus: false,
	});
}