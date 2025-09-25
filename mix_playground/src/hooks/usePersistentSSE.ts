import { useCallback, useEffect, useRef, useState } from 'react';
import { useQueryClient } from '@tanstack/react-query';
import { mix } from '@/lib/mix-sdk';
import type { TimelineEntry, UIMessage } from '@/types/message';
import type { ToolCall } from '@/types/common';
import type { Attachment } from '@/stores/attachmentSlice';
import { expandFileReferences, buildFullUrlFromPath } from '@/utils/attachmentUtils';
import { CACHE_KEYS } from '@/lib/cache-keys';
import type { SendMessageRequestBody } from "mix-typescript-sdk/models/operations/sendmessage";


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
  submitMessage: (params: {
    text: string;
    attachments?: Attachment[];
    referenceMap?: Map<string, string>;
    planMode?: boolean;
    onUserMessage?: (message: UIMessage) => void;
    onCancelledContentPersist?: (message: UIMessage) => void;
  }) => Promise<void>;

  buttonStatus: 'ready' | 'streaming' | 'paused' | 'error';
  isSubmitDisabled: boolean;

  cancelMessage: () => Promise<void>;
  resetCancelledState: () => void;
  grantPermission: (id: string) => Promise<void>;
  denyPermission: (id: string) => Promise<void>;
};

export function usePersistentSSE(sessionId: string): PersistentSSEHook {
  const queryClient = useQueryClient();
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

  const toolCallsMap = useRef<Map<string, ToolCall>>(new Map());
  const toolStartTimes = useRef<Map<string, number>>(new Map());
  const timelineRef = useRef<TimelineEntry[]>([]);
  const connectedRef = useRef<boolean>(false);
  const currentSessionRef = useRef<string>('');
  const streamAbortController = useRef<AbortController | null>(null);
  const lastEventIdRef = useRef<string | undefined>(undefined);

  useEffect(() => {
    connectedRef.current = state.connected;
  }, [state.connected]);

  // Stream processing function
  const processEventStream = useCallback(async (sessionId: string, abortController: AbortController) => {
    try {
      setState(prev => ({ ...prev, connecting: true, error: null }));

      const result = await mix.streaming.streamEvents({
        sessionId,
        lastEventID: lastEventIdRef.current,
      });

      setState(prev => ({ ...prev, connected: true, connecting: false }));

      for await (const event of result.result) {
        if (abortController.signal.aborted) {
          break;
        }

        // Store event ID for reconnection
        if (event.id) {
          lastEventIdRef.current = event.id;
        }

        // Handle different event types using SDK's discriminated union
        switch (event.event) {
          case 'connected':
            setState(prev => ({ ...prev, connected: true, connecting: false }));
            break;

          case 'heartbeat':
            // Heartbeat events keep connection alive - no UI state changes needed
            break;

          case 'thinking': {
            const thinkingContent = event.data.content || '';

            // Add to timeline
            const thinkingEntry: TimelineEntry = {
              type: 'thinking',
              timestamp: Date.now(),
              content: thinkingContent,
              id: `thinking-${Date.now()}-${Math.random().toString(36).substr(2, 9)}`
            };

            timelineRef.current = [...timelineRef.current, thinkingEntry];

            setState(prev => ({
              ...prev,
              reasoning: (prev.reasoning || '') + thinkingContent,
              timeline: [...timelineRef.current],
              processing: true,
            }));
            break;
          }

          case 'content': {
            const contentDelta = event.data.content || '';

            // Find the last entry in timeline
            const lastEntry = timelineRef.current[timelineRef.current.length - 1];

            // If the last entry is a content entry, append to it
            if (lastEntry && lastEntry.type === 'content') {
              const existingContent = lastEntry.content;
              timelineRef.current[timelineRef.current.length - 1] = {
                ...lastEntry,
                content: existingContent + contentDelta,
                timestamp: Date.now()
              };
            } else {
              // Create new content entry
              const contentEntry: TimelineEntry = {
                type: 'content',
                timestamp: Date.now(),
                content: contentDelta,
                id: `content-${Date.now()}-${Math.random().toString(36).substr(2, 9)}`
              };
              timelineRef.current = [...timelineRef.current, contentEntry];
            }

            setState(prev => ({
              ...prev,
              finalContent: (prev.finalContent || '') + contentDelta,
              timeline: [...timelineRef.current],
              processing: true,
            }));
            break;
          }

          case 'tool': {
            const toolCall: ToolCall = {
              id: event.data.id || `${event.data.name}-${Date.now()}`,
              name: event.data.name || 'unknown',
              description: event.data.name || 'Tool execution',
              status: (event.data.status as any) || 'pending',
              parameters: event.data.input
                ? typeof event.data.input === 'string'
                  ? (() => {
                    try {
                      return JSON.parse(event.data.input);
                    } catch {
                      return { input: event.data.input };
                    }
                  })()
                  : event.data.input
                : {},
              result: undefined,
              error: undefined,
            };

            if (event.data.status === 'running' && !toolStartTimes.current.has(toolCall.id)) {
              toolStartTimes.current.set(toolCall.id, Date.now());
            }

            if ((event.data.status === 'completed' || event.data.status === 'error') &&
              toolStartTimes.current.has(toolCall.id)) {
              toolStartTimes.current.delete(toolCall.id);
            }

            toolCallsMap.current.set(toolCall.id, toolCall);

            // Add to timeline when tool is first seen
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

            setState(prev => ({
              ...prev,
              toolCalls: Array.from(toolCallsMap.current.values()),
              timeline: [...timelineRef.current],
              processing: true,
            }));
            break;
          }

          case 'tool_execution_start': {
            const toolCallId = event.data.toolCallId;
            const progress = event.data.progress;

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

              setState(prev => ({
                ...prev,
                toolCalls: Array.from(toolCallsMap.current.values()),
                timeline: [...timelineRef.current],
                processing: true,
              }));
            }
            break;
          }

          case 'tool_execution_complete': {
            const toolCallId = event.data.toolCallId;
            const progress = event.data.progress;
            const success = event.data.success;

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

              if (toolStartTimes.current.has(toolCallId)) {
                toolStartTimes.current.delete(toolCallId);
              }

              timelineRef.current = timelineRef.current.map(entry =>
                entry.type === 'tool' && entry.content.id === toolCallId
                  ? { ...entry, content: updatedToolCall }
                  : entry
              );

              setState(prev => ({
                ...prev,
                toolCalls: Array.from(toolCallsMap.current.values()),
                timeline: [...timelineRef.current],
                processing: true,
              }));
            }
            break;
          }

          case 'complete':
            setState(prev => ({
              ...prev,
              reasoning: event.data.reasoning || null,
              reasoningDuration: event.data.reasoningDuration || null,
              completed: true,
              processing: false,
            }));
            break;

          case 'error':
            setState(prev => ({
              ...prev,
              error: event.data.error || 'Stream error',
              connecting: false,
              processing: false,
              rateLimit: event.data.retryAfter ? {
                retryAfter: event.data.retryAfter,
                attempt: event.data.attempt || 1,
                maxAttempts: event.data.maxAttempts || 8,
              } : undefined,
            }));
            break;

          case 'permission': {
            const permissionRequest: SSEPermissionRequest = {
              id: event.data.id,
              sessionId: event.data.sessionId,
              toolName: event.data.toolName,
              description: event.data.description,
              action: event.data.action,
              path: event.data.path || '',
              params: event.data.params || {},
            };

            setState(prev => ({
              ...prev,
              permissionRequests: [...prev.permissionRequests, permissionRequest],
            }));
            break;
          }

          default:
            // Handle any other event types
            break;
        }
      }
    } catch (error) {
      if (!abortController.signal.aborted) {
        console.error('Stream processing error:', error);
        setState(prev => ({
          ...prev,
          connected: false,
          connecting: false,
          error: error instanceof Error ? error.message : 'Stream connection failed',
        }));
      }
    }
  }, []);

  useEffect(() => {
    if (!sessionId || sessionId === currentSessionRef.current) {
      return;
    }

    // Clean up previous session
    if (streamAbortController.current) {
      streamAbortController.current.abort();
    }

    toolCallsMap.current.clear();
    toolStartTimes.current.clear();
    timelineRef.current = [];
    lastEventIdRef.current = undefined;
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

    // Create new abort controller for this session
    streamAbortController.current = new AbortController();

    // Start streaming with SDK

    processEventStream(sessionId, streamAbortController.current).catch((error) => {
      console.error('Stream processing failed:', error);
      setState(prev => ({
        ...prev,
        connected: false,
        connecting: false,
        error: error instanceof Error ? error.message : 'Stream connection failed',
      }));
    });

    return () => {
      if (streamAbortController.current) {
        streamAbortController.current.abort();
        streamAbortController.current = null;
      }
      toolCallsMap.current.clear();
      toolStartTimes.current.clear();
      timelineRef.current = [];
      currentSessionRef.current = '';
      lastEventIdRef.current = undefined;
    };
  }, [sessionId]);

  // Cleanup on component unmount
  useEffect(() => {
    return () => {
      if (streamAbortController.current) {
        streamAbortController.current.abort();
        streamAbortController.current = null;
      }
      toolCallsMap.current.clear();
      toolStartTimes.current.clear();
      timelineRef.current = [];
      currentSessionRef.current = '';
      lastEventIdRef.current = undefined;
    };
  }, []);

  const sendMessage = useCallback(
    async function (content: string) {
      if (!sessionId) {
        throw new Error('No session ID available');
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
        await mix.streaming.sendStreamingMessage({
          id: sessionId,
          requestBody: { content },
        });
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

  const cancelMessage = useCallback(async function () {
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

  const resetCancelledState = useCallback(function () {
    setState((prev) => ({ ...prev, cancelled: false }));
  }, []);

  const grantPermission = useCallback(async function (id: string) {
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

  const denyPermission = useCallback(async function (id: string) {
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

  // Clean submitMessage implementation - fixes race condition
  const submitMessage = useCallback(
    async (params: {
      text: string;
      attachments?: Attachment[];
      referenceMap?: Map<string, string>;
      planMode?: boolean;
      onUserMessage?: (message: UIMessage) => void;
      onCancelledContentPersist?: (message: UIMessage) => void;
    }) => {
      const { text, attachments = [], referenceMap = new Map(), planMode = false, onUserMessage, onCancelledContentPersist } = params;

      if (!(text && sessionId && state.connected)) {
        return;
      }

      // Persist cancelled streaming content before it gets cleared
      if (state.cancelled && (state.finalContent || state.timeline?.length || state.toolCalls?.length)) {
        const cancelledMessage: UIMessage = {
          content: state.finalContent || '',
          from: 'assistant',
          timeline: state.timeline?.length ? state.timeline : undefined,
          toolCalls: state.toolCalls?.length ? state.toolCalls : undefined,
          frontend_only: true,
        };

        onCancelledContentPersist?.(cancelledMessage);
        resetCancelledState();
      }

      try {
        // Build media URLs array
        const mediaUrls = attachments.filter((a) => a.path).map((a) => buildFullUrlFromPath(a.path!));

        // Expand file references
        const expandedText = expandFileReferences(text, referenceMap, mediaUrls);

        const messageData: SendMessageRequestBody = {
          text: expandedText,
          media: mediaUrls,
          apps: attachments.filter((a) => a.type === 'app').map((app) => app.name),
          planMode: planMode,
        };

        // Send to backend first
        await sendMessage(JSON.stringify(messageData));

        // Only add to UI after successful backend request
        const userMessage: UIMessage = {
          content: text,
          from: 'user',
          attachments: attachments.length > 0 ? attachments : undefined,
        };
        onUserMessage?.(userMessage);
      } catch (error) {
        console.error('Failed to send message:', error);
        throw error; // Re-throw so parent can handle
      }
    },
    [sessionId, state.cancelled, state.finalContent, state.timeline, state.toolCalls, state.connected, resetCancelledState, sendMessage]
  );

  // Simple button status computation
  const buttonStatus = state.cancelling
    ? 'paused'
    : state.cancelled
      ? 'streaming'
      : state.processing
        ? 'streaming'
        : state.error
          ? 'error'
          : 'ready';

  // Simple submit button disabled state
  const isSubmitDisabled = buttonStatus === 'paused' || !state.connected;

  // Stream completion handling - invalidate cache when streaming completes
  useEffect(() => {
    if (state.completed && (state.finalContent || state.toolCalls.length > 0) && !state.processing) {
      queryClient.invalidateQueries({ queryKey: CACHE_KEYS.sessionMessages(sessionId) });
    }
  }, [state.completed, state.finalContent, state.processing, state.toolCalls.length, sessionId, queryClient]);

  return {
    ...state,
    submitMessage,
    buttonStatus,
    isSubmitDisabled,
    cancelMessage,
    resetCancelledState,
    grantPermission,
    denyPermission,
  };
}
