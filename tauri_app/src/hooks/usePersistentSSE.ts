import { useCallback, useEffect, useRef, useState } from 'react';

export type SSEToolCall = {
  name: string;
  description: string;
  status: 'pending' | 'running' | 'completed' | 'error';
  parameters: Record<string, unknown>;
  result?: string;
  error?: string;
  id: string;
};

export type PersistentSSEState = {
  connected: boolean;
  connecting: boolean;
  error: string | null;
  toolCalls: SSEToolCall[];
  finalContent: string | null;
  completed: boolean;
  processing: boolean;
  isPaused: boolean;
  cancelling: boolean;
  cancelled: boolean;
  reasoning: string | null;
  reasoningDuration: number | null;
  startTime?: number;
  rateLimit?: {
    retryAfter: number;
    attempt: number;
    maxAttempts: number;
  };
};

export type PersistentSSEHook = PersistentSSEState & {
  sendMessage: (content: string) => Promise<void>;
  cancelMessage: () => Promise<void>;
  resetCancelledState: () => void;
};

const BACKEND_URL = 'http://localhost:8088';

export function usePersistentSSE(sessionId: string): PersistentSSEHook {
  const [state, setState] = useState<PersistentSSEState>({
    connected: false,
    connecting: false,
    error: null,
    toolCalls: [],
    finalContent: null,
    completed: false,
    processing: false,
    isPaused: false,
    cancelling: false,
    cancelled: false,
    reasoning: null,
    reasoningDuration: null,
    rateLimit: undefined,
  });

  const eventSourceRef = useRef<EventSource | null>(null);
  const toolCallsMap = useRef<Map<string, SSEToolCall>>(new Map());
  const toolStartTimes = useRef<Map<string, number>>(new Map());
  const connectedRef = useRef<boolean>(false);
  const currentSessionRef = useRef<string>('');
  const eventListenersRef = useRef<
    Array<{ event: string; handler: (event: any) => void }>
  >([]);

  useEffect(() => {
    connectedRef.current = state.connected;
  }, [state.connected]);

  useEffect(() => {
    if (!sessionId || sessionId === currentSessionRef.current) {
      return;
    }

    if (eventSourceRef.current) {
      // Remove all event listeners before closing
      eventListenersRef.current.forEach(({ event, handler }) => {
        eventSourceRef.current?.removeEventListener(event, handler);
      });
      eventListenersRef.current = [];

      eventSourceRef.current.close();
      eventSourceRef.current = null;
    }
    toolCallsMap.current.clear();
    toolStartTimes.current.clear();
    currentSessionRef.current = sessionId;

    setState({
      connected: false,
      connecting: true,
      error: null,
      toolCalls: [],
      finalContent: null,
      completed: false,
      processing: false,
      isPaused: false,
      cancelling: false,
      cancelled: false,
      reasoning: null,
      reasoningDuration: null,
    });

    const eventSource = new EventSource(
      `${BACKEND_URL}/stream?sessionId=${encodeURIComponent(sessionId)}`
    );
    eventSourceRef.current = eventSource;

    // Helper function to add event listener and track it
    const addTrackedEventListener = (
      event: string,
      handler: (event: any) => void
    ) => {
      eventSource.addEventListener(event, handler);
      eventListenersRef.current.push({ event, handler });
    };

    addTrackedEventListener('connected', () => {
      setState((prev) => ({ ...prev, connected: true, connecting: false }));
    });

    addTrackedEventListener('heartbeat', (_event) => {
      // Heartbeat events keep connection alive - no UI state changes needed
    });

    addTrackedEventListener('tool', (event) => {
      try {
        const data = JSON.parse(event.data);
        const toolCall: SSEToolCall = {
          id: data.id || `${data.name}-${Date.now()}`,
          name: data.name || 'unknown',
          description: data.description || data.name || 'Tool execution',
          status: data.status || 'pending',
          parameters: data.input
            ? typeof data.input === 'string'
              ? (() => {
                  try {
                    return JSON.parse(data.input);
                  } catch {
                    return { input: data.input };
                  }
                })()
              : data.input
            : {},
          result: data.result,
          error: data.error,
        };

        if (
          data.status === 'running' &&
          !toolStartTimes.current.has(toolCall.id)
        ) {
          toolStartTimes.current.set(toolCall.id, Date.now());
        }

        if (
          (data.status === 'completed' || data.status === 'error') &&
          toolStartTimes.current.has(toolCall.id)
        ) {
          toolStartTimes.current.delete(toolCall.id);
        }

        toolCallsMap.current.set(toolCall.id, toolCall);

        setState((prev) => ({
          ...prev,
          toolCalls: Array.from(toolCallsMap.current.values()),
          processing: true,
        }));
      } catch (_err) {}
    });

    addTrackedEventListener('complete', (event) => {
      try {
        const data = JSON.parse(event.data);
        setState((prev) => ({
          ...prev,
          finalContent: data.content || '',
          reasoning: data.reasoning || null,
          reasoningDuration: data.reasoningDuration || null,
          completed: true,
          processing: false,
        }));
      } catch (_err) {
        setState((prev) => ({ ...prev, processing: false }));
      }
    });

    // Handle standard error events
    addTrackedEventListener('error', (event) => {
      if (event.data) {
        try {
          const data = JSON.parse(event.data);
          setState((prev) => ({
            ...prev,
            error: data.error || 'Stream error',
            connecting: false,
            processing: false,
            rateLimit: undefined,
          }));
        } catch {
          setState((prev) => ({
            ...prev,
            error: 'Stream error',
            connecting: false,
            processing: false,
            rateLimit: undefined,
          }));
        }
      }
    });
    
    // Handle rate limit error events
    addTrackedEventListener('rate_limit_error', (event) => {
      if (event.data) {
        try {
          console.log('Rate limit error received:', event.data);
          const data = JSON.parse(event.data);
          setState((prev) => ({
            ...prev,
            error: data.error || 'Rate limit exceeded',
            connecting: false,
            processing: true, // Keep processing true to show we're still working
            rateLimit: {
              retryAfter: data.retryAfter || 60,
              attempt: data.attempt || 1,
              maxAttempts: data.maxAttempts || 8
            }
          }));
        } catch (err) {
          console.error('Failed to parse rate limit error:', err);
          setState((prev) => ({
            ...prev,
            error: 'Rate limit exceeded',
            connecting: false,
            processing: true,
            rateLimit: {
              retryAfter: 60,
              attempt: 1,
              maxAttempts: 8
            }
          }));
        }
      }
    });

    eventSource.onerror = () => {
      const readyState = eventSource.readyState;
      if (
        readyState === EventSource.CLOSED ||
        readyState === EventSource.CONNECTING
      ) {
        setState((prev) => ({
          ...prev,
          connected: false,
          connecting: readyState === EventSource.CONNECTING,
          error: readyState === EventSource.CONNECTING ? null : prev.error,
        }));
      }
    };

    return () => {
      if (eventSourceRef.current) {
        // Remove all event listeners before closing
        eventListenersRef.current.forEach(({ event, handler }) => {
          eventSourceRef.current?.removeEventListener(event, handler);
        });
        eventListenersRef.current = [];

        eventSourceRef.current.close();
        eventSourceRef.current = null;
      }
      toolCallsMap.current.clear();
      toolStartTimes.current.clear();
      currentSessionRef.current = '';
    };
  }, [sessionId]);

  // Cleanup on component unmount
  useEffect(() => {
    return () => {
      if (eventSourceRef.current) {
        // Remove all event listeners before closing
        eventListenersRef.current.forEach(({ event, handler }) => {
          eventSourceRef.current?.removeEventListener(event, handler);
        });
        eventListenersRef.current = [];

        eventSourceRef.current.close();
        eventSourceRef.current = null;
      }
      toolCallsMap.current.clear();
      toolStartTimes.current.clear();
      currentSessionRef.current = '';
    };
  }, []);

  const sendMessage = useCallback(
    async (content: string) => {
      if (!(sessionId && connectedRef.current)) {
        throw new Error('No active SSE connection');
      }

      setState((prev) => ({
        ...prev,
        error: null,
        toolCalls: [],
        startTime: Date.now(),
        finalContent: null,
        completed: false,
        processing: true,
        cancelling: false,
        cancelled: false,
        reasoning: null,
        reasoningDuration: null,
        rateLimit: undefined,
      }));

      toolCallsMap.current.clear();

      try {
        const response = await fetch(
          `${BACKEND_URL}/stream/${encodeURIComponent(sessionId)}/message`,
          {
            method: 'POST',
            headers: { 'Content-Type': 'application/json' },
            body: JSON.stringify({ content }),
          }
        );

        if (!response.ok) {
          const errorText = await response.text();
          throw new Error(
            `Failed to queue message: ${response.status} ${errorText}`
          );
        }
      } catch (error) {
        setState((prev) => ({
          ...prev,
          error:
            error instanceof Error ? error.message : 'Failed to send message',
          processing: false,
          cancelling: false,
        }));
        throw error;
      }
    },
    [sessionId]
  );

  const cancelMessage = useCallback(async () => {
    if (!sessionId) {
      throw new Error('No session ID available');
    }

    setState((prev) => ({ ...prev, cancelling: true, error: null }));

    try {
      const response = await fetch(`${BACKEND_URL}/rpc`, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({
          method: 'agent.cancel',
          params: { sessionId },
          id: Date.now(),
        }),
      });

      if (!response.ok) {
        const errorText = await response.text();
        throw new Error(
          `Failed to cancel message: ${response.status} ${errorText}`
        );
      }

      const result = await response.json();
      if (result.error) {
        throw new Error(result.error.message || 'Cancel request failed');
      }

      setState((prev) => ({
        ...prev,
        processing: false,
        cancelling: false,
        cancelled: true,
        error: null,
      }));
    } catch (error) {
      setState((prev) => ({
        ...prev,
        cancelling: false,
        error:
          error instanceof Error ? error.message : 'Failed to cancel message',
      }));
      throw error;
    }
  }, [sessionId]);

  const resetCancelledState = useCallback(() => {
    setState((prev) => ({ ...prev, cancelled: false }));
  }, []);

  return {
    ...state,
    sendMessage,
    cancelMessage,
    resetCancelledState,
  };
}
