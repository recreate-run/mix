import { useNavigate } from '@tanstack/react-router';
import {
  Accessibility,
  ArrowLeft,
  CheckCircle,
  Clock,
  Folder,
  Mic,
  Monitor,
  Plug,
  Settings,
} from 'lucide-react';
import { useEffect, useRef, useState } from 'react';
import {
  CommandEmpty,
  CommandGroup,
  CommandInput,
  CommandItem,
  CommandList,
  Command as CommandPrimitive,
} from '@/components/ui/command';
import { Switch } from '@/components/ui/switch';
import { useMCPList } from '@/hooks/useMCPList';
import {
  useAccessibilityPermission,
  useFullDiskAccessPermission,
  useMicrophonePermission,
  useScreenRecordingPermission,
} from '@/hooks/usePermissions';
import { useActiveSession } from '@/hooks/useSession';
import { formatMessageCounts } from '@/types/common';
import {
  useSessionsList,
} from '@/hooks/useSessionsList';
import { slashCommands } from '@/utils/slash-commands';
import { getDisplayTitle } from '@/utils/sessionUtils';
import type { HierarchicalModelData } from '@/types';


interface CommandSlashProps {
  onExecuteCommand: (command: string) => void;
  onClose: () => void;
  sessionId: string;
  hierarchicalModelData?: HierarchicalModelData;
  logoutData?: {
    providers: {
      id: string;
      displayName: string;
      authenticated: boolean;
      authMethod?: 'api_key' | 'oauth';
      isPreferred?: boolean;
    }[];
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
    oauthState?: string;
  };
  onProviderSelect?: (providerId: string) => void;
  onModelSelect?: (providerId: string, modelId: string) => void;
  onLogoutProviderSelect?: (providerId: string) => void;
  onStatusProviderSelect?: (providerId: string) => void;
  onLoginProviderSelect?: (providerId: string, authMethod: "api_key" | "oauth") => void;
  onApiKeySubmit?: (providerId: string, apiKey: string) => Promise<void>;
  onOAuthCodeSubmit?: (providerId: string, code: string) => Promise<void>;
}

export function CommandSlash({
  onExecuteCommand,
  onClose,
  sessionId,
  hierarchicalModelData,
  logoutData,
  statusData,
  loginData,
  onProviderSelect,
  onModelSelect,
  onLogoutProviderSelect,
  onStatusProviderSelect,
  onLoginProviderSelect,
  onApiKeySubmit,
  onOAuthCodeSubmit,
}: CommandSlashProps) {
  const [selectedValue, setSelectedValue] = useState<string>('');
  const [searchQuery, setSearchQuery] = useState<string>('');
  const [showingPermissions, setShowingPermissions] = useState(false);
  const [showingSessions, setShowingSessions] = useState(false);
  const [showingMCP, setShowingMCP] = useState(false);
  const [selectedMCPServer, setSelectedMCPServer] = useState<string | null>(
    null
  );
  const [showingHierarchicalModel, setShowingHierarchicalModel] = useState(false);
  const [showingLogout, setShowingLogout] = useState(false);
  const [showingStatus, setShowingStatus] = useState(false);
  const [showingLogin, setShowingLogin] = useState(false);
  const [selectedProvider, setSelectedProvider] = useState<string | null>(null);
  const [selectedAuthMethod, setSelectedAuthMethod] = useState<"api_key" | "oauth" | "oauth_code" | null>(null);
  const [apiKey, setApiKey] = useState<string>('');
  const [oauthCode, setOauthCode] = useState<string>('');
  const [apiKeySubmitting, setApiKeySubmitting] = useState(false);
  const [oauthCodeSubmitting, setOauthCodeSubmitting] = useState(false);
  const commandRef = useRef<HTMLDivElement>(null);
  const hierarchicalModelInitializedRef = useRef(false);
  const navigate = useNavigate();

  // Reset selection when search query changes to prevent jumping
  useEffect(() => {
    setSelectedValue('');
  }, [searchQuery]);
  
  
  // Handle API key submission
  const handleApiKeySubmit = async () => {
    if (!selectedProvider || !apiKey || apiKeySubmitting) return;
    
    try {
      setApiKeySubmitting(true);
      
      // Call the onApiKeySubmit handler provided by the parent component
      if (onApiKeySubmit) {
        await onApiKeySubmit(selectedProvider, apiKey);
        
        // Close the command palette after successful submission
        onClose();
      }
    } catch (error) {
      console.error('API key submission failed:', error);
      // Keep the command palette open on error
    } finally {
      setApiKeySubmitting(false);
    }
  };
  
  // Handle OAuth code submission
  const handleOAuthCodeSubmit = async () => {
    if (!selectedProvider || !oauthCode || oauthCodeSubmitting) {
      return;
    }
    
    try {
      setOauthCodeSubmitting(true);
      
      if (onOAuthCodeSubmit) {
        // Call the onOAuthCodeSubmit handler provided by the parent component
        await onOAuthCodeSubmit(selectedProvider, oauthCode.trim());
        
        // Close the command palette after successful submission
        onClose();
      } else {
        // Simulate success for testing only
        setTimeout(() => {
          onClose();
        }, 1000);
      }
    } catch (error) {
      console.error('OAuth code submission failed:', error);
      // Keep the command palette open on error
    } finally {
      setOauthCodeSubmitting(false);
    }
  };

  // Show hierarchical model view when data is provided
  useEffect(() => {
    if (hierarchicalModelData) {
      setShowingHierarchicalModel(true);
      setShowingPermissions(false);
      setShowingSessions(false);
      setShowingMCP(false);
      setShowingLogout(false);
      setShowingStatus(false);
      setShowingLogin(false);
      setSelectedMCPServer(null);
      // Only reset selectedProvider on initial load, not on data updates
      if (!hierarchicalModelInitializedRef.current) {
        setSelectedProvider(null);
        hierarchicalModelInitializedRef.current = true;
      }
    } else {
      // Reset when hierarchical data is cleared
      hierarchicalModelInitializedRef.current = false;
      setSelectedProvider(null);
    }
  }, [hierarchicalModelData]);
  
  // Show logout view when data is provided
  useEffect(() => {
    if (logoutData) {
      setShowingLogout(true);
      setShowingHierarchicalModel(false);
      setShowingStatus(false);
      setShowingLogin(false);
      setShowingPermissions(false);
      setShowingSessions(false);
      setShowingMCP(false);
      setSelectedMCPServer(null);
      setSelectedProvider(null);
      setSelectedAuthMethod(null);
    }
  }, [logoutData, showingLogout]);
  
  // Show status view when data is provided
  useEffect(() => {
    if (statusData) {
      setShowingStatus(true);
      setShowingLogout(false);
      setShowingLogin(false);
      setShowingHierarchicalModel(false);
      setShowingPermissions(false);
      setShowingSessions(false);
      setShowingMCP(false);
      setSelectedMCPServer(null);
      setSelectedProvider(null);
      setSelectedAuthMethod(null);
    }
  }, [statusData, showingStatus]);
  
  // Show login view when data is provided
  useEffect(() => {
    if (loginData) {
      setShowingLogin(true);
      setShowingStatus(false);
      setShowingLogout(false);
      setShowingHierarchicalModel(false);
      setShowingPermissions(false);
      setShowingSessions(false);
      setShowingMCP(false);
      setSelectedMCPServer(null);
      
      // If we already have a provider selected from before and
      // loginData.oauthState is available, it likely means we're in OAuth flow
      if (selectedProvider && loginData?.oauthState) {
        setSelectedAuthMethod("oauth_code");
      } else {
        // Otherwise reset the selection
        setSelectedProvider(null);
        setSelectedAuthMethod(null);
      }
    }
  // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [loginData, showingLogin]);

  // Permission hooks - always initialized for simplicity
  const accessibility = useAccessibilityPermission(showingPermissions);
  const fullDiskAccess = useFullDiskAccessPermission(showingPermissions);
  const screenRecording = useScreenRecordingPermission(showingPermissions);
  const microphone = useMicrophonePermission(showingPermissions);

  // Session hooks
  const { data: sessions = [], isLoading: sessionsLoading } = useSessionsList();
  const activeSession = useActiveSession(sessionId);

  // MCP hooks
  const { data: mcpServers = [], isLoading: mcpLoading } = useMCPList();

  const permissions = [
    {
      id: 'accessibility',
      label: 'Accessibility',
      icon: Accessibility,
      hook: accessibility,
    },
    {
      id: 'fullDiskAccess',
      label: 'Full Disk Access',
      icon: Folder,
      hook: fullDiskAccess,
    },
    {
      id: 'screenRecording',
      label: 'Screen Recording',
      icon: Monitor,
      hook: screenRecording,
    },
    {
      id: 'microphone',
      label: 'Microphone',
      icon: Mic,
      hook: microphone,
    },
  ];

  // Filter commands based on search query
  const filteredCommands = searchQuery.trim()
    ? slashCommands.filter(
        (command) =>
          command.name.toLowerCase().includes(searchQuery.toLowerCase()) ||
          command.description.toLowerCase().includes(searchQuery.toLowerCase())
      )
    : slashCommands;

  // Filter permissions based on search query
  const filteredPermissions = searchQuery.trim()
    ? permissions.filter((permission) =>
        permission.label.toLowerCase().includes(searchQuery.toLowerCase())
      )
    : permissions;

  // Sort sessions chronologically (most recent first) and filter by search
  const sortedAndFilteredSessions = sessions
    .sort(
      (a, b) =>
        new Date(b.createdAt).getTime() - new Date(a.createdAt).getTime()
    )
    .filter(
      (session) =>
        !searchQuery.trim() ||
        session.title.toLowerCase().includes(searchQuery.toLowerCase())
    );

  // Filter MCP servers based on search query
  const filteredMCPServers = searchQuery.trim()
    ? mcpServers.filter((server) =>
        server.name.toLowerCase().includes(searchQuery.toLowerCase())
      )
    : mcpServers;

  // Get tools for selected MCP server
  const selectedServerTools = selectedMCPServer
    ? mcpServers.find((s) => s.name === selectedMCPServer)?.tools || []
    : [];

  // Filter tools based on search query
  const filteredMCPTools = searchQuery.trim()
    ? selectedServerTools.filter(
        (tool) =>
          tool.name.toLowerCase().includes(searchQuery.toLowerCase()) ||
          tool.description.toLowerCase().includes(searchQuery.toLowerCase())
      )
    : selectedServerTools;

  // Filter providers for hierarchical model selection
  const filteredProviders = hierarchicalModelData?.providers.filter(
    (provider) =>
      !searchQuery.trim() ||
      provider.displayName.toLowerCase().includes(searchQuery.toLowerCase())
  ) || [];

  // Get models for selected provider
  const selectedProviderModels = selectedProvider
    ? hierarchicalModelData?.providers.find((p) => p.id === selectedProvider)?.models || []
    : [];

  // Filter models based on search query
  const filteredModels = searchQuery.trim()
    ? selectedProviderModels.filter((model) =>
        model.displayName.toLowerCase().includes(searchQuery.toLowerCase())
      )
    : selectedProviderModels;
    
  // Filter providers for logout
  const filteredLogoutProviders = logoutData?.providers.filter(
    (provider) =>
      !searchQuery.trim() ||
      provider.displayName.toLowerCase().includes(searchQuery.toLowerCase())
  ) || [];
  
  // Filter providers for status
  const filteredStatusProviders = statusData?.providers.filter(
    (provider) =>
      !searchQuery.trim() ||
      provider.displayName.toLowerCase().includes(searchQuery.toLowerCase())
  ) || [];
  
  // Filter providers for login
  const filteredLoginProviders = loginData?.providers.filter(
    (provider) =>
      !searchQuery.trim() ||
      provider.displayName.toLowerCase().includes(searchQuery.toLowerCase())
  ) || [];
  
  // Filter auth methods for selected provider in login view
  const selectedLoginProvider = selectedProvider ? 
    loginData?.providers.find((p) => p.id === selectedProvider) : 
    undefined;
  const filteredAuthMethods = selectedLoginProvider?.authMethods.filter(
    (method) =>
      !searchQuery.trim() ||
      method.toLowerCase().includes(searchQuery.toLowerCase())
  ) || [];


  const handleSelect = (value: string) => {
    setSearchQuery('');
    setSelectedValue('');

    if (value === 'back-to-commands') {
      // Override any existing data by calling parent's callback to reset data
      // Note: this must happen BEFORE we reset our local view flags
      if (onExecuteCommand) {
        // This will reset loginData/statusData/logoutData in parent component
        // by calling a dummy command that doesn't exist
        onExecuteCommand('__reset__');
      }
      
      // Reset selection states first
      setSelectedMCPServer(null);
      setSelectedProvider(null);
      setSelectedAuthMethod(null);
      
      // Then reset all view flags
      setShowingPermissions(false);
      setShowingSessions(false);
      setShowingMCP(false);
      setShowingHierarchicalModel(false);
      setShowingLogout(false);
      setShowingStatus(false);
      setShowingLogin(false);
      
      return;
    }

    if (value === 'back-to-providers') {
      setSelectedProvider(null);
      setSelectedAuthMethod(null);
      return;
    }
    
    if (value === 'back-to-auth-methods') {
      setSelectedAuthMethod(null);
      return;
    }
    
    // Handle login provider and auth method selection
    if (showingLogin) {
      // If we have a provider selected but no auth method yet
      if (selectedProvider && !selectedAuthMethod) {
        // Check if the value is a valid auth method
        if (value === 'api_key' || value === 'oauth') {
          setSelectedAuthMethod(value);
          
          // If auth method is oauth, keep the command menu open and prepare for OAuth
          // Don't call the handler immediately, let the user click the OAuth button
          return;
        }
      } 
      // If we already have both provider and auth method selected
      else if (selectedProvider && selectedAuthMethod) {
        // If clicked on the auth input item, call the handler
        if (value === 'auth-input' && (selectedAuthMethod === 'api_key' || selectedAuthMethod === 'oauth')) {
          onLoginProviderSelect?.(selectedProvider, selectedAuthMethod);
          onClose();
        }
        return;
      }
      // If we don't have a provider selected yet
      else {
        const provider = loginData?.providers.find((p) => p.id === value);
        if (provider) {
          setSelectedProvider(provider.id);
          
          // If the provider only has one auth method, auto-select it
          if (provider.authMethods.length === 1) {
            setSelectedAuthMethod(provider.authMethods[0]);
            onLoginProviderSelect?.(provider.id, provider.authMethods[0]);
            return;
          }
          return;
        }
      }
    }

    // Handle logout provider selection
    if (showingLogout) {
      const provider = logoutData?.providers.find((p) => p.id === value);
      if (provider) {
        onLogoutProviderSelect?.(provider.id);
        return;
      }
    }
    
    // Handle status provider selection
    if (showingStatus) {
      const provider = statusData?.providers.find((p) => p.id === value);
      if (provider) {
        onStatusProviderSelect?.(provider.id);
        return;
      }
    }

    if (value === 'permissions') {
      setShowingPermissions(true);
      setShowingSessions(false);
      setShowingMCP(false);
      setShowingHierarchicalModel(false);
      setSelectedMCPServer(null);
      setSelectedProvider(null);

      return;
    }

    if (value === 'sessions') {
      setShowingSessions(true);
      setShowingPermissions(false);
      setShowingMCP(false);
      setShowingHierarchicalModel(false);
      setSelectedMCPServer(null);
      setSelectedProvider(null);

      return;
    }

    if (value === 'mcp') {
      setShowingMCP(true);
      setShowingPermissions(false);
      setShowingSessions(false);
      setShowingHierarchicalModel(false);
      setSelectedMCPServer(null);
      setSelectedProvider(null);

      return;
    }

    // Handle session selection - direct navigation (stateless design)
    const session = sessions.find((s) => s.id === value);
    if (session) {
      // Navigate directly to the selected session
      navigate({
        to: '/$sessionId',
        params: { sessionId: session.id },
        replace: true,
      });
      onClose(); // Close the command palette

      return;
    }

    // Handle MCP server selection
    const mcpServer = mcpServers.find((s) => s.name === value);
    if (mcpServer) {
      setSelectedMCPServer(mcpServer.name);

      return;
    }

    // Handle hierarchical provider selection
    if (showingHierarchicalModel && !selectedProvider) {
      const provider = hierarchicalModelData?.providers.find((p) => p.id === value);
      if (provider) {
        if (!provider.authenticated) {
          // Don't allow selection of unauthenticated providers
          return;
        }
        setSelectedProvider(provider.id);
        onProviderSelect?.(provider.id);
        return;
      }
    }

    // Handle hierarchical model selection
    if (showingHierarchicalModel && selectedProvider) {
      const provider = hierarchicalModelData?.providers.find((p) => p.id === selectedProvider);
      const model = provider?.models.find((m) => m.id === value);
      if (model) {
        onModelSelect?.(selectedProvider, model.id);
        onClose(); // Close the command palette after model selection
        return;
      }
    }

    // Handle permission toggles
    const permission = permissions.find((p) => p.id === value);
    if (permission && !permission.hook.isGranted) {
      permission.hook.request();

      return;
    }

    // Handle regular commands
    const command = slashCommands.find((c) => c.id === value);
    if (command) {
      onExecuteCommand(command.name);
    }
  };

  const handleKeyDown = (e: React.KeyboardEvent) => {
    if (e.key === 'Escape') {
      e.preventDefault();
      if (selectedAuthMethod && showingLogin) {
        setSelectedAuthMethod(null);
      } else if (selectedProvider) {
        setSelectedProvider(null);
      } else if (selectedMCPServer) {
        setSelectedMCPServer(null);
      } else if (showingLogin) {
        setShowingLogin(false);
      } else if (showingLogout) {
        setShowingLogout(false);
      } else if (showingStatus) {
        setShowingStatus(false);
      } else if (showingHierarchicalModel) {
        setShowingHierarchicalModel(false);
      } else if (showingMCP) {
        setShowingMCP(false);
      } else if (showingPermissions) {
        setShowingPermissions(false);
      } else if (showingSessions) {
        setShowingSessions(false);
      } else {
        onClose();
      }
    }
  };

  return (
    <div className="absolute right-0 bottom-full left-0 z-50 mb-2 overflow-hidden rounded-xl border border-border bg-popover shadow-lg">
      <CommandPrimitive
        className="max-h-64"
        onKeyDown={handleKeyDown}
        onValueChange={setSelectedValue}
        ref={commandRef}
        value={selectedValue}
      >
        <CommandInput
          autoFocus
          onValueChange={setSearchQuery}
          placeholder={
            selectedProvider && showingLogin && selectedAuthMethod
              ? 'Enter API key or OAuth code...'
              : selectedProvider && showingLogin
                ? 'Search auth methods...'
                : selectedProvider
                  ? 'Search models...'
                  : selectedMCPServer
                    ? 'Search tools...'
                    : showingHierarchicalModel
                      ? 'Search providers...'
                      : showingMCP
                        ? 'Search MCP servers...'
                        : showingPermissions
                          ? 'Search permissions...'
                          : showingSessions
                            ? 'Search sessions...'
                            : showingLogin
                              ? 'Search providers...'
                              : 'Search commands...'
          }
          value={searchQuery}
        />

        <CommandList>
          {showingSessions ? (
            // Sessions View
            <>
              {sessionsLoading ? (
                <CommandEmpty>Loading sessions...</CommandEmpty>
              ) : !sortedAndFilteredSessions.length && searchQuery ? (
                <CommandEmpty>No sessions match your search</CommandEmpty>
              ) : sortedAndFilteredSessions.length ? (
                <CommandGroup
                  heading={`Sessions (${sortedAndFilteredSessions.length})`}
                >
                  {/* Back to Commands */}
                  <CommandItem
                    onSelect={() => handleSelect('back-to-commands')}
                    value="back-to-commands"
                  >
                    <ArrowLeft className="size-4 text-muted-foreground" />
                    <div className="flex-1">
                      <div className="font-medium text-sm">
                        Back to Commands
                      </div>
                    </div>
                  </CommandItem>

                  {/* Session Items */}
                  {sortedAndFilteredSessions.map((session) => {
                    const isActive = activeSession.data?.id === session.id;
                    const createdDate = new Date(session.createdAt);
                    const formatDate = (date: Date) => {
                      const now = new Date();
                      const diffDays = Math.floor(
                        (now.getTime() - date.getTime()) / (1000 * 60 * 60 * 24)
                      );

                      if (diffDays === 0) return 'Today';
                      if (diffDays === 1) return 'Yesterday';
                      if (diffDays < 7) return `${diffDays} days ago`;
                      return date.toLocaleDateString();
                    };

                    return (
                      <CommandItem
                        className={isActive ? 'bg-accent' : ''}
                        key={session.id}
                        onSelect={() => handleSelect(session.id)}
                        value={session.id}
                      >
                        <Clock className="size-4 text-muted-foreground" />
                        <div className="flex-1">
                          <div className="flex items-center gap-2 font-medium text-sm">
                            {getDisplayTitle(session)}
                            {isActive && (
                              <span className="rounded-full bg-primary px-1.5 py-0.5 text-primary-foreground text-xs">
                                current
                              </span>
                            )}
                          </div>
                          <div className="flex items-center gap-2 text-muted-foreground text-xs">
                            <span>{formatDate(createdDate)}</span>
                            <span>•</span>
                            <span>{formatMessageCounts(session)}</span>
                          </div>
                        </div>
                        <div className="ml-2 font-mono text-muted-foreground text-xs">
                          {session.id.slice(0, 8)}
                        </div>
                      </CommandItem>
                    );
                  })}
                </CommandGroup>
              ) : (
                <CommandEmpty>No sessions found</CommandEmpty>
              )}
            </>
          ) : showingPermissions ? (
            // Permissions View
            <>
              {!filteredPermissions.length && searchQuery ? (
                <CommandEmpty>No permissions match your search</CommandEmpty>
              ) : (
                <CommandGroup heading="System Permissions">
                  {/* Back to Commands */}
                  <CommandItem
                    onSelect={() => handleSelect('back-to-commands')}
                    value="back-to-commands"
                  >
                    <ArrowLeft className="size-4 text-muted-foreground" />
                    <div className="flex-1">
                      <div className="font-medium text-sm">
                        Back to Commands
                      </div>
                    </div>
                  </CommandItem>

                  {/* Permission Items */}
                  {filteredPermissions.map((permission) => {
                    const Icon = permission.icon;
                    return (
                      <CommandItem
                        className="flex items-center justify-between"
                        key={permission.id}
                        onSelect={() => handleSelect(permission.id)}
                        value={permission.id}
                      >
                        <div className="flex flex-1 items-center gap-3">
                          <Icon className="size-4 text-muted-foreground" />
                          <div className="flex-1">
                            <div className="font-medium text-sm">
                              {permission.label}
                            </div>
                            <div className="text-muted-foreground text-xs">
                              {permission.hook.isGranted
                                ? 'Granted'
                                : 'Not granted'}
                            </div>
                          </div>
                        </div>
                        <Switch
                          checked={permission.hook.isGranted}
                          disabled={
                            permission.hook.isLoading ||
                            permission.hook.isRequesting
                          }
                          onCheckedChange={(checked) => {
                            if (!checked) return; // Only allow requesting, not revoking
                            if (!permission.hook.isGranted) {
                              permission.hook.request();
                            }
                          }}
                          onClick={(e) => e.stopPropagation()}
                        />
                      </CommandItem>
                    );
                  })}
                </CommandGroup>
              )}
            </>
          ) : selectedMCPServer ? (
            // MCP Tools View
            <>
              {!filteredMCPTools.length && searchQuery ? (
                <CommandEmpty>No tools match your search</CommandEmpty>
              ) : filteredMCPTools.length ? (
                <CommandGroup
                  heading={`${selectedMCPServer} Tools (${filteredMCPTools.length})`}
                >
                  {/* Back to MCP Servers */}
                  <CommandItem
                    onSelect={() => setSelectedMCPServer(null)}
                    value="back-to-mcp"
                  >
                    <ArrowLeft className="size-4 text-muted-foreground" />
                    <div className="flex-1">
                      <div className="font-medium text-sm">
                        Back to MCP Servers
                      </div>
                    </div>
                  </CommandItem>

                  {/* Tool Items */}
                  {filteredMCPTools.map((tool) => {
                    const serverInfo = mcpServers.find(
                      (s) => s.name === selectedMCPServer
                    );
                    return (
                      <CommandItem
                        className="cursor-default"
                        key={tool.name}
                        value={tool.name}
                      >
                        <Settings className="size-4 text-muted-foreground" />
                        <div className="flex-1">
                          <div className="font-medium text-sm">{tool.name}</div>
                          <div className="text-muted-foreground text-xs">
                            {tool.description}
                          </div>
                        </div>
                        <div
                          className={`rounded-full px-2 py-0.5 text-xs ${serverInfo?.connected ? 'bg-green-100 text-green-800 dark:bg-green-800/20 dark:text-green-400' : 'bg-red-100 text-red-800 dark:bg-red-800/20 dark:text-red-400'}`}
                        >
                          {serverInfo?.connected ? 'connected' : 'disconnected'}
                        </div>
                      </CommandItem>
                    );
                  })}
                </CommandGroup>
              ) : (
                <CommandEmpty>No tools found</CommandEmpty>
              )}
            </>
          ) : showingMCP ? (
            // MCP Servers View
            <>
              {mcpLoading ? (
                <CommandEmpty>Loading MCP servers...</CommandEmpty>
              ) : !filteredMCPServers.length && searchQuery ? (
                <CommandEmpty>No servers match your search</CommandEmpty>
              ) : filteredMCPServers.length ? (
                <CommandGroup
                  heading={`MCP Servers (${filteredMCPServers.length})`}
                >
                  {/* Back to Commands */}
                  <CommandItem
                    onSelect={() => handleSelect('back-to-commands')}
                    value="back-to-commands"
                  >
                    <ArrowLeft className="size-4 text-muted-foreground" />
                    <div className="flex-1">
                      <div className="font-medium text-sm">
                        Back to Commands
                      </div>
                    </div>
                  </CommandItem>

                  {/* MCP Server Items */}
                  {filteredMCPServers.map((server) => (
                    <CommandItem
                      key={server.name}
                      onSelect={() => handleSelect(server.name)}
                      value={server.name}
                    >
                      <Plug className="size-4 text-muted-foreground" />
                      <div className="flex-1">
                        <div className="flex items-center gap-2 font-medium text-sm">
                          {server.name}
                          <div
                            className={`rounded-full px-2 py-0.5 text-xs ${server.connected ? 'bg-green-100 text-green-800 dark:bg-green-800/20 dark:text-green-400' : 'bg-red-100 text-red-800 dark:bg-red-800/20 dark:text-red-400'}`}
                          >
                            {server.status}
                          </div>
                        </div>
                        <div className="text-muted-foreground text-xs">
                          {server.tools?.length || 0} tools available
                        </div>
                      </div>
                    </CommandItem>
                  ))}
                </CommandGroup>
              ) : (
                <CommandEmpty>No MCP servers found</CommandEmpty>
              )}
            </>
          ) : showingStatus ? (
            // Status Provider Selection View
            <>
              {!filteredStatusProviders.length && searchQuery ? (
                <CommandEmpty>No providers match your search</CommandEmpty>
              ) : filteredStatusProviders.length ? (
                <CommandGroup
                  heading={`Providers (${filteredStatusProviders.length})`}
                >
                  {/* Back to Commands */}
                  <CommandItem
                    onSelect={() => handleSelect('back-to-commands')}
                    value="back-to-commands"
                  >
                    <ArrowLeft className="size-4 text-muted-foreground" />
                    <div className="flex-1">
                      <div className="font-medium text-sm">
                        Back to Commands
                      </div>
                    </div>
                  </CommandItem>

                  {/* Provider Items */}
                  {filteredStatusProviders.map((provider) => (
                    <CommandItem
                      key={provider.id}
                      onSelect={() => handleSelect(provider.id)}
                      value={provider.id}
                      className={!provider.authenticated ? 'opacity-50' : ''}
                    >
                      <Settings className="size-4 text-muted-foreground" />
                      <div className="flex-1">
                        <div className="flex items-center gap-2 font-medium text-sm">
                          {provider.displayName}
                          {provider.authenticated && (
                            <CheckCircle className="h-4 w-4 text-green-600" />
                          )}
                          {provider.isPreferred && (
                            <span className="rounded-full bg-primary px-1.5 py-0.5 text-primary-foreground text-xs">
                              preferred
                            </span>
                          )}
                        </div>
                        <div className="text-muted-foreground text-xs">
                          {provider.authenticated
                            ? `Authenticated`
                            : 'Not authenticated - select to authenticate'
                          }
                        </div>
                      </div>
                    </CommandItem>
                  ))}
                </CommandGroup>
              ) : (
                <CommandEmpty>No providers found</CommandEmpty>
              )}
            </>
          ) : showingLogin && selectedProvider && selectedAuthMethod ? (
            // Login Auth Method Detail View (API Key input or OAuth code input)
            <>
              <CommandGroup heading={`Enter ${selectedAuthMethod === 'api_key' ? 'API Key' : 'OAuth Code'}`}>
                {/* Back to Auth Methods */}
                <CommandItem
                  onSelect={() => handleSelect('back-to-auth-methods')}
                  value="back-to-auth-methods"
                >
                  <ArrowLeft className="size-4 text-muted-foreground" />
                  <div className="flex-1">
                    <div className="font-medium text-sm">
                      Back to Auth Methods
                    </div>
                  </div>
                </CommandItem>
                
                {/* Content based on auth method */}
                <div>
                  {selectedAuthMethod === 'api_key' && (
                    // API Key input field
                    <div className="p-2 border-b">
                      <div className="font-medium text-sm mb-1">
                        Enter API Key for {selectedLoginProvider?.displayName}:
                      </div>
                      <div className="text-muted-foreground text-xs mb-2">
                        Format: {selectedLoginProvider?.apiKeyFormat || "API key"}
                      </div>
                      <div className="flex gap-2">
                        <input
                          type="password"
                          value={apiKey}
                          onChange={(e) => setApiKey(e.target.value)}
                          placeholder={selectedLoginProvider?.apiKeyFormat || "Enter API key"}
                          disabled={apiKeySubmitting}
                          className="flex-1 rounded-md border border-input bg-background px-3 py-2 text-sm focus:outline-none focus:ring-2 focus:ring-primary"
                          onKeyDown={(e) => {
                            if (e.key === 'Enter' && selectedProvider && apiKey) {
                              e.preventDefault();
                              handleApiKeySubmit();
                            }
                          }}
                        />
                        <button
                          onClick={handleApiKeySubmit}
                          disabled={!apiKey || apiKeySubmitting}
                          className={`rounded-md px-3 py-2 text-sm font-medium ${!apiKey || apiKeySubmitting ? 'bg-muted text-muted-foreground' : 'bg-primary text-primary-foreground'}`}
                        >
                          {apiKeySubmitting ? 'Submitting...' : 'Submit'}
                        </button>
                      </div>
                    </div>
                  )}
                  
                  {selectedAuthMethod === "oauth_code" && (
                    // OAuth code input field
                    <div className="p-3 border-b">
                      <div className="font-medium text-sm mb-1">
                        Enter OAuth code for {selectedLoginProvider?.displayName}:
                      </div>
                      <div className="text-muted-foreground text-xs mb-2">
                        After authorizing in your browser, paste the code here
                      </div>
                      <div className="flex gap-2">
                        <input
                          type="text"
                          value={oauthCode}
                          onChange={(e) => setOauthCode(e.target.value)}
                          placeholder="Authorization code"
                          disabled={oauthCodeSubmitting}
                          className="flex-1 rounded-md border border-input bg-background px-3 py-2 text-sm focus:outline-none focus:ring-2 focus:ring-primary"
                          onKeyDown={(e) => {
                            if (e.key === 'Enter' && selectedProvider && oauthCode) {
                              e.preventDefault();
                              e.stopPropagation();
                              handleOAuthCodeSubmit();
                            }
                          }}
                        />
                        <button
                          onClick={(e) => {
                            e.preventDefault();
                            e.stopPropagation();
                            handleOAuthCodeSubmit();
                          }}
                          disabled={!oauthCode || oauthCodeSubmitting}
                          className={`rounded-md px-3 py-2 text-sm font-medium ${!oauthCode || oauthCodeSubmitting ? 'bg-muted text-muted-foreground' : 'bg-primary text-primary-foreground'}`}
                        >
                          {oauthCodeSubmitting ? 'Submitting...' : 'Submit'}
                        </button>
                      </div>
                    </div>
                  )}
                  
                  {selectedAuthMethod === "oauth" && (
                    // OAuth flow button
                    <div className="p-3 border-b">
                      <div className="font-medium text-sm mb-1">
                        Start OAuth flow for {selectedLoginProvider?.displayName}:
                      </div>
                      <div className="text-muted-foreground text-xs mb-2">
                        You'll be redirected to authorize in your browser
                      </div>
                      <button
                        onClick={(e) => {
                          // When clicked, call the onLoginProviderSelect handler directly
                          // Stop event propagation to prevent the CMDK item from capturing it
                          e.stopPropagation();
                          e.preventDefault();
                          
                          if (selectedProvider && selectedAuthMethod === 'oauth' && onLoginProviderSelect) {
                            // Start OAuth flow without closing the command palette
                            onLoginProviderSelect(selectedProvider, 'oauth');
                            
                            // After starting OAuth flow, immediately switch to code input mode
                            setTimeout(() => {
                              setSelectedAuthMethod("oauth_code");
                            }, 500);
                          }
                        }}
                        className="w-full rounded-md px-3 py-2 text-sm font-medium bg-primary text-primary-foreground"
                      >
                        Start OAuth Authorization
                      </button>
                    </div>
                  )}
                </div>
              </CommandGroup>
            </>
          ) : showingLogin && selectedProvider ? (
            // Login Auth Method Selection View
            <>
              {!filteredAuthMethods.length && searchQuery ? (
                <CommandEmpty>No auth methods match your search</CommandEmpty>
              ) : filteredAuthMethods.length ? (
                <CommandGroup
                  heading={`Authentication Methods for ${selectedLoginProvider?.displayName}`}
                >
                  {/* Back to Providers */}
                  <CommandItem
                    onSelect={() => handleSelect('back-to-providers')}
                    value="back-to-providers"
                  >
                    <ArrowLeft className="size-4 text-muted-foreground" />
                    <div className="flex-1">
                      <div className="font-medium text-sm">
                        Back to Providers
                      </div>
                    </div>
                  </CommandItem>

                  {/* Auth Method Items */}
                  {filteredAuthMethods.map((method) => (
                    <CommandItem
                      key={method}
                      onSelect={() => handleSelect(method)}
                      value={method}
                    >
                      <Settings className="size-4 text-muted-foreground" />
                      <div className="flex-1">
                        <div className="font-medium text-sm">
                          {method === 'api_key' ? 'API Key' : 'OAuth'}
                        </div>
                        <div className="text-muted-foreground text-xs">
                          {method === 'api_key' 
                            ? 'Enter your API key directly' 
                            : 'Connect through web authorization'}
                        </div>
                      </div>
                    </CommandItem>
                  ))}
                </CommandGroup>
              ) : (
                <CommandEmpty>No auth methods available</CommandEmpty>
              )}
            </>
          ) : showingLogin ? (
            // Login Provider Selection View
            <>
              {!filteredLoginProviders.length && searchQuery ? (
                <CommandEmpty>No providers match your search</CommandEmpty>
              ) : filteredLoginProviders.length ? (
                <CommandGroup
                  heading={`Providers (${filteredLoginProviders.length})`}
                >
                  {/* Back to Commands */}
                  <CommandItem
                    onSelect={() => handleSelect('back-to-commands')}
                    value="back-to-commands"
                  >
                    <ArrowLeft className="size-4 text-muted-foreground" />
                    <div className="flex-1">
                      <div className="font-medium text-sm">
                        Back to Commands
                      </div>
                    </div>
                  </CommandItem>

                  {/* Provider Items */}
                  {filteredLoginProviders.map((provider) => (
                    <CommandItem
                      key={provider.id}
                      onSelect={() => handleSelect(provider.id)}
                      value={provider.id}
                    >
                      <Settings className="size-4 text-muted-foreground" />
                      <div className="flex-1">
                        <div className="flex items-center gap-2 font-medium text-sm">
                          {provider.displayName}
                          {provider.authenticated && (
                            <CheckCircle className="h-4 w-4 text-green-600" />
                          )}
                          {provider.isPreferred && (
                            <span className="rounded-full bg-primary px-1.5 py-0.5 text-primary-foreground text-xs">
                              preferred
                            </span>
                          )}
                        </div>
                        <div className="text-muted-foreground text-xs">
                          {provider.authenticated 
                            ? `Authenticated` 
                            : `Supports: ${provider.authMethods.map(m => m === 'api_key' ? 'API Key' : 'OAuth').join(', ')}`}
                        </div>
                      </div>
                    </CommandItem>
                  ))}
                </CommandGroup>
              ) : (
                <CommandEmpty>No providers found</CommandEmpty>
              )}
            </>
          ) : showingLogout ? (
            // Logout Provider Selection View
            <>
              {!filteredLogoutProviders.length && searchQuery ? (
                <CommandEmpty>No providers match your search</CommandEmpty>
              ) : filteredLogoutProviders.length ? (
                <CommandGroup
                  heading={`Providers (${filteredLogoutProviders.length})`}
                >
                  {/* Back to Commands */}
                  <CommandItem
                    onSelect={() => handleSelect('back-to-commands')}
                    value="back-to-commands"
                  >
                    <ArrowLeft className="size-4 text-muted-foreground" />
                    <div className="flex-1">
                      <div className="font-medium text-sm">
                        Back to Commands
                      </div>
                    </div>
                  </CommandItem>

                  {/* Provider Items */}
                  {filteredLogoutProviders.map((provider) => (
                    <CommandItem
                      key={provider.id}
                      onSelect={() => handleSelect(provider.id)}
                      value={provider.id}
                    >
                      <Settings className="size-4 text-muted-foreground" />
                      <div className="flex-1">
                        <div className="flex items-center gap-2 font-medium text-sm">
                          {provider.displayName}
                          {provider.isPreferred && (
                            <span className="rounded-full bg-primary px-1.5 py-0.5 text-primary-foreground text-xs">
                              preferred
                            </span>
                          )}
                        </div>
                        <div className="text-muted-foreground text-xs">
                          {provider.authenticated && `Authenticated`}
                        </div>
                      </div>
                    </CommandItem>
                  ))}
                </CommandGroup>
              ) : (
                <CommandEmpty>No authenticated providers found</CommandEmpty>
              )}
            </>
          ) : selectedProvider ? (
            // Hierarchical Model Selection - Models View
            <>
              {!filteredModels.length && searchQuery ? (
                <CommandEmpty>No models match your search</CommandEmpty>
              ) : filteredModels.length ? (
                <CommandGroup
                  heading={`${hierarchicalModelData?.providers.find(p => p.id === selectedProvider)?.displayName} Models (${filteredModels.length})`}
                >
                  {/* Back to Providers */}
                  <CommandItem
                    onSelect={() => handleSelect('back-to-providers')}
                    value="back-to-providers"
                  >
                    <ArrowLeft className="size-4 text-muted-foreground" />
                    <div className="flex-1">
                      <div className="font-medium text-sm">
                        Back to Providers
                      </div>
                    </div>
                  </CommandItem>

                  {/* Model Items */}
                  {filteredModels.map((model) => (
                    <CommandItem
                      key={model.id}
                      onSelect={() => handleSelect(model.id)}
                      value={model.id}
                    >
                      <Settings className="size-4 text-muted-foreground" />
                      <div className="flex-1">
                        <div className="flex items-center gap-2 font-medium text-sm">
                          {model.displayName}
                          {model.isSelected && (
                            <span className="rounded-full bg-primary px-1.5 py-0.5 text-primary-foreground text-xs">
                              current
                            </span>
                          )}
                        </div>
                      </div>
                    </CommandItem>
                  ))}
                </CommandGroup>
              ) : (
                <CommandEmpty>No models found</CommandEmpty>
              )}
            </>
          ) : showingHierarchicalModel ? (
            // Hierarchical Model Selection - Providers View
            <>
              {!filteredProviders.length && searchQuery ? (
                <CommandEmpty>No providers match your search</CommandEmpty>
              ) : filteredProviders.length ? (
                <CommandGroup
                  heading={`Providers (${filteredProviders.length})`}
                >
                  {/* Back to Commands */}
                  <CommandItem
                    onSelect={() => handleSelect('back-to-commands')}
                    value="back-to-commands"
                  >
                    <ArrowLeft className="size-4 text-muted-foreground" />
                    <div className="flex-1">
                      <div className="font-medium text-sm">
                        Back to Commands
                      </div>
                    </div>
                  </CommandItem>

                  {/* Provider Items */}
                  {filteredProviders.map((provider) => (
                    <CommandItem
                      className={!provider.authenticated ? 'opacity-50 cursor-not-allowed' : ''}
                      key={provider.id}
                      onSelect={() => handleSelect(provider.id)}
                      value={provider.id}
                    >
                      <Settings className="size-4 text-muted-foreground" />
                      <div className="flex-1">
                        <div className="flex items-center gap-2 font-medium text-sm">
                          {provider.displayName}
                          {provider.isPreferred && (
                            <span className="rounded-full bg-primary px-1.5 py-0.5 text-primary-foreground text-xs">
                              preferred
                            </span>
                          )}
                          {!provider.authenticated && (
                            <span className="rounded-full bg-red-100 px-1.5 py-0.5 text-red-800 text-xs dark:bg-red-800/20 dark:text-red-400">
                              not authenticated
                            </span>
                          )}
                        </div>
                        <div className="text-muted-foreground text-xs">
                          {provider.models.length} models available
                          {provider.authenticated && ` • Authenticated`}
                        </div>
                      </div>
                    </CommandItem>
                  ))}
                </CommandGroup>
              ) : (
                <CommandEmpty>No providers found</CommandEmpty>
              )}
            </>
          ) : (
            // Commands View
            <>
              {filteredCommands.length ? (
                <CommandGroup heading="Commands">
                  {filteredCommands.map((command) => {
                    const Icon = command.icon;
                    return (
                      <CommandItem
                        key={command.id}
                        onSelect={() => handleSelect(command.id)}
                        value={command.id}
                      >
                        <Icon className="size-4 text-muted-foreground" />
                        <div className="flex-1">
                          <div className="font-medium text-sm">
                            {command.name}
                          </div>
                          <div className="text-muted-foreground text-xs">
                            {command.description}
                          </div>
                        </div>
                      </CommandItem>
                    );
                  })}
                </CommandGroup>
              ) : (
                <CommandEmpty>
                  {searchQuery
                    ? 'No commands match your search'
                    : 'No commands found'}
                </CommandEmpty>
              )}
            </>
          )}
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
                {selectedProvider ||
                selectedMCPServer ||
                showingHierarchicalModel ||
                showingMCP ||
                showingPermissions ||
                showingSessions ||
                showingLogout ||
                showingStatus
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
