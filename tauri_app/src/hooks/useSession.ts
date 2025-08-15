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
      queryClient.setQueryData(['session'], data);
    },
  });
};

// Original useSession hook - unchanged for backward compatibility
export const useSession = (workingDirectory?: string) => {
  return useQuery({
    queryKey: ['session', workingDirectory],
    queryFn: () => createSession({ title: 'Chat Session', workingDirectory }),
    staleTime: Number.POSITIVE_INFINITY,
    refetchOnWindowFocus: false,
  });
};

// Simplified hook for URL-based session management  
export const useActiveSession = (sessionId: string, workingDirectory?: string) => {
  return useQuery({
    queryKey: ['session', sessionId],
    queryFn: async (): Promise<Session> => {
      // Return session info based on sessionId
      return {
        id: sessionId,
        title: 'Chat Session',
        workingDirectory,
      };
    },
    staleTime: Number.POSITIVE_INFINITY,
    refetchOnWindowFocus: false,
  });
};
