import { IconTrash } from '@tabler/icons-react';
import { ask } from '@tauri-apps/plugin-dialog';
import { useState } from 'react';
import { useNavigate } from '@tanstack/react-router';
import { Button } from '@/components/ui/button';
import { SidebarMenuButton, SidebarMenuItem } from '@/components/ui/sidebar';
import type { SessionData } from '@/types/common';
import { useDeleteSession } from '@/hooks/useSessionsList';
import { getDisplayTitle } from '@/utils/sessionUtils';
import { rpcCall } from '@/lib/rpc';


interface SessionItemProps {
  session: SessionData;
  isActive: boolean;
  onClick: (sessionId: string) => void;
  currentSessionId?: string;
  allSessions: SessionData[];
}

export function SessionItem({ session, isActive, onClick, currentSessionId, allSessions }: SessionItemProps) {
  const [isHovered, setIsHovered] = useState(false);
  const navigate = useNavigate();
  const deleteSessionMutation = useDeleteSession();

  const handleDelete = async (e: React.MouseEvent) => {
    e.stopPropagation();
    if (deleteSessionMutation.isPending || session.isDeleting) return;
    
    if (
      await ask(
        `Are you sure you want to delete the session "${getDisplayTitle(session)}"? This action cannot be undone.`
      )
    ) {
      const isCurrentSession = currentSessionId === session.id;
      
      // If we're deleting the current session, find the next session to navigate to
      let nextSessionId: string | null = null;
      if (isCurrentSession && allSessions.length > 1) {
        const currentIndex = allSessions.findIndex(s => s.id === session.id);
        if (currentIndex !== -1) {
          // Try next session, then previous session
          if (currentIndex < allSessions.length - 1) {
            nextSessionId = allSessions[currentIndex + 1].id;
          } else if (currentIndex > 0) {
            nextSessionId = allSessions[currentIndex - 1].id;
          }
        }
      }
      
      try {
        if (isCurrentSession) {
          if (nextSessionId) {
            // First, switch to the next session to avoid backend restriction
            await rpcCall('sessions.select', { id: nextSessionId });
            
            // Navigate immediately for instant UI feedback
            navigate({
              to: '/$sessionId',
              params: { sessionId: nextSessionId },
              replace: true,
            });
          } else {
            // No other sessions available, clear current session by selecting empty ID
            await rpcCall('sessions.select', { id: '' });
            
            // Navigate to home
            navigate({ to: '/', replace: true });
          }
        }
        
        // Trigger deletion mutation (includes animation timing and backend call)
        await deleteSessionMutation.mutateAsync(session.id);
      } catch (error) {
        console.error('Failed to delete session:', error);
      }
    }
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
        className={`group relative ${
          session.isDeleting ? 'opacity-50 cursor-not-allowed pointer-events-none' : ''
        }`}
        onMouseEnter={() => !session.isDeleting && setIsHovered(true)}
        onMouseLeave={() => setIsHovered(false)}
      >
        <SidebarMenuButton
          className="flex h-auto flex-col items-start gap-1 py-2 pr-8 min-h-[60px]"
          isActive={isActive}
          onClick={() => !session.isDeleting && onClick(session.id)}
        >
          <div className="flex w-full items-center gap-2">
            <span className="flex-1 truncate font-medium text-sm">
              {getDisplayTitle(session)}
            </span>
          </div>
          <div className="flex items-center gap-2 text-muted-foreground text-xs">
            <span>{formatDate(createdDate)}</span>
          </div>
        </SidebarMenuButton>
        <Button
          className={`absolute top-4 right-1 ${
            isHovered && !session.isDeleting ? 'opacity-70 hover:opacity-100' : 'invisible'
          }`}
          disabled={deleteSessionMutation.isPending || session.isDeleting}
          onClick={handleDelete}
          size="sm"
          variant="ghost"
        >
          <IconTrash className="size-5" />
        </Button>
      </div>
    </SidebarMenuItem>
  );
}
