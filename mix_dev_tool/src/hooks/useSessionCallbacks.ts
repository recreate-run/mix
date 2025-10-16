import { useMutation, useQueryClient } from "@tanstack/react-query";
import type { UpdateSessionCallbacksCallback } from "mix-typescript-sdk/models/operations/updatesessioncallbacks.js";
import type { SessionData } from "mix-typescript-sdk/models/sessiondata.js";
import { toast } from "sonner";
import { CACHE_KEYS } from "@/lib/cache-keys";
import { mix } from "@/lib/mix-sdk";
import { useActiveSession } from "./useSession";

/**
 * Get callbacks from active session (derives from useActiveSession)
 */
export function useSessionCallbacks(sessionId: string) {
	const query = useActiveSession(sessionId);

	return {
		...query,
		data: query.data
			? {
					callbacks: (query.data.callbacks ||
						[]) as UpdateSessionCallbacksCallback[],
					sessionId: query.data.id,
				}
			: undefined,
	};
}

interface UpdateCallbacksParams {
	sessionId: string;
	callbacks: Array<UpdateSessionCallbacksCallback>;
}

interface MutationContext {
	previousData: SessionData | undefined;
}

/**
 * Update session callbacks with optimistic updates
 */
export function useUpdateCallbacks() {
	const queryClient = useQueryClient();

	return useMutation<
		SessionData,
		Error,
		UpdateCallbacksParams,
		MutationContext
	>({
		mutationFn: async ({ sessionId, callbacks }: UpdateCallbacksParams) => {
			return await mix.sessions.updateSessionCallbacks({
				id: sessionId,
				requestBody: { callbacks },
			});
		},
		onMutate: async ({ sessionId, callbacks }) => {
			// Cancel outgoing refetches to prevent them from overwriting our optimistic update
			await queryClient.cancelQueries({
				queryKey: CACHE_KEYS.session(sessionId),
			});

			// Snapshot the previous value for rollback on error
			const previousData = queryClient.getQueryData<SessionData>(
				CACHE_KEYS.session(sessionId),
			);

			// Optimistically update the cache
			queryClient.setQueryData<SessionData>(
				CACHE_KEYS.session(sessionId),
				(oldData) => {
					if (!oldData) return oldData;
					return { ...oldData, callbacks };
				},
			);

			// Return context with snapshotted value for rollback
			return { previousData };
		},
		onSuccess: (data, { sessionId }) => {
			// Update cache with server response
			queryClient.setQueryData(CACHE_KEYS.session(sessionId), data);
			toast.success("Callbacks updated");
		},
		onError: (error, { sessionId }, context) => {
			// Rollback to previous state using context (no network request needed)
			if (context?.previousData) {
				queryClient.setQueryData(
					CACHE_KEYS.session(sessionId),
					context.previousData,
				);
			}

			toast.error("Failed to update callbacks", {
				description: error instanceof Error ? error.message : String(error),
			});
		},
		onSettled: (_data, _error, { sessionId }) => {
			// Always refetch after mutation completes (success or error) to ensure server state
			queryClient.invalidateQueries({
				queryKey: CACHE_KEYS.session(sessionId),
			});
		},
	});
}
