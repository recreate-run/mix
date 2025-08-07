import {
  Accessibility,
  ArrowLeft,
  Clock,
  Folder,
  Mic,
  Monitor,
} from 'lucide-react';
import { useEffect, useRef, useState } from 'react';
import type { MessageData } from '@/components/chat-app';
import {
  CommandEmpty,
  CommandGroup,
  CommandInput,
  CommandItem,
  CommandList,
  Command as CommandPrimitive,
} from '@/components/ui/command';
import { Switch } from '@/components/ui/switch';
import {
  useAccessibilityPermission,
  useFullDiskAccessPermission,
  useMicrophonePermission,
  useScreenRecordingPermission,
} from '@/hooks/usePermissions';
import { useActiveSession } from '@/hooks/useSession';
import {
  TITLE_TRUNCATE_LENGTH,
  useSelectSession,
  useSessionsList,
} from '@/hooks/useSessionsList';
import { slashCommands } from '@/utils/slash-commands';

interface CommandSlashProps {
  onExecuteCommand: (command: string) => void;
  onClose: () => void;
}

export function CommandSlash({ onExecuteCommand, onClose }: CommandSlashProps) {
  const [selectedValue, setSelectedValue] = useState<string>('');
  const [searchQuery, setSearchQuery] = useState<string>('');
  const [showingPermissions, setShowingPermissions] = useState(false);
  const [showingSessions, setShowingSessions] = useState(false);
  const commandRef = useRef<HTMLDivElement>(null);

  // Reset selection when search query changes to prevent jumping
  useEffect(() => {
    setSelectedValue('');
  }, [searchQuery]);

  // Permission hooks - always initialized for simplicity
  const accessibility = useAccessibilityPermission(showingPermissions);
  const fullDiskAccess = useFullDiskAccessPermission(showingPermissions);
  const screenRecording = useScreenRecordingPermission(showingPermissions);
  const microphone = useMicrophonePermission(showingPermissions);

  // Session hooks
  const { data: sessions = [], isLoading: sessionsLoading } = useSessionsList();
  const selectSessionMutation = useSelectSession();
  const activeSession = useActiveSession();

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

  // Helper function to get display title (first user message or fallback to title)
  const getDisplayTitle = (session: (typeof sessions)[0]) => {
    if (!session.firstUserMessage || session.firstUserMessage.trim() === '') {
      return session.title; // fallback to original title
    }

    // Try to parse JSON and extract text from data.text field
    let displayText = session.firstUserMessage;
    try {
      const parsed = JSON.parse(session.firstUserMessage);
      if (parsed[0]?.data?.text) {
        // First parse the outer structure, then parse the inner JSON string
        const innerMessageData = JSON.parse(parsed[0].data.text) as MessageData;
        if (innerMessageData.text) {
          displayText = innerMessageData.text;
        }
      }
    } catch {
      // If parsing fails, use the raw message as fallback
      displayText = session.firstUserMessage;
      console.log('Failed to parse user message:', session.firstUserMessage);
    }

    const truncated =
      displayText.length > TITLE_TRUNCATE_LENGTH
        ? `${displayText.substring(0, TITLE_TRUNCATE_LENGTH)}...`
        : displayText;

    return truncated;
  };

  const handleSelect = (value: string) => {
    setSearchQuery('');
    setSelectedValue('');

    if (value === 'back-to-commands') {
      setShowingPermissions(false);
      setShowingSessions(false);

      return;
    }

    if (value === 'permissions') {
      setShowingPermissions(true);
      setShowingSessions(false);

      return;
    }

    if (value === 'sessions') {
      setShowingSessions(true);
      setShowingPermissions(false);

      return;
    }

    // Handle session selection
    const session = sessions.find((s) => s.id === value);
    if (session) {
      selectSessionMutation.mutate(session.id, {
        onSuccess: () => {
          onClose(); // Close the command palette
        },
      });

      return;
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
      if (showingPermissions) {
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
            showingPermissions
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
                            <span>{session.messageCount} messages</span>
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
                {showingPermissions || showingSessions ? 'back' : 'close'}
              </span>
            </div>
          </div>
        </div>
      </CommandPrimitive>
    </div>
  );
}
