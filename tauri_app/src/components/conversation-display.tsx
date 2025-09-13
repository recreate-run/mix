import { convertToAssetServerUrl } from '@/utils/assetServer';
import { isYouTubeUrl, getYouTubeEmbedUrl } from '@/utils/videoUrlDetection';
import type { ToolCall } from '@/types/common';

// Helper function to detect URLs
const isURL = (path: string): boolean => {
  return path.startsWith('http://') || path.startsWith('https://');
};

// Helper function to get media source URL
const getMediaSrc = (path: string, workingDirectory: string): string => {
  if (isURL(path)) {
    // For YouTube URLs, convert to embed format
    if (isYouTubeUrl(path)) {
      return getYouTubeEmbedUrl(path) || path;
    }
    return path;
  }
  return convertToAssetServerUrl(path, workingDirectory);
};
import { Check, Copy, Pencil } from 'lucide-react';
import { useEffect, useState } from 'react';
import { Button } from '@/components/ui/button';
import {
  AIMessage,
  AIMessageContent,
} from '@/components/ui/kibo-ui/ai/message';
import {
  AIReasoning,
  AIReasoningContent,
  AIReasoningTrigger,
} from '@/components/ui/kibo-ui/ai/reasoning';
import { AIResponse } from '@/components/ui/kibo-ui/ai/response';
import {
  AIToolContent,
  AIToolHeader,
  AIToolLadder,
  AIToolStep,
} from '@/components/ui/kibo-ui/ai/tool';
import { useCopyToClipboard } from '@/hooks/use-copy-to-clipboard';
import type { MediaOutput } from '@/types/media';
import type { TimelineEntry, UIMessage } from '@/types/message';
import { ConversationLoader } from './conversation-loader';
import { MessageAttachmentDisplay } from './message-attachment-display';
import { PlanDisplay } from './plan-display';
import { PlaylistSidebar } from './playlist-sidebar';
import { RateLimitDisplay } from './rate-limit-display';
import { GsapAnimationPreview } from './gsap/GsapAnimationPreview';
import { LazyVideoPlayer } from './LazyVideoPlayer';
import { ResponseRenderer } from './response-renderer';
import { TodoList } from './todo-list';

type StreamingState = {
  processing: boolean;
  reasoning: string | null;
  reasoningDuration: number | null;
  toolCalls: ToolCall[];
  completed: boolean;
  error?: string | null;
  timeline?: TimelineEntry[];
  rateLimit?: {
    retryAfter: number;
    attempt: number;
    maxAttempts: number;
  };
};

// Main Media Player Component
const MainMediaPlayer = ({ media, workingDirectory }: { media: MediaOutput; workingDirectory: string }) => {
  return (
    <div className="mb-2 space-y-2">
      <div>
        <h3 className="font-semibold">{media.title}</h3>
        {media.description && (
          <p className="mt-1 text-muted-foreground text-sm">
            {media.description}
          </p>
        )}
      </div>

      {media.type === 'image' && (
        <img
          alt={media.title}
          className="aspect-auto max-h-120  object-contain"
          onError={(e) => {
            e.currentTarget.style.display = 'none';
            const fallback = e.currentTarget
              .nextElementSibling as HTMLElement;
            if (fallback) fallback.style.display = 'block';
          }}
          src={getMediaSrc(media.path, workingDirectory)}
        />
      )}

      {media.type === 'video' && (
        <LazyVideoPlayer
          media={media}
          workingDirectory={workingDirectory}
        />
      )}

      {media.type === 'audio' && (
        <div className="rounded-md bg-stone-700/30 p-4">
          <audio
            className="w-full"
            controls
            onError={(e) => {
              e.currentTarget.style.display = 'none';
              const fallback = e.currentTarget
                .nextElementSibling as HTMLElement;
              if (fallback) fallback.style.display = 'block';
            }}
            preload="metadata"
            src={getMediaSrc(media.path, workingDirectory)}
          >
            Your browser does not support the audio tag.
          </audio>
          <div
            className="mt-2 text-center text-stone-400"
            style={{ display: 'none' }}
          >
            Failed to load audio: {media.path}
          </div>
        </div>
      )}

      {media.type === 'gsap_animation' && media.config && (
        <GsapAnimationPreview config={media.config as any} />
      )}

      {media.type === 'youtube' && (
        <div className="overflow-hidden rounded-md">
          <iframe
            src={getMediaSrc(media.path, workingDirectory)}
            title={media.title}
            frameBorder="0"
            allow="accelerometer; autoplay; clipboard-write; encrypted-media; gyroscope; picture-in-picture; web-share"
            referrerPolicy="strict-origin-when-cross-origin"
            allowFullScreen
            className="aspect-video w-full min-w-xl bg-black"
            onError={(e) => {
              e.currentTarget.style.display = 'none';
              const fallback = e.currentTarget
                .nextElementSibling as HTMLElement;
              if (fallback) fallback.style.display = 'block';
            }}
          />
          <div
            className="flex h-48 items-center justify-center bg-stone-700 text-stone-400"
            style={{ display: 'none' }}
          >
            Failed to load YouTube video: {media.path}
          </div>
        </div>
      )}
    </div>
  );
};

// Media Showcase Component
const MediaShowcase = ({ mediaOutputs, workingDirectory }: { mediaOutputs: MediaOutput[]; workingDirectory: string }) => {
  const [selectedIndex, setSelectedIndex] = useState(0);


  if (!mediaOutputs || mediaOutputs.length === 0) return null;

  // Single media file - show directly
  if (mediaOutputs.length === 1) {
    return <MainMediaPlayer media={mediaOutputs[0]} workingDirectory={workingDirectory} />;
  }

  // Multiple media files - show player + playlist
  return (
    <div className="space-y-4">
      <MainMediaPlayer media={mediaOutputs[selectedIndex]} workingDirectory={workingDirectory} />
      <PlaylistSidebar
        mediaOutputs={mediaOutputs}
        onSelect={setSelectedIndex}
        selectedIndex={selectedIndex}
        workingDirectory={workingDirectory}
      />
    </div>
  );
};

interface ConversationDisplayProps {
  messages: UIMessage[];
  sseStream: StreamingState;
  onPlanAction?: (
    action: 'proceed' | 'keep-planning',
    messageIndex: number
  ) => void;
  onForkMessage?: (index: number) => void;
  setUserMessageRef?: (index: number) => (el: HTMLDivElement | null) => void;
  workingDirectory?: string;
  renderStatusDisplay?: (message: UIMessage) => React.ReactNode;
}

// Helper function to extract todos from todo_write tool calls
const extractTodosFromToolCalls = (toolCalls: ToolCall[]) => {
  const todoWriteCalls = toolCalls.filter((tc) => tc.name === 'todo_write');
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
    } catch { }
  }

  // Fallback: if no calls have parameters yet, return empty array
  return [];
};

// Helper function to extract plan content from exit_plan_mode tool calls
const extractPlanFromToolCalls = (toolCalls: ToolCall[]) => {
  const planTool = toolCalls.find((tc) => tc.name === 'exit_plan_mode');
  if (!planTool) return '';

  try {
    const plan = planTool.parameters?.plan;
    return typeof plan === 'string' ? plan : '';
  } catch {
    return '';
  }
};

// Helper function to check if a message contains exit_plan_mode tool call
const hasExitPlanModeTool = (toolCalls: ToolCall[]) => {
  return toolCalls?.some((tc) => tc.name === 'exit_plan_mode');
};

// Helper function to filter out special tools (todo_write, exit_plan_mode) from toolCalls
const filterNonSpecialTools = (toolCalls: ToolCall[]) => {
  return toolCalls.filter(
    (tc) => tc.name !== 'todo_write' && tc.name !== 'exit_plan_mode'
  );
};

// Helper function to check if previous user message started with "!"
const isPreviousUserMessageCommand = (
  messages: UIMessage[],
  currentIndex: number
) => {
  for (let i = currentIndex - 1; i >= 0; i--) {
    if (messages[i].from === 'user') {
      return messages[i].content.trim().startsWith('!');
    }
  }
  return false;
};

// Helper function to render timeline entries chronologically
const renderTimelineEntries = (timeline: TimelineEntry[]) => {
  if (!timeline || timeline.length === 0) return null;

  // Group consecutive thinking entries together for better UX
  const groupedEntries: Array<{ type: 'thinking', entries: string[], timestamps: number[] } | { type: 'tool', entry: TimelineEntry }> = [];

  for (const entry of timeline) {
    if (entry.type === 'thinking') {
      const lastGroup = groupedEntries[groupedEntries.length - 1];
      if (lastGroup && lastGroup.type === 'thinking') {
        // Add to existing thinking group
        lastGroup.entries.push(entry.content);
        lastGroup.timestamps.push(entry.timestamp);
      } else {
        // Start new thinking group
        groupedEntries.push({
          type: 'thinking',
          entries: [entry.content],
          timestamps: [entry.timestamp]
        });
      }
    } else {
      // Tool entry - always separate
      groupedEntries.push({
        type: 'tool',
        entry
      });
    }
  }

  return groupedEntries.map((group, index) => {
    if (group.type === 'thinking') {
      const totalContent = group.entries.join('');
      const duration = group.timestamps.length > 1
        ? Math.round((group.timestamps[group.timestamps.length - 1] - group.timestamps[0]) / 1000)
        : 0;

      return (
        <AIReasoning
          key={`thinking-group-${index}`}
          className="mb-4 w-full"
          duration={duration > 0 ? duration : undefined}
          isStreaming={false}
        >
          <AIReasoningTrigger />
          <AIReasoningContent>{totalContent}</AIReasoningContent>
        </AIReasoning>
      );
    } else {
      const toolCall = group.entry.content as ToolCall;
      return (
        <AIToolLadder key={`tool-${group.entry.id}`}>
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
      );
    }
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
  onForkMessage,
  setUserMessageRef,
  workingDirectory,
}: ConversationDisplayProps) {
  
  const [showPlanOptions, setShowPlanOptions] = useState<number | null>(null);

  // Detect when a new message with exit_plan_mode is added and show plan options
  useEffect(() => {
    if (messages.length > 0) {
      const lastMessage = messages[messages.length - 1];
      if (
        lastMessage.from === 'assistant' &&
        lastMessage.toolCalls &&
        hasExitPlanModeTool(lastMessage.toolCalls)
      ) {
        setShowPlanOptions(messages.length - 1);
      }
    }
  }, [messages]);

  const handlePlanProceed = (messageIndex: number) => {
    setShowPlanOptions(null);
    onPlanAction?.('proceed', messageIndex);
  };

  const handlePlanKeepPlanning = (messageIndex: number) => {
    setShowPlanOptions(null);
    onPlanAction?.('keep-planning', messageIndex);
  };
  return (
    <div className="relative h-full flex-1 py-16">
      <div className="">
        {messages.length === 0 && (
          <AIMessage from="assistant">
            <AIMessageContent>
              <AIMessageContent.Content>
                Hello! I'm Mix, you AI agent for multimodal workflows. How can I
                help you today?
              </AIMessageContent.Content>
            </AIMessageContent>
          </AIMessage>
        )}
        {messages.map((message, index) => {
          return (
            <AIMessage
              from={message.from}
              key={index}
              ref={
                message.from === 'user' ? setUserMessageRef?.(index) : undefined
              }
            >
            <AIMessageContent>
              {message.from === 'assistant' ? (
                <>
                  {/* Render media outputs as primary content */}
                  {message.mediaOutputs && workingDirectory ? (
                    <MediaShowcase mediaOutputs={message.mediaOutputs} workingDirectory={workingDirectory} />
                  ) : message.mediaOutputs ? (
                    <div className="text-sm text-muted-foreground">Media content requires working directory</div>
                  ) : (
                    <AIMessageContent.Content>
                      {/* Render timeline-based interleaved thinking and tools */}
                      {message.timeline && renderTimelineEntries(message.timeline)}
                      {isPreviousUserMessageCommand(messages, index) ? (
                        <AIResponse>{`\`\`\`bash\n${message.content}\n\`\`\``}</AIResponse>
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
                      workingDirectory={workingDirectory}
                    />
                    {message.content}
                  </AIMessageContent.Content>
                  <AIMessageContent.Toolbar>
                    <MessageCopyButton content={message.content} />
                    {onForkMessage && (
                      <Button
                        className="text-muted-foreground hover:text-foreground"
                        disabled={sseStream.processing}
                        onClick={() => onForkMessage(index)}
                        size="sm"
                        variant="ghost"
                      >
                        <Pencil className="size-4" />
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
                      planContent={extractPlanFromToolCalls(message.toolCalls)}
                      showOptions={showPlanOptions === index}
                    />
                  )}
                  {/* Render todos inline without tool wrapper */}
                  {extractTodosFromToolCalls(message.toolCalls).length > 0 && (
                    <div className="mt-4">
                      <TodoList
                        todos={extractTodosFromToolCalls(message.toolCalls)}
                      />
                    </div>
                  )}
                  {/* Render regular tool calls directly ONLY when timeline is not available or empty */}
                  {(!message.timeline || message.timeline.length === 0) && filterNonSpecialTools(message.toolCalls).map((toolCall, index) => (
                    <AIToolLadder key={`direct-tool-${toolCall.id}-${index}`}>
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
                  ))}
                </>
              )}
            </AIMessageContent>
          </AIMessage>
          );
        })}
        {sseStream.processing && (
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
                        sseStream.toolCalls
                      )}
                      showOptions={false}
                    />
                  )}
                  {/* Render streaming regular tool calls directly ONLY when timeline is not available or empty */}
                  {(!sseStream.timeline || sseStream.timeline.length === 0) && filterNonSpecialTools(sseStream.toolCalls).map((toolCall, index) => (
                    <AIToolLadder key={`streaming-direct-tool-${toolCall.id}-${index}`}>
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
                  ))}
                  {!sseStream.completed && <ConversationLoader />}
                </>
              ) : (
                <ConversationLoader />
              )}
            </AIMessageContent>
          </AIMessage>
        )}
      </div>
    </div>
  );
}
