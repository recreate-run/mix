import { useQuery } from '@tanstack/react-query';
import { convertBackendMessagesToUI } from '@/lib/messageUtils';
import { mix } from '@/lib/mix-sdk';
import type { BackendMessage, UIMessage } from '@/types/message';
import { CACHE_KEYS } from '@/lib/cache-keys';

const loadSessionMessages = async (
  sessionId: string
): Promise<BackendMessage[]> => {
  const response = await mix.messages.getSession({ id: sessionId });

  return response;
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
