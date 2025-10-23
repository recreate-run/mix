import { Check, Copy, Undo2 } from "lucide-react";
import { useEffect, useState } from "react";
import { Button } from "@/components/ui/button";
import {
	AIMessage,
	AIMessageContent,
} from "@/components/ui/kibo-ui/ai/message";
import {
	AIReasoning,
	AIReasoningContent,
	AIReasoningTrigger,
} from "@/components/ui/kibo-ui/ai/reasoning";
import {
	AIToolContent,
	AIToolHeader,
	AIToolLadder,
	AIToolStep,
} from "@/components/ui/kibo-ui/ai/tool";
import { useCopyToClipboard } from "@/hooks/use-copy-to-clipboard";
import type { Attachment } from "@/stores/attachmentSlice";
import type { ToolCall } from "@/types/common";
import type { MediaOutput } from "@/types/media";
import type { TimelineEntry, UIMessage } from "@/types/message";
import { convertToAssetServerUrl } from "@/utils/assetServer";
import { getYouTubeEmbedUrl, isYouTubeUrl } from "@/utils/videoUrlDetection";
import { ConversationLoader } from "./conversation-loader";
import { MediaShowcase } from "./media-showcase";
import { MessageAttachmentDisplay } from "./message-attachment-display";
import { ModelDisplay } from "./model-display";
import { PlanDisplay } from "./plan-display";
import { ProviderDisplay } from "./provider-display";
import { RateLimitDisplay } from "./rate-limit-display";
import { ResponseRenderer } from "./response-renderer";
import { StatusUI } from "./status-ui";
import { TodoList } from "./todo-list";
import { CallbackResultDisplay } from "./callback-result-display";

type StreamingState = {
	processing: boolean;
	reasoning: string | null;
	reasoningDuration: number | null;
	toolCalls: ToolCall[];
	completed: boolean;
	cancelled: boolean;
	finalContent: string | null;
	error?: string | null;
	timeline?: TimelineEntry[];
	rateLimit?: {
		retryAfter: number;
		attempt: number;
		maxAttempts: number;
	};
	pendingUserMessage?: {
		text: string;
		attachments?: Attachment[];
	} | null;
	userMessageId: string | null;
	assistantMessageId: string | null;
	preStreamingMessageIds: Set<string>;
};

// Helper function to detect URLs
const isURL = (path: string): boolean => {
	return path.startsWith("http://") || path.startsWith("https://");
};

// Helper function to get media source URL
const getMediaSrc = (path: string, sessionId: string): string => {
	if (isURL(path)) {
		// For YouTube URLs, convert to embed format
		if (isYouTubeUrl(path)) {
			return getYouTubeEmbedUrl(path) || path;
		}
		return path;
	}
	return convertToAssetServerUrl(path, sessionId);
};

interface ConversationDisplayProps {
	messages: UIMessage[];
	sseStream: StreamingState;
	onPlanAction?: (
		action: "proceed" | "keep-planning",
		messageIndex: number,
	) => void;
	onEditMessage?: (index: number) => void;
	setUserMessageRef?: (index: number) => (el: HTMLDivElement | null) => void;
	sessionId?: string;
}

// Helper function to extract todos from TodoWrite tool calls
const extractTodosFromToolCalls = (toolCalls: ToolCall[]) => {
	const todoWriteCalls = toolCalls.filter((tc) => tc.name === "TodoWrite");
	if (todoWriteCalls.length === 0) return [];

	// Find the latest TodoWrite call with complete parameters to avoid flicker
	// When a new call starts streaming, it may not have parameters yet
	for (let i = todoWriteCalls.length - 1; i >= 0; i--) {
		const call = todoWriteCalls[i];
		try {
			const todos = call.parameters?.todos;
			if (Array.isArray(todos) && todos.length > 0) {
				return todos;
			}
		} catch {}
	}

	// Fallback: if no calls have parameters yet, return empty array
	return [];
};

// Helper function to extract plan content from ExitPlanMode tool calls
const extractPlanFromToolCalls = (toolCalls: ToolCall[]) => {
	const planTool = toolCalls.find((tc) => tc.name === "ExitPlanMode");
	if (!planTool) return "";

	try {
		const plan = planTool.parameters?.plan;
		return typeof plan === "string" ? plan : "";
	} catch {
		return "";
	}
};

// Helper function to check if a message contains ExitPlanMode tool call
const hasExitPlanModeTool = (toolCalls: ToolCall[]) => {
	return toolCalls?.some((tc) => tc.name === "ExitPlanMode");
};

// Helper function to filter out special tools (TodoWrite, ExitPlanMode, ShowMedia) from toolCalls
const filterNonSpecialTools = (toolCalls: ToolCall[]) => {
	return toolCalls.filter(
		(tc) => tc.name !== "TodoWrite" && tc.name !== "ExitPlanMode" && tc.name !== "ShowMedia",
	);
};

// Helper function to render timeline entries chronologically
const renderTimelineEntries = (
	timeline: TimelineEntry[],
	isNested = false,
	mediaOutputs?: MediaOutput[],
	sessionId?: string,
	getMediaSrc?: (path: string, sessionId: string) => string,
) => {
	if (!timeline || timeline.length === 0) return null;

	let entriesToRender = timeline;
	const nestedMap = new Map<string, TimelineEntry[]>();

	// Only group by parentToolCallId at the top level
	if (!isNested) {
		const topLevelEntries: TimelineEntry[] = [];

		for (const entry of timeline) {
			if ((entry as any).parentToolCallId) {
				// This is a subagent event - group under parent tool
				const nested = nestedMap.get((entry as any).parentToolCallId) || [];
				nested.push(entry);
				nestedMap.set((entry as any).parentToolCallId, nested);
			} else {
				// Top-level entry
				topLevelEntries.push(entry);
			}
		}

		entriesToRender = topLevelEntries;
	}

	// Group consecutive thinking entries
	const groupedEntries: Array<
		| { type: "thinking"; entries: string[]; timestamps: number[] }
		| { type: "tool"; entry: TimelineEntry; nestedEntries?: TimelineEntry[] }
		| { type: "content"; entry: TimelineEntry }
		| { type: "callback_result"; entry: TimelineEntry }
	> = [];

	for (const entry of entriesToRender) {
		if (entry.type === "thinking") {
			const lastGroup = groupedEntries[groupedEntries.length - 1];
			if (lastGroup && lastGroup.type === "thinking") {
				lastGroup.entries.push(entry.content);
				lastGroup.timestamps.push(entry.timestamp);
			} else {
				groupedEntries.push({
					type: "thinking",
					entries: [entry.content],
					timestamps: [entry.timestamp],
				});
			}
		} else if (entry.type === "tool") {
			const nested = nestedMap.get(entry.content.id);
			groupedEntries.push({
				type: "tool",
				entry,
				nestedEntries: nested,
			});
		} else if (entry.type === "callback_result") {
			groupedEntries.push({ type: "callback_result", entry });
		} else {
			groupedEntries.push({ type: "content", entry });
		}
	}

	return groupedEntries.map((group, _index) => {
		if (group.type === "thinking") {
			const totalContent = group.entries.join("");
			const duration =
				group.timestamps.length > 1
					? Math.round(
							(group.timestamps[group.timestamps.length - 1] -
								group.timestamps[0]) /
								1000,
						)
					: 0;

			return (
				<AIReasoning
					className="mb-4 w-full"
					duration={duration > 0 ? duration : undefined}
					isStreaming={false}
					key={`thinking-${group.timestamps[0]}`}
				>
					<AIReasoningTrigger />
					<AIReasoningContent>{totalContent}</AIReasoningContent>
				</AIReasoning>
			);
		}
		if (group.type === "content") {
			return (
				<div className="mb-4" key={`content-${group.entry.id}`}>
					<ResponseRenderer content={group.entry.content as string} />
				</div>
			);
		}
		if (group.type === "callback_result") {
			return (
				<CallbackResultDisplay
					key={`callback-${group.entry.id}`}
					result={group.entry.content as any}
				/>
			);
		}

		// Tool with potential nested subagent events
		const toolCall = group.entry.content as ToolCall;
		const hasNestedEvents = group.nestedEntries && group.nestedEntries.length > 0;

		// Special rendering for ShowMedia tool
		if (toolCall.name === "ShowMedia" && mediaOutputs && sessionId && getMediaSrc) {
			return (
				<div key={`media-showcase-${group.entry.id}`} className="mb-4">
					<MediaShowcase
						getMediaSrc={getMediaSrc}
						mediaOutputs={mediaOutputs}
						sessionId={sessionId}
					/>
				</div>
			);
		}

		return (
			<AIToolLadder key={`tool-${group.entry.id}`}>
				<AIToolStep isLast={true} status={toolCall.status} stepNumber={1}>
					<AIToolHeader
						description={toolCall.description}
						name={toolCall.name}
						status={toolCall.status}
						toolCall={toolCall}
					/>
					<AIToolContent toolCall={toolCall} />

					{/* Nested subagent events */}
					{hasNestedEvents && group.nestedEntries && (
						<div className="mt-4 ml-4 border-l-2 border-muted pl-4">
							{renderTimelineEntries(group.nestedEntries, true, mediaOutputs, sessionId, getMediaSrc)}
						</div>
					)}
				</AIToolStep>
			</AIToolLadder>
		);
	});
};

const MessageCopyButton = ({ content }: { content: string }) => {
	const { isCopied, copyToClipboard } = useCopyToClipboard();
	return (
		<Button
			className="text-muted-foreground hover:text-foreground"
			onClick={() => copyToClipboard(content)}
			size="sm"
			variant="ghost"
		>
			{isCopied ? <Check className="size-4" /> : <Copy className="size-4" />}
		</Button>
	);
};

export function ConversationDisplay({
	messages,
	sseStream,
	onPlanAction,
	onEditMessage,
	setUserMessageRef,
	sessionId,
}: ConversationDisplayProps) {
	const [showPlanOptions, setShowPlanOptions] = useState<number | null>(null);

	// Detect when a new message with ExitPlanMode is added and show plan options
	useEffect(() => {
		if (messages.length > 0) {
			const lastMessage = messages[messages.length - 1];
			if (
				lastMessage.from === "assistant" &&
				lastMessage.toolCalls &&
				hasExitPlanModeTool(lastMessage.toolCalls)
			) {
				setShowPlanOptions(messages.length - 1);
			}
		}
	}, [messages]);

	const handlePlanProceed = (messageIndex: number) => {
		setShowPlanOptions(null);
		onPlanAction?.("proceed", messageIndex);
	};

	const handlePlanKeepPlanning = (messageIndex: number) => {
		setShowPlanOptions(null);
		onPlanAction?.("keep-planning", messageIndex);
	};

	// Check if pending user message is already in stored messages to prevent duplicates
	const shouldShowPendingMessage = () => {
		if (!sseStream.pendingUserMessage) return false;

		// If there are no stored messages yet, show the pending message
		if (messages.length === 0) return true;

		// Prefer ID-based matching if we have a user message ID
		if (sseStream.userMessageId) {
			// Check if this message ID already exists in stored messages
			const messageExists = messages.some(
				(msg) => msg.from === "user" && msg.id === sseStream.userMessageId,
			);

			// If message exists in stored messages, don't show pending version
			if (messageExists) return false;
		}

		// Fallback to content-based matching for backward compatibility
		// Check the last few messages (up to 5) to see if pending message already exists
		const recentMessages = messages.slice(-5);
		const pendingText = sseStream.pendingUserMessage.text.trim();

		for (const msg of recentMessages) {
			if (msg.from === "user" && msg.content.trim() === pendingText) {
				// Message already exists in stored messages - don't show pending
				return false;
			}
		}

		// Pending message not found in recent stored messages - show it
		return true;
	};

	// Filter out any assistant messages that were created during current streaming session
	const getFilteredMessages = () => {
		// During streaming OR just after completion (while streaming UI is still showing),
		// filter out NEW assistant messages that appeared during this stream.
		// Keep filtering until streaming content is cleared to prevent flash/duplicates.
		// Keep messages that existed before streaming started (tracked in preStreamingMessageIds)

		// IMPORTANT: Only filter during ACTIVE streaming (processing=true)
		// After reload, completed=true but we want to show all messages from DB
		const shouldFilter =
			messages.length > 0 &&
			sseStream.preStreamingMessageIds &&
			sseStream.preStreamingMessageIds.size >= 0 &&
			(sseStream.timeline?.length || sseStream.toolCalls?.length || sseStream.finalContent) &&
			sseStream.processing; // ← Changed: Only filter during active streaming, not after completion

		if (shouldFilter) {
			const filtered = messages.filter((msg) => {
				// Keep all messages that existed before streaming started
				if (!msg.id) {
					console.warn(
						"Message missing ID during streaming filter - cannot determine if pre-existing",
						{ from: msg.from, content: msg.content?.substring(0, 50) },
					);
				}
				if (msg.id && sseStream.preStreamingMessageIds.has(msg.id)) {
					return true;
				}
				// For new messages, only keep user messages (filter out new assistant messages)
				// New assistant messages are being streamed via SSE and shouldn't show from DB
				return msg.from !== "assistant";
			});
			return filtered;
		}

		return messages;
	};

	const filteredMessages = getFilteredMessages();

	// Check if streaming assistant message is already in stored messages to prevent duplicates
	const shouldShowStreamingAssistant = () => {
		// First, check if we have an assistantMessageId and if it's already in FILTERED messages
		// This check must happen BEFORE the processing check to prevent duplicates on tab switch
		if (sseStream.assistantMessageId) {
			const messageExists = filteredMessages.some(
				(msg) => msg.id === sseStream.assistantMessageId,
			);

			// If message exists in filtered messages, don't show streaming version
			if (messageExists) return false;
		}

		// Show during active streaming (not yet completed)
		if (sseStream.processing && !sseStream.completed) return true;

		// Show streaming content if we have content and it's not in stored messages yet
		return (
			(sseStream.processing || sseStream.completed || sseStream.cancelled) &&
			(sseStream.finalContent ||
				sseStream.timeline?.length ||
				sseStream.toolCalls?.length)
		);
	};

	return (
		<div className="relative h-full flex-1 py-16">
			<div className="space-y-6">
				{filteredMessages.map((message, index) => {
					return (
						<AIMessage
							from={message.from}
							key={message.id}
							ref={
								message.from === "user" ? setUserMessageRef?.(index) : undefined
							}
						>
							<AIMessageContent>
								{message.from === "assistant" ? (
									<>
									{/* Render timeline-based interleaved thinking and tools (media rendered inline) */}
									{message.timeline && message.timeline.length > 0 ? (
										renderTimelineEntries(
											message.timeline,
											false,
											message.mediaOutputs,
											sessionId,
											getMediaSrc,
										)
									) : message.status ? (
										<AIMessageContent.Content>
											<StatusUI statusState={message.status} />
										</AIMessageContent.Content>
									) : message.provider ? (
										<AIMessageContent.Content>
											<ProviderDisplay data={message.provider} />
										</AIMessageContent.Content>
									) : message.model ? (
										<AIMessageContent.Content>
											<ModelDisplay data={message.model} />
										</AIMessageContent.Content>
									) : (
										<AIMessageContent.Content>
											<ResponseRenderer content={message.content} />
										</AIMessageContent.Content>
									)}
										{message.content && (
											<AIMessageContent.Toolbar>
												<MessageCopyButton content={message.content} />
											</AIMessageContent.Toolbar>
										)}
									</>
								) : (
									<>
										<AIMessageContent.Content>
											<MessageAttachmentDisplay
												attachments={message.attachments || []}
												sessionId={sessionId}
											/>
											{message.content}
										</AIMessageContent.Content>
										<AIMessageContent.Toolbar>
											<MessageCopyButton content={message.content} />
											{onEditMessage && (
												<Button
													aria-label="Rewind conversation to this message"
													className="text-muted-foreground hover:text-foreground"
													disabled={sseStream.processing}
													onClick={() => onEditMessage(index)}
													size="sm"
													title="Rewind to this message"
													variant="ghost"
												>
													<Undo2 className="size-4" />
												</Button>
											)}
										</AIMessageContent.Toolbar>
									</>
								)}
								{/* Render special tools (todos, plans) and legacy tools when timeline is not available */}
								{message.toolCalls && message.toolCalls.length > 0 && (
									<>
										{/* Render plan content */}
										{extractPlanFromToolCalls(message.toolCalls) && (
											<PlanDisplay
												onKeepPlanning={() => handlePlanKeepPlanning(index)}
												onProceed={() => handlePlanProceed(index)}
												planContent={extractPlanFromToolCalls(
													message.toolCalls,
												)}
												showOptions={showPlanOptions === index}
											/>
										)}
										{/* Render todos inline without tool wrapper */}
										{extractTodosFromToolCalls(message.toolCalls).length >
											0 && (
											<div className="mt-4">
												<TodoList
													todos={extractTodosFromToolCalls(message.toolCalls)}
												/>
											</div>
										)}
										{/* Render regular tool calls directly ONLY when timeline is not available or empty */}
										{(!message.timeline || message.timeline.length === 0) &&
											filterNonSpecialTools(message.toolCalls).map(
												(toolCall, index) => (
													<AIToolLadder
														key={`direct-tool-${toolCall.id}-${index}`}
													>
														<AIToolStep
															isLast={true}
															status={toolCall.status}
															stepNumber={1}
														>
															<AIToolHeader
																description={toolCall.description}
																name={toolCall.name}
																status={toolCall.status}
																toolCall={toolCall}
															/>
															<AIToolContent toolCall={toolCall} />
														</AIToolStep>
													</AIToolLadder>
												),
											)}
									</>
								)}
							</AIMessageContent>
						</AIMessage>
					);
				})}
				{/* Show optimistic user message during streaming - only if not already in stored messages */}
				{shouldShowPendingMessage() && sseStream.pendingUserMessage && (
					<AIMessage from="user">
						<AIMessageContent>
							<AIMessageContent.Content>
								{sseStream.pendingUserMessage.attachments &&
									sseStream.pendingUserMessage.attachments.length > 0 && (
										<MessageAttachmentDisplay
											attachments={sseStream.pendingUserMessage.attachments}
											sessionId={sessionId}
										/>
									)}
								{sseStream.pendingUserMessage.text}
							</AIMessageContent.Content>
						</AIMessageContent>
					</AIMessage>
				)}
				{shouldShowStreamingAssistant() ? (
					<AIMessage from="assistant">
						<AIMessageContent>
							{/* Show timeline-based interleaved thinking and tools during streaming */}
							{sseStream.timeline &&
								renderTimelineEntries(
									sseStream.timeline,
									false,
									sseStream.toolCalls?.find((tc) => tc.name === "ShowMedia")
										?.parameters?.outputs as MediaOutput[] | undefined,
									sessionId,
									getMediaSrc,
								)}
							{/* Show rate limit message when rate limiting is detected */}
							{sseStream.rateLimit ? (
								<div className="mt-4">
									<RateLimitDisplay
										attempt={sseStream.rateLimit.attempt}
										error={sseStream.error || undefined}
										maxAttempts={sseStream.rateLimit.maxAttempts}
										retryAfter={sseStream.rateLimit.retryAfter}
									/>
								</div>
							) : sseStream.toolCalls.length > 0 ? (
								<>
									{/* Render streaming todos inline without tool wrapper */}
									{extractTodosFromToolCalls(sseStream.toolCalls).length >
										0 && (
										<div className="mt-4">
											<TodoList
												todos={extractTodosFromToolCalls(sseStream.toolCalls)}
											/>
										</div>
									)}
									{/* Render streaming plan content */}
									{extractPlanFromToolCalls(sseStream.toolCalls) && (
										<PlanDisplay
											planContent={extractPlanFromToolCalls(
												sseStream.toolCalls,
											)}
											showOptions={false}
										/>
									)}
									{/* Render streaming regular tool calls directly ONLY when timeline is not available or empty */}
									{(!sseStream.timeline || sseStream.timeline.length === 0) &&
										filterNonSpecialTools(sseStream.toolCalls).map(
											(toolCall, index) => (
												<AIToolLadder
													key={`streaming-direct-tool-${toolCall.id}-${index}`}
												>
													<AIToolStep
														isLast={true}
														status={toolCall.status}
														stepNumber={1}
													>
														<AIToolHeader
															description={toolCall.description}
															name={toolCall.name}
															status={toolCall.status}
															toolCall={toolCall}
														/>
														<AIToolContent toolCall={toolCall} />
													</AIToolStep>
												</AIToolLadder>
											),
										)}
									{sseStream.cancelled ? (
										<div className="mt-4 text-muted-foreground">
											Execution paused
										</div>
									) : sseStream.processing && !sseStream.completed ? (
									<ConversationLoader />
									) : null}
								</>
							) : sseStream.cancelled ? (
								<div className="text-muted-foreground">Execution paused</div>
							) : sseStream.processing && !sseStream.completed ? (
								<ConversationLoader />
							) : null}
						</AIMessageContent>
					</AIMessage>
				) : null}
			</div>
		</div>
	);
}
