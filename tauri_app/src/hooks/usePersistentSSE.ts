import { useState, useEffect, useRef, useCallback } from 'react';
import { safeTrackEvent } from '@/lib/posthog';

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
  startTime?: number; // When message processing started
  isPaused: boolean; // True when session is paused
  reasoning: string | null; // Reasoning content from the assistant
  reasoningDuration: number | null; // Reasoning duration in seconds
};

export type PersistentSSEHook = PersistentSSEState & {
  sendMessage: (content: string) => Promise<void>;
  pauseMessage: () => Promise<void>;
  resumeMessage: () => Promise<void>;
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
    reasoning: null,
    reasoningDuration: null,
  });
  
  const eventSourceRef = useRef<EventSource | null>(null);
  const toolCallsRef = useRef<Map<string, SSEToolCall>>(new Map());
  const currentSessionRef = useRef<string>('');
  const toolStartTimeRef = useRef<Map<string, number>>(new Map());
  const connectedRef = useRef<boolean>(false);

  // Establish persistent connection when sessionId changes
  useEffect(() => {
    if (!sessionId || sessionId === currentSessionRef.current) return;
    
    // Clean up previous connection
    if (eventSourceRef.current) {
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
      reasoning: null,
      reasoningDuration: null,
    });
    
    toolCallsRef.current.clear();
    currentSessionRef.current = sessionId;

    const url = `http://localhost:8088/stream?sessionId=${encodeURIComponent(sessionId)}`;
    
    const eventSource = new EventSource(url);
    eventSourceRef.current = eventSource;

    eventSource.addEventListener('connected', (event) => {
      setState(prev => ({ ...prev, connected: true, connecting: false }));
    });

    eventSource.addEventListener('heartbeat', (event) => {
      // Heartbeat events keep connection alive - no UI state changes needed
    });

    eventSource.addEventListener('tool', (event) => {
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
        
        // If tool is starting, record start time
        if (data.status === 'running' && !toolStartTimeRef.current.has(toolCall.id)) {
          toolStartTimeRef.current.set(toolCall.id, Date.now());
        }
        
        // If tool is completing, track execution time
        if ((data.status === 'completed' || data.status === 'error') && toolStartTimeRef.current.has(toolCall.id)) {
          const startTime = toolStartTimeRef.current.get(toolCall.id);
          const executionTime = Date.now() - startTime!;
          
          safeTrackEvent('tool_execution_time', {
            tool_name: toolCall.name,
            tool_id: toolCall.id,
            execution_time_ms: executionTime,
            status: data.status,
            has_error: !!data.error,
            session_id: sessionId,
            timestamp: new Date().toISOString()
          });
          
          toolStartTimeRef.current.delete(toolCall.id);
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

    eventSource.addEventListener('complete', (event) => {
      try {
        const data = JSON.parse(event.data);
        const processingTime = state.startTime ? Date.now() - state.startTime : 0;
        
        setState(prev => ({
          ...prev,
          finalContent: data.content || '',
          reasoning: data.reasoning || null,
          reasoningDuration: data.reasoningDuration || null,
          completed: true,
          processing: false, // Message processing complete
        }));
        
        // Track response latency
        if (state.startTime) {
          safeTrackEvent('response_latency', {
            response_time_ms: processingTime,
            session_id: sessionId,
            tool_count: toolCallsRef.current.size,
            response_length: data.content?.length || 0,
            timestamp: new Date().toISOString()
          });
        }
      } catch (err) {
        console.error('Failed to parse complete event:', err, event.data);
        setState(prev => ({ ...prev, processing: false }));
      }
    });

    eventSource.addEventListener('error', (event) => {
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
        eventSourceRef.current.close();
        eventSourceRef.current = null;
      }
      currentSessionRef.current = '';
    };
  }, [sessionId]);

  // Cleanup on component unmount
  useEffect(() => {
    return () => {
      if (eventSourceRef.current) {
        eventSourceRef.current.close();
        eventSourceRef.current = null;
      }
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
      startTime: Date.now(),
      finalContent: null,
      completed: false,
      processing: true, // Mark as processing when sending message
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
      }));
      throw error;
    }
  }, [sessionId]);

  // Function to pause message processing
  const pauseMessage = useCallback(async () => {
    console.log('pausing not implemented');
  }, [sessionId]);

  // Function to resume message processing
  const resumeMessage = useCallback(async () => {
    console.log('pausing not implemented');
  }, [sessionId]);

  return {
    ...state,
    sendMessage,
    pauseMessage,
    resumeMessage,
  };
}