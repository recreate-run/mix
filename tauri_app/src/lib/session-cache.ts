import type { QueryClient } from '@tanstack/react-query';
import { CACHE_KEYS } from './cache-keys';
import type { SessionData } from '@/types/common';

export const invalidateSessionCaches = (queryClient: QueryClient, sessionId?: string) => {
  queryClient.invalidateQueries({ queryKey: CACHE_KEYS.sessions });
  if (sessionId) {
    queryClient.invalidateQueries({ queryKey: CACHE_KEYS.session(sessionId) });
    queryClient.invalidateQueries({ queryKey: CACHE_KEYS.sessionMessages(sessionId) });
  }
};

// Optimistic update functions for smooth UX
export const updateSessionInList = (
  queryClient: QueryClient,
  updater: (sessions: SessionData[]) => SessionData[]
) => {
  queryClient.setQueryData<SessionData[]>(CACHE_KEYS.sessions, (oldSessions = []) =>
    updater(oldSessions)
  );
};

export const optimisticallySelectSession = (
  queryClient: QueryClient,
) => {
  // Update the sessions list to reflect the current selection optimistically
  updateSessionInList(queryClient, (sessions) =>
    sessions.map(session => ({
      ...session,
      // Note: We don't track "current" in SessionData, so this is just for cache consistency
    }))
  );
};