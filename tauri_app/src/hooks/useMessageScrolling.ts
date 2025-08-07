import { useEffect, useRef } from 'react';

interface Message {
  from: 'user' | 'assistant';
  content: string;
}

export function useMessageScrolling(
  messages: Message[],
  isProcessing: boolean
) {
  const conversationRef = useRef<HTMLDivElement>(null);
  const userMessageRefs = useRef<(HTMLDivElement | null)[]>([]);

  // Auto-scroll to last user message when messages change
  useEffect(() => {
    // Find the index of the last user message
    const lastUserMessageIndex = messages.findLastIndex(
      (m) => m.from === 'user'
    );

    if (
      lastUserMessageIndex !== -1 &&
      userMessageRefs.current[lastUserMessageIndex]
    ) {
      // Use setTimeout to ensure DOM updates are complete
      setTimeout(() => {
        userMessageRefs.current[lastUserMessageIndex]?.scrollIntoView({
          behavior: 'smooth',
          block: 'start',
        });
      }, 0);
    }
  }, [messages, isProcessing]);

  const setUserMessageRef = (index: number) => (el: HTMLDivElement | null) => {
    userMessageRefs.current[index] = el;
  };

  return {
    conversationRef,
    setUserMessageRef,
  };
}
