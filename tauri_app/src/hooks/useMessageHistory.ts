import { useInfiniteQuery } from '@tanstack/react-query';
import { useCallback } from 'react';
import { rpcCall } from '@/lib/rpc';

interface MessageHistoryItem {
  id: string;
  role: string;
  content: string;
  sessionId: string;
  media: string[];
  apps: string[];
}

interface UseMessageHistoryOptions {
  batchSize?: number;
}

interface UseMessageHistoryReturn {
  allHistory: MessageHistoryItem[];
  isLoading: boolean;
  error: string | null;
  loadInitialHistory: () => Promise<void>;
  loadMoreHistory: () => Promise<void>;
  getAllHistoryTexts: () => string[];
  getHistoryItem: (index: number) => MessageHistoryItem | null;
  hasMoreHistory: boolean;
}

const extractMessageData = (content: string) => {
  try {
    const parsed = JSON.parse(content);
    return {
      text: parsed.text || content,
      media: parsed.media || [],
      apps: parsed.apps || [],
    };
  } catch {
    return {
      text: content,
      media: [],
      apps: [],
    };
  }
};

const fetchMessages = async (params: any): Promise<MessageHistoryItem[]> => {
  const result = await rpcCall<any[]>('messages.history', params);

  return (result || []).map((msg: any) => {
    const messageData = extractMessageData(msg.content);
    return {
      id: msg.id,
      role: msg.role,
      content: messageData.text,
      sessionId: msg.sessionId,
      media: messageData.media,
      apps: messageData.apps,
    };
  });
};

export function useMessageHistory({
  batchSize = 50,
}: UseMessageHistoryOptions): UseMessageHistoryReturn {
  const historyQuery = useInfiniteQuery({
    queryKey: ['messageHistory'],
    queryFn: ({ pageParam = 0 }) => {
      return fetchMessages({
        limit: batchSize,
        offset: pageParam,
      });
    },
    getNextPageParam: (lastPage, pages) => {
      const totalLoaded = pages.flat().length;
      return lastPage.length === batchSize ? totalLoaded : undefined;
    },
    initialPageParam: 0,
  });

  const allHistory = historyQuery.data?.pages.flat() || [];
  const isLoading = historyQuery.isLoading;
  const error = historyQuery.error?.message || null;
  const hasMoreHistory = historyQuery.hasNextPage;

  const loadInitialHistory = useCallback(async () => {
    await historyQuery.refetch();
  }, [historyQuery.refetch]);

  const loadMoreHistory = useCallback(async () => {
    if (!historyQuery.hasNextPage) return;
    await historyQuery.fetchNextPage();
  }, [historyQuery]);

  const getAllHistoryTexts = useCallback(() => {
    return allHistory.map((msg) => msg.content);
  }, [allHistory]);

  const getHistoryItem = useCallback(
    (index: number): MessageHistoryItem | null => {
      return allHistory[index] || null;
    },
    [allHistory]
  );

  return {
    allHistory,
    isLoading,
    error,
    loadInitialHistory,
    loadMoreHistory,
    getAllHistoryTexts,
    getHistoryItem,
    hasMoreHistory,
  };
}
