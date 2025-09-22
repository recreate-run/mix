import { useNavigate } from '@tanstack/react-router';
import { useEffect } from 'react';
import {
  CommandInput,
  CommandList,
  Command as CommandPrimitive,
} from '@/components/ui/command';
import { useCommandPaletteState } from '@/hooks/command-slash/useCommandPaletteState';
import { useCommandHandlers } from '@/hooks/command-slash/useCommandHandlers';
import type { CommandSlashProps, AuthMethod } from '@/types/command-slash';
import { PermissionsView } from './command-slash/PermissionsView';
import { SessionsView } from './command-slash/SessionsView';
import { MCPServersView } from './command-slash/MCPServersView';
import { MCPToolsView } from './command-slash/MCPToolsView';
import { ProvidersView } from './command-slash/ProvidersView';
import { AuthInputView } from './command-slash/AuthInputView';
import { ModelSelectionView } from './command-slash/ModelSelectionView';
import { HelpMenuView } from './command-slash/HelpMenuView';
import { CommandsListView } from './command-slash/CommandsListView';



export function CommandSlash({
  onClose,
  sessionId,
  onFeedbackMessage,
  onNewSession,
  onQueryClientInvalidate,
  onSubmitMessage,
  onAddMessage,
}: CommandSlashProps) {
  const navigate = useNavigate();

  // Use custom hooks for state management
  const state = useCommandPaletteState();

  const handlers = useCommandHandlers({
    onFeedbackMessage,
    onAddMessage,
    onQueryClientInvalidate,
    onClose,
    setStatusData: state.setStatusData,
    setLoginData: state.setLoginData,
    setLogoutData: state.setLogoutData,
    setHierarchicalModelData: state.setHierarchicalModelData,
    setHelpData: state.setHelpData,
    goToView: state.goToView,
  });

  // Navigation helpers
  const handleNavigateToSession = (sessionId: string) => {
    navigate({
      to: '/$sessionId',
      params: { sessionId },
      replace: true,
    });
    onClose();
  };

  // Command execution handler
  const handleCommandExecution = (commandId: string) => {
    // Clear search when navigating to avoid confusing states
    state.setSearchQuery('');

    if (commandId === 'clear') {
      onNewSession?.();
      onClose();
      return;
    }

    switch (commandId) {
      case 'status':
        handlers.handleStatusCommandSpecial();
        break;
      case 'login':
        handlers.handleLoginCommandSpecial();
        break;
      case 'logout':
        handlers.handleLogoutCommandSpecial();
        break;
      case 'model':
        handlers.handleUnifiedModelCommandSpecial();
        break;
      case 'help':
        handlers.handleHelpCommandSpecial();
        break;
      case 'permissions':
        state.goToView('permissions');
        break;
      case 'sessions':
        state.goToView('sessions');
        break;
      case 'mcp':
        state.goToView('mcp');
        break;
      default:
        onSubmitMessage?.(`/${commandId}`);
        onClose();
        break;
    }
  };


  // Handle view transitions based on data changes
  useEffect(() => {
    if (state.hierarchicalModelData) {
      state.goToView('hierarchical-model');
      // Only reset selectedProvider on initial load
      if (!state.hierarchicalModelInitializedRef.current) {
        state.setSelectedProvider(null);
        state.hierarchicalModelInitializedRef.current = true;
      }
    } else {
      state.hierarchicalModelInitializedRef.current = false;
      state.setSelectedProvider(null);
    }
  }, [state.hierarchicalModelData, state]);

  useEffect(() => {
    if (state.loginData) {
      // Check if we're in OAuth flow
      if (state.selectedProvider && state.loginData?.oauthState) {
        state.setSelectedAuthMethod('oauth_code');
        state.goToView('login-auth-input');
      } else {
        state.goToView('login');
        state.setSelectedProvider(null);
        state.setSelectedAuthMethod(null);
      }
    }
  }, [state.loginData, state]);

  useEffect(() => {
    if (state.logoutData) {
      state.goToView('logout');
      state.setSelectedProvider(null);
      state.setSelectedAuthMethod(null);
    }
  }, [state.logoutData, state]);

  useEffect(() => {
    if (state.statusData) {
      state.goToView('status');
      state.setSelectedProvider(null);
      state.setSelectedAuthMethod(null);
    }
  }, [state.statusData, state]);

  useEffect(() => {
    if (state.helpData) {
      state.goToView('help');
      state.setSelectedProvider(null);
      state.setSelectedAuthMethod(null);
    }
  }, [state.helpData, state]);

  // Helper functions for auth flow
  const handleAuthMethodSelect = (method: AuthMethod) => {
    state.setSelectedAuthMethod(method);
    if (method === 'api_key' || method === 'oauth' || method === 'oauth_code') {
      state.goToView('login-auth-input');
    }
  };

  const handleProviderSelect = (providerId: string) => {
    state.setSelectedProvider(providerId);
    if (state.currentView === 'login') {
      const provider = state.loginData?.providers.find(p => p.id === providerId);
      if (provider?.authMethods.length === 1) {
        state.setSelectedAuthMethod(provider.authMethods[0] as AuthMethod);
        state.goToView('login-auth-input');
      } else {
        state.goToView('login-auth-methods');
      }
    } else if (state.currentView === 'hierarchical-model') {
      state.goToView('hierarchical-models');
    }
  };

  const getSelectedLoginProvider = () => {
    return state.selectedProvider
      ? state.loginData?.providers.find(p => p.id === state.selectedProvider)
      : undefined;
  };


  // Generic selection handler
  const handleSelect = (value: string) => {
    state.setSearchQuery('');
    state.setSelectedValue('');

    // Navigation commands
    if (value === 'back-to-commands') {
      state.resetToCommands();
      return;
    }
    if (value === 'back-to-providers') {
      state.setSelectedProvider(null);
      state.setSelectedAuthMethod(null);
      if (state.currentView === 'login-auth-methods') {
        state.goToView('login');
      } else if (state.currentView === 'hierarchical-models') {
        state.goToView('hierarchical-model');
      }
      return;
    }
    if (value === 'back-to-auth-methods') {
      state.setSelectedAuthMethod(null);
      state.goToView('login-auth-methods');
      return;
    }
    if (value === 'back-to-mcp') {
      state.setSelectedMCPServer(null);
      state.goToView('mcp');
      return;
    }

    // This will be handled by individual view components
    // But we keep this as a fallback for navigation between main views
    if (['permissions', 'sessions', 'mcp'].includes(value)) {
      handleCommandExecution(value);
    }
  };

  const handleKeyDown = (e: React.KeyboardEvent) => {
    if (e.key === 'Escape') {
      e.preventDefault();

      // Use the state management hook's navigation helper
      if (state.selectedAuthMethod || state.selectedProvider || state.selectedMCPServer) {
        state.goBack();
      } else if (state.currentView !== 'commands') {
        state.resetToCommands();
      } else {
        onClose();
      }
    }
  };

  // Get placeholder text based on current view
  const getPlaceholder = () => {
    if (state.isShowingLoginAuthInput && state.selectedAuthMethod) {
      return 'Enter API key or OAuth code...';
    }
    if (state.isShowingLoginAuthMethods) return 'Search auth methods...';
    if (state.isShowingHierarchicalModels) return 'Search models...';
    if (state.isShowingMCPTools) return 'Search tools...';
    if (state.isShowingHierarchicalModel || state.isShowingLogin || state.isShowingLogout || state.isShowingStatus) {
      return 'Search providers...';
    }
    if (state.isShowingMCP) return 'Search MCP servers...';
    if (state.isShowingPermissions) return 'Search permissions...';
    if (state.isShowingSessions) return 'Search sessions...';
    if (state.isShowingHelp) return 'Search help topics...';
    return 'Search commands...';
  };

  return (
    <div className="absolute right-0 bottom-full left-0 z-50 mb-2 overflow-hidden rounded-xl border border-border bg-popover shadow-lg">
      <CommandPrimitive
        className="max-h-64"
        onKeyDown={handleKeyDown}
        onValueChange={state.setSelectedValue}
        value={state.selectedValue}
      >
        <CommandInput
          autoFocus
          onValueChange={state.setSearchQuery}
          placeholder={getPlaceholder()}
          value={state.searchQuery}
        />

        <CommandList>
          {(() => {
            // Render the appropriate view component based on current state
            if (state.isShowingSessions) {
              return (
                <SessionsView
                  sessionId={sessionId}
                  onBackToCommands={() => handleSelect('back-to-commands')}
                  onNavigateToSession={handleNavigateToSession}
                />
              );
            }

            if (state.isShowingPermissions) {
              return (
                <PermissionsView
                  onBackToCommands={() => handleSelect('back-to-commands')}
                />
              );
            }

            if (state.isShowingHelp && state.helpData) {
              return (
                <HelpMenuView
                  helpData={state.helpData}
                  onBackToCommands={() => handleSelect('back-to-commands')}
                  onClose={onClose}
                  onExecuteCommand={handleCommandExecution}
                />
              );
            }

            if (state.isShowingMCPTools && state.selectedMCPServer) {
              return (
                <MCPToolsView
                  selectedMCPServer={state.selectedMCPServer}
                  onBackToServers={() => handleSelect('back-to-mcp')}
                />
              );
            }

            if (state.isShowingMCP) {
              return (
                <MCPServersView
                  onBackToCommands={() => handleSelect('back-to-commands')}
                  onServerSelect={(serverName) => {
                    state.setSelectedMCPServer(serverName);
                    state.goToView('mcp-tools');
                  }}
                />
              );
            }

            if (state.isShowingStatus && state.statusData) {
              return (
                <ProvidersView
                  type="status"
                  providers={state.statusData.providers}
                  onBackToCommands={() => handleSelect('back-to-commands')}
                  onProviderSelect={handlers.handleProviderSelectionSpecial}
                />
              );
            }

            if (state.isShowingLoginAuthInput && state.selectedProvider) {
              const selectedProvider = getSelectedLoginProvider();
              if (selectedProvider) {
                return (
                  <AuthInputView
                    selectedProvider={selectedProvider}
                    selectedAuthMethod={state.selectedAuthMethod}
                    onBackToProviders={() => handleSelect('back-to-providers')}
                    onBackToAuthMethods={() => handleSelect('back-to-auth-methods')}
                    onAuthMethodSelect={handleAuthMethodSelect}
                    onOAuthStart={async (providerId) =>
                      await handlers.handleLoginProviderSelectionSpecial(providerId, 'oauth')
                    }
                    onApiKeySubmit={handlers.handleApiKeySubmitSpecial}
                    onOAuthCodeSubmit={(providerId, code) =>
                      handlers.handleOAuthCodeSubmitSpecial(providerId, code, state.loginData?.oauthState)
                    }
                  />
                );
              }
            }

            if (state.isShowingLogin && state.loginData) {
              return (
                <ProvidersView
                  type="login"
                  providers={state.loginData.providers}
                  onBackToCommands={() => handleSelect('back-to-commands')}
                  onProviderSelect={handleProviderSelect}
                />
              );
            }

            if (state.isShowingLogout && state.logoutData) {
              return (
                <ProvidersView
                  type="logout"
                  providers={state.logoutData.providers}
                  onBackToCommands={() => handleSelect('back-to-commands')}
                  onProviderSelect={handlers.handleLogoutProviderSelectionSpecial}
                />
              );
            }

            if (state.isShowingHierarchicalModel && state.hierarchicalModelData) {
              return (
                <ModelSelectionView
                  hierarchicalModelData={state.hierarchicalModelData}
                  selectedProvider={state.selectedProvider}
                  onBackToCommands={() => handleSelect('back-to-commands')}
                  onBackToProviders={() => handleSelect('back-to-providers')}
                  onProviderSelect={handleProviderSelect}
                  onModelSelect={handlers.handleModelSelectionSpecial}
                />
              );
            }

            // Default: Commands view
            return (
              <CommandsListView
                onCommandExecute={handleCommandExecution}
              />
            );
          })()}
        </CommandList>

        {/* Bottom Toolbar */}
        <div className="flex h-6 items-center justify-end border-gray-200/50 border-t bg-gray-50/80 px-3 py-1 text-xs dark:border-gray-700/50 dark:bg-gray-800/80">
          <div className="flex items-center gap-2">
            <div className="flex items-center gap-0.5">
              <kbd className="rounded bg-white px-1 py-0 font-mono text-[10px] text-muted-foreground dark:bg-gray-700">
                ↵
              </kbd>
              <span className="text-gray-500 dark:text-gray-400">select</span>
            </div>

            <div className="flex items-center gap-0.5">
              <kbd className="rounded bg-white px-1 py-0 font-mono text-[10px] text-muted-foreground dark:bg-gray-700">
                esc
              </kbd>
              <span className="text-gray-500 dark:text-gray-400">
                {state.selectedProvider ||
                state.selectedMCPServer ||
                state.currentView !== 'commands'
                  ? 'back'
                  : 'close'}
              </span>
            </div>
          </div>
        </div>
      </CommandPrimitive>
    </div>
  );
}
