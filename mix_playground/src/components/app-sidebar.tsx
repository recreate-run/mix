import { IconClock, IconPlus } from '@tabler/icons-react';
import { useNavigate } from '@tanstack/react-router';
import type * as React from 'react';
import { SessionItem } from '@/components/session-item';
import {
  Sidebar,
  SidebarContent,
  SidebarGroup,
  SidebarGroupContent,
  SidebarGroupLabel,
  SidebarMenu,
  SidebarMenuButton,
  SidebarMenuItem,
  SidebarTrigger,
} from '@/components/ui/sidebar';
import { useCreateSession } from '@/hooks/useSession';
import { useSessionsList } from '@/hooks/useSessionsList';

interface AppSidebarProps extends React.ComponentProps<typeof Sidebar> {
  sessionId?: string;
}

export function AppSidebar({ sessionId, ...props }: AppSidebarProps) {
  const navigate = useNavigate();
  const { data: sessions = [], isLoading: sessionsLoading } = useSessionsList();
  const createSession = useCreateSession();

  // Sort sessions chronologically (most recent first)
  const sortedSessions = sessions.sort(
    (a, b) => new Date(b.createdAt).getTime() - new Date(a.createdAt).getTime()
  );

  const handleSessionSelect = (sessionId: string) => {
    // Navigate directly to the selected session (stateless design)
    navigate({
      to: '/$sessionId',
      params: { sessionId },
      // Remove replace: true to prevent full route replacement
    });
  };

  const handleNewSession = async () => {
    try {
      // Create a new session
      const newSession = await createSession.mutateAsync({
        title: 'Chat Session',
      });
      navigate({
        to: '/$sessionId',
        params: { sessionId: newSession.id },
        // Remove replace: true for consistency
      });
    } catch (error) {
      console.error('Failed to create new session:', error);
    }
  };

  return (
    <Sidebar collapsible="offcanvas" {...props}>
      <SidebarContent>
        <SidebarGroup>
          <div className="flex items-center justify-between">
            <SidebarGroupLabel>Sessions</SidebarGroupLabel>
            <SidebarTrigger className="size-6" />
          </div>
          <SidebarGroupContent>
            <SidebarMenu>
              {/* New Session Button */}
              <SidebarMenuItem>
                <SidebarMenuButton onClick={handleNewSession}>
                  <IconPlus className="size-4" />
                  <span>New Session</span>
                </SidebarMenuButton>
              </SidebarMenuItem>

              {/* Sessions List */}
              {sessionsLoading ? (
                <SidebarMenuItem>
                  <SidebarMenuButton disabled>
                    <IconClock className="size-4" />
                    <span>Loading sessions...</span>
                  </SidebarMenuButton>
                </SidebarMenuItem>
              ) : sortedSessions.length === 0 ? (
                <SidebarMenuItem>
                  <SidebarMenuButton disabled>
                    <IconClock className="size-4" />
                    <span>No sessions</span>
                  </SidebarMenuButton>
                </SidebarMenuItem>
              ) : (
                sortedSessions.map((session) => {
                  const isActive = sessionId === session.id;
                  return (
                    <SessionItem
                      isActive={isActive}
                      key={session.id}
                      onClick={handleSessionSelect}
                      session={session}
                      currentSessionId={sessionId}
                      allSessions={sortedSessions}
                    />
                  );
                })
              )}
            </SidebarMenu>
          </SidebarGroupContent>
        </SidebarGroup>
      </SidebarContent>
      {/* <SidebarFooter>
				<NavUser user={data.user} />
			</SidebarFooter> */}
    </Sidebar>
  );
}
