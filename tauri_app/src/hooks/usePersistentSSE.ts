import { useCallback, useEffect, useRef, useState } from 'react';
import type { TimelineEntry } from '@/types/message';
import type { ToolCall } from '@/types/common';

export type SSEPermissionRequest = {
  id: string;
  sessionId: string;
  toolName: string;
  description: string;
  action: string;
  path: string;
  params: Record<string, unknown>;
};

export type PersistentSSEState = {
  connected: boolean;
  connecting: boolean;
  error: string | null;
  toolCalls: ToolCall[];
  finalContent: string | null;
  completed: boolean;
  processing: boolean;
  isPaused: boolean;
  cancelling: boolean;
  cancelled: boolean;
  reasoning: string | null;
  reasoningDuration: number | null;
  timeline: TimelineEntry[];
  startTime?: number;
  rateLimit?: {
    retryAfter: number;
    attempt: number;
    maxAttempts: number;
  };
  permissionRequests: SSEPermissionRequest[];
};

export type PersistentSSEHook = PersistentSSEState & {
  sendMessage: (content: string) => Promise<void>;
  cancelMessage: () => Promise<void>;
  resetCancelledState: () => void;
  grantPermission: (id: string) => Promise<void>;
  denyPermission: (id: string) => Promise<void>;
};


import { getBackendUrl } from '@/utils/backendUrl';
import { mix } from '@/lib/mix-sdk';

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
    timeline: [],
    rateLimit: undefined,
    permissionRequests: [],
  });

  const eventSourceRef = useRef<EventSource | null>(null);
  const toolCallsMap = useRef<Map<string, ToolCall>>(new Map());
  const toolStartTimes = useRef<Map<string, number>>(new Map());
  const timelineRef = useRef<TimelineEntry[]>([]);
  const connectedRef = useRef<boolean>(false);
  const currentSessionRef = useRef<string>('');
  const eventListenersRef = useRef<
    Array<{ event: string; handler: (event: MessageEvent) => void }>
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
    timelineRef.current = [];
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
      timeline: [],
      permissionRequests: [],
    });

    const eventSource = new EventSource(
      `${getBackendUrl()}/stream?sessionId=${encodeURIComponent(sessionId)}`
    );
    eventSourceRef.current = eventSource;

    // Helper function to add event listener and track it
    const addTrackedEventListener = (
      event: string,
      handler: (event: MessageEvent) => void
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

    addTrackedEventListener('thinking', (event) => {
      try {
        const data = JSON.parse(event.data);
        const thinkingContent = data.content || '';

        // Add to timeline
        const thinkingEntry: TimelineEntry = {
          type: 'thinking',
          timestamp: Date.now(),
          content: thinkingContent,
          id: `thinking-${Date.now()}-${Math.random().toString(36).substr(2, 9)}`
        };

        timelineRef.current = [...timelineRef.current, thinkingEntry];

        setState((prev) => ({
          ...prev,
          reasoning: (prev.reasoning || '') + thinkingContent,
          timeline: [...timelineRef.current],
          processing: true,
        }));
      } catch (err) {
        console.error('Failed to parse thinking event:', err, 'Raw event data:', event.data);
      }
    });

    addTrackedEventListener('content', (event) => {
      try {
        const data = JSON.parse(event.data);
        const contentDelta = data.content || '';

        // Find the last entry in timeline
        const lastEntry = timelineRef.current[timelineRef.current.length - 1];
        
        // If the last entry is a content entry, append to it
        // If it's a tool or thinking entry, create a new content entry
        if (lastEntry && lastEntry.type === 'content') {
          // Append delta to existing content entry
          const existingContent = lastEntry.content;
          timelineRef.current[timelineRef.current.length - 1] = {
            ...lastEntry,
            content: existingContent + contentDelta,
            timestamp: Date.now()
          };
        } else {
          // Create new content entry (after tool call or thinking)
          const contentEntry: TimelineEntry = {
            type: 'content',
            timestamp: Date.now(),
            content: contentDelta,
            id: `content-${Date.now()}-${Math.random().toString(36).substr(2, 9)}`
          };
          timelineRef.current = [...timelineRef.current, contentEntry];
        }

        // Also update finalContent by accumulating deltas
        setState((prev) => ({
          ...prev,
          finalContent: (prev.finalContent || '') + contentDelta,
          timeline: [...timelineRef.current],
          processing: true,
        }));
      } catch (err) {
        console.error('Failed to parse content event:', err, 'Raw event data:', event.data);
      }
    });

    addTrackedEventListener('tool', (event) => {
      try {
        const data = JSON.parse(event.data);
        const toolCall: ToolCall = {
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

        // Add to timeline when tool is first seen (running or pending status)
        if (!timelineRef.current.some(entry => entry.type === 'tool' && entry.content.id === toolCall.id)) {
          const toolEntry: TimelineEntry = {
            type: 'tool',
            timestamp: Date.now(),
            content: toolCall,
            id: toolCall.id
          };

          timelineRef.current = [...timelineRef.current, toolEntry];
        } else {
          // Update existing tool entry
          timelineRef.current = timelineRef.current.map(entry =>
            entry.type === 'tool' && entry.content.id === toolCall.id
              ? { ...entry, content: toolCall }
              : entry
          );
        }

        setState((prev) => ({
          ...prev,
          toolCalls: Array.from(toolCallsMap.current.values()),
          timeline: [...timelineRef.current],
          processing: true,
        }));
      } catch (err) {
        console.error('Failed to parse tool event:', err, 'Raw event data:', event.data);
        // Don't silently ignore - this could indicate backend issues
      }
    });

    // Handle tool execution start events
    addTrackedEventListener('tool_execution_start', (event) => {
      try {
        const data = JSON.parse(event.data);
        const toolCallId = data.toolCallId;
        const progress = data.progress;

        // Find the corresponding tool call by ID and update its status to running
        const existingToolCall = toolCallsMap.current.get(toolCallId);

        if (existingToolCall) {
          const updatedToolCall = {
            ...existingToolCall,
            status: 'running' as const,
            description: progress
          };

          toolCallsMap.current.set(toolCallId, updatedToolCall);
          toolStartTimes.current.set(toolCallId, Date.now());

          // Update timeline entry
          timelineRef.current = timelineRef.current.map(entry =>
            entry.type === 'tool' && entry.content.id === toolCallId
              ? { ...entry, content: updatedToolCall }
              : entry
          );

          setState((prev) => ({
            ...prev,
            toolCalls: Array.from(toolCallsMap.current.values()),
            timeline: [...timelineRef.current],
            processing: true,
          }));
        }
      } catch (err) {
        console.error('Failed to parse tool_execution_start event:', err, 'Raw event data:', event.data);
        // Don't silently ignore - this could indicate backend issues
      }
    });

    // Handle tool execution complete events
    addTrackedEventListener('tool_execution_complete', (event) => {
      try {
        const data = JSON.parse(event.data);
        const toolCallId = data.toolCallId;
        const progress = data.progress;
        const success = data.success;

        // Find the corresponding tool call by ID and update its status
        const existingToolCall = toolCallsMap.current.get(toolCallId);

        if (existingToolCall) {
          const updatedToolCall = {
            ...existingToolCall,
            status: success ? 'completed' as const : 'error' as const,
            description: progress,
            result: success ? progress : undefined,
            error: success ? undefined : progress
          };

          toolCallsMap.current.set(toolCallId, updatedToolCall);

          // Remove from start times tracking
          if (toolStartTimes.current.has(toolCallId)) {
            toolStartTimes.current.delete(toolCallId);
          }

          // Update timeline entry
          timelineRef.current = timelineRef.current.map(entry =>
            entry.type === 'tool' && entry.content.id === toolCallId
              ? { ...entry, content: updatedToolCall }
              : entry
          );

          setState((prev) => ({
            ...prev,
            toolCalls: Array.from(toolCallsMap.current.values()),
            timeline: [...timelineRef.current],
            processing: true,
          }));
        }
      } catch (err) {
        console.error('Failed to parse tool_execution_complete event:', err, 'Raw event data:', event.data);
        // Don't silently ignore - this could indicate backend issues
      }
    });

    addTrackedEventListener('complete', (event) => {
      try {
        const data = JSON.parse(event.data);
        setState((prev) => ({
          ...prev,
          // Don't update finalContent - it was already built from content deltas during streaming
          // finalContent: data.content || '', // ← REMOVED - this was causing duplication
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
          const data = JSON.parse(event.data);
          setState((prev) => ({
            ...prev,
            error: data.error || 'Rate limit exceeded',
            connecting: false,
            processing: true, // Keep processing true to show we're still working
            rateLimit: {
              retryAfter: data.retryAfter || 60,
              attempt: data.attempt || 1,
              maxAttempts: data.maxAttempts || 8,
            },
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
              maxAttempts: 8,
            },
          }));
        }
      }
    });

    // Handle permission request events
    addTrackedEventListener('permission', (event) => {
      if (event.data) {
        try {
          const data = JSON.parse(event.data);
          const permissionRequest: SSEPermissionRequest = {
            id: data.id,
            sessionId: data.sessionId,
            toolName: data.toolName,
            description: data.description,
            action: data.action,
            path: data.path,
            params: data.params || {},
          };

          setState((prev) => ({
            ...prev,
            permissionRequests: [...prev.permissionRequests, permissionRequest],
          }));
        } catch (err) {
          console.error('Failed to parse permission event:', err);
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
      timelineRef.current = [];
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
      timelineRef.current = [];
      currentSessionRef.current = '';
    };
  }, []);

  const sendMessage = useCallback(
    async function(content: string) {
      if (!(sessionId && connectedRef.current)) {
        throw new Error('No active SSE connection');
      }

      setState((prev) => ({
        ...prev,
        error: null,
        toolCalls: [],
        startTime: Date.now(),
        finalContent: '', // Reset to empty string for delta accumulation
        completed: false,
        processing: true,
        cancelling: false,
        cancelled: false,
        reasoning: null,
        reasoningDuration: null,
        timeline: [],
        rateLimit: undefined,
      }));

      toolCallsMap.current.clear();
      timelineRef.current = [];

      try {
        const response = await fetch(
          `${getBackendUrl()}/stream/${encodeURIComponent(sessionId)}/message`,
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

  const cancelMessage = useCallback(async function() {
    if (!sessionId) {
      throw new Error('No session ID available');
    }

    setState((prev) => ({ ...prev, cancelling: true, error: null }));

    try {
      await mix.messages.cancelProcessing({ id: sessionId });


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

  const resetCancelledState = useCallback(function() {
    setState((prev) => ({ ...prev, cancelled: false }));
  }, []);

  const grantPermission = useCallback(async function(id: string) {
    try {
      await mix.permissions.grant({ id });


      // Remove the permission request from state
      setState((prev) => ({
        ...prev,
        permissionRequests: prev.permissionRequests.filter(
          (req) => req.id !== id
        ),
      }));
    } catch (error) {
      console.error('Failed to grant permission:', error);
      throw error;
    }
  }, []);

  const denyPermission = useCallback(async function(id: string) {
    try {
      await mix.permissions.deny({ id });


      // Remove the permission request from state
      setState((prev) => ({
        ...prev,
        permissionRequests: prev.permissionRequests.filter(
          (req) => req.id !== id
        ),
      }));
    } catch (error) {
      console.error('Failed to deny permission:', error);
      throw error;
    }
  }, []);

  return {
    ...state,
    sendMessage,
    cancelMessage,
    resetCancelledState,
    grantPermission,
    denyPermission,
  };
}
