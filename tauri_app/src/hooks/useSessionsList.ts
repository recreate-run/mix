import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';
import { rpcCall } from '@/lib/rpc';
import type { SessionData } from '@/types/common';
import { CACHE_KEYS } from '@/lib/cache-keys';
import { invalidateSessionCaches, optimisticallySelectSession } from '@/lib/session-cache';
import { toast } from "sonner"

export const TITLE_TRUNCATE_LENGTH = 100;

const loadSessionsList = async (): Promise<SessionData[]> => {
  const result = await rpcCall<SessionData[]>('sessions.list', {});
  return result || [];
};

export const useSessionsList = () => {
  return useQuery({
    queryKey: CACHE_KEYS.sessions,
    queryFn: loadSessionsList,
    refetchOnWindowFocus: false,
  });
};

const selectSession = async (sessionId: string): Promise<void> => {
  await rpcCall('sessions.select', { id: sessionId });
};

export const useSelectSession = () => {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: selectSession,
    onMutate: () => {
      // Optimistic update for instant UI feedback
      optimisticallySelectSession(queryClient);
    },
    onSuccess: () => {
      // Only invalidate sessions list, not individual session data
      // This prevents unnecessary re-fetches that cause flashing
      queryClient.invalidateQueries({ queryKey: CACHE_KEYS.sessions });
    },
  });
};

const deleteSession = async (sessionId: string): Promise<void> => {
  await rpcCall('sessions.delete', { id: sessionId });
};

export const useDeleteSession = () => {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: deleteSession,
    onMutate: async (deletedSessionId) => {
      // Cancel outgoing refetches to prevent race conditions
      await queryClient.cancelQueries({ queryKey: CACHE_KEYS.sessions });

      // Optimistically mark session as deleting (don't remove yet)
      queryClient.setQueryData<SessionData[]>(CACHE_KEYS.sessions, (oldSessions = []) =>
        oldSessions.map(session =>
          session.id === deletedSessionId
            ? { ...session, isDeleting: true }
            : session
        )
      );
    },
    onSuccess: (_, deletedSessionId) => {
      // Now actually remove the session from cache
      queryClient.setQueryData<SessionData[]>(CACHE_KEYS.sessions, (oldSessions = []) =>
        oldSessions.filter(session => session.id !== deletedSessionId)
      );

      // Remove the individual session cache entries
      queryClient.removeQueries({ queryKey: CACHE_KEYS.session(deletedSessionId) });
      queryClient.removeQueries({ queryKey: CACHE_KEYS.sessionMessages(deletedSessionId) });

      invalidateSessionCaches(queryClient, deletedSessionId);
    },
    onError: (error, deletedSessionId) => {
      // Just undo the graying out
      queryClient.setQueryData<SessionData[]>(CACHE_KEYS.sessions, (oldSessions = []) =>
        oldSessions.map(session =>
          session.id === deletedSessionId
            ? { ...session, isDeleting: false }
            : session
        )
      );

      toast("Failed to delete session", {
        description: error.message,
      })
    },
  });
};
