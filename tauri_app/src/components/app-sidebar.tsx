import { IconClock, IconPlus } from '@tabler/icons-react';
import { Link, useNavigate } from '@tanstack/react-router';
import { Home } from 'lucide-react';
import type * as React from 'react';
import { SessionItem } from '@/components/session-item';
import {
  Sidebar,
  SidebarContent,
  SidebarGroup,
  SidebarGroupContent,
  SidebarGroupLabel,
  SidebarHeader,
  SidebarMenu,
  SidebarMenuButton,
  SidebarMenuItem,
} from '@/components/ui/sidebar';
import { useActiveSession, useCreateSession } from '@/hooks/useSession';
import { useSelectSession, useSessionsList } from '@/hooks/useSessionsList';

interface AppSidebarProps extends React.ComponentProps<typeof Sidebar> {
  sessionId?: string;
}

export function AppSidebar({ sessionId, ...props }: AppSidebarProps) {
  const navigate = useNavigate();
  const { data: sessions = [], isLoading: sessionsLoading } = useSessionsList();
  const selectSessionMutation = useSelectSession();
  const createSession = useCreateSession();
  const { data: currentSession } = useActiveSession(sessionId || '');

  // Sort sessions chronologically (most recent first)
  const sortedSessions = sessions.sort(
    (a, b) => new Date(b.createdAt).getTime() - new Date(a.createdAt).getTime()
  );

  const handleSessionSelect = (sessionId: string) => {
    selectSessionMutation.mutate(sessionId, {
      onSuccess: () => {
        navigate({
          to: '/$sessionId',
          params: { sessionId },
          replace: true,
        });
      },
    });
  };

  const handleNewSession = async () => {
    try {
      // NOTE: currentSession?.workingDirectory should always be defined here because:
      // 1. New users must select a project before accessing chat (enforced by routing)
      // 2. Backend now requires working directory for all session creation
      // 3. If currentSession is null/undefined, this will fail gracefully with backend validation
      const newSession = await createSession.mutateAsync({
        title: 'Chat Session',
        workingDirectory: currentSession?.workingDirectory,
      });
      navigate({
        to: '/$sessionId',
        params: { sessionId: newSession.id },
        replace: true,
      });
    } catch (error) {
      console.error('Failed to create new session:', error);
    }
  };

  return (
    <Sidebar collapsible="offcanvas" {...props}>
      <SidebarHeader className="border-b">
        <SidebarMenu>
          <SidebarMenuItem>
            <SidebarMenuButton
              asChild
              className="data-[slot=sidebar-menu-button]:!p-1.5"
            >
              <Link className="flex items-center gap-2" to="/">
                <Home className="!size-4 text-muted-foreground" />
                <span className="">Home</span>
              </Link>
            </SidebarMenuButton>
          </SidebarMenuItem>
        </SidebarMenu>
      </SidebarHeader>
      <SidebarContent>
        <SidebarGroup>
          <SidebarGroupLabel>Sessions</SidebarGroupLabel>
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
