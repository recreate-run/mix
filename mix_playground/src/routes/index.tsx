import { createFileRoute, useNavigate } from '@tanstack/react-router';
import { useEffect, useState } from 'react';
import { useCreateSession } from '@/hooks/useSession';
import { useSessionsList } from '@/hooks/useSessionsList';
import { LoadingScreen } from '@/components/loading/LoadingScreen';

export const Route = createFileRoute('/')({
  component: AutoRedirectHome,
});

const LAST_SESSION_KEY = 'mix-last-session-id';

function AutoRedirectHome() {
  const navigate = useNavigate();
  const createSession = useCreateSession();
  const { data: sessions, isLoading, error } = useSessionsList();
  const [isHandling, setIsHandling] = useState(false);
  const [animationComplete, setAnimationComplete] = useState(false);

  useEffect(() => {
    // Prevent multiple simultaneous executions
    if (isHandling || isLoading || error || !animationComplete) return;
    if (!sessions) return;

    setIsHandling(true);

    const handleRedirect = async () => {
      try {
        // Try to use last session
        const lastSessionId = localStorage.getItem(LAST_SESSION_KEY);
        if (lastSessionId && sessions.some((s) => s.id === lastSessionId)) {
          navigate({
            to: '/$sessionId',
            params: { sessionId: lastSessionId },
            replace: true,
          });
          return;
        }

        // Use most recent session if available
        if (sessions.length > 0) {
          const mostRecent = sessions.reduce((latest, session) =>
            new Date(session.createdAt) > new Date(latest.createdAt)
              ? session
              : latest
          );
          localStorage.setItem(LAST_SESSION_KEY, mostRecent.id);
          navigate({
            to: '/$sessionId',
            params: { sessionId: mostRecent.id },
            replace: true,
          });
          return;
        }

        // Create first session for fresh database
        const newSession = await createSession.mutateAsync({
          title: 'New Session',
        });
        localStorage.setItem(LAST_SESSION_KEY, newSession.id);
        navigate({
          to: '/$sessionId',
          params: { sessionId: newSession.id },
          replace: true,
        });
      } catch (err) {
        console.error('Failed to handle session redirect:', err);
        setIsHandling(false);
      }
    };

    handleRedirect();
  }, [sessions, isLoading, error, isHandling, navigate, createSession]);

  if (error) {
    return (
      <div className="flex min-h-screen items-center justify-center">
        <div className="text-center">
          <p className="text-red-500">Error loading sessions</p>
        </div>
      </div>
    );
  }

  return (
    <LoadingScreen duration={4} onComplete={() => setAnimationComplete(true)} />
  );
}
