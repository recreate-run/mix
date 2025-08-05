import { useState, useEffect, useRef, useCallback } from 'react';

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
  processing: boolean; // True when processing a message
  isPaused: boolean; // True when session is paused
  cancelling: boolean; // True when cancellation is in progress
  cancelled: boolean; // True after cancellation completes, until user starts typing
  reasoning: string | null; // Reasoning content from the assistant
  reasoningDuration: number | null; // Reasoning duration in seconds
};

export type PersistentSSEHook = PersistentSSEState & {
  sendMessage: (content: string) => Promise<void>;
  cancelMessage: () => Promise<void>;
  resetCancelledState: () => void;
};

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
  });
  
  const eventSourceRef = useRef<EventSource | null>(null);
  const toolCallsRef = useRef<Map<string, SSEToolCall>>(new Map());
  const currentSessionRef = useRef<string>('');
  const connectedRef = useRef<boolean>(false);
  const eventListenersRef = useRef<Array<{ event: string; handler: (event: any) => void }>>([]);
  
  // Maximum number of tool calls to keep in memory
  const MAX_TOOL_CALLS = 1000;

  // Establish persistent connection when sessionId changes
  useEffect(() => {
    if (!sessionId || sessionId === currentSessionRef.current) return;
    
    // Clean up previous connection
    if (eventSourceRef.current) {
      // Remove all event listeners before closing
      eventListenersRef.current.forEach(({ event, handler }) => {
        eventSourceRef.current?.removeEventListener(event, handler);
      });
      eventListenersRef.current = [];
      
      eventSourceRef.current.close();
      eventSourceRef.current = null;
    }

    // Reset state for new session
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
    
    toolCallsRef.current.clear();
    currentSessionRef.current = sessionId;

    const url = `http://localhost:8088/stream?sessionId=${encodeURIComponent(sessionId)}`;
    
    const eventSource = new EventSource(url);
    eventSourceRef.current = eventSource;

    // Helper function to add event listener and track it
    const addTrackedEventListener = (event: string, handler: (event: any) => void) => {
      eventSource.addEventListener(event, handler);
      eventListenersRef.current.push({ event, handler });
    };

    addTrackedEventListener('connected', (event) => {
      setState(prev => ({ ...prev, connected: true, connecting: false }));
    });

    addTrackedEventListener('heartbeat', (event) => {
      // Heartbeat events keep connection alive - no UI state changes needed
    });

    addTrackedEventListener('tool', (event) => {
      try {
        const data = JSON.parse(event.data);
        
        // Parse tool input if it's a JSON string
        let parameters = {};
        if (data.input) {
          try {
            parameters = JSON.parse(data.input);
          } catch {
            parameters = { input: data.input };
          }
        }
        
        const toolCall: SSEToolCall = {
          id: data.id || `${data.name}-${Date.now()}`,
          name: data.name || 'unknown',
          description: data.description || data.name || 'Tool execution',
          status: data.status || 'pending',
          parameters,
          result: data.result,
          error: data.error,
        };

        // Implement LRU eviction if map gets too large
        if (toolCallsRef.current.size >= MAX_TOOL_CALLS) {
          const firstKey = toolCallsRef.current.keys().next().value;
          if (firstKey) {
            toolCallsRef.current.delete(firstKey);
          }
        }

        toolCallsRef.current.set(toolCall.id, toolCall);
        
        setState(prev => ({
          ...prev,
          toolCalls: Array.from(toolCallsRef.current.values()),
          processing: true, // Mark as processing when tools are running
        }));
      } catch (err) {
        console.error('Failed to parse tool event:', err, event.data);
      }
    });

    addTrackedEventListener('complete', (event) => {
      try {
        const data = JSON.parse(event.data);
        setState(prev => ({
          ...prev,
          finalContent: data.content || '',
          reasoning: data.reasoning || null,
          reasoningDuration: data.reasoningDuration || null,
          completed: true,
          processing: false, // Message processing complete
        }));
      } catch (err) {
        console.error('Failed to parse complete event:', err, event.data);
        setState(prev => ({ ...prev, processing: false }));
      }
    });

    addTrackedEventListener('error', (event) => {
      // Backend-sent error events have JSON data
      if (event.data) {
        try {
          const data = JSON.parse(event.data);
          const errorMsg = data.error || 'Stream error';
          setState(prev => ({ 
            ...prev, 
            error: errorMsg, 
            connecting: false,
            processing: false 
          }));
        } catch (err) {
          console.error('Failed to parse backend error event:', err, event.data);
          setState(prev => ({ 
            ...prev, 
            error: 'Stream error', 
            connecting: false,
            processing: false 
          }));
        }
      }
    });

    eventSource.onerror = (event) => {
      // For persistent connections, we want to be more resilient to temporary drops
      if (eventSource.readyState === EventSource.CLOSED) {
        setState(prev => ({ 
          ...prev, 
          connected: false,
          connecting: true // Try to reconnect
        }));
      } else if (eventSource.readyState === EventSource.CONNECTING) {
        setState(prev => ({ 
          ...prev, 
          connected: false,
          connecting: true,
          error: null // Clear any previous errors
        }));
      }
    };

    // Cleanup function
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
      toolCallsRef.current.clear();
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
      toolCallsRef.current.clear();
      currentSessionRef.current = '';
    };
  }, []);

  // Update connectedRef when state.connected changes
  useEffect(() => {
    connectedRef.current = state.connected;
  }, [state.connected]);

  // Function to send messages via POST to message queue
  const sendMessage = useCallback(async (content: string) => {
    if (!sessionId || !connectedRef.current) {
      throw new Error('No active SSE connection');
    }


    // Reset state for new message
    setState(prev => ({
      ...prev,
      error: null,
      toolCalls: [],
      finalContent: null,
      completed: false,
      processing: true, // Mark as processing when sending message
      cancelling: false,
      cancelled: false,
      reasoning: null,
      reasoningDuration: null,
    }));
    
    toolCallsRef.current.clear();

    try {
      const response = await fetch(`http://localhost:8088/stream/${encodeURIComponent(sessionId)}/message`, {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
        },
        body: JSON.stringify({ content }),
      });

      if (!response.ok) {
        const errorText = await response.text();
        throw new Error(`Failed to queue message: ${response.status} ${errorText}`);
      }

      const result = await response.json();
    } catch (error) {
      console.error('Failed to send message:', error);
      setState(prev => ({
        ...prev,
        error: error instanceof Error ? error.message : 'Failed to send message',
        processing: false,
        cancelling: false,
      }));
      throw error;
    }
  }, [sessionId]);

  // Function to cancel message processing
  const cancelMessage = useCallback(async () => {
    if (!sessionId) {
      throw new Error('No session ID available');
    }

    // Set cancelling state to show disabled button
    setState(prev => ({
      ...prev,
      cancelling: true,
      error: null,
    }));

    try {
      const response = await fetch(`http://localhost:8088/rpc`, {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
        },
        body: JSON.stringify({
          method: 'agent.cancel',
          params: { sessionId },
          id: Date.now(),
        }),
      });

      if (!response.ok) {
        const errorText = await response.text();
        throw new Error(`Failed to cancel message: ${response.status} ${errorText}`);
      }

      const result = await response.json();
      
      if (result.error) {
        throw new Error(result.error.message || 'Cancel request failed');
      }

      // Update state to reflect successful cancellation
      setState(prev => ({
        ...prev,
        processing: false,
        cancelling: false,
        cancelled: true, // Mark as cancelled so button shows stop state
        error: null,
      }));

    } catch (error) {
      console.error('Failed to cancel message:', error);
      setState(prev => ({
        ...prev,
        cancelling: false, // Reset cancelling on error
        error: error instanceof Error ? error.message : 'Failed to cancel message',
      }));
      throw error;
    }
  }, [sessionId]);

  // Function to reset cancelled state when user starts typing
  const resetCancelledState = useCallback(() => {
    setState(prev => ({
      ...prev,
      cancelled: false,
    }));
  }, []);

  return {
    ...state,
    sendMessage,
    cancelMessage,
    resetCancelledState,
  };
}