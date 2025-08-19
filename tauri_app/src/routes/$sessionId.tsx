import { createFileRoute, useNavigate } from '@tanstack/react-router';
import '@/styles/App.css';
import { useEffect } from 'react';
import { AppSidebar } from '@/components/app-sidebar';
import { ChatApp } from '@/components/chat-app';
import { SidebarInset, SidebarProvider } from '@/components/ui/sidebar';
import { useActiveSession } from '@/hooks/useSession';
// import { PageHeader } from "@/components/page-header";

export const Route = createFileRoute('/$sessionId')({
  component: SessionApp,
});

function SessionApp() {
  const { sessionId } = Route.useParams();
  const navigate = useNavigate();
  const { data: session, isLoading } = useActiveSession(sessionId);

  // Redirect to home if session doesn't exist
  useEffect(() => {
    if (!isLoading && session === null) {
      navigate({ to: '/', replace: true });
    }
  }, [session, isLoading, navigate]);

  // Show loading or nothing while checking session
  if (isLoading || session === null) {
    return null;
  }

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
        {/* <PageHeader sessionId={sessionId} /> */}

        <ChatApp sessionId={sessionId} />
      </SidebarInset>
    </SidebarProvider>
  );
}
