import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';
import { rpcCall } from '@/lib/rpc';

interface CreateSessionParams {
  title: string;
  workingDirectory?: string;
}

interface Session {
  id: string;
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
    workingDirectory: result?.workingDirectory
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

export const useSession = (workingDirectory?: string) => {
  return useQuery({
    queryKey: ['session', workingDirectory],
    queryFn: () => createSession({ title: "Chat Session", workingDirectory }),
    staleTime: Infinity,
    refetchOnWindowFocus: false,
  });
};