import type { Attachment } from '@/stores/attachmentSlice';
import type { ToolCall, ToolCallData } from './common';
import type { MediaOutput } from './media';
import type { HierarchicalModelData, ProviderInfo, ModelInfo } from './provider';

// Login-specific provider info that requires authMethods
export interface LoginProviderInfo {
  id: string;
  displayName: string;
  authMethods: ("api_key" | "oauth")[];
  authenticated: boolean;
  apiKeyFormat?: string;
  isPreferred?: boolean;
}

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
    }
  | {
      type: 'content';
      timestamp: number;
      content: string;
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
    providers: ProviderInfo[];
    selectedProvider?: string;
    step: "provider_select" | "auth_method" | "api_key" | "oauth_flow" | "oauth_code";
    authUrl?: string;
    hasExistingPreferences?: boolean;
    provider?: string; // Current provider for OAuth flow
    state?: string; // OAuth state parameter
  };
  loginData?: {
    providers: LoginProviderInfo[];
    hasExistingPreferences?: boolean;
  };
  status?: {
    providers: ProviderInfo[];
    hasAuthenticatedProvider: boolean;
  };
  statusData?: {
    providers: ProviderInfo[];
  };
  provider?: {
    providers: ProviderInfo[];
    currentProvider?: string;
  };
  model?: {
    models: ModelInfo[];
    currentModel?: string;
    provider: {
      id: string;
      displayName: string;
    };
  };
  logout?: {
    providers: ProviderInfo[];
  };
  logoutData?: {
    providers: ProviderInfo[];
  };
  hierarchicalModel?: HierarchicalModelData;
  shouldInvalidatePreferencesCache?: boolean; // Signal to the UI to invalidate the preferences cache
  suppressChatMessage?: boolean; // When true, message won't be shown in the chat interface
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