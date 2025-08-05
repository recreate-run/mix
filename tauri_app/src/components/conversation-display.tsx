import { type RefObject, type RefCallback } from 'react';
import { AIMessage, AIMessageContent } from '@/components/ui/kibo-ui/ai/message';
import { AIResponse } from '@/components/ui/kibo-ui/ai/response';
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
  type AIToolStatus,
} from '@/components/ui/kibo-ui/ai/tool';
import { ResponseRenderer } from './response-renderer';
import { MessageAttachmentDisplay } from './message-attachment-display';
import { TodoList } from './todo-list';
import { PlanDisplay } from './plan-display';
import { LoadingDots } from './loading-dots';
import { type Attachment } from '@/stores/attachmentStore';

type ToolCall = {
  name: string;
  description: string;
  status: AIToolStatus;
  parameters: Record<string, unknown>;
  result?: string;
  error?: string;
};

type Message = {
  content: string;
  from: 'user' | 'assistant';
  toolCalls?: ToolCall[];
  attachments?: Attachment[];
  reasoning?: string;
  reasoningDuration?: number;
};

type StreamingState = {
  processing: boolean;
  reasoning?: string;
  reasoningDuration?: number;
  toolCalls: any[];
  completed: boolean;
};

interface ConversationDisplayProps {
  messages: Message[];
  sseStream: StreamingState;
  showPlanOptions: number | null;
  conversationRef: RefObject<HTMLDivElement>;
  setUserMessageRef: (index: number) => RefCallback<HTMLDivElement>;
  onPlanProceed: (index: number) => void;
  onPlanKeepPlanning: (index: number) => void;
}

// Helper function to extract todos from todo_write tool calls (works with both ToolCall and SSE formats)
const extractTodosFromToolCalls = (toolCalls: any[]) => {
  return toolCalls
    .filter(tc => tc.name === 'todo_write')
    .map(tc => {
      try {
        const todos = tc.parameters?.todos;
        return Array.isArray(todos) ? todos : [];
      } catch {
        return [];
      }
    })
    .flat();
};

// Helper function to extract plan content from exit_plan_mode tool calls (works with both ToolCall and SSE formats)
const extractPlanFromToolCalls = (toolCalls: any[]) => {
  const planTool = toolCalls.find(tc => tc.name === 'exit_plan_mode');
  if (!planTool) return '';
  
  try {
    return planTool.parameters?.plan || '';
  } catch {
    return '';
  }
};

// Helper function to filter out special tools (todo_write, exit_plan_mode) from toolCalls
const filterNonSpecialTools = (toolCalls: any[]) => {
  return toolCalls.filter(tc => tc.name !== 'todo_write' && tc.name !== 'exit_plan_mode');
};

// Helper function to check if previous user message started with "!"
const isPreviousUserMessageCommand = (messages: Message[], currentIndex: number) => {
  for (let i = currentIndex - 1; i >= 0; i--) {
    if (messages[i].from === 'user') {
      return messages[i].content.trim().startsWith('!');
    }
  }
  return false;
};

export function ConversationDisplay({
  messages,
  sseStream,
  showPlanOptions,
  conversationRef,
  setUserMessageRef,
  onPlanProceed,
  onPlanKeepPlanning
}: ConversationDisplayProps) {
  return (
    <div ref={conversationRef} className="relative h-full flex-1 overflow-y-auto">
      <div className="">
        {messages.map((message, index) => (
          <AIMessage 
            from={message.from} 
            key={index}
            ref={message.from === 'user' ? setUserMessageRef(index) : undefined}
          >
            <AIMessageContent>
              {message.from === 'assistant' ? (
                <>
                  {message.reasoning && (
                    <AIReasoning className="w-full mb-4" isStreaming={false} duration={message.reasoningDuration || undefined}>
                      <AIReasoningTrigger />
                      <AIReasoningContent>{message.reasoning}</AIReasoningContent>
                    </AIReasoning>
                  )}
                  {isPreviousUserMessageCommand(messages, index) ? (
                    <AIResponse>{`\`\`\`bash\n${message.content}\n\`\`\``}</AIResponse>
                  ) : (
                    <ResponseRenderer content={message.content} />
                  )}
                </>
              ) : (
                <div>
                  <MessageAttachmentDisplay attachments={message.attachments || []} />
                  {message.content}
                </div>
              )}
              {/* Render todos inline without tool wrapper */}
              {message.toolCalls && message.toolCalls.length > 0 && (
                <>
                  {/* Render plan content */}
                  {extractPlanFromToolCalls(message.toolCalls) && (
                    <PlanDisplay 
                      planContent={extractPlanFromToolCalls(message.toolCalls)}
                      showOptions={showPlanOptions === index}
                      onProceed={() => onPlanProceed(index)}
                      onKeepPlanning={() => onPlanKeepPlanning(index)}
                    />
                  )}
                  {/* Render non-special tools in ladder */}
                  {filterNonSpecialTools(message.toolCalls).length > 0 && (
                    <AIToolLadder className="mt-4">
                      {filterNonSpecialTools(message.toolCalls).map((toolCall, toolIndex) => (
                        <AIToolStep
                          key={`${index}-${toolCall.name}-${toolIndex}`}
                          status={toolCall.status}
                          stepNumber={toolIndex + 1}
                          isLast={toolIndex === filterNonSpecialTools(message.toolCalls).length - 1}
                        >
                          <AIToolHeader
                            description={toolCall.description}
                            name={toolCall.name}
                            status={toolCall.status}
                          />
                          <AIToolContent toolCall={toolCall} />
                        </AIToolStep>
                      ))}
                    </AIToolLadder>
                  )}
                </>
              )}
            </AIMessageContent>
          </AIMessage>
        ))}
        {sseStream.processing && (
          <AIMessage 
            from="assistant"
          >
            <AIMessageContent>
              {/* Show reasoning during streaming if available */}
              {sseStream.reasoning && (
                <AIReasoning className="w-full mb-4" isStreaming={true} duration={sseStream.reasoningDuration || undefined}>
                  <AIReasoningTrigger />
                  <AIReasoningContent>{sseStream.reasoning}</AIReasoningContent>
                </AIReasoning>
              )}
              {sseStream.toolCalls.length > 0 ? (
                <>
                  {/* Render streaming todos inline without tool wrapper */}
                  {extractTodosFromToolCalls(sseStream.toolCalls).length > 0 && (
                    <div className="mt-4">
                      <TodoList todos={extractTodosFromToolCalls(sseStream.toolCalls)} />
                    </div>
                  )}
                  {/* Render streaming plan content */}
                  {extractPlanFromToolCalls(sseStream.toolCalls) && (
                    <PlanDisplay 
                      planContent={extractPlanFromToolCalls(sseStream.toolCalls)}
                      showOptions={false}
                    />
                  )}
                  {/* Render streaming non-special tools in ladder */}
                  {filterNonSpecialTools(sseStream.toolCalls).length > 0 && (
                    <AIToolLadder>
                      {filterNonSpecialTools(sseStream.toolCalls).map((toolCall, toolIndex) => (
                        <AIToolStep
                          key={`streaming-${toolCall.id}-${toolIndex}`}
                          status={toolCall.status}
                          stepNumber={toolIndex + 1}
                          isLast={toolIndex === filterNonSpecialTools(sseStream.toolCalls).length - 1}
                        >
                          <AIToolHeader
                            description={toolCall.description}
                            name={toolCall.name}
                            status={toolCall.status}
                          />
                          <AIToolContent toolCall={toolCall} />
                        </AIToolStep>
                      ))}
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