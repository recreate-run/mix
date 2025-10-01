import { convertToAssetServerUrl } from '@/utils/assetServer';
import { isYouTubeUrl, getYouTubeEmbedUrl } from '@/utils/videoUrlDetection';
import type { ToolCall } from '@/types/common';


// Helper function to detect URLs
const isURL = (path: string): boolean => {
  return path.startsWith('http://') || path.startsWith('https://');
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
import { Check, Copy, Download, Undo2 } from 'lucide-react';
import { useEffect, useState } from 'react';
import { Button } from '@/components/ui/button';
import { toast } from 'sonner';
import {
  AIMessage,
  AIMessageContent,
} from '@/components/ui/kibo-ui/ai/message';
import {
  AIReasoning,
  AIReasoningContent,
  AIReasoningTrigger,
} from '@/components/ui/kibo-ui/ai/reasoning';
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
import { CsvViewer } from './CsvViewer';
import { TodoList } from './todo-list';
import { StatusUI } from './status-ui';
import { ProviderDisplay } from './provider-display';
import { ModelDisplay } from './model-display';
import { EmptyStateDisplay } from './empty-state-display';

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

// Helper to get file extension from media type and path
const getFileExtension = (media: MediaOutput): string => {
  // Try to extract extension from path first
  const pathMatch = media.path.match(/\.([^./?#]+)(?:[?#]|$)/);
  if (pathMatch) {
    return `.${pathMatch[1]}`;
  }

  // Fallback to media type
  const extensionMap: Record<string, string> = {
    image: '.jpg',
    video: '.mp4',
    audio: '.mp3',
    pdf: '.pdf',
    csv: '.csv',
    gsap_animation: '.json',
  };

  return extensionMap[media.type] || '';
};

// Media Download Button Component
const MediaDownloadButton = ({ media, sessionId }: { media: MediaOutput; sessionId: string }) => {
  const [isDownloading, setIsDownloading] = useState(false);

  const handleDownload = async () => {
    // For YouTube videos, open in new tab instead of downloading
    if (media.type === 'video' && isYouTubeUrl(media.path)) {
      window.open(media.path, '_blank');
      return;
    }

    setIsDownloading(true);

    try {
      // For GSAP animations, download the config as JSON
      if (media.type === 'gsap_animation' && media.config) {
        const configJson = JSON.stringify(media.config, null, 2);
        const blob = new Blob([configJson], { type: 'application/json' });
        const url = URL.createObjectURL(blob);
        const a = document.createElement('a');
        a.href = url;
        const filename = `${media.title || 'animation'}.json`;
        a.download = filename;
        document.body.appendChild(a);
        a.click();
        document.body.removeChild(a);
        URL.revokeObjectURL(url);

        toast.success('Download complete', {
          description: filename,
        });

        setIsDownloading(false);
        return;
      }

      // For all other media types, fetch as blob and trigger download
      const mediaUrl = getMediaSrc(media.path, sessionId);
      const response = await fetch(mediaUrl);

      if (!response.ok) {
        throw new Error(`Failed to fetch media: ${response.statusText}`);
      }

      const blob = await response.blob();
      const url = URL.createObjectURL(blob);
      const a = document.createElement('a');
      a.href = url;

      // Create filename with proper extension
      const extension = getFileExtension(media);
      const filename = media.title
        ? (media.title.includes('.') ? media.title : `${media.title}${extension}`)
        : `media${extension}`;

      a.download = filename;
      document.body.appendChild(a);
      a.click();
      document.body.removeChild(a);
      URL.revokeObjectURL(url);

      toast.success('Download complete', {
        description: filename,
      });
    } catch (error) {
      console.error('Download failed:', error);
      toast.error('Download failed', {
        description: 'Opening in new tab instead',
      });
      // Fallback: try opening in new tab
      const mediaUrl = getMediaSrc(media.path, sessionId);
      window.open(mediaUrl, '_blank');
    } finally {
      setIsDownloading(false);
    }
  };

  return (
    <Button
      className="text-muted-foreground hover:text-foreground"
      onClick={handleDownload}
      size="sm"
      variant="ghost"
      title={media.type === 'video' && isYouTubeUrl(media.path) ? 'Open in YouTube' : 'Download media'}
      disabled={isDownloading}
    >
      <Download className="size-4" />
    </Button>
  );
};

// Main Media Player Component
const MainMediaPlayer = ({ media, sessionId }: { media: MediaOutput; sessionId: string }) => {
  return (
    <div className="mb-2 space-y-2">
      <div className="flex items-start justify-between gap-2">
        <div className="flex-1">
          <h3 className="font-semibold">{media.title}</h3>
          {media.description && (
            <p className="mt-1 text-muted-foreground text-sm">
              {media.description}
            </p>
          )}
        </div>
        <MediaDownloadButton media={media} sessionId={sessionId} />
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
          src={getMediaSrc(media.path, sessionId)}
        />
      )}

      {media.type === 'video' && (
        <>
          {isYouTubeUrl(media.path) ? (
            <div className="overflow-hidden rounded-md">
              <iframe
                src={(() => {
                  const baseUrl = getMediaSrc(media.path, sessionId);
                  if (media.startTime !== undefined || media.duration !== undefined) {
                    try {
                      const url = new URL(baseUrl);
                      if (media.startTime !== undefined) {
                        url.searchParams.set('start', media.startTime.toString());
                      }
                      if (media.duration !== undefined && media.startTime !== undefined) {
                        url.searchParams.set('end', (media.startTime + media.duration).toString());
                      }
                      return url.toString();
                    } catch {
                      return baseUrl;
                    }
                  }
                  return baseUrl;
                })()}
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
          ) : (
            <LazyVideoPlayer
              media={media}
              sessionId={sessionId}
            />
          )}
        </>
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
            src={getMediaSrc(media.path, sessionId)}
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

      {media.type === 'pdf' && (
        <div className="overflow-hidden rounded-md">
          <iframe
            src={getMediaSrc(media.path, sessionId)}
            title={media.title}
            frameBorder="0"
            className="aspect-[4/5] w-full min-w-xl bg-white"
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
            Failed to load PDF document: {media.path}
          </div>
        </div>
      )}

      {media.type === 'csv' && (
        <CsvViewer
          url={getMediaSrc(media.path, sessionId)}
          title={media.title}
        />
      )}
    </div>
  );
};

// Media Showcase Component
const MediaShowcase = ({ mediaOutputs, sessionId }: { mediaOutputs: MediaOutput[]; sessionId: string }) => {
  const [selectedIndex, setSelectedIndex] = useState(0);


  if (!mediaOutputs || mediaOutputs.length === 0) return null;

  // Single media file - show directly
  if (mediaOutputs.length === 1) {
    return <MainMediaPlayer media={mediaOutputs[0]} sessionId={sessionId} />;
  }

  // Multiple media files - show player + playlist
  return (
    <div className="space-y-4">
      <MainMediaPlayer media={mediaOutputs[selectedIndex]} sessionId={sessionId} />
      <PlaylistSidebar
        mediaOutputs={mediaOutputs}
        onSelect={setSelectedIndex}
        selectedIndex={selectedIndex}
        sessionId={sessionId}
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
  onEditMessage?: (index: number) => void;
  onUpdateMessage?: (index: number, updatedMessage: UIMessage) => void;
  setUserMessageRef?: (index: number) => (el: HTMLDivElement | null) => void;
  sessionId?: string;
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

// Helper function to render timeline entries chronologically
const renderTimelineEntries = (timeline: TimelineEntry[]) => {
  if (!timeline || timeline.length === 0) return null;

  // Group consecutive thinking entries together for better UX
  const groupedEntries: Array<
    | { type: 'thinking', entries: string[], timestamps: number[] }
    | { type: 'tool', entry: TimelineEntry }
    | { type: 'content', entry: TimelineEntry }
  > = [];

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
      // Tool or content entry - always separate
      groupedEntries.push({
        type: entry.type,
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
    } else if (group.type === 'content') {
      const contentText = group.entry.content as string;
      return (
        <div key={`content-${group.entry.id}`} className="mb-4">
          <ResponseRenderer content={contentText} />
        </div>
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
  onEditMessage,
  onUpdateMessage,
  setUserMessageRef,
  sessionId,
}: ConversationDisplayProps) {

  const [showPlanOptions, setShowPlanOptions] = useState<number | null>(null);
  const [localMessages, setLocalMessages] = useState<UIMessage[]>(messages);

  // Detect when a new message with exit_plan_mode is added and show plan options
  // Update localMessages when messages prop changes
  useEffect(() => {
    setLocalMessages(messages);
  }, [messages]);

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

  // Handle UI message updates from component responses
  const handleMessageUpdate = (index: number, updatedMessage: UIMessage) => {
    // Update local message state
    setLocalMessages((prev) => [
      ...prev.slice(0, index),
      updatedMessage,
      ...prev.slice(index + 1)
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
              key={index}
              ref={
                message.from === 'user' ? setUserMessageRef?.(index) : undefined
              }
            >
              <AIMessageContent>
                {message.from === 'assistant' ? (
                  <>
                    {/* Render media outputs as primary content */}
                    {message.mediaOutputs && sessionId ? (
                      <>
                        <MediaShowcase mediaOutputs={message.mediaOutputs} sessionId={sessionId} />
                        <AIMessageContent.Content>
                          {/* Render timeline-based interleaved thinking and tools */}
                          {message.timeline && renderTimelineEntries(message.timeline)}
                        </AIMessageContent.Content>
                      </>
                    ) : message.mediaOutputs ? (
                      <>
                        <div className="text-sm text-muted-foreground">Media content requires session ID</div>
                        <AIMessageContent.Content>
                          {/* Render timeline-based interleaved thinking and tools */}
                          {message.timeline && renderTimelineEntries(message.timeline)}
                        </AIMessageContent.Content>
                      </>
                    ) : (
                      <AIMessageContent.Content>
                        {/* Render timeline-based interleaved thinking and tools */}
                        {message.timeline && renderTimelineEntries(message.timeline)}
                        {message.status ? (
                          <StatusUI
                            statusState={message.status}
                          />
                        ) : message.provider ? (
                          <ProviderDisplay
                            data={message.provider}
                            onUpdate={(updatedMessage: any) => handleMessageUpdate(index, updatedMessage)}
                          />
                        ) : message.model ? (
                          <ModelDisplay
                            data={message.model}
                            onUpdate={(updatedMessage: any) => handleMessageUpdate(index, updatedMessage)}
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
                          className="text-muted-foreground hover:text-foreground"
                          disabled={sseStream.processing}
                          onClick={() => onEditMessage(index)}
                          size="sm"
                          variant="ghost"
                          title="Rewind to this message"
                          aria-label="Rewind conversation to this message"
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
        {(sseStream.processing && !sseStream.completed) || (sseStream.cancelled && (sseStream.finalContent || sseStream.timeline?.length || sseStream.toolCalls?.length)) ? (
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
                  {sseStream.cancelled ? (
                    <div className="mt-4 text-muted-foreground">
                      Execution paused
                    </div>
                  ) : !sseStream.completed ? (
                    <ConversationLoader />
                  ) : null}
                </>
              ) : sseStream.cancelled ? (
                <div className="text-muted-foreground">
                  Execution paused
                </div>
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