
import './App.css';

import { useEffect } from 'react';
import { QueryClient, QueryClientProvider } from '@tanstack/react-query';
import { ThemeProvider } from '@/components/ui/theme-provider';
import {ChatApp} from '@/components/chat-app';
import { fetchVisibleApps } from '@/hooks/useOpenApps';
import { safeTrackEvent } from '@/lib/posthog';


const queryClient = new QueryClient();



const App = () => {
  // Track session duration
  useEffect(() => {
    const sessionStartTime = Date.now();
    
    // Track session start
    safeTrackEvent('session_started', {
      timestamp: new Date().toISOString()
    });
    
    // Track session duration when component unmounts or on page close
    const trackSessionDuration = () => {
      const duration = Date.now() - sessionStartTime;
      safeTrackEvent('session_duration', {
        duration_ms: duration,
        timestamp: new Date().toISOString()
      });
    };
    
    // Add event listener for page visibility changes and beforeunload
    window.addEventListener('beforeunload', trackSessionDuration);
    document.addEventListener('visibilitychange', () => {
      if (document.visibilityState === 'hidden') {
        trackSessionDuration();
      }
    });
    
    // Clean up event listeners on component unmount
    return () => {
      trackSessionDuration();
      window.removeEventListener('beforeunload', trackSessionDuration);
      document.removeEventListener('visibilitychange', trackSessionDuration);
    };
  }, []);
  
  useEffect(() => {
    queryClient.prefetchQuery({
      queryKey: ['openApps'],
      queryFn: fetchVisibleApps,
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
