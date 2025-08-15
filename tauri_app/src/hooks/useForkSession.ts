import { useMutation, useQueryClient } from '@tanstack/react-query';
import { rpcCall } from '@/lib/rpc';
import type { Session } from '@/types/common';

interface ForkSessionParams {
  sourceSessionId: string;
  messageIndex: number;
  title?: string;
}

const forkSession = async (params: ForkSessionParams): Promise<Session> => {
  return await rpcCall<Session>('sessions.fork', params);
};

export const useForkSession = () => {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: forkSession,
    onSuccess: (newSession) => {
      queryClient.setQueryData(['session'], newSession);
    },
  });
};
