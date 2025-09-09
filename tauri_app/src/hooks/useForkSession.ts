import { useMutation, useQueryClient } from '@tanstack/react-query';
import { mix } from '@/lib/mix-sdk';
import type { SessionData } from '@/types/common';
import { invalidateSessionCaches } from '@/lib/session-cache';

interface ForkSessionParams {
  sourceSessionId: string;
  messageIndex: number;
  title?: string;
}

const forkSession = async (params: ForkSessionParams): Promise<SessionData> => {
  const response = await mix.sessions.fork({ 
    id: params.sourceSessionId,
    requestBody: {
      messageIndex: params.messageIndex,
      title: params.title
    }
  });

  if (response.error) {
    throw new Error(response.error.message || 'Failed to fork session');
  }

  if (!response.data) {
    throw new Error('No session data returned from fork operation');
  }

  // Transform SDK SessionData to match local interface (Date -> string)
  return {
    ...response.data,
    createdAt: response.data.createdAt instanceof Date ? response.data.createdAt.toISOString() : response.data.createdAt
  };
};

export const useForkSession = () => {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: forkSession,
    onSuccess: (newSession) => {
      invalidateSessionCaches(queryClient, newSession.id);
    },
  });
};
