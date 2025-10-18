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
	assistantMessageId: string | null;
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

// Helper function to extract todos from todo_write tool calls
const extractTodosFromToolCalls = (toolCalls: ToolCall[]) => {
	const todoWriteCalls = toolCalls.filter((tc) => tc.name === "todo_write");
	if (todoWriteCalls.length === 0) return [];

	// Find the latest todo_write call with complete parameters to avoid flicker
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

// Helper function to extract plan content from exit_plan_mode tool calls
const extractPlanFromToolCalls = (toolCalls: ToolCall[]) => {
	const planTool = toolCalls.find((tc) => tc.name === "exit_plan_mode");
	if (!planTool) return "";

	try {
		const plan = planTool.parameters?.plan;
		return typeof plan === "string" ? plan : "";
	} catch {
		return "";
	}
};

// Helper function to check if a message contains exit_plan_mode tool call
const hasExitPlanModeTool = (toolCalls: ToolCall[]) => {
	return toolCalls?.some((tc) => tc.name === "exit_plan_mode");
};

// Helper function to filter out special tools (todo_write, exit_plan_mode) from toolCalls
const filterNonSpecialTools = (toolCalls: ToolCall[]) => {
	return toolCalls.filter(
		(tc) => tc.name !== "todo_write" && tc.name !== "exit_plan_mode",
	);
};

// Helper function to render timeline entries chronologically
const renderTimelineEntries = (timeline: TimelineEntry[], isNested = false) => {
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

		// Tool with potential nested subagent events
		const toolCall = group.entry.content as ToolCall;
		const hasNestedEvents = group.nestedEntries && group.nestedEntries.length > 0;

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
							{renderTimelineEntries(group.nestedEntries, true)}
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

	// Detect when a new message with exit_plan_mode is added and show plan options
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

		// Check the last few messages (up to 3) to see if pending message already exists
		// This handles cases where user switches tabs during streaming and cache refetches
		const recentMessages = messages.slice(-3);
		const pendingText = sseStream.pendingUserMessage.text;

		for (const msg of recentMessages) {
			if (msg.from === "user" && msg.content === pendingText) {
				// Message already exists in stored messages - don't show pending
				return false;
			}
		}

		// Pending message not found in recent stored messages - show it
		return true;
	};

	// Check if streaming assistant message is already in stored messages to prevent duplicates
	const shouldShowStreamingAssistant = () => {
		// Always show during active streaming (not yet completed)
		if (sseStream.processing && !sseStream.completed) return true;

		// If streaming is completed and we have an assistantMessageId, check if it's in stored messages
		if (sseStream.completed && sseStream.assistantMessageId) {
			// Check if this message ID already exists in stored messages
			const messageExists = messages.some(
				(msg) => msg.id === sseStream.assistantMessageId,
			);

			// If message exists in stored messages, don't show streaming version
			if (messageExists) return false;
		}

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
				{messages.map((message, index) => {
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
										{/* Render media outputs as primary content */}
										{message.mediaOutputs && sessionId ? (
											<>
												<MediaShowcase
													getMediaSrc={getMediaSrc}
													mediaOutputs={message.mediaOutputs}
													sessionId={sessionId}
												/>
												<AIMessageContent.Content>
													{/* Render timeline-based interleaved thinking and tools */}
													{message.timeline &&
														renderTimelineEntries(message.timeline)}
												</AIMessageContent.Content>
											</>
										) : message.mediaOutputs ? (
											<>
												<div className="text-muted-foreground text-sm">
													Media content requires session ID
												</div>
												<AIMessageContent.Content>
													{/* Render timeline-based interleaved thinking and tools */}
													{message.timeline &&
														renderTimelineEntries(message.timeline)}
												</AIMessageContent.Content>
											</>
										) : (
											<AIMessageContent.Content>
												{/* Render timeline-based interleaved thinking and tools */}
												{message.timeline &&
													renderTimelineEntries(message.timeline)}
												{message.status ? (
													<StatusUI statusState={message.status} />
												) : message.provider ? (
													<ProviderDisplay data={message.provider} />
												) : message.model ? (
													<ModelDisplay data={message.model} />
												) : (
													<ResponseRenderer content={message.content} />
												)}
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
							{sseStream.timeline && renderTimelineEntries(sseStream.timeline)}
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
