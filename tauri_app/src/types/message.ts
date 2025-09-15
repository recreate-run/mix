import type { Attachment } from '@/stores/attachmentSlice';
import type { ToolCall, ToolCallData } from './common';
import type { MediaOutput } from './media';
import type { HierarchicalModelData } from './provider';

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
  login?: {
    providers?: any[];
    selectedProvider?: string;
    step?: string;
    authUrl?: string;
    hasExistingPreferences?: boolean;
    provider?: string; // Current provider for OAuth flow
  };
  status?: {
    providers: {
      id: string;
      displayName: string;
      authenticated: boolean;
      authMethod?: 'api_key' | 'oauth';
      isPreferred?: boolean;
    }[];
    hasAuthenticatedProvider: boolean;
  };
  provider?: {
    providers: {
      id: string;
      displayName: string;
      authenticated: boolean;
      authMethod?: 'api_key' | 'oauth';
      isPreferred?: boolean;
    }[];
    currentProvider?: string;
  };
  model?: {
    models: {
      id: string;
      displayName: string;
      isSelected?: boolean;
    }[];
    currentModel?: string;
    provider: {
      id: string;
      displayName: string;
    };
  };
  hierarchicalModel?: HierarchicalModelData;
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