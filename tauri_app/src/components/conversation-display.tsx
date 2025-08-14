import { convertFileSrc } from '@tauri-apps/api/core';
import { Check, Copy, Pencil } from 'lucide-react';
import type { RefCallback, RefObject } from 'react';
import { useState, useEffect } from 'react';
import { Button } from '@/components/ui/button';
import { Skeleton } from '@/components/ui/skeleton';
import { VideoPlayer } from './video-player';
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
  type AIToolStatus,
  AIToolStep,
} from '@/components/ui/kibo-ui/ai/tool';
import { useCopyToClipboard } from '@/hooks/use-copy-to-clipboard';
import type { Attachment } from '@/stores';
import { LoadingDots } from './loading-dots';
import { MessageAttachmentDisplay } from './message-attachment-display';
import { PlanDisplay } from './plan-display';
import { RateLimitDisplay } from './rate-limit-display';
import { ResponseRenderer } from './response-renderer';
import { TodoList } from './todo-list';
import { RemotionVideoPreview } from './remotion/RemotionVideoPreview';
import { PlaylistSidebar } from './playlist-sidebar';


type ToolCall = {
  name: string;
  description: string;
  status: AIToolStatus;
  parameters: Record<string, unknown>;
  result?: string;
  error?: string;
};

type MediaOutput = {
  path: string;
  type: 'image' | 'video' | 'audio' | 'remotion_title';
  title: string;
  description?: string;
  config?: any; // For remotion configuration data
  sourceVideo?: string; // Original video path for highlights
  startTime?: number; // Highlight start time in seconds
  duration?: number; // Highlight duration in seconds
};


type Message = {
  content: string;
  from: 'user' | 'assistant';
  toolCalls?: ToolCall[];
  attachments?: Attachment[];
  reasoning?: string;
  reasoningDuration?: number;
  mediaOutputs?: MediaOutput[];
};

type StreamingState = {
  processing: boolean;
  reasoning?: string;
  reasoningDuration?: number;
  toolCalls: any[];
  completed: boolean;
};

// Main Media Player Component
const MainMediaPlayer = ({ media }: { media: MediaOutput }) => {
  const [isLoading, setIsLoading] = useState(true);

  useEffect(() => {
    setIsLoading(true);
  }, [media.path]);

  return (
    <div className="">
      <div className="mb-3">
        <h3 className="font-semibold">{media.title}</h3>
        {media.description && (
          <p className="text-sm text-muted-foreground mt-1">{media.description}</p>
        )}
      </div>

      {media.type === 'image' && (
        <div className="rounded-md overflow-hidden">
          <img
            src={convertFileSrc(media.path)}
            alt={media.title}
            className="w-full object-contain aspect-video   bg-black"
            onError={(e) => {
              e.currentTarget.style.display = 'none';
              const fallback = e.currentTarget.nextElementSibling as HTMLElement;
              if (fallback) fallback.style.display = 'block';
            }}
          />
          <div
            className="flex items-center justify-center h-48 bg-stone-700 text-stone-400"
            style={{ display: 'none' }}
          >
            Failed to load image: {media.path}
          </div>
        </div>
      )}

      {media.type === 'video' && (
        <VideoPlayer 
          path={media.path}
          title=""
          sourceVideo={media.sourceVideo}
          startTime={media.startTime}
          duration={media.duration}
        />
      )}

      {media.type === 'audio' && (
        <div className="rounded-md bg-stone-700/30 p-4">
          <audio
            src={convertFileSrc(media.path)}
            controls
            className="w-full"
            preload="metadata"
            onError={(e) => {
              e.currentTarget.style.display = 'none';
              const fallback = e.currentTarget.nextElementSibling as HTMLElement;
              if (fallback) fallback.style.display = 'block';
            }}
          >
            Your browser does not support the audio tag.
          </audio>
          <div
            className="text-center text-stone-400 mt-2"
            style={{ display: 'none' }}
          >
            Failed to load audio: {media.path}
          </div>
        </div>
      )}

      {media.type === 'remotion_title' && media.config && (
        <RemotionVideoPreview config={media.config as any} />
      )}
    </div>
  );
};


// Media Showcase Component
const MediaShowcase = ({ mediaOutputs }: { mediaOutputs: MediaOutput[] }) => {
  const [selectedIndex, setSelectedIndex] = useState(0);

  if (!mediaOutputs || mediaOutputs.length === 0) return null;

  // Single media file - show directly
  if (mediaOutputs.length === 1) {
    return <MainMediaPlayer media={mediaOutputs[0]} />;
  }

  // Multiple media files - show player + playlist
  return (
    <div className="space-y-4">
      <MainMediaPlayer media={mediaOutputs[selectedIndex]} />
      <PlaylistSidebar
        mediaOutputs={mediaOutputs}
        selectedIndex={selectedIndex}
        onSelect={setSelectedIndex}
      />
    </div>
  );
};

interface ConversationDisplayProps {
  messages: Message[];
  sseStream: StreamingState;
  conversationRef: RefObject<HTMLDivElement>;
  setUserMessageRef: (index: number) => RefCallback<HTMLDivElement>;
  onPlanAction?: (action: 'proceed' | 'keep-planning', messageIndex: number) => void;
  onForkMessage?: (index: number) => void;
}

// Helper function to extract todos from todo_write tool calls (works with both ToolCall and SSE formats)
const extractTodosFromToolCalls = (toolCalls: any[]) => {
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
    } catch {
      continue;
    }
  }
  
  // Fallback: if no calls have parameters yet, return empty array
  return [];
};

// Helper function to extract plan content from exit_plan_mode tool calls (works with both ToolCall and SSE formats)
const extractPlanFromToolCalls = (toolCalls: any[]) => {
  const planTool = toolCalls.find((tc) => tc.name === 'exit_plan_mode');
  if (!planTool) return '';

  try {
    return planTool.parameters?.plan || '';
  } catch {
    return '';
  }
};

// Helper function to check if a message contains exit_plan_mode tool call
const hasExitPlanModeTool = (toolCalls: any[]) => {
  return toolCalls?.some((tc) => tc.name === 'exit_plan_mode');
};

// Helper function to filter out special tools (todo_write, exit_plan_mode, media_showcase) from toolCalls
const filterNonSpecialTools = (toolCalls: any[]) => {
  return toolCalls.filter(
    (tc) => tc.name !== 'todo_write' && tc.name !== 'exit_plan_mode' && tc.name !== 'media_showcase'
  );
};

// Helper function to check if previous user message started with "!"
const isPreviousUserMessageCommand = (
  messages: Message[],
  currentIndex: number
) => {
  for (let i = currentIndex - 1; i >= 0; i--) {
    if (messages[i].from === 'user') {
      return messages[i].content.trim().startsWith('!');
    }
  }
  return false;
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
  conversationRef,
  setUserMessageRef,
  onPlanAction,
  onForkMessage,
}: ConversationDisplayProps) {
  const [showPlanOptions, setShowPlanOptions] = useState<number | null>(null);

  // Detect when a new message with exit_plan_mode is added and show plan options
  useEffect(() => {
    if (messages.length > 0) {
      const lastMessage = messages[messages.length - 1];
      if (lastMessage.from === 'assistant' && lastMessage.toolCalls && hasExitPlanModeTool(lastMessage.toolCalls)) {
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
    <div
      className="relative h-full flex-1 overflow-y-auto pb-16"
      ref={conversationRef}
    >
      <div className="">
        {messages.length === 0 && (
          <AIMessage from="assistant">
            <AIMessageContent>
              <AIMessageContent.Content>
                Hello! I'm Mix, you AI agent for multimodal workflows. How can I help you today?
              </AIMessageContent.Content>
            </AIMessageContent>
          </AIMessage>
        )}
        {messages.map((message, index) => (
          <AIMessage
            from={message.from}
            key={index}
            ref={message.from === 'user' ? setUserMessageRef(index) : undefined}
          >
            <AIMessageContent>
              {message.from === 'assistant' ? (
                <>
                  <AIMessageContent.Content>
                    {message.reasoning && (
                      <AIReasoning
                        className="mb-4 w-full"
                        duration={message.reasoningDuration || undefined}
                        isStreaming={false}
                      >
                        <AIReasoningTrigger />
                        <AIReasoningContent>
                          {message.reasoning}
                        </AIReasoningContent>
                      </AIReasoning>
                    )}
                    {isPreviousUserMessageCommand(messages, index) ? (
                      <AIResponse>{`\`\`\`bash\n${message.content}\n\`\`\``}</AIResponse>
                    ) : (
                      <ResponseRenderer content={message.content} />
                    )}
                  </AIMessageContent.Content>
                  {!message.mediaOutputs && (
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
              {/* Render todos inline without tool wrapper */}
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
                  {/* Render media showcase */}
                  {message.mediaOutputs && (
                    <MediaShowcase mediaOutputs={message.mediaOutputs} />
                  )}
                  {/* Render non-special tools in ladder */}
                  {filterNonSpecialTools(message.toolCalls).length > 0 && (
                    <AIToolLadder className="mt-4">
                      {filterNonSpecialTools(message.toolCalls).map(
                        (toolCall, toolIndex) => (
                          <AIToolStep
                            isLast={
                              toolIndex ===
                              filterNonSpecialTools(message.toolCalls).length -
                              1
                            }
                            key={`${index}-${toolCall.name}-${toolIndex}`}
                            status={toolCall.status}
                            stepNumber={toolIndex + 1}
                          >
                            <AIToolHeader
                              description={toolCall.description}
                              name={toolCall.name}
                              status={toolCall.status}
                            />
                            <AIToolContent toolCall={toolCall} />
                          </AIToolStep>
                        )
                      )}
                    </AIToolLadder>
                  )}
                </>
              )}
            </AIMessageContent>
          </AIMessage>
        ))}
        {sseStream.processing && (
          <AIMessage from="assistant">
            <AIMessageContent>
              {/* Show reasoning during streaming if available */}
              {sseStream.reasoning && (
                <AIReasoning
                  className="mb-4 w-full"
                  duration={sseStream.reasoningDuration || undefined}
                  isStreaming={true}
                >
                  <AIReasoningTrigger />
                  <AIReasoningContent>{sseStream.reasoning}</AIReasoningContent>
                </AIReasoning>
              )}
              {/* Show rate limit message when rate limiting is detected */}
              {sseStream.rateLimit ? (
                <div className="mt-4">
                  <RateLimitDisplay 
                    retryAfter={sseStream.rateLimit.retryAfter}
                    attempt={sseStream.rateLimit.attempt}
                    maxAttempts={sseStream.rateLimit.maxAttempts}
                    error={sseStream.error || undefined}
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
                  {/* Render streaming non-special tools in ladder */}
                  {filterNonSpecialTools(sseStream.toolCalls).length > 0 && (
                    <AIToolLadder>
                      {filterNonSpecialTools(sseStream.toolCalls).map(
                        (toolCall, toolIndex) => (
                          <AIToolStep
                            isLast={
                              toolIndex ===
                              filterNonSpecialTools(sseStream.toolCalls)
                                .length -
                              1
                            }
                            key={`streaming-${toolCall.id}-${toolIndex}`}
                            status={toolCall.status}
                            stepNumber={toolIndex + 1}
                          >
                            <AIToolHeader
                              description={toolCall.description}
                              name={toolCall.name}
                              status={toolCall.status}
                            />
                            <AIToolContent toolCall={toolCall} />
                          </AIToolStep>
                        )
                      )}
                    </AIToolLadder>
                  )}
                  {!sseStream.completed && <LoadingDots />}
                </>
              ) : (
                <LoadingDots />
              )}
            </AIMessageContent>
          </AIMessage>
        )}
      </div>
    </div>
  );
}
