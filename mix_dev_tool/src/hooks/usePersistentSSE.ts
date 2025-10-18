import { useQueryClient } from "@tanstack/react-query";
import type { SendMessageRequestBody } from "mix-typescript-sdk/models/operations/sendmessage";
import type {
	SSECompleteEvent,
	SSEContentEvent,
	SSEErrorEvent,
	SSEEventStream,
	SSEPermissionEvent,
	SSESessionCreatedEvent,
	SSEThinkingEvent,
	SSEToolEvent,
	SSEToolExecutionCompleteEvent,
	SSEToolExecutionStartEvent,
	SSEToolParameterDeltaEvent,
	SSEUserMessageCreatedEvent,
} from "mix-typescript-sdk/models/sseeventstream";
import { useCallback, useEffect, useRef, useState } from "react";
import { CACHE_KEYS } from "@/lib/cache-keys";
import { mix } from "@/lib/mix-sdk";
import type { Attachment } from "@/stores/attachmentSlice";
import type { ToolCall } from "@/types/common";
import type { TimelineEntry } from "@/types/message";
import { expandFileReferences } from "@/utils/attachmentUtils";

export type SSEPermissionRequest = {
	id: string;
	sessionId: string;
	toolName: string;
	description: string;
	action: string;
	path: string;
	params: Record<string, unknown>;
};

type PersistentSSEState = {
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
	newlyCreatedSessionId: string | null;
	pendingUserMessage: {
		text: string;
		attachments?: Attachment[];
	} | null;
	assistantMessageId: string | null;
	userMessageId: string | null;
};

type PersistentSSEHook = PersistentSSEState & {
	submitMessage: (params: {
		text: string;
		attachments?: Attachment[];
		referenceMap?: Map<string, string>;
		planMode?: boolean;
	}) => Promise<void>;

	buttonStatus: "ready" | "streaming" | "paused" | "error";
	isSubmitDisabled: boolean;

	cancelMessage: () => Promise<void>;
	resetCancelledState: () => void;
	clearNewlyCreatedSession: () => void;
	clearStreamingContent: () => void;
	clearPendingUserMessage: () => void;
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
		newlyCreatedSessionId: null,
		pendingUserMessage: null,
		assistantMessageId: null,
		userMessageId: null,
	});

	const toolCallsMap = useRef<Map<string, ToolCall>>(new Map());
	const toolStartTimes = useRef<Map<string, number>>(new Map());
	const toolParameterDeltas = useRef<Map<string, string>>(new Map()); // Accumulate partial JSON by tool ID
	const timelineRef = useRef<TimelineEntry[]>([]);
	const connectedRef = useRef<boolean>(false);
	const currentSessionRef = useRef<string>("");
	const streamAbortController = useRef<AbortController | null>(null);
	const lastEventIdRef = useRef<string | undefined>(undefined);

	useEffect(() => {
		connectedRef.current = state.connected;
	}, [state.connected]);

	// Stream processing function
	const processEventStream = useCallback(
		async (sessionId: string, abortController: AbortController) => {
			try {
				setState((prev) => ({ ...prev, connecting: true, error: null }));

				const result = await mix.streaming.streamEvents({
					sessionId,
					lastEventID: lastEventIdRef.current,
				});

				setState((prev) => ({ ...prev, connected: true, connecting: false }));

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
						case "connected": {
							// const connectedEvent = event as SSEConnectedEvent;
							setState((prev) => ({
								...prev,
								connected: true,
								connecting: false,
							}));
							break;
						}

						case "heartbeat": {
							// const heartbeatEvent = event as SSEHeartbeatEvent;
							// Heartbeat events keep connection alive - no UI state changes needed
							break;
						}

						case "thinking": {
							const thinkingEvent = event as SSEThinkingEvent;
							const thinkingContent = thinkingEvent.data.content || "";
							const parentToolCallId = thinkingEvent.data.parentToolCallId;
							const assistantMessageId = thinkingEvent.data.assistantMessageId;

							// Add to timeline
							const thinkingEntry: TimelineEntry = {
								type: "thinking",
								timestamp: Date.now(),
								content: thinkingContent,
								id: `thinking-${Date.now()}-${Math.random().toString(36).substr(2, 9)}`,
								parentToolCallId,
							};

							timelineRef.current = [...timelineRef.current, thinkingEntry];

							setState((prev) => ({
								...prev,
								reasoning: (prev.reasoning || "") + thinkingContent,
								timeline: [...timelineRef.current],
								processing: true,
								assistantMessageId:
									assistantMessageId || prev.assistantMessageId,
							}));
							break;
						}

						case "content": {
							const contentEvent = event as SSEContentEvent;
							const contentDelta = contentEvent.data.content || "";
							const parentToolCallId = contentEvent.data.parentToolCallId;
							const assistantMessageId = contentEvent.data.assistantMessageId;

							// Find the last entry in timeline
							const lastEntry =
								timelineRef.current[timelineRef.current.length - 1];

							// Only accumulate content if it's from the same parent (or both have no parent)
							if (
								lastEntry &&
								lastEntry.type === "content" &&
								lastEntry.parentToolCallId === parentToolCallId
							) {
								const existingContent = lastEntry.content;
								timelineRef.current[timelineRef.current.length - 1] = {
									...lastEntry,
									content: existingContent + contentDelta,
									timestamp: Date.now(),
									parentToolCallId, // Preserve the parentToolCallId
								};
							} else {
								// Create new content entry (different parent or first content)
								const contentEntry: TimelineEntry = {
									type: "content",
									timestamp: Date.now(),
									content: contentDelta,
									id: `content-${Date.now()}-${Math.random().toString(36).substr(2, 9)}`,
									parentToolCallId,
								};
								timelineRef.current = [...timelineRef.current, contentEntry];
							}

							setState((prev) => ({
								...prev,
								finalContent: (prev.finalContent || "") + contentDelta,
								timeline: [...timelineRef.current],
								processing: true,
								assistantMessageId:
									assistantMessageId || prev.assistantMessageId,
							}));
							break;
						}

						case "tool_parameter_delta": {
							// Handle real-time tool parameter streaming
							const deltaEvent = event as SSEToolParameterDeltaEvent;
							const toolCallId = deltaEvent.data.toolCallId;
							const inputDelta = deltaEvent.data.input;
							const assistantMessageId = deltaEvent.data.assistantMessageId;

							// Validate required fields
							if (!toolCallId || typeof toolCallId !== "string") {
								console.error(
									"tool_parameter_delta: missing or invalid toolCallId",
									deltaEvent,
								);
								break;
							}
							if (inputDelta === undefined || inputDelta === null) {
								console.error(
									"tool_parameter_delta: missing input delta for tool",
									toolCallId,
								);
								break;
							}

							// Check if tool exists - fail loudly if it doesn't
							const existingToolCall = toolCallsMap.current.get(toolCallId);
							if (!existingToolCall) {
								console.error(
									`tool_parameter_delta: received delta for non-existent tool ${toolCallId}`,
								);
								break;
							}

							// Don't accumulate deltas for completed tools
							if (
								existingToolCall.status === "completed" ||
								existingToolCall.status === "error"
							) {
								console.warn(
									`tool_parameter_delta: ignoring delta for already ${existingToolCall.status} tool ${toolCallId}`,
								);
								break;
							}

							// Accumulate the delta
							const accumulated =
								(toolParameterDeltas.current.get(toolCallId) || "") +
								inputDelta;
							toolParameterDeltas.current.set(toolCallId, accumulated);

							// Try to parse the accumulated JSON
							let parsedParams: Record<string, unknown> | null = null;
							try {
								parsedParams = JSON.parse(accumulated);
							} catch {
								// JSON not yet complete/parseable - skip update
								break;
							}

							// JSON is parseable! Update the tool call
							if (parsedParams) {
								const updatedToolCall = {
									...existingToolCall,
									parameters: parsedParams,
								};

								toolCallsMap.current.set(toolCallId, updatedToolCall);

								// Update timeline entry
								timelineRef.current = timelineRef.current.map((entry) =>
									entry.type === "tool" && entry.content.id === toolCallId
										? { ...entry, content: updatedToolCall }
										: entry,
								);

								setState((prev) => ({
									...prev,
									toolCalls: Array.from(toolCallsMap.current.values()),
									timeline: [...timelineRef.current],
									assistantMessageId:
										assistantMessageId || prev.assistantMessageId,
								}));

								// Clear accumulated deltas after successful parse to prevent stale accumulation
								toolParameterDeltas.current.delete(toolCallId);
							}
							break;
						}

						case "tool": {
							const toolEvent = event as SSEToolEvent;
							const parentToolCallId = toolEvent.data.parentToolCallId;
							const assistantMessageId = toolEvent.data.assistantMessageId;

							const toolCall: ToolCall = {
								id: toolEvent.data.id || `${toolEvent.data.name}-${Date.now()}`,
								name: toolEvent.data.name || "unknown",
								description: toolEvent.data.name || "Tool execution",
								status:
									(toolEvent.data.status as
										| "pending"
										| "running"
										| "completed"
										| "error") || "pending",
								parameters: toolEvent.data.input
									? typeof toolEvent.data.input === "string"
										? (() => {
												try {
													return JSON.parse(toolEvent.data.input);
												} catch {
													return { input: toolEvent.data.input };
												}
											})()
										: toolEvent.data.input
									: {},
								result: undefined,
								error: undefined,
							};

							if (
								toolEvent.data.status === "running" &&
								!toolStartTimes.current.has(toolCall.id)
							) {
								toolStartTimes.current.set(toolCall.id, Date.now());
							}

							if (
								(toolEvent.data.status === "completed" ||
									toolEvent.data.status === "error") &&
								toolStartTimes.current.has(toolCall.id)
							) {
								toolStartTimes.current.delete(toolCall.id);
							}

							toolCallsMap.current.set(toolCall.id, toolCall);

							// Add to timeline when tool is first seen
							if (
								timelineRef.current.some(
									(entry) =>
										entry.type === "tool" && entry.content.id === toolCall.id,
								)
							) {
								// Update existing tool entry
								timelineRef.current = timelineRef.current.map((entry) =>
									entry.type === "tool" && entry.content.id === toolCall.id
										? { ...entry, content: toolCall }
										: entry,
								);
							} else {
								const toolEntry: TimelineEntry = {
									type: "tool",
									timestamp: Date.now(),
									content: toolCall,
									id: toolCall.id,
									parentToolCallId,
								};
								timelineRef.current = [...timelineRef.current, toolEntry];
							}

							setState((prev) => ({
								...prev,
								toolCalls: Array.from(toolCallsMap.current.values()),
								timeline: [...timelineRef.current],
								processing: true,
								assistantMessageId:
									assistantMessageId || prev.assistantMessageId,
							}));
							break;
						}

						case "tool_execution_start": {
							const toolStartEvent = event as SSEToolExecutionStartEvent;
							const toolCallId = toolStartEvent.data.toolCallId;
							const progress = toolStartEvent.data.progress;

							const existingToolCall = toolCallsMap.current.get(toolCallId);
							if (existingToolCall) {
								const updatedToolCall = {
									...existingToolCall,
									status: "running" as const,
									description: progress,
								};

								toolCallsMap.current.set(toolCallId, updatedToolCall);
								toolStartTimes.current.set(toolCallId, Date.now());

								// Update timeline entry
								timelineRef.current = timelineRef.current.map((entry) =>
									entry.type === "tool" && entry.content.id === toolCallId
										? { ...entry, content: updatedToolCall }
										: entry,
								);

								setState((prev) => ({
									...prev,
									toolCalls: Array.from(toolCallsMap.current.values()),
									timeline: [...timelineRef.current],
									processing: true,
								}));
							}
							break;
						}

						case "tool_execution_complete": {
							const toolCompleteEvent = event as SSEToolExecutionCompleteEvent;
							const toolCallId = toolCompleteEvent.data.toolCallId;
							const progress = toolCompleteEvent.data.progress;
							const success = toolCompleteEvent.data.success;

							const existingToolCall = toolCallsMap.current.get(toolCallId);
							if (existingToolCall) {
								const updatedToolCall = {
									...existingToolCall,
									status: success ? ("completed" as const) : ("error" as const),
									description: progress,
									result: success ? progress : undefined,
									error: success ? undefined : progress,
								};

								toolCallsMap.current.set(toolCallId, updatedToolCall);

								if (toolStartTimes.current.has(toolCallId)) {
									toolStartTimes.current.delete(toolCallId);
								}

								timelineRef.current = timelineRef.current.map((entry) =>
									entry.type === "tool" && entry.content.id === toolCallId
										? { ...entry, content: updatedToolCall }
										: entry,
								);

								setState((prev) => ({
									...prev,
									toolCalls: Array.from(toolCallsMap.current.values()),
									timeline: [...timelineRef.current],
									processing: true,
								}));
							}
							break;
						}

						case "complete": {
							const completeEvent = event as SSECompleteEvent;
							setState((prev) => {
								return {
									...prev,
									reasoning: completeEvent.data.reasoning || null,
									reasoningDuration:
										completeEvent.data.reasoningDuration || null,
									completed: true,
									processing: false,
									assistantMessageId: completeEvent.data.messageId || null,
								};
							});
							break;
						}

						case "error": {
							const errorEvent = event as SSEErrorEvent;
							setState((prev) => ({
								...prev,
								error: errorEvent.data.error || "Stream error",
								connecting: false,
								processing: false,
								rateLimit: errorEvent.data.retryAfter
									? {
											retryAfter: errorEvent.data.retryAfter,
											attempt: errorEvent.data.attempt || 1,
											maxAttempts: errorEvent.data.maxAttempts || 8,
										}
									: undefined,
							}));
							break;
						}

						case "permission": {
							const permissionEvent = event as SSEPermissionEvent;
							const permissionRequest: SSEPermissionRequest = {
								id: permissionEvent.data.id,
								sessionId: permissionEvent.data.sessionId,
								toolName: permissionEvent.data.toolName,
								description: permissionEvent.data.description,
								action: permissionEvent.data.action,
								path: permissionEvent.data.path || "",
								params: permissionEvent.data.params || {},
							};

							setState((prev) => ({
								...prev,
								permissionRequests: [
									...prev.permissionRequests,
									permissionRequest,
								],
							}));
							break;
						}

						case "user_message_created": {
							const userMsgEvent = event as SSEUserMessageCreatedEvent;

							// Store user message ID for duplicate detection
							setState((prev) => ({
								...prev,
								userMessageId: userMsgEvent.data.messageId,
							}));
							break;
						}

						case "session_created": {
							const sessionCreatedEvent = event as SSESessionCreatedEvent;

							// Store the newly created session ID for navigation
							setState((prev) => ({
								...prev,
								newlyCreatedSessionId: sessionCreatedEvent.data.sessionId,
							}));

							// Global session events - invalidate sessions list cache for real-time updates
							queryClient.invalidateQueries({ queryKey: CACHE_KEYS.sessions });
							break;
						}

						case "session_deleted": {
							// Global session events - invalidate sessions list cache for real-time updates
							queryClient.invalidateQueries({ queryKey: CACHE_KEYS.sessions });
							break;
						}

						default: {
							// Handle any other event types that might be added in the future
							console.warn(
								"Unknown event type:",
								(event as SSEEventStream).event,
							);
							break;
						}
					}
				}
			} catch (error) {
				if (!abortController.signal.aborted) {
					console.error("Stream processing error:", error);
					setState((prev) => ({
						...prev,
						connected: false,
						connecting: false,
						error:
							error instanceof Error
								? error.message
								: "Stream connection failed",
					}));
				}
			}
		},
		[queryClient],
	);

	useEffect(() => {
		if (!sessionId) {
			return;
		}

		// Prevent duplicate connections for same session
		if (sessionId === currentSessionRef.current) {
			return;
		}

		// Clean up previous session
		if (streamAbortController.current) {
			streamAbortController.current.abort();
		}

		toolCallsMap.current.clear();
		toolStartTimes.current.clear();
		toolParameterDeltas.current.clear();
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
			newlyCreatedSessionId: null,
			pendingUserMessage: null,
			assistantMessageId: null,
			userMessageId: null,
		});

		// Create new abort controller for this session
		streamAbortController.current = new AbortController();

		// Start streaming with SDK
		processEventStream(sessionId, streamAbortController.current).catch(
			(error) => {
				console.error("Stream processing failed:", error);
				setState((prev) => ({
					...prev,
					connected: false,
					connecting: false,
					error:
						error instanceof Error ? error.message : "Stream connection failed",
				}));
			},
		);

		return () => {
			if (streamAbortController.current) {
				streamAbortController.current.abort();
				streamAbortController.current = null;
			}
			toolCallsMap.current.clear();
			toolStartTimes.current.clear();
			toolParameterDeltas.current.clear();
			timelineRef.current = [];
			currentSessionRef.current = "";
			lastEventIdRef.current = undefined;
		};
	}, [sessionId, processEventStream]);

	// Cleanup on component unmount
	useEffect(() => {
		return () => {
			if (streamAbortController.current) {
				streamAbortController.current.abort();
				streamAbortController.current = null;
			}
			toolCallsMap.current.clear();
			toolStartTimes.current.clear();
			toolParameterDeltas.current.clear();
			timelineRef.current = [];
			currentSessionRef.current = "";
			lastEventIdRef.current = undefined;
		};
	}, []);

	const sendMessage = useCallback(
		async (content: string, userText: string, attachments?: Attachment[]) => {
			if (!sessionId) {
				throw new Error("No session ID available");
			}

			setState((prev) => ({
				...prev,
				error: null,
				toolCalls: [],
				startTime: Date.now(),
				finalContent: "", // Reset to empty string for delta accumulation
				completed: false,
				processing: true,
				cancelling: false,
				cancelled: false,
				reasoning: null,
				reasoningDuration: null,
				timeline: [],
				rateLimit: undefined,
				pendingUserMessage: {
					text: userText,
					attachments,
				},
			}));

			toolCallsMap.current.clear();
			toolParameterDeltas.current.clear();
			timelineRef.current = [];

			try {
				// Use REST API to send message - this triggers agent processing once on server
				// and broadcasts events to all SSE connections
				await mix.messages.send({
					id: sessionId,
					requestBody: JSON.parse(content), // content is already JSON stringified
				});
			} catch (error) {
				console.error("Failed to send message to backend:", {
					error: error instanceof Error ? error.message : String(error),
					sessionId,
				});
				setState((prev) => ({
					...prev,
					error:
						error instanceof Error ? error.message : "Failed to send message",
					processing: false,
					cancelling: false,
					pendingUserMessage: null, // Clear on error
				}));
				throw error;
			}
		},
		[sessionId],
	);

	const cancelMessage = useCallback(async () => {
		if (!sessionId) {
			throw new Error("No session ID available");
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
					error instanceof Error ? error.message : "Failed to cancel message",
			}));
			throw error;
		}
	}, [sessionId]);

	const resetCancelledState = useCallback(() => {
		setState((prev) => ({ ...prev, cancelled: false }));
	}, []);

	const clearNewlyCreatedSession = useCallback(() => {
		setState((prev) => ({ ...prev, newlyCreatedSessionId: null }));
	}, []);

	const clearStreamingContent = useCallback(() => {
		setState((prev) => ({
			...prev,
			finalContent: null,
			timeline: [],
			toolCalls: [],
			reasoning: null,
			reasoningDuration: null,
			cancelled: false,
			completed: false,
			pendingUserMessage: null,
			assistantMessageId: null,
			userMessageId: null,
		}));
		toolCallsMap.current.clear();
		toolParameterDeltas.current.clear();
		timelineRef.current = [];
	}, []);

	const clearPendingUserMessage = useCallback(() => {
		setState((prev) => ({
			...prev,
			pendingUserMessage: null,
		}));
	}, []);

	const grantPermission = useCallback(async (id: string) => {
		try {
			await mix.permissions.grant({ id });

			// Remove the permission request from state
			setState((prev) => ({
				...prev,
				permissionRequests: prev.permissionRequests.filter(
					(req) => req.id !== id,
				),
			}));
		} catch (error) {
			console.error("Failed to grant permission:", error);
			throw error;
		}
	}, []);

	const denyPermission = useCallback(async (id: string) => {
		try {
			await mix.permissions.deny({ id });

			// Remove the permission request from state
			setState((prev) => ({
				...prev,
				permissionRequests: prev.permissionRequests.filter(
					(req) => req.id !== id,
				),
			}));
		} catch (error) {
			console.error("Failed to deny permission:", error);
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
		}) => {
			const {
				text,
				attachments = [],
				referenceMap = new Map(),
				planMode = false,
			} = params;

			if (!(text && sessionId && state.connected)) {
				return;
			}

			try {
				// Expand file references (no longer need media URLs for API)
				const expandedText = expandFileReferences(text, referenceMap);

				const messageData: SendMessageRequestBody = {
					text: expandedText,
					planMode,
				};

				// Send to backend with optimistic UI - pass original text and attachments
				await sendMessage(JSON.stringify(messageData), text, attachments);

				// Optimistic UI is now handled in state
				// Cache will be invalidated when streaming completes
			} catch (error) {
				console.error("Failed to send message:", error);
				throw error; // Re-throw so parent can handle
			}
		},
		[sessionId, state.connected, sendMessage],
	);

	// Simple button status computation
	const buttonStatus = state.cancelling
		? "paused"
		: state.cancelled
			? "streaming"
			: state.processing
				? "streaming"
				: state.error
					? "error"
					: "ready";

	// Simple submit button disabled state
	const isSubmitDisabled = buttonStatus === "paused" || !state.connected;

	return {
		...state,
		submitMessage,
		buttonStatus,
		isSubmitDisabled,
		cancelMessage,
		resetCancelledState,
		clearNewlyCreatedSession,
		clearStreamingContent,
		clearPendingUserMessage,
		grantPermission,
		denyPermission,
	};
}
