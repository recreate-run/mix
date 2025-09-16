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
  onProviderSelect?: (providerId: string) => void;
  onModelSelect?: (providerId: string, modelId: string) => void;
  onLogoutProviderSelect?: (providerId: string) => void;
  onStatusProviderSelect?: (providerId: string) => void;
}

export function CommandSlash({
  onExecuteCommand,
  onClose,
  sessionId,
  hierarchicalModelData,
  logoutData,
  statusData,
  onProviderSelect,
  onModelSelect,
  onLogoutProviderSelect,
  onStatusProviderSelect,
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
  const [selectedProvider, setSelectedProvider] = useState<string | null>(null);
  const commandRef = useRef<HTMLDivElement>(null);
  const hierarchicalModelInitializedRef = useRef(false);
  const navigate = useNavigate();

  // Reset selection when search query changes to prevent jumping
  useEffect(() => {
    setSelectedValue('');
  }, [searchQuery]);

  // Show hierarchical model view when data is provided
  useEffect(() => {
    if (hierarchicalModelData) {
      setShowingHierarchicalModel(true);
      setShowingPermissions(false);
      setShowingSessions(false);
      setShowingMCP(false);
      setShowingLogout(false);
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
      setShowingPermissions(false);
      setShowingSessions(false);
      setShowingMCP(false);
      setSelectedMCPServer(null);
      setSelectedProvider(null);
    }
  }, [logoutData]);
  
  // Show status view when data is provided
  useEffect(() => {
    if (statusData) {
      setShowingStatus(true);
      setShowingLogout(false);
      setShowingHierarchicalModel(false);
      setShowingPermissions(false);
      setShowingSessions(false);
      setShowingMCP(false);
      setSelectedMCPServer(null);
      setSelectedProvider(null);
    }
  }, [statusData]);

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


  const handleSelect = (value: string) => {
    setSearchQuery('');
    setSelectedValue('');

    if (value === 'back-to-commands') {
      setShowingPermissions(false);
      setShowingSessions(false);
      setShowingMCP(false);
      setShowingHierarchicalModel(false);
      setShowingLogout(false);
      setShowingStatus(false);
      setSelectedMCPServer(null);
      setSelectedProvider(null);

      return;
    }

    if (value === 'back-to-providers') {
      setSelectedProvider(null);
      return;
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
      if (selectedProvider) {
        setSelectedProvider(null);
      } else if (selectedMCPServer) {
        setSelectedMCPServer(null);
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
            selectedProvider
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
                            ? `Authenticated via ${provider.authMethod === 'oauth' ? 'OAuth' : 'API Key'}`
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
                          {provider.authMethod && `Authenticated via ${provider.authMethod === 'oauth' ? 'OAuth' : 'API Key'}`}
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
                          {provider.authMethod && ` • ${provider.authMethod === 'oauth' ? 'OAuth' : 'API Key'}`}
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
