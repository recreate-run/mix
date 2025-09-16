import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';
import { mix } from '@/lib/mix-sdk';
import type { SessionData } from '@/types/common';
import type { SessionData as SDKSessionData } from 'mix-typescript-sdk/models';
import { CACHE_KEYS } from '@/lib/cache-keys';
import { invalidateSessionCaches } from '@/lib/session-cache';
import { toast } from "sonner";

export const TITLE_TRUNCATE_LENGTH = 100;

const loadSessionsList = async (): Promise<SessionData[]> => {
  const response = await mix.sessions.list();

  // Handle null response from API
  if (!response) {
    return [];
  }

  // Ensure response is an array
  if (!Array.isArray(response)) {
    console.error("❌ API response is not an array:", typeof response, response);
    return [];
  }

  // Transform SDK SessionData to match local interface (Date -> string)
  const transformedSessions = response.map((session: SDKSessionData) => ({
    ...session,
    createdAt: session.createdAt instanceof Date ? session.createdAt.toISOString() : session.createdAt
  }));

  return transformedSessions;
};

export const useSessionsList = () => {
  return useQuery({
    queryKey: CACHE_KEYS.sessions,
    queryFn: loadSessionsList,
    refetchOnWindowFocus: false,
  });
};

// REMOVED: useSelectSession - part of stateless design migration
// Sessions are now selected explicitly via sessionId parameters instead of global state

const deleteSession = async (sessionId: string): Promise<void> => {
  await mix.sessions.delete({ id: sessionId });
};


// Simple utility to find next session for navigation
const findNextSession = (sessions: SessionData[], deletedSessionId: string): string | null => {
  if (sessions.length <= 1) return null;
  
  const currentIndex = sessions.findIndex(s => s.id === deletedSessionId);
  if (currentIndex === -1) return null;
  
  // Try next session, then previous
  if (currentIndex < sessions.length - 1) {
    return sessions[currentIndex + 1].id;
  } else if (currentIndex > 0) {
    return sessions[currentIndex - 1].id;
  }
  
  return null;
};

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
 * Enhanced useDeleteSession with simple navigation support
 */
export const useDeleteSession = (options: UseDeleteSessionOptions = {}) => {
  const { allSessions = [], currentSessionId, onNavigate } = options;
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

      // Handle navigation if needed
      if (currentSessionId === deletedSessionId && onNavigate) {
        const nextSessionId = findNextSession(allSessions, deletedSessionId);
        onNavigate(nextSessionId);
      }
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
      });
    },
  });
};
