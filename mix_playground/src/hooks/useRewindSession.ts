import { useMutation, useQueryClient } from '@tanstack/react-query';
import { mix } from '@/lib/mix-sdk';
import type { SessionData } from '@/types/common';
import { invalidateSessionCaches } from '@/lib/session-cache';

interface RewindSessionParams {
  sessionId: string;
  messageId: string;
  cleanupMedia?: boolean;
}

async function rewindSession(
  params: RewindSessionParams
): Promise<SessionData> {
  const response = await mix.sessions.rewindSession({
    id: params.sessionId,
    requestBody: {
      messageId: params.messageId,
      cleanupMedia: params.cleanupMedia ?? true,
    },
  });

  // Transform SDK SessionData to match local interface (Date -> string)
  return {
    ...response,
    createdAt:
      response.createdAt instanceof Date
        ? response.createdAt.toISOString()
        : response.createdAt,
  };
}

export function useRewindSession() {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: rewindSession,
    onSuccess: (updatedSession) => {
      // Invalidate caches to refresh the session data and messages
      invalidateSessionCaches(queryClient, updatedSession.id);
    },
  });
}
