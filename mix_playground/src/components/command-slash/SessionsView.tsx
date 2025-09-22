import { Clock } from 'lucide-react';
import { CommandEmpty, CommandGroup, CommandItem } from '@/components/ui/command';
import { useSessionsList } from '@/hooks/useSessionsList';
import { useActiveSession } from '@/hooks/useSession';
import { formatMessageCounts } from '@/types/common';
import { getDisplayTitle } from '@/utils/sessionUtils';
import { BackButton } from './shared/BackButton';
import { StatusBadge } from './shared/StatusBadge';

interface SessionsViewProps {
  sessionId: string;
  onBackToCommands: () => void;
  onNavigateToSession: (sessionId: string) => void;
}

export function SessionsView({
  sessionId,
  onBackToCommands,
  onNavigateToSession,
}: SessionsViewProps) {
  const { data: sessions = [], isLoading: sessionsLoading } = useSessionsList();
  const activeSession = useActiveSession(sessionId);

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

  // Sort sessions chronologically (most recent first)
  const sortedSessions = sessions
    .sort(
      (a, b) =>
        new Date(b.createdAt).getTime() - new Date(a.createdAt).getTime()
    );

  const handleSessionSelect = (sessionId: string) => {
    onNavigateToSession(sessionId);
  };

  if (sessionsLoading) {
    return <CommandEmpty>Loading sessions...</CommandEmpty>;
  }

  if (!sortedSessions.length) {
    return <CommandEmpty>No sessions found</CommandEmpty>;
  }

  return (
    <CommandGroup heading={`Sessions (${sortedSessions.length})`}>
        <BackButton
          label="Back to Commands"
          onSelect={onBackToCommands}
          value="back-to-commands"
        />

        {sortedSessions.map((session) => {
        const isActive = activeSession.data?.id === session.id;
        const createdDate = new Date(session.createdAt);

        return (
          <CommandItem
            key={session.id}
            onSelect={() => handleSessionSelect(session.id)}
            value={getDisplayTitle(session)}
            className={isActive ? 'bg-accent' : ''}
          >
            <Clock className="size-4 text-muted-foreground" />
            <div className="flex-1">
              <div className="flex items-center gap-2 font-medium text-sm">
                {getDisplayTitle(session)}
                {isActive && <StatusBadge status="current" />}
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
  );
}