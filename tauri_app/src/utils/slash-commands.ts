import { Clock, Command, HelpCircle, RefreshCw, Shield, LogIn, LogOut, UserCheck } from 'lucide-react';

interface SlashCommand {
  id: string;
  name: string;
  description: string;
  icon: React.ComponentType<{ className?: string }>;
}

export const slashCommands: SlashCommand[] = [
  {
    id: 'clear',
    name: 'clear',
    description: 'Start a new session',
    icon: RefreshCw,
  },
  {
    id: 'sessions',
    name: 'sessions',
    description: 'Browse and switch sessions',
    icon: Clock,
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
    id: 'login',
    name: 'login',
    description: 'Authenticate with Claude Code OAuth',
    icon: LogIn,
  },
  {
    id: 'logout',
    name: 'logout',
    description: 'Sign out from Claude Code',
    icon: LogOut,
  },
  {
    id: 'status',
    name: 'status',
    description: 'Check Claude Code authentication status',
    icon: UserCheck,
  },
  {
    id: 'permissions',
    name: 'permissions',
    description: 'System permissions and access',
    icon: Shield,
  },
];

export const shouldShowSlashCommands = (text: string): boolean => {
  return text === '/' || (text.startsWith('/') && !text.includes(' '));
};

export const handleSlashCommandNavigation = (
  e: React.KeyboardEvent,
  isVisible: boolean,
  selectedIndex: number,
  onIndexChange: (index: number) => void,
  onCommandSelect: (command: (typeof slashCommands)[0]) => void,
  onClose: () => void
): boolean => {
  if (!isVisible) return false;

  switch (e.key) {
    case 'ArrowDown':
      e.preventDefault();
      onIndexChange(
        selectedIndex < slashCommands.length - 1 ? selectedIndex + 1 : 0
      );
      return true;
    case 'ArrowUp':
      e.preventDefault();
      onIndexChange(
        selectedIndex > 0 ? selectedIndex - 1 : slashCommands.length - 1
      );
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
