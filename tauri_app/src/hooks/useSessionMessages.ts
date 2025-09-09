import { useQuery } from '@tanstack/react-query';
import { convertBackendMessagesToUI } from '@/lib/messageUtils';
import { mix } from '@/lib/mix-sdk';
import type { BackendMessage, UIMessage } from '@/types/message';
import { CACHE_KEYS } from '@/lib/cache-keys';

const loadSessionMessages = async (
  sessionId: string
): Promise<BackendMessage[]> => {
  const response = await mix.messages.getSession({ id: sessionId });

  if (response.error) {
    console.error('❌ SDK Error:', response.error);
    throw new Error(response.error.message || 'Failed to load session messages');
  }

  if (!response.data) {
    throw new Error('No message data returned from server');
  }

  return response.data;
};

const loadAndConvertMessages = async (
  sessionId: string
): Promise<UIMessage[]> => {
  const backendMessages = await loadSessionMessages(sessionId);
  const uiMessages = await convertBackendMessagesToUI(backendMessages);
  return uiMessages;
};

export const useSessionMessages = (sessionId: string | null) => {
  return useQuery({
    queryKey: CACHE_KEYS.sessionMessages(sessionId || ''),
    queryFn: () => sessionId ? loadAndConvertMessages(sessionId) : [],
    enabled: !!sessionId,
    refetchOnWindowFocus: false,
  });
};
