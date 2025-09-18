import type { Attachment } from '@/stores/attachmentSlice';
import type { ToolCall, ToolCallData } from './common';
import type { MediaOutput } from './media';
import type { HierarchicalModelData, ProviderInfo } from './provider';

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
    providers: ProviderInfo[];
    selectedProvider?: string;
    step: "provider_select" | "auth_method" | "api_key" | "oauth_flow" | "oauth_code";
    authUrl?: string;
    hasExistingPreferences?: boolean;
    provider?: string; // Current provider for OAuth flow
    state?: string; // OAuth state parameter
  };
  loginData?: {
    providers: {
      id: string;
      displayName: string;
      authMethods: ("api_key" | "oauth")[];
      authenticated: boolean;
      apiKeyFormat?: string;
      isPreferred?: boolean;
    }[];
    hasExistingPreferences?: boolean;
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
  statusData?: {
    providers: {
      id: string;
      displayName: string;
      authenticated: boolean;
      authMethod?: 'api_key' | 'oauth';
      isPreferred?: boolean;
    }[];
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
  logout?: {
    providers: {
      id: string;
      displayName: string;
      authenticated: boolean;
      authMethod?: 'api_key' | 'oauth';
      isPreferred?: boolean;
    }[];
  };
  logoutData?: {
    providers: {
      id: string;
      displayName: string;
      authenticated: boolean;
      authMethod?: 'api_key' | 'oauth';
      isPreferred?: boolean;
    }[];
  };
  hierarchicalModel?: HierarchicalModelData;
  shouldInvalidatePreferencesCache?: boolean; // Signal to the UI to invalidate the preferences cache
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