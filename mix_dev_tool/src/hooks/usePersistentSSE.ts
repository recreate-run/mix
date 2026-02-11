import { useQueryClient } from "@tanstack/react-query";
import { CoreToolName } from "mix-typescript-sdk/models";
import type { Type as NotificationResponseType } from "mix-typescript-sdk/models/operations/respondtonotification";
import type {
	SendMessageRequestBody,
	ThinkingLevel,
} from "mix-typescript-sdk/models/operations/sendmessage";
import type {
	SSECompleteEvent,
	SSEContentEvent,
	SSEErrorEvent,
	SSEPermissionEvent,
	SSESessionCreatedEvent,
	SSEThinkingEvent,
	SSEToolExecutionCompleteEvent,
	SSEToolExecutionStartEvent,
	SSEToolUseParameterDeltaEvent,
	SSEToolUseParameterStreamingCompleteEvent,
	SSEToolUseStartEvent,
	SSEUserMessageCreatedEvent,
} from "mix-typescript-sdk/models/sseeventstream";
import { useCallback, useEffect, useRef, useState } from "react";
import { CACHE_KEYS } from "@/lib/cache-keys";
import { mix } from "@/lib/mix-sdk";
import type { Attachment } from "@/stores/attachmentSlice";
import type { ToolCall } from "@/types/common";
import type { MediaOutput } from "@/types/media";
import type { TimelineEntry, UIMessage } from "@/types/message";
import { expandFileReferences } from "@/utils/attachmentUtils";
import { getBackendUrl } from "@/utils/backendUrl";

export type SSEPermissionRequest = {
	id: string;
	sessionId: string;
	toolName: string;
	description: string;
	action: string;
	path: string;
	params: Record<string, unknown>;
};

export type SSENotificationRequest = {
	id: string;
	sessionId: string;
	type: "info" | "warning" | "error" | "question";
	title: string;
	message: string;
	responseType: "acknowledge" | "text" | "choice";
	choices?: string[];
	timeout: number;
	createdAt: number;
};

interface SSENotificationEventData {
	id: string;
	sessionId: string;
	notificationType: "info" | "warning" | "error" | "question";
	title: string;
	message: string;
	responseType: "acknowledge" | "text" | "choice";
	choices?: string[];
	timeout: number;
	createdAt: number;
}

type RawSSEEvent = {
	event: string;
	id?: string;
	data?: Record<string, unknown>;
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
	notifications: SSENotificationRequest[];
	newlyCreatedSessionId: string | null;
	pendingUserMessage: {
		text: string;
		attachments?: Attachment[];
	} | null;
	assistantMessageId: string | null;
	userMessageId: string | null;
	preStreamingMessageIds: Set<string>;
};

type PersistentSSEHook = PersistentSSEState & {
	submitMessage: (params: {
		text: string;
		attachments?: Attachment[];
		referenceMap?: Map<string, string>;
		planMode?: boolean;
		thinkingLevel?: ThinkingLevel;
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
	respondToNotification: (
		id: string,
		response: { type: NotificationResponseType; value?: string },
	) => Promise<void>;
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
		notifications: [],
		newlyCreatedSessionId: null,
		pendingUserMessage: null,
		assistantMessageId: null,
		userMessageId: null,
		preStreamingMessageIds: new Set(),
	});

	const toolCallsMap = useRef<Map<string, ToolCall>>(new Map());
	const toolStartTimes = useRef<Map<string, number>>(new Map());
	const toolParameterDeltas = useRef<Map<string, string>>(new Map());
	const timelineRef = useRef<TimelineEntry[]>([]);
	const connectedRef = useRef<boolean>(false);
	const currentSessionRef = useRef<string>("");
	const streamAbortController = useRef<AbortController | null>(null);
	const lastEventIdRef = useRef<string | undefined>(undefined);
	const sessionIdRef = useRef<string>(sessionId);
	const userMessageIdRef = useRef<string | null>(null);
	const streamingMessageIdRef = useRef<string | null>(null);
	const pendingUserMessageRef = useRef<{
		text: string;
		attachments?: Attachment[];
	} | null>(null);

	useEffect(() => {
		sessionIdRef.current = sessionId;
	}, [sessionId]);

	useEffect(() => {
		connectedRef.current = state.connected;
	}, [state.connected]);

	const updateMessagesCache = useCallback(
		(
			activeSessionId: string,
			updater: (messages: UIMessage[]) => UIMessage[],
		) => {
			queryClient.setQueryData<UIMessage[]>(
				CACHE_KEYS.sessionMessages(activeSessionId),
				(oldMessages = []) => updater(oldMessages),
			);
		},
		[queryClient],
	);

	const updateStreamingMessage = useCallback(
		(activeSessionId: string, updater: (message: UIMessage) => UIMessage) => {
			const streamingId = streamingMessageIdRef.current;
			if (!streamingId) return;

			updateMessagesCache(activeSessionId, (oldMessages) => {
				let foundStreaming = false;
				const updatedMessages = oldMessages.map((message) => {
					if (message.id === streamingId) {
						foundStreaming = true;
						return updater(message);
					}
					return message;
				});

				if (foundStreaming) {
					return updatedMessages;
				}

				const fallbackStreaming: UIMessage = {
					id: streamingId,
					content: "",
					from: "assistant",
					toolCalls: [],
					timeline: [],
					isStreaming: true,
					streamingStatus: "streaming",
				};
				return [...updatedMessages, updater(fallbackStreaming)];
			});
		},
		[updateMessagesCache],
	);

	const handleMixEvent = useCallback(
		(event: RawSSEEvent) => {
			const activeSessionId = sessionIdRef.current;
			if (!activeSessionId) return;

			switch (event.event) {
				case "connected": {
					setState((prev) => ({
						...prev,
						connected: true,
						connecting: false,
					}));
					break;
				}

				case "heartbeat": {
					break;
				}

				case "thinking": {
					const thinkingEvent = event as SSEThinkingEvent;
					const content = thinkingEvent.data.content || "";
					const parentToolCallId = thinkingEvent.data.parentToolCallId;
					const assistantMessageId = thinkingEvent.data.assistantMessageId;

					if (!content) return;

					const thinkingEntry: TimelineEntry = {
						type: "thinking",
						timestamp: Date.now(),
						content,
						id: `thinking-${Date.now()}-${Math.random().toString(36).slice(2, 9)}`,
						parentToolCallId,
					};

					timelineRef.current = [...timelineRef.current, thinkingEntry];

					setState((prev) => ({
						...prev,
						reasoning: (prev.reasoning || "") + content,
						timeline: [...timelineRef.current],
						processing: true,
						assistantMessageId: assistantMessageId || prev.assistantMessageId,
					}));

					updateStreamingMessage(activeSessionId, (message) => ({
						...message,
						timeline: [...timelineRef.current],
						reasoning: (message.reasoning || "") + content,
						isStreaming: true,
						streamingStatus: "streaming",
					}));
					break;
				}

				case "content": {
					const contentEvent = event as SSEContentEvent;
					const contentDelta = contentEvent.data.content || "";
					const parentToolCallId = contentEvent.data.parentToolCallId;
					const assistantMessageId = contentEvent.data.assistantMessageId;

					if (!contentDelta) return;

					const lastEntry = timelineRef.current[timelineRef.current.length - 1];
					if (
						lastEntry &&
						lastEntry.type === "content" &&
						lastEntry.parentToolCallId === parentToolCallId
					) {
						timelineRef.current[timelineRef.current.length - 1] = {
							...lastEntry,
							content: `${lastEntry.content}${contentDelta}`,
							timestamp: Date.now(),
							parentToolCallId,
						};
					} else {
						timelineRef.current = [
							...timelineRef.current,
							{
								type: "content",
								timestamp: Date.now(),
								content: contentDelta,
								id: `content-${Date.now()}-${Math.random().toString(36).slice(2, 9)}`,
								parentToolCallId,
							},
						];
					}

					setState((prev) => ({
						...prev,
						finalContent: (prev.finalContent || "") + contentDelta,
						timeline: [...timelineRef.current],
						processing: true,
						assistantMessageId: assistantMessageId || prev.assistantMessageId,
					}));

					updateStreamingMessage(activeSessionId, (message) => ({
						...message,
						content: `${message.content || ""}${contentDelta}`,
						timeline: [...timelineRef.current],
						toolCalls: Array.from(toolCallsMap.current.values()),
						isStreaming: true,
						streamingStatus: "streaming",
					}));
					break;
				}

				case "tool_use_parameter_delta": {
					const deltaEvent = event as SSEToolUseParameterDeltaEvent;
					const toolCallId = deltaEvent.data.toolCallId;
					const inputDelta = deltaEvent.data.input;
					const assistantMessageId = deltaEvent.data.assistantMessageId;

					if (!toolCallId || inputDelta === undefined || inputDelta === null) {
						console.error(
							"[usePersistentSSE] tool_use_parameter_delta missing fields",
							deltaEvent.data,
						);
						break;
					}

					const existingToolCall = toolCallsMap.current.get(toolCallId);
					if (!existingToolCall) {
						console.error(
							`[usePersistentSSE] tool_use_parameter_delta for unknown tool ${toolCallId}`,
						);
						break;
					}

					if (
						existingToolCall.status === "completed" ||
						existingToolCall.status === "error"
					) {
						break;
					}

					const accumulated =
						(toolParameterDeltas.current.get(toolCallId) || "") + inputDelta;
					toolParameterDeltas.current.set(toolCallId, accumulated);

					let parsedParams: Record<string, unknown> | null = null;
					try {
						parsedParams = JSON.parse(accumulated);
					} catch {
						break;
					}

					if (parsedParams) {
						const updatedToolCall = {
							...existingToolCall,
							parameters: parsedParams,
						};

						toolCallsMap.current.set(toolCallId, updatedToolCall);
						timelineRef.current = timelineRef.current.map((entry) =>
							entry.type === "tool" && entry.content.id === toolCallId
								? { ...entry, content: updatedToolCall }
								: entry,
						);

						setState((prev) => ({
							...prev,
							toolCalls: Array.from(toolCallsMap.current.values()),
							timeline: [...timelineRef.current],
							assistantMessageId: assistantMessageId || prev.assistantMessageId,
						}));

						updateStreamingMessage(activeSessionId, (message) => ({
							...message,
							toolCalls: Array.from(toolCallsMap.current.values()),
							timeline: [...timelineRef.current],
							isStreaming: true,
							streamingStatus: "streaming",
						}));
					}
					break;
				}

				case "tool_use_start": {
					const toolEvent = event as SSEToolUseStartEvent;
					const toolCallId =
						toolEvent.data.id ||
						`${toolEvent.data.name || "tool"}-${Date.now()}`;
					const parentToolCallId = toolEvent.data.parentToolCallId;
					const assistantMessageId = toolEvent.data.assistantMessageId;

					if (toolCallsMap.current.has(toolCallId)) break;

					const toolCall: ToolCall = {
						id: toolCallId,
						name: toolEvent.data.name || "unknown",
						description: toolEvent.data.name || "Tool execution",
						status: "pending",
						parameters: {},
						result: undefined,
						error: undefined,
					};

					toolCallsMap.current.set(toolCall.id, toolCall);
					timelineRef.current = [
						...timelineRef.current,
						{
							type: "tool",
							timestamp: Date.now(),
							content: toolCall,
							id: toolCall.id,
							parentToolCallId,
						},
					];

					setState((prev) => ({
						...prev,
						toolCalls: Array.from(toolCallsMap.current.values()),
						timeline: [...timelineRef.current],
						processing: true,
						assistantMessageId: assistantMessageId || prev.assistantMessageId,
					}));

					updateStreamingMessage(activeSessionId, (message) => ({
						...message,
						toolCalls: Array.from(toolCallsMap.current.values()),
						timeline: [...timelineRef.current],
						isStreaming: true,
						streamingStatus: "streaming",
					}));
					break;
				}

				case "tool_use_parameter_streaming_complete": {
					const toolEvent = event as SSEToolUseParameterStreamingCompleteEvent;
					const toolCallId = toolEvent.data.id;
					const parentToolCallId = toolEvent.data.parentToolCallId;
					const assistantMessageId = toolEvent.data.assistantMessageId;

					if (toolCallId) {
						toolParameterDeltas.current.delete(toolCallId);
					}

					const existingToolCall = toolCallsMap.current.get(toolCallId || "");
					const parsedInput = (() => {
						const raw = toolEvent.data.input;
						if (!raw) return existingToolCall?.parameters || {};
						if (typeof raw === "string") {
							try {
								return JSON.parse(raw);
							} catch {
								return { input: raw };
							}
						}
						return raw;
					})();

					const toolCall: ToolCall = existingToolCall
						? {
								...existingToolCall,
								parameters: parsedInput,
							}
						: {
								id:
									toolCallId ||
									`${toolEvent.data.name || "tool"}-${Date.now()}`,
								name: toolEvent.data.name || "unknown",
								description: toolEvent.data.name || "Tool execution",
								status: "pending",
								parameters: parsedInput,
								result: undefined,
								error: undefined,
							};

					toolCallsMap.current.set(toolCall.id, toolCall);

					if (
						timelineRef.current.some(
							(entry) =>
								entry.type === "tool" && entry.content.id === toolCall.id,
						)
					) {
						timelineRef.current = timelineRef.current.map((entry) =>
							entry.type === "tool" && entry.content.id === toolCall.id
								? { ...entry, content: toolCall }
								: entry,
						);
					} else {
						timelineRef.current = [
							...timelineRef.current,
							{
								type: "tool",
								timestamp: Date.now(),
								content: toolCall,
								id: toolCall.id,
								parentToolCallId,
							},
						];
					}

					setState((prev) => ({
						...prev,
						toolCalls: Array.from(toolCallsMap.current.values()),
						timeline: [...timelineRef.current],
						processing: true,
						assistantMessageId: assistantMessageId || prev.assistantMessageId,
					}));

					updateStreamingMessage(activeSessionId, (message) => ({
						...message,
						toolCalls: Array.from(toolCallsMap.current.values()),
						timeline: [...timelineRef.current],
						isStreaming: true,
						streamingStatus: "streaming",
					}));
					break;
				}

				case "tool_execution_start": {
					const toolStartEvent = event as SSEToolExecutionStartEvent;
					const toolCallId = toolStartEvent.data.toolCallId;
					const progress = toolStartEvent.data.progress;

					const existingToolCall = toolCallsMap.current.get(toolCallId);
					if (existingToolCall) {
						const updatedToolCall: ToolCall = {
							...existingToolCall,
							status: "running",
							description: progress || existingToolCall.description,
						};

						toolCallsMap.current.set(toolCallId, updatedToolCall);
						toolStartTimes.current.set(toolCallId, Date.now());
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

						updateStreamingMessage(activeSessionId, (message) => ({
							...message,
							toolCalls: Array.from(toolCallsMap.current.values()),
							timeline: [...timelineRef.current],
							isStreaming: true,
							streamingStatus: "streaming",
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
						const updatedToolCall: ToolCall = {
							...existingToolCall,
							status: success ? "completed" : "error",
							description: progress || existingToolCall.description,
							result: success ? progress : undefined,
							error: success ? undefined : progress,
						};

						toolCallsMap.current.set(toolCallId, updatedToolCall);
						toolStartTimes.current.delete(toolCallId);
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

						updateStreamingMessage(activeSessionId, (message) => ({
							...message,
							toolCalls: Array.from(toolCallsMap.current.values()),
							timeline: [...timelineRef.current],
							isStreaming: true,
							streamingStatus: "streaming",
						}));
					}
					break;
				}

				case "complete": {
					const completeEvent = event as SSECompleteEvent;
					const messageId = completeEvent.data.messageId;
					const content = completeEvent.data.content || "";
					const reasoning = completeEvent.data.reasoning || "";
					const reasoningDuration = completeEvent.data.reasoningDuration;

					setState((prev) => ({
						...prev,
						reasoning: reasoning || null,
						reasoningDuration: reasoningDuration || null,
						completed: true,
						processing: false,
						assistantMessageId: messageId || null,
					}));

					updateMessagesCache(activeSessionId, (oldMessages) => {
						const userMsgId = userMessageIdRef.current;
						const asstMsgId = messageId;
						const streamingId = streamingMessageIdRef.current;

						const userExists =
							!!userMsgId && oldMessages.some((m) => m.id === userMsgId);

						let nextMessages = [...oldMessages];
						if (!userExists && pendingUserMessageRef.current) {
							nextMessages.push({
								id: userMsgId || undefined,
								content: pendingUserMessageRef.current.text,
								from: "user",
								attachments: pendingUserMessageRef.current.attachments,
							});
						}

						const toolCallsArray = Array.from(toolCallsMap.current.values());
						let finalContent = "";
						for (const entry of timelineRef.current) {
							if (entry.type === "content") {
								finalContent += entry.content;
							}
						}

						const hasAssistantPayload =
							(finalContent && finalContent.trim().length > 0) ||
							(content && content.trim().length > 0) ||
							toolCallsArray.length > 0 ||
							timelineRef.current.length > 0 ||
							reasoning.trim().length > 0;

						if (!hasAssistantPayload) {
							if (streamingId) {
								const hasStreamingMessage = nextMessages.some(
									(m) => m.id === streamingId,
								);
								if (hasStreamingMessage) {
									nextMessages = nextMessages.map((message) => {
										if (message.id === streamingId) {
											return {
												...message,
												isStreaming: false,
												streamingStatus: "final",
											};
										}
										return message;
									});
								}
							}
							return nextMessages;
						}

						const mediaOutputs = toolCallsArray.find(
							(tc) => tc.name === CoreToolName.Show,
						)?.parameters?.outputs as MediaOutput[] | undefined;

						const finalAssistant: UIMessage = {
							id: asstMsgId || streamingId || undefined,
							content: finalContent || content || "",
							from: "assistant",
							toolCalls: toolCallsArray.length > 0 ? toolCallsArray : undefined,
							timeline:
								timelineRef.current.length > 0
									? [...timelineRef.current]
									: undefined,
							mediaOutputs:
								mediaOutputs && mediaOutputs.length > 0
									? mediaOutputs
									: undefined,
							reasoning: reasoning || undefined,
							reasoningDuration: reasoningDuration || undefined,
							isStreaming: false,
							streamingStatus: "final",
						};

						const existingFinalIndex = asstMsgId
							? nextMessages.findIndex((m) => m.id === asstMsgId)
							: -1;

						if (existingFinalIndex >= 0) {
							nextMessages[existingFinalIndex] = {
								...nextMessages[existingFinalIndex],
								...finalAssistant,
								id: asstMsgId,
							};
							if (streamingId && streamingId !== asstMsgId) {
								nextMessages = nextMessages.filter((m) => m.id !== streamingId);
							}
							return nextMessages;
						}

						if (streamingId) {
							let replacedStreaming = false;
							const replaced = nextMessages.map((message) => {
								if (message.id === streamingId) {
									replacedStreaming = true;
									return {
										...finalAssistant,
										id: asstMsgId || streamingId,
									};
								}
								return message;
							});

							if (replacedStreaming) {
								return replaced;
							}

							return [...replaced, finalAssistant];
						}

						return [...nextMessages, finalAssistant];
					});

					pendingUserMessageRef.current = null;
					userMessageIdRef.current = null;
					streamingMessageIdRef.current = null;
					break;
				}

				case "error": {
					const errorEvent = event as SSEErrorEvent;
					const errorMessage = errorEvent.data.error || "Stream error";
					setState((prev) => ({
						...prev,
						error: errorMessage,
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

					updateStreamingMessage(activeSessionId, (message) => ({
						...message,
						isStreaming: false,
						streamingStatus: "error",
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
						permissionRequests: [...prev.permissionRequests, permissionRequest],
					}));
					break;
				}

				case "notification": {
					const notificationData = event.data as
						| SSENotificationEventData
						| undefined;
					if (!notificationData) {
						break;
					}
					const notification: SSENotificationRequest = {
						id: notificationData.id,
						sessionId: notificationData.sessionId,
						type: notificationData.notificationType,
						title: notificationData.title,
						message: notificationData.message,
						responseType: notificationData.responseType,
						choices: notificationData.choices,
						timeout: notificationData.timeout,
						createdAt: notificationData.createdAt,
					};
					setState((prev) => ({
						...prev,
						notifications: [...prev.notifications, notification],
					}));
					break;
				}

				case "user_message_created": {
					const userMsgEvent = event as SSEUserMessageCreatedEvent;
					const messageId = userMsgEvent.data.messageId;
					const content = userMsgEvent.data.content || "";

					if (!content) {
						console.warn(
							"user_message_created: received event with missing content",
							userMsgEvent,
						);
						break;
					}

					const previousUserId = userMessageIdRef.current;
					userMessageIdRef.current = messageId || null;
					pendingUserMessageRef.current = { text: content, attachments: [] };

					setState((prev) => ({
						...prev,
						userMessageId: messageId || null,
						pendingUserMessage: { text: content, attachments: [] },
					}));

					updateMessagesCache(activeSessionId, (oldMessages) => {
						const hasRealId =
							!!messageId && oldMessages.some((m) => m.id === messageId);
						const updated = oldMessages.map((message): UIMessage => {
							if (previousUserId && message.id === previousUserId) {
								return {
									...message,
									id: messageId || message.id,
									content,
									from: "user" as const,
								};
							}
							return message;
						});

						if (!hasRealId && !previousUserId && messageId) {
							return [
								...updated,
								{
									id: messageId,
									content,
									from: "user" as const,
								},
							];
						}

						return updated;
					});
					break;
				}

				case "session_created": {
					const sessionCreatedEvent = event as SSESessionCreatedEvent;
					setState((prev) => ({
						...prev,
						newlyCreatedSessionId: sessionCreatedEvent.data.sessionId,
					}));
					queryClient.invalidateQueries({ queryKey: CACHE_KEYS.sessions });
					break;
				}

				case "session_deleted": {
					queryClient.invalidateQueries({ queryKey: CACHE_KEYS.sessions });
					break;
				}

				default: {
					const unknownEvent = event as RawSSEEvent;
					console.warn("Unknown event type:", unknownEvent.event);
					break;
				}
			}
		},
		[queryClient, updateMessagesCache, updateStreamingMessage],
	);

	const processEventStream = useCallback(
		(activeSessionId: string, abortController: AbortController) => {
			return new Promise<void>((resolve, reject) => {
				setState((prev) => ({ ...prev, connecting: true, error: null }));

				const streamUrl = new URL("/stream", getBackendUrl());
				streamUrl.searchParams.set("sessionId", activeSessionId);
				if (lastEventIdRef.current) {
					streamUrl.searchParams.set("lastEventID", lastEventIdRef.current);
				}

				const eventSource = new EventSource(streamUrl.toString());
				let settled = false;

				const settleResolve = () => {
					if (!settled) {
						settled = true;
						resolve();
					}
				};

				const settleReject = (error: Error) => {
					if (!settled) {
						settled = true;
						reject(error);
					}
				};

				const parseMessageData = (rawData: string): Record<string, unknown> => {
					if (!rawData || rawData === "undefined") return {};
					try {
						return JSON.parse(rawData) as Record<string, unknown>;
					} catch {
						return {};
					}
				};

				const forwardEvent = (eventName: string, message: MessageEvent) => {
					if (abortController.signal.aborted) return;
					const data = parseMessageData(message.data);
					const eventId = message.lastEventId || undefined;
					if (eventId) {
						lastEventIdRef.current = eventId;
					}
					handleMixEvent({
						event: eventName,
						id: eventId,
						data,
					});
				};

				const eventTypes = [
					"connected",
					"heartbeat",
					"user_message_created",
					"thinking",
					"content",
					"tool_use_start",
					"tool_use_parameter_delta",
					"tool_use_parameter_streaming_complete",
					"tool_execution_start",
					"tool_execution_complete",
					"complete",
					"error",
					"permission",
					"notification",
					"session_created",
					"session_deleted",
				] as const;

				for (const eventType of eventTypes) {
					eventSource.addEventListener(eventType, (message) => {
						forwardEvent(eventType, message as MessageEvent);
					});
				}

				eventSource.onmessage = (message) => {
					const data = parseMessageData(message.data);
					const typedEvent =
						typeof data.type === "string" ? data.type : "message";
					forwardEvent(typedEvent, message);
				};

				eventSource.onopen = () => {
					setState((prev) => ({ ...prev, connected: true, connecting: false }));
				};

				eventSource.onerror = (error) => {
					if (abortController.signal.aborted) {
						eventSource.close();
						settleResolve();
						return;
					}
					console.error("Stream processing error:", error);
					setState((prev) => ({
						...prev,
						connected: false,
						connecting: false,
						error: "Stream connection failed",
					}));
					eventSource.close();
					settleReject(new Error("Stream connection failed"));
				};

				abortController.signal.addEventListener(
					"abort",
					() => {
						eventSource.close();
						settleResolve();
					},
					{ once: true },
				);
			});
		},
		[handleMixEvent],
	);

	useEffect(() => {
		if (!sessionId) return;
		if (sessionId === currentSessionRef.current) return;

		if (streamAbortController.current) {
			streamAbortController.current.abort();
		}

		pendingUserMessageRef.current = null;
		userMessageIdRef.current = null;
		streamingMessageIdRef.current = null;
		toolCallsMap.current.clear();
		toolStartTimes.current.clear();
		toolParameterDeltas.current.clear();
		timelineRef.current = [];
		lastEventIdRef.current = undefined;
		currentSessionRef.current = sessionId;

		setState((prev) => ({
			...prev,
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
			rateLimit: undefined,
			permissionRequests: [],
			notifications: [],
			pendingUserMessage: null,
			assistantMessageId: null,
			userMessageId: null,
			preStreamingMessageIds: new Set(),
		}));

		streamAbortController.current = new AbortController();
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
		};
	}, [sessionId, processEventStream]);

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
			pendingUserMessageRef.current = null;
			userMessageIdRef.current = null;
			streamingMessageIdRef.current = null;
		};
	}, []);

	const sendMessage = useCallback(
		async (content: string, userText: string, attachments?: Attachment[]) => {
			if (!sessionId) {
				throw new Error("No session ID available");
			}

			const existingMessages = queryClient.getQueryData<UIMessage[]>(
				CACHE_KEYS.sessionMessages(sessionId),
			);
			const preStreamingIds = new Set<string>(
				existingMessages?.map((m) => m.id).filter((id): id is string => !!id) ||
					[],
			);

			const tempUserId = `pending-user-${sessionId}-${Date.now()}`;
			const streamingId = `streaming-${sessionId}-${Date.now()}`;
			userMessageIdRef.current = tempUserId;
			streamingMessageIdRef.current = streamingId;
			pendingUserMessageRef.current = { text: userText, attachments };

			setState((prev) => ({
				...prev,
				error: null,
				toolCalls: [],
				startTime: Date.now(),
				finalContent: "",
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
				userMessageId: tempUserId,
				preStreamingMessageIds: preStreamingIds,
			}));

			updateMessagesCache(sessionId, (oldMessages) => {
				const nextMessages = [...oldMessages];
				const hasUser = nextMessages.some((msg) => msg.id === tempUserId);
				const hasStreaming = nextMessages.some((msg) => msg.id === streamingId);

				if (!hasUser) {
					nextMessages.push({
						id: tempUserId,
						content: userText,
						from: "user",
						attachments,
					});
				}

				if (!hasStreaming) {
					nextMessages.push({
						id: streamingId,
						content: "",
						from: "assistant",
						toolCalls: [],
						timeline: [],
						isStreaming: true,
						streamingStatus: "streaming",
					});
				}

				return nextMessages;
			});

			toolCallsMap.current.clear();
			toolParameterDeltas.current.clear();
			timelineRef.current = [];

			try {
				await mix.messages.send({
					id: sessionId,
					requestBody: JSON.parse(content),
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
					pendingUserMessage: null,
				}));
				throw error;
			}
		},
		[sessionId, queryClient, updateMessagesCache],
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

			const activeSessionId = sessionIdRef.current;
			if (activeSessionId) {
				updateStreamingMessage(activeSessionId, (message) => ({
					...message,
					isStreaming: false,
					streamingStatus: "cancelled",
				}));
			}
		} catch (error) {
			setState((prev) => ({
				...prev,
				cancelling: false,
				error:
					error instanceof Error ? error.message : "Failed to cancel message",
			}));
			throw error;
		}
	}, [sessionId, updateStreamingMessage]);

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
			preStreamingMessageIds: new Set(),
		}));

		const activeSessionId = sessionIdRef.current;
		const streamingId = streamingMessageIdRef.current;
		if (activeSessionId && streamingId) {
			updateMessagesCache(activeSessionId, (oldMessages) =>
				oldMessages.filter((message) => message.id !== streamingId),
			);
		}

		toolCallsMap.current.clear();
		toolParameterDeltas.current.clear();
		timelineRef.current = [];
		streamingMessageIdRef.current = null;
	}, [updateMessagesCache]);

	const clearPendingUserMessage = useCallback(() => {
		setState((prev) => ({ ...prev, pendingUserMessage: null }));
	}, []);

	const grantPermission = useCallback(async (id: string) => {
		try {
			await mix.permissions.grant({ id });
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

	const respondToNotification = useCallback(
		async (
			id: string,
			response: { type: NotificationResponseType; value?: string },
		) => {
			try {
				await mix.notifications.respondToNotification({
					id,
					requestBody: response,
				});
				setState((prev) => ({
					...prev,
					notifications: prev.notifications.filter((notif) => notif.id !== id),
				}));
			} catch (error) {
				console.error("Failed to respond to notification:", error);
				throw error;
			}
		},
		[],
	);

	const submitMessage = useCallback(
		async (params: {
			text: string;
			attachments?: Attachment[];
			referenceMap?: Map<string, string>;
			planMode?: boolean;
			thinkingLevel?: ThinkingLevel;
		}) => {
			const {
				text,
				attachments = [],
				referenceMap = new Map(),
				planMode = false,
				thinkingLevel,
			} = params;

			if (!(text && sessionId && connectedRef.current)) {
				return;
			}

			const expandedText = expandFileReferences(text, referenceMap);
			const messageData: SendMessageRequestBody = {
				text: expandedText,
				planMode,
				...(thinkingLevel !== undefined && { thinkingLevel }),
			};

			await sendMessage(JSON.stringify(messageData), text, attachments);
		},
		[sessionId, sendMessage],
	);

	const buttonStatus = state.cancelling
		? "paused"
		: state.cancelled
			? "streaming"
			: state.processing
				? "streaming"
				: state.error
					? "error"
					: "ready";

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
		respondToNotification,
	};
}
