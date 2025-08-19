import { QueryClient, QueryClientProvider } from '@tanstack/react-query';
import { createRootRoute, Link, Outlet } from '@tanstack/react-router';
import { useState } from 'react';
import { Button } from '@/components/ui/button';
import { ThemeProvider } from '@/components/ui/theme-provider';
import { TooltipProvider } from '@/components/ui/tooltip';

export const Route = createRootRoute({
  component: RootComponent,
  notFoundComponent: defaultNotFoundComponent,
  errorComponent: defaultErrorComponent,
});

const queryClient = new QueryClient();

function RootComponent() {
  return (
    <QueryClientProvider client={queryClient}>
      <ThemeProvider defaultTheme="dark" storageKey="vite-ui-theme">
        <TooltipProvider>
          <Outlet />
        </TooltipProvider>
      </ThemeProvider>
    </QueryClientProvider>
  );
}

function defaultNotFoundComponent() {
  return (
    <div>
      <p>Not found!</p>
      <Link to="/">Go home</Link>
    </div>
  );
}

interface ErrorComponentProps {
  error: Error;
  info?: { componentStack: string };
  reset: () => void;
}

function defaultErrorComponent({ error, info, reset }: ErrorComponentProps) {
  const [showError, setShowError] = useState(
    process.env.NODE_ENV === 'development'
  );

  return (
    <div className="mx-auto max-w-2xl p-6">
      <h2 className="mb-4 font-bold text-2xl text-red-600">
        Something went wrong!
      </h2>

      <div className="mb-4">
        <Button onClick={() => setShowError(!showError)}>
          {showError ? 'Hide Error' : 'Show Error'}
        </Button>

        <Button onClick={reset}>Try Again</Button>

        <Link
          className="inline-block rounded bg-gray-500 px-4 py-2 text-white hover:bg-gray-600"
          to="/"
        >
          Go Home
        </Link>
      </div>

      {showError && (
        <div className="rounded border border-red-200 bg-red-50 p-4">
          <h3 className="mb-2 font-semibold text-red-800">Error Details:</h3>
          <p className="mb-2 text-red-700">{error.message}</p>

          {info?.componentStack && (
            <details className="mt-2">
              <summary className="cursor-pointer font-medium text-red-600">
                Component Stack Trace
              </summary>
              <pre className="mt-2 overflow-x-auto whitespace-pre-wrap text-red-600 text-xs">
                {info.componentStack}
              </pre>
            </details>
          )}
        </div>
      )}
    </div>
  );
}
