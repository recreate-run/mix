import type { Attachment } from '@/stores/attachmentSlice';
import type { ToolCall, ToolCallData } from './common';
import type { MediaOutput } from './media';

export type TimelineEntry = 
  | {
      type: 'thinking';
      timestamp: number;
      content: string;
      id: string;
    }
  | {
      type: 'tool';
      timestamp: number;
      content: ToolCall;
      id: string;
    };

export interface UIMessage {
  content: string;
  from: 'user' | 'assistant';
  frontend_only?: boolean;
  toolCalls?: ToolCall[];
  attachments?: Attachment[];
  timeline?: TimelineEntry[];
  mediaOutputs?: MediaOutput[];
  reasoning?: string;
  reasoningDuration?: number;
}

export interface BackendMessage {
  id: string;
  sessionId: string;
  role: string;
  userInput: string;
  assistantResponse?: string;
  toolCalls?: ToolCallData[];
  reasoning?: string;
  reasoningDuration?: number;
}

export type MessageData = {
  text: string;
  media: string[];
  apps: string[];
  plan_mode: boolean;
};
