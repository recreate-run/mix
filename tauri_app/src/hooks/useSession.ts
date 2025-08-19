import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';
import { rpcCall } from '@/lib/rpc';
import type { Session } from '@/types/common';

interface CreateSessionParams {
  title: string;
  workingDirectory?: string;
}

const createSession = async (params: CreateSessionParams): Promise<Session> => {
  const result = await rpcCall<any>('sessions.create', params);
  const sessionId = result?.id || result;

  if (!sessionId) {
    throw new Error('No session ID returned from server');
  }

  return {
    id: sessionId,
    title: result?.title || 'Chat Session',
    workingDirectory: result?.workingDirectory,
  };
};

export const useCreateSession = () => {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: createSession,
    onSuccess: (data) => {
      queryClient.setQueryData(['session', data.id], data);
      queryClient.invalidateQueries({ queryKey: ['sessions'] });
    },
  });
};

// Fetch actual session data from backend
export const useActiveSession = (sessionId: string) => {
  return useQuery({
    queryKey: ['session', sessionId],
    queryFn: async (): Promise<Session | null> => {
      try {
        const sessionData = await rpcCall<Session>('sessions.get', {
          id: sessionId,
        });
        return sessionData;
      } catch (error) {
        console.log('Session not found:', sessionId);
        return null;
      }
    },
    staleTime: Number.POSITIVE_INFINITY,
    refetchOnWindowFocus: false,
  });
};
