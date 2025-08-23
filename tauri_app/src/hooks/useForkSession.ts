import { useMutation, useQueryClient } from '@tanstack/react-query';
import { rpcCall } from '@/lib/rpc';
import type { SessionData } from '@/types/common';
import { invalidateSessionCaches } from '@/lib/session-cache';

interface ForkSessionParams {
  sourceSessionId: string;
  messageIndex: number;
  title?: string;
}

const forkSession = async (params: ForkSessionParams): Promise<SessionData> => {
  return await rpcCall<SessionData>('sessions.fork', params);
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
