import { createFileRoute, useNavigate } from '@tanstack/react-router'
import '@/styles/App.css';
import { useEffect } from 'react';

import { ChatApp } from '@/components/chat-app';
import {
  SidebarInset,
  SidebarProvider,
} from "@/components/ui/sidebar"
import { AppSidebar } from "@/components/app-sidebar"
import { useActiveSession } from '@/hooks/useSession';

export const Route = createFileRoute('/$sessionId')({
  component: SessionApp,
})



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
      className='h-screen overflow-hidden '
      style={
        {
          "--sidebar-width": "calc(var(--spacing) * 72)",
          "--header-height": "calc(var(--spacing) * 10)",
        } as React.CSSProperties
      }
    >
      <AppSidebar variant="inset" sessionId={sessionId} />
      <SidebarInset>
        <ChatApp sessionId={sessionId} />

      </SidebarInset>
    </SidebarProvider>
  );
}



