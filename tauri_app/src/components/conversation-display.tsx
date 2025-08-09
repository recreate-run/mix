import { convertFileSrc } from '@tauri-apps/api/core';
import { Check, Copy, Pencil, Play } from 'lucide-react';
import type { RefCallback, RefObject } from 'react';
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
  type AIToolStatus,
  AIToolStep,
} from '@/components/ui/kibo-ui/ai/tool';
import { useCopyToClipboard } from '@/hooks/use-copy-to-clipboard';
import type { Attachment } from '@/stores/attachmentStore';
import { AudioWaveform } from './audio-waveform';
import { LoadingDots } from './loading-dots';
import { MessageAttachmentDisplay } from './message-attachment-display';
import { PlanDisplay } from './plan-display';
import { ResponseRenderer } from './response-renderer';
import { TodoList } from './todo-list';
import { RemotionVideoPreview } from './remotion/RemotionVideoPreview';

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

// Media Showcase Component
const MediaShowcase = ({ mediaOutputs }: { mediaOutputs: MediaOutput[] }) => {
  if (!mediaOutputs || mediaOutputs.length === 0) return null;

  return (
    <div className=" space-y-4">
      {mediaOutputs.map((output, index) => (
        <div key={index} className="rounded-lg p-2">
          <div className="mb-3">
            <h3 className="font-semibold">{output.title}</h3>
            {output.description && (
              <p className="text-sm text-muted-foreground mt-1">{output.description}</p>
            )}
          </div>
          
          {output.type === 'image' && (
            <div className="rounded-md overflow-hidden">
              <img
                src={convertFileSrc(output.path)}
                alt={output.title}
                className="w-full h-auto max-h-96 object-contain bg-black"
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
                Failed to load image: {output.path}
              </div>
            </div>
          )}
          
          {output.type === 'video' && (
            <div className="rounded-md overflow-hidden">
              <video
                src={convertFileSrc(output.path)}
                controls
                className="w-full h-auto max-h-96 bg-black"
                preload="metadata"
                onError={(e) => {
                  e.currentTarget.style.display = 'none';
                  const fallback = e.currentTarget.nextElementSibling as HTMLElement;
                  if (fallback) fallback.style.display = 'block';
                }}
              >
                Your browser does not support the video tag.
              </video>
              <div 
                className="flex items-center justify-center h-48 bg-stone-700 text-stone-400"
                style={{ display: 'none' }}
              >
                Failed to load video: {output.path}
              </div>
            </div>
          )}
          
          {output.type === 'audio' && (
            <div className="rounded-md bg-stone-700/30 p-4">
              <audio
                src={convertFileSrc(output.path)}
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
                Failed to load audio: {output.path}
              </div>
            </div>
          )}
          
          {output.type === 'remotion_title' && output.config && (
            <RemotionVideoPreview config={output.config as any} />
          )}
        </div>
      ))}
    </div>
  );
};

interface ConversationDisplayProps {
  messages: Message[];
  sseStream: StreamingState;
  showPlanOptions: number | null;
  conversationRef: RefObject<HTMLDivElement>;
  setUserMessageRef: (index: number) => RefCallback<HTMLDivElement>;
  onPlanProceed: (index: number) => void;
  onPlanKeepPlanning: (index: number) => void;
  onForkMessage?: (index: number) => void;
}

// Helper function to extract todos from todo_write tool calls (works with both ToolCall and SSE formats)
const extractTodosFromToolCalls = (toolCalls: any[]) => {
  return toolCalls
    .filter((tc) => tc.name === 'todo_write')
    .flatMap((tc) => {
      try {
        const todos = tc.parameters?.todos;
        return Array.isArray(todos) ? todos : [];
      } catch {
        return [];
      }
    });
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
  showPlanOptions,
  conversationRef,
  setUserMessageRef,
  onPlanProceed,
  onPlanKeepPlanning,
  onForkMessage,
}: ConversationDisplayProps) {
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
                  <AIMessageContent.Toolbar>
                    <MessageCopyButton content={message.content} />
                  </AIMessageContent.Toolbar>
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
                      onKeepPlanning={() => onPlanKeepPlanning(index)}
                      onProceed={() => onPlanProceed(index)}
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
              {sseStream.toolCalls.length > 0 ? (
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
