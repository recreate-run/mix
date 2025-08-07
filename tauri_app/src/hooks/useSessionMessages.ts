import { useQuery } from '@tanstack/react-query';
import {
  type BackendMessage,
  convertBackendMessagesToUI,
  type UIMessage,
} from '@/lib/messageUtils';
import { rpcCall } from '@/lib/rpc';

const loadSessionMessages = async (
  sessionId: string
): Promise<BackendMessage[]> => {
  const result = await rpcCall<BackendMessage[]>('messages.list', {
    sessionId,
  });
  return result || [];
};

const loadAndConvertMessages = async (
  sessionId: string
): Promise<UIMessage[]> => {
  try {
    const backendMessages = await loadSessionMessages(sessionId);
    return await convertBackendMessagesToUI(backendMessages);
  } catch (error) {
    console.error('Failed to convert backend messages:', error);
    return []; // Graceful fallback - empty array means just default message
  }
};

export const useSessionMessages = (sessionId: string | null) => {
  return useQuery({
    queryKey: ['uiMessages', sessionId],
    queryFn: () => (sessionId ? loadAndConvertMessages(sessionId) : []),
    enabled: !!sessionId,
    staleTime: 0, // Always refetch when session changes
    refetchOnWindowFocus: false,
  });
};
