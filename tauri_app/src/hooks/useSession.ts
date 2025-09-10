import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';
import { mix } from '@/lib/mix-sdk';
import type { Session } from '@/types/common';
import { CACHE_KEYS } from '@/lib/cache-keys';

interface CreateSessionParams {
  title: string;
  workingDirectory?: string;
}

const createSession = async (params: CreateSessionParams): Promise<Session> => {
  const response = await mix.sessions.create(params);

  if (response.error) {
    throw new Error(response.error.message || 'Failed to create session');
  }

  if (!response.data) {
    throw new Error('No session data returned from server');
  }

  return {
    id: response.data.id,
    title: response.data.title,
    workingDirectory: response.data.workingDirectory,
  };
};

export const useCreateSession = () => {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: createSession,
    onSuccess: (data) => {
      // Set both session metadata and empty messages immediately
      queryClient.setQueryData(CACHE_KEYS.session(data.id), data);
      queryClient.setQueryData(CACHE_KEYS.sessionMessages(data.id), []);
      
      // Only invalidate sessions list to show the new session in sidebar
      // No need to invalidate specific session data since we just set it
      queryClient.invalidateQueries({ queryKey: CACHE_KEYS.sessions });
    },
  });
};

// Fetch actual session data from backend
export const useActiveSession = (sessionId: string) => {
  return useQuery({
    queryKey: CACHE_KEYS.session(sessionId),
    queryFn: async (): Promise<Session | null> => {
      const response = await mix.sessions.get({ id: sessionId });

      if (response.error) {
        throw new Error(`Session not found (${sessionId}): ${response.error.message}`);
      }

      if (!response.data) {
        throw new Error(`No session data returned for session: ${sessionId}`);
      }

      return {
        id: response.data.id,
        title: response.data.title,
        workingDirectory: response.data.workingDirectory,
      };
    },
    staleTime: 5 * 60 * 1000, // 5 minutes - reduce from infinite to allow some updates
    gcTime: 10 * 60 * 1000, // 10 minutes - keep in cache longer
    refetchOnWindowFocus: false,
    refetchOnMount: false, // Don't refetch if we have cached data
    enabled: !!sessionId, // Only run when sessionId exists
  });
};
