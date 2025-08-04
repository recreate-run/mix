import { useState, useRef } from 'react';
import { Shield, HelpCircle, Command, ArrowLeft, Accessibility, Folder, Monitor, Mic, RefreshCw } from 'lucide-react';
import { safeTrackEvent } from '@/lib/posthog';
import {
  Command as CommandPrimitive,
  CommandEmpty,
  CommandGroup,
  CommandInput,
  CommandItem,
  CommandList,
} from '@/components/ui/command';
import { Switch } from '@/components/ui/switch';
import { 
  useAccessibilityPermission,
  useFullDiskAccessPermission,
  useScreenRecordingPermission,
  useMicrophonePermission
} from '@/hooks/usePermissions';

interface SlashCommand {
  id: string;
  name: string;
  description: string;
  icon: React.ComponentType<{ className?: string }>;
}

interface CommandSlashProps {
  onExecuteCommand: (command: string) => void;
  onClose: () => void;
}

const slashCommands: SlashCommand[] = [
  {
    id: 'clear',
    name: 'clear',
    description: 'Start a new session',
    icon: RefreshCw,
  },
 
  {
    id: 'context',
    name: 'context',
    description: 'Show context usage breakdown',
    icon: Command,
  },
   {
    id: 'help',
    name: 'help',
    description: 'Get assistance and guidance',
    icon: HelpCircle,
  },
  {
    id: 'mcp',
    name: 'mcp',
    description: 'Model Context Protocol',
    icon: Command,
  },

  {
    id: 'permissions',
    name: 'permissions',
    description: 'System permissions and access',
    icon: Shield,
  },
];

export function CommandSlash({ onExecuteCommand, onClose }: CommandSlashProps) {
  const [selectedValue, setSelectedValue] = useState<string>('');
  const [searchQuery, setSearchQuery] = useState<string>('');
  const [showingPermissions, setShowingPermissions] = useState(false);
  const commandRef = useRef<HTMLDivElement>(null);
  
  // Permission hooks - always initialized for simplicity
  const accessibility = useAccessibilityPermission(showingPermissions);
  const fullDiskAccess = useFullDiskAccessPermission(showingPermissions);
  const screenRecording = useScreenRecordingPermission(showingPermissions);
  const microphone = useMicrophonePermission(showingPermissions);
  
  const permissions = [
    {
      id: 'accessibility',
      label: 'Accessibility',
      icon: Accessibility,
      hook: accessibility
    },
    {
      id: 'fullDiskAccess',
      label: 'Full Disk Access', 
      icon: Folder,
      hook: fullDiskAccess
    },
    {
      id: 'screenRecording',
      label: 'Screen Recording',
      icon: Monitor,
      hook: screenRecording
    },
    {
      id: 'microphone',
      label: 'Microphone',
      icon: Mic,
      hook: microphone
    }
  ];

  // Filter commands based on search query
  const filteredCommands = searchQuery.trim() 
    ? slashCommands.filter(command => 
        command.name.toLowerCase().includes(searchQuery.toLowerCase()) ||
        command.description.toLowerCase().includes(searchQuery.toLowerCase())
      )
    : slashCommands;

  // Filter permissions based on search query
  const filteredPermissions = searchQuery.trim()
    ? permissions.filter(permission =>
        permission.label.toLowerCase().includes(searchQuery.toLowerCase())
      )
    : permissions;
  
  const handleSelect = (value: string) => {
    setSearchQuery('');
    setSelectedValue('');
    
    if (value === 'back-to-commands') {
      setShowingPermissions(false);
      
      // Track navigation back to commands
      safeTrackEvent('command_menu_navigation', {
        action: 'back_to_commands',
        from: 'permissions',
        timestamp: new Date().toISOString()
      });
      return;
    }
    
    if (value === 'permissions') {
      setShowingPermissions(true);
      
      // Track navigation to permissions
      safeTrackEvent('command_menu_navigation', {
        action: 'view_permissions',
        timestamp: new Date().toISOString()
      });
      return;
    }
    
    // Handle permission toggles
    const permission = permissions.find(p => p.id === value);
    if (permission && !permission.hook.isGranted) {
      permission.hook.request();
      
      // Track permission request
      safeTrackEvent('permission_requested', {
        permission_type: permission.id,
        permission_label: permission.label,
        timestamp: new Date().toISOString()
      });
      return;
    }
    
    // Handle regular commands
    const command = slashCommands.find(c => c.id === value);
    if (command) {
      onExecuteCommand(command.name);
      
      // Track command execution
      safeTrackEvent('slash_command_executed', {
        command_id: command.id,
        command_name: command.name,
        timestamp: new Date().toISOString()
      });
    }
  };

  const handleKeyDown = (e: React.KeyboardEvent) => {
    if (e.key === 'Escape') {
      e.preventDefault();
      if (showingPermissions) {
        setShowingPermissions(false);
        
        // Track navigation back to commands with escape key
        safeTrackEvent('command_menu_navigation', {
          action: 'back_to_commands',
          from: 'permissions',
          method: 'escape_key',
          timestamp: new Date().toISOString()
        });
      } else {
        onClose();
        
        // Track command menu closed with escape key
        safeTrackEvent('command_menu_closed', {
          method: 'escape_key',
          timestamp: new Date().toISOString()
        });
      }
    }
  };

  return (
    <div className="absolute bottom-full left-0 right-0 mb-2 bg-popover border border-border rounded-xl shadow-lg z-50 overflow-hidden">
      <CommandPrimitive 
        ref={commandRef}
        onKeyDown={handleKeyDown} 
        className="max-h-64"
        value={selectedValue}
        onValueChange={setSelectedValue}
      >
        <CommandInput 
          placeholder={showingPermissions ? "Search permissions..." : "Search commands..."} 
          value={searchQuery}
          onValueChange={setSearchQuery}
          autoFocus
        />
        
        <CommandList>
          {showingPermissions ? (
            // Permissions View
            <>
              {!filteredPermissions.length && searchQuery ? (
                <CommandEmpty>No permissions match your search</CommandEmpty>
              ) : (
                <CommandGroup heading="System Permissions">
                  {/* Back to Commands */}
                  <CommandItem
                    value="back-to-commands"
                    onSelect={() => handleSelect('back-to-commands')}
                  >
                    <ArrowLeft className="size-4 text-muted-foreground" />
                    <div className="flex-1">
                      <div className="font-medium text-sm">Back to Commands</div>
                    </div>
                  </CommandItem>
                  
                  {/* Permission Items */}
                  {filteredPermissions.map((permission) => {
                    const Icon = permission.icon;
                    return (
                      <CommandItem
                        key={permission.id}
                        value={permission.id}
                        onSelect={() => handleSelect(permission.id)}
                        className="flex items-center justify-between"
                      >
                        <div className="flex items-center gap-3 flex-1">
                          <Icon className="size-4 text-muted-foreground" />
                          <div className="flex-1">
                            <div className="font-medium text-sm">{permission.label}</div>
                            <div className="text-xs text-muted-foreground">
                              {permission.hook.isGranted ? 'Granted' : 'Not granted'}
                            </div>
                          </div>
                        </div>
                        <Switch 
                          checked={permission.hook.isGranted} 
                          onCheckedChange={(checked) => {
                            if (!checked) return; // Only allow requesting, not revoking
                            if (!permission.hook.isGranted) {
                              permission.hook.request();
                            }
                          }}
                          disabled={permission.hook.isLoading || permission.hook.isRequesting}
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
              {!filteredCommands.length ? (
                <CommandEmpty>
                  {searchQuery ? 'No commands match your search' : 'No commands found'}
                </CommandEmpty>
              ) : (
                <CommandGroup heading="Commands">
                  {filteredCommands.map((command) => {
                    const Icon = command.icon;
                    return (
                      <CommandItem
                        key={command.id}
                        value={command.id}
                        onSelect={() => handleSelect(command.id)}
                      >
                        <Icon className="size-4 text-muted-foreground" />
                        <div className="flex-1">
                          <div className="font-medium text-sm">{command.name}</div>
                          <div className="text-xs text-muted-foreground">{command.description}</div>
                        </div>
                      </CommandItem>
                    );
                  })}
                </CommandGroup>
              )}
            </>
          )}
        </CommandList>
        
        {/* Bottom Toolbar */}
        <div className="h-6 px-3 py-1 bg-gray-50/80 dark:bg-gray-800/80 border-t border-gray-200/50 dark:border-gray-700/50 flex items-center justify-end text-xs">
          
          <div className="flex items-center gap-2">
            <div className="flex items-center gap-0.5">
              <kbd className="px-1 py-0 bg-white dark:bg-gray-700  rounded text-muted-foreground font-mono text-[10px]">
                ↵
              </kbd>
              <span className="text-gray-500 dark:text-gray-400">select</span>
            </div>
            
            <div className="flex items-center gap-0.5">
              <kbd className="px-1 py-0 bg-white dark:bg-gray-700  rounded text-muted-foreground font-mono text-[10px]">
                esc
              </kbd>
              <span className="text-gray-500 dark:text-gray-400">
                {showingPermissions ? 'back' : 'close'}
              </span>
            </div>
          </div>
        </div>
      </CommandPrimitive>
    </div>
  );
}

// Export commands for external use
export { slashCommands };

// Utility functions
export const shouldShowSlashCommands = (text: string): boolean => {
  return text === '/' || (text.startsWith('/') && !text.includes(' '));
};

export const handleSlashCommandNavigation = (
  e: React.KeyboardEvent,
  isVisible: boolean,
  selectedIndex: number,
  onIndexChange: (index: number) => void,
  onCommandSelect: (command: typeof slashCommands[0]) => void,
  onClose: () => void
): boolean => {
  if (!isVisible) return false;

  switch (e.key) {
    case 'ArrowDown':
      e.preventDefault();
      onIndexChange(selectedIndex < slashCommands.length - 1 ? selectedIndex + 1 : 0);
      return true;
    case 'ArrowUp':
      e.preventDefault();
      onIndexChange(selectedIndex > 0 ? selectedIndex - 1 : slashCommands.length - 1);
      return true;
    case 'Enter':
      e.preventDefault();
      onCommandSelect(slashCommands[selectedIndex]);
      return true;
    case 'Escape':
      e.preventDefault();
      onClose();
      return true;
  }
  
  return false;
};