import { createFileRoute } from '@tanstack/react-router'
import '@/styles/App.css';
import { QueryClient, QueryClientProvider } from '@tanstack/react-query';
import { useEffect } from 'react';
import { ChatApp } from '@/components/chat-app';
import { ThemeProvider } from '@/components/ui/theme-provider';
import { fetchAppList } from '@/hooks/useOpenApps';

export const Route = createFileRoute('/$sessionId')({
  component: SessionApp,
})

const queryClient = new QueryClient();

function SessionApp() {
  const { sessionId } = Route.useParams();
  
  useEffect(() => {
    queryClient.prefetchQuery({
      queryKey: ['appList'],
      queryFn: fetchAppList,
    });
  }, []);

  return (
    <QueryClientProvider client={queryClient}>
      <ThemeProvider defaultTheme="dark" storageKey="vite-ui-theme">
        <ChatApp sessionId={sessionId} />
      </ThemeProvider>
    </QueryClientProvider>
  );
}