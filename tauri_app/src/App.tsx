import './App.css';

import { QueryClient, QueryClientProvider } from '@tanstack/react-query';
import { useEffect, useState } from 'react';
import { AutoUpdater } from '@/components/auto-updater';
import { ChatApp } from '@/components/chat-app';
import { ThemeProvider } from '@/components/ui/theme-provider';
import { fetchAppList } from '@/hooks/useOpenApps';

const queryClient = new QueryClient();

const App = () => {
  const [showUpdater, setShowUpdater] = useState(false);

  useEffect(() => {
    queryClient.prefetchQuery({
      queryKey: ['appList'],
      queryFn: fetchAppList,
    });

    // Show updater on startup
    const timer = setTimeout(() => setShowUpdater(true), 1000);
    return () => clearTimeout(timer);
  }, []);

  return (
    <QueryClientProvider client={queryClient}>
      <ThemeProvider defaultTheme="dark" storageKey="vite-ui-theme">
        {showUpdater && (
          <div className="fixed top-4 right-4 z-50">
            <AutoUpdater />
          </div>
        )}
        <ChatApp sessionId="default" />
      </ThemeProvider>
    </QueryClientProvider>
  );
};
export default App;
