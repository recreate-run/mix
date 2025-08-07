
import './App.css';

import { useEffect } from 'react';
import { QueryClient, QueryClientProvider } from '@tanstack/react-query';
import { ThemeProvider } from '@/components/ui/theme-provider';
import {ChatApp} from '@/components/chat-app';
import { fetchAppList} from '@/hooks/useOpenApps';


const queryClient = new QueryClient();



const App = () => {
  
  useEffect(() => {
    queryClient.prefetchQuery({
      queryKey: ['appList'],
      queryFn: fetchAppList,
    });
  }, []);

  return (
    <QueryClientProvider client={queryClient}>
      <ThemeProvider defaultTheme="dark" storageKey="vite-ui-theme">

        <ChatApp />

      </ThemeProvider>
    </QueryClientProvider>
  );
};
export default App;
