import type { QueryClient } from '@tanstack/react-query';
import { CACHE_KEYS } from './cache-keys';

export const invalidateSessionCaches = (queryClient: QueryClient, sessionId?: string) => {
  queryClient.invalidateQueries({ queryKey: CACHE_KEYS.sessions });
  if (sessionId) {
    queryClient.invalidateQueries({ queryKey: CACHE_KEYS.session(sessionId) });
    queryClient.invalidateQueries({ queryKey: CACHE_KEYS.sessionMessages(sessionId) });
  }
};

// REMOVED: optimisticallySelectSession - part of stateless design migration
// Session selection is now handled explicitly via sessionId parameters