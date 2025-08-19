import { IconTrash } from '@tabler/icons-react';
import { ask } from '@tauri-apps/plugin-dialog';
import { useState } from 'react';
import { Button } from '@/components/ui/button';
import { SidebarMenuButton, SidebarMenuItem } from '@/components/ui/sidebar';
import {
  type SessionData,
  TITLE_TRUNCATE_LENGTH,
  useDeleteSession,
} from '@/hooks/useSessionsList';

interface SessionItemProps {
  session: SessionData;
  isActive: boolean;
  onClick: (sessionId: string) => void;
}

export function SessionItem({ session, isActive, onClick }: SessionItemProps) {
  const [isHovered, setIsHovered] = useState(false);
  const deleteSessionMutation = useDeleteSession();

  const handleDelete = async (e: React.MouseEvent) => {
    e.stopPropagation();
    if (deleteSessionMutation.isPending) return;
    if (
      await ask(
        `Are you sure you want to delete the session "${getDisplayTitle(session)}"? This action cannot be undone.`
      )
    ) {
      deleteSessionMutation.mutate(session.id);
    }
  };

  // Helper function to get display title (copied from command-slash.tsx)
  const getDisplayTitle = (session: SessionData) => {
    if (!session.firstUserMessage || session.firstUserMessage.trim() === '') {
      return session.title;
    }

    let displayText = session.firstUserMessage;
    try {
      const parsed = JSON.parse(session.firstUserMessage);
      const textPart = parsed.find((part: any) => part.type === 'text');
      if (textPart?.data?.text) {
        displayText = textPart.data.text;
      }
    } catch {
      displayText = session.firstUserMessage;
    }

    const truncated =
      displayText.length > TITLE_TRUNCATE_LENGTH
        ? `${displayText.substring(0, TITLE_TRUNCATE_LENGTH)}...`
        : displayText;

    return truncated;
  };

  const formatDate = (date: Date) => {
    const now = new Date();
    const diffDays = Math.floor(
      (now.getTime() - date.getTime()) / (1000 * 60 * 60 * 24)
    );

    if (diffDays === 0) return 'Today';
    if (diffDays === 1) return 'Yesterday';
    if (diffDays < 7) return `${diffDays}d ago`;
    return date.toLocaleDateString('en-US', { month: 'short', day: 'numeric' });
  };

  const createdDate = new Date(session.createdAt);

  return (
    <SidebarMenuItem>
      <div
        className="group relative"
        onMouseEnter={() => setIsHovered(true)}
        onMouseLeave={() => setIsHovered(false)}
      >
        <SidebarMenuButton
          className="flex h-auto flex-col items-start gap-1 py-2 pr-8"
          isActive={isActive}
          onClick={() => onClick(session.id)}
        >
          <div className="flex w-full items-center gap-2">
            <span className="flex-1 truncate font-medium text-sm">
              {getDisplayTitle(session)}
            </span>
          </div>
          <div className="flex items-center gap-2 text-muted-foreground text-xs">
            <span>{formatDate(createdDate)}</span>
            <span>•</span>
            <span>{session.messageCount} messages</span>
          </div>
        </SidebarMenuButton>
        {isHovered && !isActive && (
          <Button
            className="-translate-y-1/2 absolute top-1/2 right-1 opacity-70 hover:opacity-100"
            disabled={deleteSessionMutation.isPending}
            onClick={handleDelete}
            size="sm"
            variant="ghost"
          >
            <IconTrash className="size-4" />
          </Button>
        )}
      </div>
    </SidebarMenuItem>
  );
}
