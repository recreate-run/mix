import { useMutation, useQueryClient } from '@tanstack/react-query';
import { mix } from '@/lib/mix-sdk';
import type { SessionData } from '@/types/common';
import { invalidateSessionCaches } from '@/lib/session-cache';

interface ForkSessionParams {
  sourceSessionId: string;
  messageIndex: number;
  title?: string;
}

async function forkSession(params: ForkSessionParams): Promise<SessionData> {
  const response = await mix.sessions.fork({ 
    id: params.sourceSessionId,
    requestBody: {
      messageIndex: params.messageIndex,
      title: params.title
    }
  });

  // Transform SDK SessionData to match local interface (Date -> string)
  return {
    ...response,
    createdAt: response.createdAt instanceof Date ? response.createdAt.toISOString() : response.createdAt
  };
}

export function useForkSession() {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: forkSession,
    onSuccess: (newSession) => {
      invalidateSessionCaches(queryClient, newSession.id);
    },
  });
}
