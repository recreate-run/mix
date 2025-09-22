import type { HierarchicalModelData } from '@/types';

export type { HierarchicalModelData };

export interface CommandSlashProps {
  onClose: () => void;
  sessionId: string;
  onFeedbackMessage?: (message: string) => void;
  onNewSession?: () => void;
  onQueryClientInvalidate?: (keys: any) => void;
  onSubmitMessage?: (message: string) => void;
  onAddMessage?: (message: any) => void;
}

export interface Provider {
  id: string;
  displayName: string;
  authenticated: boolean;
  authMethod?: 'api_key' | 'oauth';
  isPreferred?: boolean;
}

export interface LoginProvider extends Provider {
  authMethods: ('api_key' | 'oauth')[];
  apiKeyFormat?: string;
}

export interface LoginData {
  providers: LoginProvider[];
  hasExistingPreferences?: boolean;
  oauthState?: string;
}

export interface LogoutData {
  providers: Provider[];
}

export interface StatusData {
  providers: Provider[];
}

export interface HelpMenuItem {
  id: string;
  name: string;
  description: string;
  action: string;
  url?: string;
}

export interface HelpData {
  menuItems: HelpMenuItem[];
}

export type AuthMethod = 'api_key' | 'oauth' | 'oauth_code';

export type ViewState =
  | 'commands'
  | 'permissions'
  | 'sessions'
  | 'mcp'
  | 'mcp-tools'
  | 'hierarchical-model'
  | 'hierarchical-models'
  | 'login'
  | 'login-auth-methods'
  | 'login-auth-input'
  | 'logout'
  | 'status'
  | 'help';

export interface CommandPaletteState {
  currentView: ViewState;
  selectedProvider: string | null;
  selectedMCPServer: string | null;
  selectedAuthMethod: AuthMethod | null;
  hierarchicalModelData?: HierarchicalModelData;
  loginData?: LoginData;
  logoutData?: LogoutData;
  statusData?: StatusData;
  helpData?: HelpData;
}


