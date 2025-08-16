import { createFileRoute } from '@tanstack/react-router'
import '@/styles/App.css';
import { useEffect, useState } from 'react';

import { ChatApp } from '@/components/chat-app';
import {
  SidebarInset,
  SidebarProvider,
} from "@/components/ui/sidebar"
import { AppSidebar } from "@/components/app-sidebar"
import { getDefaultWorkingDir } from '@/utils/defaultWorkingDir';
import { useFolderSelection } from '@/hooks/useFolderSelection';

export const Route = createFileRoute('/$sessionId')({
  component: SessionApp,
})



function SessionApp() {
  const { sessionId } = Route.useParams();

  // Folder selection state management
  const [defaultWorkingDir, setDefaultWorkingDir] = useState<string>('~/CreativeAgentProjects');
  const { selectedFolder } = useFolderSelection();

  // Initialize default working directory
  useEffect(() => {
    getDefaultWorkingDir().then(setDefaultWorkingDir);
  }, []);


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
        <ChatApp
          sessionId={sessionId}
          selectedFolder={selectedFolder || undefined}
          defaultWorkingDir={defaultWorkingDir}
        />

      </SidebarInset>
    </SidebarProvider>
  );
}



