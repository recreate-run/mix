import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';
import { rpcCall } from '@/lib/rpc';
import { type MessageData } from '@/components/chat-app';

export const TITLE_TRUNCATE_LENGTH = 100;

export interface SessionData {
  id: string;
  title: string;
  messageCount: number;
  promptTokens: number;
  completionTokens: number;
  cost: number;
  createdAt: string; // RFC3339 date string from backend Go time.Time
  workingDirectory?: string;
  firstUserMessage?: string; // JSON string of MessageData
}

const loadSessionsList = async (): Promise<SessionData[]> => {
  const result = await rpcCall<SessionData[]>('sessions.list', {});
  return result || [];
};

export const useSessionsList = () => {
  return useQuery({
    queryKey: ['sessions', 'list'],
    queryFn: loadSessionsList,
    refetchOnWindowFocus: false,
  });
};

const selectSession = async (sessionId: string): Promise<void> => {
  await rpcCall('sessions.select', { id: sessionId });
};

export const useSelectSession = () => {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: selectSession,
    onSuccess: () => {
      // Invalidate session-related queries to refresh UI
      queryClient.invalidateQueries({ queryKey: ['sessions'] });
      queryClient.invalidateQueries({ queryKey: ['session'] });
    },
  });
};