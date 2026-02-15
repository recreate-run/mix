import { Check, Copy, Undo2 } from "lucide-react";
import { CoreToolName } from "mix-typescript-sdk/models";
import { useEffect, useMemo, useState } from "react";
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
import type { MediaOutput } from "@/types/media";
import type { TimelineEntry, UIMessage } from "@/types/message";
import { convertToAssetServerUrl } from "@/utils/assetServer";
import { getYouTubeEmbedUrl, isYouTubeUrl } from "@/utils/videoUrlDetection";
import { CallbackResultDisplay } from "./callback-result-display";
import { ConversationLoader } from "./conversation-loader";
import { JSONDisplay } from "./json-display";
import { MediaShowcase } from "./media-showcase";
import { MessageAttachmentDisplay } from "./message-attachment-display";
import { ModelDisplay } from "./model-display";
import { PlanDisplay } from "./plan-display";
import { ProviderDisplay } from "./provider-display";
import { ResponseRenderer } from "./response-renderer";
import { StatusUI } from "./status-ui";
import { TodoList } from "./todo-list";

const isURL = (path: string): boolean => {
	return path.startsWith("http://") || path.startsWith("https://");
};

const getMediaSrc = (path: string, sessionId: string): string => {
	if (isURL(path)) {
		if (isYouTubeUrl(path)) {
			return getYouTubeEmbedUrl(path) || path;
		}
		return path;
	}
	return convertToAssetServerUrl(path, sessionId);
};

interface ConversationDisplayProps {
	messages: UIMessage[];
	onPlanAction?: (
		action: "proceed" | "keep-planning",
		messageIndex: number,
	) => void;
	onEditMessage?: (index: number) => void;
	sessionId?: string;
	disableEdit?: boolean;
}

const extractTodosFromToolCalls = (toolCalls: ToolCall[]) => {
	const todoWriteCalls = toolCalls.filter(
		(tc) => tc.name === CoreToolName.TodoWrite,
	);
	if (todoWriteCalls.length === 0) return [];

	for (let i = todoWriteCalls.length - 1; i >= 0; i--) {
		const call = todoWriteCalls[i];
		try {
			const todos = call.parameters?.todos;
			if (Array.isArray(todos) && todos.length > 0) {
				return todos;
			}
		} catch {}
	}

	return [];
};

const extractPlanFromToolCalls = (toolCalls: ToolCall[]) => {
	const planTool = toolCalls.find(
		(tc) => tc.name === CoreToolName.ExitPlanMode,
	);
	if (!planTool) return "";

	try {
		const plan = planTool.parameters?.plan;
		return typeof plan === "string" ? plan : "";
	} catch {
		return "";
	}
};

const hasExitPlanModeTool = (toolCalls: ToolCall[]) => {
	return toolCalls?.some((tc) => tc.name === CoreToolName.ExitPlanMode);
};

const filterNonSpecialTools = (toolCalls: ToolCall[]) => {
	return toolCalls.filter(
		(tc) =>
			tc.name !== CoreToolName.TodoWrite &&
			tc.name !== CoreToolName.ExitPlanMode &&
			tc.name !== CoreToolName.Show,
	);
};

const renderTimelineEntries = (
	timeline: TimelineEntry[],
	isNested = false,
	sessionId?: string,
) => {
	if (!timeline || timeline.length === 0) return null;

	let entriesToRender = timeline;
	const nestedMap = new Map<string, TimelineEntry[]>();

	if (!isNested) {
		const topLevelEntries: TimelineEntry[] = [];
		for (const entry of timeline) {
			if (entry.parentToolCallId) {
				const nested = nestedMap.get(entry.parentToolCallId) || [];
				nested.push(entry);
				nestedMap.set(entry.parentToolCallId, nested);
			} else {
				topLevelEntries.push(entry);
			}
		}
		entriesToRender = topLevelEntries;
	}

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
					key={`thinking-${group.timestamps[0]}-${index}`}
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
			if (group.entry.type !== "callback_result") {
				return null;
			}
			return (
				<CallbackResultDisplay
					key={`callback-${group.entry.id}`}
					result={group.entry.content}
				/>
			);
		}

		const toolCall = group.entry.content as ToolCall;
		const hasNestedEvents =
			group.nestedEntries && group.nestedEntries.length > 0;

		if (toolCall.name === CoreToolName.Show && sessionId) {
			const outputs = toolCall.parameters?.outputs as MediaOutput[] | undefined;
			if (outputs && outputs.length > 0) {
				const statusOutputs = outputs.filter((o) => o.type === "status");
				const jsonOutputs = outputs.filter((o) => o.type === "json");
				const mediaOutputs = outputs.filter(
					(o) => o.type !== "status" && o.type !== "json",
				);

				return (
					<div key={`show-${group.entry.id}`} className="mb-4">
						{statusOutputs.map((output, idx) => (
							<div
								key={`status-${group.entry.id}-${idx}`}
								className="mb-4 rounded-lg border bg-muted/50 p-4"
							>
								<div className="mb-2 font-semibold">{output.title}</div>
								<div className="text-sm">{output.data}</div>
							</div>
						))}
						{jsonOutputs.map((output, idx) => (
							<JSONDisplay
								key={`json-${group.entry.id}-${idx}`}
								data={output.data || "{}"}
								title={output.title}
							/>
						))}
						{mediaOutputs.length > 0 && (
							<MediaShowcase
								getMediaSrc={getMediaSrc}
								mediaOutputs={mediaOutputs}
								sessionId={sessionId}
							/>
						)}
					</div>
				);
			}
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
					{hasNestedEvents && group.nestedEntries && (
						<div className="mt-4 ml-4 border-l-2 border-muted pl-4">
							{renderTimelineEntries(group.nestedEntries, true, sessionId)}
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

const isEmptyMessage = (message: UIMessage) => {
	if (message.suppressChatMessage) return true;
	if (message.isStreaming) return false;
	const hasContent = message.content?.trim().length > 0;
	const hasTools = !!message.toolCalls?.length;
	const hasTimeline = !!message.timeline?.length;
	const hasAttachments = !!message.attachments?.length;
	const hasSpecialBlocks =
		!!message.status ||
		!!message.provider ||
		!!message.model ||
		!!message.hierarchicalModel;
	return (
		!hasContent &&
		!hasTools &&
		!hasTimeline &&
		!hasAttachments &&
		!hasSpecialBlocks
	);
};

export function ConversationDisplay({
	messages,
	onPlanAction,
	onEditMessage,
	sessionId,
	disableEdit = false,
}: ConversationDisplayProps) {
	const [showPlanOptions, setShowPlanOptions] = useState<number | null>(null);

	const filteredMessages = useMemo(() => {
		return messages.filter((msg) => !isEmptyMessage(msg));
	}, [messages]);

	useEffect(() => {
		if (filteredMessages.length > 0) {
			const lastMessage = filteredMessages[filteredMessages.length - 1];
			if (
				lastMessage.from === "assistant" &&
				lastMessage.toolCalls &&
				hasExitPlanModeTool(lastMessage.toolCalls)
			) {
				setShowPlanOptions(filteredMessages.length - 1);
			}
		}
	}, [filteredMessages]);

	const handlePlanProceed = (messageIndex: number) => {
		setShowPlanOptions(null);
		onPlanAction?.("proceed", messageIndex);
	};

	const handlePlanKeepPlanning = (messageIndex: number) => {
		setShowPlanOptions(null);
		onPlanAction?.("keep-planning", messageIndex);
	};

	return (
		<div className="relative h-full flex-1 py-16">
			<div className="space-y-6">
				{filteredMessages.map((message, index) => {
					return (
						<AIMessage
							from={message.from}
							key={message.id || `message-${index}`}
						>
							<AIMessageContent>
								{message.from === "assistant" ? (
									<>
										{message.timeline && message.timeline.length > 0 ? (
											renderTimelineEntries(message.timeline, false, sessionId)
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
											message.content?.trim() && (
												<AIMessageContent.Content>
													<ResponseRenderer content={message.content} />
												</AIMessageContent.Content>
											)
										)}

										{message.toolCalls && message.toolCalls.length > 0 && (
											<>
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
												{extractTodosFromToolCalls(message.toolCalls).length >
													0 && (
													<div className="mt-4">
														<TodoList
															todos={extractTodosFromToolCalls(
																message.toolCalls,
															)}
														/>
													</div>
												)}
												{(!message.timeline || message.timeline.length === 0) &&
													filterNonSpecialTools(message.toolCalls).map(
														(toolCall, toolIndex) => (
															<AIToolLadder
																key={`direct-tool-${toolCall.id}-${toolIndex}`}
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

										{message.isStreaming && (
											<div className="mt-2">
												<ConversationLoader />
											</div>
										)}
										{message.streamingStatus === "cancelled" && (
											<div className="mt-2 text-muted-foreground">
												Execution paused
											</div>
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
													disabled={disableEdit}
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
							</AIMessageContent>
						</AIMessage>
					);
				})}
			</div>
		</div>
	);
}
