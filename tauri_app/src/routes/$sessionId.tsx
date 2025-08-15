import { createFileRoute } from '@tanstack/react-router'
import '@/styles/App.css';
import { useEffect, useState } from 'react';

import { ChatApp } from '@/components/chat-app';
import { fetchAppList } from '@/hooks/useOpenApps';
import { PageHeader } from '@/components/page-header';
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
  const { selectedFolder, selectFolder } = useFolderSelection();

  // Initialize default working directory
  useEffect(() => {
    getDefaultWorkingDir().then(setDefaultWorkingDir);
  }, []);

  const handleFolderSelect = async () => {
    try {
      const selectedFolderPath = await selectFolder();
      if (selectedFolderPath) {
        console.log('Working directory selected:', selectedFolderPath);
      }
    } catch (error) {
      console.error('Failed to select working directory:', error);
    }
  };

  // useEffect(() => {
  //   queryClient.prefetchQuery({
  //     queryKey: ['appList'],
  //     queryFn: fetchAppList,
  //   });
  // }, []);

  return (
    <SidebarProvider
      style={
        {
          "--sidebar-width": "calc(var(--spacing) * 72)",
          "--header-height": "calc(var(--spacing) * 10)",
        } as React.CSSProperties
      }
    >
      <AppSidebar variant="inset" />
      <SidebarInset>
        <PageHeader
          sessionId={sessionId}
          selectedFolder={selectedFolder}
          defaultWorkingDir={defaultWorkingDir}
          onFolderSelect={handleFolderSelect}
        />
        <ChatApp
          sessionId={sessionId}
          selectedFolder={selectedFolder}
          defaultWorkingDir={defaultWorkingDir}
        />
      </SidebarInset>
    </SidebarProvider>
  );
}



