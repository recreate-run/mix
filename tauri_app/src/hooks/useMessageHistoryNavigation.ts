import { useState } from 'react';
import { useMessageHistory } from '@/hooks/useMessageHistory';
import {
  reconstructAttachmentsFromHistory,
  useAttachmentStore,
} from '@/stores/attachmentStore';

interface UseMessageHistoryNavigationProps {
  text: string;
  setText: (text: string) => void;
  batchSize?: number;
}

export function useMessageHistoryNavigation({
  text,
  setText,
  batchSize = 50,
}: UseMessageHistoryNavigationProps) {
  const [historyIndex, setHistoryIndex] = useState(-1);
  const [originalText, setOriginalText] = useState('');

  const messageHistory = useMessageHistory({
    batchSize,
  });

  const clearAttachments = useAttachmentStore(
    (state) => state.clearAttachments
  );
  const setHistoryState = useAttachmentStore((state) => state.setHistoryState);
  const syncWithText = useAttachmentStore((state) => state.syncWithText);

  const navigateHistory = async (direction: 'up' | 'down') => {
    const allHistoryTexts = messageHistory.getAllHistoryTexts();

    // Initialize history mode on first use
    if (historyIndex === -1 && direction === 'up') {
      setOriginalText(text);
      // Load initial history if not already loaded
      if (messageHistory.allHistory.length === 0) {
        messageHistory.loadInitialHistory();
      }
    }

    const newIndex = direction === 'up' ? historyIndex + 1 : historyIndex - 1;

    if (newIndex >= 0 && newIndex < allHistoryTexts.length) {
      setHistoryIndex(newIndex);

      // Get the full history item to access media and apps
      const historyItem = messageHistory.getHistoryItem(newIndex);
      if (historyItem) {
        try {
          // Reconstruct attachment state from historical message
          const { contractedText, attachments, referenceMap } =
            await reconstructAttachmentsFromHistory(
              historyItem.content,
              historyItem.media || [],
              historyItem.apps || []
            );

          // Atomically set attachment state from history
          setHistoryState(attachments, referenceMap);

          // Set the contracted text (with @filename references)
          setText(contractedText);
          syncWithText(contractedText);
        } catch (error) {
          console.warn(
            'Failed to reconstruct attachments from history:',
            error
          );
          // Fallback to plain text
          setText(historyItem.content);
        }
      } else {
        // Fallback to plain text if no history item
        setText(allHistoryTexts[newIndex]);
      }

      // Prefetch more history when getting close to the end
      if (
        newIndex > allHistoryTexts.length - 10 &&
        messageHistory.hasMoreHistory
      ) {
        messageHistory.loadMoreHistory();
      }
    } else if (newIndex === -1) {
      // Return to original text and clear attachments
      setHistoryIndex(-1);
      setText(originalText);
      setOriginalText('');
      clearAttachments();
    }
  };

  const exitHistoryMode = () => {
    if (historyIndex !== -1) {
      setHistoryIndex(-1);
      setText(originalText);
      setOriginalText('');
      clearAttachments();
    }
  };

  const resetHistoryMode = () => {
    if (historyIndex !== -1) {
      setHistoryIndex(-1);
      setOriginalText('');
    }
  };

  const handleHistoryNavigation = (
    e: React.KeyboardEvent<HTMLTextAreaElement>,
    isInOtherMode: boolean
  ): boolean => {
    if (isInOtherMode) return false;

    const textarea = e.currentTarget;
    const cursorAtStart = textarea.selectionStart === 0;
    const cursorAtEnd = textarea.selectionStart === textarea.value.length;
    const inHistoryMode = historyIndex !== -1;

    if (e.key === 'ArrowUp' && (cursorAtStart || inHistoryMode)) {
      e.preventDefault();
      navigateHistory('up').catch((error) => {
        console.error('Error navigating history:', error);
      });
      return true;
    }
    if (e.key === 'ArrowDown' && inHistoryMode && cursorAtEnd) {
      e.preventDefault();
      navigateHistory('down').catch((error) => {
        console.error('Error navigating history:', error);
      });
      return true;
    }
    if (e.key === 'Escape' && inHistoryMode) {
      e.preventDefault();
      exitHistoryMode();
      return true;
    }

    return false;
  };

  return {
    historyIndex,
    inHistoryMode: historyIndex !== -1,
    handleHistoryNavigation,
    exitHistoryMode,
    resetHistoryMode,
    navigateHistory,
  };
}
