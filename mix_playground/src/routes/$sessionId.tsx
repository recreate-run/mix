import { createFileRoute, useNavigate } from '@tanstack/react-router';
import '@/styles/App.css';
import { useEffect } from 'react';
import { AppSidebar } from '@/components/app-sidebar';
import { ChatApp } from '@/components/chat-app';
import { SidebarInset, SidebarProvider, SidebarTrigger, useSidebar } from '@/components/ui/sidebar';
import { useActiveSession } from '@/hooks/useSession';

export const Route = createFileRoute('/$sessionId')({
  component: SessionApp,
});

const LAST_SESSION_KEY = "mix-last-session-id";

function FloatingToggle() {
  const { state } = useSidebar();
  
  if (state === 'expanded') return null;
  
  return (
    <div className="fixed top-4 left-4 z-50">
      <SidebarTrigger />
    </div>
  );
}

function SessionApp() {
  const { sessionId } = Route.useParams();
  const navigate = useNavigate();
  const { data: session, isLoading, error } = useActiveSession(sessionId);

  // Track the current session as the last used session
  useEffect(() => {
    if (session && sessionId) {
      localStorage.setItem(LAST_SESSION_KEY, sessionId);
    }
  }, [session, sessionId]);

  // Redirect to home if session doesn't exist, but only after we're sure it failed
  useEffect(() => {
    if (!isLoading && (session === null || error)) {
      navigate({ to: '/', replace: true });
    }
  }, [session, isLoading, error, navigate]);

  // Always render the shell to prevent flashing
  // The individual components will handle their own loading states
  return (
    <SidebarProvider
      className="min-h-screen overflow-hidden overscroll-none"
      style={
        {
          '--sidebar-width': 'calc(var(--spacing) * 64)',
          '--header-height': 'calc(var(--spacing) * 10)',
        } as React.CSSProperties
      }
    >
      <AppSidebar sessionId={sessionId} variant="inset" />
      <SidebarInset className="flex h-screen flex-col">
        <FloatingToggle />
        {/* Always render ChatApp - it will handle loading states internally */}
        <ChatApp sessionId={sessionId} />
      </SidebarInset>
    </SidebarProvider>
  );
}
