import type { Attachment } from '@/stores/attachmentSlice';
import type { ToolCall, ToolCallData } from './common';
import type { MediaOutput } from './media';

export interface UIMessage {
  content: string;
  from: 'user' | 'assistant';
  frontend_only?: boolean;
  toolCalls?: ToolCall[];
  attachments?: Attachment[];
  reasoning?: string;
  reasoningDuration?: number;
  mediaOutputs?: MediaOutput[];
}

export interface BackendMessage {
  id: string;
  sessionId: string;
  role: string;
  content: string;
  toolCalls?: ToolCallData[];
}

export type MessageData = {
  text: string;
  media: string[];
  apps: string[];
  plan_mode: boolean;
};
