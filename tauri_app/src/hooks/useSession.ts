import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';
import { rpcCall } from '@/lib/rpc';
import type { Session } from '@/types/common';
import { CACHE_KEYS } from '@/lib/cache-keys';
import { invalidateSessionCaches } from '@/lib/session-cache';

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
      queryClient.setQueryData(CACHE_KEYS.session(data.id), data);
      invalidateSessionCaches(queryClient);
    },
  });
};

// Fetch actual session data from backend
export const useActiveSession = (sessionId: string) => {
  return useQuery({
    queryKey: CACHE_KEYS.session(sessionId),
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
    staleTime: 5 * 60 * 1000, // 5 minutes - reduce from infinite to allow some updates
    gcTime: 10 * 60 * 1000, // 10 minutes - keep in cache longer
    refetchOnWindowFocus: false,
    refetchOnMount: false, // Don't refetch if we have cached data
    enabled: !!sessionId, // Only run when sessionId exists
  });
};
