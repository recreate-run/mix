import { useInfiniteQuery } from '@tanstack/react-query';
import { useCallback } from 'react';
import { mix } from '@/lib/mix-sdk';
import type { BackendMessage } from 'mix-typescript-sdk/models';

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

async function fetchMessages(params: {
  limit: number;
  offset: number;
}): Promise<MessageHistoryItem[]> {
  // Let SDK validation errors propagate - don't mask them with fallbacks
  const response = await mix.messages.getHistory(params);

  // Map the BackendMessage structure to our MessageHistoryItem format
  return response.map((msg: BackendMessage) => {
    return {
      id: msg.id,
      role: msg.role,
      content: msg.userInput || msg.assistantResponse || '',
      sessionId: msg.sessionId,
      media: [], // BackendMessage doesn't have media field
      apps: [], // BackendMessage doesn't have apps field
    };
  });
}

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
