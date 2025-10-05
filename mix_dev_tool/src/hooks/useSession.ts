import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { toast } from "sonner";
import { CACHE_KEYS } from "@/lib/cache-keys";
import { mix } from "@/lib/mix-sdk";
import type { Session, SessionData } from "@/types/common";

interface CreateSessionParams {
	title: string;
}

async function createSession(params: CreateSessionParams): Promise<Session> {
	const response = await mix.sessions.create(params);

	return {
		id: response.id,
		title: response.title,
	};
}

export function useCreateSession() {
	const queryClient = useQueryClient();

	return useMutation({
		mutationFn: createSession,
		onSuccess: (data) => {
			// Set both session metadata and empty messages immediately
			queryClient.setQueryData(CACHE_KEYS.session(data.id), data);
			queryClient.setQueryData(CACHE_KEYS.sessionMessages(data.id), []);

			// Only invalidate sessions list to show the new session in sidebar
			// No need to invalidate specific session data since we just set it
			queryClient.invalidateQueries({ queryKey: CACHE_KEYS.sessions });
		},
	});
}

// Fetch actual session data from backend
export function useActiveSession(sessionId: string) {
	return useQuery({
		queryKey: CACHE_KEYS.session(sessionId),
		queryFn: async (): Promise<Session | null> => {
			const response = await mix.sessions.get({ id: sessionId });

			return {
				id: response.id,
				title: response.title,
			};
		},
		refetchOnWindowFocus: false,
		refetchOnMount: false, // Don't refetch if we have cached data
		enabled: !!sessionId, // Only run when sessionId exists
	});
}

// Delete session from backend
async function deleteSession(sessionId: string): Promise<void> {
	await mix.sessions.delete({ id: sessionId });
}

// Simple utility to find next session for navigation
function findNextSession(
	sessions: SessionData[],
	deletedSessionId: string,
): string | null {
	if (sessions.length <= 1) return null;

	const currentIndex = sessions.findIndex((s) => s.id === deletedSessionId);
	if (currentIndex === -1) return null;

	// Try next session, then previous
	if (currentIndex < sessions.length - 1) {
		return sessions[currentIndex + 1].id;
	}
	if (currentIndex > 0) {
		return sessions[currentIndex - 1].id;
	}

	return null;
}

interface UseDeleteSessionOptions {
	/**
	 * All sessions for navigation logic
	 */
	allSessions?: SessionData[];
	/**
	 * Current session ID to determine if navigation is needed
	 */
	currentSessionId?: string;
	/**
	 * Callback for navigation after successful deletion
	 */
	onNavigate?: (sessionId: string | null) => void;
}

/**
 * Delete session hook with optimistic updates (no cache invalidation)
 */
export function useDeleteSession(options: UseDeleteSessionOptions = {}) {
	const { allSessions = [], currentSessionId, onNavigate } = options;
	const queryClient = useQueryClient();

	return useMutation({
		mutationFn: deleteSession,
		onMutate: async (deletedSessionId) => {
			// Cancel outgoing refetches to prevent race conditions
			await queryClient.cancelQueries({ queryKey: CACHE_KEYS.sessions });

			// Optimistically mark session as deleting (don't remove yet)
			queryClient.setQueryData<SessionData[]>(
				CACHE_KEYS.sessions,
				(oldSessions = []) =>
					oldSessions.map((session) =>
						session.id === deletedSessionId
							? { ...session, isDeleting: true }
							: session,
					),
			);
		},
		onSuccess: (_, deletedSessionId) => {
			// Remove session from cache (optimistic update only, no refetch)
			queryClient.setQueryData<SessionData[]>(
				CACHE_KEYS.sessions,
				(oldSessions = []) =>
					oldSessions.filter((session) => session.id !== deletedSessionId),
			);

			// Remove individual session cache entries
			queryClient.removeQueries({
				queryKey: CACHE_KEYS.session(deletedSessionId),
			});
			queryClient.removeQueries({
				queryKey: CACHE_KEYS.sessionMessages(deletedSessionId),
			});

			// Handle navigation if needed
			if (currentSessionId === deletedSessionId && onNavigate) {
				const nextSessionId = findNextSession(allSessions, deletedSessionId);
				onNavigate(nextSessionId);
			}
		},
		onError: (error, deletedSessionId) => {
			// Undo the graying out on error
			queryClient.setQueryData<SessionData[]>(
				CACHE_KEYS.sessions,
				(oldSessions = []) =>
					oldSessions.map((session) =>
						session.id === deletedSessionId
							? { ...session, isDeleting: false }
							: session,
					),
			);

			toast("Failed to delete session", {
				description: error.message,
			});
		},
	});
}
