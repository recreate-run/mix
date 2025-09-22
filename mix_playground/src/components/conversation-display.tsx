import { convertToAssetServerUrl } from '@/utils/assetServer';
import { isYouTubeUrl, getYouTubeEmbedUrl } from '@/utils/videoUrlDetection';
import type { ToolCall } from '@/types/common';
import { DotFlow } from '../../components/gsap/dot-flow';

// Commands for the opening screen animation - Mix-specific creative workflow examples
const commands = [
  {
    title: "Create a marketing video from these screenshots",
    frames: [
      [21, 22, 23, 24, 25, 26, 27], // Top row formation
      [14, 15, 16, 28, 29, 30, 34, 35], // Expanding down
      [7, 8, 9, 21, 22, 23, 37, 38, 39], // Cross pattern
      [0, 1, 2, 14, 15, 16, 28, 29, 30, 42, 43, 44], // Full grid formation
      [7, 8, 9, 21, 22, 23, 35, 36, 37], // Contracting
      [14, 15, 16, 28, 29, 30, 34], // Final form
      [21, 22, 23, 24, 25, 26, 27], // Return to top
    ],
    duration: 140,
    repeatCount: 1,
  },
  {
    title: "Analyze this user session recording",
    frames: [
      [0, 7, 14, 21, 28, 35, 42], // Scanning vertically
      [1, 8, 15, 22, 29, 36, 43],
      [2, 9, 16, 23, 30, 37, 44],
      [3, 10, 17, 24, 31, 38, 45],
      [4, 11, 18, 25, 32, 39, 46],
      [5, 12, 19, 26, 33, 40, 47],
      [6, 13, 20, 27, 34, 41, 48],
      [10, 11, 12, 17, 18, 19, 24, 25, 26, 31, 32, 33], // Highlighting center analysis
    ],
    duration: 110,
    repeatCount: 1,
  },
  {
    title: "Edit this video: trim and add title overlay",
    frames: [
      [0, 1, 2, 3, 4, 5, 6], // Timeline frames
      [7, 8, 9, 10, 11, 12, 13],
      [14, 15, 16, 17, 18, 19, 20],
      [21, 22, 23, 24, 25, 26, 27], // Middle section highlight
      [14, 15, 16, 17, 18, 19, 20], // Trim back
      [7, 8, 9, 10, 11, 12, 13],
      [21, 22, 23, 24, 25, 26, 27], // Final edited section
    ],
    duration: 160,
    repeatCount: 1,
  },
  {
    title: "Generate storyboard frames for concept",
    frames: [
      [10, 17, 24, 31], // Four corner frames
      [9, 10, 11, 16, 17, 18, 23, 24, 25, 30, 31, 32], // Expanding frames
      [2, 3, 4, 9, 10, 11, 16, 17, 18, 23, 24, 25, 30, 31, 32, 37, 38, 39], // Full storyboard
      [10, 11, 17, 18, 24, 25, 31, 32], // Refined frames
      [17, 24], // Key frames
      [10, 17, 24, 31], // Final four frames
    ],
    duration: 180,
    repeatCount: 1,
  },
  {
    title: "Process batch images: resize and watermark",
    frames: [
      [0, 2, 4, 6], // Scattered images
      [7, 9, 11, 13],
      [14, 16, 18, 20],
      [21, 23, 25, 27], // Processing wave
      [28, 30, 32, 34],
      [35, 37, 39, 41],
      [42, 44, 46, 48],
      [0, 2, 4, 6, 42, 44, 46, 48], // Before and after
    ],
    duration: 130,
    repeatCount: 1,
  },
];

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
import { LoginUI } from './login-ui';
import { LogoutUI } from './logout-ui';
import { StatusUI } from './status-ui';
import { ProviderDisplay } from './provider-display';
import { ModelDisplay } from './model-display';
import { ErrorBoundary } from './error-boundary';

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

// Main Media Player Component
const MainMediaPlayer = ({ media, sessionId }: { media: MediaOutput; sessionId: string }) => {
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
          src={getMediaSrc(media.path, sessionId)}
        />
      )}

      {media.type === 'video' && (
        <LazyVideoPlayer
          media={media}
          sessionId={sessionId}
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

      {media.type === 'youtube' && (
        <div className="overflow-hidden rounded-md">
          <iframe
            src={getMediaSrc(media.path, sessionId)}
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
  onForkMessage?: (index: number) => void;
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
  onForkMessage,
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
        {messages.length === 0 && (
          <div className="flex flex-col items-center justify-center min-h-[60vh] space-y-12 animate-in fade-in duration-700">
            {/* Mix Logo with gradient */}
            <div className="text-center space-y-4">
              <div className="text-6xl font-bold bg-gradient-to-r from-blue-500 via-purple-500 to-pink-500 bg-clip-text text-transparent mb-6 tracking-tight animate-in slide-in-from-top duration-1000">
                mix
              </div>
              <div className="text-xl text-muted-foreground animate-in slide-in-from-top duration-1000 delay-200">
                AI-powered creative workflows
              </div>
            </div>

            {/* Animated Commands with enhanced styling */}
            <div className="relative animate-in slide-in-from-bottom duration-1000 delay-500">
              <div className="absolute -inset-4 bg-gradient-to-r from-blue-500/10 via-purple-500/10 to-pink-500/10 rounded-xl blur-xl"></div>
              <div className="relative bg-card/50 backdrop-blur-sm border border-border/50 rounded-xl p-6 shadow-2xl [&_.dot-flow-container]:text-foreground [&_.dot-loader_.h-1\\.5]:bg-muted/30 [&_.dot-loader_.active]:bg-primary">
                <DotFlow items={commands} isPlaying={true} />
              </div>
            </div>

            {/* Enhanced hint with subtle animation */}
            <div className="text-sm text-muted-foreground/80 animate-in slide-in-from-bottom duration-1000 delay-700 flex items-center gap-2">
              <div className="w-2 h-2 bg-green-500 rounded-full animate-pulse"></div>
              Start typing your creative workflow below
            </div>
          </div>
        )}
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
                      {message.login ? (
                        <ErrorBoundary>
                          <LoginUI 
                            loginState={message.login}
                            onUpdate={(updatedMessage: any) => handleMessageUpdate(index, updatedMessage)}
                          />
                        </ErrorBoundary>
                      ) : message.logout ? (
                        <ErrorBoundary>
                          <LogoutUI 
                            logoutState={message.logout}
                            onUpdate={(updatedMessage: any) => handleMessageUpdate(index, updatedMessage)}
                          />
                        </ErrorBoundary>
                      ) : message.status ? (
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