import * as React from 'react'
import { Outlet, createRootRoute } from '@tanstack/react-router'
import { ThemeProvider } from '@/components/ui/theme-provider';
import { QueryClient, QueryClientProvider } from '@tanstack/react-query';
import { TooltipProvider } from '@/components/ui/tooltip';

export const Route = createRootRoute({
  component: RootComponent,
})

const queryClient = new QueryClient();
  
function RootComponent() {
  return (
    <React.Fragment>
      <QueryClientProvider client={queryClient}>
        <ThemeProvider defaultTheme="dark" storageKey="vite-ui-theme">
          <TooltipProvider>
            <Outlet />
          </TooltipProvider>
        </ThemeProvider>
      </QueryClientProvider>
    </React.Fragment>
  )
}
