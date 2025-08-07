import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';
import { useEffect, useState } from 'react';
import { rpcCall } from '@/lib/rpc';

interface CreateSessionParams {
  title: string;
  workingDirectory?: string;
}

interface Session {
  id: string;
  title?: string;
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
    title: result?.title,
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

// New hook for managing active session (can be overridden by forked sessions)
export const useActiveSession = (workingDirectory?: string) => {
  const [overrideSession, setOverrideSession] = useState<Session | null>(null);
  const defaultSessionQuery = useSession(workingDirectory);

  // Use override session if available, otherwise fall back to default
  const activeSession = overrideSession || defaultSessionQuery.data;
  const isLoading = !overrideSession && defaultSessionQuery.isLoading;
  const error = !overrideSession && defaultSessionQuery.error;

  const switchToSession = (session: Session) => {
    setOverrideSession(session);
  };

  const resetToDefault = () => {
    setOverrideSession(null);
  };

  return {
    data: activeSession,
    isLoading,
    error,
    switchToSession,
    resetToDefault,
    isOverride: !!overrideSession,
  };
};
