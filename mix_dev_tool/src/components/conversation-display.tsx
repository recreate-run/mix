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
import type { ToolCall } from "@/types/common";
import type { TimelineEntry, UIMessage } from "@/types/message";
import { convertToAssetServerUrl } from "@/utils/assetServer";
import { getYouTubeEmbedUrl, isYouTubeUrl } from "@/utils/videoUrlDetection";
import { ConversationLoader } from "./conversation-loader";
import { EmptyStateDisplay } from "./empty-state-display";
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
	onUpdateMessage?: (index: number, updatedMessage: UIMessage) => void;
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
const renderTimelineEntries = (timeline: TimelineEntry[]) => {
	if (!timeline || timeline.length === 0) return null;

	// Group consecutive thinking entries together for better UX
	const groupedEntries: Array<
		| { type: "thinking"; entries: string[]; timestamps: number[] }
		| { type: "tool"; entry: TimelineEntry }
		| { type: "content"; entry: TimelineEntry }
	> = [];

	for (const entry of timeline) {
		if (entry.type === "thinking") {
			const lastGroup = groupedEntries[groupedEntries.length - 1];
			if (lastGroup && lastGroup.type === "thinking") {
				// Add to existing thinking group
				lastGroup.entries.push(entry.content);
				lastGroup.timestamps.push(entry.timestamp);
			} else {
				// Start new thinking group
				groupedEntries.push({
					type: "thinking",
					entries: [entry.content],
					timestamps: [entry.timestamp],
				});
			}
		} else {
			// Tool or content entry - always separate
			groupedEntries.push({
				type: entry.type,
				entry,
			});
		}
	}

	return groupedEntries.map((group, index) => {
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
					key={`thinking-group-${index}`}
				>
					<AIReasoningTrigger />
					<AIReasoningContent>{totalContent}</AIReasoningContent>
				</AIReasoning>
			);
		}
		if (group.type === "content") {
			const contentText = group.entry.content as string;
			return (
				<div className="mb-4" key={`content-${group.entry.id}`}>
					<ResponseRenderer content={contentText} />
				</div>
			);
		}
		const toolCall = group.entry.content as ToolCall;
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
	onUpdateMessage,
	setUserMessageRef,
	sessionId,
}: ConversationDisplayProps) {
	const [showPlanOptions, setShowPlanOptions] = useState<number | null>(null);
	const [localMessages, setLocalMessages] = useState<UIMessage[]>(messages);

	// Deduplication: Check if streaming content already exists in persisted messages
	const shouldRenderStreamingUI = (() => {
		const baseCondition =
			(sseStream.processing && !sseStream.completed) ||
			(sseStream.cancelled &&
				(sseStream.finalContent ||
					sseStream.timeline?.length ||
					sseStream.toolCalls?.length));

		if (!baseCondition) {
			return false;
		}

		// If we're actively processing, always show
		if (sseStream.processing && !sseStream.completed) {
			return true;
		}

		// If cancelled, check if content already exists in persisted messages
		if (sseStream.cancelled && messages.length > 0) {
			const lastMessage = messages[messages.length - 1];
			// Check if last message is assistant and has similar timeline content
			if (
				lastMessage.from === "assistant" &&
				lastMessage.timeline?.length &&
				sseStream.timeline?.length
			) {
				// Content already persisted, don't show streaming UI
				console.log(
					"[StreamingDebug] Skipping streaming UI - content already in persisted messages",
				);
				return false;
			}
		}

		return true;
	})();

	// Debug: Track streaming UI render state
	useEffect(() => {
		console.log("[StreamingDebug] Streaming UI render decision:", {
			shouldRender: shouldRenderStreamingUI,
			processing: sseStream.processing,
			completed: sseStream.completed,
			cancelled: sseStream.cancelled,
			hasContent: !!sseStream.finalContent,
			timelineLength: sseStream.timeline?.length || 0,
			toolCallsLength: sseStream.toolCalls?.length || 0,
			persistedMessagesCount: messages.length,
		});
	}, [
		shouldRenderStreamingUI,
		sseStream.processing,
		sseStream.completed,
		sseStream.cancelled,
		sseStream.finalContent,
		sseStream.timeline?.length,
		sseStream.toolCalls?.length,
		messages.length,
	]);

	// Detect when a new message with exit_plan_mode is added and show plan options
	// Update localMessages when messages prop changes
	useEffect(() => {
		setLocalMessages(messages);
	}, [messages]);

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

	// Handle UI message updates from component responses
	const handleMessageUpdate = (index: number, updatedMessage: UIMessage) => {
		// Update local message state
		setLocalMessages((prev) => [
			...prev.slice(0, index),
			updatedMessage,
			...prev.slice(index + 1),
		]);

		// Pass update to parent component
		if (onUpdateMessage) {
			onUpdateMessage(index, updatedMessage);
		}
	};

	return (
		<div className="relative h-full flex-1 py-16">
			<div className="">
				{messages.length === 0 && <EmptyStateDisplay />}
				{localMessages.map((message, index) => {
					return (
						<AIMessage
							from={message.from}
							key={message.id || `frontend-${index}`}
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
													<ProviderDisplay
														data={message.provider}
														onUpdate={(updatedMessage: any) =>
															handleMessageUpdate(index, updatedMessage)
														}
													/>
												) : message.model ? (
													<ModelDisplay
														data={message.model}
														onUpdate={(updatedMessage: any) =>
															handleMessageUpdate(index, updatedMessage)
														}
													/>
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
				{shouldRenderStreamingUI ? (
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
									) : sseStream.completed ? null : (
										<ConversationLoader />
									)}
								</>
							) : sseStream.cancelled ? (
								<div className="text-muted-foreground">Execution paused</div>
							) : (
								<ConversationLoader />
							)}
						</AIMessageContent>
					</AIMessage>
				) : null}
			</div>
		</div>
	);
}
